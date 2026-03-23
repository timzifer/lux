package display

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Chip layout constants.
const (
	chipPadX     = 12
	chipPadY     = 6
	chipDismissW = 16
)

// ChipElement renders a compact selectable element with arbitrary label content.
type ChipElement struct {
	ui.BaseElement
	Label     ui.Element
	Selected  bool
	OnClick   func()
	OnDismiss func() // if non-nil, shows dismiss "x" button
	Disabled  bool
}

// Chip creates a compact selectable element with arbitrary label content.
func Chip(label ui.Element, selected bool, onClick func()) ui.Element {
	return ChipElement{Label: label, Selected: selected, OnClick: onClick}
}

// ChipDismissible creates a dismissible chip with a "x" button.
func ChipDismissible(label ui.Element, selected bool, onClick, onDismiss func()) ui.Element {
	return ChipElement{Label: label, Selected: selected, OnClick: onClick, OnDismiss: onDismiss}
}

// ChipDisabled creates a disabled chip.
func ChipDisabled(label ui.Element, selected bool) ui.Element {
	return ChipElement{Label: label, Selected: selected, Disabled: true}
}

// LayoutSelf implements ui.Layouter.
func (n ChipElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Measure label.
	cb := ctx.MeasureChild(n.Label, ui.Bounds{X: 0, Y: 0, W: ctx.Area.W, H: ctx.Area.H})

	labelW := cb.W
	dismissW := 0
	if n.OnDismiss != nil {
		dismissW = chipDismissW
	}
	w := labelW + chipPadX*2 + dismissW
	h := cb.H + chipPadY*2

	// Register chip click target and get hover opacity atomically.
	chipClickW := w
	var hoverOpacity float32
	if n.Disabled {
		ctx.IX.RegisterHit(draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(chipClickW), float32(h)), nil)
	} else {
		var chipClickFn func()
		if n.OnClick != nil {
			chipClickFn = n.OnClick
			if n.OnDismiss != nil {
				chipClickW = w - dismissW // exclude dismiss area
			}
		}
		hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(chipClickW), float32(h)), chipClickFn)
	}

	// Background.
	var bgColor, borderColor draw.Color
	if n.Selected {
		// Tonal fill — accent blended over surface, not full accent.
		bgColor = ui.LerpColor(ctx.Tokens.Colors.Surface.Elevated, ctx.Tokens.Colors.Accent.Primary, 0.15)
		borderColor = ui.LerpColor(ctx.Tokens.Colors.Surface.Elevated, ctx.Tokens.Colors.Accent.Primary, 0.30)
	} else {
		bgColor = ctx.Tokens.Colors.Surface.Hovered
		borderColor = ctx.Tokens.Colors.Surface.Pressed
	}
	if hoverOpacity > 0 {
		bgColor = ui.LerpColor(bgColor, ui.HoverHighlight(bgColor), hoverOpacity)
	}

	radius := minf(ctx.Tokens.Radii.Pill, float32(min(w, h))/2)
	ctx.Canvas.FillRoundRect(
		draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h)),
		radius, draw.SolidPaint(borderColor))
	ctx.Canvas.FillRoundRect(
		draw.R(float32(ctx.Area.X+1), float32(ctx.Area.Y+1), float32(max0(w-2)), float32(max0(h-2))),
		maxf(radius-1, 0), draw.SolidPaint(bgColor))

	// Label content.
	labelArea := ui.Bounds{X: ctx.Area.X + chipPadX, Y: ctx.Area.Y + chipPadY, W: labelW, H: cb.H}
	ctx.LayoutChild(n.Label, labelArea)

	// Dismiss "x".
	if n.OnDismiss != nil {
		dismissX := ctx.Area.X + chipPadX + labelW + 4
		dismissY := ctx.Area.Y + chipPadY
		dismissStyle := ctx.Tokens.Typography.LabelSmall
		textColor := ctx.Tokens.Colors.Text.Primary
		if n.Selected {
			textColor = ctx.Tokens.Colors.Accent.Primary
		}
		ctx.Canvas.DrawText("\u00d7", draw.Pt(float32(dismissX), float32(dismissY)), dismissStyle, textColor)

		// Register dismiss hit target.
		ctx.IX.RegisterHit(draw.R(float32(dismissX), float32(ctx.Area.Y), float32(chipDismissW), float32(h)),
			n.OnDismiss)
	}

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
// Returns false because ChipElement has callbacks and child elements.
func (n ChipElement) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n ChipElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n ChipElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Label, parentIdx)
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func max0(a int) int {
	if a > 0 {
		return a
	}
	return 0
}
