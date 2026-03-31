# RFC-002 — lux: Interaction & Layout

**Repository:** `github.com/timzifer/lux`
**Status:** Integriert
**Version:** 0.1.0
**Datum:** 2026-03-18
**Zuletzt abgeglichen:** 2026-03-31
**Abhängig von:** RFC-001 (Core Architecture)
**Folge-RFC:** RFC-003 (Widget Catalogue & Theme)

---

### Implementierungsstatus

| Abschnitt | Status | Anmerkung |
|-----------|--------|-----------|
| §1 Animations-System | ✅ Integriert | Kern + SpringAnim, AnimGroup/Seq, CubicBezier, AnimationID |
| §1.3 Animator-Interface | ✅ Integriert | `Animator` Interface + `Reconciler.TickAnimators` |
| §1.4 `Anim[T]` | ✅ Integriert | `anim/anim.go` — `float32`/`float64` + `LerpAnim[T]` für alle draw-Typen |
| §1.4 `Interpolatable` Constraint | ✅ Integriert | `LerpFunc[T]`/`LerpAnim[T]` Pattern; `LerpColor`, `LerpPoint`, `LerpSize`, `LerpRect`, `LerpCornerRadii` in `draw/lerp.go` |
| §1.5 `SpringAnim[T]` | ✅ Integriert | `anim/spring.go` — Feder-Dämpfer mit SpringSpec, Presets: Gentle, Snappy, Bouncy |
| §1.6 `MotionSpec` im Theme | ✅ Integriert | `DurationEasing{Duration, Easing}` pro Slot: Standard (250ms OutCubic), Emphasized (400ms InOutCubic), Quick (100ms OutExpo) |
| §1.7 Easing-Funktionen | ✅ Integriert | Linear, OutCubic, InCubic, InOutCubic, OutExpo, CubicBezier |
| §1.8 AnimationID / SetTargetWithID | ✅ Integriert | `AnimationID`, `AnimationEnded`, `SetTargetWithID()`, `anim.SendFunc` Wiring |
| §1.9 AnimGroup / AnimSeq | ✅ Integriert | `anim/group.go` — `AnimGroup` (parallel), `AnimSeq` (sequential mit onDone-Hooks) |
| §1.10 CubicBezier | ✅ Integriert | `CubicBezier(x1,y1,x2,y2)` CSS-kompatibel mit Newton-Raphson |
| §2 Input-System | ✅ Integriert | Typisierter `Key uint32`, `ModifierSet` Bitfield, Touch, MouseEnter/Leave |
| §2.2 KeyMsg, MouseMsg, ScrollMsg | ✅ Integriert | Typisiert: `Key uint32`, `ModifierSet`, `ScrollMsg.Precise` |
| §2.2 TouchMsg | ✅ Integriert | `TouchMsg` mit TouchID, Phase, Force |
| §2.2 TextInputMsg → CharMsg | ✅ Integriert | Als `CharMsg` statt `TextInputMsg` |
| §2.2 IME Compose-Window | ✅ Integriert | `IMEComposeMsg`, `IMECommitMsg`, `SetIMECursorRect` auf allen Plattformen |
| §2.3 Focus-Management | ✅ Integriert | `FocusManager`, `Focusable`-Interface, Tab-Order aus Layout |
| §2.3 FocusGainedMsg/FocusLostMsg | ✅ Integriert | `ui/focus.go` |
| §2.4 Hit-Testing | ✅ Integriert | `internal/hit/hit.go` + `Interactor` (index-safe) |
| §2.5 Keyboard-Shortcuts | ✅ Integriert | `Shortcut`, `ShortcutMsg`, `WithShortcut`, `PlatformShortcut` |
| §2.6 Input-Dispatch via RenderCtx.Events | ✅ Integriert | `Dispatcher` in `ui/dispatch.go` |
| §2.7 Cursor-Management | ✅ Integriert | `CursorKind`, `Cursable`-Interface, `SetCursor` in Platform |
| §2.8 Global Handler Layer | ✅ Integriert | `GlobalHandler`, `WithGlobalHandler`, `RegisterHandlerMsg` |
| §3 Scroll & Kinetic Scrolling | ✅ Integriert | `KineticScroll` mit Friction, Rubber-Band, Velocity-Tracking |
| §3.4 ScrollSpec im Theme | ✅ Integriert | Friction, Overscroll, TrackWidth, ThumbRadius |
| §4 Layout-System | ✅ Integriert | |
| §4.2 Constraints-Modell | ✅ Integriert | `ui/constraints.go` |
| §4.4 Flexbox-Layout | ✅ Integriert | `ui/layout/flex.go` — CSS-Spec-konform: Direction, Justify, Align, Gap, FlexWrap, FlexBasis, FlexGrow/Shrink, AlignContent, Order |
| §4.5 Grid-Layout | ✅ Integriert | `ui/layout/grid.go` — CSS-Spec-konform: Track-Sizing, fr-Units, Repeat, Gap, Span, Auto-Placement |
| §4.11 CSS Table Layout | ✅ Integriert | `ui/layout/table.go` — HTML-Spec-konformes CSS Table Layout (Fixed + Auto Algorithmus) |
| §4.5 Stack | ✅ Integriert | |
| §4.5 Padding/SizedBox/Expanded | ✅ Integriert | |
| §4.3 Layout-Interface (Custom Layouts) | ✅ Integriert | `ui/layout.go` — `Layout`, `LayoutCtx`, `CustomLayout()`, `Size` |
| §4.6 RTL-Layout-Spiegelung (i18n) | ✅ Integriert | `Insets.Resolve(dir)`, `InlineInsets`, `BlockInsets`, `LogicalInsets`, `LayoutDirection` |
| §4.9 Layout-Cache | ✅ Integriert | `ui/layout_cache.go` — `LayoutCache` mit Store/IsValid/Invalidate |
| §4.10 Insets-Typ (Start/End) | ✅ Integriert | `Start`/`End` in `Insets`, `Resolve(dir)` für physische Auflösung |
| §5 Datenbasierte Widgets | ✅ Integriert | Tree, Overlay, DynamicDataset |
| §5.2 Tree-Widget | ✅ Integriert | `ui/tree.go` mit Expand/Collapse, Animation, Selection |
| §5.3 Overlay-System | ✅ Integriert | `Overlay` Element mit Anchor, Placement, Dismissable, Animation |
| §6 DynamicDataset | ✅ Integriert | `ui/data/paged_dataset.go` — Page-basierte Lazy-Loading-Datenquelle mit SlotState, PageProvider |

