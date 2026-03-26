# Layout

lux implements three major layout algorithms and several convenience containers.

```go
import "github.com/timzifer/lux/ui/layout"
```

---

## Row and Column

The fastest way to lay out children in a line:

```go
layout.Row(child1, child2, child3)    // left → right
layout.Column(child1, child2, child3) // top → bottom
```

Both are thin wrappers over `layout.Flex` with sensible defaults.

---

## Flexbox — `layout.Flex`

A CSS-compatible flexbox implementation.

```go
layout.Flex(layout.FlexOpts{
    Direction:  layout.Row,          // Row | Column | RowReverse | ColumnReverse
    Justify:    layout.JustifyStart, // Start | End | Center | SpaceBetween | SpaceAround | SpaceEvenly
    Align:      layout.AlignStart,   // Start | End | Center | Stretch | Baseline
    Wrap:       layout.NoWrap,       // NoWrap | Wrap | WrapReverse
    Gap:        8,                   // gap between items (dp)
    RowGap:     0,                   // row gap (overrides Gap for rows)
    ColumnGap:  0,                   // column gap (overrides Gap for columns)
    Padding:    layout.Insets{Top: 16, Right: 16, Bottom: 16, Left: 16},
},
    child1,
    layout.FlexItem(child2, layout.FlexItemOpts{Grow: 1}),
    child3,
)
```

### FlexItem options

Wrap a child in `layout.FlexItem` to control its individual flex behaviour:

```go
layout.FlexItem(child, layout.FlexItemOpts{
    Grow:      1,      // flex-grow
    Shrink:    1,      // flex-shrink
    Basis:     100,    // flex-basis (dp); 0 = content size
    AlignSelf: layout.AlignCenter,
})
```

---

## CSS Grid — `layout.Grid`

A CSS Grid implementation supporting explicit tracks, `fr` units, `repeat()`, and
auto-placement.

```go
layout.Grid(layout.GridOpts{
    Columns: []layout.TrackSize{
        layout.TrackFixed(200),    // 200dp
        layout.TrackFR(1),         // 1fr (fills remaining space)
        layout.TrackFR(2),         // 2fr
    },
    Rows: []layout.TrackSize{
        layout.TrackAuto(),        // sized to content
        layout.TrackFixed(48),
    },
    Gap:       8,
    RowGap:    8,
    ColumnGap: 16,
    Padding:   layout.Insets{All: 16},
},
    child1,
    layout.GridItem(child2, layout.GridItemOpts{
        ColumnStart: 1, ColumnEnd: 3, // span 2 columns
        RowStart:    1, RowEnd:    2,
    }),
    child3,
)
```

### Track sizes

| Constructor | CSS equivalent | Description |
|-------------|----------------|-------------|
| `TrackFixed(dp)` | `200px` | Fixed size in dp |
| `TrackFR(n)` | `1fr` | Fractional remaining space |
| `TrackAuto()` | `auto` | Sized to content |
| `TrackMinMax(min, max)` | `minmax(min, max)` | Clamped track |

---

## Table — `layout.Table`

Implements the HTML table layout algorithm (fixed + auto sizing).

```go
layout.Table([]layout.TableRow{
    {Cells: []layout.TableCell{
        {Child: header1, Header: true},
        {Child: header2, Header: true},
    }},
    {Cells: []layout.TableCell{
        {Child: cell1},
        {Child: cell2, ColSpan: 1, RowSpan: 1},
    }},
})
```

---

## Stack — `layout.Stack`

Stacks children on the z-axis (all positioned at the same origin, later children render on
top):

```go
layout.Stack(
    backgroundWidget,
    foregroundWidget, // rendered on top
)
```

Stack is useful for overlays, badges on icons, and layered compositions.

---

## Box — `layout.Box`

Wraps a single child with padding, margin, size constraints, and alignment:

```go
layout.Box(layout.BoxOpts{
    Padding:    layout.Insets{All: 16},
    Margin:     layout.Insets{Top: 8},
    MinWidth:   120,
    MaxWidth:   400,
    MinHeight:  0,
    MaxHeight:  0,   // 0 = unconstrained
    Align:      layout.AlignCenter,
}, child)
```

---

## Insets

`layout.Insets` specifies edge values. The `All` shorthand sets all four sides:

```go
layout.Insets{All: 16}                           // 16dp on all sides
layout.Insets{Top: 8, Right: 16, Bottom: 8, Left: 16}
layout.Insets{Horizontal: 16, Vertical: 8}
```

---

## RTL Support

lux propagates the text direction from `RenderCtx.Locale` through the layout tree.
All layout algorithms treat `Start`/`End` as logical (direction-aware) rather than
physical `Left`/`Right`. Physical insets (`Left`, `Right`) always refer to the physical
screen direction; logical insets (`Start`, `End`) flip for RTL locales.

Set the locale at startup:

```go
app.Run(model, update, view, app.WithLocale("ar")) // Arabic → RTL
```

---

## Custom Layout

Implement `ui.Layout` for fully custom layout algorithms:

```go
type Layout interface {
    Layout(ctx LayoutCtx, children []LayoutChild) Size
}
```

`LayoutCtx` provides constraints (min/max size, DPR, direction). `LayoutChild.Measure` and
`LayoutChild.Place` are called to measure and position each child.
