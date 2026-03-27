# Architecture

## Package Map

```
github.com/timzifer/lux/
│
├── app/               App loop, Elm entry points, options, persistence, clipboard
├── ui/                Widget system, Element tree, reconciliation, layout, focus,
│   │                  gestures, accessibility tree builder
│   ├── button/        Button widgets (text, icon, segmented, split)
│   ├── data/          Data widgets (virtual list, tree)
│   ├── dialog/        Dialog & modal system
│   ├── display/       Display widgets (text, card, tabs, badge, chip, tooltip, divider)
│   ├── effects/       Visual effects (overlay with anchoring, animation)
│   ├── form/          Form controls (text field, checkbox, radio, toggle, select,
│   │                  slider, progress, textarea, password field)
│   ├── icons/         Icon rendering
│   ├── layout/        Layout engines (flex, grid, table, stack, box)
│   ├── menu/          Menu bar & context menu
│   └── nav/           Navigation (tabs, accordion, split view)
│
├── draw/              2D canvas abstraction: Canvas interface, Paint, Path,
│                      Color, Rect, Point, Shadow, Transform, TextStyle
│
├── anim/              Animation: Anim[T], LerpAnim[T], SpringAnim[T],
│                      easing functions, AnimGroup, AnimSeq
│
├── theme/             Theme interface, TokenSet, ColorScheme, TypographyScale,
│                      SpacingScale, RadiusScale, MotionSpec; LuxDark, LuxLight,
│                      LuxAuto, Slate, SlateLight, Override()
│
├── platform/          Platform interface; backends:
│   ├── glfw/          GLFW (cross-platform)
│   ├── windows/       Win32 native
│   ├── cocoa/         macOS Cocoa
│   ├── x11/           X11
│   ├── wayland/       Wayland
│   └── drm/           DRM/KMS (embedded)
│
├── surface/           External surface integration (video, 3D, custom) — planned
├── input/             Key, ModifierSet, Shortcut, MouseMsg, ScrollMsg,
│                      TouchMsg, IMEMsg, CursorKind, gesture
├── a11y/              AccessTree, A11yBridge interface, platform bridges
├── image/             ImageID, ImageStore, GPU texture management
├── validation/        Form validation helpers
├── dialog/            Native dialog utilities
├── fonts/             Bundled fonts (Noto Sans, Phosphor icons)
│
└── internal/          Framework internals (not public API)
    ├── gpu/           GPU device initialization, Renderer interface
    ├── render/        Rendering pipeline: scene graph, command recording,
    │                  layer batching, SceneCanvas
    ├── text/          Text shaping (go-text/typesetting), MSDF SDF rendering,
    │                  font atlas, BiDi (UAX #9), line breaking (UAX #14),
    │                  grapheme clusters, multi-line layout
    ├── hit/           Hit-testing & pointer target identification
    ├── loop/          Main app loop: message channel, frame scheduling
    └── wgpu/          wgpu/gogpu FFI bridge
```

## Rendering Pipeline

Each frame follows this sequence:

```
1. Drain message channel
   └─ update(model, msg) → newModel  (for each queued msg)

2. Animation tick
   └─ call Tick(dt) on every WidgetState implementing Animator
   └─ mark dirty widgets

3. View
   └─ view(model) → Element tree  (pure function)

4. Reconcile
   └─ diff new Element tree against previous tree (UID-based, O(n))
   └─ reuse WidgetState for unchanged widgets
   └─ call Widget.Render(ctx, state) for changed/new widgets

5. Layout
   └─ resolve constraints top-down, measure bottom-up
   └─ apply cached results for unchanged subtrees

6. Hit-test
   └─ dispatch pointer events to target widgets
   └─ update hover/pressed state

7. Accessibility tree update
   └─ build AccessTree from widget tree
   └─ sync with platform bridge (UIA / AT-SPI2 / NSAccessibility)

8. GPU render
   └─ record draw commands into SceneCanvas
   └─ upload dirty textures (images, font atlas)
   └─ submit command buffer to wgpu
   └─ present swapchain
```

## App Loop

The app loop runs exclusively on the **main goroutine**. This means:

- `update` and `view` are never called concurrently.
- No mutex is needed to access model state inside `update` or `view`.
- `app.Send` is the only thread-safe entry point from other goroutines.

The internal message channel has a fixed-size buffer. `Send` enqueues without blocking;
`TrySend` returns `false` if the buffer is full.

```
main goroutine: Init → platform.Run(callbacks)
                         │
                    ┌────▼────────────────┐
                    │  per-frame callback  │
                    │  1. drain msgs       │
                    │  2. anim tick        │
                    │  3. view             │
                    │  4. reconcile        │
                    │  5. layout           │
                    │  6. hit-test         │
                    │  7. a11y             │
                    │  8. GPU render       │
                    └─────────────────────┘

background goroutines: → app.Send(msg) → channel → main goroutine
```

## Widget Tree Reconciliation

The reconciler uses **UID-based diffing** (similar to React keys):

- Each `Widget` value is identified by its `UID` (a `uint64` assigned by the framework).
- On each frame, the new Element tree is compared against the previous one.
- If a widget's UID matches and it implements `Equatable`, `Equal()` is called — if `true`,
  the previous render output and state are reused without calling `Render()`.
- If the UID does not match or `Equal()` returns `false`, `Widget.Render(ctx, state)` is called
  and the returned state replaces the previous one.

## GPU Backend

lux uses **wgpu** (via `gogpu/wgpu`) as the primary GPU backend. It targets:

- Vulkan (Linux, Windows, Android)
- Metal (macOS, iOS)
- D3D12 (Windows)
- WebGPU (future browser target)

A GLFW + OpenGL 3.3 path serves as a fallback for systems without Vulkan/Metal/D3D12.

The wgpu integration lives in `internal/gpu` and `vendor_gogpu_wgpu` (vendored binding).

## Text Rendering

Text is rendered via two paths:

| Size | Method | Notes |
|------|--------|-------|
| ≥ 24dp | MSDF (Multi-channel SDF) | Sharp at any scale, GPU-rendered |
| < 24dp | Bitmap rasterisation | Pixel-perfect at small sizes |

Text shaping uses `go-text/typesetting` (GSUB, GPOS, OpenType layout). Unicode support:

- **BiDi** — UAX #9 via `internal/text/bidi.go`
- **Line breaking** — UAX #14 via `rivo/uniseg`
- **Grapheme clusters** — cursor-aware navigation via `rivo/uniseg`

## Dependency Graph (simplified)

```
app → ui → draw, anim, theme, input, a11y, image
      ui → internal/render, internal/hit
app → platform
app → internal/loop, internal/gpu
internal/render → internal/gpu, internal/text, draw
internal/text → go-text/typesetting, pierrec/msdf, rivo/uniseg
```

`anim`, `draw`, `input`, and `theme` depend only on stdlib (or each other with no cycles).
`internal/` packages are not part of the public API.
