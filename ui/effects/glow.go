package effects

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── GlowBoxElement ──────────────────────────────────────────────────

// GlowBoxElement renders a soft outer glow around its child using the shadow pipeline.
type GlowBoxElement struct {
	ui.BaseElement
	Color      draw.Color
	BlurRadius float32
	Radius     float32
	Child      ui.Element
}

func (n GlowBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	glowColor := n.Color
	if glowColor.A == 0 {
		glowColor = ctx.Tokens.Colors.Accent.Primary
		glowColor.A = 0.6
	}
	shadow := draw.Shadow{
		Color:      glowColor,
		BlurRadius: n.BlurRadius,
		Radius:     n.Radius,
	}
	ext := shadow.Extent()

	childArea := ui.Bounds{
		X: ctx.Area.X + int(ext.Left),
		Y: ctx.Area.Y + int(ext.Top),
		W: max(ctx.Area.W-int(ext.Left)-int(ext.Right), 0),
		H: max(ctx.Area.H-int(ext.Top)-int(ext.Bottom), 0),
	}
	b := ctx.LayoutChild(n.Child, childArea)
	ctx.Canvas.DrawShadow(draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H)), shadow)
	return ui.Bounds{
		X: b.X - int(ext.Left),
		Y: b.Y - int(ext.Top),
		W: b.W + int(ext.Left) + int(ext.Right),
		H: b.H + int(ext.Top) + int(ext.Bottom),
	}
}

func (n GlowBoxElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(GlowBoxElement)
	return ok && n.Color == o.Color && n.BlurRadius == o.BlurRadius && n.Radius == o.Radius
}

func (n GlowBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.Child = resolve(n.Child, 0)
	return n
}

func (n GlowBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// GlowBox renders a soft outer glow around its child using the shadow pipeline.
func GlowBox(color draw.Color, blurRadius, radius float32, child ui.Element) ui.Element {
	return GlowBoxElement{Color: color, BlurRadius: blurRadius, Radius: radius, Child: child}
}

// Glow is a convenience GlowBox using the theme's accent color.
func Glow(blurRadius, radius float32, child ui.Element) ui.Element {
	return GlowBoxElement{BlurRadius: blurRadius, Radius: radius, Child: child}
}
