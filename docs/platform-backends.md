# Platform Backends

lux abstracts the native windowing system behind the `platform.Platform` interface. Six
backends are provided; the right one is selected automatically based on build tags and the
runtime environment.

```go
import "github.com/timzifer/lux/platform"
```

---

## Platform Interface

```go
type Platform interface {
    Init(cfg Config) error
    Run(cb Callbacks) error
    Destroy()

    SetTitle(title string)
    WindowSize() (width, height int)
    FramebufferSize() (width, height int)
    ShouldClose() bool

    SetCursor(kind input.CursorKind)
    SetIMECursorRect(x, y, w, h int)
    SetSize(w, h int)
    SetFullscreen(fullscreen bool)
    RequestFrame()

    SetClipboard(text string) error
    GetClipboard() (string, error)

    CreateWGPUSurface(instance uintptr) uintptr
}
```

---

## Available Backends

### GLFW (`platform/glfw`)

Cross-platform fallback backed by GLFW 3. Works on Windows, macOS, and Linux
(both X11 and Wayland via GLFW's backend selection). Used when no native backend is built in
or explicitly selected.

Build tag: always available (default).

```go
import "github.com/timzifer/lux/platform/glfw"
```

### Windows — Win32 native (`platform/windows`)

Native Win32 backend with full HiDPI support, per-monitor DPI awareness.
Recommended for production Windows builds.

Build constraint: `GOOS=windows`.

### macOS — Cocoa native (`platform/cocoa`)

Native Cocoa/AppKit backend with Metal surface. Proper retina display support and
macOS-native window chrome.

Build constraint: `GOOS=darwin`.

### X11 (`platform/x11`)

Direct X11 backend without the GLFW dependency. Recommended for Linux desktop applications
where GLFW's indirect layer is not desired.

Build constraint: `GOOS=linux`.

### Wayland (`platform/wayland`)

Wayland backend for modern Linux desktops. Supports fractional scaling, xdg-shell, and
xdg-decoration.

Build constraint: `GOOS=linux`.

### DRM/KMS — Embedded (`platform/drm`)

Direct Rendering Manager / Kernel Mode Setting backend for embedded and bare-metal systems.
No display server required. Suitable for:

- Raspberry Pi (bare-metal, full-screen kiosk)
- Industrial HMI panels
- Set-top boxes and other Linux embedded targets
- Automotive infotainment

Build constraint: `GOOS=linux` + `CGO_ENABLED=1` (requires `libdrm`).

---

## Overriding the Backend

Pass `app.WithPlatform` to override the default platform factory:

```go
import "github.com/timzifer/lux/platform/drm"

app.Run(model, update, view,
    app.WithPlatform(func() platform.Platform {
        return drm.New()
    }),
)
```

---

## GPU Surface

Each backend implements `CreateWGPUSurface(instance uintptr) uintptr`, which creates a
platform-specific `wgpu.Surface` from the backend's native window handle:

| Backend | Surface type |
|---------|-------------|
| GLFW | `WGPUSurfaceDescriptorFromGLFWWindow` |
| Win32 | `WGPUSurfaceDescriptorFromWindowsHWND` |
| Cocoa | `WGPUSurfaceDescriptorFromMetalLayer` |
| X11 | `WGPUSurfaceDescriptorFromXlibWindow` |
| Wayland | `WGPUSurfaceDescriptorFromWaylandSurface` |
| DRM | `WGPUSurfaceDescriptorFromDRMKMSPlane` |

---

## Platform Config

```go
type Config struct {
    Title  string
    Width  int
    Height int
}
```

---

## Platform Callbacks

The `Run` loop calls back into the framework for events:

```go
type Callbacks struct {
    OnFrame       func(dt time.Duration)
    OnResize      func(width, height int)
    OnKey         func(key input.Key, mods input.ModifierSet, action input.KeyAction)
    OnChar        func(r rune, mods input.ModifierSet)
    OnMouseButton func(button input.MouseButton, action input.MouseAction, mods input.ModifierSet)
    OnMouseMove   func(x, y float32)
    OnScroll      func(dx, dy float32, precise bool)
    OnTouch       func(events []input.TouchEvent)
    OnIME         func(ev input.IMEEvent)
    OnClose       func()
}
```

These are wired automatically by `app.Run`; application code does not interact with them
directly.

---

## Accessibility Bridge

The Win32 and Cocoa backends expose an optional `A11yBridge()` method. If present, lux
automatically initialises the platform accessibility bridge:

```go
type A11yBridgeProvider interface {
    A11yBridge() a11y.A11yBridge
}
```

See [Accessibility](accessibility.md) for details.
