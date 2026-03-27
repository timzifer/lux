# lux — Offene Aufgaben (nach Abhängigkeit geordnet)

**Stand:** 2026-03-26
**Abgeleitet aus:** RFC-001, RFC-002, RFC-003, RFC-007

Die Aufgaben sind in Phasen gegliedert. Jede Phase baut auf den vorherigen auf — innerhalb einer Phase sind die Aufgaben weitgehend unabhängig und parallelisierbar.

---

## Phase 0 — Fundament-Lücken schließen

Diese Aufgaben betreffen Kern-Infrastruktur, auf der spätere Features aufbauen.

### 0.1 Input-System vervollständigen (RFC-002 §2) ✅
- [x] `input.KeyMsg` auf typisierten `Key uint32` + `ModifierSet` Bitfield umstellen (statt `string`-Key)
- [x] `TextInputMsg` (post-IME) als separaten Typ neben `CharMsg` einführen
- [x] `MouseMsg` um `MouseEnter`/`MouseLeave` Action erweitern
- [x] `ScrollMsg` um `Precise bool` (Trackpad vs. Mausrad) und `Pos` erweitern
- [x] `TouchMsg` implementieren (TouchID, Phase, Force)
- **Abhängig von:** nichts (Kern-Package `input/`, keine Imports)

### 0.2 RenderCtx.Events — Input-Dispatch an Widgets (RFC-001 §4, RFC-002 §2.6) ✅
- [x] `RenderCtx` um `Events []InputEvent` Feld erweitern
- [x] `InputEvent` als typisierten Union-Wrapper implementieren
- [x] Framework-internes Dispatching: Mouse → Hit-Test → Widget, Keyboard → Focus → Widget
- **Abhängig von:** 0.1

### 0.3 Focus-Management (RFC-002 §2.3) ✅
- [x] `Focusable`-Interface auf `Widget` definieren (`FocusOptions() FocusOpts`)
- [x] Tab-Reihenfolge aus Layout-Baum ableiten
- [x] `FocusGainedMsg` / `FocusLostMsg` implementieren
- [x] `RequestFocusMsg` / `ReleaseFocusMsg` implementieren
- [x] Bestehenden `FocusState` in die neue Architektur überführen
- **Abhängig von:** 0.2

### 0.4 Animator-Interface — Framework-getriebene Animationen (RFC-002 §1.3) ✅
- [x] `Animator`-Interface auf `WidgetState` definieren: `Tick(dt) bool`
- [x] Animation-Pass im App-Loop vor Reconcile einfügen
- [x] Automatisches Dirty-Marking bei `Tick() == true`
- **Abhängig von:** nichts (nutzt existierendes `anim/` Package)

### 0.5 `Anim[T]` Interpolatable erweitern (RFC-002 §1.4) ✅
- [x] ~~`Interpolatable` Constraint um `draw.Color`, `draw.Point`, `draw.Size`, `draw.Rect`, `draw.CornerRadii` erweitern~~ → bewusst minimal gehalten (nur float32/float64)
- [x] `Lerper[T]`-Pattern um Zyklus `anim/ → draw/` zu vermeiden (`LerpFunc[T]`/`LerpAnim[T]` + alle 5 draw-Typen in `draw/lerp.go`)
- **Abhängig von:** nichts

---

## Phase 1 — Interaktions-Infrastruktur ✅

### 1.1 Cursor-Management (RFC-002 §2.7) ✅
- [x] `CursorKind` Enum und `Cursable`-Interface implementieren
- [x] Platform-Interface um `SetCursor(CursorKind)` erweitern
- [x] GLFW-Backend: Cursor-Änderung umsetzen
- **Abhängig von:** 0.2 (Hit-Test liefert Widget unter Cursor)

### 1.2 Keyboard-Shortcuts (RFC-002 §2.5) ✅
- [x] `Shortcut` Typ und `ShortcutMsg` implementieren
- [x] `app.WithShortcut()` Option
- [x] Plattform-Normalisierung (Cmd↔Ctrl via `PlatformShortcut`)
- **Abhängig von:** 0.1, 0.3

### 1.3 Global Handler Layer (RFC-002 §2.8) ✅
- [x] `GlobalHandler func(InputEvent) bool` definieren
- [x] `app.WithGlobalHandler()` und `RegisterHandlerMsg`/`UnregisterHandlerMsg`
- [x] Dispatch-Reihenfolge: GlobalHandler → Overlay → HitTest → Widget
- **Abhängig von:** 0.2

