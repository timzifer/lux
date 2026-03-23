package layout

import (
	"github.com/timzifer/lux/ui"
)

// Expanded takes all available space on the main axis within a Flex parent.
// Outside of a Flex, it simply passes layout through to its child.
type Expanded struct {
	ui.BaseElement
	Child ui.Element
	Grow  float32
}

// Expand creates an Expanded element. An optional flex factor controls the
// proportion of remaining space (default 1).
func Expand(child ui.Element, flex ...float32) ui.Element {
	grow := float32(1)
	if len(flex) > 0 {
		grow = flex[0]
	}
	return Expanded{Child: child, Grow: grow}
}

// LayoutSelf implements ui.Layouter.
// Outside of a Flex container, Expanded simply lays out its child in the
// available area.
func (n Expanded) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ctx.LayoutChild(n.Child, ctx.Area)
}

// TreeEqual implements ui.TreeEqualizer.
func (n Expanded) TreeEqual(other ui.Element) bool {
	o, ok := other.(Expanded)
	return ok && n.Grow == o.Grow
}

// ResolveChildren implements ui.ChildResolver.
func (n Expanded) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	out.Child = resolve(n.Child, 0)
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Expanded) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}
