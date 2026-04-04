package form

import (
	"fmt"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Orientation controls the layout direction of a Stepper (RFC-004 §6.3).
type Orientation uint8

const (
	Horizontal Orientation = iota // [−] [value] [+]
	Vertical                      // [▲] [value] [▼]
)

// Layout constants for stepper.
const (
	stepperBtnSize    = 32
	stepperValueW     = 64
	stepperValueH     = 32
	stepperPad        = 4
)

// Stepper is a minimal increment/decrement widget (RFC-004 §6.3).
type Stepper struct {
	ui.BaseElement
	Value       int
	Min         int
	Max         int
	Step        int
	Label       string
	Format      func(int) string
	OnChange    func(int)
	Disabled    bool
	Orientation Orientation
}

// StepperOption configures a Stepper element.
type StepperOption func(*Stepper)

// WithStepperRange sets the min and max bounds.
func WithStepperRange(min, max int) StepperOption {
	return func(s *Stepper) { s.Min = min; s.Max = max }
}

// WithStepperStep sets the step size.
func WithStepperStep(step int) StepperOption {
	return func(s *Stepper) { s.Step = step }
}

// WithStepperLabel sets the label displayed between the buttons.
func WithStepperLabel(label string) StepperOption {
	return func(s *Stepper) { s.Label = label }
}

// WithStepperFormat sets a custom format function.
func WithStepperFormat(fn func(int) string) StepperOption {
	return func(s *Stepper) { s.Format = fn }
}

// WithOnStepperChange sets the change callback.
func WithOnStepperChange(fn func(int)) StepperOption {
	return func(s *Stepper) { s.OnChange = fn }
}

// WithStepperDisabled marks the Stepper as disabled.
func WithStepperDisabled() StepperOption {
	return func(s *Stepper) { s.Disabled = true }
}

// WithStepperOrientation sets horizontal or vertical layout.
func WithStepperOrientation(o Orientation) StepperOption {
	return func(s *Stepper) { s.Orientation = o }
}

// NewStepper creates a stepper element. Defaults: Min=0, Max=100, Step=1, Horizontal.
func NewStepper(value int, opts ...StepperOption) ui.Element {
	el := Stepper{Value: value, Min: 0, Max: 100, Step: 1}
	for _, o := range opts {
		o(&el)
	}
	return el
}

func (s Stepper) clamp(v int) int {
	if v < s.Min {
		return s.Min
	}
	if v > s.Max {
		return s.Max
	}
	return v
}

func (s Stepper) formatValue() string {
	if s.Format != nil {
		return s.Format(s.Value)
	}
	if s.Label != "" {
		return s.Label
	}
	return fmt.Sprintf("%d", s.Value)
}

// LayoutSelf implements ui.Layouter.
func (s Stepper) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	arrowStyle := tokens.Typography.LabelSmall
	arrowStyle.FontFamily = "Phosphor"

	arrowColor := tokens.Colors.Text.Secondary
	textColor := tokens.Colors.Text.Primary
	borderColor := tokens.Colors.Stroke.Border
	if s.Disabled {
		arrowColor = tokens.Colors.Text.Disabled
		textColor = tokens.Colors.Text.Disabled
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	// Focus management.
	var focused bool
	if focus != nil && !s.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	if s.Orientation == Vertical {
		return s.layoutVertical(ctx, area, canvas, tokens, ix, style, arrowStyle, arrowColor, textColor, borderColor, focused)
	}
	return s.layoutHorizontal(ctx, area, canvas, tokens, ix, style, arrowStyle, arrowColor, textColor, borderColor, focused)
}

func (s Stepper) layoutHorizontal(_ *ui.LayoutContext, area ui.Bounds, canvas draw.Canvas,
	tokens theme.TokenSet, ix *ui.Interactor, style, arrowStyle draw.TextStyle,
	arrowColor, textColor, borderColor draw.Color, focused bool) ui.Bounds {

	totalW := stepperBtnSize + stepperPad + stepperValueW + stepperPad + stepperBtnSize
	if area.W < totalW {
		totalW = area.W
	}
	h := stepperBtnSize

	// Minus button.
	minusRect := draw.R(float32(area.X), float32(area.Y), float32(stepperBtnSize), float32(h))
	canvas.FillRoundRect(minusRect, tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(minusRect, tokens.Radii.Button, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	var minusHover float32
	if s.Disabled {
		ix.RegisterHit(minusRect, nil)
	} else {
		var fn func()
		if s.OnChange != nil {
			onChange := s.OnChange
			val := s.clamp(s.Value - s.Step)
			fn = func() { onChange(val) }
		}
		minusHover = ix.RegisterHit(minusRect, fn)
	}
	if minusHover > 0 {
		canvas.FillRoundRect(minusRect, tokens.Radii.Button, draw.SolidPaint(draw.Color{A: minusHover * 0.08}))
	}
	canvas.DrawText(icons.Minus, draw.Pt(
		float32(area.X)+float32(stepperBtnSize)/2-arrowStyle.Size/2,
		float32(area.Y)+float32(h)/2-arrowStyle.Size/2,
	), arrowStyle, arrowColor)

	// Value text.
	valueX := area.X + stepperBtnSize + stepperPad
	valueRect := draw.R(float32(valueX), float32(area.Y), float32(stepperValueW), float32(h))
	canvas.FillRoundRect(valueRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(valueRect, tokens.Radii.Input, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	txt := s.formatValue()
	m := canvas.MeasureText(txt, style)
	textXc := float32(valueX) + float32(stepperValueW)/2 - m.Width/2
	textYc := float32(area.Y) + float32(h)/2 - style.Size/2
	canvas.DrawText(txt, draw.Pt(textXc, textYc), style, textColor)

	// Plus button.
	plusX := valueX + stepperValueW + stepperPad
	plusRect := draw.R(float32(plusX), float32(area.Y), float32(stepperBtnSize), float32(h))
	canvas.FillRoundRect(plusRect, tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(plusRect, tokens.Radii.Button, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	var plusHover float32
	if s.Disabled {
		ix.RegisterHit(plusRect, nil)
	} else {
		var fn func()
		if s.OnChange != nil {
			onChange := s.OnChange
			val := s.clamp(s.Value + s.Step)
			fn = func() { onChange(val) }
		}
		plusHover = ix.RegisterHit(plusRect, fn)
	}
	if plusHover > 0 {
		canvas.FillRoundRect(plusRect, tokens.Radii.Button, draw.SolidPaint(draw.Color{A: plusHover * 0.08}))
	}
	canvas.DrawText(icons.Plus, draw.Pt(
		float32(plusX)+float32(stepperBtnSize)/2-arrowStyle.Size/2,
		float32(area.Y)+float32(h)/2-arrowStyle.Size/2,
	), arrowStyle, arrowColor)

	if focused {
		outerRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(h))
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Button, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: h}
}

func (s Stepper) layoutVertical(_ *ui.LayoutContext, area ui.Bounds, canvas draw.Canvas,
	tokens theme.TokenSet, ix *ui.Interactor, style, arrowStyle draw.TextStyle,
	arrowColor, textColor, borderColor draw.Color, focused bool) ui.Bounds {

	w := stepperBtnSize
	totalH := stepperBtnSize + stepperValueH + stepperBtnSize

	// Up button.
	upRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(stepperBtnSize))
	canvas.FillRoundRect(upRect, tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(upRect, tokens.Radii.Button, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	var upHover float32
	if s.Disabled {
		ix.RegisterHit(upRect, nil)
	} else {
		var fn func()
		if s.OnChange != nil {
			onChange := s.OnChange
			val := s.clamp(s.Value + s.Step)
			fn = func() { onChange(val) }
		}
		upHover = ix.RegisterHit(upRect, fn)
	}
	if upHover > 0 {
		canvas.FillRoundRect(upRect, tokens.Radii.Button, draw.SolidPaint(draw.Color{A: upHover * 0.08}))
	}
	canvas.DrawText(icons.CaretUp, draw.Pt(
		float32(area.X)+float32(w)/2-arrowStyle.Size/2,
		float32(area.Y)+float32(stepperBtnSize)/2-arrowStyle.Size/2,
	), arrowStyle, arrowColor)

	// Value text.
	valueY := area.Y + stepperBtnSize
	valueRect := draw.R(float32(area.X), float32(valueY), float32(w), float32(stepperValueH))
	canvas.FillRoundRect(valueRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(valueRect, tokens.Radii.Input, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	txt := s.formatValue()
	m := canvas.MeasureText(txt, style)
	textXc := float32(area.X) + float32(w)/2 - m.Width/2
	textYc := float32(valueY) + float32(stepperValueH)/2 - style.Size/2
	canvas.DrawText(txt, draw.Pt(textXc, textYc), style, textColor)

	// Down button.
	downY := valueY + stepperValueH
	downRect := draw.R(float32(area.X), float32(downY), float32(w), float32(stepperBtnSize))
	canvas.FillRoundRect(downRect, tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(downRect, tokens.Radii.Button, draw.Stroke{Paint: draw.SolidPaint(borderColor), Width: 1})

	var downHover float32
	if s.Disabled {
		ix.RegisterHit(downRect, nil)
	} else {
		var fn func()
		if s.OnChange != nil {
			onChange := s.OnChange
			val := s.clamp(s.Value - s.Step)
			fn = func() { onChange(val) }
		}
		downHover = ix.RegisterHit(downRect, fn)
	}
	if downHover > 0 {
		canvas.FillRoundRect(downRect, tokens.Radii.Button, draw.SolidPaint(draw.Color{A: downHover * 0.08}))
	}
	canvas.DrawText(icons.CaretDown, draw.Pt(
		float32(area.X)+float32(w)/2-arrowStyle.Size/2,
		float32(downY)+float32(stepperBtnSize)/2-arrowStyle.Size/2,
	), arrowStyle, arrowColor)

	if focused {
		outerRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(totalH))
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Button, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (s Stepper) TreeEqual(other ui.Element) bool {
	sb, ok := other.(Stepper)
	return ok && s.Value == sb.Value && s.Min == sb.Min && s.Max == sb.Max && s.Orientation == sb.Orientation
}

// ResolveChildren implements ui.ChildResolver. Stepper is a leaf.
func (s Stepper) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return s
}

// WalkAccess implements ui.AccessWalker.
func (s Stepper) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleSpinButton,
		Label:  s.Label,
		States: a11y.AccessStates{Disabled: s.Disabled},
		NumericValue: &a11y.AccessNumericValue{
			Current: float64(s.Value),
			Min:     float64(s.Min),
			Max:     float64(s.Max),
			Step:    float64(s.Step),
		},
	}, parentIdx, a11y.Rect{})
}