---

## Inhaltsverzeichnis

1. [Animations-System](#1-animations-system)
2. [Input-System](#2-input-system)
3. [Scroll & Kinetic Scrolling](#3-scroll--kinetic-scrolling)
4. [Layout-System](#4-layout-system)
5. [Datenbasierte Widgets & Overlay-System](#5-datenbasierte-widgets--overlay-system)
6. [DynamicDataset — Länge unbekannt](#6-dynamicdataset--länge-unbekannt)

---

## 1. Animations-System

### 1.1 Designziele

Das Animations-System muss in die Elm-Architektur passen — strukturell, nicht nur konzeptuell:

- **Kein Goroutinen-Timer**: Animationen laufen nicht in Goroutinen. Kein `time.Sleep`, kein `time.After`, kein Channel-basierter Tick der in `Send` mündet.
- **Deterministisch & testbar**: Eine Animation mit `dt = 16ms` ist überall identisch. Tests brauchen keine echte Uhr.
- **Deklarativ**: Widget deklariert *was* animiert werden soll und *wohin* — das Framework entscheidet *wann* der nächste Frame kommt.
- **Usercode bleibt ruhig**: Der User-Loop sieht keine `FrameMsg` pro Frame. Animationen laufen framework-intern. Optional kann ein Widget am Ende einer Animation eine `Msg` in den User-Loop einspeisen.

### 1.2 Architektur-Überblick

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

### 1.3 Das Animator-Interface

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

### 1.4 Der generische `Anim[T]`-Typ

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

### 1.5 Spring-Animationen

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

### 1.6 MotionSpec im Theme (Verbindung zu §5.2)

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

### 1.7 Vollständiges Widget-Beispiel

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

### 1.8 Zwei Tiers: interne vs. user-initiierte Animationen

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

### 1.9 Sequenzen & Parallelanimationen

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

### 1.10 Testbarkeit

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

## 2. Input-System

### 2.1 Architektur-Überblick

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

### 2.2 Msg-Typen

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

#### IME Compose-Window

Für CJK-Eingabe (Chinesisch, Japanisch, Koreanisch) und andere Kompositions-Methoden muss das Framework ein IME Compose-Fenster unterstützen. Das ist ein natives OS-Fenster, das Kandidaten anzeigt und vom Benutzer gesteuert wird.

```go
// IMEComposeMsg wird während einer aktiven IME-Komposition gesendet.
// Das Widget zeigt den unfertigen Text inline an (unterstrichen).
type IMEComposeMsg struct {
    // Text ist der aktuelle Kompositions-Text (z.B. "にほ" bevor "日本" bestätigt wird).
    Text string
    // Cursor ist die Position innerhalb des Kompositions-Texts.
    Cursor int
    // Selection markiert den aktuell ausgewählten Kandidaten-Bereich.
    SelectionStart, SelectionEnd int
}

// IMECommitMsg wird gesendet wenn die IME-Komposition abgeschlossen ist.
// Identisch mit TextInputMsg — expliziter Typ für Klarheit im Widget-Code.
type IMECommitMsg = TextInputMsg
```

**Plattform-Pflichten:**
- Das Framework muss dem OS die **Cursor-Position in Bildschirmkoordinaten** mitteilen, damit das Kandidaten-Fenster korrekt positioniert wird (`Platform.SetIMECursorRect(rect Rect)`)
- `IMEComposeMsg` löst **keinen** `TextInputMsg` aus — erst `IMECommitMsg` nach Bestätigung
- Widgets die IME unterstützen (TextField, RichTextEditor) müssen Kompositions-Text visuell unterscheidbar rendern (typisch: Unterstreichung)
- Pro Plattform: GLFW `glfwSetPreeditCallback` (oder native API auf Cocoa/Win32/IBus)

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

### 2.3 Focus-Management

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

### 2.4 Hit-Testing

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

### 2.5 Keyboard-Shortcuts

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

### 2.6 Input-Dispatch im Widget

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

### 2.7 Cursor-Management

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

### 2.8 Global Handler Layer

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

### 2.9 Invarianten & Vertrag

- **Kein Input außerhalb des App-Loops.** `InputEvent`-Werte verlassen den Loop nie.
- **Events sind read-only.** `ctx.Events` ist eine Slice — Mutationen haben keinen Effekt.
- **Flat Dispatch, kein Bubbling.** Kein `event.StopPropagation()` — es gibt keine Propagation. Global Handler sind kein Bubbling; sie sitzen *vor* dem Dispatch, nicht danach.
- **Ein Widget empfängt immer vollständige Event-Sequenzen.** `TouchBegan` an ein Widget → alle folgenden `TouchMoved`/`TouchEnded` mit gleicher `TouchID` gehen an dasselbe Widget, auch außerhalb der Bounds. (Touch-Capture-Semantik.)
- **Global Handler sind synchron.** Sie laufen im App-Loop, dürfen `app.Send` aufrufen, aber keinen blockierenden I/O machen.

---

## 3. Scroll & Kinetic Scrolling

### 3.1 Einordnung

Kinetic Scrolling ist kein Framework-Konzept das überall eingebaut ist — es ist ein **Widget-Pattern**, das das Framework durch drei Bausteine ermöglicht:

- `ScrollMsg.Precise` (§13.2) — unterscheidet Trackpad (kinetic-fähig) von Mausrad (stepped)
- `Animator`-Interface (§12.3) — treibt den Deceleration-Tick frame-by-frame
- `MotionSpec.Scroll` im Theme (§5.2) — konfigurierbare Physik-Parameter

Ein `Scrollable`-Widget das diese Bausteine nutzt, bekommt Kinetic Scrolling out-of-the-box. Ein Widget das nur Mausrad-Scrolling braucht, ignoriert `Precise` und `Animator` vollständig.

### 3.2 Bewegungsmodell: Friction-Decay, nicht Spring

Kinetic Scrolling nutzt **exponentielle Abbremsung** (Friction-Decay), keinen Spring:

- Spring würde über den Zielwert hinausschwingen — falsch für normales Scrolling.
- Friction-Decay: Geschwindigkeit × Faktor pro Frame → asymptotische Annäherung an Stillstand.
- **Rubber-Banding** am Rand ist ein separater Spring — bewusst unterschiedliches Verhalten.

```
v(t) = v₀ × friction^(dt/frameTarget)

friction ∈ (0, 1) — typisch 0.95 bei 16ms-Frames
Stillstand wenn |v(t)| < settlingThreshold
```

### 3.3 `KineticScroll`-Typ

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

### 3.4 `ScrollSpec` im Theme

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

### 3.5 Feed-Logik im Detail

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

### 3.6 Overscroll & Rubber-Banding

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

### 3.7 Vollständiges Beispiel

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

### 3.8 User-Loop-Benachrichtigung (optional)

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

## 4. Layout-System

### 4.1 Einordnung & Designziele

Das Layout-System löst eine einzige Frage: **Welche Größe und Position bekommt jedes Widget?**

Designziele:
- **Constraint-basiert, nicht absolut** — Widgets deklarieren ihre Anforderungen, der Parent entscheidet.
- **Einmaliger Pass** — kein Layout-Thrashing durch gegenseitige Abhängigkeiten.
- **Erweiterbar** — eigene Layout-Algorithmen ohne Framework-Fork, via Interface.
- **Flexbox-kompatibel** — bekanntes Modell, gute Tooling-Unterstützung, kein CSS-Cascade-Overhead.

### 4.2 Das Constraint-Modell

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

### 4.3 Das Layout-Interface

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

### 4.4 Flexbox-Layout

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

### 4.5 Weitere eingebaute Layouts

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

### 4.6 RTL-Layout-Spiegelung (i18n)

Für RTL-Sprachen (Arabisch, Hebräisch, Farsi) muss das Layout-System die horizontale Richtung automatisch spiegeln. Das ist eine **API-Design-Entscheidung die jetzt getroffen werden muss** — Nachrüsten ist extrem teuer, weil es die gesamte Layout-API betrifft.

#### Prinzip: `Start`/`End` statt `Left`/`Right`

Die gesamte Layout-API verwendet **logische Richtungen** statt physischer:

```go
// Insets verwendet Start/End statt Left/Right.
// Start = links bei LTR, rechts bei RTL.
type Insets struct {
    Top    float32
    End    float32
    Bottom float32
    Start  float32
}

// Convenience-Konstruktoren:
func InsetStart(v float32) Insets     // nur Start-Padding
func InsetEnd(v float32) Insets       // nur End-Padding
func InsetSymmetric(h, v float32) Insets  // Start+End = h, Top+Bottom = v
func UniformInsets(v float32) Insets   // alle vier gleich
```

**Was automatisch gespiegelt wird (bei RTL-Locale):**
- `FlexRow` → Kinder fließen rechts-nach-links
- `JustifyStart` → rechts statt links
- `AlignStart` → rechts statt links
- `Insets.Start` → rechte Seite
- Icons neben Text (z.B. Pfeil in Button) → gespiegelt

**Was NICHT gespiegelt wird:**
- Fortschrittsbalken (immer links-nach-rechts)
- Telefonnummern, Timestamps
- Medien-Controls (Play/Pause-Buttons)
- `FlexColumn` (vertikale Achse ist richtungsneutral)
- Explizit physische Positionierung via `Stack` mit `Offset`

#### Locale-Propagation

```go
// Die Layout-Richtung wird aus der App-Locale abgeleitet
// und über den LayoutCtx propagiert.
type LayoutCtx struct {
    Constraints Constraints
    Direction   LayoutDirection  // LTR oder RTL, aus Locale abgeleitet
    Theme       Theme
    // ...
}

type LayoutDirection uint8
const (
    LayoutLTR LayoutDirection = iota
    LayoutRTL
)
```

Widgets die physische Richtungen brauchen, können `ctx.Direction` abfragen. Alle eingebauten Layouts respektieren `Direction` automatisch.

### 4.7 Spacing-Tokens

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

// RadiusScale: striktes 4dp-System.
// Benannte Slots statt freier Werte — Custom-Themes überschreiben
// die Slots, nicht individuelle Widget-Radien.
type RadiusScale struct {
    Input  float32  //  4 dp — Eingabefelder (scharf, präzise)
    Button float32  //  6 dp — Buttons (leicht gerundet)
    Card   float32  //  8 dp — Cards, Panels, Dialoge
    Pill   float32  // 999 dp — vollständig gerundete Tags/Badges
}

// Verwendung:
tokens := ctx.Theme.Tokens()
ui.Padding{
    Insets: ui.UniformInsets(tokens.Spacing.M),
    Child:  myContent,
}
```

### 4.8 Custom Layout

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

### 4.9 Layout-Cache & Invalidierung

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

### 4.10 Insets-Typ

```go
// Insets verwendet logische Richtungen (Start/End) statt physische (Left/Right).
// Start = links bei LTR, rechts bei RTL. Siehe §4.6 RTL-Layout-Spiegelung.
type Insets struct {
    Top, End, Bottom, Start float32  // dp
}

func UniformInsets(all float32) Insets
func SymmetricInsets(horizontal, vertical float32) Insets
func InlineInsets(start, end float32) Insets    // Start/End (horizontale Achse, richtungsabhängig)
func BlockInsets(top, bottom float32) Insets     // Top/Bottom (vertikale Achse, richtungsneutral)
```

---

## 5. Datenbasierte Widgets & Overlay-System

### 5.1 Datenbasierte Widgets: Das BuildNode-Pattern

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

### 5.2 Tree-Widget

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

### 5.3 Overlay-System

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

### 5.4 Menu-Widget

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

### 5.5 Weitere Overlay-Typen

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

## 6. DynamicDataset — Länge unbekannt ✅ Implementiert

### 6.1 Das Problem

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

### 6.2 Das `Dataset[ID]`-Interface

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

### 6.3 Eingebaute Dataset-Implementierungen

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

### 6.4 Wo leben Dataset-Instanzen?

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

### 6.5 Load-Trigger: Vom Widget in den User-Loop

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

### 6.6 Vollständiges Beispiel: Paginierte Kontaktliste

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


---

*RFC-002 — Draft. Feedback via GitHub Issues gegen `github.com/timzifer/lux`.*