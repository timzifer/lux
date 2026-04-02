// Charts-Live — live streaming line chart with auto-scrolling viewport.
//
//	go run ./examples/charts-live/
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/chart"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

const (
	bufferCapacity = 500
	windowSeconds  = 10.0
)

type Model struct {
	Signal   *chart.RingBuffer
	Noise    *chart.RingBuffer
	Elapsed  float64
	Viewport chart.Viewport
}

type NewDataMsg struct {
	Signal chart.DataPoint
	Noise  chart.DataPoint
}

func initModel() Model {
	return Model{
		Signal:   chart.NewRingBuffer(bufferCapacity),
		Noise:    chart.NewRingBuffer(bufferCapacity),
		Viewport: chart.Viewport{XMin: -windowSeconds, XMax: 0, YMin: -2, YMax: 2},
	}
}

func update(m Model, msg app.Msg) Model {
	switch msg := msg.(type) {
	case app.TickMsg:
		dt := msg.DeltaTime.Seconds()
		m.Elapsed += dt

		// Generate synthetic data: clean sine wave + noisy variant.
		t := m.Elapsed
		signalY := math.Sin(t * 2 * math.Pi * 0.3) // 0.3 Hz sine
		noiseY := signalY*0.6 + (rand.Float64()-0.5)*0.8

		m.Signal.Push(chart.DataPoint{X: t, Y: signalY})
		m.Noise.Push(chart.DataPoint{X: t, Y: noiseY})

		// Auto-scroll viewport.
		m.Viewport = chart.AutoScrollViewport(m.Signal.Slice(), windowSeconds)

	case chart.ChartViewportMsg:
		m.Viewport = msg.Viewport
	}
	return m
}

func view(m Model) ui.Element {
	vp := m.Viewport
	signalData := m.Signal.Slice()
	noiseData := m.Noise.Slice()

	return layout.Pad(draw.Insets{Top: 24, Right: 24, Bottom: 24, Left: 24},
		layout.Column(
			display.Text("Live Streaming Chart"),
			display.Text(fmt.Sprintf("Elapsed: %.1fs  |  Points: %d", m.Elapsed, m.Signal.Len())),
			display.Divider(),

			chart.Line(chart.ChartConfig{
				Width: 700, Height: 300,
				Title: "Signal Monitor",
				Viewport: &vp,
				XAxis: chart.Axis{
					Label:     "Time (s)",
					GridLines: true,
					Format: func(v float64) string {
						return fmt.Sprintf("%.0fs", v)
					},
				},
				YAxis: chart.Axis{
					Label:     "Amplitude",
					GridLines: true,
				},
			},
				chart.Series{Name: "Signal", Points: signalData},
				chart.Series{Name: "Noise", Points: noiseData},
			),

			display.Divider(),

			chart.Area(chart.ChartConfig{
				Width: 700, Height: 200,
				Title: "Signal (Area)",
				Viewport: &vp,
				XAxis: chart.Axis{GridLines: true, Format: func(v float64) string {
					return fmt.Sprintf("%.0fs", v)
				}},
				YAxis: chart.Axis{GridLines: true},
			},
				chart.Series{Name: "Signal", Points: signalData},
			),
		),
	)
}

func main() {
	if err := app.Run(initModel(), update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Live Charts"),
	); err != nil {
		log.Fatal(err)
	}
}
