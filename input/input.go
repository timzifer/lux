// Package input defines message types for user input events.
// It depends only on stdlib (per RFC §21.1).
package input

// KeyMsg is sent when a key is pressed or released.
type KeyMsg struct {
	Key       string
	Modifiers KeyModifiers
	Action    KeyAction
}

// KeyModifiers represents modifier key state.
type KeyModifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
	Super bool
}

// KeyAction indicates whether a key was pressed, released, or repeated.
type KeyAction int

const (
	KeyPress KeyAction = iota
	KeyRelease
	KeyRepeat
)

// MouseMsg is sent on mouse events.
type MouseMsg struct {
	X, Y   float32
	Button MouseButton
	Action MouseAction
}

// MouseButton identifies a mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
)

// MouseAction indicates the type of mouse event.
type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
	MouseScroll
)

// ScrollMsg is sent on scroll events.
type ScrollMsg struct {
	DeltaX, DeltaY float32
}

// ResizeMsg is sent when the window is resized.
type ResizeMsg struct {
	Width, Height int
}

// CloseMsg is sent when the user requests window close.
type CloseMsg struct{}
