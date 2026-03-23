package input

import "testing"

func TestGesturePhaseConstants(t *testing.T) {
	// LongPressPhase values must be sequential from 0.
	if LongPressBegan != 0 || LongPressEnded != 1 || LongPressCancelled != 2 {
		t.Error("LongPressPhase constants out of order")
	}

	// SwipeDirection values must be sequential from 0.
	if SwipeLeft != 0 || SwipeRight != 1 || SwipeUp != 2 || SwipeDown != 3 {
		t.Error("SwipeDirection constants out of order")
	}

	// DragPhase values must be sequential from 0.
	if DragBegan != 0 || DragMoved != 1 || DragEnded != 2 || DragCancelled != 3 {
		t.Error("DragPhase constants out of order")
	}

	// PinchPhase values must be sequential from 0.
	if PinchBegan != 0 || PinchChanged != 1 || PinchEnded != 2 || PinchCancelled != 3 {
		t.Error("PinchPhase constants out of order")
	}
}

func TestTapMsgCount(t *testing.T) {
	tap := TapMsg{Pos: GesturePoint{X: 10, Y: 20}, Count: 2}
	if tap.Count != 2 {
		t.Errorf("TapMsg.Count = %d, want 2", tap.Count)
	}
}

func TestSwipeMsgDirection(t *testing.T) {
	swipe := SwipeMsg{
		Direction: SwipeUp,
		Velocity:  500,
		Start:     GesturePoint{X: 100, Y: 400},
		End:       GesturePoint{X: 100, Y: 100},
	}
	if swipe.Direction != SwipeUp {
		t.Errorf("SwipeMsg.Direction = %d, want SwipeUp", swipe.Direction)
	}
}

func TestDragMsgDelta(t *testing.T) {
	drag := DragMsg{
		Phase: DragMoved,
		Start: GesturePoint{X: 10, Y: 10},
		Pos:   GesturePoint{X: 30, Y: 40},
		Delta: GesturePoint{X: 5, Y: 3},
	}
	if drag.Delta.X != 5 || drag.Delta.Y != 3 {
		t.Errorf("DragMsg.Delta = (%v, %v), want (5, 3)", drag.Delta.X, drag.Delta.Y)
	}
}

func TestPinchMsgScale(t *testing.T) {
	pinch := PinchMsg{
		Phase:  PinchChanged,
		Center: GesturePoint{X: 200, Y: 300},
		Scale:  1.5,
	}
	if pinch.Scale != 1.5 {
		t.Errorf("PinchMsg.Scale = %v, want 1.5", pinch.Scale)
	}
}
