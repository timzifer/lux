package chart_test

import (
	"fmt"
	"testing"

	"github.com/timzifer/lux/ui/chart"
	"github.com/timzifer/lux/uitest"
)

const (
	testW = 800
	testH = 600
)

func TestGoldenLineChart(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Line(chart.ChartConfig{Width: 400, Height: 250, Title: "Temperature"},
			chart.Series{
				Name:   "Sensor A",
				Points: []chart.DataPoint{{0, 20}, {1, 22}, {2, 19}, {3, 25}, {4, 23}, {5, 21}},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/line.golden")
}

func TestGoldenLineMultiSeries(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Line(chart.ChartConfig{Width: 400, Height: 250, Title: "Multi-Series"},
			chart.Series{
				Name:   "Temp",
				Points: []chart.DataPoint{{0, 20}, {1, 22}, {2, 19}, {3, 25}, {4, 23}},
			},
			chart.Series{
				Name:   "Humidity",
				Points: []chart.DataPoint{{0, 50}, {1, 55}, {2, 60}, {3, 52}, {4, 48}},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/line_multi_series.golden")
}

func TestGoldenAreaChart(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Area(chart.ChartConfig{Width: 400, Height: 250, Title: "Revenue"},
			chart.Series{
				Name:   "Q1",
				Points: []chart.DataPoint{{0, 100}, {1, 120}, {2, 115}, {3, 140}, {4, 135}},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/area.golden")
}

func TestGoldenBarChart(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Bar(chart.ChartConfig{Width: 400, Height: 250, Title: "Sales"},
			chart.Series{
				Name:   "Product A",
				Points: []chart.DataPoint{{0, 30}, {1, 50}, {2, 40}, {3, 60}, {4, 45}},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/bar.golden")
}

func TestGoldenBarGrouped(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Bar(chart.ChartConfig{Width: 400, Height: 250, Title: "Grouped Bars"},
			chart.Series{
				Name:   "2024",
				Points: []chart.DataPoint{{0, 30}, {1, 50}, {2, 40}},
			},
			chart.Series{
				Name:   "2025",
				Points: []chart.DataPoint{{0, 35}, {1, 55}, {2, 50}},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/bar_grouped.golden")
}

func TestGoldenScatterChart(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Scatter(chart.ChartConfig{Width: 400, Height: 250, Title: "Correlation"},
			chart.Series{
				Name: "Data",
				Points: []chart.DataPoint{
					{1, 2}, {2, 3.5}, {3, 3}, {4, 5}, {5, 4.5},
					{6, 6}, {7, 5.5}, {8, 7}, {9, 8}, {10, 7.5},
				},
			},
		),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/scatter.golden")
}

func TestGoldenPieChart(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Pie(300, 300, []chart.PieSlice{
			{Label: "A", Value: 40},
			{Label: "B", Value: 30},
			{Label: "C", Value: 20},
			{Label: "D", Value: 10},
		}),
		testW, testH,
	)
	uitest.AssertScene(t, scene, "testdata/pie.golden")
}

// TestPiePathBatchColors verifies that each pie slice produces a
// DrawPathBatch with a distinct color from the palette.
func TestPiePathBatchColors(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Pie(300, 300, []chart.PieSlice{
			{Label: "A", Value: 40},
			{Label: "B", Value: 30},
			{Label: "C", Value: 20},
			{Label: "D", Value: 10},
		}),
		testW, testH,
	)
	// Each slice produces 2 path batches: fill + stroke border.
	// So 4 slices → 8 batches.
	if len(scene.PathBatches) < 4 {
		t.Fatalf("expected ≥4 PathBatches for 4 slices, got %d", len(scene.PathBatches))
	}
	// Collect fill colors (even indices: 0, 2, 4, 6).
	var fills []string
	for i := 0; i < len(scene.PathBatches); i += 2 {
		c := scene.PathBatches[i].Color
		key := fmt.Sprintf("%.2f,%.2f,%.2f", c.R, c.G, c.B)
		fills = append(fills, key)
	}
	// All fill colors must be distinct.
	seen := map[string]bool{}
	for _, f := range fills {
		if seen[f] {
			t.Errorf("duplicate fill color %s — slices should have distinct palette colors", f)
		}
		seen[f] = true
	}
	t.Logf("pie fill colors: %v", fills)
}

// TestLineMultiSeriesPathColors verifies that each line series
// produces a DrawPathBatch with a distinct stroke color.
func TestLineMultiSeriesPathColors(t *testing.T) {
	scene := uitest.BuildScene(
		chart.Line(chart.ChartConfig{Width: 400, Height: 250},
			chart.Series{
				Name:   "A",
				Points: []chart.DataPoint{{0, 10}, {1, 20}, {2, 15}},
			},
			chart.Series{
				Name:   "B",
				Points: []chart.DataPoint{{0, 30}, {1, 25}, {2, 35}},
			},
		),
		testW, testH,
	)
	if len(scene.PathBatches) < 2 {
		t.Fatalf("expected ≥2 PathBatches for 2 series, got %d", len(scene.PathBatches))
	}
	c0 := scene.PathBatches[0].Color
	c1 := scene.PathBatches[1].Color
	if c0 == c1 {
		t.Errorf("both line series have same color %v — should be different palette entries", c0)
	}
	t.Logf("line series colors: [0]=%v [1]=%v", c0, c1)
}