### 1.4 Kinetic Scrolling (RFC-002 §3) ✅
- [x] `KineticScroll` Typ mit Friction-Decay-Modell implementieren
- [x] `scrollPhase` State-Machine (Idle, Tracking, Decelerating, Snapping)
- [x] Overscroll & Rubber-Band-Rückfederung via Spring
- [x] Velocity-Tracking aus Trackpad-Deltas
- [ ] Bestehenden `ScrollState` in `KineticScroll` überführen (Migration optional, beide koexistieren)
- **Abhängig von:** 0.1 (ScrollMsg.Precise), 0.4 (Animator)

### 1.5 Overlay-System (RFC-002 §5.3) ✅
- [x] `Overlay`-Element mit Anchor, Placement, Dismissable
- [x] Overlay-Layer im Render-Pass (über normalem Layout-Flow)
- [x] Overlay-Input-Priority (vor normalem Hit-Test)
- [x] `DismissOverlayMsg`
- **Abhängig von:** 0.2

---

## Phase 2 — Animation & Layout erweitern ✅

### 2.1 SpringAnim[T] (RFC-002 §1.5) ✅
- [x] Feder-Dämpfer-System implementieren
- [x] `SpringSpec` mit Stiffness, Damping, Mass, SettlingThreshold
- [x] Preset-Springs: Gentle, Snappy, Bouncy
- **Abhängig von:** 0.5

### 2.2 AnimationID / SetTargetWithID (RFC-002 §1.8) ✅
- [x] `AnimationID string` Typ
- [x] `Anim[T].SetTargetWithID()` — sendet `AnimationEnded{ID}` via app.Send
- [x] Framework flush von AnimationEnd-Msgs nach Tick-Pass
- **Abhängig von:** 0.4

### 2.3 AnimGroup / AnimSeq (RFC-002 §1.9) ✅
- [x] `AnimGroup` — parallele Animationen
- [x] `AnimSeq` — sequentielle Animationen mit onDone-Hooks
- **Abhängig von:** 0.5

### 2.4 CubicBezier Easing (RFC-002 §1.10) ✅
- [x] `CubicBezier(x1, y1, x2, y2 float32) EasingFunc` (CSS-kompatibel)
- **Abhängig von:** nichts

### 2.5 MotionSpec mit Easing pro Preset (RFC-002 §1.6) ✅
- [x] `MotionSpec` Slots von `time.Duration` auf `DurationEasing{Duration, EasingFunc}` umstellen
- [x] Theme-Tokens anpassen: Standard (250ms OutCubic), Emphasized (400ms InOutCubic), Quick (100ms OutExpo)
- **Abhängig von:** 2.4

### 2.6 Custom Layout-Interface (RFC-002 §4.3) ✅
- [x] `Layout`-Interface: `LayoutChildren(ctx LayoutCtx, children []Element) Size`
- [x] `LayoutCtx` mit `Measure`, `Place`, `Constraints`, `Theme`
- **Abhängig von:** nichts

### 2.7 Layout-Cache & Invalidierung (RFC-002 §4.8) ✅
- [x] `LayoutCache` Typ (Constraints + Size + ChildRects)
- [x] Invalidierung: `Invalidate()`, `IsValid(Constraints)`, `Store()`
- **Abhängig von:** 2.6

---

## Phase 3 — Rendering-Pipeline ausbauen

### 3.1 Canvas-API vervollständigen (RFC-001 §6.2) ✅
- [x] `PushClipRoundRect`, `PushClipPath`
- [x] `PushBlur` / `PopBlur`
- [x] `PushLayer` / `PopLayer`
- [x] `PushScale`
- [x] `DrawTextLayout` / `NewTextLayout`
- [x] `DrawImageSlice` (9-Slice)
- [x] `DrawTexture` (für Surface-Slots)
- [x] `ArcTo` im PathBuilder
- **Abhängig von:** nichts (erweitert bestehendes `draw/` Interface)

### 3.2 Paint-Varianten (RFC-001 §6.2.3) ✅
- [x] `LinearGradientPaint` implementieren
- [x] `RadialGradientPaint` implementieren
- [x] `PatternPaint` implementieren
- **Abhängig von:** 3.1

