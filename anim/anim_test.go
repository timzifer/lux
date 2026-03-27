package anim

import (
	"testing"
	"time"
)

func TestAnimZeroValue(t *testing.T) {
	var a Anim[float32]
	if !a.IsDone() {
		t.Error("zero-value Anim should be done")
	}
	if a.Value() != 0 {
		t.Errorf("zero-value Anim should have value 0, got %v", a.Value())
	}
}

func TestAnimSetTarget(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 200*time.Millisecond, Linear)

	if a.IsDone() {
		t.Error("should not be done after SetTarget")
	}
	if a.Value() != 0 {
		t.Errorf("value should be 0 at start, got %v", a.Value())
	}
}

func TestAnimTickLinear(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 200*time.Millisecond, Linear)

	// Tick halfway.
	running := a.Tick(100 * time.Millisecond)
	if !running {
		t.Error("should still be running at halfway")
	}
	if v := a.Value(); v < 0.49 || v > 0.51 {
		t.Errorf("expected ~0.5 at halfway, got %v", v)
	}
}

func TestAnimTickComplete(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 200*time.Millisecond, Linear)

	// Tick past duration.
	running := a.Tick(300 * time.Millisecond)
	if running {
		t.Error("should not be running after completion")
	}
	if a.Value() != 1.0 {
		t.Errorf("expected exact 1.0 at completion, got %v", a.Value())
	}
	if !a.IsDone() {
		t.Error("should be done after completion")
	}
}

func TestAnimTickExactDuration(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(10.0, 100*time.Millisecond, Linear)

	running := a.Tick(100 * time.Millisecond)
	if running {
		t.Error("should be done at exact duration")
	}
	if a.Value() != 10.0 {
		t.Errorf("expected 10.0, got %v", a.Value())
	}
}

func TestAnimSetImmediate(t *testing.T) {
	var a Anim[float32]
	a.SetImmediate(5.0)

	if !a.IsDone() {
		t.Error("SetImmediate should be immediately done")
	}
	if a.Value() != 5.0 {
		t.Errorf("expected 5.0, got %v", a.Value())
	}
}

func TestAnimRetarget(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 200*time.Millisecond, Linear)
	a.Tick(100 * time.Millisecond) // now at ~0.5

	// Retarget to 0.0 from current value (~0.5).
	a.SetTarget(0.0, 200*time.Millisecond, Linear)

	if a.IsDone() {
		t.Error("should be running after retarget")
	}

	// Value should still be ~0.5 (the from-value of the new animation).
	if v := a.Value(); v < 0.49 || v > 0.51 {
		t.Errorf("expected ~0.5 after retarget, got %v", v)
	}

	// Tick the new animation to completion.
	a.Tick(200 * time.Millisecond)
	if a.Value() != 0.0 {
		t.Errorf("expected 0.0 after retarget completion, got %v", a.Value())
	}
}

func TestAnimTickReturnsFalseWhenDone(t *testing.T) {
	var a Anim[float32]
	// Not running — Tick should return false.
	if a.Tick(16 * time.Millisecond) {
		t.Error("Tick on zero-value Anim should return false")
	}
}

func TestAnimFloat64(t *testing.T) {
	var a Anim[float64]
	a.SetTarget(100.0, 100*time.Millisecond, Linear)
	a.Tick(50 * time.Millisecond)
	if v := a.Value(); v < 49 || v > 51 {
		t.Errorf("expected ~50.0, got %v", v)
	}
	a.Tick(50 * time.Millisecond)
	if a.Value() != 100.0 {
		t.Errorf("expected exact 100.0, got %v", a.Value())
	}
}

// ── Easing tests ────────────────────────────────────────────────

func TestEasingLinearEndpoints(t *testing.T) {
	if v := Linear(0); v != 0 {
		t.Errorf("Linear(0) = %v, want 0", v)
	}
	if v := Linear(1); v != 1 {
		t.Errorf("Linear(1) = %v, want 1", v)
	}
}

func TestEasingOutCubicEndpoints(t *testing.T) {
	if v := OutCubic(0); v != 0 {
		t.Errorf("OutCubic(0) = %v, want 0", v)
	}
	if v := OutCubic(1); v != 1 {
		t.Errorf("OutCubic(1) = %v, want 1", v)
	}
}

func TestEasingInCubicEndpoints(t *testing.T) {
	if v := InCubic(0); v != 0 {
		t.Errorf("InCubic(0) = %v, want 0", v)
	}
	if v := InCubic(1); v != 1 {
		t.Errorf("InCubic(1) = %v, want 1", v)
	}
}

func TestEasingInOutCubicEndpoints(t *testing.T) {
	if v := InOutCubic(0); v != 0 {
		t.Errorf("InOutCubic(0) = %v, want 0", v)
	}
	if v := InOutCubic(1); v != 1 {
		t.Errorf("InOutCubic(1) = %v, want 1", v)
	}
}

func TestEasingOutExpoEndpoints(t *testing.T) {
	if v := OutExpo(0); v != 0 {
		t.Errorf("OutExpo(0) = %v, want 0", v)
	}
	if v := OutExpo(1); v != 1 {
		t.Errorf("OutExpo(1) = %v, want 1", v)
	}
}

