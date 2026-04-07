//go:build drm && !nogui

// Package drm implements platform.Platform using Linux DRM/KMS (Direct Rendering Manager /
// Kernel Mode Setting) for rendering without a window manager (RFC §7.3).
//
// This backend is designed for embedded systems, kiosk applications, and headless
// Linux systems where no X11 or Wayland compositor is available.
//
// It uses:
//   - libdrm for display enumeration and mode setting
//   - libinput + libudev for keyboard, mouse, and touch input
//   - VK_KHR_display via wgpu for GPU rendering
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
#include <poll.h>
#include <linux/input-event-codes.h>
#include <errno.h>
#include <string.h>
#include <time.h>

// ── DRM helpers ──────────────────────────────────────────────────────

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

// ── libinput helpers ─────────────────────────────────────────────────

static int lux_open_restricted(const char *path, int flags, void *user_data) {
    int fd = open(path, flags);
    return fd < 0 ? -errno : fd;
}

static void lux_close_restricted(int fd, void *user_data) {
    close(fd);
}

static const struct libinput_interface lux_libinput_iface = {
    .open_restricted = lux_open_restricted,
    .close_restricted = lux_close_restricted,
};

static struct libinput* lux_libinput_create(struct udev* udev) {
    struct libinput* li = libinput_udev_create_context(&lux_libinput_iface, NULL, udev);
    if (li == NULL) return NULL;
    if (libinput_udev_assign_seat(li, "seat0") != 0) {
        libinput_unref(li);
        return NULL;
    }
    return li;
}

static int lux_libinput_get_fd(struct libinput* li) {
    return libinput_get_fd(li);
}

// ── Poll helper ──────────────────────────────────────────────────────

// lux_poll_input polls the libinput fd with timeout_ms. Returns >0 if ready.
static int lux_poll_input(int fd, int timeout_ms) {
    struct pollfd pfd = { .fd = fd, .events = POLLIN };
    return poll(&pfd, 1, timeout_ms);
}

// ── Monotonic clock ──────────────────────────────────────────────────

static uint64_t lux_clock_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000 + (uint64_t)ts.tv_nsec / 1000000;
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
	connectorID uint32
	encoder     *C.drmModeEncoder
	crtc        *C.drmModeCrtc
	savedCrtc   *C.drmModeCrtc

	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	width, height int
	clipboard   string

	// libinput for input events.
	liContext    *C.struct_libinput
	liFd        C.int
	udevCtx     *C.struct_udev

	// Mouse state for absolute positioning.
	mouseX, mouseY float32

	// Frame pacing.
	frameRequested bool
}

// New creates a new DRM/KMS platform instance.
func New() *Platform {
	return &Platform{fd: -1, liFd: -1}
}

// HasCompositor implements platform.CompositorChecker.
// DRM/KMS always runs without a compositor.
func (p *Platform) HasCompositor() bool { return false }

// Init opens the DRM device, finds a suitable connector and CRTC, and initializes libinput.
func (p *Platform) Init(cfg platform.Config) error {
	p.config = cfg

	// Try /dev/dri/card0, then card1.
	for _, path := range []string{"/dev/dri/card0", "/dev/dri/card1"} {
		cPath := C.CString(path)
		fd := C.lux_drm_open(cPath)
		C.free(unsafe.Pointer(cPath))
		if fd >= 0 {
			p.fd = fd
			break
		}
	}
	if p.fd < 0 {
		return fmt.Errorf("drm: failed to open /dev/dri/card*")
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
			p.connectorID = uint32(connID)
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
		// Try finding any compatible encoder.
		encoderIDs := unsafe.Slice(p.connector.encoders, p.connector.count_encoders)
		for _, encID := range encoderIDs {
			enc := C.lux_drm_get_encoder(p.fd, encID)
			if enc != nil {
				p.encoder = enc
				break
			}
		}
	}
	if p.encoder == nil {
		return fmt.Errorf("drm: no encoder found for connector %d", p.connectorID)
	}

	// Get CRTC and save the current state for restoration on exit.
	p.crtc = C.lux_drm_get_crtc(p.fd, p.encoder.crtc_id)
	if p.crtc != nil {
		p.savedCrtc = C.lux_drm_get_crtc(p.fd, p.encoder.crtc_id)
	}

	// Initialize libinput for input handling.
	if err := p.initLibinput(); err != nil {
		// Input is non-fatal — log but continue (embedded displays may not have input).
		fmt.Printf("drm: libinput init: %v (input disabled)\n", err)
	}

	p.frameRequested = true
	return nil
}

