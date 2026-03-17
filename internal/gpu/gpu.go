// Package gpu provides the GPU rendering abstraction for the framework.
// For M1 it provides a minimal OpenGL backend that clears the screen to black.
package gpu

// Renderer abstracts GPU operations. M1 only needs ClearFrame.
// Will be replaced by a wgpu-based implementation in M2+.
type Renderer interface {
	// Init initializes the GPU context for the given window.
	Init(cfg Config) error

	// Resize updates the viewport when the window is resized.
	Resize(width, height int)

	// BeginFrame starts a new frame.
	BeginFrame()

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
