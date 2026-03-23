// Package ui — gesture.go implements the framework-internal gesture recognizer
// (RFC-004 §3.2–§3.6). It transforms raw TouchMsg sequences into semantic
// gesture events (TapMsg, LongPressMsg, SwipeMsg, DragMsg, PinchMsg).
//
// The recognizer sits between platform input and widget dispatch. It consumes
// TouchMsg sequences and emits gesture InputEvents. Unrecognised touch
// sequences are passed through unchanged.
package ui

import (
	"math"
	"time"

	"github.com/timzifer/lux/input"
)

// ── GestureConfig ────────────────────────────────────────────────

// GestureConfig holds thresholds and timing parameters for gesture
// recognition. Default values match RFC-004 §2.2 ProfileDesktop.
// When InteractionProfile is implemented, these values will be
// derived from it.
type GestureConfig struct {
	// LongPressDuration is the time a finger must rest before a
	// long-press is recognised. Default: 500ms.
	LongPressDuration time.Duration

	// DoubleTapInterval is the maximum time between two taps for
	// double-tap recognition. Default: 400ms.
	DoubleTapInterval time.Duration

	// DragThreshold is the minimum movement in dp before a touch
	// becomes a drag. Default: 10dp.
	DragThreshold float32

	// SwipeVelocityMin is the minimum velocity in dp/s for a
	// movement to be classified as swipe (vs. drag). Default: 300dp/s.
	SwipeVelocityMin float32

	// DebounceInterval is the minimum time between two accepted taps
	// on the same widget. 0 = no debounce. Default: 0 (desktop).
	DebounceInterval time.Duration

	// ScreenWidth and ScreenHeight are used for palm rejection
	// edge detection (§3.5). 0 = palm edge rejection disabled.
	ScreenWidth  float32
	ScreenHeight float32
}

// DefaultGestureConfig provides desktop defaults (RFC-004 §2.3).
var DefaultGestureConfig = GestureConfig{
	LongPressDuration: 500 * time.Millisecond,
	DoubleTapInterval: 400 * time.Millisecond,
	DragThreshold:     10,
	SwipeVelocityMin:  300,
	DebounceInterval:  0,
}

// ── touchState ───────────────────────────────────────────────────

// touchState tracks a single touch from TouchBegan to TouchEnded.
type touchState struct {
	id        int64
	startTime time.Time
	startPos  input.GesturePoint
	lastPos   input.GesturePoint
	prevPos   input.GesturePoint // position at previous frame
	prevTime  time.Time
	phase     touchGesturePhase
	rejected  bool // palm rejection
}

// touchGesturePhase tracks what the recognizer currently thinks this touch is.
type touchGesturePhase uint8

const (
	touchPending   touchGesturePhase = iota // waiting for disambiguation
	touchDragging                           // confirmed drag
	touchLongPress                          // long-press threshold reached
	touchConsumed                           // gesture emitted, done
)

// ── gestureResult ────────────────────────────────────────────────

// gestureResult pairs a gesture event with the position it occurred at
// (for hit-testing by the dispatcher).
type gestureResult struct {
	pos   input.GesturePoint
	event InputEvent
}

// ── GestureRecognizer ────────────────────────────────────────────

// GestureRecognizer transforms raw TouchMsg sequences into semantic
// gesture events (RFC-004 §3.2).
type GestureRecognizer struct {
	config GestureConfig
	now    func() time.Time // injectable clock for testing

	// Per-touch tracking.
	touches map[int64]*touchState

	// Double-tap tracking: last tap position and time.
	lastTapPos  input.GesturePoint
	lastTapTime time.Time
	tapCount    int

	// Debounce tracking per widget UID.
	lastTapPerUID map[UID]time.Time

	// Pinch tracking: the two active touch IDs for a pinch.
	pinchIDs      [2]int64
	pinchActive   bool
	pinchStartDist float32
}

// NewGestureRecognizer creates a recognizer with the given config.
func NewGestureRecognizer(config GestureConfig) *GestureRecognizer {
	return &GestureRecognizer{
		config:        config,
		now:           time.Now,
		touches:       make(map[int64]*touchState),
		lastTapPerUID: make(map[UID]time.Time),
	}
}

