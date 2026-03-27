package ui

import (
	"testing"
	"time"

	"github.com/timzifer/lux/theme"
)

// testAnimState is a WidgetState that implements Animator.
type testAnimState struct {
	pos      float32
	target   float32
	tickCount int
}

func (s *testAnimState) Tick(dt time.Duration) bool {
	s.tickCount++
	if s.pos == s.target {
		return false
	}
	// Simple step animation for testing.
	s.pos += float32(dt.Seconds()) * 100
	if s.pos >= s.target {
		s.pos = s.target
		return false
	}
	return true
}

// Compile-time check that testAnimState implements Animator.
var _ Animator = (*testAnimState)(nil)

// animWidget uses testAnimState as its WidgetState.
type animWidget struct{}

func (animWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[testAnimState](raw)
	if s.target == 0 {
		s.target = 100
	}
	return Text("anim"), s
}

func TestAnimatorInterface(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("anim", animWidget{})
	r.Reconcile(tree, th, func(_ any) {}, nil, nil, "")

	// After first reconcile, state should exist.
	uid := MakeUID(0, "anim", 0)
	raw := r.StateFor(uid)
	s, ok := raw.(*testAnimState)
	if !ok {
		t.Fatalf("expected *testAnimState, got %T", raw)
	}
	if s.target != 100 {
		t.Errorf("target = %f, want 100", s.target)
	}

	// Tick animations — should advance pos.
	running := r.TickAnimators(1 * time.Second)
	if running {
		t.Error("animation should be done after 1s (pos reaches 100 target)")
	}
	if s.pos != 100 {
		t.Errorf("pos = %f, want 100", s.pos)
	}
	if s.tickCount != 1 {
		t.Errorf("tickCount = %d, want 1", s.tickCount)
	}
}

func TestTickAnimatorsReturnsTrueWhileRunning(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("anim", animWidget{})
	r.Reconcile(tree, th, func(_ any) {}, nil, nil, "")

	// Tick with small dt — should still be running.
	running := r.TickAnimators(100 * time.Millisecond)
	if !running {
		t.Error("animation should still be running after 100ms")
	}

	uid := MakeUID(0, "anim", 0)
	s := r.StateFor(uid).(*testAnimState)
	if s.pos < 9 || s.pos > 11 {
		t.Errorf("pos = %f, want ~10 after 100ms", s.pos)
	}
}

func TestTickAnimatorsNoAnimators(t *testing.T) {
	r := NewReconciler()
	// No states — should return false and not panic.
	if r.TickAnimators(16 * time.Millisecond) {
		t.Error("should return false with no states")
	}
}

// nonAnimState is a WidgetState that does NOT implement Animator.
type nonAnimState struct {
	count int
}

type nonAnimWidget struct{}

func (nonAnimWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[nonAnimState](raw)
	s.count++
	return Text("no-anim"), s
}

func TestTickAnimatorsSkipsNonAnimators(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("noanim", nonAnimWidget{})
	r.Reconcile(tree, th, func(_ any) {}, nil, nil, "")

	// Should not panic and return false.
	running := r.TickAnimators(16 * time.Millisecond)
	if running {
		t.Error("non-animator should not report as running")
	}
}
