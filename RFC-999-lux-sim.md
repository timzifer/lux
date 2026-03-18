# RFC-999 — lux/sim: Callback-Compatibility Layer

**Repository:** `github.com/timzifer/lux`

**Status:** Draft
**Version:** 0.1.0
**Datum:** 2026-03-17
**Abhängigkeit:** RFC-999-lux-sim.md v0.2.0

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Abgrenzung](#2-abgrenzung)
3. [API-Übersicht](#3-api-übersicht)
4. [Interne Mechanik](#4-interne-mechanik)
5. [Ref\[T\] — Reaktive State-Zellen](#5-reft--reaktive-state-zellen)
6. [Widget-Wrappers](#6-widget-wrappers)
7. [Callback-Registry](#7-callback-registry)
8. [Dreiebenen-Vergleich](#8-dreiebenen-vergleich)
9. [Migrationspfad](#9-migrationspfad)
10. [Trade-offs & Einschränkungen](#10-trade-offs--einschränkungen)
11. [Paketstruktur & Dateien](#11-paketstruktur--dateien)
12. [Offene Fragen](#12-offene-fragen)

---

## 1. Motivation & Ziel

Lux (RFC-001) implementiert die Elm-Architektur: ein einzelnes immutables Model, eine reine
Update-Funktion, eine reine View-Funktion. Dieser Ansatz bietet herausragende Testbarkeit,
strukturelle Thread-Sicherheit und deterministisches Verhalten — aber er erfordert ein
Umdenken gegenüber klassischen OOP-UI-Frameworks.

Entwickler, die aus **Fyne**, **Qt**, **GTK** oder **Android/iOS-SDK** kommen, kennen:
- Widgets mit direkt setzbarem Zustand (`label.SetText(...)`)
- Callbacks auf Ereignisse (`button.OnTapped = func() { ... }`)
- Kein separates "Model" — der Zustand lebt *in* den Widgets

Das Ziel von `lux/sim` ist es, diese Vertrautheit als **Einstiegspunkt** anzubieten, ohne
die Vorteile von Lux aufzugeben. `sim` ist ein dünner Adapter: Callbacks werden intern in
Lux-Messages übersetzt, State-Zellen lösen Frame-Updates aus — der Entwickler merkt davon
zunächst nichts.

> **sim** = **Si**mulator des klassischen Widget-Modells. Kein Workaround, keine Krücke —
> ein explizit vorgesehener Onboarding-Pfad in das Lux-Ökosystem.

---

## 2. Abgrenzung

### Was sim ist

- Ein **Einstiegslayer** für Entwickler, die Elm-Architektur noch nicht kennen
- Eine **Brücke** für schnelle Prototypen und kleine Tools
- Ein **Migrationspfad**: Code lässt sich schrittweise in vollständiges Lux überführen
- Teil des offiziellen Lux-Ökosystems (`github.com/timzifer/lux/sim`)

### Was sim nicht ist

- **Kein Ziel-Zustand** für produktive, komplexe Applikationen
- **Kein vollwertiger Fyne-Ersatz** mit identischem API
- **Keine Umgehung** der Lux-Architektur (alles läuft durch denselben App-Loop)
- **Keine Performance-Optimierung** — leichter Overhead durch Callback-Registry ist bewusst
  akzeptiert

### Nicht-Ziele

- Vollständige API-Kompatibilität zu Fyne (Ziel: konzeptuell ähnlich, nicht binärkompatibel)
- Unterstützung von Callbacks die außerhalb des App-Loops ausgelöst werden (alle Callbacks
  laufen synchron im App-Loop — Thread-Sicherheit bleibt garantiert)
- Verschachteltes Mutable-State-Management (kein `widget.Container.Add()` zur Laufzeit)

---

## 3. API-Übersicht

```go
package sim

import (
    "github.com/timzifer/lux/app"
    "github.com/timzifer/lux/ui"
)

// ─── App-Einstieg ────────────────────────────────────────────────────────────

// Run startet eine sim-App. Keine generischen Parameter — der App-Entwickler
// braucht kein Model zu definieren.
func Run(root func() ui.Element, opts ...app.Option) error

// ─── Reaktive State-Zellen ───────────────────────────────────────────────────

// Ref[T] ist eine reaktive State-Zelle. Set() löst automatisch einen
// UI-Update-Zyklus aus.
type Ref[T any] struct { /* opaque */ }

func NewRef[T any](initial T) *Ref[T]
func (r *Ref[T]) Get() T
func (r *Ref[T]) Set(v T)

// Derived erstellt einen read-only Ref der automatisch aktualisiert wird
// wenn source sich ändert.
func Derived[A, B any](source *Ref[A], transform func(A) B) *Ref[B]

// ─── Widget-Wrappers ─────────────────────────────────────────────────────────

// Button mit OnClick-Callback.
func Button(label string, onClick func()) ui.Element

// Label das den Wert eines Ref anzeigt.
func Label[T any](ref *Ref[T], format func(T) string) ui.Element

// TextInput gebunden an einen Ref[string].
// OnChange wird nach jeder Änderung aufgerufen (optional, kann nil sein).
func TextInput(placeholder string, ref *Ref[string], onChange func(string)) ui.Element

// Checkbox gebunden an einen Ref[bool].
func Checkbox(label string, ref *Ref[bool]) ui.Element

// If rendert one wenn ref true ist, otherwise wenn false.
func If(ref *Ref[bool], one, otherwise ui.Element) ui.Element

// ForEach rendert für jeden Wert in ref ein Element.
// keyFn muss einen stabilen, eindeutigen String pro Item liefern (für UID-Stabilität).
func ForEach[T any](ref *Ref[[]T], keyFn func(T) string, render func(T) ui.Element) ui.Element
```

---

## 4. Interne Mechanik

`sim.Run` startet intern eine vollständige Lux-App mit automatisch generiertem Model und
Update. Der Nutzer sieht davon nichts.

### 4.1 Das interne Model

```go
// sim-intern — nie im User-API sichtbar
type simModel struct {
    // Version-Counter: jedes Set() inkrementiert diesen Wert.
    // Die View-Funktion läuft immer neu durch wenn version sich ändert.
    version uint64
}
```

Das Model ist bewusst minimal: `sim` speichert *keinen* Ref-Zustand im Model — die Refs
sind selbst der Zustand. Nur das Dirty-Signal (version) wandert durch den Elm-Loop.

### 4.2 Messages

```go
// sim-intern
type triggerMsg struct{}           // "irgendein Ref hat sich geändert — neu rendern"
type callbackMsg struct{ id uint64 } // "Callback #id ausführen"
```

### 4.3 Update-Funktion

```go
func simUpdate(m simModel, msg app.Msg) simModel {
    switch msg.(type) {
    case triggerMsg:
        m.version++
    case callbackMsg:
        cb := callbackRegistry.get(msg.(callbackMsg).id)
        if cb != nil {
            cb() // führt den User-Callback aus — kann weitere Set()-Aufrufe enthalten
        }
        m.version++
    }
    return m
}
```

Wichtig: Der Callback wird **innerhalb** des `update`-Aufrufs ausgeführt, also synchron
im App-Loop. Thread-Sicherheit ist strukturell garantiert — genau wie bei nativem Lux.

### 4.4 View-Funktion

```go
func simView(root func() ui.Element) app.ViewFunc[simModel] {
    return func(m simModel) ui.Element {
        // version wird implizit gelesen — bei jeder Änderung läuft root() neu
        return root()
    }
}
```

### 4.5 Datenfluss

```
Nutzer-Klick auf Button
        │
        ▼
ui.Button-Element enthält intern: onClickMsg{callbackID: 42}
        │  (Lux-Input-System dispatcht dies als Msg)
        ▼
simUpdate(model, callbackMsg{id: 42})
        │
        ▼
callbackRegistry[42]()  ←── das ist func() { text.Set("Neu") }
        │
        ▼
text.Set("Neu")
  → Ref-interne Wert-Änderung
  → app.Send(triggerMsg{})
        │
        ▼
simUpdate(model, triggerMsg{}) → model.version++
        │
        ▼
simView(root)() → root() erneut aufgerufen → Label liest text.Get() = "Neu"
        │
        ▼
VTree-Diff → Label-Text geändert → neu gerendert
```

---

## 5. Ref[T] — Reaktive State-Zellen

### 5.1 Implementierung

```go
package sim

import (
    "sync/atomic"
    "github.com/timzifer/lux/app"
)

var refIDCounter uint64

type Ref[T any] struct {
    id    uint64
    value T
}

func NewRef[T any](initial T) *Ref[T] {
    return &Ref[T]{
        id:    atomic.AddUint64(&refIDCounter, 1),
        value: initial,
    }
}

func (r *Ref[T]) Get() T {
    // Läuft immer im App-Loop-Thread (view wird single-threaded aufgerufen)
    // — kein Mutex nötig
    return r.value
}

func (r *Ref[T]) Set(v T) {
    r.value = v
    // Löst einen neuen Frame aus
    app.Send(triggerMsg{})
}
```

### 5.2 Warum kein Mutex?

`Get()` wird ausschließlich in der View-Funktion aufgerufen — die läuft single-threaded
im App-Loop. `Set()` darf nur aus Callbacks aufgerufen werden — die ebenfalls synchron
im App-Loop laufen (via `callbackMsg`). Es gibt keine Race Condition.

Soll `Set()` aus einer Goroutine aufgerufen werden (z.B. nach einem HTTP-Request), gilt
die standard Lux-Regel: `app.Send(triggerMsg{})` ist thread-safe, das eigentliche
`r.value = v` muss aber durch eine Message geschützt werden:

```go
// Goroutine-sichere Variante für async-Operationen
type setRefMsg[T any] struct {
    ref   *Ref[T]
    value T
}

// Im simUpdate:
case setRefMsg[T]:
    msg.ref.value = msg.value
    m.version++
```

### 5.3 Derived — abgeleitete State-Zellen

```go
// Derived erstellt einen Ref der automatisch transformiert wird.
// Wird als read-only betrachtet — Set() ist nicht verfügbar.
func Derived[A, B any](source *Ref[A], transform func(A) B) *Ref[B] {
    // Implementiert als Wrapper: Get() ruft transform(source.Get()) auf
    // Kein eigenes Caching — view() läuft bei jedem Frame ohnehin neu
    return &derivedRef[A, B]{source: source, fn: transform}
}

// Beispiel:
name := sim.NewRef("Alice")
greeting := sim.Derived(name, func(n string) string {
    return "Hallo, " + n + "!"
})
// greeting.Get() == "Hallo, Alice!"
```

---

## 6. Widget-Wrappers

### 6.1 Button

```go
func Button(label string, onClick func()) ui.Element {
    id := callbackRegistry.register(onClick)
    // Intern: ui.Button mit einer onClickMsg — Lux-nativer Button
    return ui.Button(label, callbackMsg{id: id})
}
```

**Wichtig:** `register()` wird bei jedem `view()`-Aufruf aufgerufen. Die Registry muss
daher idempotent sein oder Callbacks nach jedem Frame freigeben. Empfehlung: Registry
wird zu Beginn jedes `view()`-Aufrufs geleert und neu befüllt ("ephemeral callbacks").

### 6.2 Label

```go
func Label[T any](ref *Ref[T], format func(T) string) ui.Element {
    return ui.Text(format(ref.Get()))
}
```

Kein eigener Wrapper nötig — `Label` ist ein syntaktischer Shortcut für das Lesen des
Refs und das Erzeugen eines `ui.Text`-Elements.

### 6.3 TextInput

```go
func TextInput(placeholder string, ref *Ref[string], onChange func(string)) ui.Element {
    changeID := callbackRegistry.register(func() {
        // onChange erhält den neuen Wert — dieser wird intern via InputMsg übergeben
        // (Details: InputMsg enthält den aktuellen Text des Inputs)
    })
    return ui.TextInput(ui.TextInputProps{
        Placeholder: placeholder,
        Value:       ref.Get(),
        OnChange:    inputCallbackMsg{id: changeID, ref: ref, userCb: onChange},
    })
}
```

Das `TextInput`-Widget in Lux sendet beim Tippen eine `InputMsg` — sim fängt diese
ab und aktualisiert den Ref sowie den optionalen User-Callback.

### 6.4 Conditional Rendering: If

```go
func If(ref *Ref[bool], one, otherwise ui.Element) ui.Element {
    if ref.Get() {
        return one
    }
    return otherwise
}
```

Da `view()` bei jeder Änderung ohnehin neu läuft, ist dies ein einfaches `if` — kein
reaktiver Overhead nötig.

### 6.5 Listen: ForEach

```go
func ForEach[T any](ref *Ref[[]T], keyFn func(T) string, render func(T) ui.Element) ui.Element {
    items := ref.Get()
    children := make([]ui.Element, len(items))
    for i, item := range items {
        children[i] = ui.WithKey(keyFn(item), render(item))
    }
    return ui.Column(children...)
}
```

`ui.WithKey` ist entscheidend für UID-Stabilität (RFC-001 §4.4): Ohne expliziten Key
würde WidgetState bei Listenänderungen verloren gehen.

---

## 7. Callback-Registry

### 7.1 Ephemeral-Modell (empfohlen)

Die Registry wird zu Beginn jedes View-Durchlaufs geleert. Callbacks sind nur für den
aktuellen Frame gültig. Das verhindert Memory-Leaks und vereinfacht das Lifetime-Management.

```go
type callbackRegistry struct {
    mu      sync.Mutex  // nur für reset() nötig — view ist single-threaded
    entries map[uint64]func()
    counter uint64
}

var registry = &callbackRegistry{entries: make(map[uint64]func())}

func (r *callbackRegistry) reset() {
    r.entries = make(map[uint64]func())
    r.counter = 0
}

func (r *callbackRegistry) register(fn func()) uint64 {
    r.counter++
    id := r.counter
    r.entries[id] = fn
    return id
}

func (r *callbackRegistry) get(id uint64) func() {
    return r.entries[id]
}
```

**Ablauf:**
1. `simView()` ruft `registry.reset()` auf
2. `root()` wird aufgerufen — alle `sim.Button(...)` etc. registrieren ihre Callbacks
3. Rückgabe des Element-Baums (enthält `callbackMsg{id: N}` für jeden Button)
4. Klick → `simUpdate` → `registry.get(N)()` → Callback ausgeführt

### 7.2 Warum ephemeral statt persistent?

Persistent wäre fehleranfällig: Welcher Frame-Callback gilt noch? Garbage-Collection von
nicht mehr sichtbaren Widgets ist komplex. Ephemeral ist einfach, korrekt und transparent.

Der leichte Performance-Overhead (map-Reset pro Frame) ist bei typischen UI-Anwendungen
(≤60 fps, einige hundert Widgets) völlig vernachlässigbar.

---

## 8. Dreiebenen-Vergleich

Dasselbe Beispiel: Eine Counter-App mit Titel, Zähler und Button.

### Ebene 1: Fyne (zum Vergleich)

```go
package main

import (
    "fmt"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
)

func main() {
    a := app.New()
    w := a.NewWindow("Counter")

    count := 0
    label := widget.NewLabel("Zähler: 0")

    btn := widget.NewButton("+1", func() {
        count++
        label.SetText(fmt.Sprintf("Zähler: %d", count))
    })

    w.SetContent(container.NewVBox(
        widget.NewLabel("Counter App"),
        label,
        btn,
    ))
    w.ShowAndRun()
}
```

**Probleme im Fyne-Ansatz:**
- Zustand (`count`) und UI-Objekte (`label`) sind gekoppelt und über den Code verstreut
- Thread-Safety: `label.SetText` aus einer Goroutine → Race Condition ohne `fyne.Do`
- Testbarkeit: Schwer unit-testbar ohne echtes Fyne-Fenster

### Ebene 2: lux/sim (Einstieg — dieser RFC)

```go
package main

import (
    "fmt"
    "github.com/timzifer/lux/sim"
    "github.com/timzifer/lux/ui"
)

func main() {
    count := sim.NewRef(0)

    sim.Run(func() ui.Element {
        return ui.Column(
            ui.Text("Counter App"),
            sim.Label(count, func(n int) string {
                return fmt.Sprintf("Zähler: %d", n)
            }),
            sim.Button("+1", func() {
                count.Set(count.Get() + 1)
            }),
        )
    })
}
```

**Vorteile gegenüber Fyne:**
- Thread-Sicherheit strukturell garantiert (kein `fyne.Do` nötig)
- Zustand in `count` isoliert — keine direkten Widget-Referenzen
- Vertrautes Feeling für Fyne-Entwickler

### Ebene 3: Vollständiges Lux (Ziel)

```go
package main

import (
    "fmt"
    "github.com/timzifer/lux/app"
    "github.com/timzifer/lux/ui"
)

// Model
type Model struct {
    count int
}

// Messages
type IncrementMsg struct{}

// Update
func update(m Model, msg app.Msg) Model {
    switch msg.(type) {
    case IncrementMsg:
        return Model{count: m.count + 1}
    }
    return m
}

// View
func view(m Model) ui.Element {
    return ui.Column(
        ui.Text("Counter App"),
        ui.Text(fmt.Sprintf("Zähler: %d", m.count)),
        ui.Button("+1", IncrementMsg{}),
    )
}

func main() {
    app.Run(Model{}, update, view)
}
```

**Vorteile gegenüber lux/sim:**
- Model ist vollständig serialisierbar → Time-Travel-Debugging möglich
- Update ist eine pure Funktion → vollständig unit-testbar ohne UI
- Kein Overhead durch Callback-Registry
- Alle Zustandsübergänge explizit im `update`-Switch dokumentiert

---

## 9. Migrationspfad

Migration von `lux/sim` zu vollständigem Lux erfolgt schrittweise. Jede Stufe ist für
sich allein lauffähig.

### Stufe 0: Reines sim

```go
sim.Run(func() ui.Element {
    // komplett sim-basiert
    return sim.Button("...", func() { ref.Set(...) })
})
```

### Stufe 1: Einzelne native Lux-Widgets einführen

Native `ui.*`-Widgets können direkt in der `sim.Run`-View-Funktion verwendet werden.
Der `sim.Run`-Rahmen bleibt bestehen.

```go
sim.Run(func() ui.Element {
    return ui.Column(
        // Native ui.Text statt sim.Label:
        ui.Text(fmt.Sprintf("Zähler: %d", count.Get())),
        // sim.Button bleibt vorerst:
        sim.Button("+1", func() { count.Set(count.Get() + 1) }),
    )
})
```

### Stufe 2: Ref durch Msg-Typen ersetzen

Einen `Ref` durch eine echte Message + simUpdate-Logik ersetzen:

```go
// Alt:
count := sim.NewRef(0)
sim.Button("+1", func() { count.Set(count.Get() + 1) })

// Neu (nur diesen Ref):
type IncrementMsg struct{}
// ... in update():
case IncrementMsg:
    return Model{count: m.count + 1}
// In view():
ui.Button("+1", IncrementMsg{})
```

Dieser Schritt kann für jeden Ref separat durchgeführt werden.

### Stufe 3: sim.Run durch app.Run ersetzen

Wenn alle Refs durch Messages ersetzt sind, kann `sim.Run` durch `app.Run` ersetzt
werden. Das Ergebnis ist vollständiges Lux.

### Migrationshilfe: sim.Debug()

```go
// Gibt während der Entwicklung den aktuellen sim-Zustand aus:
// "sim state: 3 refs, 12 callbacks registered (last frame)"
sim.Debug(true)
```

---

## 10. Trade-offs & Einschränkungen

| Aspekt | lux/sim | Vollständiges Lux |
|---|---|---|
| **Lernkurve** | Gering (Fyne-ähnlich) | Höher (Elm-Konzept) |
| **Boilerplate** | Minimal | Explizit (gewollt) |
| **Testbarkeit** | Eingeschränkt (Callbacks nicht rein) | Vollständig (pure functions) |
| **Thread-Safety** | Garantiert (App-Loop) | Garantiert (App-Loop) |
| **Time-Travel-Debug** | Nicht möglich | Möglich |
| **Performance** | Minimaler Overhead (Callback-Registry) | Optimal |
| **A11y** | Via Lux-Widget-Layer vollständig | Vollständig |
| **Animations** | Via native Lux-Widgets | Vollständig |
| **Serialisierbarkeit** | Nein | Ja (Model ist plain struct) |

### Bekannte Einschränkungen

**1. Keine verschachtelten Closures über Frames**

```go
// FALSCH — capture von i in einer Loop-Closure ist undefiniert
for i := 0; i < 5; i++ {
    sim.Button(fmt.Sprintf("Button %d", i), func() {
        fmt.Println(i)  // i ist immer 5
    })
}

// RICHTIG — i als Parameter
for i := 0; i < 5; i++ {
    i := i  // shadowing
    sim.Button(fmt.Sprintf("Button %d", i), func() {
        fmt.Println(i)  // korrekt
    })
}
```

Dies ist ein Standard-Go-Problem, kein sim-spezifisches.

**2. Kein direktes Widget-State-Zugriff**

Lux's `WidgetState` (RFC-001 §4) ist framework-managed. In `lux/sim` gibt es keinen
Mechanismus, um auf den internen `WidgetState` eines Widgets von außen zuzugreifen.
Das ist gewollt — für Low-Level-Kontrolle sollte vollständiges Lux verwendet werden.

**3. Ephemeral Callback-IDs**

Callback-IDs werden pro Frame neu vergeben. Das bedeutet: Callbacks aus vorherigen
Frames (z.B. aus asynchronen Goroutinen) sind ungültig. Für async-Operationen muss
`app.Send()` direkt verwendet werden (siehe §5.2).

---

## 11. Paketstruktur & Dateien

```
lux/
└── sim/
    ├── sim.go          # Run(), internes Model/Update/View, sim.Debug()
    ├── ref.go          # Ref[T], Derived[A,B]
    ├── widgets.go      # Button(), Label(), TextInput(), Checkbox(), If(), ForEach()
    ├── callback.go     # callbackRegistry, callbackMsg, triggerMsg
    └── example_test.go # Dokumentations-Beispiele (go doc -compatible)
```

### sim.go (Struktur)

```go
package sim

import (
    "github.com/timzifer/lux/app"
    "github.com/timzifer/lux/ui"
)

type simModel struct {
    version uint64
}

type triggerMsg  struct{}
type callbackMsg struct{ id uint64 }

func simUpdate(m simModel, msg app.Msg) simModel { ... }
func simView(root func() ui.Element) app.ViewFunc[simModel] { ... }

func Run(root func() ui.Element, opts ...app.Option) error {
    return app.Run(simModel{}, simUpdate, simView(root), opts...)
}
```

### ref.go (Struktur)

```go
package sim

type Ref[T any] struct {
    id    uint64
    value T
}

type derivedRef[A, B any] struct {
    source *Ref[A]
    fn     func(A) B
}

// Ref[T] implementiert ein internes interface damit sim.Label[T]
// sowohl *Ref[T] als auch *derivedRef[A,B] akzeptiert.
type readable[T any] interface {
    Get() T
}
```

### callback.go (Struktur)

```go
package sim

var globalRegistry = newCallbackRegistry()

type callbackRegistry struct {
    entries map[uint64]func()
    counter uint64
}

func newCallbackRegistry() *callbackRegistry { ... }
func (r *callbackRegistry) reset()           { ... }
func (r *callbackRegistry) register(fn func()) uint64 { ... }
func (r *callbackRegistry) get(id uint64) func() { ... }
```

---

## 12. Offene Fragen

### 12.1 Async-Ref-Updates

Wie soll `Ref.Set()` aus einer Goroutine aufgerufen werden?

**Option A:** Panic wenn außerhalb des App-Loops aufgerufen (streng, verhindert Bugs)
**Option B:** `app.Send(setRefMsg{...})` als offiziellen async-Pfad dokumentieren
**Option C:** `Ref.SetAsync(v T)` als explizite async-Variante anbieten

*Empfehlung: Option B + Dokumentation. Option C als syntaktischer Zucker später.*

### 12.2 Callback-Deduplication

Wenn derselbe Button zweimal in der View vorkommt (z.B. in zwei Branches eines `If`),
werden zwei Callbacks registriert. Das ist korrekt, aber ineffizient. Eine Hash-basierte
Deduplication könnte helfen — aber nur wenn der Overhead messbar ist.

*Empfehlung: Erst bei nachgewiesenem Performance-Problem implementieren.*

### 12.3 sim.Form() Helper

Ein `sim.Form`-Helper für häufige Formularmuster könnte die Einstiegshürde weiter senken:

```go
sim.Form(
    sim.Field("Name",  sim.TextInput("Max Mustermann", nameRef, nil)),
    sim.Field("Email", sim.TextInput("max@example.com", emailRef, nil)),
    sim.SubmitButton("Absenden", func() { submit(nameRef.Get(), emailRef.Get()) }),
)
```

*Status: Idee — erst nach v1.0 von RFC-001.*

### 12.4 Testing-Utilities

```go
// Mögliche sim-Testhelfer
sim_test.NewTestApp(root func() ui.Element) *SimTestApp

app.Click("#submit-button")
app.AssertText("#status", "Erfolgreich!")
```

*Diese bauen auf dem Inspector-Protokoll aus RFC-001 §17 auf.*

### 12.5 Verhältnis zu lux/inspector

Der Inspector (RFC-001 §17) streamt den VTree über TCP/Unix-Socket. In einer sim-App
enthält dieser VTree `callbackMsg`-Props statt echter `Msg`-Typen — der Inspector könnte
diese decodieren und anzeigen. Kein Blocking-Problem, aber eine UX-Überlegung für das
Inspector-Team.

---

## Zusammenfassung

`lux/sim` ist eine einzige Go-Datei plus ein paar Helfer — kein separates Framework. Es
schiebt die Elm-Konzepte "unter die Haube" und gibt Entwicklern einen vertrauten
Einstiegspunkt. Die Migration zu vollständigem Lux ist graduell und nie erzwungen.

**Kern-Insight:** Callbacks sind nicht das Gegenteil von Elm — sie sind ein Elm-Msg-Dispatch
mit syntaktischem Zucker. `lux/sim` macht diesen Zucker explizit und einheitlich.

---

*Ende RFC-002 — lux/sim*
