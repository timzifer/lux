# RFC-007 — lux: WGPU Rendering Pipeline

**Repository:** `github.com/timzifer/lux`
**Status:** Teilweise integriert
**Version:** 0.1.0
**Datum:** 2026-03-19
**Zuletzt abgeglichen:** 2026-03-25
**Abhängig von:** RFC-001 (Core Architecture)

---

### Implementierungsstatus

| Phase | Status | Anmerkung |
|-------|--------|-----------|
| Phase A: gogpu-Backend | ✅ Integriert | `internal/wgpu/gogpu.go` (1227 Zeilen), vollständig via `github.com/gogpu/wgpu` |
| Phase B: Performance-Fundament | ✅ Integriert | Buffer-Pooling, Scissor-Rects |
| Phase C: Geometry Batcher | ✅ Integriert | Draw-Call-Merging, Instanced Text |
| Phase D: wgpu-native Backend | ⏳ Wartend | `native.go` minimal (3 TODOs), wartet auf libwgpu_native-Verfügbarkeit |
| Phase E: Erweiterte Features | ✅ Integriert | Surface-Pipeline, Gradient-Pipeline, WGPU-Cube Demo |
| Phase F: Blur + Multi-Window | ✅ Integriert | Separabler 2-Pass Gaussian Blur, Multi-Window-Support |
| Phase G: Visual Effects | ✅ Integriert | Soft Shadows, Frosted Glass, Opacity, Elevation, Vibrancy, Inner Shadow, Grain, Glow |

---

## Context

Lux nutzt wgpu als primären Renderer (`internal/wgpu/`, `internal/gpu/wgpu_renderer.go`):
- `gogpu.go`: Voll implementiertes pure-Go Backend via `github.com/gogpu/wgpu` (Windows DX12, macOS Metal, Linux Vulkan)
- `native.go`: Minimale Stubs — wartet auf libwgpu_native-Bibliothek (Phase D)
- `wgpu_renderer.go`: Vollständiger Renderer mit Scissor, Shadows, Blur, Gradients, Opacity, Elevation, Vibrancy
- Mehrere WGSL-Shader: Rect, Text, Surface, Gradient, Blur, Shadow

## RFC-Struktur (Gliederung)

### §1 Motivation & Ziel
- Warum wgpu: Cross-Platform (Vulkan/Metal/D3D12/WebGPU), ein Shader-Dialekt (WGSL)
- IST-Zustand: OpenGL 3.3 als primärer Renderer, wgpu-Stubs vorhanden
- SOLL-Zustand: wgpu als primärer Renderer, OpenGL als Legacy-Fallback

### §2 Backend-Strategie: gogpu → native
- Phase 1: `github.com/gogpu/wgpu` (pure Go, einfacher Einstieg, kein CGo)
- Phase 2: wgpu-native via CGo per Build-Tag `-tags wgpunative`
- Abstraktion `internal/wgpu/wgpu.go` bleibt stabil, nur Implementations wechseln
- Build-Tag-Matrix dokumentieren

### §3 Pipeline-Management
- Pipelines in Init() erstellen (bereits vorhanden) — formalisieren
- Pipeline-Cache-Strategie: Shader → Pipeline-Descriptor → Hash → Cache
- Shader-Precompilation: WGSL → SPIR-V offline für native Backend
- Hot-Reload im Debug-Modus

### §4 Geometry Batcher
- Problem: Aktuell separate Draw-Calls pro Typ, `make([]float32)` pro Frame
- Lösung: Ring-Buffer-Pool (2-3 Frames), pre-allokierte Slices, Grow-Only
- Unified Vertex Buffer für Rects + Text (gemeinsamer Buffer, Offset-basiert)
- Draw-Call-Merging: Aufeinanderfolgende gleiche Pipeline → 1 Draw
- Instanced Rendering für Rects (schon da), erweitern auf Text (per-glyph color als Vertex-Attribut)

### §5 Buffer-Management
- Aktuell: `make([]float32, 0, n)` pro Frame → GC-Druck
- Lösung: `sync.Pool` oder dedizierter Frame-Allocator
- GPU-Buffer-Strategie: Oversize-Allocate, nur bei Überschreitung neu erstellen
- Staging Buffer vs Direct Write (MAP_WRITE vs queue.WriteBuffer)

### §6 Validation-Toggle
- wgpu Validation im Debug-Modus (default): Device-Deskriptor mit Validation
- Release-Modus: Validation aus (`-tags release` oder `!debug`)
- Error-Callback registrieren für Device-Lost / OOM
- Logging-Level: Verbose im Debug, nur Errors im Release

### §7 CGo-Overhead-Minimierung (nur wgpu-native)
- Problem: Jeder CGo-Call ~200ns Overhead (goroutine-to-C context switch)
- Batch-Pattern: Mehrere wgpu-Calls in einem CGo-Aufruf bündeln
- unsafe.Pointer statt C-Type-Konvertierungen wo möglich
- Callbacks minimieren: Push-Modell statt Poll, Completion via Channel
- Alternative: wgpu-native Calls über Shared Memory / Command-Buffer

