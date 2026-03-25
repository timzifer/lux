package ui

import "sort"

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

// FocusManager tracks keyboard focus across widgets and built-in
// focusable elements (e.g. TextField). It replaces the old FocusState
// for the new architecture (RFC-002 §2.3).
type FocusManager struct {
	focusedUID UID
	focusOrder []focusEntry // populated during layout, sorted by SortOrder

	// Element-level focus support (TextField etc.).
	nextElemID int         // counter for assigning element UIDs during layout
	Input      *InputState // active TextField input state (if focused is a TextField)

	// PendingCursorOffset stores a cursor byte offset from a click that
	// occurred before the InputState was created (first click to focus).
	// -1 means no pending offset. Consumed by the layout pass.
	PendingCursorOffset int

	// FocusTrap support (RFC-001 §11.7).
	Trap *FocusTrapManager
}

type focusEntry struct {
	uid        UID
	tabIndex   int
	layoutOrder int // registration order during layout (natural order)
}

// NewFocusManager creates a ready-to-use FocusManager.
func NewFocusManager() *FocusManager {
	return &FocusManager{PendingCursorOffset: -1}
}

// FocusedUID returns the UID of the currently focused widget, or 0 if none.
func (fm *FocusManager) FocusedUID() UID {
	if fm == nil {
		return 0
	}
	return fm.focusedUID
}

// SetFocusedUID sets focus to the given UID.
func (fm *FocusManager) SetFocusedUID(uid UID) {
	if fm != nil {
		fm.focusedUID = uid
	}
}

// Blur removes focus from any widget.
func (fm *FocusManager) Blur() {
	if fm != nil {
		fm.focusedUID = 0
	}
}

// IsFocused reports whether the given UID currently has focus.
func (fm *FocusManager) IsFocused(uid UID) bool {
	return fm != nil && fm.focusedUID == uid
}

// RegisterFocusable adds a widget to the focus order. Called during layout.
// The registration order defines the natural tab order (layout-tree order).
func (fm *FocusManager) RegisterFocusable(uid UID, opts FocusOpts) {
	if fm == nil || !opts.Focusable || opts.TabIndex < 0 {
		return
	}
	fm.focusOrder = append(fm.focusOrder, focusEntry{
		uid:         uid,
		tabIndex:    opts.TabIndex,
		layoutOrder: len(fm.focusOrder),
	})
}

// ResetOrder clears the focus order for rebuilding on the next layout pass.
// Input is deliberately preserved: click handlers and key handlers modify
// CursorOffset/SelectionStart between frames, and clearing Input here
// would discard those changes before layout can read them. The layout
// pass overwrites Input when a focused text field is present; if no
// text field is focused, Input naturally becomes stale and is ignored.
func (fm *FocusManager) ResetOrder() {
	if fm == nil {
		return
	}
	fm.focusOrder = fm.focusOrder[:0]
	fm.nextElemID = 0
}

// SortOrder sorts the focus order: elements with positive TabIndex come
// first (sorted ascending by TabIndex), then elements with TabIndex == 0
// in their natural layout order. This derives the tab order from the
// layout tree (RFC-002 §2.3).
func (fm *FocusManager) SortOrder() {
	if fm == nil {
		return
	}
	sort.SliceStable(fm.focusOrder, func(i, j int) bool {
		a, b := fm.focusOrder[i], fm.focusOrder[j]
		// Positive TabIndex comes before zero.
		aPos := a.tabIndex > 0
		bPos := b.tabIndex > 0
		if aPos != bPos {
			return aPos // positive before zero
		}
		if aPos && bPos {
			return a.tabIndex < b.tabIndex
		}
		// Both zero: natural layout order.
		return a.layoutOrder < b.layoutOrder
	})
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

// OrderLen returns the number of focusable entries (test helper).
func (fm *FocusManager) OrderLen() int {
	if fm == nil {
		return 0
	}
	return len(fm.focusOrder)
}

func (fm *FocusManager) advance(dir int) UID {
	// If a focus trap is active, constrain navigation within the trap.
	if fm.Trap != nil && fm.Trap.Active() {
		next := fm.Trap.ConstrainAdvance(fm.focusedUID, dir)
		if next != 0 {
			fm.focusedUID = next
		}
		return fm.focusedUID
	}

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

// ── Element-level focus helpers ─────────────────────────────────

// elementFocusUIDBit marks UIDs as element-level (high bit set) to
// distinguish from widget-level UIDs produced by MakeUID.
const elementFocusUIDBit = UID(1) << 63

// NextElementUID assigns and returns a stable UID for a built-in
// focusable element (TextField). The UID is deterministic within a
// layout pass as long as the element order doesn't change.
func (fm *FocusManager) NextElementUID() UID {
	if fm == nil {
		return 0
	}
	fm.nextElemID++
	return elementFocusUIDBit | UID(fm.nextElemID)
}

// IsElementFocused reports whether the element with the given UID has focus.
func (fm *FocusManager) IsElementFocused(uid UID) bool {
	return fm != nil && fm.focusedUID == uid
}
