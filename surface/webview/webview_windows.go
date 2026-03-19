//go:build windows && !servo

package webview

import (
	"errors"
	"sync"
	"syscall"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
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
// The COM/DXGI members are intentionally kept behind this build-tagged file so
// Linux/macOS can add their own backends later without affecting the shared API.
type windowsBackend struct {
	w *WebView

	mu sync.Mutex

	runtimeAvailable bool
	closed           bool

	// COM anchor: ICoreWebView2CompositionController.
	environment uintptr
	controller  uintptr
	composition uintptr
	core        uintptr

	// Zero-copy anchor: DXGI Shared Handle imported by Lux/WGPU later.
	dxgiSharedHandle uintptr
	textureID        draw.TextureID
}

func newWindowsBackend(w *WebView) *windowsBackend {
	b := &windowsBackend{w: w}
	b.bootstrap()
	return b
}

func (b *windowsBackend) bootstrap() {
	if err := procCreateCoreWebView2EnvironmentWithOptions.Find(); err == nil {
		b.runtimeAvailable = true
	}
}

func (b *windowsBackend) Navigate(url string) {
	_ = url
	if !b.runtimeAvailable {
		b.w.setLoading(false)
		return
	}
	// TODO: Create WebView2 environment/controller, then call Navigate on the
	// ICoreWebView2 instance once the COM bootstrap is wired up.
	b.w.setLoading(false)
}

func (b *windowsBackend) Eval(js string) error {
	_ = js
	if !b.runtimeAvailable {
		return errors.New("webview2 runtime not available")
	}
	// TODO: Route to ICoreWebView2::ExecuteScript.
	return nil
}

func (b *windowsBackend) Reload() {
	if !b.runtimeAvailable {
		b.w.setLoading(false)
		return
	}
	// TODO: Route to ICoreWebView2::Reload.
	b.w.setLoading(false)
}

func (b *windowsBackend) Back() {
	if !b.runtimeAvailable {
		b.w.setLoading(false)
		return
	}
	// TODO: Route to native WebView2 history.
	b.w.setLoading(false)
}

func (b *windowsBackend) Forward() {
	if !b.runtimeAvailable {
		b.w.setLoading(false)
		return
	}
	// TODO: Route to native WebView2 history.
	b.w.setLoading(false)
}

func (b *windowsBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.dxgiSharedHandle = 0
	b.textureID = 0
	b.composition = 0
	b.controller = 0
	b.core = 0
	b.environment = 0
	return nil
}

func (b *windowsBackend) AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	_ = bounds
	b.w.mu.Lock()
	defer b.w.mu.Unlock()
	return b.textureID, b.w.currentTextureTokenLocked()
}

func (b *windowsBackend) ReleaseFrame(ui.FrameToken) {}

func (b *windowsBackend) HandleMsg(msg any) bool {
	switch msg.(type) {
	case ui.SurfaceMouseMsg, ui.SurfaceKeyMsg:
		// TODO: Translate Lux surface input into WebView2 pointer/key messages for
		// the composition controller path.
		return true
	default:
		return false
	}
}
