// Package platform defines the Platform interface for windowing backends (RFC §7.1).
//
// This file implements the Haptics API (RFC-004 §4.2).

package platform

// HapticStyle selects the intensity and pattern of haptic feedback.
type HapticStyle uint8

const (
	HapticLight  HapticStyle = iota // Subtle tap feedback
	HapticMedium                    // Confirmation
	HapticHeavy                     // Warning / hold-complete
	HapticError                     // Double-vibration error
)

// Haptics is an optional interface that platform backends can implement
// to provide haptic feedback on supported hardware. Widgets trigger
// haptics via this interface; on platforms without a vibration motor
// every call is a no-op.
type Haptics interface {
	Vibrate(style HapticStyle)
}

// NoopHaptics is the default implementation for platforms without
// vibration hardware. Every method is a no-op.
type NoopHaptics struct{}

// Vibrate is a no-op on platforms without haptic support.
func (NoopHaptics) Vibrate(HapticStyle) {}
