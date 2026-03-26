# lux

**lux** is a GPU-accelerated UI framework for Go built on the [Elm architecture](docs/elm-architecture.md).
It combines a deterministic, single-threaded app loop with a modern rendering stack
(wgpu/Vulkan/Metal/D3D12) and first-class support for accessibility, animations, and
embedded/bare-metal targets.

```
go get github.com/timzifer/lux
```

Requires **Go 1.25+**.

---

## Features

- **Elm architecture** — pure `update(model, msg) → model` and `view(model) → Element`; no data races possible
- **GPU rendering** — wgpu (Vulkan / Metal / D3D12 / WebGPU) with OpenGL 3.3 fallback
- **CSS-compatible layout** — Flexbox, CSS Grid, and HTML Table algorithms
- **Animation system** — deterministic, frame-driven `Anim[T]` and spring physics; no goroutines
- **Design token theming** — `LuxDark` / `LuxLight` built in; fully customisable
- **Rich widget catalogue** — 20+ widgets (buttons, forms, navigation, data, menus, dialogs)
- **Six platform backends** — GLFW · Win32 · Cocoa · X11 · Wayland · DRM/KMS
- **Accessibility** — Windows UIA · macOS NSAccessibility · Linux AT-SPI2 (pure-Go D-Bus)
- **i18n / RTL** — BCP 47 locale propagation, Unicode BiDi & line-breaking, grapheme-aware cursor

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/timzifer/lux/app"
    "github.com/timzifer/lux/theme"
    "github.com/timzifer/lux/ui"
    "github.com/timzifer/lux/ui/button"
    "github.com/timzifer/lux/ui/display"
    "github.com/timzifer/lux/ui/layout"
)

// 1. Define your model.
type Model struct{ Count int }

// 2. Define your messages.
type IncrMsg struct{}
type DecrMsg struct{}

// 3. Pure update function — no side effects, no locks.
func update(m Model, msg app.Msg) Model {
    switch msg.(type) {
    case IncrMsg:
        m.Count++
    case DecrMsg:
        m.Count--
    }
    return m
}

// 4. Pure view function — returns an Element tree.
func view(m Model) ui.Element {
    return layout.Column(
        display.Text(fmt.Sprintf("Count: %d", m.Count)),
        layout.Row(
            button.Text("-", func() { app.Send(DecrMsg{}) }),
            button.Text("+", func() { app.Send(IncrMsg{}) }),
        ),
    )
}

func main() {
    if err := app.Run(Model{}, update, view,
        app.WithTheme(theme.Default),
        app.WithTitle("Counter"),
    ); err != nil {
        log.Fatal(err)
    }
}
```

Run it:

```
go run ./examples/counter/
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](docs/getting-started.md) | Installation, prerequisites, annotated walkthrough |
| [Elm Architecture](docs/elm-architecture.md) | Model / Update / View / Cmd in depth |
| [Architecture](docs/architecture.md) | Package map, rendering pipeline, app loop internals |
| [Widget Catalogue](docs/widgets.md) | All widgets with API reference |
| [Layout](docs/layout.md) | Flexbox, CSS Grid, Table, Box, Stack, RTL |
| [Theming](docs/theming.md) | Design tokens, built-in themes, custom themes |
| [Animation](docs/animation.md) | `Anim[T]`, spring physics, easing, `AnimGroup`/`AnimSeq` |
| [Drawing API](docs/drawing.md) | `draw.Canvas` reference |
| [Input & Events](docs/input-and-events.md) | Keyboard, mouse, touch, IME, shortcuts |
| [Testing](docs/testing.md) | Golden-file scene tests with `uitest` |
| [Platform Backends](docs/platform-backends.md) | Backend selection and configuration |
| [Accessibility](docs/accessibility.md) | AccessTree, UIA, AT-SPI2, NSAccessibility |
| **Advanced** | |
| [Custom Widgets](docs/advanced/custom-widgets.md) | Implementing the `Widget` interface |
| [State Persistence](docs/advanced/state-persistence.md) | Save/restore app state across restarts |
| [RFC Index](docs/rfc/README.md) | Internal design documents |

---

## Examples

```bash
go run ./examples/counter/       # simple counter with theme toggle
go run ./examples/kitchen-sink/  # comprehensive widget showcase
go run ./examples/fenster/       # window management
```

---

## Platform Support

| Platform | Backend | GPU | Notes |
|----------|---------|-----|-------|
| Windows 10+ | Win32 native / GLFW | Vulkan · D3D12 | WebView2 surface available |
| macOS 11+ | Cocoa / GLFW | Metal | |
| Linux (X11) | X11 / GLFW | Vulkan · OpenGL | |
| Linux (Wayland) | Wayland / GLFW | Vulkan · OpenGL | |
| Embedded / bare-metal | DRM/KMS | Vulkan · OpenGL | Raspberry Pi, industrial HMI |

---

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│                 app package                 │  ← Elm loop, messages, commands
│  Run(model, update, view, opts...)          │
└──────────────┬──────────────────────────────┘
               │ Element tree
┌──────────────▼──────────────────────────────┐
│               ui package                   │  ← Widget system, layout, focus,
│  Widget → Render → reconcile → layout      │     gestures, accessibility tree
└──────────────┬──────────────────────────────┘
               │ draw.Canvas commands
┌──────────────▼──────────────────────────────┐
│           internal/render                  │  ← Scene graph, batching,
│           internal/gpu (wgpu)              │     GPU upload, wgpu backend
└──────────────┬──────────────────────────────┘
               │ native window / surface
┌──────────────▼──────────────────────────────┐
│            platform package                │  ← GLFW · Win32 · Cocoa ·
│  Init · Run · Events · Clipboard           │     X11 · Wayland · DRM/KMS
└─────────────────────────────────────────────┘
```

---

## Contributing

Design decisions are captured in the [RFC documents](docs/rfc/).
Start there before opening a pull request for a new subsystem.