func TestEasingOutCubicFasterThanLinear(t *testing.T) {
	// OutCubic should be ahead of linear at t=0.5.
	if OutCubic(0.5) <= Linear(0.5) {
		t.Error("OutCubic(0.5) should be > Linear(0.5)")
	}
}

func TestAnimWithNilEasing(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 100*time.Millisecond, nil)
	a.Tick(50 * time.Millisecond)
	// nil easing falls back to Linear.
	if v := a.Value(); v < 0.49 || v > 0.51 {
		t.Errorf("nil easing should default to linear, got %v", v)
	}
}

// ── CubicBezier tests ───────────────────────────────────────────

func TestCubicBezierEndpoints(t *testing.T) {
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	if v := ease(0); v != 0 {
		t.Errorf("CubicBezier(0) = %v, want 0", v)
	}
	if v := ease(1); v != 1 {
		t.Errorf("CubicBezier(1) = %v, want 1", v)
	}
}

func TestCubicBezierEaseOut(t *testing.T) {
	// CSS "ease": faster than linear at midpoint.
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	v := ease(0.5)
	if v <= 0.5 {
		t.Errorf("CSS ease at 0.5 should be > 0.5, got %v", v)
	}
}

func TestCubicBezierEaseInOut(t *testing.T) {
	// CSS "ease-in-out": symmetric, ~0.5 at midpoint.
	easeInOut := CubicBezier(0.42, 0, 0.58, 1.0)
	v := easeInOut(0.5)
	if v < 0.45 || v > 0.55 {
		t.Errorf("CSS ease-in-out at 0.5 should be ~0.5, got %v", v)
	}
}

func TestCubicBezierLinearEquivalent(t *testing.T) {
	// (0,0,1,1) should approximate Linear.
	linear := CubicBezier(0, 0, 1, 1)
	for _, x := range []float32{0.0, 0.25, 0.5, 0.75, 1.0} {
		v := linear(x)
		if v < x-0.02 || v > x+0.02 {
			t.Errorf("CubicBezier(0,0,1,1)(%v) = %v, want ~%v", x, v, x)
		}
	}
}

func TestCubicBezierMonotonic(t *testing.T) {
	ease := CubicBezier(0.25, 0.1, 0.25, 1.0)
	prev := float32(0)
	for i := 1; i <= 100; i++ {
		x := float32(i) / 100
		v := ease(x)
		if v < prev-0.001 {
			t.Errorf("CubicBezier should be monotonic, but f(%v)=%v < f(%v)=%v", x, v, float32(i-1)/100, prev)
		}
		prev = v
	}
}

// ── AnimationID / SetTargetWithID tests ─────────────────────────

func TestSetTargetWithIDFiresOnComplete(t *testing.T) {
	var received AnimationEnded
	SendFunc = func(msg any) {
		if ended, ok := msg.(AnimationEnded); ok {
			received = ended
		}
	}
	defer func() { SendFunc = nil }()

	var a Anim[float32]
	a.SetTargetWithID(1.0, 100*time.Millisecond, Linear, "test-anim")

	// Not yet complete.
	a.Tick(50 * time.Millisecond)
	if received.ID != "" {
		t.Error("should not fire before completion")
	}

	// Complete.
	a.Tick(100 * time.Millisecond)
	if received.ID != "test-anim" {
		t.Errorf("expected AnimationEnded{test-anim}, got %v", received)
	}
}

func TestSetTargetWithIDNoDoubleFire(t *testing.T) {
	count := 0
	SendFunc = func(msg any) {
		if _, ok := msg.(AnimationEnded); ok {
			count++
		}
	}
	defer func() { SendFunc = nil }()

	var a Anim[float32]
	a.SetTargetWithID(1.0, 100*time.Millisecond, Linear, "once")
	a.Tick(200 * time.Millisecond) // completes and fires
	a.Tick(200 * time.Millisecond) // should not fire again

	if count != 1 {
		t.Errorf("expected 1 fire, got %d", count)
	}
}

func TestSetTargetWithIDReplacesNotification(t *testing.T) {
	var received AnimationEnded
	SendFunc = func(msg any) {
		if ended, ok := msg.(AnimationEnded); ok {
			received = ended
		}
	}
	defer func() { SendFunc = nil }()

	var a Anim[float32]
	a.SetTargetWithID(1.0, 200*time.Millisecond, Linear, "first")
	a.Tick(50 * time.Millisecond)

	// Retarget with new ID.
	a.SetTargetWithID(2.0, 100*time.Millisecond, Linear, "second")
	a.Tick(200 * time.Millisecond)

	if received.ID != "second" {
		t.Errorf("expected AnimationEnded{second}, got %v", received)
	}
}

func TestSetTargetWithIDNilSendFunc(t *testing.T) {
	SendFunc = nil

	var a Anim[float32]
	a.SetTargetWithID(1.0, 100*time.Millisecond, Linear, "safe")
	// Should not panic when SendFunc is nil.
	a.Tick(200 * time.Millisecond)

	if !a.IsDone() {
		t.Error("animation should be done")
	}
}
