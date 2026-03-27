# Drawing API

The `draw.Canvas` interface is the 2D rendering surface passed to every widget's draw call.
Coordinates are in **density-independent pixels (dp)** relative to the widget's top-left corner.

```go
import "github.com/timzifer/lux/draw"
```

---

## Canvas Interface

```go
type Canvas interface {
    // Primitives
    FillRect(r Rect, paint Paint)
    FillRoundRect(r Rect, radius float32, paint Paint)
    FillRoundRectCorners(r Rect, radii CornerRadii, paint Paint)
    FillEllipse(r Rect, paint Paint)

    StrokeRect(r Rect, stroke Stroke)
    StrokeRoundRect(r Rect, radius float32, stroke Stroke)
    StrokeRoundRectCorners(r Rect, radii CornerRadii, stroke Stroke)
    StrokeEllipse(r Rect, stroke Stroke)
    StrokeLine(a, b Point, stroke Stroke)

    // Paths
    FillPath(p Path, paint Paint)
    StrokePath(p Path, stroke Stroke)

    // Text
    DrawText(text string, origin Point, style TextStyle, color Color)
    MeasureText(text string, style TextStyle) TextMetrics
    DrawTextLayout(layout TextLayout, origin Point, color Color)

    // Images & Textures
    DrawImage(img ImageID, dst Rect, opts ImageOptions)
    DrawImageScaled(img ImageID, dst Rect, mode ImageScaleMode, opts ImageOptions)
    DrawImageSlice(slice ImageSlice, dst Rect, opts ImageOptions)
    DrawTexture(tex TextureID, dst Rect)

    // Shadows
    DrawShadow(r Rect, shadow Shadow)

    // Clipping & Transform
    PushClip(r Rect)
    PushClipRoundRect(r Rect, radius float32)
    PushClipPath(p Path)
    PopClip()
    PushTransform(t Transform)
    PopTransform()
    PushOffset(dx, dy float32)
    PushScale(sx, sy float32)

    // Effects
    PushOpacity(alpha float32)
    PopOpacity()
    PushBlur(radius float32)
    PopBlur()
    PushLayer(opts LayerOptions)
    PopLayer()

    // State
    Bounds() Rect
    DPR() float32
    Save()
    Restore()
}
```

---

## Primitives

### Filled shapes

```go
c.FillRect(draw.Rect{X: 0, Y: 0, W: 100, H: 50}, draw.SolidColor(tokens.Colors.Surface.Elevated))
c.FillRoundRect(bounds, tokens.Radii.Card, draw.SolidColor(bg))
c.FillRoundRectCorners(bounds, draw.CornerRadii{TL: 8, TR: 8, BR: 0, BL: 0}, paint)
c.FillEllipse(bounds, draw.SolidColor(accentColor))
```

### Stroked shapes

```go
stroke := draw.Stroke{Color: tokens.Colors.Stroke.Border, Width: 1}
c.StrokeRect(bounds, stroke)
c.StrokeRoundRect(bounds, 6, stroke)
c.StrokeLine(draw.Pt(0, 0), draw.Pt(100, 100), stroke)
```

---

## Paths

Build arbitrary vector shapes with `draw.NewPath()`:

```go
p := draw.NewPath()
p.MoveTo(0, 0)
p.LineTo(100, 0)
p.LineTo(50, 100)
p.Close()

c.FillPath(p, draw.SolidColor(accentColor))
c.StrokePath(p, draw.Stroke{Color: borderColor, Width: 1.5})
```

### Path operations

| Method | Description |
|--------|-------------|
| `MoveTo(x, y)` | Move pen to point (no line) |
| `LineTo(x, y)` | Line from current point |
| `QuadTo(cx, cy, x, y)` | Quadratic bezier |
| `CubicTo(cx1, cy1, cx2, cy2, x, y)` | Cubic bezier |
| `ArcTo(rx, ry, angle, largeArc, sweep, x, y)` | SVG arc |
| `Close()` | Close current sub-path |

---

## Paint

`draw.Paint` is returned by constructor functions:

```go
draw.SolidColor(c Color) Paint
draw.LinearGradient(x0, y0, x1, y1 float32, stops []GradientStop) Paint
draw.RadialGradient(cx, cy, r float32, stops []GradientStop) Paint
```

