package display

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// IconElement renders a text symbol (typically from an icon font) at a fixed size.
type IconElement struct {
	ui.BaseElement
	Name string
	Size float32 // 0 = use theme Label size × 2
}

// Icon renders a text symbol at the theme's label size.
func Icon(name string) ui.Element { return IconElement{Name: name, Size: 0} }

// IconSize renders a text symbol at a specific size in dp.
func IconSize(name string, size float32) ui.Element { return IconElement{Name: name, Size: size} }

// LayoutSelf implements ui.Layouter.
func (n IconElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	size := n.Size
	if size == 0 {
		size = ctx.Tokens.Typography.Label.Size * 2
	}
	// Use the Phosphor icon font for icon elements.
	// Render into a fixed square cell so all icons have uniform size
	// regardless of individual glyph bounding boxes.
	style := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       size,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	cellSize := int(math.Ceil(float64(size)))
	metrics := ctx.Canvas.MeasureText(n.Name, style)
	offsetX := (float32(cellSize) - metrics.Width) / 2
	offsetY := (float32(cellSize) - metrics.Ascent) / 2
	ctx.Canvas.DrawText(n.Name, draw.Pt(float32(ctx.Area.X)+offsetX, float32(ctx.Area.Y)+offsetY), style, ctx.Tokens.Colors.Text.Primary)
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: cellSize, H: cellSize, Baseline: cellSize}
}

// TreeEqual implements ui.TreeEqualizer.
func (n IconElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(IconElement)
	return ok && n.Name == nb.Name && n.Size == nb.Size
}

// ResolveChildren implements ui.ChildResolver. IconElement is a leaf.
func (n IconElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op for icons.
func (n IconElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
