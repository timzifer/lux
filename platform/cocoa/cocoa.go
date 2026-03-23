//go:build darwin && cocoa && !nogui && arm64

// Package cocoa implements platform.Platform using native Cocoa/AppKit via goffi FFI.
// This backend provides direct macOS integration without CGo, enabling CGO_ENABLED=0
// builds required by the gogpu/wgpu pure-Go WebGPU backend.
package cocoa

import (
	"fmt"
	"runtime"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

func init() {
	runtime.LockOSThread()
}

// Cocoa/AppKit constants.
const (
	nsWindowStyleMaskTitled         uint64 = 1
	nsWindowStyleMaskClosable       uint64 = 1 << 1
	nsWindowStyleMaskMiniaturizable uint64 = 1 << 2
	nsWindowStyleMaskResizable      uint64 = 1 << 3
	nsWindowStyleMaskFullScreen     uint64 = 1 << 14

	nsBackingStoreBuffered uint64 = 2

	nsApplicationActivationPolicyRegular uint64 = 0

	nsEventMaskAny uint64 = 0xFFFFFFFFFFFFFFFF

	// NSEvent types.
	nsEventTypeLeftMouseDown  uint64 = 1
	nsEventTypeLeftMouseUp    uint64 = 2
	nsEventTypeRightMouseDown uint64 = 3
	nsEventTypeRightMouseUp   uint64 = 4
	nsEventTypeMouseMoved     uint64 = 5
	nsEventTypeKeyDown        uint64 = 10
	nsEventTypeKeyUp          uint64 = 11
	nsEventTypeScrollWheel    uint64 = 22
)

// Platform implements platform.Platform using native Cocoa/AppKit via FFI.
type Platform struct {
	app        uintptr // NSApplication
	window     uintptr // NSWindow
	view       uintptr // NSView
	metalLayer uintptr // CAMetalLayer

	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
	cursorKind  input.CursorKind
	fullscreen  bool
	width       int
	height      int

	// mainWork receives closures to execute on the main thread.
	// Dialog methods send work here; the event loop picks it up.
	mainWork chan func()

	// Accessibility bridge.
	axBridge *AXBridge

	// Multi-window support.
	windows map[uint32]*windowState
}

// windowState holds per-window resources for secondary windows.
type windowState struct {
	window     uintptr // NSWindow
	view       uintptr // NSView (LuxMetalView)
	metalLayer uintptr // CAMetalLayer
	width      int
	height     int
}

// New creates a new Cocoa platform instance.
func New() *Platform {
	return &Platform{
		mainWork: make(chan func(), 8),
	}
}

// RunOnMainThread schedules fn to execute on the main thread and blocks
// until it completes. Safe to call from any goroutine.
func (p *Platform) RunOnMainThread(fn func()) {
	done := make(chan struct{})
	p.mainWork <- func() {
		fn()
		close(done)
	}
	<-done
}

// Init creates the NSApplication and NSWindow via FFI.
func (p *Platform) Init(cfg platform.Config) error {
	if err := ensureRT(); err != nil {
		return err
	}

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

	pool := newAutoreleasePool()
	defer drainPool(pool)

	// [NSApplication sharedApplication]
	nsApp := getClass("NSApplication")
	p.app = msgSendPtr(nsApp, sel("sharedApplication"))
	if p.app == 0 {
		return fmt.Errorf("cocoa: NSApplication sharedApplication returned nil")
	}

	// [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular]
	msgSendVoid(p.app, sel("setActivationPolicy:"), argUInt64(nsApplicationActivationPolicyRegular))

	// Create window.
	frame := nsRect{
		Origin: nsPoint{X: 100, Y: 100},
		Size:   nsSize{Width: float64(p.width), Height: float64(p.height)},
	}
	styleMask := nsWindowStyleMaskTitled | nsWindowStyleMaskClosable |
		nsWindowStyleMaskResizable | nsWindowStyleMaskMiniaturizable

	nsWindow := getClass("NSWindow")
	win := msgSendPtr(nsWindow, sel("alloc"))
	win = msgSendPtr(win, sel("initWithContentRect:styleMask:backing:defer:"),
		argRect(frame),
		argUInt64(styleMask),
		argUInt64(nsBackingStoreBuffered),
		argBool(false),
	)
	if win == 0 {
		return fmt.Errorf("cocoa: NSWindow creation failed")
	}
	p.window = win

	// Tell the window not to release itself on close so we control lifecycle.
	msgSendVoid(p.window, sel("setReleasedWhenClosed:"), argBool(false))

	// Create CAMetalLayer.
	caMetalLayer := getClass("CAMetalLayer")
	p.metalLayer = msgSendPtr(caMetalLayer, sel("new"))
	if p.metalLayer == 0 {
		return fmt.Errorf("cocoa: CAMetalLayer creation failed")
	}

	// Register a custom NSView subclass (LuxMetalView) that overrides
	// makeBackingLayer to return our CAMetalLayer. This is equivalent to
	// the CGo LuxView class and ensures proper layer-backed compositing.
	luxViewClass := registerLuxViewClass(&p.metalLayer)
	if luxViewClass == 0 {
		return fmt.Errorf("cocoa: failed to register LuxMetalView class")
	}

	// Create the custom view instance.
	view := msgSendPtr(luxViewClass, sel("alloc"))
	// Register metalLayer BEFORE initWithFrame: because makeBackingLayer
	// is called during init and needs to find this view's layer.
	viewMetalLayers.Store(view, p.metalLayer)
	view = msgSendPtr(view, sel("initWithFrame:"), argRect(frame))
	p.view = view
	msgSendVoid(p.view, sel("setWantsLayer:"), argBool(true))

	// Configure window.
	msgSendVoid(p.window, sel("setContentView:"), argPtr(p.view))
	nsTitle := newNSString(title)
	msgSendVoid(p.window, sel("setTitle:"), argPtr(nsTitle))
	msgSendVoid(p.window, sel("setAcceptsMouseMovedEvents:"), argBool(true))

	msgSendVoid(p.window, sel("makeKeyAndOrderFront:"), argPtr(0))

	return nil
}

// Run enters the event polling loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb

	pool := newAutoreleasePool()
	defer drainPool(pool)

	// Activate the application.
	msgSendVoid(p.app, sel("activateIgnoringOtherApps:"), argBool(true))
	msgSendVoid(p.app, sel("finishLaunching"))

	// Create the run loop mode string: NSDefaultRunLoopMode = @"kCFRunLoopDefaultMode"
	runLoopMode := newNSString("kCFRunLoopDefaultMode")

	// Get distantPast for non-blocking event poll.
	nsDate := getClass("NSDate")
	distantPast := msgSendPtr(nsDate, sel("distantPast"))

	selNextEvent := sel("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent := sel("sendEvent:")
	selIsVisible := sel("isVisible")

	// Track window content size for resize detection.
	prevW, prevH := p.WindowSize()

	for !p.shouldClose {
		// Drain an inner autorelease pool each iteration to prevent leaks.
		innerPool := newAutoreleasePool()

		// Poll events.
		event := msgSendPtr(p.app, selNextEvent,
			argUInt64(nsEventMaskAny),
			argPtr(distantPast),
			argPtr(runLoopMode),
			argBool(true),
		)

		if event != 0 {
			// Dispatch event to Cocoa for default handling.
			msgSendVoid(p.app, selSendEvent, argPtr(event))

			// Process event for our callbacks.
			p.processEvent(event)
		}

		// Execute any work queued for the main thread (e.g. dialogs).
		select {
		case work := <-p.mainWork:
			work()
		default:
		}

		// Detect window resize (no delegate, so we poll).
		// With layer-backed mode (makeBackingLayer), the system manages the
		// layer's frame automatically, so we only need to notify the renderer.
		curW, curH := p.WindowSize()
		if curW != prevW || curH != prevH {
			prevW, prevH = curW, curH
			if cb.OnResize != nil {
				fbW, fbH := p.FramebufferSize()
				cb.OnResize(fbW, fbH)
			}
		}

		// Check if main window was closed.
		if !msgSendBool(p.window, selIsVisible) {
			p.shouldClose = true
			if cb.OnClose != nil {
				cb.OnClose()
			}
		}

		// Check if any secondary windows were closed (poll visibility).
		if len(p.windows) > 0 && cb.OnWindowClose != nil {
			for id, ws := range p.windows {
				if !msgSendBool(ws.window, selIsVisible) {
					viewMetalLayers.Delete(ws.view)
					delete(p.windows, id)
					cb.OnWindowClose(id)
				}
			}
		}

		// Frame callback.
		if cb.OnFrame != nil {
			cb.OnFrame()
		}

		drainPool(innerPool)
	}

	return nil
}

// processEvent dispatches a Cocoa event to the appropriate platform callback.
func (p *Platform) processEvent(event uintptr) {
	eventType := msgSendUInt64(event, sel("type"))

	switch eventType {
	case nsEventTypeLeftMouseDown:
		if p.callbacks.OnMouseButton != nil {
			x, y := p.mouseLocation(event)
			p.callbacks.OnMouseButton(x, y, 0, true)
		}
	case nsEventTypeLeftMouseUp:
		if p.callbacks.OnMouseButton != nil {
			x, y := p.mouseLocation(event)
			p.callbacks.OnMouseButton(x, y, 0, false)
		}
	case nsEventTypeRightMouseDown:
		if p.callbacks.OnMouseButton != nil {
			x, y := p.mouseLocation(event)
			p.callbacks.OnMouseButton(x, y, 1, true)
		}
	case nsEventTypeRightMouseUp:
		if p.callbacks.OnMouseButton != nil {
			x, y := p.mouseLocation(event)
			p.callbacks.OnMouseButton(x, y, 1, false)
		}
	case nsEventTypeMouseMoved:
		if p.callbacks.OnMouseMove != nil {
			x, y := p.mouseLocation(event)
			p.callbacks.OnMouseMove(x, y)
		}
	case nsEventTypeScrollWheel:
		if p.callbacks.OnScroll != nil {
			dx := msgSendDouble(event, sel("scrollingDeltaX"))
			dy := msgSendDouble(event, sel("scrollingDeltaY"))
			p.callbacks.OnScroll(float32(dx), float32(dy))
		}
	case nsEventTypeKeyDown:
		p.processKeyEvent(event, 0) // 0 = press
	case nsEventTypeKeyUp:
		p.processKeyEvent(event, 1) // 1 = release
	}
}

// mouseLocation extracts the mouse position from an event, flipping Y for top-left origin.
func (p *Platform) mouseLocation(event uintptr) (float32, float32) {
	loc := msgSendPoint(event, sel("locationInWindow"))
	_, h := p.WindowSize()
	return float32(loc.X), float32(float64(h) - loc.Y)
}

// processKeyEvent handles key down/up events.
func (p *Platform) processKeyEvent(event uintptr, action int) {
	mods := int(msgSendUInt64(event, sel("modifierFlags")))
	goMods := convertModifiers(mods)

	if p.callbacks.OnKey != nil {
		keyCode := msgSendUInt64(event, sel("keyCode"))
		keyName := keyCodeToName(uint16(keyCode))
		p.callbacks.OnKey(keyName, action, goMods)
	}

	// For key-down, also send OnChar if characters are available.
	if action == 0 && p.callbacks.OnChar != nil {
		charsPtr := msgSendPtr(event, sel("characters"))
		if charsPtr != 0 {
			chars := goString(charsPtr)
			for _, ch := range chars {
				p.callbacks.OnChar(ch)
			}
		}
	}
}

// convertModifiers maps NSEventModifierFlags to lux modifier bitmask.
func convertModifiers(nsFlags int) int {
	var mods int
	if nsFlags&(1<<17) != 0 { // NSEventModifierFlagShift
		mods |= 1
	}
	if nsFlags&(1<<18) != 0 { // NSEventModifierFlagControl
		mods |= 2
	}
	if nsFlags&(1<<19) != 0 { // NSEventModifierFlagOption (Alt)
		mods |= 4
	}
	if nsFlags&(1<<20) != 0 { // NSEventModifierFlagCommand (Super)
		mods |= 8
	}
	return mods
}

// keyCodeToName maps macOS virtual key codes to key names.
func keyCodeToName(code uint16) string {
	switch code {
	case 0x00:
		return "A"
	case 0x01:
		return "S"
	case 0x02:
		return "D"
	case 0x03:
		return "F"
	case 0x04:
		return "H"
	case 0x05:
		return "G"
	case 0x06:
		return "Z"
	case 0x07:
		return "X"
	case 0x08:
		return "C"
	case 0x09:
		return "V"
	case 0x0B:
		return "B"
	case 0x0C:
		return "Q"
	case 0x0D:
		return "W"
	case 0x0E:
		return "E"
	case 0x0F:
		return "R"
	case 0x10:
		return "Y"
	case 0x11:
		return "T"
	case 0x12:
		return "1"
	case 0x13:
		return "2"
	case 0x14:
		return "3"
	case 0x15:
		return "4"
	case 0x16:
		return "6"
	case 0x17:
		return "5"
	case 0x19:
		return "9"
	case 0x1A:
		return "7"
	case 0x1C:
		return "8"
	case 0x1D:
		return "0"
	case 0x1E:
		return "BracketRight"
	case 0x1F:
		return "O"
	case 0x20:
		return "U"
	case 0x21:
		return "BracketLeft"
	case 0x22:
		return "I"
	case 0x23:
		return "P"
	case 0x24:
		return "Enter"
	case 0x25:
		return "L"
	case 0x26:
		return "J"
	case 0x28:
		return "K"
	case 0x2C:
		return "Slash"
	case 0x2D:
		return "N"
	case 0x2E:
		return "M"
	case 0x2F:
		return "Period"
	case 0x30:
		return "Tab"
	case 0x31:
		return "Space"
	case 0x33:
		return "Backspace"
	case 0x35:
		return "Escape"
	case 0x37:
		return "Super"
	case 0x38:
		return "Shift"
	case 0x3A:
		return "Alt"
	case 0x3B:
		return "Control"
	case 0x7B:
		return "ArrowLeft"
	case 0x7C:
		return "ArrowRight"
	case 0x7D:
		return "ArrowDown"
	case 0x7E:
		return "ArrowUp"
	case 0x72:
		return "Help"
	case 0x73:
		return "Home"
	case 0x74:
		return "PageUp"
	case 0x75:
		return "Delete"
	case 0x77:
		return "End"
	case 0x79:
		return "PageDown"
	case 0x60:
		return "F5"
	case 0x61:
		return "F6"
	case 0x62:
		return "F7"
	case 0x63:
		return "F3"
	case 0x64:
		return "F8"
	case 0x65:
		return "F9"
	case 0x67:
		return "F11"
	case 0x6D:
		return "F10"
	case 0x6F:
		return "F12"
	case 0x76:
		return "F4"
	case 0x78:
		return "F2"
	case 0x7A:
		return "F1"
	default:
		return fmt.Sprintf("Key%d", code)
	}
}

// A11yBridge returns the macOS accessibility bridge, creating it on first call.
func (p *Platform) A11yBridge() a11y.A11yBridge {
	if p.axBridge == nil && p.view != 0 {
		p.axBridge = NewAXBridge(p.view, nil)
	}
	return p.axBridge
}

// SetA11ySend sets the send function for the accessibility bridge to route actions
// to the app loop.
func (p *Platform) SetA11ySend(send func(any)) {
	if p.axBridge != nil {
		p.axBridge.send = send
	}
}

// Destroy releases Cocoa resources.
func (p *Platform) Destroy() {
	if p.axBridge != nil {
		p.axBridge.Destroy()
		p.axBridge = nil
	}
	if p.window != 0 {
		pool := newAutoreleasePool()
		msgSendVoid(p.window, sel("close"))
		p.window = 0
		drainPool(pool)
	}
}

// SetTitle updates the window title.
func (p *Platform) SetTitle(title string) {
	if p.window == 0 {
		return
	}
	pool := newAutoreleasePool()
	nsTitle := newNSString(title)
	msgSendVoid(p.window, sel("setTitle:"), argPtr(nsTitle))
	msgSendVoid(nsTitle, sel("release"))
	drainPool(pool)
}

// WindowSize returns the window content size.
func (p *Platform) WindowSize() (int, int) {
	if p.window == 0 {
		return 0, 0
	}
	// Get the window frame, then content rect from it.
	frame := msgSendRect(p.window, sel("frame"))
	contentRect := msgSendRect(p.window, sel("contentRectForFrameRect:"), argRect(frame))
	return int(contentRect.Size.Width), int(contentRect.Size.Height)
}

// FramebufferSize returns the framebuffer size (2x on Retina).
func (p *Platform) FramebufferSize() (int, int) {
	w, h := p.WindowSize()
	// On Retina, backingScaleFactor is 2.0.
	scale := msgSendDouble(p.window, sel("backingScaleFactor"))
	if scale < 1 {
		scale = 1
	}
	return int(float64(w) * scale), int(float64(h) * scale)
}

// ShouldClose returns true if the window should close.
func (p *Platform) ShouldClose() bool { return p.shouldClose }

// SetCursor changes the cursor shape.
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
	if p.window == 0 {
		return
	}
	frame := msgSendRect(p.window, sel("frame"))
	frame.Size = nsSize{Width: float64(w), Height: float64(h)}
	msgSendVoid(p.window, sel("setFrame:display:"), argRect(frame), argBool(true))
}

