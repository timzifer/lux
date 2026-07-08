package ui

// DialogKind specifies the severity/icon of a dialog rendered by the
// framework (DialogLayout, ui/dialog overlays). It mirrors the platform
// package's native dialog kind without introducing a ui → platform
// dependency; the top-level dialog package maps between the two.
type DialogKind uint8

const (
	DialogInfo    DialogKind = iota // Informational (ℹ)
	DialogWarning                   // Warning (⚠)
	DialogError                     // Error (✖)
)
