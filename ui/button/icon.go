package button

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Icon is a compact icon-only button element.
type Icon struct {
	ui.BaseElement
	IconName string
	OnClick  func()
	Variant  ui.ButtonVariant
	Size     float32 // 0 = default
}

// NewIcon creates a filled icon button.
func NewIcon(icon string, onClick func()) ui.Element {
	return Icon{IconName: icon, OnClick: onClick, Variant: ui.ButtonFilled}
}

// IconButton is an alias for NewIcon.
func IconButton(icon string, onClick func()) ui.Element { return NewIcon(icon, onClick) }

// IconVariant creates an icon button with a specific variant.
func IconVariant(variant ui.ButtonVariant, icon string, onClick func()) ui.Element {
	return Icon{IconName: icon, OnClick: onClick, Variant: variant}
}

// IconButtonVariant is an alias for IconVariant.
func IconButtonVariant(variant ui.ButtonVariant, icon string, onClick func()) ui.Element {
	return IconVariant(variant, icon, onClick)
}

// LayoutSelf implements ui.Layouter.
func (n Icon) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	ix := ctx.IX

	size := n.Size
	if size == 0 {
		size = tokens.Typography.Label.Size * 2
	}
	cellSize := int(math.Ceil(float64(size)))
	w := cellSize + ui.IconButtonPad*2
	h := w // square

	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	hoverOpacity := ix.RegisterHit(buttonRect, n.OnClick)

	fillColor, borderColor, iconColor := ui.ButtonVariantColors(n.Variant, tokens, hoverOpacity)

	if n.Variant == ui.ButtonFilled {
		canvas.FillRoundRect(buttonRect,
			tokens.Radii.Button, draw.SolidPaint(borderColor))
		innerRadius := tokens.Radii.Button - float32(ui.ButtonBorder)
		if innerRadius < 0 {
			innerRadius = 0
		}
		canvas.FillRoundRect(draw.R(float32(area.X+ui.ButtonBorder), float32(area.Y+ui.ButtonBorder),
			float32(max(w-ui.ButtonBorder*2, 0)), float32(max(h-ui.ButtonBorder*2, 0))),
			innerRadius, draw.SolidPaint(fillColor))
	} else {
		if fillColor.A > 0 {
			canvas.FillRoundRect(buttonRect, tokens.Radii.Button, draw.SolidPaint(fillColor))
		}
		if borderColor.A > 0 {
			canvas.StrokeRoundRect(buttonRect, tokens.Radii.Button, draw.Stroke{
				Paint: draw.SolidPaint(borderColor),
				Width: float32(ui.ButtonBorder),
			})
		}
	}

	// Render icon centered.
	style := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       size,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	metrics := canvas.MeasureText(n.IconName, style)
	offsetX := (float32(w) - metrics.Width) / 2
	offsetY := (float32(h) - metrics.Ascent) / 2
	canvas.DrawText(n.IconName, draw.Pt(float32(area.X)+offsetX, float32(area.Y)+offsetY), style, iconColor)

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Icon) TreeEqual(other ui.Element) bool {
	_, ok := other.(Icon)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver. Icon buttons are leaves.
func (n Icon) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}
