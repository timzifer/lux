# RFC-004 — lux: HMI & Touch-Optimierung

**Repository:** `github.com/timzifer/lux`
**Status:** Teilweise integriert
**Version:** 0.1.0
**Datum:** 2026-03-19
**Zuletzt abgeglichen:** 2026-04-08
**Abhängig von:** RFC-001 (Core), RFC-002 (Interaction & Layout), RFC-003 (Widget Catalogue & Theme)

---

### Implementierungsstatus

| Abschnitt | Status | Anmerkung |
|-----------|--------|-----------|
| §1 Motivation & Scope | — | Kontext, kein Code |
| §2 Interaction-Profile | ✅ Integriert | `interaction/profile.go` — ProfileDesktop, ProfileTouch, ProfileHMI; RenderCtx-Propagation, Hover-Elimination, GestureConfig-Ableitung |
| §3 Gesture-Recognizer | ✅ Integriert | `input/gesture.go`, `ui/gesture.go` — Tap, LongPress, Pan, Pinch; Arena-basierte Disambiguierung |
| §4 Touch-Feedback & Bestätigung | ✅ Integriert | `ui/button/confirm.go` (ConfirmButton), `ui/button/hold.go` (HoldButton), `ui/button/ripple.go` (Ripple), `platform/haptics.go` (Haptics-API) |
| §5 On-Screen-Keyboard | ✅ Integriert | `ui/osk/` — OSKLayout, OSKAction, OSKKey, 4 Modi (Alpha, NumPad, Full, Condensed); DPI-aware Sizing; Framework-Overlay in `app/run.go` |
| §6 Spezialisierte Input-Widgets | ✅ Größtenteils integriert | NumericInput, Stepper, UnitInput, TimeInput, DateInput, RangeInput implementiert; DrumPicker fehlt |
| §7 HMI-Theme-Profil | ⏳ Wartend | |
| §8 Navigation & Layout-Patterns | ⏳ Wartend | |
| §9 Industrielle Anforderungen | ⏳ Wartend | |
| §10 Performance auf eingebetteter Hardware | ⏳ Wartend | |

---

## Inhaltsverzeichnis

