// Package ui — dialog.go provides fallback dialog elements rendered as overlays.
//
// These are used when the platform does not implement NativeDialogProvider,
// or when native dialogs fail. Each function returns an Overlay element
// with Backdrop: true and PlacementCenter positioning.
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
	dialogPanelWidth = 80  // dp — colored icon panel on the left
	dialogIconSize   = 32  // dp — Phosphor icon size inside the panel
	dialogGap        = 16  // dp — gap between icon panel and content
	dialogWidth      = 420 // dp — total dialog width (message & confirm)
	dialogInputWidth = 460 // dp — total dialog width (input variant)
)

// MessageDialog returns an overlay element displaying a message with an OK button.
func MessageDialog(id OverlayID, title, message string, kind platform.DialogKind, onClose func()) Element {
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onClose,
		Backdrop:    true,
		FocusTrap:   &FocusTrap{RestoreFocus: true, TrapID: string(id)},
		Content: SizedBox(dialogWidth, 0,
			DialogLayout(kind,
				Column(
					TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					Spacer(12),
					Text(message),
					Spacer(16),
					Row(
						Spacer(0),
						ButtonText("OK", onClose),
					),
				),
			),
		),
	}
}

// ConfirmDialog returns an overlay element with Confirm/Cancel buttons.
func ConfirmDialog(id OverlayID, title, message string, onConfirm, onCancel func()) Element {
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		FocusTrap:   &FocusTrap{RestoreFocus: true, TrapID: string(id)},
		Content: SizedBox(dialogWidth, 0,
			DialogLayout(platform.DialogInfo,
				Column(
					TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					Spacer(12),
					Text(message),
					Spacer(16),
					Row(
						Spacer(0),
						ButtonOutlinedText("Cancel", onCancel),
						Spacer(8),
						ButtonText("Confirm", onConfirm),
					),
				),
			),
		),
	}
}

// InputDialog returns an overlay element with a text field and OK/Cancel buttons.
func InputDialog(id OverlayID, title, message, value, placeholder string, onValueChange func(string), onConfirm, onCancel func()) Element {
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		FocusTrap:   &FocusTrap{RestoreFocus: true, TrapID: string(id)},
		Content: SizedBox(dialogInputWidth, 0,
			DialogLayout(platform.DialogInfo,
				Column(
					TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					Spacer(12),
					Text(message),
					Spacer(12),
					TextField(value, placeholder, WithOnChange(onValueChange)),
					Spacer(16),
					Row(
						Spacer(0),
						ButtonOutlinedText("Cancel", onCancel),
						Spacer(8),
						ButtonText("OK", onConfirm),
					),
				),
			),
		),
	}
}

// DialogLayout wraps content in a layout with a colored icon panel on the left.
// Exported so the ui/dialog sub-package can reuse the same visual treatment.
func DialogLayout(kind platform.DialogKind, content Element) Element {
	return dialogLayoutElement{Kind: kind, Content: content}
}

// dialogLayoutElement renders a colored icon panel on the left and content on the right.
// It measures the content first to determine the panel height (similar to CardElement).
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

// dialogKindColors returns the panel background color (15% alpha) and icon color
// for the given dialog kind, derived from the theme's status colors.
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
