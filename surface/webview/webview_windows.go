//go:build windows && !servo

package webview

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/wgpu"
	"github.com/timzifer/lux/ui"
	"github.com/zzl/go-com/com"
	wv2 "github.com/zzl/go-webview2/wv2"
	"github.com/zzl/go-win32api/v2/win32"
)

var (
	webView2Loader = syscall.NewLazyDLL("WebView2Loader.dll")

	// WebView2 + DXGI bootstrap entrypoint for the COM environment.
	procCreateCoreWebView2EnvironmentWithOptions = webView2Loader.NewProc("CreateCoreWebView2EnvironmentWithOptions")

	wvUser32             = syscall.NewLazyDLL("user32.dll")
	procWVCreateWindowEx = wvUser32.NewProc("CreateWindowExW")
	procWVDestroyWindow  = wvUser32.NewProc("DestroyWindow")
	procWVMoveWindow     = wvUser32.NewProc("MoveWindow")
	procWVShowWindow     = wvUser32.NewProc("ShowWindow")
	procWVClientToScreen = wvUser32.NewProc("ClientToScreen")
	procPrintWindow      = wvUser32.NewProc("PrintWindow")
	procPostMessageW     = wvUser32.NewProc("PostMessageW")
	procSetFocus         = wvUser32.NewProc("SetFocus")
	procGetWindow        = wvUser32.NewProc("GetWindow")
	procScreenToClient   = wvUser32.NewProc("ScreenToClient")

	wvGdi32              = syscall.NewLazyDLL("gdi32.dll")
	procCreateCompatDC   = wvGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC         = wvGdi32.NewProc("DeleteDC")
	procSelectObject     = wvGdi32.NewProc("SelectObject")
	procDeleteObject     = wvGdi32.NewProc("DeleteObject")
	procCreateDIBSection = wvGdi32.NewProc("CreateDIBSection")

)

const (
	wsPopup        = 0x80000000
	wsClipChildren = 0x02000000
	wsExToolWindow = 0x00000080
	swShow         = 5
	swShowNA       = 8 // ShowWindow without activating

	pwRenderFullContent = 0x00000002 // PrintWindow: capture from DWM

	gwChild = 5 // GetWindow: first child

	// Win32 mouse messages
	wmWVMouseMove   = 0x0200
	wmWVLButtonDown = 0x0201
	wmWVLButtonUp   = 0x0202
	wmWVRButtonDown = 0x0204
	wmWVRButtonUp   = 0x0205
	wmWVMButtonDown = 0x0207
	wmWVMButtonUp   = 0x0208
	wmWVMouseWheel  = 0x020A
	wmWVKeyDown     = 0x0100
	wmWVKeyUp       = 0x0101
	wmWVChar        = 0x0102

	mkWVLButton = 0x0001
	mkWVRButton = 0x0002
	mkWVMButton = 0x0010
)

func init() {
	newPlatformBackend = func(w *WebView) platformBackend {
		return newWindowsBackend(w)
	}
}

// windowsBackend hosts WebView2 in an offscreen popup window and captures
// its rendered output into a WGPU texture for compositing into the main
// swapchain. This eliminates the DWM dual-pipeline flicker problem and
// enables overlay support.
//
// When no renderer is configured (WithRenderer not called), the backend
// falls back to positioning the popup over the main window (legacy mode).
type windowsBackend struct {
	w *WebView

	mu sync.Mutex

	runtimeAvailable bool
	closed           bool
	comInitialized   bool

	comInit  com.Initialized
	comScope *com.Scope

	hwnd      win32.HWND // main application window
	popupHWND uintptr    // offscreen popup that hosts WebView2

	environment *wv2.ICoreWebView2Environment
	controller  *wv2.ICoreWebView2Controller
	core        *wv2.ICoreWebView2

	// --- Texture capture state (when renderer is available) ---
	renderer  *gpu.WGPURenderer
	capDC     uintptr        // memory DC for PrintWindow capture
	capBMP    uintptr        // DIB section bitmap (top-down, BGRA)
	capPixels unsafe.Pointer // direct pointer to DIB pixel buffer
	capW, capH int32         // current capture dimensions
	capTex     wgpu.Texture
	capView    wgpu.TextureView
	capTexID    draw.TextureID // stable texture ID (no churn)
	contentHWND uintptr       // WebView2's content child HWND (for input forwarding)

	// --- Legacy overlay state (when no renderer) ---
	textureID          draw.TextureID
	cachedClientBounds draw.Rect
	lastScreenRect     screenRect
	visible            bool

	lastErr error

	pendingURL     string
	pendingScripts []pendingScript

	envCompleted         *wv2.ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler
	controllerCompleted  *wv2.ICoreWebView2CreateCoreWebView2ControllerCompletedHandler
	navigationStarting   *wv2.ICoreWebView2NavigationStartingEventHandler
	navigationCompleted  *wv2.ICoreWebView2NavigationCompletedEventHandler
	historyChanged       *wv2.ICoreWebView2HistoryChangedEventHandler
	documentTitleChanged *wv2.ICoreWebView2DocumentTitleChangedEventHandler

	navigationStartingToken   wv2.EventRegistrationToken
	navigationCompleteToken   wv2.EventRegistrationToken
	historyChangedToken       wv2.EventRegistrationToken
	documentTitleChangedToken wv2.EventRegistrationToken
}

