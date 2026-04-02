package chart

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// LineChartElement renders one or more data series as connected lines.
type LineChartElement struct {
	ui.BaseElement
	Config ChartConfig
	Series []Series
}

func (n *LineChartElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	w, h := chartSize(n.Config, area.W)
	palette := defaultPalette(tokens)
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
		if len(s.Points) == 0 {
			continue
		}
		c := seriesColor(s, i, palette)
		drawLineSeries(canvas, s.Points, t, c)
	}
	canvas.PopClip()

	drawLegend(canvas, pa, n.Series, palette, tokens)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// drawLineSeries draws a single line series.
func drawLineSeries(canvas draw.Canvas, points []DataPoint, t transform, color draw.Color) {
	if len(points) < 2 {
		if len(points) == 1 {
			sp := t.toScreen(points[0])
			canvas.FillEllipse(draw.R(sp.X-3, sp.Y-3, 6, 6), draw.SolidPaint(color))
		}
		return
	}

	pb := draw.NewPath()
	sp0 := t.toScreen(points[0])
	pb.MoveTo(sp0)
	for _, p := range points[1:] {
		sp := t.toScreen(p)
		pb.LineTo(sp)
	}
	path := pb.Build()

	canvas.StrokePath(path, draw.Stroke{
		Paint: draw.SolidPaint(color),
		Width: 2,
		Cap:   draw.StrokeCapRound,
		Join:  draw.StrokeJoinRound,
	})
}

func (n *LineChartElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(*LineChartElement)
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

func (n *LineChartElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n *LineChartElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleImage,
		Label: "Line chart",
	}, parentIdx, a11y.Rect{})
}
