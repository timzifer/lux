# Plan: RFC-007-lux-WGPU-rendering.md

## Context

Lux hat bereits eine wgpu-Architektur (`internal/wgpu/`, `internal/gpu/wgpu_renderer.go`), aber:
- `native.go`: Alle Methoden sind TODOs — kein echter wgpu-native-Call implementiert
- `gogpu.go`: Pure No-Op-Stubs, nutzt nicht das `github.com/gogpu/wgpu`-Paket
- `wgpu_renderer.go`: Strukturell korrekt, aber pro Frame Heap-Allokationen (`make([]float32, ...)`),
kein Buffer-Pooling, kein Scissor-Support, Text-Shader ohne Per-Glyph-Farbe
- Keine Validation-Kontrolle, keine CGo-Overhead-Strategie

## Vorgehen

RFC-007-lux-WGPU-rendering.md als Datei im Root erstellen (konsistent mit RFC-001..006).
Die Datei wird auf Deutsch verfasst, im selben Stil wie die bestehenden RFCs.

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
- Blur-Pipeline: Gaussian Blur via Compute-Shader oder Multi-Pass (TODO)

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
- TODO: Blur via Compute-Shader
- TODO: Multi-Window-Support

### Anhang: Kritische Dateien
- `internal/wgpu/wgpu.go` — Interface-Definitionen (stabil, kaum Änderungen)
- `internal/wgpu/gogpu.go` — Hauptarbeit Phase A
- `internal/wgpu/native.go` — Hauptarbeit Phase D
- `internal/gpu/wgpu_renderer.go` — Hauptarbeit Phase B+C
- `internal/gpu/wgpu_shaders.go` — Shader-Anpassungen (Per-Glyph-Color etc.)
- `app/defaults_*.go` — Build-Tag-Routing
 