1. [Motivation & Scope](#1-motivation--scope)
2. [Interaction-Profile](#2-interaction-profile)
3. [Gesture-Recognizer](#3-gesture-recognizer)
4. [Touch-Feedback & Bestätigung](#4-touch-feedback--bestätigung)
5. [On-Screen-Keyboard (OSK)](#5-on-screen-keyboard-osk)
6. [Spezialisierte Input-Widgets](#6-spezialisierte-input-widgets)
7. [HMI-Theme-Profil](#7-hmi-theme-profil)
8. [Navigation & Layout-Patterns](#8-navigation--layout-patterns)
9. [Industrielle Anforderungen](#9-industrielle-anforderungen)
10. [Performance auf eingebetteter Hardware](#10-performance-auf-eingebetteter-hardware)

---

## 1. Motivation & Scope

lux zielt u.a. auf den Einsatz auf Touch-basierten Maschinen-Interfaces (HMI). Diese Umgebungen unterscheiden sich fundamental von Desktop-Anwendungen:

- **Kein Mauszeiger** → kein Hover-State, keine Tooltips, kein Rechtsklick
- **Finger statt Maus** → unpräzise Eingabe, größere Touch-Targets, Handschuhe
- **Eingeschränkte Hardware** → schwache GPUs, niedrige Auflösungen, feste Display-Größen
- **Kritische Aktionen** → Maschinensteuerung erfordert explizite Bestätigung
- **Keine physische Tastatur** → On-Screen-Keyboard für jede Texteingabe
- **Schichtbetrieb** → Wechselnde Lichtverhältnisse, verschmutzte Displays

Dieses RFC definiert die Architektur-Erweiterungen, Theme-Anpassungen und spezialisierten Widgets die lux für diese Szenarien benötigt. Es erweitert bestehende Konzepte aus RFC-002 (Input-System) und RFC-003 (Widget-Katalog) — es ersetzt sie nicht.

### 1.1 Abgrenzung

Dieses RFC behandelt **nicht**:
- Spezifische Hardware-Treiber (DRM/KMS-Backend → RFC-001 §7)
- Accessibility im Desktop-Sinne (Screenreader → RFC-001 §11)
- Allgemeine Widget-Spezifikationen (→ RFC-003 §4)

---

## 2. Interaction-Profile

### 2.1 Motivation

Ein HMI-Button braucht andere Dimensionen als ein Desktop-Button. Ein Slider auf einem 7"-Touch-Panel muss sich anders verhalten als auf einem 27"-Monitor mit Maus. Statt separate Widget-Sets zu pflegen, definiert lux **Interaction-Profile** — Konfigurationsschichten, die das Verhalten und die Dimensionierung aller Widgets global anpassen.

### 2.2 Das InteractionProfile-Interface

```go
// InteractionProfile beschreibt die Interaktionseigenschaften
// der Zielumgebung. Lebt neben dem Theme, ist aber kein Theme-Token
// (es beeinflusst Layout, nicht Rendering).
type InteractionProfile struct {
    // PointerKind: Art der primären Eingabe.
    PointerKind PointerKind

    // MinTouchTarget: Minimale interaktive Fläche in dp.
    // Desktop: 24dp, Touch: 48dp, Glove: 64dp.
    MinTouchTarget float32

    // TouchTargetSpacing: Mindestabstand zwischen interaktiven
    // Elementen in dp. Verhindert Fehlbedienungen.
    // Desktop: 0dp, Touch: 8dp, Glove: 12dp.
    TouchTargetSpacing float32

    // HasHover: Ob Hover-States existieren.
    // false auf reinen Touch-Geräten.
    HasHover bool

    // HasPhysicalKeyboard: Ob eine physische Tastatur vorhanden ist.
    // false → OSK wird bei TextField-Focus geöffnet.
    HasPhysicalKeyboard bool

    // LongPressDuration: Dauer bis Long-Press ausgelöst wird.
    // Default: 500ms. HMI: 400ms (schneller Workflow).
    LongPressDuration time.Duration

    // DoubleTapInterval: Maximale Zeit zwischen zwei Taps
    // für Double-Tap-Erkennung.
    DoubleTapInterval time.Duration

    // DragThreshold: Minimale Bewegung in dp bevor ein Tap
    // zu einem Drag wird. Höher bei Touch (Fingerzittern).
    // Desktop: 4dp, Touch: 10dp.
    DragThreshold float32

    // DebounceInterval: Mindestzeit zwischen zwei akzeptierten
    // Taps auf dasselbe Element. Verhindert Doppelauslösung.
    // 0 = kein Debounce. HMI-Default: 200ms.
    DebounceInterval time.Duration

    // ScaleTypography: Faktor für globale Schrift-Skalierung.
    // 1.0 = Desktop-Default (13dp Body). HMI: 1.3–1.5.
    ScaleTypography float32
}

type PointerKind uint8
const (
    PointerMouse  PointerKind = iota  // Maus/Trackpad — präzise
    PointerFinger                     // Kapazitiver Touch — 7mm Kontaktfläche
    PointerGlove                      // Handschuh-Touch — ≥15mm Kontaktfläche
    PointerStylus                     // Stift — präzise, aber kein Hover
)
```

### 2.3 Vordefinierte Profile

```go
// ProfileDesktop: Standard-Desktop mit Maus und Tastatur.
var ProfileDesktop = InteractionProfile{
    PointerKind:         PointerMouse,
    MinTouchTarget:      24,
    TouchTargetSpacing:  0,
    HasHover:            true,
    HasPhysicalKeyboard: true,
    LongPressDuration:   500 * time.Millisecond,
    DoubleTapInterval:   400 * time.Millisecond,
    DragThreshold:       4,
    DebounceInterval:    0,
    ScaleTypography:     1.0,
}

// ProfileTouch: Kapazitiver Touchscreen ohne Tastatur.
var ProfileTouch = InteractionProfile{
    PointerKind:         PointerFinger,
    MinTouchTarget:      48,
    TouchTargetSpacing:  8,
    HasHover:            false,
    HasPhysicalKeyboard: false,
    LongPressDuration:   400 * time.Millisecond,
    DoubleTapInterval:   350 * time.Millisecond,
    DragThreshold:       10,
    DebounceInterval:    200 * time.Millisecond,
    ScaleTypography:     1.3,
}

// ProfileHMI: Industrielles Touch-Panel mit Handschuh-Bedienung.
var ProfileHMI = InteractionProfile{
    PointerKind:         PointerGlove,
    MinTouchTarget:      64,
    TouchTargetSpacing:  12,
    HasHover:            false,
    HasPhysicalKeyboard: false,
    LongPressDuration:   400 * time.Millisecond,
    DoubleTapInterval:   350 * time.Millisecond,
    DragThreshold:       14,
    DebounceInterval:    250 * time.Millisecond,
    ScaleTypography:     1.5,
}
```

### 2.4 Profil-Aktivierung

```go
app.Run(myModel, myView, myUpdate,
    app.WithTheme(theme.Slate),
    app.WithInteractionProfile(interaction.ProfileHMI),
)
```

Das Framework propagiert das Profil über `RenderCtx`:

```go
func (b Button) Render(ctx RenderCtx, state WidgetState) (Element, WidgetState) {
    profile := ctx.InteractionProfile
    // MinTouchTarget wird vom Layout-System erzwungen —
    // das Widget muss es nicht manuell beachten.
    // ...
}
```

### 2.5 Layout-Enforcement

Das Layout-System erzwingt `MinTouchTarget` automatisch für alle Widgets die `Focusable` implementieren:

```go
// Intern im Layout-Pass:
if widget.implements(Focusable) && profile.MinTouchTarget > 0 {
    constraints.MinWidth = max(constraints.MinWidth, profile.MinTouchTarget)
    constraints.MinHeight = max(constraints.MinHeight, profile.MinTouchTarget)
}
```

Widgets die visuell kleiner sein sollen (z.B. ein 12×12-Icon-Button), erhalten trotzdem die volle Touch-Fläche — das Layout expandiert den Hit-Test-Bereich, nicht das visuelle Rendering:

```go
// Widget rendert sich in seiner gewünschten Größe.
// Das Framework expandiert nur die Hit-Test-Region:
//
//   ┌─────────────────────┐  ← Hit-Test-Bereich (64×64)
//   │                     │
//   │    ┌───────────┐    │
//   │    │  Button    │    │  ← Visueller Bereich (40×40)
//   │    │           │    │
//   │    └───────────┘    │
//   │                     │
//   └─────────────────────┘
```

### 2.6 Hover-Elimination

Wenn `HasHover == false`, werden alle Hover-States im Framework eliminiert:

- `DrawCtx.Hovered` ist immer `false`
- `Surface.Hovered` wird im DrawFunc nie als Background gewählt
- Tooltips werden nicht gerendert
- `MouseMsg` mit `Action == MouseMove` wird nicht dispatcht

Informationen die bisher nur via Tooltip zugänglich waren, müssen über alternative Pfade bereitgestellt werden (→ §8.3 Info-Disclosure-Pattern).

---

## 3. Gesture-Recognizer

### 3.1 Motivation

RFC-002 §2.2 definiert das rohe `TouchMsg` mit Phasen (Began/Moved/Ended/Cancelled). Für HMI-Anwendungen braucht man höherwertige Gesten — Tap, Long-Press, Swipe, Pinch. Der Gesture-Recognizer transformiert Touch-Sequenzen in semantische Gesten-Msgs.

### 3.2 Architektur

```
TouchMsg (roh)
    │
    ▼
GestureRecognizer (framework-intern)
    │
    ├── TapMsg
    ├── DoubleTapMsg
    ├── LongPressMsg
    ├── SwipeMsg
    ├── DragMsg / DragEndMsg
    ├── PinchMsg
    └── TouchMsg (unverbraucht, an Widget weiter)
```

Der Recognizer lebt zwischen Platform-Input und Widget-Dispatch — er konsumiert `TouchMsg`-Sequenzen und emittiert Gesten-Msgs. Nicht erkannte Touch-Sequenzen werden unverändert weitergereicht.

### 3.3 Gesten-Msgs

```go
// TapMsg: Finger berührt und hebt ab innerhalb von
// DragThreshold und LongPressDuration.
type TapMsg struct {
    Pos   Point    // Position des Taps
    Count int      // 1 = Single-Tap, 2 = Double-Tap, etc.
}

// LongPressMsg: Finger ruht länger als LongPressDuration
// ohne DragThreshold zu überschreiten.
type LongPressMsg struct {
    Pos   Point
    Phase LongPressPhase
}
type LongPressPhase uint8
const (
    LongPressBegan     LongPressPhase = iota  // Schwelle erreicht
    LongPressEnded                             // Finger abgehoben
    LongPressCancelled                         // Finger bewegt / OS-Interrupt
)

// SwipeMsg: Schnelle lineare Bewegung über SwipeThreshold.
type SwipeMsg struct {
    Direction SwipeDirection
    Velocity  float32  // dp/s
    Start     Point
    End       Point
}
type SwipeDirection uint8
const (
    SwipeLeft  SwipeDirection = iota
    SwipeRight
    SwipeUp
    SwipeDown
)

// DragMsg: Langsame Bewegung über DragThreshold.
// Unterschied zu Swipe: Geschwindigkeit unter SwipeVelocityThreshold.
type DragMsg struct {
    Phase  DragPhase
    Start  Point     // Startposition
    Pos    Point     // Aktuelle Position
    Delta  Point     // Bewegung seit letztem Frame
}
type DragPhase uint8
const (
    DragBegan DragPhase = iota
    DragMoved
    DragEnded
    DragCancelled
)

// PinchMsg: Zwei-Finger-Geste. Scale ist relativ zum Start (1.0 = unverändert).
type PinchMsg struct {
    Phase  PinchPhase
    Center Point     // Mittelpunkt zwischen den Fingern
    Scale  float32   // >1.0 = Vergrößerung, <1.0 = Verkleinerung
    // Rotation: float32  // Radians — optional, nicht für HMI v1
}
type PinchPhase uint8
const (
    PinchBegan PinchPhase = iota
    PinchChanged
    PinchEnded
    PinchCancelled
)
```

### 3.4 Gesture-Arena (Disambiguation)

Wenn ein Widget sowohl Tap als auch Drag unterstützt, muss der Recognizer entscheiden welche Geste gemeint ist. Das Modell ist eine **Gesture-Arena** (wie Flutter):

1. Bei `TouchBegan` registrieren alle interessierten Recognizer ihren Anspruch
2. Während `TouchMoved` werden Recognizer disqualifiziert (z.B. Tap-Recognizer wenn DragThreshold überschritten)
3. Der letzte verbleibende Recognizer gewinnt und erhält die restlichen Events

```go
// Widget registriert Interesse via RenderCtx:
func (s mySlider) Render(ctx RenderCtx, state WidgetState) (Element, WidgetState) {
    ctx.HandleGesture(GestureKindDrag)  // "Ich will Drags"
    // ...
}
```

### 3.5 Palm Rejection

Auf industriellen Touchscreens ist versehentlicher Handballenkon­takt häufig. Der Recognizer implementiert Palm Rejection über:

- **Kontaktfläche**: Touch-Events mit `Force > 0.8` und gleichzeitig großer Kontaktfläche (falls von Hardware geliefert) werden als Handballen klassifiziert
- **Position**: Touches am äußersten Bildschirmrand (< 10dp vom Edge) werden verzögert und nur akzeptiert wenn keine gleichzeitigen Touches im Hauptbereich aktiv sind
- **Timing**: Ein neuer Touch der innerhalb von 50ms nach einem bestehenden Touch beginnt und > 100dp entfernt ist, wird als Handballen-Kandidat markiert

### 3.6 Debouncing

`InteractionProfile.DebounceInterval` wird vom Framework automatisch auf alle `TapMsg`-Auslösungen angewandt:

```go
// Framework-intern:
if timeSinceLastTap < profile.DebounceInterval {
    // TapMsg wird nicht dispatcht
    return
}
```

Das betrifft nur Taps — Drags, Swipes und Pinches haben kein Debouncing (sie sind continuous).

---

## 4. Touch-Feedback & Bestätigung

### 4.1 Visuelles Feedback

Da Hover-States auf Touch nicht existieren, ist unmittelbares Pressed-Feedback essenziell. Die Regeln:

1. **Pressed-State sofort sichtbar** — bei `TouchBegan`, nicht erst bei `TouchEnded`. Der User muss sehen, dass das System seine Berührung registriert hat.
2. **Kontrastreicher Pressed-State** — `Surface.Pressed` muss sich deutlich von `Surface.Elevated` abheben (nicht nur subtile Opacity-Änderung).
3. **Ripple-Effekt optional** — ein radialer Feedback-Effekt ab dem Berührungspunkt. Im HMI-Profil standardmäßig aus (Performance), per Theme aktivierbar.

### 4.2 Haptisches Feedback (Platform-API)

```go
// platform.Haptics ist ein optionales Interface das Backends implementieren können.
type Haptics interface {
    // Vibrate löst haptisches Feedback aus.
    // Auf Hardware ohne Vibrator ist dies ein No-Op.
    Vibrate(style HapticStyle)
}

type HapticStyle uint8
const (
    HapticLight  HapticStyle = iota  // Subtiler Tap-Feedback
    HapticMedium                     // Bestätigung
    HapticHeavy                      // Warnung/Fehler
    HapticError                      // Fehler-Vibration (doppelt, schnell)
)
```

Widgets lösen Haptics über `RenderCtx.Haptics()` aus — wenn die Platform es nicht unterstützt, ist der Aufruf ein No-Op.

### 4.3 Bestätigungsmuster für kritische Aktionen

Auf Maschinen-Interfaces gibt es Aktionen die nicht versehentlich ausgelöst werden dürfen (Motor starten, Werkzeug einfahren, Reset). lux bietet drei Abstufungen:

#### Stufe 1: Debounce (Standard)

Jeder Touch-Button hat `DebounceInterval` — versehentliche Doppel-Taps lösen nur einmal aus. Kein zusätzliches UI.

#### Stufe 2: ConfirmButton

Ein Button der eine explizite Zweischritt-Bestätigung erfordert:

```go
ui.ConfirmButton{
    Label:        "Motor starten",
    ConfirmLabel: "Bestätigen: Motor starten",
    Icon:         icon.Power,
    Variant:      ui.ButtonDanger,
    OnConfirm:    func() Msg { return StartMotorMsg{} },

    // Optionen:
    ConfirmTimeout: 3 * time.Second,  // Zurück zu Idle nach 3s ohne Bestätigung
    RequireRelease: true,             // Erst bei Finger-Abheben, nicht bei Touch-Down
}
```

**Ablauf:**
1. Erster Tap → Button wechselt in Confirm-State (andere Farbe, anderes Label)
2. Zweiter Tap (innerhalb `ConfirmTimeout`) → `OnConfirm` wird ausgelöst
3. Timeout ohne zweiten Tap → zurück zu Idle, kein Event

#### Stufe 3: HoldButton

Ein Button der gedrückt gehalten werden muss:

```go
ui.HoldButton{
    Label:        "Notfall-Stopp",
    HoldDuration: 2 * time.Second,
    OnComplete:   func() Msg { return EmergencyStopMsg{} },

    // Visuelles Feedback: Fortschrittsring der sich während
    // des Haltens füllt (Anim[float32], 0→1 über HoldDuration)
    ShowProgress: true,
}
```

**Ablauf:**
1. Finger berührt → Fortschrittsring startet
2. Finger hebt vor `HoldDuration` ab → Abbruch, kein Event, Ring animiert zurück
3. Finger bleibt für `HoldDuration` → `OnComplete` wird ausgelöst, Haptic-Heavy

---

## 5. On-Screen-Keyboard (OSK)

### 5.1 Architektur

Das OSK ist kein Widget im User-Tree — es ist ein **Framework-Overlay** das automatisch eingeblendet wird wenn ein Textfeld Fokus erhält und `HasPhysicalKeyboard == false`. Das verhindert, dass der User-Code sich um OSK-Management kümmern muss.

```
┌──────────────────────────────────────┐
│          App-Content                 │
│                                      │
│    ┌──────────────────────┐          │
│    │  fokussiertes Feld   │ ← sichtbar über OSK
│    └──────────────────────┘          │
│                                      │
├══════════════════════════════════════┤  ← OSK-Grenze
│                                      │
│          On-Screen-Keyboard          │
│                                      │
└──────────────────────────────────────┘
```

### 5.2 OSK-Layout-Shift

Wenn das OSK eingeblendet wird, reduziert sich der verfügbare Layout-Bereich. Das Framework:

1. Berechnet die OSK-Höhe (abhängig vom Keyboard-Layout)
2. Setzt die App-Content-Constraints auf `WindowHeight - OSKHeight`
3. Scrollt den fokussierten Input in den sichtbaren Bereich (wenn nötig)
4. Animiert den Übergang (Slide-Up, `MotionSpec.Normal`)

```go
// Framework-intern:
type OSKState struct {
    Visible   bool
    Height    Anim[float32]  // Animiert für sanften Übergang
    Layout    OSKLayout      // Welches Keyboard ist aktiv?
    Target    UID            // Welches Widget hat den OSK ausgelöst?
}
```

### 5.3 OSK-Layouts

Das OSK unterstützt mehrere Layouts, die vom fokussierten Widget angefordert werden:

```go
type OSKLayout uint8
const (
    // OSKLayoutAlpha: Vollständiges QWERTY/QWERTZ/AZERTY-Layout.
    // Umschaltbar zwischen Buchstaben, Zahlen und Sonderzeichen.
    OSKLayoutAlpha OSKLayout = iota

    // OSKLayoutNumeric: Nur Ziffern 0–9, Dezimaltrenner, Vorzeichen.
    // Kompaktes Layout für numerische Eingaben.
    OSKLayoutNumeric

    // OSKLayoutNumericInteger: Nur Ziffern 0–9 und Vorzeichen.
    // Kein Dezimaltrenner.
    OSKLayoutNumericInteger

    // OSKLayoutPhone: Telefonnum­mer-Layout (0–9, +, *, #).
    OSKLayoutPhone

    // OSKLayoutNone: Das Widget stellt sein eigenes Inline-Keypad bereit.
    // Das globale OSK wird unterdrückt.
    OSKLayoutNone OSKLayout = 255
)
```

### 5.4 OSK-Anforderung via Widget-Props

Widgets fordern ihr bevorzugtes OSK-Layout über ein Interface an:

```go
// OSKRequester wird von Widgets implementiert die ein bestimmtes
// OSK-Layout benötigen. Das Framework fragt dieses Interface
// bei FocusGained ab.
type OSKRequester interface {
    OSKLayout() OSKLayout
}
```

Beispiel: Ein `NumericInput` (→ §6.2) gibt `OSKLayoutNumeric` zurück. Das Framework zeigt automatisch das numerische Keyboard.

### 5.5 OSK-Rendering

Das OSK wird **vom Theme gerendert** — nicht hardcodiert. `DrawFunc(WidgetKindOSK)` ermöglicht vollständiges Custom-Rendering:

```go
// OSK-Keys sind semantisch, nicht visuell:
type OSKKey struct {
    Label    string     // Anzeige-Text ("Q", "123", "⌫")
    Action   OSKAction  // Was passiert beim Tap?
    Width    float32    // Relative Breite (1.0 = Standard-Taste)
}

type OSKAction uint8
const (
    OSKActionChar      OSKAction = iota  // Zeichen eingeben
    OSKActionBackspace                   // Zeichen löschen
    OSKActionEnter                       // Eingabe bestätigen / Fokus weiter
    OSKActionShift                       // Umschalttaste
    OSKActionSwitch                      // Layout wechseln (Alpha ↔ Numeric)
    OSKActionSpace                       // Leerzeichen
    OSKActionDismiss                     // OSK schließen
    OSKActionTab                         // Zum nächsten Feld
    OSKActionSign                        // Vorzeichen +/- umschalten
    OSKActionDecimal                     // Dezimaltrenner (locale-abhängig)
)
```

### 5.6 Locale-Awareness

Der Dezimaltrenner (`OSKActionDecimal`) ist locale-abhängig:
- `de-DE`: Komma (`,`)
- `en-US`: Punkt (`.`)

Das OSK liest die aktive Locale aus `RenderCtx.Locale` (→ RFC-003 §3.8) und passt die Darstellung und das generierte `CharMsg` an.

### 5.7 OSK für externe physische Keyboards

Wenn `HasPhysicalKeyboard == true`, wird das OSK nie automatisch angezeigt. Ein Widget kann trotzdem explizit ein OSK anfordern:

```go
// Programmatisch:
app.Send(ShowOSKMsg{Layout: OSKLayoutNumeric})
app.Send(DismissOSKMsg{})
```

---

## 6. Spezialisierte Input-Widgets

### 6.1 Überblick

Standard-Form-Widgets (TextField, Slider, Select) sind für Desktop-Nutzung mit Maus und Tastatur optimiert. Für HMI-Szenarien braucht man spezialisierte Eingabe-Widgets, die auf Touch-Bedienung und typische Maschinen-Eingaben zugeschnitten sind.

| Widget | Zweck | OSK-Layout | Tier |
|--------|-------|------------|------|
| `NumericInput` | Integer/Float-Eingabe mit Grenzen | Numeric/NumericInteger | HMI |
| `Stepper` | Inkrement/Dekrement mit fester Schrittweite | — (kein OSK) | HMI |
| `DrumPicker` | Auswahl aus diskreten Werten (Rollen-Metapher) | — | HMI |
| `UnitInput` | Wert + Einheit (z.B. "12.5 mm", "200 °C") | Numeric | HMI |
| `TimeInput` | Uhrzeit-Eingabe (HH:MM oder HH:MM:SS) | NumericInteger | HMI |
| `DateInput` | Datum-Eingabe (DrumPicker oder Direkteingabe) | NumericInteger | HMI |
| `RangeInput` | Min/Max-Bereichseingabe (Dual-Slider) | — | HMI |

### 6.2 NumericInput

Das zentrale Eingabe-Widget für Zahlenwerte auf HMI. Ersetzt den generischen `TextField` für alle Fälle in denen ein numerischer Wert erwartet wird.

#### Props

```go
type NumericInput struct {
    // Value: Aktueller Wert. Wird als String formatiert angezeigt.
    Value float64

    // Kind: Integer oder Fließkomma. Beeinflusst Validierung und OSK.
    Kind NumericKind

    // Min, Max: Wertebereich. Zero-Value = unbegrenzt.
    Min *float64
    Max *float64

    // Step: Schrittweite für die eingebetteten +/- Buttons.
    // 0 = keine +/- Buttons (nur Direkteingabe).
    Step float64

    // Precision: Anzahl Nachkommastellen für Fließkomma-Anzeige.
    // Nur relevant wenn Kind == NumericFloat.
    Precision int

    // Unit: Optionaler Einheiten-Suffix (z.B. "mm", "°C", "rpm").
    // Wird read-only neben dem Wert angezeigt.
    Unit string

    // Placeholder: Platzhalter-Text wenn Value == 0 und kein Fokus.
    Placeholder string

    // Label: Beschriftung über/neben dem Input.
    Label string

    // OnChange: Msg die bei Wertänderung gesendet wird.
    OnChange func(float64) Msg

    // Disabled: Deaktiviert das Widget.
    Disabled bool

    // Clamping: Verhalten bei Über-/Unterschreitung von Min/Max.
    Clamping ClampBehavior
}

type NumericKind uint8
const (
    NumericInteger NumericKind = iota  // Ganzzahl
    NumericFloat                       // Fließkommazahl
)

type ClampBehavior uint8
const (
    // ClampOnCommit: Wert wird beim Verlassen des Feldes (Blur)
    // auf [Min, Max] begrenzt.
    ClampOnCommit ClampBehavior = iota

    // ClampOnInput: Jede Eingabe die außerhalb [Min, Max] liegt,
    // wird sofort abgelehnt (Key wird nicht akzeptiert).
    ClampOnInput

    // ClampOnStep: Nur +/- Buttons werden begrenzt.
    // Direkteingabe darf überschreiten (mit Fehler-Anzeige).
    ClampOnStep
)
```

#### WidgetState

```go
type numericInputState struct {
    editing    bool       // true wenn Fokus + aktive Texteingabe
    textBuffer string     // Roher Eingabe-String während editing
    cursorPos  int        // Cursor-Position im textBuffer
    valid      bool       // Ergebnis der letzten Validierung
    errorText  string     // Fehlermeldung (z.B. "Wert muss ≥ 0 sein")

    // Animation-States:
    focusBorder Anim[float32]  // 0→1 bei Focus-Gained
    errorShake  Anim[float32]  // Shake-Animation bei ungültiger Eingabe
    stepAnim    Anim[float32]  // Flash-Animation bei +/- Tap
}
```

#### Msgs

```go
// NumericChangedMsg wird bei jeder validen Wertänderung gesendet.
type NumericChangedMsg struct {
    Value float64
}

// NumericCommitMsg wird beim Verlassen des Feldes gesendet.
// Enthält den finalen (ggf. geclampten) Wert.
type NumericCommitMsg struct {
    Value    float64
    Clamped  bool     // true wenn der Wert angepasst wurde
}
```

#### Verhalten: Touch-Modus

```
┌─────────────────────────────────────────┐
│  Temperatur                             │  ← Label
│  ┌───┐ ┌───────────────────────┐ ┌───┐ │
│  │ − │ │         23.5          │ │ + │ │  ← [Step-Button] [Value] [Step-Button]
│  └───┘ └───────────────────────┘ └───┘ │
│                                    °C   │  ← Unit
└─────────────────────────────────────────┘
```

1. **Tap auf +/−**: Wert wird um `Step` erhöht/verringert. Long-Press auf +/− startet Auto-Repeat (beschleunigend: erst Step, dann 2×Step, dann 5×Step).
2. **Tap auf Wert-Feld**: Öffnet OSK (`OSKLayoutNumeric` für Float, `OSKLayoutNumericInteger` für Integer). Das aktuelle Feld wird über die OSK gescrollt. Der bisherige Wert wird selektiert (Select-All) damit sofortige Überschreibung möglich ist.
3. **OSK-Eingabe**: Jedes eingegebene Zeichen wird live validiert. Ungültige Zeichen (Buchstaben, zweiter Dezimaltrenner) werden ignoriert. Bei `ClampOnInput` wird die Eingabe abgelehnt wenn der Teilwert bereits außerhalb des Bereichs liegt.
4. **Enter auf OSK / Tap außerhalb**: Commit. Wert wird geparst, validiert, optional geclampt. `NumericCommitMsg` wird gesendet. OSK schließt.
5. **Validierungsfehler**: Rotes Border, Shake-Animation, `errorText` wird unter dem Feld angezeigt.

#### Verhalten: Desktop-Modus

- +/− Buttons werden ausgeblendet (Maus ist präzise genug für Direkteingabe)
- Scroll-Wheel auf dem Feld ändert den Wert um `Step`
- Arrow-Up/Down ändert den Wert um `Step`
- Shift+Arrow: 10×Step, Ctrl+Arrow: 0.1×Step

#### Input-Filterung

```go
// isValidChar prüft ob ein Zeichen im aktuellen Kontext erlaubt ist.
func (s *numericInputState) isValidChar(ch rune, kind NumericKind) bool {
    switch {
    case ch >= '0' && ch <= '9':
        return true
    case ch == '-' || ch == '+':
        return s.cursorPos == 0  // Vorzeichen nur am Anfang
    case ch == '.' || ch == ',':
        return kind == NumericFloat && !strings.ContainsAny(s.textBuffer, ".,")
    default:
        return false
    }
}
```

#### A11y

```
AccessRole: SpinButton
AccessNode: { Value: "23.5", ValueMin: "0", ValueMax: "100",
              ValueText: "23.5 Grad Celsius", Label: "Temperatur" }
```

#### Theme-Tokens

```
State     Background          Border              Text           Step-Buttons
──────────────────────────────────────────────────────────────────────────────
Idle      Surface.Elevated    Stroke.Border       Text.Primary   Surface.Hovered
Focused   Surface.Elevated    Stroke.Focus (2px)  Text.Primary   Accent.Primary
Error     Surface.Elevated    Status.Error (2px)  Status.Error   Surface.Hovered
Disabled  Surface.Base        Stroke.Divider      Text.Disabled  —
```

### 6.3 Stepper

Ein Minimal-Widget für Inkrement/Dekrement — wenn nur die +/− Buttons benötigt werden, ohne Direkteingabe.

```go
type Stepper struct {
    Value    int
    Min      int
    Max      int
    Step     int         // Default: 1
    Label    string      // Angezeigt zwischen den Buttons
    Format   func(int) string  // Custom-Formatierung (z.B. "Tag %d")
    OnChange func(int) Msg
    Disabled bool

    // Orientation: Horizontal (Default) oder Vertikal.
    Orientation Orientation
}
```

```
Horizontal:                    Vertikal:
┌───┐ ┌──────────┐ ┌───┐      ┌───┐
│ − │ │    42     │ │ + │      │ ▲ │
└───┘ └──────────┘ └───┘      ├───┤
                               │42 │
                               ├───┤
                               │ ▼ │
                               └───┘
```

**Long-Press**: Auto-Repeat wie bei NumericInput.

### 6.4 DrumPicker

Auswahl aus einer diskreten Wertemenge über eine Rollen-Metapher (wie iOS UIPickerView). Ideal für Datum, Uhrzeit, Menü-Auswahl auf Touch.

```go
type DrumPicker struct {
    // Items: Die auswählbaren Werte.
    Items []DrumItem

    // SelectedIndex: Index des aktuell ausgewählten Items.
    SelectedIndex int

    // VisibleCount: Anzahl sichtbarer Zeilen (ungerade Zahl).
    // Default: 5 (2 oben + Selected + 2 unten).
    VisibleCount int

    // OnSelect: Msg bei Auswahl-Änderung.
    OnSelect func(index int) Msg

    // Looping: Ob die Liste zyklisch ist (letztes Element → erstes).
    Looping bool

    // Haptic: Haptisches Feedback bei jedem Raster-Snap.
    Haptic bool
}

type DrumItem struct {
    Label string
    Value any     // Opaker Wert für OnSelect
}
```

```
     ┌──────────────────┐
     │   ╌╌ 08 ╌╌      │  ← gedimmt, skaliert
     │   ╌╌ 09 ╌╌      │  ← gedimmt
  ═══│══════ 10 ════════│═══  ← Selected (hervorgehoben)
     │   ╌╌ 11 ╌╌      │  ← gedimmt
     │   ╌╌ 12 ╌╌      │  ← gedimmt, skaliert
     └──────────────────┘
```

**Scroll-Physik**: Kinetic Scrolling (RFC-002 §3) mit Raster-Snap. Nach dem Loslassen schnappt die Rolle auf das nächste Item ein. Snap-Animation nutzt `MotionSpec.Fast` (≤100ms).

#### Zusammengesetzte DrumPicker

Für Datum/Zeit werden mehrere DrumPicker horizontal kombiniert:

```go
// Convenience-Konstruktor:
ui.TimePicker{
    Value:      time.Now(),
    Format:     TimeFormatHHMM,  // oder HHMMss
    OnChange:   func(t time.Time) Msg { return TimeChangedMsg{t} },
}

// Intern: Zwei (oder drei) DrumPicker nebeneinander:
// ┌──────┐ : ┌──────┐
// │  14  │ : │  30  │
// └──────┘   └──────┘
//   Stunde    Minute
```

### 6.5 UnitInput

Numerische Eingabe mit Einheiten-Auswahl.

```go
type UnitInput struct {
    Value    float64
    Unit     string       // Aktive Einheit
    Units    []UnitDef    // Verfügbare Einheiten
    OnChange func(value float64, unit string) Msg
    Disabled bool

    // Alle NumericInput-Props werden eingebettet:
    NumericInput
}

type UnitDef struct {
    Symbol string    // "mm", "cm", "in"
    Label  string    // "Millimeter" (für Dropdown)
    Factor float64   // Umrechnungsfaktor relativ zur Basiseinheit
}
```

```
┌───┐ ┌─────────────────┐ ┌──────┐
│ − │ │       23.5       │ │ mm ▾ │  ← Unit-Dropdown
└───┘ └─────────────────┘ └──────┘
```

**Verhalten:**
- Tap auf Unit-Dropdown öffnet eine Auswahl der verfügbaren Einheiten
- Einheitenwechsel rechnet den Wert automatisch um (`Value * Factor`)
- Der Wert im User-Model bleibt in der Basiseinheit (normalisiert)

### 6.6 TimeInput / DateInput

Zeitspezifische Eingabe-Widgets die DrumPicker und Direkteingabe kombinieren.

```go
type TimeInput struct {
    Value    time.Time
    Format   TimeFormat
    OnChange func(time.Time) Msg
    Disabled bool

    // MinuteStep: Schrittweite für Minuten im DrumPicker.
    // 1 = jede Minute, 5 = 5-Minuten-Raster, 15 = Viertelstunden.
    MinuteStep int
}

type TimeFormat uint8
const (
    TimeFormatHHMM   TimeFormat = iota  // 14:30
    TimeFormatHHMMSS                    // 14:30:45
    TimeFormat12h                       // 2:30 PM
)

type DateInput struct {
    Value      time.Time
    OnChange   func(time.Time) Msg
    Disabled   bool

    // Format: Locale-abhängig (DD.MM.YYYY, MM/DD/YYYY, YYYY-MM-DD).
    // Default: aus RenderCtx.Locale abgeleitet.
    Format     DateFormat

    // Min, Max: Einschränkung des auswählbaren Bereichs.
    Min        *time.Time
    Max        *time.Time

    // Mode: Art der Eingabe.
    Mode       DateInputMode
}

type DateInputMode uint8
const (
    // DateModeDrum: DrumPicker (Tag | Monat | Jahr).
    // Ideal für Touch/HMI.
    DateModeDrum DateInputMode = iota

    // DateModeCalendar: Kalender-Popup.
    // Besser für Desktop.
    DateModeCalendar

    // DateModeDirect: Direkteingabe mit numerischer Maske.
    // Für Power-User.
    DateModeDirect
)
```

**Touch-Verhalten (DateModeDrum):**

```
┌──────────┐  ┌──────────┐  ┌──────────┐
│    18    │  │   März   │  │   2026   │
│  → 19 ← │  │ → März ← │  │ → 2026 ← │
│    20    │  │   April  │  │   2027   │
└──────────┘  └──────────┘  └──────────┘
     Tag         Monat          Jahr
```

- Jede Spalte ist ein `DrumPicker` mit Raster-Snap
- Tag-Picker passt sich automatisch an Monat/Jahr an (28/29/30/31 Tage)
- Monatsnamen kommen aus `RenderCtx.Locale`
- Haptisches Feedback bei jedem Raster-Snap

### 6.7 RangeInput (Dual-Slider)

Zwei gekoppelte Slider-Handles für Min/Max-Bereiche.

```go
type RangeInput struct {
    Low      float64
    High     float64
    Min      float64
    Max      float64
    Step     float64
    OnChange func(low, high float64) Msg
    Disabled bool

    // Labels: Wert-Anzeige an den Handles.
    ShowLabels bool

    // Format: Formatierung der angezeigten Werte.
    Format func(float64) string
}
```

```
         Low              High
          ▼                ▼
──────────●════════════════●──────────
0        20               80        100
```

- Touch: Jeder Handle hat `MinTouchTarget`-Größe
- Handles können nicht übereinander gezogen werden (`Low ≤ High` ist invariant)
- Tap auf die Schiene (zwischen den Handles) bewegt den nächstgelegenen Handle

### 6.8 OSK-Integration der Input-Widgets

Alle spezialisierten Input-Widgets implementieren `OSKRequester`:

```go
// NumericInput → OSKLayoutNumeric / OSKLayoutNumericInteger
// Im Touch-Modus öffnet NumericInput stattdessen ein eigenes NumericKeypad-Overlay
// und das globale OSK wird nicht angezeigt.
func (n NumericInput) OSKLayout() OSKLayout {
    if n.Kind == NumericInteger {
        return OSKLayoutNumericInteger
    }
    return OSKLayoutNumeric
}

// TimeInput, DateInput (im Direct-Mode) → OSKLayoutNumericInteger
func (t TimeInput) OSKLayout() OSKLayout { return OSKLayoutNumericInteger }
```

Das Framework öffnet das passende OSK-Layout automatisch wenn eines dieser Widgets Fokus erhält und `HasPhysicalKeyboard == false`. Widgets die `OSKLayoutNone` zurückgeben (oder ihr eigenes Keypad bereitstellen), unterdrücken das globale OSK.

### 6.9 Focus-Kette & Tab-Navigation

Auf Touch-HMI gibt es keine Tab-Taste — aber das OSK zeigt einen "Weiter"-Button (`OSKActionTab`). Dieser bewegt den Fokus zum nächsten Focusable in der Tab-Order. Das ermöglicht Formular-Durchlauf ohne OSK schließen/öffnen:

```
[NumericInput: Temperatur] → [Tab] → [NumericInput: Druck] → [Tab] → [Select: Modus]
```

Das OSK bleibt geöffnet und wechselt ggf. sein Layout wenn das nächste Widget ein anderes `OSKLayout()` anfordert.

---

## 7. HMI-Theme-Profil

### 7.1 theme.SlateHMI

Ein spezialisiertes Theme-Override für industrielle Touch-Panels:

```go
var SlateHMI = theme.Override(Slate, theme.OverrideSpec{
    Typography: &TypographyScale{
        // Skaliert auf 1.5×:
        H1:        TextStyle{Size: 30, Weight: 600, LineHeight: 1.3},
        H2:        TextStyle{Size: 24, Weight: 600, LineHeight: 1.3},
        H3:        TextStyle{Size: 21, Weight: 500, LineHeight: 1.4},
        Body:      TextStyle{Size: 20, Weight: 400, LineHeight: 1.5},
        BodySmall: TextStyle{Size: 18, Weight: 400, LineHeight: 1.5},
        Label:     TextStyle{Size: 18, Weight: 500, LineHeight: 1.0},
        Code:      TextStyle{Size: 20, Weight: 400, LineHeight: 1.6,
                             FontFamily: "JetBrains Mono"},
    },

    Spacing: &SpacingScale{XS: 8, S: 12, M: 24, L: 32, XL: 48, XXL: 64},

    Radii: &RadiusScale{
        Input:  8,   // Größere Radien für dickere Finger
        Button: 10,
        Card:   12,
        Pill:   999,
    },

    Colors: &ColorScheme{
        // Höherer Kontrast für Sonnenlicht/verschmutzte Displays:
        Surface: {
            Base:     Color{Hex: "#000000"},   // Pures Schwarz
            Elevated: Color{Hex: "#1a1a1a"},
            Hovered:  Color{Hex: "#333333"},   // Irrelevant im HMI-Profil
            Pressed:  Color{Hex: "#4d4d4d"},   // Deutlich sichtbar
        },
        Text: {
            Primary:   Color{Hex: "#ffffff"},   // Pures Weiß
            Secondary: Color{Hex: "#b3b3b3"},
            Disabled:  Color{Hex: "#666666"},
            OnAccent:  Color{Hex: "#ffffff"},
        },
        Stroke: {
            Border:  Color{Hex: "#ffffff", A: 0.15},  // Stärker als Desktop
            Focus:   Color{Hex: "#3b82f6"},
            Divider: Color{Hex: "#ffffff", A: 0.10},
        },
    },
})
```

### 7.2 Hochkontrast-Mode

Für extreme Lichtverhältnisse oder Sehbehinderungen:

```go
var SlateHMIHighContrast = theme.Override(SlateHMI, theme.OverrideSpec{
    Colors: &ColorScheme{
        Surface: {
            Base:     Color{Hex: "#000000"},
            Elevated: Color{Hex: "#1a1a1a"},
            Pressed:  Color{Hex: "#666666"},
        },
        Text: {
            Primary:  Color{Hex: "#ffffff"},
            Secondary: Color{Hex: "#ffff00"},  // Gelb für Kontrast
        },
        Stroke: {
            Border: Color{Hex: "#ffffff", A: 0.30},
            Focus:  Color{Hex: "#ffff00"},  // Gelber Focus-Ring
        },
        Status: {
            Error:   Color{Hex: "#ff0000"},  // Sattes Rot
            Warning: Color{Hex: "#ffff00"},  // Sattes Gelb
            Success: Color{Hex: "#00ff00"},  // Sattes Grün
        },
    },
})
```

### 7.3 Nachtmodus / Schichtbetrieb

Für Nachtschichten mit dunkler Umgebung:

```go
var SlateHMINight = theme.Override(SlateHMI, theme.OverrideSpec{
    Colors: &ColorScheme{
        // Maximale Dimming — kein reines Weiß, kein helles Blau:
        Surface: {
            Base:     Color{Hex: "#000000"},
            Elevated: Color{Hex: "#0d0d0d"},
            Pressed:  Color{Hex: "#262626"},
        },
        Accent: {
            Primary:         Color{Hex: "#1a4b8c"},  // Gedämpftes Blau
            PrimaryContrast: Color{Hex: "#cccccc"},
        },
        Text: {
            Primary:   Color{Hex: "#999999"},  // Gedämpftes Grau
            Secondary: Color{Hex: "#666666"},
        },
    },
})
```

---

## 8. Navigation & Layout-Patterns

### 8.1 Seitenbasierte Navigation

HMI-Interfaces sind seitenbasiert, nicht fensterbasiert. Navigation erfolgt über:

- **Tab-Bar** (am unteren oder seitlichen Rand) für Hauptbereiche
- **Breadcrumbs** für Hierarchie-Navigation
- **Swipe-Gesten** (links/rechts) für Seitenwechsel innerhalb eines Bereichs

```go
// PageNavigator: Framework-Widget für seitenbasierte Navigation.
type PageNavigator struct {
    Pages       []Page
    ActiveIndex int
    OnNavigate  func(index int) Msg

    // ShowTabBar: Tab-Bar anzeigen (Default: true).
    ShowTabBar bool

    // TabBarPosition: Unten (Default) oder Links.
    TabBarPosition TabBarPosition

    // SwipeNavigation: Seitenwechsel per Swipe (Default: true).
    SwipeNavigation bool
}

type Page struct {
    Label   string
    Icon    Icon
    Content func() Element  // Lazy — nur die aktive Seite wird gerendert
}
```

### 8.2 Scroll-Minimierung

Auf Touch-HMI sollte Scrollen minimiert werden:

- **Feste Seitenlayouts** bevorzugen — alles Wichtige ist ohne Scrollen sichtbar
- **Pagination statt Infinite-Scroll** für Listen
- **Keine verschachtelten Scroll-Views** — auf Touch extrem frustrierend
- **Scroll-Indikatoren immer sichtbar** (kein Auto-Hide)

### 8.3 Info-Disclosure-Pattern

Da Tooltips auf Touch nicht funktionieren, braucht man Alternativen für zusätzliche Informationen:

```go
// InfoButton: Ein kleines (i)-Icon das bei Tap ein Info-Panel
// unterhalb/neben dem referenzierten Element einblendet.
type InfoButton struct {
    Content  Element    // Der Info-Inhalt
    Position Position   // Below (Default), Above, Inline
}
```

```
[Temperatur] (i)
              │
              ▼
┌─────────────────────────────────┐
│ Zieltemperatur der Heizzone 3.  │
│ Bereich: 0–300 °C.              │
│ Empfohlener Wert: 180–220 °C.   │
└─────────────────────────────────┘
```

---

## 9. Industrielle Anforderungen

### 9.1 Alarm- und Warn-System

Maschinenzustands-Anzeigen erfordern besondere Widget-Patterns:

```go
// AlarmBanner: Dauerhaft sichtbares Banner am oberen Bildschirmrand.
// Nicht durch Navigation oder Scrollen verdeckbar.
type AlarmBanner struct {
    Severity AlarmSeverity
    Message  string
    OnAcknowledge func() Msg  // nil = keine Quittierung nötig

    // Blink: Visuelles Blinken für kritische Alarme.
    // Frequenz: 1Hz für Critical, kein Blink für Warning/Info.
    Blink bool
}

type AlarmSeverity uint8
const (
    AlarmInfo     AlarmSeverity = iota  // Blau — Information
    AlarmWarning                        // Gelb — Warnung
    AlarmCritical                       // Rot, blinkend — Sofortmaßnahme nötig
)
```

**Regeln:**
- Alarme sind immer sichtbar, auch über Overlays und Modals
- Kritische Alarme erfordern explizite Quittierung (`OnAcknowledge`)
- Alarm-Sounds/Vibration via `Platform.Haptics` und `Platform.Audio`

### 9.2 Statusfarben: Nicht nur Farbe

Farbblindheit betrifft ~8% der männlichen Bevölkerung. Auf HMI darf Status nie nur durch Farbe kommuniziert werden:

```
✓  OK     ─ Grün + Häkchen-Icon + Text "OK"
⚠  Warnung ─ Gelb + Dreieck-Icon + Text "Warnung"
✕  Fehler  ─ Rot  + Kreuz-Icon  + Text "Fehler"
```

### 9.3 Inaktivitäts-Timeout

```go
// InactivityConfig: Automatische Aktionen nach Inaktivität.
type InactivityConfig struct {
    // ScreenDimAfter: Display dimmen nach Inaktivität.
    // 0 = kein Dimmen.
    ScreenDimAfter time.Duration

    // ScreenLockAfter: Bildschirm sperren nach Inaktivität.
    // 0 = kein Sperren. Erfordert PIN-Eingabe zum Entsperren.
    ScreenLockAfter time.Duration

    // OnInactive: Custom-Msg bei Inaktivität.
    OnInactive func(duration time.Duration) Msg
}
```

### 9.4 Notfall-Bedienelemente

Notfall-Buttons (z.B. Emergency-Stop) haben besondere Anforderungen:

- **Immer sichtbar** — nie hinter Navigation, Scroll oder Overlays versteckt
- **Kein Debounce** — sofortige Auslösung bei erstem Touch
- **Kein Confirm-Dialog** — Bestätigung wäre lebensgefährlich
- **Maximale Touch-Fläche** — mindestens 80×80 dp
- **Visuell distinct** — Rot, groß, mit eindeutigem Icon
- **Framework-Level-Overlay** — rendert über allem anderen

```go
// EmergencyButton lebt als Framework-Overlay, nicht im Widget-Tree.
app.Run(myModel, myView, myUpdate,
    app.WithEmergencyAction(EmergencyConfig{
        Label:    "NOT-HALT",
        Icon:     icon.Stop,
        Position: BottomRight,   // Feste Position, nicht scrollbar
        Size:     80,            // 80×80 dp minimum
        OnPress:  func() Msg { return EmergencyStopMsg{} },
    }),
)
```

---

## 10. Performance auf eingebetteter Hardware

### 10.1 Frame-Budget

Industrielle Touch-Panels haben typisch:
- **CPU**: ARM Cortex-A53 / A72 (1–2 GHz)
- **GPU**: Mali-400/450 oder Vivante GC
- **RAM**: 512 MB – 2 GB
- **Display**: 7"–15", 800×480 bis 1920×1080

Frame-Budget bei 30 FPS: **33ms** pro Frame. Bei 60 FPS: **16ms**.

### 10.2 Optimierungsstrategien

**Reduzierte Animationen:**
```go
// Im HMI-Profil kann ReducedMotion aktiviert werden:
type InteractionProfile struct {
    // ... (§2.2)

    // ReducedMotion: Animationen auf Minimum reduzieren.
    // Transitions werden durch sofortige Zustandswechsel ersetzt.
    // Nur essenzielle Animationen (Fortschritt, Alarme) bleiben aktiv.
    ReducedMotion bool
}
```

**Dirty-Region-Rendering:**
- Nur geänderte Bereiche neu zeichnen (→ RFC-001 §6 VTree-Diff)
- Statische Bereiche (Labels, Rahmen) in einem Hintergrund-Layer cachen

**Asset-Optimierung:**
- SDF-Fonts statt Bitmap-Atlase (bereits implementiert, RFC-001 §6.3)
- Icon-Set als SDF-Atlas, nicht als einzelne Texturen
- Keine hochauflösenden Bitmaps für UI-Elemente

**OSK-Performance:**
- Das OSK wird einmal gerendert und als Textur gecacht
- Nur Key-Press-Highlights werden pro Frame aktualisiert
- Layout-Wechsel (Alpha → Numeric) invalidiert den OSK-Cache

### 10.3 Profiling-Hooks

```go
// Für die Entwicklung auf Ziel-Hardware:
app.Run(myModel, myView, myUpdate,
    app.WithFrameBudget(33 * time.Millisecond),  // Warnt wenn überschritten
    app.WithFrameCallback(func(stats FrameStats) {
        // stats.UpdateDuration, stats.LayoutDuration,
        // stats.PaintDuration, stats.TotalDuration
        if stats.TotalDuration > 33*time.Millisecond {
            log.Printf("frame budget exceeded: %v", stats.TotalDuration)
        }
    }),
)
```

---

## Anhang A: Widget-Tier-Einordnung

Die in diesem RFC definierten Widgets bilden einen eigenen Tier:

**Tier HMI — Touch & Maschinen-Interfaces** *(v1.x)*
`NumericInput`, `Stepper`, `DrumPicker`, `UnitInput`, `TimeInput`, `DateInput`, `RangeInput`, `ConfirmButton`, `HoldButton`, `AlarmBanner`, `InfoButton`, `PageNavigator`

Diese Widgets hängen von den Tier-1/2-Widgets ab (Text, Button, Slider, ScrollView) und erweitern sie für Touch/HMI-Szenarien.

## Anhang B: Invarianten

1. **Kein Widget kennt sein InteractionProfile** — es beeinflusst Layout und Dispatch, nicht Widget-Logik
2. **OSK ist Framework-Overlay** — kein Widget im User-Tree, kein User-State
3. **Gesture-Msgs sind synthetisch** — sie ersetzen TouchMsg, sie ergänzen es nicht (ein Tap erzeugt TapMsg, nicht TouchBegan+TouchEnded+TapMsg)
4. **EmergencyButton rendert über allem** — einschließlich OSK, Modals, Overlays
5. **HMI-Widgets validieren lokal** — der User-Loop bekommt nur valide Werte via OnChange/OnCommit
6. **Locale bestimmt Darstellung** — Dezimaltrenner, Datumsformat, Monatsnamen kommen aus `RenderCtx.Locale`
7. **ReducedMotion eliminiert non-essential Animationen** — Fortschrittsring (HoldButton), Alarme (Blink) bleiben aktiv

---