### §8 Fehlende Renderer-Features
- ✅ Scissor/Clip-Rects: `RenderPass.SetScissorRect()` für UI-Clipping (Phase B)
- ✅ Per-Glyph-Farbe: Vertex-Attribut statt Uniform (Phase A)
- ✅ Surface-Pipeline: Texture-Blit für externe Surfaces (Phase E)
- ✅ Atlas-Resize: Texture-Neuerststellung + BindGroup-Update bei Atlas-Wachstum (Phase A)
- ✅ Gradient-Support: Linear/Radial Gradients als eigene Pipeline (Phase E)
- ✅ Blur-Pipeline: Gaussian Blur via Fragment-Shader, 2-Pass separabel (Phase F)
- ✅ Multi-Window: Per-Window Surface/Renderer, sekundäre Fenster (Phase F)
- ✅ Soft Shadows (Box Shadow) — SDF-basierter Shadow-Shader mit Gauss'schem Falloff (Phase G)
- ✅ Opacity — Stack-basierter Opacity-Multiplikator via `PushOpacity`/`PopOpacity` (Phase G)
- ✅ Frosted Glass Overlays, Vibrancy, Glow — `FrostedGlass()`, `Vibrancy()`, `GlowBox()` (Phase G)

### §9 Present-Mode & Frame-Pacing
- Fifo (VSync) als Default (bereits gesetzt)
- Mailbox für Low-Latency-Szenarien
- Immediate für Benchmarking
- Adaptive: Mailbox wenn verfügbar, Fallback Fifo
- Frame-Pacing: RequestFrame + Dirty-Flag = keine unnötigen Frames

### §10 Error-Handling & Robustheit
- Device-Lost-Recovery: Surface neu erstellen, Pipelines neu bauen
- OOM: Buffer-Größe limitieren, Fallback auf kleinere Atlanten
- Surface-Timeout: Graceful Retry bei GetCurrentTexture-Fehler

### §11 Phasenplan (Implementierung)

**Phase A: gogpu-Backend funktionsfähig machen**
- `internal/wgpu/gogpu.go` mit `github.com/gogpu/wgpu` implementieren
- Alle Interface-Methoden an echte gogpu-Calls anbinden
- Basis-Rendering verifizieren (Rects + Text + MSDF)

**Phase A – Bekannte Einschränkungen (TODO):**
- **MSDF Glyph-Alignment:** Große MSDF-Texte zeigen vertikale Verschiebungen einzelner
  Buchstaben (z.B. "S" tiefer als andere). Ursache vermutlich im Bearing/Baseline-Roundtrip
  zwischen `text.InsertMSDF()` und `canvas.drawTextTextured()`. Muss debuggt werden —
  betrifft alle Renderer, nicht nur gogpu (GDI-Renderer hat kein MSDF).
- **MSDF Corner-Artefakte:** Weiße/schwarze Pixel an scharfen Ecken (e/a-Öffnungen).
  `median3()` versagt an MSDF-Kanalübergängen. Langfristige Lösung: **MTSDF (4-Kanal)**
  statt MSDF (3-Kanal) in `github.com/pierrec/msdf` generieren. Die 4. Kanal-Information
  (true signed distance) eliminiert Corner-Artefakte ohne Shader-Hacks. Alternativ:
  Chlumsky-Error-Correction im Shader, benötigt aber Zusatzdaten im Atlas.
- **naga HLSL Backend:** `textureDimensions()` erzeugt undeklarierten Identifier `_dim_w`
  auf DX12. Workaround: Atlas-Größe als Uniform übergeben. Upstream-Bug in gogpu/naga.

**Phase B: Performance-Fundament**
- Buffer-Pooling (Ring-Buffer, Frame-Allocator)
- Scissor-Rects für Clipping

**Phase C: Geometry Batcher**
- Draw-Call-Merging
- Unified Vertex Buffer
- Instanced Text-Rendering

**Phase D: wgpu-native Backend (optional, `-tags wgpunative`)**
- `internal/wgpu/native.go` mit echten C-Calls implementieren
- CGo-Batch-Patterns
- Validation-Toggle
- Shader-Precompilation

**Phase E: Erweiterte Features** ✅ (Surface-Pipeline, Gradient-Pipeline, WGPU-Cube)
- ✅ Surface-Pipeline für externes Content (Texture-Registry, Blit-Shader, `RegisterSurfaceTexture` API)
- ✅ Gradient-Pipeline (Linear + Radial, bis 8 Stops, SDF-Rounded-Corners, per-Gradient Uniform mit 512-Byte-Stride)
- ✅ WGPU-Cube Demo (`pyramid_wgpu.go`, Offscreen-Render mit Depth/Stencil, Index-Buffer, Back-Face-Culling)
- ✅ wgpu-Interface erweitert: `VertexFormatFloat32x3`, `DepthStencilState`, `SetIndexBuffer`/`DrawIndexed`, `CullMode`/`FrontFace`
- ✅ `ui.GradientRect` Element + Canvas-Routing (`FillRect`/`FillRoundRect` → Gradient-Pipeline bei Gradient-Paint)
- ✅ Build-Tag-Matrix: `pyramid_wgpu.go` (gogpu), `pyramid.go` (OpenGL, Linux/macOS), `pyramid_noop.go` (nogui/Windows-default)

