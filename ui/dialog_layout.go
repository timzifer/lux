// Package ui — dialog_layout.go provides the DialogLayout element used by
// the ui/dialog sub-package to render dialog overlays with a colored icon panel.
package ui

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui/icons"
)

// Dialog layout constants.
const (
	dialogPanelWidth = 80 // dp — colored icon panel on the left
	dialogIconSize   = 32 // dp — Phosphor icon size inside the panel
	dialogGap        = 16 // dp — gap between icon panel and content
)

// DialogLayout wraps content in a layout with a colored icon panel on the left.
// Exported so the ui/dialog sub-package can reuse the same visual treatment.
func DialogLayout(kind platform.DialogKind, content Element) Element {
	return dialogLayoutElement{Kind: kind, Content: content}
}

// dialogLayoutElement renders a colored icon panel on the left and content on the right.
type dialogLayoutElement struct {
	Kind    platform.DialogKind
	Content Element
}

func (dialogLayoutElement) isElement() {}

// LayoutSelf implements Layouter.
func (d dialogLayoutElement) LayoutSelf(ctx *LayoutContext) Bounds {
	panelW := dialogPanelWidth
	gap := dialogGap

	// Measure content to determine height.
	contentArea := Bounds{
		X: ctx.Area.X + panelW + gap,
		Y: ctx.Area.Y,
		W: max(ctx.Area.W-panelW-gap, 0),
		H: ctx.Area.H,
	}
	cb := ctx.MeasureChild(d.Content, contentArea)

	contentH := cb.H
	if contentH < 64 {
		contentH = 64
	}

	// Draw the colored icon panel with left-only rounded corners.
	panelColor, iconColor := dialogKindColors(d.Kind, ctx.Tokens)
	radius := ctx.Tokens.Radii.Card
	panelRect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(panelW), float32(contentH))
	ctx.Canvas.FillRoundRectCorners(panelRect, draw.CornerRadii{
		TopLeft:     radius,
		BottomLeft:  radius,
		TopRight:    0,
		BottomRight: 0,
	}, draw.SolidPaint(panelColor))

	// Draw the Phosphor icon centered in the panel.
	iconGlyph := dialogPhosphorIcon(d.Kind)
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       dialogIconSize,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	metrics := ctx.Canvas.MeasureText(iconGlyph, iconStyle)
	cellSize := float32(math.Ceil(float64(dialogIconSize)))
	iconX := float32(ctx.Area.X) + (float32(panelW)-cellSize)/2 + (cellSize-metrics.Width)/2
	iconY := float32(ctx.Area.Y) + (float32(contentH)-metrics.Ascent)/2
	ctx.Canvas.DrawText(iconGlyph, draw.Pt(iconX, iconY), iconStyle, iconColor)

	// Render content to the right of the panel.
	ctx.LayoutChild(d.Content, contentArea)

	totalW := panelW + gap + cb.W
	if totalW > ctx.Area.W {
		totalW = ctx.Area.W
	}
	return Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: totalW, H: contentH, Baseline: contentH}
}

// TreeEqual implements TreeEqualizer.
func (d dialogLayoutElement) TreeEqual(other Element) bool {
	o, ok := other.(dialogLayoutElement)
	return ok && d.Kind == o.Kind
}

// ResolveChildren implements ChildResolver.
func (d dialogLayoutElement) ResolveChildren(resolve func(Element, int) Element) Element {
	if d.Content != nil {
		d.Content = resolve(d.Content, 0)
	}
	return d
}

// dialogPhosphorIcon returns a Phosphor icon codepoint for the given dialog kind.
func dialogPhosphorIcon(kind platform.DialogKind) string {
	switch kind {
	case platform.DialogWarning:
		return icons.Warning
	case platform.DialogError:
		return icons.X
	default:
		return icons.Info
	}
}

// dialogKindColors returns the panel background color and icon color
// for the given dialog kind.
func dialogKindColors(kind platform.DialogKind, tokens theme.TokenSet) (panelBG, icon draw.Color) {
	var base draw.Color
	switch kind {
	case platform.DialogWarning:
		base = tokens.Colors.Status.Warning
	case platform.DialogError:
		base = tokens.Colors.Status.Error
	default:
		base = tokens.Colors.Status.Info
	}
	panelBG = draw.Color{R: base.R, G: base.G, B: base.B, A: 0.15}
	icon = base
	return
}
