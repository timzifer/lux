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
