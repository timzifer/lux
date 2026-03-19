//go:build wayland && !nogui

// Package wayland implements platform.Platform using the Wayland display protocol.
// This backend provides native Wayland support without GLFW, using the
// wayland-client library via CGo and xdg-shell for window decoration.
package wayland

/*
#cgo pkg-config: wayland-client xkbcommon

#include <wayland-client.h>
#include <xkbcommon/xkbcommon.h>
#include <stdlib.h>
#include <string.h>

// XDG shell protocol headers (generated from XML).
// In a full build, these would be generated from xdg-shell.xml.
// For now, we declare the minimal required types.
struct xdg_wm_base;
struct xdg_surface;
struct xdg_toplevel;

static struct wl_display* lux_wl_display_connect() {
    return wl_display_connect(NULL);
}

static void lux_wl_display_disconnect(struct wl_display* display) {
    wl_display_disconnect(display);
}

static int lux_wl_display_dispatch(struct wl_display* display) {
    return wl_display_dispatch(display);
}

static int lux_wl_display_roundtrip(struct wl_display* display) {
    return wl_display_roundtrip(display);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

func init() {
	runtime.LockOSThread()
}

// Platform implements platform.Platform using Wayland.
type Platform struct {
	display     *C.struct_wl_display
	compositor  *C.struct_wl_compositor
	surface     *C.struct_wl_surface
	seat        *C.struct_wl_seat
	keyboard    *C.struct_wl_keyboard
	pointer     *C.struct_wl_pointer

	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	fullscreen  bool
	width, height int

	// XKB keyboard state
	xkbContext *C.struct_xkb_context
	xkbKeymap  *C.struct_xkb_keymap
	xkbState   *C.struct_xkb_state

	// Clipboard
	clipboard  string
	clipMu     sync.Mutex

	// Frame callback
	frameRequested bool
}

// New creates a new Wayland platform instance.
func New() *Platform {
	return &Platform{}
}

// Init connects to the Wayland display and creates a surface.
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

	// Connect to Wayland display.
	display := C.lux_wl_display_connect()
	if display == nil {
		return fmt.Errorf("wayland: failed to connect to display")
	}
	p.display = display

	// Initialize XKB context for keyboard input.
	p.xkbContext = C.xkb_context_new(C.XKB_CONTEXT_NO_FLAGS)
	if p.xkbContext == nil {
		return fmt.Errorf("wayland: failed to create xkb context")
	}

	// Get registry and bind globals (compositor, seat, xdg_wm_base).
	// In a full implementation, this would use wl_registry_listener callbacks.
	C.lux_wl_display_roundtrip(display)

	return nil
}

// Run enters the Wayland event loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb

	for !p.shouldClose {
		// Dispatch Wayland events.
		ret := C.lux_wl_display_dispatch(p.display)
		if ret < 0 {
			return fmt.Errorf("wayland: display dispatch error")
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

// Destroy releases Wayland resources.
func (p *Platform) Destroy() {
	if p.xkbState != nil {
		C.xkb_state_unref(p.xkbState)
	}
	if p.xkbKeymap != nil {
		C.xkb_keymap_unref(p.xkbKeymap)
	}
	if p.xkbContext != nil {
		C.xkb_context_unref(p.xkbContext)
	}
	if p.display != nil {
		C.lux_wl_display_disconnect(p.display)
		p.display = nil
	}
}

// SetTitle updates the window title via xdg_toplevel_set_title.
func (p *Platform) SetTitle(title string) {
	// TODO: Call xdg_toplevel_set_title when xdg_toplevel is bound.
}

// WindowSize returns the current window size.
func (p *Platform) WindowSize() (int, int) {
	return p.width, p.height
}

// FramebufferSize returns the framebuffer size (same as window size on Wayland,
// scaled by the output scale factor for HiDPI).
func (p *Platform) FramebufferSize() (int, int) {
	return p.width, p.height
}

// ShouldClose returns true if the window close was requested.
func (p *Platform) ShouldClose() bool {
	return p.shouldClose
}

// SetCursor changes the cursor shape via wl_pointer_set_cursor.
func (p *Platform) SetCursor(kind input.CursorKind) {
	p.cursorKind = kind
	// TODO: Load cursor from cursor theme and set via wl_pointer_set_cursor.
}

// SetIMECursorRect positions the IME candidate window.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {
	// TODO: Implement via zwp_text_input_v3 protocol.
}

// SetSize resizes the window.
func (p *Platform) SetSize(w, h int) {
	p.width = w
	p.height = h
	// Wayland clients cannot resize themselves — the compositor controls size.
	// We can set preferred size via xdg_toplevel_set_min_size/set_max_size.
}

// SetFullscreen toggles fullscreen mode via xdg_toplevel_set_fullscreen.
func (p *Platform) SetFullscreen(fullscreen bool) {
	p.fullscreen = fullscreen
	// TODO: Call xdg_toplevel_set_fullscreen or xdg_toplevel_unset_fullscreen.
}

// RequestFrame requests a new frame callback from the compositor.
func (p *Platform) RequestFrame() {
	p.frameRequested = true
	// TODO: Call wl_surface_frame to register a frame callback.
}

// SetClipboard sets the clipboard text via wl_data_source.
func (p *Platform) SetClipboard(text string) error {
	p.clipMu.Lock()
	defer p.clipMu.Unlock()
	p.clipboard = text
	// TODO: Create wl_data_source and offer text/plain via wl_data_device.
	return nil
}

// GetClipboard returns the clipboard text via wl_data_offer.
func (p *Platform) GetClipboard() (string, error) {
	p.clipMu.Lock()
	defer p.clipMu.Unlock()
	// TODO: Read from wl_data_offer via pipe.
	return p.clipboard, nil
}

// CreateWGPUSurface creates a wgpu surface from the Wayland display and surface.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	// wgpu-native creates Wayland surfaces via WGPUSurfaceDescriptorFromWaylandSurface
	// which needs (wl_display*, wl_surface*).
	// Return the wl_surface pointer encoded as uintptr.
	return uintptr(unsafe.Pointer(p.surface))
}

// NativeHandle returns the Wayland display pointer for renderer initialization.
func (p *Platform) NativeHandle() uintptr {
	return uintptr(unsafe.Pointer(p.display))
}

// Ensure imports are used.
var _ = unsafe.Pointer(nil)