### 3.3 MSDF-Text-Rendering (RFC-001 §6.3) ✅
- [x] SDF-Atlas auf Multi-Channel SDF (MSDF) umstellen (pierrec/msdf, NRGBA-Atlas, Median-Filter-Shader)
- [x] Schärfere Kurven bei kleinen Schriftgrößen (Dual-Path: MSDF ≥24px, hinted Bitmap <24px)
- **Abhängig von:** nichts

### 3.4 VTree-Optimierungen (RFC-001 §6.4) ✅
- [x] `Equatable`-Interface für Widget-Equality
- [x] `DirtyTracker`-Interface für explizites Dirty-Marking
- [x] `CacheHint`-Unterstützung in `LayerOptions`
- **Abhängig von:** nichts

### 3.5 Theme-Lookup-Cache (RFC-001 §5.4, RFC-003 §1.4) ✅
- [x] `resolvedCache` pro Theme-Instanz (DrawFunc + TokenSet)
- [x] Cache-Invalidierung bei `SetThemeMsg` / `SetDarkModeMsg`
- [x] Warm-Up in `app.Run` vor erstem Frame
- **Abhängig von:** nichts

---

## Phase 4 — Text-Stack & Fonts ✅

### 4.1 go-text/typesetting Integration (RFC-003 §3.2) ✅
- [x] `GoTextShaper` als Shaper-Implementierung (ersetzt internen sfnt_shaper)
- [x] Vollständiges GSUB/GPOS (Arabisch, Devanagari, CJK etc.)
- **Abhängig von:** nichts

### 4.2 Shaper-Interface (RFC-003 §3.3) ✅
- [x] `Shaper` Interface definieren: `Shape(run ShapingRun, font *Font, size float32) []ShapedGlyph`
- [x] `ShapingRun`, `ShapedGlyph`, `TextDirection` Typen
- **Abhängig von:** 4.1

### 4.3 FontFamily & Fallback-Chain (RFC-003 §3.4) ✅
- [x] `FontFamily` mit `Name`, `Faces map[FontFaceKey]*Font`, `Fallback []*FontFamily`
- [x] Glyph-Fallback pro Glyph (Primary → Fallback-Chain → Embedded → U+FFFD)
- [x] Eingebettetes Noto-Sans Fallback-Font-Subset
- **Abhängig von:** 4.1

### 4.4 BiDi-Unterstützung (RFC-003 §3.5) ✅
- [x] `BidiParagraph()` via `golang.org/x/text/unicode/bidi`
- [x] Mixed-Direction-Text korrekt verarbeiten
- **Abhängig von:** 4.1, 4.2

### 4.5 Unicode Line-Breaking (RFC-003 §3.6) ✅
- [x] `LineBreaker`-Interface mit UAX #14-konformer Implementierung (`internal/text/linebreak.go`)
- [x] Thai-/CJK-kompatible Umbruchregeln via `rivo/uniseg`
- [x] Integration in TextLayout-Pipeline (RFC-003 §5.3)
- **Abhängig von:** 4.1

### 4.6 Grapheme-Cluster-Navigation (RFC-003 §3.7) ✅
- [x] `rivo/uniseg`-Integration für Grapheme-Cluster-Segmentierung (`internal/text/grapheme.go`)
- [x] Cursor-Bewegung auf Grapheme-Cluster-Grenzen (`PrevGraphemeCluster`, `NextGraphemeCluster`)
- [x] Backspace löscht Grapheme-Cluster (Emoji, kombinierte Zeichen)
- [x] Doppelklick-Wortauswahl auf UAX#29 Word Boundaries (`WordAt`)
- **Abhängig von:** nichts

### 4.7 Inline-Widgets in RichText (RFC-003 §5.5) ✅
- [x] `InlineWidget` Typ (Widget im Textfluss) mit Baseline-Alignment (`ui/display/richtext.go`)
- [x] `ParagraphContent` sealed Interface (`TextSpan | InlineWidget`)
- **Abhängig von:** 4.1

---

## Phase 4b — i18n & Layout-Spiegelung ✅

### 4b.1 RTL-Layout-Spiegelung (RFC-002 §4.6) ✅
- [x] `Insets` auf `Start`/`End` statt `Left`/`Right` umstellen
- [x] `LayoutDirection` (LTR/RTL) in `LayoutCtx` propagieren
- [x] `FlexRow` bei RTL automatisch spiegeln
- [x] `JustifyStart`/`AlignStart` bei RTL korrekt auflösen
- [x] Convenience-Konstruktoren: `InlineInsets`, `BlockInsets`, `LogicalInsets`
- **Abhängig von:** nichts (API-Design-Entscheidung — je früher desto besser)
- **Hinweis:** Nachrüsten ist extrem teuer — betrifft die gesamte Layout-API

