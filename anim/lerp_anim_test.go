package anim

import (
	"testing"
	"time"
)

// point is a test type to exercise LerpAnim without importing draw.
type point struct{ X, Y float32 }

func lerpPoint(a, b point, t float32) point {
	return point{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
	}
}

func TestLerpAnimZeroValue(t *testing.T) {
	var a LerpAnim[point]
	if !a.IsDone() {
		t.Error("zero-value LerpAnim should be done")
	}
	if v := a.Value(); v.X != 0 || v.Y != 0 {
		t.Errorf("zero-value should be {0,0}, got %v", v)
	}
}

func TestLerpAnimSetTarget(t *testing.T) {
	var a LerpAnim[point]
	a.SetTarget(point{100, 200}, 200*time.Millisecond, Linear, lerpPoint)
	if a.IsDone() {
		t.Error("should not be done after SetTarget")
	}
}

func TestLerpAnimTickHalfway(t *testing.T) {
	var a LerpAnim[point]
	a.SetTarget(point{100, 200}, 200*time.Millisecond, Linear, lerpPoint)
	running := a.Tick(100 * time.Millisecond)
	if !running {
		t.Error("should still be running at halfway")
	}
	v := a.Value()
	if v.X < 49 || v.X > 51 || v.Y < 99 || v.Y > 101 {
		t.Errorf("expected ~{50,100} at halfway, got %v", v)
	}
}

func TestLerpAnimTickComplete(t *testing.T) {
	var a LerpAnim[point]
	a.SetTarget(point{100, 200}, 200*time.Millisecond, Linear, lerpPoint)
	a.Tick(300 * time.Millisecond)
	if !a.IsDone() {
		t.Error("should be done after completion")
	}
	v := a.Value()
	if v.X != 100 || v.Y != 200 {
		t.Errorf("expected exact {100,200}, got %v", v)
	}
}

func TestLerpAnimSetImmediate(t *testing.T) {
	var a LerpAnim[point]
	a.SetImmediate(point{42, 84})
	if !a.IsDone() {
		t.Error("SetImmediate should be done")
	}
	v := a.Value()
	if v.X != 42 || v.Y != 84 {
		t.Errorf("expected {42,84}, got %v", v)
	}
}

func TestLerpAnimRetarget(t *testing.T) {
	var a LerpAnim[point]
	a.SetTarget(point{100, 0}, 200*time.Millisecond, Linear, lerpPoint)
	a.Tick(100 * time.Millisecond) // ~{50,0}

	a.SetTarget(point{0, 0}, 200*time.Millisecond, Linear, lerpPoint)
	v := a.Value()
	if v.X < 49 || v.X > 51 {
		t.Errorf("expected ~50 after retarget, got %v", v.X)
	}

	a.Tick(200 * time.Millisecond)
	v = a.Value()
	if v.X != 0 || v.Y != 0 {
		t.Errorf("expected {0,0} after retarget completion, got %v", v)
	}
}

func TestLerpAnimTickReturnsFalseWhenDone(t *testing.T) {
	var a LerpAnim[point]
	if a.Tick(16 * time.Millisecond) {
		t.Error("Tick on zero-value LerpAnim should return false")
	}
}
