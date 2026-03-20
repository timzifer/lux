# RFC-008 — lux/default-theme: Ein konsistentes Standard-Theme für Lux

**Repository:** `github.com/timzifer/lux`
**Status:** Proposed
**Version:** 0.1.0
**Datum:** 2026-03-20
**Abhängig von:** RFC-001 (Core Architecture), RFC-002 (Interaction & Layout), RFC-003 (Widget Catalogue & Theme), RFC-007 (WGPU Rendering)

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Design-Prinzipien](#2-design-prinzipien)
3. [Visuelle Sprache von `lux`](#3-visuelle-sprache-von-lux)
4. [Farbmodell & semantische Rollen](#4-farbmodell--semantische-rollen)
5. [Light- und Dark-Mode](#5-light--und-dark-mode)
6. [Typografie](#6-typografie)
7. [Spacing, Radii und Dichte](#7-spacing-radii-und-dichte)
8. [Elevation, Borders und Materialität](#8-elevation-borders-und-materialität)
9. [Interaktionszustände & Motion](#9-interaktionszustände--motion)
10. [GPU-Effekte: subtil, nicht spektakulär](#10-gpu-effekte-subtil-nicht-spektakulär)
11. [Komponenten-Leitlinien](#11-komponenten-leitlinien)
12. [Implementierungsstrategie](#12-implementierungsstrategie)
13. [Nicht-Ziele](#13-nicht-ziele)
14. [Migration von `Slate` zu `lux`](#14-migration-von-slate-zu-lux)
15. [Anhang A: Konkrete Token-Werte](#15-anhang-a-konkrete-token-werte)
16. [Anhang B: Design-Rezepte](#16-anhang-b-design-rezepte)

---

## 1. Motivation & Ziel

Lux braucht ein Standard-Theme, das nicht nur „brauchbar“ ist, sondern die
architektonischen Eigenschaften des Frameworks sichtbar macht:

- **ruhig, präzise, professionell**
- **auf Desktop und HMI glaubwürdig**
- **mit Light- und Dark-Mode aus einem Guss**
- **mit subtilen GPU-Effekten statt plakativer Effekthascherei**
- **nah genug an etablierten Mustern, um sofort vertraut zu wirken**
- **eigenständig genug, um Lux eine klare visuelle Identität zu geben**

Das bisherige `Slate`-Theme erfüllt die Rolle eines soliden technischen
Defaults, formuliert aber noch keine vollständige visuelle Sprache. Dieses RFC
spezifiziert deshalb ein neues Default-Theme namens **`lux`**.

`lux` ist kein Branding-Theme und kein Showcase-Theme. Es ist das
**Referenz-Theme des Frameworks**: die eine Gestaltung, die „out of the box“
serienreif wirken soll.

### Zielbild

`lux` soll sich anfühlen wie:

- **standard-ish**: sofort lesbar, nicht exzentrisch
- **professional-calm**: nüchtern, hochwertig, unaufgeregt
- **desktop-first**: präzise, kompakt, informationsdicht ohne hektisch zu sein
- **subtle-fancy**: sanfte Tiefe, leichte Materialität, kontrollierte Motion

Kurzform:

> `lux` ist die visuelle Übersetzung von Lux' Architektur: ruhig, typisiert,
> hochwertig, kontrolliert.

---

## 2. Design-Prinzipien

### 2.1 Familiar before novel

`lux` will nicht originell um jeden Preis sein. Benutzer sollen nicht über das
Design nachdenken müssen. Das Theme lehnt sich deshalb an verbreitete Desktop-
Muster an:

- neutrale Grundflächen
- klare Hierarchie
- subtile Section-Trennung
- sparsame Akzentfarbe
- deutlich erkennbare Hover-, Focus- und Disabled-Zustände

### 2.2 Calm surfaces, strong information hierarchy

Oberflächen bleiben zurückhaltend. Hierarchie entsteht primär durch:

1. Typografie
2. Spacing
3. Layer/Elevation
4. erst danach Farbe

Das verhindert die „Dashboard-Krankheit“, bei der jede Fläche um Aufmerksamkeit
konkurriert.

### 2.3 Accent is semantic, not decorative

Accent-Farben markieren Interaktion und Fokus — nicht bloß Dekoration.
Deshalb gilt:

- Primär-Buttons
- aktive Tabs / Selektionen
- Focus-Indikatoren
- Links / aktive Schalter

verwenden Accent, aber normale Panels, Cards und Inputs bleiben überwiegend
neutral.

### 2.4 Effects must support structure

Blur, Glow, Vibrancy und weiche Shadows sind erlaubt — aber nur, wenn sie
Informationsarchitektur verstärken:

- Floating UI von Basisinhalt abheben
- Fokus sichtbar machen
- Materialität andeuten
- große monotone Flächen auflockern

Nicht erlaubt ist „Effekt als Dekoration ohne Informationswert“.

### 2.5 One system, two luminance modes

Light und Dark sind keine zwei separaten Themes, sondern zwei Ausprägungen
derselben visuellen Sprache.

Gleich bleiben:

- semantische Rollen
- Abstände
- Typografie
- Radii
- Elevation-Logik
- Motion
- Zustands- und Fokus-Logik

Anders sind nur:

- Luminanzverhältnisse
- Border-Opazitäten
- Shadow-Dichte
- Stärke von Tints/Blur/Glow

---

## 3. Visuelle Sprache von `lux`

### 3.1 Charakter

Das Theme kombiniert drei Einflüsse:

- **Desktop-Nüchternheit**: präzise, produktiv, informationsorientiert
- **leichte Materialität**: sanfte Tiefe statt flacher Sterilität
- **moderne System-Anmutung**: standardnah statt markenhaft-expressiv

Nicht gewünscht sind:

- stark glossy
- verspielt
- neonartig
- brutalistisch hart
- mobile-first oversized

### 3.2 Formensprache

- überwiegend rechteckige Geometrie mit **sanft gerundeten** Ecken
- kleine bis mittlere Radien
- scharfe Kanten nur bei Dividern und Tabellenstrukturen
- Pill-/Capsule-Formen nur für Chips, Badges, segmented controls, toggles

### 3.3 Materialmodell

`lux` kennt vier Grundmaterialien:

1. **Base Surface** — Fenster-/App-Hintergrund
2. **Raised Surface** — Cards, Panels, Toolbars, Inputs
3. **Floating Surface** — Menus, Popovers, Dialoge, Kontextflächen
4. **Accent Surface** — selektierte oder primäre interaktive Flächen

Die UI soll hauptsächlich aus den ersten drei Materialien bestehen. Accent ist
sparsam.

### 3.4 Kontrastphilosophie

- Textkontrast hoch genug für produktive Arbeit
- Flächenkontraste kleiner als Textkontraste
- Borders subtil, aber zuverlässig
- Focus immer klar erkennbar, auch ohne Farbe allein

---

## 4. Farbmodell & semantische Rollen

`lux` verwendet keine „freien“ Farben als Primärsprache, sondern semantische
Rollen innerhalb des bestehenden `theme.TokenSet`.

### 4.1 Surface-Rollen

- `Surface.Base` — App-Hintergrund
- `Surface.Elevated` — Karten, Eingabefelder, sekundäre Panels
- `Surface.Hovered` — ruhiger Hover-Film auf neutralen Komponenten
- `Surface.Pressed` — gedrückter Zustand
- `Surface.Scrim` — Modaldialog-Hintergrund

### 4.2 Accent-Rollen

- `Accent.Primary` — primäre Aktion, Fokus, aktive Selektion
- `Accent.PrimaryContrast` — Text/Icon auf primärem Accent
- `Accent.Secondary` — optionale sekundäre Hervorhebung, sparsam genutzt

### 4.3 Stroke-Rollen

- `Stroke.Border` — Standard-Kanten
- `Stroke.Focus` — fokussierter Rahmen oder Basis für Glow
- `Stroke.Divider` — feinere Abschnittstrennung

### 4.4 Text-Rollen

- `Text.Primary` — Haupttext
- `Text.Secondary` — Metadaten, Caption, Hilfstexte
- `Text.Disabled` — deaktivierte Inhalte
- `Text.OnAccent` — Text auf Accent-Flächen

### 4.5 Status-Rollen

Statusfarben bleiben funktional und werden nicht überinszeniert:

- `Success`
- `Warning`
- `Error`
- `Info`

Statusflächen sollen in `lux` **leicht getönt** und nicht vollflächig laut sein,
außer bei bewusst prominenten Alerts.

---

## 5. Light- und Dark-Mode

### 5.1 Gemeinsame Logik

Beide Modi teilen:

- dieselbe Akzentfarbe
- dieselben Radii
- dieselbe Typografie
- dieselben Komponentenregeln
- dieselbe Motion

Der Benutzer soll beim Umschalten keinen Stilbruch erleben, sondern nur eine
Luminanzänderung.

### 5.2 Dark Mode

Der Dark-Mode ist der primäre Referenzmodus für Lux.

Eigenschaften:

- neutral-kühle, leicht graphiteartige Basis
- keine tiefschwarzen Flächen außer maximal im Frame-Hintergrund
- erhöhte Flächen als sanfte Tonstufen, nicht als harte Platten
- Borders eher über Opazität als über starke Helligkeit
- Shadow über Dichte und Weichheit, nicht über extreme Dunkelheit

Wirkung:

- ruhig
- fokusfreundlich
- hochwertig
- geeignet für lange Arbeitsphasen

### 5.3 Light Mode

Der Light-Mode bleibt professionell und blendfrei.

Eigenschaften:

- leicht warm-neutrale oder neutrale helle Basis
- keine reinweiße klinische Gesamtfläche als Standardzustand
- Raised/Floating-Surfaces mit leichtem Materialkontrast
- Schatten sichtbarer, aber weiterhin weich
- Borders etwas präsenter als im Dark-Mode

Wirkung:

- vertraut
- sachlich
- hell ohne Sterilität

### 5.4 Akzentfarbe

Die Default-Akzentfarbe von `lux` ist ein ruhiges, systemnahes Blau mit guter
Barrierefreiheit in hellen und dunklen Kontexten.

Sie soll:

- professionell wirken
- nicht zu „consumer“-bunt sein
- in Focus-Ringen und Glow funktionieren
- mit Statusfarben klar unterscheidbar bleiben

---

## 6. Typografie

### 6.1 Grundhaltung

Typografie ist in `lux` das wichtigste Mittel zur Hierarchie. Die Typografie
soll deshalb nicht ornamental, sondern präzise sein.

### 6.2 Stilcharakter

- Sans-Serif als Standard
- gute Bildschirmlesbarkeit
- moderate Größenunterschiede
- wenig extreme Tracking-Experimente
- Überschriften leicht kompakter, Body stabil lesbar

### 6.3 Empfohlene Skala

Die bestehende Lux-Skala aus RFC-003 bleibt erhalten, wird aber gestalterisch
klarer interpretiert:

- `H1` — Seitenkopf / primäre Paneelüberschrift
- `H2` — Bereichsüberschrift
- `H3` — Unterbereich / Gruppentitel
- `Body` — Standardtext
- `BodySmall` — Hilfstext, Meta, Tabellennebeninformation
- `Label` — Buttons, Tabs, Controls
- `LabelSmall` — Badge, Chip, dichte UI
- `Code` / `CodeSmall` — Entwickler-UI, Logs, technische Felder

### 6.4 Typografische Regeln

- keine übergroßen mobilen Touch-Headlines
- Labels tendenziell Medium statt Bold
- Secondary Text nie so schwach, dass er wie Disabled wirkt
- Text auf Accent-Flächen immer hochkontrastig

---

## 7. Spacing, Radii und Dichte

### 7.1 Dichte

`lux` ist **desktop-kompakt**, aber nicht gedrängt.

Ziel:

- mehr Informationsdichte als Mobile Design Systems
- mehr Luft und Rhythmus als klassische Legacy-Desktop-UIs

### 7.2 Spacing-Skala

Die bestehende 4/8/16/24/32/48-Skala bleibt erhalten. Interpretation:

- `XS` = Mikroabstände in dichten Controls
- `S` = Standard-Abstand innerhalb kleiner Komponenten
- `M` = Standard-Innenabstand für Panels und Gruppen
- `L` = Abschnittswechsel
- `XL` / `XXL` = großräumige Trennung auf Seitenebene

### 7.3 Radii

`lux` verwendet kleine bis mittlere Radien, aber bewusst konservativer als
viele aktuelle Consumer-UIs. Die Formensprache soll eher „Werkzeug“ als
„Lifestyle-App“ kommunizieren.

- Inputs: präzise, eher scharf als weich
- Buttons: leicht weicher als Inputs, aber weiterhin desktop-nah
- Cards / Menus / Dialoge: sichtbar weich, aber nicht blob-artig
- Pill-Radius nur für Komponenten mit klarer semantischer Rechtfertigung

Wichtig: Der Schritt von `Slate` zu `lux` soll **evolutionär**, nicht
revolutionär sein. Gerade Input-Felder dürfen nicht plötzlich weicher und
„consumeriger“ wirken als der Rest des Systems.

### 7.4 Regel

Wenn unklar ist, ob Spacing oder Farbe zur Gruppierung genutzt werden soll,
gilt:

> erst Spacing, dann Border, dann Background-Tint.

---

## 8. Elevation, Borders und Materialität

### 8.1 Elevation ist funktional

Elevation ist in `lux` kein Deko-Effekt, sondern dient drei Zwecken:

1. visuelle Hierarchie
2. Hover-/Interaktions-Feedback
3. Trennung von Floating UI

### 8.2 Elevation-Stufen

Empfohlen werden vier semantische Stufen:

- **Level 0** — flach / eingebettet
- **Level 1** — leichte Anhebung (Inputs, Cards, Toolbars)
- **Level 2** — klar abgehoben (Panels, aktive Sektionen)
- **Level 3** — floating (Menus, Dialoge, Popovers)

### 8.3 Borders

Borders sind in `lux` wichtig. Sie verhindern, dass die UI im Namen von
Modernität zu weich und konturlos wird.

Regeln:

- 1px-ähnliche subtile Borders als Default
- Divider noch feiner als Borders
- Focus darf Border ergänzen oder temporär ersetzen
- bei Floating UI: Border + Shadow gemeinsam, nicht nur eines von beidem

### 8.4 Materialität ohne Skeuomorphismus

Materialität entsteht durch die Kombination aus:

- leichter Flächendifferenz
- feiner Border
- weicher Shadow
- optional minimalem Verlauf

Nicht durch starke Texturen oder harte Lichtsimulation.

### 8.5 Scrim-Semantik

`Surface.Scrim` ist kein abstrakter Stimmungswert, sondern ein konkret
definierter Render-Schritt:

- Scrim wird als **vollflächiges FillRect über dem gesamten Viewport**
  gezeichnet
- der Scrim liegt **oberhalb des Basisinhalts**, aber **unterhalb** von Dialog,
  Popover oder anderem modalen Floating-Content
- Scrim nutzt standardmäßig **keinen Backdrop-Blur**
- Scrim wird **nicht** als `PushOpacity` auf den gesamten Content-Layer
  implementiert, damit Farben, Text und bereits gezeichnete Overlays nicht
  unterschiedlich stark „verwaschen“

Damit ist das Default-Verhalten auf Desktop- und DRM/KMS-Zielen vorhersagbar
und performant. Blur hinter Modal-Overlays bleibt ein optionaler
Theme-Override, nicht Teil des Default-Themes.

---

## 9. Interaktionszustände & Motion

### 9.1 Zustandsmodell

Jede interaktive Komponente soll konsistent auf folgende Zustände reagieren:

- Rest
- Hover
- Pressed / Active
- Focused
- Selected
- Disabled

### 9.2 Hover

Hover ist in `lux` **spürbar, aber leise**:

- leichte Tonwertänderung
- optional kleiner Elevation-Lift
- kein farbiger Hover als Default auf neutralen Flächen

### 9.3 Pressed

Pressed soll eindeutig, aber kurzlebig sein:

- dunkler oder dichter als Hover
- Shadow reduziert sich statt zuzunehmen
- Bewegung minimal nach „innen“ denkbar, aber subtil

### 9.4 Focus

Focus ist ein First-Class-Zustand.

Empfohlene Hierarchie:

1. klarer Focus-Stroke
2. optional sanfter Glow in Accent-Farbe
3. hoher Kontrast zu Umgebung

`lux` bevorzugt einen **ruhigen Focus-Ring mit leichter Aura** statt eines
aggressiven Neon-Effekts.

### 9.5 Motion

Motion in `lux` ist:

- schnell
- weich
- funktional
- niedrig-amplitudig

Empfohlene Defaults:

- Hover/Fade: `Quick`
- Standardzustandswechsel: `Standard`
- Dialoge/Overlays: zwischen `Standard` und `Emphasized`

Keine Komponente soll „floaty“ oder verspielt federn, außer dies ist bewusst
für einzelne Showcase-Komponenten gewünscht.

### 9.6 Disabled-Zustände

Disabled ist in `lux` ein eigener semantischer Zustand und **kein pauschales
`opacity: 0.38` über das gesamte Widget**.

Default-Regeln:

- Text und Icons verwenden `Text.Disabled`
- Accent-Farben werden entfernt oder auf neutrale Flächen zurückgeführt
- Borders werden abgeschwächt, aber nicht unsichtbar
- Hintergründe bleiben lesbar, werden jedoch tonwertlich neutralisiert
- Shadows, Glow und Hover-Reaktionen entfallen

Bewusst **nicht** empfohlen wird ein globaler `PushOpacity`-Layer für das ganze
Widget, weil dieser Text, Border, Icon und Hintergrund gleichermaßen absenkt
und dadurch auf hellen wie dunklen Themes oft matschig wirkt. Eine globale
Disabled-Opacity bleibt für Sonderfälle oder Theme-Overrides erlaubt, ist aber
nicht die Standardregel von `lux`.

---

## 10. GPU-Effekte: subtil, nicht spektakulär

RFC-007 etabliert Blur-, Shadow-, Glow- und weitere Effektpfade. `lux` nutzt
sie bewusst konservativ.

### 10.1 Soft Shadows

Soft Shadows sind der wichtigste Effekt im Theme.

Regeln:

- weiche Kanten
- geringer bis mittlerer Offset
- nie pechschwarz in Light-Mode
- im Dark-Mode mehr über Dichte als über Offset arbeiten

### 10.2 Frosted Glass

Frosted Glass wird **nicht global**, sondern selektiv eingesetzt:

- Command Palette
- Dropdown / Popover
- Floating Inspector
- sekundäre Side-Panels über visuell aktivem Hintergrund

Wichtig: **Context Menus gehören nicht zur Default-Bühne für Frosted Glass.**
Sie erscheinen unter dem Cursor, werden häufig geöffnet und müssen auch auf
Bare-Metal-/DRM-Zielen ohne dedizierte GPU billig renderbar bleiben.

Für Context Menus gilt im Default-Theme daher:

- normale Floating Surface
- feine Border
- weiche Shadow
- optional Fade-/Scale-Motion
- **kein Blur-Pass per Default**

Frosted-Glass-Menus bleiben ein opt-in Verhalten in separaten Theme-Overrides.

### 10.3 Vibrancy / Tinted Blur

Vibrancy ist in `lux` optional und sparsam. Wenn verwendet, dann mit sehr
niedriger Tönung. Ziel ist Materialität, nicht Farbeffekt.

### 10.4 Glow

Glow ist **kein genereller Dekorationseffekt**, sondern reserviert für:

- Focus
- aktive Auswahl in wenigen Schlüsselkomponenten
- besonders wichtige interaktive Zustände

### 10.5 Noise / Grain

Subtiles Grain darf auf großen Verläufen oder ruhigen Paneelen eingesetzt
werden, um Banding und digitale Sterilität zu reduzieren. Die Intensität muss
so niedrig sein, dass sie eher „gefühlt“ als bewusst gesehen wird.

---

## 11. Komponenten-Leitlinien

### 11.1 Fenster / Seiten

- Base Surface als ruhiger Hintergrund
- Sektionen über Spacing und H2/H3 strukturieren
- großflächige Accent-Flächen vermeiden

### 11.2 Buttons

#### Filled Button
- Accent-Hintergrund
- klare Lesbarkeit
- Hover: leicht heller oder dichter
- Pressed: minimal dunkler, weniger Lift

#### Outlined Button
- neutraler Hintergrund
- subtile Border
- Hover über Surface-Hover-Tint

#### Ghost Button
- Standard für sekundäre Toolbar- oder Inline-Aktionen
- Hover über sehr feinen Film

#### Tonal Button
- getönte, aber nicht vollakzentige Fläche
- geeignet für sekundäre Betonung

### 11.3 TextFields und Form Controls

- Inputs wirken leicht eingelassen oder ruhig aufgesetzt
- Border wichtiger als starke Hintergrunddifferenz
- Focus sichtbar über Border + subtile Aura
- Error bevorzugt als Kombination aus Border + leichter Tönung, nicht nur Rot

### 11.4 Cards und Panels

- `Surface.Elevated`
- weiche Low-/Med-Shadow
- feine Border
- großzügiger Innenabstand
- Accent nur für Inhalt, nicht für Gesamtfläche

### 11.5 Menus, Tooltips, Overlays

Dies ist die primäre Bühne für Lux' „subtle-fancy“-Qualität:

- Floating Surface
- feine Border
- weiche Shadow
- optional neutraler Frosted-Glass-Effekt für ausgewählte Overlays
- schnelle Fade-/Scale-Motion

Präzisierung:

- Tooltips, Command Palette und ausgewählte Popovers dürfen Frosted Glass
  verwenden
- Context Menus verwenden im Default-Theme **kein** Frosted Glass
- Dialoge verwenden standardmäßig Scrim + Floating Surface; Blur bleibt opt-in

### 11.6 Tabs, Chips, Badges

- aktive Elemente eher tonal als laut gefüllt
- Badges kompakt und lesbar
- Chips mit Pill-Radius, aber zurückhaltender Tönung

### 11.7 Dialoge

- deutliche Floating-Elevation
- klare Hintergrundtrennung via Scrim
- evtl. minimale Scale/Fade-In Motion
- keine übermäßige Glas-/Glow-Inszenierung

---

## 12. Implementierungsstrategie

### 12.1 Theme-Familie

Das Framework soll eine Theme-Familie `lux` bereitstellen:

- `theme.LuxDark`
- `theme.LuxLight`
- `theme.LuxAuto`

Die Bezeichnung `lux` ist bewusst identisch mit dem Frameworknamen: Das
Standard-Theme ist die Referenzdarstellung des Systems.

`theme.LuxAuto` ist der empfohlene ergonomische Einstieg:

- folgt dem OS-/System-Dark-Mode-Signal
- startet in Light oder Dark abhängig von der Systempräferenz
- wechselt zur Laufzeit via `SetDarkModeMsg` bzw. äquivalenter
  Plattform-Integration automatisch mit

`theme.LuxDark` und `theme.LuxLight` bleiben die expliziten Varianten für Apps,
die ihre Erscheinung bewusst fest verdrahten wollen.

### 12.2 Beziehung zu `Slate`

`Slate` kann als historischer oder kompatibler Vorgänger bestehen bleiben,
mittelfristig sollte jedoch gelten:

- `LuxDark` ersetzt `Slate` als empfohlene Standardwahl
- `LuxLight` ersetzt `SlateLight` als empfohlene helle Wahl
- `LuxAuto` ersetzt den bisherigen impliziten Wunsch nach „irgendeinem Default“
- Dokumentation und Beispiele referenzieren primär `lux`

### 12.3 Tokens statt Sonderlogik

Das Theme soll möglichst vollständig durch vorhandene Tokens und DrawFuncs
formulierbar sein. Neue Theme-Logik ist nur dann gerechtfertigt, wenn eine
wiederkehrende visuelle Regel sich mit den bestehenden Rollen nicht klar
abbilden lässt.

### 12.4 Effektpolitik im Default

Nicht alle verfügbaren GPU-Effekte müssen im Default-Theme sofort aktiv sein.
`lux` priorisiert:

1. weiche Shadows
2. ruhige Focus-Auren
3. leichte Floating-Glass-Behandlung für Overlays
4. optionale subtile Verläufe
5. optionales sehr schwaches Grain

---

## 13. Nicht-Ziele

Dieses RFC ist **kein** Versuch,

- ein brand-spezifisches Marketing-Theme zu definieren
- Material, Fluent oder Cupertino zu kopieren
- möglichst viele Stilrichtungen gleichzeitig abzudecken
- Effektreichtum zum Selbstzweck zu maximieren
- Mobile-First-Touch-Overdesign als Desktop-Default zu etablieren

---

## 14. Migration von `Slate` zu `lux`

### 14.1 Ziel

Bestehende Lux-Apps sollen ohne massiven Bruch migrieren können.

### 14.2 Strategie

1. `lux` zunächst parallel zu `Slate` einführen
2. Kitchen-Sink und Beispiel-Apps auf `lux` umstellen
3. Doku und Screenshots auf `lux` ausrichten
4. `theme.Default` mittelfristig auf `lux` umbiegen
5. `Slate` als Kompatibilitätsalias oder Legacy-Theme beibehalten

### 14.3 Erwarteter visueller Unterschied

Gegenüber `Slate` soll `lux`:

- etwas warmer / menschlicher wirken
- konsistenter in Light und Dark sein
- klarere Materialhierarchien haben
- bessere Default-Elevation besitzen
- Overlays hochwertiger erscheinen lassen

---

## 15. Anhang A: Konkrete Token-Werte

Die folgenden Werte sind Startwerte, keine unverrückbaren Naturgesetze. Sie
formulieren die gewünschte Richtung präzise genug für eine erste Implementierung.

### 15.1 `lux` Dark

```go
TokenSet{
    Colors: ColorScheme{
        Surface: SurfaceColors{
            Base:     draw.Hex("#0f1115"),
            Elevated: draw.Hex("#171a20"),
            Hovered:  draw.Hex("#1d222a"),
            Pressed:  draw.Hex("#252b35"),
            Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.46},
        },
        Accent: AccentColors{
            Primary:         draw.Hex("#4c8dff"),
            PrimaryContrast: draw.Hex("#ffffff"),
            Secondary:       draw.Hex("#7aa8ff"),
        },
        Stroke: StrokeColors{
            Border:  draw.Color{R: 1, G: 1, B: 1, A: 0.10},
            Focus:   draw.Hex("#7aa8ff"),
            Divider: draw.Color{R: 1, G: 1, B: 1, A: 0.06},
        },
        Text: TextColors{
            Primary:   draw.Hex("#eef2f7"),
            Secondary: draw.Hex("#a8b0bc"),
            Disabled:  draw.Hex("#606975"),
            OnAccent:  draw.Hex("#ffffff"),
        },
        Status: StatusColors{
            Success:   draw.Hex("#3bb273"),
            Warning:   draw.Hex("#d9a441"),
            Error:     draw.Hex("#de5b6d"),
            Info:      draw.Hex("#4c8dff"),
            OnSuccess: draw.Hex("#ffffff"),
            OnError:   draw.Hex("#ffffff"),
        },
    },
}
```

### 15.2 `lux` Light

```go
TokenSet{
    Colors: ColorScheme{
        Surface: SurfaceColors{
            Base:     draw.Hex("#f5f7fb"),
            Elevated: draw.Hex("#ffffff"),
            Hovered:  draw.Hex("#edf1f7"),
            Pressed:  draw.Hex("#e4e9f1"),
            Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.18},
        },
        Accent: AccentColors{
            Primary:         draw.Hex("#2f6fe4"),
            PrimaryContrast: draw.Hex("#ffffff"),
            Secondary:       draw.Hex("#5e92ef"),
        },
        Stroke: StrokeColors{
            Border:  draw.Color{R: 0.09, G: 0.12, B: 0.18, A: 0.12},
            Focus:   draw.Hex("#2f6fe4"),
            Divider: draw.Color{R: 0.09, G: 0.12, B: 0.18, A: 0.08},
        },
        Text: TextColors{
            Primary:   draw.Hex("#17202b"),
            Secondary: draw.Hex("#5e6a78"),
            Disabled:  draw.Hex("#9aa4b2"),
            OnAccent:  draw.Hex("#ffffff"),
        },
        Status: StatusColors{
            Success:   draw.Hex("#278f5a"),
            Warning:   draw.Hex("#b27d1f"),
            Error:     draw.Hex("#c94b5d"),
            Info:      draw.Hex("#2f6fe4"),
            OnSuccess: draw.Hex("#ffffff"),
            OnError:   draw.Hex("#ffffff"),
        },
    },
}
```

### 15.3 Typografie

Die bestehende Typografie-Skala aus RFC-003 bleibt gültig. Für `lux` wird
folgende Interpretation empfohlen:

```go
TypographyScale{
    H1:         draw.TextStyle{Size: 20, Weight: draw.FontWeightSemiBold, LineHeight: 1.25, Tracking: -0.01},
    H2:         draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold, LineHeight: 1.30},
    H3:         draw.TextStyle{Size: 14, Weight: draw.FontWeightMedium,   LineHeight: 1.35},
    Body:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular,  LineHeight: 1.50},
    BodySmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular,  LineHeight: 1.45},
    Label:      draw.TextStyle{Size: 12, Weight: draw.FontWeightMedium,   LineHeight: 1.00},
    LabelSmall: draw.TextStyle{Size: 11, Weight: draw.FontWeightMedium,   LineHeight: 1.00},
    Code:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular,  LineHeight: 1.45},
    CodeSmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular,  LineHeight: 1.40},
}
```

### 15.4 Spacing und Radii

```go
SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48}
RadiusScale{Input: 4, Button: 6, Card: 10, Pill: 999}
```

### 15.5 Elevation

Beispielwerte:

```go
ElevationScale{
    None: draw.Shadow{},
    Low:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.14}, BlurRadius: 10, OffsetY: 2, Radius: 8},
    Med:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.18}, BlurRadius: 18, OffsetY: 6, Radius: 12},
    High: draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.22}, BlurRadius: 28, OffsetY: 10, Radius: 14},
}
```

Im Light-Mode dürfen dieselben Geometrien mit etwas geringerer Alpha oder
leicht angepasster Verteilung verwendet werden.

### 15.6 Motion

```go
MotionSpec{
    Standard:   DurationEasing{Duration: 220 * time.Millisecond, Easing: anim.OutCubic},
    Emphasized: DurationEasing{Duration: 320 * time.Millisecond, Easing: anim.InOutCubic},
    Quick:      DurationEasing{Duration: 110 * time.Millisecond, Easing: anim.OutExpo},
}
```

---

## 16. Anhang B: Design-Rezepte

### B.1 Standard-Seite

- `Surface.Base` als Gesamtfläche
- Abschnitte mit `H2` und `Spacer(L)`
- Form-Gruppen auf `Surface.Elevated`
- Primäraktion als einzelner Filled Button
- alle Nebenaktionen Ghost oder Outlined

### B.2 Settings-Panel

- Panel auf `Surface.Elevated`
- feine Border
- Low-Shadow
- Gruppen mit Divider statt zusätzlicher bunter Hintergründe

### B.3 Command Palette / Quick Switcher

- Floating Surface
- High-Elevation
- optional neutraler Frosted-Glass-Hintergrund
- Accent nur für aktive Zeile und Fokusindikatoren

### B.4 Inspector / Developer Tooling

- dichteres Layout
- Code-/Mono-Stile für Werte
- Secondary Text für Pfade und Meta
- sparsame Statusfarben

### B.5 HMI-Variante

`lux` ist bewusst standardnah. Eine spätere HMI-spezifische Variante kann auf
`lux` aufsetzen, aber:

- größere Targets
- stärkere Kontraste
- weniger feine Hover-Abhängigkeiten
- robustere Statussignalisierung

Das ist eine Ableitung, nicht der Default.

---

## 17. Implementierungsstatus

Stand: 2026-03-20

### Erledigt

- [x] **§4 Farbmodell** — Alle semantischen Farbrollen (Surface, Accent, Stroke,
  Text, Status) in `LuxDark` und `LuxLight` definiert (`theme/theme.go`)
- [x] **§5 Light-/Dark-Mode** — `LuxDark`, `LuxLight`, `LuxAuto` implementiert;
  `ThemePair`-Interface für dynamischen Wechsel via `SetDarkModeMsg`
- [x] **§6 Typografie** — Vollständige Skala (H1–CodeSmall) mit Tracking (-0.01
  auf H1), JetBrains Mono für Code
- [x] **§7 Spacing/Radii** — 4/8/16/24/32/48 Spacing; Input=4, Button=6,
  Card=10, Pill=999
- [x] **§8 Elevation** — Weiche Shadows (Low/Med/High) mit RFC-konformen
  BlurRadius/Alpha/OffsetY-Werten; GPU-SDF-Shadow-Pipeline aktiv
- [x] **§9.2 Hover** — Leise Tonwertänderung via `HoverState`, Dauer aus
  `Motion.Quick`
- [x] **§9.4 Focus** — `drawFocusRing()` mit Glow-Aura + `Stroke.Focus`-Ring
  auf allen interaktiven Widgets (Button, TextField, Checkbox, Radio, Toggle,
  Slider, Select)
- [x] **§9.5 Motion** — `Motion.Quick` (Toggle, Hover), `Motion.Standard`
  (Tree expand/collapse) aus Theme-Tokens konsumiert; `ToggleState.update()`
  akzeptiert `DurationEasing`
- [x] **§10.1 Soft Shadows** — GPU-Pipeline mit SDF-basiertem Blur-Falloff
- [x] **§10.2 Frosted Glass** — `FrostedGlass()`, `TintedBlur()` als Primitiven
  vorhanden; Context Menus ohne Blur per Default
- [x] **§10.3 Vibrancy** — `Vibrancy()` als Accent-getönte Frosted-Glass-Variante
- [x] **§10.4 Glow** — `GlowBox()`/`Glow()` via Shadow-Pipeline; Focus-Glow
  via `drawFocusRing()`
- [x] **§11.2 Buttons** — Filled, Outlined, Ghost, Tonal Varianten implementiert
- [x] **§11.3 TextFields** — Focus-Aura via `drawFocusRing()`
- [x] **§11.7 Dialoge** — Scrim als FillRect-Overlay, Floating-Elevation
- [x] **§12.1 Theme-Familie** — `theme.LuxDark`, `theme.LuxLight`,
  `theme.LuxAuto` exportiert
- [x] **§12.2 Beziehung zu Slate** — `theme.Default = LuxDark`
- [x] **§15 Token-Werte** — Alle konkreten Farb-, Typografie-, Spacing-, Radii-,
  Elevation- und Motion-Werte aus Anhang A umgesetzt

### Offen

- [x] **§9.3 Pressed** — Zweistufige Hover→Pressed-Differenzierung für
  Checkbox, Radio, Toggle und Slider implementiert (`hoverOpacity >= 0.9`
  triggert `Surface.Pressed`-Blend)
- [x] **§9.6 Disabled** — `Disabled`-Feld auf allen interaktiven Element-Typen
  (Button, Checkbox, Radio, Toggle, Slider, Chip, TextField, Select).
  `disabledColor()` mutet Farben 50% Richtung `Surface.Base`;
  `Text.Disabled` für Labels; Focus und Hover deaktiviert.
  Convenience-Konstruktoren: `ButtonTextDisabled`, `CheckboxDisabled`,
  `RadioDisabled`, `ToggleDisabled`, `SliderDisabled`, `ChipDisabled`,
  `WithTextFieldDisabled()`, `WithSelectDisabled()`.
  `DrawCtx.Disabled` für Custom-Theme-Dispatch.
- [x] **§9.5 Motion.Emphasized** — `OverlayAnimFadeScale` nutzt
  `Motion.Emphasized`; einfachere Animationen nutzen `Motion.Standard`.
  `overlayEntry` trägt `Animation` + `Duration` für Framework-seitige
  Enter/Exit-Steuerung.
- [ ] **§10.5 Noise/Grain** — Kein Grain-Shader oder -Token im Framework.
  Erfordert GPU-Pipeline-Erweiterung (Noise-Pass als optionaler Post-Effekt)
- [x] **§11.4 Cards** — Card-Widget nutzt `DrawShadow` mit `Elevation.Low`
  + feine `StrokeRoundRect`-Border als Default statt des alten
  Doppel-FillRoundRect-Ansatzes
- [x] **§11.5 Menus/Tooltips** — Opt-in Frosted-Glass-Backdrop für Tooltip
  (`TooltipBlur`) und ContextMenu (`ContextMenuBlur`) via `PushClipRoundRect` +
  `PushBlur(8)` + halbtransparenter Tint-Fill. Nicht-Blur-Pfad behält opaken
  Fill bei. Shared Border-Stroke auf beiden Pfaden.
- [ ] **§11.6 Tabs/Chips/Badges** — Implementiert, aber aktive Elemente nutzen
  noch vollflächigen Accent statt tonaler Abstufung
- [ ] **§14 Migration** — Kitchen-Sink und Beispiel-Apps referenzieren `Slate`
  statt `lux`; Screenshots und Doku noch nicht aktualisiert

### Out-of-Scope (nicht Teil dieses RFCs)

- **HMI-Variante (§B.5)** — Bewusst als spätere Ableitung geplant
- **Brand-/Marketing-Theme (§13)** — Explizit als Nicht-Ziel definiert
- **Mobile-First-Touch-Overdesign (§13)** — Nicht gewünscht
- **Backdrop-Blur hinter Scrim (§8.5)** — Bewusst opt-in, nicht Default
- **Frosted-Glass auf Context Menus (§10.2)** — Bewusst opt-in per
  Theme-Override, nicht Default
- **Kopie von Material/Fluent/Cupertino (§13)** — Eigenständige Identität
