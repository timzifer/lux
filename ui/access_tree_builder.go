package ui

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// BuildAccessTree constructs an AccessTree from a resolved element tree.
// It performs a depth-first walk, extracting accessibility information from
// AccessibleWidget implementations and SemanticProvider surfaces.
// The windowBounds specify the window's screen-space position for coordinate mapping.
// The dispatcher provides per-widget screen bounds from the layout pass.
//
// The tree always starts with a synthetic root node at index 0 (RoleGroup)
// that wraps all top-level elements. The UIA root provider maps to this node.
func BuildAccessTree(root Element, reconciler *Reconciler, windowBounds a11y.Rect, dispatcher ...*EventDispatcher) a11y.AccessTree {
	b := accessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: windowBounds,
	}
	if len(dispatcher) > 0 {
		b.dispatcher = dispatcher[0]
	}
	// Create synthetic root node at index 0. All elements become its children.
	rootIdx := b.addNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, windowBounds)
	b.walk(root, int32(rootIdx), windowBounds)
	tree := a11y.AccessTree{Nodes: b.nodes}
	tree.EnsureIndex()
	return tree
}

type accessTreeBuilder struct {
	reconciler   *Reconciler
	dispatcher   *EventDispatcher
	windowBounds a11y.Rect
	nodes        []a11y.AccessTreeNode
	nextID       a11y.AccessNodeID
}

// boundsForWidget returns the screen-space bounds for a widget.
// Falls back to the provided fallback if per-widget bounds aren't available.
func (b *accessTreeBuilder) boundsForWidget(uid UID, fallback a11y.Rect) a11y.Rect {
	if b.dispatcher != nil {
		if db, ok := b.dispatcher.BoundsForWidget(uid); ok {
			return a11y.Rect{
				X:      float64(db.X),
				Y:      float64(db.Y),
				Width:  float64(db.W),
				Height: float64(db.H),
			}
		}
	}
	return fallback
}

func (b *accessTreeBuilder) allocID() a11y.AccessNodeID {
	b.nextID++
	return b.nextID
}

// addNode appends a node, links siblings, and returns the node's index.
func (b *accessTreeBuilder) addNode(node a11y.AccessNode, parentIdx int32, bounds a11y.Rect) int {
	if bounds.Width == 0 && bounds.Height == 0 {
		bounds = b.windowBounds
	}
	idx := int32(len(b.nodes))
	id := b.allocID()

	n := a11y.AccessTreeNode{
		ID:          id,
		ParentIndex: parentIdx,
		FirstChild:  -1,
		NextSibling: -1,
		PrevSibling: -1,
		ChildCount:  0,
		Node:        node,
		Bounds:      bounds,
	}
	b.nodes = append(b.nodes, n)

	// Link to parent's child list.
	if parentIdx >= 0 {
		parent := &b.nodes[parentIdx]
		parent.ChildCount++
		if parent.FirstChild < 0 {
			parent.FirstChild = idx
		} else {
			// Find last sibling and link.
			last := parent.FirstChild
			for b.nodes[last].NextSibling >= 0 {
				last = b.nodes[last].NextSibling
			}
			b.nodes[last].NextSibling = idx
			b.nodes[idx].PrevSibling = last
		}
	}

	return int(idx)
}

// divideVertical splits avail into n horizontal strips.
func divideVertical(avail a11y.Rect, i, n int) a11y.Rect {
	if n <= 0 {
		return avail
	}
	h := avail.Height / float64(n)
	return a11y.Rect{X: avail.X, Y: avail.Y + float64(i)*h, Width: avail.Width, Height: h}
}

// divideHorizontal splits avail into n vertical strips.
func divideHorizontal(avail a11y.Rect, i, n int) a11y.Rect {
	if n <= 0 {
		return avail
	}
	w := avail.Width / float64(n)
	return a11y.Rect{X: avail.X + float64(i)*w, Y: avail.Y, Width: w, Height: avail.Height}
}

