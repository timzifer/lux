# RFC-012 — lux/inspector: Widget-Inspector über Vellum (PoC)

**Repository:** `github.com/timzifer/lux`

**Status:** Integriert (PoC)
**Version:** 0.1.0
**Datum:** 2026-03-28
**Zuletzt abgeglichen:** 2026-03-31
**Abhängig von:** RFC-001 (Core Architecture, §12), RFC-011 (Vellum-Protokoll)
**Berührt:** RFC-002 (Interaction & Layout), RFC-006 (Surface Semantics), RFC-007 (WGPU Rendering)
**Löst ab:** ToDo 6.7 (Inspector & Debugging)

---

### Implementierungsstatus

| Abschnitt | Status |
|-----------|--------|
| §3 Architektur | ✅ Implementiert |
| §4 Debug-Extensions | ✅ Implementiert |
| §5 Server-Seite | ✅ Implementiert |
| §6 Client-Seite (Inspector-Binary) | ✅ Implementiert (Basis-UI) |
| §7 Serialisierungsformat | ✅ Implementiert |
| §8 Phase A (Scene-Serialisierung) | ✅ Implementiert + Tests |
| §8 Phase B (Unix-Socket-Transport) | ✅ Implementiert + Tests |
| §8 Phase C (Debug-Extensions) | ✅ Implementiert + Tests |
| §8 Phase D (Inspector-UI) | ✅ Implementiert (Basis-Panels) |
| §9 Verifikation | ✅ 20 Tests bestanden |

---

## Inhaltsverzeichnis

