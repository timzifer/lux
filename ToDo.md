# lux — Offene Aufgaben (nach Abhängigkeit geordnet)

**Stand:** 2026-03-18
**Abgeleitet aus:** RFC-001, RFC-002, RFC-003

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

### 0.2 RenderCtx.Events — Input-Dispatch an Widgets (RFC-001 §4, RFC-002 §2.6) 🔶
- [x] `RenderCtx` um `Events []InputEvent` Feld erweitern
- [x] `InputEvent` als typisierten Union-Wrapper implementieren
- [ ] Framework-internes Dispatching: Mouse → Hit-Test → Widget, Keyboard → Focus → Widget
- **Abhängig von:** 0.1

### 0.3 Focus-Management (RFC-002 §2.3) 🔶
- [x] `Focusable`-Interface auf `Widget` definieren (`FocusOptions() FocusOpts`)
- [ ] Tab-Reihenfolge aus Layout-Baum ableiten
- [x] `FocusGainedMsg` / `FocusLostMsg` implementieren
- [x] `RequestFocusMsg` / `ReleaseFocusMsg` implementieren
- [ ] Bestehenden `FocusState` in die neue Architektur überführen
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

## Phase 1 — Interaktions-Infrastruktur

### 1.1 Cursor-Management (RFC-002 §2.7)
- [ ] `CursorKind` Enum und `Cursable`-Interface implementieren
- [ ] Platform-Interface um `SetCursor(CursorKind)` erweitern
- [ ] GLFW-Backend: Cursor-Änderung umsetzen
- **Abhängig von:** 0.2 (Hit-Test liefert Widget unter Cursor)

### 1.2 Keyboard-Shortcuts (RFC-002 §2.5)
- [ ] `Shortcut` Typ und `ShortcutMsg` implementieren
- [ ] `app.WithShortcut()` Option
- [ ] Plattform-Normalisierung (Cmd↔Ctrl via `PlatformShortcut`)
- **Abhängig von:** 0.1, 0.3

### 1.3 Global Handler Layer (RFC-002 §2.8)
- [ ] `GlobalHandler func(InputEvent) bool` definieren
- [ ] `app.WithGlobalHandler()` und `RegisterHandlerMsg`/`UnregisterHandlerMsg`
- [ ] Dispatch-Reihenfolge: GlobalHandler → Overlay → HitTest → Widget
- **Abhängig von:** 0.2

### 1.4 Kinetic Scrolling (RFC-002 §3)
- [ ] `KineticScroll` Typ mit Friction-Decay-Modell implementieren
- [ ] `scrollPhase` State-Machine (Idle, Tracking, Decelerating, Snapping)
- [ ] Overscroll & Rubber-Band-Rückfederung via Spring
- [ ] Velocity-Tracking aus Trackpad-Deltas
- [ ] Bestehenden `ScrollState` in `KineticScroll` überführen
- **Abhängig von:** 0.1 (ScrollMsg.Precise), 0.4 (Animator)

### 1.5 Overlay-System (RFC-002 §5.3)
- [ ] `Overlay`-Element mit Anchor, Placement, Dismissable
- [ ] Overlay-Layer im Render-Pass (über normalem Layout-Flow)
- [ ] Overlay-Input-Priority (vor normalem Hit-Test)
- [ ] `DismissOverlayMsg`
- **Abhängig von:** 0.2

---

## Phase 2 — Animation & Layout erweitern

### 2.1 SpringAnim[T] (RFC-002 §1.5)
- [ ] Feder-Dämpfer-System implementieren
- [ ] `SpringSpec` mit Stiffness, Damping, Mass, SettlingThreshold
- [ ] Preset-Springs: Gentle, Snappy, Bouncy
- **Abhängig von:** 0.5

### 2.2 AnimationID / SetTargetWithID (RFC-002 §1.8)
- [ ] `AnimationID string` Typ
- [ ] `Anim[T].SetTargetWithID()` — sendet `AnimationEnded{ID}` via app.Send
- [ ] Framework flush von AnimationEnd-Msgs nach Tick-Pass
- **Abhängig von:** 0.4

### 2.3 AnimGroup / AnimSeq (RFC-002 §1.9)
- [ ] `AnimGroup` — parallele Animationen
- [ ] `AnimSeq` — sequentielle Animationen mit onDone-Hooks
- **Abhängig von:** 0.5