// walk traverses the element tree and creates access tree nodes.
// avail is the approximate screen-space bounds available to this element.
// Container elements divide avail among their children so that each
// element gets a unique region for hit-testing.
func (b *accessTreeBuilder) walk(el Element, parentIdx int32, avail a11y.Rect) {
	switch node := el.(type) {
	case nil, emptyElement:
		return

	case widgetBoundsElement:
		widget := b.reconciler.widgets[node.WidgetUID]
		state := b.reconciler.states[node.WidgetUID]

		var accessNode a11y.AccessNode
		if aw, ok := widget.(AccessibleWidget); ok {
			accessNode = aw.Accessibility(state)
		} else {
			accessNode = a11y.AccessNode{Role: a11y.RoleGroup}
		}

		bounds := b.boundsForWidget(node.WidgetUID, avail)
		idx := b.addNode(accessNode, parentIdx, bounds)
		b.walk(node.Child, int32(idx), bounds)

	case surfaceElement:
		if sp, ok := node.Provider.(SemanticProvider); ok {
			bounds := draw.R(0, 0, float32(node.Width), float32(node.Height))
			semantics := sp.SnapshotSemantics(bounds)
			b.mergeSurfaceSemantics(semantics, parentIdx, node.Width, node.Height)
		} else {
			b.addNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Surface"}, parentIdx, a11y.Rect{
				Width: float64(node.Width), Height: float64(node.Height),
			})
		}

	case textElement:
		b.addNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: node.Content}, parentIdx, avail)

	case buttonElement:
		label := extractButtonLabel(node.Content)
		accessNode := a11y.AccessNode{
			Role:  a11y.RoleButton,
			Label: label,
		}
		if node.OnClick != nil {
			accessNode.Actions = []a11y.AccessAction{
				{Name: "activate", Trigger: node.OnClick},
			}
		}
		idx := b.addNode(accessNode, parentIdx, avail)
		b.walk(node.Content, int32(idx), avail)

	// ── Container elements: divide bounds among children ──
	case boxElement:
		n := len(node.Children)
		for i, child := range node.Children {
			var childBounds a11y.Rect
			if node.Axis == AxisColumn {
				childBounds = divideVertical(avail, i, n)
			} else {
				childBounds = divideHorizontal(avail, i, n)
			}
			b.walk(child, parentIdx, childBounds)
		}
	case stackElement:
		// Stack overlaps children; each gets full bounds.
		for _, child := range node.Children {
			b.walk(child, parentIdx, avail)
		}
	case flexElement:
		n := len(node.Children)
		for i, child := range node.Children {
			var childBounds a11y.Rect
			if node.Direction == FlexRow {
				childBounds = divideHorizontal(avail, i, n)
			} else {
				childBounds = divideVertical(avail, i, n)
			}
			b.walk(child, parentIdx, childBounds)
		}
	case gridElement:
		n := len(node.Children)
		for i, child := range node.Children {
			childBounds := divideVertical(avail, i, n)
			b.walk(child, parentIdx, childBounds)
		}
	case themedElement:
		n := len(node.Children)
		for i, child := range node.Children {
			childBounds := divideVertical(avail, i, n)
			b.walk(child, parentIdx, childBounds)
		}
	case customLayoutElement:
		n := len(node.Children)
		for i, child := range node.Children {
			childBounds := divideVertical(avail, i, n)
			b.walk(child, parentIdx, childBounds)
		}

	// ── Wrapper elements: pass through bounds ──
	case paddingElement:
		b.walk(node.Child, parentIdx, avail)
	case sizedBoxElement:
		if node.Child != nil {
			b.walk(node.Child, parentIdx, avail)
		}
	case expandedElement:
		b.walk(node.Child, parentIdx, avail)
	case scrollViewElement:
		b.walk(node.Child, parentIdx, avail)
	case keyedElement:
		b.walk(node.Child, parentIdx, avail)
	case blurBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case shadowBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case opacityBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case frostedGlassElement:
		b.walk(node.Child, parentIdx, avail)
	case innerShadowBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case elevationBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case elevationCardElement:
		b.walk(node.Child, parentIdx, avail)
	case vibrancyElement:
		b.walk(node.Child, parentIdx, avail)
	case glowBoxElement:
		b.walk(node.Child, parentIdx, avail)
	case cardElement:
		b.walk(node.Child, parentIdx, avail)
	case badgeElement:
		b.walk(node.Content, parentIdx, avail)
	case chipElement:
		b.walk(node.Label, parentIdx, avail)
	case tooltipElement:
		b.walk(node.Trigger, parentIdx, avail)
	case splitViewElement:
		// Split: first gets left/top half, second gets right/bottom half.
		half1 := a11y.Rect{X: avail.X, Y: avail.Y, Width: avail.Width / 2, Height: avail.Height}
		half2 := a11y.Rect{X: avail.X + avail.Width/2, Y: avail.Y, Width: avail.Width / 2, Height: avail.Height}
		b.walk(node.First, parentIdx, half1)
		b.walk(node.Second, parentIdx, half2)

	// ── Interactive leaf elements with a11y semantics ──
	case checkboxElement:
		an := a11y.AccessNode{
			Role:   a11y.RoleCheckbox,
			Label:  node.Label,
			States: a11y.AccessStates{Checked: node.Checked},
		}
		if node.OnToggle != nil {
			toggle := node.OnToggle
			checked := node.Checked
			an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { toggle(!checked) }}}
		}
		b.addNode(an, parentIdx, avail)
	case toggleElement:
		an := a11y.AccessNode{
			Role:   a11y.RoleToggle,
			States: a11y.AccessStates{Checked: node.On},
		}
		if node.OnToggle != nil {
			toggle := node.OnToggle
			on := node.On
			an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { toggle(!on) }}}
		}
		b.addNode(an, parentIdx, avail)
	case sliderElement:
		b.addNode(a11y.AccessNode{
			Role: a11y.RoleSlider,
		}, parentIdx, avail)
	case progressBarElement:
		b.addNode(a11y.AccessNode{
			Role: a11y.RoleProgressBar,
		}, parentIdx, avail)
	case textFieldElement:
		b.addNode(a11y.AccessNode{
			Role:  a11y.RoleTextInput,
			Label: node.Placeholder,
			Value: node.Value,
		}, parentIdx, avail)
	case radioElement:
		b.addNode(a11y.AccessNode{
			Role:   a11y.RoleCheckbox,
			Label:  node.Label,
			States: a11y.AccessStates{Checked: node.Selected},
		}, parentIdx, avail)
	case selectElement:
		b.addNode(a11y.AccessNode{
			Role:  a11y.RoleCombobox,
			Value: node.Value,
		}, parentIdx, avail)

	// ── Data-driven elements with dynamic children ──
	case treeElement:
		treeIdx := b.addNode(a11y.AccessNode{Role: a11y.RoleTree, Label: "Tree"}, parentIdx, avail)
		b.walkTreeNodes(node, node.RootIDs, int32(treeIdx), 0, avail)
	case virtualListElement:
		listIdx := b.addNode(a11y.AccessNode{Role: a11y.RoleListbox, Label: "List"}, parentIdx, avail)
		if node.BuildItem != nil {
			for i := 0; i < node.ItemCount; i++ {
				item := node.BuildItem(i)
				childBounds := divideVertical(avail, i, node.ItemCount)
				b.walk(item, int32(listIdx), childBounds)
			}
		}

	default:
		// Leaf-only elements (divider, spacer, icon, gradient, etc.)
		// or unrecognized types — no a11y node needed.
	}
}