type screenRect struct {
	left, top, right, bottom int32
}

type pendingScript struct {
	js   string
	done chan error
}

// bitmapInfoHeader is the BITMAPINFOHEADER structure for CreateDIBSection.
type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func newWindowsBackend(w *WebView) *windowsBackend {
	b := &windowsBackend{
		w:        w,
		hwnd:     win32.HWND(w.cfg.parentWindow),
		renderer: w.cfg.renderer,
	}
	b.bootstrap()
	return b
}

// useCapture reports whether the backend should capture to a WGPU texture
// instead of overlaying the popup.
func (b *windowsBackend) useCapture() bool {
	return b.renderer != nil
}

// createPopup creates the popup HWND. When using capture mode, the popup
// is positioned offscreen so WebView2 renders without being visible.
func (b *windowsBackend) createPopup() uintptr {
	className := syscall.StringToUTF16Ptr("LuxM1Window")

	// Offscreen position for capture mode; on-screen placeholder for legacy.
	x, y := 0, 0
	if b.useCapture() {
		x, y = -32000, -32000
	}

	hwnd, _, _ := procWVCreateWindowEx.Call(
		uintptr(wsExToolWindow),            // no taskbar entry
		uintptr(unsafe.Pointer(className)), // reuse existing class
		0,                                  // no title
		uintptr(wsPopup|wsClipChildren),    // borderless popup
		uintptr(x), uintptr(y), 1, 1,      // initial size; resized in applyBounds
		uintptr(b.hwnd), // owner = main window
		0, 0, 0,
	)
	return hwnd
}

