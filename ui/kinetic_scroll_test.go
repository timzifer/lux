package ui

import (
	"testing"
	"time"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

func testScrollSpec() theme.ScrollSpec {
	return theme.ScrollSpec{
		Friction:          0.95,
		Overscroll:        40,
		TrackWidth:        8,
		ThumbRadius:       4,
		SettlingThreshold: 0.5,
		StepSize:          48,
		MultiplierPrecise: 1.5,
	}
}

func TestKineticScroll_DiscreteStep(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 1000)

	// Scroll down (negative deltaY in mouse wheel = scroll content down = increase offset).
	ks.Feed(input.ScrollMsg{DeltaY: -1, Precise: false})
	if ks.OffsetY != 48 {
		t.Errorf("OffsetY after discrete scroll down = %f, want 48", ks.OffsetY)
	}

	// Scroll up.
	ks.Feed(input.ScrollMsg{DeltaY: 1, Precise: false})
	if ks.OffsetY != 0 {
		t.Errorf("OffsetY after discrete scroll up = %f, want 0", ks.OffsetY)
	}
}

func TestKineticScroll_DiscreteStepClampsToBounds(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 100)

	// Scroll past max bound.
	for i := 0; i < 10; i++ {
		ks.Feed(input.ScrollMsg{DeltaY: -1, Precise: false})
	}
	if ks.OffsetY != 100 {
		t.Errorf("OffsetY after over-scroll = %f, want 100 (clamped)", ks.OffsetY)
	}
}

func TestKineticScroll_FrictionDecay(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 10000)

	// Simulate trackpad scroll with several deltas to build velocity.
	for i := 0; i < 5; i++ {
		ks.Feed(input.ScrollMsg{DeltaY: -10, Precise: true})
		ks.Tick(16 * time.Millisecond)
	}
	// Now lift finger: no more Feed, next Tick starts deceleration.
	ks.Tick(16 * time.Millisecond)

	if ks.Phase() != scrollDecelerating {
		t.Errorf("phase after finger lift = %d, want scrollDecelerating (%d)", ks.Phase(), scrollDecelerating)
	}

	// Tick several times — velocity should decrease.
	initialOffset := ks.OffsetY
	for i := 0; i < 100; i++ {
		ks.Tick(16 * time.Millisecond)
	}
	if ks.OffsetY <= initialOffset {
		t.Error("offset should have increased during deceleration")
	}
}

func TestKineticScroll_SettlingThreshold(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 10000)

	// Small trackpad scroll.
	ks.Feed(input.ScrollMsg{DeltaY: -5, Precise: true})
	ks.Tick(16 * time.Millisecond)

	// Lift finger.
	ks.Tick(16 * time.Millisecond)

	// Tick until settled.
	for i := 0; i < 1000; i++ {
		if !ks.Tick(16 * time.Millisecond) {
			break
		}
	}
	if !ks.IsDone() {
		t.Error("expected IsDone() = true after enough ticks")
	}
}

func TestKineticScroll_OverscrollRubberBand(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 100)

	// Scroll past top bound.
	ks.Feed(input.ScrollMsg{DeltaY: 20, Precise: true})
	ks.Tick(16 * time.Millisecond)

	if ks.OffsetY >= 0 {
		t.Errorf("OffsetY = %f, expected negative (overscrolled past min)", ks.OffsetY)
	}

	// Lift finger and let spring pull back.
	for i := 0; i < 500; i++ {
		if !ks.Tick(16 * time.Millisecond) {
			break
		}
	}
	// Should have snapped back to min bound.
	if ks.OffsetY < -1 || ks.OffsetY > 1 {
		t.Errorf("OffsetY after snap = %f, expected near 0", ks.OffsetY)
	}
}

func TestKineticScroll_SnapToImmediate(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	ks.SetBounds(0, 0, 0, 1000)

	ks.SnapToImmediate(0, 500)
	if ks.OffsetY != 500 {
		t.Errorf("OffsetY = %f, want 500", ks.OffsetY)
	}
	if !ks.IsDone() {
		t.Error("expected IsDone() = true after SnapToImmediate")
	}
}

func TestKineticScroll_NoBoundsNoScroll(t *testing.T) {
	ks := NewKineticScroll(testScrollSpec())
	// Bounds where max < min → content smaller than viewport.
	ks.SetBounds(0, 0, 0, -10)

	ks.Feed(input.ScrollMsg{DeltaY: -1, Precise: false})
	if ks.OffsetY != 0 {
		t.Errorf("OffsetY = %f, expected 0 (no scroll possible)", ks.OffsetY)
	}
}
