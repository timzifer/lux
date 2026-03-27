//go:build darwin && cocoa && !nogui && !arm64

// Package cocoa implements platform.Platform using native Cocoa/AppKit via CGo.
// This backend provides direct macOS integration without GLFW.
package cocoa

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore

#include <Cocoa/Cocoa.h>
#include <QuartzCore/CAMetalLayer.h>

// Forward declarations for the Objective-C implementation.
void* lux_cocoa_init(const char* title, int width, int height);
void lux_cocoa_run(void* handle);
void lux_cocoa_destroy(void* handle);
void lux_cocoa_set_title(void* handle, const char* title);
void lux_cocoa_get_size(void* handle, int* width, int* height);
void lux_cocoa_set_size(void* handle, int width, int height);
void lux_cocoa_set_fullscreen(void* handle, int fullscreen);
void lux_cocoa_set_clipboard(const char* text);
const char* lux_cocoa_get_clipboard(void);
void* lux_cocoa_get_metal_layer(void* handle);
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

// Platform implements platform.Platform using native Cocoa/AppKit.
type Platform struct {
	handle      unsafe.Pointer
	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	fullscreen  bool
	width, height int
}

// New creates a new Cocoa platform instance.
func New() *Platform {
	return &Platform{}
}

// Init creates the NSApplication and NSWindow.
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

	title := cfg.Title
	if title == "" {
		title = "lux"
	}

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	p.handle = C.lux_cocoa_init(cTitle, C.int(p.width), C.int(p.height))
	if p.handle == nil {
		return fmt.Errorf("cocoa: failed to initialize")
	}

	return nil
}

// Run enters the NSApplication run loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb
	C.lux_cocoa_run(p.handle)
	return nil
}

// Destroy releases Cocoa resources.
func (p *Platform) Destroy() {
	if p.handle != nil {
		C.lux_cocoa_destroy(p.handle)
		p.handle = nil
	}
}

// SetTitle updates the window title.
func (p *Platform) SetTitle(title string) {
	if p.handle == nil {
		return
	}
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.lux_cocoa_set_title(p.handle, cTitle)
}

// WindowSize returns the window size.
func (p *Platform) WindowSize() (int, int) {
	if p.handle == nil {
		return 0, 0
	}
	var w, h C.int
	C.lux_cocoa_get_size(p.handle, &w, &h)
	return int(w), int(h)
}

// FramebufferSize returns the framebuffer size (2x on Retina).
func (p *Platform) FramebufferSize() (int, int) {
	// On Retina displays, the framebuffer is 2x the window size.
	w, h := p.WindowSize()
	return w * 2, h * 2
}

// ShouldClose returns true if the window should close.
func (p *Platform) ShouldClose() bool { return p.shouldClose }

// SetCursor changes the cursor shape via NSCursor.
func (p *Platform) SetCursor(kind input.CursorKind) {
	p.cursorKind = kind
	// TODO: Map CursorKind to NSCursor and call [cursor set].
}

// SetIMECursorRect positions the IME candidate window.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {
	// TODO: Implement via NSTextInputClient.
}

// SetSize resizes the window.
func (p *Platform) SetSize(w, h int) {
	if p.handle != nil {
		C.lux_cocoa_set_size(p.handle, C.int(w), C.int(h))
	}
}

// SetFullscreen toggles fullscreen mode via NSWindow toggleFullScreen.
func (p *Platform) SetFullscreen(fullscreen bool) {
	if p.handle != nil && fullscreen != p.fullscreen {
		p.fullscreen = fullscreen
		C.lux_cocoa_set_fullscreen(p.handle, boolToInt(fullscreen))
	}
}

// RequestFrame marks the view as needing display.
func (p *Platform) RequestFrame() {
	// TODO: Call [view setNeedsDisplay:YES].
}

// SetClipboard sets the macOS pasteboard.
func (p *Platform) SetClipboard(text string) error {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.lux_cocoa_set_clipboard(cText)
	return nil
}

// GetClipboard returns the macOS pasteboard text.
func (p *Platform) GetClipboard() (string, error) {
	cText := C.lux_cocoa_get_clipboard()
	if cText == nil {
		return "", nil
	}
	return C.GoString(cText), nil
}

// CreateWGPUSurface creates a wgpu surface via CAMetalLayer.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	if p.handle == nil {
		return 0
	}
	// Return the CAMetalLayer pointer — wgpu-native uses it to create
	// a WGPUSurfaceDescriptorFromMetalLayer.
	return uintptr(C.lux_cocoa_get_metal_layer(p.handle))
}

// NativeHandle returns the CAMetalLayer pointer.
// The wgpu Metal backend expects a CAMetalLayer* as the native handle.
func (p *Platform) NativeHandle() uintptr {
	if p.handle == nil {
		return 0
	}
	return uintptr(C.lux_cocoa_get_metal_layer(p.handle))
}

func boolToInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
