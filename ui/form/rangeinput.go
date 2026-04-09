package form

import (
	"fmt"
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Layout constants for range input.
const (
	rangeInputW      = 200
	rangeTrackH      = 4
	rangeHeight      = 24
	rangeThumbD      = 16
	rangeLabelOffset = 2
)

// RangeInput is a dual-slider for min/max range selection (RFC-004 §6.10).
type RangeInput struct {
	ui.BaseElement

	// Low is the lower bound of the selected range.
	Low float64

	// High is the upper bound of the selected range.
	High float64

	// Min, Max define the full slider range.
	Min float64
	Max float64

	// Step is the snap increment. 0 = continuous.
	Step float64

	// OnChange is called when Low or High changes.
	OnChange func(low, high float64)

	// ShowLabels shows value labels at the handles.
	ShowLabels bool

	// Format formats displayed values. Default: "%.0f".
	Format func(float64) string

	Disabled bool
}

// RangeInputOption configures a RangeInput element.
type RangeInputOption func(*RangeInput)

// WithRangeStep sets the step.
func WithRangeStep(step float64) RangeInputOption {
	return func(r *RangeInput) { r.Step = step }
}

// WithOnRangeChange sets the change callback.
func WithOnRangeChange(fn func(float64, float64)) RangeInputOption {
	return func(r *RangeInput) { r.OnChange = fn }
}

// WithRangeLabels enables value labels on the handles.
func WithRangeLabels() RangeInputOption {
	return func(r *RangeInput) { r.ShowLabels = true }
}

// WithRangeFormat sets the value formatter.
func WithRangeFormat(fn func(float64) string) RangeInputOption {
	return func(r *RangeInput) { r.Format = fn }
}

// WithRangeDisabled disables the widget.
func WithRangeDisabled() RangeInputOption {
	return func(r *RangeInput) { r.Disabled = true }
}

// NewRangeInput creates a dual-slider range input.
func NewRangeInput(low, high, min, max float64, opts ...RangeInputOption) ui.Element {
	el := RangeInput{Low: low, High: high, Min: min, Max: max}
	for _, o := range opts {
		o(&el)
	}
	return el
}

func (r RangeInput) snap(v float64) float64 {
	if r.Step <= 0 {
		return v
	}
	return math.Round((v-r.Min)/r.Step)*r.Step + r.Min
}

func (r RangeInput) clampLow(v float64) float64 {
	if v < r.Min {
		return r.Min
	}
	if v > r.High {
		return r.High
	}
	return v
}

func (r RangeInput) clampHigh(v float64) float64 {
	if v < r.Low {
		return r.Low
	}
	if v > r.Max {
		return r.Max
	}
	return v
}

func (r RangeInput) formatVal(v float64) string {
	if r.Format != nil {
		return r.Format(v)
	}
	return fmt.Sprintf("%.0f", v)
}

// LayoutSelf implements ui.Layouter.
func (r RangeInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	// Touch-adaptive sizing: grow thumbs to meet MinTouchTarget.
	thumbD := rangeThumbD
	height := rangeHeight
	if ctx.IsTouch() && ctx.Profile != nil {
		thumbD = int(ctx.Profile.MinTouchTarget)
		height = thumbD + 8
	}

	trackW := rangeInputW
	if area.W < trackW {
		trackW = area.W
	}

	labelH := 0
	if r.ShowLabels {
		labelH = int(tokens.Typography.LabelSmall.Size) + rangeLabelOffset
	}
	totalH := height + labelH

	// Focus management.
	var focused bool
	if focus != nil && !r.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	trackY := area.Y + (height-rangeTrackH)/2

	// Full track background.
	trackColor := tokens.Colors.Surface.Pressed
	canvas.FillRoundRect(
		draw.R(float32(area.X), float32(trackY), float32(trackW), float32(rangeTrackH)),
		float32(rangeTrackH)/2, draw.SolidPaint(trackColor))

	// Normalize positions.
	span := r.Max - r.Min
	if span <= 0 {
		span = 1
	}
	lowFrac := (r.Low - r.Min) / span
	highFrac := (r.High - r.Min) / span
	if lowFrac < 0 {
		lowFrac = 0
	}
	if highFrac > 1 {
		highFrac = 1
	}

	thumbR := float32(thumbD) / 2
	usableW := float32(trackW) - float32(thumbD)
	if usableW < 0 {
		usableW = 0
	}
	lowX := float32(area.X) + thumbR + usableW*float32(lowFrac)
	highX := float32(area.X) + thumbR + usableW*float32(highFrac)

	// Filled range between handles.
	filledColor := tokens.Colors.Accent.Primary
	if r.Disabled {
		filledColor = ui.DisabledColor(filledColor, tokens.Colors.Surface.Base)
	}
	if highX > lowX {
		canvas.FillRoundRect(
			draw.R(lowX, float32(trackY), highX-lowX, float32(rangeTrackH)),
			float32(rangeTrackH)/2, draw.SolidPaint(filledColor))
	}

	// Low handle.
	lowThumbX := lowX - float32(thumbD)/2
	lowThumbY := float32(area.Y) + float32(height-thumbD)/2
	lowThumbRect := draw.R(lowThumbX, lowThumbY, float32(thumbD), float32(thumbD))
	thumbColor := tokens.Colors.Accent.Primary
	if r.Disabled {
		thumbColor = ui.DisabledColor(thumbColor, tokens.Colors.Surface.Base)
	}
	canvas.FillEllipse(lowThumbRect, draw.SolidPaint(thumbColor))

	if !r.Disabled && r.OnChange != nil {
		onChange := r.OnChange
		rMin := r.Min
		rMax := r.Max
		high := r.High
		step := r.Step
		trackStart := float32(area.X) + thumbR
		uw := usableW
		ix.RegisterDrag(lowThumbRect, func(x, _ float32) {
			frac := float64(0)
			if uw > 0 {
				frac = float64((x - trackStart) / uw)
			}
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			newLow := rMin + frac*(rMax-rMin)
			if step > 0 {
				newLow = math.Round((newLow-rMin)/step)*step + rMin
			}
			if newLow > high {
				newLow = high
			}
			if newLow < rMin {
				newLow = rMin
			}
			onChange(newLow, high)
		})
	}

	// High handle.
	highThumbX := highX - float32(thumbD)/2
	highThumbY := float32(area.Y) + float32(height-thumbD)/2
	highThumbRect := draw.R(highThumbX, highThumbY, float32(thumbD), float32(thumbD))
	canvas.FillEllipse(highThumbRect, draw.SolidPaint(thumbColor))

	if !r.Disabled && r.OnChange != nil {
		onChange := r.OnChange
		rMin := r.Min
		rMax := r.Max
		low := r.Low
		step := r.Step
		trackStart := float32(area.X) + thumbR
		uw := usableW
		ix.RegisterDrag(highThumbRect, func(x, _ float32) {
			frac := float64(0)
			if uw > 0 {
				frac = float64((x - trackStart) / uw)
			}
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			newHigh := rMin + frac*(rMax-rMin)
			if step > 0 {
				newHigh = math.Round((newHigh-rMin)/step)*step + rMin
			}
			if newHigh < low {
				newHigh = low
			}
			if newHigh > rMax {
				newHigh = rMax
			}
			onChange(low, newHigh)
		})
	}

	// Value labels.
	if r.ShowLabels {
		labelStyle := tokens.Typography.LabelSmall
		labelColor := tokens.Colors.Text.Secondary
		if r.Disabled {
			labelColor = tokens.Colors.Text.Disabled
		}
		labelY := float32(area.Y + height + rangeLabelOffset)

		lowLabel := r.formatVal(r.Low)
		lm := canvas.MeasureText(lowLabel, labelStyle)
		canvas.DrawText(lowLabel, draw.Pt(lowX-lm.Width/2, labelY), labelStyle, labelColor)

		highLabel := r.formatVal(r.High)
		hm := canvas.MeasureText(highLabel, labelStyle)
		canvas.DrawText(highLabel, draw.Pt(highX-hm.Width/2, labelY), labelStyle, labelColor)
	}

	if focused {
		outerRect := draw.R(float32(area.X), float32(area.Y), float32(trackW), float32(totalH))
		ui.DrawFocusRing(canvas, outerRect, float32(rangeTrackH)/2, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: trackW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (r RangeInput) TreeEqual(other ui.Element) bool {
	rb, ok := other.(RangeInput)
	return ok && r.Low == rb.Low && r.High == rb.High && r.Min == rb.Min && r.Max == rb.Max
}

// ResolveChildren implements ui.ChildResolver. RangeInput is a leaf.
func (r RangeInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return r
}

// WalkAccess implements ui.AccessWalker.
func (r RangeInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	groupIdx := b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleGroup,
		Label: "Range",
	}, parentIdx, a11y.Rect{})

	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleSlider,
		Label: "Low",
		NumericValue: &a11y.AccessNumericValue{
			Current: r.Low,
			Min:     r.Min,
			Max:     r.High,
			Step:    r.Step,
		},
		States: a11y.AccessStates{Disabled: r.Disabled},
	}, int32(groupIdx), a11y.Rect{})

	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleSlider,
		Label: "High",
		NumericValue: &a11y.AccessNumericValue{
			Current: r.High,
			Min:     r.Low,
			Max:     r.Max,
			Step:    r.Step,
		},
		States: a11y.AccessStates{Disabled: r.Disabled},
	}, int32(groupIdx), a11y.Rect{})
}
