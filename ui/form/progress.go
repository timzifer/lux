package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Layout constants for progress bar.
const (
	progressBarH    = 6
	progressBarMaxW = 200
)

// ProgressBar is a determinate or indeterminate progress indicator.
type ProgressBar struct {
	ui.BaseElement
	Value         float32
	Indeterminate bool
	Phase         float32 // 0.0-1.0, drives indeterminate animation position
}

// NewProgressBar creates a determinate progress bar (0.0-1.0).
func NewProgressBar(value float32) ui.Element {
	return ProgressBar{Value: value}
}

// Indeterminate creates an indeterminate progress bar.
// An optional phase (0.0-1.0) controls the animation position; pass
// a value derived from app.TickMsg to animate the bar.
func Indeterminate(phase ...float32) ui.Element {
	var p float32
	if len(phase) > 0 {
		p = phase[0]
	}
	return ProgressBar{Indeterminate: true, Phase: p}
}

// LayoutSelf implements ui.Layouter.
func (n ProgressBar) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	trackW := progressBarMaxW
	if area.W < trackW {
		trackW = area.W
	}

	// Track
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(progressBarH)),
		float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Surface.Pressed))

	if n.Indeterminate {
		// Animated 30% bar that slides across the track.
		barW := int(float32(trackW) * 0.3)
		phase := n.Phase
		if phase < 0 {
			phase = 0
		}
		if phase > 1 {
			phase -= float32(int(phase)) // wrap
		}
		travel := trackW - barW
		barX := area.X + int(float32(travel)*phase)
		canvas.FillRoundRect(
			draw.R(float32(barX), float32(area.Y), float32(barW), float32(progressBarH)),
			float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Accent.Primary))
	} else {
		// Determinate fill
		val := n.Value
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		filledW := int(float32(trackW) * val)
		if filledW > 0 {
			canvas.FillRoundRect(
				draw.R(float32(area.X), float32(area.Y), float32(filledW), float32(progressBarH)),
				float32(progressBarH)/2, draw.SolidPaint(tokens.Colors.Accent.Primary))
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: trackW, H: progressBarH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n ProgressBar) TreeEqual(other ui.Element) bool {
	nb, ok := other.(ProgressBar)
	return ok && n.Value == nb.Value && n.Indeterminate == nb.Indeterminate
}

// ResolveChildren implements ui.ChildResolver. ProgressBar is a leaf.
func (n ProgressBar) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n ProgressBar) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleProgressBar,
		States: a11y.AccessStates{ReadOnly: true},
	}
	if !n.Indeterminate {
		an.NumericValue = &a11y.AccessNumericValue{
			Current: float64(n.Value),
			Min:     0,
			Max:     1,
		}
	} else {
		an.States.Busy = true
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

// ProgressBarIndeterminate is an alias for Indeterminate.
func ProgressBarIndeterminate(phase ...float32) ui.Element { return Indeterminate(phase...) }
