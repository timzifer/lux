package input

import "testing"

func TestKeyMsgFields(t *testing.T) {
	msg := KeyMsg{
		Key:       KeyA,
		Rune:      'A',
		Action:    KeyPress,
		Modifiers: ModShift,
	}
	if msg.Key != KeyA {
		t.Errorf("Key = %d, want KeyA", msg.Key)
	}
	if msg.Rune != 'A' {
		t.Errorf("Rune = %c, want A", msg.Rune)
	}
	if msg.Action != KeyPress {
		t.Errorf("Action = %d, want KeyPress", msg.Action)
	}
	if !msg.Modifiers.Has(ModShift) {
		t.Error("Shift should be set")
	}
	if msg.Modifiers.Has(ModCtrl) {
		t.Error("Ctrl should not be set")
	}
}

func TestModifierSetBitfield(t *testing.T) {
	ms := ModShift | ModCtrl
	if !ms.Has(ModShift) {
		t.Error("should have Shift")
	}
	if !ms.Has(ModCtrl) {
		t.Error("should have Ctrl")
	}
	if ms.Has(ModAlt) {
		t.Error("should not have Alt")
	}
	if ms.Has(ModSuper) {
		t.Error("should not have Super")
	}
	// Has with combined mask.
	if !ms.Has(ModShift | ModCtrl) {
		t.Error("should have Shift+Ctrl combined")
	}
}

func TestModsFromBits(t *testing.T) {
	ms := ModsFromBits(0b1011) // Shift + Ctrl + Super
	if !ms.Has(ModShift) {
		t.Error("should have Shift")
	}
	if !ms.Has(ModCtrl) {
		t.Error("should have Ctrl")
	}
	if ms.Has(ModAlt) {
		t.Error("should not have Alt")
	}
	if !ms.Has(ModSuper) {
		t.Error("should have Super")
	}
}

func TestScrollMsgFields(t *testing.T) {
	msg := ScrollMsg{X: 10, Y: 20, DeltaX: 1.5, DeltaY: -3.0, Precise: true}
	if msg.DeltaX != 1.5 {
		t.Errorf("DeltaX = %f, want 1.5", msg.DeltaX)
	}
	if msg.DeltaY != -3.0 {
		t.Errorf("DeltaY = %f, want -3.0", msg.DeltaY)
	}
	if !msg.Precise {
		t.Error("Precise should be true")
	}
	if msg.X != 10 || msg.Y != 20 {
		t.Errorf("Pos = (%f, %f), want (10, 20)", msg.X, msg.Y)
	}
}

func TestMouseMsgActions(t *testing.T) {
	if MousePress != 0 {
		t.Errorf("MousePress = %d, want 0", MousePress)
	}
	if MouseRelease != 1 {
		t.Errorf("MouseRelease = %d, want 1", MouseRelease)
	}
	if MouseMove != 2 {
		t.Errorf("MouseMove = %d, want 2", MouseMove)
	}
	if MouseScroll != 3 {
		t.Errorf("MouseScroll = %d, want 3", MouseScroll)
	}
}

func TestMouseEnterLeaveActions(t *testing.T) {
	if MouseEnter != 4 {
		t.Errorf("MouseEnter = %d, want 4", MouseEnter)
	}
	if MouseLeave != 5 {
		t.Errorf("MouseLeave = %d, want 5", MouseLeave)
	}
}

func TestCharMsgPrintable(t *testing.T) {
	msg := CharMsg{Char: 'Z'}
	if msg.Char != 'Z' {
		t.Errorf("Char = %c, want Z", msg.Char)
	}
}

func TestKeyActions(t *testing.T) {
	if KeyPress != 0 {
		t.Errorf("KeyPress = %d, want 0", KeyPress)
	}
	if KeyRelease != 1 {
		t.Errorf("KeyRelease = %d, want 1", KeyRelease)
	}
	if KeyRepeat != 2 {
		t.Errorf("KeyRepeat = %d, want 2", KeyRepeat)
	}
}

func TestTextInputMsg(t *testing.T) {
	msg := TextInputMsg{Text: "über"}
	if msg.Text != "über" {
		t.Errorf("Text = %q, want über", msg.Text)
	}
}

func TestTouchMsg(t *testing.T) {
	msg := TouchMsg{
		ID:    42,
		X:     100.5,
		Y:     200.3,
		Phase: TouchMoved,
		Force: 0.75,
	}
	if msg.ID != 42 {
		t.Errorf("ID = %d, want 42", msg.ID)
	}
	if msg.Phase != TouchMoved {
		t.Errorf("Phase = %d, want TouchMoved", msg.Phase)
	}
	if msg.Force != 0.75 {
		t.Errorf("Force = %f, want 0.75", msg.Force)
	}
}

