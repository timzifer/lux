// Package app provides the public API for lux applications (RFC §3).
//
// A lux application follows the Elm architecture:
//
//	app.Run(initialModel, update, view, opts...)
//
// The update function processes messages and returns a new model.
// The view function renders the model as an Element tree.
// Both run exclusively on the app loop goroutine — no races possible.
package app

import (
	"time"

	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/loop"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Msg is any value sent through the app loop. Every type is a valid Msg.
type Msg = any

// SetThemeMsg switches the active theme at runtime (RFC §5.5).
type SetThemeMsg struct{ Theme theme.Theme }

// SetDarkModeMsg toggles between the built-in dark and light themes (RFC §5.5).
type SetDarkModeMsg struct{ Dark bool }

// TickMsg is sent every frame with the elapsed delta time.
// Use this to drive animations, timers, or physics.
type TickMsg struct{ DeltaTime time.Duration }

// UpdateFunc is the signature for the update function (RFC §3.1).
type UpdateFunc[M any] func(M, Msg) M

// ViewFunc is the signature for the view function (RFC §3.1).
type ViewFunc[M any] func(M) ui.Element

// globalLoop holds the active loop instance for package-level Send/TrySend.
var globalLoop *loop.Loop

// globalFocus holds the app-level FocusManager for keyboard input routing.
var globalFocus = ui.NewFocusManager()

// Focus returns the app-level FocusManager for keyboard input routing.
// Pass this to ui.WithFocus when creating TextFields.
func Focus() *ui.FocusManager { return globalFocus }

// Send enqueues a message into the app loop. Thread-safe, never blocks (RFC §3.2).
func Send(msg Msg) {
	if globalLoop != nil {
		globalLoop.Send(msg)
	}
}

// TrySend attempts to enqueue a message. Returns false if the buffer is full.
func TrySend(msg Msg) bool {
	if globalLoop == nil {
		return false
	}
	return globalLoop.TrySend(msg)
}

// options holds configuration parsed from Option functions.
type options struct {
	title           string
	width           int
	height          int
	maxFrameDelta   time.Duration
	theme           theme.Theme
	platformFactory func() platform.Platform
	rendererFactory func() gpu.Renderer
}

func defaultOptions() options {
	return options{
		title:           "lux",
		width:           800,
		height:          600,
		maxFrameDelta:   loop.DefaultMaxFrameDelta,
		theme:           theme.Default,
		platformFactory: defaultPlatformFactory,
		rendererFactory: defaultRendererFactory,
	}
}

// Option configures the application.
type Option func(*options)

// WithTitle sets the window title.
func WithTitle(title string) Option {
	return func(o *options) { o.title = title }
}

// WithSize sets the initial window size in screen coordinates.
func WithSize(w, h int) Option {
	return func(o *options) {
		o.width = w
		o.height = h
	}
}

// WithMaxFrameDelta overrides the default dt clamp (RFC §3.3).
func WithMaxFrameDelta(d time.Duration) Option {
	return func(o *options) { o.maxFrameDelta = d }
}

// WithTheme sets the application theme (RFC §5).
func WithTheme(t theme.Theme) Option {
	return func(o *options) { o.theme = t }
}

// WithPlatform overrides the platform backend.
func WithPlatform(f func() platform.Platform) Option {
	return func(o *options) { o.platformFactory = f }
}

// WithRenderer overrides the GPU renderer.
func WithRenderer(f func() gpu.Renderer) Option {
	return func(o *options) { o.rendererFactory = f }
}
