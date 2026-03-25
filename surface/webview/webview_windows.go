//go:build windows && !servo

package webview

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
	"github.com/zzl/go-com/com"
	wv2 "github.com/zzl/go-webview2/wv2"
	"github.com/zzl/go-win32api/v2/win32"
)

var (
	webView2Loader = syscall.NewLazyDLL("WebView2Loader.dll")

	// WebView2 + DXGI bootstrap entrypoint for the COM environment.
	procCreateCoreWebView2EnvironmentWithOptions = webView2Loader.NewProc("CreateCoreWebView2EnvironmentWithOptions")
)

func init() {
	newPlatformBackend = func(w *WebView) platformBackend {
		return newWindowsBackend(w)
	}
}

// windowsBackend models the RFC-004 §7 integration path:
// ICoreWebView2CompositionController -> DXGI Shared Handle -> Lux TextureID.
//
// The backend now performs real WebView2 COM bootstrap calls and wires core
// navigation/title/history events. The DXGI Shared Handle export path remains
// the next step once Lux has a concrete DComp/WGPU import bridge.
type windowsBackend struct {
	w *WebView

	mu sync.Mutex

	runtimeAvailable bool
	closed           bool
	comInitialized   bool

	comInit  com.Initialized
	comScope *com.Scope

	hwnd win32.HWND

	environment *wv2.ICoreWebView2Environment
	env3        *wv2.ICoreWebView2Environment3
	controller  *wv2.ICoreWebView2Controller

	// COM anchor: ICoreWebView2CompositionController.
	composition *wv2.ICoreWebView2CompositionController
	core        *wv2.ICoreWebView2

	// Zero-copy anchor: DXGI Shared Handle imported by Lux/WGPU later.
	dxgiSharedHandle uintptr
	textureID        draw.TextureID

	lastErr error

	pendingURL     string
	pendingScripts []pendingScript

	envCompleted         *wv2.ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler
	controllerCompleted  *wv2.ICoreWebView2CreateCoreWebView2CompositionControllerCompletedHandler
	navigationStarting   *wv2.ICoreWebView2NavigationStartingEventHandler
	navigationCompleted  *wv2.ICoreWebView2NavigationCompletedEventHandler
	historyChanged       *wv2.ICoreWebView2HistoryChangedEventHandler
	documentTitleChanged *wv2.ICoreWebView2DocumentTitleChangedEventHandler

	navigationStartingToken   wv2.EventRegistrationToken
	navigationCompleteToken   wv2.EventRegistrationToken
	historyChangedToken       wv2.EventRegistrationToken
	documentTitleChangedToken wv2.EventRegistrationToken
}

type pendingScript struct {
	js   string
	done chan error
}

func newWindowsBackend(w *WebView) *windowsBackend {
	b := &windowsBackend{
		w:    w,
		hwnd: win32.HWND(w.cfg.parentWindow),
	}
	b.bootstrap()
	return b
}

