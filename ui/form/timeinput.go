package form

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for time input.
const (
	timeInputDrumW   = 80
	timeInputDrumGap = 8
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
type TimeInput struct {
	ui.BaseElement

	// Value is the current time.
	Value time.Time

	// Format selects HH:MM, HH:MM:SS, or 12h display.
	Format TimeFormat

	// OnChange is called when the time changes.
	OnChange func(time.Time)

	// MinuteStep is the step for minutes in the DrumPicker (1, 5, 15).
	MinuteStep int

	Disabled bool
}

// TimeInputOption configures a TimeInput element.
type TimeInputOption func(*TimeInput)

// WithTimeFormat sets the display format.
func WithTimeFormat(f TimeFormat) TimeInputOption {
	return func(t *TimeInput) { t.Format = f }
}

// WithOnTimeInputChange sets the change callback.
func WithOnTimeInputChange(fn func(time.Time)) TimeInputOption {
	return func(t *TimeInput) { t.OnChange = fn }
}

// WithMinuteStep sets the minute step (e.g. 5 for 5-minute intervals).
func WithMinuteStep(s int) TimeInputOption {
	return func(t *TimeInput) { t.MinuteStep = s }
}

// WithTimeInputDisabled disables the widget.
func WithTimeInputDisabled() TimeInputOption {
	return func(t *TimeInput) { t.Disabled = true }
}

// NewTimeInput creates an HMI time input.
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

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
func (t TimeInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumericInteger }

func (t TimeInput) columnCount() int {
	if t.Format == TimeFormatHHMMSS {
		return 3
	}
	return 2
}

// LayoutSelf implements ui.Layouter.
func (t TimeInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	cols := t.columnCount()
	vis := drumDefaultVisible

	totalW := cols*timeInputDrumW + (cols-1)*timeInputSepW
	if area.W < totalW {
		totalW = area.W
	}
	totalH := vis * drumItemH

	// Focus management.
	var focused bool
	if focus != nil && !t.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	hour := t.Value.Hour()
	minute := t.Value.Minute()
	second := t.Value.Second()

	borderColor := tokens.Colors.Stroke.Border
	if t.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	// Hour drum.
	t.drawDrumColumn(ctx, area.X, area.Y, timeInputDrumW, totalH, vis,
		hour, 0, 23, 1, true, func(v int) {
			if t.OnChange != nil {
				newT := time.Date(t.Value.Year(), t.Value.Month(), t.Value.Day(),
					v, minute, second, 0, t.Value.Location())
				t.OnChange(newT)
			}
		}, canvas, tokens, ix, style, borderColor)

	// ":" separator.
	sepX := area.X + timeInputDrumW
	sepColor := tokens.Colors.Text.Secondary
	if t.Disabled {
		sepColor = tokens.Colors.Text.Disabled
	}
	sepY := float32(area.Y) + float32(totalH)/2 - style.Size/2
	sm := canvas.MeasureText(":", style)
	canvas.DrawText(":", draw.Pt(float32(sepX)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

	// Minute drum.
	minuteX := sepX + timeInputSepW
	minuteStep := t.MinuteStep
	if minuteStep < 1 {
		minuteStep = 1
	}
	t.drawDrumColumn(ctx, minuteX, area.Y, timeInputDrumW, totalH, vis,
		minute, 0, 59, minuteStep, true, func(v int) {
			if t.OnChange != nil {
				newT := time.Date(t.Value.Year(), t.Value.Month(), t.Value.Day(),
					hour, v, second, 0, t.Value.Location())
				t.OnChange(newT)
			}
		}, canvas, tokens, ix, style, borderColor)

	// Second drum (if HH:MM:SS).
	if cols == 3 {
		sep2X := minuteX + timeInputDrumW
		canvas.DrawText(":", draw.Pt(float32(sep2X)+float32(timeInputSepW)/2-sm.Width/2, sepY), style, sepColor)

		secondX := sep2X + timeInputSepW
		t.drawDrumColumn(ctx, secondX, area.Y, timeInputDrumW, totalH, vis,
			second, 0, 59, 1, true, func(v int) {
				if t.OnChange != nil {
					newT := time.Date(t.Value.Year(), t.Value.Month(), t.Value.Day(),
						hour, minute, v, 0, t.Value.Location())
					t.OnChange(newT)
				}
			}, canvas, tokens, ix, style, borderColor)
	}

	if focused {
		outerRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// drawDrumColumn renders a single drum picker column inline.
func (t TimeInput) drawDrumColumn(_ *ui.LayoutContext, x, y, w, h, vis int,
	selected, minVal, maxVal, step int, looping bool,
	onSelect func(int),
	canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor,
	style draw.TextStyle, borderColor draw.Color) {

	outerRect := draw.R(float32(x), float32(y), float32(w), float32(h))
	fillColor := tokens.Colors.Surface.Elevated
	if t.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(outerRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
	canvas.FillRoundRect(
		draw.R(float32(x+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Highlight band.
	half := vis / 2
	selBandY := y + half*drumItemH
	canvas.FillRect(
		draw.R(float32(x+1), float32(selBandY), float32(max(w-2, 0)), float32(drumItemH)),
		draw.SolidPaint(draw.Color{
			R: tokens.Colors.Accent.Primary.R,
			G: tokens.Colors.Accent.Primary.G,
			B: tokens.Colors.Accent.Primary.B,
			A: 0.15,
		}))

	// Build value list.
	values := make([]int, 0)
	for v := minVal; v <= maxVal; v += step {
		values = append(values, v)
	}
	n := len(values)
	if n == 0 {
		return
	}

	// Find current index.
	selIdx := 0
	for i, v := range values {
		if v == selected {
			selIdx = i
			break
		}
	}

	for i := 0; i < vis; i++ {
		idx := selIdx - half + i
		var val int
		var valid bool
		if looping {
			wrapped := ((idx % n) + n) % n
			val = values[wrapped]
			valid = true
		} else if idx >= 0 && idx < n {
			val = values[idx]
			valid = true
		}
		if !valid {
			continue
		}

		itemY := y + i*drumItemH
		itemRect := draw.R(float32(x), float32(itemY), float32(w), float32(drumItemH))

		if !t.Disabled && onSelect != nil {
			v := val
			fn := onSelect
			ho := ix.RegisterHit(itemRect, func() { fn(v) })
			if ho > 0 && i != half {
				canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{A: ho * 0.05}))
			}
		}

		textColor := tokens.Colors.Text.Primary
		if t.Disabled {
			textColor = tokens.Colors.Text.Disabled
		} else if i != half {
			textColor = tokens.Colors.Text.Secondary
		}

		label := fmt.Sprintf("%02d", val)
		m := canvas.MeasureText(label, style)
		textX := float32(x) + float32(w)/2 - m.Width/2
		textY := float32(itemY) + float32(drumItemH)/2 - style.Size/2
		canvas.DrawText(label, draw.Pt(textX, textY), style, textColor)
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (t TimeInput) TreeEqual(other ui.Element) bool {
	tb, ok := other.(TimeInput)
	return ok && t.Value.Equal(tb.Value) && t.Format == tb.Format
}

// ResolveChildren implements ui.ChildResolver. TimeInput is a leaf.
func (t TimeInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return t
}

// WalkAccess implements ui.AccessWalker.
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

// Compile-time interface checks.
var _ osk.OSKRequester = TimeInput{}
