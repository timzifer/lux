package input

import "testing"

func TestKeyMsgFields(t *testing.T) {
	msg := KeyMsg{
		Key:    "A",
		Action: KeyPress,
		Modifiers: KeyModifiers{
			Shift: true,
			Ctrl:  false,
		},
	}
	if msg.Key != "A" {
		t.Errorf("Key = %q, want A", msg.Key)
	}
	if msg.Action != KeyPress {
		t.Errorf("Action = %d, want KeyPress", msg.Action)
	}
	if !msg.Modifiers.Shift {
		t.Error("Shift should be true")
	}
}

func TestScrollMsgFields(t *testing.T) {
	msg := ScrollMsg{DeltaX: 1.5, DeltaY: -3.0}
	if msg.DeltaX != 1.5 {
		t.Errorf("DeltaX = %f, want 1.5", msg.DeltaX)
	}
	if msg.DeltaY != -3.0 {
		t.Errorf("DeltaY = %f, want -3.0", msg.DeltaY)
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
