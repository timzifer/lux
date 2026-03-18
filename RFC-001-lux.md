# RFC-001 — lux: Technische Spezifikation

**Repository:** `github.com/timzifer/lux`

**Status:** Active
**Version:** 0.2.0  
**Datum:** 2026-03-17  

---

## Inhaltsverzeichnis

1. [Motivation & Abgrenzung](#1-motivation--abgrenzung)
2. [Architektur-Überblick](#2-architektur-überblick)
3. [Kern: Elm-Architektur & App-Loop](#3-kern-elm-architektur--app-loop)
4. [Widget-System & WidgetState](#4-widget-system--widgetstate)
5. [Theming-System](#5-theming-system)
6. [Rendering-Pipeline](#6-rendering-pipeline)
7. [Platform-Abstraktion](#7-platform-abstraktion)
8. [Externe Surfaces](#8-externe-surfaces)
9. [Offene Fragen](#9-offene-fragen)
10. [Nicht-Ziele](#10-nicht-ziele)
11. [Accessibility (A11y)](#11-accessibility-a11y--first-class-feature)
12. [Animations-System](#12-animations-system)
13. [Input-System](#13-input-system)
14. [Scroll & Kinetic Scrolling](#14-scroll--kinetic-scrolling)
15. [Layout-System](#15-layout-system)
16. [Text-Stack, i18n & Package-Name](#16-text-stack-i18n--package-name)
17. [Ausblick: Inspector & Debugging-Tools](#17-ausblick-inspector--debugging-tools)
18. [Datenbasierte Widgets & Overlay-System](#18-datenbasierte-widgets--overlay-system)
19. [DynamicDataset — Länge unbekannt](#19-dynamicdataset--länge-unbekannt)
20. [Rich Text & Texteditierung](#20-rich-text--texteditierung)
21. [Implementierungs-Leitfaden](#21-implementierungs-leitfaden)

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

type ColorScheme struct {
    Background  Color
    Surface     Color
    Primary     Color
    Secondary   Color
    OnPrimary   Color
    OnSurface   Color
    Error       Color
    Outline     Color
    // Erweiterbar: Tokens sind benannte Slots, nicht enum-limitiert
    Custom      map[string]Color
}

type TypographyScale struct {
    DisplayLarge  TextStyle
    DisplayMedium TextStyle
    HeadlineLarge TextStyle
    BodyMedium    TextStyle
    LabelSmall    TextStyle
    // ... analog Material Design 3 Type Scale
}

type TextStyle struct {
    FontFamily string
    Size       float32   // dp
    Weight     FontWeight
    LineHeight float32   // multiplier
    Tracking   float32   // em
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
        Style:   tokens.Typography.LabelSmall,
        Color:   tokens.Colors.Primary,
    }
    // ...
}
```

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

### 11.7 Live-Regions & Dynamische Updates

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

### 11.8 Testing

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

## 12. Animations-System

### 12.1 Designziele

Das Animations-System muss in die Elm-Architektur passen — strukturell, nicht nur konzeptuell:

- **Kein Goroutinen-Timer**: Animationen laufen nicht in Goroutinen. Kein `time.Sleep`, kein `time.After`, kein Channel-basierter Tick der in `Send` mündet.
- **Deterministisch & testbar**: Eine Animation mit `dt = 16ms` ist überall identisch. Tests brauchen keine echte Uhr.
- **Deklarativ**: Widget deklariert *was* animiert werden soll und *wohin* — das Framework entscheidet *wann* der nächste Frame kommt.
- **Usercode bleibt ruhig**: Der User-Loop sieht keine `FrameMsg` pro Frame. Animationen laufen framework-intern. Optional kann ein Widget am Ende einer Animation eine `Msg` in den User-Loop einspeisen.

### 12.2 Architektur-Überblick

```
Frame-Loop (§3.3)
    │
    ├── 1. Msg-Verarbeitung (update)
    │
    ├── 2. AnimationTick-Pass          ← NEU
    │       Für alle WidgetStates die Animator implementieren:
    │       state.Tick(dt) → bool (noch aktiv?)
    │       Aktive Animatoren → Widget automatisch dirty
    │
    ├── 3. Reconcile (VTree-Diff)
    │
    ├── 4. Layout
    │
    └── 5. Paint (DrawFunc liest Anim[T].Value())
```

Der `AnimationTick-Pass` läuft **vor** dem Reconcile — damit der Paint-Pass im selben Frame bereits den aktuellen interpolierten Wert sieht. Keine Frame-Verzögerung.

### 12.3 Das Animator-Interface

```go
// Animator ist ein optionales Interface auf WidgetState.
// Widgets die es implementieren, werden vom Framework vor jedem
// Paint-Pass getickert — ohne Usercode-Beteiligung.
type Animator interface {
    WidgetState

    // Tick wird einmal pro Frame aufgerufen solange mindestens
    // eine Animation aktiv ist.
    // dt: vergangene Zeit seit dem letzten Frame (wall-clock, geclampt).
    // Gibt true zurück wenn noch mindestens eine Animation läuft
    // (= Widget will weiteren Frame).
    // Gibt false zurück wenn alle Animationen abgeschlossen sind
    // (= Framework stoppt den Tick bis zur nächsten Mutation).
    Tick(dt time.Duration) (stillRunning bool)
}
```

**Kopplung mit `DirtyTracker` (§6.4):** Ein Widget das `Animator` implementiert muss nicht zusätzlich `DirtyTracker` implementieren. Das Framework markiert ein Widget automatisch dirty wenn `Tick()` true zurückgibt. `DirtyTracker` bleibt für Widgets die sich aus anderen Gründen dirty markieren (z.B. externe Daten-Pushes via Surface).

### 12.4 Der generische `Anim[T]`-Typ

`Anim[T]` ist der Baustein — ein interpolierbarer Wert mit Ziel, Dauer und Easing:

```go
// Anim[T] ist ein animierter Wert. Lebt in WidgetState.
// Zero-Value ist ein sofort-fertiger Animator (kein Tick nötig).
type Anim[T Interpolatable] struct {
    from     T
    current  T
    to       T
    elapsed  time.Duration
    duration time.Duration
    easing   EasingFunc
}

// Value liefert den aktuell interpolierten Wert.
// Vor dem ersten SetTarget: Zero-Value von T.
// Nach Abschluss: exakt den Zielwert (kein Overshooting durch Floating-Point).
func (a *Anim[T]) Value() T

// SetTarget startet eine neue Animation zum Zielwert.
// Wenn bereits eine Animation läuft, wird von deren aktuellem
// Wert weiteranimiert (kein "snap to start").
func (a *Anim[T]) SetTarget(to T, spec MotionSpec)

// SetImmediate setzt den Wert sofort ohne Animation.
// Nützlich für initiale Zustände und Tests.
func (a *Anim[T]) SetImmediate(v T)

// IsDone gibt true zurück wenn keine Animation läuft.
func (a *Anim[T]) IsDone() bool

// Tick rückt die Animation um dt vor. Gibt true zurück wenn noch aktiv.
// Wird vom Framework via Animator.Tick() aufgerufen — Usercode ruft
// dies normalerweise nicht direkt auf (außer in Tests).
func (a *Anim[T]) Tick(dt time.Duration) bool
```

#### Das `Interpolatable`-Constraint

```go
// Interpolatable erlaubt generisches Lerpen über beliebige Typen.
type Interpolatable interface {
    float32 | float64 | Color | Point | Size | Rect | CornerRadii
}

// Drittanbieter können eigene Typen interpolierbar machen via:
type CustomInterpolator[T any] struct {
    Value T
    Lerp  func(from, to T, t float32) T
}
```

### 12.5 Spring-Animationen

Neben duration-basierten Animationen gibt es physikalisch basierte Springs — keine feste Dauer, das System schwingt sich auf den Zielwert ein:

```go
// SpringAnim[T] simuliert ein Feder-Dämpfer-System.
// Keine feste Duration — konvergiert asymptotisch.
type SpringAnim[T Interpolatable] struct {
    current  T
    velocity T
    target   T
    spec     SpringSpec
}

type SpringSpec struct {
    // Stiffness: Federhärte. Höher = schnelleres Einpendeln.
    // Typische Werte: 100 (soft) bis 800 (snappy).
    Stiffness float32

    // Damping: Dämpfung. Niedrig = Überschwingen, hoch = overdamped.
    // Critical damping ≈ 2 * sqrt(Stiffness * Mass).
    Damping float32

    // Mass: Träge Masse. Meist 1.0 — erhöhen für intentionales Überschwingen.
    Mass float32

    // SettlingThreshold: Unterhalb dieser Geschwindigkeit + Distanz
    // gilt die Animation als abgeschlossen. Default: 0.001.
    SettlingThreshold float32
}

func (s *SpringAnim[T]) Tick(dt time.Duration) bool
func (s *SpringAnim[T]) SetTarget(to T)
func (s *SpringAnim[T]) Value() T
func (s *SpringAnim[T]) IsDone() bool

// Preset-Springs — ausgerichtet an MotionSpec-Tokens im Theme:
var (
    SpringGentle = SpringSpec{Stiffness: 120, Damping: 14, Mass: 1.0}
    SpringSnappy = SpringSpec{Stiffness: 400, Damping: 28, Mass: 1.0}
    SpringBouncy = SpringSpec{Stiffness: 200, Damping: 10, Mass: 1.0}
)
```

### 12.6 MotionSpec im Theme (Verbindung zu §5.2)

`MotionSpec` im `TokenSet` definiert die Defaults für duration-basierte Animationen. Widgets nutzen sie um themenkonform zu animieren:

```go
type MotionSpec struct {
    Duration time.Duration
    Easing   EasingFunc
}

type EasingFunc func(t float32) float32  // t ∈ [0,1] → [0,1]

// Eingebaute Easing-Funktionen:
var (
    EaseLinear     EasingFunc
    EaseInCubic    EasingFunc
    EaseOutCubic   EasingFunc  // Standard für UI-Bewegungen
    EaseInOutCubic EasingFunc
    EaseOutExpo    EasingFunc  // Für "snappy" Übergänge
)

// CubicBezier erzeugt eine EasingFunc aus zwei Kontrollpunkten (CSS-kompatibel).
func CubicBezier(x1, y1, x2, y2 float32) EasingFunc

// MotionSpec hat drei eingebaute Slots:
type MotionSpec struct {
    Standard   DurationEasing  // 250ms EaseOutCubic  — Standard-Übergänge
    Emphasized DurationEasing  // 400ms EaseInOutCubic — betonte Übergänge
    Quick      DurationEasing  // 100ms EaseOutExpo   — schnelle Reaktionen
}

type DurationEasing struct {
    Duration time.Duration
    Easing   EasingFunc
}

// Aus dem Theme lesen — Widget bleibt themenkonform:
tokens := ctx.Theme.Tokens()
myAnim.SetTarget(1.0, tokens.Motion.Standard)
myAnim.SetTarget(1.0, tokens.Motion.Emphasized)
myAnim.SetTarget(1.0, tokens.Motion.Quick)
```

### 12.7 Vollständiges Widget-Beispiel

Ein Button mit Hover-Opacity-Animation — zeigt das komplette Pattern:

```go
type ButtonState struct {
    hoverOpacity Anim[float32]
}

// Animator-Interface: Framework tickert ButtonState automatisch.
func (s *ButtonState) Tick(dt time.Duration) bool {
    return s.hoverOpacity.Tick(dt)
    // Mehrere Animatoren: return a.Tick(dt) || b.Tick(dt)
}

func (b Button) Render(ctx RenderCtx, rawState WidgetState) (Element, WidgetState) {
    state := adoptState[ButtonState](rawState)
    tokens := ctx.Theme.Tokens()

    if b.Hovered && state.hoverOpacity.Value() < 1.0 {
        state.hoverOpacity.SetTarget(1.0, tokens.Motion.Quick)
    } else if !b.Hovered && state.hoverOpacity.Value() > 0.0 {
        state.hoverOpacity.SetTarget(0.0, tokens.Motion.Quick)
    }

    return ui.Box(
        ui.WithOpacity(state.hoverOpacity.Value() * 0.08),
        ui.Fill(SolidPaint(tokens.Colors.OnSurface)),
    ), state
}
```

Kein Goroutinen-Timer. Kein `Send`. Der Framework-Loop treibt den Tick — solange `hoverOpacity.IsDone() == false`.

### 12.8 Zwei Tiers: interne vs. user-initiierte Animationen

Nicht jede Animation ist für den User-Loop relevant. Das Framework unterscheidet zwei Tiers — strukturell, nicht nur konventionell:

#### Tier 1 — Framework-intern (Widgets)

Button-Hover, Fokus-Ring, Ripple, Scroll-Easing: Diese Animationen leben vollständig in `WidgetState`. Der User-Loop sieht sie nie — keine Msg, keine ID, kein Noise.

```go
type ButtonState struct {
    hoverOpacity Anim[float32]  // Tier 1: rein intern, kein User-Kontakt
}
```

Kein `SetTarget`-Aufruf aus dem User-Loop. Kein `AnimationEnded`. Für die Rendering-Instanz relevant, für den User nicht.

#### Tier 2 — User-initiiert (mit AnimationID)

Wenn der User eine Animation startet — z.B. ein Element ausblenden bevor es aus dem Model entfernt wird — gibt er ihr eine `AnimationID`. Das Framework sendet `AnimationEnded{ID}` in den User-Loop wenn die Animation abgeschlossen ist.

```go
// AnimationID ist ein typisierter String-Alias.
// Verhindert versehentliche Verwechslung mit anderen String-Werten.
type AnimationID string

// SetTargetWithID: wie SetTarget, aber mit ID für User-Loop-Benachrichtigung.
// Framework sendet AnimationEnded{ID: id} via app.Send() wenn fertig.
// Fire-once. Bei erneutem SetTargetWithID mit gleicher ID: vorherige
// Benachrichtigung wird verworfen, kein Double-Fire.
func (a *Anim[T]) SetTargetWithID(to T, spec MotionSpec, id AnimationID)

// Eingebaute Framework-Msg — kein Import von user-packages nötig.
type AnimationEnded struct {
    ID AnimationID
}
```

**Beispiel — Element nach Fade-Out entfernen:**

```go
// In Widget.Render: Animation starten
state.fadeOpacity.SetTargetWithID(0.0, tokens.Motion.Standard, "fade-out-item")

// In update: auf Abschluss reagieren
func update(m Model, msg app.Msg) Model {
    switch msg := msg.(type) {
    case AnimationEnded:
        if msg.ID == "fade-out-item" {
            m.Items = removeItem(m.Items, m.RemovingItemID)
        }
    }
    return m
}
```

**Warum kein `OnEnd(func())`?** Ein direkter Callback würde Mutation des User-Models aus dem Widget-Code heraus ermöglichen — Elm-Invariante gebrochen. `AnimationEnded` als Msg hält die Grenze sauber: Widget → Framework → `app.Send` → `update`. Kein Shortcut.

#### Zusammenfassung: Wer sieht was?

| Animation | Typ | User-Loop | Beispiel |
|-----------|-----|-----------|---------|
| Button-Hover-Opacity | Tier 1 | Nein | `hoverOpacity Anim[float32]` |
| Fokus-Ring-Expansion | Tier 1 | Nein | `focusRadius Anim[float32]` |
| Element-Fade-Out | Tier 2 | `AnimationEnded{ID}` | `SetTargetWithID(...)` |
| Dialog-Einblenden | Tier 2 | `AnimationEnded{ID}` | `SetTargetWithID(...)` |
| Seiten-Übergang | Tier 2 | `AnimationEnded{ID}` | `SetTargetWithID(...)` |

### 12.9 Sequenzen & Parallelanimationen

Für komplexere Choreografien gibt es `AnimGroup` und `AnimSeq`:

```go
// AnimGroup tickert mehrere Animatoren parallel.
// IsDone() == true wenn alle done sind.
type AnimGroup struct{ anims []tickable }

func (g *AnimGroup) Add(a tickable)
func (g *AnimGroup) Tick(dt time.Duration) bool
func (g *AnimGroup) IsDone() bool

// AnimSeq spielt Animationen nacheinander ab.
// Nächste startet wenn vorherige IsDone() == true.
type AnimSeq struct {
    steps   []seqStep
    current int
}
type seqStep struct {
    anim   tickable
    onDone func()  // Hook zwischen Steps, z.B. SetTarget auf nächstem Anim
}

func (s *AnimSeq) Then(a tickable, onDone ...func()) *AnimSeq  // Builder
func (s *AnimSeq) Tick(dt time.Duration) bool
func (s *AnimSeq) IsDone() bool
```

### 12.10 Testbarkeit

Da `Anim[T].Tick(dt)` keinen wall-clock-Zugriff hat, sind Animationen vollständig in Unit-Tests kontrollierbar:

```go
func TestButtonHoverAnimation(t *testing.T) {
    state := &ButtonState{}
    state.hoverOpacity.SetTarget(1.0, MotionSpec{
        Duration: 100 * time.Millisecond,
        Easing:   EaseOutCubic,
    })

    // 3 Frames à 16ms simulieren:
    for i := 0; i < 3; i++ {
        state.Tick(16 * time.Millisecond)
    }
    assert.Greater(t, state.hoverOpacity.Value(), float32(0.5))

    // Bis zum Abschluss simulieren:
    for !state.hoverOpacity.IsDone() {
        state.Tick(16 * time.Millisecond)
    }
    assert.Equal(t, float32(1.0), state.hoverOpacity.Value())
}
```

Kein Display-Server. Kein laufender App-Loop. Kein `time.Sleep`.

---

## 13. Input-System

### 13.1 Architektur-Überblick

Input-Events werden von der Platform-Implementierung (§7) empfangen, in typisierte `Msg`-Werte umgewandelt und in den App-Loop eingespeist — über denselben `msgChannel` wie alle anderen Msgs. Es gibt keinen separaten Input-Pfad, keine Callbacks, keine Observer.

```
Platform (Wayland / Win32 / DRM/KMS / ...)
    │  raw events (key press, mouse move, touch, ...)
    ▼
InputTranslator (framework-intern)
    │  typisierte Msgs
    ▼
msgChannel
    │
    ▼
App-Loop
    ├── Focus-Manager (framework-intern)
    │     Keyboard-Msgs → focused Widget
    │     Mouse-Msgs    → hit-getestetes Widget
    │
    └── Widget.Render(ctx, state) erhält Input-Msgs via RenderCtx.Events
```

**Dispatch-Modell: Flat, kein Bubbling.**
Keyboard-Events gehen direkt an das aktuell fokussierte Widget. Mouse-Events gehen an das Widget das den Treffer enthält (Hit-Test via Layout-Baum). Es gibt kein Bubbling, kein Capturing, keine Propagation-Kette. Was nicht konsumiert wird, ist verloren — bewusste Entscheidung für Vorhersagbarkeit.

### 13.2 Msg-Typen

Alle Input-Msgs leben im `input`-Package. Sie sind gewöhnliche Go-Structs — kein Interface, kein Marker.

#### Keyboard

```go
// KeyMsg wird für jeden Key-Down, Key-Up und Key-Repeat ausgelöst.
type KeyMsg struct {
    Key       Key           // Logischer Tastencode (layout-unabhängig)
    Rune      rune          // Zeichen das produziert wurde; 0 wenn keins
    Action    KeyAction     // Press, Release, Repeat
    Modifiers ModifierSet
}

type KeyAction uint8
const (
    KeyPress   KeyAction = iota
    KeyRelease
    KeyRepeat          // Held-Repeat vom OS
)

type Key uint32  // Plattformunabhängiger logischer Keycode

// Eingebaute Key-Konstanten (Auswahl — vollständige Liste im package):
const (
    KeyUnknown Key = iota
    // Alphanumerisch: KeyA ... KeyZ, Key0 ... Key9
    KeyA; KeyB /* ... */
    // Navigation:
    KeyArrowLeft; KeyArrowRight; KeyArrowUp; KeyArrowDown
    KeyHome; KeyEnd; KeyPageUp; KeyPageDown
    // Aktionen:
    KeyEnter; KeyEscape; KeyTab; KeyBackspace; KeyDelete; KeySpace
    // Funktionstasten:
    KeyF1 /* ... */ KeyF12
    // Modifier (als eigenständige Keys):
    KeyLeftShift; KeyRightShift
    KeyLeftCtrl;  KeyRightCtrl
    KeyLeftAlt;   KeyRightAlt
    KeyLeftMeta;  KeyRightMeta  // Cmd auf macOS, Win-Taste auf Windows
)

type ModifierSet uint8
const (
    ModShift ModifierSet = 1 << iota
    ModCtrl
    ModAlt
    ModMeta
)

func (m ModifierSet) Has(mod ModifierSet) bool
func (m ModifierSet) Only(mod ModifierSet) bool  // Exactly diese Modifier, keine anderen

// TextInputMsg wird für druckbare Zeichen ausgelöst — nach IME-Komposition.
// Für Textfelder immer TextInputMsg verwenden, nicht KeyMsg.Rune.
type TextInputMsg struct {
    Text string  // UTF-8, kann mehrere Runes enthalten (IME-Komposition)
}
```

**`KeyMsg` vs. `TextInputMsg`:** Keyboard-Shortcuts nutzen `KeyMsg` (logischer Key + Modifier). Texteingabe nutzt `TextInputMsg` (post-IME, korrekte Unicode-Komposition für CJK, Akzente etc.). Beides zu vermischen ist ein klassischer Fehler — das Framework macht die Trennung explizit.

#### Mouse

```go
// MouseMsg wird für alle Mausereignisse ausgelöst.
type MouseMsg struct {
    Pos     Point         // Position relativ zum Widget-Origin
    Button  MouseButton   // None wenn nur Bewegung
    Action  MouseAction
    Modifiers ModifierSet
}

type MouseButton uint8
const (
    MouseButtonNone   MouseButton = iota
    MouseButtonLeft
    MouseButtonRight
    MouseButtonMiddle
    MouseButtonBack    // Seitenknopf zurück
    MouseButtonForward // Seitenknopf vor
)

type MouseAction uint8
const (
    MousePress   MouseAction = iota
    MouseRelease
    MouseMove
    MouseEnter   // Cursor betritt Widget-Bounds
    MouseLeave   // Cursor verlässt Widget-Bounds
)

// ScrollMsg wird für Mausrad und Trackpad-Scroll ausgelöst.
type ScrollMsg struct {
    Pos    Point    // Position relativ zum Widget-Origin
    DeltaX float32  // dp, positiv = rechts
    DeltaY float32  // dp, positiv = unten
    // Precise gibt an ob das Event von einem hochauflösenden
    // Eingabegerät kommt (Trackpad, nicht Mausrad).
    // Nützlich für Kinetic-Scrolling-Entscheidungen (§14).
    Precise bool
    Modifiers ModifierSet
}
```

#### Touch

Touch ist nicht unified mit Mouse — Hover-State existiert bei Touch nicht, und Multi-Touch ist strukturell anders als einzelne Maus-Pointer.

```go
// TouchMsg repräsentiert einen einzelnen Touch-Kontakt.
type TouchMsg struct {
    ID      TouchID   // Eindeutig pro aktiven Kontakt; stabil über Phase hinweg
    Pos     Point     // Position relativ zum Widget-Origin
    Phase   TouchPhase
    // Force: Druck (0.0–1.0). Nur auf Geräten die Force Touch unterstützen.
    // 0.0 auf allen anderen Geräten.
    Force   float32
}

type TouchID uint32

type TouchPhase uint8
const (
    TouchBegan    TouchPhase = iota  // Finger berührt Display
    TouchMoved                       // Finger bewegt sich
    TouchStationary                  // Finger ruht (kein Move seit letztem Frame)
    TouchEnded                       // Finger hebt ab
    TouchCancelled                   // OS hat Touch abgebrochen (z.B. eingehender Anruf)
)
```

#### Focus

```go
// FocusGainedMsg wird an ein Widget gesendet wenn es Fokus erhält.
type FocusGainedMsg struct {
    Source FocusSource  // Keyboard (Tab), Pointer (Click), Programmatic
}

// FocusLostMsg wird an ein Widget gesendet wenn es Fokus verliert.
type FocusLostMsg struct {
    Reason FocusLostReason  // NewFocus, WindowDeactivated, Disabled
}

type FocusSource uint8
const (
    FocusSourceKeyboard    FocusSource = iota  // Tab / Shift+Tab
    FocusSourcePointer                         // Maus-Click / Touch
    FocusSourceProgrammatic                    // app.Send(RequestFocusMsg{})
)

type FocusLostReason uint8
const (
    FocusLostToNewFocus     FocusLostReason = iota
    FocusLostWindowBlur
    FocusLostWidgetDisabled
)
```

### 13.3 Focus-Management

#### Focus lebt framework-intern

Focus ist UI-State, kein Applikations-State — er lebt framework-intern neben `Map[UID]WidgetState`, nie im User-Model. Das verhindert, dass Focus-Logik den User-Loop verschmutzt.

```go
// Framework-interner Focus-State:
type focusState struct {
    focusedUID  UID       // Zero = kein Focus
    focusOrder  []UID     // Tab-Reihenfolge, aus dem Layout-Baum abgeleitet
}
```

#### Focusable-Interface

Widgets deklarieren ihre Fokussierbarkeit explizit:

```go
// Focusable ist ein optionales Interface auf Widget.
// Widgets die es nicht implementieren, sind nicht fokussierbar
// (kein Tab-Stop, kein Keyboard-Input).
type Focusable interface {
    Widget
    // FocusOptions gibt an wie dieses Widget Fokus behandelt.
    FocusOptions() FocusOpts
}

type FocusOpts struct {
    // Focusable: Widget kann Fokus erhalten. Default false.
    Focusable bool

    // TabIndex: Reihenfolge in der Tab-Sequenz.
    // 0 = natürliche Dokumentreihenfolge (aus Layout-Position).
    // Positiver Wert = explizite Reihenfolge (wie HTML tabindex).
    // -1 = fokussierbar aber nicht in Tab-Sequenz (nur programmatisch/per Klick).
    TabIndex int

    // FocusOnClick: Fokus beim Klick erhalten. Default true für TabIndex >= 0.
    FocusOnClick bool
}
```

#### Tab-Reihenfolge

Die Tab-Reihenfolge wird aus dem Layout-Baum abgeleitet (Dokumentreihenfolge, left-to-right, top-to-bottom) und gecacht. Sie wird bei jedem Reconcile aktualisiert — jedoch nur für Teilbäume die sich geändert haben.

```
Tab      → nächstes fokussierbares Widget in Tab-Reihenfolge
Shift+Tab → vorheriges fokussierbares Widget
```

#### Focus programmatisch setzen

```go
// RequestFocusMsg setzt den Fokus auf ein Widget per UID.
// Wird via app.Send() aus dem User-Loop oder aus Widget-Code geschickt.
type RequestFocusMsg struct {
    UID    UID
    Source FocusSource  // Fast immer FocusSourceProgrammatic
}

// ReleaseFocusMsg gibt den Fokus frei (kein Widget fokussiert).
type ReleaseFocusMsg struct{}

// Aktuell fokussiertes Widget aus dem User-Loop lesen:
// Nicht direkt möglich — Focus ist framework-intern.
// Stattdessen: Widget setzt in seinem WidgetState ein Flag wenn es Fokus hat.
// Das User-Model erhält dieses Flag nicht — es interessiert ihn nicht.
```

**Warum kann der User-Loop Focus nicht direkt lesen?** Weil Focus sich zwischen zwei `update`-Aufrufen ändern kann (z.B. durch Maus-Click der keine Msg an `update` sendet). Ein gecachter Focus-Wert im User-Model wäre stale. Der korrekte Weg: wenn der User-Loop auf Focus reagieren muss, schickt das Widget beim `FocusGainedMsg`/`FocusLostMsg`-Empfang eine eigene Msg in den Loop.

### 13.4 Hit-Testing

Mouse- und Touch-Events landen beim Widget das den geometrischen Treffer enthält. Hit-Testing läuft framework-intern gegen den Layout-Baum:

```go
// HitTestResult aus dem Layout-Baum:
type HitTestResult struct {
    UID      UID     // Tiefstes Widget das den Punkt enthält
    LocalPos Point   // Punkt relativ zum Widget-Origin
}
```

**Hit-Test-Reihenfolge:** Depth-first, letztes Child gewinnt (painter's algorithm — was oben gezeichnet ist, bekommt den Input). Transparent-Bereiche (alpha = 0 in der Paint-Ausgabe) werden ignoriert wenn das Widget `HitTestBehavior: HitTestTransparent` deklariert.

```go
// HitTestBehavior ist optional auf Widget — Default ist HitTestOpaque.
type HitTestable interface {
    Widget
    HitTestBehavior() HitTestBehavior
}

type HitTestBehavior uint8
const (
    HitTestOpaque      HitTestBehavior = iota  // Gesamte Bounds reagieren
    HitTestTransparent                          // Widget ignoriert Mouse/Touch
    HitTestChildren                             // Nur Children reagieren, nicht der Container
)
```

### 13.5 Keyboard-Shortcuts

Shortcuts sind deklarativ — kein globaler Event-Handler, kein `switch key { case "Ctrl+S": ... }` im `update`.

```go
// Shortcut beschreibt eine Tastenkombination.
type Shortcut struct {
    Key       Key
    Modifiers ModifierSet
}

// ShortcutMsg wird in den User-Loop gesendet wenn ein Shortcut ausgelöst wird.
type ShortcutMsg struct {
    Shortcut Shortcut
    ID       ShortcutID  // Vom User vergeben — analog zu AnimationID
}

type ShortcutID string

// Shortcuts werden via app.Option registriert, nicht im Widget-Code:
app.Run(model, update, view,
    app.WithShortcut(Shortcut{KeyS, ModCtrl},   "save"),
    app.WithShortcut(Shortcut{KeyZ, ModCtrl},   "undo"),
    app.WithShortcut(Shortcut{KeyZ, ModCtrl | ModShift}, "redo"),
)

// In update:
case ShortcutMsg:
    switch msg.ID {
    case "save": return saveModel(m)
    case "undo": return undoModel(m)
    }
```

**Scope:** `app.WithShortcut` registriert globale Shortcuts (immer aktiv). Widget-lokale Shortcuts (nur wenn das Widget fokussiert ist) werden über `KeyMsg` im Widget selbst behandelt — kein separater Mechanismus.

**Plattform-Konventionen:** Das Framework normalisiert plattformspezifische Konventionen:

| Aktion | macOS | Windows/Linux |
|--------|-------|---------------|
| Kopieren | `Cmd+C` | `Ctrl+C` |
| Einfügen | `Cmd+V` | `Ctrl+V` |
| Rückgängig | `Cmd+Z` | `Ctrl+Z` |

```go
// PlatformShortcut löst automatisch zur plattformkorrekten Kombination auf:
app.WithShortcut(input.PlatformShortcut(PlatformActionCopy), "copy")
// → macOS:   Shortcut{KeyC, ModMeta}
// → Windows: Shortcut{KeyC, ModCtrl}
```

### 13.6 Input-Dispatch im Widget

Widgets empfangen Input-Events nicht als Parameter von `Render`, sondern als Events im `RenderCtx`. Das ist die einzige Möglichkeit für ein Widget auf Input zu reagieren:

```go
type RenderCtx struct {
    UID    UID
    Theme  Theme
    Send   func(Msg)

    // Events enthält alle Input-Events die dieses Widget in diesem
    // Frame erhalten hat. Meist 0–2 Einträge. Nie nil.
    Events []InputEvent
}

// InputEvent ist ein typisierter Union-Wrapper.
type InputEvent struct {
    Kind InputEventKind
    // Genau eines der folgenden Felder ist befüllt:
    Key       KeyMsg
    TextInput TextInputMsg
    Mouse     MouseMsg
    Scroll    ScrollMsg
    Touch     TouchMsg
    Focus     FocusGainedMsg
    FocusLost FocusLostMsg
}

type InputEventKind uint8
const (
    InputKindKey InputEventKind = iota
    InputKindTextInput
    InputKindMouse
    InputKindScroll
    InputKindTouch
    InputKindFocusGained
    InputKindFocusLost
)
```

**Beispiel — einfaches Textfeld:**

```go
func (t TextField) Render(ctx RenderCtx, rawState WidgetState) (Element, WidgetState) {
    state := adoptState[TextFieldState](rawState)

    for _, ev := range ctx.Events {
        switch ev.Kind {
        case InputKindTextInput:
            state.text += ev.TextInput.Text
            state.cursor += len(ev.TextInput.Text)

        case InputKindKey:
            if ev.Key.Action == KeyPress || ev.Key.Action == KeyRepeat {
                switch ev.Key.Key {
                case KeyBackspace:
                    state.text, state.cursor = deleteBeforeCursor(state.text, state.cursor)
                case KeyArrowLeft:
                    state.cursor = moveCursorLeft(state.text, state.cursor, ev.Key.Modifiers)
                case KeyArrowRight:
                    state.cursor = moveCursorRight(state.text, state.cursor, ev.Key.Modifiers)
                }
            }

        case InputKindFocusGained:
            state.showCursor = true
        case InputKindFocusLost:
            state.showCursor = false
        }
    }

    return renderTextField(ctx, state), state
}
```

Kein globaler Event-Handler. Kein `switch msg.(type)` im User-Loop für UI-interne Eingaben. Der User-Loop sieht nur was das Widget explizit via `ctx.Send(msg)` hochreicht.

### 13.7 Cursor-Management

```go
// CursorKind deklariert den gewünschten System-Cursor.
// Wird vom Framework gesetzt wenn das Widget unter dem Maus-Pointer liegt.
type CursorKind uint8

const (
    CursorDefault    CursorKind = iota
    CursorText                   // I-Beam für Texteingabe
    CursorPointer                // Hand für Links/Buttons
    CursorMove                   // Vierpfeil für Drag
    CursorResizeNS               // Vertikaler Resize
    CursorResizeEW               // Horizontaler Resize
    CursorResizeNESW
    CursorResizeNWSE
    CursorNotAllowed
    CursorCrosshair
    CursorGrab
    CursorGrabbing
    CursorWait
    CursorProgress
    CursorNone                   // Cursor verstecken (z.B. Vollbild-Video)
)

// Cursable ist ein optionales Interface — Default ist CursorDefault.
type Cursable interface {
    Widget
    Cursor(state WidgetState) CursorKind
}
```

### 13.8 Global Handler Layer

Neben dem normalen Widget-Dispatch (§13.1) gibt es einen **Global Handler Layer** — registrierte Handler die Events sehen *bevor* sie an Widgets dispatched werden.

Das ist der korrekte Mechanismus für zwei Patterns:

**1. Erweiterte globale Shortcuts** — §13.5 deckt einfache Tastenkombinationen. Der Global Handler Layer deckt alles was komplexer ist: kontextsensitive Shortcuts, Shortcuts die auf Mouse-Events reagieren, oder Shortcuts die nur in bestimmten App-Zuständen aktiv sind.

**2. Event Delegation** — ein Container-Widget das Events für seine Children zentralisiert behandelt, ohne dass jedes Child einen eigenen Handler registriert. Nützlich für große Listen wo ein einzelner Click-Handler für alle Items effizienter ist als N individuelle Handler.

```go
// GlobalHandler verarbeitet Events vor dem normalen Widget-Dispatch.
// Gibt true zurück → Event ist konsumiert, kein Widget-Dispatch.
// Gibt false zurück → Event geht weiter zum normalen Dispatch.
type GlobalHandler func(event InputEvent) (consumed bool)

// Registrierung via app.Option:
app.Run(model, update, view,
    app.WithGlobalHandler(myHandler),
    // Mehrere Handler möglich — Reihenfolge der Registrierung = Priorität
)

// Alternativ: dynamisch via Msg (für zustandsabhängige Handler):
type RegisterHandlerMsg struct {
    ID      HandlerID
    Handler GlobalHandler
}
type UnregisterHandlerMsg struct {
    ID HandlerID
}
```

**Event Delegation Beispiel** — ein List-Container der Clicks für alle Items behandelt:

```go
// In view: Handler registrieren solange Liste sichtbar ist
func view(m Model) ui.Element {
    return ui.Stack(
        ui.Overlay{...},  // evtl. offene Overlays
        myListContainer(m),
    )
}

// Der Handler delegiert via HitTest-Ergebnis:
listHandler := func(ev InputEvent) bool {
    if ev.Kind != InputKindMouse || ev.Mouse.Action != MousePress {
        return false
    }
    // HitTest liefert welches Widget unter dem Cursor liegt:
    result := app.HitTest(ev.Mouse.Pos)
    if item, ok := itemFromUID(result.UID); ok {
        app.Send(ItemClickedMsg{ID: item.ID})
        return true  // konsumiert
    }
    return false
}
```

**Dispatch-Reihenfolge komplett:**

```
Input-Event eingetroffen
    │
    ▼
1. Global Handler Layer (Reihenfolge der Registrierung)
    │ consumed? → fertig
    │
    ▼
2. Offene Overlays (Z-Order, neueste zuerst) [§18.3]
    │ consumed? → fertig
    │
    ▼
3. Normaler Hit-Test im Layout-Baum
    │
    ▼
4. Widget erhält Event in ctx.Events
```

### 13.9 Invarianten & Vertrag

- **Kein Input außerhalb des App-Loops.** `InputEvent`-Werte verlassen den Loop nie.
- **Events sind read-only.** `ctx.Events` ist eine Slice — Mutationen haben keinen Effekt.
- **Flat Dispatch, kein Bubbling.** Kein `event.StopPropagation()` — es gibt keine Propagation. Global Handler sind kein Bubbling; sie sitzen *vor* dem Dispatch, nicht danach.
- **Ein Widget empfängt immer vollständige Event-Sequenzen.** `TouchBegan` an ein Widget → alle folgenden `TouchMoved`/`TouchEnded` mit gleicher `TouchID` gehen an dasselbe Widget, auch außerhalb der Bounds. (Touch-Capture-Semantik.)
- **Global Handler sind synchron.** Sie laufen im App-Loop, dürfen `app.Send` aufrufen, aber keinen blockierenden I/O machen.

---

## 14. Scroll & Kinetic Scrolling

### 14.1 Einordnung

Kinetic Scrolling ist kein Framework-Konzept das überall eingebaut ist — es ist ein **Widget-Pattern**, das das Framework durch drei Bausteine ermöglicht:

- `ScrollMsg.Precise` (§13.2) — unterscheidet Trackpad (kinetic-fähig) von Mausrad (stepped)
- `Animator`-Interface (§12.3) — treibt den Deceleration-Tick frame-by-frame
- `MotionSpec.Scroll` im Theme (§5.2) — konfigurierbare Physik-Parameter

Ein `Scrollable`-Widget das diese Bausteine nutzt, bekommt Kinetic Scrolling out-of-the-box. Ein Widget das nur Mausrad-Scrolling braucht, ignoriert `Precise` und `Animator` vollständig.

### 14.2 Bewegungsmodell: Friction-Decay, nicht Spring

Kinetic Scrolling nutzt **exponentielle Abbremsung** (Friction-Decay), keinen Spring:

- Spring würde über den Zielwert hinausschwingen — falsch für normales Scrolling.
- Friction-Decay: Geschwindigkeit × Faktor pro Frame → asymptotische Annäherung an Stillstand.
- **Rubber-Banding** am Rand ist ein separater Spring — bewusst unterschiedliches Verhalten.

```
v(t) = v₀ × friction^(dt/frameTarget)

friction ∈ (0, 1) — typisch 0.95 bei 16ms-Frames
Stillstand wenn |v(t)| < settlingThreshold
```

### 14.3 `KineticScroll`-Typ

```go
// KineticScroll verwaltet den kompletten Scroll-Zustand eines scrollbaren
// Widgets: Position, Geschwindigkeit, Overscroll und Rubber-Band-Rückfederung.
// Lebt in WidgetState. Implementiert Animator (§12.3).
type KineticScroll struct {
    // Aktuelle Scroll-Position in dp. Öffentlich lesbar, nie direkt schreiben.
    OffsetX float32
    OffsetY float32

    // Interne Felder:
    velX, velY       float32         // Aktuelle Geschwindigkeit (dp/frame)
    phase            scrollPhase     // Idle, Tracking, Decelerating, Snapping
    spec             ScrollSpec
    boundsX, boundsY scrollBounds    // [min, max] erlaubter Offset
}

type scrollPhase uint8
const (
    scrollIdle         scrollPhase = iota
    scrollTracking     // Finger/Trackpad aktiv, direkte Positionssteuerung
    scrollDecelerating // Finger weg, Friction-Decay läuft
    scrollSnapping     // Rubber-Band-Rückfederung via Spring
)

// SetBounds definiert den scrollbaren Bereich.
// minX/minY: meist 0. maxX/maxY: contentSize - viewportSize.
// Negative max = Content kleiner als Viewport → kein Scroll möglich.
func (k *KineticScroll) SetBounds(minX, maxX, minY, maxY float32)

// Feed verarbeitet ein ScrollMsg aus ctx.Events.
// Muss für jedes ScrollMsg aufgerufen werden — kein automatisches Dispatch.
func (k *KineticScroll) Feed(msg ScrollMsg)

// Tick implementiert Animator — wird vom Framework aufgerufen (§12.3).
func (k *KineticScroll) Tick(dt time.Duration) bool

// SnapTo scrollt programmatisch zu einem Offset — mit Rubber-Band-Animation.
func (k *KineticScroll) SnapTo(x, y float32)

// SnapToImmediate scrollt ohne Animation (z.B. bei initialem State).
func (k *KineticScroll) SnapToImmediate(x, y float32)

// IsDone gibt true zurück wenn kein aktiver Scroll-Vorgang läuft.
func (k *KineticScroll) IsDone() bool
```

### 14.4 `ScrollSpec` im Theme

`ScrollSpec` ist Teil des `TokenSet` (§5.2) — Scroll-Physik ist Theme-Konfiguration:

```go
type ScrollSpec struct {
    // Friction: Abbremsfaktor pro Frame bei 60fps-Basis.
    // 0.95 = sanftes Ausrollen, 0.80 = schnelles Stoppen.
    Friction float32  // Default: 0.95

    // OverscrollDistance: Maximale Rubber-Band-Auslenkung in dp.
    OverscrollDistance float32  // Default: 80.0

    // OverscrollSpring: Physik für die Rückfederung.
    // Kein Überschwingen gewünscht → Damping nahe critical.
    OverscrollSpring SpringSpec  // Default: {Stiffness: 300, Damping: 30}

    // SettlingThreshold: Unter dieser Geschwindigkeit (dp/frame) gilt
    // der Scroll als abgeschlossen.
    SettlingThreshold float32  // Default: 0.5

    // StepSize: Scroll-Betrag pro Mausrad-Klick (Precise == false).
    // In dp.
    StepSize float32  // Default: 48.0

    // MultiplierPrecise: Faktor für Trackpad-Deltas (Precise == true).
    // Trackpad-Deltas sind meist kleiner als Mausrad-Steps.
    MultiplierPrecise float32  // Default: 1.5
}
```

### 14.5 Feed-Logik im Detail

`Feed` entscheidet anhand von `ScrollMsg.Precise` welcher Pfad genommen wird:

```go
func (k *KineticScroll) Feed(msg ScrollMsg) {
    if msg.Precise {
        // Trackpad: direkte Positionssteuerung + Velocity-Tracking
        // → Phase: scrollTracking
        // Geschwindigkeit aus den letzten N Deltas schätzen (gleitender Schnitt)
        // → Beim letzten Event (kleine Deltas): Phase → scrollDecelerating
    } else {
        // Mausrad: diskrete Schritte, keine Kinetik
        // → Direkt SnapTo(current + step * sign(delta))
        // → Kein Deceleration-Pass nötig
    }

    // Overscroll-Check: wenn neuer Offset außerhalb bounds → Rubber-Band
    // Offset darf bounds überschreiten, aber mit gedämpftem Faktor:
    // overscrollDelta = clampedDelta + (rawDelta - clampedDelta) * 0.3
}
```

**Velocity-Tracking:** Das Trackpad sendet viele kleine `ScrollMsg`s. Velocity wird aus einem gleitenden Schnitt der letzten 4–6 Deltas berechnet (gewichtet nach Alter). Beim Abheben (Deltas werden sehr klein) wechselt die Phase von `scrollTracking` nach `scrollDecelerating`.

### 14.6 Overscroll & Rubber-Banding

Overscroll ist erlaubt — mit gedämpftem Widerstand während des Trackings und Spring-Rückfederung danach:

```
Tracking (Finger aktiv):
  overscroll = (rawOffset - clampedOffset) × 0.3
  → Offset "folgt" dem Finger, aber nur 30% des Weges

Decelerating (Finger weg, Offset außerhalb bounds):
  → KineticScroll wechselt sofort zu SpringAnim zurück zu bounds
  → spec.OverscrollSpring bestimmt die Rückfederung
  → IsDone() == false bis Spring settled
```

### 14.7 Vollständiges Beispiel

```go
type ScrollViewState struct {
    scroll KineticScroll
}

func (s *ScrollViewState) Tick(dt time.Duration) bool {
    return s.scroll.Tick(dt)
}

func (sv ScrollView) Render(ctx RenderCtx, rawState WidgetState) (Element, WidgetState) {
    state := adoptState[ScrollViewState](rawState)
    tokens := ctx.Theme.Tokens()

    // Bounds aus Content- und Viewport-Größe setzen:
    state.scroll.SetBounds(0, sv.ContentWidth-sv.ViewportWidth,
                           0, sv.ContentHeight-sv.ViewportHeight)

    // ScrollMsg verarbeiten:
    for _, ev := range ctx.Events {
        if ev.Kind == InputKindScroll {
            state.scroll.Feed(ev.Scroll)
        }
    }

    // Offset für Rendering lesen:
    return ui.Box(
        ui.ClipContent(true),
        ui.Child(
            ui.WithOffset(-state.scroll.OffsetX, -state.scroll.OffsetY),
            sv.Content,
        ),
    ), state
}
```

Kein `time.Sleep`, kein Goroutinen-Timer, kein `app.Send` für den Scroll-Tick. Das Framework ruft `Tick` über das `Animator`-Interface auf.

### 14.8 User-Loop-Benachrichtigung (optional)

Der User-Loop bekommt standardmäßig nichts vom Scroll mit. Wenn er die aktuelle Scroll-Position kennen muss (z.B. für "Lade mehr Daten wenn Scroll-Ende erreicht"):

```go
// ScrollPositionMsg kann ein Widget selbst verschicken wenn nötig:
type ScrollPositionMsg struct {
    WidgetUID UID
    OffsetX   float32
    OffsetY   float32
    AtBottom  bool
    AtTop     bool
}

// Im Widget.Render, nach Feed:
if state.scroll.OffsetY > sv.ContentHeight - sv.ViewportHeight - 200 {
    ctx.Send(ScrollNearBottomMsg{WidgetUID: ctx.UID})
}
```

Kein automatisches Dispatching durch das Framework — das Widget entscheidet selbst ob und wann es den User-Loop informiert.

---

## 15. Layout-System

### 15.1 Einordnung & Designziele

Das Layout-System löst eine einzige Frage: **Welche Größe und Position bekommt jedes Widget?**

Designziele:
- **Constraint-basiert, nicht absolut** — Widgets deklarieren ihre Anforderungen, der Parent entscheidet.
- **Einmaliger Pass** — kein Layout-Thrashing durch gegenseitige Abhängigkeiten.
- **Erweiterbar** — eigene Layout-Algorithmen ohne Framework-Fork, via Interface.
- **Flexbox-kompatibel** — bekanntes Modell, gute Tooling-Unterstützung, kein CSS-Cascade-Overhead.

### 15.2 Das Constraint-Modell

Jedes Widget bekommt vom Parent einen `Constraints`-Wert und gibt eine `Size` zurück:

```go
// Constraints definieren den erlaubten Größenbereich für ein Widget.
type Constraints struct {
    MinWidth, MaxWidth float32  // dp; MaxWidth = +Inf bedeutet "unbegrenzt"
    MinHeight, MaxHeight float32
}

// Tight: Widget muss exakt diese Größe haben.
func TightConstraints(w, h float32) Constraints

// Loose: Widget darf bis zu dieser Größe sein, aber auch kleiner.
func LooseConstraints(maxW, maxH float32) Constraints

// Unbounded: Widget bestimmt seine Größe selbst.
func UnboundedConstraints() Constraints
```

**Layout-Protokoll:**

```
Parent ruft layout(child, constraints) auf
    → Child berechnet seine Size
    → Child ruft rekursiv layout(grandchild, childConstraints) auf
    → Jedes Widget gibt genau eine Size zurück
    → Parent positioniert Child mit offset(x, y)
```

Kein zweiter Pass, kein "measure then layout". Einmaliger Depth-first-Durchlauf.

### 15.3 Das Layout-Interface

```go
// Layout ist ein optionales Interface auf Widget.
// Widgets die es nicht implementieren, erhalten Tight-Constraints
// (nehmen genau den Platz den ihr Parent zuweist).
type Layout interface {
    Widget
    // LayoutChildren berechnet Size und Position aller Children.
    // ctx.Measure(child, constraints) misst ein Child.
    // ctx.Place(child, offset) positioniert es.
    // Gibt die eigene Size zurück.
    LayoutChildren(ctx LayoutCtx, children []Widget) Size
}

type LayoutCtx struct {
    // Constraints die dieser Widget vom Parent bekommen hat.
    Constraints Constraints

    // Measure misst ein Child unter den gegebenen Constraints.
    // Gibt die vom Child gewünschte Size zurück.
    // Darf mehrfach aufgerufen werden — ist jedoch teuer für komplexe Subtrees.
    Measure func(child Widget, c Constraints) Size

    // Place positioniert ein Child relativ zum eigenen Origin.
    // Muss nach Measure aufgerufen werden.
    Place func(child Widget, offset Point)

    // Theme für layout-relevante Tokens (Spacing, etc.)
    Theme Theme
}
```

### 15.4 Flexbox-Layout

Das eingebaute Haupt-Layout ist Flexbox — bekannt aus CSS, vereinfacht auf die für Desktop-UI relevanten Eigenschaften:

```go
type Flex struct {
    // Direction: Hauptachse des Layouts.
    Direction FlexDirection  // Row (default), Column

    // Wrap: Zeilenumbruch wenn Platz nicht ausreicht.
    Wrap FlexWrap  // NoWrap (default), Wrap, WrapReverse

    // Justify: Ausrichtung entlang der Hauptachse.
    Justify JustifyContent
    // Align: Ausrichtung entlang der Querachse.
    Align AlignItems

    // Gap zwischen Children (dp).
    RowGap    float32
    ColumnGap float32

    Children []FlexChild
}

type FlexChild struct {
    Widget Widget

    // Grow: Anteil des verfügbaren Restraums den dieses Child bekommt.
    // 0 = kein Wachsen. 1 = gleicher Anteil wie andere Grow-1-Children.
    Grow float32

    // Shrink: Wie stark dieses Child schrumpfen darf.
    // 0 = nicht schrumpfen. 1 = proportional. Default: 1.
    Shrink float32

    // Basis: Ausgangsgröße vor Grow/Shrink.
    // Auto = Widget bestimmt seine natürliche Größe.
    Basis FlexBasis

    // AlignSelf überschreibt Align für dieses Child.
    AlignSelf AlignSelf  // Auto = erbt von Parent

    // MinWidth/MaxWidth/MinHeight/MaxHeight als direkte Constraints.
    // Überschreiben die vom Flex-Algorithmus berechneten Constraints.
    MinWidth, MaxWidth   float32
    MinHeight, MaxHeight float32
}

type FlexDirection uint8
const (
    FlexRow    FlexDirection = iota  // Links nach rechts (default)
    FlexColumn                       // Oben nach unten
    FlexRowReverse
    FlexColumnReverse
)

type JustifyContent uint8
const (
    JustifyStart        JustifyContent = iota
    JustifyEnd
    JustifyCenter
    JustifySpaceBetween
    JustifySpaceAround
    JustifySpaceEvenly
)

type AlignItems uint8
const (
    AlignStart   AlignItems = iota
    AlignEnd
    AlignCenter
    AlignStretch  // Default: Children füllen Querachse
    AlignBaseline
)

type FlexBasis struct {
    Kind  FlexBasisKind
    Value float32  // dp, nur wenn Kind == FlexBasisFixed
}

type FlexBasisKind uint8
const (
    FlexBasisAuto  FlexBasisKind = iota  // Natürliche Widget-Größe
    FlexBasisFixed                        // Expliziter dp-Wert
    FlexBasisFill                         // 100% der Constraint-Größe
)
```

### 15.5 Weitere eingebaute Layouts

Flexbox deckt ~90% der Fälle. Für den Rest:

```go
// Stack: Children übereinandergelegt (Z-Achse), wie CSS position: absolute.
// Jedes Child positioniert sich selbst via Alignment oder explizitem Offset.
type Stack struct {
    Children []StackChild
}
type StackChild struct {
    Widget    Widget
    Alignment Alignment  // TopLeft, Center, BottomRight etc.
    Offset    Point      // Zusätzlicher Offset nach Alignment
}

type Alignment uint8
const (
    AlignTopLeft Alignment = iota
    AlignTopCenter
    AlignTopRight
    AlignCenterLeft
    AlignCenter
    AlignCenterRight
    AlignBottomLeft
    AlignBottomCenter
    AlignBottomRight
)

// Grid: Gleichmäßiges Raster.
type Grid struct {
    Columns  int          // Anzahl Spalten; Zeilen ergeben sich automatisch
    RowGap   float32
    ColGap   float32
    Children []Widget
}

// Padding: Fügt Innenabstand um ein einzelnes Child hinzu.
type Padding struct {
    Insets  Insets   // Top, Right, Bottom, Left in dp
    Child   Widget
}

// SizedBox: Erzwingt eine bestimmte Größe für ein Child.
// Child = nil → leeres Widget mit der gegebenen Größe (Spacer).
type SizedBox struct {
    Width, Height float32
    Child         Widget   // Optional
}

// Expanded: Nimmt allen verfügbaren Platz auf der Hauptachse ein.
// Nur sinnvoll als direktes Child eines Flex.
type Expanded struct {
    Grow  float32  // Default: 1
    Child Widget
}

// Hinweis: Ein "Intrinsic"-Widget (wie in Flutter) existiert bewusst nicht.
// Das Constraint-Modell ist strikt top-down: jedes Widget bekommt
// Constraints vom Parent und gibt eine Size zurück — das reicht immer.
// Szenarien die sich nach Intrinsic anfühlen (z.B. "alle Buttons so breit
// wie der breiteste") sind ein Layout-Design-Geruch und werden mit Grid
// oder Flex/AlignStretch sauber gelöst. Ein zweiter Mess-Pass ist nie nötig.
```

### 15.6 Spacing-Tokens

Abstände kommen aus dem Theme — kein Magic-Number-Streusel im Widget-Code:

```go
type SpacingScale struct {
    XS  float32  //  4 dp
    S   float32  //  8 dp
    M   float32  // 16 dp  (Standard-Innenabstand)
    L   float32  // 24 dp
    XL  float32  // 32 dp
    XXL float32  // 48 dp
}

// Verwendung:
tokens := ctx.Theme.Tokens()
ui.Padding{
    Insets: ui.UniformInsets(tokens.Spacing.M),
    Child:  myContent,
}
```

### 15.7 Custom Layout

Eigene Layout-Algorithmen implementieren das `Layout`-Interface:

```go
// Beispiel: WrapLayout — bricht auf mehrere Zeilen um, wie Flexbox Wrap,
// aber mit anpassbarer Zeilen-Alignment-Logik.
type WrapLayout struct {
    Children   []Widget
    RowSpacing float32
    ColSpacing float32
}

func (w WrapLayout) LayoutChildren(ctx LayoutCtx, children []Widget) Size {
    x, y, rowHeight := float32(0), float32(0), float32(0)
    maxW := ctx.Constraints.MaxWidth

    for _, child := range children {
        size := ctx.Measure(child, LooseConstraints(maxW-x, ctx.Constraints.MaxHeight))
        if x > 0 && x+size.Width > maxW {
            // Zeilenumbruch
            x = 0
            y += rowHeight + w.RowSpacing
            rowHeight = 0
        }
        ctx.Place(child, Point{x, y})
        x += size.Width + w.ColSpacing
        rowHeight = max(rowHeight, size.Height)
    }

    return Size{maxW, y + rowHeight}
}
```

Das Framework muss kein Widget kennen das `Layout` implementiert — jeder Drittanbieter kann eigene Layout-Container bauen.

### 15.8 Layout-Cache & Invalidierung

Layout ist teuer verglichen mit Paint. Das Framework cachet Layout-Ergebnisse auf Sub-Tree-Ebene:

```go
// Intern: jeder VNode trägt sein letztes Layout-Ergebnis
type layoutCache struct {
    constraints Constraints  // Unter diesen Constraints berechnet
    size        Size
    childRects  []Rect       // Positionen der Children
    valid       bool
}
```

**Invalidierung:** Ein Node wird als layout-dirty markiert wenn:
- Seine eigenen Props sich geändert haben
- Seine Constraints sich geändert haben (weil ein Ancestor sich geändert hat)
- Ein direktes Child layout-dirty ist (weil das die eigene Size beeinflussen kann)

**Kein Layout-Thrashing:** Da `Measure` ausschließlich top-down aufgerufen werden darf — ein Child kann nie seinen Parent oder einen Sibling messen — gibt es strukturell keine zirkulären Abhängigkeiten und keinen zweiten Pass. O(n) im schlechtesten Fall, O(dirty subtree) im Normalfall.

Das ist kein Zufall: das API macht es unmöglich einen Bottom-up-Mess-Pass zu bauen. `LayoutCtx.Measure` nimmt nur `child Widget` als Parameter — kein Zugriff auf Parent oder Siblings.

### 15.9 Insets-Typ

```go
type Insets struct {
    Top, Right, Bottom, Left float32  // dp
}

func UniformInsets(all float32) Insets
func SymmetricInsets(horizontal, vertical float32) Insets
func HorizontalInsets(left, right float32) Insets
func VerticalInsets(top, bottom float32) Insets
```

---

## 16. Text-Stack, i18n & Package-Name

### 16.1 CGo-Strategie: Minimal und explizit

**Eine CGo-Abhängigkeit existiert: wgpu-native.**

Das ist die ehrliche Aussage. wgpu (§6.1) bindet gegen native GPU-APIs — Vulkan, Metal, D3D12. Ein reines Go-Äquivalent für GPU-Abstraktion auf allen Zielplattformen existiert nicht. Diese Abhängigkeit ist fundamental und bewusst gewählt.

Alles andere ist CGo-frei:

| Bereich | Abhängigkeit | CGo? |
|---------|-------------|------|
| GPU-Rendering (v1.0) | wgpu-native via CGo | **Ja** — temporär |
| GPU-Rendering (Ziel) | `gogpu/wgpu` pure Go | **Nein** — `CGO_ENABLED=0` |
| Text-Shaping | go-text/typesetting | Nein |
| BiDi | golang.org/x/text | Nein |
| Font-Rasterisierung | golang.org/x/image | Nein |
| Platform (Wayland/X11) | xgb / wayland-go | Nein |
| Platform (Win32/Cocoa) | Eigene Bindings via `gogpu/gogpu` | Nein¹ |
| DRM/KMS | Eigene syscall-Bindings | Nein |
| A11y (Linux) | D-Bus via godbus | Nein |
| A11y (macOS/Windows) | NSAccessibility/UIA | **Ja** — opt-in via Build-Tag |
| System-Fonts | fontconfig/CoreText | **Ja** — opt-in via `-tags systemfonts` |

¹ `gogpu/gogpu` implementiert Cocoa via pure Go FFI (`goffi`, `cgo_import_dynamic`) — kein C-Compiler nötig.

**CGo-Freiheit ist heute per Build-Tag erreichbar.** `-tags gogpu` aktiviert `gogpu/wgpu` als pure-Go-Backend (§6.1) — damit entfällt die GPU-Rendering-Abhängigkeit vollständig. Der Default bleibt `wgpu-native` solange `gogpu/wgpu` noch Produktionsreife aufbaut.

**`gogpu/gg` als Canvas-Implementierungsbasis:** Die Canvas-API in §6.2 und `gogpu/gg` sind konzeptuell sehr ähnlich — GPU-Beschleunigung, SDF-Text, MSDF-Atlas, Path-Builder. Es ist sinnvoll `gogpu/gg` als Implementierungsbasis ernsthaft zu evaluieren, statt den 2D-Renderer komplett selbst zu bauen. Die öffentliche Canvas-API des Frameworks (§6.2) bleibt davon unberührt — `gogpu/gg` wäre ein Implementierungsdetail.

Das war ursprünglich als Kompromiss geplant: `BasicShaper` (pure Go, eingeschränkt) als Standard, HarfBuzz via CGo als opt-in für komplexe Schriften. Dieser Kompromiss ist nicht nötig, weil er auf einer falschen Prämisse beruht: dass vollständiges OpenType-Shaping in pure Go nicht existiert.

**`github.com/go-text/typesetting`** ist der offizielle Text-Stack des Gio-Projekts — pure Go, OpenType-vollständig, produktionserprobt:

- Vollständiges GSUB (Glyph Substitution) und GPOS (Glyph Positioning)
- Arabisch, Hebräisch, Devanagari, Bengali, Thai, Khmer, Myanmar — echtes Script-Shaping
- BiDi via `golang.org/x/text/unicode/bidi`
- Aktiv entwickelt von Google-Mitarbeitern und der Gio-Community

HarfBuzz (CGo) ist damit für Desktop-Anwendungen nicht mehr das richtige Werkzeug. Eine CGo-Abhängigkeit für Text-Shaping entfällt vollständig.

### 16.2 Der vollständige Text-Stack

Alle Abhängigkeiten sind pure Go, alle sind offizielle oder produktionserprobte Projekte:

```
Eingabe: string (UTF-8)
    │
    ▼ Normalisierung & Segmentierung
    │   golang.org/x/text/unicode/norm     — NFC/NFD-Normalisierung
    │   golang.org/x/text/unicode/bidi     — BiDi-Paragraph-Analyse
    │   golang.org/x/text                  — Grapheme-Cluster-Segmentierung
    │
    ▼ Script-Erkennung & Run-Segmentierung
    │   go-text/typesetting/font/opentype  — Script/Language-Tags
    │   Eingabe → []ShapingRun (je ein Font, eine Richtung, ein Script)
    │
    ▼ Text-Shaping
    │   go-text/typesetting/shaping        — OpenType GSUB/GPOS, alle Scripts
    │   Ausgabe: []ShapedGlyph mit Advance, Offset, Cluster-Index
    │
    ▼ Rasterisierung → SDF-Atlas
        golang.org/x/image/font/sfnt       — Glyph-Outlines aus TTF/OTF
        Eigener MSDF-Rasterizer             — pure Go, SDF-Textur-Generierung
```

Kein CGo in dieser Pipeline. Kein Build-Tag. Kein Kompromiss.

### 16.3 Das interne Shaper-Interface

Das Interface bleibt wie in §16 ursprünglich spezifiziert — aber es gibt nur noch eine Implementierung:

```go
// Shaper formt einen einzelnen ShapingRun in positionierte Glyphen.
// Einzige Implementierung: GoTextShaper (go-text/typesetting).
// Das Interface existiert für Testbarkeit und Drittanbieter-Erweiterungen,
// nicht für einen CGo-Austausch.
type Shaper interface {
    Shape(run ShapingRun, font *Font, size float32) []ShapedGlyph
}

type ShapingRun struct {
    Text      string
    Direction TextDirection  // LTR, RTL
    Script    language.Script
    Language  language.Language
}

type ShapedGlyph struct {
    GlyphID  GlyphID
    Advance  float32  // Horizontaler Vorschub (dp)
    OffsetX  float32  // Kerning, Ligatur-Feinposition
    OffsetY  float32
    Cluster  int      // Index in Eingabe-String (Cursor-Positionierung)
}

type TextDirection uint8
const (
    TextDirectionLTR  TextDirection = iota
    TextDirectionRTL
    TextDirectionAuto  // Aus erstem stark-direktionalem Zeichen abgeleitet
)
```

### 16.4 Font-Loading & Fallback-Chain

#### Font-Loading (pure Go)

```go
// Font ist eine geladene TTF/OTF-Datei. Immutable nach Load.
type Font struct { /* intern: go-text/typesetting/font/opentype.Font */ }

// Aus Datei:
font, err := fonts.LoadFile("assets/Inter-Regular.ttf")

// Eingebettet via go:embed (empfohlen für reproduzierbare Builds):
//go:embed assets/fonts
var fontFS embed.FS
font, err := fonts.LoadFS(fontFS, "assets/fonts/Inter-Regular.ttf")

// Aus Bytes:
font, err := fonts.LoadBytes(data)
```

#### FontFamily & Fallback-Chain

```go
type FontFamily struct {
    Name     string
    Faces    map[FontFaceKey]*Font
    Fallback []*FontFamily  // Konsultiert bei fehlenden Glyphen, in Reihenfolge
}

type FontFaceKey struct {
    Weight FontWeight  // 100 (Thin) … 900 (Black); 400 = Regular
    Style  FontStyle   // Normal, Italic, Oblique
}
```

#### Eingebettetes Fallback-Font

```go
// fonts.Fallback: Noto-Sans-Subset, eingebettet via go:embed.
//
// Abgedeckte Schriften:
//   Noto Sans:          Latin, Kyrillisch, Griechisch, CJK-Basis
//   Noto Sans Arabic:   Arabisch (mit vollem Shaping via go-text)
//   Noto Sans Devanagari: Hindi, Sanskrit
//   Noto Emoji:         Subset häufiger Emoji
//
// Größe: ~2.5 MB komprimiert im Binary.
// Deterministisch — kein Filesystem-Zugriff zur Laufzeit.
var Fallback *FontFamily
```

#### Glyph-Fallback-Algorithmus

```
Für jede Glyphe im ShapedRun:
  1. Primärer Font der FontFamily → vorhanden? ✓
  2. FontFamily.Fallback[0]       → vorhanden? ✓
  3. FontFamily.Fallback[1]       → ...
  4. fonts.Fallback (eingebettet) → vorhanden? ✓
  5. U+FFFD □                     — nie panic, nie leerer Render
```

Fallback läuft pro Glyph — ein Emoji mitten in Latin-Text löst nur für dieses Emoji einen Fallback aus.

#### Optionales System-Font-Scanning

```go
// -tags systemfonts: scannt OS-Fonts und erweitert die Fallback-Chain.
//
// Linux:   /usr/share/fonts direktes Parsen — pure Go
// macOS:   CoreText-Bindings — CGo
// Windows: DirectWrite-Bindings — CGo
//
// Bewusst nicht im Standard-Build:
//   • Bricht Reproduzierbarkeit (unterschiedliche Fonts je System)
//   • Erfordert CGo auf macOS/Windows
//   • Für die meisten Desktop-Anwendungen nicht nötig
```

Das ist das einzige verbleibende CGo-Berührungspunkt — und er ist strikt opt-in.

### 16.5 BiDi: Vollständige Unicode-Unterstützung

```go
// BidiParagraph analysiert einen Paragraph und gibt geordnete Runs zurück.
// Implementierung: golang.org/x/text/unicode/bidi — pure Go, UAX#9-konform.
func BidiParagraph(text string, baseDir TextDirection) []ShapingRun

// Paragraph-Basisrichtung:
//   TextDirectionAuto → aus erstem stark-direktionalem Zeichen (UAX#9 P2/P3)
//   TextDirectionLTR  → explizit links-nach-rechts
//   TextDirectionRTL  → explizit rechts-nach-links (z.B. arabische UI)
```

Mixed-Direction-Text (Arabisch mit eingebetteten Zahlen oder lateinischen Begriffen) wird korrekt verarbeitet — Bidi-Embedding-Levels, Mirroring-Characters, Neutral-Characters. Das ist keine Best-Effort-Implementierung sondern UAX#9-Konformität.

### 16.6 Package-Name

Der Root-Name ist eine öffentliche API-Entscheidung die für die Lebensdauer des Projekts gilt.

**Kriterien:** Kurz (≤2 Silben), kein Konflikt mit stdlib oder bekannten Packages, kein Trademark, aussprech- und merkbar, neutrales Tooling-Verhalten (kein Konflikt mit `go doc`, `gopls`, etc.).

**Kandidaten:**

| Name | Import | Pro | Contra |
|------|--------|-----|--------|
| `arc` | `github.com/x/arc` | Kurz, einprägsam | Belegt als Archiv-Tool |
| `keel` | `github.com/x/keel` | Strukturmetapher, unbelegt | Wenig UI-Assoziation |
| `nova` | `github.com/x/nova` | Frisch, klangvoll | Generisch |
| `lux` ✓ | `github.com/timzifer/lux` | Licht/Rendering-Assoziation | **Gewählt** |
| `fir` | `github.com/x/fir` | Sehr kurz, unbelegt | Keine offensichtliche Bedeutung |
| `yew` | `github.com/x/yew` | Kurz, unbelegt in Go | Rust-Framework Yew |

**Empfehlung:** Entscheidung bis zum ersten öffentlichen Release offenlassen. Paketname: `lux` — `github.com/timzifer/lux`. Die Sub-Package-Struktur ist unabhängig und kann jetzt festgelegt werden:

```
lux/app        — Run, Send, Option
lux/ui         — Element, Widget, alle eingebauten Widgets
lux/theme      — Theme, TokenSet, Default
lux/draw       — Canvas, Paint, Path, Color
lux/input      — KeyMsg, MouseMsg, alle Input-Typen
lux/fonts      — Font, FontFamily, Fallback
lux/anim       — Anim[T], SpringAnim, AnimGroup, AnimSeq
lux/platform   — Platform-Interface (für Drittanbieter-Platforms)
lux/a11y       — AccessNode, AccessRole (öffentlich für Tests, §11.8)
lux/layout     — Constraints, Flexbox, Stack, Grid (eingebaute Layouts)
```

---

## 17. Ausblick: Inspector & Debugging-Tools

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

## 18. Datenbasierte Widgets & Overlay-System

### 18.1 Datenbasierte Widgets: Das BuildNode-Pattern

Datenbasierte Widgets wie Trees, Menüs und verschachtelte Listen folgen demselben Grundprinzip wie `VirtualList` (§6.4) — der VTree enthält nur sichtbare Nodes, die Datenbasis lebt im User-Model.

Das zentrale Pattern: **eine Funktion statt eine Slice.** Das Widget bekommt keine fertige Liste von Child-Widgets, sondern eine Funktion die für einen gegebenen Datenpunkt das Widget erzeugt. Das Framework entscheidet wann und für welche Punkte sie aufgerufen wird.

```go
// Das BuildNode-Grundmuster — verwendet von Tree und Menu:
type BuildNode[ID comparable] func(id ID, ctx NodeCtx) Widget

type NodeCtx struct {
    Depth    int      // Tiefe im Baum (0 = Root)
    Expanded bool     // Ist dieser Node expandiert? (aus WidgetState)
    Selected bool     // Ist dieser Node selektiert?
    Index    int      // Flacher Index in der sichtbaren Sequenz
}
```

### 18.2 Tree-Widget

```go
// Tree ist ein virtualisiertes, hierarchisches Listen-Widget.
// Die Datenbasis lebt im User-Model — Tree kennt nur IDs.
type Tree[ID comparable] struct {
    // RootIDs: Top-Level-Nodes. Reihenfolge ist signifikant.
    RootIDs []ID

    // Children gibt die direkten Kinder einer ID zurück.
    // Wird lazy aufgerufen — nur für expandierte Nodes.
    // Gibt nil/leere Slice zurück → Node ist ein Leaf.
    Children func(id ID) []ID

    // BuildNode erzeugt das Widget für eine gegebene ID.
    // Wird nur für sichtbare (nicht-geklappte) Nodes aufgerufen.
    BuildNode BuildNode[ID]

    // NodeHeight: Einheitliche Node-Höhe in dp.
    // Für variable Höhen: NodeHeightFunc.
    NodeHeight     float32
    NodeHeightFunc func(id ID) float32

    // IndentWidth: Einrückung pro Tiefenebene in dp.
    // Default: tokens.Spacing.L (aus Theme)
    IndentWidth float32

    // Overscan: Nodes über/unter dem Viewport (Default: 3)
    Overscan int
}
```

**Expand/Collapse-State lebt in `TreeState` (WidgetState), nie im User-Model:**

```go
type TreeState[ID comparable] struct {
    expanded  map[ID]bool
    selected  ID
    scroll    KineticScroll
}

// TreeState implementiert Animator für Scroll-Kinetics
func (s *TreeState[ID]) Tick(dt time.Duration) bool {
    return s.scroll.Tick(dt)
}
```

Der User-Loop sieht Expand/Collapse nicht — das ist UI-State. Wenn der User-Loop reagieren muss (z.B. lazy-load von Child-Daten), schickt das Widget eine Msg:

```go
// Vom Tree automatisch gesendet wenn ein Node expandiert wird
// dessen Children-Slice leer ist (potentiell lazy):
type TreeNodeExpandedMsg[ID comparable] struct {
    NodeID ID
}

// In update: Daten nachladen
case TreeNodeExpandedMsg[FileID]:
    m.LoadingNodes[msg.NodeID] = true
    return m, func() Msg {
        children := fs.ReadDir(msg.NodeID)  // I/O in Command
        return TreeChildrenLoadedMsg[FileID]{
            ParentID: msg.NodeID,
            Children: children,
        }
    }
```

**Virtualisierung:** Der Tree flacht den expandierten Baum intern auf eine sichtbare Sequenz ab. Nur Nodes im Viewport ± Overscan landen im VTree:

```
RootIDs: [A, B, C]
A expanded → Children: [A1, A2]
A1 expanded → Children: [A1a]

Sichtbare Sequenz (flach):
  0: A       (depth=0, expanded=true)
  1: A1      (depth=1, expanded=true)
  2: A1a     (depth=2, expanded=false)
  3: A2      (depth=1, expanded=false)
  4: B       (depth=0, expanded=false)
  5: C       (depth=0, expanded=false)

VTree enthält nur Viewport-Ausschnitt dieser Sequenz.
```

### 18.3 Overlay-System

Menus, Tooltips, Dropdowns und Dialoge brauchen ein Konzept das aus dem normalen Layout-Flow ausbricht: **Overlays**.

#### Das Problem

Ein normales Widget ist durch seinen Parent geclipt — ein Dropdown-Button kann kein Menü rendern das über den Button-Container hinausgeht. Es braucht eine Ebene *über* dem Layout-Baum.

#### `Overlay`-Element

```go
// Overlay ist ein Element das außerhalb des normalen Layout-Flows rendert.
// Es erscheint in einem separaten Layer über allen anderen Widgets.
// Position ist relativ zum Fenster-Ursprung, nicht zum Parent.
type Overlay struct {
    // ID: Stabil über Frames. Overlays mit gleicher ID werden ge-diffed,
    // nicht neu erstellt. Wichtig für Animations-Kontinuität.
    ID      OverlayID

    // Anchor: Position des Overlays im Fenster (dp).
    // Typisch: Bounds des auslösenden Widgets.
    Anchor  Rect

    // Placement: Wie der Overlay relativ zum Anchor positioniert wird.
    Placement OverlayPlacement

    // Content: Der eigentliche Inhalt.
    Content Widget

    // Dismissable: Klick außerhalb schließt den Overlay.
    // Sendet DismissOverlayMsg{ID} in den User-Loop.
    Dismissable bool

    // Animation: Ein- und Ausblend-Verhalten.
    Animation OverlayAnimation
}

type OverlayID string

type OverlayPlacement uint8
const (
    PlacementBelow      OverlayPlacement = iota  // Unter dem Anchor
    PlacementAbove                               // Über dem Anchor
    PlacementRight
    PlacementLeft
    PlacementCenter                              // Zentriert im Fenster (für Dialoge)
    PlacementCursor                             // An der aktuellen Mausposition
)

// DismissOverlayMsg wird gesendet wenn ein Dismissable-Overlay
// durch Klick außerhalb oder Escape geschlossen wird.
type DismissOverlayMsg struct {
    ID OverlayID
}
```

#### Overlays im View-Baum

Overlays werden im `view`-Return-Wert deklariert — sie sind Teil des VTree, aber der Renderer platziert sie in einem eigenen Layer:

```go
func view(m Model) ui.Element {
    return ui.Stack(
        // Normaler UI-Inhalt:
        ui.Column(
            myToolbar(m),
            myContent(m),
        ),

        // Overlay — nur sichtbar wenn m.MenuOpen:
        ui.When(m.MenuOpen,
            ui.Overlay{
                ID:          "file-menu",
                Anchor:      m.MenuAnchorRect,
                Placement:   ui.PlacementBelow,
                Dismissable: true,
                Content:     fileMenu(m),
                Animation:   ui.OverlayFadeScale,
            },
        ),
    )
}
```

Das Overlay lebt im VTree — Diff, Animation und Dismissal-Logik folgen denselben Regeln wie alle anderen Widgets. Kein imperatives "openMenu()" / "closeMenu()".

#### Input-Capture für Overlays

Wenn ein Overlay offen ist, bekommt es Input-Priorität:

```
Input-Dispatch-Reihenfolge (höchste Priorität zuerst):
  1. Offene Overlays (Z-Order, neueste zuerst)
  2. Normaler Hit-Test im Layout-Baum
```

Klick außerhalb aller Overlay-Bounds → `DismissOverlayMsg` für den obersten Dismissable-Overlay → User-Loop setzt `m.MenuOpen = false` → `view` gibt keinen Overlay zurück → Framework entfernt ihn.

### 18.4 Menu-Widget

Menu ist ein Overlay-Wrapper der das BuildNode-Pattern für Menüeinträge nutzt:

```go
type Menu struct {
    Items []MenuItem
}

type MenuItem struct {
    ID       string
    Label    string
    Icon     ImageID      // Optional
    Shortcut *Shortcut    // Optional — wird rechts angezeigt
    Disabled bool
    // Submenu: öffnet einen kaskadierten Overlay
    Submenu  *Menu
    // Separator: zeigt eine Trennlinie (Label/Icon werden ignoriert)
    Separator bool
}

// MenuSelectedMsg wird gesendet wenn ein Item ausgewählt wird.
type MenuSelectedMsg struct {
    MenuID string   // ID des Menu-Overlays
    ItemID string
}
```

**Kaskadierte Submenus** sind Overlays mit `PlacementRight` auf dem Parent-MenuItem. Jedes Submenu ist ein eigener `Overlay`-Eintrag im VTree mit einer eigenen `OverlayID` — Dismiss-Logik und Diff funktionieren automatisch.

### 18.5 Weitere Overlay-Typen

Das Overlay-System ist generisch — Tooltip, Dialog und Popover nutzen dieselbe Infrastruktur:

```go
// Tooltip: Dismissable=false, PlacementAbove/Below, kurze Verzögerung
// Dialog:  PlacementCenter, Dismissable=true (oder false für modale Dialoge),
//          Backdrop-Dimming via PushOpacity auf dem Content-Layer
// Popover: Wie Dropdown, aber mit beliebigem Content-Widget

// Modal-Dialog Beispiel:
ui.Overlay{
    ID:          "confirm-delete",
    Placement:   ui.PlacementCenter,
    Dismissable: false,   // Explizite Bestätigung nötig
    Content:     confirmDialog(m),
}
```

**Modale Overlays** (Dismissable=false) blockieren Input zum darunter liegenden Layer vollständig — kein Hit-Test im normalen Layout-Baum solange der Overlay offen ist.

---

## 19. DynamicDataset — Länge unbekannt

### 19.1 Das Problem

`VirtualList` (§6.4) und `Tree` (§18.2) setzen voraus dass die Gesamtlänge bekannt ist:

```go
ItemCount int   // Was wenn ich das nicht weiß?
RootIDs   []ID  // Was wenn die erste Seite noch nicht geladen ist?
```

Das versagt bei:
- Cursor-basierten Datenbank-Queries (`LIMIT 50 OFFSET ?` ohne `COUNT(*)`)
- Echtzeit-Streams (neue Nachrichten kommen oben oder unten)
- Suchresultaten (Gesamtanzahl erst nach dem ersten Request bekannt)
- Infinite Scroll (es gibt kein "Ende")

### 19.2 Das `Dataset[ID]`-Interface

```go
// Dataset[ID] abstrahiert über alle Längen-Szenarien.
// Ersetzt ItemCount int und RootIDs []ID in VirtualList und Tree.
type Dataset[ID comparable] interface {
    // Len gibt die bekannte Länge zurück.
    // -1 = Länge unbekannt (noch nicht geladen oder nie bekannt).
    Len() int

    // Get gibt das Item an Index i zurück.
    // loaded=false → Item ist noch nicht verfügbar (wird gerade geladen).
    // Das Widget zeigt dann einen Skeleton/Placeholder für diesen Slot.
    Get(index int) (id ID, loaded bool)
}

// DatasetSlot beschreibt den Zustand eines einzelnen Index.
// Intern genutzt von Dataset-Implementierungen.
type DatasetSlot[ID comparable] struct {
    ID     ID
    State  SlotState
}

type SlotState uint8
const (
    SlotLoaded  SlotState = iota  // ID ist verfügbar
    SlotLoading                   // Anfrage läuft
    SlotError                     // Laden fehlgeschlagen
)
```

`VirtualList` und `Tree` erhalten ein `Dataset[ID]` statt `ItemCount`/`RootIDs`:

```go
type VirtualList struct {
    Dataset    Dataset[int]   // int-Index als ID für einfache Listen
    BuildItem  func(index int, loaded bool) Widget
    // ... Rest unverändert
}

type Tree[ID comparable] struct {
    Dataset    Dataset[ID]
    Children   func(id ID) Dataset[ID]  // auch Children können dynamisch sein
    BuildNode  BuildNode[ID]
    // ... Rest unverändert
}
```

`BuildItem`/`BuildNode` bekommt jetzt `loaded bool` — das Widget entscheidet selbst wie es einen ungeladenen Slot darstellt (Skeleton, Spinner, leer).

### 19.3 Eingebaute Dataset-Implementierungen

#### `SliceDataset[ID]` — statisch, Länge bekannt

```go
// Wrapper um eine Slice — Länge sofort bekannt, alle Items sofort geladen.
// Drop-in-Ersatz für den alten ItemCount + BuildItem-Ansatz.
type SliceDataset[ID comparable] struct {
    Items []ID
}

func (d *SliceDataset[ID]) Len() int                    { return len(d.Items) }
func (d *SliceDataset[ID]) Get(i int) (ID, bool)        { return d.Items[i], true }
```

#### `PagedDataset[ID]` — paginiert, Länge eventuell bekannt

```go
// PagedDataset verwaltet geladene Seiten und triggert Nachladen
// via RequestMsg wenn ungeladene Slots sichtbar werden.
type PagedDataset[ID comparable] struct {
    // TotalCount: -1 wenn noch unbekannt.
    // Wird nach dem ersten Load gesetzt.
    TotalCount int

    // PageSize: Anzahl Items pro Seite.
    PageSize int

    // pages: intern — geladene Seiten
    pages map[int][]ID  // pageIndex → []ID
}

func NewPagedDataset[ID comparable](pageSize int) *PagedDataset[ID]

func (d *PagedDataset[ID]) Len() int
func (d *PagedDataset[ID]) Get(index int) (ID, bool)

// SetPage fügt eine geladene Seite ein.
// Aufgerufen aus update wenn die Daten ankommen.
func (d *PagedDataset[ID]) SetPage(pageIndex int, ids []ID, totalCount int)

// SetError markiert eine Seite als fehlgeschlagen.
func (d *PagedDataset[ID]) SetError(pageIndex int)
```

#### `StreamDataset[ID]` — Echtzeit, keine Gesamtlänge

```go
// StreamDataset für append-only Streams (Chat, Log, Feed).
// Len() gibt immer -1 zurück — kein Ende bekannt.
// Neue Items werden via Prepend (oben) oder Append (unten) hinzugefügt.
type StreamDataset[ID comparable] struct {
    items []ID
    mode  StreamMode
}

type StreamMode uint8
const (
    StreamAppend  StreamMode = iota  // Neue Items unten (Log, Chat-History)
    StreamPrepend                    // Neue Items oben (Social Feed, Inbox)
)

func NewStreamDataset[ID comparable](mode StreamMode) *StreamDataset[ID]

func (d *StreamDataset[ID]) Len() int              { return -1 }  // nie bekannt
func (d *StreamDataset[ID]) Get(i int) (ID, bool)

func (d *StreamDataset[ID]) Append(ids ...ID)
func (d *StreamDataset[ID]) Prepend(ids ...ID)
func (d *StreamDataset[ID]) Len() int
```

### 19.4 Wo leben Dataset-Instanzen?

**Im User-Model** — nicht in `WidgetState`. Das Dataset *ist* Applikations-State: es enthält geladene IDs, Seitenstatus, Fehlerzustände. Der User-Loop mutiert es via `update`:

```go
type Model struct {
    Messages *StreamDataset[MessageID]
    Contacts *PagedDataset[ContactID]
}

func update(m Model, msg Msg) Model {
    switch msg := msg.(type) {

    case ContactsPageLoadedMsg:
        m.Contacts.SetPage(msg.Page, msg.IDs, msg.Total)

    case NewMessageMsg:
        m.Messages.Prepend(msg.ID)
    }
    return m
}
```

**`PagedDataset` und `StreamDataset` sind pointer-basiert** — sie werden nie kopiert, sondern via Pointer im Model gehalten. `update` gibt dasselbe Model zurück (mit mutation am Pointer-Ziel), nicht eine Deep Copy. Das ist eine bewusste Ausnahme vom "flaches Model"-Prinzip für große Datenmengen — analog zu §3.1 ("große Daten nur als Referenzen/Interfaces").

### 19.5 Load-Trigger: Vom Widget in den User-Loop

Wenn `VirtualList` einen ungeladenen Slot sichtbar macht, muss jemand das Laden anstoßen. Das Widget triggert via `ctx.Send` — der User-Loop entscheidet ob und wie er lädt:

```go
// Automatisch vom VirtualList gesendet wenn ein ungeladener
// Slot in den Viewport scrollt:
type DatasetLoadRequestMsg struct {
    WidgetUID  UID
    PageIndex  int   // welche Seite wird benötigt
    StartIndex int   // erster ungeladener Index
    EndIndex   int   // letzter ungeladener Index (inkl.)
}

// In update:
case DatasetLoadRequestMsg:
    if m.Contacts.IsPageLoading(msg.PageIndex) {
        return m  // bereits unterwegs
    }
    m.Contacts.SetLoading(msg.PageIndex)
    return m, func() Msg {
        // I/O in Command (§3.6):
        ids, total, err := db.LoadContacts(msg.PageIndex, pageSize)
        if err != nil {
            return ContactsPageErrorMsg{Page: msg.PageIndex, Err: err}
        }
        return ContactsPageLoadedMsg{Page: msg.PageIndex, IDs: ids, Total: total}
    }
```

Der Framework sendet `DatasetLoadRequestMsg` nur einmal pro Seite — nicht bei jedem Frame. Er trackt intern welche Seiten bereits angefragt wurden (via `SlotState.Loading`).

### 19.6 Vollständiges Beispiel: Paginierte Kontaktliste

```go
// Model
type Model struct {
    Contacts *PagedDataset[ContactID]
    ContactDetails map[ContactID]Contact  // geladene Daten
}

func initialModel() Model {
    return Model{
        Contacts:       NewPagedDataset[ContactID](50),
        ContactDetails: make(map[ContactID]Contact),
    }
}

// update
func update(m Model, msg app.Msg) Model {
    switch msg := msg.(type) {
    case ContactsPageLoadedMsg:
        m.Contacts.SetPage(msg.Page, msg.IDs, msg.Total)
        for _, c := range msg.Contacts {
            m.ContactDetails[c.ID] = c
        }
    }
    return m
}

// view
func view(m Model) ui.Element {
    return ui.VirtualList{
        Dataset: m.Contacts,
        BuildItem: func(index int, loaded bool) ui.Widget {
            if !loaded {
                return SkeletonRow{}  // Placeholder während Laden
            }
            id, _ := m.Contacts.Get(index)
            contact := m.ContactDetails[id]
            return ContactRow{Contact: contact}
        },
        NodeHeight: 56,
    }
}
```

Kein manuelles Paginierungs-Tracking im User-Model. Kein "welche Seite bin ich gerade?". Der `PagedDataset` verwaltet das intern — der User-Loop reagiert nur auf Load-Requests und fügt Ergebnisse ein.

---

## 20. Rich Text & Texteditierung

### 20.1 Einordnung: Vier Ebenen

Rich Text ist kein einzelnes Feature sondern ein Spektrum — von einfach gestyltem Text bis hin zu einem vollständigen Dokument-Editor. Das Framework deckt Ebenen 1–2 ab; Ebene 3 ist ein eigenständiges Widget-Paket; Ebene 4 ist der Surface-Slot-Pfad (§8).

```
Ebene 1  TextLayout          Bereits §6.2 — single-style, MeasureText
Ebene 2  RichText            Dieses Kapitel — gemischte Spans, read-only
Ebene 3  RichTextEditor      Separates Paket — Cursor, Selection, Undo/Redo
Ebene 4  External Surface    §8 — Browser-Engine, CodeMirror, vollst. HTML/CSS
```

### 20.2 Das Span-Modell

Rich Text besteht aus `TextSpan`-Runs — zusammenhängende Textsegmente mit einheitlichem Styling. Mehrere Spans bilden einen `Paragraph`, mehrere Paragraphen ein `RichText`-Widget.

```go
// TextSpan: ein gestylter Run innerhalb eines Paragraphs.
type TextSpan struct {
    Text  string

    // Style überschreibt den Paragraph-Default für diesen Span.
    // Zero-Value = erbt vom Paragraph.
    Style SpanStyle
}

type SpanStyle struct {
    FontFamily string      // leer = Paragraph-Default
    Size       float32     // 0 = Paragraph-Default
    Weight     FontWeight  // 0 = Paragraph-Default
    Italic     bool
    Underline  bool
    Strikethrough bool

    Color      Color       // Zero-Value = Paragraph-Default
    Background Color       // Zero-Value = transparent (kein Highlight)

    // Link: wenn gesetzt, ist dieser Span ein klickbarer Link.
    // Sendet LinkClickedMsg{Href} via ctx.Send wenn angeklickt.
    Link       string
}

// Paragraph: eine Texteinheit mit Block-Level-Eigenschaften.
type Paragraph struct {
    Spans []TextSpan

    // Block-Level-Stil:
    Alignment   TextAlignment   // Start, Center, End, Justify
    LineHeight  float32         // Multiplikator; 0 = 1.2 (Default)
    SpaceBefore float32         // dp Abstand vor dem Paragraph
    SpaceAfter  float32         // dp Abstand nach dem Paragraph

    // Einrückung:
    Indent      float32         // dp, erste Zeile
    HangingIndent float32       // dp, Folgezeilen (negatives Indent)

    // Fallback-Style für alle Spans ohne expliziten Wert:
    DefaultStyle SpanStyle
}

type TextAlignment uint8
const (
    TextAlignStart   TextAlignment = iota  // LTR: links, RTL: rechts
    TextAlignEnd
    TextAlignCenter
    TextAlignJustify
)
```

### 20.3 TextLayout-Pipeline

`go-text/typesetting` übernimmt die gesamte Layout-Arbeit — das Framework orchestriert nur:

```
[]Paragraph
    │
    ▼ 1. Pro Span: ShapingRun erstellen
    │   (Font + Size + Script + Direction aus BiDi-Analyse)
    │
    ▼ 2. go-text/typesetting/shaping: Shape(runs, maxWidth)
    │   → gemischte Glyph-Runs, Line-Breaking (UAX #14), BiDi-Reorder
    │
    ▼ 3. Cluster-Map aufbauen
    │   GlyphID + Advance + Cluster-Index → für Cursor-Positionierung
    │
    ▼ 4. RichTextLayout (gecacht in WidgetState)
        Zeilen + Positionen + Glyph-Atlas-Referenzen
```

**Caching:** Das Layout ist teuer (Shaping + Line-Breaking). Es wird in `RichTextState` gecacht und nur bei Änderung der Spans oder der verfügbaren Breite neu berechnet — nicht bei jedem Frame.

```go
type RichTextState struct {
    layout     richTextLayout  // gecachtes Ergebnis
    layoutFor  layoutCacheKey  // unter welchen Bedingungen berechnet
}

type layoutCacheKey struct {
    spansHash uint64   // schneller Hash der Span-Inhalte
    maxWidth  float32
    dpr       float32
}
```

### 20.4 RichText-Widget (Ebene 2)

```go
// RichText: read-only, rich-formatted Text.
// Für editierbaren Text: RichTextEditor (Ebene 3, separates Paket).
type RichText struct {
    Paragraphs []Paragraph

    // MaxWidth: 0 = Constraint-Breite aus Layout-System (Default)
    MaxWidth float32

    // SelectionEnabled: Text kann selektiert und kopiert werden,
    // aber nicht editiert. Sendet TextSelectedMsg wenn Selektion ändert.
    SelectionEnabled bool
}

// Msg die RichText senden kann:
type LinkClickedMsg struct {
    Href string
}
type TextSelectedMsg struct {
    Text string  // Der selektierte Klartext
}
```

### 20.5 Inline-Widgets

Ein wichtiger Sonderfall: Inline-Widgets — nicht-Text-Elemente die im Textfluss mitschwimmen (Inline-Images, Custom-Badges, Emoji-Ersatz durch Bitmaps).

```go
// InlineWidget bettet ein Widget in den Textfluss ein.
// Breite und Höhe werden vom Widget selbst bestimmt (via Intrinsic-Messung).
// Baseline-Alignment: Unterkante des Inline-Widgets sitzt auf der Textbaseline.
type InlineWidget struct {
    Widget  Widget
    Baseline float32  // 0 = Unterkante auf Baseline; positiv = höher
}

// InlineWidget kann als Span-Alternative in einem Paragraph genutzt werden:
type ParagraphContent interface{ isParagraphContent() }

func (TextSpan) isParagraphContent()    {}
func (InlineWidget) isParagraphContent() {}

// Paragraph mit gemischtem Content:
type Paragraph struct {
    Content []ParagraphContent  // TextSpan oder InlineWidget
    // ... Block-Level-Properties wie zuvor
}
```

### 20.6 RichTextEditor (Ebene 3 — separates Paket)

Der Editor ist ein eigenständiges Paket (`lux/richtext`) das `RichText` als Basis nutzt und Editierbarkeit hinzufügt. Er gehört nicht in den Framework-Kern weil sein `WidgetState` erheblich schwerer ist und seine Abhängigkeiten (Clipboard, IME, Undo-Stack) den Kern unnötig belasten würden.

```go
// RichTextEditor: editierbares Rich-Text-Widget.
// Paket: lux/richtext
type RichTextEditor struct {
    // Value: aktueller Dokument-Inhalt.
    // Wird nicht im WidgetState gehalten — gehört ins User-Model.
    Value    Document

    // OnChange: wird via ctx.Send gesendet wenn der Inhalt sich ändert.
    OnChange DocumentChangedMsg

    // Toolbar: optionale eingebettete Formatierungs-Toolbar.
    Toolbar *EditorToolbar

    // ReadOnly: Editor akzeptiert keine Eingaben (aber Selection/Copy).
    ReadOnly bool
}
```

**Was in `RichTextEditorState` lebt (WidgetState, framework-intern):**

```go
type RichTextEditorState struct {
    // Cursor & Selection:
    cursor    CursorPosition   // Paragraph + Span + Offset
    selection Selection        // Anchor + Focus, nil wenn keine Selektion

    // Undo-Stack (lebt im WidgetState, nicht im User-Model):
    // Begründung: Undo-History ist UI-State, nicht Applikations-State.
    // Der User-Loop bekommt nur das finale Dokument via OnChange.
    undoStack []DocumentEdit
    redoStack []DocumentEdit

    // IME-Composing:
    composing    bool
    composeText  string
    composeRange [2]int

    // Layout-Cache (wie RichText):
    layout richTextLayout
}
```

**`Document` als User-Model-Typ:**

```go
// Document ist der serialisierbare Dokument-Inhalt.
// Lebt im User-Model — kann persistiert werden (§3.4).
type Document struct {
    Paragraphs []Paragraph
}

// DocumentChangedMsg wird gesendet wenn der User den Inhalt ändert.
// Das User-Model ersetzt seinen Document-Wert damit.
type DocumentChangedMsg struct {
    Document Document
}
```

**Warum lebt der Undo-Stack im `WidgetState` und nicht im User-Model?**
Undo-History ist UI-State: sie gehört zum Editor-Widget, nicht zur Applikationslogik. Ein "Undo" in einem Text-Feld sollte nicht das gesamte App-Model zurückrollen. Der User-Loop bekommt nur das fertige `Document` via `OnChange` — was er daraus macht (speichern, validieren, weiterverarbeiten) liegt bei ihm.

### 20.7 Externe Rendering-Grenze (Ebene 4)

Für Anwendungsfälle die über das hinausgehen was `go-text/typesetting` leisten kann — vollständiges HTML/CSS-Rendering, komplexe mathematische Notation (LaTeX), eingebettete PDF-Seiten — ist der Surface-Slot-Pfad (§8) der korrekte Weg:

```
Benötigt man...                           → Lösung
─────────────────────────────────────────────────────
Fett, Kursiv, Links, Inline-Bilder       → RichText (Ebene 2)
Vollständiger Texteditor                  → RichTextEditor (Ebene 3)
HTML mit CSS (z.B. Markdown-Preview)      → WebView als Surface-Slot
LaTeX / MathML                            → External Renderer als Surface-Slot
PDF-Seiten                                → External Renderer als Surface-Slot
Code-Editor mit LSP                       → CodeMirror/Monaco als Surface-Slot
                                            oder nativer Code-Editor (Ebene 3+)
```

Die Grenze ist klar: alles was `go-text/typesetting` + die Canvas-API leisten können, bleibt im Framework. Was einen vollständigen Browser-Engine oder spezialisierten Renderer erfordert, dockt als Surface-Slot an.

---

## 21. Implementierungs-Leitfaden

### 21.1 Dependency-Graph der internen Packages

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

### 21.2 Implementierungs-Reihenfolge

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

### 21.3 Testbarkeit der Kern-Invarianten

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

### 21.4 Empfohlene Entwicklungs-Milestones

| Milestone | Inhalt | Kriterium |
|-----------|--------|-----------|
| **M1** Fenster | Stufe 1 | Schwarzes Fenster öffnet und schließt |
| **M2** Hello World | Stufe 2 | Text + Button rendern |
| **M3** Counter | Stufe 3 | Anhang-B-Beispiel läuft vollständig |
| **M4** Themed | Stufe 4 | Dark/Light-Switch, Hover-Animation |
| **M5** Alpha | Stufe 5 ohne A11y | VirtualList, Tree, RichText |
| **M6** Beta | Stufe 5 komplett | A11y, DRM/KMS, gogpu-Tag |
| **v1.0** | Alle §§ | Stable API, alle Plattformen grün |

### 21.5 Was bewusst nicht in v1.0 ist

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

*RFC-001 — Draft. Feedback und Änderungsvorschläge bitte als Issue gegen dieses Dokument.*
