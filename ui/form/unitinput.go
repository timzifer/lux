package form

import (
	"fmt"
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for unit input.
const (
	unitInputW        = 200
	unitDropdownW     = 60
	unitDropdownItemH = 28
	unitDropdownPad   = 4
)

// UnitDef defines an available unit with conversion factor (RFC-004 §6.7).
type UnitDef struct {
	Symbol string  // "mm", "cm", "in"
	Label  string  // "Millimeter" (for dropdown)
	Factor float64 // Conversion factor relative to base unit
}

// UnitInputState holds the dropdown open/close state.
type UnitInputState struct {
	Open bool
}

// NewUnitInputState creates a new UnitInputState.
func NewUnitInputState() *UnitInputState { return &UnitInputState{} }

// UnitInput is a numeric input with unit selection (RFC-004 §6.7).
// Composes a NumericInput for the value area via ctx.LayoutChild().
type UnitInput struct {
	ui.BaseElement

	Value    float64
	Unit     string
	Units    []UnitDef
	OnChange func(value float64, unit string)
	Min      *float64
	Max      *float64
	Step     float64
	State    *UnitInputState
	Disabled bool
}

// UnitInputOption configures a UnitInput element.
type UnitInputOption func(*UnitInput)

func WithUnitRange(min, max float64) UnitInputOption {
	return func(u *UnitInput) { u.Min = &min; u.Max = &max }
}

func WithUnitStep(step float64) UnitInputOption {
	return func(u *UnitInput) { u.Step = step }
}

func WithUnitState(s *UnitInputState) UnitInputOption {
	return func(u *UnitInput) { u.State = s }
}

func WithOnUnitChange(fn func(float64, string)) UnitInputOption {
	return func(u *UnitInput) { u.OnChange = fn }
}

func WithUnitDisabled() UnitInputOption {
	return func(u *UnitInput) { u.Disabled = true }
}

func NewUnitInput(value float64, unit string, units []UnitDef, opts ...UnitInputOption) ui.Element {
	el := UnitInput{Value: value, Unit: unit, Units: units, Step: 1}
	for _, o := range opts {
		o(&el)
	}
	return el
}

func (u UnitInput) DisplayValue() float64 {
	f := u.unitFactor(u.Unit)
	if f == 0 {
		return u.Value
	}
	return u.Value / f
}

func (u UnitInput) ConvertToBase(value float64, unitSymbol string) float64 {
	f := u.unitFactor(unitSymbol)
	if f == 0 {
		return value
	}
	return value * f
}

func (u UnitInput) unitFactor(symbol string) float64 {
	for _, ud := range u.Units {
		if ud.Symbol == symbol {
			return ud.Factor
		}
	}
	return 1
}

func (u UnitInput) minVal() float64 {
	if u.Min != nil {
		return *u.Min
	}
	return math.Inf(-1)
}

func (u UnitInput) maxVal() float64 {
	if u.Max != nil {
		return *u.Max
	}
	return math.Inf(1)
}

func (u UnitInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumeric }

func (u UnitInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := unitInputW
	if area.W < w {
		w = area.W
	}

	// ── Left part: compose a NumericInput via ctx.LayoutChild ──
	numericW := w - unitDropdownW
	if numericW < 0 {
		numericW = w
	}

	onChange := u.OnChange
	unit := u.Unit
	var numOpts []NumericInputOption
	if u.Min != nil && u.Max != nil {
		numOpts = append(numOpts, WithNumericRange(*u.Min, *u.Max))
	}
	numOpts = append(numOpts, WithNumericStep(u.Step))
	numOpts = append(numOpts, WithNumericKind(NumericFloat))
	numOpts = append(numOpts, WithPrecision(2))
	if u.Disabled {
		numOpts = append(numOpts, WithNumericDisabled())
	}
	if onChange != nil {
		numOpts = append(numOpts, WithOnNumericChange(func(v float64) {
			onChange(v, unit)
		}))
	}

	numChild := NewNumericInput(u.Value, numOpts...)
	ctx.LayoutChild(numChild, ui.Bounds{X: area.X, Y: area.Y, W: numericW, H: h})

	// ── Right part: unit dropdown button ──
	unitBtnX := area.X + numericW
	unitBtnRect := draw.R(float32(unitBtnX), float32(area.Y), float32(unitDropdownW), float32(h))

	borderColor := tokens.Colors.Stroke.Border
	if u.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	fillColor := tokens.Colors.Surface.Elevated
	if u.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(unitBtnRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
	canvas.FillRoundRect(
		draw.R(float32(unitBtnX+1), float32(area.Y+1), float32(max(unitDropdownW-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	isOpen := u.State != nil && u.State.Open && !u.Disabled

	var unitHover float32
	if u.Disabled {
		ix.RegisterHit(unitBtnRect, nil)
	} else if u.State != nil {
		state := u.State
		unitHover = ix.RegisterHit(unitBtnRect, func() { state.Open = !state.Open })
	}
	if unitHover > 0 {
		canvas.FillRect(unitBtnRect, draw.SolidPaint(draw.Color{A: unitHover * 0.08}))
	}

	// Unit symbol + caret.
	textY := area.Y + textFieldPadY
	arrowStyle := tokens.Typography.LabelSmall
	arrowStyle.FontFamily = "Phosphor"
	unitColor := tokens.Colors.Text.Secondary
	if u.Disabled {
		unitColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(u.Unit, draw.Pt(float32(unitBtnX+4), float32(textY)), style, unitColor)
	arrowX := float32(area.X+w-textFieldPadX) - arrowStyle.Size
	canvas.DrawText(icons.CaretDown, draw.Pt(arrowX, float32(textY)), arrowStyle, unitColor)

	// ── Dropdown overlay ──
	if isOpen && u.State != nil && overlays != nil {
		state := u.State
		baseValue := u.Value
		units := u.Units
		curUnit := u.Unit
		winW := overlays.WindowW
		winH := overlays.WindowH
		dropX := unitBtnX
		dropY := area.Y + h
		dropW := unitDropdownW + 40
		dropH := len(units)*unitDropdownItemH + unitDropdownPad*2

		if dropY+dropH > winH && area.Y-dropH >= 0 {
			dropY = area.Y - dropH
		}

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					state.Open = false
				})
				bgRect := draw.R(float32(dropX), float32(dropY), float32(dropW), float32(dropH))
				canvas.FillRoundRect(bgRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				canvas.StrokeRoundRect(bgRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				bodyStyle := tokens.Typography.Body
				for i, ud := range units {
					itemY := dropY + unitDropdownPad + i*unitDropdownItemH
					itemRect := draw.R(float32(dropX), float32(itemY), float32(dropW), float32(unitDropdownItemH))
					sym := ud.Symbol
					itemClick := func() {
						if onChange != nil {
							onChange(baseValue, sym)
						}
						state.Open = false
					}
					ho := ix.RegisterHit(itemRect, itemClick)
					if ho > 0 {
						canvas.FillRect(itemRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}
					if ud.Symbol == curUnit {
						canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{
							R: tokens.Colors.Accent.Primary.R,
							G: tokens.Colors.Accent.Primary.G,
							B: tokens.Colors.Accent.Primary.B,
							A: 0.15,
						}))
					}
					label := ud.Symbol
					if ud.Label != "" {
						label += " — " + ud.Label
					}
					canvas.DrawText(label,
						draw.Pt(float32(dropX+unitDropdownPad), float32(itemY+4)),
						bodyStyle, tokens.Colors.Text.Primary)
				}
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func (u UnitInput) TreeEqual(other ui.Element) bool {
	ub, ok := other.(UnitInput)
	return ok && u.Value == ub.Value && u.Unit == ub.Unit
}

func (u UnitInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return u
}

func (u UnitInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleSpinButton,
		Value:  fmt.Sprintf("%.2f %s", u.DisplayValue(), u.Unit),
		States: a11y.AccessStates{Disabled: u.Disabled},
		NumericValue: &a11y.AccessNumericValue{
			Current: u.Value,
			Min:     u.minVal(),
			Max:     u.maxVal(),
			Step:    u.Step,
		},
	}, parentIdx, a11y.Rect{})
}

var _ osk.OSKRequester = UnitInput{}
