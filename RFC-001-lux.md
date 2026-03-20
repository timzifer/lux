# RFC-001 — lux: Core Architecture

**Repository:** `github.com/timzifer/lux`
**Status:** Teilweise integriert
**Version:** 0.2.0
**Datum:** 2026-03-18
**Zuletzt abgeglichen:** 2026-03-20
**Folge-RFCs:** RFC-002 (Interaction & Layout), RFC-003 (Widget Catalogue & Theme)

---

### Implementierungsstatus

| Abschnitt | Status | Anmerkung |
|-----------|--------|-----------|
| §1 Motivation & Abgrenzung | — | Kontext, kein Code |
| §2 Architektur-Überblick | ✅ Integriert | Alle Layer vorhanden (app, ui, draw, theme, platform, internal) |
| §3 Elm-Architektur & App-Loop | ✅ Integriert | `Run[M]`, `Send`/`TrySend`, `TickMsg`, dt-Clamping |
| §3.4 State Persistence | ✅ Integriert | `WithPersistence`, `PersistenceConfig[M]`, Encode/Decode Hooks |
| §3.5 Sub-Models | ✅ Integriert | `SubModel[Parent, Child]`, `Delegate`, `SubModelWithCmd` |
| §3.6 Commands (`Cmd`) | ✅ Integriert | `Cmd`, `UpdateWithCmd[M]`, `Batch`, `None` |
| §4 Widget-System | ✅ Integriert | `Widget`, `WidgetState`, `RenderCtx`, `AdoptState[S]`, `Element`, `UID`, `WithKey` |
| §4 RenderCtx.Events | ✅ Integriert | Input-Dispatch via `Dispatcher` an Widgets |
| §4 Equatable Interface | ✅ Integriert | `ui/element.go` |
| §4 DirtyTracker Interface | ✅ Integriert | `ui/element.go` |
| §5 Theming-System | ✅ Integriert | `Theme`-Interface, `TokenSet`, `DrawFunc`, `DrawCtx`, `Override` |
| §5 Slate Dark + Light | ✅ Integriert | Vollständige Token-Werte wie spezifiziert |
| §5.4 Resolved-Cache | ✅ Integriert | `CachedTheme` in `theme/cache.go` |
| §6 Rendering-Pipeline | ✅ Integriert | wgpu (gogpu) + OpenGL 3.3+ Fallback; Scene-Graph, Shadows, Blur, Gradients, Opacity |
| §6.2 Canvas-API | ✅ Integriert | Alle Primitives: Blur, Gradients, Layer, ArcTo, DrawTextLayout, DrawImageSlice, DrawTexture, PushScale, PushClipRoundRect, PushClipPath |
| §6.3 SDF-Text | ✅ Integriert | MSDF-Atlas (NRGBA, 32px Range), Dual-Path (MSDF ≥24px, Bitmap <24px) |
| §6.4 VTree-Diff / Reconcile | ✅ Integriert | `ui/reconcile.go` |
| §6.4 VirtualList | ✅ Integriert | `ui/virtual_list.go` |
| §7 Platform-Abstraktion | ✅ Integriert | Interface + GLFW, Wayland, X11, Win32, Cocoa, DRM/KMS Backends |
| §8 Externe Surfaces | ✅ Integriert | `SurfaceProvider`-Interface, `AcquireFrame`/`ReleaseFrame`, Input-Routing |
| §11 Accessibility (A11y) | 🔶 Teilweise | Core-Typen (`AccessRole`, `AccessNode`, `AccessStates`, `SemanticProvider`) vorhanden; Bridges + AccessTree-Konstruktion ausstehend |
| §12 Inspector & Debugging | ⏳ Wartend | |

---

## Inhaltsverzeichnis

