// Package hit provides hit-testing for interactive elements (RFC §13, Stufe 3).
package hit

import "github.com/timzifer/lux/draw"

// Target is a clickable region with an associated callback.
type Target struct {
	Bounds  draw.Rect
	OnClick func()
}

// Map collects hit targets during layout and resolves pointer hits.
type Map struct {
	targets []Target
}

// Add registers a clickable region. Ignored if onClick is nil.
func (m *Map) Add(bounds draw.Rect, onClick func()) {
	if onClick == nil {
		return
	}
	m.targets = append(m.targets, Target{Bounds: bounds, OnClick: onClick})
}

// HitTest returns the top-most target containing (x, y), or nil.
func (m *Map) HitTest(x, y float32) *Target {
	for i := len(m.targets) - 1; i >= 0; i-- {
		if m.targets[i].Bounds.Contains(draw.Pt(x, y)) {
			return &m.targets[i]
		}
	}
	return nil
}

// Reset clears all targets for the next frame.
func (m *Map) Reset() {
	m.targets = m.targets[:0]
}

// HitTestIndex returns the index of the top-most target containing (x, y), or -1.
func (m *Map) HitTestIndex(x, y float32) int {
	for i := len(m.targets) - 1; i >= 0; i-- {
		if m.targets[i].Bounds.Contains(draw.Pt(x, y)) {
			return i
		}
	}
	return -1
}

// Len returns the number of registered targets.
func (m *Map) Len() int {
	return len(m.targets)
}