// initLibinput sets up the libinput context and assigns seat0.
func (p *Platform) initLibinput() error {
	p.udevCtx = C.udev_new()
	if p.udevCtx == nil {
		return fmt.Errorf("udev_new failed")
	}

	p.liContext = C.lux_libinput_create(p.udevCtx)
	if p.liContext == nil {
		return fmt.Errorf("libinput_udev_create_context failed (need seat0 access)")
	}

	p.liFd = C.lux_libinput_get_fd(p.liContext)
	return nil
}

// Run enters the DRM rendering loop with input polling.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb
	p.mouseX = float32(p.width) / 2
	p.mouseY = float32(p.height) / 2

	for !p.shouldClose {
		// Poll libinput with a short timeout to avoid busy-waiting.
		// If a frame is requested, use 0ms (non-blocking) to render immediately.
		timeout := 16 // ~60fps cadence
		if p.frameRequested {
			timeout = 0
		}

		if p.liContext != nil && p.liFd >= 0 {
			C.lux_poll_input(p.liFd, C.int(timeout))
			p.dispatchInput()
		} else if timeout > 0 {
			// No libinput — use a simple sleep for frame pacing.
			C.lux_poll_input(-1, C.int(timeout))
		}

		if p.frameRequested {
			p.frameRequested = false
			if cb.OnFrame != nil {
				cb.OnFrame()
			}
			// Always request another frame (continuous rendering for DRM).
			p.frameRequested = true
		}
	}

	return nil
}

// dispatchInput reads all pending libinput events and dispatches callbacks.
func (p *Platform) dispatchInput() {
	C.libinput_dispatch(p.liContext)

	for {
		ev := C.libinput_get_event(p.liContext)
		if ev == nil {
			break
		}
		p.handleEvent(ev)
		C.libinput_event_destroy(ev)
	}
}

// handleEvent processes a single libinput event.
func (p *Platform) handleEvent(ev *C.struct_libinput_event) {
	evType := C.libinput_event_get_type(ev)
	switch evType {
	// ── Keyboard ──
	case C.LIBINPUT_EVENT_KEYBOARD_KEY:
		p.handleKeyboard(C.libinput_event_get_keyboard_event(ev))

	// ── Pointer (mouse) motion ──
	case C.LIBINPUT_EVENT_POINTER_MOTION:
		p.handlePointerMotion(C.libinput_event_get_pointer_event(ev))
	case C.LIBINPUT_EVENT_POINTER_MOTION_ABSOLUTE:
		p.handlePointerMotionAbsolute(C.libinput_event_get_pointer_event(ev))

	// ── Pointer button ──
	case C.LIBINPUT_EVENT_POINTER_BUTTON:
		p.handlePointerButton(C.libinput_event_get_pointer_event(ev))

	// ── Scroll ──
	case C.LIBINPUT_EVENT_POINTER_SCROLL_WHEEL,
		C.LIBINPUT_EVENT_POINTER_SCROLL_FINGER,
		C.LIBINPUT_EVENT_POINTER_SCROLL_CONTINUOUS:
		p.handlePointerScroll(C.libinput_event_get_pointer_event(ev))

	// ── Touch ──
	case C.LIBINPUT_EVENT_TOUCH_DOWN:
		p.handleTouchDown(C.libinput_event_get_touch_event(ev))
	case C.LIBINPUT_EVENT_TOUCH_UP:
		p.handleTouchUp(C.libinput_event_get_touch_event(ev))
	case C.LIBINPUT_EVENT_TOUCH_MOTION:
		p.handleTouchMotion(C.libinput_event_get_touch_event(ev))
	}
}

