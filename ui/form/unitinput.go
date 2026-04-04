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
	unitInputW     = 200
	unitDropdownW  = 60
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
type UnitInput struct {
	ui.BaseElement

	// Value is the current value in the base unit.
	Value float64

	// Unit is the currently displayed unit symbol.
	Unit string

	// Units is the list of available units.
	Units []UnitDef

	// OnChange is called when value or unit changes (value in base unit).
	OnChange func(value float64, unit string)

	// Min, Max for the numeric input.
	Min *float64
	Max *float64

	// Step for +/- buttons.
	Step float64

	// State for dropdown.
	State *UnitInputState

	Disabled bool
}

// UnitInputOption configures a UnitInput element.
type UnitInputOption func(*UnitInput)

// WithUnitRange sets the value range.
func WithUnitRange(min, max float64) UnitInputOption {
	return func(u *UnitInput) { u.Min = &min; u.Max = &max }
}

// WithUnitStep sets the step size.
func WithUnitStep(step float64) UnitInputOption {
	return func(u *UnitInput) { u.Step = step }
}

// WithUnitState links the dropdown state.
func WithUnitState(s *UnitInputState) UnitInputOption {
	return func(u *UnitInput) { u.State = s }
}

// WithOnUnitChange sets the change callback.
func WithOnUnitChange(fn func(float64, string)) UnitInputOption {
	return func(u *UnitInput) { u.OnChange = fn }
}

// WithUnitDisabled disables the widget.
func WithUnitDisabled() UnitInputOption {
	return func(u *UnitInput) { u.Disabled = true }
}

// NewUnitInput creates a unit input element.
func NewUnitInput(value float64, unit string, units []UnitDef, opts ...UnitInputOption) ui.Element {
	el := UnitInput{Value: value, Unit: unit, Units: units, Step: 1}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// DisplayValue returns the value converted to the currently selected unit.
func (u UnitInput) DisplayValue() float64 {
	factor := u.unitFactor(u.Unit)
	if factor == 0 {
		return u.Value
	}
	return u.Value / factor
}

// ConvertToBase converts a value from a given unit back to the base unit.
func (u UnitInput) ConvertToBase(value float64, unitSymbol string) float64 {
	factor := u.unitFactor(unitSymbol)
	if factor == 0 {
		return value
	}
	return value * factor
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

func (u UnitInput) clamp(v float64) float64 {
	if v < u.minVal() {
		return u.minVal()
	}
	if v > u.maxVal() {
		return u.maxVal()
	}
	return v
}

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (u UnitInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumeric }

// LayoutSelf implements ui.Layouter.
func (u UnitInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := unitInputW
	if area.W < w {
		w = area.W
	}

	// Focus management.
	var focused bool
	if focus != nil && !u.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	borderColor := tokens.Colors.Stroke.Border
	if u.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	// Border and fill.
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
	fillColor := tokens.Colors.Surface.Elevated
	if u.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	if focused {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Value text.
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	textColor := tokens.Colors.Text.Primary
	if u.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}
	displayVal := u.DisplayValue()
	displayText := fmt.Sprintf("%.2f", displayVal)
	canvas.DrawText(displayText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Unit dropdown button.
	unitBtnX := area.X + w - unitDropdownW
	unitBtnRect := draw.R(float32(unitBtnX), float32(area.Y), float32(unitDropdownW), float32(h))

	// Divider.
	canvas.FillRect(
		draw.R(float32(unitBtnX), float32(area.Y+1), 1, float32(max(h-2, 0))),
		draw.SolidPaint(borderColor))

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
	unitText := u.Unit + " "
	arrowStyle := tokens.Typography.LabelSmall
	arrowStyle.FontFamily = "Phosphor"
	unitColor := tokens.Colors.Text.Secondary
	if u.Disabled {
		unitColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(unitText, draw.Pt(float32(unitBtnX+4), float32(textY)), style, unitColor)
	arrowX := float32(area.X+w-textFieldPadX) - arrowStyle.Size
	canvas.DrawText(icons.CaretDown, draw.Pt(arrowX, float32(textY)), arrowStyle, unitColor)

	// +/- step buttons via drag on value area.
	valueRect := draw.R(float32(area.X), float32(area.Y), float32(max(w-unitDropdownW, 0)), float32(h))
	if !u.Disabled && u.OnChange != nil && u.Step > 0 {
		onChange := u.OnChange
		baseVal := u.Value
		minV := u.minVal()
		maxV := u.maxVal()
		step := u.Step
		unit := u.Unit
		valueW := float32(max(w-unitDropdownW, 1))
		pressX := float32(-1)
		ix.RegisterDrag(valueRect, func(x, _ float32) {
			if pressX < 0 {
				pressX = x
				return
			}
			delta := float64((x - pressX) / valueW)
			newVal := baseVal + delta*(maxV-minV)*0.5
			if step > 0 {
				newVal = math.Round(newVal/step) * step
			}
			if newVal < minV {
				newVal = minV
			}
			if newVal > maxV {
				newVal = maxV
			}
			onChange(newVal, unit)
		})
	}

	// Dropdown overlay.
	if isOpen && u.State != nil && overlays != nil {
		state := u.State
		onChange := u.OnChange
		curUnit := u.Unit
		baseValue := u.Value
		units := u.Units
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
				// Backdrop.
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
					var itemClick func()
					if onChange != nil || state != nil {
						itemClick = func() {
							if onChange != nil {
								onChange(baseValue, sym)
							}
							state.Open = false
						}
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
						label = ud.Symbol + " — " + ud.Label
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

// TreeEqual implements ui.TreeEqualizer.
func (u UnitInput) TreeEqual(other ui.Element) bool {
	ub, ok := other.(UnitInput)
	return ok && u.Value == ub.Value && u.Unit == ub.Unit
}

// ResolveChildren implements ui.ChildResolver. UnitInput is a leaf.
func (u UnitInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return u
}

// WalkAccess implements ui.AccessWalker.
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

// Compile-time interface checks.
var _ osk.OSKRequester = UnitInput{}
