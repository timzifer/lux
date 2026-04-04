package form

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for time input.
const (
	timeInputDrumW   = 80
	timeInputSepW    = 16
)

// TimeFormat selects the time display format (RFC-004 §6.8).
type TimeFormat uint8

const (
	TimeFormatHHMM   TimeFormat = iota // 14:30
	TimeFormatHHMMSS                   // 14:30:45
	TimeFormat12h                      // 2:30 PM
)

// TimeInput is an HMI-optimized time entry widget using DrumPickers (RFC-004 §6.8).
// Composes DrumPicker children via ctx.LayoutChild().
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
	vis := drumDefaultVisible
	totalW := cols*timeInputDrumW + (cols-1)*timeInputSepW
	if area.W < totalW {
		totalW = area.W
	}
	totalH := vis * drumItemH

	hour := t.Value.Hour()
	minute := t.Value.Minute()
	second := t.Value.Second()
	onChange := t.OnChange
	val := t.Value

	// ── Hour DrumPicker ──
	hourItems := IntItems(0, 23)
	hourDrum := NewDrumPicker(hourItems, hour,
		WithDrumPickerVisible(vis),
		WithDrumLooping(),
		WithOnDrumSelect(func(idx int) {
			if onChange != nil {
				newT := time.Date(val.Year(), val.Month(), val.Day(),
					idx, minute, second, 0, val.Location())
				onChange(newT)
			}
		}),
	)
	if t.Disabled {
		hourDrum = NewDrumPicker(hourItems, hour,
			WithDrumPickerVisible(vis), WithDrumLooping(), WithDrumDisabled())
	}
	ctx.LayoutChild(hourDrum, ui.Bounds{X: area.X, Y: area.Y, W: timeInputDrumW, H: totalH})

	// ":" separator
	sepX := area.X + timeInputDrumW
	sepColor := tokens.Colors.Text.Secondary
	if t.Disabled {
		sepColor = tokens.Colors.Text.Disabled
	}
	sepY := float32(area.Y) + float32(totalH)/2 - style.Size/2
	sm := canvas.MeasureText(":", style)
	canvas.DrawText(":", draw.Pt(float32(sepX)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

	// ── Minute DrumPicker ──
	minuteStep := t.MinuteStep
	if minuteStep < 1 {
		minuteStep = 1
	}
	var minuteItems []DrumItem
	for m := 0; m < 60; m += minuteStep {
		minuteItems = append(minuteItems, DrumItem{Label: fmt.Sprintf("%02d", m), Value: m})
	}
	// Find the index matching the current minute.
	minuteIdx := 0
	for i, item := range minuteItems {
		if item.Value.(int) == minute {
			minuteIdx = i
			break
		}
	}
	minuteX := sepX + timeInputSepW
	minuteDrum := NewDrumPicker(minuteItems, minuteIdx,
		WithDrumPickerVisible(vis),
		WithDrumLooping(),
		WithOnDrumSelect(func(idx int) {
			if onChange != nil && idx < len(minuteItems) {
				newMin := minuteItems[idx].Value.(int)
				newT := time.Date(val.Year(), val.Month(), val.Day(),
					hour, newMin, second, 0, val.Location())
				onChange(newT)
			}
		}),
	)
	if t.Disabled {
		minuteDrum = NewDrumPicker(minuteItems, minuteIdx,
			WithDrumPickerVisible(vis), WithDrumLooping(), WithDrumDisabled())
	}
	ctx.LayoutChild(minuteDrum, ui.Bounds{X: minuteX, Y: area.Y, W: timeInputDrumW, H: totalH})

	// ── Second DrumPicker (if HH:MM:SS) ──
	if cols == 3 {
		sep2X := minuteX + timeInputDrumW
		canvas.DrawText(":", draw.Pt(float32(sep2X)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

		secondX := sep2X + timeInputSepW
		secondItems := IntItems(0, 59)
		secondDrum := NewDrumPicker(secondItems, second,
			WithDrumPickerVisible(vis),
			WithDrumLooping(),
			WithOnDrumSelect(func(idx int) {
				if onChange != nil {
					newT := time.Date(val.Year(), val.Month(), val.Day(),
						hour, minute, idx, 0, val.Location())
					onChange(newT)
				}
			}),
		)
		if t.Disabled {
			secondDrum = NewDrumPicker(secondItems, second,
				WithDrumPickerVisible(vis), WithDrumLooping(), WithDrumDisabled())
		}
		ctx.LayoutChild(secondDrum, ui.Bounds{X: secondX, Y: area.Y, W: timeInputDrumW, H: totalH})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
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
