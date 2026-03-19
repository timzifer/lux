# RFC-004 — lux/surface/webview: Browser-Engine-Integration via Surface-Slots

**Repository:** `github.com/timzifer/lux`

**Status:** Theoretical
**Version:** 0.1.0
**Datum:** 2026-03-19
**Abhängigkeit:** RFC-001-lux.md §8 (Surface-Slots)

---

## Inhaltsverzeichnis

1. [Motivation & Ziel](#1-motivation--ziel)
2. [Abgrenzung](#2-abgrenzung)
3. [Evaluierte Engines](#3-evaluierte-engines)
4. [Bewertungsmatrix](#4-bewertungsmatrix)
5. [Architektur: Plattform-nativer OS-Shim](#5-architektur-plattform-nativer-os-shim)
6. [Linux: WPE WebKit](#6-linux-wpe-webkit)
7. [Windows: WebView2](#7-windows-webview2)
8. [macOS: Servo (experimentell)](#8-macos-servo-experimentell)
9. [Build-Tag `-tags servo`: Servo auf allen Plattformen](#9-build-tag--tags-servo-servo-auf-allen-plattformen)
10. [Fallback-Strategie](#10-fallback-strategie)
11. [Vergleich: OS-Shim vs. CEF-Monolith](#11-vergleich-os-shim-vs-cef-monolith)
12. [Offene Fragen](#12-offene-fragen)
13. [Quellen](#13-quellen)

---

## 1. Motivation & Ziel

Lux (RFC-001) stellt mit dem Surface-Slot-Mechanismus (§8) einen Escape-Hatch bereit, über den externe Renderer — Browser-Engines, Video-Decoder, 3D-Engines — als `SurfaceProvider` an den Widget-Baum andocken. Das Framework selbst rendert kein HTML/CSS (§10 Nicht-Ziele), aber viele Anwendungen benötigen Web-Content:

- Markdown-Preview mit vollständigem HTML/CSS-Rendering
- Code-Editor via CodeMirror/Monaco
- OAuth-Login-Flows
- Eingebettete Web-Applikationen

Diese RFC spezifiziert `lux/surface/webview` — ein **plattform-natives OS-Shim**, das pro Betriebssystem die jeweils beste Browser-Engine nutzt und über ein gemeinsames Go-Interface exponiert.

---

## 2. Abgrenzung

| In Scope | Nicht in Scope |
|---|---|
| Browser-Engine-Auswahl pro Plattform | HTML/CSS als Framework-Renderer (bleibt §10 Nicht-Ziel) |
| `SurfaceProvider`-Implementierung für Web-Content | JavaScript-Bridge / Bidirektionale JS↔Go-Kommunikation |
| Zero-Copy Texture-Sharing wo möglich | Eigene Browser-Engine entwickeln |
| Build-Tag für Servo-Override | CEF-Integration (evaluiert, aber nicht empfohlen) |
| Fallback auf CPU-Copy (§8.2) | WebRTC, WebGL-Passthrough |

---

## 3. Evaluierte Engines

### 3.1 CEF (Chromium Embedded Framework)

| Eigenschaft | Bewertung |
|---|---|
| OSR | `OnAcceleratedPaint()` — GPU-Texturen direkt, stabil |
| Zero-Copy | DXGI (Windows, stabil), IOSurface (macOS), DMA-buf/GBM (Linux, experimentell) |
| Web-Kompatibilität | 5/5 — es *ist* Chromium |
| Go-Integration | C-API (`cef_capi.h`) → CGo direkt möglich |
| Binary-Größe | ~150–200 MB pro Plattform |
| RAM | ~100–150 MB Minimum (Multi-Prozess) |
| Lizenz | BSD-3-Clause |

Bekannte Probleme: Linux Shared-Texture-OSR erfordert ANGLE/EGL, Nvidia-GBM-Inkompatibilitäten. `OnAcceleratedPaint()` liefert keine Dirty-Rects.

**Ergebnis:** Funktional stark, aber Binary-Größe und RAM-Verbrauch widersprechen Lux' Leichtgewichts-Philosophie. Als Monolith-Lösung nicht empfohlen, aber valide Alternative falls der OS-Shim-Ansatz scheitert.

### 3.2 Servo

| Eigenschaft | Bewertung |
|---|---|
| OSR | Unterstützt, wgpu-intern — potentiell bester Zero-Copy-Pfad |
| Zero-Copy | wgpu ↔ wgpu = Shared TextureView ohne OS-Level Texture-Sharing möglich |
| Web-Kompatibilität | 2/5 — viele CSS/JS-Features noch unvollständig |
| Go-Integration | Kein stabiles C-FFI. `libservo` experimentell |
| Binary-Größe | ~50–80 MB |
| Lizenz | MPL-2.0 |

Architektonisch der beste Fit (gleiche GPU-Abstraktion), aber Embedding-API (v0.0.4) und Web-Kompatibilität noch nicht produktionsreif. Neues Delegate-basiertes WebView-API seit Feb 2025 in aktiver Entwicklung.

**Ergebnis:** Langfristig vielversprechendste Engine. Default auf macOS, opt-in auf allen Plattformen via `-tags servo`.

### 3.3 WPE WebKit

| Eigenschaft | Bewertung |
|---|---|
| OSR | Ja, DMA-buf Zero-Copy als First-Class-Feature |
| Zero-Copy | GPU Texture Atlas + DMA-buf Worker-Thread-Upload, produktionsreif |
| Web-Kompatibilität | 4/5 — Safari/WebKit-Level |
| Go-Integration | GObject C-API → CGo möglich |
| Binary-Größe | ~60–100 MB |
| Cross-Platform | **Nur Linux** — kein nativer macOS/Windows-Support |
| Lizenz | LGPLv2.1 |

Exzellent für Linux. Real-World-Referenz: Neomacs nutzt WPE + DMA-buf für Zero-Copy-Embedding. Neue WPEPlatform-API (Richtung 1.0) in Entwicklung.

**Ergebnis:** Klare Wahl für Linux. Produktionsreif, aktiv gepflegt (Igalia).

### 3.4 WebView2

| Eigenschaft | Bewertung |
|---|---|
| OSR | `ICoreWebView2CompositionController` → DXGI Shared Handle |
| Zero-Copy | DXGI Shared Handle — stabil |
| Web-Kompatibilität | 5/5 — Chromium (Edge) |
| Go-Integration | COM-API → CGo (oder go-ole) |
| Binary-Größe | ~0 MB — vorinstalliert ab Windows 11, Evergreen-Runtime für Windows 10 |
| Cross-Platform | **Nur Windows** |
| Lizenz | Proprietär, aber kostenlos und redistributierbar |

**Ergebnis:** Klare Wahl für Windows. Zero Binary-Overhead, automatische Updates via OS/Edge.

### 3.5 Ultralight — Ausgeschlossen

Proprietäre Lizenz für kommerzielle Nutzung. Inkompatibel mit Open-Source-Projekt.

### 3.6 go-webview / System-WebView — Ausgeschlossen

Kein OSR, kein Textur-Export, keine Input-Injection. Unbrauchbar für Surface-Slot-Architektur.

---

## 4. Bewertungsmatrix

| Kriterium (Gewicht) | CEF | Servo | WPE WebKit | WebView2 |
|---|---|---|---|---|
| OSR-Fähigkeit (20%) | 5/5 | 4/5 | 5/5 | 5/5 |
| Zero-Copy Texture-Sharing (20%) | 4/5 | 5/5 | 5/5 | 5/5 |
| Web-Kompatibilität (15%) | 5/5 | 2/5 | 4/5 | 5/5 |
| Go-Integration (10%) | 4/5 | 2/5 | 3/5 | 4/5 |
| Cross-Platform (10%) | 5/5 | 4/5 | 2/5 | 1/5 |
| Binary-Größe/RAM (10%) | 1/5 | 3/5 | 3/5 | 5/5 |
| Lizenz (5%) | 5/5 | 5/5 | 5/5 | 4/5 |
| Langzeit-Viabilität (10%) | 5/5 | 4/5 | 4/5 | 5/5 |

---

## 5. Architektur: Plattform-nativer OS-Shim

Statt einer einzigen Cross-Platform-Engine wird ein **OS-Shim** empfohlen, der pro Plattform die jeweils beste Engine nutzt. Gemeinsames Interface, plattformspezifische Implementierung via Build-Tags — konsistent mit dem Platform-Ansatz in RFC-001 §7.

### 5.1 Paketstruktur

```
lux/surface/webview/
├── webview.go               // Gemeinsames Interface + Typen
├── webview_linux.go          // → WPE WebKit + DMA-buf       (Default Linux)
├── webview_windows.go        // → WebView2 + DXGI            (Default Windows)
├── webview_darwin.go         // → Servo + wgpu               (Default macOS)
├── webview_servo.go          // → Servo + wgpu               (Build-Tag: servo)
├── webview_linux_servo.go    // → Servo-Override für Linux    (Build-Tag: servo)
└── webview_windows_servo.go  // → Servo-Override für Windows  (Build-Tag: servo)
```

### 5.2 Build-Tag-Auswahl

```
go build                      → Linux: WPE, Windows: WebView2, macOS: Servo
go build -tags servo          → Alle Plattformen: Servo
```

### 5.3 Gemeinsames Interface

```go
package webview

import "github.com/timzifer/lux/surface"

// WebView ist ein SurfaceProvider der Web-Content rendert.
// Die konkrete Browser-Engine wird plattformspezifisch gewählt.
type WebView struct {
    surface.Base
}

// Option konfiguriert einen WebView.
type Option func(*config)

// New erstellt einen neuen WebView mit der plattformspezifischen Engine.
func New(url string, opts ...Option) *WebView

// Navigate lädt eine neue URL.
func (w *WebView) Navigate(url string)

// Eval führt JavaScript im Kontext der Seite aus.
func (w *WebView) Eval(js string) error

// Close gibt alle Engine-Ressourcen frei.
func (w *WebView) Close() error

// --- SurfaceProvider-Interface (aus RFC-001 §8) ---

// AcquireFrame liefert die aktuelle Textur des Web-Contents.
func (w *WebView) AcquireFrame(bounds Rect) (wgpu.TextureView, FrameToken)

// ReleaseFrame gibt den Frame zurück.
func (w *WebView) ReleaseFrame(token FrameToken)

// HandleMsg routet Input-Events an die Engine.
func (w *WebView) HandleMsg(msg Msg) bool
```

### 5.4 Element-Integration

```go
func view(m Model) ui.Element {
    return ui.Column(
        ui.Text("Meine App"),
        // WebView als Surface-Slot im Widget-Baum
        webview.Element(m.webview,
            webview.URL("https://example.com"),
            ui.FlexGrow(1),
        ),
        ui.Button("Zurück", MsgBack{}),
    )
}
```

---

## 6. Linux: WPE WebKit

### 6.1 Integrationspfad

```
WPE WebKit Render → DMA-buf fd
    → wgpu Import External Memory (VK_EXT_external_memory_dma_buf)
    → wgpu.TextureView
    → SurfaceProvider.AcquireFrame() return
```

### 6.2 Details

| Aspekt | Detail |
|---|---|
| Engine | WPE WebKit (via WPEPlatform-API) |
| Zero-Copy | DMA-buf → wgpu External Memory — **produktionsreif** |
| Go-Anbindung | GObject C-API → CGo |
| Web-Kompatibilität | Safari/WebKit-Level (4/5) |
| Binary-Overhead | ~60–100 MB |
| Status | **Stabil.** Igalia pflegt aktiv, WPEPlatform-API Richtung 1.0. |

### 6.3 Referenz-Implementierungen

- **Neomacs** (GPU-powered Emacs in Rust): nutzt WPE + DMA-buf für Zero-Copy-Embedding von Web-Content in eigene GPU-Pipeline. Demonstriert die Machbarkeit des hier beschriebenen Ansatzes.

### 6.4 CGo-Anbindung (Skizze)

```go
//go:build linux && !servo

package webview

/*
#cgo pkg-config: wpe-webkit-2.0 wpe-platform-1.0
#include <wpe/webkit.h>
#include <wpe/wpe-platform.h>

// Callback wenn ein neuer Frame verfügbar ist
extern void goOnFrameReady(void *userdata, int dmabuf_fd, int width, int height);
*/
import "C"
```

---

## 7. Windows: WebView2

### 7.1 Integrationspfad

```
WebView2 CompositionController → DXGI Shared Handle
    → wgpu Import External Memory (ID3D12Resource)
    → wgpu.TextureView
    → SurfaceProvider.AcquireFrame() return
```

### 7.2 Details

| Aspekt | Detail |
|---|---|
| Engine | WebView2 (Edge/Chromium, Microsoft) |
| Zero-Copy | `ICoreWebView2CompositionController` → DXGI Shared Handle |
| Go-Anbindung | COM-API → CGo (oder go-ole) |
| Web-Kompatibilität | Chromium-Level (5/5) |
| Binary-Overhead | ~0 MB — vorinstalliert ab Windows 11 |
| Status | **Stabil.** Automatische Updates über Edge-Kanal. |

### 7.3 Vorteil gegenüber CEF

WebView2 nutzt die bereits installierte Edge-Runtime. Kein eigener Chromium-Build nötig, kein ~200 MB Binary-Overhead, automatische Security-Updates über OS/Edge-Kanal.

---

## 8. macOS: Servo (experimentell)

> **⚠ Achtung:** Servo ist ein aktives Forschungs-/Entwicklungsprojekt. Die Embedding-API (`libservo`) ist **nicht stabil** (v0.0.4), die Web-Kompatibilität ist **lückenhaft**, und es existiert noch kein stabiles C-FFI. Die Integration auf macOS ist bewusst als **experimentell / best-effort** einzustufen. Produktionskritische Anwendungen müssen jederzeit auf den §8.2 Fallback-Pfad (CPU-Copy) zurückfallen können.

### 8.1 Integrationspfad

```
Servo (wgpu-intern) → wgpu.TextureView direkt
    → SurfaceProvider.AcquireFrame() return
    // Kein IOSurface nötig — gleiche wgpu-Instanz
```

### 8.2 Details

| Aspekt | Detail |
|---|---|
| Engine | Servo (via `libservo` / C-FFI, sobald stabil) |
| Zero-Copy | wgpu ↔ wgpu = **Shared TextureView direkt** — kein OS-Level Texture-Sharing nötig |
| Go-Anbindung | C-FFI → CGo (sobald `libservo` stabil). Alternativ: IPC als Brücke |
| Web-Kompatibilität | 2/5 — viele CSS/JS-Features unvollständig |
| Binary-Overhead | ~50–80 MB |
| Status | **WIP.** Neues Delegate-basiertes WebView-API seit Feb 2025. Linux Foundation Projekt. |

### 8.3 Warum Servo auf macOS (und nicht CEF)?

1. **wgpu → wgpu** ist der architektonisch sauberste Pfad — Zero-Copy ohne OS-Primitiv
2. Servo ist ~50 MB statt ~200 MB (CEF)
3. macOS hat keine leichtgewichtige OSR-Alternative (`WKWebView` bietet kein Texture-Sharing)
4. Frühe Investition in die langfristig vielversprechendste Engine
5. Der §8.2 Fallback-Pfad (CPU-Copy) fängt Servo-Lücken zuverlässig auf

### 8.4 Beobachtungs-Meilensteine

Upgrade von "experimentell" auf "stabil" wenn folgende Meilensteine erreicht sind:

- [ ] Stabile C-FFI (`libservo` als versionierte API)
- [ ] CSS Grid + Flexbox vollständig
- [ ] Servo 1.0 Release
- [ ] Mindestens eine produktive Embedding-Referenz

---

## 9. Build-Tag `-tags servo`: Servo auf allen Plattformen

Servo unterstützt Linux, macOS und Windows. Obwohl der OS-Shim auf Linux und Windows jeweils nativere Engines bevorzugt, soll es möglich sein, **Servo bewusst auf allen Plattformen zu testen und zu nutzen**.

### 9.1 Motivation

- **Konsistenz-Tests:** Gleiches Rendering-Verhalten auf allen Plattformen verifizieren
- **Servo-Entwicklung:** Servo-Bugs auf Linux/Windows reproduzieren und melden
- **Zukunftsinvestition:** Wenn Servo reift, kann es zum Default auf allen Plattformen werden
- **Experimentieren:** Entwickler die Servo evaluieren wollen, brauchen einen einfachen Weg

### 9.2 Build-Tag-Mechanik

```
go build -tags servo
```

Dieser Tag aktiviert auf **allen** Plattformen die Servo-Implementierung statt der plattform-nativen Engine:

```go
//go:build servo

package webview

// Servo-Implementierung wird auf allen Plattformen verwendet.
// Überschreibt WPE WebKit (Linux) und WebView2 (Windows).
```

### 9.3 Datei-Auswahl via Build-Constraints

| Datei | Build-Constraint | Aktiv wenn |
|---|---|---|
| `webview_linux.go` | `//go:build linux && !servo` | Linux ohne `-tags servo` |
| `webview_windows.go` | `//go:build windows && !servo` | Windows ohne `-tags servo` |
| `webview_darwin.go` | `//go:build darwin` | macOS (immer Servo) |
| `webview_servo.go` | `//go:build servo \|\| darwin` | `-tags servo` oder macOS |

macOS nutzt **immer** Servo — der Tag ändert dort nichts. Auf Linux und Windows ersetzt `-tags servo` die native Engine.

### 9.4 Servo-Integrationspfad (alle Plattformen)

```
Servo (wgpu-intern) → wgpu.TextureView direkt
    → SurfaceProvider.AcquireFrame() return
```

Da Servo intern wgpu nutzt und Lux ebenfalls auf wgpu basiert, entfällt auf allen Plattformen der Umweg über OS-Level Texture-Sharing (DMA-buf, DXGI, IOSurface). Die TextureView kann direkt geteilt werden — vorausgesetzt beide nutzen dieselbe wgpu-Instanz oder kompatible Adapter.

### 9.5 CGo-Anbindung an Servo (Skizze)

```go
//go:build servo || darwin

package webview

/*
#cgo LDFLAGS: -lservo
#include <servo/servo.h>

// Servo rendert in eine wgpu-Textur.
// Die TextureView wird direkt an den SurfaceProvider durchgereicht.
extern void goOnServoFrameReady(void *userdata, void *texture_view);
*/
import "C"
```

> **Hinweis:** Die exakte C-FFI-Signatur hängt von der Stabilisierung von `libservo` ab. Die obige Skizze ist konzeptionell — die tatsächliche API wird sich ändern.

### 9.6 Erwartete Einschränkungen mit `-tags servo`

| Einschränkung | Detail |
|---|---|
| Web-Kompatibilität | Deutlich unter Chromium/WebKit-Level. Komplexe Seiten können fehlerhaft rendern. |
| Stabilität | Crashes bei bestimmtem Web-Content sind möglich. |
| Performance | Noch nicht auf dem Niveau von CEF/WebView2 optimiert. |
| Features | Kein WebRTC, eingeschränktes WebGL, lückenhafte CSS-Unterstützung. |

Diese Einschränkungen gelten für den aktuellen Stand von Servo (v0.0.4). Der Tag existiert explizit, um den Fortschritt zu verfolgen und die Engine zu testen.

---

## 10. Fallback-Strategie

Auf allen Plattformen gilt: wenn die primäre Engine nicht verfügbar ist oder ein Feature nicht unterstützt, greift der RFC-001 §8.2 Fallback-Pfad:

```
Engine OSR → Shared Memory Buffer → CPU-Copy → wgpu Upload → TextureView
```

Dies ist langsamer (kein Zero-Copy), aber universell und stellt sicher, dass Surface-Slots auch bei Engine-Problemen funktionieren.

### 10.1 Fallback-Auslöser

| Auslöser | Verhalten |
|---|---|
| Engine nicht installiert (z.B. WPE fehlt) | Fehler beim `New()` — Anwendung entscheidet |
| Shared-Texture nicht unterstützt | Automatischer Fallback auf CPU-Copy |
| Engine-Crash | `SurfaceProvider` signalisiert Fehler, Anwendung kann neu laden |
| `-tags servo` + Servo nicht gebaut | Compile-Fehler — bewusste Entscheidung |

---

## 11. Vergleich: OS-Shim vs. CEF-Monolith

| Aspekt | OS-Shim (WPE/WebView2/Servo) | CEF überall |
|---|---|---|
| Binary-Größe | ~0 MB (Win) / ~60 MB (Linux) / ~50 MB (macOS) | ~200 MB pro Plattform |
| Zero-Copy-Qualität | Produktionsreif (Linux, Windows), experimentell (macOS) | Stabil (Win/macOS), fragil (Linux) |
| Wartung | Plattform-Updates automatisch (Win), Igalia (Linux), Community (macOS) | Chromium-Release-Zyklus, eigener Build |
| Web-Kompatibilität | Konsistent auf Win/Linux, eingeschränkt macOS | Konsistent überall (Chromium) |
| Komplexität | Drei Backends pflegen | Ein Backend, aber Chromium-Build-System |
| Philosophie | Passt zu Lux RFC-001 §7 (Build-Tags pro Plattform) | Monolithischer Fremdkörper |
| Security-Updates | Automatisch (Win), schnell (Linux/Igalia), Community (macOS) | Manuell, Chromium-Zyklus |

---

## 12. Offene Fragen

### 12.1 Servo C-FFI Timeline

Wann wird `libservo` eine stabile, versionierte C-API haben? Dies bestimmt ob die macOS-Integration über C-FFI/CGo oder über einen IPC-Mechanismus (Servo als Subprozess, Texturen über Shared Memory) realisiert wird.

### 12.2 wgpu-Instanz-Sharing

Können Servo und Lux dieselbe wgpu-Instanz / denselben Adapter nutzen? Falls ja, ist TextureView-Sharing trivial. Falls nein, muss ein OS-Level Sharing-Mechanismus (IOSurface auf macOS) zwischengeschaltet werden — was den Vorteil von wgpu ↔ wgpu reduziert.

### 12.3 WebView2 CompositionController Reife

Wie stabil ist `ICoreWebView2CompositionController` für die DXGI-Shared-Handle-Extraktion? Microsoft dokumentiert den Pfad, aber Praxisberichte für Custom-Compositor-Integration sind rar.

### 12.4 WPE WebKit Android-Support

WPE arbeitet an Android-Support. Falls dieser reift, könnte der OS-Shim um Android erweitert werden (WPE statt System-WebView).

### 12.5 CEF als Fallback-Backend

Sollte CEF als viertes Backend (`-tags cef`) für Nutzer bereitgehalten werden, die maximale Web-Kompatibilität auf allen Plattformen brauchen — auf Kosten der Binary-Größe?

---

## 13. Quellen

- [Servo Embedding API Rework — Phoronix (2025)](https://www.phoronix.com/news/Servo-Embed-API-2025-Progress)
- [Servo WebView API — Blog (Feb 2025)](https://servo.org/blog/2025/02/19/this-month-in-servo/)
- [Servo November 2025 Update](https://servo.org/blog/2025/12/15/november-in-servo/)
- [WPE WebKit — Igalia Periodical #50 (2025)](https://blogs.igalia.com/webkit/blog/2025/wip-50/)
- [WPE WebKit — Offizielle Seite](https://webkit.org/wpe/)
- [Neomacs — WPE + DMA-buf Zero-Copy Embedding](https://github.com/eval-exec/neomacs)
- [WPE Backend-Architektur](https://wpewebkit.org/blog/07-creating-wpe-backends.html)

---

*RFC-003 — Theoretical. Feedback und Änderungsvorschläge bitte als Issue gegen dieses Dokument.*
