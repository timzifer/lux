# RFC-998 — Eigene Browser-Engine auf Basis des Lux UI-Kits

**Repository:** `github.com/timzifer/lux`

**Status:** Very Theoretical — nicht zur Umsetzung vorgesehen
**Version:** 0.1.0
**Datum:** 2026-03-26
**Zuletzt abgeglichen:** 2026-03-26
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
| RichText | `ui/display/richtext.go` | Inline Text-Runs | Wichtiger Startpunkt für Inline-Layout |
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
| Form-Controls | `ui/form/*.go` | `<input>`, `<select>`, `<textarea>`, `<progress>` | Viele Controls bereits als native Widgets |
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

**Zwischenfazit:** Der „untere“ Teil der Rendering-Pipeline (Text, Paint, Compositor, Input, A11y) ist im Vergleich zu typischen Hobby-Engine-Projekten außergewöhnlich weit.

---

## 4. Geplante Lux-Komponenten mit Browser-Synergie

| Lux-Komponente | Status | Browser-Äquivalent | Synergie |
|---|---|---|---|
| DataTable / CSS Table Layout | ✅ Table Layout integriert | `<table>` | `ui/layout/table.go` — HTML-Spec-konformes CSS Table Layout (Fixed + Auto); DataTable-Widget darauf aufbauend ausstehend |
| DatePicker | ⏳ Phase 7.1 | `<input type="date">` | Native Browser-Controls leichter abbildbar |
| ColorPicker | ⏳ Phase 7.1 | `<input type="color">` | Direktes Control-Mapping |
| Toolbar | ⏳ Phase 7.1 | Browser-Chrome | Adressleiste/Navigation/UI-Shell |
| FilePicker | ⏳ Phase 7.1 | `<input type="file">` | Upload-Flow mit OS-Dialog |
| RichTextEditor | ⏳ Phase 7.1 | `contenteditable` | Gemeinsame Editier-/Selection-Logik |
| Inline Widgets in RichText | ⏳ Phase 4.7 | Replaced Elements Inline | Kernstück für gemischten Text-/Widget-Flow |
| Code Editor | 🔶 RFC-010 | Script/CSS-Editor in DevTools | Reuse für Inspektor-/Source-Views |
| SVG Support | ⏳ Stub (`image/svg.go`) | `<svg>` | Vektorpfad für Web-Inhalte und Lux |
| Inspector/DevTools | ⏳ Phase 6.7 | Browser DevTools | Debug-Protokolle direkt wiederverwertbar |
| DynamicDataset | ⏳ Phase 6.6 | Infinite/Lazy DOM | Large-Page-Strategien für beide Welten |

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

### 6.2 CSS Parsing & Cascade

**Extern:** Optional Parser-Libs; oft sinnvoll: eigener schlanker Parser für kontrollierte Teilmenge.  
**Neubau:** Selector Engine, Specificity, Inheritance, Computed Values, Initial Values, Shorthands.  
**Komplexität:** Hoch.

**Schätzung:** ~12–25 KLOC.

### 6.3 Layout Engine (Block/Inline/Flex/Grid/Table)

**Reuse:** Teile von `flex.go`, `grid.go`, RichText/Textlayout.  
**Neubau:** Voller CSS-Formatting-Context (anonymous boxes, line boxes, containing blocks, margin-collapsing, positioning, floats, fragmentation optional).

**Komplexität:** Sehr hoch (kritischer Pfad).

**Schätzung:** ~35–70 KLOC (ohne Print/Paged Media).

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

---

## 7. Aufwandsmatrix

| Subsystem | Komplexität | Lux-Reuse | Extern | Neubau-Anteil |
|---|---|---|---|---|
| HTML Parsing & DOM | Hoch | Niedrig-Mittel | Parser ja | Hoch |
| CSS Parsing & Cascade | Hoch | Niedrig | optional | Sehr hoch |
| Layout Core | Sehr hoch | Mittel | nein | Sehr hoch |
| Painting/Compositing | Mittel-Hoch | Sehr hoch | nein | Mittel |
| Text/Fonts | Mittel-Hoch | Sehr hoch | nein | Mittel |
| Networking/Loader | Hoch | Niedrig | stdlib ja | Hoch |
| JS Runtime Integration | Hoch | Niedrig | JS Engine ja | Hoch |
| DOM↔JS Bridge | Sehr hoch | Niedrig | nein | Sehr hoch |
| Events | Mittel | Hoch | nein | Mittel |
| Accessibility | Mittel-Hoch | Hoch | nein | Mittel |
| Formular-Semantik | Mittel | Hoch | nein | Mittel |

**Grobe Gesamtordnung:**

- **MVP „statischer HTML/CSS-Viewer“:** ca. 40–80 KLOC
- **Interaktives DOM + JS-Teilmenge:** ca. 90–180 KLOC
- **Robust gegen reale Websites:** realistisch >250 KLOC + lange Stabilisierung

(Die Zahlen sind bewusst breit — der Variationsfaktor hängt stark am CSS- und JS-API-Scope.)

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

| Browser-Subsystem | Rückfluss in Lux |
|---|---|
| CSS Flexbox spec-konform | ✅ Bereits CSS-Spec-konform in `ui/layout/flex.go` |
| CSS Grid spec-näher | ✅ Bereits CSS-Spec-konform in `ui/layout/grid.go` |
| Table Layout | ✅ Bereits HTML-Spec-konform in `ui/layout/table.go` |
| Inline Layout | Basis für Inline Widgets in RichText |
| contenteditable-nahe Logik | Fundament für RichTextEditor |
| SVG Pipeline | Vervollständigt `image/svg.go` |
| DevTools-Protokolle | Stärkung Inspector/Debug-Tooling |

**Strategischer Punkt:** Selbst wenn nie ein vollständiger Browser entsteht, sind zentrale Teilinvestitionen direkt für Lux-Widgets und Entwickler-Tools verwertbar.

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

- Nicht als Produkt-Roadmap für „Lux Browser“.
- Sondern als Architektur-Kompass für selektive Investitionen mit hohem Rückfluss in Lux selbst (DataTable, RichTextEditor, SVG, Inspector, Layout-Engine-Qualität).
