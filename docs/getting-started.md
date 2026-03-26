# Getting Started

## Prerequisites

- **Go 1.25** or later
- Platform-specific GPU dependencies:
  - **Linux**: Vulkan loader (`libvulkan1`) or Mesa drivers; for Wayland: `libwayland-client`
  - **macOS**: Metal is included with the OS (macOS 11+)
  - **Windows**: DirectX 12 (Windows 10+) or Vulkan drivers
  - **Embedded (DRM/KMS)**: Kernel 4.14+ with DRM/KMS support

## Installation

```bash
go get github.com/timzifer/lux
```

## Your First Application

Every lux application has three parts: a **model**, an **update** function, and a **view** function.

### Step 1 — Define your model

The model is plain Go data. It can be any struct.

```go
type Model struct {
    Count int
    Dark  bool
}
```

### Step 2 — Define your messages

Messages are plain Go types. Any type is a valid message.

```go
type IncrMsg struct{}
type DecrMsg struct{}
type ToggleThemeMsg struct{}
```

### Step 3 — Write the update function

The update function is a **pure function**: it takes the current model and a message, and returns
a new model. No side effects, no goroutines, no locks needed.

```go
func update(m Model, msg app.Msg) Model {
    switch msg.(type) {
    case IncrMsg:
        m.Count++
    case DecrMsg:
        m.Count--
    case app.ModelRestoredMsg:
        // Re-apply side effects after state restore.
        app.Send(app.SetDarkModeMsg{Dark: m.Dark})
    case ToggleThemeMsg:
        m.Dark = !m.Dark
        app.Send(app.SetDarkModeMsg{Dark: m.Dark})
    }
    return m
}
```

`app.Send` is the escape hatch for side effects. It enqueues a message into the app loop from
any goroutine and returns immediately.

### Step 4 — Write the view function

The view function is also a **pure function**: it takes the model and returns an Element tree.
It is called after every update.

```go
func view(m Model) ui.Element {
    themeLabel := "LIGHT"
    if m.Dark {
        themeLabel = "DARK"
    }
    return layout.Column(
        display.Text(fmt.Sprintf("Count: %d", m.Count)),
        display.Divider(),
        layout.Row(
            button.Text("-", func() { app.Send(DecrMsg{}) }),
            button.Text("+", func() { app.Send(IncrMsg{}) }),
        ),
        display.Divider(),
        button.Text(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
    )
}
```

### Step 5 — Start the application

```go
func main() {
    if err := app.Run(Model{Count: 0, Dark: true}, update, view,
        app.WithTheme(theme.Default),
        app.WithTitle("Counter"),
    ); err != nil {
        log.Fatal(err)
    }
}
```

`app.Run` blocks until the window is closed.

### Full source

The complete example lives at [`examples/counter/main.go`](../examples/counter/main.go).

```bash
go run ./examples/counter/
```

## Exploring Other Examples

```bash
# Comprehensive widget showcase
go run ./examples/kitchen-sink/

# Window management
go run ./examples/fenster/
```

## Next Steps

| Topic | Document |
|-------|----------|
| How messages and commands work | [Elm Architecture](elm-architecture.md) |
| All available widgets | [Widget Catalogue](widgets.md) |
| Flexbox / Grid layout | [Layout](layout.md) |
| Animations | [Animation](animation.md) |
| Theming and design tokens | [Theming](theming.md) |
| Custom widgets | [Advanced: Custom Widgets](advanced/custom-widgets.md) |