// ── Keyboard handling ────────────────────────────────────────────────

func (p *Platform) handleKeyboard(ev *C.struct_libinput_event_keyboard) {
	if ev == nil || p.callbacks.OnKey == nil {
		return
	}

	key := C.libinput_event_keyboard_get_key(ev)
	state := C.libinput_event_keyboard_get_key_state(ev)

	var action int
	if state == C.LIBINPUT_KEY_STATE_PRESSED {
		action = 0 // press
	} else {
		action = 1 // release
	}

	keyName := linuxKeyName(uint32(key))
	if keyName == "" {
		return
	}

	p.callbacks.OnKey(keyName, action, 0)
}

// ── Pointer motion ───────────────────────────────────────────────────

func (p *Platform) handlePointerMotion(ev *C.struct_libinput_event_pointer) {
	if ev == nil {
		return
	}

	dx := float32(C.libinput_event_pointer_get_dx(ev))
	dy := float32(C.libinput_event_pointer_get_dy(ev))
	p.mouseX += dx
	p.mouseY += dy

	// Clamp to screen bounds.
	if p.mouseX < 0 {
		p.mouseX = 0
	}
	if p.mouseY < 0 {
		p.mouseY = 0
	}
	if p.mouseX >= float32(p.width) {
		p.mouseX = float32(p.width) - 1
	}
	if p.mouseY >= float32(p.height) {
		p.mouseY = float32(p.height) - 1
	}

	if p.callbacks.OnMouseMove != nil {
		p.callbacks.OnMouseMove(p.mouseX, p.mouseY)
	}
}

func (p *Platform) handlePointerMotionAbsolute(ev *C.struct_libinput_event_pointer) {
	if ev == nil {
		return
	}

	p.mouseX = float32(C.libinput_event_pointer_get_absolute_x_transformed(ev, C.uint32_t(p.width)))
	p.mouseY = float32(C.libinput_event_pointer_get_absolute_y_transformed(ev, C.uint32_t(p.height)))

	if p.callbacks.OnMouseMove != nil {
		p.callbacks.OnMouseMove(p.mouseX, p.mouseY)
	}
}

// ── Pointer button ───────────────────────────────────────────────────

func (p *Platform) handlePointerButton(ev *C.struct_libinput_event_pointer) {
	if ev == nil || p.callbacks.OnMouseButton == nil {
		return
	}

	btn := C.libinput_event_pointer_get_button(ev)
	state := C.libinput_event_pointer_get_button_state(ev)
	pressed := state == C.LIBINPUT_BUTTON_STATE_PRESSED

	// Map Linux button codes to framework button IDs.
	var button int
	switch btn {
	case C.BTN_LEFT:
		button = 0
	case C.BTN_RIGHT:
		button = 1
	case C.BTN_MIDDLE:
		button = 2
	default:
		return
	}

	p.callbacks.OnMouseButton(p.mouseX, p.mouseY, button, pressed)
}

// ── Scroll ───────────────────────────────────────────────────────────

func (p *Platform) handlePointerScroll(ev *C.struct_libinput_event_pointer) {
	if ev == nil || p.callbacks.OnScroll == nil {
		return
	}

	var dx, dy float32
	if C.libinput_event_pointer_has_axis(ev, C.LIBINPUT_POINTER_AXIS_SCROLL_HORIZONTAL) != 0 {
		dx = float32(C.libinput_event_pointer_get_scroll_value(ev, C.LIBINPUT_POINTER_AXIS_SCROLL_HORIZONTAL))
	}
	if C.libinput_event_pointer_has_axis(ev, C.LIBINPUT_POINTER_AXIS_SCROLL_VERTICAL) != 0 {
		dy = float32(C.libinput_event_pointer_get_scroll_value(ev, C.LIBINPUT_POINTER_AXIS_SCROLL_VERTICAL))
	}

	if dx != 0 || dy != 0 {
		p.callbacks.OnScroll(dx, dy)
	}
}

