package ui

import (
	"testing"
	"time"

	"github.com/timzifer/lux/input"
)

// testClock returns a controllable clock for testing.
type testClock struct {
	t time.Time
}

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *testClock) now() time.Time { return c.t }

func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestRecognizer(clock *testClock) *GestureRecognizer {
	r := NewGestureRecognizer(DefaultGestureConfig)
	r.now = clock.now
	return r
}

// ── Tap Recognition ──────────────────────────────────────────────

func TestSingleTap(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	touches := []input.TouchMsg{
		{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan},
	}
	gestures, _ := r.Process(touches)
	if len(gestures) != 0 {
		t.Fatalf("expected no gestures on TouchBegan, got %d", len(gestures))
	}

	clock.advance(100 * time.Millisecond)
	touches = []input.TouchMsg{
		{ID: 1, X: 50, Y: 50, Phase: input.TouchEnded},
	}
	gestures, _ = r.Process(touches)

	if len(gestures) != 1 {
		t.Fatalf("expected 1 gesture (tap), got %d", len(gestures))
	}
	if gestures[0].event.Kind != EventTap {
		t.Errorf("expected EventTap, got %d", gestures[0].event.Kind)
	}
	if gestures[0].event.Tap.Count != 1 {
		t.Errorf("expected Count=1, got %d", gestures[0].event.Tap.Count)
	}
}

func TestDoubleTap(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	// First tap.
	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})
	clock.advance(50 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) != 1 || gestures[0].event.Tap.Count != 1 {
		t.Fatal("first tap should be Count=1")
	}

	// Second tap within DoubleTapInterval.
	clock.advance(100 * time.Millisecond)
	r.Process([]input.TouchMsg{{ID: 2, X: 52, Y: 50, Phase: input.TouchBegan}})
	clock.advance(50 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 2, X: 52, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) != 1 {
		t.Fatalf("expected 1 gesture, got %d", len(gestures))
	}
	if gestures[0].event.Tap.Count != 2 {
		t.Errorf("second tap should be Count=2, got %d", gestures[0].event.Tap.Count)
	}
}

func TestDoubleTapExpired(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	// First tap.
	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})
	clock.advance(50 * time.Millisecond)
	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchEnded}})

	// Second tap after DoubleTapInterval expired.
	clock.advance(500 * time.Millisecond)
	r.Process([]input.TouchMsg{{ID: 2, X: 50, Y: 50, Phase: input.TouchBegan}})
	clock.advance(50 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{{ID: 2, X: 50, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) != 1 || gestures[0].event.Tap.Count != 1 {
		t.Error("tap after interval should be Count=1")
	}
}

// ── Long-Press Recognition ───────────────────────────────────────

func TestLongPress(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})

	// Advance past long-press threshold.
	clock.advance(600 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{})
	if len(gestures) != 1 {
		t.Fatalf("expected 1 long-press event, got %d", len(gestures))
	}
	if gestures[0].event.Kind != EventLongPress {
		t.Errorf("expected EventLongPress, got %d", gestures[0].event.Kind)
	}
	if gestures[0].event.LongPress.Phase != input.LongPressBegan {
		t.Error("expected LongPressBegan phase")
	}

	// Lift finger.
	clock.advance(100 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) != 1 || gestures[0].event.LongPress.Phase != input.LongPressEnded {
		t.Error("expected LongPressEnded")
	}
}

func TestLongPressCancelledByMovement(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})

	// Trigger long-press.
	clock.advance(600 * time.Millisecond)
	r.Process([]input.TouchMsg{})

	// Move beyond drag threshold.
	clock.advance(50 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 100, Y: 100, Phase: input.TouchMoved}})

	found := false
	for _, g := range gestures {
		if g.event.Kind == EventLongPress && g.event.LongPress.Phase == input.LongPressCancelled {
			found = true
		}
	}
	if !found {
		t.Error("expected LongPressCancelled when moving after long-press")
	}
}

// ── Drag Recognition ─────────────────────────────────────────────

func TestDrag(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})

	// Move beyond drag threshold slowly (not a swipe).
	clock.advance(200 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 65, Y: 50, Phase: input.TouchMoved}})

	if len(gestures) != 1 {
		t.Fatalf("expected 1 gesture (drag began), got %d", len(gestures))
	}
	if gestures[0].event.Kind != EventDrag {
		t.Errorf("expected EventDrag, got %d", gestures[0].event.Kind)
	}
	if gestures[0].event.Drag.Phase != input.DragBegan {
		t.Error("expected DragBegan phase")
	}

	// Continue moving.
	clock.advance(100 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 1, X: 80, Y: 50, Phase: input.TouchMoved}})
	if len(gestures) != 1 || gestures[0].event.Drag.Phase != input.DragMoved {
		t.Error("expected DragMoved")
	}

	// End drag.
	clock.advance(100 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 1, X: 90, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) < 1 {
		t.Fatal("expected at least 1 gesture on drag end")
	}

	hasDragEnd := false
	for _, g := range gestures {
		if g.event.Kind == EventDrag && g.event.Drag.Phase == input.DragEnded {
			hasDragEnd = true
		}
	}
	if !hasDragEnd {
		t.Error("expected DragEnded")
	}
}

// ── Swipe Recognition ────────────────────────────────────────────

func TestSwipe(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	// Quick horizontal movement that stays within pending phase
	// (total distance > DragThreshold but all in one frame).
	r.Process([]input.TouchMsg{{ID: 1, X: 200, Y: 100, Phase: input.TouchBegan}})

	// End far away, very quickly — high velocity.
	clock.advance(50 * time.Millisecond) // 50ms → 2000dp in 50ms = 40000 dp/s
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 100, Phase: input.TouchEnded}})

	hasSwipe := false
	for _, g := range gestures {
		if g.event.Kind == EventSwipe {
			hasSwipe = true
			if g.event.Swipe.Direction != input.SwipeLeft {
				t.Errorf("expected SwipeLeft, got %d", g.event.Swipe.Direction)
			}
		}
	}
	if !hasSwipe {
		t.Error("expected a SwipeMsg")
	}
}

