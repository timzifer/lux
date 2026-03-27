// Package anim provides deterministic, testable animations for the
// lux UI toolkit (RFC §12).
//
// Animations run within the app loop — no goroutines, no timers.
// The framework calls Tick(dt) each frame; tests inject dt directly.
package anim

import (
	"math"
	"time"
)

// Interpolatable constrains the types that Anim[T] can animate.
// Only numeric types for now; draw.Color/Point/etc. will be added
// via a Lerper[T] pattern when needed (anim/ depends only on stdlib).
type Interpolatable interface {
	~float32 | ~float64
}

// ── Easing Functions ────────────────────────────────────────────

// EasingFunc maps a normalized time t ∈ [0,1] to an output ∈ [0,1].
type EasingFunc func(t float32) float32

// Built-in easing functions (RFC §12.6).
var (
	Linear     EasingFunc = func(t float32) float32 { return t }
	OutCubic   EasingFunc = func(t float32) float32 { t--; return 1 + t*t*t }
	InCubic    EasingFunc = func(t float32) float32 { return t * t * t }
	InOutCubic EasingFunc = func(t float32) float32 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		t = 2*t - 2
		return 0.5*t*t*t + 1
	}
	OutExpo EasingFunc = func(t float32) float32 {
		if t >= 1 {
			return 1
		}
		return 1 - float32(math.Pow(2, float64(-10*t)))
	}
)

// CubicBezier returns a CSS-compatible cubic-bezier easing function.
// The two control points (x1,y1) and (x2,y2) define the curve shape,
// just like CSS transition-timing-function: cubic-bezier(x1,y1,x2,y2).
// Uses Newton-Raphson iteration to solve for the parametric t given input x.
func CubicBezier(x1, y1, x2, y2 float32) EasingFunc {
	return func(x float32) float32 {
		if x <= 0 {
			return 0
		}
		if x >= 1 {
			return 1
		}
		// Newton-Raphson: find parametric t where bezierX(t) == x.
		t := x // initial guess
		for i := 0; i < 8; i++ {
			bx := cubicBezierSample(t, x1, x2) - x
			if bx > -1e-6 && bx < 1e-6 {
				break
			}
			dx := cubicBezierDerivative(t, x1, x2)
			if dx < 1e-6 && dx > -1e-6 {
				break
			}
			t -= bx / dx
		}
		// Clamp t to [0,1].
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		return cubicBezierSample(t, y1, y2)
	}
}

// cubicBezierSample evaluates a 1D cubic bezier at parameter t.
// Control points: P0=0, P1=p1, P2=p2, P3=1.
func cubicBezierSample(t, p1, p2 float32) float32 {
	// B(t) = 3(1-t)^2*t*p1 + 3(1-t)*t^2*p2 + t^3
	omt := 1 - t
	return 3*omt*omt*t*p1 + 3*omt*t*t*p2 + t*t*t
}

// cubicBezierDerivative returns dB/dt for the 1D cubic bezier.
func cubicBezierDerivative(t, p1, p2 float32) float32 {
	// B'(t) = 3(1-t)^2*p1 + 6(1-t)*t*(p2-p1) + 3*t^2*(1-p2)
	omt := 1 - t
	return 3*omt*omt*p1 + 6*omt*t*(p2-p1) + 3*t*t*(1-p2)
}

// ── Tickable Interface ────────────────────────────────────────
//
// Tickable is the common interface for anything that can be ticked
// by the animation system (Anim, SpringAnim, AnimGroup, AnimSeq).

// Tickable is implemented by all animation types.
type Tickable interface {
	Tick(dt time.Duration) bool
	IsDone() bool
}

// ── AnimationID (RFC-002 §1.8) ──────────────────────────────────

// AnimationID is a typed string for user-initiated animation completion events.
type AnimationID string

// AnimationEnded is sent via SendFunc when a Tier-2 animation completes.
type AnimationEnded struct {
	ID AnimationID
}

// SendFunc is set by the app package to enable anim → app.Send
// without circular imports. Nil until the app loop starts.
var SendFunc func(msg any)

// ── Anim[T] ────────────────────────────────────────────────────

// Anim is a deterministic, interpolated animation value (RFC §12.4).
// The zero value is immediately done with the zero value of T.
type Anim[T Interpolatable] struct {
	from        T
	current     T
	to          T
	elapsed     time.Duration
	duration    time.Duration
	easing      EasingFunc
	running     bool
	animID      AnimationID
	notifyOnEnd bool
}

