# Custom Widgets

A widget is any type that implements the `ui.Widget` interface:

```go
type Widget interface {
    Render(ctx RenderCtx, state WidgetState) (Element, WidgetState)
}
```

- `ctx` — context for this render call: UID, theme, send function, events, locale.
- `state` — the widget's private mutable state from the previous frame (`nil` on first call).
- Returns — a new `Element` (the widget's subtree) and updated state.

---

## Minimal Example

```go
type ClickCounter struct {
    Label string
}

type clickCounterState struct {
    count int
}

func (w *ClickCounter) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
    s := ui.AdoptState[clickCounterState](raw)

    for _, ev := range ctx.Events {
        if me, ok := ev.(ui.MouseEvent); ok &&
            me.Button == input.MouseButtonLeft &&
            me.Action == input.Press {
            s.count++
        }
    }

    return display.Text(fmt.Sprintf("%s: %d", w.Label, s.count)), s
}
```

Use it like any other widget:

```go
&ClickCounter{Label: "Clicks"}
```

---

## `ui.AdoptState[S]`

```go
func AdoptState[S WidgetState](raw WidgetState) *S
```

Type-asserts `raw` to `*S` and returns it. If `raw` is `nil` (first call) or the wrong type,
it allocates and returns a zeroed `*S`. Always use this instead of a bare type assertion.

---

## `RenderCtx`

```go
type RenderCtx struct {
    UID    UID              // stable identity across frames
    Theme  theme.Theme      // current theme; access tokens via Theme.Tokens()
    Send   func(any)        // enqueue a message (widget-local, bound to UID)
    Events []InputEvent     // input events dispatched to this widget
    Locale string           // BCP 47 language tag, e.g. "de", "en-US"
}
```

`ctx.Send` is equivalent to `app.Send` but is bound to the widget's UID for tracing and
debugging. Prefer `ctx.Send` inside `Render` and `app.Send` in callbacks.

---

## Optional Interfaces

### `Animator`

Implement on your `WidgetState` to receive `Tick(dt)` before each paint pass:

```go
type Animator interface {
    Tick(dt time.Duration) (stillRunning bool)
}
```

If `Tick` returns `true`, the framework marks the widget dirty and schedules a repaint.

```go
type myState struct {
    fade anim.Anim[float32]
}

func (s *myState) Tick(dt time.Duration) bool {
    return s.fade.Tick(dt)
}
```

### `Equatable`

Implement on your `Widget` to skip `Render` when props have not changed:

```go
type Equatable interface {
    Widget
    Equal(other Widget) bool
}
```

```go
func (w *MyWidget) Equal(other ui.Widget) bool {
    o, ok := other.(*MyWidget)
    if !ok {
        return false
    }
    return w.Label == o.Label && w.Value == o.Value
}
```

When `Equal` returns `true`, the reconciler reuses the previous render output and state
without calling `Render`. This is a performance optimisation — it is always safe to omit.

### `DirtyTracker`

For widgets whose state can change independently of props (e.g. a video surface, external
data feed):

```go
type DirtyTracker interface {
    IsDirty() bool
    ClearDirty()
}
```

The framework calls `IsDirty()` after the animation tick. If true, the widget is re-rendered
even if its props have not changed.

### `Cursable`

Declare the desired cursor when the pointer hovers over the widget:

```go
type Cursable interface {
    Cursor() input.CursorKind
}

func (w *MyWidget) Cursor() input.CursorKind {
    return input.CursorPointer
}
```

---

## UID and Reconciliation

`UID` is a `uint64` assigned by the framework based on the widget's position in the Element
tree. It is stable across frames as long as the widget's position in the tree does not change.

For dynamic lists where position can change, wrap children in `ui.Keyed(key, widget)` to
assign a stable key:

```go
for i, item := range items {
    children = append(children, ui.Keyed(item.ID, &MyItem{Data: item}))
}
```

---

## Drawing in a Widget

Widgets produce `Element` trees — they do not draw directly. Drawing happens in a theme
`DrawFunc` or in a `draw.Canvas` passed by the framework to the DrawFunc.

To draw custom graphics, return a `ui.Custom` element that wraps a draw function:

```go
func (w *MyCanvas) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
    s := ui.AdoptState[myCanvasState](raw)
    return ui.Custom{
        MinSize: draw.Pt(w.Width, w.Height),
        Draw: func(c draw.Canvas) {
            c.FillRect(c.Bounds(), draw.SolidColor(draw.Hex("#1a1a2e")))
            c.StrokeLine(draw.Pt(0, 0), draw.Pt(c.Bounds().W, c.Bounds().H),
                draw.Stroke{Color: draw.Hex("#4c8dff"), Width: 2})
        },
    }, s
}
```

---

## Widget Registration (for theme DrawFuncs)

If your custom widget should be theme-renderable (other developers can provide custom draw
implementations for it), define a new `theme.WidgetKind` constant and use
`theme.DrawFunc(kind)` in your `Render` to dispatch to the theme.

This pattern mirrors the built-in widgets and is optional for application-level widgets.
