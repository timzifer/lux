// Package ui — kinetic_scroll.go implements physics-based scrolling (RFC-002 §3).
//
// KineticScroll manages scroll position, velocity, friction decay,
// overscroll rubber-banding, and snap-to animations. It is intended
// to live in WidgetState and be ticked by the framework via Animator.
package ui

import (
	"math"
	"time"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

// scrollPhase tracks the current state of the kinetic scroll state machine.
type scrollPhase uint8

const (
	scrollIdle         scrollPhase = iota
	scrollTracking     // finger/trackpad actively driving position
	scrollDecelerating // finger released, friction decay running
	scrollSnapping     // rubber-band spring back into bounds
)

// scrollBounds defines the valid scroll range on one axis.
type scrollBounds struct {
	Min, Max float32
}

// velocitySample records a single delta for velocity estimation.
type velocitySample struct {
	delta float32
}

// frameTarget is the reference frame duration for friction exponent normalization.
const frameTarget = 16 * time.Millisecond

// maxVelocitySamples is the sliding window size for trackpad velocity estimation.
const maxVelocitySamples = 5

// KineticScroll manages the complete scroll state of a scrollable widget:
// position, velocity, overscroll and rubber-band spring (RFC-002 §3.3).
// It lives in WidgetState and can be ticked via the Animator interface.
type KineticScroll struct {
	// Current scroll offset in dp. Read publicly, never write directly.
	OffsetX float32
	OffsetY float32

	velX, velY       float32
	phase            scrollPhase
	spec             theme.ScrollSpec
	boundsX, boundsY scrollBounds

	// Velocity tracking for trackpad (sliding window of recent deltas).
	recentDeltasY []velocitySample

	// feedThisFrame tracks whether Feed was called during the current frame.
	// Used to detect end-of-tracking (trackpad finger lifted).
	feedThisFrame bool
}

// NewKineticScroll creates a KineticScroll with the given scroll physics.
func NewKineticScroll(spec theme.ScrollSpec) *KineticScroll {
	return &KineticScroll{
		spec: spec,
	}
}

// SetBounds defines the scrollable range.
// minY is typically 0, maxY is contentHeight - viewportHeight.
// Negative max means content is smaller than viewport (no scrolling).
func (k *KineticScroll) SetBounds(minX, maxX, minY, maxY float32) {
	k.boundsX = scrollBounds{Min: minX, Max: maxX}
	k.boundsY = scrollBounds{Min: minY, Max: maxY}
}

// Feed processes a ScrollMsg from the input system.
// Must be called for every ScrollMsg targeted at this scroll container.
func (k *KineticScroll) Feed(msg input.ScrollMsg) {
	k.feedThisFrame = true

	if msg.Precise {
		// Trackpad: direct position control with velocity tracking.
		k.phase = scrollTracking
		delta := -msg.DeltaY * k.spec.MultiplierPrecise

		// Apply rubber-band damping when beyond bounds.
		if k.isOverscrolledY() {
			delta *= 0.3 // dampen input in overscroll zone
		}

		k.OffsetY += delta

		// Clamp overscroll to max displacement.
		k.OffsetY = clampOverscroll(k.OffsetY, k.boundsY, k.spec.Overscroll)

		// Track velocity from recent deltas.
		k.recentDeltasY = append(k.recentDeltasY, velocitySample{delta: delta})
		if len(k.recentDeltasY) > maxVelocitySamples {
			k.recentDeltasY = k.recentDeltasY[1:]
		}
	} else {
		// Mouse wheel: discrete step, no kinematics.
		step := k.spec.StepSize
		if msg.DeltaY > 0 {
			k.OffsetY -= step
		} else if msg.DeltaY < 0 {
			k.OffsetY += step
		}
		// Clamp to bounds (no overscroll for discrete steps).
		k.OffsetY = clampToBounds(k.OffsetY, k.boundsY)
		k.phase = scrollIdle
		k.velY = 0
	}
}

// Tick advances the scroll physics by dt. Returns true if still animating.
// This follows the Animator pattern (RFC-002 §1.3).
func (k *KineticScroll) Tick(dt time.Duration) bool {
	defer func() { k.feedThisFrame = false }()

	switch k.phase {
	case scrollIdle:
		return false

	case scrollTracking:
		// If no Feed this frame, the finger was lifted → start decelerating.
		if !k.feedThisFrame {
			k.velY = k.estimateVelocityY()
			k.recentDeltasY = k.recentDeltasY[:0]
			if k.isOverscrolledY() {
				k.phase = scrollSnapping
			} else {
				k.phase = scrollDecelerating
			}
		}
		return true

	case scrollDecelerating:
		// Friction decay: v *= friction^(dt/frameTarget)
		exponent := float32(dt) / float32(frameTarget)
		k.velY *= float32(math.Pow(float64(k.spec.Friction), float64(exponent)))
		k.OffsetY += k.velY

		// Check if we've crossed bounds → switch to snapping.
		if k.isOverscrolledY() {
			k.phase = scrollSnapping
			return true
		}

		// Check settling threshold.
		if abs32(k.velY) < k.spec.SettlingThreshold {
			k.velY = 0
			k.OffsetY = clampToBounds(k.OffsetY, k.boundsY)
			k.phase = scrollIdle
			return false
		}
		return true

	case scrollSnapping:
		// Spring-based rubber-band back to valid bounds.
		target := clampToBounds(k.OffsetY, k.boundsY)
		displacement := k.OffsetY - target

		// Spring physics: F = -stiffness * x - damping * v
		const stiffness float32 = 300
		const damping float32 = 30
		dtSec := float32(dt.Seconds())

		force := -stiffness*displacement - damping*k.velY
		k.velY += force * dtSec
		k.OffsetY += k.velY * dtSec

		// Check if settled.
		newDisplacement := k.OffsetY - target
		if abs32(newDisplacement) < 0.5 && abs32(k.velY) < k.spec.SettlingThreshold {
			k.OffsetY = target
			k.velY = 0
			k.phase = scrollIdle
			return false
		}
		return true
	}

	return false
}

// SnapTo programmatically scrolls to an offset with spring animation.
func (k *KineticScroll) SnapTo(_, y float32) {
	// Keep current position; the spring in Tick will pull toward the target.
	// We temporarily set bounds so the snap target becomes the valid range.
	k.velY = (y - k.OffsetY) * 10 // initial impulse toward target
	k.boundsY = scrollBounds{Min: y, Max: y}
	k.phase = scrollSnapping
}

// SnapToImmediate sets the scroll position without animation.
func (k *KineticScroll) SnapToImmediate(x, y float32) {
	k.OffsetX = x
	k.OffsetY = y
	k.velX = 0
	k.velY = 0
	k.phase = scrollIdle
}

// IsDone reports whether no scroll animation is running.
func (k *KineticScroll) IsDone() bool {
	return k.phase == scrollIdle
}

// Phase returns the current scroll phase (for testing/debugging).
func (k *KineticScroll) Phase() scrollPhase {
	return k.phase
}

// estimateVelocityY computes the average velocity from recent trackpad deltas.
func (k *KineticScroll) estimateVelocityY() float32 {
	if len(k.recentDeltasY) == 0 {
		return 0
	}
	var sum float32
	for _, s := range k.recentDeltasY {
		sum += s.delta
	}
	return sum / float32(len(k.recentDeltasY))
}

// isOverscrolledY reports whether the Y offset is beyond valid bounds.
func (k *KineticScroll) isOverscrolledY() bool {
	return k.OffsetY < k.boundsY.Min || k.OffsetY > k.boundsY.Max
}

// clampToBounds clamps v to [b.Min, b.Max].
func clampToBounds(v float32, b scrollBounds) float32 {
	if b.Max < b.Min {
		return b.Min // content smaller than viewport
	}
	if v < b.Min {
		return b.Min
	}
	if v > b.Max {
		return b.Max
	}
	return v
}

// clampOverscroll limits how far beyond bounds the offset can go.
func clampOverscroll(v float32, b scrollBounds, maxOverscroll float32) float32 {
	if v < b.Min-maxOverscroll {
		return b.Min - maxOverscroll
	}
	if v > b.Max+maxOverscroll {
		return b.Max + maxOverscroll
	}
	return v
}

// abs32 returns the absolute value of a float32.
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
