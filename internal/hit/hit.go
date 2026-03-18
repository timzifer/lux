// Package hit provides hit-testing for interactive elements (RFC §13, Stufe 3).
package hit

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
)

// Target is a clickable region with an associated callback.
type Target struct {
	Bounds    draw.Rect
	OnClick   func()
	OnClickAt func(x, y float32) // positional click (e.g. Slider)
	Draggable bool               // if true, OnClickAt fires continuously during drag
	Cursor    input.CursorKind   // cursor to show when hovering (RFC-002 §2.7)
}

// ScrollTarget is a scrollable region linked to a ScrollState.
type ScrollTarget struct {
	Bounds        draw.Rect
	ContentHeight float32
	ViewportHeight float32
	OnScroll      func(deltaY float32) // called with scroll delta
}

// Map collects hit targets during layout and resolves pointer hits.
type Map struct {
	targets       []Target
	scrollTargets []ScrollTarget
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
	m.scrollTargets = m.scrollTargets[:0]
}

// AddScroll registers a scrollable viewport region.
func (m *Map) AddScroll(bounds draw.Rect, contentH, viewportH float32, onScroll func(deltaY float32)) {
	if onScroll == nil {
		return
	}
	m.scrollTargets = append(m.scrollTargets, ScrollTarget{
		Bounds:         bounds,
		ContentHeight:  contentH,
		ViewportHeight: viewportH,
		OnScroll:       onScroll,
	})
}

// HitTestScroll returns the scroll target containing (x, y), or nil.
func (m *Map) HitTestScroll(x, y float32) *ScrollTarget {
	for i := len(m.scrollTargets) - 1; i >= 0; i-- {
		if m.scrollTargets[i].Bounds.Contains(draw.Pt(x, y)) {
			return &m.scrollTargets[i]
		}
	}
	return nil
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

// AddAt registers a clickable region with a positional callback. Ignored if onClick is nil.
func (m *Map) AddAt(bounds draw.Rect, onClick func(x, y float32)) {
	if onClick == nil {
		return
	}
	m.targets = append(m.targets, Target{Bounds: bounds, OnClickAt: onClick})
}

// AddDrag registers a draggable region. OnClickAt fires on initial click and
// continuously while the mouse is held and moved.
func (m *Map) AddDrag(bounds draw.Rect, onClick func(x, y float32)) {
	if onClick == nil {
		return
	}
	m.targets = append(m.targets, Target{Bounds: bounds, OnClickAt: onClick, Draggable: true})
}

// Len returns the number of registered targets.
func (m *Map) Len() int {
	return len(m.targets)
}
