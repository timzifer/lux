// Package interaction defines InteractionProfile and predefined profiles
// for adapting widget sizing, gesture thresholds, and input behavior to
// different target environments (RFC-004 §2).
//
// An InteractionProfile is not a Theme — it influences layout and input
// dispatch, not visual rendering. Profiles are propagated through
// RenderCtx and consumed by the layout system and gesture recognizer.
package interaction

import "time"

// OSKPresentation controls how the on-screen keyboard is presented
// when HasPhysicalKeyboard is false (RFC-004 §5).
type OSKPresentation uint8

const (
	// OSKPresentationInline is deprecated. The framework now always uses
	// ActionSheet mode. Kept as the zero value for backward compatibility.
	OSKPresentationInline OSKPresentation = iota

	// OSKPresentationActionSheet opens an ActionSheet overlay containing
	// an interactive copy (input proxy) of the focused widget at the top
	// and the keyboard at the bottom. The app viewport is not shrunk.
	// This is now the only supported mode.
	OSKPresentationActionSheet
)

// PointerKind describes the primary input device (RFC-004 §2.2).
type PointerKind uint8

const (
	PointerMouse  PointerKind = iota // Mouse/trackpad — precise
	PointerFinger                    // Capacitive touch — ~7mm contact area
	PointerGlove                     // Glove touch — ≥15mm contact area
	PointerStylus                    // Stylus — precise, but no hover
)

// InteractionProfile describes the interaction characteristics of the
// target environment. It lives alongside the Theme but is not a Theme
// token — it influences layout and dispatch, not rendering (RFC-004 §2.2).
type InteractionProfile struct {
	// PointerKind: primary input device type.
	PointerKind PointerKind

	// MinTouchTarget: minimum interactive area in dp.
	// Desktop: 24dp, Touch: 48dp, Glove: 64dp.
	MinTouchTarget float32

	// TouchTargetSpacing: minimum spacing between interactive elements in dp.
	// Prevents mis-taps. Desktop: 0dp, Touch: 8dp, Glove: 12dp.
	TouchTargetSpacing float32

	// HasHover: whether hover states exist.
	// false on pure touch devices — eliminates all hover feedback.
	HasHover bool

	// HasPhysicalKeyboard: whether a physical keyboard is present.
	// false → OSK is shown when a text field receives focus.
	HasPhysicalKeyboard bool

	// LongPressDuration: duration until a long-press is triggered.
	// Default: 500ms. HMI: 400ms (faster workflow).
	LongPressDuration time.Duration

	// DoubleTapInterval: maximum time between two taps for double-tap recognition.
	DoubleTapInterval time.Duration

	// DragThreshold: minimum movement in dp before a tap becomes a drag.
	// Higher on touch devices to compensate for finger tremor.
	// Desktop: 4dp, Touch: 10dp.
	DragThreshold float32

	// DebounceInterval: minimum time between two accepted taps on the same
	// element. Prevents accidental double-activation.
	// 0 = no debounce. HMI default: 200ms.
	DebounceInterval time.Duration

	// ScaleTypography: global typography scale factor.
	// 1.0 = desktop default (13dp body). HMI: 1.3–1.5.
	ScaleTypography float32

	// ReducedMotion: when true, non-essential animations are replaced by
	// immediate state changes. Only essential animations (progress rings,
	// alarm blinks) remain active (RFC-004 §10.2).
	ReducedMotion bool

	// OSKPresentation controls how the on-screen keyboard appears.
	// The framework now always uses ActionSheet mode regardless of this value.
	// Kept for backward compatibility.
	OSKPresentation OSKPresentation

	// NoCompositor indicates the app runs without a window compositor
	// (e.g. DRM/KMS direct rendering). When true, multi-window calls
	// are redirected to an internal tab panel instead of creating
	// real OS windows.
	NoCompositor bool
}

// ProfileDesktop is the standard desktop profile with mouse and keyboard.
var ProfileDesktop = InteractionProfile{
	PointerKind:         PointerMouse,
	MinTouchTarget:      24,
	TouchTargetSpacing:  0,
	HasHover:            true,
	HasPhysicalKeyboard: true,
	LongPressDuration:   500 * time.Millisecond,
	DoubleTapInterval:   400 * time.Millisecond,
	DragThreshold:       4,
	DebounceInterval:    0,
	ScaleTypography:     1.0,
	ReducedMotion:       false,
}

// ProfileTouch is for capacitive touchscreens without a physical keyboard.
var ProfileTouch = InteractionProfile{
	PointerKind:         PointerFinger,
	MinTouchTarget:      48,
	TouchTargetSpacing:  8,
	HasHover:            false,
	HasPhysicalKeyboard: false,
	LongPressDuration:   400 * time.Millisecond,
	DoubleTapInterval:   350 * time.Millisecond,
	DragThreshold:       10,
	DebounceInterval:    200 * time.Millisecond,
	ScaleTypography:     1.3,
	ReducedMotion:       false,
	OSKPresentation:     OSKPresentationActionSheet,
}

// ProfileHMI is for industrial touch panels with glove operation.
var ProfileHMI = InteractionProfile{
	PointerKind:         PointerGlove,
	MinTouchTarget:      64,
	TouchTargetSpacing:  12,
	HasHover:            false,
	HasPhysicalKeyboard: false,
	LongPressDuration:   400 * time.Millisecond,
	DoubleTapInterval:   350 * time.Millisecond,
	DragThreshold:       14,
	DebounceInterval:    250 * time.Millisecond,
	ScaleTypography:     1.5,
	ReducedMotion:       false,
	OSKPresentation:     OSKPresentationActionSheet,
}
