package form

import (
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for time input.
const (
	timeInputFieldW = 80
	timeInputSepW   = 16
)

// TimeFormat selects the time display format (RFC-004 §6.8).
type TimeFormat uint8

const (
	TimeFormatHHMM   TimeFormat = iota // 14:30
	TimeFormatHHMMSS                   // 14:30:45
	TimeFormat12h                      // 2:30 PM
)

// TimeInput is an HMI-optimized time entry widget (RFC-004 §6.8).
// Composes NumericInput children via ctx.LayoutChild() — each column
// (hour, minute, optional second) is a NumericInput with Wrapping enabled.
// In touch mode the NumericInputs automatically use large increment/decrement
// buttons above and below the value (see NumericInput.layoutTouch).
type TimeInput struct {
	ui.BaseElement
	Value      time.Time
	Format     TimeFormat
	OnChange   func(time.Time)
	MinuteStep int
	Disabled   bool
}

// TimeInputOption configures a TimeInput element.
type TimeInputOption func(*TimeInput)

func WithTimeFormat(f TimeFormat) TimeInputOption {
	return func(t *TimeInput) { t.Format = f }
}

func WithOnTimeInputChange(fn func(time.Time)) TimeInputOption {
	return func(t *TimeInput) { t.OnChange = fn }
}

func WithMinuteStep(s int) TimeInputOption {
	return func(t *TimeInput) { t.MinuteStep = s }
}

func WithTimeInputDisabled() TimeInputOption {
	return func(t *TimeInput) { t.Disabled = true }
}

func NewTimeInput(value time.Time, opts ...TimeInputOption) ui.Element {
	el := TimeInput{Value: value, MinuteStep: 1}
	for _, o := range opts {
		o(&el)
	}
	if el.MinuteStep < 1 {
		el.MinuteStep = 1
	}
	return el
}

func (t TimeInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumericInteger }

func (t TimeInput) columnCount() int {
	if t.Format == TimeFormatHHMMSS {
		return 3
	}
	return 2
}

func (t TimeInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	style := tokens.Typography.Body
	cols := t.columnCount()
	totalW := cols*timeInputFieldW + (cols-1)*timeInputSepW
	if area.W < totalW {
		totalW = area.W
	}

	hour := t.Value.Hour()
	minute := t.Value.Minute()
	second := t.Value.Second()
	onChange := t.OnChange
	val := t.Value
	minuteStep := t.MinuteStep
	if minuteStep < 1 {
		minuteStep = 1
	}

	// ── Hour NumericInput ──
	hourMin, hourMax := 0.0, 23.0
	var hourOpts []NumericInputOption
	hourOpts = append(hourOpts,
		WithNumericRange(hourMin, hourMax),
		WithNumericStep(1),
		WithNumericKind(NumericInteger),
		WithWrapping(),
	)
	if t.Disabled {
		hourOpts = append(hourOpts, WithNumericDisabled())
	}
	if onChange != nil {
		hourOpts = append(hourOpts, WithOnNumericChange(func(v float64) {
			newT := time.Date(val.Year(), val.Month(), val.Day(),
				int(v), minute, second, 0, val.Location())
			onChange(newT)
		}))
	}
	hourEl := NewNumericInput(float64(hour), hourOpts...)
	hourBounds := ctx.LayoutChild(hourEl, ui.Bounds{X: area.X, Y: area.Y, W: timeInputFieldW, H: area.H})

	// ":" separator
	sepX := area.X + timeInputFieldW
	sepColor := tokens.Colors.Text.Secondary
	if t.Disabled {
		sepColor = tokens.Colors.Text.Disabled
	}
	sepY := float32(area.Y) + float32(hourBounds.H)/2 - style.Size/2
	sm := canvas.MeasureText(":", style)
	canvas.DrawText(":", draw.Pt(float32(sepX)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

	// ── Minute NumericInput ──
	minuteMin, minuteMax := 0.0, float64(59)
	var minuteOpts []NumericInputOption
	minuteOpts = append(minuteOpts,
		WithNumericRange(minuteMin, minuteMax),
		WithNumericStep(float64(minuteStep)),
		WithNumericKind(NumericInteger),
		WithWrapping(),
	)
	if t.Disabled {
		minuteOpts = append(minuteOpts, WithNumericDisabled())
	}
	if onChange != nil {
		minuteOpts = append(minuteOpts, WithOnNumericChange(func(v float64) {
			newT := time.Date(val.Year(), val.Month(), val.Day(),
				hour, int(v), second, 0, val.Location())
			onChange(newT)
		}))
	}
	minuteX := sepX + timeInputSepW
	minuteEl := NewNumericInput(float64(minute), minuteOpts...)
	ctx.LayoutChild(minuteEl, ui.Bounds{X: minuteX, Y: area.Y, W: timeInputFieldW, H: area.H})

	// ── Second NumericInput (if HH:MM:SS) ──
	if cols == 3 {
		sep2X := minuteX + timeInputFieldW
		canvas.DrawText(":", draw.Pt(float32(sep2X)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

		secondX := sep2X + timeInputSepW
		secondMin, secondMax := 0.0, 59.0
		var secondOpts []NumericInputOption
		secondOpts = append(secondOpts,
			WithNumericRange(secondMin, secondMax),
			WithNumericStep(1),
			WithNumericKind(NumericInteger),
			WithWrapping(),
		)
		if t.Disabled {
			secondOpts = append(secondOpts, WithNumericDisabled())
		}
		if onChange != nil {
			secondOpts = append(secondOpts, WithOnNumericChange(func(v float64) {
				newT := time.Date(val.Year(), val.Month(), val.Day(),
					hour, minute, int(v), 0, val.Location())
				onChange(newT)
			}))
		}
		secondEl := NewNumericInput(float64(second), secondOpts...)
		ctx.LayoutChild(secondEl, ui.Bounds{X: secondX, Y: area.Y, W: timeInputFieldW, H: area.H})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: hourBounds.H}
}

func (t TimeInput) TreeEqual(other ui.Element) bool {
	tb, ok := other.(TimeInput)
	return ok && t.Value.Equal(tb.Value) && t.Format == tb.Format
}

func (t TimeInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return t
}

func (t TimeInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	var val string
	switch t.Format {
	case TimeFormatHHMMSS:
		val = t.Value.Format("15:04:05")
	case TimeFormat12h:
		val = t.Value.Format("3:04 PM")
	default:
		val = t.Value.Format("15:04")
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  val,
		Label:  "Time",
		States: a11y.AccessStates{Disabled: t.Disabled},
	}, parentIdx, a11y.Rect{})
}

var _ osk.OSKRequester = TimeInput{}
