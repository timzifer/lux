//go:build !nogui && !windows && !(darwin && arm64) && !(js && wasm)

// Package glfw implements platform.Platform using GLFW.
// This is the M1 windowing backend for macOS and Linux.
package glfw

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

func init() {
	// GLFW must be called from the main thread on macOS.
	runtime.LockOSThread()
}

// Platform implements platform.Platform using GLFW.
type Platform struct {
	window       *glfw.Window
	config       platform.Config
	cursors      map[input.CursorKind]*glfw.Cursor
	cursorKind   input.CursorKind
	fullscreen   bool
	savedX, savedY, savedW, savedH int // saved window geometry before fullscreen
	frameRequested bool
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
	p.initCursors()
	return nil
}

// initCursors creates standard GLFW cursors for each CursorKind.
func (p *Platform) initCursors() {
	p.cursors = map[input.CursorKind]*glfw.Cursor{
		input.CursorDefault:    glfw.CreateStandardCursor(glfw.ArrowCursor),
		input.CursorText:       glfw.CreateStandardCursor(glfw.IBeamCursor),
		input.CursorPointer:    glfw.CreateStandardCursor(glfw.HandCursor),
		input.CursorCrosshair:  glfw.CreateStandardCursor(glfw.CrosshairCursor),
		input.CursorResizeEW:   glfw.CreateStandardCursor(glfw.HResizeCursor),
		input.CursorResizeNS:   glfw.CreateStandardCursor(glfw.VResizeCursor),
	}
}

// SetCursor changes the system cursor shape (RFC-002 §2.7).
func (p *Platform) SetCursor(kind input.CursorKind) {
	if p.window == nil || kind == p.cursorKind {
		return
	}
	p.cursorKind = kind
	if kind == input.CursorNone {
		p.window.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		return
	}
	p.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	if c, ok := p.cursors[kind]; ok {
		p.window.SetCursor(c)
	} else {
		// Fallback to default arrow for unsupported cursor types.
		p.window.SetCursor(p.cursors[input.CursorDefault])
	}
}

// SetIMECursorRect positions the IME candidate window near the text cursor (RFC-002 §2.2).
// GLFW 3.3 does not expose glfwSetPreeditCursorRectangle, so this is a no-op.
// When upgrading to GLFW 3.4+, call glfwSetPreeditCursorRectangle here.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {
	// GLFW 3.3 does not support preedit cursor rectangle.
	// This will be implemented when upgrading to GLFW 3.4.
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

	if cb.OnMouseButton != nil {
		p.window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
			x, y := w.GetCursorPos()
			pressed := action == glfw.Press
			cb.OnMouseButton(float32(x), float32(y), int(button), pressed)
		})
	}

	if cb.OnMouseMove != nil {
		p.window.SetCursorPosCallback(func(_ *glfw.Window, xpos, ypos float64) {
			cb.OnMouseMove(float32(xpos), float32(ypos))
		})
	}

	if cb.OnScroll != nil {
		p.window.SetScrollCallback(func(_ *glfw.Window, xoff, yoff float64) {
			cb.OnScroll(float32(xoff), float32(yoff))
		})
	}

	if cb.OnKey != nil {
		p.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
			name := glfwKeyName(key)
			act := int(action) // glfw: 0=release, 1=press, 2=repeat
			// Remap to our convention: 0=press, 1=release, 2=repeat
			switch action {
			case glfw.Press:
				act = 0
			case glfw.Release:
				act = 1
			case glfw.Repeat:
				act = 2
			}
			m := 0
			if mods&glfw.ModShift != 0 {
				m |= 1
			}
			if mods&glfw.ModControl != 0 {
				m |= 2
			}
			if mods&glfw.ModAlt != 0 {
				m |= 4
			}
			if mods&glfw.ModSuper != 0 {
				m |= 8
			}
			cb.OnKey(name, act, m)
		})
	}

	if cb.OnChar != nil {
		p.window.SetCharCallback(func(_ *glfw.Window, ch rune) {
			cb.OnChar(ch)
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

// SetSize resizes the window to the given dimensions in screen coordinates (RFC §7.1).
func (p *Platform) SetSize(w, h int) {
	if p.window != nil {
		p.window.SetSize(w, h)
	}
}

// SetFullscreen toggles fullscreen mode (RFC §7.1).
func (p *Platform) SetFullscreen(fullscreen bool) {
	if p.window == nil || fullscreen == p.fullscreen {
		return
	}
	p.fullscreen = fullscreen
	if fullscreen {
		// Save current window position and size.
		p.savedX, p.savedY = p.window.GetPos()
		p.savedW, p.savedH = p.window.GetSize()
		// Switch to primary monitor's fullscreen mode.
		monitor := glfw.GetPrimaryMonitor()
		mode := monitor.GetVideoMode()
		p.window.SetMonitor(monitor, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
	} else {
		// Restore windowed mode with saved geometry.
		p.window.SetMonitor(nil, p.savedX, p.savedY, p.savedW, p.savedH, 0)
	}
}

// RequestFrame requests a new frame to be rendered as soon as possible (RFC §7.1).
func (p *Platform) RequestFrame() {
	p.frameRequested = true
	if p.window != nil {
		glfw.PostEmptyEvent()
	}
}

// SetClipboard sets the system clipboard text (RFC §7.1).
func (p *Platform) SetClipboard(text string) error {
	if p.window != nil {
		p.window.SetClipboardString(text)
	}
	return nil
}

// GetClipboard returns the current system clipboard text (RFC §7.1).
func (p *Platform) GetClipboard() (string, error) {
	if p.window == nil {
		return "", nil
	}
	return p.window.GetClipboardString(), nil
}

// CreateWGPUSurface creates a wgpu surface for this window (RFC §7.1).
// Returns 0 — wgpu surface creation requires wgpu-native integration
// which will use GLFW's native window handle.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	// TODO: Implement via wgpu-native's wgpuInstanceCreateSurface with
	// platform-specific GLFW native access (glfwGetX11Window, glfwGetCocoaWindow, etc.)
	return 0
}

// Window returns the underlying GLFW window.
func (p *Platform) Window() *glfw.Window {
	return p.window
}

// glfwKeyName maps a GLFW key code to a human-readable name.
func glfwKeyName(key glfw.Key) string {
	switch key {
	case glfw.KeySpace:
		return "Space"
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return "Enter"
	case glfw.KeyTab:
		return "Tab"
	case glfw.KeyBackspace:
		return "Backspace"
	case glfw.KeyEscape:
		return "Escape"
	case glfw.KeyLeft:
		return "Left"
	case glfw.KeyRight:
		return "Right"
	case glfw.KeyUp:
		return "Up"
	case glfw.KeyDown:
		return "Down"
	case glfw.KeyHome:
		return "Home"
	case glfw.KeyEnd:
		return "End"
	case glfw.KeyDelete:
		return "Delete"
	case glfw.KeyLeftShift, glfw.KeyRightShift:
		return "Shift"
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		return "Ctrl"
	case glfw.KeyLeftAlt, glfw.KeyRightAlt:
		return "Alt"
	case glfw.KeyLeftSuper, glfw.KeyRightSuper:
		return "Super"
	default:
		if key >= glfw.KeyA && key <= glfw.KeyZ {
			return string(rune('A' + (key - glfw.KeyA)))
		}
		if key >= glfw.Key0 && key <= glfw.Key9 {
			return string(rune('0' + (key - glfw.Key0)))
		}
		return fmt.Sprintf("Key(%d)", key)
	}
}
