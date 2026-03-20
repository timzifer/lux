package ui

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// BuildAccessTree constructs an AccessTree from a resolved element tree.
// It performs a depth-first walk, extracting accessibility information from
// AccessibleWidget implementations and SemanticProvider surfaces.
// The windowBounds specify the window's screen-space position for coordinate mapping.
func BuildAccessTree(root Element, reconciler *Reconciler, windowBounds a11y.Rect) a11y.AccessTree {
	b := accessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: windowBounds,
	}
	b.walk(root, -1)
	tree := a11y.AccessTree{Nodes: b.nodes}
	tree.EnsureIndex()
	return tree
}

type accessTreeBuilder struct {
	reconciler   *Reconciler
	windowBounds a11y.Rect
	nodes        []a11y.AccessTreeNode
	nextID       a11y.AccessNodeID
}

func (b *accessTreeBuilder) allocID() a11y.AccessNodeID {
	b.nextID++
	return b.nextID
}

// addNode appends a node, links siblings, and returns the node's index.
func (b *accessTreeBuilder) addNode(node a11y.AccessNode, parentIdx int32, bounds a11y.Rect) int {
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

		idx := b.addNode(accessNode, parentIdx, a11y.Rect{
			X: b.windowBounds.X, Y: b.windowBounds.Y,
			Width: b.windowBounds.Width, Height: b.windowBounds.Height,
		})
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

	case themedElement:
		for _, child := range node.Children {
			b.walk(child, parentIdx)
		}

	default:
		// Unknown/leaf elements that we don't have special handling for.
		// Just create a generic group node.
		b.addNode(a11y.AccessNode{Role: a11y.RoleGroup}, parentIdx, a11y.Rect{})
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
		Role:        sn.Role,
		Label:       sn.Label,
		Description: sn.Description,
		Value:       sn.Value,
		States:      sn.States,
	}
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