**Phase F: Blur + Multi-Window** ✅
- ✅ Blur via Fragment-Shader (separabler 2-Pass Gaussian, Ping-Pong zwischen 3 Offscreen-Texturen)
  - `wgslBlurShader`: Fullscreen-Triangle, `textureSample`-basiert (DX12/HLSL-kompatibel)
  - `wgslBlurBlitShader`: Blit-Shader für Rückprojektion auf die Surface
  - Per-Region Radius (256-Byte-aligned Uniform-Buffer-Offsets pro BlurRegion)
  - Scissor-basierte Region-Isolation: unblurred Scene → Surface, dann blurred Overlay per Region
  - `PushBlur(radius)`/`PopBlur()` auf Canvas-API, `BlurRegion` in Scene, `ui.BlurBox` Widget
- ✅ Multi-Window-Support
  - `app.WindowID`, `OpenWindow`/`CloseWindow` Commands, `WindowOpenedMsg`/`WindowClosedMsg`
  - `platform.MultiWindowPlatform` Interface (optional, kein Breaking Change)
  - Win32: `CreateWindow`/`DestroyWindow`, separater `secondaryWindowProc` (kein PostQuitMessage)
  - `gpu.WindowRenderer` Interface für per-Window Rendering
  - KitchenSink Demo: Blur-Section (5 Radii) + Multi-Window-Section (Open/Close)

**Phase G: Visual Effects Pipeline** ✅

Die folgenden Effekte bauen auf der bestehenden Blur-Infrastruktur auf und
transformieren die UI von "funktional" zu "subtle-fancy" (Premium-Feel).

*Tier 1 — Sofortiger Premium-Effekt:*

- ✅ **Soft Shadows (Box Shadow)** — SDF-basierter Shadow-Shader (`wgslShadowShader`),
  `DrawShadow` mit Color, BlurRadius, SpreadRadius, OffsetX/Y, Radius, Inset.
  Shadows werden vor Rects gerendert (Behind-Content). Eigene GPU-Pipeline.
  Dateien: `internal/gpu/wgpu_renderer.go`, `internal/gpu/wgpu_shaders.go`,
  `internal/render/canvas.go`, `draw/paint.go`.

- ✅ **Frosted Glass Overlays** — `ui.FrostedGlass` Widget: `PushBlur()` +
  halbtransparenter Tint + Overlay-Rendering. Blurred Backdrop + scharfer
  Content. Dateien: `ui/element.go`.

- ✅ **Opacity (PushOpacity/PopOpacity)** — Stack-basierter Opacity-Multiplikator
  in SceneCanvas. `effectiveOpacity()` wird auf alle Fill-Operationen angewendet.
  Dateien: `internal/render/canvas.go`.

*Tier 2 — Subtile Verfeinerungen:*

- ✅ **Elevation (Hover-responsive Shadows)** — `ui.ElevationBox` (Rest/Hover/Press
  Shadow-States mit `draw.LerpShadow`-Interpolation), `ui.ElevationCard`
  (Theme-Presets Low/High/None). Dateien: `ui/element.go`.

- ✅ **Tinted Blur (Vibrancy)** — `ui.Vibrancy` (Accent-Tinted FrostedGlass),
  `ui.TintedBlur` (expliziter Alias). Rein kompositorisch über FrostedGlass.
  Dateien: `ui/element.go`.

- ✅ **Inner Shadow / Inset** — `ui.InnerShadowBox` mit `Shadow.Inset=true`.
  GPU berechnet invertierte SDF-Fade von Kanten nach innen. Overlay-Rendering
  für korrekte Z-Ordnung. Dateien: `ui/element.go`, `internal/gpu/wgpu_shaders.go`.

*Tier 3 — Polish:*

- ✅ **Subtle Noise/Grain Texture** — `noise_hash()` im Rect-Fragment-Shader
  für Anti-Aliasing-Dither. Verhindert Banding bei Gradients.
  Dateien: `internal/gpu/wgpu_shaders.go`.

- ✅ **Glow (Focus Ring)** — `ui.GlowBox` / `ui.Glow`: Shadow-Pipeline mit
  Offset=0, Spread=0, Accent-Farbe. Weicher äußerer Schein um fokussierte
  Elemente. Dateien: `ui/element.go`.

### Anhang: Kritische Dateien
- `internal/wgpu/wgpu.go` — Interface-Definitionen (stabil, kaum Änderungen)
- `internal/wgpu/gogpu.go` — Hauptarbeit Phase A
- `internal/wgpu/native.go` — Hauptarbeit Phase D
- `internal/gpu/wgpu_renderer.go` — Hauptarbeit Phase B+C
- `internal/gpu/wgpu_shaders.go` — Shader-Anpassungen (Per-Glyph-Color etc.)
- `app/defaults_*.go` — Build-Tag-Routing
 