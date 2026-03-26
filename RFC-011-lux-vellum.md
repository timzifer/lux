# RFC-011 — lux/vellum: Remote-Rendering-Protokoll

**Repository:** `github.com/timzifer/lux`

**Status:** Theoretical
**Version:** 0.1.0
**Datum:** 2026-03-26
**Zuletzt abgeglichen:** 2026-03-26
**Abhängig von:** RFC-001 (Core Architecture), RFC-002 (Interaction & Layout), RFC-006 (Surface Semantics), RFC-007 (WGPU Rendering)
**Berührt:** RFC-004 (WebView), RFC-004-HMI (Touch/HMI), RFC-010 (Code-Editor), RFC-998 (Browser-Engine)

---

### Implementierungsstatus

| Abschnitt | Status |
|-----------|--------|
| Alle Abschnitte | ⏸ Theoretical — keine Umsetzung geplant |

---

## Inhaltsverzeichnis

1. [Motivation & Vision](#1-motivation--vision)
2. [Abgrenzung](#2-abgrenzung)
3. [Architektur-Überblick](#3-architektur-überblick)
4. [Protokoll-Schichten](#4-protokoll-schichten)
   - 4.1 Layer 1 — Canvas-Kommando-Protokoll
   - 4.2 Layer 2 — Layout-Constraint-Protokoll
   - 4.3 Layer 3 — Interaktions-Protokoll
   - 4.4 Layer 4 — Surface-Slot-Protokoll (Komposition)
5. [Kanal-Architektur](#5-kanal-architektur)
6. [Session-Management & Multi-User](#6-session-management--multi-user)
7. [Latenz-Kompensation — Optimistic Updates](#7-latenz-kompensation--optimistic-updates)
8. [Asset-Streaming & Caching](#8-asset-streaming--caching)
9. [Accessibility über Vellum](#9-accessibility-über-vellum)
10. [Sicherheit & Isolation](#10-sicherheit--isolation)
11. [Integration in Lux — Null-Architektur-Änderung](#11-integration-in-lux--null-architektur-änderung)
12. [Abgeleitete Produkte](#12-abgeleitete-produkte)
13. [Aufwandsschätzung](#13-aufwandsschätzung)
14. [Vergleich mit existierenden Protokollen](#14-vergleich-mit-existierenden-protokollen)
15. [Phasenmodell](#15-phasenmodell)
16. [Risiken & offene Fragen](#16-risiken--offene-fragen)
17. [Fazit](#17-fazit)

---

## 1. Motivation & Vision

### 1.1 Das Problem

Jedes Projekt, das heute eine eigene UI-Rendering-Engine baut — Zed (GPUI), Lapce, Cosmic Desktop, Iced, Slint — löst dieselben Probleme: Flex-Layout, Text-Shaping, SDF-Rendering, Hit-Testing, Focus-Management, Accessibility. Keine dieser Lösungen ist für andere nutzbar, weil alles in library-interne Datenstrukturen gegossen ist.

Gleichzeitig bleibt die "Remote-UI"-Welt in den 1990ern stecken: VNC streamt Pixel (auflösungsgebunden, kein A11y, kein Text-Select), X11 ist ein Relikt, und moderne Alternativen wie Wayland haben bewusst kein Netzwerk-Protokoll.

### 1.2 Die Beobachtung

Lux besitzt mit seiner Canvas-API (RFC-001 §6.2) bereits eine vollständige, deklarative Beschreibung aller Zeichenoperationen: `FillRoundRect`, `DrawTextLayout`, `PushBlur`, `DrawShadow`, `PushOpacity`. Diese API ist:

- **Deklarativ genug**, um bandbreitenschonend serialisiert zu werden
- **Konkret genug**, um auf jeder GPU-Pipeline direkt ausgeführt zu werden
- **Stabil** — die Canvas-API ist das stabilste öffentliche Interface des Frameworks

Die Erkenntnis: Die Canvas-API *ist* bereits das Protokoll. Sie muss nur serialisierbar werden.

### 1.3 Die Vision

**Vellum** ist ein Protokoll, das die Schnittstelle zwischen "jemand, der weiß, wie ein UI-Baum aussieht" (Server) und "jemand, der weiß, wie man Pixel auf den Bildschirm bekommt" (Client) standardisiert.

Ein Vellum-Server hält die App-Logik (Model + Update + View). Ein Vellum-Client rendert den resultierenden Canvas-Stream nativ — mit seiner eigenen GPU, seinem eigenen DPI, seiner eigenen Accessibility-Pipeline.

Das Ergebnis ist kein VNC-Klon. Es ist ein **Vektor-VNC**: Der gesamte Frame ist ein Kommando-Stream aus geometrischen Primitiven — vielleicht 4 KB statt 4 MB. Der Client rendert in seiner nativen Auflösung, mit seinem nativen Font-Hinting. 4K, 8K, ein 800×480 HMI-Panel — derselbe Stream, pixel-perfekt überall.

### 1.4 Namensgebung

**Vellum** — Pergament. Die Schicht, auf der geschrieben wird, nicht die Tinte selbst.

---

## 2. Abgrenzung

| In Scope | Nicht in Scope |
|---|---|
| Serialisierung des Canvas-Kommando-Streams | Eigene Browser-Engine (→ RFC-998) |
| Multi-Session / Multi-User-Fähigkeit | Peer-to-Peer / dezentrales Rendering |
| Latenz-Kompensation (Optimistic Updates) | Offline-First / CRDT (Phase 2+) |
| Asset-Streaming (Fonts, Images) | Video-Streaming / WebRTC |
| Accessibility-Tree-Transport | Vollständige Web-API-Kompatibilität |
| Input-Event-Routing mit SenderID | Plugin-System für Client-Extensions |
| Transport über TCP / Unix-Domain-Socket / WebSocket | Eigenes Transport-Protokoll (nutzt bestehende) |
| Client-lokale Surface-Injektion (Video, 3D, Kamera) | Surface-Provider-Implementierungen (sind client-seitig) |
| Rekursive Vellum-Komposition (App-in-App) | Automatisches Service-Discovery |
| Surface-Slot-Lifecycle über Netzwerk | Shared-Memory-IPC für lokale Surfaces (nutzt RFC-001 §8) |

---

## 3. Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────┐
│                     Vellum Server                           │
│                                                             │
│   Model ────► update(model, msg) ────► view(model, session) │
│                      ▲                        │             │
│                      │                        ▼             │
│              EnvelopedMsg              Canvas-Kommandos     │
│              (mit SenderID)            + Surface-Slot-Decls │
│                      ▲                        │             │
│                      │                        ▼             │
│   ┌─────────────┐    │    ┌──────────────────────────┐      │
│   │ Event-      │────┘    │ Canvas-Stream-Encoder    │      │
│   │ Decoder     │         │ (Interceptor auf Canvas) │      │
│   └──────┬──────┘         └────────────┬─────────────┘      │
│          │                             │                    │
│          │    ┌──────────────────┐      │                    │
│          │    │ Asset-Manager    │      │                    │
│          │    │ (Content-Hash)   │      │                    │
│          │    └────────┬─────────┘      │                    │
└──────────┼─────────────┼───────────────┼────────────────────┘
           │             │               │
      ═════╪═════════════╪═══════════════╪═══════ Netzwerk ════
           │             │               │
┌──────────┼─────────────┼───────────────┼────────────────────┐
│          │    Vellum Client             │                    │
│          ▼             ▼               ▼                    │
│   ┌──────────┐  ┌────────────┐  ┌──────────────┐           │
│   │ Event-   │  │ Asset-     │  │ Canvas-      │           │
│   │ Encoder  │  │ Cache      │  │ Stream-      │           │
│   └──────────┘  └────────────┘  │ Decoder      │           │
│        ▲                        └──────┬───────┘           │
│        │                               │                    │
│   ┌────┴──────────┐            ┌───────▼──────────┐        │
│   │ Input-System  │            │ Prediction-      │        │
│   │ (nativ)       │            │ Schicht          │        │
│   └───────────────┘            └───────┬──────────┘        │
│                                        │                    │
│   ┌───────────────────┐        ┌───────▼──────────┐        │
│   │ Surface-          │        │ WGPU Compositor  │        │
│   │ Compositor        ├───────►│ (lokal, nativ)   │        │
│   │                   │        └──────────────────┘        │
│   │ ┌───────────────┐ │                                    │
│   │ │ Local Surface │ │  ← Video, Kamera, 3D, ...         │
│   │ │ Providers     │ │                                    │
│   │ └───────────────┘ │                                    │
│   │ ┌───────────────┐ │                                    │
│   │ │ Upstream      │ │  ← Weitere Vellum-Verbindungen    │
│   │ │ Vellum Conns  │ │     (rekursiv, beliebig tief)     │
│   │ └───────────────┘ │                                    │
│   └───────────────────┘                                    │
└─────────────────────────────────────────────────────────────┘
```

### Kernprinzip: Null-Architektur-Änderung

Der Vellum-Server *ist* eine normale Lux-App. `model`, `update`, `view` werden nicht angerührt. Der Unterschied ist nur, ob der Canvas-Stream lokal an die GPU oder über einen Socket an Remote-Clients geht.

```go
// Lokale App (unverändert):
app.Run(model, update, view,
    app.WithTheme(theme.Default),
)

// Vellum-Server (eine Option dazu):
app.Run(model, update, view,
    app.WithTheme(theme.Default),
    vellum.Serve(":9900"),  // ← das ist alles
)

// Beides gleichzeitig (lokales Fenster + Remote-Clients):
app.Run(model, update, view,
    app.WithTheme(theme.Default),
    vellum.Serve(":9900"),  // Remote-Clients
    // Lokales Fenster rendert weiterhin direkt
)
```

---

## 4. Protokoll-Schichten

Das Protokoll ist in drei unabhängige Schichten gegliedert. Jede Schicht ist eigenständig nutzbar — ein Client muss nicht alle Schichten implementieren.

### 4.1 Layer 1 — Canvas-Kommando-Protokoll (Pflicht)

Die Serialisierung der Lux Canvas-API (RFC-001 §6.2). Ein Client, der Layer 1 versteht, kann jeden Lux-Frame pixelgenau rendern.

**Kommando-Typen (1:1 Mapping auf Canvas-Interface):**

```
// ── Primitive ────────────────────────────────────────────
FillRect(rect, paint)
FillRoundRect(rect, radius, paint)
FillRoundRectCorners(rect, radii, paint)
FillEllipse(rect, paint)
StrokeRect(rect, stroke)
StrokeRoundRect(rect, radius, stroke)
StrokeRoundRectCorners(rect, radii, stroke)
StrokeEllipse(rect, stroke)
StrokeLine(a, b, stroke)

// ── Pfade ────────────────────────────────────────────────
FillPath(path, paint)
StrokePath(path, stroke)

// ── Text ─────────────────────────────────────────────────
DrawText(text, origin, style, color)
DrawTextLayout(layoutID, origin)

// ── Bilder & Texturen ────────────────────────────────────
DrawImage(assetHash, dst, opts)
DrawImageSlice(assetHash, dst, insets, opts)
DrawTexture(textureHash, dst, opts)

// ── Schatten ─────────────────────────────────────────────
DrawShadow(rect, shadow)

// ── Clipping & Transform ─────────────────────────────────
PushClip(rect)
PushClipRoundRect(rect, radii)
PushClipPath(path)
PopClip()
PushTransform(transform)
PopTransform()
PushOffset(dx, dy)
PushScale(sx, sy)

// ── Effekte ──────────────────────────────────────────────
PushBlur(radius)
PopBlur()
PushOpacity(alpha)
PopOpacity()
PushLayer(opts)
PopLayer()
```

**Serialisierungsformat:** Binär, TLV-basiert (Type-Length-Value). Jedes Kommando hat einen 1-Byte-Opcode, gefolgt von der Payload-Länge (varint) und den typisierten Feldern. Kein JSON, kein Protobuf — zu viel Overhead für hochfrequente Streams.

**Frame-Semantik:** Ein Frame ist eine Sequenz von Kommandos, eingeschlossen in `BeginFrame(frameID, bounds, dpr)` / `EndFrame()`. Der Client darf annehmen, dass ein vollständiger Frame zwischen diesen Markern einen konsistenten Zustand repräsentiert.

**Delta-Komprimierung:** Der Server schickt optional nur die Differenz zum vorherigen Frame. Da die Canvas-Kommandos eine geordnete Liste sind, ist ein einfaches LCS-Diff (Longest Common Subsequence) auf Kommando-Ebene möglich. Unveränderte Bereiche werden als `RepeatRange(fromCmd, toCmd)` referenziert.

### 4.2 Layer 2 — Layout-Constraint-Protokoll (Optional)

Für Clients, die Text-Metrik-Hoheit haben wollen: Der Server schickt statt fertiger Canvas-Kommandos die Layout-Constraints (Flex-Container, Grid-Tracks, Text-Runs mit Style-Info). Der Client berechnet das Layout lokal.

```
// ── Layout-Container ─────────────────────────────────────
FlexContainer(direction, justify, align, gap, wrap)
GridContainer(tracks, gap)
Box(constraints, padding, margin)

// ── Text-Runs ────────────────────────────────────────────
TextRun(text, fontHash, size, weight, lineHeight)
TextBlock(runs[], maxWidth, alignment)

// ── Layout-Ergebnis (Client → Server) ────────────────────
ReflowReport(nodeID, computedBounds)
```

**Hybrides Bottom-Up-Modell:** Der Server schickt den logischen Widget-Baum als Constraints. Der Client vermisst Text-Runs mit seinem nativen Text-Stack, berechnet das Layout lokal, und meldet signifikante Abweichungen zurück (ReflowReport). Nur wenn das lokale Layout JS-Events auslösen könnte (IntersectionObserver, ResizeObserver), entsteht ein Roundtrip.

**Wann Layer 2 relevant ist:** Wenn ein Client eigene Font-Installation hat und präzises Font-Hinting sicherstellen will. Für die meisten Szenarien reicht Layer 1 (Server berechnet Layout, Client rendert).

### 4.3 Layer 3 — Interaktions-Protokoll (Pflicht für interaktive UIs)

```
// ── Events (Client → Server) ─────────────────────────────
KeyEvent(sessionID, key, modifiers, action)
MouseEvent(sessionID, pos, button, action)
ScrollEvent(sessionID, delta, precise, pos)
TouchEvent(sessionID, touchID, phase, pos, force)
IMEEvent(sessionID, composeText | commitText)
ResizeEvent(sessionID, newBounds, newDPR)
FocusEvent(sessionID, gained | lost)

// ── Deklarative Zustandsübergänge (Server → Client) ──────
DeclareStates(nodeID, {
    hover:   { paint: ... },
    focus:   { paint: ..., shadow: ... },
    active:  { paint: ..., transform: ... },
})

// ── Cursor (Server → Client) ─────────────────────────────
SetCursor(cursorKind)
```

**Deklarative Hover/Focus/Active-States:** Damit visuelle Feedback-Effekte (Hover-Farbe, Focus-Ring, Active-Skalierung) ohne Roundtrip zum Server funktionieren, annotiert der Server Nodes mit deklarativen Zustandsübergängen. Der Client wendet diese lokal an — nur semantische Events (Click, Input) gehen an den Server.

### 4.4 Layer 4 — Surface-Slot-Protokoll (Komposition)

Surface-Slots (RFC-001 §8) sind "Löcher" im Render-Baum, in die ein externer Renderer eine GPU-Textur liefert. In einer lokalen Lux-App dockt ein `SurfaceProvider` direkt an den Widget-Baum an — Browser-Engine, Video-Decoder, 3D-Renderer.

Über Vellum hinweg muss dieses Konzept überleben. Der Server kann nicht wissen, welche Surfaces der Client lokal hat — und der Client kann nicht wissen, welche Surfaces der Server eingebettet haben will. Surface-Slots werden deshalb zu einem **beidseitig befüllbaren Rendezvous-Punkt** im Protokoll.

#### 4.4.1 Surface-Slot-Typen

```
// ── Server deklariert einen Slot im Canvas-Stream ────────
DeclareSurfaceSlot(slotID, bounds, zIndex, {
    source:      SurfaceSource,
    inputRouting: bool,           // Client routet Input-Events an diesen Slot
    a11yNodeID:  AccessNodeID,    // Semantik-Anker im AccessTree (RFC-006)
})

// ── Surface-Quellen ──────────────────────────────────────
SurfaceSource =
    | ServerProvided     // Server liefert Textur-Daten über Asset-Kanal
    | ClientLocal        // Client füllt den Slot selbst (Video, Kamera, ...)
    | VellumUpstream     // Slot wird von einem anderen Vellum-Server gefüllt
    | VellumDownstream   // Server leitet seinen eigenen Surface-Slot durch
```

#### 4.4.2 Client-Lokale Surfaces

Der Server deklariert einen Slot und sagt: "Hier ist ein Loch, der Client füllt es selbst."

```go
// Server-seitig (in view):
ui.Surface(surfaceID,
    surface.ClientLocal("video-feed", surface.Hints{
        Kind:      surface.KindVideo,
        MimeType:  "video/h264",
        Fallback:  ui.Text("Video nicht verfügbar"),
    }),
    ui.FlexGrow(1),
)
```

Im Canvas-Stream wird das zu:

```
DeclareSurfaceSlot(slotID="video-feed", bounds=Rect{...}, {
    source: ClientLocal { kind: Video, hints: "video/h264" },
})
```

Der Client entscheidet lokal, was er in den Slot rendert:

```go
// Client-seitig: lokaler SurfaceProvider
client.RegisterLocalSurface("video-feed", &webcamProvider{
    device: "/dev/video0",
})
```

Das `SurfaceProvider`-Interface (RFC-001 §8) lebt dabei **auf dem Client**, nicht auf dem Server. `AcquireFrame()` liefert eine lokale GPU-Textur, die der Client in den reservierten Bounds composited — Zero-Copy, kein Netzwerk.

**Use-Cases:**

| Szenario | Server deklariert | Client füllt |
|---|---|---|
| Videokonferenz | Slot "camera-feed" | Lokale Webcam via V4L2/AVFoundation |
| Media-Player | Slot "video-player" | Lokaler Hardware-Decoder (VAAPI/VideoToolbox) |
| 3D-Viewport | Slot "3d-scene" | Lokaler wgpu-Renderer mit eigener Scene |
| Karten-Widget | Slot "map-view" | Lokaler Tile-Renderer mit GPU-Cache |

#### 4.4.3 Rekursive Vellum-Verbindungen

Ein Surface-Slot auf dem Client kann von einem **anderen Vellum-Server** befüllt werden. Der Client wird zum Vermittler zwischen zwei unabhängigen App-Servern.

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│  App-Server  │─Vellum──│    Client    │─Vellum──│ Browser-     │
│  (Server A)  │  :9900  │              │  :9901  │ Server       │
│              │         │  ┌────────┐  │         │ (Server B)   │
│  ┌────────┐  │         │  │ App UI │  │         │              │
│  │ App UI │──────────────▶│        │  │         │              │
│  │        │  │         │  │ ┌────┐ │  │         │              │
│  │ [Slot] │  │         │  │ │Web │◀───────────────  Browser   │
│  │        │  │         │  │ │View│ │  │         │  Engine      │
│  └────────┘  │         │  │ └────┘ │  │         │              │
│              │         │  └────────┘  │         │              │
└──────────────┘         └──────────────┘         └──────────────┘
```

Im Protokoll:

```
// Server A deklariert einen Slot als Vellum-Upstream:
DeclareSurfaceSlot(slotID="browser", bounds=Rect{...}, {
    source: VellumUpstream {
        addr: "server-b.local:9901",  // Empfehlung, Client entscheidet
        // Alternativ: Client kennt die Adresse selbst
    },
    inputRouting: true,   // Input in diesem Bereich → Server B
})
```

Der Client öffnet eine zweite Vellum-Verbindung zu Server B und composited dessen Canvas-Stream in den reservierten Slot. **Input-Routing** wird anhand der Bounds aufgeteilt:

- Mausklick innerhalb des Slot-Bounds → Event geht an Server B
- Mausklick außerhalb → Event geht an Server A
- Keyboard-Events folgen dem Focus (welcher Server hat den fokussierten Node?)

#### 4.4.4 Rekursion, beliebig tief

Da ein Vellum-Server selbst Surface-Slots deklarieren kann, die wiederum von Vellum-Servern befüllt werden, entsteht eine **beliebig tiefe Kompositionskette:**

```
Server A (Haupt-App)
  └── Slot: Server B (Browser-Engine)
        └── Slot: Server C (eingebettetes Widget in der Webseite)
              └── Slot: Client-lokal (Video-Feed)
```

Jede Ebene hat ihren eigenen Canvas-Stream, ihren eigenen AccessTree, ihr eigenes Input-Routing. Der Client composited alle Streams in ein einziges WGPU-Render-Target.

**Wichtig:** Die Rekursion ist logisch, nicht physisch. Der Client hält N flache Vellum-Verbindungen und composited deren Outputs. Es gibt keine verschachtelten Netzwerk-Tunnel.

#### 4.4.5 AccessTree-Merge bei Surface-Slots

RFC-006 (Surface Semantics) definiert bereits das `SemanticProvider`-Interface für lokale Surface-Slots. Über Vellum erweitert sich das:

| Surface-Typ | AccessTree-Quelle |
|---|---|
| Server-Provided | Teil des Server-AccessTree (kommt über Kanal 0) |
| Client-Local | Client liefert lokalen Subtree via `SemanticProvider` |
| Vellum-Upstream | Kommt als AccessTree von Server B über dessen Kanal 0 |

Der Client merged alle AccessTree-Fragmente in einen einzigen Baum und speist ihn in seine lokale A11y-Bridge. Aus Sicht des Screenreaders ist die App ein einziger, kohärenter semantischer Baum — egal wie viele Server und lokale Surfaces dazu beitragen.

#### 4.4.6 Surface-Slot-Lifecycle

```
// ── Server → Client ──────────────────────────────────────
DeclareSurfaceSlot(slotID, ...)    // Slot erscheint im nächsten Frame
UpdateSurfaceSlot(slotID, ...)     // Bounds/Z-Index ändern sich
RemoveSurfaceSlot(slotID)          // Slot verschwindet

// ── Client → Server ──────────────────────────────────────
SurfaceReady(slotID)               // Client hat den Slot erfolgreich befüllt
SurfaceError(slotID, error)        // Client kann den Slot nicht füllen
SurfaceLost(slotID)                // Upstream-Verbindung verloren
```

Wenn ein Client einen `ClientLocal`-Slot nicht füllen kann (kein Video-Device, kein Upstream-Server erreichbar), rendert er den Server-deklarierten Fallback-Content. Der Server wird via `SurfaceError` informiert und kann im nächsten `update`-Zyklus reagieren.

---

## 5. Kanal-Architektur

Eine Vellum-Verbindung besteht aus logisch getrennten Kanälen über einen Multiplex-Stream:

```
Vellum-Verbindung
    │
    ├── Kanal 1: Canvas-Stream (Server → Client)
    │     Hochfrequent, klein (typisch 2–8 KB/Frame)
    │     BeginFrame → Kommandos → EndFrame
    │     Optional: Delta-komprimiert
    │
    ├── Kanal 2: Asset-Stream (Server → Client, bidirektional für Cache-Negotiation)
    │     Niederfrequent, größere Payloads
    │     Fonts, Images, Shader
    │     Content-addressable (Hash-basiert)
    │
    ├── Kanal 3: Event-Stream (Client → Server)
    │     Hochfrequent bei Interaktion
    │     Jedes Event trägt SessionID
    │     Batching: Mehrere Mouse-Moves pro Netzwerk-Paket
    │
    ├── Kanal 4: Surface-Stream (bidirektional)
    │     Surface-Slot-Lifecycle (Declare, Update, Remove, Ready, Error)
    │     Upstream-Vellum-Adressen für rekursive Verbindungen
    │     Client-Local-Surface-Negotiation
    │
    └── Kanal 0: Control-Stream (bidirektional)
          Handshake, Capability-Negotiation
          Session-Management (Join, Leave, Heartbeat)
          Asset-Manifest, Cache-Sync
          AccessTree-Updates (inkl. Surface-Subtree-Merge)
```

### 5.1 Transport

Vellum definiert kein eigenes Transport-Protokoll. Es nutzt bestehende Transporte:

| Transport | Use-Case |
|---|---|
| Unix-Domain-Socket | Lokaler Inspector, Same-Machine-Szenarien |
| TCP + TLS | LAN / WAN |
| WebSocket + TLS | Browser-Client |
| Shared Memory + Futex | In-Process (lokales Rendering als Vellum-Stream, Zero-Copy) |

### 5.2 Handshake & Capability-Negotiation

```
Client → Server:  VellumHello {
    protocolVersion:  1,
    supportedLayers:  [Layer1, Layer3, Layer4],  // Layer2/4 optional
    maxFrameRate:     60,
    viewport:         { 1920, 1080 },
    dpr:              2.0,
    knownAssets:      [hash1, hash2, ...],       // Bereits gecachte Assets
    a11yCapabilities: [ATSPI2],                  // Verfügbare A11y-Bridges
    locale:           "de-DE",
    surfaceCapabilities: {
        localSurfaces:    [Video, WebGL, Camera], // Lokal verfügbare Surface-Typen
        vellumUpstream:   true,                   // Kann rekursive Vellum-Verbindungen öffnen
        maxSurfaceSlots:  8,                      // Gleichzeitig composit-bare Slots
    },
}

Server → Client:  VellumWelcome {
    sessionID:        "sess-a7f3",
    assetManifest:    AssetManifest { ... },
    initialFrame:     Frame { ... },
    surfaceSlots:     []SurfaceSlotDecl { ... },  // Initial deklarierte Slots
    serverCapabilities: { maxSessions: 16, supportsLayer2: true },
}
```

---

## 6. Session-Management & Multi-User

### 6.1 Architektonischer Vorteil durch Elm

Die Elm-Architektur liefert Multi-User-Fähigkeit als strukturelle Eigenschaft, nicht als nachgerüstetes Feature:

- **Single Source of Truth:** Der Server hält *ein* Model. Kein verteiltes Consensus-Problem.
- **Totale Ordnung:** Jede Mutation läuft durch `update(model, msg)`. Alle Operationen sind serialisiert.
- **Determinismus:** Gleiche Msg-Sequenz → gleicher Zustand. Reproduzierbar, testbar.

### 6.2 EnvelopedMsg

Jede Msg vom Client wird mit Sender-Metadaten versehen:

```go
type EnvelopedMsg struct {
    Sender    SessionID
    Timestamp uint64      // Lamport-Timestamp (monoton, nicht Wallclock)
    Msg       Msg
}
```

Die `update`-Funktion sieht weiterhin eine lineare Sequenz von Messages. Sie muss nicht wissen, dass drei User gleichzeitig tippen — sie verarbeitet Msg für Msg.

### 6.3 Shared Model + Per-Session-State

```go
type ServerState struct {
    Model    Model                        // Shared, Single Source of Truth
    Sessions map[SessionID]*SessionState  // Per-User UI State
}

type SessionState struct {
    Cursor    Position      // Jeder User hat eigenen Cursor
    Selection Selection     // Jeder User hat eigene Selektion
    Viewport  Rect          // Jeder User scrollt unabhängig
    Theme     Theme         // Jeder User kann ein anderes Theme haben
    DPR       float32       // Jeder Client hat eigenes DPI
}
```

**Entscheidende Trennung:** Das `Model` gehört der App-Logik (Dokument-Inhalt, Business-State). Der `SessionState` gehört der Vellum-Infrastruktur (Cursor, Viewport, Theme). Die `update`-Funktion sieht nur das `Model` — sie muss sich nicht um Session-Management kümmern.

### 6.4 Per-Session View

Die `view`-Funktion erhält optional einen `SessionContext`:

```go
// Single-User (unverändert, bestehende Lux-Apps):
type ViewFunc[M any] func(M) Element

// Multi-User (Vellum-Erweiterung):
type MultiViewFunc[M any] func(M, SessionContext) Element

type SessionContext struct {
    Self       SessionID
    Peers      []PeerInfo    // Andere User: Name, Farbe, Cursor-Position
    Viewport   Rect          // Viewport dieses Users
}

type PeerInfo struct {
    ID         SessionID
    Name       string
    Color      Color         // Zugewiesene Presence-Farbe
    Cursor     Position      // Cursor-Position im Dokument
    Selection  Selection     // Aktive Selektion
}
```

Der Reconciler berechnet pro Session einen eigenen VTree und damit einen eigenen Canvas-Stream. Die VTrees überlappen sich fast vollständig — der einzige Unterschied ist typischerweise Viewport-Offset, Cursor-Position, Selektion, Presence-Overlays.

### 6.5 Multi-User-Text-Editing

Für kollaborative Texteditierung gibt es zwei Phasen:

**Phase 1 — Zentraler Server (Vellum v1.0):**
Der `update`-Loop serialisiert alle Operationen. Da alle Inputs durch eine einzige Funktion laufen, entsteht natürliches OT (Operational Transformation) durch die Serialisierung. Kein CRDT nötig.

**Phase 2 — Offline-Fähigkeit (post-v1.0):**
Erst wenn Clients offline-fähig sein oder Peer-to-Peer kommunizieren sollen, werden CRDTs relevant. Der Rope/Piece-Table aus RFC-010 ist die natürliche Grundlage dafür. Dieser Schritt ist explizit nicht Teil von Vellum v1.0.

---

## 7. Latenz-Kompensation — Optimistic Updates

### 7.1 Das Problem

Bei 50ms RTT fühlt sich Text-Input ohne lokale Kompensation "laggy" an. Der User will den Buchstaben sofort sehen, nicht nach einem Roundtrip.

### 7.2 Prediction-Schicht

Der Client führt kein eigenes Model — aber eine Prediction-Schicht über dem letzten bestätigten Server-Frame:

```
Server-Frame N (bestätigt)
    │
    ├── User tippt "a"     → Client wendet Prediction an (sofort)
    ├── User tippt "b"     → Client wendet Prediction an (sofort)
    │
    ▼
Server-Frame N+1 kommt an (enthält "ab" + alles andere)
    │
    └── Client vergleicht: Prediction == Server-Ergebnis?
        ├── Ja  → Prediction verwerfen, Server-Frame übernehmen (seamless)
        └── Nein → Server gewinnt, kurzer visueller Korrektur-Snap
```

### 7.3 Prediction-Kategorien

| Kategorie | Prediction | Korrektheit |
|---|---|---|
| Text-Input | Buchstabe erscheint sofort am Cursor | >99%, solange kein anderer User an derselben Stelle tippt |
| Cursor-Bewegung | Pfeiltasten, Mausklick in Text | >99%, rein lokal berechenbar |
| Selektion | Shift+Pfeiltasten, Shift+Klick | >99%, rein lokal berechenbar |
| Scroll | Immer lokal (lebt in SessionState) | 100% |
| Hover-States | Deklarativ, immer lokal | 100% |
| Button-Click | Keine Prediction — wartet auf Server | n/a (50ms RTT für Buttons nicht wahrnehmbar) |
| Formular-Submit | Keine Prediction — wartet auf Server | n/a |

### 7.4 Prediction-Regeln als Protokoll-Bestandteil

Prediction-Regeln sind Teil des Vellum-Protokolls, nicht der App. Der App-Entwickler schreibt weiterhin nur `update` und `view`:

```go
// Client-seitig, Teil des Vellum-Clients
type PredictionRule interface {
    // CanPredict prüft ob diese Input-Msg lokal vorhersagbar ist
    CanPredict(msg InputMsg, nodeType NodeType) bool

    // Apply mutiert den lokalen Prediction-Buffer
    Apply(prediction *PredictionState, msg InputMsg)

    // Reconcile gleicht mit dem Server-Frame ab
    Reconcile(prediction *PredictionState, serverFrame Frame)
}
```

**Eingebaute Regeln:**

- `TextInputPrediction` — Buchstaben-Insertion am Cursor-Offset
- `CursorMovePrediction` — Cursor-Offset-Veränderung (Pfeiltasten, Home, End)
- `SelectionPrediction` — Selektion erweitern/reduzieren
- `ScrollPrediction` — Viewport-Offset (immer lokal)
- `HoverPrediction` — Deklarative Hover-States (immer lokal)

### 7.5 Server-Validierung

Der Server kann optional einen Model-Hash mit jedem Frame mitsenden:

```
EndFrame(frameID, modelHash)
```

Der Client vergleicht: Hat meine Prediction denselben Zustand produziert? Bei Mismatch übernimmt der Server-Frame autoritativ. Da `update` eine reine Funktion ist, ist diese Validierung deterministisch.

---

## 8. Asset-Streaming & Caching

### 8.1 Content-Addressable Assets

Assets fließen nicht im Canvas-Stream. Sie haben einen separaten Kanal, content-addressable, einmalig:

```go
type AssetManifest struct {
    Fonts    []AssetRef
    Images   []AssetRef
    Shaders  []AssetRef
}

type AssetRef struct {
    Hash     uint64       // Content-Hash (xxHash64)
    Size     uint32       // Bytes
    Kind     AssetKind    // Font, Image, Shader
    MimeType string       // "font/ttf", "image/png", etc.
    Inline   []byte       // Für kleine Assets (<4 KB): direkt im Manifest
}
```

Der Canvas-Stream referenziert Assets nur per Hash:

```
DrawImage(hash=0x5678ABCD, dst=Rect{...}, opts=ImageOpts{...})
```

Nicht:

```
DrawImage(pixels=[]byte{...}, dst=Rect{...}, opts=ImageOpts{...})
```

### 8.2 Cache-Negotiation beim Handshake

```
Client → Server:  VellumHello { knownAssets: [hash1, hash2, hash3] }
Server → Client:  VellumWelcome { assetManifest: { /* nur unbekannte Assets */ } }
```

Der Client cached Assets lokal — per Hash, persistent über Sessions hinweg. Beim Reconnect schickt der Client seine bekannten Hashes, der Server schickt nur die Deltas.

### 8.3 Font-Streaming — Metriken zuerst

Fonts sind der kritischste Asset-Typ. Ohne den Font kann der Client kein korrektes Text-Rendering durchführen. Zwei Mechanismen, die sich ergänzen:

**Sofort-Metriken:** Der Server liefert mit dem Manifest die Font-Metriken als kompakte Daten:

```go
type FontMetrics struct {
    Hash         uint64
    Ascent       float32
    Descent      float32
    LineGap      float32
    UnitsPerEm   uint16
    XHeight      float32
    AvgCharWidth float32
    // Glyph-Breiten für die häufigsten ~256 Codepoints:
    CommonWidths [256]float32
}
```

Das sind ~1–2 KB pro Font. Damit kann der Client sofort layouten — auch bevor die Font-Datei vollständig übertragen ist.

**Progressive Schärfe:** Die Font-Dateien selbst werden parallel gestreamt. Sobald sie ankommen, baut der Client den MSDF-Atlas lokal und das Rendering wird schärfer. Der Übergang ist visuell fast unsichtbar, weil die Metriken bereits stimmen und sich nur die Rendering-Qualität verbessert.

### 8.4 Image-Streaming — Progressive Qualität

Für Bilder bietet der Server optional mehrere Qualitätsstufen:

```go
type ImageAsset struct {
    Hash       uint64
    Levels     []ImageLevel
}

type ImageLevel struct {
    Quality    ImageQuality  // Placeholder, Low, Medium, Full
    Hash       uint64        // Eigener Content-Hash
    Size       uint32        // Bytes
}
```

Der Client zeigt sofort den Placeholder (2 KB, mit `PushBlur` gerendert — visuell elegant), lädt dann die volle Qualität nach, und blendet scharf.

---

## 9. Accessibility über Vellum

### 9.1 Das VNC-Problem

VNC killt jede Accessibility — ein Screenreader sieht nur einen Pixel-Blob. Bei Vellum reist der AccessTree als paralleler Daten-Stream mit.

### 9.2 AccessTree-Transport

Der Server sendet nach jedem Reconcile den aktualisierten AccessTree über Kanal 0:

```
AccessTreeUpdate {
    version:    uint64,
    nodes:      []AccessNodeWire {
        id:          AccessNodeID,
        parentID:    AccessNodeID,
        role:        AccessRole,
        label:       string,
        description: string,
        value:       string,
        lang:        string,       // BCP 47
        bounds:      Rect,
        states:      AccessStates,
        actions:     []ActionDesc,
        relations:   []RelationDesc,
    },
}
```

Der Client speist diesen Baum in seine lokale A11y-Bridge:

| Client-Platform | Bridge |
|---|---|
| Linux | AT-SPI2 via D-Bus (bereits in Lux vorhanden) |
| Windows | UIA via COM (bereits in Lux vorhanden) |
| macOS | NSAccessibility via ObjC (bereits in Lux vorhanden) |
| Browser (WebSocket-Client) | ARIA-Attribute auf Canvas-Overlay |

### 9.3 Screenreader-Interaktion

Wenn ein Screenreader eine Action auslöst (z.B. "Activate" auf einem Button), routet der Client das als `AccessActionEvent(sessionID, nodeID, actionName)` an den Server. Der Server führt die Action im `update`-Loop aus — derselbe Pfad wie für Maus- oder Keyboard-Events.

### 9.4 Strategische Bedeutung

Ein Remote-UI-Protokoll mit integrierter Accessibility ist ein Alleinstellungsmerkmal. Kein existierendes Remote-Display-Protokoll (VNC, RDP, X11, Wayland) transportiert einen semantischen A11y-Baum. Vellum wäre das erste.

Für den HMI-Einsatz (RFC-004-hmi-touch) bedeutet das: Ein Industrie-Panel am Band wird über Vellum von einem Server bedient. Im Leitstand, 200 Meter weiter, sieht der Schichtleiter dasselbe Interface — mit vollem Screenreader-Support, wenn er ihn braucht. EN 301 549-Compliance über Remote-Rendering.

---

## 10. Sicherheit & Isolation

### 10.1 Trust-Modell

Der Vellum-Server ist vertrauenswürdig — er hält die App-Logik und das Model. Der Client ist *nicht* vertrauenswürdig:

- Der Client sendet nur Input-Events — keine Mutations-Befehle, keine Model-Fragmente.
- Der Server validiert jeden Input gegen die `update`-Funktion. Es gibt keinen Weg, den Server-State direkt zu mutieren.
- Die `update`-Funktion ist die einzige Stelle, an der State-Änderungen stattfinden — das ist eine strukturelle Garantie der Elm-Architektur, kein Security-Layer.

### 10.2 Session-Isolation

Jede Session hat ihren eigenen `SessionState`. Sessions können sich gegenseitig nicht beeinflussen — außer über das shared Model, und da nur über den `update`-Pfad.

### 10.3 Transport-Sicherheit

Vellum setzt TLS für TCP/WebSocket voraus. Unix-Domain-Sockets nutzen Dateisystem-Permissions. Vellum definiert kein eigenes Authentifizierungs-Protokoll — es delegiert an den Transport (TLS Client Certificates, Token-basierte Auth im Handshake).

---

## 11. Integration in Lux — Null-Architektur-Änderung

### 11.1 Server-Seite: Canvas-Interceptor

Die einzige Änderung an Lux ist ein Interceptor, der Canvas-Calls aufzeichnet:

```go
// internal/vellum/encoder.go

type CanvasEncoder struct {
    inner  draw.Canvas    // Der echte Canvas (GPU)
    buffer *FrameBuffer   // Der serialisierte Stream
}

// Jeder Canvas-Call wird 1:1 durchgereicht UND aufgezeichnet:
func (e *CanvasEncoder) FillRoundRect(r Rect, radius float32, paint Paint) {
    e.inner.FillRoundRect(r, radius, paint)          // GPU (lokal)
    e.buffer.WriteFillRoundRect(r, radius, paint)    // Stream (Remote)
}
```

Wenn kein Vellum-Listener aktiv ist, wird der Interceptor nicht eingehängt — kein Overhead.

### 11.2 Client-Seite: Canvas-Decoder

Der Vellum-Client ist ein eigenständiges Go-Package — kein Model, kein Update, kein View:

```go
// vellum/client/client.go

func Connect(addr string, opts ...ClientOption) (*Client, error)

type Client struct {
    renderer  *gpu.WGPURenderer   // Lux' bestehender Renderer
    decoder   *CanvasDecoder       // Stream → Canvas-Calls
    predictor *PredictionEngine    // Optimistic Updates
    cache     *AssetCache          // Persistenter Asset-Cache
    a11y      *AccessTreeBridge    // Lokale A11y-Bridge
}
```

**Geschätzter Umfang:**

| Komponente | LOC (geschätzt) |
|---|---|
| Canvas-Stream-Encoder (Server) | ~1.500 |
| Canvas-Stream-Decoder (Client) | ~1.500 |
| Session-Registry (Server) | ~800 |
| Asset-Manager + Cache | ~1.200 |
| Prediction-Engine (Client) | ~1.500 |
| Event-Encoder/Decoder | ~600 |
| Handshake + Control-Kanal | ~800 |
| AccessTree-Serialisierung | ~600 |
| Surface-Compositor (Client) | ~1.500 |
| Surface-Slot-Protocol + Lifecycle | ~800 |
| Upstream-Vellum-Manager (rekursiv) | ~1.200 |
| **Gesamt** | **~12.000** |

### 11.3 Kein Fork, kein Breaking Change

Vellum ist ein optionales Package (`lux/vellum`). Bestehende Lux-Apps ignorieren es komplett. Eine App wird zum Vellum-Server durch eine einzige Option in `app.Run`. Der `view`-Code, der `update`-Code, das Theme — alles bleibt identisch.

---

## 12. Abgeleitete Produkte

Der Canvas-Stream ist ein universelles Primitiv. Aus einem Protokoll fallen mehrere Produkte:

### 12.1 Remote-Display

Ein Vellum-Client rendert den Canvas-Stream auf einem entfernten Bildschirm — nativ, in lokaler Auflösung, mit lokalem DPI und lokalem A11y-Support. Nicht VNC. Vektor-VNC.

### 12.2 Inspector / DevTools (RFC-001 §12)

Der Inspector ist ein Vellum-Client, der den Canvas-Stream liest und mit Debug-Overlays anreichert (Bounds, Padding, Margins, Hit-Test-Highlighting). Das Debug-Protocol aus RFC-001 §12 wird zu Vellum + Debug-Extensions.

### 12.3 Test-Harness / Golden-File-Tests

Ein headless Vellum-Client, der den Canvas-Stream in eine Datei schreibt und gegen Golden Files vergleicht. Pixel-perfekte Regressions-Tests ohne echte GPU — der Stream *ist* die Repräsentation.

### 12.4 Screen-Recording / Replay

Ein Vellum-Client, der den Stream aufzeichnet und später abspielen kann. Da der Stream deterministisch ist, ist das Replay frame-perfekt. Als Vektor-Aufzeichnung ist es um Größenordnungen kleiner als Video — und nachträglich in beliebiger Auflösung renderbar.

### 12.5 Browser-Client (WebAssembly)

Ein Vellum-Client als WASM-Bundle, das in einem `<canvas>`-Element rendert (via WebGPU). Jede Lux-App wird automatisch zu einer Web-App — ohne dass der Entwickler eine Zeile Web-Code schreibt. Der Server bleibt ein Go-Binary.

### 12.6 Multi-User Collaboration

Der Server akzeptiert mehrere Sessions. Jede Session bekommt ihren eigenen Canvas-Stream (unterschiedlicher Viewport, eigener Cursor). Presence-Overlays (Cursor anderer User, Selektionen) sind Teil des `view`-Outputs — der Framework-Entwickler entscheidet, wie sie aussehen.

### 12.7 HMI-Fernwartung (RFC-004-hmi-touch)

Ein Industrie-Panel läuft lokal (DRM/KMS). Der Leitstand-PC ist ein Vellum-Client. Derselbe App-Server bedient beide — ein lokales Fenster und N Remote-Clients. Der Leitstand kann ein anderes Theme haben (größere Schrift, höherer Kontrast), weil Theming per Session konfigurierbar ist.

### 12.8 Composable Applications — Surface-Slot-Komposition

Durch Layer 4 (§4.4) wird der Vellum-Client zu einem **Compositor**, der Inhalte aus beliebig vielen Quellen in einem einzigen Fenster vereint:

**Szenario 1 — Remote-App mit lokalem Video:**
Ein Telemedizin-Interface läuft als Vellum-Server. Der Client zeigt die App remote, injiziert aber den lokalen Kamera-Feed direkt in den Surface-Slot — ohne Umweg über den Server. Das Video bleibt lokal, nur die UI ist remote.

**Szenario 2 — App-in-App (rekursive Vellum):**
Eine IDE (Server A) bettet einen Browser (Server B) ein, der wiederum ein interaktives Widget (Server C) hostet. Der Client hält drei flache Vellum-Verbindungen und composited alle Canvas-Streams in ein Fenster. Input-Routing folgt dem Focus über Server-Grenzen hinweg. Der AccessTree merged alle drei Semantik-Bäume nahtlos.

**Szenario 3 — Micro-Frontend-Architektur:**
Ein Dashboard besteht aus unabhängigen Services. Jedes Panel ist ein eigener Vellum-Server. Der Client composited N Streams in einem Grid — jedes Panel hat seinen eigenen App-State, sein eigenes Update-Model, seinen eigenen Lifecycle. Fällt ein Service aus, zeigt der Client den Fallback-Content für dessen Slot, während der Rest weiterläuft.

---

## 13. Aufwandsschätzung

### 13.1 Implementierungsaufwand

| Phase | Umfang | Dauer (1 Entwickler) |
|---|---|---|
| Phase 1: Canvas-Stream-Roundtrip | Encoder + Decoder + lokale Pipe | 3–4 Wochen |
| Phase 2: Netzwerk-Transport | TCP/TLS + Session-Management | 2–3 Wochen |
| Phase 3: Asset-Streaming | Font-Metriken + Image-Cache | 2 Wochen |
| Phase 4: Prediction-Engine | Text-Input + Cursor + Scroll | 3–4 Wochen |
| Phase 5: A11y-Transport | AccessTree-Serialisierung | 1–2 Wochen |
| Phase 6: Surface-Komposition | Lokale Surfaces + rekursives Vellum | 3–4 Wochen |
| **Gesamt** | **~12.000 LOC** | **~15–19 Wochen** |

### 13.2 Abhängigkeiten an Lux

Vellum erfordert keine Änderungen an Lux-Kern-APIs. Es nutzt:

- `draw.Canvas` Interface (stabil, RFC-001 §6.2) — Interceptor-Punkt
- `a11y.AccessTree` (stabil, RFC-001 §11) — Serialisierung
- `input.*Msg` Typen (stabil, RFC-002 §2) — Event-Encoding
- `app.Option` Pattern (stabil, RFC-001 §3) — `vellum.Serve()` als Option

---

## 14. Vergleich mit existierenden Protokollen

| Eigenschaft | VNC/RFB | RDP | X11 | Wayland | **Vellum** |
|---|---|---|---|---|---|
| Datenformat | Pixel-Bitmap | Pixel + GDI-Ops | Zeichenprimitive | Lokal only | Vektor-Kommandos |
| Auflösungsunabhängig | Nein | Teilweise | Nein | n/a | **Ja** |
| Bandbreite (UI-Frame) | ~50–500 KB | ~10–100 KB | ~5–20 KB | n/a | **~2–8 KB** |
| Text selektierbar | Nein | Ja (lokal) | Ja | Ja | **Ja** |
| Accessibility | Nein | Nein | Nein | Nein | **Ja (AccessTree)** |
| Multi-User | Nein | Nein | Nein | Nein | **Ja (Sessions)** |
| Client-seitiges Theming | Nein | Nein | Nein | Nein | **Ja** |
| Latenz-Kompensation | Nein | Ja (partiell) | Nein | n/a | **Ja (Prediction)** |
| DPI-Anpassung | Nein | Ja (Scaling) | Nein | Ja | **Ja (nativ)** |
| Lokale Surface-Injektion | Nein | Nein | Nein | Nein | **Ja (Layer 4)** |
| Rekursive Komposition | Nein | RemoteApp (begrenzt) | Nein | Nein | **Ja (beliebig tief)** |

---

## 15. Phasenmodell

### Phase 1 — Proof of Concept: Canvas-Roundtrip

**Ziel:** Lux-App rendert einen Frame, serialisiert den Canvas-Stream, ein separater Prozess deserialisiert und rendert identisch.

**Vorgehen:**
1. Interceptor auf `draw.Canvas` schreiben (CanvasEncoder)
2. Binäres Serialisierungsformat für Canvas-Kommandos definieren
3. CanvasDecoder schreiben, der den Stream auf einem zweiten WGPU-Renderer abspielt
4. KitchenSink-Demo als Testfall: Frame-Stream in Datei schreiben, zweiter Prozess rendert

**Testbar wenn:** Beide Fenster zeigen identischen Output. Frame-Größe ist <10 KB für eine typische UI.

**Nebenprodukt:** Renderer-Replay für Inspector und Golden-File-Tests.

### Phase 2 — Netzwerk: Single-User Remote-Rendering

**Ziel:** Lux-App auf Machine A, Fenster auf Machine B.

**Vorgehen:**
1. TCP-Transport mit Multiplexing (Kanäle 0–3)
2. TLS-Handshake + Capability-Negotiation
3. Input-Event-Routing (Client → Server)
4. Asset-Manifest + Cache-Sync beim Connect

**Testbar wenn:** KitchenSink-Demo läuft remote, interaktiv, mit <100ms gefühlter Latenz im LAN.

### Phase 3 — Latenz: Optimistic Updates

**Ziel:** Text-Input fühlt sich lokal an, auch bei 50ms RTT.

**Vorgehen:**
1. Prediction-Engine für Text-Input, Cursor, Selektion
2. Reconciliation mit Server-Frames
3. Deklarative Hover/Focus/Active-States ohne Roundtrip

**Testbar wenn:** Text-Editor über 50ms-simulierte Latenz fühlt sich "lokal" an.

### Phase 4 — Multi-User

**Ziel:** Zwei User sehen dasselbe Dokument mit eigenen Cursors.

**Vorgehen:**
1. SessionID in allen Events
2. Per-Session ViewFunc
3. Presence-Overlays (Cursor, Selektion anderer User)
4. Concurrent-Edit-Handling im `update`-Loop

**Testbar wenn:** Zwei Clients editieren dasselbe Dokument. Beide sehen den Cursor des anderen.

### Phase 5 — Ecosystem

**Ziel:** Vellum als eigenständig nutzbares Protokoll.

**Vorgehen:**
1. Protokoll-Spezifikation als eigenständiges Dokument
2. WASM-Client (Browser)
3. AccessTree-Transport + A11y
4. Referenz-Client als eigenständiges Binary

### Phase 6 — Surface-Komposition

**Ziel:** Client kann lokale Surfaces injizieren und rekursive Vellum-Verbindungen öffnen.

**Vorgehen:**
1. Surface-Slot-Protokoll (Declare, Update, Remove, Ready, Error)
2. Client-lokale `SurfaceProvider`-Registry (Video, Kamera, 3D)
3. Surface-Compositor im Client-Renderer (Slot-Bounds → Texture compositing)
4. Upstream-Vellum-Manager: zweite Vellum-Verbindung öffnen, Canvas-Stream in Slot rendern
5. Input-Routing über Surface-Grenzen (Bounds-basiert + Focus-Tracking)
6. AccessTree-Merge: lokale Subtrees + Upstream-Subtrees in einen Baum

**Testbar wenn:** Eine App zeigt einen Remote-Surface-Slot, der von einem zweiten Vellum-Server befüllt wird. Input im Slot geht an Server B, Input außerhalb an Server A. Ein lokaler Video-Feed läuft in einem dritten Slot ohne Netzwerk-Roundtrip.

---

## 16. Risiken & offene Fragen

### 16.1 Canvas-API-Stabilität

Vellum ist an die Stabilität der Canvas-API gekoppelt. Jede Änderung am Canvas-Interface erfordert eine Protokoll-Version-Bump. Die Canvas-API ist als "stabilstes öffentliches Interface" deklariert (RFC-001 §6.2) — das Risiko ist gering, aber nicht null.

**Mitigation:** Versionierte Opcodes. Unbekannte Opcodes werden vom Client ignoriert (Forward-Compatibility).

### 16.2 Frame-Größe bei komplexen UIs

Die 2–8 KB Schätzung pro Frame gilt für typische Desktop-UIs. UIs mit vielen Pfaden, Gradients oder komplexen Clip-Stacks können größer werden.

**Mitigation:** Delta-Komprimierung (nur geänderte Kommandos senden). Empirische Messung mit realen Apps als Teil von Phase 1.

### 16.3 Prediction-Korrektheit bei Multi-User-Edits

Wenn zwei User an derselben Stelle tippen, kann die Prediction des einen durch die Operation des anderen invalidiert werden. Das führt zu visuellen Korrektur-Snaps.

**Mitigation:** Das ist ein bekanntes Problem in jedem kollaborativen Editor (Google Docs hat dasselbe). Die Korrektur ist visuell kurz (<1 Frame) und selten (zwei User müssen exakt dieselbe Textstelle editieren). In Phase 1–3 ist es kein Problem (Single-User). In Phase 4 ist es akzeptabel.

### 16.4 State-Größe des Canvas-Streams

Der Canvas-Stream enthält keinen State über Frames hinweg — jeder Frame ist eigenständig (oder Delta zum vorherigen). Das vereinfacht das Protokoll, bedeutet aber, dass ein Client nach einem Reconnect den vollständigen Frame braucht, nicht nur die letzten Deltas.

**Mitigation:** Server cached den letzten vollständigen Frame pro Session. Reconnect = vollständiger Frame + Asset-Delta.

### 16.5 WebSocket-Performance für Browser-Clients

WebSocket über TLS hat höheren Overhead als raw TCP. Für hochfrequente Canvas-Streams (60 fps) könnte das zum Engpass werden.

**Mitigation:** Frame-Rate-Anpassung pro Client. Browser-Clients bekommen typisch 30 fps statt 60 fps. Delta-Komprimierung reduziert die Payload. WebTransport (HTTP/3) als zukünftige Alternative.

### 16.6 Offene Leitfragen

1. **Opcode-Format:** TLV mit 1-Byte-Opcode reicht für ~256 Kommando-Typen. Ist das genug für zukünftige Canvas-Erweiterungen?
2. **Font-Subset-Streaming:** Soll der Server nur die Glyphen streamen, die im aktuellen Frame sichtbar sind (wie progressive Web Fonts)?
3. **Shader-Streaming:** Soll der WGSL-Shader-Code für Custom-Paint (v2) über den Asset-Kanal kommen?
4. **Video-Integration:** Wie interagiert Vellum mit Surface-Slots (RFC-001 §8), die Video streamen? Separater Media-Kanal?
5. **Backward-Compatibility:** Wie streng ist die Protokoll-Versionierung? Muss ein v1.1-Server mit einem v1.0-Client sprechen können?

### 16.7 Surface-Slot-spezifische Risiken

1. **Rekursive Schleifen:** Server A deklariert einen Slot, der von Server B gefüllt wird, der einen Slot deklariert, der von Server A gefüllt wird. Der Client muss Zyklen erkennen und abbrechen (max depth + Adress-Deduplication im Upstream-Manager).
2. **Input-Routing-Ambiguität bei überlappenden Slots:** Wenn zwei Surface-Slots sich überlappen, muss der Client eine Z-Order-Regel haben. Empfehlung: Z-Index aus `DeclareSurfaceSlot`, bei Gleichheit gewinnt der zuletzt deklarierte Slot.
3. **Latenz-Kaskadierung:** Bei rekursivem Vellum (A → B → C) addieren sich die Latenzen. Prediction funktioniert nur auf der ersten Ebene. Für tiefe Ketten wird die gefühlte Latenz zum Problem. Mitigation: Der Client kann Upstream-Verbindungen direkt öffnen statt sie durch den primären Server zu tunneln.
4. **AccessTree-Konsistenz:** Wenn Server A und Server B unabhängig ihre AccessTrees updaten, kann der merged Tree temporär inkonsistent sein (Server A referenziert einen Node, der in Server B's nächstem Update verschwindet). Mitigation: Versionierte Subtree-Merges mit Tombstone-Markern.
5. **Asset-Isolation:** Wenn Server A und Server B denselben Font-Hash verwenden, aber verschiedene Font-Dateien meinen (Hash-Kollision), entsteht ein Rendering-Fehler. Mitigation: xxHash64 hat <10⁻¹⁸ Kollisionswahrscheinlichkeit bei <10⁶ Assets — akzeptabel. Alternativ: Namespace-Prefix pro Server.

---

## 17. Fazit

Vellum ist kein separates Produkt — es ist die logische Konsequenz aus Lux' Architekturentscheidungen:

- Die **Elm-Architektur** liefert deterministische, serialisierbare State-Updates — die Grundlage für Multi-User und Prediction.
- Die **Canvas-API** ist bereits eine vollständige, deklarative Beschreibung aller Zeichenoperationen — sie muss nur serialisiert werden.
- Der **AccessTree** transportiert Semantik parallel zum visuellen Stream — ein Alleinstellungsmerkmal gegenüber jedem existierenden Remote-Display-Protokoll.
- Die **WGPU-Pipeline** rendert den deserialisierten Stream auf jeder Plattform nativ — ohne Pixel-Scaling, ohne Qualitätsverlust.

Das Resultat ist ein Protokoll, aus dem fünf Produkte fallen: Remote-Display, Inspector, Test-Harness, Screen-Recording, Browser-Client — und als Krönung: Multi-User-Collaboration out of the box. Durch Layer 4 (Surface-Slots) kommt eine sechste Dimension hinzu: **Composable Applications** — unabhängige Services, die ihre UIs in einem einzigen Client-Fenster vereinen, mit lokaler Surface-Injektion (Video, 3D) und rekursiver Komposition über Vellum-Server-Grenzen hinweg.

Der geschätzte Aufwand (~12.000 LOC, ~15–19 Wochen) ist überschaubar, weil Vellum kein neues System baut, sondern eine Serialisierungsschicht über ein bestehendes System legt. Die härteste Arbeit — Layout-Engine, Renderer, Input-System, Accessibility, Surface-Slots — ist bereits getan.

**Empfohlene Lesart von RFC-005:**

- Als strategische Erweiterung, die Lux von einem UI-Toolkit zu einer **Application-Delivery-Platform** macht.
- Als Protokoll-Entwurf, der den nächsten konkreten Schritt definiert: Phase 1 (Canvas-Roundtrip) ist ein 3–4-Wochen-Projekt mit sofortigem Nutzen (Inspector, Golden-File-Tests).

---

*RFC-005 — Theoretical. Feedback und Änderungsvorschläge bitte als Issue gegen dieses Dokument.*