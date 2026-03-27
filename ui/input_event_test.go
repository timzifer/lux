package ui

import (
	"testing"

	"github.com/timzifer/lux/input"
)

func TestKeyEventConstructor(t *testing.T) {
	msg := input.KeyMsg{Key: input.KeyA, Action: input.KeyPress, Modifiers: input.ModShift}
	ev := KeyEvent(msg)
	if ev.Kind != EventKey {
		t.Errorf("Kind = %d, want EventKey", ev.Kind)
	}
	if ev.Key == nil || ev.Key.Key != input.KeyA {
		t.Error("Key field not populated correctly")
	}
	// Other fields should be nil.
	if ev.Mouse != nil || ev.Scroll != nil || ev.Touch != nil {
		t.Error("other fields should be nil")
	}
}

func TestTextInputEventConstructor(t *testing.T) {
	msg := input.TextInputMsg{Text: "hello"}
	ev := TextInputEvent(msg)
	if ev.Kind != EventTextInput {
		t.Errorf("Kind = %d, want EventTextInput", ev.Kind)
	}
	if ev.TextInput == nil || ev.TextInput.Text != "hello" {
		t.Error("TextInput field not populated correctly")
	}
}

func TestCharEventConstructor(t *testing.T) {
	msg := input.CharMsg{Char: 'X'}
	ev := CharEvent(msg)
	if ev.Kind != EventChar {
		t.Errorf("Kind = %d, want EventChar", ev.Kind)
	}
	if ev.Char == nil || ev.Char.Char != 'X' {
		t.Error("Char field not populated correctly")
	}
}

func TestMouseEventConstructor(t *testing.T) {
	msg := input.MouseMsg{X: 10, Y: 20, Action: input.MousePress, Button: input.MouseButtonLeft}
	ev := MouseEvent(msg)
	if ev.Kind != EventMouse {
		t.Errorf("Kind = %d, want EventMouse", ev.Kind)
	}
	if ev.Mouse == nil || ev.Mouse.X != 10 {
		t.Error("Mouse field not populated correctly")
	}
}

func TestScrollEventConstructor(t *testing.T) {
	msg := input.ScrollMsg{DeltaY: -3, Precise: true, X: 5, Y: 10}
	ev := ScrollEvent(msg)
	if ev.Kind != EventScroll {
		t.Errorf("Kind = %d, want EventScroll", ev.Kind)
	}
	if ev.Scroll == nil || !ev.Scroll.Precise || ev.Scroll.DeltaY != -3 {
		t.Error("Scroll field not populated correctly")
	}
}

func TestTouchEventConstructor(t *testing.T) {
	msg := input.TouchMsg{ID: 1, X: 100, Y: 200, Phase: input.TouchBegan, Force: 0.5}
	ev := TouchEvent(msg)
	if ev.Kind != EventTouch {
		t.Errorf("Kind = %d, want EventTouch", ev.Kind)
	}
	if ev.Touch == nil || ev.Touch.ID != 1 || ev.Touch.Force != 0.5 {
		t.Error("Touch field not populated correctly")
	}
}

func TestFocusGainedEventConstructor(t *testing.T) {
	msg := FocusGainedMsg{Source: FocusSourceTab}
	ev := FocusGainedEvent(msg)
	if ev.Kind != EventFocusGained {
		t.Errorf("Kind = %d, want EventFocusGained", ev.Kind)
	}
	if ev.FocusGained == nil || ev.FocusGained.Source != FocusSourceTab {
		t.Error("FocusGained field not populated correctly")
	}
}

func TestFocusLostEventConstructor(t *testing.T) {
	msg := FocusLostMsg{Source: FocusSourceClick}
	ev := FocusLostEvent(msg)
	if ev.Kind != EventFocusLost {
		t.Errorf("Kind = %d, want EventFocusLost", ev.Kind)
	}
	if ev.FocusLost == nil || ev.FocusLost.Source != FocusSourceClick {
		t.Error("FocusLost field not populated correctly")
	}
}

