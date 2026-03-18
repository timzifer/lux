// Package input defines message types for user input events.
// It depends only on stdlib (per RFC §2.1).
package input

// ── Key Type ──────────────────────────────────────────────────────

// Key is a typed identifier for physical/logical keys (RFC-002 §2.2).
type Key uint32

//nolint:revive // Key constants use Key prefix for clarity.
const (
	KeyUnknown Key = 0

	// Printable keys (matching USB HID page 0x07 where practical).
	KeyA Key = iota + 4
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ

	Key0 Key = iota + 19
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9

	// Function keys.
	KeyF1 Key = iota + 49
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12

	// Navigation & editing.
	KeyEscape Key = iota + 69
	KeyEnter
	KeyTab
	KeyBackspace
	KeyInsert
	KeyDelete
	KeyRight
	KeyLeft
	KeyDown
	KeyUp
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd

	// Symbols.
	KeySpace Key = iota + 83
	KeyMinus
	KeyEqual
	KeyLeftBracket
	KeyRightBracket
	KeyBackslash
	KeySemicolon
	KeyApostrophe
	KeyGraveAccent
	KeyComma
	KeyPeriod
	KeySlash

	// Modifier keys.
	KeyLeftShift Key = iota + 95
	KeyLeftCtrl
	KeyLeftAlt
	KeyLeftSuper
	KeyRightShift
	KeyRightCtrl
	KeyRightAlt
	KeyRightSuper

	// Misc.
	KeyCapsLock Key = iota + 103
	KeyPrintScreen
	KeyScrollLock
	KeyPause
	KeyNumLock
	KeyMenu
)

// ── Modifiers ─────────────────────────────────────────────────────

// ModifierSet is a bitfield of active modifier keys (RFC-002 §2.2).
type ModifierSet uint8

const (
	ModShift ModifierSet = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
)

// Has reports whether all bits in m are set.
func (ms ModifierSet) Has(m ModifierSet) bool { return ms&m == m }

// ── KeyMsg ────────────────────────────────────────────────────────

// KeyAction indicates whether a key was pressed, released, or repeated.
type KeyAction int

const (
	KeyPress KeyAction = iota
	KeyRelease
	KeyRepeat
)

// KeyMsg is sent when a key is pressed or released (RFC-002 §2.2).
type KeyMsg struct {
	Key       Key
	Rune      rune // Unicode codepoint if printable, 0 otherwise
	Action    KeyAction
	Modifiers ModifierSet
}

// ── TextInputMsg ──────────────────────────────────────────────────

// TextInputMsg carries post-IME composed text (RFC-002 §2.2).
// Unlike CharMsg it may contain multi-codepoint strings from
// dead-key or IME composition.
type TextInputMsg struct {
	Text string
}

// ── CharMsg ───────────────────────────────────────────────────────

// CharMsg is sent when a single Unicode character is input (text entry).
type CharMsg struct {
	Char rune
}

// ── MouseMsg ──────────────────────────────────────────────────────

// MouseButton identifies a mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
)

// MouseAction indicates the type of mouse event (RFC-002 §2.2).
type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
	MouseScroll
	MouseEnter // Cursor entered widget/window bounds
	MouseLeave // Cursor left widget/window bounds
)

// MouseMsg is sent on mouse events (RFC-002 §2.2).
type MouseMsg struct {
	X, Y      float32
	Button    MouseButton
	Action    MouseAction
	Modifiers ModifierSet
}

// ── ScrollMsg ─────────────────────────────────────────────────────

// ScrollMsg is sent on scroll events (RFC-002 §2.2).
type ScrollMsg struct {
	X, Y           float32 // Cursor position at scroll time
	DeltaX, DeltaY float32 // Scroll deltas
	Precise        bool    // true for trackpad (high-resolution), false for mouse wheel
	Modifiers      ModifierSet
}

// ── TouchMsg ──────────────────────────────────────────────────────

// TouchPhase describes the lifecycle phase of a touch (RFC-002 §2.2).
type TouchPhase int

const (
	TouchBegan TouchPhase = iota
	TouchMoved
	TouchEnded
	TouchCancelled
)

