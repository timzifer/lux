package chart

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ScatterChartElement renders one or more data series as individual points.
type ScatterChartElement struct {
	ui.BaseElement
	Config ChartConfig
	Series []Series
}

const scatterPointRadius float32 = 4

func (n *ScatterChartElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	w, h := chartSize(n.Config, area.W)
	palette := resolvePalette(n.Config.Palette)
	hasLegend := len(n.Series) > 1
	pa := computePlotArea(float32(area.X), float32(area.Y), w, h, n.Config.Title, hasLegend)

	xMin, xMax := resolveRange(n.Config.XAxis, n.Config.Viewport, n.Series, 'x')
	yMin, yMax := resolveRange(n.Config.YAxis, n.Config.Viewport, n.Series, 'y')
	t := transform{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax, plot: pa.inner}

	drawBackground(canvas, pa, tokens)
	drawGridLines(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawAxes(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawTitle(canvas, pa, n.Config.Title, tokens)

	canvas.PushClip(pa.inner)
	for i, s := range n.Series {
		c := seriesColor(s, i, palette)
		drawScatterPoints(canvas, s.Points, t, c)
	}
	canvas.PopClip()

	drawLegend(canvas, pa, n.Series, palette, tokens)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func drawScatterPoints(canvas draw.Canvas, points []DataPoint, t transform, color draw.Color) {
	r := scatterPointRadius
	for _, p := range points {
		sp := t.toScreen(p)
		canvas.FillEllipse(
			draw.R(sp.X-r, sp.Y-r, r*2, r*2),
			draw.SolidPaint(color),
		)
	}
}

func (n *ScatterChartElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(*ScatterChartElement)
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

func (n *ScatterChartElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n *ScatterChartElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleImage,
		Label: "Scatter chart",
	}, parentIdx, a11y.Rect{})
}