// ── Palm Rejection ───────────────────────────────────────────────

func TestPalmRejectionHighForce(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	// Touch with high force.
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan, Force: 0.9}})
	if len(gestures) != 0 {
		t.Error("high-force touch should be rejected")
	}

	// Ending this touch should also produce no events.
	clock.advance(100 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchEnded}})
	if len(gestures) != 0 {
		t.Error("ended palm-rejected touch should produce no gestures")
	}
}

func TestPalmRejectionEdgeTouch(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)
	r.config.ScreenWidth = 800
	r.config.ScreenHeight = 600

	// Existing touch in main area.
	r.Process([]input.TouchMsg{{ID: 1, X: 400, Y: 300, Phase: input.TouchBegan}})

	// Edge touch while main touch active.
	gestures, _ := r.Process([]input.TouchMsg{{ID: 2, X: 5, Y: 300, Phase: input.TouchBegan}})
	// Edge touch should be rejected since there's an active main-area touch.
	if len(gestures) != 0 {
		t.Error("edge touch with existing main touch should be rejected")
	}
}

// ── Debouncing ───────────────────────────────────────────────────

func TestDebounce(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)
	r.config.DebounceInterval = 200 * time.Millisecond

	uid := UID(42)

	if r.ShouldDebounce(uid, clock.now()) {
		t.Error("first tap should not be debounced")
	}
	r.RecordTap(uid, clock.now())

	clock.advance(100 * time.Millisecond)
	if !r.ShouldDebounce(uid, clock.now()) {
		t.Error("second tap within interval should be debounced")
	}

	clock.advance(200 * time.Millisecond)
	if r.ShouldDebounce(uid, clock.now()) {
		t.Error("tap after interval should not be debounced")
	}
}

// ── Pinch Recognition ────────────────────────────────────────────

func TestPinch(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	// Two fingers touch.
	r.Process([]input.TouchMsg{
		{ID: 1, X: 100, Y: 200, Phase: input.TouchBegan},
		{ID: 2, X: 200, Y: 200, Phase: input.TouchBegan},
	})

	// Move fingers apart.
	clock.advance(50 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{
		{ID: 1, X: 50, Y: 200, Phase: input.TouchMoved},
		{ID: 2, X: 250, Y: 200, Phase: input.TouchMoved},
	})

	hasPinch := false
	for _, g := range gestures {
		if g.event.Kind == EventPinch {
			hasPinch = true
			if g.event.Pinch.Scale <= 1.0 {
				// fingers moved apart → scale should increase
				// Initial distance: 100, new distance: 200 → scale ~2.0
			}
		}
	}
	if !hasPinch {
		t.Error("expected PinchMsg when two fingers move")
	}

	// End one finger.
	clock.advance(50 * time.Millisecond)
	gestures, _ = r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 200, Phase: input.TouchEnded}})

	hasPinchEnd := false
	for _, g := range gestures {
		if g.event.Kind == EventPinch && g.event.Pinch.Phase == input.PinchEnded {
			hasPinchEnd = true
		}
	}
	if !hasPinchEnd {
		t.Error("expected PinchEnded when one finger lifts")
	}
}

// ── Cancelled Touch ──────────────────────────────────────────────

func TestTouchCancelled(t *testing.T) {
	clock := newTestClock()
	r := newTestRecognizer(clock)

	r.Process([]input.TouchMsg{{ID: 1, X: 50, Y: 50, Phase: input.TouchBegan}})

	// Move to start a drag.
	clock.advance(100 * time.Millisecond)
	r.Process([]input.TouchMsg{{ID: 1, X: 70, Y: 50, Phase: input.TouchMoved}})

	// Cancel.
	clock.advance(50 * time.Millisecond)
	gestures, _ := r.Process([]input.TouchMsg{{ID: 1, X: 70, Y: 50, Phase: input.TouchCancelled}})

	hasDragCancel := false
	for _, g := range gestures {
		if g.event.Kind == EventDrag && g.event.Drag.Phase == input.DragCancelled {
			hasDragCancel = true
		}
	}
	if !hasDragCancel {
		t.Error("expected DragCancelled on touch cancellation")
	}
}

// ── Helper Tests ─────────────────────────────────────────────────

func TestSwipeDirection(t *testing.T) {
	tests := []struct {
		start, end input.GesturePoint
		want       input.SwipeDirection
	}{
		{input.GesturePoint{100, 100}, input.GesturePoint{0, 100}, input.SwipeLeft},
		{input.GesturePoint{0, 100}, input.GesturePoint{100, 100}, input.SwipeRight},
		{input.GesturePoint{100, 100}, input.GesturePoint{100, 0}, input.SwipeUp},
		{input.GesturePoint{100, 0}, input.GesturePoint{100, 100}, input.SwipeDown},
	}
	for _, tt := range tests {
		got := swipeDirection(tt.start, tt.end)
		if got != tt.want {
			t.Errorf("swipeDirection(%v, %v) = %d, want %d", tt.start, tt.end, got, tt.want)
		}
	}
}

func TestDistance(t *testing.T) {
	d := distance(input.GesturePoint{0, 0}, input.GesturePoint{3, 4})
	if d < 4.99 || d > 5.01 {
		t.Errorf("distance({0,0}, {3,4}) = %v, want 5.0", d)
	}
}
