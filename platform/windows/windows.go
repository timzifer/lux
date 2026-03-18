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
	wmClose        = 0x0010
	wmQuit         = 0x0012
	wmLButtonDown  = 0x0201
	wmRButtonDown  = 0x0204
	wmMButtonDown  = 0x0207

	pmRemove      = 0x0001
	swShowDefault = 10
	idcArrow      = 32512
	colorWindow   = 5
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW = user32.NewProc("CreateWindowExW")
	procDefWindowProcW  = user32.NewProc("DefWindowProcW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procDispatchMessage = user32.NewProc("DispatchMessageW")
	procGetClientRect   = user32.NewProc("GetClientRect")
	procLoadCursorW     = user32.NewProc("LoadCursorW")
	procPeekMessageW    = user32.NewProc("PeekMessageW")
	procPostQuitMessage = user32.NewProc("PostQuitMessage")
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procSetWindowTextW  = user32.NewProc("SetWindowTextW")
	procShowWindow      = user32.NewProc("ShowWindow")
	procTranslateMsg    = user32.NewProc("TranslateMessage")
	procUpdateWindow    = user32.NewProc("UpdateWindow")

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
	hwnd        uintptr
	config      platform.Config
	callbacks   platform.Callbacks
	shouldClose bool
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
			Style:      csHRedraw | csVRedraw,
			WndProc:    wndProc,
			Instance:   hInstance,
			Cursor:     cursor,
			Background: colorWindow + 1,
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
		switch msg {
		case wmSize:
			if p.callbacks.OnResize != nil {
				width := int(uint32(lParam & 0xFFFF))
				height := int(uint32((lParam >> 16) & 0xFFFF))
				p.callbacks.OnResize(width, height)
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
