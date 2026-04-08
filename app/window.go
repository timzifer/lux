package app

import "github.com/timzifer/lux/ui/nav"

// WindowID identifies a window in the multi-window system.
type WindowID uint32

// MainWindow is the ID of the primary application window.
const MainWindow WindowID = 0

// WindowType describes the kind of window being created.
type WindowType int

const (
	// WindowTypeNormal is a standard resizable application window.
	WindowTypeNormal WindowType = iota
	// WindowTypeDialog is a dialog-style window (non-resizable by default).
	WindowTypeDialog
	// WindowTypeToolbar is a toolbar/palette window (always on top, no taskbar entry).
	WindowTypeToolbar
	// WindowTypePopup is a borderless popup window.
	WindowTypePopup
)

// WindowConfig describes parameters for creating a new window.
type WindowConfig struct {
	Title     string
	Type      WindowType
	Width     int
	Height    int
	Resizable bool // default follows Type: true for Normal, false for Dialog
}

// WindowOpenedMsg is sent when a window has been successfully created.
type WindowOpenedMsg struct{ Window WindowID }

// WindowClosedMsg is sent when a window has been destroyed.
type WindowClosedMsg struct{ Window WindowID }

// OpenWindowMsg requests opening a new window with the given configuration.
type OpenWindowMsg struct {
	ID     WindowID
	Config WindowConfig
}

// CloseWindowMsg requests closing an existing window.
type CloseWindowMsg struct{ ID WindowID }

// OpenWindow returns a Cmd that opens a new window.
func OpenWindow(id WindowID, cfg WindowConfig) Cmd {
	return func() Msg {
		return OpenWindowMsg{ID: id, Config: cfg}
	}
}

// CloseWindow returns a Cmd that closes a window.
func CloseWindow(id WindowID) Cmd {
	return func() Msg {
		return CloseWindowMsg{ID: id}
	}
}

// SelectWindowTabMsg is an internal message used by the no-compositor tab
// panel to switch the active tab.
type SelectWindowTabMsg struct{ ID WindowID }

// SetTabBarPositionMsg changes the tab bar position at runtime.
// Only effective when the application is in no-compositor or fullscreen mode.
type SetTabBarPositionMsg struct{ Position nav.TabPosition }
