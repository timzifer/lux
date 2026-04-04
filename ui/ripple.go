package ui

import (
	"math"
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
)

// ── Filled Radial Pulse (RFC-004 §4.1) ─────────────────────────
//
// A filled disc expands from the touch point and fades out, giving
// clear visual confirmation of touch input on HMI displays.

const (
	RippleExpandDur = 400 * time.Millisecond // disc expansion duration
	RippleFadeDur   = 500 * time.Millisecond // fade-out duration
	RippleMaxAlpha  = float32(0.35)          // peak disc opacity
)

// RippleState holds animation state for a single radial pulse.
type RippleState struct {
	// Touch origin in absolute (screen) coordinates, set by Trigger.
	cx, cy float32

	radius  anim.Anim[float32]
	opacity anim.Anim[float32]

	Active bool
}

// Trigger starts a new ripple at (cx, cy) expanding to maxRadius.
func (rs *RippleState) Trigger(cx, cy, maxRadius float32) {
	rs.cx = cx
	rs.cy = cy
	rs.Active = true

	rs.radius.SetImmediate(0)
	rs.radius.SetTarget(maxRadius, RippleExpandDur, anim.OutCubic)
	rs.opacity.SetImmediate(RippleMaxAlpha)
	rs.opacity.SetTarget(0, RippleFadeDur, anim.OutCubic)
}

// Tick advances all ripple animations. Returns true if still animating.
func (rs *RippleState) Tick(dt time.Duration) bool {
	if !rs.Active {
		return false
	}

	r := rs.radius.Tick(dt)
	o := rs.opacity.Tick(dt)

	running := r || o
	if !running {
		rs.Active = false
	}
	return running
}

// Draw renders the ripple on the canvas, clipped to clipRect.
func (rs *RippleState) Draw(canvas draw.Canvas, clipRect draw.Rect, clipRadius float32, color draw.Color) {
	if !rs.Active {
		return
	}

	radius := rs.radius.Value()
	opacity := rs.opacity.Value()
	if opacity <= 0 || radius <= 1 {
		return
	}

	canvas.PushClipRoundRect(clipRect, clipRadius)
	defer canvas.PopClip()

	fillColor := draw.Color{R: color.R, G: color.G, B: color.B, A: color.A * opacity}

	cx, cy, r := rs.cx, rs.cy, radius
	p := draw.NewPath().
		MoveTo(draw.Pt(cx-r, cy)).
		ArcTo(r, r, 0, false, true, draw.Pt(cx+r, cy)).
		ArcTo(r, r, 0, false, true, draw.Pt(cx-r, cy)).
		Close().
		Build()

	canvas.FillPath(p, draw.SolidPaint(fillColor))
}

// MaxRippleRadius computes a good maximum radius for a ripple originating
// at (cx, cy) within a rectangle, so the disc reaches all corners.
func MaxRippleRadius(cx, cy, x, y, w, h float32) float32 {
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
