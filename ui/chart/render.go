package chart

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

const (
	axisPadLeft   float32 = 50 // space for Y-axis labels
	axisPadBottom float32 = 30 // space for X-axis labels
	axisPadTop    float32 = 10 // top margin (or title space)
	axisPadRight  float32 = 10 // right margin
	titleHeight   float32 = 24 // space for chart title
	legendHeight  float32 = 20 // space for legend row
	tickLen       float32 = 4  // tick mark length
)

// plotArea describes the drawable rectangle within the chart bounds.
type plotArea struct {
	outer draw.Rect // full widget rect
	inner draw.Rect // data plotting area
}

// computePlotArea calculates the inner drawing area after axis/title reserves.
func computePlotArea(x, y float32, w, h int, title string, hasLegend bool) plotArea {
	outer := draw.R(x, y, float32(w), float32(h))
	top := axisPadTop
	if title != "" {
		top += titleHeight
	}
	bottom := axisPadBottom
	if hasLegend {
		bottom += legendHeight
	}
	inner := draw.R(
		outer.X+axisPadLeft,
		outer.Y+top,
		outer.W-axisPadLeft-axisPadRight,
		outer.H-top-bottom,
	)
	if inner.W < 1 {
		inner.W = 1
	}
	if inner.H < 1 {
		inner.H = 1
	}
	return plotArea{outer: outer, inner: inner}
}

// transform maps data coordinates to screen coordinates and back.
type transform struct {
	xMin, xMax float64
	yMin, yMax float64
	plot       draw.Rect
}

func (t transform) toScreen(dp DataPoint) draw.Point {
	xRange := t.xMax - t.xMin
	yRange := t.yMax - t.yMin
	if xRange == 0 {
		xRange = 1
	}
	if yRange == 0 {
		yRange = 1
	}
	sx := t.plot.X + float32((dp.X-t.xMin)/xRange)*t.plot.W
	sy := t.plot.Y + t.plot.H - float32((dp.Y-t.yMin)/yRange)*t.plot.H
	return draw.Pt(sx, sy)
}

func (t transform) fromScreen(p draw.Point) DataPoint {
	xRange := t.xMax - t.xMin
	yRange := t.yMax - t.yMin
	dx := float64(p.X-t.plot.X) / float64(t.plot.W) * xRange
	dy := float64(t.plot.Y+t.plot.H-p.Y) / float64(t.plot.H) * yRange
	return DataPoint{X: t.xMin + dx, Y: t.yMin + dy}
}

// resolveRange resolves the axis range: explicit bounds > viewport > auto from data.
func resolveRange(axis Axis, vp *Viewport, series []Series, which byte) (float64, float64) {
	// Explicit axis bounds take highest priority.
	mn, mx := math.NaN(), math.NaN()
	if axis.Min != nil {
		mn = *axis.Min
	}
	if axis.Max != nil {
		mx = *axis.Max
	}

	// Viewport fills in missing bounds.
	if vp != nil {
		if math.IsNaN(mn) {
			if which == 'x' {
				mn = vp.XMin
			} else {
				mn = vp.YMin
			}
		}
		if math.IsNaN(mx) {
			if which == 'x' {
				mx = vp.XMax
			} else {
				mx = vp.YMax
			}
		}
	}

	// Auto-range from data for anything still unset.
	if math.IsNaN(mn) || math.IsNaN(mx) {
		autoMin, autoMax := multiSeriesRange(series, which)
		if math.IsNaN(mn) {
			mn = autoMin
		}
		if math.IsNaN(mx) {
			mx = autoMax
		}
	}

	if mn > mx {
		mn, mx = mx, mn
	}
	if mn == mx {
		mn -= 1
		mx += 1
	}
	return mn, mx
}

