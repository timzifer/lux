//go:build windows && !nogui

// Package windows implements a native Win32 platform handler for M1.
package windows

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

const (
	csVRedraw = 0x0001
	csHRedraw = 0x0002

	cwUseDefault = 0x80000000

	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000

	wmDestroy      = 0x0002
	wmSize         = 0x0005
	wmEraseBkgnd     = 0x0014
	wmClose          = 0x0010
	wmEnterSizeMove  = 0x0231
	wmExitSizeMove   = 0x0232
	wmTimer          = 0x0113
	wmQuit         = 0x0012
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmRButtonDown  = 0x0204
	wmRButtonUp    = 0x0205
	wmMButtonDown  = 0x0207
	wmMButtonUp    = 0x0208
	wmMouseWheel   = 0x020A
	wmKeyDown      = 0x0100
	wmKeyUp        = 0x0101
	wmChar         = 0x0102

	pmRemove      = 0x0001
	swShowDefault = 10
	idcArrow      = 32512
	idcIBeam      = 32513
	idcCross      = 32515
	idcHand       = 32649
	idcSizeNS     = 32645
	idcSizeEW     = 32644
	idcSizeNESW   = 32643
	idcSizeNWSE   = 32642
	idcSizeAll    = 32646
	idcNo         = 32648
	idcWait       = 32514
	idcAppStarting = 32650
	colorWindow   = 5
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procSetCursor        = user32.NewProc("SetCursor")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procTranslateMsg     = user32.NewProc("TranslateMessage")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procSetWindowLongW   = user32.NewProc("SetWindowLongW")
	procGetWindowLongW   = user32.NewProc("GetWindowLongW")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	windowClassName = syscall.StringToUTF16Ptr("LuxM1Window")
	wndProc         = syscall.NewCallback(windowProc)

	registerClassOnce sync.Once
	registerClassErr  error
	platformsByHWND   sync.Map
)

func init() {
	runtime.LockOSThread()
}

// Platform implements the Win32 desktop backend.
type Platform struct {
	hwnd           uintptr
	config         platform.Config
	callbacks      platform.Callbacks
	shouldClose    bool
	cursorKind     input.CursorKind
	cursors        map[input.CursorKind]uintptr
	fullscreen     bool
	savedStyle     uintptr
	savedRect      rect
	frameRequested bool
	windows        map[uint32]*windowState
	uiaBridge      *UIABridge
}

// windowState tracks per-window state for multi-window support.
type windowState struct {
	hwnd   uintptr
	width  int
	height int
}

// New creates a new Win32 platform instance.
func New() *Platform {
	return &Platform{}
}

// Init creates the native window.
func (p *Platform) Init(cfg platform.Config) error {
	if err := ensureWindowClass(); err != nil {
		return err
	}

	w, h := cfg.Width, cfg.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}

	title := cfg.Title
	if title == "" {
		title = "lux"
	}

	hInstance, _, err := procGetModuleHandleW.Call(0)
	if hInstance == 0 {
		return fmt.Errorf("get module handle: %w", err)
	}

	titlePtr := syscall.StringToUTF16Ptr(title)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(windowClassName)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsOverlappedWindow|wsVisible,
		cwUseDefault,
		cwUseDefault,
		uintptr(w),
		uintptr(h),
		0,
		0,
		hInstance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("create window: %w", err)
	}

	p.hwnd = hwnd
	p.config = cfg
	p.shouldClose = false
	platformsByHWND.Store(hwnd, p)

	procShowWindow.Call(hwnd, swShowDefault)
	procUpdateWindow.Call(hwnd)

	return nil
}

// Run enters the Win32 event loop.
func (p *Platform) Run(cb platform.Callbacks) error {
	p.callbacks = cb

	var msg msg
	for !p.shouldClose {
		for {
			hasMessage, _, _ := procPeekMessageW.Call(
				uintptr(unsafe.Pointer(&msg)),
				0,
				0,
				0,
				pmRemove,
			)
			if hasMessage == 0 {
				break
			}
			if msg.Message == wmQuit {
				p.shouldClose = true
				break
			}
			procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}

		if p.shouldClose {
			break
		}

		if p.callbacks.OnFrame != nil {
			p.callbacks.OnFrame()
		}

		time.Sleep(time.Second / 60)
	}

	return nil
}

