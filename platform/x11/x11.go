//go:build x11 && !nogui

// Package x11 implements platform.Platform using X11 (Xlib) via CGo.
// This backend provides direct X11 support without GLFW.
package x11

/*
#cgo pkg-config: x11 xi xfixes

#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <X11/cursorfont.h>
#include <X11/extensions/XInput2.h>
#include <X11/extensions/Xfixes.h>
#include <stdlib.h>
#include <string.h>

static Display* lux_open_display() {
    return XOpenDisplay(NULL);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

func init() {
	runtime.LockOSThread()
}

// Platform implements platform.Platform using X11.
type Platform struct {
	display     *C.Display
	screen      C.int
	window      C.Window
	rootWindow  C.Window
	gc          C.GC
	wmDelete    C.Atom
	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	fullscreen  bool
	width, height int
	clipboard   string

	// Atoms for EWMH/ICCCM protocols.
	atomWMState        C.Atom
	atomWMStateFS      C.Atom
	atomClipboard      C.Atom
	atomUTF8String     C.Atom
	atomTargets        C.Atom
	atomNetWMName      C.Atom

	// Saved geometry for fullscreen restore.
	savedX, savedY, savedW, savedH int
}

// New creates a new X11 platform instance.
func New() *Platform {
	return &Platform{}
}

// Init opens the X11 display and creates a window.
func (p *Platform) Init(cfg platform.Config) error {
	p.config = cfg
	p.width = cfg.Width
	p.height = cfg.Height
	if p.width <= 0 {
		p.width = 800
	}
	if p.height <= 0 {
		p.height = 600
	}

	// Open display.
	p.display = C.lux_open_display()
	if p.display == nil {
		return fmt.Errorf("x11: failed to open display")
	}

	p.screen = C.XDefaultScreen(p.display)
	p.rootWindow = C.XRootWindow(p.display, p.screen)

	// Create window.
	p.window = C.XCreateSimpleWindow(
		p.display, p.rootWindow,
		0, 0, C.uint(p.width), C.uint(p.height), 0,
		C.XBlackPixel(p.display, p.screen),
		C.XWhitePixel(p.display, p.screen),
	)

	// Select input events.
	C.XSelectInput(p.display, p.window,
		C.ExposureMask|C.KeyPressMask|C.KeyReleaseMask|
			C.ButtonPressMask|C.ButtonReleaseMask|
			C.PointerMotionMask|C.StructureNotifyMask)

	// Set WM_DELETE_WINDOW protocol.
	cName := C.CString("WM_DELETE_WINDOW")
	defer C.free(unsafe.Pointer(cName))
	p.wmDelete = C.XInternAtom(p.display, cName, C.False)
	C.XSetWMProtocols(p.display, p.window, &p.wmDelete, 1)

	// Intern useful atoms.
	p.atomWMState = p.internAtom("_NET_WM_STATE")
	p.atomWMStateFS = p.internAtom("_NET_WM_STATE_FULLSCREEN")
	p.atomClipboard = p.internAtom("CLIPBOARD")
	p.atomUTF8String = p.internAtom("UTF8_STRING")
	p.atomTargets = p.internAtom("TARGETS")
	p.atomNetWMName = p.internAtom("_NET_WM_NAME")

	// Set window title.
	if cfg.Title != "" {
		p.SetTitle(cfg.Title)
	}

	// Map (show) window.
	C.XMapWindow(p.display, p.window)
	C.XFlush(p.display)

	return nil
}

// Run enters the X11 event loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb
	var event C.XEvent

	for !p.shouldClose {
		// Process all pending events.
		for C.XPending(p.display) > 0 {
			C.XNextEvent(p.display, &event)
			p.handleEvent(&event)
		}

		if p.shouldClose {
			break
		}

		if cb.OnFrame != nil {
			cb.OnFrame()
		}
	}

	return nil
}

func (p *Platform) handleEvent(event *C.XEvent) {
	evType := *(*C.int)(unsafe.Pointer(event))
	switch evType {
	case C.ConfigureNotify:
		ce := (*C.XConfigureEvent)(unsafe.Pointer(event))
		newW, newH := int(ce.width), int(ce.height)
		if newW != p.width || newH != p.height {
			p.width = newW
			p.height = newH
			if p.callbacks.OnResize != nil {
				p.callbacks.OnResize(newW, newH)
			}
		}
	case C.KeyPress:
		ke := (*C.XKeyEvent)(unsafe.Pointer(event))
		name := x11KeyName(p.display, ke.keycode)
		mods := x11Mods(ke.state)
		if p.callbacks.OnKey != nil {
			p.callbacks.OnKey(name, 0, mods) // press
		}
		// Lookup character for text input.
		var buf [32]C.char
		n := C.XLookupString(ke, &buf[0], 32, nil, nil)
		if n > 0 && p.callbacks.OnChar != nil {
			for i := C.int(0); i < n; i++ {
				ch := rune(buf[i])
				if ch >= 32 {
					p.callbacks.OnChar(ch)
				}
			}
		}
	case C.KeyRelease:
		ke := (*C.XKeyEvent)(unsafe.Pointer(event))
		name := x11KeyName(p.display, ke.keycode)
		mods := x11Mods(ke.state)
		if p.callbacks.OnKey != nil {
			p.callbacks.OnKey(name, 1, mods) // release
		}
	case C.ButtonPress:
		be := (*C.XButtonEvent)(unsafe.Pointer(event))
		x, y := float32(be.x), float32(be.y)
		switch be.button {
		case 1, 2, 3: // Left, Middle, Right
			btn := int(be.button - 1)
			if btn == 1 {
				btn = 2
			} else if btn == 2 {
				btn = 1
			}
			if p.callbacks.OnMouseButton != nil {
				p.callbacks.OnMouseButton(x, y, btn, true)
			}
		case 4: // Scroll up
			if p.callbacks.OnScroll != nil {
				p.callbacks.OnScroll(0, 1)
			}
		case 5: // Scroll down
			if p.callbacks.OnScroll != nil {
				p.callbacks.OnScroll(0, -1)
			}
		}
	case C.ButtonRelease:
		be := (*C.XButtonEvent)(unsafe.Pointer(event))
		x, y := float32(be.x), float32(be.y)
		if be.button >= 1 && be.button <= 3 {
			btn := int(be.button - 1)
			if btn == 1 {
				btn = 2
			} else if btn == 2 {
				btn = 1
			}
			if p.callbacks.OnMouseButton != nil {
				p.callbacks.OnMouseButton(x, y, btn, false)
			}
		}
	case C.MotionNotify:
		me := (*C.XMotionEvent)(unsafe.Pointer(event))
		if p.callbacks.OnMouseMove != nil {
			p.callbacks.OnMouseMove(float32(me.x), float32(me.y))
		}
	case C.ClientMessage:
		cm := (*C.XClientMessageEvent)(unsafe.Pointer(event))
		data := (*[5]C.long)(unsafe.Pointer(&cm.data))
		if C.Atom(data[0]) == p.wmDelete {
			p.shouldClose = true
			if p.callbacks.OnClose != nil {
				p.callbacks.OnClose()
			}
		}
	}
}

// Destroy releases X11 resources.
func (p *Platform) Destroy() {
	if p.window != 0 {
		C.XDestroyWindow(p.display, p.window)
		p.window = 0
	}
	if p.display != nil {
		C.XCloseDisplay(p.display)
		p.display = nil
	}
}

// SetTitle updates the window title.
func (p *Platform) SetTitle(title string) {
	if p.display == nil {
		return
	}
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.XChangeProperty(p.display, p.window, p.atomNetWMName, p.atomUTF8String,
		8, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(cTitle)), C.int(len(title)))
	C.XFlush(p.display)
}

// WindowSize returns the window size.
func (p *Platform) WindowSize() (int, int) { return p.width, p.height }

// FramebufferSize returns the framebuffer size (same as window on X11).
func (p *Platform) FramebufferSize() (int, int) { return p.width, p.height }

// ShouldClose returns true if the window should close.
func (p *Platform) ShouldClose() bool { return p.shouldClose }

// SetCursor changes the cursor shape.
func (p *Platform) SetCursor(kind input.CursorKind) {
	if p.display == nil || kind == p.cursorKind {
		return
	}
	p.cursorKind = kind

	var shape C.uint
	switch kind {
	case input.CursorText:
		shape = C.XC_xterm
	case input.CursorPointer:
		shape = C.XC_hand2
	case input.CursorCrosshair:
		shape = C.XC_crosshair
	case input.CursorResizeEW:
		shape = C.XC_sb_h_double_arrow
	case input.CursorResizeNS:
		shape = C.XC_sb_v_double_arrow
	case input.CursorMove:
		shape = C.XC_fleur
	default:
		shape = C.XC_left_ptr
	}

	cursor := C.XCreateFontCursor(p.display, shape)
	C.XDefineCursor(p.display, p.window, cursor)
	C.XFreeCursor(p.display, cursor)
}

// SetIMECursorRect positions the IME candidate window.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {
	// TODO: Implement via XIM.
}

// SetSize resizes the window.
func (p *Platform) SetSize(w, h int) {
	if p.display == nil {
		return
	}
	C.XResizeWindow(p.display, p.window, C.uint(w), C.uint(h))
	C.XFlush(p.display)
}

// SetFullscreen toggles fullscreen via EWMH _NET_WM_STATE_FULLSCREEN.
func (p *Platform) SetFullscreen(fullscreen bool) {
	if p.display == nil || fullscreen == p.fullscreen {
		return
	}
	p.fullscreen = fullscreen

	var event C.XEvent
	ce := (*C.XClientMessageEvent)(unsafe.Pointer(&event))
	*(*C.int)(unsafe.Pointer(&event)) = C.ClientMessage
	ce.window = p.window
	ce.message_type = p.atomWMState
	ce.format = 32
	data := (*[5]C.long)(unsafe.Pointer(&ce.data))
	if fullscreen {
		data[0] = 1 // _NET_WM_STATE_ADD
	} else {
		data[0] = 0 // _NET_WM_STATE_REMOVE
	}
	data[1] = C.long(p.atomWMStateFS)

	C.XSendEvent(p.display, p.rootWindow, C.False,
		C.SubstructureRedirectMask|C.SubstructureNotifyMask, &event)
	C.XFlush(p.display)
}

// RequestFrame forces a repaint.
func (p *Platform) RequestFrame() {
	if p.display == nil {
		return
	}
	// Send an Expose event to trigger repaint.
	var event C.XEvent
	ee := (*C.XExposeEvent)(unsafe.Pointer(&event))
	*(*C.int)(unsafe.Pointer(&event)) = C.Expose
	ee.window = p.window
	ee.count = 0
	C.XSendEvent(p.display, p.window, C.False, C.ExposureMask, &event)
	C.XFlush(p.display)
}

// SetClipboard sets the X11 clipboard via CLIPBOARD selection.
func (p *Platform) SetClipboard(text string) error {
	p.clipboard = text
	if p.display != nil {
		C.XSetSelectionOwner(p.display, p.atomClipboard, p.window, C.CurrentTime)
	}
	return nil
}

// GetClipboard returns the clipboard text.
func (p *Platform) GetClipboard() (string, error) {
	// TODO: Full X11 selection protocol (request, wait for SelectionNotify).
	return p.clipboard, nil
}

// CreateWGPUSurface creates a wgpu surface from the X11 display and window.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	return uintptr(p.window)
}

// NativeHandle returns the X11 window.
func (p *Platform) NativeHandle() uintptr {
	return uintptr(p.window)
}

func (p *Platform) internAtom(name string) C.Atom {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return C.XInternAtom(p.display, cName, C.False)
}

func x11KeyName(display *C.Display, keycode C.uint) string {
	keysym := C.XKeycodeToKeysym(display, C.KeyCode(keycode), 0)
	switch keysym {
	case C.XK_Return:
		return "Enter"
	case C.XK_Escape:
		return "Escape"
	case C.XK_BackSpace:
		return "Backspace"
	case C.XK_Tab:
		return "Tab"
	case C.XK_space:
		return "Space"
	case C.XK_Left:
		return "Left"
	case C.XK_Right:
		return "Right"
	case C.XK_Up:
		return "Up"
	case C.XK_Down:
		return "Down"
	case C.XK_Home:
		return "Home"
	case C.XK_End:
		return "End"
	case C.XK_Delete:
		return "Delete"
	case C.XK_Shift_L, C.XK_Shift_R:
		return "Shift"
	case C.XK_Control_L, C.XK_Control_R:
		return "Ctrl"
	case C.XK_Alt_L, C.XK_Alt_R:
		return "Alt"
	case C.XK_Super_L, C.XK_Super_R:
		return "Super"
	default:
		if keysym >= C.XK_a && keysym <= C.XK_z {
			return string(rune('A' + (keysym - C.XK_a)))
		}
		if keysym >= C.XK_A && keysym <= C.XK_Z {
			return string(rune(keysym))
		}
		if keysym >= C.XK_0 && keysym <= C.XK_9 {
			return string(rune(keysym))
		}
		return fmt.Sprintf("Key(%d)", keysym)
	}
}

func x11Mods(state C.uint) int {
	mods := 0
	if state&C.ShiftMask != 0 {
		mods |= 1
	}
	if state&C.ControlMask != 0 {
		mods |= 2
	}
	if state&C.Mod1Mask != 0 { // Alt
		mods |= 4
	}
	if state&C.Mod4Mask != 0 { // Super
		mods |= 8
	}
	return mods
}