### 2.4 CubicBezier Easing (RFC-002 §1.10)
- [ ] `CubicBezier(x1, y1, x2, y2 float32) EasingFunc` (CSS-kompatibel)
- **Abhängig von:** nichts

### 2.5 MotionSpec mit Easing pro Preset (RFC-002 §1.6)
- [ ] `MotionSpec` Slots von `time.Duration` auf `DurationEasing{Duration, EasingFunc}` umstellen
- [ ] Theme-Tokens anpassen: Standard (250ms OutCubic), Emphasized (400ms InOutCubic), Quick (100ms OutExpo)
- **Abhängig von:** 2.4

### 2.6 Custom Layout-Interface (RFC-002 §4.3)
- [ ] `Layout`-Interface: `LayoutChildren(ctx LayoutCtx, children []Widget) Size`
- [ ] `LayoutCtx` mit `Measure`, `Place`, `Constraints`, `Theme`
- **Abhängig von:** nichts

### 2.7 Layout-Cache & Invalidierung (RFC-002 §4.8)
- [ ] `layoutCache` pro VNode (Constraints + Size + ChildRects)
- [ ] Dirty-Propagation: Props geändert → Layout-Dirty aufwärts
- **Abhängig von:** 2.6

---

## Phase 3 — Rendering-Pipeline ausbauen

### 3.1 Canvas-API vervollständigen (RFC-001 §6.2)
- [ ] `PushClipRoundRect`, `PushClipPath`
- [ ] `PushBlur` / `PopBlur`
- [ ] `PushLayer` / `PopLayer`
- [ ] `PushScale`
- [ ] `DrawTextLayout` / `NewTextLayout`
- [ ] `DrawImageSlice` (9-Slice)
- [ ] `DrawTexture` (für Surface-Slots)
- [ ] `ArcTo` im PathBuilder
- **Abhängig von:** nichts (erweitert bestehendes `draw/` Interface)

### 3.2 Paint-Varianten (RFC-001 §6.2.3)
- [ ] `LinearGradientPaint` implementieren
- [ ] `RadialGradientPaint` implementieren
- [ ] `PatternPaint` implementieren
- **Abhängig von:** 3.1

### 3.3 MSDF-Text-Rendering (RFC-001 §6.3) ✅
- [x] SDF-Atlas auf Multi-Channel SDF (MSDF) umstellen (pierrec/msdf, NRGBA-Atlas, Median-Filter-Shader)
- [x] Schärfere Kurven bei kleinen Schriftgrößen (Dual-Path: MSDF ≥24px, hinted Bitmap <24px)
- **Abhängig von:** nichts

### 3.4 VTree-Optimierungen (RFC-001 §6.4)
- [ ] `Equatable`-Interface für Widget-Equality
- [ ] `DirtyTracker`-Interface für explizites Dirty-Marking
- [ ] `CacheHint`-Unterstützung in `LayerOptions`
- **Abhängig von:** nichts

### 3.5 Theme-Lookup-Cache (RFC-001 §5.4, RFC-003 §1.4)
- [ ] `resolvedCache` pro Theme-Instanz (DrawFunc + TokenSet)
- [ ] Cache-Invalidierung bei `SetThemeMsg` / `SetDarkModeMsg`
- [ ] Warm-Up in `app.Run` vor erstem Frame
- **Abhängig von:** nichts

---

## Phase 4 — Text-Stack & Fonts

### 4.1 go-text/typesetting Integration (RFC-003 §3.2)
- [ ] `GoTextShaper` als Shaper-Implementierung (ersetzt internen sfnt_shaper)
- [ ] Vollständiges GSUB/GPOS (Arabisch, Devanagari, CJK etc.)
- **Abhängig von:** nichts

### 4.2 Shaper-Interface (RFC-003 §3.3)
- [ ] `Shaper` Interface definieren: `Shape(run ShapingRun, font *Font, size float32) []ShapedGlyph`
- [ ] `ShapingRun`, `ShapedGlyph`, `TextDirection` Typen
- **Abhängig von:** 4.1

