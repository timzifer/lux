# Theming

lux uses a **design token** system. Widgets read semantic color, typography, spacing, and
motion values from a `TokenSet` — they never hard-code colours. This makes it possible to
switch the entire visual language of an application at runtime.

```go
import "github.com/timzifer/lux/theme"
```

---

## Built-in Themes

| Variable | Description |
|----------|-------------|
| `theme.Default` | Alias for `theme.LuxDark` — recommended for new apps |
| `theme.LuxDark` | Lux dark theme (cool-blue dark, `#0f1115` background) |
| `theme.LuxLight` | Lux light theme (`#f5f7fb` background) |
| `theme.LuxAuto` | Follows the OS dark-mode signal (starts dark) |
| `theme.Slate` | Legacy dark theme (Zinc-950 background) — retained for compatibility |
| `theme.SlateLight` | Legacy light theme |

### Selecting a theme at startup

```go
app.Run(model, update, view, app.WithTheme(theme.LuxDark))
```

### Runtime switching

```go
// Toggle between LuxDark and LuxLight
app.Send(app.SetDarkModeMsg{Dark: true})  // → LuxDark
app.Send(app.SetDarkModeMsg{Dark: false}) // → LuxLight

// Switch to a completely different theme
app.Send(app.SetThemeMsg{Theme: myTheme})
```

`SetDarkModeMsg` works with any theme that implements `theme.ThemePair`:

```go
type ThemePair interface {
    DarkVariant() Theme
    LightVariant() Theme
}
```

`LuxDark`, `LuxLight`, and `LuxAuto` implement `ThemePair`. If the active theme does not
implement `ThemePair`, `SetDarkModeMsg` falls back to `LuxDark`/`LuxLight`.

---

## The `Theme` Interface

```go
type Theme interface {
    Tokens() TokenSet
    DrawFunc(kind WidgetKind) DrawFunc
    Parent() Theme
}
```

- `Tokens()` returns the full design token set.
- `DrawFunc(kind)` returns a custom rendering function for a widget kind, or `nil` to use the
  framework default.
- `Parent()` returns a fallback theme for token inheritance.

---

## Token Set

`TokenSet` is a flat struct — all tokens are available directly:

```go
type TokenSet struct {
    Colors     ColorScheme
    Typography TypographyScale
    Spacing    SpacingScale
    Radii      RadiusScale
    Motion     MotionSpec
    Elevation  ElevationScale
    Scroll     ScrollSpec
    Grain      float32  // noise/grain intensity; 0 = off
    HintMode   HintMode // how form hints are shown
}
```

### Colors

```go
tokens.Colors.Surface.Base       // window background
tokens.Colors.Surface.Elevated   // cards, overlays
tokens.Colors.Surface.Hovered    // hover state background
tokens.Colors.Surface.Pressed    // pressed state background
tokens.Colors.Surface.Scrim      // modal backdrop

tokens.Colors.Accent.Primary     // buttons, links, focus ring
tokens.Colors.Accent.PrimaryContrast // text on Primary
tokens.Colors.Accent.Secondary

tokens.Colors.Stroke.Border      // 1dp borders
tokens.Colors.Stroke.Focus       // focus outline
tokens.Colors.Stroke.Divider     // section dividers

tokens.Colors.Text.Primary       // body text
tokens.Colors.Text.Secondary     // labels, metadata
tokens.Colors.Text.Disabled
tokens.Colors.Text.OnAccent      // text on accent background

tokens.Colors.Status.Success / .Warning / .Error / .Info
tokens.Colors.Status.OnSuccess / .OnError

tokens.Colors.Custom["myKey"]    // app-specific custom tokens
```

### Typography

```go
tokens.Typography.H1         // 20dp SemiBold — page title
tokens.Typography.H2         // 16dp SemiBold — section title
tokens.Typography.H3         // 14dp Medium — subtitle
tokens.Typography.Body       // 13dp Regular — standard body
tokens.Typography.BodySmall  // 12dp Regular — metadata
tokens.Typography.Label      // 12dp Medium — button/tab labels
tokens.Typography.LabelSmall // 11dp Medium — badges, chips
tokens.Typography.Code       // 13dp Monospace
tokens.Typography.CodeSmall  // 12dp Monospace
```