func (b *windowsBackend) bootstrap() {
	if err := procCreateCoreWebView2EnvironmentWithOptions.Find(); err != nil {
		b.setError(err)
		return
	}
	if b.hwnd == 0 {
		b.setError(errors.New("webview2 composition controller requires a parent HWND; use WithParentWindow"))
		return
	}

	b.comInit = com.Initialize()
	b.comScope = com.NewScope()
	b.comInitialized = true
	b.runtimeAvailable = true

	b.envCompleted = wv2.NewICoreWebView2CreateCoreWebView2EnvironmentCompletedHandlerByFunc(
		func(errorCode com.Error, createdEnvironment *wv2.ICoreWebView2Environment) com.Error {
			if errorCode < 0 || createdEnvironment == nil {
				b.setError(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions callback failed: 0x%08x", uint32(errorCode)))
				return errorCode
			}
			createdEnvironment.AddRef()

			b.mu.Lock()
			b.environment = createdEnvironment
			b.env3 = (*wv2.ICoreWebView2Environment3)(unsafe.Pointer(createdEnvironment))
			b.mu.Unlock()

			b.controllerCompleted = wv2.NewICoreWebView2CreateCoreWebView2CompositionControllerCompletedHandlerByFunc(
				func(errorCode com.Error, webView *wv2.ICoreWebView2CompositionController) com.Error {
					if errorCode < 0 || webView == nil {
						b.setError(fmt.Errorf("CreateCoreWebView2CompositionController callback failed: 0x%08x", uint32(errorCode)))
						return errorCode
					}
					webView.AddRef()

					controller, core, err := b.bindCompositionController(webView)
					if err != nil {
						b.setError(err)
						return com.Error(win32.E_FAIL)
					}

					b.mu.Lock()
					b.composition = webView
					b.controller = controller
					b.core = core
					b.mu.Unlock()

					b.applyBounds(b.w.lastBounds)
					b.installEventHandlers(core)
					b.flushPending()
					b.syncCoreState()
					return 0
				}, true,
			)

			if err := b.env3.CreateCoreWebView2CompositionController(b.hwnd, b.controllerCompleted); err < 0 {
				b.setError(fmt.Errorf("CreateCoreWebView2CompositionController failed: 0x%08x", uint32(err)))
				return err
			}
			return 0
		}, true,
	)

	userData := uintptr(0)
	if b.w.cfg.userDataDir != "" {
		userData = uintptr(win32.StrToPointer(b.w.cfg.userDataDir))
	}

	r1, _, callErr := procCreateCoreWebView2EnvironmentWithOptions.Call(
		0,
		userData,
		0,
		uintptr(unsafe.Pointer(b.envCompleted)),
	)
	hr = win32.HRESULT(r1)
	if callErr != syscall.Errno(0) {
		b.setError(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions call failed: %w", callErr))
		return
	}
	if win32.FAILED(hr) {
		b.setError(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions failed: 0x%08x", uint32(hr)))
	}
}

func (b *windowsBackend) bindCompositionController(comp *wv2.ICoreWebView2CompositionController) (*wv2.ICoreWebView2Controller, *wv2.ICoreWebView2, error) {
	var controller *wv2.ICoreWebView2Controller
	if hr := ((*win32.IUnknown)(unsafe.Pointer(comp))).QueryInterface(&wv2.IID_ICoreWebView2Controller, unsafe.Pointer(&controller)); win32.FAILED(hr) {
		return nil, nil, fmt.Errorf("QueryInterface(ICoreWebView2Controller) failed: 0x%08x", uint32(hr))
	}
	if controller == nil {
		return nil, nil, errors.New("QueryInterface(ICoreWebView2Controller) returned nil controller")
	}

	var core *wv2.ICoreWebView2
	if err := controller.GetCoreWebView2(&core); err < 0 {
		return nil, nil, fmt.Errorf("GetCoreWebView2 failed: 0x%08x", uint32(err))
	}
	if core == nil {
		return nil, nil, errors.New("GetCoreWebView2 returned nil core")
	}
	core.AddRef()
	return controller, core, nil
}

func (b *windowsBackend) installEventHandlers(core *wv2.ICoreWebView2) {
	b.navigationStarting = wv2.NewICoreWebView2NavigationStartingEventHandlerByFunc(
		func(sender *wv2.ICoreWebView2, args *wv2.ICoreWebView2NavigationStartingEventArgs) com.Error {
			if args != nil {
				var uri win32.PWSTR
				if err := args.GetUri(&uri); err >= 0 && uri != nil {
					b.w.setCurrentURL(win32.PwstrToStr(uri))
					win32.CoTaskMemFree(unsafe.Pointer(uri))
				}
			}
			b.w.setLoading(true)
			return 0
		}, true,
	)
	_ = core.Add_NavigationStarting(b.navigationStarting, &b.navigationStartingToken)

	b.navigationCompleted = wv2.NewICoreWebView2NavigationCompletedEventHandlerByFunc(
		func(sender *wv2.ICoreWebView2, args *wv2.ICoreWebView2NavigationCompletedEventArgs) com.Error {
			b.w.setLoading(false)
			b.syncCoreState()
			return 0
		}, true,
	)
	_ = core.Add_NavigationCompleted(b.navigationCompleted, &b.navigationCompleteToken)

	b.historyChanged = wv2.NewICoreWebView2HistoryChangedEventHandlerByFunc(
		func(sender *wv2.ICoreWebView2, _ *win32.IUnknown) com.Error {
			b.syncCoreState()
			return 0
		}, true,
	)
	_ = core.Add_HistoryChanged(b.historyChanged, &b.historyChangedToken)

	b.documentTitleChanged = wv2.NewICoreWebView2DocumentTitleChangedEventHandlerByFunc(
		func(sender *wv2.ICoreWebView2, _ *win32.IUnknown) com.Error {
			var title win32.PWSTR
			if err := sender.GetDocumentTitle(&title); err >= 0 && title != nil {
				b.w.setTitle(win32.PwstrToStr(title))
				win32.CoTaskMemFree(unsafe.Pointer(title))
			}
			return 0
		}, true,
	)
	_ = core.Add_DocumentTitleChanged(b.documentTitleChanged, &b.documentTitleChangedToken)
}

func (b *windowsBackend) syncCoreState() {
	b.mu.Lock()
	core := b.core
	b.mu.Unlock()
	if core == nil {
		return
	}

	var canBack, canForward int32
	if err := core.GetCanGoBack(&canBack); err >= 0 || err == 0 {
		if err2 := core.GetCanGoForward(&canForward); err2 >= 0 || err2 == 0 {
			b.w.setHistoryAvailability(canBack != 0, canForward != 0)
		}
	}

	var title win32.PWSTR
	if err := core.GetDocumentTitle(&title); err >= 0 && title != nil {
		b.w.setTitle(win32.PwstrToStr(title))
		win32.CoTaskMemFree(unsafe.Pointer(title))
	}
}

func (b *windowsBackend) flushPending() {
	b.mu.Lock()
	pendingURL := b.pendingURL
	pendingScripts := append([]pendingScript(nil), b.pendingScripts...)
	b.pendingURL = ""
	b.pendingScripts = nil
	core := b.core
	b.mu.Unlock()

	if core == nil {
		return
	}
	if pendingURL != "" {
		if err := b.navigateCore(core, pendingURL); err != nil {
			b.setError(err)
		}
	}
	for _, script := range pendingScripts {
		script.done <- b.executeScriptCore(core, script.js)
		close(script.done)
	}
}

func (b *windowsBackend) Navigate(url string) {
	b.mu.Lock()
	core := b.core
	if core == nil {
		b.pendingURL = url
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	if err := b.navigateCore(core, url); err != nil {
		b.setError(err)
		b.w.setLoading(false)
	}
}

func (b *windowsBackend) navigateCore(core *wv2.ICoreWebView2, url string) error {
	if err := core.Navigate(url); err < 0 {
		return fmt.Errorf("ICoreWebView2.Navigate failed: 0x%08x", uint32(err))
	}
	return nil
}

func (b *windowsBackend) Eval(js string) error {
	b.mu.Lock()
	core := b.core
	lastErr := b.lastErr
	b.mu.Unlock()
	if core == nil {
		if lastErr != nil {
			return lastErr
		}
		return errors.New("webview2 core not ready")
	}
	return b.executeScriptCore(core, js)
}

func (b *windowsBackend) executeScriptCore(core *wv2.ICoreWebView2, js string) error {
	done := make(chan error, 1)
	h := wv2.NewICoreWebView2ExecuteScriptCompletedHandlerByFunc(
		func(errorCode com.Error, _ string) com.Error {
			if errorCode < 0 {
				done <- fmt.Errorf("ICoreWebView2.ExecuteScript callback failed: 0x%08x", uint32(errorCode))
			} else {
				done <- nil
			}
			return 0
		}, true,
	)
	if err := core.ExecuteScript(js, h); err < 0 {
		return fmt.Errorf("ICoreWebView2.ExecuteScript failed: 0x%08x", uint32(err))
	}
	return <-done
}

func (b *windowsBackend) Reload() {
	b.mu.Lock()
	core := b.core
	b.mu.Unlock()
	if core == nil {
		b.w.setLoading(false)
		return
	}
	if err := core.Reload(); err < 0 {
		b.setError(fmt.Errorf("ICoreWebView2.Reload failed: 0x%08x", uint32(err)))
	}
	b.w.setLoading(false)
}

func (b *windowsBackend) Back() {
	b.mu.Lock()
	core := b.core
	b.mu.Unlock()
	if core == nil {
		b.w.setLoading(false)
		return
	}
	if err := core.GoBack(); err < 0 {
		b.setError(fmt.Errorf("ICoreWebView2.GoBack failed: 0x%08x", uint32(err)))
	}
	b.w.setLoading(false)
}

func (b *windowsBackend) Forward() {
	b.mu.Lock()
	core := b.core
	b.mu.Unlock()
	if core == nil {
		b.w.setLoading(false)
		return
	}
	if err := core.GoForward(); err < 0 {
		b.setError(fmt.Errorf("ICoreWebView2.GoForward failed: 0x%08x", uint32(err)))
	}
	b.w.setLoading(false)
}

func (b *windowsBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true

	if b.core != nil {
		b.core.Release()
		b.core = nil
	}
	if b.composition != nil {
		b.composition.Release()
		b.composition = nil
	}
	if b.controller != nil {
		_ = b.controller.Close()
		b.controller.Release()
		b.controller = nil
	}
	if b.environment != nil {
		b.environment.Release()
		b.environment = nil
		b.env3 = nil
	}
	b.dxgiSharedHandle = 0
	b.textureID = 0
	if b.comInitialized {
		if b.comScope != nil {
			b.comScope.Leave()
			b.comScope = nil
		}
		b.comInit.Uninitialize()
		b.comInitialized = false
	}
	return b.lastErr
}

func (b *windowsBackend) AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	b.applyBounds(bounds)
	b.w.mu.Lock()
	defer b.w.mu.Unlock()
	return b.textureID, b.w.currentTextureTokenLocked()
}

func (b *windowsBackend) applyBounds(bounds draw.Rect) {
	b.mu.Lock()
	controller := b.controller
	b.mu.Unlock()
	if controller == nil || bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	_ = controller.SetBounds(wv2.TagRECT{
		Left:   int32(bounds.X),
		Top:    int32(bounds.Y),
		Right:  int32(bounds.X + bounds.W),
		Bottom: int32(bounds.Y + bounds.H),
	})
}

func (b *windowsBackend) ReleaseFrame(ui.FrameToken) {}

func (b *windowsBackend) HandleMsg(msg any) bool {
	b.mu.Lock()
	composition := b.composition
	b.mu.Unlock()
	if composition == nil {
		return false
	}

	switch msg := msg.(type) {
	case ui.SurfaceMouseMsg:
		kind, mouseData, ok := mouseEventKind(msg)
		if !ok {
			return false
		}
		virtualKeys := mouseVirtualKeys(msg)
		point := wv2.TagPOINT{X: int32(msg.Pos.X), Y: int32(msg.Pos.Y)}
		return composition.SendMouseInput(kind, virtualKeys, mouseData, point) >= 0
	case ui.SurfaceKeyMsg:
		// Keyboard input for the composition path is still delivered via the host
		// HWND message pump. The parent window supplied via WithParentWindow is the
		// expected route for WM_KEY* forwarding.
		_ = msg
		return false
	default:
		return false
	}
}

func mouseEventKind(msg ui.SurfaceMouseMsg) (kind int32, mouseData uint32, ok bool) {
	switch msg.Action {
	case input.MouseMove:
		return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_MOVE, 0, true
	case input.MouseEnter:
		return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_MOVE, 0, true
	case input.MouseLeave:
		return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_LEAVE, 0, true
	case input.MouseScroll:
		return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_WHEEL, uint32(120), true
	case input.MousePress:
		switch msg.Button {
		case input.MouseButtonLeft:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_LEFT_BUTTON_DOWN, 0, true
		case input.MouseButtonRight:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_RIGHT_BUTTON_DOWN, 0, true
		case input.MouseButtonMiddle:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_MIDDLE_BUTTON_DOWN, 0, true
		}
	case input.MouseRelease:
		switch msg.Button {
		case input.MouseButtonLeft:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_LEFT_BUTTON_UP, 0, true
		case input.MouseButtonRight:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_RIGHT_BUTTON_UP, 0, true
		case input.MouseButtonMiddle:
			return wv2.COREWEBVIEW2_MOUSE_EVENT_KIND.COREWEBVIEW2_MOUSE_EVENT_KIND_MIDDLE_BUTTON_UP, 0, true
		}
	}
	return 0, 0, false
}

func mouseVirtualKeys(msg ui.SurfaceMouseMsg) int32 {
	var keys int32
	switch msg.Button {
	case input.MouseButtonLeft:
		keys |= wv2.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS_LEFT_BUTTON
	case input.MouseButtonRight:
		keys |= wv2.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS_RIGHT_BUTTON
	case input.MouseButtonMiddle:
		keys |= wv2.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS.COREWEBVIEW2_MOUSE_EVENT_VIRTUAL_KEYS_MIDDLE_BUTTON
	}
	return keys
}

func (b *windowsBackend) setError(err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastErr = err
	b.runtimeAvailable = false
}