### 4.3 FontFamily & Fallback-Chain (RFC-003 §3.4)
- [ ] `FontFamily` mit `Name`, `Faces map[FontFaceKey]*Font`, `Fallback []*FontFamily`
- [ ] Glyph-Fallback pro Glyph (Primary → Fallback-Chain → Embedded → U+FFFD)
- [ ] Eingebettetes Noto-Sans Fallback-Font-Subset
- **Abhängig von:** 4.1

### 4.4 BiDi-Unterstützung (RFC-003 §3.5)
- [ ] `BidiParagraph()` via `golang.org/x/text/unicode/bidi`
- [ ] Mixed-Direction-Text korrekt verarbeiten
- **Abhängig von:** 4.1, 4.2

### 4.5 Inline-Widgets in RichText (RFC-003 §5.5)
- [ ] `InlineWidget` Typ (Widget im Textfluss)
- [ ] `ParagraphContent` Interface (`TextSpan | InlineWidget`)
- **Abhängig von:** 4.1

---

## Phase 5 — Platform-Erweiterung

### 5.1 wgpu als GPU-Backend (RFC-001 §6.1)
- [ ] wgpu-Shim-Interface (`internal/wgpu/`)
- [ ] `wgpu-native` Implementierung (CGo, Default)
- [ ] `gogpu/wgpu` Implementierung (pure Go, `-tags gogpu`)
- [ ] Migration von OpenGL 3.3 auf wgpu
- **Abhängig von:** 3.1

### 5.2 Native Platform-Backends (RFC-001 §7.2)
- [ ] Wayland-Backend
- [ ] X11-Backend
- [ ] Win32-Backend (Bestehendes `platform/windows/` ausbauen)
- [ ] Cocoa/AppKit-Backend
- [ ] DRM/KMS-Backend (RFC-001 §7.3)
- **Abhängig von:** nichts (pro Backend unabhängig)

### 5.3 Platform-Interface erweitern (RFC-001 §7.1)
- [ ] `SetSize(w, h int)` hinzufügen
- [ ] `SetFullscreen(bool)` hinzufügen
- [ ] `RequestFrame()` hinzufügen
- [ ] `SetCursor(CursorKind)` hinzufügen
- [ ] `SetClipboard(text string)` / `GetClipboard() string` hinzufügen
- [ ] `CreateSurface(instance wgpu.Instance) wgpu.Surface` hinzufügen
- **Abhängig von:** 5.1 (für Surface-Erstellung)

---

## Phase 6 — Erweiterte Features

### 6.1 Externe Surfaces (RFC-001 §8)
- [ ] `Surface`-Element und `SurfaceProvider`-Interface
- [ ] Zero-Copy-Pfade (IOSurface, DMA-buf, DXGI)
- [ ] Input-Routing an Surfaces
- **Abhängig von:** 5.1

### 6.2 Accessibility (RFC-001 §11)
- [ ] `AccessibleWidget`-Interface
- [ ] `AccessNode`, `AccessRole`, `AccessStates` Typen
- [ ] `AccessTree`-Konstruktion aus VTree
- [ ] AT-SPI2 Bridge (Linux)
- [ ] UIA Bridge (Windows)
- [ ] NSAccessibility Bridge (macOS)
- **Abhängig von:** 0.2, 0.3 (Focus-Management für A11y nötig)

### 6.3 State Persistence (RFC-001 §3.4)
- [ ] `app.WithPersistence(PersistenceConfig[Model])` Option
- [ ] Encode/Decode Hooks
- [ ] Plattformspezifische Storage-Pfade
- **Abhängig von:** nichts

### 6.4 Commands (`Cmd`) (RFC-001 §3.6)
- [ ] `type Cmd func() Msg` definieren
- [ ] `UpdateWithCmd[M]` Signatur: `func(M, Msg) (M, Cmd)`
- [ ] `app.Run` für beide Signaturen erweitern
- **Abhängig von:** nichts

### 6.5 Sub-Models (RFC-001 §3.5)
- [ ] `SubModel[Parent, Child]` mit Get/Set/Update
- [ ] Delegation im Haupt-Loop
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

### 7.1 Tier 4 Widgets (RFC-003 §4.1)
- [ ] DatePicker
- [ ] ColorPicker
- [ ] DataTable
- [ ] SplitView
- [ ] Toolbar
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