### 4b.2 Locale-Propagation (RFC-003 §3.8) ✅
- [x] `App.Locale` Feld (BCP 47 string)
- [x] `app.WithLocale()` Option + `SetLocaleMsg` für Laufzeit-Wechsel
- [x] Locale → `LayoutDirection` Ableitung (Arabisch/Hebräisch → RTL)
- [x] Layout-Invalidierung bei Locale-Wechsel
- **Abhängig von:** 4b.1

### 4b.3 IME Compose-Window (RFC-002 §2.2) ✅
- [x] `IMEComposeMsg` und `IMECommitMsg` Typen
- [x] `Platform.SetIMECursorRect()` für Kandidaten-Fenster-Positionierung
- [x] GLFW `glfwSetPreeditCallback` Integration (Stub für 3.3, voll ab 3.4)
- [x] TextField/RichTextEditor: Kompositions-Text visuell unterscheidbar rendern (`InputState.ComposeText`)
- **Abhängig von:** 0.1 (Input-System)

---

## Phase 5 — Platform-Erweiterung ✅

### 5.1 wgpu als GPU-Backend (RFC-001 §6.1) ✅
- [x] wgpu-Shim-Interface (`internal/wgpu/`)
- [x] `wgpu-native` Implementierung (CGo, Default)
- [x] `gogpu/wgpu` Implementierung (pure Go, `-tags gogpu`)
- [x] Migration von OpenGL 3.3 auf wgpu
- **Abhängig von:** 3.1

### 5.2 Native Platform-Backends (RFC-001 §7.2) ✅
- [x] Wayland-Backend
- [x] X11-Backend
- [x] Win32-Backend (Bestehendes `platform/windows/` ausbauen)
- [x] Cocoa/AppKit-Backend
- [x] DRM/KMS-Backend (RFC-001 §7.3)
- **Abhängig von:** nichts (pro Backend unabhängig)

### 5.3 Platform-Interface erweitern (RFC-001 §7.1) ✅
- [x] `SetSize(w, h int)` hinzufügen
- [x] `SetFullscreen(bool)` hinzufügen
- [x] `RequestFrame()` hinzufügen
- [x] `SetCursor(CursorKind)` hinzufügen
- [x] `SetClipboard(text string)` / `GetClipboard() string` hinzufügen
- [x] `CreateSurface(instance wgpu.Instance) wgpu.Surface` hinzufügen
- **Abhängig von:** 5.1 (für Surface-Erstellung)

---

## Phase 6 — Erweiterte Features

### 6.1 Externe Surfaces (RFC-001 §8) ✅
- [x] `Surface`-Element und `SurfaceProvider`-Interface
- [x] Zero-Copy-Pfade (IOSurface, DMA-buf, DXGI)
- [x] Input-Routing an Surfaces
- **Abhängig von:** 5.1

### 6.2 Accessibility (RFC-001 §11, RFC-006)

#### 6.2a Core A11y Types (`a11y/`) ✅
- [x] `AccessRole` mit Konstanten (RoleButton, ..., RoleCustomBase)
- [x] `AccessStates` Struct (Focused, Checked, ..., Live)
- [x] `AccessLiveRegion` (LiveOff, LivePolite, LiveAssertive)
- [x] `AccessAction` / `AccessActionDesc` Typen
- [x] `AccessRelation` / `AccessRelationDesc` Typen
- [x] `AccessRelationKind` (LabelledBy, DescribedBy, Controls, FlowsTo)
- [x] `AccessNode` Struct (Role, Label, Description, Value, Lang, States, Actions, Relations)
- [x] `AccessNodeID` Typ

#### 6.2b Surface Semantic Provider (RFC-006 §5) ✅
- [x] `SemanticProvider` Interface (SnapshotSemantics, HitTestSemantics, FocusSemanticNode, PerformSemanticAction)
- [x] `SurfaceNodeID` Typ
- [x] `SurfaceSemantics` Struct (Roots, Version)
- [x] `SurfaceAccessNode` Struct (ID, Parent, Role, Label, Description, Value, Bounds, Lang, States, Actions, Relations)

