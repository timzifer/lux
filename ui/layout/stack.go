package layout

import (
	"github.com/timzifer/lux/ui"
)

// Stack overlays children on top of each other (z-axis).
// The first child is the bottom layer, the last child is the top layer.
type Stack struct {
	ui.BaseElement
	Children []ui.Element
}

// NewStack creates a Stack element.
func NewStack(children ...ui.Element) ui.Element {
	return Stack{Children: children}
}

// LayoutSelf implements ui.Layouter.
func (n Stack) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	maxW := 0
	maxH := 0
	firstBaseline := 0
	for i, child := range n.Children {
		childBounds := ctx.LayoutChild(child, area)
		if childBounds.W > maxW {
			maxW = childBounds.W
		}
		if childBounds.H > maxH {
			maxH = childBounds.H
		}
		if i == 0 {
			firstBaseline = childBounds.Baseline
		}
	}
	if maxW == 0 && maxH == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}
	if firstBaseline == 0 {
		firstBaseline = maxH
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: firstBaseline}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Stack) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n Stack) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Stack) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}
