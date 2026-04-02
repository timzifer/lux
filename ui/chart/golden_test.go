package chart_test

import (
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
