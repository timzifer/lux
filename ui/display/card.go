package display

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Card layout constants.
const (
	cardPadding = 16
	cardBorder  = 1
)

// CardElement renders a container with elevated surface, border, and card radius.
type CardElement struct {
	ui.BaseElement
	Child ui.Element
}

// Card creates a container with elevated surface, border, and card radius.
// If multiple children are given, they are wrapped in a vertical BoxElement.
func Card(children ...ui.Element) ui.Element {
	if len(children) == 1 {
		return CardElement{Child: children[0]}
	}
	return CardElement{Child: ui.BoxElement{Axis: ui.AxisColumn, Children: children}}
}

// LayoutSelf implements ui.Layouter.
func (n CardElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Measure child to determine card size.
	childArea := ui.Bounds{
		X: ctx.Area.X + cardPadding,
		Y: ctx.Area.Y + cardPadding,
		W: max(ctx.Area.W-cardPadding*2, 0),
		H: max(ctx.Area.H-cardPadding*2, 0),
	}
	cb := ctx.MeasureChild(n.Child, childArea)

	w := cb.W + cardPadding*2
	h := cb.H + cardPadding*2
	if w > ctx.Area.W {
		w = ctx.Area.W
	}

	cardRect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))

	// Elevation shadow.
	ctx.Canvas.DrawShadow(cardRect, ctx.Tokens.Elevation.Low)

	// Fill.
	ctx.Canvas.FillRoundRect(cardRect,
		ctx.Tokens.Radii.Card, draw.SolidPaint(ctx.Tokens.Colors.Surface.Elevated))

	// Fine border.
	ctx.Canvas.StrokeRoundRect(cardRect, ctx.Tokens.Radii.Card, draw.Stroke{
		Paint: draw.SolidPaint(ctx.Tokens.Colors.Stroke.Border),
		Width: float32(cardBorder),
	})

	// Child content.
	ctx.LayoutChild(n.Child, childArea)

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
// Returns false because CardElement has a child element.
func (n CardElement) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n CardElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n CardElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}
