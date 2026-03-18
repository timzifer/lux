package hit

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestHitTestFindsTarget(t *testing.T) {
	var m Map
	called := false
	m.Add(draw.R(10, 10, 100, 50), func() { called = true })

	target := m.HitTest(50, 30)
	if target == nil {
		t.Fatal("expected target, got nil")
	}
	target.OnClick()
	if !called {
		t.Error("OnClick was not called")
	}
}

func TestHitTestMiss(t *testing.T) {
	var m Map
	m.Add(draw.R(10, 10, 100, 50), func() {})

	if m.HitTest(200, 200) != nil {
		t.Error("expected nil for miss")
	}
}

func TestHitTestTopMost(t *testing.T) {
	var m Map
	var which int
	m.Add(draw.R(0, 0, 200, 200), func() { which = 1 })
	m.Add(draw.R(50, 50, 100, 100), func() { which = 2 })

	target := m.HitTest(75, 75)
	if target == nil {
		t.Fatal("expected target")
	}
	target.OnClick()
	if which != 2 {
		t.Errorf("expected top-most target (2), got %d", which)
	}
}

func TestHitTestNilOnClickIgnored(t *testing.T) {
	var m Map
	m.Add(draw.R(0, 0, 100, 100), nil)
	if m.Len() != 0 {
		t.Errorf("nil OnClick should be ignored, got len=%d", m.Len())
	}
}

func TestReset(t *testing.T) {
	var m Map
	m.Add(draw.R(0, 0, 100, 100), func() {})
	if m.Len() != 1 {
		t.Fatalf("expected 1 target, got %d", m.Len())
	}
	m.Reset()
	if m.Len() != 0 {
		t.Errorf("after reset expected 0, got %d", m.Len())
	}
}

func TestHitTestEmpty(t *testing.T) {
	var m Map
	if m.HitTest(10, 10) != nil {
		t.Error("empty map should return nil")
	}
}

func TestHitTestEdge(t *testing.T) {
	var m Map
	m.Add(draw.R(10, 10, 100, 50), func() {})

	// Exact top-left corner is inside
	if m.HitTest(10, 10) == nil {
		t.Error("top-left corner should hit")
	}
	// Just outside bottom-right
	if m.HitTest(110, 60) != nil {
		t.Error("bottom-right edge should miss")
	}
}
