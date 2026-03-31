# RFC-998 — Eigene Browser-Engine auf Basis des Lux UI-Kits

**Repository:** `github.com/timzifer/lux`

**Status:** Very Theoretical — nicht zur Umsetzung vorgesehen
**Version:** 0.3.0
**Datum:** 2026-03-26
**Zuletzt abgeglichen:** 2026-03-31
**Abhängigkeiten:** RFC-001 (Core), RFC-002 (Interaction/Layout), RFC-003 (Widget-Katalog), RFC-004 (WebView)

> **Hinweis:** Dieses RFC ist eine theoretische Machbarkeitsanalyse. Ziel ist **nicht** der Start eines Engine-Projekts, sondern eine ehrliche Einordnung: *Was wäre mit Lux bereits möglich, was fehlt, und wo entstehen Synergien für Lux selbst?*

### Implementierungsstatus

| Abschnitt | Status |
|-----------|--------|
| Alle Abschnitte | ⏸ Very Theoretical — keine Umsetzung geplant |

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Abgrenzung](#2-abgrenzung)
3. [Bestandsaufnahme: Lux als Browser-Fundament](#3-bestandsaufnahme-lux-als-browser-fundament)
4. [Geplante Lux-Komponenten mit Browser-Synergie](#4-geplante-lux-komponenten-mit-browser-synergie)
5. [Architektur-Überblick](#5-architektur-überblick)
6. [Subsystem-Analyse](#6-subsystem-analyse)
7. [Aufwandsmatrix](#7-aufwandsmatrix)
8. [Vergleich mit existierenden Engines](#8-vergleich-mit-existierenden-engines)
9. [Realistische Einordnung](#9-realistische-einordnung)
10. [Phasenmodell](#10-phasenmodell)
11. [Synergie-Effekte für Lux](#11-synergie-effekte-für-lux)
12. [Risiken & offene Fragen](#12-risiken--offene-fragen)
13. [Fazit](#13-fazit)

---

## 1. Motivation & Ziel

RFC-004 beschreibt einen pragmatischen Weg über bestehende Engines (WebView2/WPE/Servo) und benennt klar deren Trade-offs (Binary-Größe, RAM, Embedding-Reife, plattformspezifische Integrationen).

Dieses RFC betrachtet die Gegenfrage:

> Wie komplex wäre eine **eigene** Browser-Engine auf Basis von Lux, wenn HTML/CSS-Parser und JS-Engine extern kommen?

### Ziele

- Transparente Zerlegung in Subsysteme (DOM, CSS, Layout, Paint, JS-Bridge, etc.)
- Nüchterne Komplexitäts- und LOC-Schätzung
- Klare Trennung: Was kann Lux heute schon, was muss neu gebaut werden
- Herausarbeiten der Rückflüsse in das Lux-Core-Framework

### Nicht-Ziel

- Kein „CEF-Ersatz in 6 Monaten“-Narrativ
- Kein Commitment auf Implementierung
- Keine Vollständigkeitszusage gegenüber WHATWG/W3C-Spezifikationen

---

## 2. Abgrenzung

### In Scope

- Engine-internes Rendering im Lux-Renderpfad
- HTML/CSS/JS Laufzeit auf Desktop-Zielplattformen
- Einordnung für ein realistisches Teilmengen-Web (Reader-/App-Szenarien)

### Nicht in Scope

- Vollständige Web-Kompatibilität auf Blink/WebKit/Gecko-Niveau
- Multi-Prozess-Sandbox-Architektur wie moderne Browser
- Vollständige Security-Härtung (Site Isolation, Spectre-Mitigation, etc.)
- Browser-Produkt (Sync, Extensions, Profile-Management, etc.)

---

## 3. Bestandsaufnahme: Lux als Browser-Fundament

Die zentrale Beobachtung: Lux hat bereits ungewöhnlich viele primitive und halb-hohe UI-Bausteine, die in einer Browser-Engine direkt nutzbar sind.

### 3.1 Direkt wiederverwendbar (implementiert ✅)

| Lux-Komponente | Dateipfad | Browser-Äquivalent | Vereinfachung |
|---|---|---|---|
| Flexbox Layout | `ui/layout/flex.go` | CSS `display:flex` | ✅ CSS-Spec-konform: Direction, Justify, Align, Gap, FlexWrap, FlexBasis, FlexGrow/Shrink, AlignContent, Order |
| Grid Layout | `ui/layout/grid.go` | CSS `display:grid` | ✅ CSS-Spec-konform: Track-Sizing, fr-Units, Repeat, Gap, Span, Auto-Placement |
| Stack Layout | `ui/layout/stack.go` | Positioning / Z-Stacking | Basis für absolute Overlays & Stacking |
| ScrollView + Kinetic | `ui/nav/scroll.go`, `ui/kinetic_scroll.go` | `overflow:auto/scroll` | Scrolling/Scrollbar/Trägheit bereits gelöst |
| VirtualList | `ui/data/virtuallist.go` | Viewport-Culling | Direkt nutzbar für große Dokumente |
| Tree Widget | `ui/data/tree.go` | DOM-Inspector | DevTools-Struktur nahezu direkt |
| RichText + Inline/Block Widgets | `ui/display/richtext.go` (897 LOC) | Inline Text-Runs + Replaced Elements + Block Elements + Floats + Lists | Inline-Layout mit gemischtem Text-/Widget-Flow; `InlineWidget` (Baseline + Block-Modus), `ImageSpan` (Float: None/Left/Right/Block), Listen (ul/ol, Nesting, 9 Marker-Stile), CSS-Paragraph-Styling (Align/Indent/LineHeight/ParaSpacing) |
| Link Widget | `ui/link/link.go` (235 LOC) | HTML `<a>` | Klickbarer Inline-Link mit Hover/Focus-States, A11y, Theme-Integration; einbettbar als `InlineWidget` in RichText |
| Text Shaping | `internal/text/gotext_shaper.go` | `font-*`, Shaping | GSUB/GPOS + Fallback-Ketten vorhanden |
| Font-Fallback | `fonts/fonts.go` | `font-family` Cascade | Fallback-Mechanik bereits robust |
| Line Breaking | `internal/text/linebreak.go` | `word-wrap`, `overflow-wrap` | Unicode-konforme Zeilenumbrüche vorhanden |
| BiDi | `internal/text/bidi.go` | `direction`, `unicode-bidi` | RTL/LTR-Infrastruktur vorhanden |
| Canvas | `draw/canvas.go` | Painting | Primitive für Rect/Path/Text/Image/Clip/Transform |
| Paint/Gradients | `draw/paint.go` | CSS Hintergründe/Verläufe | Linear/Radial als starke Basis |
| Shadows | `ui/effects/shadow.go` | `box-shadow` | Outer/Inner Shadow bereits vorhanden |
| Blur | `ui/effects/blur.go` | `filter: blur()` | Für Filter-/Backdrop-Pfade nutzbar |
| Opacity | `ui/effects/opacity.go` | `opacity` | Layer-Opacity vorhanden |
| Clipping/Transform | `draw/canvas.go` | `overflow:hidden`, `transform` | Clip-Stack + Transform-Stack vorhanden |
| Image | `ui/display/image.go`, `image/` | `<img>` | Laden/Skalieren bereits umgesetzt |
| 9-Slice | `draw/canvas.go` | `border-image` | Nützlicher Spezialfall |
| Form-Controls | `ui/form/*.go` | `<input>`, `<select>`, `<textarea>`, `<progress>`, `<input type="date/color/time/number">` | Breites Control-Spektrum inkl. DatePicker, ColorPicker, TimePicker, NumericInput, Spinner |
| Button/Tabs/Dialog/Tooltip/Menu | `ui/button/`, `ui/nav/`, `ui/dialog/`, `ui/menu/` | Browser-UI + HTML-nahe Controls | Wiederverwendung für Browser-Chrome und interaktive Elemente |
| Hit Testing | `internal/hit/hit.go` | Event-Target-Findung | Essenziell für Pointer-Dispatch |
| Dispatch/Focus | `ui/dispatch.go`, `ui/focus.go`, `ui/focus_trap.go` | Capture/Bubble + Tab-Navigation | Framework-Grundlage bereits da |
| Gestures/Cursor | `ui/gesture.go`, `ui/interactor.go` | Pointer-/Touch-Ebene | Praktisch für mobile Gesten/Pointer-Cursor |
| A11y Tree + Bridges | `ui/access_tree_builder.go`, `a11y/`, `platform/atspi/` | ARIA/Accessibility | Großer strategischer Vorteil gegenüber Greenfield |
| Animation | `anim/` | CSS Animation/Transition | Timing-, Spring- und Gruppensystem vorhanden |
| Theme/Tokens | `theme/theme.go` | Teilaspekt CSS-Cascade/Variablen | Für UA-Styles und Theming nutzbar |
| Reconcile + Dirty Tracking | `ui/reconcile.go` | Incremental Rendering | Für effiziente Reflow/Repaint-Zyklen sehr wertvoll |
| GPU Renderer | `internal/gpu/wgpu_renderer.go` | Compositing-Backend | Bereits production-naher Rendererpfad |
| Scene Builder | `internal/render/canvas.go` | Display List / Batching | Fundament für Paint/Composite-Pipeline |
| Overlay System | `ui/overlay.go` | Popups/Floating UI | Wichtig für Menüs, Selects, Tooltips |
| SVG Rendering | `image/svg.go`, `image/svgpath.go` | `<svg>` | GPU-beschleunigte Vektorisierung; path/rect/circle/ellipse/line/polygon/polyline/g |
| Table Layout | `ui/layout/table.go` | CSS `display:table` | HTML-Spec-konformes CSS Table Layout (Fixed + Auto, 1038 LOC) |
| SplitView | `ui/nav/splitview.go` | Resizable Panes | Kollabierbare/resize-fähige Panel-Trennung |

**Zwischenfazit:** Der „untere” Teil der Rendering-Pipeline (Text, Paint, Compositor, Input, A11y) ist im Vergleich zu typischen Hobby-Engine-Projekten außergewöhnlich weit. Seit der Erstfassung dieses RFC sind SVG-Rendering, Table Layout, Inline Widgets in RichText und fünf weitere Form-Controls (DatePicker, ColorPicker, TimePicker, NumericInput, Spinner) hinzugekommen — der Reuse-Anteil ist damit nochmals signifikant gestiegen.

---

## 4. Geplante Lux-Komponenten mit Browser-Synergie

| Lux-Komponente | Status | Browser-Äquivalent | Synergie |
|---|---|---|---|
| DataTable / CSS Table Layout | ✅ Integriert | `<table>` | `ui/layout/table.go` (1038 LOC) — HTML-Spec-konformes CSS Table Layout (Fixed + Auto); DataTable-Widget in `ui/data/datatable.go` mit Pagination, Sortierung, Filter |
| DatePicker | ✅ Integriert | `<input type="date">` | `ui/form/datepicker.go` — Kalender-Dropdown mit Monatsnavigation |
| ColorPicker | ✅ Integriert | `<input type="color">` | `ui/form/colorpicker.go` — 16-Farben-Palette mit Dropdown |
| TimePicker | ✅ Integriert | `<input type="time">` | `ui/form/timepicker.go` — HH:MM-Auswahl (neu seit v0.1.0) |
| NumericInput | ✅ Integriert | `<input type="number">` | `ui/form/numericinput.go` — Stepper, Drag-to-Adjust, Unit-Suffix (neu seit v0.1.0) |
| Spinner | ✅ Integriert | CSS `animation` Spinner | `ui/form/spinner.go` — Animierter Ladeindikator (neu seit v0.1.0) |
| Toolbar | ✅ Integriert | Browser-Chrome | `ui/nav/toolbar.go` — Item-Groups, Separators, Toggle-Buttons |
| FilePicker | ✅ Integriert | `<input type="file">` | `ui/form/filepicker.go` — Open/Save mit OS-Dialog |
| RichTextEditor | ✅ Integriert | `contenteditable` | `richtext/` — Tagged-Range `AttributedString`, 17 Attribut-Typen (Span/Paragraph/List), Cursor, Selection, Undo/Redo, ToolbarCommands (Bold/Italic/Underline/Strikethrough/Align/List/Indent) |
| Inline/Block Widgets in RichText | ✅ Integriert | Replaced Elements (Inline + Block) + Floats | `ui/display/richtext.go` (897 LOC) — `InlineWidget` (Baseline + Block-Modus), `ImageSpan` (Float: None/Left/Right/Block), Listen (ul/ol, Nesting, Marker-Stile), CSS-Paragraph-Styling |
| Link Widget | ✅ Integriert | HTML `<a>` | `ui/link/link.go` (235 LOC) — klickbarer Inline-Link, Hover/Focus, A11y, einbettbar als InlineWidget |
| Code Editor | 🔶 RFC-010 | Script/CSS-Editor in DevTools | Reuse für Inspektor-/Source-Views; RFC-Design abgeschlossen, Implementierung ausstehend |
| SVG Support | ✅ Integriert | `<svg>` | `image/svg.go` (785 LOC) + `svgpath.go` (697 LOC) — GPU-beschleunigte Vektorisierung; Unterstützung für path/rect/circle/ellipse/line/polygon/polyline/g |
| Inspector/DevTools | ✅ Integriert | Browser DevTools | RFC-012 Inspector-Vellum PoC: Widget-Tree, Layout-Overlay, Event-Log, State-Dump, Frame-Metriken |
| DynamicDataset | ✅ Integriert | Infinite/Lazy DOM | `ui/data/paged_dataset.go` — Page-basierte Lazy-Loading-Datenquelle |

---

## 5. Architektur-Überblick

Geplante Pipeline (vereinfacht):

```text
HTML bytes ─┬─> HTML Parser ─> DOM
            └─> CSS Parser  ─> Stylesheets

DOM + Stylesheets
   └─> Selector Matching + Cascade
       └─> Computed Style
           └─> Render Tree (inkl. anonymous boxes)
               └─> Layout (block/inline/flex/grid/table)
                   └─> Display List / Paint Ops
                       └─> Lux Canvas + GPU Renderer
                           └─> Composite + Present

JS Engine <─> DOM/CSSOM Bridge <─> Event Loop / Tasks / Microtasks
```

### Leitidee

- Parser/JS extern einkaufen
- CSS-Layout- und DOM/CSSOM-Kern als Eigenleistung
- Painting/Compositing/Input/A11y maximal auf Lux-Reuse stützen

---

## 6. Subsystem-Analyse

### 6.1 HTML Parsing & DOM

**Extern:** `golang.org/x/net/html` als HTML5-nahe Parserbasis.  
**Neubau:** Eigenes DOM-Modell (Node-Typen, Attribute, Mutation, Traversal, Live Collections).  
**Komplexität:** Mittel bis hoch.

**Schätzung:** ~8–15 KLOC (ohne vollständige Web-API-Kompatibilität).

**Einordnung zu Go-Libraries:**  
Es gibt fertige Go-Bausteine wie `golang.org/x/net/html` (Parser) und darauf aufbauende Pakete (`goquery`, `cascadia`) für Query/Selektoren. Diese ersetzen aber in der Regel **nicht** ein browsernahes, lebendes DOM mit Mutation-Semantik, Event-Integration, Live-Collections und JS-exponierten Interfaces.

### 6.2 CSS Parsing & Cascade

**Extern:** Optional Parser-Libs; oft sinnvoll: eigener schlanker Parser für kontrollierte Teilmenge.  
**Neubau:** Selector Engine, Specificity, Inheritance, Computed Values, Initial Values, Shorthands.  
**Komplexität:** Hoch.

**Schätzung:** ~12–25 KLOC.

**Einordnung zu Go-Libraries:**  
Es existieren CSS-Parser/Selector-Libraries in Go (z. B. `tdewolff/parse`, `cascadia` für Selektoren). Für eine Engine helfen sie beim Parsing/Matching, decken aber typischerweise nicht die vollständige Cascade-/Computed-Style- und CSSOM-Semantik ab.

### 6.3 Layout Engine (Block/Inline/Flex/Grid/Table)

**Reuse:** Hoch bei Flex und Table (Flex inkl. Basis/Wrap/Order-Bausteine bereits vorhanden), zusätzlich Teile von `grid.go` sowie RichText/Textlayout.  
**Neubau:** Weiterhin groß: vollständiger CSS-Formatting-Context (anonymous boxes, line boxes, containing blocks, margin-collapsing, positioning, floats, fragmentation optional), plus spec-nahe Grid-Details.

**Komplexität:** Hoch bis sehr hoch (kritischer Pfad bleibt Block/Inline/Positioning/Floats).

**Schätzung (aktualisiert):** ~25–55 KLOC (ohne Print/Paged Media).

### 6.4 Painting & Compositing

**Reuse:** Sehr hoch (`draw/*`, GPU-Renderer, Scene Builder, Effekte).  
**Neubau:** CSS Painting Semantik (stacking contexts, paint order, blend/filter details), invalidation rules.

**Komplexität:** Mittel bis hoch.

**Schätzung:** ~10–20 KLOC.

### 6.5 Text & Fonts

**Reuse:** Sehr hoch (Shaping, Fallback, BiDi, LineBreak).  
**Neubau:** CSS Inline Formatting Details (baseline alignment, inline-level box metrics, white-space modes, text-decoration edge cases).

**Komplexität:** Mittel bis hoch.

**Schätzung:** ~8–18 KLOC.

### 6.6 Networking & Resource Loading

**Extern:** Go-stdlib (`net/http`, `crypto/tls`, `net/url`).  
**Neubau:** Priorisierung, Cache-Layer, CORS/CSP-Grundlagen, Referrer/Origin-Policy, Redirect/Content sniffing.

**Komplexität:** Hoch (Security-relevant).

**Schätzung:** ~10–22 KLOC.

### 6.7 JavaScript Integration

**Extern:** QuickJS/V8/Goja.  
**Neubau:** Host Runtime, Task Queue, Timers, Promise Microtask Scheduling, Exception-Mapping.

**Komplexität:** Hoch.

**Schätzung:** ~8–18 KLOC (ohne große Web-API-Fläche).

### 6.8 DOM↔JS Bridge

**Neubau:** Bindings für Node/Element/Document/Events, Lifetime/Garbage-Interop, Property Hooks, Mutation-Observability.

**Komplexität:** Sehr hoch.

**Schätzung:** ~15–35 KLOC.

**Begriffspräzisierung „JS-Bridge“:**  
In diesem RFC sind **zwei** Brücken gemeint:

1. **DOM/CSSOM ↔ JS (Engine-intern):**  
   `document`, `element.style`, Events, DOM-Mutationen aus JS und Rückwirkung ins Layout/Paint.
2. **Host-/Welt-Bridge ↔ JS (Engine-extern):**  
   Anbindung von JS an die Laufzeitumgebung (`fetch`, Timer, Storage, Navigation, ggf. App-spezifische Host-APIs).

Die erste Brücke ist Kern jeder Browser-Engine. Die zweite ist der „Web-Platform/Host“-Teil.

### 6.9 Event-System

**Reuse:** Dispatch, HitTest, Focus sind da.  
**Neubau:** DOM Events incl. capture/target/bubble Semantik, default actions, cancelation, synthetic events, pointer/keyboard/text-input mapping.

**Komplexität:** Mittel.

**Schätzung:** ~6–14 KLOC.

### 6.10 Accessibility

**Reuse:** Stark (A11y Tree + Bridges).  
**Neubau:** ARIA-Rollenmapping aus DOM/CSS, Name/Description-Berechnung, dynamische A11y-Updates.

**Komplexität:** Mittel bis hoch.

**Schätzung:** ~7–16 KLOC.

### 6.11 Formular-Controls

**Reuse:** Sehr hoch durch bestehende Lux-Widgets.  
**Neubau:** HTML-Form-Semantik (form owner, submission encoding, validation model, constraint API).

**Komplexität:** Mittel.

**Schätzung:** ~6–14 KLOC.

### 6.12 Teststrategie & Konformität (automatisiert)

Die zentrale Lehre aus Servo/Blink/Gecko/WebKit: Browser-Kompatibilität entsteht nicht primär durch Code, sondern durch **kontinuierliche Spezifikations-Tests**.

**Testpyramide (empfohlen):**

1. **Unit-Tests (schnell, lokal):**  
   Parser-Bausteine, CSS-Cascade-Regeln, Layout-Algorithmen, Event-Dispatch, DOM-Mutationen.
2. **Deterministische Engine-Integrationstests:**  
   HTML→DOM→Style→Layout→Paint auf künstlichen Fixtures; Snapshot/Pixel- oder Strukturvergleiche.
3. **Web Platform Tests (WPT):**  
   Normative API-/Verhaltens-Tests gegen Web-Standards als primäre Kompatibilitätsmetrik.
4. **Reftests/Visual-Tests:**  
   Referenz-Rendering-Vergleiche (A/B-Seiten), nützlich für CSS/Layout/painting-Regressions.
5. **Intermittency/Flake-Monitoring in CI:**  
   Wiederholte Läufe, Quarantäne, getrennte Erwartungsdateien für known-fail/known-flaky.

**Wie Servo das praktisch macht (als Vorbild):**

- `mach test-unit` für Unit-Tests
- `mach test-wpt` für WPT-Integration
- `mach test-tidy`/`mach fmt` für Hygiene
- Erwartungs-/Manifest-Updates über `mach update-manifest` bzw. WPT-Update-Workflows
- Optionaler WebDriver-basierter Harness (`servodriver`) für WPT-Ausführung

**Übertrag auf Lux-Browser-Track (Vorschlag):**

- Pro Subsystem einen festen Testordner (`tests/dom`, `tests/css`, `tests/layout`, `tests/wpt`).
- CI-Gates je PR:
  - Gate A: Unit + schnelle Integration (Pflicht)
  - Gate B: gezielte WPT-Sets nach betroffenem Bereich (Pflicht)
  - Gate C: breiter Nightly-WPT-Lauf inkl. Flake-Analyse (pflichtig vor Release-Milestones)
- Öffentliche Fortschrittsmetrik:
  - Passrate nach Subtree (DOM, CSS, HTML, Events, Forms)
  - Delta je Woche/Monat
  - Anteil neuer Regressionen vs. behobener Failures

---

## 7. Aufwandsmatrix

| Subsystem | Komplexität | Lux-Reuse | Extern | Neubau-Anteil | Δ seit v0.1.0 |
|---|---|---|---|---|---|
| HTML Parsing & DOM | Hoch | Niedrig-Mittel | Parser ja | Hoch | — |
| CSS Parsing & Cascade | Hoch | Niedrig | optional | Sehr hoch | — |
| Layout Core | Sehr hoch | **Hoch** | nein | **Hoch** | ⬆ Table Layout (1038 LOC) + Inline Widgets reduzieren Neubau |
| Painting/Compositing | Mittel-Hoch | Sehr hoch | nein | Mittel | — |
| Text/Fonts | Mittel-Hoch | Sehr hoch | nein | Mittel | ⬆ Async MSDF-Atlas verbessert Performance |
| Networking/Loader | Hoch | Niedrig | stdlib ja | Hoch | — |
| JS Runtime Integration | Hoch | Niedrig | JS Engine ja | Hoch | — |
| DOM↔JS Bridge | Sehr hoch | Niedrig | nein | Sehr hoch | — |
| Events | Mittel | Hoch | nein | Mittel | — |
| Accessibility | Mittel-Hoch | Hoch | nein | Mittel | — |
| Formular-Semantik | Mittel | **Sehr hoch** | nein | **Niedrig-Mittel** | ⬆ 5 neue Controls (Date/Color/Time/Numeric/Spinner) |
| SVG / Vektorgrafik | Mittel | **Hoch** | nein | **Mittel** | ⬆ Neu: GPU-beschleunigte SVG-Pipeline |

**Grobe Gesamtordnung (aktualisiert):**

- **MVP „statischer HTML/CSS-Viewer”:** ca. **30–60 KLOC** (↓ gegenüber 35–70 KLOC durch Table Layout, SVG, Form-Controls und Inline-Layout-Reuse)
- **Interaktives DOM + JS-Teilmenge:** ca. 75–150 KLOC (↓ leicht)
- **Robust gegen reale Websites:** realistisch >200 KLOC + lange Stabilisierung

(Die Zahlen sind bewusst breit — der Variationsfaktor hängt stark am CSS- und JS-API-Scope. Die Reduktion für den MVP-Track ist spürbar, die oberen Stufen profitieren weniger, da dort DOM/JS-Bridge/Web-APIs dominieren.)

---

## 8. Vergleich mit existierenden Engines

| Engine | Größenordnung | Organisation |
|---|---|---|
| Blink (Chromium) | Mehrere Mio. LOC gesamt | Großes Vollzeit-Team über viele Jahre |
| WebKit | Mehrere Mio. LOC gesamt | Langjähriges Multi-Team-Projekt |
| Gecko | Mehrere Mio. LOC gesamt | Große Organisation + Historie |
| Servo | Deutlich kleiner als obige, aber trotzdem groß | Jahrelange Forschung/Entwicklung |

**Einordnung:** Selbst mit starkem Lux-Unterbau bleibt der fehlende Teil (DOM/CSSOM/Layout/JS-Bridge/Web-APIs) groß genug, dass ein „vollwertiger Browser“ ein Multi-Jahresprojekt für ein dediziertes Team wäre.

---

## 9. Realistische Einordnung

### Was realistisch ist

- Ein kontrollierter „Web Document Renderer“ für interne/kuratierte Inhalte
- Gute Performance in Text-/UI-lastigen Dokumenten durch Lux-Renderer
- Exzellente Integration in Lux-Apps (Look-and-Feel, A11y, Input)

### Was kurzfristig unrealistisch ist

- Kompatibilität mit dem offenen Web auf Browser-Niveau
- Vollständige CSS-/JS-/Web-API-Abdeckung
- Sicherheitsniveau etablierter Browserarchitekturen

---

## 10. Phasenmodell

### Phase 1 — Static HTML/CSS Viewer

- HTML Parse → DOM
- CSS Teilmenge (Typ-, Klassen-, ID-Selektoren; Basiseigenschaften)
- Block + einfache Inline-Layoutregeln
- Kein JS

### Phase 2 — Interaktive Dokumente (ohne volle Web-API)

- DOM Events (Klick, Fokus, Keyboard)
- Form Controls Rendering + Basissubmission
- Einfache Navigation/History

### Phase 3 — Scripted UI

- JS Engine Integration
- DOM Mutationen via JS
- Grundlegende Task/Microtask Integration

### Phase 4 — Breiteres App-Web

- Erweiterte CSS Features (flex/grid deutlich tiefer, table, positioning)
- Mehr Web APIs (`fetch`, Storage, History-Details)
- DevTools-Basics

**Hinweis:** Jede Phase ist als eigenständig nutzbares Ziel zu definieren; kein Big-Bang.

---

## 11. Synergie-Effekte für Lux

| Browser-Subsystem | Rückfluss in Lux | Status |
|---|---|---|
| CSS Flexbox spec-konform | ✅ Bereits CSS-Spec-konform in `ui/layout/flex.go` | Erledigt |
| CSS Grid spec-näher | ✅ Bereits CSS-Spec-konform in `ui/layout/grid.go` | Erledigt |
| Table Layout | ✅ Bereits HTML-Spec-konform in `ui/layout/table.go` | Erledigt |
| Inline Layout / Replaced Elements | ✅ `InlineWidget` (Inline + Block-Modus), `ImageSpan` (Float), Listen (ul/ol), Link-Widget, CSS-Paragraph-Styling | Erledigt (erweitert v0.3.0) |
| SVG Pipeline | ✅ GPU-beschleunigte SVG-Vektorisierung in `image/svg.go` | Erledigt (neu) |
| Form-Controls für HTML-Inputs | ✅ DatePicker, ColorPicker, TimePicker, NumericInput, Spinner | Erledigt (neu) |
| contenteditable-nahe Logik | ✅ RichTextEditor in `richtext/` — Tagged-Range AttributedString, 17 Attribut-Typen, ToolbarCommands, Listen-Support | Erledigt (erweitert v0.3.0) |
| DevTools-Protokolle | ✅ Inspector-PoC via RFC-012 — Vellum-basiertes Debug-Protokoll | Erledigt |
| Code Editor (RFC-010) | Syntax-Highlighting, LSP, Multi-Cursor für DevTools-Source-Views | Design fertig, Implementierung ausstehend |

**Strategischer Punkt:** Seit der Erstfassung dieses RFC wurden 8 der 11 identifizierten Synergie-Investitionen realisiert. Der verbleibende Baustein (Code Editor, RFC-010) ist die letzte architektonisch anspruchsvolle Synergie-Investition.

---

## 12. Risiken & offene Fragen

1. **Scope Creep:** Web-Standards ziehen schnell unkontrolliert große Featureflächen nach.
2. **Security Debt:** CORS/CSP/Sandboxing/URL-Handling sind nicht optional.
3. **JS Bridge-Komplexität:** Laufzeitgrenzen und Memory-Lifetime sind fehleranfällig.
4. **Spec-Compliance vs. Delivery:** Jede Abkürzung erzeugt Kompatibilitätskosten.
5. **Testaufwand:** Ohne WPT-nahe Teststrategie droht instabiles Verhalten.
6. **Ressourcenfrage:** Teamgröße und Dauer entscheiden stärker als reine Architektur.

**Offene Leitfrage:** Soll Lux mittelfristig eher „beste Integration bestehender Engines“ (RFC-004) oder „kuratierter eigener Document-Engine-Track“ priorisieren?

---

## 13. Fazit

Eine eigene Browser-Engine auf Lux-Basis ist **technisch denkbar**, weil Lux im unteren Stack (Rendering, Text, Input, A11y) bereits sehr stark ist.

Gleichzeitig bleibt der fehlende obere Stack (DOM/CSSOM/Layout/JS-Bridge/Security/Web-APIs) so groß, dass ein vollwertiger Browser-Anspruch kurzfristig nicht realistisch ist.

**Empfohlene Lesart von RFC-998:**

- Nicht als Produkt-Roadmap für „Lux Browser”.
- Sondern als Architektur-Kompass für selektive Investitionen mit hohem Rückfluss in Lux selbst (DataTable, RichTextEditor, SVG, Inspector, Layout-Engine-Qualität).

---

## 14. Re-Evaluierung v0.2.0 (2026-03-26)

### Was sich seit der Erstfassung geändert hat

Seit der Erstfassung (v0.1.0, ebenfalls 2026-03-26) wurde in kurzer Zeit erheblich nachgelegt. Die Änderungen betreffen primär den **Reuse-Anteil** — also den Teil des Stacks, den Lux bereits für eine hypothetische Browser-Engine mitbringt.

#### Neu integrierte Komponenten

| Komponente | Dateien | Browser-Relevanz |
|---|---|---|
| **SVG-Rendering** | `image/svg.go` (785 LOC), `image/svgpath.go` (697 LOC) | `<svg>` — GPU-beschleunigte Pfad-Vektorisierung; Elementtypen path/rect/circle/ellipse/line/polygon/polyline/g |
| **Inline/Block Widgets + Listen in RichText** | `ui/display/richtext.go` (897 LOC) | Replaced Elements im Inline-/Block-Flow, Float-Bilder, Listen (ul/ol mit Nesting/Marker), CSS-Paragraph-Styling — Kernbaustein für gemischten Text-/Widget-Satz |
| **DatePicker** | `ui/form/datepicker.go` (352 LOC) | `<input type=”date”>` |
| **ColorPicker** | `ui/form/colorpicker.go` (270 LOC) | `<input type=”color”>` |
| **TimePicker** | `ui/form/timepicker.go` (308 LOC) | `<input type=”time”>` |
| **NumericInput** | `ui/form/numericinput.go` (273 LOC) | `<input type=”number”>` mit Stepper und Drag-to-Adjust |
| **Spinner** | `ui/form/spinner.go` (118 LOC) | Animierter Ladeindikator |
| **Async MSDF-Atlas** | Font-Pipeline | Performance-Verbesserung für Text-Rendering; Bitmap-Fallback |
| **SplitView** | `ui/nav/splitview.go` | Resizable Panels (DevTools-Layout) |

#### Auswirkung auf die Subsystem-Analyse

1. **Layout (§6.3):** Table Layout (1038 LOC) und Inline Widgets reduzieren den Neubau-Anteil für CSS Formatting Contexts. Reuse-Einstufung steigt von „Mittel” auf „Hoch”. Die Schätzung sinkt leicht auf **~22–50 KLOC**.

2. **Painting/SVG (§6.4):** SVG-Pipeline (1482 LOC) macht `<svg>`-Rendering im MVP realistisch, ohne separaten SVG-Stack aufbauen zu müssen. Painting-Neubau bleibt bei ~10–20 KLOC, aber SVG kommt quasi „gratis” dazu.

3. **Formular-Controls (§6.11):** Mit DatePicker, ColorPicker, TimePicker, NumericInput und Spinner sind nun **fast alle** gängigen HTML-Input-Typen als native Widgets vorhanden. Neubau reduziert sich auf HTML-Form-Semantik (submission, validation, constraint API). Schätzung sinkt auf **~4–10 KLOC**.

4. **Text/Inline (§6.5):** Inline/Block Widgets, Float-Bilder, Listen (ul/ol) und CSS-Paragraph-Styling im RichText schaffen eine breite Grundlage für CSS Inline + Block Formatting Contexts (replaced elements, floats, list-style). Die Brücke zwischen Lux-RichText und Browser-Inline-Layout wird deutlich kürzer.

#### Gesamtbewertung

| Metrik | v0.1.0 | v0.2.0 | v0.3.0 | Δ (v0.2→v0.3) |
|---|---|---|---|---|
| Reuse-Komponenten (§3.1) | 27 | 31 | **33** | +2 (Link-Widget, Listen-Rendering) |
| Synergie-Investitionen realisiert (§11) | 3/11 | 6/11 | **8/11** | +2 (CSS-Paragraph-Styling, Link/`<a>`) |
| MVP-Schätzung | 35–70 KLOC | 30–60 KLOC | **28–55 KLOC** | ↓ ~8% |
| Interaktiv + JS | 80–160 KLOC | 75–150 KLOC | 72–145 KLOC | ↓ ~3% |
| Robust / Full Web | >220 KLOC | >200 KLOC | >195 KLOC | ↓ marginal |

#### Was sich **nicht** geändert hat

Die Kernaussage von RFC-998 bleibt bestehen:

- **DOM/CSSOM/JS-Bridge/Security** dominieren weiterhin den Aufwand und sind unverändert groß.
- Der **obere Stack** (>60% des Gesamtaufwands) profitiert kaum von den neuen Lux-Komponenten.
- Ein vollwertiger Browser bleibt ein **Multi-Jahresprojekt für ein dediziertes Team**.

#### Neue strategische Beobachtungen

1. **RFC-010 (Code Editor)** ist vollständig designed. Bei Umsetzung entsteht der letzte verbleibende Synergie-Baustein für DevTools/Inspector.

2. **RFC-011 (Vellum / Remote Rendering)** eröffnet ein theoretisches Alternativszenario: Statt einer eigenen Browser-Engine könnte ein schlanker „Document Renderer” über Vellum remote gestreamt werden — konzeptuell interessant, aber ebenfalls rein theoretisch.

3. **RFC-004 (WebView)** ist auf post-V1 zurückgestellt. Bisherige Implementierung im Branch `feature/webview` gesichert. Für den pragmatischen Pfad bleibt Embedding bestehender Engines die empfohlene Strategie für reale Web-Inhalte.

4. **RichTextEditor** ist seit v0.1.0 integriert und in v0.3.0 erheblich erweitert: Tagged-Range `AttributedString` (17 Attribut-Typen), CSS-Paragraph-Styling (Align/Indent/Spacing), Listen-Support (ul/ol, Nesting, 9 Marker-Stile), Inline-Font-Formatting, ToolbarCommands. Die `contenteditable`-nahe Logik ist damit über das Basic-Niveau hinaus realisiert.

5. **Inspector/DevTools** ist als PoC via RFC-012 integriert (Vellum-basiert: Widget-Tree, Layout-Overlay, Event-Log, State-Dump, Frame-Metriken). Das Debug-Protokoll-Fundament steht.

6. **Nächster Hebel für RFC-998:** Der einzig verbleibende Synergie-Baustein ist der **Code Editor (RFC-010)**. Seine Realisierung würde den MVP-Track weiter senken und gleichzeitig Lux als Framework aufwerten — unabhängig davon, ob je eine Browser-Engine gebaut wird.

#### Empfehlung

Die Empfehlung von v0.1.0 wird **bestätigt — 2 von 3 priorisierten Investitionen sind realisiert:**

> RFC-998 als Architektur-Kompass nutzen. Status der priorisierten Synergie-Investitionen:
> 1. ~~**RichTextEditor**~~ — ✅ Integriert (`richtext/`, Toolbar, Bild-Support)
> 2. ~~**Inspector/DevTools**~~ — ✅ Integriert (RFC-012 Inspector-Vellum PoC)
> 3. **Code Editor (RFC-010)** — Verbleibt als letzte Synergie-Investition (Syntax + LSP, Fundament für Source-View in DevTools)
>
> Der Code Editor ist der letzte verbliebene Doppelnutzen-Baustein: Lux-Framework-Wert **und** Browser-Engine-Readiness.

### 14b. Re-Evaluierung v0.3.0 (2026-03-31)

#### Neu seit v0.2.0

| Komponente | Dateien | Browser-Relevanz |
|---|---|---|
| **Tagged-Range AttributedString** | `richtext/document.go` (797 LOC) | NSAttributedString-nahes Modell mit 17 Attribut-Typen — direkte Grundlage für CSS-Cascading auf Inline-Ebene |
| **CSS-Paragraph-Styling** | `richtext/document.go`, `ui/display/richtext.go` | `text-align`, `text-indent`, `line-height`, Paragraph-Spacing — Block Formatting Context nähert sich CSS-Spec |
| **Listen (ul/ol)** | `draw/list.go` (26 LOC), `richtext/command.go`, `ui/display/richtext.go` | `list-style-type` mit 9 Marker-Stilen, Nesting (0–8), `<ol start>` — vollständiges HTML-Listen-Rendering |
| **Inline Font Formatting** | `richtext/document.go`, `richtext/command.go` | Bold/Italic/Underline/Strikethrough als Toggle-Commands — `contenteditable`-nahe Toolbar-Logik |
| **Block-Modus InlineWidget** | `ui/display/richtext.go` | CSS `display: block` für Replaced Elements im Textfluss |
| **Link-Widget** | `ui/link/link.go` (235 LOC) | HTML `<a>` — klickbarer Inline-Link mit Hover/Focus, A11y, Theme-Support |
| **ToolbarCommands** | `richtext/command.go` (286 LOC) | Pluggbare Editor-Commands (Default, Alignment, List) — erweiterbar für Custom-Formatierung |

#### Auswirkung

1. **Text/Inline (§6.5):** Der RichText-Stack deckt jetzt Inline Formatting, Block Formatting, Float-Bilder, Listen und Paragraph-Styling ab. Der Reuse-Anteil für CSS Text/Inline steigt von „Hoch" auf „Sehr Hoch".

2. **Formular-Controls (§6.11):** Mit dem Link-Widget ist nun auch HTML `<a>` als nativer Baustein vorhanden.

3. **Reuse-Komponenten:** +2 (Link-Widget, Listen-Rendering) gegenüber v0.2.0.

#### Empfehlung

Empfehlung von v0.2.0 bleibt bestätigt. Der RichText-Stack ist jetzt der am weitesten entwickelte Subsystem-Bereich relativ zu einem Browser-MVP.
