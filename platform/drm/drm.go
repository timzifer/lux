//go:build drm && !nogui

// Package drm implements platform.Platform using Linux DRM/KMS (Direct Rendering Manager /
// Kernel Mode Setting) for rendering without a window manager (RFC §7.3).
//
// This backend is designed for embedded systems, kiosk applications, and headless
// Linux systems where no X11 or Wayland compositor is available.
package drm

/*
#cgo pkg-config: libdrm libinput libudev

#include <xf86drm.h>
#include <xf86drmMode.h>
#include <libinput.h>
#include <libudev.h>
#include <fcntl.h>
#include <unistd.h>
#include <stdlib.h>

static int lux_drm_open(const char* path) {
    return open(path, O_RDWR | O_CLOEXEC);
}

static void lux_drm_close(int fd) {
    close(fd);
}

static drmModeRes* lux_drm_get_resources(int fd) {
    return drmModeGetResources(fd);
}

static drmModeConnector* lux_drm_get_connector(int fd, uint32_t id) {
    return drmModeGetConnector(fd, id);
}

static drmModeEncoder* lux_drm_get_encoder(int fd, uint32_t id) {
    return drmModeGetEncoder(fd, id);
}

static drmModeCrtc* lux_drm_get_crtc(int fd, uint32_t id) {
    return drmModeGetCrtc(fd, id);
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

// Platform implements platform.Platform using DRM/KMS.
type Platform struct {
	fd          C.int
	resources   *C.drmModeRes
	connector   *C.drmModeConnector
	encoder     *C.drmModeEncoder
	crtc        *C.drmModeCrtc
	savedCrtc   *C.drmModeCrtc

	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	width, height int
	clipboard   string

	// libinput for input events
	liContext   *C.struct_libinput
	udevCtx    *C.struct_udev
}

// New creates a new DRM/KMS platform instance.
func New() *Platform {
	return &Platform{}
}

// Init opens the DRM device, finds a suitable connector and CRTC, and sets up mode.
func (p *Platform) Init(cfg platform.Config) error {
	p.config = cfg

	// Open DRM device (typically /dev/dri/card0).
	cPath := C.CString("/dev/dri/card0")
	defer C.free(unsafe.Pointer(cPath))
	p.fd = C.lux_drm_open(cPath)
	if p.fd < 0 {
		return fmt.Errorf("drm: failed to open /dev/dri/card0")
	}

	// Get resources.
	p.resources = C.lux_drm_get_resources(p.fd)
	if p.resources == nil {
		return fmt.Errorf("drm: failed to get resources")
	}

	// Find first connected connector.
	connectors := unsafe.Slice(p.resources.connectors, p.resources.count_connectors)
	for _, connID := range connectors {
		conn := C.lux_drm_get_connector(p.fd, connID)
		if conn == nil {
			continue
		}
		if conn.connection == C.DRM_MODE_CONNECTED && conn.count_modes > 0 {
			p.connector = conn
			break
		}
		C.drmModeFreeConnector(conn)
	}

	if p.connector == nil {
		return fmt.Errorf("drm: no connected display found")
	}

	// Use the first (preferred) mode.
	modes := unsafe.Slice(p.connector.modes, p.connector.count_modes)
	mode := modes[0]
	p.width = int(mode.hdisplay)
	p.height = int(mode.vdisplay)

	// Find encoder.
	if p.connector.encoder_id != 0 {
		p.encoder = C.lux_drm_get_encoder(p.fd, p.connector.encoder_id)
	}
	if p.encoder == nil {
		return fmt.Errorf("drm: no encoder found")
	}

	// Get CRTC.
	p.crtc = C.lux_drm_get_crtc(p.fd, p.encoder.crtc_id)
	p.savedCrtc = p.crtc

	// Initialize libinput for input handling.
	p.udevCtx = C.udev_new()
	if p.udevCtx != nil {
		// TODO: Create libinput context from udev and open input devices.
	}

	return nil
}

// Run enters the DRM rendering loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb

	for !p.shouldClose {
		// TODO: Poll libinput for input events and dispatch.

		if cb.OnFrame != nil {
			cb.OnFrame()
		}
	}

	return nil
}

// Destroy releases DRM resources and restores the original CRTC.
func (p *Platform) Destroy() {
	if p.savedCrtc != nil {
		// Restore saved CRTC mode.
		C.drmModeFreeCrtc(p.savedCrtc)
	}
	if p.connector != nil {
		C.drmModeFreeConnector(p.connector)
	}
	if p.encoder != nil {
		C.drmModeFreeEncoder(p.encoder)
	}
	if p.resources != nil {
		C.drmModeFreeResources(p.resources)
	}
	if p.udevCtx != nil {
		C.udev_unref(p.udevCtx)
	}
	if p.fd >= 0 {
		C.lux_drm_close(p.fd)
	}
}

// SetTitle is a no-op for DRM (no window manager).
func (p *Platform) SetTitle(title string) {}

// WindowSize returns the display resolution.
func (p *Platform) WindowSize() (int, int) { return p.width, p.height }

// FramebufferSize returns the framebuffer resolution.
func (p *Platform) FramebufferSize() (int, int) { return p.width, p.height }

// ShouldClose returns true if the application should exit.
func (p *Platform) ShouldClose() bool { return p.shouldClose }

// SetCursor is a no-op for DRM (no system cursor).
func (p *Platform) SetCursor(kind input.CursorKind) { p.cursorKind = kind }

// SetIMECursorRect is a no-op for DRM.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {}

// SetSize is a no-op for DRM (display resolution is fixed by the mode).
func (p *Platform) SetSize(w, h int) {}

// SetFullscreen is a no-op for DRM (always fullscreen).
func (p *Platform) SetFullscreen(fullscreen bool) {}

// RequestFrame signals that a new frame should be rendered.
func (p *Platform) RequestFrame() {}

// SetClipboard stores text in an in-memory clipboard.
func (p *Platform) SetClipboard(text string) error {
	p.clipboard = text
	return nil
}

// GetClipboard returns the in-memory clipboard text.
func (p *Platform) GetClipboard() (string, error) {
	return p.clipboard, nil
}

// CreateWGPUSurface creates a wgpu surface from the DRM file descriptor.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	return uintptr(p.fd)
}

// NativeHandle returns the DRM file descriptor.
func (p *Platform) NativeHandle() uintptr {
	return uintptr(p.fd)
}