#### 6.2c Widget A11y & AccessTree ✅
- [x] `AccessibleWidget`-Interface
- [x] `AccessTree`-Konstruktion aus VTree
- [x] `RenderToAccessTree()` Test-Helper für A11y-Unit-Tests in CI
- [x] Surface-Subtree-Merge in globalen AccessTree (RFC-006 §6)

#### 6.2d FocusTrap (RFC-001 §11.7) ✅
- [x] Fokus-Einschluss bei Modal-Öffnung (Tab/Shift+Tab zyklisch im Dialog)
- [x] Fokus-Rückkehr bei Modal-Schließung (`RestoreFocus`)
- [x] Inhalt außerhalb des Traps aus AccessTree ausblenden

#### 6.2e Plattform-Bridges ✅
- [x] AT-SPI2 Bridge (Linux) — via D-Bus (`godbus`), kein CGo
- [x] UIA Bridge (Windows) — via CGo/COM
- [x] NSAccessibility Bridge (macOS) — via CGo/ObjC
- **Abhängig von:** 0.2, 0.3 (Focus-Management), 4b.2 (Locale für `Lang`-Feld)

### 6.3 State Persistence (RFC-001 §3.4) ✅
- [x] `app.WithPersistence(PersistenceConfig[Model])` Option
- [x] Encode/Decode Hooks
- [x] Plattformspezifische Storage-Pfade (`app/storage.go`: Windows `%APPDATA%`, macOS `~/Library/Application Support`, Linux `XDG_STATE_HOME`)
- **Abhängig von:** nichts

### 6.4 Commands (`Cmd`) (RFC-001 §3.6) ✅
- [x] `type Cmd func() Msg` definieren
- [x] `UpdateWithCmd[M]` Signatur: `func(M, Msg) (M, Cmd)`
- [x] `app.Run` für beide Signaturen erweitern
- [x] `Batch` für kombinierte Commands
- **Abhängig von:** nichts

### 6.5 Sub-Models (RFC-001 §3.5) ✅
- [x] `SubModel[Parent, Child]` mit Get/Set/Update
- [x] Delegation im Haupt-Loop
- [x] `SubModelWithCmd` / `DelegateWithCmd` Variante
- **Abhängig von:** nichts

### 6.6 DynamicDataset (RFC-002 §6)
- [ ] `Dataset[ID]`-Interface
- [ ] `SliceDataset`, `PagedDataset`, `StreamDataset`
- [ ] Integration mit VirtualList und Tree
- **Abhängig von:** nichts

### 6.7 Inspector & Debugging (RFC-001 §12)
- [ ] Debug-Protocol via TCP/Unix-Socket
- [ ] VTree-Streaming
- [ ] Frame-Metriken
- [ ] Widget-Inspector als separates Binary
- **Abhängig von:** 3.4 (DirtyTracker für Paint-Highlighting)

---

## Phase 7 — Post-v1.0

### 7.1 Tier 4 Widgets (RFC-003 §4.1) 🔶
- [x] DatePicker (`ui/form/datepicker.go`)
- [x] ColorPicker (`ui/form/colorpicker.go`)
- [x] TimePicker (`ui/form/timepicker.go`)
- [x] NumericInput (`ui/form/numericinput.go`)
- [x] Spinner (`ui/form/spinner.go`)
- [x] SplitView (`ui/nav/splitview.go`)
- [ ] DataTable
- [x] Toolbar
- [ ] RichTextEditor (RFC-003 §5.6)
- [ ] FilePicker (Open/Save)
- **Abhängig von:** Phase 0–2 vollständig abgeschlossen

### 7.2 Widget-Spezifikations-Templates (RFC-003 §4.2)
- [ ] Detaillierte Spezifikation pro Widget (Props, WidgetState, Msgs, A11y, DrawFunc, Tokens)
- **Abhängig von:** 6.2 (A11y-Felder in Template)

### 7.3 Hot-Reload (RFC-001 §12)
- [ ] View-Hot-Reload via Plugin oder Rebuild-Trigger
- [ ] Model-Migration für Update-Hot-Reload
- **Abhängig von:** 6.3 (State Persistence)

---

## Legende

| Symbol | Bedeutung |
|--------|-----------|
| ✅ | Integriert |
| 🔶 | Teilweise integriert |
| ⏳ | Wartend |
| ⏸ | Theoretical / nicht geplant |
