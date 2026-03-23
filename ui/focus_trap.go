package ui

// FocusTrap constrains Tab navigation within a subtree (RFC-001 §11.7).
// Activated automatically when a modal dialog opens. Tab/Shift+Tab cycle
// through focusable widgets inside the trap; focus cannot escape.
type FocusTrap struct {
	// RestoreFocus controls whether focus returns to the previously focused
	// widget when the trap is released. Default: true.
	RestoreFocus bool

	// InitialFocus is the UID of the widget that receives focus when the
	// trap activates. Zero means the first focusable widget in the trap.
	InitialFocus UID

	// TrapID identifies the trap owner (typically an OverlayID).
	TrapID string
}

// activeTrap holds runtime state for a pushed focus trap.
type activeTrap struct {
	trap       FocusTrap
	savedFocus UID   // UID that had focus before the trap activated
	order      []UID // focusable UIDs within the trap, in tab order
}

// FocusTrapManager maintains a stack of active focus traps.
// Nested traps are supported (dialog inside dialog).
type FocusTrapManager struct {
	stack []activeTrap
}

// NewFocusTrapManager creates a ready-to-use FocusTrapManager.
func NewFocusTrapManager() *FocusTrapManager {
	return &FocusTrapManager{}
}

// PushTrap activates a new focus trap. The current focus is saved for
// later restoration. focusableUIDs lists the widgets within the trap
// in their tab order.
func (tm *FocusTrapManager) PushTrap(trap FocusTrap, currentFocus UID, focusableUIDs []UID) UID {
	if tm == nil {
		return 0
	}
	at := activeTrap{
		trap:       trap,
		savedFocus: currentFocus,
		order:      make([]UID, len(focusableUIDs)),
	}
	copy(at.order, focusableUIDs)
	tm.stack = append(tm.stack, at)

	// Determine initial focus target.
	if trap.InitialFocus != 0 {
		for _, uid := range at.order {
			if uid == trap.InitialFocus {
				return trap.InitialFocus
			}
		}
	}
	// Default: first focusable widget in trap.
	if len(at.order) > 0 {
		return at.order[0]
	}
	return 0
}

// PopTrap releases the trap identified by trapID and returns the UID
// to restore focus to (if RestoreFocus was true), or 0.
func (tm *FocusTrapManager) PopTrap(trapID string) UID {
	if tm == nil {
		return 0
	}
	for i := len(tm.stack) - 1; i >= 0; i-- {
		if tm.stack[i].trap.TrapID == trapID {
			at := tm.stack[i]
			tm.stack = append(tm.stack[:i], tm.stack[i+1:]...)
			if at.trap.RestoreFocus {
				return at.savedFocus
			}
			return 0
		}
	}
	return 0
}

// Active reports whether any focus trap is currently active.
func (tm *FocusTrapManager) Active() bool {
	return tm != nil && len(tm.stack) > 0
}

// ActiveTrapID returns the TrapID of the topmost active trap, or "".
func (tm *FocusTrapManager) ActiveTrapID() string {
	if tm == nil || len(tm.stack) == 0 {
		return ""
	}
	return tm.stack[len(tm.stack)-1].trap.TrapID
}

// IsInActiveTrap reports whether the given UID belongs to the topmost
// active trap's focusable set.
func (tm *FocusTrapManager) IsInActiveTrap(uid UID) bool {
	if tm == nil || len(tm.stack) == 0 {
		return false
	}
	top := tm.stack[len(tm.stack)-1]
	for _, u := range top.order {
		if u == uid {
			return true
		}
	}
	return false
}

// ConstrainAdvance moves focus within the topmost trap. dir is +1 for
// Tab (forward) or -1 for Shift+Tab (backward). Returns the UID that
// should receive focus. currentFocus is the currently focused UID.
func (tm *FocusTrapManager) ConstrainAdvance(currentFocus UID, dir int) UID {
	if tm == nil || len(tm.stack) == 0 {
		return 0
	}
	top := tm.stack[len(tm.stack)-1]
	n := len(top.order)
	if n == 0 {
		return 0
	}

	// Find current index within trap order.
	currentIdx := -1
	for i, uid := range top.order {
		if uid == currentFocus {
			currentIdx = i
			break
		}
	}

	var nextIdx int
	if currentIdx < 0 {
		// Focus not in trap — start at beginning (forward) or end (backward).
		if dir > 0 {
			nextIdx = 0
		} else {
			nextIdx = n - 1
		}
	} else {
		nextIdx = (currentIdx + dir + n) % n
	}

	return top.order[nextIdx]
}

// UpdateTrapOrder replaces the focusable UIDs for the topmost trap.
// Called after layout when the trap's contents may have changed.
func (tm *FocusTrapManager) UpdateTrapOrder(trapID string, focusableUIDs []UID) {
	if tm == nil {
		return
	}
	for i := len(tm.stack) - 1; i >= 0; i-- {
		if tm.stack[i].trap.TrapID == trapID {
			tm.stack[i].order = make([]UID, len(focusableUIDs))
			copy(tm.stack[i].order, focusableUIDs)
			return
		}
	}
}