// SetFullscreen toggles fullscreen mode.
func (p *Platform) SetFullscreen(fullscreen bool) {
	if p.window == 0 {
		return
	}
	styleMask := msgSendUInt64(p.window, sel("styleMask"))
	isFS := styleMask&nsWindowStyleMaskFullScreen != 0
	if fullscreen != isFS {
		p.fullscreen = fullscreen
		msgSendVoid(p.window, sel("toggleFullScreen:"), argPtr(0))
	}
}

// RequestFrame marks the view as needing display.
func (p *Platform) RequestFrame() {
	if p.view != 0 {
		msgSendVoid(p.view, sel("setNeedsDisplay:"), argBool(true))
	}
}

// SetClipboard sets the macOS pasteboard.
func (p *Platform) SetClipboard(text string) error {
	pool := newAutoreleasePool()
	defer drainPool(pool)

	nsPasteboard := getClass("NSPasteboard")
	pb := msgSendPtr(nsPasteboard, sel("generalPasteboard"))
	msgSendVoid(pb, sel("clearContents"))

	nsText := newNSString(text)
	defer msgSendVoid(nsText, sel("release"))

	nsType := newNSString("public.utf8-plain-text")
	defer msgSendVoid(nsType, sel("release"))

	msgSendVoid(pb, sel("setString:forType:"), argPtr(nsText), argPtr(nsType))
	return nil
}

