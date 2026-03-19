package anim

import "time"

// SpringSpec defines a spring-damper system (RFC-002 §1.5).
type SpringSpec struct {
	Stiffness         float32 // Spring force. Higher = faster settling. 100–800.
	Damping           float32 // Damping force. Critical ≈ 2*sqrt(Stiffness*Mass).
	Mass              float32 // Inertial mass. Typically 1.0.
	SettlingThreshold float32 // Below this velocity+distance the spring is done.
}

// Preset springs aligned with MotionSpec tokens.
var (
	SpringGentle = SpringSpec{Stiffness: 120, Damping: 14, Mass: 1.0, SettlingThreshold: 0.001}
	SpringSnappy = SpringSpec{Stiffness: 400, Damping: 28, Mass: 1.0, SettlingThreshold: 0.001}
	SpringBouncy = SpringSpec{Stiffness: 200, Damping: 10, Mass: 1.0, SettlingThreshold: 0.001}
)

// SpringAnim simulates a spring-damper system for numeric types.
// No fixed duration — converges asymptotically to the target value.
type SpringAnim[T Interpolatable] struct {
	current  T
	velocity T
	target   T
	spec     SpringSpec
	running  bool
}

// Value returns the current spring value.
func (s *SpringAnim[T]) Value() T { return s.current }

// IsDone reports whether the spring has settled.
func (s *SpringAnim[T]) IsDone() bool { return !s.running }

// SetTarget sets a new target value using the current SpringSpec.
// If no spec has been set, SpringGentle is used as default.
func (s *SpringAnim[T]) SetTarget(to T) {
	s.target = to
	if s.spec.Stiffness == 0 && s.spec.Damping == 0 && s.spec.Mass == 0 {
		s.spec = SpringGentle
	}
	s.running = true
}

// SetTargetWithSpec sets a new target and overrides the spring spec.
func (s *SpringAnim[T]) SetTargetWithSpec(to T, spec SpringSpec) {
	s.spec = spec
	s.target = to
	s.running = true
}

// SetImmediate snaps the value without animation.
func (s *SpringAnim[T]) SetImmediate(v T) {
	s.current = v
	s.target = v
	s.velocity = 0
	s.running = false
}

// Tick advances the spring simulation by dt using semi-implicit Euler.
// Returns true if the spring is still running.
func (s *SpringAnim[T]) Tick(dt time.Duration) bool {
	if !s.running {
		return false
	}

	dtSec := float32(dt.Seconds())
	if dtSec <= 0 {
		return s.running
	}

	mass := s.spec.Mass
	if mass <= 0 {
		mass = 1.0
	}
	threshold := s.spec.SettlingThreshold
	if threshold <= 0 {
		threshold = 0.001
	}

	// Spring force: F = -k*(x - target) - d*v
	displacement := float32(s.current - s.target)
	vel := float32(s.velocity)
	force := -s.spec.Stiffness*displacement - s.spec.Damping*vel
	accel := force / mass

	// Semi-implicit Euler: update velocity first, then position.
	vel += accel * dtSec
	pos := float32(s.current) + vel*dtSec

	s.velocity = T(vel)
	s.current = T(pos)

	// Check settling: both velocity and displacement are small.
	newDisplacement := float32(s.current - s.target)
	if newDisplacement < 0 {
		newDisplacement = -newDisplacement
	}
	absVel := vel
	if absVel < 0 {
		absVel = -absVel
	}

	if newDisplacement < threshold && absVel < threshold {
		s.current = s.target
		s.velocity = 0
		s.running = false
		return false
	}

	return true
}
