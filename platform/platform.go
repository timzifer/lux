// Package platform defines the Platform interface for windowing backends (RFC §7.1).
package platform

// Platform abstracts the native windowing system.
// Implementations exist for GLFW (M1), with native Cocoa/Wayland/X11/DRM planned.
type Platform interface {
	// Init creates the window and initializes the platform.
	Init(cfg Config) error

	// Run enters the platform event loop. It blocks until the window is closed.
	// The provided callbacks are invoked from the event loop.
	Run(cb Callbacks) error

	// Destroy releases all platform resources.
	Destroy()

	// SetTitle updates the window title.
	SetTitle(title string)

	// WindowSize returns the current window size in pixels.
	WindowSize() (width, height int)

	// FramebufferSize returns the framebuffer size in pixels (may differ from
	// WindowSize on HiDPI displays).
	FramebufferSize() (width, height int)

	// ShouldClose returns true if the user has requested window close.
	ShouldClose() bool
}

// Config holds platform initialization parameters.
type Config struct {
	Title  string
	Width  int
	Height int
}

// Callbacks are invoked by the platform event loop.
type Callbacks struct {
	// OnFrame is called each iteration of the event loop.
	OnFrame func()

	// OnResize is called when the window or framebuffer is resized.
	OnResize func(width, height int)

	// OnClose is called when the user requests window close.
	OnClose func()

	// OnMouseButton is called when a mouse button is pressed or released (M3).
	// button: 0=left, 1=right, 2=middle. pressed: true=down, false=up.
	OnMouseButton func(x, y float32, button int, pressed bool)

	// OnMouseMove is called when the mouse cursor moves within the window (M4).
	OnMouseMove func(x, y float32)
}
