package chart

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// BarChartElement renders one or more data series as vertical bars.
type BarChartElement struct {
	ui.BaseElement
	Config ChartConfig
	Series []Series
}

func (n *BarChartElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	w, h := chartSize(n.Config, area.W)
	palette := resolvePalette(n.Config.Palette)
	hasLegend := len(n.Series) > 1
	pa := computePlotArea(float32(area.X), float32(area.Y), w, h, n.Config.Title, hasLegend)

	xMin, xMax := resolveRange(n.Config.XAxis, n.Config.Viewport, n.Series, 'x')
	yMin, yMax := resolveRange(n.Config.YAxis, n.Config.Viewport, n.Series, 'y')
	// Bars should start from zero on Y-axis when auto-ranging.
	if n.Config.YAxis.Min == nil && (n.Config.Viewport == nil) && yMin > 0 {
		yMin = 0
	}
	t := transform{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax, plot: pa.inner}

	drawBackground(canvas, pa, tokens)
	drawGridLines(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawAxes(canvas, pa, t, n.Config.XAxis, n.Config.YAxis, tokens)
	drawTitle(canvas, pa, n.Config.Title, tokens)

	canvas.PushClip(pa.inner)
	drawBars(canvas, n.Series, t, palette)
	canvas.PopClip()

	drawLegend(canvas, pa, n.Series, palette, tokens)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// drawBars draws grouped vertical bars for all series.
func drawBars(canvas draw.Canvas, series []Series, t transform, palette []draw.Color) {
	if len(series) == 0 {
		return
	}

	// Find the number of unique X positions across all series.
	nPoints := 0
	for _, s := range series {
		if len(s.Points) > nPoints {
			nPoints = len(s.Points)
		}
	}
	if nPoints == 0 {
		return
	}

	nSeries := len(series)
	xRange := t.xMax - t.xMin
	if xRange == 0 {
		xRange = 1
	}

	// Bar geometry.
	groupW := t.plot.W / float32(nPoints) * 0.8
	barW := groupW / float32(nSeries)
	if barW < 2 {
		barW = 2
	}
	gap := t.plot.W/float32(nPoints) - groupW

	baseY := t.toScreen(DataPoint{X: 0, Y: 0}).Y
	// Clamp baseline to plot area.
	if baseY < t.plot.Y {
		baseY = t.plot.Y
	}
	if baseY > t.plot.Y+t.plot.H {
		baseY = t.plot.Y + t.plot.H
	}

	for si, s := range series {
		c := seriesColor(s, si, palette)
		for pi, p := range s.Points {
			sp := t.toScreen(p)
			x := t.plot.X + gap/2 + float32(pi)*(groupW+gap) + float32(si)*barW
			var barH float32
			var barY float32
			if sp.Y < baseY {
				barY = sp.Y
				barH = baseY - sp.Y
			} else {
				barY = baseY
				barH = sp.Y - baseY
			}
			if barH < 1 {
				barH = 1
			}
			canvas.FillRoundRect(
				draw.R(x, barY, barW, barH),
				2, draw.SolidPaint(c),
			)
		}
	}
}

func (n *BarChartElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(*BarChartElement)
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

func (n *BarChartElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n *BarChartElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleImage,
		Label: "Bar chart",
	}, parentIdx, a11y.Rect{})
}