// drawAxes draws axis lines, tick marks, and labels.
func drawAxes(canvas draw.Canvas, pa plotArea, t transform, xAxis, yAxis Axis, tokens theme.TokenSet) {
	axisColor := tokens.Colors.Stroke.Border
	textColor := tokens.Colors.Text.Secondary
	textStyle := tokens.Typography.LabelSmall

	axisStroke := draw.Stroke{
		Paint: draw.SolidPaint(axisColor),
		Width: 1,
	}

	// X-axis line (bottom of plot area).
	canvas.StrokeLine(
		draw.Pt(pa.inner.X, pa.inner.Y+pa.inner.H),
		draw.Pt(pa.inner.X+pa.inner.W, pa.inner.Y+pa.inner.H),
		axisStroke,
	)

	// Y-axis line (left of plot area).
	canvas.StrokeLine(
		draw.Pt(pa.inner.X, pa.inner.Y),
		draw.Pt(pa.inner.X, pa.inner.Y+pa.inner.H),
		axisStroke,
	)

	// X-axis ticks.
	xTicks := computeTicks(t.xMin, t.xMax, xAxis.TickCount)
	fmtX := xAxis.Format
	if fmtX == nil {
		fmtX = formatTick
	}
	for _, v := range xTicks {
		sp := t.toScreen(DataPoint{X: v, Y: t.yMin})
		if sp.X < pa.inner.X || sp.X > pa.inner.X+pa.inner.W {
			continue
		}
		// Tick mark.
		canvas.StrokeLine(sp, draw.Pt(sp.X, sp.Y+tickLen), axisStroke)
		// Label.
		label := fmtX(v)
		canvas.DrawText(label, draw.Pt(sp.X-10, sp.Y+tickLen+2), textStyle, textColor)
	}

	// Y-axis ticks.
	yTicks := computeTicks(t.yMin, t.yMax, yAxis.TickCount)
	fmtY := yAxis.Format
	if fmtY == nil {
		fmtY = formatTick
	}
	for _, v := range yTicks {
		sp := t.toScreen(DataPoint{X: t.xMin, Y: v})
		if sp.Y < pa.inner.Y || sp.Y > pa.inner.Y+pa.inner.H {
			continue
		}
		// Tick mark.
		canvas.StrokeLine(draw.Pt(pa.inner.X-tickLen, sp.Y), draw.Pt(pa.inner.X, sp.Y), axisStroke)
		// Label.
		label := fmtY(v)
		canvas.DrawText(label, draw.Pt(pa.inner.X-axisPadLeft+2, sp.Y-5), textStyle, textColor)
	}
}

// drawGridLines draws horizontal and vertical grid lines.
func drawGridLines(canvas draw.Canvas, pa plotArea, t transform, xAxis, yAxis Axis, tokens theme.TokenSet) {
	gridColor := tokens.Colors.Stroke.Divider
	gridStroke := draw.Stroke{
		Paint: draw.SolidPaint(gridColor),
		Width: 1,
		Dash:  []float32{4, 4},
	}

	if xAxis.GridLines {
		xTicks := computeTicks(t.xMin, t.xMax, xAxis.TickCount)
		for _, v := range xTicks {
			sp := t.toScreen(DataPoint{X: v, Y: t.yMin})
			if sp.X <= pa.inner.X || sp.X >= pa.inner.X+pa.inner.W {
				continue
			}
			canvas.StrokeLine(
				draw.Pt(sp.X, pa.inner.Y),
				draw.Pt(sp.X, pa.inner.Y+pa.inner.H),
				gridStroke,
			)
		}
	}

	if yAxis.GridLines {
		yTicks := computeTicks(t.yMin, t.yMax, yAxis.TickCount)
		for _, v := range yTicks {
			sp := t.toScreen(DataPoint{X: t.xMin, Y: v})
			if sp.Y <= pa.inner.Y || sp.Y >= pa.inner.Y+pa.inner.H {
				continue
			}
			canvas.StrokeLine(
				draw.Pt(pa.inner.X, sp.Y),
				draw.Pt(pa.inner.X+pa.inner.W, sp.Y),
				gridStroke,
			)
		}
	}
}

// drawTitle draws the chart title centered above the plot area.
func drawTitle(canvas draw.Canvas, pa plotArea, title string, tokens theme.TokenSet) {
	if title == "" {
		return
	}
	style := tokens.Typography.H3
	color := tokens.Colors.Text.Primary
	metrics := canvas.MeasureText(title, style)
	x := pa.inner.X + (pa.inner.W-metrics.Width)/2
	y := pa.outer.Y + axisPadTop
	canvas.DrawText(title, draw.Pt(x, y), style, color)
}

// drawLegend draws a compact legend below the plot area.
func drawLegend(canvas draw.Canvas, pa plotArea, series []Series, palette []draw.Color, tokens theme.TokenSet) {
	if len(series) <= 1 {
		return
	}
	style := tokens.Typography.LabelSmall
	textColor := tokens.Colors.Text.Secondary
	x := pa.inner.X
	y := pa.inner.Y + pa.inner.H + axisPadBottom + 2
	boxSize := float32(8)

	for i, s := range series {
		c := seriesColor(s, i, palette)
		// Color box.
		canvas.FillRect(draw.R(x, y, boxSize, boxSize), draw.SolidPaint(c))
		// Label text.
		name := s.Name
		if name == "" {
			name = formatInt(float64(i + 1))
		}
		canvas.DrawText(name, draw.Pt(x+boxSize+3, y-1), style, textColor)
		metrics := canvas.MeasureText(name, style)
		x += boxSize + 3 + metrics.Width + 12
	}
}

// drawBackground fills the chart area with the surface base color.
func drawBackground(canvas draw.Canvas, pa plotArea, tokens theme.TokenSet) {
	canvas.FillRect(pa.outer, draw.SolidPaint(tokens.Colors.Surface.Base))
	canvas.FillRect(pa.inner, draw.SolidPaint(tokens.Colors.Surface.Elevated))
}