1. [Motivation — Warum zusammenführen?](#1-motivation--warum-zusammenführen)
2. [Scope — Was der PoC abdeckt](#2-scope--was-der-poc-abdeckt)
3. [Architektur — Inspector als Vellum-Client](#3-architektur--inspector-als-vellum-client)
4. [Debug-Extensions auf Vellum](#4-debug-extensions-auf-vellum)
5. [Server-Seite: Integration in Lux](#5-server-seite-integration-in-lux)
6. [Client-Seite: Inspector-Binary](#6-client-seite-inspector-binary)
7. [Serialisierungsformat](#7-serialisierungsformat)
8. [Implementierungsphasen](#8-implementierungsphasen)
9. [Verifikation](#9-verifikation)
10. [Risiken & offene Fragen](#10-risiken--offene-fragen)

---

## 1. Motivation — Warum zusammenführen?

ToDo 6.7 definiert den Widget-Inspector als eigenständiges Arbeitspaket: Debug-Protocol via TCP/Unix-Socket, VTree-Streaming, Frame-Metriken, separates Binary. RFC-011 definiert mit Vellum ein allgemeines Remote-Rendering-Protokoll mit exakt denselben Transport-Mechanismen: Canvas-Stream über Sockets, Kanal-Architektur, AccessTree-Transport.

Die Beobachtung ist einfach: **Der Inspector braucht kein eigenes Protokoll — Vellum IST das Protokoll.**

RFC-011 §12.2 formuliert das explizit:

> Der Inspector ist ein Vellum-Client, der den Canvas-Stream liest und mit Debug-Overlays anreichert.

RFC-001 §12 definiert sechs Inspector-Features:

```
✦ Widget-Tree-Ansicht: Alle VNodes mit Props, UID, WidgetState-Typ
✦ Layout-Overlay: Bounds, Margins, Padding als visuelle Einblendung
✦ Paint-Highlighting: Welche Widgets wurden in diesem Frame neu gezeichnet?
✦ Event-Log: Input-Events und ihre Dispatch-Ziele in Echtzeit
✦ State-Dump: WidgetState als JSON für jeden Node
✦ Performance: Frame-Zeit, Layout-Zeit, Paint-Zeit pro Frame
```

Jedes dieser Features lässt sich auf Vellum-Mechanismen abbilden — der Canvas-Stream liefert das visuelle Abbild, der Control-Kanal (Kanal 0) transportiert Debug-Metadaten als Extensions. Ein eigenes Debug-Protocol wäre eine Parallelstruktur, die dieselben Probleme (Serialisierung, Transport, Framing) ein zweites Mal löst.

**Strategischer Vorteil:** Der Inspector als erster Vellum-Client validiert das gesamte Protokoll-Design. Wenn der Inspector funktioniert — Canvas-Roundtrip, AccessTree-Transport, Kanal-Architektur — ist Phase 1 von Vellum implizit abgeschlossen. Zwei Meilensteine für den Preis von einem.

---

## 2. Scope — Was der PoC abdeckt

### In Scope (aus RFC-011)

| Vellum-Komponente | Abschnitt | Beschreibung |
|---|---|---|
| Layer 1: Canvas-Kommando-Protokoll | RFC-011 §4.1 | Serialisierung aller `draw.Canvas`-Aufrufe |
| Kanal 0: Control-Stream | RFC-011 §5 | Handshake, AccessTree-Updates, Debug-Extensions |
| Transport: Unix-Domain-Socket | RFC-011 §5.1 | Lokaler Transport für Same-Machine-Inspector |
| CanvasEncoder (Server) | RFC-011 §11.1 | Interceptor auf `draw.Canvas` |
| CanvasDecoder (Client) | RFC-011 §11.2 | Stream → Canvas-Calls |

### In Scope (aus ToDo 6.7 / RFC-001 §12)

| Inspector-Feature | Quelle |
|---|---|
| Widget-Tree-Ansicht (VNodes, Props, UID, WidgetState) | RFC-001 §12 |
| Layout-Overlay (Bounds, Margins, Padding) | RFC-001 §12 |
| Paint-Highlighting (via DirtyTracker — Task 3.4 ✅) | RFC-001 §12, ToDo 3.4 |
| Frame-Metriken (Frame-Zeit, Layout-Zeit, Paint-Zeit) | RFC-001 §12 |
| Event-Log (Input-Events + Dispatch-Ziele) | RFC-001 §12 |
| State-Dump (WidgetState als JSON) | RFC-001 §12 |
| Inspector als separates Binary | RFC-001 §12 |

### Nicht in Scope (bleibt in RFC-011 für spätere Phasen)

- **Layer 2** (Layout-Constraint-Protokoll) — der Inspector braucht keine eigene Layout-Berechnung
- **Layer 3** (Interaktions-Protokoll) — der Inspector sendet keine Events an die App (read-only)
- **Layer 4** (Surface-Slot-Protokoll) — keine Surface-Komposition im Inspector
- **Netzwerk-Transport** (TCP/TLS, WebSocket) — nur Unix-Domain-Socket für lokales Debugging
- **Multi-User / Sessions** — Inspector ist immer ein Single-Client-Szenario
- **Latenz-Kompensation / Prediction** — lokal irrelevant
- **Asset-Streaming** (Fonts, Images) — Inspector nutzt lokalen Zugriff auf Assets
- **Browser-Client** (WASM) — kein Browser-basierter Inspector im PoC
- **Delta-Komprimierung** — kommt in späteren Vellum-Phasen

---

## 3. Architektur — Inspector als Vellum-Client

### 3.1 Überblick

```
┌─────────────────────────────────────────────────────────────────┐
│                        Lux-App (Server)                         │
│                                                                 │
│   Model ──► update(model, msg) ──► view(model)                  │
│                                        │                        │
│                                        ▼                        │
│                              ┌──────────────────┐               │
│                              │  ui.BuildScene   │               │
│                              └────────┬─────────┘               │
│                                       │                         │
│                              ┌────────▼─────────┐               │
│                              │  CanvasEncoder   │               │
│                              │  (Interceptor)   │               │
│                              └───┬──────────┬───┘               │
│                                  │          │                    │
│                           ┌──────▼──┐  ┌────▼───────────┐       │
│                           │ GPU     │  │ FrameBuffer    │       │
│                           │ (lokal) │  │ (serialisiert) │       │
│                           └─────────┘  └────┬───────────┘       │
│                                             │                   │
│                              ┌──────────────▼──────────┐        │
│                              │  Debug-Extension-       │        │
│                              │  Collector               │        │
│                              │  • DebugFrameInfo       │        │
│                              │  • DebugWidgetTree      │        │
│                              │  • DebugEventLog        │        │
│                              └──────────────┬──────────┘        │
│                                             │                   │
└─────────────────────────────────────────────┼───────────────────┘
                                              │
                                    Unix-Domain-Socket
                                   /tmp/lux-inspector.sock
                                              │
┌─────────────────────────────────────────────┼───────────────────┐
│                    Inspector-Binary (Client)  │                   │
│                                             │                   │
│                              ┌──────────────▼──────────┐        │
│                              │  CanvasDecoder          │        │
│                              │  + Debug-Extension-     │        │
│                              │    Parser               │        │
│                              └──────────────┬──────────┘        │
│                                             │                   │
│                    ┌────────────────────────┬┴──────────┐        │
│                    │                        │           │        │
│           ┌────────▼───────┐  ┌─────────────▼──┐  ┌────▼─────┐  │
│           │ Canvas-Replay  │  │ Debug-Data-    │  │ Overlay- │  │
│           │ (Scene-View)   │  │ Store          │  │ Renderer │  │
│           └────────────────┘  └────────────────┘  └──────────┘  │
│                                                                 │
│           ┌────────────────────────────────────────────────┐     │
│           │              Inspector-UI (Lux)               │     │
│           │  ┌──────────┐ ┌───────────┐ ┌──────────────┐  │     │
│           │  │ Widget-  │ │ Event-    │ │ Frame-       │  │     │
│           │  │ Tree     │ │ Log       │ │ Metriken     │  │     │
│           │  ├──────────┤ ├───────────┤ ├──────────────┤  │     │
│           │  │ State-   │ │ Layout-   │ │ Paint-       │  │     │
│           │  │ Inspector│ │ Overlay   │ │ Highlighting │  │     │
│           │  └──────────┘ └───────────┘ └──────────────┘  │     │
│           └────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Feature-Mapping auf Vellum-Mechanismen

| Inspector-Feature | Vellum-Mechanismus | Kanal |
|---|---|---|
| Widget-Tree | AccessTree über Kanal 0 + Debug-Extensions (UID, Props, WidgetState-Typ) | 0 |
| Layout-Overlay | Bounds aus AccessTree + Canvas-Stream-Replay mit Overlay-Rendering | 0 + 1 |
| Paint-Highlighting | DirtyTracker-Info als `DebugFrameInfo`-Extension | 0 |
| Event-Log | Debug-Extension: Event-Stream-Mirror (`DebugEventLog`) | 0 |
| State-Dump | Debug-Extension: WidgetState-Serialisierung (`DebugWidgetTree`) | 0 |
| Frame-Metriken | Debug-Extension: Timing-Daten in EndFrame-Kommando (`DebugFrameInfo`) | 1 |

### 3.3 Kanal-Nutzung im Inspector-Kontext

Von den fünf Vellum-Kanälen (RFC-011 §5) nutzt der Inspector-PoC nur zwei:

```
Inspector-Verbindung
    │
    ├── Kanal 0: Control-Stream (bidirektional)
    │     • Vereinfachter Handshake (keine Capability-Negotiation)
    │     • AccessTree-Updates (Widget-Tree für Inspector)
    │     • Debug-Extensions: DebugWidgetTree, DebugEventLog
    │
    └── Kanal 1: Canvas-Stream (Server → Client)
          • BeginFrame → Kommandos → EndFrame
          • DebugFrameInfo als Extension in EndFrame
          • Kein Delta-Compression (full frames)

    Nicht genutzt:
    ├── Kanal 2: Asset-Stream — Inspector nutzt lokale Assets
    ├── Kanal 3: Event-Stream — Inspector ist read-only
    └── Kanal 4: Surface-Stream — keine Surface-Komposition
```

---

## 4. Debug-Extensions auf Vellum

Debug-Extensions sind optionale Datenblöcke, die ein normaler Vellum-Client ignoriert. Der Inspector erkennt sie anhand reservierter Opcodes im Debug-Bereich (Opcode-Range `0xD0`–`0xDF`).

### 4.1 DebugFrameInfo

Wird als Extension in das `EndFrame`-Kommando eingebettet. Enthält Performance-Metriken und die Liste der in diesem Frame neu gezeichneten Widgets.

```go
// internal/vellum/debug.go

// DebugFrameInfo carries per-frame debug data.
// Opcode: 0xD0, attached to EndFrame.
type DebugFrameInfo struct {
    FrameID      uint64        // Monoton steigender Frame-Zähler
    FrameTime    time.Duration // Gesamtzeit: update + reconcile + layout + paint
    UpdateTime   time.Duration // Nur update(model, msg)
    ReconcileTime time.Duration // Nur reconcile (VTree-Diff)
    LayoutTime   time.Duration // Nur Layout-Pass
    PaintTime    time.Duration // Nur BuildScene + Canvas-Calls
    WidgetCount  uint32        // Gesamtzahl Widgets im VTree
    DirtyWidgets []uint64      // UIDs der in diesem Frame neu gezeichneten Widgets
}
```

**Erfassung im Server:** Die Zeitmessungen werden um die bestehenden Aufrufe in `app.Run` gelegt — `update()`, `reconciler.Reconcile()`, `ui.BuildScene()`. Der DirtyTracker (bereits implementiert, Task 3.4 ✅) liefert die `DirtyWidgets`-Liste.

### 4.2 DebugWidgetTree

Wird nach jedem Reconcile über Kanal 0 gesendet. Erweitert den AccessTree um Inspector-spezifische Daten: Widget-Typ, Props, State-Dump, Dirty-Status.

```go
// DebugWidgetTree extends the AccessTree with inspector data.
// Opcode: 0xD1, sent on Channel 0 after AccessTreeUpdate.
type DebugWidgetTree struct {
    Version uint64                     // Korrespondiert mit AccessTreeUpdate.Version
    Nodes   []DebugWidgetNode
}

type DebugWidgetNode struct {
    UID        uint64                  // Element-UID (korrespondiert mit AccessNodeID)
    TypeName   string                  // Go-Typ des Widgets, z.B. "ui.Button", "form.TextField"
    Props      map[string]string       // Widget-Props als Key-Value (String-Repräsentation)
    StateDump  string                  // WidgetState als JSON (leer wenn stateless)
    Bounds     draw.Rect               // Layout-Bounds in Screen-Koordinaten
    Padding    draw.Insets             // Padding (für Layout-Overlay)
    Margin     draw.Insets             // Margin (für Layout-Overlay)
    Dirty      bool                    // true wenn in diesem Frame neu gezeichnet
}
```

**Erfassung im Server:** Der Reconciler kennt bereits alle VNodes mit UIDs. `ui.BuildAccessTree()` baut den AccessTree — die Debug-Extension ergänzt die zusätzlichen Felder. Die Bounds kommen aus `EventDispatcher.BoundsForWidget()`, der State-Dump aus `WidgetState`-Serialisierung (analog zu `uitest.SerializeScene()`).

### 4.3 DebugEventLog

Spiegelt alle Input-Events mit ihren Dispatch-Zielen. Wird auf Kanal 0 gesendet, gebatched pro Frame.

```go
// DebugEventLog mirrors dispatched events for the inspector.
// Opcode: 0xD2, sent on Channel 0 after DebugWidgetTree.
type DebugEventLog struct {
    FrameID uint64
    Events  []DebugEvent
}

type DebugEvent struct {
    Timestamp  time.Duration           // Relativ zum Frame-Start
    Kind       string                  // "KeyPress", "MouseClick", "Scroll", "IMECommit", ...
    TargetUID  uint64                  // UID des Widgets, an das der Event dispatched wurde
    TargetType string                  // Typ-Name des Ziel-Widgets
    Detail     string                  // Kurzform: "key=Enter", "button=Left pos=120,340", ...
    Consumed   bool                    // true wenn das Widget den Event konsumiert hat
}
```

**Erfassung im Server:** `ui.EventDispatcher` sammelt bereits Events in `dispatcher.ResetEvents()` / `dispatcher.Dispatch()`. Die Debug-Extension serialisiert die gesammelten Events mit ihren Dispatch-Zielen vor dem Reset.

### 4.4 Optionalität

Die Debug-Extensions sind ein Opt-in. Der Server sendet sie nur, wenn ein Inspector verbunden ist und im Handshake `debugExtensions: true` angefordert hat. Ohne Inspector: kein Overhead.

Ein normaler Vellum-Client (RFC-011 Phase 2+) ignoriert Opcodes im `0xD0`–`0xDF`-Bereich — Forward-Compatibility gemäß RFC-011 §16.1.

---

## 5. Server-Seite: Integration in Lux

### 5.1 Aktivierung

`app.WithInspector()` (RFC-001 §12) wird zu `vellum.Inspect()` — eine `app.Option`, die den Vellum-Server im Debug-Modus startet:

```go
// In der App aktivieren (nur Debug-Builds empfohlen):
app.Run(model, update, view,
    app.WithTheme(theme.Default),
    vellum.Inspect("unix:///tmp/lux-inspector.sock"),
)
```

Intern passiert:

1. Ein `CanvasEncoder` wird als Interceptor auf `draw.Canvas` eingehängt
2. Ein Unix-Socket-Listener wird gestartet
3. Der `DebugExtensionCollector` wird an den Reconciler und EventDispatcher angebunden
4. Bei jedem Frame: Canvas-Stream + Debug-Extensions werden über den Socket gesendet

### 5.2 CanvasEncoder — Interceptor auf draw.Canvas

Der `CanvasEncoder` implementiert `draw.Canvas` und leitet jeden Aufruf an den echten Canvas durch, während er gleichzeitig in den `FrameBuffer` serialisiert (RFC-011 §11.1):

```go
// internal/vellum/encoder.go

// CanvasEncoder records all Canvas operations into a binary stream
// while forwarding them to the real (GPU) Canvas.
type CanvasEncoder struct {
    inner  draw.Canvas    // Der echte Canvas (render.SceneCanvas → GPU)
    buf    *FrameBuffer   // Serialisierter Stream
}

var _ draw.Canvas = (*CanvasEncoder)(nil)

func NewCanvasEncoder(inner draw.Canvas, buf *FrameBuffer) *CanvasEncoder {
    return &CanvasEncoder{inner: inner, buf: buf}
}

// ── Primitives ──────────────────────────────────────────────────

func (e *CanvasEncoder) FillRect(r draw.Rect, paint draw.Paint) {
    e.inner.FillRect(r, paint)
    e.buf.WriteOp(OpFillRect, r, paint)
}

func (e *CanvasEncoder) FillRoundRect(r draw.Rect, radius float32, paint draw.Paint) {
    e.inner.FillRoundRect(r, radius, paint)
    e.buf.WriteOp(OpFillRoundRect, r, radius, paint)
}

func (e *CanvasEncoder) FillRoundRectCorners(r draw.Rect, radii draw.CornerRadii, paint draw.Paint) {
    e.inner.FillRoundRectCorners(r, radii, paint)
    e.buf.WriteOp(OpFillRoundRectCorners, r, radii, paint)
}

func (e *CanvasEncoder) FillEllipse(r draw.Rect, paint draw.Paint) {
    e.inner.FillEllipse(r, paint)
    e.buf.WriteOp(OpFillEllipse, r, paint)
}

// ... analog für alle weiteren Canvas-Methoden:
// StrokeRect, StrokeRoundRect, StrokeRoundRectCorners, StrokeEllipse, StrokeLine,
// FillPath, StrokePath,
// DrawText, MeasureText, DrawTextLayout,
// DrawImage, DrawImageScaled, DrawImageSlice, DrawTexture,
// DrawShadow,
// PushClip, PushClipRoundRect, PushClipPath, PopClip,
// PushTransform, PopTransform, PushOffset, PushScale,
// PushOpacity, PopOpacity, PushBlur, PopBlur, PushLayer, PopLayer,
// Bounds, DPR, Save, Restore

// ── Frame-Markierungen ──────────────────────────────────────────

func (e *CanvasEncoder) BeginFrame(frameID uint64, bounds draw.Rect, dpr float32) {
    e.buf.WriteOp(OpBeginFrame, frameID, bounds, dpr)
}

func (e *CanvasEncoder) EndFrame(info *DebugFrameInfo) {
    if info != nil {
        e.buf.WriteOp(OpDebugFrameInfo, info)
    }
    e.buf.WriteOp(OpEndFrame)
}
```

### 5.3 Integration in app.Run

Die Integration in die bestehende Run-Loop (Auszug aus `app/run.go`) erfordert minimale Änderungen:

```go
// Innerhalb von runInternal(), nach Canvas-Erstellung:

// --- Bestehender Code: ---
canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))

// --- Neuer Code (nur wenn Inspector aktiv): ---
var encoder *vellum.CanvasEncoder
if inspectorActive {
    frameBuf := vellum.NewFrameBuffer()
    encoder = vellum.NewCanvasEncoder(canvas, frameBuf)
    canvas = encoder // Interceptor wird zum Canvas
}

// --- Bestehender Code (unverändert): ---
scene := ui.BuildScene(currentTree, canvas, activeTheme, w, h, ix, fm)
```

Die Zeitmessungen für `DebugFrameInfo` umschließen die bestehenden Aufrufe:

```go
// Pseudo-Code für Frame-Metriken-Erfassung:
t0 := time.Now()
newModel, cmd := update(currentModel, msg)
updateTime := time.Since(t0)

t1 := time.Now()
currentTree, _ = reconciler.Reconcile(newTree, ...)
reconcileTime := time.Since(t1)

t2 := time.Now()
scene := ui.BuildScene(currentTree, canvas, ...)
paintTime := time.Since(t2)

if encoder != nil {
    encoder.EndFrame(&vellum.DebugFrameInfo{
        FrameID:       frameCounter,
        FrameTime:     updateTime + reconcileTime + paintTime,
        UpdateTime:    updateTime,
        ReconcileTime: reconcileTime,
        PaintTime:     paintTime,
        DirtyWidgets:  reconciler.DirtyUIDs(),
    })
}
```

### 5.4 Bestehende Infrastruktur

Der PoC nutzt ausschließlich bestehende Lux-Infrastruktur:

| Komponente | Datei | Nutzung |
|---|---|---|
| `draw.Canvas` Interface | `draw/canvas.go` | Interceptor-Punkt für CanvasEncoder |
| `draw.Scene` Struct | `draw/canvas.go` | Scene ist die Draw-List — Referenz für Serialisierungsformat |
| `a11y.AccessTree` | `a11y/access_tree.go` | Widget-Tree-Transport über Kanal 0 |
| `ui.EventDispatcher` | `ui/element.go` | `RegisterWidgetBounds` / `BoundsForWidget` für Layout-Overlay |
| DirtyTracker | `ui/reconcile.go`, `ui/element.go` | Paint-Highlighting (`DirtyUIDs()`) |
| `uitest.SerializeScene()` | `uitest/` | Referenz für Scene-Serialisierung |
| `app.Run` + Options | `app/run.go` | Integrationspunkt für `vellum.Inspect()` |

---

## 6. Client-Seite: Inspector-Binary

### 6.1 Programm-Struktur

```
cmd/lux-inspector/
├── main.go           // Entry-Point: Connect + Lux-App starten
├── model.go          // Inspector-Model (aktueller Frame, selektierter Widget, ...)
├── update.go         // Inspector-Update (Frame empfangen, Widget selektieren, ...)
└── view.go           // Inspector-View (Panels)
```

Der Inspector ist selbst eine Lux-App — er nutzt das Framework, um den Framework-Stream zu visualisieren:

```go
// cmd/lux-inspector/main.go

func main() {
    addr := flag.String("addr", "unix:///tmp/lux-inspector.sock", "Inspector-Socket")
    flag.Parse()

    client, err := vellum.Connect(*addr, vellum.WithDebugExtensions())
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    model := NewInspectorModel(client)

    app.Run(model, inspectorUpdate, inspectorView,
        app.WithTitle("Lux Inspector"),
        app.WithSize(1400, 900),
        app.WithTheme(theme.LuxDark),
    )
}
```

### 6.2 Inspector-Model

```go
// cmd/lux-inspector/model.go

type InspectorModel struct {
    Client       *vellum.Client

    // Aktueller Frame-Zustand
    CurrentFrame *vellum.DecodedFrame   // Deserialisierter Canvas-Stream
    FrameInfo    *vellum.DebugFrameInfo // Metriken des letzten Frames
    WidgetTree   *vellum.DebugWidgetTree // Widget-Baum mit Debug-Daten
    EventLog     []vellum.DebugEvent     // Event-Historie (ring buffer, max 500)

    // UI-Zustand des Inspectors
    SelectedUID  uint64                 // Aktuell selektierter Widget im Tree
    ActivePanel  Panel                  // WidgetTree, EventLog, Metriken, State
    ShowOverlay  bool                   // Layout-Overlay ein/aus
    ShowDirty    bool                   // Paint-Highlighting ein/aus
    Paused       bool                   // Frame-Stream pausieren

    // Metriken-Historie
    FrameHistory []FrameMetric          // Letzte 120 Frames für Diagramm
}

type Panel int
const (
    PanelWidgetTree Panel = iota
    PanelEventLog
    PanelMetrics
    PanelState
)

type FrameMetric struct {
    FrameID       uint64
    FrameTime     time.Duration
    UpdateTime    time.Duration
    ReconcileTime time.Duration
    PaintTime     time.Duration
    WidgetCount   uint32
}
```

### 6.3 Inspector-Panels

Das Inspector-UI besteht aus drei Bereichen:

```
┌─────────────────────────────────────────────────────────────────┐
│  Lux Inspector                              [▶ Pause] [⟳ Reset]│
├─────────────────────────┬───────────────────────────────────────┤
│                         │                                       │
│   Canvas-Replay         │   Widget-Tree / Event-Log / Metriken  │
│   (live scene view)     │                                       │
│                         │   ┌─ Widget-Tree ──────────────────┐  │
│   ┌───────────────────┐ │   │ ▸ Window (uid:1)               │  │
│   │                   │ │   │   ▸ VStack (uid:2)             │  │
│   │   [App-Inhalt     │ │   │     ▸ Text "Hello" (uid:3)     │  │
│   │    mit optionalem │ │   │     ▸ Button "Click" (uid:4) ● │  │
│   │    Layout-Overlay] │ │   │       ▸ Text "Click" (uid:5)  │  │
│   │                   │ │   │   ▸ HStack (uid:6)             │  │
│   │                   │ │   │     ▸ Slider (uid:7)           │  │
│   └───────────────────┘ │   └────────────────────────────────┘  │
│                         │                                       │
│   Frame: 1247           │   ┌─ State-Inspector ──────────────┐  │
│   FPS: 60.0             │   │ uid:4 — ui.Button              │  │
│   Frame-Time: 2.1ms     │   │                                │  │
│   Widgets: 47           │   │ Props:                         │  │
│                         │   │   label: "Click me"            │  │
│                         │   │   disabled: false              │  │
│                         │   │                                │  │
│                         │   │ State:                         │  │
│                         │   │   { "pressed": false,          │  │
│                         │   │     "hoverT": 0.0 }            │  │
│                         │   │                                │  │
│                         │   │ Bounds: (120, 40, 200, 48)     │  │
│                         │   │ Padding: (8, 16, 8, 16)        │  │
│                         │   └────────────────────────────────┘  │
│                         │                                       │
├─────────────────────────┴───────────────────────────────────────┤
│  Frame-Metriken                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  ██▁▁██▁▁▁██▁▁▁▁██▁▁▁▁▁██▁▁▁▁██▁  (Frame-Time pro Frame) ││
│  │  2ms ──────────────────── 16ms                              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

**Linke Hälfte — Canvas-Replay:** Der deserialisierte Canvas-Stream wird auf einem Lux-Canvas abgespielt. Bei aktiviertem Layout-Overlay werden die Bounds, Padding und Margins des selektierten Widgets als halbtransparente Rechtecke darüber gezeichnet. Paint-Highlighting markiert dirty Widgets mit einer farbigen Umrandung.

**Rechte Hälfte — Debug-Panels:** Tree-Widget (bestehendes Lux `ui.Tree`) zeigt den Widget-Baum. Klick auf einen Node selektiert ihn — der State-Inspector darunter zeigt Props und State-Dump. Dirty-Widgets werden mit einem farbigen Punkt (●) markiert.

**Untere Leiste — Frame-Metriken:** Balkendiagramm der letzten 120 Frame-Zeiten, farbcodiert nach Phase (Update, Reconcile, Paint).

---

## 7. Serialisierungsformat

### 7.1 Grundformat (aus RFC-011 §4.1)

Binär, TLV-basiert:

```
┌──────────┬───────────────┬──────────────────┐
│ Opcode   │ Length         │ Payload           │
│ (1 Byte) │ (varint, 1-4) │ (Length Bytes)     │
└──────────┴───────────────┴──────────────────┘
```

### 7.2 Opcode-Zuordnung

```
// ── Frame-Kontrolle ─────────────────────────────────
0x01  BeginFrame(frameID uint64, bounds Rect, dpr float32)
0x02  EndFrame()

// ── Primitive (1:1 Mapping auf draw.Canvas) ──────────
0x10  FillRect(r Rect, paint Paint)
0x11  FillRoundRect(r Rect, radius float32, paint Paint)
0x12  FillRoundRectCorners(r Rect, radii CornerRadii, paint Paint)
0x13  FillEllipse(r Rect, paint Paint)
0x14  StrokeRect(r Rect, stroke Stroke)
0x15  StrokeRoundRect(r Rect, radius float32, stroke Stroke)
0x16  StrokeRoundRectCorners(r Rect, radii CornerRadii, stroke Stroke)
0x17  StrokeEllipse(r Rect, stroke Stroke)
0x18  StrokeLine(a Point, b Point, stroke Stroke)

// ── Paths ────────────────────────────────────────────
0x20  FillPath(path Path, paint Paint)
0x21  StrokePath(path Path, stroke Stroke)

// ── Text ─────────────────────────────────────────────
0x30  DrawText(text string, origin Point, style TextStyle, color Color)
0x31  DrawTextLayout(layout TextLayout, origin Point, color Color)

// ── Images & Textures ────────────────────────────────
0x40  DrawImage(img ImageID, dst Rect, opts ImageOptions)
0x41  DrawImageScaled(img ImageID, dst Rect, mode ImageScaleMode, opts ImageOptions)
0x42  DrawImageSlice(slice ImageSlice, dst Rect, opts ImageOptions)
0x43  DrawTexture(tex TextureID, dst Rect)

// ── Shadows ──────────────────────────────────────────
0x50  DrawShadow(r Rect, shadow Shadow)

// ── Clipping & Transform ─────────────────────────────
0x60  PushClip(r Rect)
0x61  PushClipRoundRect(r Rect, radius float32)
0x62  PushClipPath(path Path)
0x63  PopClip()
0x64  PushTransform(t Transform)
0x65  PopTransform()
0x66  PushOffset(dx float32, dy float32)
0x67  PushScale(sx float32, sy float32)

// ── Effects ──────────────────────────────────────────
0x70  PushOpacity(alpha float32)
0x71  PopOpacity()
0x72  PushBlur(radius float32)
0x73  PopBlur()
0x74  PushLayer(opts LayerOptions)
0x75  PopLayer()

// ── State ────────────────────────────────────────────
0x80  Bounds(r Rect)
0x81  DPR(dpr float32)
0x82  Save()
0x83  Restore()

// ── Control (Kanal 0) ────────────────────────────────
0xC0  Handshake(version uint32, debugExtensions bool)
0xC1  AccessTreeUpdate(version uint64, nodes []AccessNodeWire)

// ── Debug-Extensions (0xD0-0xDF) ─────────────────────
0xD0  DebugFrameInfo(...)
0xD1  DebugWidgetTree(...)
0xD2  DebugEventLog(...)
```

### 7.3 Payload-Encoding

Felder innerhalb der Payload werden ohne Padding sequentiell geschrieben:

| Typ | Encoding | Bytes |
|---|---|---|
| `float32` | IEEE 754, little-endian | 4 |
| `uint64` | Little-endian fixed | 8 |
| `uint32` | Little-endian fixed | 4 |
| `varint` | LEB128 | 1–4 |
| `string` | Length-prefixed (varint + UTF-8) | variabel |
| `Rect` | 4× float32 (X, Y, W, H) | 16 |
| `Point` | 2× float32 (X, Y) | 8 |
| `Color` | 4× uint8 (R, G, B, A) | 4 |
| `bool` | 1 Byte (0x00 / 0x01) | 1 |
| `Duration` | int64 Nanosekunden, little-endian | 8 |

### 7.4 Frame-Semantik

Ein Frame ist eine Sequenz von Kommandos, eingeschlossen in `BeginFrame` / `EndFrame`:

```
BeginFrame(frameID=1247, bounds=Rect{0,0,1920,1080}, dpr=2.0)
  FillRect(...)
  PushClip(...)
    FillRoundRect(...)
    DrawText(...)
  PopClip()
  DrawShadow(...)
  PushOpacity(0.5)
    FillRect(...)
  PopOpacity()
  [DebugFrameInfo(...)]    ← nur wenn Inspector verbunden
EndFrame()
```

Der Client darf den Frame erst anzeigen, wenn `EndFrame` empfangen wurde — das garantiert einen konsistenten visuellen Zustand (kein Tearing).

---

## 8. Implementierungsphasen

### Phase A — Scene-Serialisierung (~1 Woche)

**Ziel:** Canvas-Stream in eine Datei schreiben, zweiter Prozess deserialisiert — identischer Output.

**Vorgehen:**

1. `FrameBuffer` implementieren: TLV-basierter Byte-Buffer mit `WriteOp(opcode, fields...)` und `ReadOp() (opcode, fields, err)`
2. `CanvasEncoder` schreiben: Implementiert `draw.Canvas`, leitet an `inner` durch und schreibt in `FrameBuffer`
3. `CanvasDecoder` schreiben: Liest `FrameBuffer`, ruft die entsprechenden `draw.Canvas`-Methoden auf dem Ziel-Canvas auf
4. Roundtrip-Test: `uitest.SerializeScene()`-Referenz nutzen — Scene über Encoder serialisieren, über Decoder deserialisieren, Scene-Output vergleichen

**Dateien:**

```
internal/vellum/
├── opcode.go       // Opcode-Konstanten (OpFillRect, OpBeginFrame, ...)
├── wire.go         // Payload-Encoding: WriteRect, ReadRect, WriteColor, ...
├── framebuf.go     // FrameBuffer: WriteOp, ReadOp, Bytes, Reset
├── encoder.go      // CanvasEncoder (draw.Canvas → FrameBuffer)
└── decoder.go      // CanvasDecoder (FrameBuffer → draw.Canvas)
```

**Verifikation:**

```go
func TestCanvasRoundtrip(t *testing.T) {
    // 1. Lux-Scene über CanvasEncoder aufzeichnen
    buf := vellum.NewFrameBuffer()
    recorder := vellum.NewCanvasEncoder(nil, buf) // nil inner = record-only
    buildTestScene(recorder) // FillRect, DrawText, PushClip, ...

    // 2. Aufzeichnung über CanvasDecoder auf neuen Canvas abspielen
    var replayed draw.Scene
    target := render.NewSceneCanvas(800, 600)
    decoder := vellum.NewCanvasDecoder(target)
    decoder.Decode(buf.Bytes())
    replayed = target.Finish()

    // 3. Vergleich
    assertEqual(t, originalScene, replayed)
}
```

### Phase B — Unix-Socket-Transport (~1 Woche)

**Ziel:** Live-Stream über Unix-Domain-Socket. Inspector-Prozess empfängt Frames in Echtzeit.

**Vorgehen:**

1. Server: `vellum.Serve("unix:///tmp/lux-inspector.sock")` — Listener, Accept, Frame-Loop
2. Vereinfachter Handshake: Client sendet `Handshake(version=1, debugExtensions=true)`, Server antwortet mit `Handshake(version=1, debugExtensions=true)`
3. Frame-Streaming: Server sendet nach jedem `EndFrame` den `FrameBuffer` über den Socket
4. Framing über Socket: Length-prefixed Messages (4 Byte uint32 Big-Endian + Payload)

**Dateien:**

```
internal/vellum/
├── server.go       // vellum.Serve(), Accept-Loop, Frame-Broadcast
├── client.go       // vellum.Connect(), Handshake, Frame-Receive-Loop
└── transport.go    // Length-prefixed framing über net.Conn
```

**Verifikation:**

```bash
# Terminal 1: Lux-App mit Inspector-Socket starten
go run ./examples/kitchensink -inspect unix:///tmp/lux-inspector.sock

# Terminal 2: Dump-Tool zum Verifizieren (Vorstufe des Inspectors)
go run ./cmd/lux-inspect-dump -addr unix:///tmp/lux-inspector.sock
# Output: "Frame 1: 4.2 KB, 23 ops | Frame 2: 4.1 KB, 23 ops | ..."
```

### Phase C — Debug-Extensions (~1–2 Wochen)

**Ziel:** Inspector empfängt Widget-Tree, Event-Log und Frame-Metriken in Echtzeit.

**Vorgehen:**

1. `DebugFrameInfo` in EndFrame: Zeitmessungen in `app.Run` einbauen, DirtyTracker-UIDs sammeln
2. `DebugWidgetTree`: AccessTree serialisieren, um Inspector-spezifische Felder erweitern (TypeName, Props, StateDump)
3. `DebugEventLog`: EventDispatcher-Events nach Dispatch serialisieren
4. Kanal-0-Framing: Debug-Extensions werden als separate Messages auf demselben Socket gesendet, mit Kanal-Header (1 Byte Kanal-ID vor jeder Length-prefixed Message)

**Dateien:**

```
internal/vellum/
├── debug.go            // DebugFrameInfo, DebugWidgetTree, DebugEventLog Typen
├── debug_collector.go  // DebugExtensionCollector: sammelt Daten aus Reconciler + Dispatcher
├── debug_wire.go       // Serialisierung der Debug-Extensions
```

**Verifikation:**

```go
func TestDebugWidgetTree(t *testing.T) {
    // App mit Inspector starten
    // Widget-Tree empfangen
    // Prüfen: Jeder Node hat UID, TypeName, Bounds
    // Prüfen: Button-Node hat Props["label"] == "Click me"
    // Prüfen: Mindestens ein Node hat Dirty==true nach Interaktion
}
```

### Phase D — Inspector-UI (~2 Wochen)

**Ziel:** Vollständiges Inspector-Binary mit allen sechs Features aus RFC-001 §12.

**Vorgehen:**

1. **Widget-Tree-Panel:** Lux `ui.Tree`-Widget mit AccessTree-Daten füttern. Klick auf Node → Selektion
2. **Layout-Overlay:** Bounds des selektierten Widgets als halbtransparentes Overlay auf den Canvas-Replay zeichnen. Margin (orange), Padding (grün), Content (blau) — analog zu Browser DevTools
3. **Frame-Metriken-Dashboard:** Balkendiagramm der letzten N Frames, farbcodiert (Update=blau, Reconcile=gelb, Paint=rot)
4. **Event-Log-Panel:** Scrollbare Liste der letzten Events mit Timestamp, Kind, Target, Consumed-Status
5. **State-Inspector:** JSON-View für den State-Dump des selektierten Widgets. Readonly — kein State-Editing im PoC

**Dateien:**

```
cmd/lux-inspector/
├── main.go
├── model.go
├── update.go
├── view.go
├── panel_tree.go       // Widget-Tree-Panel
├── panel_event.go      // Event-Log-Panel
├── panel_metrics.go    // Frame-Metriken-Dashboard
├── panel_state.go      // State-Inspector-Panel
└── overlay.go          // Layout-Overlay + Paint-Highlighting
```

---

## 9. Verifikation

### 9.1 Automatisierte Tests

| Test | Package | Prüft |
|---|---|---|
| `TestOpcodeRoundtrip` | `internal/vellum` | Jeder Opcode: Encode → Decode → identische Felder |
| `TestCanvasRoundtrip` | `internal/vellum` | Vollständige Scene: Encoder → Decoder → Scene-Vergleich |
| `TestFrameBufferTLV` | `internal/vellum` | TLV-Format: varint-Length, Payload-Boundaries |
| `TestSocketTransport` | `internal/vellum` | Unix-Socket: Connect, Handshake, Frame-Receive |
| `TestDebugFrameInfo` | `internal/vellum` | DebugFrameInfo: Timing-Daten korrekt serialisiert |
| `TestDebugWidgetTree` | `internal/vellum` | DebugWidgetTree: UIDs, TypeNames, Bounds korrekt |
| `TestDebugEventLog` | `internal/vellum` | DebugEventLog: Events mit Dispatch-Zielen korrekt |
| Golden-File-Tests | `uitest` | Serialisierter Scene-Stream gegen gespeicherte Referenz |

### 9.2 Manuelle Verifikation

1. **KitchenSink + Inspector:** `examples/kitchensink` mit `-inspect` Flag starten. `cmd/lux-inspector` verbinden. Prüfen:
   - Canvas-Replay zeigt identisches Bild
   - Widget-Tree zeigt alle Widgets mit korrekten Typen
   - Klick auf Widget in der App → Dirty-Highlighting im Inspector
   - Event-Log zeigt Input-Events mit korrekten Dispatch-Zielen
   - Frame-Metriken-Diagramm zeigt plausible Werte

2. **Frame-Größe messen:** Ziel ist <10 KB pro Frame für eine typische UI (RFC-011 §15, Phase 1). Das `lux-inspect-dump`-Tool gibt die Frame-Größe pro Frame aus.

3. **Overhead messen:** App mit und ohne Inspector starten, Frame-Zeiten vergleichen. Der `CanvasEncoder`-Overhead sollte <0.5 ms pro Frame sein (serialisiert in einen Pre-allocated Buffer, keine Allokationen im Hot-Path).

---

## 10. Risiken & offene Fragen

### 10.1 MeasureText-Semantik

`draw.Canvas.MeasureText()` ist eine Query (gibt `TextMetrics` zurück), kein Zeichenbefehl. Der `CanvasEncoder` kann sie aufzeichnen, aber der `CanvasDecoder` kann sie nicht sinnvoll replaying — er hat keinen Font-Stack. Für den Inspector-PoC ist das akzeptabel: `MeasureText` wird im Stream aufgezeichnet, aber beim Replay ignoriert. Der Inspector braucht keine eigene Textmessung — er zeigt die Bounds, die der Server berechnet hat.

### 10.2 Path-Serialisierung

`draw.Path` ist ein Interface. Für die Serialisierung muss der Encoder die Path-Daten als Sequenz von Move/Line/Curve/Close-Kommandos schreiben. Das erfordert entweder ein `PathIterator`-Interface auf `draw.Path` oder eine Konvention, dass Paths ihre Segmente exponieren. Falls `draw.Path` das nicht unterstützt, werden `FillPath`/`StrokePath` im PoC nicht serialisiert (kein Show-Stopper — die meisten UI-Widgets nutzen Rect/RoundRect/Ellipse).

### 10.3 Canvas-API-Stabilität

Der Inspector koppelt direkt an die Canvas-API. Jede neue Canvas-Methode erfordert einen neuen Opcode und Anpassungen in Encoder + Decoder. Die Canvas-API ist als "stabilstes öffentliches Interface" deklariert (RFC-001 §6.2), aber nicht eingefroren. Mitigation: Der Opcode-Raum ist groß genug (256 Opcodes im 1-Byte-Format, erweiterbar). Unbekannte Opcodes werden vom Decoder übersprungen (Length ermöglicht Skip).

### 10.4 Scene vs. Canvas

Lux hat zwei Repräsentationen: den Canvas-Stream (High-Level: `FillRoundRect`, `DrawText`) und die `draw.Scene` (Low-Level: tessellierte Geometrie, Atlas-Glyphen). Der Inspector serialisiert den Canvas-Stream, nicht die Scene. Das bedeutet, der Inspector-Replay sieht dieselben Canvas-Calls, produziert aber eine eigene Scene mit eigenem Font-Atlas. Visuell sollte das Ergebnis identisch sein — aber es ist kein Bit-für-Bit-Match auf GPU-Ebene. Für den Inspector ist das ausreichend.

### 10.5 Kein Rückkanal im PoC

Der Inspector ist im PoC read-only — er kann keine Events an die App senden (kein Layer 3). Funktionen wie "Widget selektieren durch Klick in den Canvas-Replay" erfordern ein lokales Hit-Testing im Inspector, nicht einen Roundtrip zum Server. Das ist machbar (der Inspector hat die Bounds aus `DebugWidgetTree`), aber erhöht die Komplexität. Im PoC wird die Widget-Selektion ausschließlich über das Tree-Panel gemacht.

### 10.6 Leitfragen

1. **Soll `vellum.Inspect()` auch über TCP erreichbar sein?** Für lokales Debugging reicht Unix-Socket. Für Remote-Debugging (Inspector auf einem anderen Rechner) wäre TCP nötig — aber das widerspricht dem PoC-Scope. Empfehlung: Nur Unix-Socket im PoC, TCP in Phase 2 von Vellum.
2. **Soll der Inspector State-Editing unterstützen?** Das würde einen Rückkanal erfordern (Inspector → App). Empfehlung: Nein im PoC. State-Editing ist ein Feature für einen späteren "DevTools"-Ausbau.
3. **Overlay-Rendering: Canvas-Replay oder AccessTree-Only?** Das Layout-Overlay braucht die Bounds. Diese kommen aus `DebugWidgetTree.Bounds`. Der Canvas-Replay zeigt das visuelle Abbild. Beides wird composited: Canvas-Replay als Hintergrund, Overlay als halbtransparente Ebene darüber.

---

*RFC-012 — Draft. Feedback und Änderungsvorschläge bitte als Issue gegen dieses Dokument.*
