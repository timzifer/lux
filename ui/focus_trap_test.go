package ui

import "testing"

func TestFocusTrapManager_PushPop(t *testing.T) {
	tm := NewFocusTrapManager()

	if tm.Active() {
		t.Error("expected no active trap initially")
	}
	if tm.ActiveTrapID() != "" {
		t.Error("expected empty active trap ID")
	}

	trap := FocusTrap{RestoreFocus: true, TrapID: "dialog-1"}
	focusable := []UID{10, 20, 30}
	initialFocus := tm.PushTrap(trap, 5, focusable)

	if !tm.Active() {
		t.Error("expected trap to be active")
	}
	if tm.ActiveTrapID() != "dialog-1" {
		t.Errorf("expected active trap 'dialog-1', got %q", tm.ActiveTrapID())
	}
	// InitialFocus is 0, so first focusable widget should be returned.
	if initialFocus != 10 {
		t.Errorf("expected initial focus 10, got %d", initialFocus)
	}

	// Pop the trap — should return saved focus.
	restored := tm.PopTrap("dialog-1")
	if restored != 5 {
		t.Errorf("expected restored focus 5, got %d", restored)
	}
	if tm.Active() {
		t.Error("expected no active trap after pop")
	}
}

func TestFocusTrapManager_InitialFocus(t *testing.T) {
	tm := NewFocusTrapManager()

	trap := FocusTrap{RestoreFocus: true, TrapID: "d1", InitialFocus: 20}
	initial := tm.PushTrap(trap, 0, []UID{10, 20, 30})
	if initial != 20 {
		t.Errorf("expected initial focus 20, got %d", initial)
	}
	tm.PopTrap("d1")
}

func TestFocusTrapManager_InitialFocusNotInOrder(t *testing.T) {
	tm := NewFocusTrapManager()

	// InitialFocus UID 99 is not in the focusable list — falls back to first.
	trap := FocusTrap{RestoreFocus: true, TrapID: "d1", InitialFocus: 99}
	initial := tm.PushTrap(trap, 0, []UID{10, 20})
	if initial != 10 {
		t.Errorf("expected fallback to first (10), got %d", initial)
	}
	tm.PopTrap("d1")
}

func TestFocusTrapManager_NestedTraps(t *testing.T) {
	tm := NewFocusTrapManager()

	tm.PushTrap(FocusTrap{RestoreFocus: true, TrapID: "outer"}, 1, []UID{10, 20})
	tm.PushTrap(FocusTrap{RestoreFocus: true, TrapID: "inner"}, 10, []UID{30, 40})

	if tm.ActiveTrapID() != "inner" {
		t.Errorf("expected active trap 'inner', got %q", tm.ActiveTrapID())
	}

	// IsInActiveTrap checks topmost trap only.
	if tm.IsInActiveTrap(10) {
		t.Error("UID 10 should not be in inner trap")
	}
	if !tm.IsInActiveTrap(30) {
		t.Error("UID 30 should be in inner trap")
	}

	// Pop inner, should restore to 10.
	restored := tm.PopTrap("inner")
	if restored != 10 {
		t.Errorf("expected restored focus 10, got %d", restored)
	}
	if tm.ActiveTrapID() != "outer" {
		t.Errorf("expected active trap 'outer', got %q", tm.ActiveTrapID())
	}

	// Pop outer, should restore to 1.
	restored = tm.PopTrap("outer")
	if restored != 1 {
		t.Errorf("expected restored focus 1, got %d", restored)
	}
	if tm.Active() {
		t.Error("expected no active trap")
	}
}

func TestFocusTrapManager_ConstrainAdvance(t *testing.T) {
	tm := NewFocusTrapManager()
	tm.PushTrap(FocusTrap{TrapID: "d1"}, 0, []UID{10, 20, 30})

	// Forward from 10 → 20.
	next := tm.ConstrainAdvance(10, 1)
	if next != 20 {
		t.Errorf("expected 20, got %d", next)
	}

	// Forward from 30 → wraps to 10.
	next = tm.ConstrainAdvance(30, 1)
	if next != 10 {
		t.Errorf("expected wrap to 10, got %d", next)
	}

	// Backward from 10 → wraps to 30.
	next = tm.ConstrainAdvance(10, -1)
	if next != 30 {
		t.Errorf("expected wrap to 30, got %d", next)
	}

	// Backward from 20 → 10.
	next = tm.ConstrainAdvance(20, -1)
	if next != 10 {
		t.Errorf("expected 10, got %d", next)
	}

	tm.PopTrap("d1")
}