// ── Touch handling ───────────────────────────────────────────────────

func (p *Platform) handleTouchDown(ev *C.struct_libinput_event_touch) {
	if ev == nil || p.callbacks.OnMouseButton == nil {
		return
	}

	x := float32(C.libinput_event_touch_get_x_transformed(ev, C.uint32_t(p.width)))
	y := float32(C.libinput_event_touch_get_y_transformed(ev, C.uint32_t(p.height)))
	p.mouseX = x
	p.mouseY = y

	if p.callbacks.OnMouseMove != nil {
		p.callbacks.OnMouseMove(x, y)
	}
	p.callbacks.OnMouseButton(x, y, 0, true)
}

func (p *Platform) handleTouchUp(ev *C.struct_libinput_event_touch) {
	if ev == nil || p.callbacks.OnMouseButton == nil {
		return
	}
	p.callbacks.OnMouseButton(p.mouseX, p.mouseY, 0, false)
}

func (p *Platform) handleTouchMotion(ev *C.struct_libinput_event_touch) {
	if ev == nil || p.callbacks.OnMouseMove == nil {
		return
	}

	x := float32(C.libinput_event_touch_get_x_transformed(ev, C.uint32_t(p.width)))
	y := float32(C.libinput_event_touch_get_y_transformed(ev, C.uint32_t(p.height)))
	p.mouseX = x
	p.mouseY = y
	p.callbacks.OnMouseMove(x, y)
}

// ── Platform interface ───────────────────────────────────────────────

