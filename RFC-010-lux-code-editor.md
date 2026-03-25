# RFC-010 — lux: Nativer Code-Editor als Widget

**Repository:** `github.com/timzifer/lux`

**Status:** Theoretical
**Version:** 0.1.0
**Datum:** 2026-03-25
**Abhängigkeit:** RFC-001 (Core Architecture), RFC-007 (WGPU Rendering Pipeline)

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Abgrenzung](#2-abgrenzung)
3. [Evaluierung: Externe Editoren einbetten](#3-evaluierung-externe-editoren-einbetten)
4. [Evaluierung: Lapce-Proxy als Backend (Option A)](#4-evaluierung-lapce-proxy-als-backend-option-a)
5. [Empfehlung: Nativer Go-Editor (Option B)](#5-empfehlung-nativer-go-editor-option-b)
6. [Architektur-Überblick](#6-architektur-überblick)
7. [Modul 1: Buffer — Rope/Piece-Table](#7-modul-1-buffer--ropepiece-table)
8. [Modul 2: Syntax-Highlighting — Tree-sitter + Chroma](#8-modul-2-syntax-highlighting--tree-sitter--chroma)
9. [Modul 3: LSP-Client](#9-modul-3-lsp-client)
10. [Modul 4: Editor-Widget](#10-modul-4-editor-widget)
11. [Vorhandene Lux-Bausteine](#11-vorhandene-lux-bausteine)
12. [Go-Ökosystem: Abhängigkeiten & Reifegrad](#12-go-ökosystem-abhängigkeiten--reifegrad)
13. [Offene Fragen](#13-offene-fragen)

---

## 1. Motivation & Ziel

Viele Lux-Anwendungen benötigen eine Code-Editing-Fähigkeit — sei es für Konfigurationsdateien, Scripting-Konsolen, Log-Inspektion oder vollwertige Entwicklungsumgebungen. RFC-004 evaluiert die Einbettung von Web-Editoren (Monaco/CodeMirror) über WebView-Surfaces, aber ein **nativer, WGPU-gerenderter Code-Editor als erstklassiges Lux-Widget** bietet:

- Nahtlose Integration in den Widget-Tree (Theming, A11y, Fokus, Animationen)
- Keine externe Runtime-Abhängigkeit (kein Chromium, kein Rust-Binary)
- Volle Kontrolle über Rendering-Performance in der WGPU-Pipeline
- Ein einzelnes Go-Binary als Deployment-Artefakt

**Ziel:** Spezifikation eines modularen, nativen Code-Editor-Widgets für Lux, das als `ui/editor`-Package bereitgestellt wird.

---

## 2. Abgrenzung

| In Scope | Nicht in Scope |
|---|---|
| Text-Buffer mit Rope/Piece-Table für Dateien bis 100+ MB | Vollständige IDE (Projekt-Explorer, Debugger-UI, etc.) |
| Syntax-Highlighting via Tree-sitter + Chroma-Fallback | Eigene Language Server entwickeln |
| LSP-Client für Completion, Hover, Diagnostics, Go-to-Definition | Collaborative Editing (CRDT) — evaluiert, aber nicht in v1 |
| Line Numbers, Gutter, Minimap, Code Folding | Terminal-Emulator (separates Widget) |
| Multi-Cursor, Rectangular Selection | Vim/Emacs-Modi in v1 (erweiterbar designed) |
| Undo/Redo mit History-Tree | Plugin-System (WASI oder Lua) |
| Search & Replace (Regex) | Git-Integration (separates Modul) |

---

## 3. Evaluierung: Externe Editoren einbetten

### 3.1 Zed (GPUI-basiert, Rust)

| Aspekt | Bewertung |
|---|---|
| Editor als Library | **Nicht möglich.** ~40+ intern verkoppelte Crates, kein Library-API |
| GPUI (UI-Framework) | Separat nutzbar, Apache-2.0, auf crates.io publiziert (`gpui` v0.2.2) |
| Editor-Widget für GPUI | `gpui-component` (Longbridge) bietet Code-Editor mit LSP auf GPUI-Basis |
| Rendering | Seit März 2026 auf wgpu migriert (vorher Blade) |
| C FFI / Go-Bindings | **Existieren nicht.** Trait/Closure-basiertes API macht FFI extrem schwierig |
| Lizenz Editor | GPL-3.0-or-later (Editor-Crates), Apache-2.0 (GPUI, Utility-Crates) |
| Lizenz GPUI | Apache-2.0 |

**Ergebnis:** GPUI ist als eigenständiges UI-Framework nutzbar (15+ Standalone-Apps in `awesome-gpui`), aber der Zed-Editor selbst ist nicht extrahierbar. Keine FFI-Brücke nach Go. Für Lux nicht direkt verwendbar.

### 3.2 Lapce (Floem-basiert, Rust)

| Aspekt | Bewertung |
|---|---|
| Editor als Library | **Nicht möglich.** `lapce-app` ist monolithisch, Crates nicht auf crates.io |
| Floem (UI-Framework) | Separat nutzbar, MIT-Lizenz, auf crates.io (`floem` v0.2.0) |
| `floem-editor-core` | Buffer, Cursor, Selection, Vim-Modi — **unabhängig nutzbar** (MIT, crates.io) |
| Rendering | vger (default), Skia, Vello, tiny_skia — kein wgpu |
| C FFI / Go-Bindings | **Existieren nicht** |
| Proxy-Architektur | Frontend/Backend getrennt via JSON-RPC über stdio |
| Lizenz | Apache-2.0 (Lapce), MIT (Floem), MPL-2.0 (Syntax aus Helix) |

**Ergebnis:** Floem + `floem-editor-core` sind architektonisch sauber getrennt, aber rein Rust ohne FFI. Der Lapce-Proxy ist das einzige Stück das sprachunabhängig ansprechbar wäre (§4).

### 3.3 VS Code (Electron/Web)

| Aspekt | Bewertung |
|---|---|
| Einbettung | Möglich via RFC-004 WebView-Surface (vscode.dev in WebView2) |
| Rendering | Chromium rendert → GPU-Texture → WGPU-Compositing |
| Nachteile | Externe Runtime (Chromium), hoher RAM (~150 MB+), keine native Widget-Integration |

**Ergebnis:** Funktional, aber widerspricht Lux' Leichtgewichts-Philosophie. Valider Fallback, wenn ein nativer Editor zu aufwendig ist.

---

## 4. Evaluierung: Lapce-Proxy als Backend (Option A)

### 4.1 Architektur

```
┌─────────────────┐      JSON-RPC/stdio      ┌─────────────────┐
│   Lux-Frontend   │ ◄──────────────────────► │  lapce-proxy    │
│   (Go + WGPU)    │                           │  (Rust-Binary)  │
│                   │   ProxyNotification:      │                 │
│  • Editor-Widget  │   Update(RopeDelta)       │  • xi-rope      │
│  • Gutter/Minimap │                           │  • LSP-Client   │
│  • Completion-UI  │   CoreNotification:       │  • Tree-sitter  │
│  • Diagnostics-UI │   CompletionResponse,     │  • Git (git2)   │
│                   │   PublishDiagnostics,      │  • Search (rg)  │
│                   │   SemanticStyles           │  • Wasmtime     │
└─────────────────┘                            └─────────────────┘
```

### 4.2 Proxy-Protokoll (Detail)

**Wire-Format:** Newline-delimited JSON (nicht strikt JSON-RPC 2.0, kein `"jsonrpc": "2.0"` Feld).

**ProxyRequest** (Request/Response):
- Buffer: `NewBuffer`, `BufferHead`, `Save`, `SaveBufferAs`
- LSP Navigation: `GetDefinition`, `GetTypeDefinition`, `GetReferences`, `GotoImplementation`
- LSP Intelligence: `GetHover`, `GetSignature`, `CompletionResolve`, `GetInlayHints`, `GetSemanticTokens`
- LSP Actions: `GetCodeActions`, `PrepareRename`, `Rename`, `GetDocumentFormatting`
- Search: `GlobalSearch`
- File-Ops: `GetFiles`, `ReadDir`, `CreateFile`, `TrashPath`, `RenamePath`
- Git: `GitGetRemoteFileUrl`

**ProxyNotification** (Fire-and-forget):
- Editing: `Update` (sendet `RopeDelta`), `OpenFileChanged`
- Lifecycle: `Initialize`, `Shutdown`
- Terminal: `NewTerminal`, `TerminalWrite`, `TerminalResize`
- Plugins: `InstallVolt`, `RemoveVolt`, `EnableVolt`

**CoreNotification** (Proxy → Frontend):
- `CompletionResponse`, `PublishDiagnostics`, `DiffInfo`, `SemanticStyles`
- `UpdateTerminal`, `WorkDoneProgress`, `ShowMessage`

### 4.3 Syntax-Highlighting via Proxy

Der Proxy liefert **voraufgelöste Farb-Ranges** (keine Token-Typen):

```
SemanticStyles {
    rev:    u64,           // Buffer-Revision für Staleness-Detection
    path:   PathBuf,
    styles: Vec<LineStyle>,  // pro Zeile: start, end (Byte-Offsets), fg_color (String)
}
```

Tree-sitter/LSP Semantic Tokens werden proxy-intern aufgelöst. Das Frontend malt nur farbige Spans.

### 4.4 Kritische Kopplungspunkte

| Punkt | Schwierigkeit | Beschreibung |
|---|---|---|
| **RopeDelta-Format** | **Hoch** | Buffer-Edits müssen als `lapce-xi-rope` `RopeDelta` serialisiert werden — ein komplexes CRDT-Delta-Format. In Go nachbauen = erheblicher Aufwand |
| **Doppelte Buffer-Haltung** | Mittel | Proxy hält Rope, Frontend braucht eigene Kopie für Rendering → Synchronisierung nötig |
| **Undo/Redo** | Mittel | Proxy hat keinen Undo-Stack. Frontend muss Deltas invertieren und selbst verwalten |
| **Volt-Plugin-System** | Niedrig | Lapce-spezifisch, kann ignoriert oder minimal unterstützt werden |

### 4.5 Bewertung Option A

| Dimension | Bewertung |
|---|---|
| Performance | ★★★☆☆ — JSON-Serialisierung pro Keystroke, doppelte Buffer-Haltung |
| Aufwand | ★★★☆☆ — ~8-12 Wochen, RopeDelta-Compat ist versteckter Hauptaufwand |
| Freiheit | ★★☆☆☆ — Proxy-Protokoll definiert das Feature-Ceiling |
| Deployment | ★★☆☆☆ — +30 MB Rust-Binary, Cross-Compilation für alle Plattformen |
| Wartbarkeit | ★★☆☆☆ — Abhängig von Lapce-Releases, Breaking Changes im RPC möglich |

---

## 5. Empfehlung: Nativer Go-Editor (Option B)

### 5.1 Begründung

Option B (alles in Go, im selben Prozess) wird empfohlen:

**Performance:**
- Kein Serialisierungs-Overhead — Buffer-Edit ist ein Funktionsaufruf (~0 ms vs. ~1-5 ms)
- Tree-sitter inkrementell im gleichen Prozess, nutzt unveränderte Subtrees
- Ein Buffer, eine Wahrheit — keine Synchronisierung zwischen Prozessen
- LSP-Kommunikation: eine Hop weniger (direkt an Language Server, ohne Proxy-Zwischenschritt)

**Freiheit:**
- Freie Wahl der Buffer-Datenstruktur (Rope, Piece-Table, Hybrid)
- Tree-sitter + Chroma als Dual-Strategy (strukturell + Regex-Fallback für 200+ Sprachen)
- Multi-Cursor, Custom-Completion, AI-Completion, Domain-spezifische Features — keine Limits
- Buffer/AST/Cursor leben im selben Prozess wie der Lux Widget-Tree → direkte A11y-Anbindung

**Deployment:**
- Ein Go-Binary für alle Plattformen (Tree-sitter braucht CGo, aber kein externes Binary)
- Keine Abhängigkeit von Lapce-Releases oder Proxy-Protokoll-Stabilität

**Aufwand:**
- Geschätzt ~12-16 Wochen (4 Wochen mehr als Option A)
- Der Mehraufwand zahlt sich zurück: LSP-Client ist wiederverwendbar, Buffer-Modul ist eigenständig testbar, keine RopeDelta-Kompatibilitätsarbeit

### 5.2 Vergleich Option A vs. Option B

```
                    Option A              Option B
                    (Lapce-Proxy)         (Nativer Go-Editor)
─────────────────────────────────────────────────────────────
Performance         ★★★☆☆                 ★★★★★
                    JSON-Serialisierung   In-Process

Aufwand             ★★★☆☆                 ★★☆☆☆
                    ~10 Wochen            ~14 Wochen

Freiheit            ★★☆☆☆                 ★★★★★
                    Proxy = Ceiling       Keine Limits

Wartbarkeit         ★★☆☆☆                 ★★★★☆
                    Externe Dep           Eigener Code

Deployment          ★★☆☆☆                 ★★★★★
                    +30 MB Rust-Binary    Ein Go-Binary
```

---

## 6. Architektur-Überblick

```
ui/editor/
├── editor.go            // Editor-Widget (implementiert ui.Widget)
├── buffer/
│   ├── rope.go          // Rope-Datenstruktur
│   ├── piece_table.go   // Alternative: Piece-Table
│   ├── undo.go          // Undo/Redo History-Tree
│   └── cursor.go        // Cursor, Selection, Multi-Cursor
├── highlight/
│   ├── treesitter.go    // Tree-sitter Integration (inkrementell)
│   ├── chroma.go        // Chroma-Fallback (regex-basiert)
│   └── theme.go         // Syntax-Token → Lux-Theme-Color Mapping
├── lsp/
│   ├── client.go        // JSON-RPC 2.0 Client
│   ├── protocol.go      // LSP-Typen (Completion, Diagnostics, Hover, etc.)
│   ├── manager.go       // Language-Server-Lifecycle (Start/Stop/Restart)
│   └── capabilities.go  // Server-Capability-Negotiation
├── render/
│   ├── viewport.go      // Sichtbarer Ausschnitt, virtuelles Scrolling
│   ├── gutter.go        // Line Numbers, Breakpoints, Fold-Markers
│   ├── minimap.go       // Minimap-Rendering
│   └── decorations.go   // Diagnostics-Underlines, Bracket-Matching, Indent-Guides
└── keymap/
    ├── keymap.go        // Keybinding-Dispatch
    └── default.go       // Default-Keybindings (VS-Code-ähnlich)
```

### Integration in Lux

```go
// Verwendung als normales Lux-Widget
func view(m Model) ui.Element {
    return editor.New(editor.Props{
        Buffer:     m.Buffer,
        Language:   "go",
        Theme:      editor.ThemeFromLux(m.Theme), // Lux-Theme → Editor-Token-Colors
        OnChange:   func(buf *buffer.Buffer) Msg { return BufferChanged{buf} },
        OnSave:     func(path string) Msg { return FileSaved{path} },
        ReadOnly:   false,
        LineNumbers: true,
        Minimap:    true,
    })
}
```

Das Editor-Widget verhält sich wie jedes andere Lux-Widget — es empfängt `InputEvent`s über den Focus-Manager, rendert über `Canvas`, und exponiert seinen Zustand an das A11y-System.

---

## 7. Modul 1: Buffer — Rope/Piece-Table

### 7.1 Anforderungen

- Effiziente Inserts/Deletes an beliebiger Position: O(log n)
- Effizientes Line-Lookup: Zeile N → Byte-Offset in O(log n)
- Unterstützung für Dateien bis 100+ MB (>1M Zeilen)
- Grapheme-Cluster-aware (UAX #29) — Lux hat bereits `internal/text/grapheme.go`
- Snapshot-fähig für Undo/Redo (Copy-on-Write oder persistente Datenstruktur)

### 7.2 Design-Entscheidung: Rope

Ein **Rope** (balanced B-Tree über Textfragmente) wird empfohlen:

- Natürliches Line-Count-Tracking via augmentierte Knoten (jeder Knoten speichert Byte-Länge + Zeilenanzahl)
- Inkrementelle Tree-sitter-Updates: `tree.Edit()` braucht Byte-Offsets → Rope liefert O(log n) Offset-Lookup
- Persistente Snapshots für Undo: Structural Sharing zwischen Versionen (nur geänderte Pfade kopieren)

### 7.3 Go-Ökosystem

| Library | Stars | Status | Bewertung |
|---|---|---|---|
| `zyedidia/rope` | 6 | Abandoned (2021) | Nicht geeignet |
| `vinzmay/go-rope` | 23 | Dormant (2021), kein Rebalancing | Nicht geeignet |
| `deadpixi/rope` | 89 | Funktional, LGPL-2.1 | Lizenz problematisch |

**Empfehlung:** Eigene Rope-Implementierung in Go. Referenz-Designs: xi-editor's `Rope` (Rust), `ropey` (Rust), oder Zed's `sum_tree`. Geschätzter Aufwand: ~2-3 Wochen für eine produktionsreife Implementierung mit Line-Index und Snapshot-Support.

### 7.4 Undo/Redo

History-Tree (nicht lineare Liste) — ermöglicht Branch-Navigation:

```
     Edit1 → Edit2 → Edit3 (← current)
                  ↘ Edit2b → Edit2c
```

Jeder Knoten speichert einen Forward-Patch und Reverse-Patch. Undo = Reverse-Patch anwenden + Pointer verschieben. Kein vollständiger Buffer-Snapshot nötig dank Rope's Structural Sharing.

---

## 8. Modul 2: Syntax-Highlighting — Tree-sitter + Chroma

### 8.1 Dual-Strategie

```
                    ┌──────────────┐
                    │  Language?   │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼                         ▼
    Tree-sitter Grammar           Kein Grammar
    verfügbar?                    verfügbar
              │                         │
              ▼                         ▼
    Inkrementelles Parsing      Chroma Lexer
    + Highlight Queries         (Regex, 200+ Sprachen)
    (~30 Kernsprachen)          Nicht inkrementell
```

### 8.2 Tree-sitter Integration

**Library:** `smacker/go-tree-sitter` (544 Stars, 30+ Sprachen inline, inkrementelles Parsing)

Für 400+ Sprachen: `alexaandru/go-sitter-forest` als Erweiterung.

```go
// Inkrementelles Parsing nach jedem Edit
parser.SetLanguage(golang.GetLanguage())
tree := parser.Parse(nil, sourceCode)

// Nach Edit:
tree.Edit(tsEdit)  // Byte-Offsets des Edits
newTree := parser.Parse(tree, newSourceCode)  // Reused unveränderte Subtrees
```

**Highlight-Queries:** Tree-sitter liefert AST-Knoten, die via `.scm`-Queries auf Token-Typen gemappt werden (z.B. `(function_name) @function`, `(string_literal) @string`). Token-Typen werden dann via Theme auf Farben gemappt.

**CGo-Hinweis:** Tree-sitter ist eine C-Library. `smacker/go-tree-sitter` nutzt CGo. Dies ist die einzige CGo-Abhängigkeit im Editor-Stack. Build-Tag `-tags notreesitter` als Fallback auf Chroma-only.

### 8.3 Chroma Fallback

**Library:** `alecthomas/chroma` v2 (4.900 Stars, 200+ Sprachen, regex-basiert)

- Kein inkrementelles Highlighting — bei jedem Edit wird ab einem sicheren Punkt re-lexed
- Für Dateien < 10.000 Zeilen performant genug (< 5 ms pro Re-lex)
- Liefert `Token`-Stream mit Typ (Keyword, String, Comment, etc.) → Theme-Mapping

---

## 9. Modul 3: LSP-Client

### 9.1 Architektur

```
┌───────────────────┐       JSON-RPC 2.0 / stdio       ┌──────────────┐
│  lsp/client.go    │ ◄──────────────────────────────► │   gopls      │
│                   │                                    │   rust-analyzer
│  • Initialize     │       JSON-RPC 2.0 / stdio       │   pyright    │
│  • Completion     │ ◄──────────────────────────────► │   ...        │
│  • Hover          │                                    └──────────────┘
│  • Diagnostics    │
│  • Go-to-Def      │
│  • Rename         │
│  • Formatting     │
│  • Code Actions   │
└───────────────────┘
```

Der LSP-Client kommuniziert **direkt** mit Language Servern (eine Hop weniger als über Lapce-Proxy).

### 9.2 Go-Ökosystem

| Library | Stars | Status | Bewertung |
|---|---|---|---|
| `go.lsp.dev/protocol` + `jsonrpc2` | 128 | Dormant (2022) | Beste verfügbare Typen, Transport brauchbar |
| `sourcegraph/go-lsp` + `jsonrpc2` | — | Functional | Alternative Typen |
| `tliron/glsp` | 265 | Server-only | Nicht für Client nutzbar |
| gopls `internal/protocol` | — | Bestgepflegt, aber `internal` | Nicht importierbar, Fork möglich |

**Empfehlung:** `go.lsp.dev/protocol` für Typen + eigene Client-Dispatch-Logik. Der LSP-Client ist die größte Einzelinvestition (~3-4 Wochen), aber wiederverwendbar für jedes Lux-Tool das Language-Intelligence braucht.

### 9.3 Manager

Der `lsp/manager.go` verwaltet den Lifecycle:
- Auto-Detection: Dateiendung → Language Server Binary finden
- Start/Stop/Restart mit exponential Backoff
- Capability-Negotiation: Nur Features nutzen die der Server unterstützt
- Multi-Root: Mehrere Workspace-Roots für Monorepo-Support

---

## 10. Modul 4: Editor-Widget

### 10.1 Rendering-Strategie

Das Editor-Widget rendert über Lux' `Canvas`-API in die WGPU-Pipeline:

| Element | Rendering-Methode |
|---|---|
| Text (Code) | MSDF-Text-Rendering (>24px) / Bitmap (<24px) — bereits in Lux vorhanden |
| Gutter (Line Numbers) | Monospace-Text, rechtsbündig, eigener Scissor-Rect |
| Cursor | Animiertes Rect (1px breit), Blink via `Anim[float32]` |
| Selection | Semi-transparente Rects pro Zeile — wie in `TextArea` bereits implementiert |
| Diagnostics | Wavy Underline via Fragment-Shader oder Canvas-Path |
| Bracket-Matching | Farbige Rect-Hinterlegung |
| Indent-Guides | Vertikale Linien (1px, subtile Farbe) |
| Minimap | Stark verkleinerter Text (1-2px pro Zeile), Viewport-Indicator als Overlay |

### 10.2 Virtuelles Scrolling

Nur sichtbare Zeilen rendern (mit Overscan-Buffer):

```
Buffer: 100.000 Zeilen
Viewport: Zeile 5.000 - 5.060 (60 sichtbare Zeilen)
Overscan: ±20 Zeilen
Gerendert: Zeile 4.980 - 5.080 (100 Zeilen)
```

Lux' `VirtualList` bietet die Grundlage — muss für horizontales Scrolling und variable Zeilenhöhe (Code Folding) erweitert werden.

### 10.3 Multi-Cursor

Jeder Cursor ist ein unabhängiges `(position, selection_anchor)`-Paar. Edits werden für alle Cursor gleichzeitig angewendet (von unten nach oben, um Offset-Shifts zu vermeiden).

---

## 11. Vorhandene Lux-Bausteine

Bereits im Framework vorhanden und direkt wiederverwendbar:

| Baustein | Paket | Relevanz |
|---|---|---|
| MSDF + Bitmap Text-Rendering | `internal/text/`, `internal/gpu/` | Kern des Editor-Renderings |
| Glyph-Atlas-Management | `internal/text/atlas.go` | Glyph-Caching |
| Text-Shaping (OpenType GSUB/GPOS) | `internal/text/` via `go-text/typesetting` | Korrekte Schrift-Darstellung |
| `TextMeasure` / `TextMetrics` | `draw/canvas.go` | Zeilen-Layout, Cursor-Positionierung |
| `TextField` / `TextArea` | `ui/form/` | Referenz für Cursor, Selection, IME |
| Grapheme-Cluster-Handling (UAX #29) | `internal/text/grapheme.go` | Cursor-Navigation |
| Multiline Cursor Movement | `internal/text/multiline.go` | `CursorUp`, `CursorDown`, `LineStart`, `LineEnd` |
| Selection Rendering | `ui/form/textarea.go` | Semi-transparente Highlight-Rects |
| `InputState` (Cursor, Selection) | `ui/form/` | Basis für Editor-State |
| Clipboard (Ctrl+X/C/V) | `app/app.go` | Bereits im Framework-Keyboard-Handler |
| IME-Support | `app/run.go` | Compose, Commit, CursorRect |
| Focus-Management | `ui/` | Tab-Order, Focus-Gained/Lost |
| Kinetic Scrolling | `ui/kinetic_scroll.go` | Momentum-Scrolling |
| VirtualList | `ui/data/virtuallist.go` | Lazy-Rendering für große Listen |
| `RichTextElement` | `ui/` | Farbige Spans (Basis für Syntax-Highlighting-Display) |
| `Typography.Code` / `Typography.CodeSmall` | `theme/` | Monospace-Font-Tokens |
| Animations (`Anim[T]`, Spring) | `anim/` | Cursor-Blink, Smooth-Scroll |
| A11y (`AccessNode`, `AccessTextState`) | `a11y/` | Screen-Reader-Support |

**Einschätzung:** ~60-70% der UI-Infrastruktur für einen Code-Editor existiert bereits. Der Hauptaufwand liegt in Buffer-Datenstruktur, Syntax-Highlighting-Integration und LSP-Client.

---

## 12. Go-Ökosystem: Abhängigkeiten & Reifegrad

| Bereich | Beste Option | Reifegrad | Anmerkung |
|---|---|---|---|
| **Rope/Buffer** | Eigene Implementierung | — | Kein produktionsreifes Go-Rope vorhanden |
| **Tree-sitter** | `smacker/go-tree-sitter` | ★★★★☆ | 544 Stars, 30+ inline Grammars, inkrementelles Parsing |
| **Tree-sitter 400+ Sprachen** | `alexaandru/go-sitter-forest` | ★★★☆☆ | 490+ Grammars, regelmäßig regeneriert |
| **Syntax (Regex-Fallback)** | `alecthomas/chroma` v2 | ★★★★★ | 4.900 Stars, 200+ Sprachen, sehr stabil |
| **LSP-Typen** | `go.lsp.dev/protocol` | ★★★☆☆ | Komplett, aber unmaintained seit 2022 |
| **JSON-RPC Transport** | `go.lsp.dev/jsonrpc2` | ★★★☆☆ | Funktional, minimaler Wrapper |
| **Text-Shaping** | `go-text/typesetting` | ★★★★☆ | Pure-Go HarfBuzz, genutzt von Fyne/Gio/Ebitengine |

### Referenz-Projekte in Go

| Projekt | Stars | Typ | Lernwert |
|---|---|---|---|
| **micro** | 28.300 | Terminal-Editor | LineArray-Buffer, Regex-Highlighting, Lua-Plugins |
| **aretext** | 279 | Terminal-Editor | Vim-Bindings, minimales Design |
| **jmigpin/editor** | 441 | GUI-Editor (pure Go) | Acme-inspiriert, basale LSP-Anbindung |
| **oligo/gvcode** | 24 | Gio Editor-Widget | Piece-Table, Bracket-Completion, Gio-basiert |

---

## 13. Offene Fragen

| # | Frage | Kontext |
|---|---|---|
| 1 | Rope vs. Piece-Table: Welche Datenstruktur passt besser zur WGPU-Render-Pipeline? | Piece-Table hat effizientere sequentielle Reads (gut für zeilenweises Rendering), Rope hat besseres Structural Sharing (gut für Undo) |
| 2 | CGo-Akzeptanz: Ist Tree-sitter's CGo-Abhängigkeit für alle Zielplattformen tragbar? | Windows, macOS, Linux, Embedded (DRM/KMS) — CGo braucht C-Compiler im Build |
| 3 | LSP-Typen: `go.lsp.dev/protocol` forken oder gopls' `internal/protocol` extrahieren? | Ersteres ist einfacher, letzteres ist aktueller (auto-generiert aus Spec) |
| 4 | Minimap-Rendering: Eigener Render-Pass mit 1-2px Glyphen oder Textur-Downscale? | Performance vs. Qualität |
| 5 | Vim/Emacs-Modi: Ab wann? In v1 vorbereiten (Keymap-Abstraction) oder nach v1? | Keymap-Dispatch muss modal sein, wenn Vim geplant ist — besser früh designen |
| 6 | Collaborative Editing: CRDT im Buffer vorsehen oder nachträglich einbauen? | CRDT beeinflusst die Buffer-Datenstruktur fundamental — nachträglich einbauen ist teuer |
| 7 | File-Watcher: `fsnotify` oder OS-native Watcher? | `fsnotify` ist Standard in Go, aber hat bekannte Limitations auf macOS (kqueue) |

---

**Nächste Schritte (wenn RFC akzeptiert):**

1. Modul 1: Rope-Prototyp mit Line-Index und Benchmark gegen naive String-Concat
2. Modul 2: Tree-sitter PoC — Go-Datei parsen, Highlight-Queries anwenden, Spans an `RichTextElement` übergeben
3. Modul 4: Minimaler Editor-Widget-Prototyp — Monospace-Text + Cursor + Scrolling auf Basis von `TextArea`
4. Modul 3: LSP-Client PoC — `gopls` starten, `Initialize` + `textDocument/completion` implementieren
