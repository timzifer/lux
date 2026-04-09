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
		{X: 0, Y: 22.1}, {X: 1, Y: 22.4}, {X: 2, Y: 23.0}, {X: 3, Y: 23.8}, {X: 4, Y: 24.2},
		{X: 5, Y: 24.5}, {X: 6, Y: 25.0}, {X: 7, Y: 24.8}, {X: 8, Y: 24.3}, {X: 9, Y: 23.9},
		{X: 10, Y: 23.5}, {X: 11, Y: 23.0}, {X: 12, Y: 22.5}, {X: 13, Y: 22.8}, {X: 14, Y: 23.2},
		{X: 15, Y: 23.7}, {X: 16, Y: 24.1}, {X: 17, Y: 24.6}, {X: 18, Y: 25.1}, {X: 19, Y: 24.9},
	}
}

func sensorB() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 0, Y: 21.0}, {X: 1, Y: 21.3}, {X: 2, Y: 21.8}, {X: 3, Y: 22.5}, {X: 4, Y: 23.0},
		{X: 5, Y: 23.4}, {X: 6, Y: 23.9}, {X: 7, Y: 24.2}, {X: 8, Y: 24.0}, {X: 9, Y: 23.5},
		{X: 10, Y: 23.0}, {X: 11, Y: 22.4}, {X: 12, Y: 21.8}, {X: 13, Y: 21.5}, {X: 14, Y: 22.0},
		{X: 15, Y: 22.6}, {X: 16, Y: 23.1}, {X: 17, Y: 23.7}, {X: 18, Y: 24.3}, {X: 19, Y: 24.0},
	}
}

func throughputIn() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 0, Y: 120}, {X: 1, Y: 135}, {X: 2, Y: 128}, {X: 3, Y: 142}, {X: 4, Y: 155},
		{X: 5, Y: 148}, {X: 6, Y: 162}, {X: 7, Y: 170}, {X: 8, Y: 165}, {X: 9, Y: 158},
		{X: 10, Y: 175}, {X: 11, Y: 180}, {X: 12, Y: 172}, {X: 13, Y: 168}, {X: 14, Y: 185},
	}
}

func production1() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 0, Y: 450}, {X: 1, Y: 520}, {X: 2, Y: 480}, {X: 3, Y: 550}, {X: 4, Y: 610},
		{X: 5, Y: 580}, {X: 6, Y: 620}, {X: 7, Y: 590}, {X: 8, Y: 640}, {X: 9, Y: 670},
		{X: 10, Y: 650}, {X: 11, Y: 700},
	}
}

func production2() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 0, Y: 380}, {X: 1, Y: 420}, {X: 2, Y: 400}, {X: 3, Y: 460}, {X: 4, Y: 500},
		{X: 5, Y: 470}, {X: 6, Y: 510}, {X: 7, Y: 490}, {X: 8, Y: 530}, {X: 9, Y: 560},
		{X: 10, Y: 540}, {X: 11, Y: 580},
	}
}

func scatterDataA() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 100, Y: 92}, {X: 200, Y: 88}, {X: 300, Y: 85}, {X: 400, Y: 80}, {X: 500, Y: 75},
		{X: 150, Y: 90}, {X: 250, Y: 87}, {X: 350, Y: 83}, {X: 450, Y: 78}, {X: 550, Y: 72},
		{X: 120, Y: 91}, {X: 220, Y: 89}, {X: 320, Y: 84}, {X: 420, Y: 79}, {X: 520, Y: 74},
	}
}

func scatterDataB() []chart.DataPoint {
	return []chart.DataPoint{
		{X: 100, Y: 95}, {X: 200, Y: 93}, {X: 300, Y: 90}, {X: 400, Y: 86}, {X: 500, Y: 82},
		{X: 150, Y: 94}, {X: 250, Y: 92}, {X: 350, Y: 88}, {X: 450, Y: 84}, {X: 550, Y: 80},
		{X: 120, Y: 94}, {X: 220, Y: 91}, {X: 320, Y: 89}, {X: 420, Y: 85}, {X: 520, Y: 81},
	}
}
