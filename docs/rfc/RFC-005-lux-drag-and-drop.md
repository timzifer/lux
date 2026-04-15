# RFC-005 — lux: Drag-and-Drop-System

**Repository:** `github.com/timzifer/lux`
**Status:** Integriert
**Version:** 0.1.0
**Datum:** 2026-04-15
**Abhängig von:** RFC-001 (Core), RFC-002 (Interaction & Layout), RFC-004 (HMI Touch)

---

### Implementierungsstatus

| Abschnitt | Status | Anmerkung |
|-----------|--------|-----------|
| §1 Motivation & Abgrenzung | — | Kontext, kein Code |
| §2 Architektur-Überblick | ✅ Integriert | `ui/dnd_session.go`, `ui/dispatch.go` — DnDManager im Framework-Level |
| §3 Datenmodell | ✅ Integriert | `input/dnd.go` — DragData, DragItem, DragOperation, DropEffect, MIME-Typen |
| §4 DnD-Session-Manager | ✅ Integriert | `ui/dnd_session.go` — DnDManager, DragSession, DropZone |
| §5 Event-Typen | ✅ Integriert | `ui/input_event.go` — EventDragEnter/Over/Leave/Drop, Message-Structs |
| §6 DragSource-Widget | ✅ Integriert | `ui/data/drag_source.go` — Wrapper-Element, Placeholder, HandleOnly |
| §7 DropTarget-Widget | ✅ Integriert | `ui/data/drop_target.go` — Accept-Prädikat, Highlight-Styles, Priority |
| §8 SortableList-Widget | ✅ Integriert | `ui/data/sortable_list.go` — Reorderable Liste, Cross-List via GroupID |
| §9 DragHandle | ✅ Integriert | `ui/data/drag_handle.go` — 6-Punkt Grip-Icon, CursorGrab |
| §10 Visuelle Effekte | ✅ Integriert | `ui/dnd_preview.go` — Ghost-Preview, Placeholder, Drop-Zone-Highlight |
| §11 Tastatur-DnD | ✅ Integriert | `ui/dnd_keyboard.go` — Space/Tab/Enter/Escape-Workflow |
| §12 Touch-Geräte | ✅ Integriert | LongPress-Initiierung über bestehenden GestureRecognizer |
| §13 Plattform-Integration | ✅ Integriert | Modifier-Tracking in dispatch.go, Cursor-Mapping |
| §14 Testabdeckung | ✅ Integriert | `input/dnd_test.go`, `ui/dnd_session_test.go` |

---

## Inhaltsverzeichnis