// TouchMsg carries a single touch event (RFC-002 §2.2).
type TouchMsg struct {
	ID    int64   // Stable identifier for the touch across phases
	X, Y  float32 // Position in dp
	Phase TouchPhase
	Force float32 // Normalised pressure [0,1]; 0 if unavailable
}

// ── ResizeMsg & CloseMsg ──────────────────────────────────────────

// ResizeMsg is sent when the window is resized.
type ResizeMsg struct {
	Width, Height int
}

// CloseMsg is sent when the user requests window close.
type CloseMsg struct{}

// ── Legacy Compatibility ──────────────────────────────────────────

// KeyModifiers is the legacy modifier type. Prefer ModifierSet.
// Retained so existing platform code compiles during migration.
type KeyModifiers = ModifierSet

// KeyNameToKey maps GLFW-style key name strings to typed Key values.
// Used by platform backends that still report keys as strings.
var KeyNameToKey = map[string]Key{
	"A": KeyA, "B": KeyB, "C": KeyC, "D": KeyD, "E": KeyE,
	"F": KeyF, "G": KeyG, "H": KeyH, "I": KeyI, "J": KeyJ,
	"K": KeyK, "L": KeyL, "M": KeyM, "N": KeyN, "O": KeyO,
	"P": KeyP, "Q": KeyQ, "R": KeyR, "S": KeyS, "T": KeyT,
	"U": KeyU, "V": KeyV, "W": KeyW, "X": KeyX, "Y": KeyY,
	"Z": KeyZ,
	"0": Key0, "1": Key1, "2": Key2, "3": Key3, "4": Key4,
	"5": Key5, "6": Key6, "7": Key7, "8": Key8, "9": Key9,
	"F1": KeyF1, "F2": KeyF2, "F3": KeyF3, "F4": KeyF4,
	"F5": KeyF5, "F6": KeyF6, "F7": KeyF7, "F8": KeyF8,
	"F9": KeyF9, "F10": KeyF10, "F11": KeyF11, "F12": KeyF12,
	"Escape": KeyEscape, "Enter": KeyEnter, "Tab": KeyTab,
	"Backspace": KeyBackspace, "Insert": KeyInsert, "Delete": KeyDelete,
	"Right": KeyRight, "Left": KeyLeft, "Down": KeyDown, "Up": KeyUp,
	"PageUp": KeyPageUp, "PageDown": KeyPageDown,
	"Home": KeyHome, "End": KeyEnd,
	"Space": KeySpace, "Minus": KeyMinus, "Equal": KeyEqual,
	"LeftBracket": KeyLeftBracket, "RightBracket": KeyRightBracket,
	"Backslash": KeyBackslash, "Semicolon": KeySemicolon,
	"Apostrophe": KeyApostrophe, "GraveAccent": KeyGraveAccent,
	"Comma": KeyComma, "Period": KeyPeriod, "Slash": KeySlash,
	"LeftShift": KeyLeftShift, "LeftCtrl": KeyLeftCtrl,
	"LeftAlt": KeyLeftAlt, "LeftSuper": KeyLeftSuper,
	"RightShift": KeyRightShift, "RightCtrl": KeyRightCtrl,
	"RightAlt": KeyRightAlt, "RightSuper": KeyRightSuper,
	"CapsLock": KeyCapsLock, "PrintScreen": KeyPrintScreen,
	"ScrollLock": KeyScrollLock, "Pause": KeyPause,
	"NumLock": KeyNumLock, "Menu": KeyMenu,
}

// ModsFromBits converts GLFW-style modifier bits to ModifierSet.
// Bit 0=Shift, 1=Ctrl, 2=Alt, 3=Super.
func ModsFromBits(bits int) ModifierSet {
	var ms ModifierSet
	if bits&1 != 0 {
		ms |= ModShift
	}
	if bits&2 != 0 {
		ms |= ModCtrl
	}
	if bits&4 != 0 {
		ms |= ModAlt
	}
	if bits&8 != 0 {
		ms |= ModSuper
	}
	return ms
}
