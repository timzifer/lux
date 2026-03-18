# RFC-003 — lux: Widget Catalogue & Theme

**Repository:** `github.com/timzifer/lux`
**Status:** Teilweise integriert
**Version:** 0.1.0
**Datum:** 2026-03-18
**Zuletzt abgeglichen:** 2026-03-18
**Abhängig von:** RFC-001, RFC-002

---

### Implementierungsstatus

| Abschnitt | Status | Anmerkung |
|-----------|--------|-----------|
| §1 Theming-System — Token-Werte | ✅ Integriert | `theme/theme.go` — vollständiger `TokenSet`, `ColorScheme`, `TypographyScale`, alle Scales |
| §1.1 Theme-Interface | ✅ Integriert | `Tokens()`, `DrawFunc()`, `Parent()` |
| §1.2 TokenSet | ✅ Integriert | Alle Token-Gruppen vorhanden |
| §1.3 DrawFunc & Custom Rendering | ✅ Integriert | `DrawFunc`, `DrawCtx` |
| §1.4 Theme-Lookup-Cache | ⏳ Wartend | Kein Resolved-Cache |
| §1.5 Theme-Wechsel (`SetThemeMsg`) | ✅ Integriert | `app.SetThemeMsg`, `app.SetDarkModeMsg` |
| §1.6 `theme.Override` | ✅ Integriert | `OverrideSpec` mit Pointer-Feldern |
| §2 theme.Slate Dark + Light | ✅ Integriert | Alle Token-Werte wie spezifiziert |
| §3 Text-Stack, i18n & Fonts | 🔶 Teilweise | |
| §3.1 CGo-Strategie | 🔶 Teilweise | Aktuell: OpenGL+GLFW (CGo); wgpu/gogpu noch nicht integriert |
| §3.2 go-text/typesetting | ⏳ Wartend | Aktuell: eigener `sfnt_shaper`; kein vollständiges OpenType-Shaping |
| §3.3 Shaper-Interface | ⏳ Wartend | |
| §3.4 Font-Loading & Fallback-Chain | 🔶 Teilweise | `fonts` Package mit Embedded-Fonts; kein vollständiges FontFamily/Fallback |
| §3.5 BiDi | ⏳ Wartend | |
| §3.6 Unicode Line-Breaking (UAX #14) | ⏳ Wartend | |
| §3.7 Grapheme-Cluster & Cursor-Navigation | ⏳ Wartend | |
| §3.8 i18n & l10n | ⏳ Wartend | Locale-Propagation, RTL-Layout (→ RFC-002 §4.6) |
| §3.9 Package-Name | ✅ Integriert | `github.com/timzifer/lux` |
| §4 Widget-Katalog | | |
| §4.1 Tier 1 — Kern | ✅ Integriert | Text, Button, Icon, Row, Column, Stack, ScrollView, Divider, Spacer |
| §4.1 Tier 2 — Formulare | ✅ Integriert | TextField, Checkbox, Radio, Toggle, Slider, ProgressBar, Select |
| §4.1 Tier 3 — Struktur | ✅ Integriert | Card, Tabs, Accordion, Tooltip, Badge, Chip, MenuBar, ContextMenu |
| §4.1 Tier 4 — Erweitert | ⏳ Wartend | DatePicker, ColorPicker, DataTable, SplitView, etc. |
| §4.2 Widget-Spezifikations-Template | ⏳ Wartend | Detailspezifikationen pro Widget fehlen |
| §5 Rich Text & Texteditierung | 🔶 Teilweise | |
| §5.2–5.4 RichText (Ebene 2) | ✅ Integriert | `ui/rich_text.go` mit Spans |
| §5.5 Inline-Widgets | ⏳ Wartend | |
| §5.6 RichTextEditor (Ebene 3) | ⏳ Wartend | |

---

## Inhaltsverzeichnis

1. [Theming-System — Token-Werte & Konventionen](#1-theming-system--token-werte--konventionen)
2. [theme.Slate — Das Default-Theme](#2-themeslate--das-default-theme)
3. [Text-Stack, i18n & Fonts](#3-text-stack-i18n--fonts)
4. [Widget-Katalog](#4-widget-katalog)
5. [Rich Text & Texteditierung](#5-rich-text--texteditierung)

---

## 1. Theming-System — Token-Werte & Konventionen

Das Theming-System ist der neue Bestandteil, der zu den Key-Aspekten hinzukommt. Es verfolgt diese Designziele:

- **Nicht im User-Model** — Themes sind Laufzeit-Konfiguration, kein Applikationszustand.
- **Vollständig erweiterbar** — Custom-Draw-Hooks ohne Framework-Fork.
- **Composable** — Themes bauen auf anderen Themes auf (Prototype-Chain-Semantik).
- **Updatebar zur Laufzeit** — Theme-Wechsel (Dark/Light, Branding) via `Send`.

### 1.1 Das Theme-Interface

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

### 1.2 TokenSet

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

### 1.3 DrawFunc & Custom Rendering

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

### 1.4 Theme-Lookup-Algorithmus & Caching

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

### 1.5 Theme-Wechsel zur Laufzeit

Themes sind kein Userland-State, aber der Framework-Loop kennt das aktive Theme:

```go
// Eingebaute Msg-Typen des Frameworks:
type SetThemeMsg struct{ Theme Theme }
type SetDarkModeMsg struct{ Dark bool }

// Usage:
app.Send(SetThemeMsg{Theme: myBrandTheme{}})
```

Der Loop wendet das neue Theme beim nächsten Frame an. Es gibt kein Re-Rendering des gesamten Baums — nur Widgets, deren `DrawFunc` sich geändert hat, werden neu gezeichnet (via Dirty-Tracking).

### 1.6 Theme-Komposition: Partial Overrides

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

## 2. theme.Slate — Das Default-Theme


`theme.Slate` ist das eingebaute Default-Theme von lux. Philosophie: **die Nüchternheit von Linear, die Schärfe von Fluent Design** — ohne die Plattform-Konnotationen von Material oder Cupertino.

#### Design-Prinzipien

**1px Solid Borders statt Schatten.** Schatten erfordern Multi-Pass-Rendering (Blur). Eine `1px solid Border` ist auf allen Backends — insbesondere DRM/KMS — ein einzelner Draw-Call. Das ist kein ästhetisches Zugeständnis sondern eine Performance-Entscheidung die auf dem kleinstmöglichen Nenner aufbaut. Wer Schatten will, überschreibt `DrawFunc(WidgetKindCard)` im Theme.

**Semantische Slots, keine Farb-Literale.** Kein Widget kennt `#18181b` — es kennt `tokens.Colors.Surface.Elevated`. Ein Custom-Theme muss nur die Slots neu belegen, nicht jeden Widget-DrawFunc überschreiben.

**Desktop-First Proportionen.** 13dp Body, 6dp Button-Radius, 4dp Input-Radius. Kompakt, präzise, werkzeughafte Ästhetik.

#### Token-Werte (Dark Mode — Default)

```go
var Slate = theme.New(TokenSet{
    Colors: ColorScheme{
        Surface: {
            Base:     Color{Hex: "#09090b"},  // Zinc-950 — tiefstes Schwarz
            Elevated: Color{Hex: "#18181b"},  // Zinc-900 — Cards, Overlays
            Hovered:  Color{Hex: "#27272a"},  // Zinc-800 — Hover-State
            Pressed:  Color{Hex: "#3f3f46"},  // Zinc-700 — Active-State
        },
        Accent: {
            Primary:         Color{Hex: "#3b82f6"},  // Blue-500 — Go-affines Blau
            PrimaryContrast: Color{Hex: "#ffffff"},
            Secondary:       Color{Hex: "#6366f1"},  // Indigo-500
        },
        Stroke: {
            Border:  Color{Hex: "#ffffff", A: 0.10},  // 10% Weiß — subtil
            Focus:   Color{Hex: "#3b82f6"},            // = Accent.Primary
            Divider: Color{Hex: "#ffffff", A: 0.06},  // 6% Weiß — fast unsichtbar
        },
        Text: {
            Primary:   Color{Hex: "#fafafa"},  // Zinc-50
            Secondary: Color{Hex: "#a1a1aa"},  // Zinc-400 — Metadaten
            Disabled:  Color{Hex: "#52525b"},  // Zinc-600
            OnAccent:  Color{Hex: "#ffffff"},
        },
        Status: {
            Success: Color{Hex: "#22c55e"},  // Green-500
            Warning: Color{Hex: "#f59e0b"},  // Amber-500
            Error:   Color{Hex: "#ef4444"},  // Red-500
            Info:    Color{Hex: "#3b82f6"},  // Blue-500
        },
    },

    Typography: TypographyScale{
        H1:        TextStyle{Size: 20, Weight: 600, LineHeight: 1.3},
        H2:        TextStyle{Size: 16, Weight: 600, LineHeight: 1.3},
        H3:        TextStyle{Size: 14, Weight: 500, LineHeight: 1.4},
        Body:      TextStyle{Size: 13, Weight: 400, LineHeight: 1.5},
        BodySmall: TextStyle{Size: 12, Weight: 400, LineHeight: 1.5},
        Label:     TextStyle{Size: 12, Weight: 500, LineHeight: 1.0},
        Code:      TextStyle{Size: 13, Weight: 400, LineHeight: 1.6,
                             FontFamily: "JetBrains Mono"},
        CodeSmall: TextStyle{Size: 12, Weight: 400, LineHeight: 1.6,
                             FontFamily: "JetBrains Mono"},
    },

    Radii: RadiusScale{
        Input:  4,
        Button: 6,
        Card:   8,
        Pill:   999,
    },

    Spacing: SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48},
})
```

#### Button-State-Konventionen

Der Button ist die häufigste Interaktionsfläche — seine State-Logik definiert die visuelle Sprache des gesamten Themes:

```
State     Background          Border              Text
──────────────────────────────────────────────────────────
Idle      Surface.Elevated    Stroke.Border       Text.Primary
Hover     Surface.Hovered     Stroke.Border       Text.Primary
Pressed   Surface.Pressed     Stroke.Border       Text.Primary
Focused   Surface.Elevated    Stroke.Focus (2px)  Text.Primary
Disabled  Surface.Base        Stroke.Divider      Text.Disabled
Primary   Accent.Primary      –                   Text.OnAccent
```

Die Unterscheidung `Button` vs. `Primary Button` ist keine Variante im Widget — sie ist ein `variant`-Token das das Widget beim DrawFunc-Aufruf übergeben kann. Custom-Themes überschreiben nur die Token-Werte, nie die State-Logik selbst.

#### Light Mode

```go
var SlateLight = theme.Override(Slate, theme.OverrideSpec{
    Colors: &ColorScheme{
        Surface: {
            Base:     Color{Hex: "#ffffff"},
            Elevated: Color{Hex: "#f4f4f5"},  // Zinc-100
            Hovered:  Color{Hex: "#e4e4e7"},  // Zinc-200
            Pressed:  Color{Hex: "#d4d4d8"},  // Zinc-300
        },
        Stroke: {
            Border:  Color{Hex: "#000000", A: 0.10},
            Focus:   Color{Hex: "#3b82f6"},
            Divider: Color{Hex: "#000000", A: 0.06},
        },
        Text: {
            Primary:   Color{Hex: "#09090b"},  // Zinc-950
            Secondary: Color{Hex: "#71717a"},  // Zinc-500
            Disabled:  Color{Hex: "#a1a1aa"},  // Zinc-400
            OnAccent:  Color{Hex: "#ffffff"},
        },
    },
})
```

Dark/Light-Wechsel via `app.Send(SetDarkModeMsg{Dark: false})` — das Framework swappt zwischen `Slate` und `SlateLight`.

### 2.2 Token-Zugriff in Widgets

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

## 3. Text-Stack, i18n & Package-Name

### 3.1 CGo-Strategie: Minimal und explizit

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

### 3.2 Der vollständige Text-Stack

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

### 3.3 Das interne Shaper-Interface

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

### 3.4 Font-Loading & Fallback-Chain

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

### 3.5 BiDi: Vollständige Unicode-Unterstützung

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

### 3.6 Unicode Line-Breaking (UAX #14)

Zeilenumbrüche sind nicht trivial — am Leerzeichen brechen reicht nur für Latin-Text:

- **Thai** hat keine Leerzeichen zwischen Wörtern — Line-Breaking erfordert Wörterbuch-basierte Segmentierung
- **CJK** bricht an fast jedem Zeichen, aber nicht vor bestimmten Satzzeichen (z.B. Klammern, Punktuation)
- **Bindestrich-Trennung** folgt sprachspezifischen Regeln

```go
// LineBreaker segmentiert Text in umbrechbare Einheiten gemäß UAX #14.
// Implementierung: rivo/uniseg oder eigene UAX#14-Implementierung auf Basis
// von golang.org/x/text/unicode/segment (wenn verfügbar).
type LineBreaker interface {
    // Breaks gibt die erlaubten Umbruchpositionen im Text zurück.
    // Jeder Break hat einen Typ: Mandatory (Zeilenende), Opportunity (darf umbrechen),
    // oder NoBreak (darf hier nicht umbrechen).
    Breaks(text string) []LineBreak
}

type LineBreak struct {
    Offset int           // Byte-Offset im Text
    Kind   LineBreakKind // Mandatory, Opportunity
}
```

Die TextLayout-Pipeline (§5.3) nutzt den LineBreaker für Zeilenumbruch. Ohne UAX#14-Konformität ist mehrzeiliger Text in nicht-lateinischen Schriften kaputt.

### 3.7 Grapheme-Cluster & Cursor-Navigation

Ein sichtbares "Zeichen" ist nicht immer eine Rune. Go-Strings sind UTF-8 und `[]rune` zählt Unicode-Codepoints — aber weder Bytes noch Runes entsprechen dem, was ein Benutzer als Zeichen wahrnimmt:

| Sichtbar | Runes | Grapheme-Cluster |
|----------|-------|------------------|
| é | 1 oder 2 (precomposed oder e + ◌́) | 1 |
| 👨‍👩‍👧 | 5 (Person + ZWJ + Person + ZWJ + Person) | 1 |
| 🇩🇪 | 2 (Regional Indicator D + E) | 1 |

```go
// Grapheme-Cluster-Segmentierung via rivo/uniseg — pure Go, UAX#29-konform.
// Wird verwendet für:
//   - Cursor-Bewegung: ←/→ springt über einen Grapheme-Cluster, nicht eine Rune
//   - Backspace: löscht einen Grapheme-Cluster
//   - Textauswahl: Doppelklick markiert Wort-Grenzen (UAX#29 Word Boundaries)
//   - Text-Messung: Cursor-Positionen im TextLayout
import "github.com/rivo/uniseg"
```

**Regel:** Jede Cursor-Operation im Framework arbeitet auf Grapheme-Cluster-Grenzen, nie auf Byte- oder Rune-Indizes. Das betrifft `TextField`, `RichTextEditor` und die `TextLayout`-Pipeline.

### 3.8 Internationalisierung (i18n) & Lokalisierung (l10n)

Das Framework liefert die **Primitiven** für i18n — die Lokalisierung von App-Strings ist Sache der Anwendung.

#### Was das Framework bereitstellt

| Primitiv | Implementierung | Wo spezifiziert |
|----------|----------------|-----------------|
| RTL-Layout-Spiegelung | `LayoutDirection` im `LayoutCtx` | RFC-002 §4.6 |
| BiDi-Text | `BidiParagraph()` | §3.5 |
| Complex Script Shaping | `GoTextShaper` | §3.2, §3.3 |
| Unicode Line-Breaking | `LineBreaker` | §3.6 |
| Grapheme-Cluster-Navigation | `rivo/uniseg` | §3.7 |
| IME-Support | `IMEComposeMsg` | RFC-002 §2.2 |
| Locale-Propagation | `App.Locale` → `LayoutCtx.Direction` | RFC-002 §4.6 |
| A11y Sprach-Tag | `AccessNode.Lang` | RFC-001 §11.3 |

#### Was die Anwendung selbst macht (Framework-agnostisch)

**Locale-aware Formatierung** via `golang.org/x/text/message`:

```go
import "golang.org/x/text/message"

p := message.NewPrinter(language.German)
label := p.Sprintf("%d Dateien", count)
// → "1.234 Dateien" (deutsches Tausender-Trennzeichen)

price := p.Sprintf("%.2f €", amount)
// → "12,99 €" (deutsches Dezimalkomma)
```

**String-Kataloge** via `nicksnyder/go-i18n` oder ähnliche Libraries:

```
messages/
  en.toml    # greeting = "Hello, {name}"
  de.toml    # greeting = "Hallo, {name}"
  ar.toml    # greeting = "مرحبا، {name}"
```

Das Framework erzwingt kein bestimmtes i18n-Pattern — aber es liefert ein `App.Locale`-Feld (BCP 47 `language.Tag`), das allen Primitiven zugrunde liegt. Widgets können darauf zugreifen um richtungsabhängige Entscheidungen zu treffen.

#### App-Locale setzen

```go
app.Run(model, update, view,
    app.WithLocale(language.German),   // Explizit
    // oder: app.WithLocale(language.Und) → aus OS-Locale ableiten (Default)
)
```

Locale-Wechsel zur Laufzeit via `SetLocaleMsg` — triggert Layout-Invalidierung (weil sich die Richtung ändern kann) und AccessTree-Update (weil sich `Lang` ändert).

### 3.9 Package-Name

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

## 4. Widget-Katalog

> **Status: In Bearbeitung.** Dieser Abschnitt wird in einer separaten Session ausgearbeitet.
> Die Tier-Einteilung und vollständige Widget-Spezifikation folgt.

### 4.1 Tier-Übersicht

**Tier 1 — Kern** *(v1.0, ohne diese geht nichts)*
`Text`, `Button`, `Icon`, `Row`, `Column`, `Stack`, `ScrollView`, `Divider`, `Spacer`

**Tier 2 — Formulare** *(v1.0)*
`TextField`, `Checkbox`, `Radio`, `Toggle`, `Slider`, `ProgressBar`, `Select`

**Tier 3 — Struktur** *(v1.0)*
`Card`, `Tabs`, `Accordion`, `Tooltip`, `Badge`, `Chip`, `MenuBar`, `ContextMenu`

**Tier 4 — Erweitert** *(post-v1.0)*
`DatePicker`, `ColorPicker`, `DataTable`, `SplitView`, `Toolbar`, `RichTextEditor`, `FilePicker` (Open/Save), `TextArea`

### 4.2 Widget-Spezifikations-Template

Jedes Widget wird nach folgendem Schema spezifiziert:

```
Widget:        Name
Props:         Öffentliche Felder (typisiert)
WidgetState:   Interner State (WidgetState-Interface)
Msgs:          Welche Msgs sendet das Widget via ctx.Send?
A11y:          AccessRole + Pflichtfelder in AccessNode
DrawFunc:      State-zu-Token-Mapping (Idle/Hover/Pressed/Focused/Disabled)
Theme-Tokens:  Welche TokenSet-Felder werden genutzt?
Beispiel:      Minimales Code-Snippet
```

---

## 5. Rich Text & Texteditierung

### 5.1 Einordnung: Vier Ebenen

Rich Text ist kein einzelnes Feature sondern ein Spektrum — von einfach gestyltem Text bis hin zu einem vollständigen Dokument-Editor. Das Framework deckt Ebenen 1–2 ab; Ebene 3 ist ein eigenständiges Widget-Paket; Ebene 4 ist der Surface-Slot-Pfad (§8).

```
Ebene 1  TextLayout          Bereits §6.2 — single-style, MeasureText
Ebene 2  RichText            Dieses Kapitel — gemischte Spans, read-only
Ebene 3  RichTextEditor      Separates Paket — Cursor, Selection, Undo/Redo
Ebene 4  External Surface    §8 — Browser-Engine, CodeMirror, vollst. HTML/CSS
```

### 5.2 Das Span-Modell

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

### 5.3 TextLayout-Pipeline

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

### 5.4 RichText-Widget (Ebene 2)

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

### 5.5 Inline-Widgets

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

### 5.6 RichTextEditor (Ebene 3 — separates Paket)

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

### 5.7 Externe Rendering-Grenze (Ebene 4)

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


---

*RFC-003 — Draft. Feedback via GitHub Issues gegen `github.com/timzifer/lux`.*