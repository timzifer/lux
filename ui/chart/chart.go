package chart

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// ChartConfig is the common configuration for Cartesian chart types.
type ChartConfig struct {
	Width    float32 // desired width in dp; 0 = fill parent
	Height   float32 // desired height in dp; 0 = 300
	XAxis    Axis
	YAxis    Axis
	Viewport *Viewport // nil = auto-range; non-nil = controlled pan/zoom
	Title    string
}

// defaultPalette returns a set of distinguishable colors derived from theme tokens.
func defaultPalette(tokens theme.TokenSet) []draw.Color {
	return []draw.Color{
		tokens.Colors.Accent.Primary,
		tokens.Colors.Accent.Secondary,
		tokens.Colors.Status.Success,
		tokens.Colors.Status.Warning,
		tokens.Colors.Status.Error,
		tokens.Colors.Status.Info,
		// Extended palette via hue shifts.
		draw.RGBA(0x9C, 0x27, 0xB0, 0xFF), // purple
		draw.RGBA(0x00, 0x96, 0x88, 0xFF), // teal
		draw.RGBA(0xFF, 0x57, 0x22, 0xFF), // deep orange
		draw.RGBA(0x60, 0x7D, 0x8B, 0xFF), // blue-grey
	}
}

// seriesColor returns the effective color for a series at index i.
func seriesColor(s Series, i int, palette []draw.Color) draw.Color {
	if s.Color != (draw.Color{}) {
		return s.Color
	}
	if i < len(palette) {
		return palette[i]
	}
	return draw.RGBA(0x42, 0x42, 0x42, 0xFF)
}

// withAlpha returns the given color with a modified alpha.
func withAlpha(c draw.Color, a float32) draw.Color {
	return draw.Color{R: c.R, G: c.G, B: c.B, A: a}
}

// Constructors create chart elements.

// Line creates a line chart element.
func Line(cfg ChartConfig, series ...Series) *LineChartElement {
	return &LineChartElement{Config: cfg, Series: series}
}

// Area creates an area chart element.
func Area(cfg ChartConfig, series ...Series) *AreaChartElement {
	return &AreaChartElement{Config: cfg, Series: series}
}

// Bar creates a bar chart element.
func Bar(cfg ChartConfig, series ...Series) *BarChartElement {
	return &BarChartElement{Config: cfg, Series: series}
}

// Scatter creates a scatter chart element.
func Scatter(cfg ChartConfig, series ...Series) *ScatterChartElement {
	return &ScatterChartElement{Config: cfg, Series: series}
}

// Pie creates a pie chart element.
func Pie(width, height float32, slices []PieSlice) *PieChartElement {
	return &PieChartElement{PieWidth: width, PieHeight: height, Slices: slices}
}

// chartSize returns the resolved width and height for a chart config.
func chartSize(cfg ChartConfig, areaW int) (int, int) {
	w := int(cfg.Width)
	if w <= 0 {
		w = areaW
	}
	if w <= 0 {
		w = 400
	}
	h := int(cfg.Height)
	if h <= 0 {
		h = 300
	}
	return w, h
}
