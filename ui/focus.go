package ui

// ── Focus Messages (RFC-002 §2.3) ────────────────────────────────

// FocusSource describes how focus was gained or lost.
type FocusSource int

const (
	FocusSourceTab      FocusSource = iota // Tab / Shift+Tab navigation
	FocusSourceClick                       // Mouse click
	FocusSourceProgram                     // RequestFocusMsg / ReleaseFocusMsg
)

// FocusGainedMsg is delivered to a widget when it receives keyboard focus.
type FocusGainedMsg struct {
	Source FocusSource
}

// FocusLostMsg is delivered to a widget when it loses keyboard focus.
type FocusLostMsg struct {
	Source FocusSource
}

// RequestFocusMsg asks the framework to move focus to a specific widget.
type RequestFocusMsg struct {
	Target UID
}

// ReleaseFocusMsg asks the framework to blur the currently focused widget.
type ReleaseFocusMsg struct{}

// ── Focusable Interface (RFC-002 §2.3) ───────────────────────────

// FocusOpts configures how a widget participates in focus management.
type FocusOpts struct {
	Focusable    bool // Whether this widget can receive focus
	TabIndex     int  // Tab order (0 = natural layout order, negative = skip)
	FocusOnClick bool // Automatically focus when clicked
}

// Focusable is an optional interface for Widgets that can receive
// keyboard focus (RFC-002 §2.3). The framework calls FocusOptions()
// during layout to build the tab-order.
type Focusable interface {
	FocusOptions() FocusOpts
}

// ── Focus Manager ────────────────────────────────────────────────

// FocusManager tracks keyboard focus across widgets. It replaces the
// simple FocusState for the new architecture (RFC-002 §2.3).
type FocusManager struct {
	focusedUID UID
	focusOrder []focusEntry // sorted by tab order then layout position
}

type focusEntry struct {
	uid      UID
	tabIndex int
}

// FocusedUID returns the UID of the currently focused widget, or 0 if none.
func (fm *FocusManager) FocusedUID() UID { return fm.focusedUID }

// SetFocusedUID sets focus to the given UID.
func (fm *FocusManager) SetFocusedUID(uid UID) { fm.focusedUID = uid }

// Blur removes focus from any widget.
func (fm *FocusManager) Blur() { fm.focusedUID = 0 }

// IsFocused reports whether the given UID currently has focus.
func (fm *FocusManager) IsFocused(uid UID) bool { return fm.focusedUID == uid }

// RegisterFocusable adds a widget to the focus order. Called during layout.
func (fm *FocusManager) RegisterFocusable(uid UID, opts FocusOpts) {
	if !opts.Focusable {
		return
	}
	fm.focusOrder = append(fm.focusOrder, focusEntry{uid: uid, tabIndex: opts.TabIndex})
}

// ResetOrder clears the focus order for rebuilding on the next frame.
func (fm *FocusManager) ResetOrder() {
	fm.focusOrder = fm.focusOrder[:0]
}

// FocusNext moves focus to the next focusable widget in tab order.
// Returns the UID that received focus, or 0 if no focusable widgets exist.
func (fm *FocusManager) FocusNext() UID {
	return fm.advance(1)
}

// FocusPrev moves focus to the previous focusable widget in tab order.
func (fm *FocusManager) FocusPrev() UID {
	return fm.advance(-1)
}

func (fm *FocusManager) advance(dir int) UID {
	n := len(fm.focusOrder)
	if n == 0 {
		return 0
	}

	// Find current index.
	currentIdx := -1
	for i, e := range fm.focusOrder {
		if e.uid == fm.focusedUID {
			currentIdx = i
			break
		}
	}

	var nextIdx int
	if currentIdx < 0 {
		// Nothing focused — start at beginning (forward) or end (backward).
		if dir > 0 {
			nextIdx = 0
		} else {
			nextIdx = n - 1
		}
	} else {
		nextIdx = (currentIdx + dir + n) % n
	}

	fm.focusedUID = fm.focusOrder[nextIdx].uid
	return fm.focusedUID
}
