package form

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for numeric input.
const (
	numericInputW        = 200
	numericInputPadX     = 8
	numericInputPadY     = 8
	numericStepperW      = 24
	numericStepperBorder = 1
)

// NumericKind distinguishes integer from floating-point input (RFC-004 §6.2).
type NumericKind uint8

const (
	NumericInteger NumericKind = iota // Integer input
	NumericFloat                      // Floating-point input
)

// ClampBehavior controls when values are clamped to [Min, Max] (RFC-004 §6.2).
type ClampBehavior uint8

const (
	// ClampOnCommit clamps the value when the field loses focus (blur).
	ClampOnCommit ClampBehavior = iota

	// ClampOnInput rejects any input that would place the value outside [Min, Max].
	ClampOnInput

	// ClampOnStep clamps only +/- button increments; direct entry may exceed bounds
	// (shown with error styling).
	ClampOnStep
)

// NumericInput is a number input with stepper buttons and optional unit suffix.
// Implements RFC-004 §6.2.
type NumericInput struct {
	ui.BaseElement

	// Value is the current numeric value.
	Value float64

	// Kind selects integer or floating-point mode. Affects validation and OSK layout.
	Kind NumericKind

	// Min, Max define the value range. nil = unbounded.
	Min *float64
	Max *float64

	// Step is the increment/decrement size for +/- buttons.
	// 0 means no +/- buttons (direct entry only).
	Step float64

	// Precision is the number of decimal places for float display.
	// Only relevant when Kind == NumericFloat.
	Precision int

	// Unit is an optional suffix displayed after the value (e.g. "mm", "°C").
	Unit string

	// Placeholder is shown when Value == 0 and the field has no focus.
	Placeholder string

	// Label is displayed above the input field.
	Label string

	// OnChange is called when the value changes.
	OnChange func(float64)

	// Disabled disables the widget.
	Disabled bool

	// Clamping controls when bounds are enforced.
	Clamping ClampBehavior

	// Wrapping enables cyclic value behavior: incrementing past Max wraps to Min,
	// decrementing below Min wraps to Max. Used by TimeInput (hours 23→0) and
	// DateInput (months 12→1).
	Wrapping bool
}

// NumericInputOption configures a NumericInput element.
type NumericInputOption func(*NumericInput)

