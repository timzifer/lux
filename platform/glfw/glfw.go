//go:build !nogui

// Package glfw implements platform.Platform using GLFW.
// This is the M1 windowing backend, suitable for macOS, Linux, and Windows.
package glfw

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/timzifer/lux/platform"
)

func init() {
	// GLFW must be called from the main thread on macOS.
	runtime.LockOSThread()
}

// Platform implements platform.Platform using GLFW.
type Platform struct {
	window *glfw.Window
	config platform.Config
}

// New creates a new GLFW platform instance.
func New() *Platform {
	return &Platform{}
}

// Init creates the GLFW window with an OpenGL 3.3 Core context.
func (p *Platform) Init(cfg platform.Config) error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw init: %w", err)
	}

	// OpenGL 3.3 Core for M1 rendering. Will switch to NoAPI + wgpu in M2+.
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	w, h := cfg.Width, cfg.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}

	win, err := glfw.CreateWindow(w, h, cfg.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return fmt.Errorf("create window: %w", err)
	}

	win.MakeContextCurrent()

	// Enable VSync for ~60fps.
	glfw.SwapInterval(1)

	p.window = win
	p.config = cfg
	return nil
}

// Run enters the GLFW event loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	if cb.OnResize != nil {
		p.window.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
			cb.OnResize(w, h)
		})
	}

	if cb.OnClose != nil {
		p.window.SetCloseCallback(func(_ *glfw.Window) {
			cb.OnClose()
		})
	}

	for !p.window.ShouldClose() {
		glfw.PollEvents()
		if cb.OnFrame != nil {
			cb.OnFrame()
		}
		p.window.SwapBuffers()
	}

	return nil
}

// Destroy releases GLFW resources.
func (p *Platform) Destroy() {
	if p.window != nil {
		p.window.Destroy()
		p.window = nil
	}
	glfw.Terminate()
}

// SetTitle updates the window title.
func (p *Platform) SetTitle(title string) {
	if p.window != nil {
		p.window.SetTitle(title)
	}
}

// WindowSize returns the window size in screen coordinates.
func (p *Platform) WindowSize() (int, int) {
	if p.window == nil {
		return 0, 0
	}
	return p.window.GetSize()
}

// FramebufferSize returns the framebuffer size in pixels.
func (p *Platform) FramebufferSize() (int, int) {
	if p.window == nil {
		return 0, 0
	}
	return p.window.GetFramebufferSize()
}

// ShouldClose returns true if the user has requested window close.
func (p *Platform) ShouldClose() bool {
	if p.window == nil {
		return true
	}
	return p.window.ShouldClose()
}

// Window returns the underlying GLFW window.
func (p *Platform) Window() *glfw.Window {
	return p.window
}
