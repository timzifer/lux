package app

// runMultiWindow is a placeholder for the multi-window run loop.
// Full implementation requires per-window context management:
//   - Per-window reconciler, hitMap, hoverState, canvas, currentTree
//   - OpenWindowMsg → creates window via platform + renderer, sends WindowOpenedMsg
//   - CloseWindowMsg → destroys window, sends WindowClosedMsg
//   - Frame loop: call multiView(model), reconcile+build+draw per window
//   - Route input events by windowID
//
// This stub ensures the types compile and can be wired into the main loop.

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
)

// multiWindowState tracks state for secondary windows.
type multiWindowState struct {
	id       WindowID
	platform platform.Platform
	renderer gpu.Renderer
	width    int
	height   int
}

// handleOpenWindow processes an OpenWindowMsg in the app loop.
func handleOpenWindow(msg OpenWindowMsg, plat platform.Platform, rend gpu.Renderer) *multiWindowState {
	mwp, ok := plat.(platform.MultiWindowPlatform)
	if !ok {
		return nil
	}
	cfg := platform.Config{
		Title:  msg.Config.Title,
		Width:  msg.Config.Width,
		Height: msg.Config.Height,
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
	return &multiWindowState{
		id:       msg.ID,
		platform: plat,
		renderer: rend,
		width:    msg.Config.Width,
		height:   msg.Config.Height,
	}
}

// handleCloseWindow processes a CloseWindowMsg in the app loop.
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
