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
	b := ctx.LayoutChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
	glowColor := n.Color
	if glowColor.A == 0 {
		glowColor = ctx.Tokens.Colors.Accent.Primary
		glowColor.A = 0.6
	}
	ctx.Canvas.DrawShadow(r, draw.Shadow{
		Color:      glowColor,
		BlurRadius: n.BlurRadius,
		Radius:     n.Radius,
	})
	return b
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
