// Package effects provides visual-effect element types (blur, shadow, opacity, glow)
// for the Lux UI framework.
package effects

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── BlurBoxElement ──────────────────────────────────────────────────

// BlurBoxElement applies a Gaussian blur to the region behind its child.
type BlurBoxElement struct {
	ui.BaseElement
	Radius float32
	Child  ui.Element
}

func (n BlurBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Layout child first to determine its actual bounds.
	b := ctx.LayoutChild(n.Child, ctx.Area)
	// Push a tight clip for the child's bounds, then PushBlur captures
	// exactly that region (not the full parent content area).
	ctx.Canvas.PushClip(draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H)))
	ctx.Canvas.PushBlur(n.Radius)
	ctx.Canvas.PopBlur()
	ctx.Canvas.PopClip()
	return b
}

func (n BlurBoxElement) TreeEqual(other ui.Element) bool { return false }

func (n BlurBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n BlurBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// BlurBox applies a Gaussian blur of the given radius to the backdrop behind child.
func BlurBox(radius float32, child ui.Element) ui.Element {
	return BlurBoxElement{Radius: radius, Child: child}
}

// ── FrostedGlassElement ─────────────────────────────────────────────

// FrostedGlassElement renders a frosted-glass effect: backdrop blur + semi-transparent tint overlay.
type FrostedGlassElement struct {
	ui.BaseElement
	BlurRadius float32
	Tint       draw.Color
	Child      ui.Element
}

func (n FrostedGlassElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// 1. Measure child with NullCanvas (no drawing) to get bounds.
	b := ctx.MeasureChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))

	// 2. Register blur region in main scene.
	ctx.Canvas.PushClip(r)
	ctx.Canvas.PushBlur(n.BlurRadius)
	ctx.Canvas.PopBlur()
	ctx.Canvas.PopClip()

	// 3. Draw tint + child in overlay mode (rendered after blur post-processing).
	type overlayModeSetter interface{ SetOverlayMode(bool) }
	if oms, ok := ctx.Canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(true)
		ctx.Canvas.FillRoundRect(r, n.BlurRadius*0.5, draw.SolidPaint(n.Tint))
		ctx.LayoutChild(n.Child, ctx.Area)
		oms.SetOverlayMode(false)
	} else {
		// Fallback: draw in main scene (blur will affect tint+child too).
		ctx.Canvas.FillRoundRect(r, n.BlurRadius*0.5, draw.SolidPaint(n.Tint))
		ctx.LayoutChild(n.Child, ctx.Area)
	}
	return b
}

func (n FrostedGlassElement) TreeEqual(other ui.Element) bool { return false }

func (n FrostedGlassElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n FrostedGlassElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// FrostedGlass renders a frosted-glass effect: backdrop blur + semi-transparent tint overlay.
func FrostedGlass(blurRadius float32, tint draw.Color, child ui.Element) ui.Element {
	return FrostedGlassElement{BlurRadius: blurRadius, Tint: tint, Child: child}
}

// TintedBlur is an alias for FrostedGlass with explicit naming for tinted blur effects.
func TintedBlur(blurRadius float32, tint draw.Color, child ui.Element) ui.Element {
	return FrostedGlassElement{BlurRadius: blurRadius, Tint: tint, Child: child}
}

// ── VibrancyElement ─────────────────────────────────────────────────

// VibrancyElement applies a system-accent-tinted blur to its child's backdrop.
type VibrancyElement struct {
	ui.BaseElement
	TintAlpha float32
	Child     ui.Element
}

func (n VibrancyElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Vibrancy: accent-tinted blur using FrostedGlassElement under the hood.
	tint := ctx.Tokens.Colors.Accent.Primary
	tint.A = n.TintAlpha
	fg := FrostedGlassElement{BlurRadius: 20, Tint: tint, Child: n.Child}
	return ctx.LayoutChild(fg, ctx.Area)
}

func (n VibrancyElement) TreeEqual(other ui.Element) bool { return false }

func (n VibrancyElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n VibrancyElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// Vibrancy applies a system-accent-tinted blur to its child's backdrop.
// tintAlpha controls the opacity of the accent tint overlay (0.0–1.0).
func Vibrancy(tintAlpha float32, child ui.Element) ui.Element {
	return VibrancyElement{TintAlpha: tintAlpha, Child: child}
}
