package ui

import "github.com/timzifer/lux/input"

// InputEventKind identifies the concrete type inside an InputEvent (RFC-002 §2.6).
type InputEventKind int

const (
	EventKey InputEventKind = iota
	EventTextInput
	EventChar
	EventMouse
	EventScroll
	EventTouch
	EventFocusGained
	EventFocusLost
	EventIMECompose // IME composition state changed (RFC-002 §2.2)
	EventIMECommit  // IME text committed (RFC-002 §2.2)

	// Gesture events derived from TouchMsg sequences (RFC-004 §3.3).
	EventTap
	EventLongPress
	EventSwipe
	EventDrag
	EventPinch

	// Drag-and-drop events (RFC-005 §5).
	EventDragEnter // cursor carrying drag data entered a drop target
	EventDragOver  // cursor moving within a drop target
	EventDragLeave // cursor left a drop target
	EventDrop      // data dropped on a target
)

// InputEvent is a typed union wrapper for all input events delivered to
// a widget via RenderCtx.Events (RFC-002 §2.6). Exactly one of the
// typed fields is populated, identified by Kind.
type InputEvent struct {
	Kind InputEventKind

	Key       *input.KeyMsg
	TextInput *input.TextInputMsg
	Char      *input.CharMsg
	Mouse     *input.MouseMsg
	Scroll    *input.ScrollMsg
	Touch     *input.TouchMsg

	FocusGained *FocusGainedMsg
	FocusLost   *FocusLostMsg

	IMECompose *input.IMEComposeMsg // pre-edit composition (RFC-002 §2.2)
	IMECommit  *input.IMECommitMsg  // final committed text (RFC-002 §2.2)

	// Gesture events (RFC-004 §3.3).
	Tap       *input.TapMsg
	LongPress *input.LongPressMsg
	Swipe     *input.SwipeMsg
	Drag      *input.DragMsg
	Pinch     *input.PinchMsg

	// Drag-and-drop events (RFC-005 §5).
	DnDEnter *DragEnterMsg
	DnDOver  *DragOverMsg
	DnDLeave *DragLeaveMsg
	DnDDrop  *DropMsg
}

// KeyEvent constructs an InputEvent from a KeyMsg.
func KeyEvent(msg input.KeyMsg) InputEvent {
	return InputEvent{Kind: EventKey, Key: &msg}
}

// TextInputEvent constructs an InputEvent from a TextInputMsg.
func TextInputEvent(msg input.TextInputMsg) InputEvent {
	return InputEvent{Kind: EventTextInput, TextInput: &msg}
}

// CharEvent constructs an InputEvent from a CharMsg.
func CharEvent(msg input.CharMsg) InputEvent {
	return InputEvent{Kind: EventChar, Char: &msg}
}

// MouseEvent constructs an InputEvent from a MouseMsg.
func MouseEvent(msg input.MouseMsg) InputEvent {
	return InputEvent{Kind: EventMouse, Mouse: &msg}
}

// ScrollEvent constructs an InputEvent from a ScrollMsg.
func ScrollEvent(msg input.ScrollMsg) InputEvent {
	return InputEvent{Kind: EventScroll, Scroll: &msg}
}

// TouchEvent constructs an InputEvent from a TouchMsg.
func TouchEvent(msg input.TouchMsg) InputEvent {
	return InputEvent{Kind: EventTouch, Touch: &msg}
}

// FocusGainedEvent constructs an InputEvent for focus gain.
func FocusGainedEvent(msg FocusGainedMsg) InputEvent {
	return InputEvent{Kind: EventFocusGained, FocusGained: &msg}
}

// FocusLostEvent constructs an InputEvent for focus loss.
func FocusLostEvent(msg FocusLostMsg) InputEvent {
	return InputEvent{Kind: EventFocusLost, FocusLost: &msg}
}

// IMEComposeEvent constructs an InputEvent from an IMEComposeMsg.
func IMEComposeEvent(msg input.IMEComposeMsg) InputEvent {
	return InputEvent{Kind: EventIMECompose, IMECompose: &msg}
}

// IMECommitEvent constructs an InputEvent from an IMECommitMsg.
func IMECommitEvent(msg input.IMECommitMsg) InputEvent {
	return InputEvent{Kind: EventIMECommit, IMECommit: &msg}
}

// TapEvent constructs an InputEvent from a TapMsg.
func TapEvent(msg input.TapMsg) InputEvent {
	return InputEvent{Kind: EventTap, Tap: &msg}
}

// LongPressEvent constructs an InputEvent from a LongPressMsg.
func LongPressEvent(msg input.LongPressMsg) InputEvent {
	return InputEvent{Kind: EventLongPress, LongPress: &msg}
}

// SwipeEvent constructs an InputEvent from a SwipeMsg.
func SwipeEvent(msg input.SwipeMsg) InputEvent {
	return InputEvent{Kind: EventSwipe, Swipe: &msg}
}

// DragEvent constructs an InputEvent from a DragMsg.
func DragEvent(msg input.DragMsg) InputEvent {
	return InputEvent{Kind: EventDrag, Drag: &msg}
}

// PinchEvent constructs an InputEvent from a PinchMsg.
func PinchEvent(msg input.PinchMsg) InputEvent {
	return InputEvent{Kind: EventPinch, Pinch: &msg}
}

// ── Drag-and-Drop Event Types (RFC-005 §5) ──────────────────────

// DragEnterMsg is delivered when a drag cursor enters a registered drop target.
type DragEnterMsg struct {
	Data      *input.DragData
	Pos       input.GesturePoint
	Modifiers input.ModifierSet
	Operation input.DragOperation // current operation based on modifiers
}

// DragOverMsg is delivered continuously while a drag cursor moves within a drop target.
type DragOverMsg struct {
	Data      *input.DragData
	Pos       input.GesturePoint
	Modifiers input.ModifierSet
	Operation input.DragOperation
}

// DragLeaveMsg is delivered when a drag cursor leaves a drop target.
type DragLeaveMsg struct {
	Data *input.DragData
}

// DropMsg is delivered when the user releases a drag over an accepting drop target.
type DropMsg struct {
	Data      *input.DragData
	Pos       input.GesturePoint
	Effect    input.DropEffect // resolved operation
	Modifiers input.ModifierSet
}

// DragEnterEvent constructs an InputEvent from a DragEnterMsg.
func DragEnterEvent(msg DragEnterMsg) InputEvent {
	return InputEvent{Kind: EventDragEnter, DnDEnter: &msg}
}

// DragOverEvent constructs an InputEvent from a DragOverMsg.
func DragOverEvent(msg DragOverMsg) InputEvent {
	return InputEvent{Kind: EventDragOver, DnDOver: &msg}
}

// DragLeaveEvent constructs an InputEvent from a DragLeaveMsg.
func DragLeaveEvent(msg DragLeaveMsg) InputEvent {
	return InputEvent{Kind: EventDragLeave, DnDLeave: &msg}
}

// DropEvent constructs an InputEvent from a DropMsg.
func DropEvent(msg DropMsg) InputEvent {
	return InputEvent{Kind: EventDrop, DnDDrop: &msg}
}
