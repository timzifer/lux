// Package osk — actionsheet.go provides the keyboard ActionSheet overlay.
// When OSKPresentation is ActionSheet, the framework renders this instead of
// the inline OSK. It combines an interactive InputProxy (top) with the
// on-screen keyboard (bottom) inside a scrim-backed modal sheet.
package osk

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/interaction"
	"github.com/timzifer/lux/ui"
)

// ActionSheet layout constants.
const (
	// askbMaxHeightFrac: the sheet occupies at most 85% of window height.
	askbMaxHeightFrac = 0.85
	// askbCornerRadius: rounded corners for the sheet.
	askbCornerRadius float32 = 16
	// askbHandleW/H: drag-handle pill dimensions.
	askbHandleW float32 = 36
	askbHandleH float32 = 4
	// askbHandleMarginY: spacing above and below the handle.
	askbHandleMarginY float32 = 8
	// askbPadY: vertical padding inside the sheet.
	askbPadY float32 = 8
	// askbInputGap: gap between input proxy and keyboard.
	askbInputGap float32 = 12
	// askbBottomMargin: spacing from bottom window edge.
	askbBottomMargin float32 = 0
)

// KeyboardActionSheetElement renders the keyboard ActionSheet overlay.
// It contains an InputProxy at the top and the OSK keyboard at the bottom,
// backed by a scrim that dismisses the keyboard on tap.
type KeyboardActionSheetElement struct {
	ui.BaseElement
	State   *OSKState
	ScreenW int
	ScreenH int
	Input   *ui.InputState
	Profile *interaction.InteractionProfile
}

// NewKeyboardActionSheet creates a KeyboardActionSheetElement.
func NewKeyboardActionSheet(state *OSKState, screenW, screenH int, input *ui.InputState, profile *interaction.InteractionProfile) ui.Element {
	if state == nil || !state.Visible {
		return ui.Empty()
	}
	return KeyboardActionSheetElement{
		State:   state,
		ScreenW: screenW,
		ScreenH: screenH,
		Input:   input,
		Profile: profile,
	}
}

// LayoutSelf implements ui.Layouter.
func (el KeyboardActionSheetElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	state := el.State
	if state == nil || !state.Visible {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	th := ctx.Theme

	dpr := canvas.DPR()
	_, keyH, gap := ComputeKeySize(el.ScreenW, el.ScreenH, dpr, state.Mode)
	rows := RowsForState(state)
	oskH := float32(len(rows))*(keyH+gap) + gap

	// Input proxy height.
	bodyStyle := tokens.Typography.Body
	inputH := bodyStyle.Size + inputProxyPadY*2

	// Handle area.
	handleAreaH := askbHandleMarginY*2 + askbHandleH

	// Total sheet height.
	contentH := handleAreaH + inputH + askbInputGap + oskH + askbPadY
	maxH := float32(el.ScreenH) * askbMaxHeightFrac
	if contentH > maxH {
		contentH = maxH
	}

	sheetW := float32(el.ScreenW)
	sheetX := float32(0)
	sheetY := float32(el.ScreenH) - contentH - askbBottomMargin

	// ── 1. Scrim backdrop ──────────────────────────────────────────
	scrimRect := draw.R(0, 0, float32(el.ScreenW), float32(el.ScreenH))
	canvas.FillRect(scrimRect, draw.SolidPaint(tokens.Colors.Surface.Scrim))
	if ix != nil {
		ix.RegisterHit(scrimRect, func() {
			send(OSKDismissMsg{})
		})
	}

	// ── 2. Sheet background ────────────────────────────────────────
	sheetRect := draw.R(sheetX, sheetY, sheetW, contentH)
	if ix != nil {
		// Eat clicks on the sheet body so they don't dismiss via scrim.
		ix.RegisterHit(sheetRect, func() {})
	}
	canvas.FillRoundRect(sheetRect, askbCornerRadius, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(sheetRect, askbCornerRadius, draw.Stroke{
		Paint: draw.SolidPaint(tokens.Colors.Stroke.Border),
		Width: 1,
	})

	// ── 3. Drag handle pill ────────────────────────────────────────
	handleX := sheetX + (sheetW-askbHandleW)/2
	handleY := sheetY + askbHandleMarginY
	canvas.FillRoundRect(
		draw.R(handleX, handleY, askbHandleW, askbHandleH),
		askbHandleH/2,
		draw.SolidPaint(tokens.Colors.Stroke.Border),
	)

	// ── 4. Input proxy ─────────────────────────────────────────────
	proxyY := sheetY + handleAreaH
	proxyArea := ui.Bounds{
		X: int(sheetX + inputProxyPadX),
		Y: int(proxyY),
		W: int(sheetW - inputProxyPadX*2),
		H: int(inputH),
	}
	proxy := NewInputProxy(el.Input, el.Profile)
	ctx.LayoutChild(proxy, proxyArea)

	// ── 5. OSK keyboard (button-based rendering) ─────────────────
	oskY := proxyY + inputH + askbInputGap
	_ = th // reserved for custom DrawFunc dispatch
	RenderButtonKeyboard(canvas, tokens, ix, state, el.ScreenW, el.ScreenH, dpr, sheetX, oskY, sheetW)

	return ui.Bounds{X: int(sheetX), Y: int(sheetY), W: int(sheetW), H: int(contentH)}
}

// TreeEqual implements ui.TreeEqualizer.
func (el KeyboardActionSheetElement) TreeEqual(other ui.Element) bool {
	o, ok := other.(KeyboardActionSheetElement)
	if !ok {
		return false
	}
	if el.State == nil && o.State == nil {
		return true
	}
	if el.State == nil || o.State == nil {
		return false
	}
	return el.State.Visible == o.State.Visible &&
		el.State.Layout == o.State.Layout &&
		el.State.Mode == o.State.Mode &&
		el.State.Shifted == o.State.Shifted &&
		el.ScreenW == o.ScreenW &&
		el.ScreenH == o.ScreenH
}

// ResolveChildren implements ui.ChildResolver (no children).
func (el KeyboardActionSheetElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return el
}
