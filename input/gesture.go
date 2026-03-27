// Package input — gesture.go defines high-level gesture message types
// derived from raw TouchMsg sequences (RFC-004 §3.3).
//
// The GestureRecognizer (in package ui) transforms TouchMsg streams
// into these semantic messages. They are delivered to widgets via
// RenderCtx.Events, replacing the consumed TouchMsgs.
package input

// ── Gesture Point ────────────────────────────────────────────────

// GesturePoint is a 2D position in dp, used by gesture messages.
type GesturePoint struct {
	X, Y float32
}

// ── TapMsg ───────────────────────────────────────────────────────

// TapMsg is emitted when a finger touches and lifts within
// DragThreshold and LongPressDuration (RFC-004 §3.3).
type TapMsg struct {
	Pos   GesturePoint // Position of the tap
	Count int          // 1 = single-tap, 2 = double-tap, etc.
}

// ── LongPressMsg ─────────────────────────────────────────────────

// LongPressPhase describes the lifecycle of a long-press gesture.
type LongPressPhase uint8

const (
	LongPressBegan     LongPressPhase = iota // Threshold reached
	LongPressEnded                           // Finger lifted
	LongPressCancelled                       // Finger moved / OS interrupt
)

// LongPressMsg is emitted when a finger rests longer than
// LongPressDuration without exceeding DragThreshold (RFC-004 §3.3).
type LongPressMsg struct {
	Pos   GesturePoint
	Phase LongPressPhase
}

// ── SwipeMsg ─────────────────────────────────────────────────────

// SwipeDirection identifies the primary axis of a swipe gesture.
type SwipeDirection uint8

const (
	SwipeLeft SwipeDirection = iota
	SwipeRight
	SwipeUp
	SwipeDown
)

// SwipeMsg is emitted for a fast linear movement exceeding the
// swipe threshold (RFC-004 §3.3).
type SwipeMsg struct {
	Direction SwipeDirection
	Velocity  float32      // dp/s
	Start     GesturePoint
	End       GesturePoint
}

// ── DragMsg ──────────────────────────────────────────────────────

// DragPhase describes the lifecycle of a drag gesture.
type DragPhase uint8

const (
	DragBegan     DragPhase = iota
	DragMoved
	DragEnded
	DragCancelled
)

// DragMsg is emitted for slow movement exceeding DragThreshold.
// Unlike SwipeMsg, velocity is below SwipeVelocityThreshold (RFC-004 §3.3).
type DragMsg struct {
	Phase DragPhase
	Start GesturePoint // Start position
	Pos   GesturePoint // Current position
	Delta GesturePoint // Movement since last frame
}

// ── PinchMsg ─────────────────────────────────────────────────────

// PinchPhase describes the lifecycle of a two-finger pinch gesture.
type PinchPhase uint8

const (
	PinchBegan     PinchPhase = iota
	PinchChanged
	PinchEnded
	PinchCancelled
)

// PinchMsg is emitted for a two-finger gesture (RFC-004 §3.3).
// Scale is relative to the start distance (1.0 = unchanged).
type PinchMsg struct {
	Phase  PinchPhase
	Center GesturePoint // Midpoint between the two fingers
	Scale  float32      // >1.0 = zoom in, <1.0 = zoom out
}
