// Charts — static chart gallery showcasing all chart types.
//
//	go run ./examples/charts/
package main

import (
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/chart"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
)

type Model struct {
	Scroll *ui.ScrollState
}

func update(m Model, msg app.Msg) Model {
	return m
}

func view(m Model) ui.Element {
	return nav.NewScrollView(
		layout.Pad(draw.Insets{Top: 24, Right: 24, Bottom: 24, Left: 24},
			layout.Column(
				display.Text("Charting Widget Gallery"),
				display.Divider(),

				// ── Line Chart ──
				display.Text("Line Chart — Temperature Sensors"),
				chart.Line(chart.ChartConfig{
					Width: 600, Height: 280,
					Title: "Temperature (°C)",
					XAxis: chart.Axis{Label: "Time (s)", GridLines: true},
					YAxis: chart.Axis{Label: "°C", GridLines: true},
				},
					chart.Series{
						Name:   "Sensor A",
						Points: sensorA(),
					},
					chart.Series{
						Name:   "Sensor B",
						Points: sensorB(),
					},
				),

				display.Divider(),

				// ── Area Chart ──
				display.Text("Area Chart — Throughput"),
				chart.Area(chart.ChartConfig{
					Width: 600, Height: 250,
					Title: "Throughput (MB/s)",
					XAxis: chart.Axis{GridLines: true},
					YAxis: chart.Axis{GridLines: true},
				},
					chart.Series{
						Name:   "Inbound",
						Points: throughputIn(),
					},
				),

				display.Divider(),

				// ── Bar Chart (grouped) ──
				display.Text("Bar Chart — Monthly Production"),
				chart.Bar(chart.ChartConfig{
					Width: 600, Height: 250,
					Title: "Production Units",
					YAxis: chart.Axis{GridLines: true},
				},
					chart.Series{
						Name:   "Line 1",
						Points: production1(),
					},
					chart.Series{
						Name:   "Line 2",
						Points: production2(),
					},
				),

				display.Divider(),

				// ── Scatter Chart ──
				display.Text("Scatter Chart — Quality vs Speed"),
				chart.Scatter(chart.ChartConfig{
					Width: 600, Height: 250,
					Title: "Quality vs Speed",
					XAxis: chart.Axis{Label: "Speed (rpm)", GridLines: true},
					YAxis: chart.Axis{Label: "Quality (%)", GridLines: true},
				},
					chart.Series{
						Name:   "Batch A",
						Points: scatterDataA(),
					},
					chart.Series{
						Name:   "Batch B",
						Points: scatterDataB(),
					},
				),

				display.Divider(),

				// ── Pie Chart ──
				display.Text("Pie Chart — Error Distribution"),
				chart.Pie(350, 350, []chart.PieSlice{
					{Label: "Timeout", Value: 42},
					{Label: "CRC", Value: 28},
					{Label: "Overflow", Value: 15},
					{Label: "Parity", Value: 10},
					{Label: "Other", Value: 5},
				}),
			),
		), 800, m.Scroll,
	)
}

func main() {
	if err := app.Run(Model{Scroll: &ui.ScrollState{}}, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Chart Gallery"),
	); err != nil {
		log.Fatal(err)
	}
}

// ── Sample Data ─────────────────────────────────────────────────

func sensorA() []chart.DataPoint {
	return []chart.DataPoint{
		{0, 22.1}, {1, 22.4}, {2, 23.0}, {3, 23.8}, {4, 24.2},
		{5, 24.5}, {6, 25.0}, {7, 24.8}, {8, 24.3}, {9, 23.9},
		{10, 23.5}, {11, 23.0}, {12, 22.5}, {13, 22.8}, {14, 23.2},
		{15, 23.7}, {16, 24.1}, {17, 24.6}, {18, 25.1}, {19, 24.9},
	}
}

func sensorB() []chart.DataPoint {
	return []chart.DataPoint{
		{0, 21.0}, {1, 21.3}, {2, 21.8}, {3, 22.5}, {4, 23.0},
		{5, 23.4}, {6, 23.9}, {7, 24.2}, {8, 24.0}, {9, 23.5},
		{10, 23.0}, {11, 22.4}, {12, 21.8}, {13, 21.5}, {14, 22.0},
		{15, 22.6}, {16, 23.1}, {17, 23.7}, {18, 24.3}, {19, 24.0},
	}
}

func throughputIn() []chart.DataPoint {
	return []chart.DataPoint{
		{0, 120}, {1, 135}, {2, 128}, {3, 142}, {4, 155},
		{5, 148}, {6, 162}, {7, 170}, {8, 165}, {9, 158},
		{10, 175}, {11, 180}, {12, 172}, {13, 168}, {14, 185},
	}
}

func production1() []chart.DataPoint {
	return []chart.DataPoint{
		{0, 450}, {1, 520}, {2, 480}, {3, 550}, {4, 610},
		{5, 580}, {6, 620}, {7, 590}, {8, 640}, {9, 670},
		{10, 650}, {11, 700},
	}
}

func production2() []chart.DataPoint {
	return []chart.DataPoint{
		{0, 380}, {1, 420}, {2, 400}, {3, 460}, {4, 500},
		{5, 470}, {6, 510}, {7, 490}, {8, 530}, {9, 560},
		{10, 540}, {11, 580},
	}
}

func scatterDataA() []chart.DataPoint {
	return []chart.DataPoint{
		{100, 92}, {200, 88}, {300, 85}, {400, 80}, {500, 75},
		{150, 90}, {250, 87}, {350, 83}, {450, 78}, {550, 72},
		{120, 91}, {220, 89}, {320, 84}, {420, 79}, {520, 74},
	}
}

func scatterDataB() []chart.DataPoint {
	return []chart.DataPoint{
		{100, 95}, {200, 93}, {300, 90}, {400, 86}, {500, 82},
		{150, 94}, {250, 92}, {350, 88}, {450, 84}, {550, 80},
		{120, 94}, {220, 91}, {320, 89}, {420, 85}, {520, 81},
	}
}