func (b *windowsBackend) bootstrap() {
	if err := procCreateCoreWebView2EnvironmentWithOptions.Find(); err != nil {
		b.setError(err)
		return
	}
	if b.hwnd == 0 {
		b.setError(errors.New("webview2 controller requires a parent HWND; use WithParentWindow"))
		return
	}

	b.popupHWND = b.createPopup()
	log.Printf("[webview] createPopup: hwnd=%#x popup=%#x capture=%v", b.hwnd, b.popupHWND, b.useCapture())
	if b.popupHWND == 0 {
		b.setError(errors.New("failed to create WebView2 popup host window"))
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
			b.mu.Unlock()

			b.controllerCompleted = wv2.NewICoreWebView2CreateCoreWebView2ControllerCompletedHandlerByFunc(
				func(errorCode com.Error, controller *wv2.ICoreWebView2Controller) com.Error {
					if errorCode < 0 || controller == nil {
						b.setError(fmt.Errorf("CreateCoreWebView2Controller callback failed: 0x%08x", uint32(errorCode)))
						return errorCode
					}
					controller.AddRef()

					var core *wv2.ICoreWebView2
					if err := controller.GetCoreWebView2(&core); err < 0 {
						b.setError(fmt.Errorf("GetCoreWebView2 failed: 0x%08x", uint32(err)))
						return com.Error(win32.E_FAIL)
					}
					if core == nil {
						b.setError(errors.New("GetCoreWebView2 returned nil core"))
						return com.Error(win32.E_FAIL)
					}
					core.AddRef()

					log.Printf("[webview] controller ready, popup=%#x", b.popupHWND)
					controller.SetIsVisible(1)

					// Find WebView2's content child HWND for input forwarding.
					childHWND, _, _ := procGetWindow.Call(b.popupHWND, gwChild)
					if childHWND != 0 {
						// WebView2 nests: popup → Chrome_WidgetWin → content.
						// Walk to the deepest first-child for input delivery.
						for {
							next, _, _ := procGetWindow.Call(childHWND, gwChild)
							if next == 0 {
								break
							}
							childHWND = next
						}
					}
					log.Printf("[webview] content child HWND=%#x", childHWND)

					b.mu.Lock()
					b.controller = controller
					b.core = core
					b.contentHWND = childHWND
					b.mu.Unlock()

					// Show the popup so DWM maintains its surface for capture.
					if b.useCapture() {
						procWVShowWindow.Call(b.popupHWND, swShowNA)
					}

					b.applyBounds(b.w.lastBounds)
					b.installEventHandlers(core)
					b.flushPending()
					b.syncCoreState()
					return 0
				}, true,
			)

			if err := createdEnvironment.CreateCoreWebView2Controller(
				win32.HWND(b.popupHWND), b.controllerCompleted,
			); err < 0 {
				b.setError(fmt.Errorf("CreateCoreWebView2Controller failed: 0x%08x", uint32(err)))
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
	hr := win32.HRESULT(r1)
	if callErr != syscall.Errno(0) {
		b.setError(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions call failed: %w", callErr))
		return
	}
	if win32.FAILED(hr) {
		b.setError(fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions failed: 0x%08x", uint32(hr)))
	}
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
	if b.controller != nil {
		_ = b.controller.Close()
		b.controller.Release()
		b.controller = nil
	}
	if b.environment != nil {
		b.environment.Release()
		b.environment = nil
	}
	if b.popupHWND != 0 {
		procWVDestroyWindow.Call(b.popupHWND)
		b.popupHWND = 0
	}

	// Clean up capture resources.
	b.destroyCaptureDC()
	if b.capView != nil {
		b.capView.Destroy()
		b.capView = nil
	}
	if b.capTex != nil {
		b.capTex.Destroy()
		b.capTex = nil
	}
	if b.capTexID != 0 && b.renderer != nil {
		b.renderer.UnregisterSurfaceTexture(b.capTexID)
	}

	b.textureID = 0
	b.visible = false
	b.cachedClientBounds = draw.Rect{}
	b.lastScreenRect = screenRect{}
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

// ─── AcquireFrame ──────────────────────────────────────────────────

func (b *windowsBackend) AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	if b.useCapture() {
		return b.acquireFrameCapture(bounds)
	}
	return b.acquireFrameLegacy(bounds)
}

// acquireFrameCapture captures the offscreen popup via PrintWindow and
// uploads the pixels to a WGPU texture for blitting.
//
// Hot path optimizations:
//   - Top-down DIB (negative Height) → no row flip needed
//   - DIB pixel buffer passed directly to WriteTexture → zero alloc per frame
//   - Stable texture ID → no register/unregister churn
func (b *windowsBackend) acquireFrameCapture(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	b.mu.Lock()
	controller := b.controller
	popup := b.popupHWND
	b.mu.Unlock()

	w, h := int32(bounds.W), int32(bounds.H)
	if controller == nil || popup == 0 || w <= 0 || h <= 0 {
		b.w.mu.Lock()
		defer b.w.mu.Unlock()
		return 0, b.w.currentTextureTokenLocked()
	}

	// Resize the offscreen popup and capture resources when bounds change.
	if w != b.capW || h != b.capH {
		procWVMoveWindow.Call(popup, uintptr(0xFFFF8300), uintptr(0xFFFF8300), uintptr(w), uintptr(h), 0) // -32000
		_ = controller.SetBounds(wv2.TagRECT{Left: 0, Top: 0, Right: w, Bottom: h})
		b.capW = w
		b.capH = h
		b.ensureCaptureDC(w, h)
		b.ensureCaptureTexture(w, h)
	}

	// Capture + upload (zero-alloc hot path).
	if b.capDC != 0 && b.capPixels != nil && b.capTex != nil {
		procPrintWindow.Call(popup, b.capDC, pwRenderFullContent)

		// DIB is top-down (negative Height) → pixels are already in correct
		// order for WGPU. Pass the DIB memory directly, no copy.
		stride := uint32(w) * 4
		pixels := unsafe.Slice((*byte)(b.capPixels), int(stride)*int(h))

		b.renderer.Queue().WriteTexture(
			&wgpu.ImageCopyTexture{Texture: b.capTex},
			pixels,
			&wgpu.TextureDataLayout{BytesPerRow: stride, RowsPerImage: uint32(h)},
			wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		)
	}

	// Use a stable texture ID — the view doesn't change between frames.
	if b.capTexID == 0 {
		b.capTexID = 1
	}
	if b.capView != nil {
		b.renderer.RegisterSurfaceTexture(b.capTexID, b.capView)
	}

	b.w.mu.Lock()
	defer b.w.mu.Unlock()
	return b.capTexID, b.w.currentTextureTokenLocked()
}

// acquireFrameLegacy positions the popup over the main window (old behavior).
func (b *windowsBackend) acquireFrameLegacy(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	b.applyBounds(bounds)
	b.w.mu.Lock()
	defer b.w.mu.Unlock()
	return b.textureID, b.w.currentTextureTokenLocked()
}

// ─── Capture infrastructure ────────────────────────────────────────

// ensureCaptureDC creates or recreates the GDI memory DC and DIB section
// for PrintWindow capture at the given dimensions.
func (b *windowsBackend) ensureCaptureDC(w, h int32) {
	b.destroyCaptureDC()

	hdc, _, _ := procCreateCompatDC.Call(0) // NULL = screen DC
	if hdc == 0 {
		log.Println("[webview] CreateCompatibleDC failed")
		return
	}

	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    w,
		Height:   -h, // negative = top-down DIB (row 0 = top, no flip needed)
		Planes:   1,
		BitCount: 32, // BGRA
	}
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(
		hdc,
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
		uintptr(unsafe.Pointer(&bits)),
		0, 0,
	)
	if bmp == 0 {
		procDeleteDC.Call(hdc)
		log.Println("[webview] CreateDIBSection failed")
		return
	}

	procSelectObject.Call(hdc, bmp)

	b.capDC = hdc
	b.capBMP = bmp
	b.capPixels = bits
}

func (b *windowsBackend) destroyCaptureDC() {
	if b.capBMP != 0 {
		procDeleteObject.Call(b.capBMP)
		b.capBMP = 0
	}
	if b.capDC != 0 {
		procDeleteDC.Call(b.capDC)
		b.capDC = 0
	}
	b.capPixels = nil
}

// ensureCaptureTexture creates or recreates the WGPU texture for upload.
func (b *windowsBackend) ensureCaptureTexture(w, h int32) {
	if b.capView != nil {
		b.capView.Destroy()
		b.capView = nil
	}
	if b.capTex != nil {
		b.capTex.Destroy()
		b.capTex = nil
	}

	device := b.renderer.Device()
	b.capTex = device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "webview-capture",
		Size:   wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if b.capTex != nil {
		b.capView = b.capTex.CreateView()
	}
}

// ─── Legacy overlay applyBounds ────────────────────────────────────

func (b *windowsBackend) applyBounds(bounds draw.Rect) {
	b.mu.Lock()
	controller := b.controller
	prevScreen := b.lastScreenRect
	wasVisible := b.visible
	popup := b.popupHWND
	lastW := int32(b.cachedClientBounds.W)
	lastH := int32(b.cachedClientBounds.H)
	b.mu.Unlock()

	if controller == nil || popup == 0 || bounds.W <= 0 || bounds.H <= 0 {
		return
	}

	w, h := int32(bounds.W), int32(bounds.H)
	sizeChanged := w != lastW || h != lastH

	var pt [2]int32
	pt[0] = int32(bounds.X)
	pt[1] = int32(bounds.Y)
	procWVClientToScreen.Call(uintptr(b.hwnd), uintptr(unsafe.Pointer(&pt[0])))

	cur := screenRect{left: pt[0], top: pt[1], right: pt[0] + w, bottom: pt[1] + h}

	if !sizeChanged && prevScreen != (screenRect{}) {
		dx := cur.left - prevScreen.left
		dy := cur.top - prevScreen.top
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx <= 2 && dy <= 2 {
			return
		}
	}

	procWVMoveWindow.Call(popup, uintptr(pt[0]), uintptr(pt[1]), uintptr(w), uintptr(h), 0)

	if sizeChanged {
		_ = controller.SetBounds(wv2.TagRECT{Left: 0, Top: 0, Right: w, Bottom: h})
	}

	b.mu.Lock()
	b.cachedClientBounds = bounds
	b.lastScreenRect = cur
	b.mu.Unlock()

	if !wasVisible {
		procWVShowWindow.Call(popup, swShow)
		b.mu.Lock()
		b.visible = true
		b.mu.Unlock()
	}
}

func (b *windowsBackend) ReleaseFrame(ui.FrameToken) {}

func (b *windowsBackend) HandleMsg(msg any) bool {
	b.mu.Lock()
	child := b.contentHWND
	popup := b.popupHWND
	b.mu.Unlock()

	// In legacy (overlay) mode, input goes directly to the popup via the
	// Windows message pump — no forwarding needed.
	if !b.useCapture() || child == 0 {
		return false
	}

	switch m := msg.(type) {
	case ui.SurfaceMouseMsg:
		x := int32(m.Pos.X)
		y := int32(m.Pos.Y)
		lParam := uintptr(uint16(y))<<16 | uintptr(uint16(x))

		switch m.Action {
		case input.MousePress:
			// Give Win32 focus to the WebView2 content HWND so it receives
			// keyboard input directly through the message pump.
			procSetFocus.Call(child)

			wmDown, mk := mouseButtonToWin32(m.Button)
			if wmDown != 0 {
				procPostMessageW.Call(child, uintptr(wmDown), uintptr(mk), lParam)
			}

		case input.MouseMove:
			// During drag, report which button is held.
			_, mk := mouseButtonToWin32(m.Button)
			procPostMessageW.Call(child, wmWVMouseMove, uintptr(mk), lParam)

		case input.MouseRelease:
			wmUp, _ := mouseButtonRelease(m.Button)
			if wmUp != 0 {
				procPostMessageW.Call(child, uintptr(wmUp), 0, lParam)
			}
		}
		return true

	case ui.SurfaceKeyMsg:
		// Keyboard events are normally delivered directly by Win32 focus.
		// This handles the case where the framework intercepts a key
		// before it reaches the popup (e.g. Tab navigation).
		vk := keyToVK(m.Key)
		if vk == 0 {
			return false
		}
		switch m.Action {
		case input.KeyPress, input.KeyRepeat:
			procPostMessageW.Call(popup, wmWVKeyDown, uintptr(vk), 0)
		case input.KeyRelease:
			procPostMessageW.Call(popup, wmWVKeyUp, uintptr(vk), 0)
		}
		return true
	}
	return false
}

// mouseButtonToWin32 returns the WM_*BUTTONDOWN message and MK_* flag.
func mouseButtonToWin32(btn input.MouseButton) (uint32, uint32) {
	switch btn {
	case input.MouseButtonLeft:
		return wmWVLButtonDown, mkWVLButton
	case input.MouseButtonRight:
		return wmWVRButtonDown, mkWVRButton
	case input.MouseButtonMiddle:
		return wmWVMButtonDown, mkWVMButton
	}
	return 0, 0
}

// mouseButtonRelease returns the WM_*BUTTONUP message.
func mouseButtonRelease(btn input.MouseButton) (uint32, uint32) {
	switch btn {
	case input.MouseButtonLeft:
		return wmWVLButtonUp, 0
	case input.MouseButtonRight:
		return wmWVRButtonUp, 0
	case input.MouseButtonMiddle:
		return wmWVMButtonUp, 0
	}
	return 0, 0
}

// keyToVK maps framework key constants to Win32 virtual key codes.
func keyToVK(k input.Key) uintptr {
	switch k {
	case input.KeyTab:
		return 0x09
	case input.KeyEnter:
		return 0x0D
	case input.KeyEscape:
		return 0x1B
	case input.KeyBackspace:
		return 0x08
	case input.KeyDelete:
		return 0x2E
	case input.KeyLeft:
		return 0x25
	case input.KeyUp:
		return 0x26
	case input.KeyRight:
		return 0x27
	case input.KeyDown:
		return 0x28
	case input.KeyHome:
		return 0x24
	case input.KeyEnd:
		return 0x23
	case input.KeySpace:
		return 0x20
	case input.KeyA:
		return 0x41
	case input.KeyC:
		return 0x43
	case input.KeyV:
		return 0x56
	case input.KeyX:
		return 0x58
	}
	return 0
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