// mergeSurfaceSemantics converts SurfaceSemantics into AccessTreeNodes.
func (b *accessTreeBuilder) mergeSurfaceSemantics(sem SurfaceSemantics, parentIdx int32, w, h float32) {
	surfaceIDToTreeIdx := make(map[SurfaceNodeID]int32)

	for _, sn := range sem.Roots {
		node := surfaceAccessNodeToAccessNode(sn)
		treeBounds := a11y.Rect{
			X:      float64(sn.Bounds.X),
			Y:      float64(sn.Bounds.Y),
			Width:  float64(sn.Bounds.W),
			Height: float64(sn.Bounds.H),
		}
		p := parentIdx
		if sn.Parent != 0 {
			if mapped, ok := surfaceIDToTreeIdx[sn.Parent]; ok {
				p = mapped
			}
		}
		idx := b.addNode(node, p, treeBounds)
		surfaceIDToTreeIdx[sn.ID] = int32(idx)
	}
}

func surfaceAccessNodeToAccessNode(sn SurfaceAccessNode) a11y.AccessNode {
	return a11y.AccessNode{
		Role:        sn.Role,
		Label:       sn.Label,
		Description: sn.Description,
		Value:       sn.Value,
		States:      sn.States,
	}
}

// walkTreeNodes recursively walks a treeElement's nodes and adds them to the access tree.
func (b *accessTreeBuilder) walkTreeNodes(tree treeElement, ids []string, parentIdx int32, depth int, avail a11y.Rect) {
	n := len(ids)
	for i, id := range ids {
		expanded := tree.State != nil && tree.State.IsExpanded(id)
		selected := tree.State != nil && tree.State.Selected == id

		var kids []string
		if tree.Children != nil {
			kids = tree.Children(id)
		}
		hasKids := len(kids) > 0

		var label string
		if tree.BuildNode != nil {
			nodeEl := tree.BuildNode(id, depth, expanded, selected)
			label = extractElementLabel(nodeEl)
		}
		if label == "" {
			label = id
		}

		an := a11y.AccessNode{
			Role:  a11y.RoleGroup,
			Label: label,
			States: a11y.AccessStates{
				Selected: selected,
				Expanded: hasKids && expanded,
			},
		}
		if tree.OnSelect != nil {
			selectID := id
			onSelect := tree.OnSelect
			an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { onSelect(selectID) }}}
		}

		childBounds := divideVertical(avail, i, n)
		idx := b.addNode(an, parentIdx, childBounds)

		if hasKids && expanded {
			b.walkTreeNodes(tree, kids, int32(idx), depth+1, childBounds)
		}
	}
}

// extractElementLabel tries to extract a text label from an arbitrary element.
func extractElementLabel(el Element) string {
	switch node := el.(type) {
	case textElement:
		return node.Content
	case boxElement:
		for _, c := range node.Children {
			if l := extractElementLabel(c); l != "" {
				return l
			}
		}
	case paddingElement:
		return extractElementLabel(node.Child)
	case sizedBoxElement:
		if node.Child != nil {
			return extractElementLabel(node.Child)
		}
	case expandedElement:
		return extractElementLabel(node.Child)
	case widgetBoundsElement:
		return extractElementLabel(node.Child)
	case keyedElement:
		return extractElementLabel(node.Child)
	}
	return ""
}

// extractButtonLabel tries to get a text label from a button's content element.
func extractButtonLabel(el Element) string {
	switch node := el.(type) {
	case textElement:
		return node.Content
	case boxElement:
		for _, c := range node.Children {
			if label := extractButtonLabel(c); label != "" {
				return label
			}
		}
	}
	return ""
}
