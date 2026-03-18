package app

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/ui"
)

// Run starts the application. It blocks until the window is closed (RFC §3.1).
//
// The model, update, and view form the Elm architecture triad:
//   - model:  initial application state
//   - update: processes a Msg and returns a new model (pure function)
//   - view:   renders the model as an Element tree (pure function)
//
// Both update and view run exclusively on the calling goroutine.
func Run[M any](model M, update UpdateFunc[M], view ViewFunc[M], opts ...Option) error {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create the frame loop.
	var loopOpts []loop.Option
	if cfg.maxFrameDelta != loop.DefaultMaxFrameDelta {
		loopOpts = append(loopOpts, loop.WithMaxFrameDelta(cfg.maxFrameDelta))
	}
	appLoop := loop.New(loopOpts...)
	globalLoop = appLoop
	defer func() { globalLoop = nil }()

	// Create platform and GPU renderer via platform-specific defaults.
	plat := cfg.platformFactory()
	if err := plat.Init(platform.Config{
		Title:  cfg.title,
		Width:  cfg.width,
		Height: cfg.height,
	}); err != nil {
		return fmt.Errorf("platform init: %w", err)
	}
	defer plat.Destroy()

	renderer := cfg.rendererFactory()
	fbW, fbH := plat.FramebufferSize()
	if err := renderer.Init(gpu.Config{
		Width:  fbW,
		Height: fbH,
	}); err != nil {
		return fmt.Errorf("gpu init: %w", err)
	}
	defer renderer.Destroy()

	// Current model and element tree.
	currentModel := model
	_ = view(currentModel) // Initial view call (result unused in M1).

	lastFrame := time.Now()

	// Run the platform event loop.
	return plat.Run(platform.Callbacks{
		OnFrame: func() {
			// Drain pending messages and run update (RFC §3.3).
			appLoop.DrainMessages(func(msg any) bool {
				newModel := update(currentModel, msg)
				if anyChanged(currentModel, newModel) {
					currentModel = newModel
					_ = view(currentModel)
					return true
				}
				return false
			})

			// dt calculation with clamping (RFC §3.3).
			now := time.Now()
			rawDt := now.Sub(lastFrame)
			_ = appLoop.ClampDt(rawDt) // dt used for animations in M2+.
			lastFrame = now

			// Render: clear to black (M1).
			renderer.BeginFrame()
			renderer.EndFrame()
		},

		OnResize: func(width, height int) {
			renderer.Resize(width, height)
		},
	})
}

// anyChanged does a shallow pointer-based check. For M1 this is always true
// since Go structs are value types. M2+ will use VTree diffing.
func anyChanged[M any](_, _ M) bool {
	return true
}

// Ensure ViewFunc constraint is satisfied at compile time.
var _ ViewFunc[struct{}] = func(_ struct{}) ui.Element { return ui.Empty() }
