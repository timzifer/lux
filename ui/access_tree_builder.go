package ui

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// BuildAccessTree constructs an AccessTree from a resolved element tree.
// It performs a depth-first Walk, extracting accessibility information from
// AccessibleWidget implementations and SemanticProvider surfaces.
// The windowBounds specify the window's screen-space position for coordinate mapping.
// The dispatcher provides per-widget screen bounds from the layout pass.
//
// The tree always starts with a synthetic root node at index 0 (RoleGroup)
// that wraps all top-level elements. The UIA root provider maps to this node.
func BuildAccessTree(root Element, reconciler *Reconciler, windowBounds a11y.Rect, dispatcher ...*EventDispatcher) a11y.AccessTree {
	b := AccessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: windowBounds,
	}
	if len(dispatcher) > 0 {
		b.dispatcher = dispatcher[0]
	}
	// Create synthetic root node at index 0. All elements become its children.
	rootIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, windowBounds)
	b.Walk(root, int32(rootIdx))
	tree := a11y.AccessTree{Nodes: b.nodes}
	tree.EnsureIndex()
	return tree
}

type AccessTreeBuilder struct {
	reconciler   *Reconciler
	dispatcher   *EventDispatcher
	windowBounds a11y.Rect
	nodes        []a11y.AccessTreeNode
	nextID       a11y.AccessNodeID

	// ActiveTrapID, when non-empty, causes nodes outside the matching
	// Overlay subtree to be excluded from the access tree (RFC-001 §11.7).
	// Content outside a modal dialog is effectively aria-hidden.
	ActiveTrapID string
	insideTrap   bool // true while walking inside the trap's overlay
}

// BoundsForWidget returns the screen-space bounds for a widget.
// Falls back to windowBounds if per-widget bounds aren't available.
func (b *AccessTreeBuilder) BoundsForWidget(uid UID) a11y.Rect {
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

func (b *AccessTreeBuilder) AllocID() a11y.AccessNodeID {
	b.nextID++
	return b.nextID
}

// AddNode appends a node, links siblings, and returns the node's index.
// If bounds are zero-sized, falls back to windowBounds so elements are
// visible to the accessibility system.
func (b *AccessTreeBuilder) AddNode(node a11y.AccessNode, parentIdx int32, bounds a11y.Rect) int {
	if bounds.Width == 0 && bounds.Height == 0 {
		bounds = b.windowBounds
	}
	idx := int32(len(b.nodes))
	id := b.AllocID()

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

func (b *AccessTreeBuilder) Walk(el Element, parentIdx int32) {
	// When a FocusTrap is active, skip non-overlay content outside the trap.
	// This effectively marks background content as aria-hidden (RFC-001 §11.7).
	if b.ActiveTrapID != "" && !b.insideTrap {
		if _, isOverlay := el.(Overlay); !isOverlay {
			// Allow container elements through so we can find nested overlays.
			if cr, ok := el.(ChildResolver); ok {
				cr.ResolveChildren(func(child Element, _ int) Element {
					b.Walk(child, parentIdx)
					return child
				})
				return
			}
			return // Skip non-container content outside the trap.
		}
	}

	// Interface-based dispatch for sub-package element types.
	if aw, ok := el.(AccessWalker); ok {
		// When trap is active and we're outside it, skip sub-package elements.
		if b.ActiveTrapID != "" && !b.insideTrap {
			return
		}
		aw.WalkAccess(b, parentIdx)
		return
	}
	switch node := el.(type) {
	case nil:
		return

	case WidgetBoundsElement:
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
		bounds := b.BoundsForWidget(node.WidgetUID)

		idx := b.AddNode(accessNode, parentIdx, bounds)
		b.Walk(node.Child, int32(idx))

	case SurfaceElement:
		// Check if the surface's provider implements SemanticProvider.
		if sp, ok := node.Provider.(SemanticProvider); ok {
			bounds := draw.R(0, 0, float32(node.Width), float32(node.Height))
			semantics := sp.SnapshotSemantics(bounds)
			b.MergeSurfaceSemantics(semantics, parentIdx, node.Width, node.Height)
		} else {
			// Fallback: generic group node for surfaces without semantics.
			b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Surface"}, parentIdx, a11y.Rect{
				Width: float64(node.Width), Height: float64(node.Height),
			})
		}

	case ThemedElement:
		for _, child := range node.Children {
			b.Walk(child, parentIdx)
		}
	case CustomLayoutElement:
		for _, child := range node.Children {
			b.Walk(child, parentIdx)
		}

	case KeyedElement:
		b.Walk(node.Child, parentIdx)

	// ── Overlay elements with focus trap support (RFC-001 §11.7) ──
	case Overlay:
		// Modal overlays with backdrop get a RoleDialog node.
		// When a FocusTrap is active, the overlay content is the only
		// accessible content; everything else is excluded.
		role := a11y.RoleGroup
		if node.Backdrop {
			role = a11y.RoleDialog
		}
		overlayNode := a11y.AccessNode{Role: role, Label: string(node.ID)}
		idx := b.AddNode(overlayNode, parentIdx, b.windowBounds)

		// Track whether we're inside the active trap.
		if b.ActiveTrapID != "" && string(node.ID) == b.ActiveTrapID {
			prev := b.insideTrap
			b.insideTrap = true
			b.Walk(node.Content, int32(idx))
			b.insideTrap = prev
		} else {
			b.Walk(node.Content, int32(idx))
		}

	default:
		// Leaf-only elements (divider, spacer, icon, gradient, etc.)
		// or unrecognized types — no a11y node needed.
	}
}

