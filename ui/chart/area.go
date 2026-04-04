package chart

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// AreaChartElement renders one or more data series as filled areas.
type AreaChartElement struct {
	ui.BaseElement
	Config ChartConfig
	Series []Series
}

func (n *AreaChartElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	w, h := chartSize(n.Config, area.W)
	palette := resolvePalette(n.Config.Palette)
	hasLegend := len(n.Series) > 1
	pa := computePlotArea(float32(area.X), float32(area.Y), w, h, n.Config.Title, hasLegend)

	xMin, xMax := resolveRange(n.Config.XAxis, n.Config.Viewport, n.Series, 'x')
	yMin, yMax := resolveRange(n.Config.YAxis, n.Config.Viewport, n.Series, 'y')
	// Area fill should start from zero on Y-axis when auto-ranging.
	if n.Config.YAxis.Min == nil && n.Config.Viewport == nil && yMin > 0 {
		yMin = 0
	}
	t := transform{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax, plot: pa.inner}

	drawBackground(canvas, pa, tokens)
	drawGridLines(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawAxes(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawTitle(canvas, pa, n.Config.Title, tokens)

	canvas.PushClip(pa.inner)
	for i, s := range n.Series {
		if len(s.Points) == 0 {
			continue
		}
		c := seriesColor(s, i, palette)
		drawAreaSeries(canvas, s.Points, t, c)
	}
	canvas.PopClip()

	drawLegend(canvas, pa, n.Series, palette, tokens)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// drawAreaSeries draws a filled area + line for a single series.
func drawAreaSeries(canvas draw.Canvas, points []DataPoint, t transform, color draw.Color) {
	if len(points) < 2 {
		return
	}

	// Build filled area path: top edge → baseline.
	pb := draw.NewPath()
	baseY := t.plot.Y + t.plot.H // bottom of plot = Y-axis baseline

	sp0 := t.toScreen(points[0])
	pb.MoveTo(draw.Pt(sp0.X, baseY))
	pb.LineTo(sp0)
	for _, p := range points[1:] {
		sp := t.toScreen(p)
		pb.LineTo(sp)
	}
	spLast := t.toScreen(points[len(points)-1])
	pb.LineTo(draw.Pt(spLast.X, baseY))
	pb.Close()
	fillPath := pb.Build()

	canvas.FillPath(fillPath, draw.SolidPaint(withAlpha(color, 0.25)))

	// Draw top edge as a line.
	drawLineSeries(canvas, points, t, color)
}

func (n *AreaChartElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(*AreaChartElement)
	if !ok || len(n.Series) != len(o.Series) {
		return false
	}
	for i := range n.Series {
		if len(n.Series[i].Points) != len(o.Series[i].Points) {
			return false
		}
	}
	return true
}

func (n *AreaChartElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n *AreaChartElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleImage,
		Label: "Area chart",
	}, parentIdx, a11y.Rect{})
}
