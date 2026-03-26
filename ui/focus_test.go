package ui

import "testing"

func TestFocusManagerInitialState(t *testing.T) {
	var fm FocusManager
	if fm.FocusedUID() != 0 {
		t.Error("initial FocusedUID should be 0")
	}
	if fm.IsFocused(42) {
		t.Error("no widget should be focused initially")
	}
}

func TestFocusManagerSetAndBlur(t *testing.T) {
	var fm FocusManager
	fm.SetFocusedUID(42)
	if !fm.IsFocused(42) {
		t.Error("42 should be focused")
	}
	if fm.IsFocused(99) {
		t.Error("99 should not be focused")
	}

	fm.Blur()
	if fm.FocusedUID() != 0 {
		t.Error("FocusedUID should be 0 after Blur")
	}
}

func TestFocusManagerTabNavigation(t *testing.T) {
	var fm FocusManager
	fm.RegisterFocusable(10, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(20, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(30, FocusOpts{Focusable: true, TabIndex: 0})

	// First FocusNext should focus first widget.
	uid := fm.FocusNext()
	if uid != 10 {
		t.Errorf("first FocusNext = %d, want 10", uid)
	}

	// Second FocusNext should advance.
	uid = fm.FocusNext()
	if uid != 20 {
		t.Errorf("second FocusNext = %d, want 20", uid)
	}

	uid = fm.FocusNext()
	if uid != 30 {
		t.Errorf("third FocusNext = %d, want 30", uid)
	}

	// Wrap around.
	uid = fm.FocusNext()
	if uid != 10 {
		t.Errorf("wrapped FocusNext = %d, want 10", uid)
	}
}

func TestFocusManagerShiftTab(t *testing.T) {
	var fm FocusManager
	fm.RegisterFocusable(10, FocusOpts{Focusable: true})
	fm.RegisterFocusable(20, FocusOpts{Focusable: true})
	fm.RegisterFocusable(30, FocusOpts{Focusable: true})

	// FocusPrev from unfocused → last widget.
	uid := fm.FocusPrev()
	if uid != 30 {
		t.Errorf("first FocusPrev = %d, want 30", uid)
	}

	uid = fm.FocusPrev()
	if uid != 20 {
		t.Errorf("second FocusPrev = %d, want 20", uid)
	}
}

func TestFocusManagerResetOrder(t *testing.T) {
	var fm FocusManager
	fm.RegisterFocusable(10, FocusOpts{Focusable: true})
	fm.RegisterFocusable(20, FocusOpts{Focusable: true})
	fm.SetFocusedUID(10)

	fm.ResetOrder()

	// Focus should be preserved, but order is empty.
	if fm.FocusedUID() != 10 {
		t.Error("focus should be preserved after ResetOrder")
	}

	// After reset, FocusNext with empty order returns 0.
	uid := fm.FocusNext()
	if uid != 0 {
		t.Errorf("FocusNext with empty order = %d, want 0", uid)
	}
}

func TestFocusManagerSkipsNonFocusable(t *testing.T) {
	var fm FocusManager
	fm.RegisterFocusable(10, FocusOpts{Focusable: false}) // should be skipped
	fm.RegisterFocusable(20, FocusOpts{Focusable: true})

	uid := fm.FocusNext()
	if uid != 20 {
		t.Errorf("FocusNext = %d, want 20 (10 is not focusable)", uid)
	}
}

func TestFocusManagerEmptyOrder(t *testing.T) {
	var fm FocusManager
	uid := fm.FocusNext()
	if uid != 0 {
		t.Errorf("FocusNext on empty = %d, want 0", uid)
	}
	uid = fm.FocusPrev()
	if uid != 0 {
		t.Errorf("FocusPrev on empty = %d, want 0", uid)
	}
}

func TestFocusOptsDefaults(t *testing.T) {
	opts := FocusOpts{}
	if opts.Focusable {
		t.Error("default Focusable should be false")
	}
	if opts.TabIndex != 0 {
		t.Error("default TabIndex should be 0")
	}
	if opts.FocusOnClick {
		t.Error("default FocusOnClick should be false")
	}
}

func TestFocusSourceValues(t *testing.T) {
	if FocusSourceTab != 0 {
		t.Errorf("FocusSourceTab = %d, want 0", FocusSourceTab)
	}
	if FocusSourceClick != 1 {
		t.Errorf("FocusSourceClick = %d, want 1", FocusSourceClick)
	}
	if FocusSourceProgram != 2 {
		t.Errorf("FocusSourceProgram = %d, want 2", FocusSourceProgram)
	}
}

func TestRequestFocusMsgTarget(t *testing.T) {
	msg := RequestFocusMsg{Target: 42}
	if msg.Target != 42 {
		t.Errorf("Target = %d, want 42", msg.Target)
	}
}

func TestFocusGainedLostMsgs(t *testing.T) {
	gained := FocusGainedMsg{Source: FocusSourceClick}
	if gained.Source != FocusSourceClick {
		t.Errorf("FocusGainedMsg.Source = %d, want FocusSourceClick", gained.Source)
	}

	lost := FocusLostMsg{Source: FocusSourceProgram}
	if lost.Source != FocusSourceProgram {
		t.Errorf("FocusLostMsg.Source = %d, want FocusSourceProgram", lost.Source)
	}
}

// ── Tab-order sorting (RFC-002 §2.3) ─────────────────────────────

func TestSortOrderNaturalLayoutOrder(t *testing.T) {
	fm := NewFocusManager()
	fm.RegisterFocusable(30, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(10, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(20, FocusOpts{Focusable: true, TabIndex: 0})
	fm.SortOrder()

	// All TabIndex=0 → natural layout order (registration order).
	uid := fm.FocusNext()
	if uid != 30 {
		t.Errorf("first = %d, want 30 (registered first)", uid)
	}
	uid = fm.FocusNext()
	if uid != 10 {
		t.Errorf("second = %d, want 10", uid)
	}
	uid = fm.FocusNext()
	if uid != 20 {
		t.Errorf("third = %d, want 20", uid)
	}
}

func TestSortOrderPositiveTabIndexFirst(t *testing.T) {
	fm := NewFocusManager()
	fm.RegisterFocusable(100, FocusOpts{Focusable: true, TabIndex: 0}) // natural
	fm.RegisterFocusable(200, FocusOpts{Focusable: true, TabIndex: 2}) // explicit 2
	fm.RegisterFocusable(300, FocusOpts{Focusable: true, TabIndex: 1}) // explicit 1
	fm.SortOrder()

	// Positive TabIndex first (sorted ascending), then TabIndex=0.
	uid := fm.FocusNext()
	if uid != 300 {
		t.Errorf("first = %d, want 300 (TabIndex=1)", uid)
	}
	uid = fm.FocusNext()
	if uid != 200 {
		t.Errorf("second = %d, want 200 (TabIndex=2)", uid)
	}
	uid = fm.FocusNext()
	if uid != 100 {
		t.Errorf("third = %d, want 100 (TabIndex=0, natural)", uid)
	}
}

func TestSortOrderSkipsNegativeTabIndex(t *testing.T) {
	fm := NewFocusManager()
	fm.RegisterFocusable(10, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(20, FocusOpts{Focusable: true, TabIndex: -1}) // should be excluded
	fm.RegisterFocusable(30, FocusOpts{Focusable: true, TabIndex: 0})

	if fm.OrderLen() != 2 {
		t.Errorf("OrderLen = %d, want 2 (negative TabIndex excluded)", fm.OrderLen())
	}
}

// ── Element-level focus UIDs ──────────────────────────────────────

func TestNextElementUIDDeterministic(t *testing.T) {
	fm := NewFocusManager()
	uid1 := fm.NextElementUID()
	uid2 := fm.NextElementUID()

	if uid1 == 0 || uid2 == 0 {
		t.Error("element UIDs should be non-zero")
	}
	if uid1 == uid2 {
		t.Error("consecutive element UIDs should differ")
	}
	// High bit should be set.
	if uid1&elementFocusUIDBit == 0 {
		t.Error("element UIDs should have high bit set")
	}
}

func TestElementFocusRoundTrip(t *testing.T) {
	fm := NewFocusManager()
	uid := fm.NextElementUID()
	fm.SetFocusedUID(uid)

	if !fm.IsElementFocused(uid) {
		t.Error("element should be focused after SetFocusedUID")
	}
	fm.Blur()
	if fm.IsElementFocused(uid) {
		t.Error("element should not be focused after Blur")
	}
}

func TestResetOrderClearsElementCounter(t *testing.T) {
	fm := NewFocusManager()
	uid1 := fm.NextElementUID()
	fm.ResetOrder()
	uid2 := fm.NextElementUID()

	// After reset, counter restarts → same UID.
	if uid1 != uid2 {
		t.Errorf("after ResetOrder, NextElementUID should produce same UID: %d != %d", uid1, uid2)
	}
}

func TestNilFocusManagerSafety(t *testing.T) {
	var fm *FocusManager
	if fm.FocusedUID() != 0 {
		t.Error("nil FocusManager should return 0")
	}
	if fm.IsFocused(42) {
		t.Error("nil FocusManager should not report focused")
	}
	fm.SetFocusedUID(42) // should not panic
	fm.Blur()            // should not panic
	fm.ResetOrder()      // should not panic
	fm.SortOrder()       // should not panic
	fm.RegisterFocusable(1, FocusOpts{Focusable: true}) // should not panic

	if fm.NextElementUID() != 0 {
		t.Error("nil FocusManager NextElementUID should return 0")
	}
	if fm.OrderLen() != 0 {
		t.Error("nil FocusManager OrderLen should return 0")
	}
}

func TestInputStateHasFocusUID(t *testing.T) {
	fm := NewFocusManager()
	uid := fm.NextElementUID()
	fm.Input = &InputState{
		Value:    "hello",
		FocusUID: uid,
		OnChange: func(string) {},
	}
	if fm.Input.FocusUID != uid {
		t.Errorf("InputState.FocusUID = %d, want %d", fm.Input.FocusUID, uid)
	}
}

func TestResetElementUIDsStabilizesUIDs(t *testing.T) {
	fm := NewFocusManager()

	// Simulate first BuildScene: assign UIDs to 3 elements.
	fm.ResetElementUIDs()
	_ = fm.NextElementUID() // element 1
	uid2 := fm.NextElementUID() // element 2 (our TextArea)
	_ = fm.NextElementUID() // element 3

	// Focus element 2 and set Input.
	fm.SetFocusedUID(uid2)
	fm.Input = &InputState{
		Value:        "hello\nworld",
		FocusUID:     uid2,
		CursorOffset: 5,
		Multiline:    true,
		OnChange:     func(string) {},
	}

	// Simulate arrow key: modify cursor offset.
	fm.Input.CursorOffset = 3

	// Simulate next BuildScene (idle frame, no ResetOrder).
	fm.ResetElementUIDs()
	_ = fm.NextElementUID()
	uid2again := fm.NextElementUID()
	_ = fm.NextElementUID()

	// UID must be the same.
	if uid2again != uid2 {
		t.Errorf("UID drifted: got %d, want %d", uid2again, uid2)
	}
	// Focus check must still pass.
	if !fm.IsElementFocused(uid2again) {
		t.Error("IsElementFocused should return true after ResetElementUIDs")
	}
	// Input must survive (not nil'd).
	if fm.Input == nil {
		t.Fatal("fm.Input should not be nil")
	}
	if fm.Input.CursorOffset != 3 {
		t.Errorf("CursorOffset = %d, want 3", fm.Input.CursorOffset)
	}
	// FocusUID match check (used in TextArea to preserve cursor).
	if fm.Input.FocusUID != uid2again {
		t.Errorf("Input.FocusUID = %d, want %d", fm.Input.FocusUID, uid2again)
	}
}

func TestInputSurvivesResetOrder(t *testing.T) {
	fm := NewFocusManager()

	fm.ResetElementUIDs()
	uid := fm.NextElementUID()
	fm.SetFocusedUID(uid)
	fm.Input = &InputState{
		Value:        "test",
		FocusUID:     uid,
		CursorOffset: 2,
		Multiline:    true,
		OnChange:     func(string) {},
	}

	// Simulate event: cursor moves.
	fm.Input.CursorOffset = 1

	// Simulate modelDirty frame: ResetOrder + ResetElementUIDs + BuildScene.
	fm.ResetOrder()
	fm.ResetElementUIDs()
	uidAfter := fm.NextElementUID()

	if uidAfter != uid {
		t.Errorf("UID changed after ResetOrder: got %d, want %d", uidAfter, uid)
	}
	if fm.Input == nil {
		t.Fatal("fm.Input should survive ResetOrder")
	}
	if fm.Input.CursorOffset != 1 {
		t.Errorf("CursorOffset = %d, want 1", fm.Input.CursorOffset)
	}
	if !fm.IsElementFocused(uidAfter) {
		t.Error("element should still be focused after ResetOrder")
	}
}