func TestFocusTrapManager_ConstrainAdvanceUnknownFocus(t *testing.T) {
	tm := NewFocusTrapManager()
	tm.PushTrap(FocusTrap{TrapID: "d1"}, 0, []UID{10, 20})

	// Current focus not in trap → forward starts at first.
	next := tm.ConstrainAdvance(99, 1)
	if next != 10 {
		t.Errorf("expected 10, got %d", next)
	}

	// Current focus not in trap → backward starts at last.
	next = tm.ConstrainAdvance(99, -1)
	if next != 20 {
		t.Errorf("expected 20, got %d", next)
	}

	tm.PopTrap("d1")
}

func TestFocusTrapManager_NoRestoreFocus(t *testing.T) {
	tm := NewFocusTrapManager()
	tm.PushTrap(FocusTrap{RestoreFocus: false, TrapID: "d1"}, 5, []UID{10})

	restored := tm.PopTrap("d1")
	if restored != 0 {
		t.Errorf("expected 0 (no restore), got %d", restored)
	}
}

func TestFocusTrapManager_UpdateTrapOrder(t *testing.T) {
	tm := NewFocusTrapManager()
	tm.PushTrap(FocusTrap{TrapID: "d1"}, 0, []UID{10, 20})

	// Update to include a new widget.
	tm.UpdateTrapOrder("d1", []UID{10, 20, 30})

	next := tm.ConstrainAdvance(20, 1)
	if next != 30 {
		t.Errorf("expected 30 after order update, got %d", next)
	}
	tm.PopTrap("d1")
}

func TestFocusTrapManager_NilSafe(t *testing.T) {
	var tm *FocusTrapManager

	// All methods should be nil-safe.
	if tm.Active() {
		t.Error("nil manager should not be active")
	}
	if tm.ActiveTrapID() != "" {
		t.Error("nil manager should return empty trap ID")
	}
	if tm.IsInActiveTrap(1) {
		t.Error("nil manager should return false for IsInActiveTrap")
	}
	if tm.ConstrainAdvance(1, 1) != 0 {
		t.Error("nil manager should return 0 for ConstrainAdvance")
	}
	if tm.PushTrap(FocusTrap{}, 0, nil) != 0 {
		t.Error("nil manager should return 0 for PushTrap")
	}
	if tm.PopTrap("x") != 0 {
		t.Error("nil manager should return 0 for PopTrap")
	}
}

func TestFocusManager_AdvanceWithTrap(t *testing.T) {
	fm := NewFocusManager()
	fm.Trap = NewFocusTrapManager()

	// Register some widgets in global focus order.
	fm.RegisterFocusable(1, FocusOpts{Focusable: true})
	fm.RegisterFocusable(2, FocusOpts{Focusable: true})
	fm.RegisterFocusable(3, FocusOpts{Focusable: true})
	fm.SortOrder()
	fm.SetFocusedUID(1)

	// Without trap, advance goes 1 → 2.
	next := fm.FocusNext()
	if next != 2 {
		t.Errorf("expected 2 without trap, got %d", next)
	}

	// Push trap constraining to UIDs 2 and 3.
	fm.SetFocusedUID(2)
	fm.Trap.PushTrap(FocusTrap{TrapID: "modal"}, fm.FocusedUID(), []UID{2, 3})

	// Advance within trap: 2 → 3.
	next = fm.FocusNext()
	if next != 3 {
		t.Errorf("expected 3 within trap, got %d", next)
	}

	// Wrap: 3 → 2 (stays in trap).
	next = fm.FocusNext()
	if next != 2 {
		t.Errorf("expected wrap to 2, got %d", next)
	}

	// Backward: 2 → 3 (wrap backward).
	next = fm.FocusPrev()
	if next != 3 {
		t.Errorf("expected backward wrap to 3, got %d", next)
	}

	// Pop trap.
	restored := fm.Trap.PopTrap("modal")
	if restored != 2 {
		t.Errorf("expected restored focus 2, got %d", restored)
	}
}