// Process takes the raw touch events for this frame and returns
// gesture events to be dispatched. Unconsumed touches are returned
// separately so the dispatcher can still route them to widgets.
func (g *GestureRecognizer) Process(touches []input.TouchMsg) (gestures []gestureResult, passthrough []input.TouchMsg) {
	now := g.now()

	for i := range touches {
		t := &touches[i]

		// Palm rejection (§3.5).
		if g.isPalmRejected(t, now) {
			continue
		}

		switch t.Phase {
		case input.TouchBegan:
			g.handleBegan(t, now)
		case input.TouchMoved:
			results := g.handleMoved(t, now)
			gestures = append(gestures, results...)
		case input.TouchEnded:
			results := g.handleEnded(t, now)
			gestures = append(gestures, results...)
		case input.TouchCancelled:
			results := g.handleCancelled(t)
			gestures = append(gestures, results...)
		}
	}

	// Check for long-press on pending touches.
	for _, ts := range g.touches {
		if ts.phase == touchPending && !ts.rejected {
			elapsed := now.Sub(ts.startTime)
			if elapsed >= g.config.LongPressDuration {
				ts.phase = touchLongPress
				gestures = append(gestures, gestureResult{
					pos: ts.startPos,
					event: LongPressEvent(input.LongPressMsg{
						Pos:   ts.startPos,
						Phase: input.LongPressBegan,
					}),
				})
			}
		}
	}

	// Check for pinch gesture with two active touches.
	if !g.pinchActive {
		g.detectPinchStart(now)
	}

	return gestures, passthrough
}

// ── Touch phase handlers ─────────────────────────────────────────

func (g *GestureRecognizer) handleBegan(t *input.TouchMsg, now time.Time) {
	pos := input.GesturePoint{X: t.X, Y: t.Y}
	g.touches[t.ID] = &touchState{
		id:        t.ID,
		startTime: now,
		startPos:  pos,
		lastPos:   pos,
		prevPos:   pos,
		prevTime:  now,
		phase:     touchPending,
	}
}

func (g *GestureRecognizer) handleMoved(t *input.TouchMsg, now time.Time) []gestureResult {
	ts, ok := g.touches[t.ID]
	if !ok || ts.rejected {
		return nil
	}

	pos := input.GesturePoint{X: t.X, Y: t.Y}
	delta := input.GesturePoint{X: pos.X - ts.lastPos.X, Y: pos.Y - ts.lastPos.Y}
	ts.prevPos = ts.lastPos
	ts.prevTime = now
	ts.lastPos = pos

	// If we're already in a pinch, update pinch state.
	if g.pinchActive && (t.ID == g.pinchIDs[0] || t.ID == g.pinchIDs[1]) {
		return g.updatePinch(now)
	}

	dist := distance(ts.startPos, pos)

	switch ts.phase {
	case touchPending:
		if dist > g.config.DragThreshold {
			// Exceeded drag threshold — start a drag.
			ts.phase = touchDragging
			return []gestureResult{{
				pos: pos,
				event: DragEvent(input.DragMsg{
					Phase: input.DragBegan,
					Start: ts.startPos,
					Pos:   pos,
					Delta: delta,
				}),
			}}
		}
	case touchLongPress:
		if dist > g.config.DragThreshold {
			// Long-press cancelled by movement.
			ts.phase = touchConsumed
			return []gestureResult{{
				pos: ts.startPos,
				event: LongPressEvent(input.LongPressMsg{
					Pos:   ts.startPos,
					Phase: input.LongPressCancelled,
				}),
			}}
		}
	case touchDragging:
		return []gestureResult{{
			pos: pos,
			event: DragEvent(input.DragMsg{
				Phase: input.DragMoved,
				Start: ts.startPos,
				Pos:   pos,
				Delta: delta,
			}),
		}}
	}

	return nil
}

func (g *GestureRecognizer) handleEnded(t *input.TouchMsg, now time.Time) []gestureResult {
	ts, ok := g.touches[t.ID]
	if !ok || ts.rejected {
		delete(g.touches, t.ID)
		return nil
	}
	defer delete(g.touches, t.ID)

	pos := input.GesturePoint{X: t.X, Y: t.Y}

	// If pinch was active and this touch ends, end the pinch.
	if g.pinchActive && (t.ID == g.pinchIDs[0] || t.ID == g.pinchIDs[1]) {
		return g.endPinch()
	}

	switch ts.phase {
	case touchPending:
		// Touch ended without exceeding drag threshold or long-press.
		// Check velocity for swipe vs tap.
		dist := distance(ts.startPos, pos)
		elapsed := now.Sub(ts.startTime)

		if dist > g.config.DragThreshold && elapsed > 0 {
			velocity := dist / float32(elapsed.Seconds())
			if velocity >= g.config.SwipeVelocityMin {
				return []gestureResult{{
					pos: ts.startPos,
					event: SwipeEvent(input.SwipeMsg{
						Direction: swipeDirection(ts.startPos, pos),
						Velocity:  velocity,
						Start:     ts.startPos,
						End:       pos,
					}),
				}}
			}
		}

		// It's a tap.
		return g.emitTap(ts.startPos, now)

	case touchLongPress:
		return []gestureResult{{
			pos: ts.startPos,
			event: LongPressEvent(input.LongPressMsg{
				Pos:   ts.startPos,
				Phase: input.LongPressEnded,
			}),
		}}

	case touchDragging:
		// Check if the end velocity qualifies as a swipe.
		endDist := distance(ts.prevPos, pos)
		endElapsed := now.Sub(ts.prevTime)
		if endElapsed > 0 {
			velocity := endDist / float32(endElapsed.Seconds())
			if velocity >= g.config.SwipeVelocityMin {
				totalDist := distance(ts.startPos, pos)
				if totalDist > g.config.DragThreshold {
					return []gestureResult{
						{
							pos: pos,
							event: DragEvent(input.DragMsg{
								Phase: input.DragEnded,
								Start: ts.startPos,
								Pos:   pos,
								Delta: input.GesturePoint{X: pos.X - ts.lastPos.X, Y: pos.Y - ts.lastPos.Y},
							}),
						},
						{
							pos: ts.startPos,
							event: SwipeEvent(input.SwipeMsg{
								Direction: swipeDirection(ts.startPos, pos),
								Velocity:  velocity,
								Start:     ts.startPos,
								End:       pos,
							}),
						},
					}
				}
			}
		}

		return []gestureResult{{
			pos: pos,
			event: DragEvent(input.DragMsg{
				Phase: input.DragEnded,
				Start: ts.startPos,
				Pos:   pos,
				Delta: input.GesturePoint{X: pos.X - ts.lastPos.X, Y: pos.Y - ts.lastPos.Y},
			}),
		}}
	}

	return nil
}

