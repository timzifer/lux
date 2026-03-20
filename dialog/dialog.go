// Package dialog provides high-level Cmd-producing functions for modal dialogs.
//
// Each function returns an app.Cmd that:
//  1. Checks the active platform for NativeDialogProvider support.
//  2. If native is available, calls the native dialog on a background goroutine
//     and returns a result message when it completes.
//  3. If native is unavailable (or fails), returns a fallback message that the
//     user's update function should handle by showing a ui.MessageDialog,
//     ui.ConfirmDialog, or ui.InputDialog overlay.
package dialog

import (
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/platform"
)

// ── Result messages (sent when a dialog completes) ──────────────

// MessageResultMsg is sent when a message dialog is dismissed.
type MessageResultMsg struct{}

// ConfirmResultMsg is sent when a confirm dialog completes.
type ConfirmResultMsg struct {
	Confirmed bool
}

// InputResultMsg is sent when an input dialog completes.
type InputResultMsg struct {
	Value     string
	Confirmed bool
}

// ── Fallback messages (no native support — show overlay) ────────

// ShowFallbackMessageMsg requests the app to show a framework-rendered message dialog.
type ShowFallbackMessageMsg struct {
	Title   string
	Message string
	Kind    platform.DialogKind
}

// ShowFallbackConfirmMsg requests the app to show a framework-rendered confirm dialog.
type ShowFallbackConfirmMsg struct {
	Title   string
	Message string
}

// ShowFallbackInputMsg requests the app to show a framework-rendered input dialog.
type ShowFallbackInputMsg struct {
	Title        string
	Message      string
	DefaultValue string
}

// ── Cmd-producing functions ─────────────────────────────────────

// ShowMessage returns a Cmd that displays a message dialog.
func ShowMessage(title, message string, kind platform.DialogKind) app.Cmd {
	return func() app.Msg {
		plat := app.ActivePlatform()
		if dp, ok := plat.(platform.NativeDialogProvider); ok {
			if err := dp.ShowMessageDialog(title, message, kind); err == nil {
				return MessageResultMsg{}
			}
		}
		return ShowFallbackMessageMsg{Title: title, Message: message, Kind: kind}
	}
}

// ShowConfirm returns a Cmd that displays a confirm dialog.
func ShowConfirm(title, message string) app.Cmd {
	return func() app.Msg {
		plat := app.ActivePlatform()
		if dp, ok := plat.(platform.NativeDialogProvider); ok {
			confirmed, err := dp.ShowConfirmDialog(title, message)
			if err == nil {
				return ConfirmResultMsg{Confirmed: confirmed}
			}
		}
		return ShowFallbackConfirmMsg{Title: title, Message: message}
	}
}

// ShowInput returns a Cmd that displays an input dialog.
func ShowInput(title, message, defaultValue string) app.Cmd {
	return func() app.Msg {
		plat := app.ActivePlatform()
		if dp, ok := plat.(platform.NativeDialogProvider); ok {
			value, confirmed, err := dp.ShowInputDialog(title, message, defaultValue)
			if err == nil {
				return InputResultMsg{Value: value, Confirmed: confirmed}
			}
		}
		return ShowFallbackInputMsg{Title: title, Message: message, DefaultValue: defaultValue}
	}
}
