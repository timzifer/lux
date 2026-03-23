package effects

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── ShadowBoxElement ────────────────────────────────────────────────

// ShadowBoxElement draws a soft shadow behind its child element.
type ShadowBoxElement struct {
	ui.BaseElement
	Shadow draw.Shadow
	Radius float32
	Child  ui.Element
}

func (n ShadowBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Draw shadow first (behind content), then layout child on top.
	b := ctx.LayoutChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
	shadow := n.Shadow
	if n.Radius > 0 {
		shadow.Radius = n.Radius
	}
	ctx.Canvas.DrawShadow(r, shadow)
	return b
}

func (n ShadowBoxElement) TreeEqual(other ui.Element) bool { return false }

func (n ShadowBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n ShadowBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// ShadowBox draws a soft shadow behind a child element.
func ShadowBox(shadow draw.Shadow, radius float32, child ui.Element) ui.Element {
	return ShadowBoxElement{Shadow: shadow, Radius: radius, Child: child}
}

// ── InnerShadowBoxElement ───────────────────────────────────────────

// InnerShadowBoxElement draws an inner shadow on top of its child content.
type InnerShadowBoxElement struct {
	ui.BaseElement
	Shadow draw.Shadow
	Radius float32
	Child  ui.Element
}

func (n InnerShadowBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Layout child first, then draw inner shadow ON TOP of child content.
	b := ctx.LayoutChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
	shadow := n.Shadow
	shadow.Inset = true
	if n.Radius > 0 {
		shadow.Radius = n.Radius
	}
	type overlayModeSetter interface{ SetOverlayMode(bool) }
	if oms, ok := ctx.Canvas.(overlayModeSetter); ok {
		oms.SetOverlayMode(true)
		ctx.Canvas.DrawShadow(r, shadow)
		oms.SetOverlayMode(false)
	} else {
		ctx.Canvas.DrawShadow(r, shadow)
	}
	return b
}

func (n InnerShadowBoxElement) TreeEqual(other ui.Element) bool { return false }

func (n InnerShadowBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n InnerShadowBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// InnerShadowBox draws an inner shadow on top of its child content.
func InnerShadowBox(shadow draw.Shadow, radius float32, child ui.Element) ui.Element {
	shadow.Inset = true
	return InnerShadowBoxElement{Shadow: shadow, Radius: radius, Child: child}
}

// ── ElevationBoxElement ─────────────────────────────────────────────

// ElevationBoxElement renders a hover-responsive shadow behind its child.
type ElevationBoxElement struct {
	ui.BaseElement
	Rest    draw.Shadow
	Hover   draw.Shadow
	Press   draw.Shadow
	Radius  float32
	OnClick func()
	Child   ui.Element
}

func (n ElevationBoxElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Layout child, register hover, interpolate shadow.
	b := ctx.LayoutChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
	hoverOpacity := ctx.IX.RegisterHit(r, n.OnClick)
	shadow := draw.LerpShadow(n.Rest, n.Hover, hoverOpacity)
	if n.Radius > 0 {
		shadow.Radius = n.Radius
	}
	ctx.Canvas.DrawShadow(r, shadow)
	return b
}

func (n ElevationBoxElement) TreeEqual(other ui.Element) bool { return false }

func (n ElevationBoxElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n ElevationBoxElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// ElevationBox renders a hover-responsive shadow behind its child.
func ElevationBox(rest, hover, press draw.Shadow, radius float32, onClick func(), child ui.Element) ui.Element {
	return ElevationBoxElement{Rest: rest, Hover: hover, Press: press, Radius: radius, OnClick: onClick, Child: child}
}

// ── ElevationCardElement ────────────────────────────────────────────

// ElevationCardElement is a convenience wrapper using theme elevation presets.
type ElevationCardElement struct {
	ui.BaseElement
	OnClick func()
	Child   ui.Element
}

func (n ElevationCardElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Convenience: uses theme elevation presets (Low → High → None).
	b := ctx.LayoutChild(n.Child, ctx.Area)
	r := draw.R(float32(b.X), float32(b.Y), float32(b.W), float32(b.H))
	hoverOpacity := ctx.IX.RegisterHit(r, n.OnClick)
	shadow := draw.LerpShadow(ctx.Tokens.Elevation.Low, ctx.Tokens.Elevation.High, hoverOpacity)
	ctx.Canvas.DrawShadow(r, shadow)
	return b
}

func (n ElevationCardElement) TreeEqual(other ui.Element) bool { return false }

func (n ElevationCardElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

func (n ElevationCardElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// ElevationCard is a convenience wrapper around ElevationBox using theme elevation presets.
func ElevationCard(onClick func(), child ui.Element) ui.Element {
	return ElevationCardElement{OnClick: onClick, Child: child}
}
