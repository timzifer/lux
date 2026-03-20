// Package gpu provides the GPU rendering abstraction for the framework.
package gpu

import "github.com/timzifer/lux/draw"

// Renderer abstracts GPU operations.
type Renderer interface {
	// Init initializes the GPU context for the given window.
	Init(cfg Config) error

	// Resize updates the viewport when the window is resized.
	Resize(width, height int)

	// BeginFrame starts a new frame.
	BeginFrame()

	// Draw renders the current scene.
	Draw(scene draw.Scene)

	// EndFrame presents the rendered frame.
	EndFrame()

	// Destroy releases GPU resources.
	Destroy()
}

// Config holds GPU initialization parameters.
type Config struct {
	Width        int
	Height       int
	NativeHandle uintptr // Platform-specific window handle (HWND on Windows, 0 otherwise).
}

// WindowRenderer extends Renderer with multi-window support.
// Implementations that can render to multiple windows should implement this interface.
type WindowRenderer interface {
	Renderer
	InitWindow(id uint32, cfg Config) error
	DestroyWindow(id uint32)
	ResizeWindow(id uint32, width, height int)
	BeginFrameWindow(id uint32)
	DrawWindow(id uint32, scene draw.Scene)
	EndFrameWindow(id uint32)
}
