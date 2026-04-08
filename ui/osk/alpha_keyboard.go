package osk

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// keyVariant maps an OSKAction to a ButtonVariant for consistent styling.
func keyVariant(action OSKAction) ui.ButtonVariant {
	switch action {
	case OSKActionShift, OSKActionSwitch, OSKActionBackspace, OSKActionDismiss:
		return ui.ButtonOutlined
	case OSKActionEnter, OSKActionTab:
		return ui.ButtonFilled
	case OSKActionSpace:
		return ui.ButtonGhost
	default:
		return ui.ButtonTonal
	}
}

// RenderButtonKeyboard draws the OSK keyboard using ButtonVariantColors for
// consistent styling with the button component. It replaces the custom key
// rendering in both the inline OSKElement and the ActionSheet.
func RenderButtonKeyboard(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor,
	state *OSKState, screenW, screenH int, dpr float32, oskX, oskY, oskW float32) {

	if state == nil || !state.Visible {
		return
	}

	_, keyH, gap := ComputeKeySize(screenW, screenH, dpr, state.Mode)
	rows := RowsForState(state)

	keyStyle := tokens.Typography.Body
	keyStyle.Size = keyH * 0.4
	if keyStyle.Size < 12 {
		keyStyle.Size = 12
	}
	if keyStyle.Size > 22 {
		keyStyle.Size = 22
	}

	radius := tokens.Radii.Button
	if radius < 4 {
		radius = 4
	}

	for rowIdx, row := range rows {
		var totalRelW float32
		for _, k := range row {
			totalRelW += k.Width
		}
		if totalRelW == 0 {
			continue
		}

		availRowW := oskW - gap*2
		unit := (availRowW - gap*float32(len(row)-1)) / totalRelW
		rowW := totalRelW*unit + gap*float32(len(row)-1)
		startX := oskX + (oskW-rowW)/2

		y := oskY + gap + float32(rowIdx)*(keyH+gap)
		x := startX

		for _, key := range row {
			kw := key.Width * unit
			if kw < 1 {
				x += kw + gap
				continue
			}

			keyRect := draw.R(x, y, kw, keyH)

			// Register hit target and get hover opacity.
			hoverOpacity := ix.RegisterHit(keyRect, keyAction(key, state))

			// Get button-variant colors.
			variant := keyVariant(key.Action)
			fill, border, labelColor := ui.ButtonVariantColors(variant, tokens, hoverOpacity)

			// Draw key background.
			if fill.A > 0 {
				canvas.FillRoundRect(keyRect, radius, draw.SolidPaint(fill))
			}
			if border.A > 0 {
				canvas.StrokeRoundRect(keyRect, radius, draw.Stroke{
					Paint: draw.SolidPaint(border),
					Width: 1,
				})
			}

			// Draw key label.
			if key.Label != "" {
				m := canvas.MeasureText(key.Label, keyStyle)
				tx := x + (kw-m.Width)/2
				ty := y + (keyH-keyStyle.Size)/2
				canvas.DrawText(key.Label, draw.Pt(tx, ty), keyStyle, labelColor)
			}

			x += kw + gap
		}
	}
}
