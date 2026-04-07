package form

import (
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for date input.
const (
	dateInputFieldW = 80
	dateInputSepW   = 8
)

// DateFormat selects the date display format (RFC-004 §6.8).
type DateFormat uint8

const (
	DateFormatDMY DateFormat = iota // DD.MM.YYYY
	DateFormatMDY                   // MM/DD/YYYY
	DateFormatYMD                   // YYYY-MM-DD
)

// DateInputMode selects the input method (RFC-004 §6.8).
type DateInputMode uint8

const (
	DateModeNumeric  DateInputMode = iota // Three NumericInputs (Day | Month | Year)
	DateModeCalendar                      // Calendar popup (not yet implemented)
	DateModeDirect                        // Direct numeric entry
)

// DateModeDrum is kept as an alias for backward compatibility.
const DateModeDrum = DateModeNumeric

// DateInput is an HMI-optimized date entry widget (RFC-004 §6.8).
// Composes NumericInput children via ctx.LayoutChild() — each column
// (day, month, year) is a NumericInput with appropriate bounds.
// In touch mode the NumericInputs automatically use large increment/decrement
// buttons above and below the value.
type DateInput struct {
	ui.BaseElement
	Value    time.Time
	OnChange func(time.Time)
	Format   DateFormat
	Mode     DateInputMode
	Min      *time.Time
	Max      *time.Time
	Disabled bool
}

// DateInputOption configures a DateInput element.
type DateInputOption func(*DateInput)

func WithDateFormat(f DateFormat) DateInputOption {
	return func(d *DateInput) { d.Format = f }
}

func WithDateMode(m DateInputMode) DateInputOption {
	return func(d *DateInput) { d.Mode = m }
}

func WithOnDateInputChange(fn func(time.Time)) DateInputOption {
	return func(d *DateInput) { d.OnChange = fn }
}

func WithDateRange(min, max time.Time) DateInputOption {
	return func(d *DateInput) { d.Min = &min; d.Max = &max }
}

func WithDateInputDisabled() DateInputOption {
	return func(d *DateInput) { d.Disabled = true }
}

func NewDateInput(value time.Time, opts ...DateInputOption) ui.Element {
	el := DateInput{Value: value}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// DaysInMonth returns the number of days in the given month/year.
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// MonthNames returns month names (1-indexed, index 0 is empty).
var MonthNames = [13]string{
	"", "Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

func (d DateInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumericInteger }

func (d DateInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if d.Mode == DateModeNumeric || d.Mode == 0 {
		return d.layoutNumeric(ctx)
	}
	return d.layoutSimple(ctx)
}

func (d DateInput) layoutNumeric(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens

	style := tokens.Typography.Body
	totalW := 3*dateInputFieldW + 2*dateInputSepW
	if area.W < totalW {
		totalW = area.W
	}

	year := d.Value.Year()
	month := int(d.Value.Month())
	day := d.Value.Day()
	onChange := d.OnChange
	val := d.Value

	daysInMonth := DaysInMonth(year, time.Month(month))

	sepColor := tokens.Colors.Text.Secondary
	if d.Disabled {
		sepColor = tokens.Colors.Text.Disabled
	}

	// Column order depends on format.
	type column struct {
		value float64
		min   float64
		max   float64
		step  float64
		wrap  bool
		fn    func(float64)
	}

	dayCol := column{
		value: float64(day), min: 1, max: float64(daysInMonth), step: 1, wrap: true,
		fn: func(v float64) {
			if onChange != nil {
				newT := time.Date(year, time.Month(month), int(v), 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		},
	}
	monthCol := column{
		value: float64(month), min: 1, max: 12, step: 1, wrap: true,
		fn: func(v float64) {
			if onChange != nil {
				newMonth := int(v)
				newDays := DaysInMonth(year, time.Month(newMonth))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(year, time.Month(newMonth), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		},
	}
	minYear := year - 50
	maxYear := year + 50
	yearCol := column{
		value: float64(year), min: float64(minYear), max: float64(maxYear), step: 1, wrap: false,
		fn: func(v float64) {
			if onChange != nil {
				newYear := int(v)
				newDays := DaysInMonth(newYear, time.Month(month))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(newYear, time.Month(month), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		},
	}

	var cols [3]column
	var seps [2]string
	switch d.Format {
	case DateFormatMDY:
		cols = [3]column{monthCol, dayCol, yearCol}
		seps = [2]string{"/", "/"}
	case DateFormatYMD:
		cols = [3]column{yearCol, monthCol, dayCol}
		seps = [2]string{"-", "-"}
	default: // DMY
		cols = [3]column{dayCol, monthCol, yearCol}
		seps = [2]string{".", "."}
	}

	x := area.X
	var firstH int
	for i, col := range cols {
		var opts []NumericInputOption
		opts = append(opts,
			WithNumericRange(col.min, col.max),
			WithNumericStep(col.step),
			WithNumericKind(NumericInteger),
		)
		if col.wrap {
			opts = append(opts, WithWrapping())
		}
		if d.Disabled {
			opts = append(opts, WithNumericDisabled())
		}
		if col.fn != nil {
			fn := col.fn
			opts = append(opts, WithOnNumericChange(fn))
		}

		el := NewNumericInput(col.value, opts...)
		cb := ctx.LayoutChild(el, ui.Bounds{X: x, Y: area.Y, W: dateInputFieldW, H: area.H})
		if i == 0 {
			firstH = cb.H
		}
		x += dateInputFieldW

		// Separator between columns.
		if i < 2 {
			sm := canvas.MeasureText(seps[i], style)
			sepY := float32(area.Y) + float32(cb.H)/2 - style.Size/2
			canvas.DrawText(seps[i], draw.Pt(float32(x)+float32(dateInputSepW)/2-sm.Width/2, sepY), style, sepColor)
			x += dateInputSepW
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: firstH}
}

func (d DateInput) layoutSimple(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2
	w := datePickerW
	if area.W < w {
		w = area.W
	}

	var focused bool
	if focus != nil && !d.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	borderColor := tokens.Colors.Stroke.Border
	if d.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
	fillColor := tokens.Colors.Surface.Elevated
	if d.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	var dateText string
	switch d.Format {
	case DateFormatDMY:
		dateText = d.Value.Format("02.01.2006")
	case DateFormatMDY:
		dateText = d.Value.Format("01/02/2006")
	default:
		dateText = d.Value.Format("2006-01-02")
	}
	textColor := tokens.Colors.Text.Primary
	if d.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(dateText, draw.Pt(float32(area.X+textFieldPadX), float32(area.Y+textFieldPadY)), style, textColor)

	if !d.Disabled {
		ix.RegisterHit(fieldRect, nil)
	}
	if focused {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

func (d DateInput) TreeEqual(other ui.Element) bool {
	db, ok := other.(DateInput)
	return ok && d.Value.Equal(db.Value) && d.Format == db.Format && d.Mode == db.Mode
}

func (d DateInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return d
}

func (d DateInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  d.Value.Format("2006-01-02"),
		Label:  "Date",
		States: a11y.AccessStates{Disabled: d.Disabled},
	}, parentIdx, a11y.Rect{})
}

var _ osk.OSKRequester = DateInput{}
