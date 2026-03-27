package ui

import "github.com/timzifer/lux/draw"

// globalDirection holds the current layout direction set by the app.
// Defaults to LTR. Updated by app.SetLocaleMsg or app.WithLocale.
var globalDirection draw.LayoutDirection

// SetDirection sets the global layout direction (RFC-002 §4.6).
// Called by the app package when locale changes.
func SetDirection(dir draw.LayoutDirection) {
	globalDirection = dir
}

// Direction returns the current global layout direction.
func Direction() draw.LayoutDirection {
	return globalDirection
}