// Destroy releases DRM resources, restores the original CRTC, and shuts down libinput.
func (p *Platform) Destroy() {
	if p.liContext != nil {
		C.libinput_unref(p.liContext)
		p.liContext = nil
	}
	if p.savedCrtc != nil {
		C.drmModeFreeCrtc(p.savedCrtc)
	}
	if p.crtc != nil && p.crtc != p.savedCrtc {
		C.drmModeFreeCrtc(p.crtc)
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

// SetCursor stores the cursor kind but does not render it (no hardware cursor support yet).
func (p *Platform) SetCursor(kind input.CursorKind) { p.cursorKind = kind }

// SetIMECursorRect is a no-op for DRM.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {}

// SetSize is a no-op for DRM (display resolution is fixed by the mode).
func (p *Platform) SetSize(w, h int) {}

// SetFullscreen is a no-op for DRM (always fullscreen).
func (p *Platform) SetFullscreen(fullscreen bool) {}

// RequestFrame signals that a new frame should be rendered.
func (p *Platform) RequestFrame() {
	p.frameRequested = true
}

// SetClipboard stores text in an in-memory clipboard.
func (p *Platform) SetClipboard(text string) error {
	p.clipboard = text
	return nil
}

// GetClipboard returns the in-memory clipboard text.
func (p *Platform) GetClipboard() (string, error) {
	return p.clipboard, nil
}

// CreateWGPUSurface returns 0 for DRM — surface creation uses DRMfd/DRMConnectorID instead.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	return 0
}

// NativeHandle returns 0 for DRM (no windowing-system handle).
func (p *Platform) NativeHandle() uintptr {
	return 0
}

// DRMfd returns the DRM device file descriptor for VK_KHR_display surface creation.
func (p *Platform) DRMfd() int {
	return int(p.fd)
}

// DRMConnectorID returns the DRM connector ID for display selection.
func (p *Platform) DRMConnectorID() uint32 {
	return p.connectorID
}

// ── Key name mapping ─────────────────────────────────────────────────

// linuxKeyName maps Linux input-event-codes (KEY_*) to platform key names.
func linuxKeyName(code uint32) string {
	switch code {
	case C.KEY_ESC:
		return "Escape"
	case C.KEY_1:
		return "1"
	case C.KEY_2:
		return "2"
	case C.KEY_3:
		return "3"
	case C.KEY_4:
		return "4"
	case C.KEY_5:
		return "5"
	case C.KEY_6:
		return "6"
	case C.KEY_7:
		return "7"
	case C.KEY_8:
		return "8"
	case C.KEY_9:
		return "9"
	case C.KEY_0:
		return "0"
	case C.KEY_MINUS:
		return "-"
	case C.KEY_EQUAL:
		return "="
	case C.KEY_BACKSPACE:
		return "Backspace"
	case C.KEY_TAB:
		return "Tab"
	case C.KEY_Q:
		return "Q"
	case C.KEY_W:
		return "W"
	case C.KEY_E:
		return "E"
	case C.KEY_R:
		return "R"
	case C.KEY_T:
		return "T"
	case C.KEY_Y:
		return "Y"
	case C.KEY_U:
		return "U"
	case C.KEY_I:
		return "I"
	case C.KEY_O:
		return "O"
	case C.KEY_P:
		return "P"
	case C.KEY_LEFTBRACE:
		return "["
	case C.KEY_RIGHTBRACE:
		return "]"
	case C.KEY_ENTER:
		return "Enter"
	case C.KEY_LEFTCTRL:
		return "LeftCtrl"
	case C.KEY_A:
		return "A"
	case C.KEY_S:
		return "S"
	case C.KEY_D:
		return "D"
	case C.KEY_F:
		return "F"
	case C.KEY_G:
		return "G"
	case C.KEY_H:
		return "H"
	case C.KEY_J:
		return "J"
	case C.KEY_K:
		return "K"
	case C.KEY_L:
		return "L"
	case C.KEY_SEMICOLON:
		return ";"
	case C.KEY_APOSTROPHE:
		return "'"
	case C.KEY_LEFTSHIFT:
		return "LeftShift"
	case C.KEY_BACKSLASH:
		return "\\"
	case C.KEY_Z:
		return "Z"
	case C.KEY_X:
		return "X"
	case C.KEY_C:
		return "C"
	case C.KEY_V:
		return "V"
	case C.KEY_B:
		return "B"
	case C.KEY_N:
		return "N"
	case C.KEY_M:
		return "M"
	case C.KEY_COMMA:
		return ","
	case C.KEY_DOT:
		return "."
	case C.KEY_SLASH:
		return "/"
	case C.KEY_RIGHTSHIFT:
		return "RightShift"
	case C.KEY_LEFTALT:
		return "LeftAlt"
	case C.KEY_SPACE:
		return "Space"
	case C.KEY_CAPSLOCK:
		return "CapsLock"
	case C.KEY_F1:
		return "F1"
	case C.KEY_F2:
		return "F2"
	case C.KEY_F3:
		return "F3"
	case C.KEY_F4:
		return "F4"
	case C.KEY_F5:
		return "F5"
	case C.KEY_F6:
		return "F6"
	case C.KEY_F7:
		return "F7"
	case C.KEY_F8:
		return "F8"
	case C.KEY_F9:
		return "F9"
	case C.KEY_F10:
		return "F10"
	case C.KEY_F11:
		return "F11"
	case C.KEY_F12:
		return "F12"
	case C.KEY_RIGHTCTRL:
		return "RightCtrl"
	case C.KEY_RIGHTALT:
		return "RightAlt"
	case C.KEY_HOME:
		return "Home"
	case C.KEY_UP:
		return "Up"
	case C.KEY_PAGEUP:
		return "PageUp"
	case C.KEY_LEFT:
		return "Left"
	case C.KEY_RIGHT:
		return "Right"
	case C.KEY_END:
		return "End"
	case C.KEY_DOWN:
		return "Down"
	case C.KEY_PAGEDOWN:
		return "PageDown"
	case C.KEY_INSERT:
		return "Insert"
	case C.KEY_DELETE:
		return "Delete"
	case C.KEY_LEFTMETA:
		return "LeftSuper"
	case C.KEY_RIGHTMETA:
		return "RightSuper"
	default:
		return ""
	}
}
