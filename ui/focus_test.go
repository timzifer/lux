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