func TestInputEventKindValues(t *testing.T) {
	// Ensure enum values are sequential starting from 0.
	if EventKey != 0 {
		t.Errorf("EventKey = %d, want 0", EventKey)
	}
	if EventFocusLost != 7 {
		t.Errorf("EventFocusLost = %d, want 7", EventFocusLost)
	}
	// Gesture event kinds follow IME kinds.
	if EventTap != 10 {
		t.Errorf("EventTap = %d, want 10", EventTap)
	}
	if EventPinch != 14 {
		t.Errorf("EventPinch = %d, want 14", EventPinch)
	}
}

func TestTapEventConstructor(t *testing.T) {
	msg := input.TapMsg{Pos: input.GesturePoint{X: 50, Y: 60}, Count: 2}
	ev := TapEvent(msg)
	if ev.Kind != EventTap {
		t.Errorf("Kind = %d, want EventTap", ev.Kind)
	}
	if ev.Tap == nil || ev.Tap.Count != 2 || ev.Tap.Pos.X != 50 {
		t.Error("Tap field not populated correctly")
	}
}

func TestLongPressEventConstructor(t *testing.T) {
	msg := input.LongPressMsg{Pos: input.GesturePoint{X: 10, Y: 20}, Phase: input.LongPressBegan}
	ev := LongPressEvent(msg)
	if ev.Kind != EventLongPress {
		t.Errorf("Kind = %d, want EventLongPress", ev.Kind)
	}
	if ev.LongPress == nil || ev.LongPress.Phase != input.LongPressBegan {
		t.Error("LongPress field not populated correctly")
	}
}

func TestSwipeEventConstructor(t *testing.T) {
	msg := input.SwipeMsg{Direction: input.SwipeLeft, Velocity: 400, Start: input.GesturePoint{X: 200, Y: 100}, End: input.GesturePoint{X: 50, Y: 100}}
	ev := SwipeEvent(msg)
	if ev.Kind != EventSwipe {
		t.Errorf("Kind = %d, want EventSwipe", ev.Kind)
	}
	if ev.Swipe == nil || ev.Swipe.Direction != input.SwipeLeft || ev.Swipe.Velocity != 400 {
		t.Error("Swipe field not populated correctly")
	}
}

func TestDragEventConstructor(t *testing.T) {
	msg := input.DragMsg{Phase: input.DragMoved, Start: input.GesturePoint{X: 10, Y: 10}, Pos: input.GesturePoint{X: 50, Y: 50}, Delta: input.GesturePoint{X: 3, Y: 4}}
	ev := DragEvent(msg)
	if ev.Kind != EventDrag {
		t.Errorf("Kind = %d, want EventDrag", ev.Kind)
	}
	if ev.Drag == nil || ev.Drag.Phase != input.DragMoved || ev.Drag.Delta.X != 3 {
		t.Error("Drag field not populated correctly")
	}
}

func TestPinchEventConstructor(t *testing.T) {
	msg := input.PinchMsg{Phase: input.PinchChanged, Center: input.GesturePoint{X: 100, Y: 100}, Scale: 1.5}
	ev := PinchEvent(msg)
	if ev.Kind != EventPinch {
		t.Errorf("Kind = %d, want EventPinch", ev.Kind)
	}
	if ev.Pinch == nil || ev.Pinch.Scale != 1.5 {
		t.Error("Pinch field not populated correctly")
	}
}

func TestRenderCtxEventsField(t *testing.T) {
	ctx := RenderCtx{
		Events: []InputEvent{
			KeyEvent(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress}),
			MouseEvent(input.MouseMsg{X: 10, Y: 20, Action: input.MouseMove}),
		},
	}
	if len(ctx.Events) != 2 {
		t.Fatalf("Events length = %d, want 2", len(ctx.Events))
	}
	if ctx.Events[0].Kind != EventKey {
		t.Error("first event should be EventKey")
	}
	if ctx.Events[1].Kind != EventMouse {
		t.Error("second event should be EventMouse")
	}
}
