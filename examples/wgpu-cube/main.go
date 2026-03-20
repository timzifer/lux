// WGPU Cube — standalone demo of the Lux Surface pipeline with a WGPU-rendered RGB cube.
//
//	go run -tags gogpu ./examples/wgpu-cube/
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Cube *CubeSurface
}

type TickMsg struct{ DeltaTime time.Duration }

func update(m Model, msg any) (Model, []func() any) {
	switch msg := msg.(type) {
	case TickMsg:
		if m.Cube != nil {
			m.Cube.Tick(msg.DeltaTime)
		}
	}
	return m, nil
}

func view(m Model) ui.Element {
	return ui.Column(
		ui.Padding(ui.Insets{Top: 16, Left: 16, Right: 16, Bottom: 8},
			ui.Text("WGPU RGB Cube — drag to rotate"),
		),
		ui.Padding(ui.UniformInsets(16),
			ui.Surface(1, m.Cube, 500, 400),
		),
		ui.Padding(ui.Insets{Left: 16, Bottom: 16},
			ui.Text(fmt.Sprintf("Preferred zero-copy mode: %d", ui.PreferredZeroCopyMode())),
		),
	)
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	cube := NewCubeSurface()

	initial := Model{
		Cube: cube,
	}

	// Tick animation at ~60 fps.
	lastTick := time.Now()
	ticker := func() any {
		now := time.Now()
		dt := now.Sub(lastTick)
		lastTick = now
		return TickMsg{DeltaTime: dt}
	}

	runOpts := []app.Option{
		app.WithTitle("WGPU Cube"),
		app.WithSize(550, 500),
		app.WithTick(time.Second/60, ticker),
	}
	if rf := cubeRendererFactory(cube); rf != nil {
		runOpts = append(runOpts, app.WithRenderer(rf))
	}

	if err := app.Run(initial, update, view, runOpts...); err != nil {
		log.Fatal(err)
	}
}

// Ensure draw is used (for draw.Rect in surface interface).
var _ = draw.Rect{}
