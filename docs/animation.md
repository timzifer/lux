# Animation

lux animations are **deterministic and frame-driven**. They run in the app loop — no goroutines,
no timers, no `time.Sleep`. Tests inject `dt` directly to verify animation behaviour.

```go
import "github.com/timzifer/lux/anim"
```

---

## `Anim[T]` — Interpolated Animation

`Anim[T]` animates any numeric type (`float32`, `float64`, or any named type based on them).

```go
type MyState struct {
    Opacity anim.Anim[float32]
    OffsetX anim.Anim[float32]
}
```

### Starting an animation

```go
state.Opacity.SetTarget(1.0, 300*time.Millisecond, anim.OutCubic)
```

If an animation is already running, `SetTarget` continues from the current value (no snap).

### Snapping without animation

```go
state.Opacity.SetImmediate(0.0)
```

### Reading the value

```go
alpha := state.Opacity.Value() // current interpolated value
done  := state.Opacity.IsDone()
```

### Wiring to the framework

For the framework to call `Tick(dt)` on your animations, implement the `Animator` interface
on your `WidgetState`:

```go
func (s *MyState) Tick(dt time.Duration) (stillRunning bool) {
    a := s.Opacity.Tick(dt)
    b := s.OffsetX.Tick(dt)
    return a || b
}
```

The framework calls `Tick` before each paint pass and marks the widget dirty when
`stillRunning` is `true`.

---

## `LerpAnim[T]` — Arbitrary-type Animation

For types that are not numeric (e.g. `draw.Color`, `draw.Point`), use `LerpAnim[T]` with a
custom lerp function:

```go
type MyState struct {
    BgColor anim.LerpAnim[draw.Color]
}

state.BgColor.SetTarget(targetColor, 250*time.Millisecond, anim.OutCubic, draw.LerpColor)
```

`draw.LerpColor` interpolates RGBA channels. You can supply any `LerpFunc[T]`:

```go
type LerpFunc[T any] func(a, b T, t float32) T
```

---

## `SpringAnim[T]` — Spring Physics

`SpringAnim[T]` simulates a spring-damper system. Unlike `Anim[T]`, it has no fixed duration —
it converges asymptotically.

```go
type MyState struct {
    Scale anim.SpringAnim[float32]
}

// Set target with default spec (SpringGentle)
state.Scale.SetTarget(1.2)

// Set target with explicit spec
state.Scale.SetTargetWithSpec(1.0, anim.SpringSnappy)

// Snap
state.Scale.SetImmediate(1.0)
```

### Built-in spring presets

| Preset | Stiffness | Damping | Character |
|--------|-----------|---------|-----------|
| `anim.SpringGentle` | 120 | 14 | Smooth, slow settle |
| `anim.SpringSnappy` | 400 | 28 | Fast, crisp |
| `anim.SpringBouncy` | 200 | 10 | Overshoots and bounces |

### Custom spring

```go
state.Scale.SetTargetWithSpec(1.0, anim.SpringSpec{
    Stiffness:         300,
    Damping:           20,
    Mass:              1.0,
    SettlingThreshold: 0.001,
})
```

---

## Easing Functions

All easing functions have type `func(t float32) float32` where `t ∈ [0, 1]`.

| Constant | Description |
|----------|-------------|
| `anim.Linear` | No easing |
| `anim.OutCubic` | Decelerates — standard transitions |
| `anim.InCubic` | Accelerates |
| `anim.InOutCubic` | Symmetric ease-in/ease-out |
| `anim.OutExpo` | Exponential deceleration — fast reactions |
| `anim.CubicBezier(x1, y1, x2, y2)` | CSS-compatible cubic-bezier |

```go
// CSS equivalent: cubic-bezier(0.4, 0, 0.2, 1)
easing := anim.CubicBezier(0.4, 0, 0.2, 1)
state.Opacity.SetTarget(1.0, 300*time.Millisecond, easing)
```

---

## Completion Callbacks — `AnimationID`

To sequence actions after an animation completes, use `SetTargetWithID`. When the animation
finishes, an `anim.AnimationEnded{ID}` message is sent via `app.Send`:

```go
const SlideInDone anim.AnimationID = "slide-in"

// In Render or Tick:
state.OffsetX.SetTargetWithID(0, 300*time.Millisecond, anim.OutCubic, SlideInDone)

// In update:
func update(m Model, msg app.Msg) Model {
    switch v := msg.(type) {
    case anim.AnimationEnded:
        if v.ID == SlideInDone {
            // animation is complete
        }
    }
    return m
}
```

---

## `AnimGroup` — Parallel Animations

Run multiple animations concurrently; completes when all are done:

```go
group := anim.NewGroup(animA, animB, animC)
running := group.Tick(dt)
```

---

## `AnimSeq` — Sequential Animations

Run animations one after another; each starts when the previous completes:

```go
seq := anim.NewSeq(animA, animB, animC)
running := seq.Tick(dt)
```

---

## Frame-rate Independence

All animations use the `dt` (delta time) from `app.TickMsg.DeltaTime`. The framework clamps
`dt` to a maximum (default: 100ms) to prevent large jumps after tab switches or system sleep.
This maximum can be overridden with `app.WithMaxFrameDelta`.

---

## Testing Animations

Because animations accept `dt` directly, testing is straightforward:

```go
state := &MyState{}
state.Opacity.SetTarget(1.0, 100*time.Millisecond, anim.Linear)

state.Opacity.Tick(50 * time.Millisecond) // half-way: 0.5
assert.InDelta(t, 0.5, float64(state.Opacity.Value()), 0.001)

state.Opacity.Tick(50 * time.Millisecond) // done: 1.0
assert.Equal(t, float32(1.0), state.Opacity.Value())
assert.True(t, state.Opacity.IsDone())
```
