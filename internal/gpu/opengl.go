//go:build !nogui && !windows

package gpu

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
)

// OpenGLRenderer implements Renderer using OpenGL 3.3 Core.
type OpenGLRenderer struct {
	width  int
	height int
	bgColor draw.Color
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
	gl.Enable(gl.SCISSOR_TEST)

	return nil
}

// SetBackgroundColor sets the clear color for BeginFrame.
func (r *OpenGLRenderer) SetBackgroundColor(c draw.Color) {
	r.bgColor = c
}

// Resize updates the viewport.
func (r *OpenGLRenderer) Resize(width, height int) {
	r.width = width
	r.height = height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// BeginFrame clears the screen.
func (r *OpenGLRenderer) BeginFrame() {
	gl.Disable(gl.SCISSOR_TEST)
	gl.ClearColor(r.bgColor.R, r.bgColor.G, r.bgColor.B, r.bgColor.A)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.Enable(gl.SCISSOR_TEST)
}

// Draw renders the scene via gl.Scissor + gl.Clear.
func (r *OpenGLRenderer) Draw(scene draw.Scene) {
	for _, rect := range scene.Rects {
		r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
	}
	for _, glyph := range scene.Glyphs {
		r.drawGlyph(glyph)
	}
}

// EndFrame is a no-op for OpenGL (swap is handled by GLFW).
func (r *OpenGLRenderer) EndFrame() {}

// Destroy releases OpenGL resources.
func (r *OpenGLRenderer) Destroy() {}

func (r *OpenGLRenderer) fillRect(x, y, w, h int, color draw.Color) {
	if w <= 0 || h <= 0 || r.width <= 0 || r.height <= 0 {
		return
	}
	gl.ClearColor(color.R, color.G, color.B, color.A)
	gl.Scissor(int32(x), int32(r.height-y-h), int32(w), int32(h))
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (r *OpenGLRenderer) drawGlyph(cmd draw.DrawGlyph) {
	if cmd.Scale <= 0 {
		cmd.Scale = 1
	}
	render.RenderBitmapGlyph(cmd.Text, cmd.X, cmd.Y, cmd.Scale, func(px, py, w, h int) {
		r.fillRect(px, py, w, h, cmd.Color)
	})
}
