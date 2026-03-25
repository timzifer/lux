package webview

import (
	"sync"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/ui"
)

// Option configures a WebView.
type Option func(*config)

type config struct {
	title        string
	parentWindow uintptr
	userDataDir  string
	renderer     *gpu.WGPURenderer
}

// WithTitle sets the initial page title metadata.
func WithTitle(title string) Option {
	return func(cfg *config) { cfg.title = title }
}

// WithParentWindow sets the host HWND required by the Windows
// ICoreWebView2CompositionController path.
func WithParentWindow(hwnd uintptr) Option {
	return func(cfg *config) { cfg.parentWindow = hwnd }
}

// WithUserDataDir overrides the WebView2 user-data folder.
func WithUserDataDir(dir string) Option {
	return func(cfg *config) { cfg.userDataDir = dir }
}

// WithRenderer connects the WebView to the WGPU renderer for texture capture.
// When set, the backend renders WebView2 content into a WGPU texture that is
// blitted into the main swapchain (no popup overlay, full overlay support).
func WithRenderer(r *gpu.WGPURenderer) Option {
	return func(cfg *config) { cfg.renderer = r }
}

// WebView is a ui.SurfaceProvider that renders web content.
//
// The public API is intentionally small so Linux/macOS implementations can be
// added later via build tags without changing call sites.
type WebView struct {
	mu sync.RWMutex

	backend platformBackend
	cfg     config

	currentURL string
	title      string
	loading    bool

	history      []string
	historyIndex int
	canGoBack    bool
	canGoForward bool

	closed bool

	lastBounds draw.Rect
	nextToken  ui.FrameToken
}

type platformBackend interface {
	Navigate(url string)
	Eval(js string) error
	Reload()
	Back()
	Forward()
	Close() error
	AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken)
	ReleaseFrame(token ui.FrameToken)
	HandleMsg(msg any) bool
}

var newPlatformBackend = func(w *WebView) platformBackend {
	return &stubBackend{w: w}
}

// New creates a new WebView using the platform-specific engine.
func New(url string, opts ...Option) *WebView {
	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	w := &WebView{
		cfg:          cfg,
		title:        cfg.title,
		historyIndex: -1,
	}
	w.backend = newPlatformBackend(w)
	if url != "" {
		w.Navigate(url)
	}
	return w
}

// Navigate loads a new URL.
func (w *WebView) Navigate(url string) {
	if url == "" {
		return
	}

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.pushHistoryLocked(url)
	w.loading = true
	backend := w.backend
	w.mu.Unlock()

	backend.Navigate(url)
}

// Eval prepares JavaScript execution support for platform implementations.
func (w *WebView) Eval(js string) error {
	w.mu.RLock()
	backend := w.backend
	w.mu.RUnlock()
	return backend.Eval(js)
}

// Reload reloads the current page.
func (w *WebView) Reload() {
	w.mu.Lock()
	if w.closed || w.currentURL == "" {
		w.mu.Unlock()
		return
	}
	w.loading = true
	backend := w.backend
	w.mu.Unlock()

	backend.Reload()
}

// Back navigates backwards in history if possible.
func (w *WebView) Back() {
	w.mu.Lock()
	if w.closed || !w.canGoBack || w.historyIndex <= 0 {
		w.mu.Unlock()
		return
	}
	w.historyIndex--
	w.currentURL = w.history[w.historyIndex]
	w.loading = true
	w.syncHistoryFlagsLocked()
	backend := w.backend
	w.mu.Unlock()

	backend.Back()
}

// Forward navigates forwards in history if possible.
func (w *WebView) Forward() {
	w.mu.Lock()
	if w.closed || !w.canGoForward || w.historyIndex+1 >= len(w.history) {
		w.mu.Unlock()
		return
	}
	w.historyIndex++
	w.currentURL = w.history[w.historyIndex]
	w.loading = true
	w.syncHistoryFlagsLocked()
	backend := w.backend
	w.mu.Unlock()

	backend.Forward()
}

// CanGoBack reports whether backwards navigation is currently possible.
func (w *WebView) CanGoBack() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.canGoBack
}

// CanGoForward reports whether forward navigation is currently possible.
func (w *WebView) CanGoForward() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.canGoForward
}

// Close releases all engine resources.
func (w *WebView) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.loading = false
	backend := w.backend
	w.mu.Unlock()

	return backend.Close()
}

// AcquireFrame returns the current texture for the web content.
func (w *WebView) AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	w.mu.Lock()
	w.lastBounds = bounds
	backend := w.backend
	w.mu.Unlock()

	return backend.AcquireFrame(bounds)
}

// ReleaseFrame releases a frame token previously returned by AcquireFrame.
func (w *WebView) ReleaseFrame(token ui.FrameToken) {
	w.mu.RLock()
	backend := w.backend
	w.mu.RUnlock()
	backend.ReleaseFrame(token)
}

// HandleMsg routes surface input to the platform engine.
func (w *WebView) HandleMsg(msg any) bool {
	w.mu.RLock()
	backend := w.backend
	w.mu.RUnlock()
	return backend.HandleMsg(msg)
}

func (w *WebView) pushHistoryLocked(url string) {
	if w.historyIndex+1 < len(w.history) {
		w.history = append([]string(nil), w.history[:w.historyIndex+1]...)
	}
	w.history = append(w.history, url)
	w.historyIndex = len(w.history) - 1
	w.currentURL = url
	w.syncHistoryFlagsLocked()
}

func (w *WebView) syncHistoryFlagsLocked() {
	w.canGoBack = w.historyIndex > 0
	w.canGoForward = w.historyIndex >= 0 && w.historyIndex < len(w.history)-1
}

// Title returns the current page title.
func (w *WebView) Title() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.title
}

// CurrentURL returns the URL of the currently loaded page.
func (w *WebView) CurrentURL() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentURL
}

// IsLoading reports whether a navigation is currently in progress.
func (w *WebView) IsLoading() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.loading
}

func (w *WebView) setTitle(title string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.title = title
}

func (w *WebView) setLoading(loading bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.loading = loading
}

func (w *WebView) setCurrentURL(url string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.currentURL = url
}

func (w *WebView) setHistoryAvailability(canGoBack, canGoForward bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.canGoBack = canGoBack
	w.canGoForward = canGoForward
}

func (w *WebView) currentTextureTokenLocked() ui.FrameToken {
	w.nextToken++
	if w.nextToken == 0 {
		w.nextToken++
	}
	return w.nextToken
}

type stubBackend struct {
	w *WebView
}

func (b *stubBackend) Navigate(string)            { b.w.setLoading(false) }
func (b *stubBackend) Eval(string) error          { return nil }
func (b *stubBackend) Reload()                    { b.w.setLoading(false) }
func (b *stubBackend) Back()                      { b.w.setLoading(false) }
func (b *stubBackend) Forward()                   { b.w.setLoading(false) }
func (b *stubBackend) Close() error               { return nil }
func (b *stubBackend) ReleaseFrame(ui.FrameToken) {}
func (b *stubBackend) HandleMsg(any) bool         { return false }
func (b *stubBackend) AcquireFrame(draw.Rect) (draw.TextureID, ui.FrameToken) {
	b.w.mu.Lock()
	defer b.w.mu.Unlock()
	return 0, b.w.currentTextureTokenLocked()
}
