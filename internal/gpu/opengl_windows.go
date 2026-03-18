//go:build windows && !nogui

package gpu

import "github.com/timzifer/lux/ui"

// OpenGLRenderer implements the Windows M1/M2 bootstrap renderer.
type OpenGLRenderer struct {
	width  int
	height int
}

// NewOpenGL creates the Windows bootstrap renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
}

// Init stores the framebuffer size.
func (r *OpenGLRenderer) Init(cfg Config) error {
	r.width = cfg.Width
	r.height = cfg.Height
	return nil
}

// Resize updates the tracked framebuffer size.
func (r *OpenGLRenderer) Resize(width, height int) {
	r.width = width
	r.height = height
}

// BeginFrame is intentionally a no-op for the Windows bootstrap renderer.
func (r *OpenGLRenderer) BeginFrame() {}

// Draw is a no-op until the native Windows GPU backend lands.
func (r *OpenGLRenderer) Draw(scene ui.Scene) {}

// EndFrame is a no-op because swap is handled by the platform layer.
func (r *OpenGLRenderer) EndFrame() {}

// Destroy releases renderer resources.
func (r *OpenGLRenderer) Destroy() {}
