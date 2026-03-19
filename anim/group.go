package anim

import "time"

// ── AnimGroup (RFC-002 §1.9) ──────────────────────────────────
//
// AnimGroup ticks multiple animations in parallel.
// IsDone() returns true when all animations are done.

// AnimGroup runs animations in parallel.
type AnimGroup struct {
	anims []Tickable
}

// NewAnimGroup creates a group from the given animations.
func NewAnimGroup(anims ...Tickable) *AnimGroup {
	return &AnimGroup{anims: anims}
}

// Add appends an animation to the group.
func (g *AnimGroup) Add(a Tickable) {
	g.anims = append(g.anims, a)
}

// Tick advances all animations by dt. Returns true if any are still running.
func (g *AnimGroup) Tick(dt time.Duration) bool {
	anyRunning := false
	for _, a := range g.anims {
		if a.Tick(dt) {
			anyRunning = true
		}
	}
	return anyRunning
}

// IsDone returns true when all animations are done.
func (g *AnimGroup) IsDone() bool {
	for _, a := range g.anims {
		if !a.IsDone() {
			return false
		}
	}
	return true
}

// ── AnimSeq (RFC-002 §1.9) ───────────────────────────────────
//
// AnimSeq plays animations sequentially. The next step starts
// when the current one is done. Optional onDone hooks run between steps.

type seqStep struct {
	anim   Tickable
	onDone func()
}

// AnimSeq plays animations one after another.
type AnimSeq struct {
	steps   []seqStep
	current int
	started bool
}

// NewAnimSeq creates an empty sequence.
func NewAnimSeq() *AnimSeq {
	return &AnimSeq{}
}

// Then appends an animation step with an optional onDone hook.
// Returns the sequence for chaining.
func (s *AnimSeq) Then(a Tickable, onDone ...func()) *AnimSeq {
	var hook func()
	if len(onDone) > 0 {
		hook = onDone[0]
	}
	s.steps = append(s.steps, seqStep{anim: a, onDone: hook})
	return s
}

// Tick advances the current animation by dt. When the current step
// finishes, its onDone hook is called and the next step begins.
// Returns true if the sequence is still running.
func (s *AnimSeq) Tick(dt time.Duration) bool {
	if len(s.steps) == 0 || s.current >= len(s.steps) {
		return false
	}

	step := &s.steps[s.current]
	running := step.anim.Tick(dt)

	if !running && step.anim.IsDone() {
		// Step completed — run hook and advance.
		if step.onDone != nil {
			step.onDone()
		}
		s.current++
		// Don't tick the next step this frame — let it start next Tick.
	}

	return s.current < len(s.steps)
}

// IsDone returns true when all steps have completed.
func (s *AnimSeq) IsDone() bool {
	return len(s.steps) == 0 || s.current >= len(s.steps)
}
