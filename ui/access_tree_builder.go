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
	b.walk(root, int32(rootIdx))
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
// Falls back to windowBounds if per-widget bounds aren't available.
func (b *accessTreeBuilder) boundsForWidget(uid UID) a11y.Rect {
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
	return b.windowBounds
}

func (b *accessTreeBuilder) allocID() a11y.AccessNodeID {
	b.nextID++
	return b.nextID
}

// addNode appends a node, links siblings, and returns the node's index.
// If bounds are zero-sized, falls back to windowBounds so elements are
// visible to the accessibility system.
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

func (b *accessTreeBuilder) walk(el Element, parentIdx int32) {
	switch node := el.(type) {
	case nil, emptyElement:
		return

	case widgetBoundsElement:
		// Check if the widget implements AccessibleWidget.
		widget := b.reconciler.widgets[node.WidgetUID]
		state := b.reconciler.states[node.WidgetUID]

		var accessNode a11y.AccessNode
		if aw, ok := widget.(AccessibleWidget); ok {
			accessNode = aw.Accessibility(state)
		} else {
			accessNode = a11y.AccessNode{Role: a11y.RoleGroup}
		}

		// Use per-widget bounds from the layout pass if available.
		bounds := b.boundsForWidget(node.WidgetUID)

		idx := b.addNode(accessNode, parentIdx, bounds)
		b.walk(node.Child, int32(idx))

	case surfaceElement:
		// Check if the surface's provider implements SemanticProvider.
		if sp, ok := node.Provider.(SemanticProvider); ok {
			bounds := draw.R(0, 0, float32(node.Width), float32(node.Height))
			semantics := sp.SnapshotSemantics(bounds)
			b.mergeSurfaceSemantics(semantics, parentIdx, node.Width, node.Height)
		} else {
			// Fallback: generic group node for surfaces without semantics.
			b.addNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Surface"}, parentIdx, a11y.Rect{
				Width: float64(node.Width), Height: float64(node.Height),
			})
		}

	case textElement:
		b.addNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: node.Content}, parentIdx, a11y.Rect{})

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
		idx := b.addNode(accessNode, parentIdx, a11y.Rect{})
		b.walk(node.Content, int32(idx))

	// ── Container elements: walk children under the same parent ──
	case boxElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}
	case stackElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}
	case flexElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}
	case gridElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}
	case themedElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}
	case customLayoutElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}

	// ── Wrapper elements: pass through to single child ──
	case paddingElement:
		b.walk(node.Child, parentIdx)
	case sizedBoxElement:
		if node.Child != nil {
			b.walk(node.Child, parentIdx)
		}
	case expandedElement:
		b.walk(node.Child, parentIdx)
	case scrollViewElement:
		b.walk(node.Child, parentIdx)
	case keyedElement:
		b.walk(node.Child, parentIdx)
	case blurBoxElement:
		b.walk(node.Child, parentIdx)
	case shadowBoxElement:
		b.walk(node.Child, parentIdx)
	case opacityBoxElement:
		b.walk(node.Child, parentIdx)
	case frostedGlassElement:
		b.walk(node.Child, parentIdx)
	case innerShadowBoxElement:
		b.walk(node.Child, parentIdx)
	case elevationBoxElement:
		b.walk(node.Child, parentIdx)
	case elevationCardElement:
		b.walk(node.Child, parentIdx)
	case vibrancyElement:
		b.walk(node.Child, parentIdx)
	case glowBoxElement:
		b.walk(node.Child, parentIdx)
	case cardElement:
		b.walk(node.Child, parentIdx)
	case badgeElement:
		b.walk(node.Content, parentIdx)
	case chipElement:
		b.walk(node.Label, parentIdx)
	case tooltipElement:
		// Only the trigger is relevant for a11y; tooltip content is overlay.
		b.walk(node.Trigger, parentIdx)
	case splitViewElement:
		b.walk(node.First, parentIdx)
		b.walk(node.Second, parentIdx)

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
		b.addNode(an, parentIdx, a11y.Rect{})
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
		b.addNode(an, parentIdx, a11y.Rect{})
	case sliderElement:
		an := a11y.AccessNode{
			Role:   a11y.RoleSlider,
			States: a11y.AccessStates{Disabled: node.Disabled},
			NumericValue: &a11y.AccessNumericValue{
				Current: float64(node.Value),
				Min:     0,
				Max:     1,
				Step:    0, // continuous
			},
		}
		b.addNode(an, parentIdx, a11y.Rect{})
	case progressBarElement:
		an := a11y.AccessNode{
			Role:   a11y.RoleProgressBar,
			States: a11y.AccessStates{ReadOnly: true},
		}
		if !node.Indeterminate {
			an.NumericValue = &a11y.AccessNumericValue{
				Current: float64(node.Value),
				Min:     0,
				Max:     1,
			}
		} else {
			an.States.Busy = true
		}
		b.addNode(an, parentIdx, a11y.Rect{})
	case textFieldElement:
		an := a11y.AccessNode{
			Role:   a11y.RoleTextInput,
			Label:  node.Placeholder,
			Value:  node.Value,
			States: a11y.AccessStates{Disabled: node.Disabled},
			TextState: &a11y.AccessTextState{
				Length:         len([]rune(node.Value)),
				CaretOffset:    -1,
				SelectionStart: -1,
				SelectionEnd:   -1,
			},
		}
		b.addNode(an, parentIdx, a11y.Rect{})
	case radioElement:
		b.addNode(a11y.AccessNode{
			Role:   a11y.RoleCheckbox,
			Label:  node.Label,
			States: a11y.AccessStates{Checked: node.Selected},
		}, parentIdx, a11y.Rect{})
	case selectElement:
		b.addNode(a11y.AccessNode{
			Role:  a11y.RoleCombobox,
			Value: node.Value,
		}, parentIdx, a11y.Rect{})

	// ── Data-driven elements with dynamic children ──
	case treeElement:
		treeIdx := b.addNode(a11y.AccessNode{Role: a11y.RoleTree, Label: "Tree"}, parentIdx, a11y.Rect{})
		b.walkTreeNodes(node, node.RootIDs, int32(treeIdx), 0)
	case virtualListElement:
		listIdx := b.addNode(a11y.AccessNode{Role: a11y.RoleListbox, Label: "List"}, parentIdx, a11y.Rect{})
		if node.BuildItem != nil {
			for i := 0; i < node.ItemCount; i++ {
				item := node.BuildItem(i)
				b.walk(item, int32(listIdx))
			}
		}

	default:
		// Leaf-only elements (divider, spacer, icon, gradient, etc.)
		// or unrecognized types — no a11y node needed.
	}
}

