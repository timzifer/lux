# Input and Events

```go
import "github.com/timzifer/lux/input"
```

---

## Key Type

Keys are represented as `input.Key` (`uint32`), not strings. This avoids locale-dependent
string mappings and is consistent with USB HID codes.

```go
const (
    KeyA … KeyZ
    Key0 … Key9
    KeyF1 … KeyF12
    KeyEscape, KeyEnter, KeyTab, KeyBackspace
    KeyInsert, KeyDelete, KeyRight, KeyLeft, KeyDown, KeyUp
    KeyPageUp, KeyPageDown, KeyHome, KeyEnd
    KeySpace, KeyMinus, KeyEqual, …
    KeyLeftShift, KeyLeftCtrl, KeyLeftAlt, KeyLeftSuper
    KeyRightShift, KeyRightCtrl, KeyRightAlt, KeyRightSuper
)
```

### Modifier set

```go
type ModifierSet uint32
const (
    ModShift ModifierSet = 1 << iota
    ModCtrl
    ModAlt
    ModSuper // Cmd on macOS, Win on Windows
)
```

Check a modifier:

```go
if mods.Has(input.ModCtrl) { ... }
```

---

## Keyboard Events

Keyboard events arrive in `RenderCtx.Events` as `ui.InputEvent` values. Type-switch to
handle them:

```go
func (w *MyWidget) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
    for _, ev := range ctx.Events {
        switch e := ev.(type) {
        case ui.KeyEvent:
            if e.Key == input.KeyEnter && e.Action == input.Press {
                ctx.Send(SubmitMsg{})
            }
        case ui.CharEvent:
            ctx.Send(TextInputMsg{Rune: e.Rune})
        }
    }
    // ...
}
```

### `ui.KeyEvent`

```go
type KeyEvent struct {
    Key      input.Key
    Mods     input.ModifierSet
    Action   input.KeyAction // Press | Release | Repeat
}
```

### `ui.CharEvent`

Fired for printable character input. Always paired with `KeyEvent`; prefer `CharEvent` for
text input to handle dead keys, AltGr, and compose sequences correctly.

```go
type CharEvent struct {
    Rune rune
    Mods input.ModifierSet
}
```

---

## Mouse Events

```go
case ui.MouseEvent:
    if e.Button == input.MouseButtonLeft && e.Action == input.Press {
        ctx.Send(ClickMsg{X: e.X, Y: e.Y})
    }
```

```go
type MouseEvent struct {
    Button input.MouseButton // Left | Right | Middle | Back | Forward
    Action input.MouseAction // Press | Release
    X, Y   float32           // position in widget-local coordinates
    Mods   input.ModifierSet
}

type MouseMoveEvent struct {
    X, Y float32
    Mods input.ModifierSet
}
```

---

## Scroll Events

```go
case ui.ScrollEvent:
    // e.DX, e.DY: scroll delta in dp
    // e.Precise: true for trackpad (high-resolution), false for scroll wheel
```

```go
type ScrollEvent struct {
    DX, DY  float32
    Precise bool
    Mods    input.ModifierSet
}
```

---

## Touch Events

```go
case ui.TouchEvent:
    switch e.Phase {
    case input.TouchBegan:
    case input.TouchMoved:
    case input.TouchEnded:
    case input.TouchCancelled:
    }
```

```go
type TouchEvent struct {
    ID    input.TouchID
    Phase input.TouchPhase
    X, Y  float32
    Force float32 // 0–1; 0 if not supported
}
```

Multiple touch events per frame correspond to multiple simultaneous fingers.

---

## IME (Input Method Editor)

For CJK and other compose-heavy input, the `IMEEvent` carries the compose string:

```go
case ui.IMEEvent:
    switch e.Type {
    case input.IMEPreedit:
        // e.Text is the in-progress compose string (show underline)
    case input.IMECommit:
        // e.Text is the final committed string
    }
```

---

## Keyboard Shortcuts

Register global shortcuts at startup:

```go
const QuitID input.ShortcutID = "quit"

app.Run(model, update, view,
    app.WithShortcut(input.Shortcut{
        Key: input.KeyQ,
        Mod: input.ModCtrl, // ModSuper on macOS is handled automatically
    }, QuitID),
)
```

When the combination is pressed, `input.ShortcutMsg{ID: QuitID}` is sent:

```go
func update(m Model, msg app.Msg) Model {
    switch v := msg.(type) {
    case input.ShortcutMsg:
        if v.ID == QuitID {
            // handle quit
        }
    }
    return m
}
```

---

## Global Input Handlers

For app-level input interception (e.g. modal overlays that capture all input):

```go
// Static handler registered at startup
app.Run(model, update, view,
    app.WithGlobalHandler(func(ev ui.InputEvent) (consumed bool) {
        if ke, ok := ev.(ui.KeyEvent); ok && ke.Key == input.KeyEscape {
            app.Send(CloseModalMsg{})
            return true // consume — prevent delivery to widgets
        }
        return false
    }),
)

// Dynamic handler registered at runtime
app.Send(app.RegisterHandlerMsg{
    ID:      "modal",
    Handler: myHandler,
})

// Remove it later
app.Send(app.UnregisterHandlerMsg{ID: "modal"})
```

Global handlers run before widget dispatch. Return `true` to consume the event.

---

## Focus Management

Focus determines which widget receives keyboard events.

```go
// Get the app-level focus manager
fm := app.Focus()

// Pass it to widgets that need keyboard input
form.TextField(form.WithFocus(fm))
```

Tab order is derived from layout order by default. The `ui.FocusManager` maintains a list of
focusable widget UIDs and tracks the currently focused one.

### Focus trap

Modal dialogs use `ui.FocusTrap` to confine Tab navigation:

```go
ui.WithFocusTrap(dialogContent) // Tab cannot escape the dialog
```

---

## Cursor

Widgets can request a specific cursor by implementing the `ui.Cursable` interface:

```go
type Cursable interface {
    Cursor() input.CursorKind
}
```

```go
const (
    CursorDefault    CursorKind = iota
    CursorText                  // I-beam
    CursorPointer               // hand
    CursorCrosshair
    CursorMove
    CursorResizeNS
    CursorResizeEW
    CursorResizeNWSE
    CursorResizeNESW
    CursorNotAllowed
    CursorWait
    CursorProgress
)
```

The framework calls `Cursor()` after hit-testing and sets the platform cursor accordingly.