All typography values are `draw.TextStyle` structs.

### Spacing

```go
tokens.Spacing.XS  // 4dp
tokens.Spacing.S   // 8dp
tokens.Spacing.M   // 16dp
tokens.Spacing.L   // 24dp
tokens.Spacing.XL  // 32dp
tokens.Spacing.XXL // 48dp
```

### Corner Radii

```go
tokens.Radii.Input  // 4dp — text fields
tokens.Radii.Button // 6dp — buttons
tokens.Radii.Card   // 8–10dp — cards
tokens.Radii.Pill   // 999dp — pill shapes
```

### Motion

```go
tokens.Motion.Standard.Duration   // 220ms
tokens.Motion.Standard.Easing     // anim.OutCubic
tokens.Motion.Emphasized.Duration // 320ms
tokens.Motion.Emphasized.Easing   // anim.InOutCubic
tokens.Motion.Quick.Duration      // 110ms
tokens.Motion.Quick.Easing        // anim.OutExpo
```

### Elevation (shadows)

```go
tokens.Elevation.None  // no shadow
tokens.Elevation.Low   // subtle shadow
tokens.Elevation.Med   // medium shadow (dialogs)
tokens.Elevation.High  // strong shadow (overlays)
```

### Scroll physics

```go
tokens.Scroll.Friction          // kinetic deceleration
tokens.Scroll.Overscroll        // rubber-band distance
tokens.Scroll.StepSize          // scroll per mouse wheel click
tokens.Scroll.MultiplierPrecise // trackpad multiplier
```

---

## Accessing Tokens in a Widget

Inside `Widget.Render`, access the theme via `RenderCtx`:

```go
func (w *MyWidget) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
    tokens := ctx.Theme.Tokens()
    bg := tokens.Colors.Surface.Elevated
    // ...
}
```

---

## Partial Token Override

`theme.Override` creates a derived theme that replaces selected token groups:

```go
myTheme := theme.Override(theme.LuxDark, theme.OverrideSpec{
    Colors: &theme.ColorScheme{
        Accent: theme.AccentColors{
            Primary:         draw.Hex("#e63946"),
            PrimaryContrast: draw.Hex("#ffffff"),
            Secondary:       draw.Hex("#ff6b6b"),
        },
    },
})
```

Only the specified fields of the override spec are replaced; everything else is inherited
from the base theme.

---

## Custom Theme

Implement the `Theme` interface to create a fully custom theme:

```go
type myTheme struct{}

func (t *myTheme) Tokens() theme.TokenSet {
    return theme.TokenSet{
        Colors:     myColors,
        Typography: myTypography,
        Spacing:    theme.SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48},
        Radii:      theme.RadiusScale{Input: 4, Button: 8, Card: 12, Pill: 999},
        Motion: theme.MotionSpec{
            Standard:   theme.DurationEasing{220 * time.Millisecond, anim.OutCubic},
            Emphasized: theme.DurationEasing{320 * time.Millisecond, anim.InOutCubic},
            Quick:      theme.DurationEasing{110 * time.Millisecond, anim.OutExpo},
        },
        // ...
    }
}

func (t *myTheme) DrawFunc(kind theme.WidgetKind) theme.DrawFunc {
    if kind == theme.WidgetKindButton {
        return myButtonDrawFunc
    }
    return nil // use framework default
}

func (t *myTheme) Parent() theme.Theme { return nil }
```

### Custom DrawFunc

Override how a built-in widget is drawn:

```go
func myButtonDrawFunc(ctx theme.DrawCtx, tokens theme.TokenSet, state any) {
    // ctx.Canvas — draw.Canvas
    // ctx.Bounds  — widget bounds in local coordinates
    // ctx.Hovered, ctx.Pressed, ctx.Focused, ctx.Disabled
    bg := tokens.Colors.Accent.Primary
    if ctx.Hovered {
        bg = tokens.Colors.Surface.Hovered
    }
    ctx.Canvas.FillRoundRect(ctx.Bounds, tokens.Radii.Button, draw.SolidColor(bg))
    // draw label, icon, etc.
}
```
