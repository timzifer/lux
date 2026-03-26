# Elm Architecture

lux is built around the [Elm architecture](https://guide.elm-lang.org/architecture/): a
single-threaded update loop that eliminates data races by design.

```
          ┌───────────┐
          │   Model   │ ← application state (any struct)
          └─────┬─────┘
                │ view(model) → Element tree
          ┌─────▼─────┐
          │   View    │ ← pure function, no side effects
          └─────┬─────┘
                │ user action → Msg
          ┌─────▼─────┐
          │  Update   │ ← pure function: (Model, Msg) → Model
          └─────┬─────┘
                │ new Model
                └──────────► loop
```

## Entry Points

### `app.Run`

The simplest form — update returns only a new model:

```go
func Run[M any](model M, update UpdateFunc[M], view ViewFunc[M], opts ...Option) error
```

```go
type UpdateFunc[M any] func(M, Msg) M
type ViewFunc[M any]   func(M) ui.Element
```

### `app.RunWithCmd`

When your update needs to start side effects:

```go
func RunWithCmd[M any](model M, update UpdateWithCmd[M], view ViewFunc[M], opts ...Option) error
```

```go
type UpdateWithCmd[M any] func(M, Msg) (M, Cmd)
```

Both functions block until the window is closed.

## Messages

`app.Msg` is `any` — every Go type is a valid message. Use type switches to match:

```go
func update(m Model, msg app.Msg) Model {
    switch v := msg.(type) {
    case MyMsg:
        // ...
    case OtherMsg:
        // use v.Field
    }
    return m
}
```

### Built-in messages

| Type | When it arrives | Use it to |
|------|----------------|-----------|
| `app.TickMsg{DeltaTime}` | Every frame | Drive animations, timers, physics |
| `app.SetThemeMsg{Theme}` | After `app.Send(SetThemeMsg{...})` | Switch to a custom theme |
| `app.SetDarkModeMsg{Dark}` | After `app.Send(SetDarkModeMsg{...})` | Toggle built-in dark/light |
| `app.ModelRestoredMsg{}` | Once, after persistence restore | Re-apply side effects from restored state |
| `input.ShortcutMsg{ID}` | On keyboard shortcut | Handle a registered shortcut |

## Commands

`Cmd` is a side-effect function that optionally returns a message:

```go
type Cmd func() Msg
```

Returned from `UpdateWithCmd`, commands are dispatched **asynchronously** after each update:

```go
func update(m Model, msg app.Msg) (Model, app.Cmd) {
    switch msg.(type) {
    case LoadDataMsg:
        return m, func() app.Msg {
            data, err := fetchData()
            return DataLoadedMsg{Data: data, Err: err}
        }
    case DataLoadedMsg:
        // handle result
    }
    return m, app.None
}
```

### `app.Batch`

Combine multiple commands into one:

```go
return m, app.Batch(cmdA, cmdB, cmdC)
```

### `app.None`

A readable nil-command sentinel — equivalent to returning `nil` but more explicit:

```go
return m, app.None
```

## Sending Messages

Use `app.Send` to enqueue a message from any goroutine (button callbacks, Cmds, background
workers). It is thread-safe and never blocks:

```go
app.Send(MyMsg{})
```

`app.TrySend` returns `false` if the internal buffer is full (non-blocking best-effort):

```go
if !app.TrySend(msg) {
    // buffer full — drop or retry later
}
```

## Options

`app.Run` accepts option functions:

```go
app.Run(model, update, view,
    app.WithTitle("My App"),
    app.WithSize(1024, 768),
    app.WithTheme(theme.Default),
    app.WithFullscreen(false),
    app.WithShortcut(input.Shortcut{Key: input.KeyQ, Mod: input.ModCtrl}, QuitID),
    app.WithGlobalHandler(myGlobalHandler),
    app.WithPersistence(app.PersistenceConfig[Model]{
        Encode:     json.Marshal,
        Decode:     func(b []byte) (Model, error) { var m Model; return m, json.Unmarshal(b, &m) },
        StorageKey: "my-app",
    }),
)
```

| Option | Default | Description |
|--------|---------|-------------|
| `WithTitle(s)` | `"lux"` | Window title |
| `WithSize(w, h)` | `800×600` | Initial window size (screen coordinates) |
| `WithTheme(t)` | `theme.Default` | Initial theme |
| `WithFullscreen(b)` | `false` | Start in fullscreen |
| `WithMaxFrameDelta(d)` | 100ms | Cap on dt passed to `TickMsg` (prevents spiral-of-death) |
| `WithShortcut(s, id)` | — | Register a global keyboard shortcut |
| `WithGlobalHandler(h)` | — | Register a pre-dispatch input handler |
| `WithPersistence(cfg)` | — | Enable model persistence |
| `WithPlatform(f)` | auto | Override platform backend factory |
| `WithRenderer(f)` | auto | Override GPU renderer factory |
| `WithImageStore(s)` | — | Register an image store for GPU texture sync |

## Multi-window

For applications that need multiple windows, use `RunMulti`:

```go
type MultiViewFunc[M any] func(M) map[WindowID]ui.Element
```

Each key in the returned map corresponds to a managed window. Windows are created/destroyed
automatically as keys appear or disappear across view calls.

## Clipboard

```go
app.SetClipboard("hello")
text, err := app.GetClipboard()
```

## Window Messages

```go
app.Send(app.SetSizeMsg{Width: 1280, Height: 800})
app.Send(app.SetFullscreenMsg{Fullscreen: true})
```

## See Also

- [State Persistence](advanced/state-persistence.md) — `WithPersistence` and `ModelRestoredMsg`
- [Input & Events](input-and-events.md) — shortcuts and global handlers
- [Custom Widgets](advanced/custom-widgets.md) — `RenderCtx.Send` for widget-local messages