// Destroy releases Win32 resources.
func (p *Platform) Destroy() {
	if p.hwnd == 0 {
		return
	}
	if p.uiaBridge != nil {
		p.uiaBridge.Destroy()
		p.uiaBridge = nil
	}
	platformsByHWND.Delete(p.hwnd)
	procDestroyWindow.Call(p.hwnd)
	p.hwnd = 0
}

// SetTitle updates the window title.
func (p *Platform) SetTitle(title string) {
	if p.hwnd == 0 {
		return
	}
	procSetWindowTextW.Call(p.hwnd, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
}

// WindowSize returns the client-area size.
func (p *Platform) WindowSize() (int, int) {
	if p.hwnd == 0 {
		return 0, 0
	}
	var rect rect
	ok, _, _ := procGetClientRect.Call(p.hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return 0, 0
	}
	return int(rect.Right - rect.Left), int(rect.Bottom - rect.Top)
}

// FramebufferSize returns the current framebuffer size.
func (p *Platform) FramebufferSize() (int, int) {
	return p.WindowSize()
}

// ShouldClose reports whether the window has been closed.
func (p *Platform) ShouldClose() bool {
	return p.shouldClose
}

// NativeHandle returns the Win32 HWND for the renderer.
func (p *Platform) NativeHandle() uintptr {
	return p.hwnd
}

// A11yBridge returns the UIA accessibility bridge, creating it on first call.
func (p *Platform) A11yBridge() a11y.A11yBridge {
	if p.uiaBridge == nil && p.hwnd != 0 {
		p.uiaBridge = NewUIABridge(p.hwnd, nil)
	}
	return p.uiaBridge
}

// SetA11ySend sets the send function for the UIA bridge to route actions
// to the app loop.
func (p *Platform) SetA11ySend(send func(any)) {
	if p.uiaBridge != nil {
		p.uiaBridge.send = send
	}
}

// SetCursor changes the system cursor shape (RFC-002 §2.7).
func (p *Platform) SetCursor(kind input.CursorKind) {
	if kind == p.cursorKind {
		return
	}
	p.cursorKind = kind
	if p.cursors == nil {
		p.initCursors()
	}
	h, ok := p.cursors[kind]
	if !ok {
		h = p.cursors[input.CursorDefault]
	}
	procSetCursor.Call(h)
}

// SetIMECursorRect positions the IME candidate window near the text cursor (RFC-002 §2.2).
// TODO: Implement via ImmSetCompositionWindow when IMM32 integration is added.
func (p *Platform) SetIMECursorRect(x, y, w, h int) {
	// Win32 IMM32 integration not yet implemented.
}

// SetSize resizes the window to the given dimensions in screen coordinates (RFC §7.1).
func (p *Platform) SetSize(w, h int) {
	if p.hwnd == 0 {
		return
	}
	const swpNoMove = 0x0002
	const swpNoZOrder = 0x0004
	procSetWindowPos.Call(p.hwnd, 0, 0, 0, uintptr(w), uintptr(h), swpNoMove|swpNoZOrder)
}

// SetFullscreen toggles fullscreen mode (RFC §7.1).
func (p *Platform) SetFullscreen(fullscreen bool) {
	if p.hwnd == 0 || fullscreen == p.fullscreen {
		return
	}
	p.fullscreen = fullscreen

	const gwlStyle = 0xFFFFFFF0 // GWL_STYLE (-16 as uint32)
	const swpFrameChanged = 0x0020
	const swpNoZOrder = 0x0004
	const smCXScreen = 0
	const smCYScreen = 1

	if fullscreen {
		// Save current style and window rect.
		style, _, _ := procGetWindowLongW.Call(p.hwnd, uintptr(gwlStyle))
		p.savedStyle = style
		procGetWindowRect.Call(p.hwnd, uintptr(unsafe.Pointer(&p.savedRect)))

		// Remove borders (WS_OVERLAPPEDWINDOW bits) and make popup.
		const wsPopup = 0x80000000
		procSetWindowLongW.Call(p.hwnd, uintptr(gwlStyle), wsPopup|wsVisible)

		// Resize to full screen.
		screenW, _, _ := procGetSystemMetrics.Call(smCXScreen)
		screenH, _, _ := procGetSystemMetrics.Call(smCYScreen)
		procSetWindowPos.Call(p.hwnd, 0, 0, 0, screenW, screenH, swpFrameChanged|swpNoZOrder)
	} else {
		// Restore saved style and rect.
		procSetWindowLongW.Call(p.hwnd, uintptr(gwlStyle), p.savedStyle)
		w := int(p.savedRect.Right - p.savedRect.Left)
		h := int(p.savedRect.Bottom - p.savedRect.Top)
		procSetWindowPos.Call(p.hwnd, 0,
			uintptr(p.savedRect.Left), uintptr(p.savedRect.Top),
			uintptr(w), uintptr(h),
			swpFrameChanged|swpNoZOrder)
	}
}

// RequestFrame requests a new frame to be rendered (RFC §7.1).
func (p *Platform) RequestFrame() {
	if p.hwnd == 0 {
		return
	}
	p.frameRequested = true
	procInvalidateRect.Call(p.hwnd, 0, 0)
}

// SetClipboard sets the system clipboard text (RFC §7.1).
func (p *Platform) SetClipboard(text string) error {
	const cfUnicodeText = 13
	const gmemMoveable = 0x0002

	utf16 := syscall.StringToUTF16(text)
	size := len(utf16) * 2

	ok, _, err := procOpenClipboard.Call(p.hwnd)
	if ok == 0 {
		return fmt.Errorf("open clipboard: %w", err)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	hMem, _, err := procGlobalAlloc.Call(gmemMoveable, uintptr(size))
	if hMem == 0 {
		return fmt.Errorf("global alloc: %w", err)
	}

	pMem, _, _ := procGlobalLock.Call(hMem)
	if pMem == 0 {
		return fmt.Errorf("global lock failed")
	}

	// Copy UTF-16 data.
	src := unsafe.Pointer(&utf16[0])
	dst := unsafe.Pointer(pMem)
	for i := 0; i < size; i++ {
		*(*byte)(unsafe.Add(dst, i)) = *(*byte)(unsafe.Add(src, i))
	}

	procGlobalUnlock.Call(hMem)
	procSetClipboardData.Call(cfUnicodeText, hMem)
	return nil
}

// GetClipboard returns the current system clipboard text (RFC §7.1).
func (p *Platform) GetClipboard() (string, error) {
	const cfUnicodeText = 13

	ok, _, err := procOpenClipboard.Call(p.hwnd)
	if ok == 0 {
		return "", fmt.Errorf("open clipboard: %w", err)
	}
	defer procCloseClipboard.Call()

	hData, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if hData == 0 {
		return "", nil
	}

	pData, _, _ := procGlobalLock.Call(hData)
	if pData == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(hData)

	// Convert UTF-16 pointer to Go string (syscall.UTF16PtrToString is not available).
	ptr := (*uint16)(unsafe.Pointer(pData))
	var utf16Chars []uint16
	for i := 0; ; i++ {
		ch := *(*uint16)(unsafe.Add(unsafe.Pointer(ptr), uintptr(i)*2))
		if ch == 0 {
			break
		}
		utf16Chars = append(utf16Chars, ch)
	}
	return string(syscall.UTF16ToString(utf16Chars)), nil
}

// CreateWGPUSurface creates a wgpu surface for this window (RFC §7.1).
// Returns the HWND as a handle that wgpu-native can use to create a surface.
func (p *Platform) CreateWGPUSurface(instance uintptr) uintptr {
	// The HWND is passed directly — wgpu-native uses it to create a
	// WGPUSurfaceDescriptorFromWindowsHWND.
	return p.hwnd
}

func (p *Platform) initCursors() {
	load := func(id uintptr) uintptr {
		h, _, _ := procLoadCursorW.Call(0, id)
		return h
	}
	p.cursors = map[input.CursorKind]uintptr{
		input.CursorDefault:    load(idcArrow),
		input.CursorText:       load(idcIBeam),
		input.CursorPointer:    load(idcHand),
		input.CursorCrosshair:  load(idcCross),
		input.CursorMove:       load(idcSizeAll),
		input.CursorResizeNS:   load(idcSizeNS),
		input.CursorResizeEW:   load(idcSizeEW),
		input.CursorResizeNESW: load(idcSizeNESW),
		input.CursorResizeNWSE: load(idcSizeNWSE),
		input.CursorNotAllowed: load(idcNo),
		input.CursorWait:       load(idcWait),
		input.CursorProgress:   load(idcAppStarting),
		input.CursorGrab:       load(idcHand),
		input.CursorGrabbing:   load(idcHand),
	}
}

var _ platform.MultiWindowPlatform = (*Platform)(nil)

// CreateWindow creates a secondary window and returns its native handle.
func (p *Platform) CreateWindow(id uint32, cfg platform.Config) (uintptr, error) {
	if err := ensureWindowClass(); err != nil {
		return 0, err
	}
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

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(windowClassName)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		wsOverlappedWindow|wsVisible,
		cwUseDefault, cwUseDefault,
		uintptr(w), uintptr(h),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW for window %d: %w", id, err)
	}

	if p.windows == nil {
		p.windows = make(map[uint32]*windowState)
	}
	p.windows[id] = &windowState{hwnd: hwnd, width: w, height: h}
	platformsByHWND.Store(hwnd, p)

	procShowWindow.Call(hwnd, swShowDefault)
	procUpdateWindow.Call(hwnd)

	return hwnd, nil
}

// DestroyWindow destroys a secondary window.
func (p *Platform) DestroyWindow(id uint32) {
	if p.windows == nil {
		return
	}
	ws, ok := p.windows[id]
	if !ok {
		return
	}
	procDestroyWindow.Call(ws.hwnd)
	platformsByHWND.Delete(ws.hwnd)
	delete(p.windows, id)
}

// SetWindowTitle updates the title of a specific window.
func (p *Platform) SetWindowTitle(id uint32, title string) {
	if p.windows == nil {
		return
	}
	ws, ok := p.windows[id]
	if !ok {
		return
	}
	procSetWindowTextW.Call(ws.hwnd, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
}

// WindowSizeByID returns the size of a specific window.
func (p *Platform) WindowSizeByID(id uint32) (int, int) {
	if p.windows == nil {
		return 0, 0
	}
	ws, ok := p.windows[id]
	if !ok {
		return 0, 0
	}
	return ws.width, ws.height
}

// FramebufferSizeByID returns the framebuffer size of a specific window.
func (p *Platform) FramebufferSizeByID(id uint32) (int, int) {
	return p.WindowSizeByID(id) // 1:1 on Windows (DPI scaling handled separately)
}

func ensureWindowClass() error {
	registerClassOnce.Do(func() {
		hInstance, _, err := procGetModuleHandleW.Call(0)
		if hInstance == 0 {
			registerClassErr = fmt.Errorf("get module handle: %w", err)
			return
		}

		cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
		wc := wndClassEx{
			Size:       uint32(unsafe.Sizeof(wndClassEx{})),
			Style:      0, // no CS_HREDRAW/CS_VREDRAW — app redraws every frame
			WndProc:    wndProc,
			Instance:   hInstance,
			Cursor:     cursor,
			Background: 0, // NULL brush — prevent GDI background erase flash
			ClassName:  windowClassName,
		}
		atom, _, callErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			registerClassErr = fmt.Errorf("register class: %w", callErr)
		}
	})

	return registerClassErr
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if value, ok := platformsByHWND.Load(hwnd); ok {
		p := value.(*Platform)

		// Determine if this is a secondary window.
		isSecondary := hwnd != p.hwnd
		if isSecondary {
			return p.secondaryWindowProc(hwnd, msg, wParam, lParam)
		}

		switch msg {
		case wmGetObject:
			// UIA: respond to automation requests (RFC-001 §11).
			if uint32(lParam) == uiaRootObjectId {
				if p.uiaBridge != nil {
					return uiaReturnRawElementProvider(hwnd, wParam, lParam, p.uiaBridge.RootProvider())
				}
			}

		case wmEraseBkgnd:
			// Suppress GDI background erase to prevent flash during resize/fullscreen toggle.
			return 1
		case wmEnterSizeMove:
			// Windows runs a modal loop during resize/move that blocks our main loop.
			// Start a 16ms timer so OnFrame keeps ticking inside the modal loop.
			const resizeTimerID = 1
			procSetTimer.Call(hwnd, resizeTimerID, 16, 0)
			return 0
		case wmExitSizeMove:
			const resizeTimerID = 1
			procKillTimer.Call(hwnd, resizeTimerID)
			return 0
		case wmTimer:
			// Fire a frame from inside the modal resize/move loop.
			if p.callbacks.OnFrame != nil {
				p.callbacks.OnFrame()
			}
			return 0
		case wmSize:
			if p.callbacks.OnResize != nil {
				width := int(uint32(lParam & 0xFFFF))
				height := int(uint32((lParam >> 16) & 0xFFFF))
				p.callbacks.OnResize(width, height)
			}
			return 0
		case wmMouseMove:
			if p.callbacks.OnMouseMove != nil {
				x := float32(int16(lParam & 0xFFFF))
				y := float32(int16((lParam >> 16) & 0xFFFF))
				p.callbacks.OnMouseMove(x, y)
			}
			return 0
		case wmLButtonDown, wmRButtonDown, wmMButtonDown:
			if p.callbacks.OnMouseButton != nil {
				x := float32(int16(lParam & 0xFFFF))
				y := float32(int16((lParam >> 16) & 0xFFFF))
				button := 0
				if msg == wmRButtonDown {
					button = 1
				} else if msg == wmMButtonDown {
					button = 2
				}
				p.callbacks.OnMouseButton(x, y, button, true)
			}
			return 0
		case wmLButtonUp, wmRButtonUp, wmMButtonUp:
			if p.callbacks.OnMouseButton != nil {
				x := float32(int16(lParam & 0xFFFF))
				y := float32(int16((lParam >> 16) & 0xFFFF))
				button := 0
				if msg == wmRButtonUp {
					button = 1
				} else if msg == wmMButtonUp {
					button = 2
				}
				p.callbacks.OnMouseButton(x, y, button, false)
			}
			return 0
		case wmMouseWheel:
			if p.callbacks.OnScroll != nil {
				delta := float32(int16(wParam>>16)) / 120.0
				p.callbacks.OnScroll(0, delta)
			}
			return 0
		case wmKeyDown:
			vk := int(wParam)
			// DEBUG: change window title to confirm wmKeyDown fires for Enter
			if vk == 0x0D {
				title, _ := syscall.UTF16PtrFromString(fmt.Sprintf("ENTER PRESSED vk=0x%X", vk))
				procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(title)))
			}
			if p.callbacks.OnKey != nil {
				name := win32KeyName(vk)
				p.callbacks.OnKey(name, 0, 0) // press
			}
			// Enter: also fire as OnChar('\n') so multiline TextArea works
			// even if TranslateMessage/wmChar doesn't deliver it.
			if vk == 0x0D && p.callbacks.OnChar != nil {
				p.callbacks.OnChar('\n')
			}
			return 0
		case wmKeyUp:
			if p.callbacks.OnKey != nil {
				name := win32KeyName(int(wParam))
				p.callbacks.OnKey(name, 1, 0) // release
			}
			return 0
		case wmChar:
			if p.callbacks.OnChar != nil {
				ch := rune(wParam)
				// Skip \r from Enter — already handled in wmKeyDown as \n.
				if ch == '\r' {
					return 0
				}
				p.callbacks.OnChar(ch)
			}
			return 0
		case wmClose:
			p.shouldClose = true
			if p.callbacks.OnClose != nil {
				p.callbacks.OnClose()
			}
			procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			p.shouldClose = true
			p.hwnd = 0
			platformsByHWND.Delete(hwnd)
			procPostQuitMessage.Call(0)
			return 0
		}
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return result
}

