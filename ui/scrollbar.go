package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// DrawScrollbar renders a vertical scrollbar track+thumb and registers
// hit-targets for click-to-scroll and thumb-drag via the Interactor.
// It returns the width consumed by the scrollbar (in pixels).
func DrawScrollbar(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor, state *ScrollState, trackX, trackY, viewportH int, contentH, offset float32) int {
	trackW := int(tokens.Scroll.TrackWidth)
	if trackW <= 0 {
		trackW = 8
	}
	thumbR := tokens.Scroll.ThumbRadius
	if thumbR <= 0 {
		thumbR = 4
	}

	trackColor := tokens.Colors.Surface.Hovered
	thumbColor := tokens.Colors.Surface.Pressed

	// Track
	canvas.FillRoundRect(
		draw.R(float32(trackX), float32(trackY), float32(trackW), float32(viewportH)),
		thumbR, draw.SolidPaint(trackColor))

	// Thumb
	ratio := float32(viewportH) / contentH
	thumbH := int(float32(viewportH) * ratio)
	if thumbH < 20 {
		thumbH = 20
	}

	maxScroll := contentH - float32(viewportH)
	thumbTravel := float32(viewportH - thumbH)
	var thumbY float32
	if maxScroll > 0 {
		thumbY = float32(trackY) + (offset/maxScroll)*thumbTravel
	} else {
		thumbY = float32(trackY)
	}

	canvas.FillRoundRect(
		draw.R(float32(trackX), thumbY, float32(trackW), float32(thumbH)),
		thumbR, draw.SolidPaint(thumbColor))

	// Track-click hit target.
	if state != nil {
		st := state
		ms := maxScroll
		tY := float32(trackY)
		vH := float32(viewportH)
		ix.RegisterClickAt(
			draw.R(float32(trackX), float32(trackY), float32(trackW), float32(viewportH)),
			func(_, y float32) {
				frac := (y - tY) / vH
				if frac < 0 {
					frac = 0
				}
				if frac > 1 {
					frac = 1
				}
				st.Offset = frac * ms
			},
		)
	}

	// Thumb-drag hit target — allows dragging the scrollbar thumb.
	// Registered AFTER the track-click so it wins in hit-testing
	// (HitTest iterates last-to-first).
	if state != nil && thumbTravel > 0 {
		st := state
		ms := maxScroll
		tY := float32(trackY)
		tH := float32(thumbH)
		tt := thumbTravel
		thumbRect := draw.R(float32(trackX), thumbY, float32(trackW), float32(thumbH))
		ix.RegisterDrag(thumbRect, func(_, y float32) {
			// Map mouse Y to scroll offset; treat y as centre of thumb.
			frac := (y - tY - tH/2) / tt
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			st.Offset = frac * ms
		})
	}

	return trackW
}
