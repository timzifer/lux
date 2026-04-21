# Evaluation: Scoped Theme Overrides via `Themed` Element

**Status:** Entwurf / Draft
**Datum:** 2026-03-19

---

## 1. Fragestellung

> Ein Pseudo-Element, das ein eigenes Theme entgegennehmen kann und dieses an
> die Unter-Elemente vererbt — z.B. um speziell eingefärbte Buttons anzuzeigen.

## 2. Ist-Zustand

Das Theme wird heute **global** im App-Loop gehalten (`activeTheme` in
`app/run.go`) und über den Reconciler als **ein einziges** `theme.Theme` an
**alle** Widgets weitergereicht:

```
app.Run()
  → CachedTheme(cfg.theme)
    → Reconciler.Reconcile(tree, activeTheme, …)
      → resolveTree(…, th, …)        // th wird unverändert durchgereicht
        → RenderCtx{Theme: th}       // jedes Widget bekommt dasselbe Theme
```

Ein Widget kann sein Theme nur lesen (`ctx.Theme.Tokens()`), aber nicht für
seine Kinder überschreiben. `theme.Override()` existiert bereits, wird aber
nur global via `SetThemeMsg` eingesetzt.

## 3. Bewertung der Optionen

### Option A: `Themed`-Element (Empfehlung ✓)

Ein neues **Container-Element**, das ein `theme.Theme` trägt und dieses an den
gesamten Subtree vererbt:

```go
// API-Entwurf
ui.Themed(dangerTheme,
    ui.Button("Löschen", func() { … }),
    ui.Button("Abbrechen", func() { … }),
)
```

**Implementierung (Aufwand: ~40 Zeilen):**

1. **Element-Typ** in `ui/element.go`:
   ```go
   type themedElement struct {
       Theme    theme.Theme
       Children []Element
   }

   func Themed(th theme.Theme, children ...Element) Element {
       return themedElement{Theme: th, Children: children}
   }
   ```

2. **Reconciler-Case** in `ui/reconcile.go` (`resolveTree`):
   ```go
   case themedElement:
       // Theme für den Subtree ersetzen
       sub := theme.NewCachedTheme(node.Theme)
       children := make([]Element, len(node.Children))
       for i, c := range node.Children {
           children[i] = r.resolveTree(c, parentUID, i, seen, sub, send, dispatcher, fm)
       }
       return boxElement{Axis: AxisColumn, Children: children}
   ```

3. **Layout** — kein neuer Layout-Case nötig, da `themedElement` nach der
   Reconciliation zu einem normalen `boxElement` aufgelöst wird.

4. **treeEqual** — Case für `themedElement` ergänzen (Theme-Pointer-Vergleich
   + Kinder-Vergleich).

**Vorteile:**
- Minimalinvasiv: ~40 Zeilen Änderung, kein neues Konzept
- Konsistent mit dem bestehenden Muster (wie `paddingElement`, `keyedElement`)
- Komponierbar: verschachtelbar, kombinierbar mit `Override()`
- Kein globaler Seiteneffekt
- CachedTheme-Wrapping sorgt für performantes Lookup im Subtree

**Nachteile:**
- Minimaler Overhead: ein zusätzlicher `CachedTheme`-Wrapper pro `Themed`-Scope
- Theme-Wechsel im Subtree erzwingt Re-Render aller Kinder (aber: das ist
  gewünscht und passiert auch bei globalem Theme-Wechsel)

---

### Option B: Per-Button Style-Props

```go
ui.Button("Löschen", fn, ui.ButtonColor(dangerAccent))
```

**Bewertung:** Löst nur den Button-Fall, nicht den generischen. Jedes Widget
bräuchte eigene Color-Props. Führt zu API-Bloat und ist inkompatibel mit dem
Token-basierten Ansatz (Farben sollen aus dem Theme kommen, nicht hart kodiert
werden).

**Fazit:** Nicht empfohlen als Primärmechanismus.

---

### Option C: CSS-Variablen-artiger Custom-Token-Layer

```go
ui.WithTokens(map[string]any{
    "accent.primary": dangerRed,
}, children...)
```

**Bewertung:** Verliert Typsicherheit. Erfordert String-basierte Token-Lookup-
Infrastruktur, die dem aktuellen Struct-basierten `TokenSet` widerspricht.
Deutlich komplexer und fehleranfälliger als Option A.

**Fazit:** Over-Engineering für den Use-Case.

---

### Option D: Kontext-basierter Ansatz (à la React Context)

Ein separater Key-Value-Kontext, der neben dem Theme propagiert wird.

**Bewertung:** Lux hat bewusst kein Context-System (nur `RenderCtx`). Ein
vollständiges Context-System einzuführen wäre ein architektonischer
Paradigmenwechsel. Das Theme **ist** bereits der Kontext — es muss nur
scopebar gemacht werden.

**Fazit:** Kanone auf Spatzen.

---

## 4. Empfehlung

**Option A: `Themed`-Element** ist der klare Favorit.

Es nutzt die vorhandene Infrastruktur (`theme.Override`, `CachedTheme`,
`resolveTree`-Propagation) und erfordert minimale Änderungen:

| Datei | Änderung |
|---|---|
| `ui/element.go` | +15 Zeilen: `themedElement` Typ + `Themed()` Konstruktor |
| `ui/reconcile.go` | +10 Zeilen: `case themedElement` in `resolveTree` |
| `ui/reconcile.go` | +5 Zeilen: `case themedElement` in `treeEqual` |
| `theme/cache.go` | ggf. `NewCachedTheme` exportieren (falls noch nicht public) |

**Typisches Nutzungsmuster:**

```go
// Danger-Zone mit rotem Accent
danger := theme.Override(theme.Default, theme.OverrideSpec{
    Colors: &theme.ColorScheme{
        Accent: theme.AccentColors{
            Primary:         draw.Hex("#dc2626"),
            PrimaryContrast: draw.Hex("#ffffff"),
        },
    },
})

func (m Model) View() ui.Element {
    return ui.Column(
        ui.Button("Normal", normalAction),

        // Lokal überschriebenes Theme
        ui.Themed(danger,
            ui.Button("Löschen", deleteAction),   // rot
            ui.Button("Alles zurücksetzen", resetAction), // auch rot
        ),

        ui.Button("Auch normal", otherAction),
    )
}
```

## 5. Offene Fragen

1. **Axis-Default:** Soll `Themed` sich wie `Column` (vertikal) oder wie ein
   transparenter Wrapper (kein eigenes Layout) verhalten? Empfehlung: wie
   `Column` mit optionalem Axis-Parameter, oder: nur ein einziges Kind
   akzeptieren (wie `Padding`).

2. **CachedTheme-Sharing:** Wenn dasselbe Override-Theme in mehreren Frames
   verwendet wird, sollte der Nutzer das `CachedTheme` außerhalb cachen, oder
   soll der Reconciler das automatisch tun (z.B. per Theme-Pointer-Map)?

3. **DrawFunc-Vererbung:** `overrideTheme.DrawFunc()` delegiert heute an
   `base.DrawFunc()`. Soll `Themed` auch eigene DrawFuncs pro Scope erlauben?
   (Vermutlich ja — ist bereits durch `theme.Theme`-Interface abgedeckt.)