func (g *GestureRecognizer) handleCancelled(t *input.TouchMsg) []gestureResult {
	ts, ok := g.touches[t.ID]
	if !ok {
		return nil
	}
	defer delete(g.touches, t.ID)

	// If pinch was active and this touch is cancelled, cancel the pinch.
	if g.pinchActive && (t.ID == g.pinchIDs[0] || t.ID == g.pinchIDs[1]) {
		g.pinchActive = false
		return []gestureResult{{
			pos: ts.startPos,
			event: PinchEvent(input.PinchMsg{
				Phase: input.PinchCancelled,
			}),
		}}
	}

	switch ts.phase {
	case touchLongPress:
		return []gestureResult{{
			pos: ts.startPos,
			event: LongPressEvent(input.LongPressMsg{
				Pos:   ts.startPos,
				Phase: input.LongPressCancelled,
			}),
		}}
	case touchDragging:
		return []gestureResult{{
			pos: ts.lastPos,
			event: DragEvent(input.DragMsg{
				Phase: input.DragCancelled,
				Start: ts.startPos,
				Pos:   ts.lastPos,
			}),
		}}
	}

	return nil
}

// ── Tap emission with double-tap detection ───────────────────────

func (g *GestureRecognizer) emitTap(pos input.GesturePoint, now time.Time) []gestureResult {
	count := 1
	if g.config.DoubleTapInterval > 0 {
		elapsed := now.Sub(g.lastTapTime)
		dist := distance(g.lastTapPos, pos)
		if elapsed <= g.config.DoubleTapInterval && dist <= g.config.DragThreshold {
			g.tapCount++
			count = g.tapCount
		} else {
			g.tapCount = 1
		}
	}
	g.lastTapPos = pos
	g.lastTapTime = now

	return []gestureResult{{
		pos: pos,
		event: TapEvent(input.TapMsg{
			Pos:   pos,
			Count: count,
		}),
	}}
}

// ── Pinch detection ──────────────────────────────────────────────

func (g *GestureRecognizer) detectPinchStart(now time.Time) {
	if len(g.touches) < 2 {
		return
	}

	// Find the two oldest pending/active touches.
	var ids [2]int64
	var oldest [2]time.Time
	found := 0
	for id, ts := range g.touches {
		if ts.rejected || ts.phase == touchConsumed {
			continue
		}
		if found < 2 {
			ids[found] = id
			oldest[found] = ts.startTime
			found++
		} else {
			// Replace the newest of the two if this one is older.
			for i := 0; i < 2; i++ {
				if ts.startTime.Before(oldest[i]) {
					ids[i] = id
					oldest[i] = ts.startTime
					break
				}
			}
		}
	}

	if found < 2 {
		return
	}

	t0 := g.touches[ids[0]]
	t1 := g.touches[ids[1]]

	g.pinchIDs = ids
	g.pinchActive = true
	g.pinchStartDist = distance(t0.lastPos, t1.lastPos)
	if g.pinchStartDist < 1 {
		g.pinchStartDist = 1 // avoid division by zero
	}

	// Cancel any in-progress gestures on these touches.
	t0.phase = touchConsumed
	t1.phase = touchConsumed
	// PinchBegan is emitted on the first updatePinch call.
}

