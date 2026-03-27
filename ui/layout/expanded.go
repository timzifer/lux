package layout

import (
	"github.com/timzifer/lux/ui"
)

// Expanded takes all available space on the main axis within a Flex parent.
// Outside of a Flex, it simply passes layout through to its child.
// It carries per-child flex properties: Grow, Shrink, Basis, AlignSelf, Order.
type Expanded struct {
	ui.BaseElement
	Child     ui.Element
	Grow      float32   // flex-grow factor (default 1 when created via Expand)
	Shrink    float32   // flex-shrink factor (default 1)
	Basis     FlexBasis // flex-basis (default Auto)
	AlignSelf AlignSelf // align-self override (default Auto = inherit)
	Order     int       // visual order (default 0, lower values first)
}

// ExpandOption configures an Expanded element.
type ExpandOption func(*Expanded)

// Expand creates an Expanded element. An optional flex factor controls the
// proportion of remaining space (default 1).
func Expand(child ui.Element, flex ...float32) ui.Element {
	grow := float32(1)
	if len(flex) > 0 {
		grow = flex[0]
	}
	return Expanded{Child: child, Grow: grow, Shrink: 1, Basis: FlexBasis{Auto: true}}
}

// FlexChild creates an Expanded element with full control over flex properties.
func FlexChild(child ui.Element, opts ...ExpandOption) ui.Element {
	e := Expanded{Child: child, Shrink: 1, Basis: FlexBasis{Auto: true}}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithGrow sets the flex-grow factor.
func WithGrow(g float32) ExpandOption {
	return func(e *Expanded) { e.Grow = g }
}

// WithShrink sets the flex-shrink factor.
func WithShrink(s float32) ExpandOption {
	return func(e *Expanded) { e.Shrink = s }
}

// WithBasis sets the flex-basis.
func WithBasis(b FlexBasis) ExpandOption {
	return func(e *Expanded) { e.Basis = b }
}

// WithAlignSelf sets the per-child cross-axis alignment.
func WithAlignSelf(a AlignSelf) ExpandOption {
	return func(e *Expanded) { e.AlignSelf = a }
}

// WithOrder sets the visual order.
func WithOrder(o int) ExpandOption {
	return func(e *Expanded) { e.Order = o }
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
	return ok && n.Grow == o.Grow && n.Shrink == o.Shrink &&
		n.Basis == o.Basis && n.AlignSelf == o.AlignSelf && n.Order == o.Order
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