1. [Motivation & Abgrenzung](#1-motivation--abgrenzung)
2. [Architektur-Überblick](#2-architektur-überblick)
3. [Datenmodell](#3-datenmodell)
4. [DnD-Session-Manager](#4-dnd-session-manager)
5. [Event-Typen](#5-event-typen)
6. [DragSource-Widget](#6-dragsource-widget)
7. [DropTarget-Widget](#7-droptarget-widget)
8. [SortableList-Widget](#8-sortablelist-widget)
9. [DragHandle](#9-draghandle)
10. [Visuelle Effekte](#10-visuelle-effekte)
11. [Tastatur-basiertes DnD](#11-tastatur-basiertes-dnd)
12. [Touch-Geräte](#12-touch-geräte)
13. [Plattform-Integration](#13-plattform-integration)
14. [Testabdeckung](#14-testabdeckung)

---

## 1. Motivation & Abgrenzung

Desktop-Umgebungen (Windows, KDE, GNOME) bieten umfassende Drag-and-Drop-Funktionalität:
- **Datei-Manager**: Dateien zwischen Ordnern verschieben/kopieren
- **Kanban-Boards**: Aufgaben zwischen Spalten ziehen
- **Sortierbare Listen**: Reihenfolge per Drag ändern
- **Cross-Widget-Transfer**: Daten zwischen unabhängigen Widgets austauschen

lux unterstützt bisher nur die Low-Level-Geste `DragMsg` (Phase/Start/Pos/Delta), die rein geometrisch ist. Es fehlt ein semantisches System für Daten-Transfer, Drop-Zonen, visuelle Effekte und Accessibility.

Dieses RFC definiert ein vollständiges Drag-and-Drop-System, das Desktop-Umgebungen in nichts nachsteht.

### 1.1 Abgrenzung

Dieses RFC behandelt **nicht**:
- OS-Level Drag (Dateien vom Desktop in die App ziehen) — das erfordert Platform-Backend-Erweiterungen
- Drag zwischen separaten Fenstern (Multi-Window DnD) — erfordert RFC-001 §7 Erweiterungen

---

## 2. Architektur-Überblick

Das DnD-System bildet eine semantische Schicht über dem bestehenden Gesten-System:

```
Platform Input → GestureRecognizer → DragMsg (existiert, RFC-004)
                                        ↓
                                  DnDManager (neu) → DragEnter/Over/Leave/Drop
                                        ↓
                                DragSource / DropTarget Widgets (neu)
```

### 2.1 Designprinzipien

- **Framework-Level-Koordination**: Der DnDManager lebt im Framework (wie FocusManager, EventDispatcher), nicht im User-Model
- **Elm-konform**: Widgets kommunizieren via `ctx.Send()`, der Framework-Loop fängt `StartDragSessionMsg` ab
- **Keine Hover-Slot-Interferenz**: Drop-Zonen sind von Hit-Targets getrennt (wie Scroll-Targets)
- **Frame-basiert**: Drop-Zonen werden jeden Frame neu registriert (wie Hit-Targets)

### 2.2 Lebenszyklus

1. User startet Drag-Geste → `DragMsg.DragBegan` an DragSource-Widget
2. DragSource sendet `StartDragSessionMsg` → Framework ruft `DnDManager.StartDrag()`
3. Pro Frame: `DnDManager.UpdateDrag(pos, mods)` aktualisiert Position und Hit-Testet Drop-Zonen
4. `DnDManager.DispatchDnDEvents()` generiert DragEnter/Over/Leave für betroffene DropTargets
5. Bei Release: `DnDManager.EndDrag()` prüft Accept → generiert DropEvent → Cleanup

---

## 3. Datenmodell

### 3.1 DragItem & DragData

```go
type DragItem struct {
    MIMEType string  // z.B. "text/plain", "application/json"
    Data     any     // Payload
}

type DragData struct {
    Items      []DragItem
    AllowedOps DragOperation
    SourceID   string
}
```

Mehrere Items mit unterschiedlichen MIME-Typen ermöglichen Format-Negotiation (analog zu Clipboard).

### 3.2 DragOperation & DropEffect

```go
type DragOperation uint8
const (
    DragOperationNone DragOperation = 0
    DragOperationMove DragOperation = 1 << iota
    DragOperationCopy
    DragOperationLink
)

type DropEffect uint8
const (
    DropEffectNone DropEffect = iota
    DropEffectMove
    DropEffectCopy
    DropEffectLink
)
```

### 3.3 Modifier-zu-Operation-Mapping

| Modifier | Operation |
|----------|-----------|
| Kein | Move (Default) |
| Ctrl | Copy |
| Shift | Move (explizit) |
| Ctrl+Shift | Link |

Fallback-Kette: Move → Copy → Link → None

### 3.4 Well-Known MIME-Typen

| Konstante | Wert | Verwendung |
|-----------|------|------------|
| `MIMEText` | `text/plain` | Allgemeiner Text |
| `MIMEURIList` | `text/uri-list` | URI-Listen |
| `MIMEJSON` | `application/json` | Strukturierte Daten |
| `MIMESortableKey` | `application/x-lux-sortable-key` | SortableList-Elemente |
| `MIMEWidgetID` | `application/x-lux-widget-id` | Widget-Referenzen |

---

## 4. DnD-Session-Manager

### 4.1 DragSession

```go
type DragSession struct {
    Phase           DragSessionPhase
    Data            *input.DragData
    SourceUID       UID
    SourceBounds    draw.Rect
    StartPos        input.GesturePoint
    CurrentPos      input.GesturePoint
    Modifiers       input.ModifierSet
    Operation       input.DragOperation
    Preview         Element
    PreviewOffset   draw.Point
    ShowPlaceholder bool
}
```

### 4.2 Drop-Zone-Registrierung

Drop-Zonen werden pro Frame registriert (wie Hit-Targets), leben aber in einer separaten Liste:

```go
type DropZone struct {
    UID      UID
    Bounds   draw.Rect
    Accept   func(*input.DragData, input.DragOperation) bool
    Priority int
}
```

### 4.3 Hit-Testing

Bei überlappenden Drop-Zonen gewinnt:
1. Höchste Priority
2. Bei gleicher Priority: kleinste Fläche (spezifischste Zone)

### 4.4 Event-Generierung

Der DnDManager verfolgt `hoveredZone` und `prevHovered`:
- Zone gewechselt → DragLeave(alt) + DragEnter(neu) + DragOver(neu)
- Gleiche Zone → nur DragOver
- Zone verlassen → DragLeave

---

## 5. Event-Typen

Vier neue InputEventKinds erweitern das bestehende Event-System:

```go
EventDragEnter  // Cursor mit Drag-Daten betritt Drop-Target
EventDragOver   // Cursor bewegt sich innerhalb eines Drop-Targets
EventDragLeave  // Cursor verlässt Drop-Target
EventDrop       // Daten wurden auf Target abgelegt
```

Message-Structs tragen `Data`, `Pos`, `Modifiers` und `Operation/Effect`.

---

## 6. DragSource-Widget

`DragSource` ist ein Wrapper-Widget (stateful, `Widget.Render`-Pattern) das jedes Child-Element draggable macht:

```go
type DragSource struct {
    ui.BaseElement
    Child       ui.Element
    Data        func() *input.DragData
    Operations  input.DragOperation
    Preview     func() ui.Element
    Placeholder bool
    HandleOnly  bool
    OnDragStart func()
    OnDragEnd   func(input.DropEffect)
}
```

### 6.1 Placeholder-Modus

Bei `Placeholder: true` wird am Ursprungsort ein gestricheltes Rechteck gezeichnet während das Element gezogen wird.

### 6.2 HandleOnly-Modus

Bei `HandleOnly: true` startet der Drag nur wenn der Griff-Bereich (DragHandle) berührt wird.

---

## 7. DropTarget-Widget

`DropTarget` ist ein Wrapper-Widget das eine Zone als Drop-Ziel registriert:

```go
type DropTarget struct {
    ui.BaseElement
    Child     ui.Element
    Accept    func(*input.DragData, input.DragOperation) bool
    OnDrop    func(*input.DragData, input.GesturePoint, input.DragOperation)
    Highlight DropHighlightStyle
    Priority  int
}
```

### 7.1 Highlight-Styles

| Style | Visueller Effekt |
|-------|-----------------|
| `DropHighlightBorder` | Akzent-Rahmen + leichter Hintergrund-Tint |
| `DropHighlightFill` | Semi-transparente Akzent-Füllung |
| `DropHighlightInsert` | Horizontale Insertions-Linie |
| `DropHighlightNone` | Kein automatisches Highlighting |

---

## 8. SortableList-Widget

`SortableList` kombiniert DragSource und DropTarget pro Item für Reorder-by-Drag:

```go
type SortableList struct {
    ui.BaseElement
    Items       []string
    BuildItem   func(key string, index int, dragging bool) ui.Element
    ItemHeight  float32
    OnReorder   func(fromIndex, toIndex int)
    State       *SortableListState
    GroupID     string   // Cross-List-Drag
    OnInsert    func(index int, data *input.DragData)
    OnRemove    func(index int)
}
```

### 8.1 Animations

Item-Verschiebungen werden mit `anim.Anim[float32]` animiert (Pattern von `TreeState.expandAnim`).

### 8.2 Cross-List

Items können zwischen SortableLists mit gleichem `GroupID` gezogen werden.

---

## 9. DragHandle

`DragHandle` zeigt ein 6-Punkt-Grip-Icon und setzt den `CursorGrab`-Cursor:

```go
type DragHandle struct {
    ui.BaseElement
    Size float32  // Default: 24dp
}
```

---

## 10. Visuelle Effekte

### 10.1 Drag-Preview (Ghost)

Der Drag-Preview wird als Overlay über allen Inhalten gerendert (im Overlay-Modus von BuildScene):
- Custom-Preview: User-definiertes Element mit 70% Opacity
- Default-Preview: Semi-transparentes Akzent-Rechteck

### 10.2 Drop-Zone-Highlight

Während eines aktiven Drags werden Drop-Zonen visuell hervorgehoben:
- **Akzeptierende Zone**: Akzent-Rahmen + leichter Hintergrund-Tint
- **Ablehnende Zone**: Error-Farbe mit niedrigem Alpha

### 10.3 Placeholder

Bei `ShowPlaceholder: true` wird am Ursprungsort ein gestrichelter Rahmen gezeichnet.

---

## 11. Tastatur-basiertes DnD

Barrierefreie Alternative für Nutzer ohne Maus/Touch (WCAG 2.1 AA):

1. **Fokus** auf DragSource → **Space/Enter** startet Drag-Modus
2. **Tab/Shift+Tab** zykliert durch verfügbare DropTargets
3. **Enter** bestätigt Drop am fokussierten Target
4. **Escape** bricht ab
5. **Pfeiltasten** in SortableList: Item hoch/runter bewegen

Implementiert über `FilterCollectedEvents` (Global Handler Layer, RFC-002 §2.8).

### 11.1 Screen-Reader-Ankündigungen

- „Element gegriffen" (DragStart)
- „Über Drop-Zone: [Label]" (DragEnter)
- „Abgelegt auf: [Label]" (Drop)
- „Drag abgebrochen" (Cancel)

---

## 12. Touch-Geräte

### 12.1 Long-Press-Initiierung

Auf Touch-Geräten (InteractionProfile.PointerKind != PointerMouse) wird der Drag nicht sofort bei DragBegan gestartet, sondern erst nach LongPress (500ms Default). Dies vermeidet Konflikte mit Scroll-Gesten.

### 12.2 Haptisches Feedback

`platform.Haptics.Impact()` wird ausgelöst:
- Beim Start des Drags
- Beim Betreten einer akzeptierenden Drop-Zone
- Beim Ablegen

### 12.3 Vergrößerte Drop-Zonen

Auf Touch-Geräten werden Drop-Zonen um `MinTouchTarget` (48dp) erweitert.

---

## 13. Plattform-Integration

### 13.1 Modifier-Tracking

Der EventDispatcher verfolgt den Modifier-Status (Ctrl/Shift/Alt) über KeyPress/KeyRelease-Events. Der DnDManager liest diesen Status für die Operation-Auflösung.

### 13.2 Cursor-Mapping

| Zustand | Cursor |
|---------|--------|
| Drag aktiv, keine Zone | `CursorGrabbing` |
| Über akzeptierender Zone | `CursorMove` |
| Über ablehnender Zone | `CursorNotAllowed` |

### 13.3 Interactor-Integration

`RegisterDropZone()` am Interactor leitet an den DnDManager weiter. Drop-Zonen verbrauchen keine Hover-Animation-Slots (separate Liste, wie Scroll-Targets).

---

## 14. Testabdeckung

### 14.1 Unit-Tests

| Datei | Tests | Abdeckung |
|-------|-------|-----------|
| `input/dnd_test.go` | DragData, DragOperation, DropEffect, ResolveOperation | Alle Methoden und Edge Cases |
| `ui/dnd_session_test.go` | DnDManager Lifecycle, Hit-Testing, Event-Dispatch, Cursor, Priority | 19 Tests, vollständige Abdeckung |

### 14.2 Integrations-Tests

Die KitchenSink-Demo (`examples/kitchen-sink/`) enthält 8 DnD-Sektionen:
- Basic DnD, Copy-on-Drag, Sortable List, Multiple Drop Zones
- Placeholder Drag, Kanban Board, Drag Handle, Keyboard DnD