// GetClipboard returns the macOS pasteboard text.
func (p *Platform) GetClipboard() (string, error) {
	pool := newAutoreleasePool()
	defer drainPool(pool)

	nsPasteboard := getClass("NSPasteboard")
	pb := msgSendPtr(nsPasteboard, sel("generalPasteboard"))

	nsType := newNSString("public.utf8-plain-text")
	defer msgSendVoid(nsType, sel("release"))

	nsStr := msgSendPtr(pb, sel("stringForType:"), argPtr(nsType))
	if nsStr == 0 {
		return "", nil
	}
	return goString(nsStr), nil
}

// CreateWGPUSurface returns the CAMetalLayer pointer for wgpu surface creation.
func (p *Platform) CreateWGPUSurface(_ uintptr) uintptr {
	return p.metalLayer
}

// NativeHandle returns the CAMetalLayer pointer.
// The wgpu Metal backend expects a CAMetalLayer* as the native handle.
func (p *Platform) NativeHandle() uintptr {
	return p.metalLayer
}

// ── MultiWindowPlatform ──────────────────────────────────────

// Compile-time check that Platform implements MultiWindowPlatform.
var _ platform.MultiWindowPlatform = (*Platform)(nil)

// CreateWindow creates a secondary NSWindow with its own CAMetalLayer.
// Returns the CAMetalLayer pointer as the native handle for wgpu surface creation.
func (p *Platform) CreateWindow(id uint32, cfg platform.Config) (uintptr, error) {
	title := cfg.Title
	if title == "" {
		title = "Lux Window"
	}
	w, h := cfg.Width, cfg.Height
	if w <= 0 {
		w = 640
	}
	if h <= 0 {
		h = 480
	}

	pool := newAutoreleasePool()
	defer drainPool(pool)

	frame := nsRect{
		Origin: nsPoint{X: 150, Y: 150},
		Size:   nsSize{Width: float64(w), Height: float64(h)},
	}
	styleMask := nsWindowStyleMaskTitled | nsWindowStyleMaskClosable |
		nsWindowStyleMaskResizable | nsWindowStyleMaskMiniaturizable

	win := msgSendPtr(getClass("NSWindow"), sel("alloc"))
	win = msgSendPtr(win, sel("initWithContentRect:styleMask:backing:defer:"),
		argRect(frame),
		argUInt64(styleMask),
		argUInt64(nsBackingStoreBuffered),
		argBool(false),
	)
	if win == 0 {
		return 0, fmt.Errorf("cocoa: CreateWindow %d failed", id)
	}
	msgSendVoid(win, sel("setReleasedWhenClosed:"), argBool(false))

	// Create CAMetalLayer for this window.
	metalLayer := msgSendPtr(getClass("CAMetalLayer"), sel("new"))
	if metalLayer == 0 {
		return 0, fmt.Errorf("cocoa: CAMetalLayer for window %d failed", id)
	}

	// Create LuxMetalView with makeBackingLayer returning this window's metalLayer.
	// We need a per-window metalLayer pointer for the callback.
	ws := &windowState{
		window:     win,
		metalLayer: metalLayer,
		width:      w,
		height:     h,
	}

	luxViewClass := registerLuxViewClass(&ws.metalLayer)
	if luxViewClass == 0 {
		return 0, fmt.Errorf("cocoa: LuxMetalView class for window %d failed", id)
	}

	view := msgSendPtr(luxViewClass, sel("alloc"))
	// Register metalLayer BEFORE initWithFrame: because makeBackingLayer
	// is called during init and needs to find this view's layer.
	viewMetalLayers.Store(view, metalLayer)
	view = msgSendPtr(view, sel("initWithFrame:"), argRect(frame))
	ws.view = view
	msgSendVoid(view, sel("setWantsLayer:"), argBool(true))

	msgSendVoid(win, sel("setContentView:"), argPtr(view))
	nsTitle := newNSString(title)
	msgSendVoid(win, sel("setTitle:"), argPtr(nsTitle))
	msgSendVoid(win, sel("makeKeyAndOrderFront:"), argPtr(0))

	if p.windows == nil {
		p.windows = make(map[uint32]*windowState)
	}
	p.windows[id] = ws

	return metalLayer, nil
}

