package layout

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Padding adds inner spacing around a single child.
type Padding struct {
	ui.BaseElement
	Insets draw.Insets
	Child  ui.Element
}

// Pad creates a Padding element.
func Pad(insets draw.Insets, child ui.Element) ui.Element {
	return Padding{Insets: insets, Child: child}
}

// LayoutSelf implements ui.Layouter.
func (n Padding) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	// Resolve logical Start/End insets to physical Left/Right.
	left, right := n.Insets.Resolve(ui.Direction())
	inL := int(left)
	inT := int(n.Insets.Top)
	inR := int(right)
	inB := int(n.Insets.Bottom)
	childArea := ui.Bounds{
		X: area.X + inL,
		Y: area.Y + inT,
		W: max(area.W-inL-inR, 0),
		H: max(area.H-inT-inB, 0),
	}
	cb := ctx.LayoutChild(n.Child, childArea)
	return ui.Bounds{X: area.X, Y: area.Y, W: cb.W + inL + inR, H: cb.H + inT + inB, Baseline: inT + cb.Baseline}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Padding) TreeEqual(other ui.Element) bool {
	o, ok := other.(Padding)
	return ok && n.Insets == o.Insets
}

// ResolveChildren implements ui.ChildResolver.
func (n Padding) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	out.Child = resolve(n.Child, 0)
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Padding) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// SizedBox enforces a specific size on a child. If Child is nil,
// it acts as an empty spacer with the given dimensions.
type SizedBox struct {
	ui.BaseElement
	Width, Height float32
	Child         ui.Element // nil = empty spacer
}

// Sized creates a SizedBox element. If a child is provided, it is
// constrained to the given dimensions.
func Sized(width, height float32, child ...ui.Element) ui.Element {
	var c ui.Element
	if len(child) > 0 {
		c = child[0]
	}
	return SizedBox{Width: width, Height: height, Child: c}
}

// LayoutSelf implements ui.Layouter.
func (n SizedBox) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	w := int(n.Width)
	h := int(n.Height)
	// Zero means "inherit from parent area" so callers can constrain
	// only one dimension (e.g. Sized(0, 120, child) for height-only).
	if w == 0 {
		w = area.W
	}
	if h == 0 {
		h = area.H
	}
	var baseline int
	if n.Child != nil {
		childArea := ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
		cb := ctx.LayoutChild(n.Child, childArea)
		baseline = cb.Baseline
	}
	if baseline == 0 {
		baseline = h
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: baseline}
}

// TreeEqual implements ui.TreeEqualizer.
func (n SizedBox) TreeEqual(other ui.Element) bool {
	o, ok := other.(SizedBox)
	return ok && n.Width == o.Width && n.Height == o.Height
}

// ResolveChildren implements ui.ChildResolver.
func (n SizedBox) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	if n.Child == nil {
		return n
	}
	out := n
	out.Child = resolve(n.Child, 0)
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n SizedBox) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}
