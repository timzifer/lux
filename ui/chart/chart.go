package chart

import "github.com/timzifer/lux/draw"

// DefaultPalette provides 10 high-contrast, perceptually distinct colors
// suitable for differentiating data series on both light and dark backgrounds.
// Inspired by the Tableau 10 categorical palette.
var DefaultPalette = []draw.Color{
	draw.Hex("#4e79a7"), // blue
	draw.Hex("#f28e2b"), // orange
	draw.Hex("#e15759"), // red
	draw.Hex("#76b7b2"), // teal
	draw.Hex("#59a14f"), // green
	draw.Hex("#edc948"), // gold
	draw.Hex("#b07aa1"), // purple
	draw.Hex("#ff9da7"), // pink
	draw.Hex("#9c755f"), // brown
	draw.Hex("#bab0ac"), // grey
}

// ChartConfig is the common configuration for Cartesian chart types.
type ChartConfig struct {
	Width    float32      // desired width in dp; 0 = fill parent
	Height   float32      // desired height in dp; 0 = 300
	XAxis    Axis
	YAxis    Axis
	Viewport *Viewport    // nil = auto-range; non-nil = controlled pan/zoom
	Title    string
	Palette  []draw.Color // custom series colors; nil = DefaultPalette
}

// resolvePalette returns the custom palette if non-empty, else DefaultPalette.
func resolvePalette(custom []draw.Color) []draw.Color {
	if len(custom) > 0 {
		return custom
	}
	return DefaultPalette
}

// seriesColor returns the effective color for a series at index i,
// cycling through the palette when there are more series than colors.
func seriesColor(s Series, i int, palette []draw.Color) draw.Color {
	if s.Color != (draw.Color{}) {
		return s.Color
	}
	return palette[i%len(palette)]
}

// sliceColor returns the effective color for a pie slice at index i.
func sliceColor(s PieSlice, i int, palette []draw.Color) draw.Color {
	if s.Color != (draw.Color{}) {
		return s.Color
	}
	return palette[i%len(palette)]
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
