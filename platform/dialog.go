// Package platform — dialog.go defines the optional NativeDialogProvider interface.
//
// Backends that support OS-native modal dialogs implement this interface.
// The dialog package checks for it via type assertion at runtime.
package platform

// DialogKind specifies the severity/icon of a message dialog.
type DialogKind uint8

const (
	DialogInfo    DialogKind = iota // Informational (ℹ)
	DialogWarning                   // Warning (⚠)
	DialogError                     // Error (✖)
)

// NativeDialogProvider is an optional interface that Platform implementations
// can implement to show OS-native modal dialogs. Usage:
//
//	if dp, ok := plat.(NativeDialogProvider); ok { ... }
type NativeDialogProvider interface {
	// ShowMessageDialog displays a message dialog and blocks until dismissed.
	ShowMessageDialog(title, message string, kind DialogKind) error

	// ShowConfirmDialog displays a Yes/No dialog and returns the user's choice.
	ShowConfirmDialog(title, message string) (confirmed bool, err error)

	// ShowInputDialog displays a dialog with a text input field.
	// Returns the entered text and whether the user confirmed (OK vs Cancel).
	ShowInputDialog(title, message, defaultValue string) (value string, confirmed bool, err error)
}