1. [Motivation & Abgrenzung](#1-motivation--abgrenzung)
2. [Architektur-Überblick](#2-architektur-überblick)
3. [Kern: Elm-Architektur & App-Loop](#3-kern-elm-architektur--app-loop)
4. [Widget-System & WidgetState](#4-widget-system--widgetstate)
5. [Theming-System — Interfaces & Caching](#5-theming-system--interfaces--caching)
6. [Rendering-Pipeline](#6-rendering-pipeline)
7. [Platform-Abstraktion](#7-platform-abstraktion)
8. [Externe Surfaces](#8-externe-surfaces)
9. [Offene Fragen](#9-offene-fragen)
10. [Nicht-Ziele](#10-nicht-ziele)
11. [Accessibility (A11y)](#11-accessibility-a11y--first-class-feature)
12. [Ausblick: Inspector & Debugging-Tools](#12-ausblick-inspector--debugging-tools)
13. [Implementierungs-Leitfaden](#13-implementierungs-leitfaden)

**Anhänge**
- [Anhang A: Naming Conventions](#anhang-a-naming-conventions)
- [Anhang B: Minimales Beispielprogramm](#anhang-b-minimales-beispielprogramm)

---

## 1. Motivation & Abgrenzung

### Das Problem mit bestehenden Go-UI-Toolkits

Go fehlt ein UI-Toolkit, das die Spracheigenschaften von Go — starke Typisierung, einfache Concurrency, explizite Interfaces — konsequent auf UI überträgt. Die bestehenden Optionen scheitern an vorhersehbaren Stellen:

- **Fyne**: Native Widgets, aber Race-Conditions durch `fyne.Do`-Pflicht, schwer erweiterbar.
- **Gio**: GPU-beschleunigt und gut designt, aber Immediate-Mode-API ist ungewohnt und Widget-Komposition ist komplex.
- **Wails / go-app**: Browser-Engine als Backend — kein echtes natives Rendering, JS-Brücke als Engpass.

Keines davon ist strukturell thread-safe ohne Disziplin vom Aufrufer. Keines unterstützt Bare-Metal (DRM/KMS) als First-Class-Target.

### Zielgruppe

- Go-Entwickler, die Desktop-Anwendungen bauen
- Embedded/Kiosk/Industrie-HMI (Bare Metal)
- Entwickler, die externe Komponenten (Browser, 3D) integrieren müssen
- Teams, die Race-Condition-freien UI-Code ohne Review-Overhead wollen

### Der Mindset-Shift: Elm für Go-Entwickler

Die Elm-Architektur ist für Go-Entwickler die aus `net/http`-Handlern, Goroutinen und geteiltem Mutex-geschütztem State kommen ein echter Sprung. Das soll nicht verschwiegen werden.

**Was sich anfühlt wie eine Einschränkung, ist eine Garantie:**

```go
// Gewohnt: State irgendwo mutieren, GUI irgendwie updaten
mu.Lock()
app.items = append(app.items, newItem)
mu.Unlock()
listWidget.Refresh()  // Vergessen? Race Condition.

// Hier: eine Funktion, ein Ergebnis, kein shared State
func update(m Model, msg Msg) Model {
    switch msg := msg.(type) {
    case AddItemMsg:
        m.Items = append(m.Items, msg.Item)
    }
    return m  // Neues Model — Framework kümmert sich um den Rest
}
```

Der Lernaufwand konzentriert sich auf die erste Woche. Danach gibt es keine versteckten Konzepte mehr die einen überraschen — das ist das Versprechen der Architektur.

**Für Go-Entwickler die noch nie Elm gesehen haben** empfiehlt sich ein Blick auf `github.com/charmbracelet/bubbletea` — dieselbe Architektur für Terminal-UIs, sehr verbreitet in der Go-Community. Wer Bubbletea kennt, kennt bereits 80% der Konzepte dieses Toolkits.

---

## 2. Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────┐
│                        User Code                            │
│   Model (struct)   │   update(Model, Msg) Model   │  view   │
└────────────────────┴──────────────────────────────┴─────────┘
            │ app.Send(msg)                  │ view()
            ▼                               ▼
┌─────────────────────────────────────────────────────────────┐
│                     Framework Core                          │
│                                                             │
│  ┌──────────────┐    ┌──────────────────────────────────┐   │
│  │  Msg Channel │───▶│  Single-Threaded App Loop        │   │
│  │  (buffered)  │    │  update → diff → render          │   │
│  └──────────────┘    └──────────────┬───────────────────┘   │
│                                     │                        │
│   ┌─────────────────────────────────┼───────────────────┐   │
│   │   Map[UID]WidgetState           │  Theme             │   │
│   │   (framework-intern)            │  (injiziert)       │   │
│   └─────────────────────────────────┴───────────────────┘   │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Rendering Layer                             │
│   wgpu (Vulkan / Metal / D3D12 / WebGPU)                    │
│   2D Renderer   │   SDF Text   │   Surface Slots            │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Platform Layer                              │
│   Wayland  │  X11  │  Win32  │  Cocoa  │  DRM/KMS           │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Kern: Elm-Architektur & App-Loop

### 3.1 Einstiegspunkt

```go
func Run[M any](model M, update UpdateFunc[M], view ViewFunc[M], opts ...Option) error
```

Das ist der einzige Weg, eine Applikation zu starten. `Run` blockiert bis zum Ende der Anwendung.

**Typaliase:**
```go
type UpdateFunc[M any] func(M, Msg) M
type ViewFunc[M any]   func(M) Element
```

### 3.2 Externe Kommunikation

```go
// Sendet eine Nachricht in den App-Loop. Thread-safe. Blockiert nie (gepufferter Channel).
func Send(msg Msg)

// Variante mit Timeout-Semantik für Fire-and-Forget aus Goroutinen
func TrySend(msg Msg) bool
```

Es gibt keinen anderen Weg, den App-Zustand zu beeinflussen. Goroutinen, HTTP-Callbacks, Timer-Ticks — alle laufen durch `Send`.

### 3.3 Der App-Loop

Der Loop läuft single-threaded in der Goroutine, die `Run` aufruft.

```
lastFrame := now()

for {
    select {
    case msg := <-msgChannel:
        newModel = update(currentModel, msg)
        if newModel != currentModel {
            newVTree = view(newModel)
            diff(currentVTree, newVTree)  → patchList
            applyPatches(patchList)       → renderCommands
            currentModel = newModel
            currentVTree = newVTree
        }

    case <-renderTick:
        // ── dt-Berechnung & Clamping ──────────────────────────────
        now   := now()
        dt    := now - lastFrame
        dt     = min(dt, maxFrameDelta)   // Clamping (siehe unten)
        lastFrame = now

        // ── Animation-Pass (§12) ──────────────────────────────────
        for uid, state := range widgetStates {
            if anim, ok := state.(Animator); ok {
                stillRunning := anim.Tick(dt)
                if stillRunning {
                    markDirty(uid)   // → wird neu gezeichnet
                }
                // AnimationEnded-Msgs für user-initiierte Anims (§12.8):
                flushAnimationEndMsgs(uid, state)
            }
        }

        gpu.Submit(pendingCommands)

    case <-platformEvents:
        // Keyboard, Mouse, Resize → in Msg umwandeln → msgChannel
    }
}
```

#### dt-Clamping

`dt` (delta time) ist die verstrichene Zeit zwischen zwei Frames. Ohne Clamping entstehen zwei Probleme:

**Problem 1 — App-Freeze:** Die App war kurz eingefroren (Debugger, OS-Scheduler, Sleep). Erster Frame danach: `dt` = mehrere Sekunden. Alle laufenden Animationen machen einen riesigen Sprung oder schließen sofort ab.

**Problem 2 — Spiral of Death:** Ein Frame dauert länger als erwartet → nächster `dt` ist groß → Animation-Berechnung dauert länger → nächster `dt` noch größer.

Lösung: `dt` wird nach oben geclampt:

```go
const maxFrameDelta = 100 * time.Millisecond
// Wahl von 100ms: entspricht ~2 verpassten Frames bei 16ms-Target (60fps).
// Genug Puffer für kurze Hänger, aber kein unkontrollierbarer Zeitsprung.
// Kann via app.WithMaxFrameDelta(d) überschrieben werden.
```

**Verhalten bei großem dt nach einem Freeze:**
- Animationen springen auf einen Wert der maximal 100ms Fortschritt entspricht.
- Bei einer 250ms-Animation: maximal 40% Sprung statt sofortigem Abschluss.
- Für die meisten UI-Animationen ist das nicht wahrnehmbar.

**Wichtig:** Das Clamping betrifft ausschließlich den Animation-Pass. `update` und `view` haben kein Zeitkonzept — sie bleiben vollständig deterministisch und erhalten nie einen `dt`-Wert.

**Invarianten:**
- `update` wird ausschließlich im App-Loop aufgerufen.
- Kein Usercode (außer `view` und `update`) hat Zugriff auf `currentModel`.
- `view` ist eine reine Funktion — keine Seiteneffekte, kein I/O.
- `dt` ist immer ∈ `(0, maxFrameDelta]` — niemals Null, niemals negativ, niemals unbegrenzt.

### 3.4 State Restoration & Persistenz

Das User-Model ist ein typisierter Struct — damit ist es von Haus aus serialisierbar. Das Framework bietet optionale Hooks für persistenten State zwischen App-Neustarts.

```go
// WithPersistence registriert Encode/Decode-Funktionen für das Model.
// Das Framework ruft Encode beim Beenden auf und Decode beim Start.
// Gibt Decode einen Fehler zurück (z.B. inkompatibles Format nach Update),
// wird das initiale Model aus Run verwendet — kein Crash.
app.Run(initialModel, update, view,
    app.WithPersistence(app.PersistenceConfig[Model]{
        // Encode serialisiert das Model. Empfohlen: encoding/json oder
        // encoding/gob — beides pure Go, keine Framework-Abhängigkeit.
        Encode: func(m Model) ([]byte, error) {
            return json.Marshal(m)
        },
        Decode: func(data []byte) (Model, error) {
            var m Model
            return m, json.Unmarshal(data, &m)
        },
        // StorageKey: Identifiziert den gespeicherten State (für mehrere
        // Fenster oder mehrere App-Instanzen).
        StorageKey: "main-window",
    }),
)
```

**Was persistiert wird und was nicht:**

| State | Persistiert? | Begründung |
|-------|-------------|------------|
| User-Model | Opt-in via Encode/Decode | Volle Kontrolle beim Entwickler |
| WidgetState (intern) | Nein | UI-State, nicht Applikations-State |
| Focus | Nein | Wird beim Start neu gesetzt |
| Scroll-Positionen | Nein¹ | UI-State |
| Theme | Nein² | Via SetThemeMsg beim Start wiederherstellbar |

¹ Scroll-Positionen können im User-Model gespeichert werden wenn gewünscht — `KineticScroll.SnapToImmediate` setzt die initiale Position ohne Animation.
² Theme-Präferenz (Dark/Light) gehört ins User-Model wenn sie persistiert werden soll.

**Storage-Backend:** Das Framework schreibt in eine plattformübliche Location:
- Linux: `$XDG_STATE_HOME/<app>/<key>.bin` (Fallback: `~/.local/state`)
- macOS: `~/Library/Application Support/<app>/<key>.bin`
- Windows: `%APPDATA%/<app>/<key>.bin`
- DRM/KMS: Konfigurierbarer Pfad via `app.WithStoragePath`

### 3.5 Sub-Models

Für große Applikationen können Sub-Models registriert werden. Jedes Sub-Model hat seine eigene `update`-Funktion; der Haupt-Loop delegiert.

```go
type SubModel[Parent, Child any] struct {
    Get    func(Parent) Child
    Set    func(Parent, Child) Parent
    Update UpdateFunc[Child]
}
```

Die Komposition erfolgt explizit — keine Magie, keine Reflection.

### 3.6 Nachrichten & Commands

```go
type Msg interface{}  // Marker-frei, jeder Typ ist eine Msg

// Commands sind Seiteneffekte, die update zurückgeben kann
type Cmd func() Msg   // Wird asynchron ausgeführt, Ergebnis via Send

// update-Variante mit Command-Support
type UpdateWithCmd[M any] func(M, Msg) (M, Cmd)
```

`Run` akzeptiert beide Signaturen. Commands sind optional — einfache Apps brauchen sie nicht.

---

## 4. Widget-System & WidgetState

### 4.1 Das WidgetState-Interface

```go
// WidgetState ist ein offenes Interface — keine Marker-Methoden.
// Drittanbieter implementieren es vollständig außerhalb des Framework-Packages.
type WidgetState interface {
    // Intentionally empty — jeder Typ ist WidgetState
}
```

Der Framework verwaltet eine interne Map:

```go
// Intern, nie im Userland sichtbar
type stateRegistry map[UID]WidgetState
```

### 4.2 Widget-Interface

```go
type Widget interface {
    // Render gibt einen Element-Baum zurück.
    // state ist nil beim ersten Aufruf.
    Render(ctx RenderCtx, state WidgetState) (Element, WidgetState)
}
```

```go
type RenderCtx struct {
    UID    UID
    Theme  Theme      // Aktuelles Theme (siehe §5)
    Send   func(Msg)  // Lokales Send — bindet UID automatisch
    Events []InputEvent  // Input-Events dieses Frames (§13.6)
}

// adoptState ist eine generische Hilfsfunktion die vom Framework bereitgestellt wird.
// Gibt den bestehenden WidgetState als konkreten Typ zurück, oder einen neuen
// Zero-Value wenn rawState nil oder ein anderer Typ ist (erster Render-Aufruf,
// oder Widget-Typ hat gewechselt).
//
//   state := adoptState[ButtonState](rawState)
//
// Implementiert via Generics + Type-Assert — kein Reflection.
func adoptState[S WidgetState](raw WidgetState) *S
```

### 4.3 Element-Typen

```go
type Element interface{ isElement() }

// Eingebaute Element-Typen:
type Box      struct { ... }  // Layout-Container
type Text     struct { ... }  // Text-Knoten
type Image    struct { ... }  // Textur/Bitmap
type Surface  struct { ... }  // Externer Surface-Slot (§8)
type Custom   struct { ... }  // Eigener Draw-Call
```

### 4.4 UID-System

UIDs sind stabil über Frames hinweg und werden deterministisch aus der Position im Element-Baum abgeleitet (ähnlich React's `key`). Explizite Keys sind möglich:

```go
WithKey("my-list-item-42", myWidget)
```

#### Re-parenting & UID-Stabilität

Ohne expliziten Key ist die UID positionsbasiert — ein Widget das im Baum verschoben wird (Re-parenting) bekommt eine neue UID. Das hat zwei Konsequenzen:

- **`WidgetState` geht verloren** — der neue Node startet mit nil-State (erster Render-Aufruf).
- **Laufende Animationen brechen ab** — `Anim[T]`-Werte im alten `WidgetState` sind weg. Pending `AnimationEnded`-Msgs mit einer `AnimationID` werden nie gesendet.

**Lösung: expliziter Key bei Re-parenting.**

```go
// Ohne Key: UID = Position im Baum → instabil bei Re-parenting
ui.ListItem(item)

// Mit Key: UID = hash("item-" + item.ID) → stabil über Re-parenting hinweg
ui.WithKey("item-"+item.ID, ui.ListItem(item))
```

Mit explizitem Key bleibt `WidgetState` und damit alle laufenden Animationen erhalten — unabhängig davon wohin das Widget im Baum verschoben wird.

**Faustregel:** Jedes Widget das in einer Liste, einem Tree oder einem dynamisch umstrukturierten Layout lebt und `Animator` implementiert, *muss* einen expliziten Key bekommen. Das Framework gibt in Debug-Builds eine Warnung aus wenn ein Widget ohne Key seinen State verliert und `Animator` implementiert hatte.

---

## 5. Theming-System

Das Theming-System ist der neue Bestandteil, der zu den Key-Aspekten hinzukommt. Es verfolgt diese Designziele:

- **Nicht im User-Model** — Themes sind Laufzeit-Konfiguration, kein Applikationszustand.
- **Vollständig erweiterbar** — Custom-Draw-Hooks ohne Framework-Fork.
- **Composable** — Themes bauen auf anderen Themes auf (Prototype-Chain-Semantik).
- **Updatebar zur Laufzeit** — Theme-Wechsel (Dark/Light, Branding) via `Send`.

### 5.1 Das Theme-Interface

```go
type Theme interface {
    // Tokens liefert das Design-Token-Set dieses Themes.
    Tokens() TokenSet

    // DrawFunc liefert eine optionale custom Draw-Implementierung für einen
    // Widget-Typ. Gibt nil zurück → Framework-Default wird verwendet.
    DrawFunc(widgetKind WidgetKind) DrawFunc

    // Parent liefert das Parent-Theme für Fallback-Lookups.
    // Nil = kein Parent (Root-Theme).
    Parent() Theme
}
```

### 5.2 TokenSet

```go
type TokenSet struct {
    Colors     ColorScheme
    Typography TypographyScale
    Spacing    SpacingScale
    Radii      RadiusScale
    Elevation  ElevationScale
    Motion     MotionSpec     // Durations, Easing-Curves (§12.6)
    Scroll     ScrollSpec     // Scroll-Physik: Friction, Overscroll (§14.4)
}

// ColorScheme nutzt semantische Slots statt hart kodierter Farben.
// Jeder Slot hat eine definierte Bedeutung — Custom-Renderer können sich
// auf diese Semantik verlassen ohne konkrete Werte zu kennen.
type ColorScheme struct {
    // Surface-Gruppe: Hintergrundflächen
    Surface struct {
        Base     Color  // Fenster-Hintergrund — tiefste Ebene
        Elevated Color  // Cards, Overlays — eine Ebene höher
        Hovered  Color  // Widget-Hover-State (kein Border-Token-Missbrauch)
        Pressed  Color  // Widget-Active-State
    }

    // Accent-Gruppe: Primäre Interaktionsfarbe
    Accent struct {
        Primary         Color  // Hauptfarbe (Buttons, Links, Focus-Indikator)
        PrimaryContrast Color  // Text auf Primary (meist Weiß oder Schwarz)
        Secondary       Color  // Optionale zweite Akzentfarbe
    }

    // Stroke-Gruppe: Linien und Rahmen
    Stroke struct {
        Border Color  // Subtile Trennung (1px solid, niedrige Opacity)
        Focus  Color  // Starker Kontrast für Keyboard-Navigation
        Divider Color // Noch subtiler als Border (Abschnittstrennungen)
    }

    // Text-Gruppe: Schriftfarben
    Text struct {
        Primary   Color  // Haupttext
        Secondary Color  // Beschriftungen, Metadaten (gedimmt)
        Disabled  Color  // Deaktivierte Elemente
        OnAccent  Color  // Text auf Accent.Primary
    }

    // Status-Gruppe: Semantische Zustände
    Status struct {
        Success Color
        Warning Color
        Error   Color
        Info    Color
        // Je ein "Contrast"-Token für Text auf dem Status-Hintergrund:
        OnSuccess Color
        OnError   Color
    }

    // Erweiterbar ohne Breaking Change:
    Custom map[string]Color
}

// TypographyScale: Desktop-First.
// Desktop-User haben Mäuse, keine Daumen — kompaktere Größen als Mobile.
type TypographyScale struct {
    // Überschriften:
    H1 TextStyle  // 20dp, SemiBold — Seitentitel
    H2 TextStyle  // 16dp, SemiBold — Abschnittstitel
    H3 TextStyle  // 14dp, Medium   — Untertitel

    // Fließtext:
    Body     TextStyle  // 13dp, Regular — Standard-Fließtext
    BodySmall TextStyle // 12dp, Regular — Metadaten, Beschriftungen

    // Labels:
    Label     TextStyle  // 12dp, Medium  — Button-Text, Tab-Labels
    LabelSmall TextStyle // 11dp, Medium  — Badges, Chips

    // Code/Mono — für Go-Entwickler besonders wichtig:
    Code      TextStyle  // 13dp, Regular, Monospace
    CodeSmall TextStyle  // 12dp, Regular, Monospace — Inline-Code
}

type TextStyle struct {
    FontFamily string
    Size       float32    // dp
    Weight     FontWeight
    LineHeight float32    // multiplier; 1.4 Standard, 1.6 für Body
    Tracking   float32    // em; negativ für enge Headlines
}
```

### 5.3 DrawFunc & Custom Rendering

```go
type DrawFunc func(ctx DrawCtx, tokens TokenSet, state WidgetState)

type DrawCtx struct {
    Canvas  Canvas     // 2D-Zeichenoperationen (§6.2)
    Bounds  Rect
    DPR     float32    // Device-Pixel-Ratio
    Focused bool
    Hovered bool
    Pressed bool
}
```

Ein Theme kann für jeden Widget-Typ eine eigene `DrawFunc` registrieren. Das ist der Escape-Hatch für vollständig custom-gerenderte Widgets (z.B. Branded Components, Spiele-UI-Elemente):

```go
type myBrandTheme struct {
    base theme.Default // Eingebettetes Basis-Theme
}

func (t myBrandTheme) DrawFunc(kind WidgetKind) DrawFunc {
    switch kind {
    case WidgetKindButton:
        return drawMyBrandButton   // Vollständig custom
    default:
        return t.base.DrawFunc(kind)  // Fallback auf Base-Theme
    }
}

func drawMyBrandButton(ctx DrawCtx, tokens TokenSet, state WidgetState) {
    bs := state.(ButtonState)
    // Direkter Canvas-Zugriff — keine Einschränkungen
    ctx.Canvas.FillRoundRect(ctx.Bounds, tokens.Radii.Medium, tokens.Colors.Primary)
    ctx.Canvas.DrawText(bs.Label, tokens.Typography.LabelSmall, tokens.Colors.OnPrimary)
    if bs.Pressed {
        ctx.Canvas.FillRect(ctx.Bounds, Color{A: 0.12})  // Ripple-Overlay
    }
}
```

### 5.4 Theme-Lookup-Algorithmus & Caching

#### Lookup-Semantik

Token- und DrawFunc-Lookups folgen der Parent-Chain:

```
DrawFunc(kind) → eigene Map → Parent.DrawFunc(kind) → ... → nil (Framework-Default)
Token(key)     → eigener TokenSet → Parent.Token(key) → ... → panic (fehlender Required-Token)
```

Das verhindert, dass ein Custom-Theme unvollständig ist und Runtime-Panics verursacht.

#### Das Performance-Problem

Eine naive Implementierung würde bei jedem Frame für jeden Widget-Draw-Call die Parent-Chain traversieren. Bei einer Chain der Tiefe *d* und *n* Widgets pro Frame kostet das O(n·d) Pointer-Dereferenzierungen — kein katastrophaler Overhead, aber unnötig und nicht cache-freundlich.

#### Lösung: Flacher Resolved-Cache pro Theme-Instanz

Jede `Theme`-Instanz trägt intern einen **Resolved-Cache** — eine flache Map die beim ersten Lookup für einen gegebenen `WidgetKind` oder Token-Key befüllt wird:

```go
// Intern im Framework — nie im Usercode sichtbar
type resolvedCache struct {
    drawFuncs map[WidgetKind]DrawFunc   // nil-Einträge = Framework-Default
    tokens    *TokenSet                  // vollständig aufgelöst, keine Lücken
    valid     bool
}
```

**Beim ersten Lookup** für `DrawFunc(ButtonKind)`:
1. Traversiere Parent-Chain einmalig
2. Schreibe Ergebnis in `resolvedCache.drawFuncs[ButtonKind]`
3. Alle folgenden Lookups in diesem Frame: direkter Map-Zugriff, O(1)

**Cache-Invalidierung:**
Der Cache ist an die Theme-Instanz gebunden, nicht an den Frame. Er wird invalidiert wenn:
- `SetThemeMsg` ein neues Theme aktiviert (neue Instanz → neuer leerer Cache)
- `SetDarkModeMsg` das Token-Set wechselt (partieller Invalidate: nur `tokens`)

Da Themes immutable sind sobald sie registriert wurden, gibt es keine Invalidierung durch Mutation — es gibt keine Mutation.

#### TokenSet: Einmalig aufgelöst, dann kopiert

`TokenSet` wird beim ersten Zugriff vollständig durch die Parent-Chain aufgelöst und als flache Kopie gecacht. Das hat zwei Konsequenzen:

- **Kein Pointer-Chasing zur Laufzeit**: Alle Token-Zugriffe in `DrawFunc` und `Widget.Render` treffen ein vollständig befülltes Struct — kein nil-Check, kein Fallback-Lookup.
- **Cache-Größe ist konstant**: Unabhängig von der Tiefe der Parent-Chain ist der Cache immer ein einzelnes `TokenSet`-Struct. Speicher ist O(1), nicht O(d).

```
Theme-Chain (Tiefe d):          Resolved Cache (Tiefe 1):
  myBrandTheme                    TokenSet {
    └─ theme.MaterialDark           Colors:     { Primary: #C0392B, ... }  ← aus myBrandTheme
         └─ theme.Base              Typography: { ... }                     ← aus MaterialDark
                                    Spacing:    { ... }                     ← aus Base
                                  }
```

#### Warm-Up: Kein First-Frame-Hitch

Um den ersten Frame nicht mit Lookup-Arbeit zu belasten, löst `app.Run` den Theme-Cache **synchron vor dem ersten Frame** auf — als Teil der Initialisierungsphase, nicht lazy:

```go
// Intern in app.Run, vor dem Loop:
activeTheme.resolveCache(knownWidgetKinds)
```

`knownWidgetKinds` ist eine statische Liste aller eingebauten Widget-Typen. Drittanbieter-Widgets die nach dem Start registriert werden, lösen ihren ersten Lookup lazy aus — danach gecacht.

#### Zusammenfassung

| Szenario | Kosten |
|---|---|
| Erster Lookup eines Widget-Kinds | O(d) — einmalig |
| Folge-Lookups im selben Frame | O(1) |
| Theme-Wechsel zur Laufzeit | O(k) einmalig — k = Anzahl bekannter Widget-Kinds |
| Token-Zugriff in DrawFunc | O(1) — Struct-Feld-Zugriff |

### 5.5 Theme-Wechsel zur Laufzeit

Themes sind kein Userland-State, aber der Framework-Loop kennt das aktive Theme:

```go
// Eingebaute Msg-Typen des Frameworks:
type SetThemeMsg struct{ Theme Theme }
type SetDarkModeMsg struct{ Dark bool }

// Usage:
app.Send(SetThemeMsg{Theme: myBrandTheme{}})
```

Der Loop wendet das neue Theme beim nächsten Frame an. Es gibt kein Re-Rendering des gesamten Baums — nur Widgets, deren `DrawFunc` sich geändert hat, werden neu gezeichnet (via Dirty-Tracking).

### 5.6 Theme-Komposition: Partial Overrides

Ein häufiges Pattern ist, ein Base-Theme minimal zu überschreiben (z.B. nur Farben ändern):

```go
// theme.Override ist ein Convenience-Wrapper
myTheme := theme.Override(theme.Default, theme.OverrideSpec{
    Colors: &ColorScheme{
        Primary:   mustParseHex("#C0392B"),
        OnPrimary: mustParseHex("#FFFFFF"),
    },
})
```

Intern erstellt `theme.Override` ein Theme, das für alle nicht-überschriebenen Felder an den Parent delegiert.

### 5.7 Token-Zugriff in Widgets

Innerhalb eines Widgets greift man über `RenderCtx.Theme` auf Tokens zu:

```go
func (b Button) Render(ctx RenderCtx, state WidgetState) (Element, WidgetState) {
    tokens := ctx.Theme.Tokens()
    label := Text{
        Content: b.Label,
        Style:   tokens.Typography.Label,
        Color:   tokens.Colors.Text.Primary,
    }
    // ...
}
```

**Vollständige Token-Werte, `theme.Slate` und Widget-State-Konventionen:** → RFC-003 §2.

---

## 6. Rendering-Pipeline

### 6.1 wgpu als GPU-Backend

Das Framework abstrahiert den wgpu-Zugriff hinter einem internen Shim-Interface. Zwei Implementierungen existieren, umschaltbar via Build-Tag — analog zur Platform-Abstraktion in §7.2:

```
go build              → wgpu-native (CGo, produktionsreif, Default)
go build -tags gogpu  → gogpu/wgpu  (pure Go, CGO_ENABLED=0)
```

#### Das interne Shim-Interface

```go
// wgpu-Shim — internes Package lux/internal/wgpu
// Usercode sieht dieses Interface nie.
// Beide Implementierungen müssen es vollständig erfüllen.
type Instance interface {
    CreateSurface(desc SurfaceDescriptor) Surface
    RequestAdapter(opts AdapterOptions) (Adapter, error)
}

type Adapter interface {
    RequestDevice(desc DeviceDescriptor) (Device, error)
    Info() AdapterInfo
}

type Device interface {
    CreateBuffer(desc BufferDescriptor) Buffer
    CreateTexture(desc TextureDescriptor) Texture
    CreateShaderModule(desc ShaderModuleDescriptor) ShaderModule
    CreateRenderPipeline(desc RenderPipelineDescriptor) RenderPipeline
    CreateCommandEncoder() CommandEncoder
    Queue() Queue
}

// ... Surface, Queue, Buffer, Texture — vollständige WebGPU-Abdeckung
// Die Typen spiegeln die WebGPU-Spec 1:1 — beide Implementierungen
// nutzen dieselbe Spec als gemeinsame Grundlage.
```

Da beide Implementierungen — `wgpu-native` und `gogpu/wgpu` — die WebGPU-Spezifikation implementieren, ist das Shim dünn: es mappt im Wesentlichen Typen und Methodennamen, keine Logik. Ein Aufruf wie `device.CreateBuffer(desc)` ist auf beiden Seiten semantisch identisch.

#### Implementierung A — Default: `wgpu-native` via CGo

```go
// +build !gogpu

// lux/internal/wgpu/native/instance.go
// Thin wrapper um go-webgpu/webgpu (CGo-Binding gegen wgpu-native).
type nativeInstance struct {
    inner gowebgpu.Instance
}

func NewInstance() wgpu.Instance {
    return &nativeInstance{inner: gowebgpu.CreateInstance(nil)}
}
```

#### Implementierung B — `-tags gogpu`: `gogpu/wgpu` pure Go

```go
// +build gogpu

// lux/internal/wgpu/gogpu/instance.go
// Thin wrapper um gogpu/wgpu (pure Go, CGO_ENABLED=0).
type gogpuInstance struct {
    inner gogpuwgpu.Instance
}

func NewInstance() wgpu.Instance {
    return &gogpuInstance{inner: gogpuwgpu.CreateInstance(nil)}
}
```

#### Backend-Unterstützung (beide Implementierungen)

| Platform | Backend |
|----------|---------|
| Linux (Wayland/X11) | Vulkan |
| macOS | Metal |
| Windows | D3D12 (D3D11 Fallback) |
| Bare Metal (DRM/KMS) | Vulkan |
| Software (CI, Headless) | CPU-Rasterizer |
| Browser (WASM) | WebGPU |

#### Shim-Stabilität

Der Shim ist das stabilste interne Interface des Frameworks — eine Änderung hier betrifft beide Implementierungen. Er folgt deshalb der WebGPU-Spec strikt und fügt keine eigenen Abstraktionen hinzu. Wenn die WebGPU-Spec eine Operation hat, hat der Shim sie auch — nichts mehr, nichts weniger.

### 6.2 Canvas-API (2D-Renderer)

Die Canvas-API ist das stabilste öffentliche Interface des Frameworks — sie ist der primäre Kontaktpunkt für alle Drittanbieter-Widgets und Custom-DrawFuncs (§5.3). Stabilität und Vollständigkeit haben hier Vorrang vor Schlankheit.

#### 6.2.1 Koordinatensystem & Einheiten

Alle Koordinaten sind in **dp (density-independent pixels)**. Die Umrechnung in physikalische Pixel erfolgt intern via `DPR` (Device-Pixel-Ratio) aus `DrawCtx`. Usercode rechnet nie in physikalischen Pixeln.

```
1 dp = 1 px bei DPR 1.0  (96 dpi, Standard-Desktop)
1 dp = 2 px bei DPR 2.0  (Retina, HiDPI)
1 dp = 3 px bei DPR 3.0  (Mobile, bestimmte Industrie-Displays)
```

Ursprung ist die **obere linke Ecke** des Widget-Bounds. Alle Koordinaten sind relativ zum aktuellen Transform-Stack. Negative Koordinaten sind erlaubt (zeichnen außerhalb des Bounds, werden durch den Clip-Stack beschnitten).

#### 6.2.2 Das vollständige Interface

```go
type Canvas interface {

    // ── Primitive ────────────────────────────────────────────────────

    // Gefülltes Rechteck.
    FillRect(r Rect, paint Paint)

    // Gefülltes Rechteck mit einheitlichem Radius auf allen Ecken.
    FillRoundRect(r Rect, radius float32, paint Paint)

    // Gefülltes Rechteck mit per-Ecke-Radius.
    FillRoundRectCorners(r Rect, radii CornerRadii, paint Paint)

    // Gefüllte Ellipse (Kreis wenn w == h).
    FillEllipse(r Rect, paint Paint)

    // Stroke-Varianten aller Primitives.
    StrokeRect(r Rect, stroke Stroke)
    StrokeRoundRect(r Rect, radius float32, stroke Stroke)
    StrokeRoundRectCorners(r Rect, radii CornerRadii, stroke Stroke)
    StrokeEllipse(r Rect, stroke Stroke)
    StrokeLine(a, b Point, stroke Stroke)

    // ── Pfade ────────────────────────────────────────────────────────

    // Pfad füllen. Fill-Rule ist Teil von Path.
    FillPath(p Path, paint Paint)

    // Pfad stroken.
    StrokePath(p Path, stroke Stroke)

    // ── Text ─────────────────────────────────────────────────────────

    // Einfaches Text-Rendering: ein String, ein Style, eine Farbe.
    // Für reichhaltigeres Layout: DrawTextLayout.
    DrawText(text string, origin Point, style TextStyle, color Color)

    // Vorberechnetes TextLayout rendern (z.B. für Wrapping, Selektion).
    DrawTextLayout(layout TextLayout, origin Point)

    // Metriken — keine Allocation wenn layout gecacht wird.
    MeasureText(text string, style TextStyle) TextMetrics
    NewTextLayout(text string, style TextStyle, maxWidth float32) TextLayout

    // ── Bilder & Texturen ────────────────────────────────────────────

    // Bild rendern, auf dst skaliert.
    DrawImage(img ImageID, dst Rect, opts ImageOptions)

    // 9-Slice-Skalierung für Buttons, Panels etc.
    DrawImageSlice(img ImageID, dst Rect, insets Insets, opts ImageOptions)

    // Rohe GPU-Textur rendern (für Surface-Slots, §8).
    DrawTexture(tex TextureID, dst Rect, opts ImageOptions)

    // ── Schatten ─────────────────────────────────────────────────────

    // Schatten unter einem Rechteck (Elevation-basiert, §5.2).
    // Muss VOR dem Fill-Call aufgerufen werden (Back-to-Front).
    DrawShadow(r Rect, shadow Shadow)

    // ── Gradienten ───────────────────────────────────────────────────
    // Gradienten sind Paint-Werte, nicht eigene Draw-Calls.
    // Siehe Paint-Typ §6.2.3.

    // ── Clipping & Transform ─────────────────────────────────────────

    // Rechteck-Clip (intersect mit aktuellem Clip).
    PushClip(r Rect)
    // Runder Clip.
    PushClipRoundRect(r Rect, radii CornerRadii)
    // Pfad-Clip (teurer — nur wenn nötig).
    PushClipPath(p Path)
    PopClip()

    // Affine 2D-Transformation (Translation, Scale, Rotation, Shear).
    PushTransform(t Transform)
    PopTransform()

    // Convenience-Shortcuts:
    PushOffset(dx, dy float32)   // = PushTransform(Transform.Translate(dx, dy))
    PushScale(sx, sy float32)    // = PushTransform(Transform.Scale(sx, sy))

    // ── Effekte ──────────────────────────────────────────────────────

    // Gaussian Blur auf alles was innerhalb dieses Scopes gezeichnet wird.
    // Radius in dp.
    PushBlur(radius float32)
    PopBlur()

    // Opacity für einen Subtree (0.0–1.0).
    // Wichtig: PushOpacity/PopOpacity ist nötig für korrekte Compositing-Semantik.
    // Ein einfaches alpha in Color reicht nicht für Widget-Subtrees.
    PushOpacity(alpha float32)
    PopOpacity()

    // ── Layer / Offscreen ────────────────────────────────────────────

    // Zeichnet in einen Offscreen-Buffer, der danach mit opts composited wird.
    // Für komplexe Blend-Modes, Filter-Effekte, Cache-Hints.
    PushLayer(opts LayerOptions)
    PopLayer()

    // ── Zustand ──────────────────────────────────────────────────────

    // Liefert die aktuellen Bounds (relativ zum aktuellen Transform).
    Bounds() Rect

    // Device-Pixel-Ratio — nötig wenn physikalische Pixel relevant sind
    // (z.B. 1px-Border die scharf bleiben soll).
    DPR() float32

    // Snapshot/Restore für komplexe Zustandsverwaltung.
    Save()
    Restore()
}
```

#### 6.2.3 Hilfstypen

```go
// Paint beschreibt wie eine Fläche gefüllt wird.
// Tagged union — immer genau eine Variante aktiv.
//
// Erweiterbarkeit: Paint ist als versionierter Union-Typ angelegt.
// Neue Varianten können in späteren Versionen hinzukommen ohne Breaking
// Change — bestehender Code nutzt ausschließlich die Convenience-
// Konstruktoren und ist damit isoliert gegenüber Union-Erweiterungen.
type Paint struct {
    Kind PaintKind

    // PaintSolid
    Color Color

    // PaintLinearGradient
    Linear LinearGradient

    // PaintRadialGradient
    Radial RadialGradient

    // PaintPattern: Tile einer ImageID
    Pattern PatternPaint

    // PaintShader (v2): Output eines wgpu Compute-Shaders als Paint-Quelle.
    // Ermöglicht GPU-prozedurales Rendering (Noise, SDF-Felder, etc.)
    // direkt als Fill — ohne Umweg über eine Surface.
    // Shader ShaderPaint

    // PaintSurface (v2): Textur eines Surface-Slots (§8) als Paint-Quelle.
    // Beispiel: Button-Hintergrund zeigt einen Live-Video-Feed.
    // Surface SurfacePaint
}

type PaintKind uint8
const (
    PaintSolid PaintKind = iota
    PaintLinearGradient
    PaintRadialGradient
    PaintPattern
    // PaintShader  — reserviert, v2
    // PaintSurface — reserviert, v2
)

// Convenience-Konstruktoren — kein struct-Literal-Rauschen im Usercode.
// Usercode soll ausschließlich diese nutzen, nie Paint direkt konstruieren —
// das hält ihn stabil gegenüber zukünftigen Union-Erweiterungen.
func SolidPaint(c Color) Paint
func LinearGradientPaint(stops []GradientStop, from, to Point) Paint
func RadialGradientPaint(stops []GradientStop, center Point, radius float32) Paint

type GradientStop struct {
    Pos   float32  // 0.0–1.0
    Color Color
}

// Stroke beschreibt eine Kontur.
type Stroke struct {
    Paint     Paint
    Width     float32     // dp
    Cap       StrokeCap   // Butt, Round, Square
    Join      StrokeJoin  // Miter, Round, Bevel
    MiterLimit float32
    Dash      []float32   // Länge der Dash/Gap-Segmente; nil = solid
    DashOffset float32
}

// Shadow für Elevation-Effekte.
type Shadow struct {
    Color   Color
    BlurRadius  float32  // dp
    SpreadRadius float32 // dp, negativ = inset
    OffsetX float32      // dp
    OffsetY float32      // dp
}

// CornerRadii für per-Ecke-Radien.
type CornerRadii struct {
    TopLeft     float32
    TopRight    float32
    BottomRight float32
    BottomLeft  float32
}

func UniformRadii(r float32) CornerRadii  // Alle gleich

// Transform: 2D affine Matrix (3×3, letzter Row implizit [0 0 1]).
type Transform [6]float32  // [a, b, c, d, tx, ty]

func (Transform) Translate(dx, dy float32) Transform
func (Transform) Scale(sx, sy float32) Transform
func (Transform) Rotate(radians float32) Transform
func (Transform) Concat(other Transform) Transform

// TextMetrics vom Renderer.
type TextMetrics struct {
    Width    float32
    Ascent   float32
    Descent  float32
    Leading  float32
}

// LayerOptions für PushLayer.
type LayerOptions struct {
    BlendMode BlendMode  // Normal, Multiply, Screen, Overlay, ...
    Opacity   float32

    // CacheHint ist ein Versprechen des Widget-Autors an das Framework:
    // "Der Inhalt dieses Layers ändert sich ausschließlich wenn
    //  DirtyTracker.IsDirty() == true für das zugehörige Widget."
    //
    // Bei CacheHint == true darf das Framework den aufgezeichneten
    // GPU-Command-Buffer zwischen Frames wiederverwenden, ohne die
    // DrawFunc erneut aufzurufen.
    //
    // Implementiert das Widget DirtyTracker NICHT, wird CacheHint
    // ignoriert — konservatives Fallback: immer neu aufzeichnen.
    //
    // Falsch gesetztes CacheHint (Widget markiert sich nicht dirty
    // obwohl sich der Inhalt geändert hat) führt zu sichtbaren
    // Rendering-Artefakten. In Debug-Builds warnt das Framework wenn
    // ein gecachter Layer einen geänderten WidgetState-Pointer hat.
    CacheHint bool
}
```

#### 6.2.4 Path-Builder

```go
type Path struct {
    FillRule FillRule  // NonZero, EvenOdd
    // Intern: []pathCmd — nie direkt manipulieren
}

// PathBuilder ist ein Builder-Pattern.
// Gibt einen unveränderlichen Path zurück.
type PathBuilder struct{ /* ... */ }

func NewPath() *PathBuilder

func (b *PathBuilder) MoveTo(p Point) *PathBuilder
func (b *PathBuilder) LineTo(p Point) *PathBuilder
func (b *PathBuilder) QuadTo(ctrl, end Point) *PathBuilder   // Quadratische Bézierkurve
func (b *PathBuilder) CubicTo(c1, c2, end Point) *PathBuilder // Kubische Bézierkurve
func (b *PathBuilder) ArcTo(r Size, xRot float32, large, sweep bool, end Point) *PathBuilder
func (b *PathBuilder) Close() *PathBuilder
func (b *PathBuilder) Build() Path

// Convenience:
func PathFromRect(r Rect) Path
func PathFromRoundRect(r Rect, radii CornerRadii) Path
func PathFromEllipse(r Rect) Path
```

#### 6.2.5 Invarianten & Vertrag

Der `Canvas` ist nur innerhalb einer `DrawFunc` (§5.3) oder einer `Widget.Render`-Implementierung gültig. Das Framework garantiert:

- **Kein Zugriff außerhalb des App-Loops**: Canvas-Methoden sind nicht thread-safe und dürfen nicht aus Goroutinen aufgerufen werden. Das ist strukturell erzwungen — `DrawCtx` (der `Canvas` enthält) verlässt den App-Loop nie.
- **Stack-Balance**: Jedes `Push*` muss ein korrespondierendes `Pop*` haben. Bei Imbalance: Panic in Debug-Builds, Auto-Restore in Release-Builds.
- **Paint-Lifetime**: `Paint`-Werte sind Copy — keine Pointer, keine GC-Pressure in Hot-Paths.
- **Path-Immutabilität**: `Path` nach `Build()` ist unveränderlich und kann gecacht werden.

#### 6.2.6 Was Canvas bewusst nicht bietet

| Feature | Begründung |
|---------|-----------|
| Mesh/3D-Rendering | → Surface-Slot (§8), nicht Canvas |
| Video-Decode | → Surface-Slot (§8) |
| Direkte wgpu-Kommandos | → `PushLayer` mit Custom-RenderPass als Extension-Punkt (v2) |
| SVG-Rendering | → Drittanbieter-Widget via `FillPath`/`StrokePath` |
| Animationen | → Animations-System (§12), Canvas hat kein Zeitkonzept |

### 6.3 SDF-Text

Text wird via Signed-Distance-Field-Rendering gezeichnet:

- Glyphen werden einmalig in eine SDF-Textur gerastert (Font-Atlas).
- Rendering ist skalierungs- und rotationsinvariant ohne Qualitätsverlust.
- Multi-channel SDF (MSDF) für scharfe Kurven bei kleinen Größen.

Font-Loading und Glyph-Rasterisierung laufen vollständig in pure Go — via `golang.org/x/image/font/sfnt` für Outline-Extraktion und einem eigenen MSDF-Rasterizer. Kein CGo, kein OS-Font-Subsystem. Details: §16.

### 6.4 VTree-Diff: Kein Reflection, kein Overhead

**Wichtige Klarstellung:** Es gibt kein Reflection-basiertes State-Diffing auf dem User-Model. Der Diff passiert ausschließlich auf dem VTree — einem leichtgewichtigen, strukturierten Baum aus reinen Daten-Structs.

#### Was ist der VTree?

`view(model)` gibt einen `Element`-Baum zurück. Jedes `Element` ist ein einfaches Go-Struct mit bekannten Feldern (keine Interfaces, keine `any`-Felder im heißen Pfad):

```go
// Intern — nie direkt vom User instantiiert
type VNode struct {
    Kind     NodeKind    // Box, Text, Image, Widget, Surface
    Key      UID
    Props    Props       // typed union, kein map[string]any
    Children []VNode
}
```

#### Der Diff-Algorithmus

```
OldTree                NewTree
   │                      │
   └── reconcile() ───────┘
           │
           ▼
    PatchList: []Patch
    ┌──────────────────────┐
    │ UpdateProps(uid, ...) │  → nur geänderte Props
    │ InsertNode(uid, ...)  │  → neues Widget
    │ RemoveNode(uid)       │  → entferntes Widget
    │ MoveNode(uid, idx)    │  → Reihenfolge geändert
    └──────────────────────┘
```

`reconcile` ist ein einfacher O(n)-Tree-Walk mit Key-basiertem Matching — kein Reflection, kein `reflect.DeepEqual`.

#### Widget-Equality: Das `Equatable`-Interface

VNode-Vergleiche brauchen eine Antwort auf die Frage: "Hat sich dieses Widget geändert?" Das User-Model wird nie direkt verglichen — aber die Widget-Structs die `view` zurückgibt, enthalten Felder die aus dem Model stammen. Diese Felder können Typen enthalten die in Go nicht `comparable` sind: Slices, Maps, Funktionen.

```go
// Equatable ist ein optionales Interface auf Widget.
// Implementiert es ein Widget, nutzt reconcile Equal() für den Vergleich.
// Implementiert es ein Widget nicht, gilt der sichere Default: immer re-rendern.
type Equatable interface {
    Widget
    // Equal gibt true zurück wenn dieses Widget und other identischen
    // Render-Output produzieren würden.
    // other ist garantiert der gleiche konkrete Typ — kein Type-Assert nötig.
    Equal(other Widget) bool
}
```

**Drei Kategorien:**

| Widget-Typ | Strategie | Begründung |
|---|---|---|
| Nur `comparable` Felder | `Equatable` via generiertes `Equal()` | Compiler-generierbar, zero Overhead |
| Enthält Slice/Map | `Equatable` mit manuellem `Equal()` | Entwickler entscheidet Semantik |
| Implementiert `Equatable` nicht | Immer re-rendern | Sicher, möglicherweise suboptimal |

```go
// Beispiel: Button mit comparable Feldern — Equal() ist trivial
type Button struct {
    Label    string
    Disabled bool
    Icon     ImageID
}
func (b Button) Equal(other Widget) bool {
    o := other.(Button)
    return b.Label == o.Label && b.Disabled == o.Disabled && b.Icon == o.Icon
}

// Beispiel: Widget mit Slice — Equal() entscheidet Semantik
type TagList struct {
    Tags []string  // nicht comparable
}
func (t TagList) Equal(other Widget) bool {
    o := other.(TagList)
    if len(t.Tags) != len(o.Tags) { return false }
    for i := range t.Tags {
        if t.Tags[i] != o.Tags[i] { return false }
    }
    return true
}

// Beispiel: Widget ohne Equatable → immer re-rendern (safe default)
type VideoPlayer struct {
    Source  string
    OnFrame func(frame []byte)  // Funktion: nie comparable
}
// Kein Equal() → reconcile rendert VideoPlayer immer neu
// Das ist korrekt: Funktionswerte haben keine definierte Gleichheit
```

**Wichtig:** "Re-rendern" bedeutet `Widget.Render()` aufrufen und das Ergebnis mit dem vorherigen VNode vergleichen — nicht neu zeichnen. Ein Widget das `Equatable` nicht implementiert zahlt den Preis eines `Render()`-Aufrufs pro Frame, nicht eines Paint-Passes.

#### Warum kein Reflection?

| | Reflection (`reflect.DeepEqual`) | VTree-Diff |
|---|---|---|
| Allocations | Pro Vergleich | Nur für tatsächliche Patches |
| Typsicherheit | Keine | Vollständig |
| Debuggbarkeit | Schlecht (opake Diffs) | PatchList ist inspizierbar |
| Geschwindigkeit | O(n) mit hohem Constant | O(n) mit minimalem Constant |
| Funktioniert mit | Jedem Typ | Nur mit `Element`-Typen (gewollt) |

Das User-Model ist niemals Gegenstand eines Diffs. `update` gibt ein neues Model zurück — der Framework vergleicht nur ob `view()` einen anderen Baum liefert, nicht warum.

#### Dirty-Tracking via WidgetState

Zusätzlich zum strukturellen Diff gibt es ein optionales Dirty-Flag auf `WidgetState`:

```go
type DirtyTracker interface {
    WidgetState
    // IsDirty gibt true zurück wenn das Widget neu gezeichnet werden muss,
    // auch wenn sich seine Props nicht geändert haben (z.B. Animation).
    IsDirty() bool
    ClearDirty()
}
```

Widgets, die `DirtyTracker` implementieren (z.B. Animationen, Video-Surfaces), können sich selbst für den nächsten Frame markieren ohne dass sich ihr VNode geändert hat.

#### Performance bei langen Listen: Virtualisierung

Ein O(n)-Diff auf 10.000 Listeneinträgen ist unnötig wenn nur 20 davon sichtbar sind. Das Framework bietet dafür `VirtualList` — ein eingebautes Widget das den VTree auf den sichtbaren Viewport beschränkt:

```go
// VirtualList rendert nur die aktuell sichtbaren Items.
// Der VTree enthält maximal (visibleCount + overscan) VNodes — unabhängig
// von der Gesamtlänge der Liste.
type VirtualList struct {
    // Dataset: Datenquelle. Länge kann bekannt oder unbekannt sein.
    // Für einfache Fälle: &SliceDataset[int]{Items: ids} (§19.3)
    // Für paginierte DBs: *PagedDataset[ID] (§19.3)
    // Für Streams: *StreamDataset[ID] (§19.3)
    Dataset Dataset[int]

    // ItemHeight: Einheitliche Item-Höhe in dp.
    // Für variable Höhen: ItemHeightFunc.
    ItemHeight float32

    // ItemHeightFunc: optionale Funktion für variable Höhen.
    // Wenn gesetzt, wird ItemHeight ignoriert.
    // Achtung: wird für alle Items im Viewport aufgerufen — sollte O(1) sein.
    ItemHeightFunc func(index int) float32

    // BuildItem: erzeugt das Widget für Index i.
    // loaded=false → Item noch nicht verfügbar, Skeleton anzeigen.
    // Wird nur für sichtbare Items aufgerufen.
    BuildItem func(index int, loaded bool) Widget

    // Overscan: Anzahl Items die über den sichtbaren Bereich hinaus
    // gerendert werden (oben und unten). Reduziert Flackern beim Scrollen.
    // Default: 3.
    Overscan int

    // ScrollState: KineticScroll (§14) für dieses Widget.
    // Lebt in WidgetState — hier nur die initiale Konfiguration.
    InitialOffset float32
}
```

**Diff-Komplexität mit VirtualList:**
- Ohne VirtualList, 10.000 Items: O(10.000) Diff + O(10.000) Layout
- Mit VirtualList, 10.000 Items, 20 sichtbar: O(20 + overscan) Diff + O(20) Layout

Der Schlüssel: `BuildItem` wird vom Framework nur für sichtbare Indizes aufgerufen. Der VTree enthält nie mehr Nodes als der Viewport fassen kann. `key` (§4.4) sorgt dafür dass beim Scrollen wiederverwendbare Nodes korrekt gemappt werden.

**Kopplung mit `LayerOptions.CacheHint` (§6.2.3):** `CacheHint` ist nur sinnvoll wenn das Widget `DirtyTracker` implementiert. Die Regel ist einfach:

| Widget implementiert | CacheHint Effekt |
|---|---|
| `DirtyTracker` | Layer-Command-Buffer wird gecacht; neu aufgezeichnet wenn `IsDirty() == true` |
| kein `DirtyTracker` | `CacheHint` wird ignoriert; immer neu aufzeichnen (sicheres Fallback) |

Ein Widget das `CacheHint` setzen will, **muss** `DirtyTracker` implementieren — sonst ist das Hint wirkungslos. Das Framework erzwingt das nicht zur Compile-Zeit (würde die Widget-API verkomplizieren), aber ein Debug-Build-Warning macht den Fehler sofort sichtbar.

### 6.5 Render-Pipeline Stages

```
view() → VTree
    │
    ▼
reconcile(oldVTree, newVTree) → PatchList
    │
    ▼
Layout (Constraint-basiert, top-down pass)  [nur für gepatche Teilbäume]
    │
    ▼
Paint (Widget.DrawFunc → Canvas-Kommandos)  [nur für dirty Nodes]
    │
    ▼
Command-Buffer (wgpu RenderPass)
    │
    ▼
GPU Submit → Swapchain Present
```

Der Layout-Algorithmus ist Yoga-kompatibel (Flexbox), kann aber erweitert werden (eigene Layout-Typen via Interface).

Layout und Paint laufen nur für Teilbäume die tatsächlich gepatcht wurden — kein vollständiges Re-Layout bei jedem Frame.

**Performance-Hinweis für DRM/KMS:** Jeder `PushBlur`- und `PushLayer`-Aufruf erfordert einen zusätzlichen Render-Pass (Offscreen-Buffer → Blit). Auf DRM/KMS-Targets ohne dedizierte GPU kann das teuer sein. `theme.Slate` nutzt bewusst `1px Solid Borders` statt Blur-Schatten — das reduziert die Anzahl der Render-Passes auf ein Minimum. Custom-Themes die Blur intensiv nutzen sollten `LayerOptions.CacheHint = true` setzen (§6.2.3).

---

## 7. Platform-Abstraktion

### 7.1 Das Platform-Interface

```go
type Platform interface {
    // Lifecycle
    Init(cfg PlatformConfig) error
    Run(loop FrameLoop) error   // Blockiert
    Destroy()

    // Window
    SetTitle(title string)
    SetSize(w, h int)
    SetFullscreen(bool)
    RequestFrame()              // Weckt den Loop für nächsten Frame

    // Input
    SetCursor(CursorKind)
    SetClipboard(text string)
    GetClipboard() string

    // GPU Surface
    CreateSurface(instance wgpu.Instance) wgpu.Surface
}
```

### 7.2 Build-Tag-basierte Platform-Auswahl

```
go build -tags wayland  → Wayland-Backend
go build -tags x11      → X11-Backend
go build -tags drm      → DRM/KMS-Backend (kein Display-Server)
go build -tags win32    → Win32-Backend
go build -tags cocoa    → Cocoa/AppKit-Backend
```

Ohne expliziten Tag: automatische Auswahl zur Laufzeit (Wayland > X11 auf Linux).

### 7.3 DRM/KMS als First-Class-Platform

DRM/KMS ist eine vollständige Implementierung von `Platform`, nicht ein Hack:

```go
// +build drm

type DRMPlatform struct {
    fd       int        // /dev/dri/card0
    crtc     uint32
    connector uint32
    // ...
}

func (d *DRMPlatform) CreateSurface(inst wgpu.Instance) wgpu.Surface {
    // wgpu unterstützt DRM nativ über wgpuInstanceCreateSurface
    // mit WGPUSurfaceDescriptorFromWaylandSurface / ...FromDrmFd
    return inst.CreateSurfaceFromDRM(d.fd, d.crtc)
}
```

Input kommt via `libinput` (evdev), ebenfalls hinter dem `Platform`-Interface.

---

## 8. Externe Surfaces

### 8.1 Konzept

Browser-Instanzen, 3D-Renderer, Video-Decoder etc. docken als **Surface-Slot** an einen Platz im Widget-Baum. Aus Sicht des Frameworks ist ein Surface-Slot ein `Element` mit einer GPU-Textur.

```go
type Surface struct {
    ID      SurfaceID
    Bounds  Rect
    ZIndex  int
}

type SurfaceProvider interface {
    // Wird vom Framework aufgerufen wenn eine neue GPU-Textur erwartet wird.
    // Gibt eine wgpu.TextureView zurück.
    AcquireFrame(bounds Rect) (wgpu.TextureView, FrameToken)
    ReleaseFrame(token FrameToken)

    // Input-Weiterleitung (optional)
    HandleMsg(msg Msg) bool  // true = verbraucht
}
```

### 8.2 Zero-Copy-Pfade

| Platform | Mechanismus |
|----------|-------------|
| macOS | IOSurface → wgpu Shared Texture |
| Linux | DMA-buf → wgpu External Memory |
| Windows | DXGI Shared Handle |
| Fallback | OSR (Off-Screen Rendering) → CPU-Copy → Upload |

Der Framework wählt automatisch den besten verfügbaren Pfad.

### 8.3 Input-Routing

Eingaben in einen Surface-Bereich werden via denselben Msg-Channel geroutet — keine Sonderbehandlung:

```go
type SurfaceMouseMsg struct {
    SurfaceID SurfaceID
    Pos       Point
    Button    MouseButton
    Action    MouseAction
}
```

Der `SurfaceProvider` empfängt seine Msgs über `HandleMsg` (synchron im App-Loop) oder über einen eigenen gepufferten Channel (asynchron).

---

## 9. Offene Fragen

Alle ursprünglichen offenen Punkte sind aufgelöst und in die jeweiligen Abschnitte überführt worden:

### 9.1 ~~Accessibility (A11y)~~ → Verschoben nach §11

A11y ist kein offener Punkt mehr, sondern eine vollständige Spezifikation — siehe §11.

### 9.2 ~~Internationalisierung (i18n) & Bidirektionaler Text~~ → Verschoben nach §16

### 9.3 ~~Font-Fallback-Chain~~ → Verschoben nach §16

### 9.4 ~~Scroll-Overscroll & Kinetic Scrolling~~ → Verschoben nach §14

### 9.5 ~~Package-Name~~ → Verschoben nach §16

---

## 10. Nicht-Ziele

Folgende Punkte sind explizit **außerhalb** des Scope:

- **HTML/CSS-Rendering** — kein Webkit, kein Servo als Rendering-Engine des Frameworks selbst.
- **Native Widget-Integration** — kein "wrap a GTK Button". Externe Surfaces (§8) sind der korrekte Pfad für Native-UI-Integration wenn unbedingt nötig.
- **Automatisches State-Diffing via Reflection** — der Diff passiert auf dem VTree, nicht auf dem User-Model.
- **JavaScript-Bridge als primärer Pfad** — WASM/Browser sind eine Platform, nicht das Fundament.
- **CSS-kompatibler Layout-Algorithmus** — Flexbox ja, aber kein vollständiges CSS-Cascade-Model.

---

## 11. Accessibility (A11y) — First-Class-Feature

### 11.1 Warum A11y ein USP ist

Kein Go-UI-Toolkit liefert heute vollständige, plattformkonforme Accessibility:

| Toolkit | A11y-Status |
|---------|-------------|
| Fyne | Rudimentär, nicht screenreader-kompatibel |
| Gio | Keines |
| Wails | Browser-A11y (nur Web-Kontext) |
| **Dieses Toolkit** | **AT-SPI2 / UIA / NSAccessibility, vollständig** |

Das ist insbesondere für Industrie-HMI, öffentliche Verwaltung (EN 301 549) und Enterprise-Kunden relevant — Märkte, die strukturell auf barrierefreie Software angewiesen sind.

### 11.2 Architektureller Vorteil durch eigenes Rendering

Da das Framework kein natives Widget-System nutzt, hat es **vollständige Kontrolle über den semantischen Baum**. Es muss keine A11y-Informationen aus nativen Widgets rückwärts extrahieren — der Accessibility-Tree wird direkt aus dem VTree konstruiert.

```
VTree (§6.4)
    │
    ├──── Render-Pipeline → GPU
    │
    └──── A11y-Pipeline → AccessTree → Plattform-A11y-API
```

Beide Pipelines laufen synchron im App-Loop — der AccessTree ist immer konsistent mit dem gerenderten Frame.

### 11.3 Das AccessibleWidget-Interface

A11y ist ein optionales Interface — eingebaute Widgets implementieren es vollständig, Drittanbieter-Widgets können es implementieren oder weglassen (Framework liefert dann einen generischen Fallback-Node):

```go
// AccessibleWidget ist optional. Widgets die es nicht implementieren,
// erhalten einen generischen AccessNode mit Role=Group.
type AccessibleWidget interface {
    Widget
    Accessibility(state WidgetState) AccessNode
}

type AccessNode struct {
    Role        AccessRole
    Label       string        // Primärer Name (aria-label Äquivalent)
    Description string        // Längere Beschreibung (aria-describedby)
    Value       string        // Aktueller Wert (für Sliders, Inputs etc.)
    Lang        language.Tag  // BCP 47 Sprach-Tag (z.B. "de", "ar-EG")
                              // Screenreader wechselt Stimme/Aussprache pro Node.
                              // Leer = erbt von Parent-Node oder App-Locale.
    States      AccessStates  // Bitfield: Focused, Checked, Disabled, ...
    Actions     []AccessAction
    // Relation zu anderen Nodes (LabelledBy, DescribedBy, Controls, ...)
    Relations   []AccessRelation
}

type AccessRole uint32

const (
    RoleButton AccessRole = iota
    RoleCheckbox
    RoleCombobox
    RoleDialog
    RoleGrid
    RoleHeading
    RoleImage
    RoleLink
    RoleListbox
    RoleMenu
    RoleProgressBar
    RoleScrollBar
    RoleSlider
    RoleSpinButton
    RoleTab
    RoleTable
    RoleTextInput
    RoleToggle
    RoleTree
    // Erweiterbar durch Drittanbieter
    RoleCustomBase AccessRole = 1 << 16
)

type AccessAction struct {
    Name    string
    Trigger func()  // Ausgeführt im App-Loop via Send
}

type AccessStates struct {
    Focused   bool
    Checked   bool
    Selected  bool
    Expanded  bool
    Disabled  bool
    ReadOnly  bool
    Required  bool
    Invalid   bool
    Busy      bool
    Live      AccessLiveRegion
}
```

### 11.4 AccessTree-Konstruktion

Der AccessTree wird nach jedem Reconcile aus dem VTree abgeleitet:

```go
func buildAccessTree(vTree []VNode, registry stateRegistry) AccessTree {
    // Depth-first walk
    // Nodes die AccessibleWidget implementieren → AccessNode via .Accessibility()
    // Nodes die es nicht implementieren → generischer Group-Node
    // Unsichtbare/deaktivierte Nodes → aus AccessTree ausgeschlossen
}
```

Der AccessTree ist eine flache Slice mit Parent-Indizes (cache-freundlich), keine verschachtelte Baumstruktur.

### 11.5 Plattform-Bridges

Jede Platform-Implementierung (§7) implementiert optional die `A11yBridge`:

```go
type A11yBridge interface {
    // Wird aufgerufen wenn sich der AccessTree geändert hat.
    // tree ist unveränderlich — die Bridge darf nur lesen.
    UpdateTree(tree AccessTree)

    // Focus-Änderung
    NotifyFocus(nodeID AccessNodeID)

    // Live-Region-Update (z.B. Statusmeldung)
    NotifyLiveRegion(nodeID AccessNodeID, text string)
}
```

| Platform | Bridge-Implementierung |
|----------|----------------------|
| Linux (Wayland/X11) | AT-SPI2 via D-Bus (CGo oder `godbus`) |
| Windows | UI Automation (UIA) via CGo/COM |
| macOS | NSAccessibility via CGo/ObjC |
| DRM/KMS | AT-SPI2 (gleicher Code wie Linux-Display-Server-Pfad) |
| WASM/Browser | ARIA-Attribute auf Canvas-Overlay |

### 11.6 DRM/KMS: Kein Rückschritt bei A11y

Auf DRM/KMS (kein Display-Server) läuft AT-SPI2 trotzdem — über den System-D-Bus, der auch ohne Display-Server verfügbar ist. Screenreader wie Orca kommunizieren via D-Bus, nicht via Wayland.

Das bedeutet: Industrie-HMI auf Bare Metal kann vollständig barrierefrei sein. Das ist ein Alleinstellungsmerkmal gegenüber jedem nativen Widget-Toolkit, das auf Display-Server-A11y-Integration angewiesen ist.

### 11.7 Focus-Trapping & Modale Dialoge

Korrekte Fokus-Verwaltung bei dynamischen Inhalten ist eine A11y-Pflicht, nicht Komfort:

```go
// FocusTrap fängt Tab-Navigation innerhalb eines Teilbaums ein.
// Wird automatisch aktiviert wenn ein Dialog mit Modal=true geöffnet wird.
type FocusTrap struct {
    // RestoreFocus: Wenn der Trap aufgelöst wird (Dialog schließt),
    // kehrt der Fokus zum Widget zurück, das ihn vorher hatte.
    RestoreFocus bool  // Default: true

    // InitialFocus: UID des Widgets das beim Öffnen des Traps
    // den initialen Fokus erhalten soll.
    // Leer = erstes fokussierbares Widget im Trap.
    InitialFocus UID
}
```

**Regeln:**
- **Modal öffnet** → Fokus wandert in den Dialog (`InitialFocus` oder erstes fokussierbares Widget)
- **Tab am letzten Widget** → Fokus springt zum ersten Widget im Trap (nicht aus dem Dialog heraus)
- **Shift+Tab am ersten Widget** → Fokus springt zum letzten Widget im Trap
- **Escape** → Dialog schließt, Fokus kehrt zum auslösenden Widget zurück (`RestoreFocus`)
- **Screenreader:** Der Inhalt außerhalb des Traps wird als `aria-hidden` markiert (bzw. aus dem AccessTree entfernt)

Nicht-modale Dialoge (z.B. Tooltips, Popovers) verwenden **keinen** FocusTrap — dort bleibt die normale Tab-Navigation aktiv.

### 11.8 Live-Regions & Dynamische Updates

Für dynamische Inhalte (Statusmeldungen, Benachrichtigungen) gibt es `AccessLiveRegion`:

```go
type AccessLiveRegion uint8

const (
    LiveOff      AccessLiveRegion = iota  // Kein Live-Update
    LivePolite                            // Warte auf Idle
    LiveAssertive                         // Sofort unterbrechen
)
```

Widgets markieren ihren Content-Bereich mit `LivePolite` oder `LiveAssertive`. Der Framework sendet `NotifyLiveRegion` an die Bridge wenn sich der `Value` oder `Label` eines solchen Nodes ändert.

### 11.9 Testing

A11y-Korrektheit ist testbar ohne Screenreader:

```go
func TestButtonAccessibility(t *testing.T) {
    tree := renderToAccessTree(view(Model{...}))
    btn := tree.FindByRole(RoleButton)
    assert.Equal(t, "Speichern", btn.Label)
    assert.False(t, btn.States.Disabled)
}
```

`renderToAccessTree` ist eine reine Go-Funktion — kein GUI, kein Display-Server nötig. A11y-Tests laufen in CI wie normale Unit-Tests.

---

| Konzept | Go-Typ | Package |
|---------|--------|---------|
| App-Einstiegspunkt | `func Run[M any](...)` | `app` |
| Nachricht | `type Msg interface{}` | `app` |
| Element | `type Element interface{}` | `ui` |
| Widget | `type Widget interface{}` | `ui` |
| Theme | `type Theme interface{}` | `theme` |
| Canvas | `type Canvas interface{}` | `draw` |
| Platform | `type Platform interface{}` | `platform` |

---

## Anhang A: Naming Conventions

| Konzept | Go-Typ | Package |
|---------|--------|---------|
| App-Einstiegspunkt | `func Run[M any](...)` | `lux/app` |
| Nachricht | `type Msg interface{}` | `lux/app` |
| Element | `type Element interface{}` | `lux/ui` |
| Widget | `type Widget interface{}` | `lux/ui` |
| WidgetState | `type WidgetState interface{}` | `lux/ui` |
| Theme | `type Theme interface{}` | `lux/theme` |
| TokenSet | `type TokenSet struct{...}` | `lux/theme` |
| Canvas | `type Canvas interface{}` | `lux/draw` |
| Paint | `type Paint struct{...}` | `lux/draw` |
| Path / PathBuilder | `type Path struct{...}` | `lux/draw` |
| KeyMsg, MouseMsg, ... | Input-Msg-Typen | `lux/input` |
| Shortcut | `type Shortcut struct{...}` | `lux/input` |
| Anim[T], SpringAnim[T] | Animations-Typen | `lux/anim` |
| AnimationID | `type AnimationID string` | `lux/anim` |
| KineticScroll | `type KineticScroll struct{...}` | `lux/ui` |
| Constraints, Flex, Stack | Layout-Typen | `lux/layout` |
| Font, FontFamily | Font-Typen | `lux/fonts` |
| AccessNode, AccessRole | A11y-Typen | `lux/a11y` |
| Platform | `type Platform interface{}` | `lux/platform` |

**Konventionen:**
- Alle öffentlichen Interfaces sind marker-frei (keine Pflicht-Methoden ohne Semantik)
- Convenience-Konstruktoren folgen dem Muster `NewX(...)` oder `XFromY(...)`
- Build-Tags: `drm`, `shaping` (reserviert, aktuell nicht genutzt), `systemfonts`
- Interne Packages: `lux/internal/...` — nie direkt importierbar

---

## Anhang B: Minimales Beispielprogramm

```go
package main

import (
    "github.com/timzifer/lux/app"
    "github.com/timzifer/lux/ui"
    "github.com/timzifer/lux/theme"
)

type Model struct {
    Count int
}

type IncrMsg struct{}
type DecrMsg struct{}

func update(m Model, msg app.Msg) Model {
    switch msg.(type) {
    case IncrMsg:
        m.Count++
    case DecrMsg:
        m.Count--
    }
    return m
}

func view(m Model) ui.Element {
    return ui.Column(
        ui.Text(fmt.Sprintf("Count: %d", m.Count)),
        ui.Row(
            ui.Button("−", func() { app.Send(DecrMsg{}) }),
            ui.Button("+", func() { app.Send(IncrMsg{}) }),
        ),
    )
}

func main() {
    app.Run(Model{Count: 0}, update, view,
        app.WithTheme(theme.Default),
        app.WithTitle("Counter"),
    )
}
```

Das gesamte Programm benötigt keine Kenntnis von Goroutinen, Locks, Channels oder GPU-Details.

---

## 12. Ausblick: Inspector & Debugging-Tools

Dieser Abschnitt beschreibt geplante Werkzeuge die nicht zum v1.0-Kern gehören, aber früh als Erweiterungspunkte vorgesehen sein müssen damit sie nachträglich sauber integrierbar sind.

### 17.1 Widget Inspector

Ein visuelles Debugging-Tool analog zu Browser DevTools oder Flutter Inspector:

```
Geplante Features:
  ✦ Widget-Tree-Ansicht: Alle VNodes mit Props, UID, WidgetState-Typ
  ✦ Layout-Overlay: Bounds, Margins, Padding als visuelle Einblendung
  ✦ Paint-Highlighting: Welche Widgets wurden in diesem Frame neu gezeichnet?
  ✦ Event-Log: Input-Events und ihre Dispatch-Ziele in Echtzeit
  ✦ State-Dump: WidgetState als JSON für jeden Node
  ✦ Performance: Frame-Zeit, Layout-Zeit, Paint-Zeit pro Frame
```

**Architektureller Erweiterungspunkt:** Der Inspector dockt als separater Prozess via einem Debug-Protocol an — nicht als In-Process-Overlay. Das Framework öffnet optional einen lokalen TCP/Unix-Socket der den aktuellen VTree und Frame-Metriken als Stream ausgibt. Der Inspector ist ein eigenes Go-Binary das diesen Stream konsumiert.

```go
// In der App aktivieren (nur Debug-Builds empfohlen):
app.Run(model, update, view,
    app.WithInspector(app.InspectorConfig{
        Addr: "localhost:9876",  // leer = zufälliger Port
    }),
)
```

Damit ist der Inspector:
- Vollständig optional — kein Overhead in Production
- Plattformunabhängig — der Inspector läuft als natives Tool, nicht im App-Prozess
- Erweiterbar — Third-Party-Inspectoren können dasselbe Protocol sprechen

### 17.2 Hot-Reload

`view` ist eine reine Funktion ohne Seiteneffekte. Das macht partielles Hot-Reload strukturell einfacher als in anderen Frameworks:

- Model bleibt erhalten (State Restoration, §3.4)
- Neues `view`-Binary wird geladen (Go Plugin oder Rebuild-Trigger)
- Nächster Frame nutzt die neue `view`-Implementierung

Vollständiges Hot-Reload (inklusive `update`) erfordert Modell-Migrations-Logik — das ist ein separates Problem und nicht Teil von v1.0.

---

## 13. Implementierungs-Leitfaden

### 13.1 Dependency-Graph der internen Packages

Kein internes Package darf außerhalb seiner erlaubten Imports liegen — Zyklen in der ersten Woche sind das häufigste Produktivitätskiller bei größeren Go-Projekten.

```
github.com/timzifer/lux/
│
├── app/            ← Einstiegspunkt (Run, Send, Option)
│   depends on:     internal/loop, internal/reconcile, ui, theme, input, anim
│
├── ui/             ← Widgets, Element, WidgetState, Layout-Typen
│   depends on:     draw, theme, input, anim, fonts
│   DARF NICHT:     app (kein Rückwärts-Import)
│
├── draw/           ← Canvas, Paint, Path, Color
│   depends on:     internal/wgpu
│   DARF NICHT:     ui, app, theme
│
├── theme/          ← Theme, TokenSet, Default-Theme
│   depends on:     draw, fonts
│   DARF NICHT:     ui, app
│
├── input/          ← KeyMsg, MouseMsg, alle Input-Typen
│   depends on:     (nur stdlib)
│   DARF NICHT:     ui, app, draw
│
├── anim/           ← Anim[T], SpringAnim, AnimGroup, AnimSeq
│   depends on:     (nur stdlib)
│   DARF NICHT:     ui, app, draw
│
├── fonts/          ← Font, FontFamily, Fallback
│   depends on:     go-text/typesetting, golang.org/x/image
│   DARF NICHT:     ui, app, draw
│
├── layout/         ← Constraints, Flex, Stack, Grid
│   depends on:     ui (Widget-Interface), draw (Rect, Size, Point)
│   DARF NICHT:     app, theme
│
├── a11y/           ← AccessNode, AccessRole (öffentlich für Tests)
│   depends on:     (nur stdlib)
│   DARF NICHT:     ui, app, draw
│
├── platform/       ← Platform-Interface (für Drittanbieter)
│   depends on:     internal/wgpu, input
│   DARF NICHT:     ui, app, theme
│
└── internal/
    ├── loop/       ← App-Loop, Frame-Tick, dt-Clamping
    ├── reconcile/  ← VTree-Diff, PatchList
    ├── wgpu/       ← Shim (native/ und gogpu/)
    ├── render/     ← 2D-Renderer, SDF-Atlas, Paint-Execution
    ├── focus/      ← Focus-Manager, Tab-Reihenfolge
    ├── hit/        ← Hit-Testing gegen Layout-Baum
    ├── overlay/    ← Overlay-Layer, Z-Order
    ├── a11ytree/   ← AccessTree-Konstruktion, Platform-Bridges
    └── text/       ← TextLayout-Cache, Shaping-Pipeline
```

**Die eiserne Regel:** `internal/` importiert nie öffentliche `lux/`-Packages (außer reine Datentypen aus `draw`, `input`, `anim`). Öffentliche Packages importieren nie `internal/` direkt — nur `app/` darf das als Orchestrator.

### 13.2 Implementierungs-Reihenfolge

In dieser Reihenfolge bauen — jede Stufe ist ohne die nächste testbar:

#### Stufe 1 — Fundament (Woche 1–2)
Ziel: ein leeres Fenster das sich öffnet und schließt.

```
internal/wgpu/native  → wgpu-native Shim, CreateInstance, CreateSurface
platform/             → eine Platform (Empfehlung: Win32 oder Wayland)
internal/loop/        → minimaler Frame-Loop, dt-Clamping, msgChannel
app/                  → Run() mit leerem view, Send()
```

**Testbar wenn:** `app.Run(struct{}{}, update, view)` öffnet ein schwarzes Fenster.

#### Stufe 2 — Rendering-Kern (Woche 3–4)
Ziel: einfache Shapes auf dem Fenster.

```
internal/render/      → wgpu RenderPass, Rect/RoundRect via SDF
draw/                 → Canvas-Interface, SolidPaint, FillRect
fonts/                → Font-Loading, Fallback-Font einbetten
internal/text/        → go-text/typesetting anbinden, DrawText
```

**Testbar wenn:** `canvas.FillRect(...)` und `canvas.DrawText(...)` funktionieren.

#### Stufe 3 — Widget-System (Woche 5–6)
Ziel: erste echte Widgets.

```
ui/                   → Widget, WidgetState, Element-Typen, adoptState
internal/reconcile/   → VTree-Diff, PatchList
layout/               → Constraints, Flex (Row/Column), Padding, SizedBox
internal/focus/       → Focus-Manager
internal/hit/         → Hit-Testing
input/                → KeyMsg, MouseMsg, alle Msg-Typen
```

**Testbar wenn:** `ui.Button`, `ui.Text`, `ui.Column` rendern und auf Clicks reagieren.

#### Stufe 4 — Theming & Animation (Woche 7–8)
Ziel: das Framework fühlt sich "fertig" an.

```
theme/                → Theme-Interface, TokenSet, Default-Theme, Lookup-Cache
anim/                 → Anim[T], SpringAnim, Animator-Interface
internal/loop/        → AnimationTick-Pass, DirtyTracker
internal/overlay/     → Overlay-Layer, Dismiss-Logik
```

**Testbar wenn:** Theme-Wechsel via `SetThemeMsg`, Button-Hover animiert sich.

#### Stufe 5 — Vollständigkeit (Woche 9–12)
Ziel: produktionsfähig.

```
a11y/                 → AccessTree, AT-SPI2-Bridge (Linux zuerst)
internal/a11ytree/    → AccessTree-Konstruktion
VirtualList           → Dataset-Interface, PagedDataset, StreamDataset
Tree, Menu            → BuildNode-Pattern, Overlay-Integration
RichText              → TextSpan, Paragraph, go-text/typesetting Pipeline
platform/drm          → DRM/KMS als zweite Platform
internal/wgpu/gogpu   → gogpu/wgpu Shim (-tags gogpu)
```

### 13.3 Testbarkeit der Kern-Invarianten

Diese Invarianten sind nicht trivial testbar — hier sind konkrete Patterns:

#### Single-Threaded App-Loop

```go
// internal/loop/loop_test.go
func TestUpdateRunsOnlyInLoop(t *testing.T) {
    var updateGoroutineID int64
    var externalGoroutineID int64

    update := func(m Model, msg app.Msg) Model {
        updateGoroutineID = goroutineID()  // via runtime stack parse
        return m
    }

    // update muss immer auf derselben Goroutine laufen wie Run()
    go app.Run(Model{}, update, view)
    time.Sleep(10 * time.Millisecond)

    app.Send(TestMsg{})
    time.Sleep(10 * time.Millisecond)

    externalGoroutineID = goroutineID()
    assert.NotEqual(t, updateGoroutineID, externalGoroutineID)
}
```

#### Canvas verlässt niemals den Loop

```go
// draw/canvas_test.go
func TestCanvasNotAccessibleOutsideLoop(t *testing.T) {
    var leaked draw.Canvas

    drawFunc := func(ctx draw.DrawCtx, _ theme.TokenSet, _ ui.WidgetState) {
        leaked = ctx.Canvas  // Canvas "stehlen"
    }

    // Nach dem Frame: Canvas-Methoden müssen paniken
    runOneFrame(drawFunc)

    assert.Panics(t, func() {
        leaked.FillRect(draw.Rect{}, draw.SolidPaint(draw.Black))
    })
}
```

**Implementierung:** `Canvas` hat intern ein `valid bool` das am Ende jedes `DrawFunc`-Aufrufs auf `false` gesetzt wird. Alle Methoden prüfen `valid` und paniken wenn `false`.

#### dt-Clamping

```go
// internal/loop/dt_test.go
func TestDtClamping(t *testing.T) {
    loop := newTestLoop(WithMaxFrameDelta(100 * time.Millisecond))

    // Simuliere einen 5-Sekunden-Freeze
    ticks := loop.TickWithDelta(5 * time.Second)

    for _, dt := range ticks {
        assert.LessOrEqual(t, dt, 100*time.Millisecond)
    }
}
```

#### update bekommt kein dt

```go
// Kompilier-Test: UpdateFunc[M] hat Signatur func(M, Msg) M
// Kein dt-Parameter möglich — der Compiler erzwingt das.
// Kein Laufzeit-Test nötig.
var _ app.UpdateFunc[Model] = update  // kompiliert nur wenn Signatur stimmt
```

### 13.4 Empfohlene Entwicklungs-Milestones

| Milestone | Inhalt | Kriterium |
|-----------|--------|-----------|
| **M1** Fenster | Stufe 1 | Schwarzes Fenster öffnet und schließt |
| **M2** Hello World | Stufe 2 | Text + Button rendern |
| **M3** Counter | Stufe 3 | Anhang-B-Beispiel läuft vollständig |
| **M4** Themed | Stufe 4 | Dark/Light-Switch, Hover-Animation |
| **M5** Alpha | Stufe 5 ohne A11y | VirtualList, Tree, RichText |
| **M6** Beta | Stufe 5 komplett | A11y, DRM/KMS, gogpu-Tag |
| **v1.0** | Alle §§ | Stable API, alle Plattformen grün |

### 13.5 Was bewusst nicht in v1.0 ist

Folgendes ist spezifiziert aber explizit für nach v1.0 vorgesehen:

```
- RichTextEditor (§20.6)     — separates Paket lux/richtext, post-v1.0
- Inspector / Hot-Reload (§17) — post-v1.0
- gogpu/wgpu als Default     — sobald Produktionsreife erreicht
- PaintShader / PaintSurface (§6.2) — v2
- -tags harfbuzz             — post-v1.0 wenn Bedarf besteht
- WASM/Browser als Platform  — post-v1.0
```

Diese Grenze explizit zu ziehen verhindert Scope-Creep in der ersten Implementierungsphase.

---

---

> **Browser-Engine-Integration:** Siehe [RFC-003 — lux/surface/webview](RFC-003-lux-webview.md) für die vollständige Evaluierung und Spezifikation der Browser-Engine-Integration via Surface-Slots.

---

*RFC-001 — Draft. Feedback und Änderungsvorschläge bitte als Issue gegen dieses Dokument.*

---

*RFC-001 — Draft. Feedback via GitHub Issues gegen `github.com/timzifer/lux`.*