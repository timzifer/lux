# Testing

lux provides a `uitest` package for golden-file scene tests. Tests run headlessly — no GPU,
no window, no OS integration required.

```go
import "github.com/timzifer/lux/uitest"
```

---

## Golden-file Tests

A golden-file test renders an `Element` tree into a deterministic scene description and
compares it to a stored reference file.

### Basic pattern

```go
func TestMyWidget(t *testing.T) {
    root := layout.Column(
        display.Text("Hello"),
        button.Text("Click me", func() {}),
    )

    scene := uitest.BuildScene(root, 400, 200)
    uitest.AssertScene(t, scene, "testdata/my_widget.golden")
}
```

### `uitest.BuildScene`

```go
func BuildScene(root ui.Element, width, height int) draw.Scene
```

Renders `root` at the given size using the default theme (`theme.Default`). Returns a
`draw.Scene` — a serialisable representation of all draw commands.

### `uitest.AssertScene`

```go
func AssertScene(t *testing.T, scene draw.Scene, goldenPath string)
```

- Serialises the scene to a text format.
- Compares to the file at `goldenPath` (relative to the test's working directory).
- If the file does not exist, the test fails with instructions to run `-update`.
- With the `-update` flag, the file is written/overwritten instead of compared.

---

## Creating and Updating Golden Files

Run tests with `-update` to write the initial golden files:

```bash
go test ./... -update
```

After a deliberate visual change, update the affected goldens:

```bash
go test ./ui/button/ -update
```

Check the diff before committing:

```bash
git diff testdata/
```

---

## File Layout

Convention: store golden files in a `testdata/` subdirectory within the package:

```
ui/button/
  button_test.go
  testdata/
    button_default.golden
    button_hover.golden
    button_disabled.golden
```

---

## Testing Animations

Because animations accept `dt` directly, animation behaviour can be tested without a running
event loop:

```go
func TestFadeIn(t *testing.T) {
    state := &MyWidgetState{}
    state.Opacity.SetTarget(1.0, 100*time.Millisecond, anim.Linear)

    state.Opacity.Tick(50 * time.Millisecond)
    if got := state.Opacity.Value(); math.Abs(float64(got-0.5)) > 0.001 {
        t.Errorf("expected 0.5, got %v", got)
    }

    state.Opacity.Tick(50 * time.Millisecond)
    if !state.Opacity.IsDone() {
        t.Error("expected animation to be done")
    }
}
```

---

## Testing Update Logic

The `update` function is a pure function — unit-test it directly without any framework setup:

```go
func TestUpdate(t *testing.T) {
    m := Model{Count: 0}
    m = update(m, IncrMsg{})
    m = update(m, IncrMsg{})
    m = update(m, DecrMsg{})
    if m.Count != 1 {
        t.Errorf("expected 1, got %d", m.Count)
    }
}
```

---

## Headless Mode

`uitest.BuildScene` uses the headless rendering path internally. For custom headless rendering:

```go
canvas := render.NewSceneCanvas(800, 600)
scene := ui.BuildScene(root, canvas, theme.Default, 800, 600, nil)
```

---

## CI

Golden-file tests pass without any display server or GPU driver. They are safe to run in
standard CI environments (Linux containers, GitHub Actions, etc.).

The only test flag needed:

```yaml
- run: go test ./...
```

No `Xvfb`, no Mesa software renderer, no display configuration required.