func TestTouchPhases(t *testing.T) {
	if TouchBegan != 0 {
		t.Errorf("TouchBegan = %d, want 0", TouchBegan)
	}
	if TouchMoved != 1 {
		t.Errorf("TouchMoved = %d, want 1", TouchMoved)
	}
	if TouchEnded != 2 {
		t.Errorf("TouchEnded = %d, want 2", TouchEnded)
	}
	if TouchCancelled != 3 {
		t.Errorf("TouchCancelled = %d, want 3", TouchCancelled)
	}
}

func TestKeyNameToKeyMapping(t *testing.T) {
	tests := []struct {
		name string
		want Key
	}{
		{"A", KeyA},
		{"Z", KeyZ},
		{"Escape", KeyEscape},
		{"Backspace", KeyBackspace},
		{"Enter", KeyEnter},
		{"Space", KeySpace},
		{"Tab", KeyTab},
		{"F1", KeyF1},
		{"F12", KeyF12},
	}
	for _, tt := range tests {
		got, ok := KeyNameToKey[tt.name]
		if !ok {
			t.Errorf("KeyNameToKey[%q] not found", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("KeyNameToKey[%q] = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestKeyConstantsNonZero(t *testing.T) {
	// All named keys should be non-zero (KeyUnknown = 0).
	keys := []Key{
		KeyA, KeyZ, Key0, Key9,
		KeyF1, KeyF12,
		KeyEscape, KeyEnter, KeyTab, KeyBackspace,
		KeySpace, KeyLeftShift, KeyMenu,
	}
	for _, k := range keys {
		if k == KeyUnknown {
			t.Errorf("key constant should not be KeyUnknown (0)")
		}
	}
}

func TestMouseMsgWithModifiers(t *testing.T) {
	msg := MouseMsg{
		X: 10, Y: 20,
		Button:    MouseButtonLeft,
		Action:    MousePress,
		Modifiers: ModCtrl | ModShift,
	}
	if !msg.Modifiers.Has(ModCtrl) {
		t.Error("should have Ctrl modifier")
	}
	if !msg.Modifiers.Has(ModShift) {
		t.Error("should have Shift modifier")
	}
}

// ── Shortcut Tests (RFC-002 §2.5) ──────────────────────────────

func TestShortcutEquality(t *testing.T) {
	a := Shortcut{Key: KeyS, Modifiers: ModCtrl}
	b := Shortcut{Key: KeyS, Modifiers: ModCtrl}
	if a != b {
		t.Error("identical shortcuts should be equal")
	}
	c := Shortcut{Key: KeyS, Modifiers: ModCtrl | ModShift}
	if a == c {
		t.Error("shortcuts with different modifiers should not be equal")
	}
}

func TestPlatformShortcut(t *testing.T) {
	s := PlatformShortcut(PlatformActionCopy)
	if s.Key != KeyC {
		t.Errorf("copy shortcut key = %d, want KeyC", s.Key)
	}
	// Platform modifier: Ctrl on Linux/Windows, Super (Cmd) on macOS.
	mod := platformModifier()
	if !s.Modifiers.Has(mod) {
		t.Errorf("copy shortcut should have platform modifier %d", mod)
	}
}

func TestPlatformShortcutRedo(t *testing.T) {
	s := PlatformShortcut(PlatformActionRedo)
	if s.Key != KeyZ {
		t.Errorf("redo key = %d, want KeyZ", s.Key)
	}
	if !s.Modifiers.Has(ModShift) {
		t.Error("redo should have Shift modifier")
	}
}

func TestShortcutMsgFields(t *testing.T) {
	msg := ShortcutMsg{
		Shortcut: Shortcut{Key: KeyS, Modifiers: ModCtrl},
		ID:       "save",
	}
	if msg.ID != "save" {
		t.Errorf("ID = %q, want save", msg.ID)
	}
}

// ── CursorKind Tests (RFC-002 §2.7) ────────────────────────────

func TestCursorKindValues(t *testing.T) {
	if CursorDefault != 0 {
		t.Errorf("CursorDefault = %d, want 0", CursorDefault)
	}
	if CursorNone != 14 {
		t.Errorf("CursorNone = %d, want 14", CursorNone)
	}
	// Ensure all cursor kinds are distinct.
	seen := make(map[CursorKind]bool)
	cursors := []CursorKind{
		CursorDefault, CursorText, CursorPointer, CursorMove,
		CursorResizeNS, CursorResizeEW, CursorResizeNESW, CursorResizeNWSE,
		CursorNotAllowed, CursorCrosshair, CursorGrab, CursorGrabbing,
		CursorWait, CursorProgress, CursorNone,
	}
	for _, c := range cursors {
		if seen[c] {
			t.Errorf("duplicate CursorKind value: %d", c)
		}
		seen[c] = true
	}
}