// MergeSurfaceSemantics converts SurfaceSemantics into AccessTreeNodes.
func (b *AccessTreeBuilder) MergeSurfaceSemantics(sem SurfaceSemantics, parentIdx int32, w, h float32) {
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
		idx := b.AddNode(node, p, treeBounds)
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

// TreeAccessor is an interface for tree-like elements that support
// accessibility tree walking. Implemented by data.Tree.
type TreeAccessor interface {
	TreeState() *TreeState
	TreeChildren(id string) []string
	TreeBuildNode(id string, depth int, expanded, selected bool) Element
	TreeOnSelect() func(string)
}

// WalkTreeNodes recursively walks a tree's nodes and adds them to the access tree.
func (b *AccessTreeBuilder) WalkTreeNodes(tree TreeAccessor, ids []string, parentIdx int32, depth int) {
	state := tree.TreeState()
	for _, id := range ids {
		expanded := state != nil && state.IsExpanded(id)
		selected := state != nil && state.Selected == id

		kids := tree.TreeChildren(id)
		hasKids := len(kids) > 0

		// Build the display element to extract a label.
		var label string
		nodeEl := tree.TreeBuildNode(id, depth, expanded, selected)
		if nodeEl != nil {
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
		if onSelect := tree.TreeOnSelect(); onSelect != nil {
			selectID := id
			an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { onSelect(selectID) }}}
		}

		idx := b.AddNode(an, parentIdx, a11y.Rect{})

		// Recurse into expanded children.
		if hasKids && expanded {
			b.WalkTreeNodes(tree, kids, int32(idx), depth+1)
		}
	}
}

// ExtractElementLabel tries to extract a text label from an arbitrary element.
// Exported for use by sub-packages that need to build a11y tree nodes.
func ExtractElementLabel(el Element) string {
	return extractElementLabel(el)
}

// extractElementLabel tries to extract a text label from an arbitrary element.
func extractElementLabel(el Element) string {
	if l, ok := el.(Labeler); ok {
		return l.ElementLabel()
	}
	switch node := el.(type) {
	case WidgetBoundsElement:
		return extractElementLabel(node.Child)
	case KeyedElement:
		return extractElementLabel(node.Child)
	}
	if cr, ok := el.(ChildResolver); ok {
		var label string
		cr.ResolveChildren(func(child Element, _ int) Element {
			if label == "" {
				label = extractElementLabel(child)
			}
			return child
		})
		return label
	}
	return ""
}

// extractButtonLabel tries to get a text label from a button's content element.
func extractButtonLabel(el Element) string {
	if l, ok := el.(Labeler); ok {
		return l.ElementLabel()
	}
	if cr, ok := el.(ChildResolver); ok {
		var label string
		cr.ResolveChildren(func(child Element, _ int) Element {
			if label == "" {
				label = extractButtonLabel(child)
			}
			return child
		})
		return label
	}
	return ""
}
