// Package gpu provides the GPU rendering abstraction for the framework.
package gpu

import "github.com/timzifer/lux/ui"

// Renderer abstracts GPU operations.
type Renderer interface {
	// Init initializes the GPU context for the given window.
	Init(cfg Config) error

	// Resize updates the viewport when the window is resized.
	Resize(width, height int)

	// BeginFrame starts a new frame.
	BeginFrame()

	// Draw renders the current scene.
	Draw(scene ui.Scene)

	// EndFrame presents the rendered frame.
	EndFrame()

	// Destroy releases GPU resources.
	Destroy()
}

// Config holds GPU initialization parameters.
type Config struct {
	Width  int
	Height int
}
