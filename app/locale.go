package app

import (
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// SetLocaleMsg changes the application locale at runtime (RFC-003 §3.8).
// The layout direction is derived automatically from the language tag.
// This triggers a full layout invalidation.
type SetLocaleMsg struct {
	// Locale is a BCP 47 language tag (e.g. "en", "ar", "he", "fa").
	Locale string
}

// WithLocale sets the initial application locale (BCP 47 tag).
// The layout direction (LTR/RTL) is derived from the primary language.
func WithLocale(locale string) Option {
	return func(o *options) {
		o.locale = locale
	}
}

// DirectionFromLocale derives the LayoutDirection from a BCP 47 language tag.
// Returns DirRTL for Arabic, Hebrew, Farsi/Persian, Urdu, Pashto, Sindhi,
// and other known RTL scripts. Returns DirLTR for everything else.
func DirectionFromLocale(locale string) draw.LayoutDirection {
	// Extract primary language subtag (before any '-' or '_').
	lang := strings.ToLower(locale)
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		lang = lang[:i]
	}
	switch lang {
	case "ar", "he", "fa", "ur", "ps", "sd", "yi", "ckb", "ug", "dv", "syr", "arc":
		return draw.DirRTL
	default:
		return draw.DirLTR
	}
}

// applyLocale updates the global layout direction from a locale string.
func applyLocale(locale string) {
	dir := DirectionFromLocale(locale)
	ui.SetDirection(dir)
}
