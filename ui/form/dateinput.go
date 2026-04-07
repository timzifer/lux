package form

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/osk"
)

// Layout constants for date input.
const (
	dateInputDrumW = 80
	dateInputSepW  = 8
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
	DateModeDrum     DateInputMode = iota // DrumPickers (Day | Month | Year)
	DateModeCalendar                      // Calendar popup
	DateModeDirect                        // Direct numeric entry
)

// DateInput is an HMI-optimized date entry widget (RFC-004 §6.8).
// Composes DrumPicker children via ctx.LayoutChild().
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
	if d.Mode == DateModeDrum || d.Mode == 0 {
		return d.layoutDrum(ctx)
	}
	return d.layoutSimple(ctx)
}

func (d DateInput) layoutDrum(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	vis := drumDefaultVisible
	totalW := 3*dateInputDrumW + 2*dateInputSepW
	if area.W < totalW {
		totalW = area.W
	}
	totalH := vis * drumItemH

	year := d.Value.Year()
	month := int(d.Value.Month())
	day := d.Value.Day()
	onChange := d.OnChange
	val := d.Value

	// ── Day DrumPicker ──
	daysInMonth := DaysInMonth(year, time.Month(month))
	dayItems := IntItems(1, daysInMonth)
	dayDrum := NewDrumPicker(dayItems, day-1, // 0-indexed
		WithDrumPickerVisible(vis),
		WithOnDrumSelect(func(idx int) {
			if onChange != nil && idx < len(dayItems) {
				newDay := dayItems[idx].Value.(int)
				newT := time.Date(year, time.Month(month), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}),
	)
	if d.Disabled {
		dayDrum = NewDrumPicker(dayItems, day-1, WithDrumPickerVisible(vis), WithDrumDisabled())
	}
	ctx.LayoutChild(dayDrum, ui.Bounds{X: area.X, Y: area.Y, W: dateInputDrumW, H: totalH})

	// ── Month DrumPicker ──
	var monthItems []DrumItem
	for m := 1; m <= 12; m++ {
		monthItems = append(monthItems, DrumItem{Label: MonthNames[m], Value: m})
	}
	monthX := area.X + dateInputDrumW + dateInputSepW
	monthDrum := NewDrumPicker(monthItems, month-1, // 0-indexed
		WithDrumPickerVisible(vis),
		WithOnDrumSelect(func(idx int) {
			if onChange != nil && idx < len(monthItems) {
				newMonth := monthItems[idx].Value.(int)
				newDays := DaysInMonth(year, time.Month(newMonth))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(year, time.Month(newMonth), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}),
	)
	if d.Disabled {
		monthDrum = NewDrumPicker(monthItems, month-1, WithDrumPickerVisible(vis), WithDrumDisabled())
	}
	ctx.LayoutChild(monthDrum, ui.Bounds{X: monthX, Y: area.Y, W: dateInputDrumW, H: totalH})

	// ── Year DrumPicker ──
	minYear := year - 10
	maxYear := year + 10
	yearItems := IntItems(minYear, maxYear)
	yearIdx := year - minYear
	yearX := monthX + dateInputDrumW + dateInputSepW
	yearDrum := NewDrumPicker(yearItems, yearIdx,
		WithDrumPickerVisible(vis),
		WithOnDrumSelect(func(idx int) {
			if onChange != nil && idx < len(yearItems) {
				newYear := yearItems[idx].Value.(int)
				newDays := DaysInMonth(newYear, time.Month(month))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(newYear, time.Month(month), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}),
	)
	if d.Disabled {
		yearDrum = NewDrumPicker(yearItems, yearIdx, WithDrumPickerVisible(vis), WithDrumDisabled())
	}
	ctx.LayoutChild(yearDrum, ui.Bounds{X: yearX, Y: area.Y, W: dateInputDrumW, H: totalH})

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
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
