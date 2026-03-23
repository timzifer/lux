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

	luximage "github.com/timzifer/lux/image"
	"github.com/timzifer/lux/input"
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

// ModelRestoredMsg is sent once after a persisted model is successfully loaded (RFC §3.4).
// Handle this in your update function to apply side effects from the restored state
// (e.g., sending SetDarkModeMsg to match a persisted theme preference).
type ModelRestoredMsg struct{}

// TickMsg is sent every frame with the elapsed delta time.
// Use this to drive animations, timers, or physics.
type TickMsg struct{ DeltaTime time.Duration }

// Cmd is a function that performs a side effect and optionally returns a Msg (RFC §3.6).
// A nil Cmd means "no command".
type Cmd func() Msg

// None is a readable nil-Cmd sentinel.
var None Cmd

// UpdateFunc is the signature for the update function (RFC §3.1).
type UpdateFunc[M any] func(M, Msg) M

// UpdateWithCmd is the signature for update functions that return commands (RFC §3.6).
type UpdateWithCmd[M any] func(M, Msg) (M, Cmd)

// ViewFunc is the signature for the view function (RFC §3.1).
type ViewFunc[M any] func(M) ui.Element

// MultiViewFunc is the signature for a multi-window view function.
// It returns a map of window IDs to their element trees.
type MultiViewFunc[M any] func(M) map[WindowID]ui.Element

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

// shortcutEntry binds a key combination to a user-defined ID.
type shortcutEntry struct {
	shortcut input.Shortcut
	id       input.ShortcutID
}

// globalHandlerEntry binds an optional ID to a global input handler.
type globalHandlerEntry struct {
	id      HandlerID
	handler GlobalHandler
}

// options holds configuration parsed from Option functions.
type options struct {
	title           string
	width           int
	height          int
	maxFrameDelta   time.Duration
	theme           theme.Theme
	locale          string // BCP 47 language tag (RFC-003 §3.8)
	platformFactory func() platform.Platform
	rendererFactory func() gpu.Renderer
	shortcuts       []shortcutEntry
	globalHandlers  []globalHandlerEntry
	persistence     *persistenceHooks
	storagePath     string
	fullscreen      bool
	imageStore      *luximage.Store
}

// Batch combines multiple Cmds into a single Cmd.
// Nil commands are filtered out. If no live commands remain, nil is returned.
func Batch(cmds ...Cmd) Cmd {
	var live []Cmd
	for _, c := range cmds {
		if c != nil {
			live = append(live, c)
		}
	}
	if len(live) == 0 {
		return nil
	}
	if len(live) == 1 {
		return live[0]
	}
	return func() Msg {
		for _, c := range live[:len(live)-1] {
			go func(cmd Cmd) {
				if r := cmd(); r != nil {
					Send(r)
				}
			}(c)
		}
		return live[len(live)-1]()
	}
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

// WithImageStore registers an image store for automatic GPU texture sync.
// Before each frame, dirty images are uploaded to the renderer.
func WithImageStore(s *luximage.Store) Option {
	return func(o *options) { o.imageStore = s }
}

// WithShortcut registers a global keyboard shortcut (RFC-002 §2.5).
// When the key combination is pressed, a ShortcutMsg with the given ID
// is sent into the user update loop.
func WithShortcut(s input.Shortcut, id input.ShortcutID) Option {
	return func(o *options) {
		o.shortcuts = append(o.shortcuts, shortcutEntry{shortcut: s, id: id})
	}
}

// ── Global Handler Layer (RFC-002 §2.8) ─────────────────────────

// HandlerID identifies a dynamically registered global handler.
type HandlerID string

// GlobalHandler processes input events before normal widget dispatch.
// Return true to consume the event (prevent widget delivery).
type GlobalHandler func(event ui.InputEvent) (consumed bool)

// RegisterHandlerMsg dynamically registers a global handler at runtime.
type RegisterHandlerMsg struct {
	ID      HandlerID
	Handler GlobalHandler
}

// UnregisterHandlerMsg removes a dynamically registered handler.
type UnregisterHandlerMsg struct {
	ID HandlerID
}

// WithGlobalHandler registers a static global handler (RFC-002 §2.8).
// Static handlers are always active and checked before dynamic ones.
func WithGlobalHandler(h GlobalHandler) Option {
	return func(o *options) {
		o.globalHandlers = append(o.globalHandlers, globalHandlerEntry{handler: h})
	}
}

// ── Phase 5 — Platform Extension (RFC §7.1) ─────────────────────

// SetSizeMsg requests a window resize.
type SetSizeMsg struct{ Width, Height int }

// SetFullscreenMsg requests fullscreen toggle.
type SetFullscreenMsg struct{ Fullscreen bool }

// WithFullscreen starts the application in fullscreen mode (RFC §7.1).
func WithFullscreen(fullscreen bool) Option {
	return func(o *options) { o.fullscreen = fullscreen }
}

// activePlatform holds the platform for clipboard access from package-level functions.
var activePlatform platform.Platform

// ActivePlatform returns the active platform backend.
// Used by the dialog package to detect NativeDialogProvider support.
func ActivePlatform() platform.Platform {
	return activePlatform
}

// SetActivePlatformForTest sets the active platform for testing purposes.
// This should only be used in tests.
func SetActivePlatformForTest(p platform.Platform) {
	activePlatform = p
}

// SetClipboard sets the system clipboard text. Thread-safe.
func SetClipboard(text string) error {
	if activePlatform != nil {
		return activePlatform.SetClipboard(text)
	}
	return nil
}

// GetClipboard returns the system clipboard text. Thread-safe.
func GetClipboard() (string, error) {
	if activePlatform != nil {
		return activePlatform.GetClipboard()
	}
	return "", nil
}