// WithNumericRange sets the min and max bounds.
func WithNumericRange(min, max float64) NumericInputOption {
	return func(n *NumericInput) {
		n.Min = &min
		n.Max = &max
	}
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

// WithNumericKind sets integer or float mode.
func WithNumericKind(kind NumericKind) NumericInputOption {
	return func(n *NumericInput) { n.Kind = kind }
}

// WithPrecision sets the decimal places for float display.
func WithPrecision(p int) NumericInputOption {
	return func(n *NumericInput) { n.Precision = p }
}

// WithPlaceholder sets the placeholder text.
func WithPlaceholder(s string) NumericInputOption {
	return func(n *NumericInput) { n.Placeholder = s }
}

// WithNumericLabel sets the label displayed above the field.
func WithNumericLabel(s string) NumericInputOption {
	return func(n *NumericInput) { n.Label = s }
}

// WithClamping sets the clamping behavior.
func WithClamping(c ClampBehavior) NumericInputOption {
	return func(n *NumericInput) { n.Clamping = c }
}

// WithWrapping enables cyclic value wrapping (Max→Min on increment, Min→Max on decrement).
func WithWrapping() NumericInputOption {
	return func(n *NumericInput) { n.Wrapping = true }
}

// NewNumericInput creates a numeric input element.
// Defaults: Min=0, Max=100, Step=1.
func NewNumericInput(value float64, opts ...NumericInputOption) ui.Element {
	min, max := 0.0, 100.0
	el := NumericInput{Value: value, Min: &min, Max: &max, Step: 1}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// NumericInputDisabled creates a disabled numeric input.
func NumericInputDisabled(value float64, opts ...NumericInputOption) ui.Element {
	min, max := 0.0, 100.0
	el := NumericInput{Value: value, Min: &min, Max: &max, Step: 1, Disabled: true}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// minVal returns the effective minimum, or -Inf if unbounded.
func (n NumericInput) minVal() float64 {
	if n.Min != nil {
		return *n.Min
	}
	return math.Inf(-1)
}

// maxVal returns the effective maximum, or +Inf if unbounded.
func (n NumericInput) maxVal() float64 {
	if n.Max != nil {
		return *n.Max
	}
	return math.Inf(1)
}

func (n NumericInput) clamp(v float64) float64 {
	lo, hi := n.minVal(), n.maxVal()
	if n.Wrapping && n.Min != nil && n.Max != nil {
		if v > hi {
			return lo
		}
		if v < lo {
			return hi
		}
		return v
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// snapToStep rounds v to the nearest multiple of Step relative to Min,
// eliminating floating-point drift from repeated increment/decrement.
func (n NumericInput) snapToStep(v float64) float64 {
	if n.Step <= 0 {
		return v
	}
	base := 0.0
	if n.Min != nil {
		base = *n.Min
	}
	return math.Round((v-base)/n.Step)*n.Step + base
}

// IsValidChar checks whether a character is allowed in the current input context (RFC-004 §6.2).
func (n NumericInput) IsValidChar(ch rune, buffer string, cursorPos int) bool {
	switch {
	case ch >= '0' && ch <= '9':
		return true
	case ch == '-' || ch == '+':
		return cursorPos == 0
	case ch == '.' || ch == ',':
		return n.Kind == NumericFloat && !strings.ContainsAny(buffer, ".,")
	default:
		return false
	}
}

// formatValue formats the current value for display.
func (n NumericInput) formatValue() string {
	if n.Kind == NumericFloat && n.Precision > 0 {
		return fmt.Sprintf("%.*f", n.Precision, n.Value)
	}
	if n.Step == math.Trunc(n.Step) && n.Step >= 1 {
		return fmt.Sprintf("%.0f", n.Value)
	}
	return fmt.Sprintf("%.2f", n.Value)
}

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (n NumericInput) OSKLayout() osk.OSKLayout {
	if n.Kind == NumericInteger {
		return osk.OSKLayoutNumericInteger
	}
	return osk.OSKLayoutNumeric
}

// LayoutSelf implements ui.Layouter.
func (n NumericInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if ctx.IsTouch() && n.Step > 0 {
		return n.layoutTouch(ctx)
	}
	return n.layoutDesktop(ctx)
}

// layoutDesktop renders the classic layout: text field with narrow stepper column on the right.
func (n NumericInput) layoutDesktop(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	labelStyle := tokens.Typography.Label
	h := int(style.Size) + numericInputPadY*2
	y := area.Y

	w := numericInputW
	if area.W < w {
		w = area.W
	}

	// Label above the field.
	if n.Label != "" {
		labelColor := tokens.Colors.Text.Secondary
		if n.Disabled {
			labelColor = tokens.Colors.Text.Disabled
		}
		canvas.DrawText(n.Label, draw.Pt(float32(area.X), float32(y)), labelStyle, labelColor)
		m := canvas.MeasureText(n.Label, labelStyle)
		y += int(math.Ceil(float64(m.Ascent))) + 4
	}

	// Focus management.
	var focused bool
	var focusUID ui.UID
	if focus != nil && !n.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(focusUID)
	}

	fieldRect := draw.R(float32(area.X), float32(y), float32(w), float32(h))

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
		draw.R(float32(area.X+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Focus glow.
	if focused {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Value text + unit.
	textX := area.X + numericInputPadX
	textY := y + numericInputPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}

	displayText := n.formatValue()
	if n.Value == 0 && n.Placeholder != "" && !focused {
		displayText = n.Placeholder
		textColor = tokens.Colors.Text.Secondary
	} else if n.Unit != "" {
		displayText += " " + n.Unit
	}
	canvas.DrawText(displayText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Stepper buttons (only if Step > 0).
	if n.Step > 0 {
		// Stepper divider line.
		stepperX := area.X + w - numericStepperW
		canvas.FillRect(
			draw.R(float32(stepperX), float32(y+1), 1, float32(max(h-2, 0))),
			draw.SolidPaint(borderColor))

		// Up button (top half of stepper area).
		halfH := h / 2
		upRect := draw.R(float32(stepperX), float32(y), float32(numericStepperW), float32(halfH))
		var upHover float32
		if n.Disabled {
			ix.RegisterHit(upRect, nil)
		} else {
			var upFn func()
			if n.OnChange != nil {
				onChange := n.OnChange
				val := n.clamp(n.snapToStep(n.Value + n.Step))
				upFn = func() { onChange(val) }
			}
			upHover = ix.RegisterHit(upRect, upFn)
		}
		if upHover > 0 {
			canvas.FillRect(upRect, draw.SolidPaint(draw.Color{R: 0, G: 0, B: 0, A: upHover * 0.08}))
		}

		// Down button (bottom half of stepper area).
		downRect := draw.R(float32(stepperX), float32(y+halfH), float32(numericStepperW), float32(h-halfH))
		var downHover float32
		if n.Disabled {
			ix.RegisterHit(downRect, nil)
		} else {
			var downFn func()
			if n.OnChange != nil {
				onChange := n.OnChange
				val := n.clamp(n.snapToStep(n.Value - n.Step))
				downFn = func() { onChange(val) }
			}
			downHover = ix.RegisterHit(downRect, downFn)
		}
		if downHover > 0 {
			canvas.FillRect(downRect, draw.SolidPaint(draw.Color{R: 0, G: 0, B: 0, A: downHover * 0.08}))
		}

		// Stepper mid-divider.
		canvas.FillRect(
			draw.R(float32(stepperX), float32(y+halfH), float32(numericStepperW), 1),
			draw.SolidPaint(borderColor))

		// Up/Down arrow icons.
		arrowStyle := tokens.Typography.LabelSmall
		arrowStyle.FontFamily = "Phosphor"
		arrowColor := tokens.Colors.Text.Secondary
		if n.Disabled {
			arrowColor = tokens.Colors.Text.Disabled
		}
		arrowCenterX := float32(stepperX) + float32(numericStepperW)/2 - arrowStyle.Size/2
		upArrowY := float32(y) + float32(halfH)/2 - arrowStyle.Size/2
		canvas.DrawText(icons.CaretUp, draw.Pt(arrowCenterX, upArrowY), arrowStyle, arrowColor)
		downArrowY := float32(y+halfH) + float32(h-halfH)/2 - arrowStyle.Size/2
		canvas.DrawText(icons.CaretDown, draw.Pt(arrowCenterX, downArrowY), arrowStyle, arrowColor)
	}

	// Value area for direct text entry.
	stepperW := 0
	if n.Step > 0 {
		stepperW = numericStepperW
	}
	valueRect := draw.R(float32(area.X), float32(y), float32(max(w-stepperW, 0)), float32(h))

	n.setupInputState(focus, focusUID, focused, valueRect, ix)

	totalH := (y - area.Y) + h
	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: totalH}
}

// layoutTouch renders the touch-optimized layout: a compact text field that,
// when focused, opens an inline NumericKeypad overlay positioned below (or above)
// the field. The old full-width +/- buttons are replaced by ▲/▼ keys inside
// the keypad.
func (n NumericInput) layoutTouch(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus
	overlays := ctx.Overlays

	style := tokens.Typography.Body
	labelStyle := tokens.Typography.Label
	h := int(style.Size) + numericInputPadY*2
	y := area.Y

	w := numericInputW
	if area.W < w {
		w = area.W
	}

	// Label above the field.
	if n.Label != "" {
		labelColor := tokens.Colors.Text.Secondary
		if n.Disabled {
			labelColor = tokens.Colors.Text.Disabled
		}
		canvas.DrawText(n.Label, draw.Pt(float32(area.X), float32(y)), labelStyle, labelColor)
		m := canvas.MeasureText(n.Label, labelStyle)
		y += int(math.Ceil(float64(m.Ascent))) + 4
	}

	// Focus management.
	var focused bool
	var focusUID ui.UID
	if focus != nil && !n.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(focusUID)
	}

	fieldRect := draw.R(float32(area.X), float32(y), float32(w), float32(h))

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
		draw.R(float32(area.X+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Focus glow.
	if focused {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Value text + unit.
	textX := area.X + numericInputPadX
	textY := y + numericInputPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}

	displayText := n.formatValue()
	if n.Value == 0 && n.Placeholder != "" && !focused {
		displayText = n.Placeholder
		textColor = tokens.Colors.Text.Secondary
	} else if n.Unit != "" {
		displayText += " " + n.Unit
	}
	canvas.DrawText(displayText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Draw caret when focused.
	if focused && focus != nil {
		cursorOff := len(n.formatValue())
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			if cursorOff > len(n.formatValue()) {
				cursorOff = len(n.formatValue())
			}
		}
		metrics := canvas.MeasureText(n.formatValue()[:cursorOff], style)
		cursorX := float32(textX) + metrics.Width
		canvas.FillRect(draw.R(cursorX, float32(textY), 2, style.Size),
			draw.SolidPaint(tokens.Colors.Text.Primary))
	}

	// Register InputState, hit target, and focus bounds via shared helper.
	n.setupInputState(focus, focusUID, focused, fieldRect, ix)

	// Suppress the global OSK — this widget provides its own keypad overlay.
	if focused && focus != nil && focus.Input != nil {
		focus.Input.SuppressOSK = true
	}

	// ── Keypad overlay (shown when focused) ──
	if focused && !n.Disabled && overlays != nil {
		kpState := getKeypadState(focusUID)
		if !kpState.Open {
			kpState.Open = true
			kpState.OriginalValue = n.Value
		}

		anchor := fieldRect
		onChange := n.OnChange
		kind := n.Kind
		step := n.Step
		minP := n.Min
		maxP := n.Max
		prec := n.Precision
		winW := overlays.WindowW
		winH := overlays.WindowH
		// Capture the InputState pointer so the keypad can write directly into it.
		inputState := focus.Input

		focusMgr := focus
		fuid := focusUID

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				renderNumericKeypad(numericKeypadConfig{
					State:    kpState,
					Input:    inputState,
					Focus:    focusMgr,
					FocusUID: fuid,
					Kind:     kind,
					Step:     step,
					Min:      minP,
					Max:      maxP,
					Precision: prec,
					OnSubmit: func(v float64) {
						kpState.Open = false
						if onChange != nil {
							onChange(v)
						}
					},
				}, anchor, canvas, tokens, ix, winW, winH)
			},
		})
	}

	totalH := (y - area.Y) + h
	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: totalH}
}

// setupInputState connects keyboard/OSK input for direct numeric entry and
// registers the focus hit target.
func (n NumericInput) setupInputState(focus *ui.FocusManager, focusUID ui.UID, focused bool, valueRect draw.Rect, ix *ui.Interactor) {
	// InputState: connect keyboard/OSK input for direct numeric entry.
	if focused && focus != nil && n.OnChange != nil {
		displayStr := n.formatValue()
		cursorOff := len(displayStr)
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			if cursorOff > len(displayStr) {
				cursorOff = len(displayStr)
			}
		}
		onChange := n.OnChange
		kind := n.Kind
		minV := n.minVal()
		maxV := n.maxVal()
		focus.Input = &ui.InputState{
			Value: displayStr,
			OnChange: func(newVal string) {
				filtered := filterNumericChars(newVal, kind)
				v, err := strconv.ParseFloat(filtered, 64)
				if err != nil {
					return
				}
				if v < minV {
					v = minV
				}
				if v > maxV {
					v = maxV
				}
				onChange(v)
			},
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: -1,
		}
	}

	// Hit target for focus acquisition on value area.
	if focus != nil && !n.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(valueRect, func() { fm.SetFocusedUID(uid) })
	}

	// Store focused bounds for scroll-into-view (OSK visibility).
	if focused && focus != nil {
		r := valueRect
		focus.FocusedBounds = &r
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (n NumericInput) TreeEqual(other ui.Element) bool {
	nb, ok := other.(NumericInput)
	if !ok {
		return false
	}
	return n.Value == nb.Value && n.Unit == nb.Unit &&
		ptrF64Eq(n.Min, nb.Min) && ptrF64Eq(n.Max, nb.Max) &&
		n.Kind == nb.Kind && n.Label == nb.Label
}

func ptrF64Eq(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
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
			Min:     n.minVal(),
			Max:     n.maxVal(),
			Step:    n.Step,
		},
	}
	if n.Label != "" {
		an.Label = n.Label
	} else if n.Unit != "" {
		an.Label = n.Unit
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

// filterNumericChars filters a string to only valid numeric characters.
func filterNumericChars(s string, kind NumericKind) string {
	var b strings.Builder
	hasDot := false
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case (r == '-' || r == '+') && i == 0:
			b.WriteRune(r)
		case (r == '.' || r == ',') && kind == NumericFloat && !hasDot:
			b.WriteRune('.')
			hasDot = true
		}
	}
	return b.String()
}

// Compile-time interface checks.
var (
	_ osk.OSKRequester = NumericInput{}
	_ ui.Layouter      = NumericInput{}
)
