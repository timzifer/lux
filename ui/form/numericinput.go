package form

import (
	"fmt"
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Layout constants for numeric input.
const (
	numericInputW        = 200
	numericInputPadX     = 8
	numericInputPadY     = 8
	numericStepperW      = 24
	numericStepperBorder = 1
)

// NumericInput is a number input with stepper buttons and optional unit suffix.
type NumericInput struct {
	ui.BaseElement
	Value    float64
	Min      float64
	Max      float64
	Step     float64
	Unit     string
	OnChange func(float64)
	Disabled bool
}

// NumericInputOption configures a NumericInput element.
type NumericInputOption func(*NumericInput)

// WithNumericRange sets the min and max bounds.
func WithNumericRange(min, max float64) NumericInputOption {
	return func(n *NumericInput) { n.Min = min; n.Max = max }
}

// WithNumericStep sets the increment/decrement step.
func WithNumericStep(step float64) NumericInputOption {
	return func(n *NumericInput) { n.Step = step }
}

// WithUnit sets the unit suffix displayed after the value.
func WithUnit(unit string) NumericInputOption {
	return func(n *NumericInput) { n.Unit = unit }
}

// WithOnNumericChange sets the callback invoked when the value changes.
func WithOnNumericChange(fn func(float64)) NumericInputOption {
	return func(n *NumericInput) { n.OnChange = fn }
}

// WithNumericDisabled marks the NumericInput as disabled.
func WithNumericDisabled() NumericInputOption {
	return func(n *NumericInput) { n.Disabled = true }
}

// NewNumericInput creates a numeric input element.
// Defaults: Min=0, Max=100, Step=1.
func NewNumericInput(value float64, opts ...NumericInputOption) ui.Element {
	el := NumericInput{Value: value, Min: 0, Max: 100, Step: 1}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// NumericInputDisabled creates a disabled numeric input.
func NumericInputDisabled(value float64, opts ...NumericInputOption) ui.Element {
	el := NumericInput{Value: value, Min: 0, Max: 100, Step: 1, Disabled: true}
	for _, o := range opts {
		o(&el)
	}
	return el
}

func (n NumericInput) clamp(v float64) float64 {
	if v < n.Min {
		return n.Min
	}
	if v > n.Max {
		return n.Max
	}
	return v
}

// LayoutSelf implements ui.Layouter.
func (n NumericInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + numericInputPadY*2

	w := numericInputW
	if area.W < w {
		w = area.W
	}

	// Focus management.
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Border
	borderColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill
	fillColor := tokens.Colors.Surface.Elevated
	if n.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Focus glow.
	if focused {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Value text + unit.
	textX := area.X + numericInputPadX
	textY := area.Y + numericInputPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}

	displayText := formatNumeric(n.Value, n.Step)
	if n.Unit != "" {
		displayText += " " + n.Unit
	}
	canvas.DrawText(displayText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Stepper divider line.
	stepperX := area.X + w - numericStepperW
	canvas.FillRect(
		draw.R(float32(stepperX), float32(area.Y+1), 1, float32(max(h-2, 0))),
		draw.SolidPaint(borderColor))

	// Up button (top half of stepper area).
	halfH := h / 2
	upRect := draw.R(float32(stepperX), float32(area.Y), float32(numericStepperW), float32(halfH))
	var upHover float32
	if n.Disabled {
		ix.RegisterHit(upRect, nil)
	} else {
		var upFn func()
		if n.OnChange != nil {
			onChange := n.OnChange
			val := n.clamp(n.Value + n.Step)
			upFn = func() { onChange(val) }
		}
		upHover = ix.RegisterHit(upRect, upFn)
	}
	if upHover > 0 {
		canvas.FillRect(upRect, draw.SolidPaint(draw.Color{R: 0, G: 0, B: 0, A: upHover * 0.08}))
	}

	// Down button (bottom half of stepper area).
	downRect := draw.R(float32(stepperX), float32(area.Y+halfH), float32(numericStepperW), float32(h-halfH))
	var downHover float32
	if n.Disabled {
		ix.RegisterHit(downRect, nil)
	} else {
		var downFn func()
		if n.OnChange != nil {
			onChange := n.OnChange
			val := n.clamp(n.Value - n.Step)
			downFn = func() { onChange(val) }
		}
		downHover = ix.RegisterHit(downRect, downFn)
	}
	if downHover > 0 {
		canvas.FillRect(downRect, draw.SolidPaint(draw.Color{R: 0, G: 0, B: 0, A: downHover * 0.08}))
	}

	// Stepper mid-divider.
	canvas.FillRect(
		draw.R(float32(stepperX), float32(area.Y+halfH), float32(numericStepperW), 1),
		draw.SolidPaint(borderColor))

	// Up/Down arrow icons.
	arrowStyle := tokens.Typography.LabelSmall
	arrowStyle.FontFamily = "Phosphor"
	arrowColor := tokens.Colors.Text.Secondary
	if n.Disabled {
		arrowColor = tokens.Colors.Text.Disabled
	}
	arrowCenterX := float32(stepperX) + float32(numericStepperW)/2 - arrowStyle.Size/2
	upArrowY := float32(area.Y) + float32(halfH)/2 - arrowStyle.Size/2
	canvas.DrawText(icons.CaretUp, draw.Pt(arrowCenterX, upArrowY), arrowStyle, arrowColor)
	downArrowY := float32(area.Y+halfH) + float32(h-halfH)/2 - arrowStyle.Size/2
	canvas.DrawText(icons.CaretDown, draw.Pt(arrowCenterX, downArrowY), arrowStyle, arrowColor)

	// Drag on value area for continuous adjustment.
	valueRect := draw.R(float32(area.X), float32(area.Y), float32(max(w-numericStepperW, 0)), float32(h))
	if !n.Disabled && n.OnChange != nil {
		areaX := float32(area.X)
		valueW := float32(max(w-numericStepperW, 1))
		onChange := n.OnChange
		baseVal := n.Value
		minV := n.Min
		maxV := n.Max
		ix.RegisterDrag(valueRect, func(x, _ float32) {
			delta := float64((x - areaX - valueW/2) / valueW)
			newVal := baseVal + delta*(maxV-minV)*0.5
			if newVal < minV {
				newVal = minV
			}
			if newVal > maxV {
				newVal = maxV
			}
			onChange(newVal)
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// formatNumeric formats a float value for display. If step is integer-like, show no decimals.
func formatNumeric(value, step float64) string {
	if step == math.Trunc(step) && step >= 1 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

// TreeEqual implements ui.TreeEqualizer.
func (n NumericInput) TreeEqual(other ui.Element) bool {
	nb, ok := other.(NumericInput)
	return ok && n.Value == nb.Value && n.Unit == nb.Unit && n.Min == nb.Min && n.Max == nb.Max
}

// ResolveChildren implements ui.ChildResolver. NumericInput is a leaf.
func (n NumericInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n NumericInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleSpinButton,
		States: a11y.AccessStates{Disabled: n.Disabled},
		NumericValue: &a11y.AccessNumericValue{
			Current: n.Value,
			Min:     n.Min,
			Max:     n.Max,
			Step:    n.Step,
		},
	}
	if n.Unit != "" {
		an.Label = n.Unit
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}
