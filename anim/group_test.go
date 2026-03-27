package anim

import (
	"testing"
	"time"
)

// ── AnimGroup Tests ──────────────────────────────────────────────

func TestAnimGroupEmpty(t *testing.T) {
	g := NewAnimGroup()
	if !g.IsDone() {
		t.Error("empty group should be done")
	}
	if g.Tick(16 * time.Millisecond) {
		t.Error("empty group Tick should return false")
	}
}

func TestAnimGroupParallel(t *testing.T) {
	var a Anim[float32]
	var b Anim[float32]
	a.SetTarget(1.0, 100*time.Millisecond, Linear)
	b.SetTarget(1.0, 200*time.Millisecond, Linear)

	g := NewAnimGroup(&a, &b)

	if g.IsDone() {
		t.Error("group should not be done at start")
	}

	// Tick past first animation's duration.
	g.Tick(150 * time.Millisecond)
	if a.Value() != 1.0 {
		t.Errorf("a should be done, got %v", a.Value())
	}
	if g.IsDone() {
		t.Error("group should not be done yet (b still running)")
	}

	// Tick past second.
	g.Tick(100 * time.Millisecond)
	if !g.IsDone() {
		t.Error("group should be done after all anims complete")
	}
	if b.Value() != 1.0 {
		t.Errorf("b should be 1.0, got %v", b.Value())
	}
}

func TestAnimGroupAdd(t *testing.T) {
	g := NewAnimGroup()
	var a Anim[float32]
	a.SetTarget(1.0, 100*time.Millisecond, Linear)

	g.Add(&a)
	if g.IsDone() {
		t.Error("group with added anim should not be done")
	}

	g.Tick(200 * time.Millisecond)
	if !g.IsDone() {
		t.Error("group should be done after tick past duration")
	}
}

// ── AnimSeq Tests ────────────────────────────────────────────────

func TestAnimSeqEmpty(t *testing.T) {
	s := NewAnimSeq()
	if !s.IsDone() {
		t.Error("empty seq should be done")
	}
	if s.Tick(16 * time.Millisecond) {
		t.Error("empty seq Tick should return false")
	}
}

func TestAnimSeqSequential(t *testing.T) {
	var a Anim[float32]
	var b Anim[float32]
	a.SetTarget(1.0, 100*time.Millisecond, Linear)
	b.SetTarget(1.0, 100*time.Millisecond, Linear)

	s := NewAnimSeq().Then(&a).Then(&b)

	// Tick a to completion.
	s.Tick(150 * time.Millisecond)
	if a.Value() != 1.0 {
		t.Errorf("a should be done, got %v", a.Value())
	}
	if s.IsDone() {
		t.Error("seq should not be done (b not started)")
	}

	// Tick b to completion.
	s.Tick(150 * time.Millisecond)
	if b.Value() != 1.0 {
		t.Errorf("b should be done, got %v", b.Value())
	}
	if !s.IsDone() {
		t.Error("seq should be done after both steps")
	}
}

func TestAnimSeqOnDoneHook(t *testing.T) {
	var a Anim[float32]
	a.SetTarget(1.0, 100*time.Millisecond, Linear)

	hookCalled := false
	var b Anim[float32]

	s := NewAnimSeq().
		Then(&a, func() {
			hookCalled = true
			b.SetTarget(1.0, 100*time.Millisecond, Linear)
		}).
		Then(&b)

	// Complete first step.
	s.Tick(150 * time.Millisecond)
	if !hookCalled {
		t.Error("onDone hook should have been called")
	}

	// b should now be running (started by hook).
	if b.IsDone() {
		t.Error("b should be running after hook started it")
	}

	// Complete second step.
	s.Tick(150 * time.Millisecond)
	if !s.IsDone() {
		t.Error("seq should be done")
	}
}

func TestAnimSeqChaining(t *testing.T) {
	var a, b, c Anim[float32]
	a.SetTarget(1.0, 50*time.Millisecond, Linear)
	b.SetTarget(1.0, 50*time.Millisecond, Linear)
	c.SetTarget(1.0, 50*time.Millisecond, Linear)

	s := NewAnimSeq().Then(&a).Then(&b).Then(&c)

	// Run all three.
	for i := 0; i < 10; i++ {
		if s.IsDone() {
			break
		}
		s.Tick(60 * time.Millisecond)
	}

	if !s.IsDone() {
		t.Error("seq with 3 steps should complete")
	}
}
