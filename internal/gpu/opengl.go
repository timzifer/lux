//go:build !nogui && !windows

package gpu

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// OpenGLRenderer implements Renderer using OpenGL 3.3 Core.
// This is the M1 GPU backend — it only clears the screen to black.
// Will be replaced by wgpu in M2+.
type OpenGLRenderer struct {
	width  int
	height int
}

// NewOpenGL creates an OpenGL-based renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
}

// Init initializes OpenGL.
func (r *OpenGLRenderer) Init(cfg Config) error {
	if err := gl.Init(); err != nil {
		return fmt.Errorf("opengl init: %w", err)
	}

	r.width = cfg.Width
	r.height = cfg.Height

	gl.Viewport(0, 0, int32(r.width), int32(r.height))
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	return nil
}

// Resize updates the viewport.
func (r *OpenGLRenderer) Resize(width, height int) {
	r.width = width
	r.height = height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// BeginFrame clears the screen to black.
func (r *OpenGLRenderer) BeginFrame() {
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// EndFrame is a no-op for OpenGL (swap is handled by GLFW).
func (r *OpenGLRenderer) EndFrame() {
	// Swap buffers is done by the platform layer (GLFW).
}

// Destroy releases OpenGL resources.
func (r *OpenGLRenderer) Destroy() {
	// Nothing to release for M1.
}
