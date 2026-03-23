package display

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── EmptyElement ─────────────────────────────────────────────────

// EmptyElement renders nothing.
type EmptyElement struct {
	ui.BaseElement
}

// Empty returns an Element that renders nothing.
func Empty() ui.Element { return EmptyElement{} }

// LayoutSelf implements ui.Layouter.
func (n EmptyElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}

// TreeEqual implements ui.TreeEqualizer.
func (n EmptyElement) TreeEqual(other ui.Element) bool {
	_, ok := other.(EmptyElement)
	return ok
}

// ResolveChildren implements ui.ChildResolver. EmptyElement is a leaf.
func (n EmptyElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op.
func (n EmptyElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}

// ── SpacerElement ────────────────────────────────────────────────

// SpacerElement creates invisible spacing of the given size in dp.
type SpacerElement struct {
	ui.BaseElement
	Size float32
}

// Spacer creates invisible spacing of the given size in dp.
func Spacer(size float32) ui.Element { return SpacerElement{Size: size} }

// LayoutSelf implements ui.Layouter.
func (n SpacerElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	s := int(n.Size)
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: s, H: s, Baseline: s}
}

// TreeEqual implements ui.TreeEqualizer.
func (n SpacerElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(SpacerElement)
	return ok && n.Size == nb.Size
}

// ResolveChildren implements ui.ChildResolver. SpacerElement is a leaf.
func (n SpacerElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op.
func (n SpacerElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}

// ── DividerElement ───────────────────────────────────────────────

// DividerElement renders a horizontal divider line.
type DividerElement struct {
	ui.BaseElement
}

// Divider creates a horizontal divider line.
func Divider() ui.Element { return DividerElement{} }

// LayoutSelf implements ui.Layouter.
func (n DividerElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	h := 1
	ctx.Canvas.FillRect(draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(ctx.Area.W), float32(h)),
		draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: ctx.Area.W, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n DividerElement) TreeEqual(other ui.Element) bool {
	_, ok := other.(DividerElement)
	return ok
}

// ResolveChildren implements ui.ChildResolver. DividerElement is a leaf.
func (n DividerElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op.
func (n DividerElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}

// ── GradientRectElement ──────────────────────────────────────────

// GradientRectElement renders a gradient-filled rectangle of a fixed size.
type GradientRectElement struct {
	ui.BaseElement
	Width, Height float32
	Radius        float32
	Paint         draw.Paint
}

// GradientRect renders a gradient-filled rectangle of a fixed size.
func GradientRect(width, height, radius float32, paint draw.Paint) ui.Element {
	return GradientRectElement{Width: width, Height: height, Radius: radius, Paint: paint}
}

// LayoutSelf implements ui.Layouter.
func (n GradientRectElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w := int(n.Width)
	h := int(n.Height)
	if w > ctx.Area.W {
		w = ctx.Area.W
	}
	if h > ctx.Area.H {
		h = ctx.Area.H
	}
	r := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))
	if n.Radius > 0 {
		ctx.Canvas.FillRoundRect(r, n.Radius, n.Paint)
	} else {
		ctx.Canvas.FillRect(r, n.Paint)
	}
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n GradientRectElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(GradientRectElement)
	return ok && n.Width == nb.Width && n.Height == nb.Height && n.Radius == nb.Radius && n.Paint == nb.Paint
}

// ResolveChildren implements ui.ChildResolver. GradientRectElement is a leaf.
func (n GradientRectElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op.
func (n GradientRectElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}

// ── CheckerRectElement ───────────────────────────────────────────

// CheckerRectElement renders a colorful checkerboard pattern.
type CheckerRectElement struct {
	ui.BaseElement
	Width, Height, CellSize float32
}

// CheckerRect renders a colorful checkerboard pattern of the given size.
func CheckerRect(width, height, cellSize float32) ui.Element {
	return CheckerRectElement{Width: width, Height: height, CellSize: cellSize}
}

// LayoutSelf implements ui.Layouter.
func (n CheckerRectElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w := int(n.Width)
	h := int(n.Height)
	if w > ctx.Area.W {
		w = ctx.Area.W
	}
	if h > ctx.Area.H {
		h = ctx.Area.H
	}
	cell := n.CellSize
	if cell < 1 {
		cell = 8
	}
	colors := [6]draw.Color{
		{R: 0.93, G: 0.27, B: 0.27, A: 1}, // red
		{R: 0.96, G: 0.62, B: 0.04, A: 1}, // amber
		{R: 0.13, G: 0.77, B: 0.37, A: 1}, // green
		{R: 0.23, G: 0.51, B: 0.96, A: 1}, // blue
		{R: 0.55, G: 0.36, B: 0.96, A: 1}, // violet
		{R: 0.93, G: 0.35, B: 0.60, A: 1}, // pink
	}
	for row := float32(0); row < float32(h); row += cell {
		for col := float32(0); col < float32(w); col += cell {
			ci := (int(row/cell) + int(col/cell)) % len(colors)
			cw := cell
			ch := cell
			if col+cw > float32(w) {
				cw = float32(w) - col
			}
			if row+ch > float32(h) {
				ch = float32(h) - row
			}
			ctx.Canvas.FillRect(
				draw.R(float32(ctx.Area.X)+col, float32(ctx.Area.Y)+row, cw, ch),
				draw.SolidPaint(colors[ci]),
			)
		}
	}
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n CheckerRectElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(CheckerRectElement)
	return ok && n.Width == nb.Width && n.Height == nb.Height && n.CellSize == nb.CellSize
}

// ResolveChildren implements ui.ChildResolver. CheckerRectElement is a leaf.
func (n CheckerRectElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op.
func (n CheckerRectElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
