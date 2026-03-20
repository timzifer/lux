// Package platform defines the Platform interface for windowing backends (RFC §7.1).
package platform

import "github.com/timzifer/lux/input"

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

	// SetCursor changes the system cursor shape (RFC-002 §2.7).
	SetCursor(kind input.CursorKind)

	// SetIMECursorRect informs the platform of the text cursor position
	// so the IME candidate window can be placed near the insertion point
	// (RFC-002 §2.2). x, y, w, h are in screen coordinates.
	SetIMECursorRect(x, y, w, h int)

	// SetSize resizes the window to the given dimensions in screen coordinates (RFC §7.1).
	SetSize(w, h int)

	// SetFullscreen toggles fullscreen mode (RFC §7.1).
	SetFullscreen(fullscreen bool)

	// RequestFrame requests a new frame to be rendered as soon as possible (RFC §7.1).
	RequestFrame()

	// SetClipboard sets the system clipboard text (RFC §7.1).
	SetClipboard(text string) error

	// GetClipboard returns the current system clipboard text (RFC §7.1).
	GetClipboard() (string, error)

	// CreateWGPUSurface creates a wgpu surface for GPU rendering (RFC §7.1).
	// The instance parameter is a wgpu.Instance handle (passed as uintptr to avoid circular imports).
	// Returns a wgpu.Surface handle, or 0 if not supported.
	CreateWGPUSurface(instance uintptr) uintptr
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

	// OnScroll is called when the mouse wheel or trackpad is scrolled.
	// deltaX and deltaY are in scroll units (positive = right/down).
	OnScroll func(deltaX, deltaY float32)

	// OnKey is called when a key is pressed, released, or repeated.
	// key is the key name (e.g. "A", "Enter", "Escape").
	// action: 0=press, 1=release, 2=repeat.
	// mods encodes modifier state: bit 0=Shift, 1=Ctrl, 2=Alt, 3=Super.
	OnKey func(key string, action int, mods int)

	// OnChar is called when a Unicode character is input (for text entry).
	OnChar func(ch rune)

	// OnIMECompose is called when the IME composition state changes (RFC-002 §2.2).
	// text is the current pre-edit string, cursorStart/cursorEnd define the
	// cursor range within the pre-edit text (in runes).
	OnIMECompose func(text string, cursorStart, cursorEnd int)

	// OnIMECommit is called when the IME commits final text (RFC-002 §2.2).
	OnIMECommit func(text string)

	// ── Multi-window callbacks ────────────────────────────────────
	OnWindowResize      func(windowID uint32, width, height int)
	OnWindowClose       func(windowID uint32)
	OnWindowMouseButton func(windowID uint32, x, y float32, button int, pressed bool)
	OnWindowMouseMove   func(windowID uint32, x, y float32)
	OnWindowKey         func(windowID uint32, key string, action int, mods int)
	OnWindowChar        func(windowID uint32, ch rune)
	OnWindowScroll      func(windowID uint32, deltaX, deltaY float32)
}

// MultiWindowPlatform extends Platform with multi-window support.
// Implementations that support multiple windows should implement this interface.
type MultiWindowPlatform interface {
	Platform
	CreateWindow(id uint32, cfg Config) (nativeHandle uintptr, err error)
	DestroyWindow(id uint32)
	SetWindowTitle(id uint32, title string)
	WindowSizeByID(id uint32) (int, int)
	FramebufferSizeByID(id uint32) (int, int)
}