// mergeSurfaceSemantics converts SurfaceSemantics into AccessTreeNodes.
func (b *accessTreeBuilder) mergeSurfaceSemantics(sem SurfaceSemantics, parentIdx int32, w, h float32) {
	// Build an index of surface node IDs to tree indices for parent resolution.
	surfaceIDToTreeIdx := make(map[SurfaceNodeID]int32)

	// Process roots first, then children.
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
		Role:         sn.Role,
		Label:        sn.Label,
		Description:  sn.Description,
		Value:        sn.Value,
		States:       sn.States,
		NumericValue: sn.NumericValue,
		TextState:    sn.TextState,
	}
}

// walkTreeNodes recursively walks a treeElement's nodes and adds them to the access tree.
func (b *accessTreeBuilder) walkTreeNodes(tree treeElement, ids []string, parentIdx int32, depth int) {
	for _, id := range ids {
		expanded := tree.State != nil && tree.State.IsExpanded(id)
		selected := tree.State != nil && tree.State.Selected == id

		var kids []string
		if tree.Children != nil {
			kids = tree.Children(id)
		}
		hasKids := len(kids) > 0

		// Build the display element to extract a label.
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

		idx := b.addNode(an, parentIdx, a11y.Rect{})

		// Recurse into expanded children.
		if hasKids && expanded {
			b.walkTreeNodes(tree, kids, int32(idx), depth+1)
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