```go
paint := draw.LinearGradient(0, 0, 0, bounds.H,
    []draw.GradientStop{
        {Offset: 0, Color: draw.Hex("#4c8dff")},
        {Offset: 1, Color: draw.Hex("#2f6fe4")},
    },
)
c.FillRoundRect(bounds, 8, paint)
```

---

## Text

### Simple text

```go
c.DrawText("Hello, world", draw.Pt(0, 20), tokens.Typography.Body, tokens.Colors.Text.Primary)
```

`origin` is the **baseline** of the first line.

### Measuring text

```go
metrics := c.MeasureText("Hello", tokens.Typography.Label)
// metrics.Width, metrics.Height, metrics.Ascent, metrics.Descent
```

### `TextStyle`

```go
style := draw.TextStyle{
    FontFamily: "",       // "" = system default (Noto Sans)
    Size:       13,       // dp
    Weight:     draw.FontWeightRegular,  // 100–900
    LineHeight: 1.5,      // multiplier
    Tracking:   0,        // em spacing
    Raster:     false,    // force bitmap (skip MSDF)
}
```

Font weights: `FontWeightThin` (100) → `FontWeightExtraLight` → `FontWeightLight` →
`FontWeightRegular` (400) → `FontWeightMedium` → `FontWeightSemiBold` → `FontWeightBold` (700)
→ `FontWeightExtraBold` → `FontWeightBlack` (900).

---

## Images

Images are managed through `image.ImageID`. Register images with an `image.Store` and pass the
store to the app via `app.WithImageStore`.

```go
c.DrawImage(imgID, dst, draw.ImageOptions{Opacity: 1.0})
c.DrawImageScaled(imgID, dst, draw.ImageScaleModeFill, opts)
c.DrawImageSlice(draw.ImageSlice{ID: imgID, Center: draw.Rect{...}}, dst, opts)
```

---

## Shadows

```go
c.DrawShadow(bounds, tokens.Elevation.Med)
// draw.Shadow{Color, BlurRadius, OffsetX, OffsetY, Radius (corner radius)}
```

Always draw the shadow before the widget's fill.

---

## Clipping

```go
c.PushClip(bounds)
// ... draw within bounds ...
c.PopClip()

c.PushClipRoundRect(bounds, 8)
// ... draw clipped to rounded rect ...
c.PopClip()
```

Push/Pop calls must be balanced.

---

## Transforms

```go
c.PushOffset(dx, dy)
c.PushScale(sx, sy)
c.PushTransform(draw.Transform{...})
// ... draw ...
c.PopTransform()
```

---

## Effects

### Opacity

```go
c.PushOpacity(0.5) // 50% alpha on everything drawn until Pop
// ...
c.PopOpacity()
```

### Blur

```go
c.PushBlur(12) // Gaussian blur radius in dp
// ...
c.PopBlur()
```

### Layers

`PushLayer` groups drawing commands into an off-screen layer with optional compositing:

```go
c.PushLayer(draw.LayerOptions{
    Opacity:   0.8,
    BlurBelow: 8,   // backdrop blur
    CacheHint: true, // cache the layer between frames (for static content)
})
// ...
c.PopLayer()
```

---

## Utility Types

### `draw.Rect`

```go
r := draw.Rect{X: 10, Y: 20, W: 100, H: 50}
r2 := draw.RectXYWH(10, 20, 100, 50)
r3 := draw.RectFromPoints(draw.Pt(10, 20), draw.Pt(110, 70))
```

### `draw.Point`

```go
p := draw.Pt(10, 20)
```

### `draw.Color`

```go
c := draw.Hex("#4c8dff")
c2 := draw.RGBA(0.3, 0.55, 1.0, 1.0)
c3 := draw.Color{R: 0.3, G: 0.55, B: 1.0, A: 1.0}
```

### `draw.CornerRadii`

```go
radii := draw.CornerRadii{TL: 8, TR: 8, BR: 0, BL: 0} // top rounded, bottom sharp
```

### `draw.Shadow`

```go
shadow := draw.Shadow{
    Color:      draw.Color{R: 0, G: 0, B: 0, A: 0.28},
    BlurRadius: 24,
    OffsetY:    6,
    Radius:     12, // corner radius of the shadow shape
}
```

---

## DPR — Device Pixel Ratio

`c.DPR()` returns the display's device pixel ratio (1.0 on standard displays, 2.0 on HiDPI).
All coordinates passed to Canvas are in dp — the framework handles the conversion to physical
pixels. You need `DPR()` only when working with textures or images at exact physical sizes.
