package anim

import (
	"testing"
	"time"
)

func TestSpringAnimZeroValue(t *testing.T) {
	var s SpringAnim[float32]
	if !s.IsDone() {
		t.Error("zero-value SpringAnim should be done")
	}
	if s.Value() != 0 {
		t.Errorf("zero-value SpringAnim should have value 0, got %v", s.Value())
	}
}

func TestSpringAnimSetTarget(t *testing.T) {
	var s SpringAnim[float32]
	s.SetTarget(1.0)
	if s.IsDone() {
		t.Error("should not be done after SetTarget")
	}
}

func TestSpringAnimConverges(t *testing.T) {
	var s SpringAnim[float32]
	s.SetTargetWithSpec(1.0, SpringSnappy)

	// Simulate ~2 seconds of frames at 60fps.
	dt := 16 * time.Millisecond
	for i := 0; i < 120; i++ {
		s.Tick(dt)
		if s.IsDone() {
			break
		}
	}

	if !s.IsDone() {
		t.Errorf("SpringSnappy should settle within 2s, current=%v", s.Value())
	}
	if s.Value() != 1.0 {
		t.Errorf("expected exact 1.0 at settling, got %v", s.Value())
	}
}

func TestSpringAnimGentleConverges(t *testing.T) {
	var s SpringAnim[float32]
	s.SetTargetWithSpec(100.0, SpringGentle)

	dt := 16 * time.Millisecond
	for i := 0; i < 300; i++ {
		s.Tick(dt)
		if s.IsDone() {
			break
		}
	}

	if !s.IsDone() {
		t.Errorf("SpringGentle should settle within 5s, current=%v", s.Value())
	}
}

func TestSpringAnimBouncyOvershoots(t *testing.T) {
	var s SpringAnim[float32]
	s.SetTargetWithSpec(1.0, SpringBouncy)

	dt := 16 * time.Millisecond
	overshot := false
	for i := 0; i < 300; i++ {
		s.Tick(dt)
		if s.Value() > 1.0 {
			overshot = true
		}
		if s.IsDone() {
			break
		}
	}

	if !overshot {
		t.Error("SpringBouncy should overshoot the target")
	}
	if !s.IsDone() {
		t.Errorf("SpringBouncy should eventually settle, current=%v", s.Value())
	}
}

func TestSpringAnimSetImmediate(t *testing.T) {
	var s SpringAnim[float32]
	s.SetImmediate(5.0)

	if !s.IsDone() {
		t.Error("SetImmediate should be immediately done")
	}
	if s.Value() != 5.0 {
		t.Errorf("expected 5.0, got %v", s.Value())
	}
}

func TestSpringAnimTickWhenDone(t *testing.T) {
	var s SpringAnim[float32]
	if s.Tick(16 * time.Millisecond) {
		t.Error("Tick on zero-value SpringAnim should return false")
	}
}

func TestSpringAnimRetarget(t *testing.T) {
	var s SpringAnim[float32]
	s.SetTargetWithSpec(1.0, SpringSnappy)

	// Run a few frames.
	for i := 0; i < 5; i++ {
		s.Tick(16 * time.Millisecond)
	}

	// Retarget.
	s.SetTarget(0.0)
	if s.IsDone() {
		t.Error("should be running after retarget")
	}

	// Should eventually settle at 0.
	for i := 0; i < 300; i++ {
		s.Tick(16 * time.Millisecond)
		if s.IsDone() {
			break
		}
	}

	if !s.IsDone() {
		t.Error("should settle after retarget")
	}
	if s.Value() != 0.0 {
		t.Errorf("expected 0.0, got %v", s.Value())
	}
}

func TestSpringAnimFloat64(t *testing.T) {
	var s SpringAnim[float64]
	s.SetTargetWithSpec(100.0, SpringSpec{
		Stiffness:         400,
		Damping:           28,
		Mass:              1.0,
		SettlingThreshold: 0.001,
	})

	dt := 16 * time.Millisecond
	for i := 0; i < 200; i++ {
		s.Tick(dt)
		if s.IsDone() {
			break
		}
	}

	if !s.IsDone() {
		t.Errorf("float64 spring should settle, current=%v", s.Value())
	}
}