// DestroyWindow closes and releases a secondary window.
func (p *Platform) DestroyWindow(id uint32) {
	if p.windows == nil {
		return
	}
	ws, ok := p.windows[id]
	if !ok {
		return
	}
	pool := newAutoreleasePool()
	viewMetalLayers.Delete(ws.view)
	msgSendVoid(ws.window, sel("close"))
	drainPool(pool)
	delete(p.windows, id)
}

// SetWindowTitle updates the title of a secondary window.
func (p *Platform) SetWindowTitle(id uint32, title string) {
	if p.windows == nil {
		return
	}
	ws, ok := p.windows[id]
	if !ok {
		return
	}
	pool := newAutoreleasePool()
	nsTitle := newNSString(title)
	msgSendVoid(ws.window, sel("setTitle:"), argPtr(nsTitle))
	msgSendVoid(nsTitle, sel("release"))
	drainPool(pool)
}

// WindowSizeByID returns the content size of a secondary window.
func (p *Platform) WindowSizeByID(id uint32) (int, int) {
	if p.windows == nil {
		return 0, 0
	}
	ws, ok := p.windows[id]
	if !ok {
		return 0, 0
	}
	frame := msgSendRect(ws.window, sel("frame"))
	contentRect := msgSendRect(ws.window, sel("contentRectForFrameRect:"), argRect(frame))
	return int(contentRect.Size.Width), int(contentRect.Size.Height)
}

// FramebufferSizeByID returns the framebuffer size of a secondary window.
func (p *Platform) FramebufferSizeByID(id uint32) (int, int) {
	w, h := p.WindowSizeByID(id)
	if p.windows == nil {
		return w, h
	}
	ws, ok := p.windows[id]
	if !ok {
		return w, h
	}
	scale := msgSendDouble(ws.window, sel("backingScaleFactor"))
	if scale < 1 {
		scale = 1
	}
	return int(float64(w) * scale), int(float64(h) * scale)
}