func (g *GestureRecognizer) updatePinch(now time.Time) []gestureResult {
	t0, ok0 := g.touches[g.pinchIDs[0]]
	t1, ok1 := g.touches[g.pinchIDs[1]]
	if !ok0 || !ok1 {
		return g.endPinch()
	}

	currentDist := distance(t0.lastPos, t1.lastPos)
	scale := currentDist / g.pinchStartDist
	center := midpoint(t0.lastPos, t1.lastPos)

	return []gestureResult{{
		pos: center,
		event: PinchEvent(input.PinchMsg{
			Phase:  input.PinchChanged,
			Center: center,
			Scale:  scale,
		}),
	}}
}

func (g *GestureRecognizer) endPinch() []gestureResult {
	if !g.pinchActive {
		return nil
	}

	t0 := g.touches[g.pinchIDs[0]]
	t1 := g.touches[g.pinchIDs[1]]

	var center input.GesturePoint
	var scale float32 = 1.0
	if t0 != nil && t1 != nil {
		currentDist := distance(t0.lastPos, t1.lastPos)
		scale = currentDist / g.pinchStartDist
		center = midpoint(t0.lastPos, t1.lastPos)
	} else if t0 != nil {
		center = t0.lastPos
	} else if t1 != nil {
		center = t1.lastPos
	}

	g.pinchActive = false

	return []gestureResult{{
		pos: center,
		event: PinchEvent(input.PinchMsg{
			Phase:  input.PinchEnded,
			Center: center,
			Scale:  scale,
		}),
	}}
}

// ── Palm rejection (§3.5) ────────────────────────────────────────

func (g *GestureRecognizer) isPalmRejected(t *input.TouchMsg, now time.Time) bool {
	// Rule 1: High force → palm.
	if t.Force > 0.8 {
		if ts, ok := g.touches[t.ID]; ok {
			ts.rejected = true
		}
		return true
	}

	// Rule 2: Touch at screen edge (<10dp from edge).
	if g.config.ScreenWidth > 0 && g.config.ScreenHeight > 0 {
		const edgeMargin float32 = 10
		if t.X < edgeMargin || t.Y < edgeMargin ||
			t.X > g.config.ScreenWidth-edgeMargin ||
			t.Y > g.config.ScreenHeight-edgeMargin {
			// For edge touches during TouchBegan, check if there are
			// active touches in the main area — if so, reject this one.
			if t.Phase == input.TouchBegan {
				for _, ts := range g.touches {
					if !ts.rejected && ts.id != t.ID {
						// Existing touch in main area — reject edge touch.
						return true
					}
				}
			}
		}
	}

	// Rule 3: New touch within 50ms and >100dp from an existing touch.
	if t.Phase == input.TouchBegan {
		pos := input.GesturePoint{X: t.X, Y: t.Y}
		for _, ts := range g.touches {
			if ts.rejected || ts.id == t.ID {
				continue
			}
			elapsed := now.Sub(ts.startTime)
			dist := distance(pos, ts.lastPos)
			if elapsed <= 50*time.Millisecond && dist > 100 {
				return true
			}
		}
	}

	return false
}

// ── Debounce check ───────────────────────────────────────────────

// ShouldDebounce returns true if a TapMsg to the given UID should be
// suppressed because it's too soon after the last tap (§3.6).
func (g *GestureRecognizer) ShouldDebounce(uid UID, now time.Time) bool {
	if g.config.DebounceInterval <= 0 {
		return false
	}
	if last, ok := g.lastTapPerUID[uid]; ok {
		if now.Sub(last) < g.config.DebounceInterval {
			return true
		}
	}
	g.lastTapPerUID[uid] = now
	return false
}

// RecordTap records a tap time for debounce tracking.
func (g *GestureRecognizer) RecordTap(uid UID, now time.Time) {
	g.lastTapPerUID[uid] = now
}

// ── Helpers ──────────────────────────────────────────────────────

func distance(a, b input.GesturePoint) float32 {
	dx := float64(b.X - a.X)
	dy := float64(b.Y - a.Y)
	return float32(math.Sqrt(dx*dx + dy*dy))
}

func midpoint(a, b input.GesturePoint) input.GesturePoint {
	return input.GesturePoint{
		X: (a.X + b.X) / 2,
		Y: (a.Y + b.Y) / 2,
	}
}

func swipeDirection(start, end input.GesturePoint) input.SwipeDirection {
	dx := end.X - start.X
	dy := end.Y - start.Y

	// Determine primary axis.
	absDx := dx
	if absDx < 0 {
		absDx = -absDx
	}
	absDy := dy
	if absDy < 0 {
		absDy = -absDy
	}

	if absDx >= absDy {
		if dx < 0 {
			return input.SwipeLeft
		}
		return input.SwipeRight
	}
	if dy < 0 {
		return input.SwipeUp
	}
	return input.SwipeDown
}
