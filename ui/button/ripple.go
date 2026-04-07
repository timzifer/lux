package button

import (
	"math"
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
)

// ── Waterdrop Ripple Effect (RFC-004 §4.1) ─────────────────────
//
// A sonar-ping style ripple: a ring expands outward from the touch
// point and fades to transparent. Two concentric rings with a slight
// delay create the "ping" feel.

const (
	rippleExpandDur = 500 * time.Millisecond
	rippleFadeDur   = 650 * time.Millisecond
	rippleRing2Delay = 80 * time.Millisecond // second ring starts later
	rippleStrokeW   = 2.5                    // ring stroke width (dp)
	rippleMaxAlpha  = float32(0.55)          // peak ring opacity
)

// RippleState holds animation state for a single waterdrop ripple.
// Allocate once, store in your Model, and call Tick each frame.
type RippleState struct {
	// Touch origin in absolute (screen) coordinates, set by Trigger.
	cx, cy float32

	// Ring 1 — starts immediately.
	ring1Radius  anim.Anim[float32]
	ring1Opacity anim.Anim[float32]

	// Ring 2 — delayed start for sonar-ping feel.
	ring2Radius  anim.Anim[float32]
	ring2Opacity anim.Anim[float32]
	ring2Delay   time.Duration // counts down to 0 then starts

	active bool
}

// Active reports whether the ripple animation is still running.
func (rs *RippleState) Active() bool { return rs.active }

// NewRippleState creates a ready-to-use RippleState.
func NewRippleState() *RippleState { return &RippleState{} }

// Trigger starts a new ripple at (cx, cy) expanding to maxRadius.
func (rs *RippleState) Trigger(cx, cy, maxRadius float32) {
	rs.cx = cx
	rs.cy = cy
	rs.active = true

	// Ring 1: expand + fade.
	rs.ring1Radius.SetImmediate(0)
	rs.ring1Radius.SetTarget(maxRadius, rippleExpandDur, anim.OutCubic)
	rs.ring1Opacity.SetImmediate(rippleMaxAlpha)
	rs.ring1Opacity.SetTarget(0, rippleFadeDur, anim.Linear)

	// Ring 2: delayed.
	rs.ring2Delay = rippleRing2Delay
	rs.ring2Radius.SetImmediate(0)
	rs.ring2Opacity.SetImmediate(0)
}

// Tick advances all ripple animations. Returns true if still animating.
func (rs *RippleState) Tick(dt time.Duration) bool {
	if !rs.active {
		return false
	}

	r1 := rs.ring1Radius.Tick(dt)
	o1 := rs.ring1Opacity.Tick(dt)

	var r2, o2 bool
	if rs.ring2Delay > 0 {
		rs.ring2Delay -= dt
		if rs.ring2Delay <= 0 {
			// Start ring 2.
			maxR := rs.ring1Radius.Value() + (rs.ring1Radius.Value() * 0.3) // slightly larger target
			// Use the same max radius as ring 1 would reach.
			// Re-derive from ring1's target by reading the anim internals is
			// not possible, so we estimate: ring1 is partially expanded, final
			// radius ≈ current * (expandDur / elapsed). Simpler: just use a
			// fixed fraction of the button diagonal passed via Trigger.
			rs.ring2Radius.SetImmediate(0)
			if maxR < 20 {
				maxR = 80 // fallback
			}
			rs.ring2Radius.SetTarget(maxR, rippleExpandDur, anim.OutCubic)
			rs.ring2Opacity.SetImmediate(rippleMaxAlpha * 0.5)
			rs.ring2Opacity.SetTarget(0, rippleFadeDur, anim.Linear)
		}
		r2, o2 = true, true // still pending
	} else {
		r2 = rs.ring2Radius.Tick(dt)
		o2 = rs.ring2Opacity.Tick(dt)
	}

	running := r1 || o1 || r2 || o2
	if !running {
		rs.active = false
	}
	return running
}

// Draw renders the ripple rings on the canvas, clipped to clipRect.
func (rs *RippleState) Draw(canvas draw.Canvas, clipRect draw.Rect, clipRadius float32, accent draw.Color) {
	if !rs.active {
		return
	}

	canvas.PushClipRoundRect(clipRect, clipRadius)
	defer canvas.PopClip()

	rs.drawRing(canvas, rs.ring1Radius.Value(), rs.ring1Opacity.Value(), accent)
	if rs.ring2Delay <= 0 {
		rs.drawRing(canvas, rs.ring2Radius.Value(), rs.ring2Opacity.Value(), accent)
	}
}

func (rs *RippleState) drawRing(canvas draw.Canvas, radius, opacity float32, accent draw.Color) {
	if opacity <= 0 || radius <= 1 {
		return
	}

	ringColor := draw.Color{R: accent.R, G: accent.G, B: accent.B, A: accent.A * opacity}

	// Build a circular arc path (full circle) at (cx, cy) with given radius.
	// SVG-style: two semicircular arcs.
	cx, cy, r := rs.cx, rs.cy, radius
	p := draw.NewPath().
		MoveTo(draw.Pt(cx-r, cy)).
		ArcTo(r, r, 0, false, true, draw.Pt(cx+r, cy)).
		ArcTo(r, r, 0, false, true, draw.Pt(cx-r, cy)).
		Close().
		Build()

	canvas.StrokePath(p, draw.Stroke{
		Paint: draw.SolidPaint(ringColor),
		Width: float32(rippleStrokeW),
	})
}

// maxRippleRadius computes a good maximum radius for a ripple originating
// at (cx, cy) within a rectangle, so the ring reaches all corners.
func maxRippleRadius(cx, cy, x, y, w, h float32) float32 {
	// Distance to the farthest corner.
	dx0, dy0 := cx-x, cy-y
	dx1, dy1 := cx-(x+w), cy-(y+h)
	d1 := float32(math.Sqrt(float64(dx0*dx0 + dy0*dy0)))
	d2 := float32(math.Sqrt(float64(dx1*dx1 + dy0*dy0)))
	d3 := float32(math.Sqrt(float64(dx0*dx0 + dy1*dy1)))
	d4 := float32(math.Sqrt(float64(dx1*dx1 + dy1*dy1)))
	m := d1
	if d2 > m {
		m = d2
	}
	if d3 > m {
		m = d3
	}
	if d4 > m {
		m = d4
	}
	return m
}
