// Package osk — input_proxy.go provides an interactive proxy for the focused
// text field inside the keyboard ActionSheet. It reads from the shared
// InputState pointer (owned by FocusManager) and renders the current value
// with cursor and selection, allowing touch-based cursor repositioning.
package osk

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/interaction"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/ui"
)

// InputProxy layout constants.
const (
	inputProxyPadX float32 = 12
	inputProxyPadY float32 = 10
)

// InputProxyElement renders the current InputState as an interactive text field
// proxy inside the keyboard ActionSheet. It shares the same *InputState as
// the real text field, so any changes made via the OSK are immediately visible.
type InputProxyElement struct {
	ui.BaseElement
	Input   *ui.InputState
	Profile *interaction.InteractionProfile
}

// NewInputProxy creates an InputProxyElement.
func NewInputProxy(input *ui.InputState, profile *interaction.InteractionProfile) ui.Element {
	if input == nil {
		return ui.Empty()
	}
	return InputProxyElement{Input: input, Profile: profile}
}

// LayoutSelf implements ui.Layouter.
func (el InputProxyElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if el.Input == nil {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	area := ctx.Area

	style := tokens.Typography.Body
	h := int(style.Size + inputProxyPadY*2)
	w := area.W

	// Background.
	rect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	canvas.FillRoundRect(rect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Stroke.Border))
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0),
		draw.SolidPaint(tokens.Colors.Surface.Elevated),
	)

	// Focus ring (always shown since this proxy is "always focused").
	ui.DrawFocusRing(canvas, rect, tokens.Radii.Input, tokens)

	// Clip text content.
	textX := float32(area.X) + inputProxyPadX
	textY := float32(area.Y) + inputProxyPadY
	clipRect := draw.R(float32(area.X)+inputProxyPadX, float32(area.Y), float32(w)-inputProxyPadX*2, float32(h))
	canvas.PushClip(clipRect)

	// Draw text value.
	value := el.Input.Value
	textColor := tokens.Colors.Text.Primary
	if value != "" {
		canvas.DrawText(value, draw.Pt(textX, textY), style, textColor)
	}

	// Selection highlight.
	if el.Input.HasSelection() {
		selA, selB := el.Input.SelectionRange()
		selA = clampOff(selA, len(value))
		selB = clampOff(selB, len(value))
		mA := canvas.MeasureText(value[:selA], style)
		mB := canvas.MeasureText(value[:selB], style)
		selX := textX + mA.Width
		selW := mB.Width - mA.Width
		selColor := tokens.Colors.Accent.Primary
		selColor.A = 0.3
		canvas.FillRect(draw.R(selX, textY, selW, style.Size), draw.SolidPaint(selColor))
	}

	// Cursor.
	cursorOff := clampOff(el.Input.CursorOffset, len(value))
	metrics := canvas.MeasureText(value[:cursorOff], style)
	cursorX := textX + metrics.Width
	canvas.FillRect(draw.R(cursorX, textY, 2, style.Size), draw.SolidPaint(tokens.Colors.Text.Primary))

	canvas.PopClip()

	// Hit target for touch-based cursor repositioning.
	if ix != nil {
		is := el.Input
		boundaries := text.GraphemeClusters(value)
		boundaryXs := make([]float32, len(boundaries))
		for i, boff := range boundaries {
			if boff == 0 {
				boundaryXs[i] = textX
			} else {
				m := canvas.MeasureText(value[:boff], style)
				boundaryXs[i] = textX + m.Width
			}
		}
		bXs := boundaryXs
		bOffs := boundaries
		ix.RegisterDrag(rect,
			func(mx, _ float32) {
				off := closestProxyBoundary(bXs, bOffs, mx)
				is.CursorOffset = off
				is.ClearSelection()
			},
		)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// closestProxyBoundary finds the grapheme boundary closest to mx.
func closestProxyBoundary(xs []float32, offsets []int, mx float32) int {
	if len(xs) == 0 {
		return 0
	}
	bestIdx := 0
	bestDist := abs32(mx - xs[0])
	for i := 1; i < len(xs); i++ {
		d := abs32(mx - xs[i])
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	return offsets[bestIdx]
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clampOff(off, length int) int {
	if off < 0 {
		return 0
	}
	if off > length {
		return length
	}
	return off
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// TreeEqual implements ui.TreeEqualizer.
func (el InputProxyElement) TreeEqual(other ui.Element) bool {
	_, ok := other.(InputProxyElement)
	return ok
}

// ResolveChildren implements ui.ChildResolver (no children).
func (el InputProxyElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return el
}
