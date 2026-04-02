package chart

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// PieChartElement renders a pie chart from slices.
type PieChartElement struct {
	ui.BaseElement
	PieWidth  float32
	PieHeight float32
	Slices    []PieSlice
	Palette   []draw.Color // custom slice colors; nil = DefaultPalette
}

func (n *PieChartElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	w := int(n.PieWidth)
	if w <= 0 {
		w = 300
	}
	h := int(n.PieHeight)
	if h <= 0 {
		h = 300
	}

	palette := resolvePalette(n.Palette)

	// Background.
	outer := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	canvas.FillRect(outer, draw.SolidPaint(tokens.Colors.Surface.Base))

	// Calculate total.
	total := 0.0
	for _, s := range n.Slices {
		if s.Value > 0 {
			total += s.Value
		}
	}
	if total == 0 {
		return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
	}

	// Pie geometry.
	cx := float32(area.X) + float32(w)/2
	cy := float32(area.Y) + float32(h)/2
	radius := float32(w) / 2
	if float32(h)/2 < radius {
		radius = float32(h) / 2
	}
	radius -= 20 // padding for labels

	startAngle := -math.Pi / 2 // start at 12 o'clock
	for i, s := range n.Slices {
		if s.Value <= 0 {
			continue
		}
		fraction := s.Value / total
		sweepAngle := fraction * 2 * math.Pi

		c := sliceColor(s, i, palette)

		drawPieSlice(canvas, cx, cy, radius, float32(startAngle), float32(sweepAngle), c)

		// Label at midpoint of slice arc.
		if s.Label != "" {
			midAngle := startAngle + sweepAngle/2
			labelR := radius * 0.65
			lx := cx + float32(math.Cos(midAngle))*labelR
			ly := cy + float32(math.Sin(midAngle))*labelR
			style := tokens.Typography.LabelSmall
			canvas.DrawText(s.Label, draw.Pt(lx-10, ly-5), style, tokens.Colors.Text.Primary)
		}

		startAngle += sweepAngle
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// drawPieSlice draws a single filled wedge.
func drawPieSlice(canvas draw.Canvas, cx, cy, radius, startAngle, sweepAngle float32, color draw.Color) {
	if sweepAngle <= 0 {
		return
	}

	const segments = 64
	pb := draw.NewPath()
	pb.MoveTo(draw.Pt(cx, cy))

	endAngle := startAngle + sweepAngle
	step := sweepAngle / segments

	for i := 0; i <= segments; i++ {
		a := startAngle + float32(i)*step
		if a > endAngle {
			a = endAngle
		}
		x := cx + radius*float32(math.Cos(float64(a)))
		y := cy + radius*float32(math.Sin(float64(a)))
		pb.LineTo(draw.Pt(x, y))
	}
	pb.Close()
	path := pb.Build()

	canvas.FillPath(path, draw.SolidPaint(color))

	// Slice border.
	canvas.StrokePath(path, draw.Stroke{
		Paint: draw.SolidPaint(draw.Color{R: 1, G: 1, B: 1, A: 0.8}),
		Width: 1,
	})
}

func (n *PieChartElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(*PieChartElement)
	if !ok || len(n.Slices) != len(o.Slices) {
		return false
	}
	for i := range n.Slices {
		if n.Slices[i].Value != o.Slices[i].Value {
			return false
		}
	}
	return true
}

func (n *PieChartElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n *PieChartElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleImage,
		Label: "Pie chart",
	}, parentIdx, a11y.Rect{})
}
