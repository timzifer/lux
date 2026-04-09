// Package osk implements the On-Screen Keyboard (RFC-004 §5).
//
// The OSK is a framework-level overlay that appears automatically when a
// text field receives focus and HasPhysicalKeyboard == false in the active
// InteractionProfile. It injects characters and control keys through the
// existing input pipeline (input.CharMsg / input.KeyMsg).
package osk

// OSKLayout identifies which keyboard layout the focused widget requests (RFC-004 §5.3).
type OSKLayout uint8

const (
	// OSKLayoutAlpha: full QWERTY/QWERTZ/AZERTY layout with letter/symbol toggle.
	OSKLayoutAlpha OSKLayout = iota

	// OSKLayoutNumeric: digits 0–9, decimal separator, sign.
	OSKLayoutNumeric

	// OSKLayoutNumericInteger: digits 0–9 and sign, no decimal separator.
	OSKLayoutNumericInteger

	// OSKLayoutPhone: telephone number layout (0–9, +, *, #).
	OSKLayoutPhone

	// OSKLayoutNone signals that the widget provides its own inline keypad
	// and the global OSK should not appear.
	OSKLayoutNone OSKLayout = 255
)

// OSKAction describes what happens when an OSK key is tapped (RFC-004 §5.5).
type OSKAction uint8

const (
	OSKActionChar      OSKAction = iota // Insert a character
	OSKActionBackspace                  // Delete character before cursor
	OSKActionEnter                      // Confirm input / move to next field
	OSKActionShift                      // Toggle shift state
	OSKActionSwitch                     // Switch layout layer (alpha ↔ numeric/symbol)
	OSKActionSpace                      // Insert space
	OSKActionDismiss                    // Close the OSK
	OSKActionTab                        // Move focus to next field
	OSKActionSign                       // Toggle +/- sign
	OSKActionDecimal                    // Insert locale-aware decimal separator
)

// OSKKey represents a single key on the on-screen keyboard (RFC-004 §5.5).
type OSKKey struct {
	Label  string    // Display text ("Q", "123", or Phosphor codepoint)
	Action OSKAction // What happens on tap
	Width  float32   // Relative width (1.0 = standard key)
	Char   rune      // Character to insert (only for OSKActionChar)
	IsIcon bool      // Label is a Phosphor icon codepoint
}

// OSKMode is a higher-level presentation mode controlling the overall
// keyboard appearance. This extends the RFC's per-widget OSKLayout with
// user-facing display modes.
type OSKMode uint8

const (
	// ModeAlpha shows the alphabetic keyboard with a number row.
	ModeAlpha OSKMode = iota

	// ModeNumPad shows a numeric keypad only.
	ModeNumPad

	// ModeFull shows the alphabetic keyboard and numpad side by side.
	ModeFull

	// ModeCondensed shows a compact phone-style keyboard.
	ModeCondensed
)

// OSKRequester is implemented by widgets that need a specific OSK layout
// when they receive focus (RFC-004 §5.4).
type OSKRequester interface {
	OSKLayout() OSKLayout
}

// OSKState holds the current on-screen keyboard state.
// Managed by the framework, not by user code.
type OSKState struct {
	Visible bool      // Whether the OSK is currently shown
	Layout  OSKLayout // Which layout was requested by the focused widget
	Mode    OSKMode   // Active presentation mode
	Shifted bool      // Shift key is active (uppercase / symbols)
}

// Height returns the OSK height in dp for the given screen dimensions and DPR.
// Returns 0 when the OSK is not visible.
func (s *OSKState) Height(screenW, screenH int, dpr float32) float32 {
	if s == nil || !s.Visible {
		return 0
	}
	rows := s.Mode.rows()
	_, keyH, gap := ComputeKeySize(screenW, screenH, dpr, s.Mode)
	return float32(rows)*(keyH+gap) + gap
}

// rows returns the number of key rows for the given mode.
func (m OSKMode) rows() int {
	switch m {
	case ModeAlpha:
		return 5 // number row + 3 letter rows + bottom bar
	case ModeNumPad:
		return 4
	case ModeFull:
		return 5
	case ModeCondensed:
		return 4 // 3 letter rows + bottom bar
	default:
		return 5
	}
}

// ModeForLayout returns the default OSKMode for a given OSKLayout.
func ModeForLayout(layout OSKLayout) OSKMode {
	switch layout {
	case OSKLayoutNumeric, OSKLayoutNumericInteger, OSKLayoutPhone:
		return ModeNumPad
	case OSKLayoutNone:
		return ModeNumPad // irrelevant — global OSK won't show
	default:
		return ModeAlpha
	}
}

// ComputeKeySize calculates key dimensions in dp, capping at ~68dp
// (approximately 18mm physical size at 96 DPI) to prevent oversized keys.
func ComputeKeySize(screenW, screenH int, dpr float32, mode OSKMode) (keyW, keyH, gap float32) {
	keysPerRow := 10 // alpha default
	switch mode {
	case ModeNumPad:
		keysPerRow = 3
	case ModeFull:
		keysPerRow = 14 // 10 alpha + gap + 3 numpad
	case ModeCondensed:
		keysPerRow = 10
	}

	gap = 4.0
	availW := float32(screenW) - gap*2 // outer padding
	keyW = (availW - gap*float32(keysPerRow-1)) / float32(keysPerRow)
	keyH = keyW * 0.8

	// DPI cap: max ~68dp ≈ 18mm at standard 96 DPI.
	const maxKeyDp = 68.0
	if keyW > maxKeyDp {
		keyW = maxKeyDp
	}
	if keyH > maxKeyDp {
		keyH = maxKeyDp
	}

	// Minimum usable size.
	const minKeyDp = 28.0
	if keyW < minKeyDp {
		keyW = minKeyDp
	}
	if keyH < minKeyDp {
		keyH = minKeyDp
	}

	return keyW, keyH, gap
}