// windowIDByHWND finds the window ID for a secondary window HWND.
func (p *Platform) windowIDByHWND(hwnd uintptr) (uint32, bool) {
	for id, ws := range p.windows {
		if ws.hwnd == hwnd {
			return id, true
		}
	}
	return 0, false
}

// secondaryWindowProc handles messages for secondary (non-main) windows.
func (p *Platform) secondaryWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	wid, _ := p.windowIDByHWND(hwnd)
	switch msg {
	case wmEraseBkgnd:
		return 1
	case wmSize:
		width := int(uint32(lParam & 0xFFFF))
		height := int(uint32((lParam >> 16) & 0xFFFF))
		// Update stored size.
		if ws, ok := p.windows[wid]; ok {
			ws.width = width
			ws.height = height
		}
		if p.callbacks.OnWindowResize != nil {
			p.callbacks.OnWindowResize(wid, width, height)
		}
		return 0
	case wmMouseMove:
		if p.callbacks.OnWindowMouseMove != nil {
			x := float32(int16(lParam & 0xFFFF))
			y := float32(int16((lParam >> 16) & 0xFFFF))
			p.callbacks.OnWindowMouseMove(wid, x, y)
		}
		return 0
	case wmLButtonDown, wmRButtonDown, wmMButtonDown:
		if p.callbacks.OnWindowMouseButton != nil {
			x := float32(int16(lParam & 0xFFFF))
			y := float32(int16((lParam >> 16) & 0xFFFF))
			button := 0
			if msg == wmRButtonDown {
				button = 1
			} else if msg == wmMButtonDown {
				button = 2
			}
			p.callbacks.OnWindowMouseButton(wid, x, y, button, true)
		}
		return 0
	case wmLButtonUp, wmRButtonUp, wmMButtonUp:
		if p.callbacks.OnWindowMouseButton != nil {
			x := float32(int16(lParam & 0xFFFF))
			y := float32(int16((lParam >> 16) & 0xFFFF))
			button := 0
			if msg == wmRButtonUp {
				button = 1
			} else if msg == wmMButtonUp {
				button = 2
			}
			p.callbacks.OnWindowMouseButton(wid, x, y, button, false)
		}
		return 0
	case wmKeyDown:
		if p.callbacks.OnWindowKey != nil {
			p.callbacks.OnWindowKey(wid, win32KeyName(int(wParam)), 0, 0)
		}
		return 0
	case wmKeyUp:
		if p.callbacks.OnWindowKey != nil {
			p.callbacks.OnWindowKey(wid, win32KeyName(int(wParam)), 1, 0)
		}
		return 0
	case wmChar:
		if p.callbacks.OnWindowChar != nil {
			p.callbacks.OnWindowChar(wid, rune(wParam))
		}
		return 0
	case wmMouseWheel:
		if p.callbacks.OnWindowScroll != nil {
			delta := float32(int16(wParam>>16)) / 120.0
			p.callbacks.OnWindowScroll(wid, 0, delta)
		}
		return 0
	case wmClose:
		// Do NOT call PostQuitMessage — only destroy the secondary window.
		if p.callbacks.OnWindowClose != nil {
			p.callbacks.OnWindowClose(wid)
		}
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		// Clean up without quitting the application.
		platformsByHWND.Delete(hwnd)
		delete(p.windows, wid)
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return result
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// win32KeyName maps a Win32 virtual key code to a human-readable name.
func win32KeyName(vk int) string {
	switch vk {
	case 0x08:
		return "Backspace"
	case 0x09:
		return "Tab"
	case 0x0D:
		return "Enter"
	case 0x1B:
		return "Escape"
	case 0x20:
		return "Space"
	case 0x25:
		return "Left"
	case 0x26:
		return "Up"
	case 0x27:
		return "Right"
	case 0x28:
		return "Down"
	case 0x23:
		return "End"
	case 0x24:
		return "Home"
	case 0x2E:
		return "Delete"
	default:
		if vk >= 0x41 && vk <= 0x5A {
			return string(rune(vk))
		}
		if vk >= 0x30 && vk <= 0x39 {
			return string(rune(vk))
		}
		return fmt.Sprintf("Key(%d)", vk)
	}
}
