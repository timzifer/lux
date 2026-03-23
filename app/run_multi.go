package app

import (
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/ui"
)

// windowContext holds per-window rendering and input state for multi-window mode.
type windowContext struct {
	id          WindowID
	config      WindowConfig
	reconciler  *ui.Reconciler
	hitMap      hit.Map
	hoverState  ui.HoverState
	dispatcher  *ui.EventDispatcher
	mouseX      float32
	mouseY      float32
	dragCB      func(x, y float32)
	dragRelease func(x, y float32)
	cursor      input.CursorKind
	currentTree ui.Element
	width       int
	height      int
}

// newWindowContext creates a windowContext for a secondary window.
func newWindowContext(id WindowID, cfg WindowConfig, fm *ui.FocusManager) *windowContext {
	return &windowContext{
		id:         id,
		config:     cfg,
		reconciler: ui.NewReconciler(),
		dispatcher: ui.NewEventDispatcher(fm),
		width:      cfg.Width,
		height:     cfg.Height,
	}
}

// handleOpenWindow processes an OpenWindowMsg: creates the platform window,
// initializes the per-window renderer, and returns a windowContext.
func handleOpenWindow(msg OpenWindowMsg, plat platform.Platform, rend gpu.Renderer, fm *ui.FocusManager) *windowContext {
	mwp, ok := plat.(platform.MultiWindowPlatform)
	if !ok {
		return nil
	}
	cfg := platform.Config{
		Title:     msg.Config.Title,
		Width:     msg.Config.Width,
		Height:    msg.Config.Height,
		Type:      int(msg.Config.Type),
		Resizable: msg.Config.Resizable,
	}
	handle, err := mwp.CreateWindow(uint32(msg.ID), cfg)
	if err != nil {
		return nil
	}

	// Initialize per-window renderer if supported.
	if wr, ok := rend.(gpu.WindowRenderer); ok {
		gpuCfg := gpu.Config{
			Width:        msg.Config.Width,
			Height:       msg.Config.Height,
			NativeHandle: handle,
		}
		if err := wr.InitWindow(uint32(msg.ID), gpuCfg); err != nil {
			mwp.DestroyWindow(uint32(msg.ID))
			return nil
		}
	}

	Send(WindowOpenedMsg{Window: msg.ID})
	return newWindowContext(msg.ID, msg.Config, fm)
}

// handleCloseWindow processes a CloseWindowMsg: destroys the platform window
// and per-window renderer resources.
func handleCloseWindow(msg CloseWindowMsg, plat platform.Platform, rend gpu.Renderer) {
	mwp, ok := plat.(platform.MultiWindowPlatform)
	if !ok {
		return
	}
	if wr, ok := rend.(gpu.WindowRenderer); ok {
		wr.DestroyWindow(uint32(msg.ID))
	}
	mwp.DestroyWindow(uint32(msg.ID))
	Send(WindowClosedMsg{Window: msg.ID})
}
