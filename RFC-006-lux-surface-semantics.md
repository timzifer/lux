# RFC-006 — lux/surface/semantic: Semantik-Subtrees für externe Surfaces

**Repository:** `github.com/timzifer/lux`

**Status:** Theoretical
**Version:** 0.1.0
**Datum:** 2026-03-19
**Abhängigkeit:** RFC-001-lux.md §8 (Surface-Slots), §11 (Accessibility); RFC-004-lux-webview.md

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Abgrenzung](#2-abgrenzung)
3. [Problemstellung](#3-problemstellung)
4. [Architektur: Rendering-Pfad vs. Semantik-Pfad](#4-architektur-rendering-pfad-vs-semantik-pfad)
5. [Das `SemanticProvider`-Interface](#5-das-semanticprovider-interface)
6. [Integration in den globalen AccessTree](#6-integration-in-den-globalen-accesstree)
7. [Fokus, Aktionen und Hit-Testing](#7-fokus-aktionen-und-hit-testing)
8. [Use-Cases](#8-use-cases)
9. [Fallbacks & Degradationsstrategie](#9-fallbacks--degradationsstrategie)
10. [Testing](#10-testing)
11. [Offene Fragen](#11-offene-fragen)
12. [Quellen & Referenzen](#12-quellen--referenzen)

---

## 1. Motivation & Ziel

Lux modelliert Accessibility bewusst als First-Class-Feature: Der semantische Baum wird aus dem VTree konstruiert und an die Plattform-Bridges (AT-SPI2, UIA, NSAccessibility) weitergereicht. Für normale Widgets ist das ideal.

Mit RFC-001 §8 führt Lux jedoch gleichzeitig **Surface-Slots** als Escape-Hatch für externe Renderer ein: Browser-Engines, Video-Decoder, PDF-Renderer, 3D-Engines oder native Spezialkomponenten liefern eine GPU-Textur, die in den Widget-Baum eingebettet wird.

Damit entsteht eine Lücke:

- **Rendering** externer Inhalte ist vorgesehen.
- **Input-Routing** zu externer Surface ist vorgesehen.
- **Semantik/A11y** für externe Surface-Inhalte ist bislang **nicht** spezifiziert.

Das ist insbesondere problematisch für:

- PDF-Viewer mit Text, Links, Formularfeldern und Lesereihenfolge
- Video-Player mit Untertiteln (CC), Kapitelmarken und Bedienelementen
- Browser-Content mit interaktiven Controls
- eingebettete Dokument- oder Editor-Komponenten mit eigener semantischer Struktur

Ziel dieses RFC ist ein **paralleler Semantik-Pfad** zu Surface-Slots:

- Der visuelle Inhalt bleibt eine externe Surface.
- Optional kann dieselbe Surface zusätzlich einen **semantischen Teilbaum** an Lux liefern.
- Lux merged diesen Teilbaum in den globalen `AccessTree` und reicht ihn an die Plattform-A11y-Bridges weiter.

Das Ergebnis ist kein "magisches Extrahieren" aus Pixeln, sondern ein expliziter, testbarer und plattformneutraler Vertrag zwischen eingebettetem Renderer und Framework.

---

## 2. Abgrenzung

| In Scope | Nicht in Scope |
|---|---|
| Optionaler Semantik-Kanal für Surface-Slots | Automatische Semantik-Erkennung aus Pixeln / OCR |
| Merge von Surface-Subtrees in den globalen `AccessTree` | Vollständige Re-Spezifikation von RFC-001 §11 |
| Fokus-, Action- und Hit-Test-Routing für Surface-Semantik | JavaScript-Bridge oder DOM-Bridge als primärer Pfad |
| Use-Cases: PDF, Video+CC, WebView, native Spezialkomponenten | Allgemeine Native-Widget-Integration außerhalb von Surface-Slots |
| Fallback-Strategien wenn keine Semantik exportierbar ist | Layout-/Render-Details des externen Renderers |

**Nicht-Ziel:** Lux versucht **nicht**, Semantik aus einer beliebigen Surface rückwärts zu rekonstruieren. Wenn ein externer Renderer keinen Semantik-Export anbietet, bleibt die Surface semantisch eine Black Box mit expliziten Fallback-Nodes.

---

## 3. Problemstellung

### 3.1 Der aktuelle Zustand

RFC-001 spezifiziert für Surface-Slots folgendes Minimalmodell:

```go
type SurfaceProvider interface {
    AcquireFrame(bounds Rect) (wgpu.TextureView, FrameToken)
    ReleaseFrame(token FrameToken)
    HandleMsg(msg Msg) bool
}
```

Dieses Interface beschreibt korrekt den visuellen und interaktiven Pfad, aber nicht den semantischen.

### 3.2 Warum generische Group-Fallbacks nicht reichen

Ohne zusätzlichen Vertrag kann Lux für eine externe Surface höchstens einen generischen `AccessNode` erzeugen, z.B.:

```go
AccessNode{
    Role:  RoleGroup,
    Label: "PDF-Ansicht",
}
```

Das ist besser als nichts, verliert aber alle wichtigen Eigenschaften:

- Text ist nicht lesbar für Screenreader
- Links/Formularfelder sind nicht fokussierbar
- Untertitel können nicht als Live-Region exponiert werden
- Explore-by-touch / Hit-Testing auf Touch-Hardware ist unmöglich
- Selektion, Cursor, Fokus und Aktionen sind nicht adressierbar

### 3.3 Kein universeller Standard auf Surface-Ebene

Es gibt keinen plattformneutralen "einen" Standard, über den beliebige eingebettete Inhalte automatisch ihren Semantik-Baum nach außen stülpen. Unterschiedliche Engines haben unterschiedliche Fähigkeiten, APIs und Reifegrade.

Daraus folgt: Lux braucht einen **eigenen, engine-agnostischen Adapter-Vertrag**.

---

## 4. Architektur: Rendering-Pfad vs. Semantik-Pfad

Die Kernidee ist die saubere Trennung zweier paralleler Pfade:

```text
                 ┌────────────────────────────┐
                 │  Eingebetteter Renderer    │
                 │  (PDF / Video / WebView)   │
                 └─────────────┬──────────────┘
                               │
            ┌──────────────────┴──────────────────┐
            │                                     │
            ▼                                     ▼
  Rendering-Pfad                           Semantik-Pfad
AcquireFrame()/ReleaseFrame()         SnapshotSemantics()/Actions
            │                                     │
            ▼                                     ▼
      GPU-Textur                           Surface-Subtree
            │                                     │
            └──────────────┬──────────────────────┘
                           ▼
                Lux Widget-/AccessTree Merge
                           ▼
                  Plattform-A11y-Bridge
```

Wichtig:

1. **Kein Zwang:** Nicht jede Surface muss Semantik liefern.
2. **Snapshots statt Live-Mutation:** Semantik wird als unveränderlicher Snapshot geliefert.
3. **Stabile IDs:** Knoten innerhalb eines Surface-Subtrees müssen über Frames hinweg stabil identifizierbar sein.
4. **Lux bleibt Owner des globalen AccessTree:** Die Surface liefert einen Teilbaum, nicht die vollständige Plattform-Bridge.

---

## 5. Das `SemanticProvider`-Interface

### 5.1 Grundidee

`SurfaceProvider` bleibt bewusst klein und renderzentriert. Semantik wird über ein **optionales zweites Interface** angeboten:

```go
package surface

// SemanticProvider ist optional. Surfaces die es nicht implementieren,
// bleiben semantisch Black Boxes mit Fallback-Node.
type SemanticProvider interface {
    // SnapshotSemantics liefert einen unveränderlichen Snapshot des
    // semantischen Teilbaums relativ zu den aktuellen Surface-Bounds.
    SnapshotSemantics(bounds Rect) SurfaceSemantics

    // HitTest ermittelt den semantischen Knoten an einer Position relativ
    // zu den Surface-Bounds. Relevant für Explore-by-touch und Fokus.
    HitTestSemantics(p Point) (SurfaceNodeID, bool)

    // Fokus-Wechsel aus der Plattform-A11y oder dem Lux-Fokusmodell.
    FocusSemanticNode(id SurfaceNodeID) bool

    // Führt eine semantische Aktion aus (z.B. activate, increment, scroll).
    PerformSemanticAction(id SurfaceNodeID, action string) bool
}
```

### 5.2 Typen

```go
package surface

type SurfaceNodeID uint64

type SurfaceSemantics struct {
    Roots []SurfaceAccessNode

    // Optional: monoton steigende Version. Erleichtert Diffs,
    // Caching und Bridge-Optimierungen.
    Version uint64
}

type SurfaceAccessNode struct {
    ID          SurfaceNodeID
    Parent      SurfaceNodeID // 0 = Root innerhalb der Surface
    Role        a11y.AccessRole
    Label       string
    Description string
    Value       string
    Bounds      draw.Rect     // relativ zur Surface in dp
    Lang        language.Tag
    States      a11y.AccessStates
    Actions     []a11y.AccessActionDesc
    Relations   []a11y.AccessRelationDesc
}
```

### 5.3 Warum kein `Accessibility(state WidgetState) AccessNode`

Das bestehende `AccessibleWidget`-Modell passt nicht direkt auf Surface-Slots:

- eine Surface kann **viele** semantische Knoten enthalten
- die Struktur ist oft **dynamisch und renderer-intern**
- Hit-Testing und Action-Routing müssen relativ zu Surface-Koordinaten funktionieren
- externe Renderer besitzen häufig ein eigenes Fokus-/Selection-Modell

Deshalb ist hier ein **Subtree-Snapshot** geeigneter als ein einzelner `AccessNode`.

### 5.4 Minimale Implementierung

Eine Surface kann die RFC schrittweise implementieren:

1. Nur `SnapshotSemantics` mit statischen Nodes
2. zusätzlich `HitTestSemantics`
3. zusätzlich `FocusSemanticNode`
4. zusätzlich `PerformSemanticAction`

Nicht implementierte Teilaspekte führen zu kontrollierter Degradation.

---

## 6. Integration in den globalen AccessTree

### 6.1 Merge-Modell

Lux behandelt die Surface im Widget-Baum als normalen Container-Knoten und hängt den Surface-Subtree darunter ein:

```text
App Root
└── Dialog
    ├── Button "Zurück"
    └── Surface "PDF-Ansicht"
        ├── Heading "Kapitel 1"
        ├── Link "https://..."
        └── Text "..."
```

Die Surface selbst bleibt im globalen Baum als stabiler Host-Knoten sichtbar. Die von `SemanticProvider` gelieferten Knoten werden als Kinder dieses Host-Knotens eingefügt.

### 6.2 Host-Node

Jede Surface erhält immer einen Host-Node:

```go
AccessNode{
    Role:  RoleGroup,
    Label: "PDF-Ansicht",
}
```

Wenn `SemanticProvider` verfügbar ist, werden die Surface-Kinder darunter angehängt. Ohne `SemanticProvider` bleibt nur der Host-Node bestehen.

### 6.3 ID-Namespace

Der globale `AccessTree` darf keine ID-Kollisionen zwischen Lux-Widgets und Surface-Nodes haben. Deshalb gilt:

- Lux-eigene Nodes behalten ihre normalen `AccessNodeID`s.
- Surface-Nodes werden intern in einen separaten Namespace gemappt, z.B. `(surfaceUID, surfaceNodeID)`.

Die Plattform-Bridges sehen nur stabile globale IDs; die Aufteilung bleibt intern.

### 6.4 Snapshot-Konsistenz

Der AccessTree für eine Surface muss semantisch zum dargestellten Frame passen.

Empfohlene Regel:

- `AcquireFrame(bounds)` und `SnapshotSemantics(bounds)` beziehen sich auf denselben internen Zustand des externen Renderers.
- Der Renderer darf intern doppelt puffern, muss aber konsistente Snapshots liefern.

### 6.5 Partielle Updates

Große Browser- oder PDF-Subtrees können tausende Nodes enthalten. Deshalb darf Lux semantische Updates optimieren:

- Rebuild nur bei geänderter `Version`
- diff-basierter Update-Pfad in der `A11yBridge`
- optional Dirty-Flags auf Surface-Ebene

Das ist eine Optimierung, keine semantische Anforderung.

---

## 7. Fokus, Aktionen und Hit-Testing

### 7.1 Fokusmodell

Der globale Fokus bleibt Lux-kontrolliert. Sobald der Fokus auf einen Surface-Node fällt, delegiert Lux an den `SemanticProvider`:

```go
if sp, ok := surface.(SemanticProvider); ok {
    sp.FocusSemanticNode(nodeID)
}
```

Die Plattform-A11y darf dadurch z.B. den Cursor in ein PDF-Formularfeld setzen oder ein Browser-Control fokussieren.

### 7.2 Aktionen

Semantische Aktionen werden vom globalen AccessTree in Surface-Aktionen übersetzt:

- `activate`
- `increment`
- `decrement`
- `showMenu`
- `scrollForward`
- `scrollBackward`
- `setValue`

Lux routet diese Aktionen an `PerformSemanticAction(...)`. Der externe Renderer entscheidet, wie sie konkret umgesetzt werden.

### 7.3 Explore-by-touch / Maus-HitTest

Für Touch-A11y und Screenreader-Hit-Testing ist die Zuordnung Bildschirmposition → semantischer Knoten entscheidend.

Deshalb sollte `HitTestSemantics` für folgende Fälle unterstützt werden:

- Screenreader Explore-by-touch
- Magnifier / Hover-A11y
- Kontextabhängige Ankündigungen
- Fokusrestauration nach Scroll/Zoom

### 7.4 Scroll, Zoom und Koordinatensysteme

`Bounds` der Surface-Nodes sind **immer relativ zur sichtbaren Surface** in dp.

Wenn die Surface intern scrollt oder zoomt, muss der Snapshot bereits die transformierten, aktuell sichtbaren Bounds enthalten. Lux rechnet nicht in den internen Dokumentkoordinaten des Renderers.

---

## 8. Use-Cases

### 8.1 PDF-Viewer

#### Ziel

Ein PDF wird extern gerendert, aber seine semantische Struktur wird als Subtree exportiert:

- Dokumenttitel
- Überschriften
- Absätze
- Links
- Formularfelder
- Lesereihenfolge

#### Beispiel

```go
type PDFView struct {
    surface.Base
}

func (p *PDFView) AcquireFrame(bounds Rect) (wgpu.TextureView, FrameToken)
func (p *PDFView) ReleaseFrame(token FrameToken)
func (p *PDFView) HandleMsg(msg Msg) bool

func (p *PDFView) SnapshotSemantics(bounds Rect) SurfaceSemantics {
    return SurfaceSemantics{
        Roots: []SurfaceAccessNode{
            {ID: 1, Role: a11y.RoleHeading, Label: "Kapitel 1"},
            {ID: 2, Role: a11y.RoleLink, Label: "Mehr Informationen"},
            {ID: 3, Role: a11y.RoleTextInput, Label: "Name"},
        },
    }
}
```

#### Nutzen

- Screenreader kann PDF-Inhalt lesen
- Formularfelder werden fokussierbar
- Links werden aktivierbar
- Touch-A11y funktioniert auch bei eingebettetem PDF

### 8.2 Video-Player mit Untertiteln (CC)

Das Video selbst bleibt eine Surface. Die Controls können normale Lux-Widgets sein. Für Untertitel gibt es zwei sinnvolle Pfade:

1. Sichtbare Untertitel als normales Lux-Overlay
2. zusätzlicher semantischer Knoten / Live-Region

Beispiel:

```go
SurfaceAccessNode{
    ID:     42,
    Role:   a11y.RoleCustomBase + 1, // z.B. Caption role
    Label:  "Untertitel",
    Value:  "Bitte verlassen Sie das Gebäude.",
    States: a11y.AccessStates{Live: a11y.LivePolite},
}
```

### 8.3 WebView

Für `lux/surface/webview` kann dieselbe Architektur genutzt werden:

- visuell: Web-Inhalt via Surface
- semantisch: Browser-/Engine-spezifischer A11y-Subtree → `SurfaceSemantics`

Diese RFC schreibt **nicht** vor, wie eine Browser-Engine ihre Semantik intern gewinnt. Sie fordert nur das nach außen sichtbare Lux-Interface.

### 8.4 Native Spezialkomponenten

Auch proprietäre Dokument-Viewer, CAD-Komponenten oder HMIs können so integriert werden, solange sie einen stabilen Semantik-Snapshot liefern.

---

## 9. Fallbacks & Degradationsstrategie

### 9.1 Keine Semantik verfügbar

Wenn eine Surface keinen `SemanticProvider` implementiert:

- Lux erzeugt nur den Host-Node
- `RoleGroup` + Label/Description reichen als Minimal-A11y
- Fokus landet auf der Surface als Ganzes, nicht auf internen Controls

### 9.2 Teilweise Semantik verfügbar

Wenn nur `SnapshotSemantics` verfügbar ist:

- Screenreader kann lesen
- aber Aktionen/Fokus/Hit-Testing sind eingeschränkt

### 9.3 Nur Live-Content verfügbar

Für manche Fälle (z.B. Video-CC) reicht ein kleiner Snapshot mit wenigen Live-Nodes. Vollständige interne Semantik ist nicht nötig.

### 9.4 Fehlerfälle

Wenn der externe Renderer abstürzt oder temporär keine Semantik liefern kann:

- letzter valider Snapshot darf kurzzeitig bestehen bleiben
- danach Fallback auf Host-Node mit `Busy=true` oder Fehlerbeschreibung
- A11y-Bridge erhält ein konsistentes Update, keine halbinvaliden Bäume

---

## 10. Testing

### 10.1 Unit-Tests

Die Integration muss ohne GUI testbar bleiben. Analog zu RFC-001 gilt deshalb:

```go
func TestPDFSurfaceSemanticsMergedIntoAccessTree(t *testing.T) {
    tree := renderToAccessTree(view(Model{PDF: fakePDFSurface()}))

    pdf := tree.FindByLabel("PDF-Ansicht")
    heading := tree.FindByLabel("Kapitel 1")

    assert.NotNil(t, pdf)
    assert.Equal(t, a11y.RoleHeading, heading.Role)
}
```

### 10.2 Snapshot-Tests

Geeignet für:

- stabile Node-IDs
- Bounds-Mapping bei Zoom/Scroll
- Merge-Reihenfolge in den globalen `AccessTree`
- Live-Region-Updates für Untertitel

### 10.3 Plattform-Bridge-Tests

Jede Plattform-Bridge sollte validieren können:

- Host-Node + Surface-Subtree werden korrekt exportiert
- Fokus-Wechsel in Surface-Knoten funktionieren
- Aktionen korrekt zurück an den `SemanticProvider` delegiert werden

---

## 11. Offene Fragen

### 11.1 Rollen-Erweiterung

Reichen `RoleCustomBase + n` für Spezialrollen wie Caption, DocumentPage, Annotation? Oder sollte RFC-001 mittelfristig um zusätzliche Standardrollen erweitert werden?

### 11.2 Text-Granularität

Soll ein PDF-Renderer Absätze, Zeilen oder einzelne Text-Runs exportieren? Mehr Granularität verbessert Navigation, erhöht aber Node-Zahl und Update-Kosten.

### 11.3 Selection & Caret

Für editierbare oder textselektionstaugliche Surfaces fehlt noch ein präziser Vertrag für:

- Caret-Position
- Text-Range-Selektion
- `setSelection` / `replaceText`

Das ist insbesondere für Browser- oder Editor-Surfaces relevant und kann ein Folge-RFC werden.

### 11.4 Delegation vs. Mirror

Soll Lux die Semantik immer in den globalen AccessTree spiegeln, oder dürfen Engines in Einzelfällen ihre Plattform-A11y direkt exponieren? Dieses RFC bevorzugt klar den Mirror-Ansatz, weil er konsistent, testbar und plattformneutral ist.

---

## 12. Quellen & Referenzen

- RFC-001-lux.md §8 — Externe Surfaces
- RFC-001-lux.md §11 — Accessibility (A11y)
- RFC-003-lux-widget-catalogue.md §5.7 — Externe Rendering-Grenze
- RFC-004-lux-webview.md — Browser-Engine-Integration via Surface-Slots

---

*RFC-006 — Draft. Feedback via GitHub Issues gegen `github.com/timzifer/lux`.*
