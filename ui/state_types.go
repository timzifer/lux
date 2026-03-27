// Package ui — state_types.go contains state and configuration types
// shared between the core ui package and sub-packages (data, nav, form, etc.).
package ui

import (
	"time"

	"github.com/timzifer/lux/anim"
)

// ── TreeState ────────────────────────────────────────────────────

// TreeState tracks expand/collapse and selection state for a Tree widget.
type TreeState struct {
	Expanded   map[string]bool
	Selected   string
	Scroll     ScrollState
	expandAnim map[string]*anim.Anim[float32] // per-node expand animation (0=collapsed, 1=expanded)
	motionDur  time.Duration                  // cached from theme tokens during layout
	motionEase anim.EasingFunc                // cached from theme tokens during layout

	// Double-click tracking for expand-on-double-click.
	lastClickID   string
	lastClickTime time.Time
}

// NewTreeState creates a ready-to-use TreeState.
func NewTreeState() *TreeState {
	return &TreeState{
		Expanded:   make(map[string]bool),
		expandAnim: make(map[string]*anim.Anim[float32]),
	}
}

// IsExpanded reports whether the given node is expanded.
func (ts *TreeState) IsExpanded(id string) bool {
	return ts != nil && ts.Expanded[id]
}

// Toggle flips the expand/collapse state of a node with animation.
func (ts *TreeState) Toggle(id string) {
	if ts == nil {
		return
	}
	ts.Expanded[id] = !ts.Expanded[id]

	dur := ts.motionDur
	eas := ts.motionEase
	if dur == 0 {
		dur = 220 * time.Millisecond
		eas = anim.OutCubic
	}
	a := ts.getOrCreateAnim(id)
	if ts.Expanded[id] {
		a.SetTarget(1.0, dur, eas)
	} else {
		a.SetTarget(0.0, dur, eas)
	}
}

// ExpandProgress returns the current expand animation progress for a node.
func (ts *TreeState) ExpandProgress(id string) float32 {
	return ts.expandProgress(id)
}

func (ts *TreeState) expandProgress(id string) float32 {
	if ts == nil {
		return 0
	}
	if a, ok := ts.expandAnim[id]; ok {
		return a.Value()
	}
	if ts.Expanded[id] {
		return 1.0
	}
	return 0.0
}

// IsAnimating reports whether a node's expand/collapse is currently animating.
func (ts *TreeState) IsAnimating(id string) bool {
	return ts.isAnimating(id)
}

func (ts *TreeState) isAnimating(id string) bool {
	if ts == nil {
		return false
	}
	a, ok := ts.expandAnim[id]
	return ok && !a.IsDone()
}

func (ts *TreeState) getOrCreateAnim(id string) *anim.Anim[float32] {
	if ts.expandAnim == nil {
		ts.expandAnim = make(map[string]*anim.Anim[float32])
	}
	a, ok := ts.expandAnim[id]
	if !ok {
		a = &anim.Anim[float32]{}
		if ts.Expanded[id] {
			a.SetImmediate(0.0)
		} else {
			a.SetImmediate(1.0)
		}
		ts.expandAnim[id] = a
	}
	return a
}

// Tick advances all expand/collapse animations by dt.
func (ts *TreeState) Tick(dt time.Duration) {
	if ts == nil {
		return
	}
	for id, a := range ts.expandAnim {
		if !a.Tick(dt) {
			delete(ts.expandAnim, id)
		}
	}
}

// CacheMotion stores the motion spec from theme tokens so that Toggle()
// can use them.
func (ts *TreeState) CacheMotion(dur time.Duration, easing anim.EasingFunc) {
	if ts == nil {
		return
	}
	ts.motionDur = dur
	ts.motionEase = easing
}

// SetSelected sets the currently selected node.
func (ts *TreeState) SetSelected(id string) {
	if ts != nil {
		ts.Selected = id
	}
}

// LastClickID returns the ID of the last clicked node.
func (ts *TreeState) LastClickID() string {
	if ts == nil {
		return ""
	}
	return ts.lastClickID
}

// LastClickTime returns the time of the last click.
func (ts *TreeState) LastClickTime() time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.lastClickTime
}

// RecordClick records a click on a node for double-click detection.
func (ts *TreeState) RecordClick(id string, t time.Time) {
	if ts != nil {
		ts.lastClickID = id
		ts.lastClickTime = t
	}
}

// ── TreeConfig ───────────────────────────────────────────────────

// TreeConfig configures a Tree element.
type TreeConfig struct {
	RootIDs     []string
	Children    func(id string) []string                              // returns child IDs; nil/empty = leaf
	BuildNode   func(id string, depth int, expanded, selected bool) Element // builds the display for a node
	NodeHeight  float32                                                // uniform height per node (dp); 0 = 28dp
	IndentWidth float32                                                // per-level indent (dp); 0 = 20dp
	MaxHeight   float32                                                // viewport height (dp)
	State       *TreeState
	OnSelect    func(id string) // called when a node is clicked
}

// ── VirtualListConfig ────────────────────────────────────────────

// VirtualListConfig configures a VirtualList element.
type VirtualListConfig struct {
	ItemCount  int                    // Total number of items.
	ItemHeight float32                // Uniform height per item in dp.
	BuildItem  func(index int) Element // Builds the element for a given index.
	MaxHeight  float32                // Viewport height in dp.
	State      *ScrollState           // Scroll state (required for scrolling).
}
