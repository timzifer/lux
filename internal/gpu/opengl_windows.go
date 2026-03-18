//go:build windows && !nogui

package gpu

// OpenGLRenderer implements the Windows M1 renderer.
//
// The platform layer still creates and presents the window/context. For M1 on
// Windows we keep rendering intentionally minimal so the application can open a
// window and drive the frame loop without depending on platform-specific GL
// bindings during the bootstrap milestone.
type OpenGLRenderer struct {
	width  int
	height int
}

// NewOpenGL creates the Windows M1 renderer.
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

// EndFrame is a no-op because swap is handled by the platform layer.
func (r *OpenGLRenderer) EndFrame() {}

// Destroy releases renderer resources.
func (r *OpenGLRenderer) Destroy() {}