// Value returns the current interpolated value.
func (a *Anim[T]) Value() T { return a.current }

// SetTarget starts a new animation from the current value to `to`.
// If an animation is already running, it continues from the current
// interpolated value (no snap-to-start).
func (a *Anim[T]) SetTarget(to T, dur time.Duration, easing EasingFunc) {
	if easing == nil {
		easing = Linear
	}
	a.from = a.current
	a.to = to
	a.elapsed = 0
	a.duration = dur
	a.easing = easing
	a.running = true
}

// SetTargetWithID starts a new animation and sends AnimationEnded{ID}
// via SendFunc when the animation completes (RFC-002 §1.8).
// Re-calling with the same ID replaces the pending notification.
func (a *Anim[T]) SetTargetWithID(to T, dur time.Duration, easing EasingFunc, id AnimationID) {
	a.SetTarget(to, dur, easing)
	a.animID = id
	a.notifyOnEnd = true
}

// SetImmediate snaps the value without animation (RFC §12.4).
func (a *Anim[T]) SetImmediate(v T) {
	a.from = v
	a.current = v
	a.to = v
	a.elapsed = 0
	a.duration = 0
	a.easing = nil
	a.running = false
}

// IsDone reports whether the animation has completed or was never started.
func (a *Anim[T]) IsDone() bool { return !a.running }

// Tick advances the animation by dt. Returns true if still running.
// Called by the framework via the Animator interface — user code
// normally calls this only in tests.
func (a *Anim[T]) Tick(dt time.Duration) bool {
	if !a.running {
		return false
	}

	a.elapsed += dt

	if a.elapsed >= a.duration {
		// Snap to exact target — no floating-point drift.
		a.current = a.to
		a.running = false
		if a.notifyOnEnd && SendFunc != nil {
			a.notifyOnEnd = false
			SendFunc(AnimationEnded{ID: a.animID})
		}
		return false
	}

	t := float32(a.elapsed) / float32(a.duration)
	if a.easing != nil {
		t = a.easing(t)
	}
	a.current = lerp(a.from, a.to, t)
	return true
}

// lerp linearly interpolates between a and b by t ∈ [0,1].
func lerp[T Interpolatable](a, b T, t float32) T {
	return a + T(float32(b-a)*t)
}

// ── LerpFunc & LerpAnim[T] (RFC-002 §1.4) ────────────────────────
//
// LerpAnim animates arbitrary types via a caller-supplied LerpFunc.
// This avoids a cyclic dependency between anim/ and draw/ — the
// concrete lerpers for Color, Point, etc. live in draw/anim (or the
// caller's package).

// LerpFunc interpolates between a and b by t ∈ [0,1].
type LerpFunc[T any] func(a, b T, t float32) T

// LerpAnim is a deterministic, interpolated animation for any type T.
// The caller must supply a LerpFunc when calling SetTarget.
type LerpAnim[T any] struct {
	from     T
	current  T
	to       T
	elapsed  time.Duration
	duration time.Duration
	easing   EasingFunc
	lerpFn   LerpFunc[T]
	running  bool
}

// Value returns the current interpolated value.
func (a *LerpAnim[T]) Value() T { return a.current }

// SetTarget starts a new animation from the current value to `to`.
func (a *LerpAnim[T]) SetTarget(to T, dur time.Duration, easing EasingFunc, lerpFn LerpFunc[T]) {
	if easing == nil {
		easing = Linear
	}
	a.from = a.current
	a.to = to
	a.elapsed = 0
	a.duration = dur
	a.easing = easing
	a.lerpFn = lerpFn
	a.running = true
}

// SetImmediate snaps the value without animation.
func (a *LerpAnim[T]) SetImmediate(v T) {
	a.from = v
	a.current = v
	a.to = v
	a.elapsed = 0
	a.duration = 0
	a.easing = nil
	a.running = false
}

// IsDone reports whether the animation has completed or was never started.
func (a *LerpAnim[T]) IsDone() bool { return !a.running }

// Tick advances the animation by dt. Returns true if still running.
func (a *LerpAnim[T]) Tick(dt time.Duration) bool {
	if !a.running {
		return false
	}
	a.elapsed += dt
	if a.elapsed >= a.duration {
		a.current = a.to
		a.running = false
		return false
	}
	t := float32(a.elapsed) / float32(a.duration)
	if a.easing != nil {
		t = a.easing(t)
	}
	if a.lerpFn != nil {
		a.current = a.lerpFn(a.from, a.to, t)
	}
	return true
}
