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

// Layout constants for date input.
const (
	dateInputDrumW   = 80
	dateInputSepW    = 8
)

// DateFormat selects the date display format (RFC-004 §6.8).
type DateFormat uint8

const (
	DateFormatDMY  DateFormat = iota // DD.MM.YYYY
	DateFormatMDY                    // MM/DD/YYYY
	DateFormatYMD                    // YYYY-MM-DD
)

// DateInputMode selects the input method (RFC-004 §6.8).
type DateInputMode uint8

const (
	// DateModeDrum uses DrumPickers (Day | Month | Year). Ideal for Touch/HMI.
	DateModeDrum DateInputMode = iota

	// DateModeCalendar uses a calendar popup. Better for Desktop.
	DateModeCalendar

	// DateModeDirect uses direct numeric entry with mask.
	DateModeDirect
)

// DateInput is an HMI-optimized date entry widget (RFC-004 §6.8).
type DateInput struct {
	ui.BaseElement

	// Value is the current date.
	Value time.Time

	// OnChange is called when the date changes.
	OnChange func(time.Time)

	// Format selects the display format.
	Format DateFormat

	// Mode selects the input method.
	Mode DateInputMode

	// Min, Max restrict the selectable range.
	Min *time.Time
	Max *time.Time

	Disabled bool
}

// DateInputOption configures a DateInput element.
type DateInputOption func(*DateInput)

// WithDateFormat sets the date format.
func WithDateFormat(f DateFormat) DateInputOption {
	return func(d *DateInput) { d.Format = f }
}

// WithDateMode sets the input mode.
func WithDateMode(m DateInputMode) DateInputOption {
	return func(d *DateInput) { d.Mode = m }
}

// WithOnDateInputChange sets the change callback.
func WithOnDateInputChange(fn func(time.Time)) DateInputOption {
	return func(d *DateInput) { d.OnChange = fn }
}

// WithDateRange restricts the selectable date range.
func WithDateRange(min, max time.Time) DateInputOption {
	return func(d *DateInput) { d.Min = &min; d.Max = &max }
}

// WithDateInputDisabled disables the widget.
func WithDateInputDisabled() DateInputOption {
	return func(d *DateInput) { d.Disabled = true }
}

// NewDateInput creates an HMI date input. Default mode: DateModeDrum.
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

// OSKLayout implements osk.OSKRequester (RFC-004 §6.11).
// Only relevant in DateModeDirect.
func (d DateInput) OSKLayout() osk.OSKLayout { return osk.OSKLayoutNumericInteger }

// LayoutSelf implements ui.Layouter.
func (d DateInput) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// DateModeDrum: three drum columns (Day, Month, Year).
	// DateModeCalendar/DateModeDirect: fall back to simple display for now.
	if d.Mode == DateModeDrum || d.Mode == 0 {
		return d.layoutDrum(ctx)
	}
	return d.layoutSimple(ctx)
}

func (d DateInput) layoutDrum(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	vis := drumDefaultVisible

	totalW := 3*dateInputDrumW + 2*dateInputSepW
	if area.W < totalW {
		totalW = area.W
	}
	totalH := vis * drumItemH

	// Focus management.
	var focused bool
	if focus != nil && !d.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	year := d.Value.Year()
	month := int(d.Value.Month())
	day := d.Value.Day()

	borderColor := tokens.Colors.Stroke.Border
	if d.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}

	onChange := d.OnChange
	val := d.Value

	// Day column.
	daysInMonth := DaysInMonth(year, time.Month(month))
	d.drawDateDrumColumn(ctx, area.X, area.Y, dateInputDrumW, totalH, vis,
		day, 1, daysInMonth, func(v int) {
			if onChange != nil {
				clamped := v
				if clamped > daysInMonth {
					clamped = daysInMonth
				}
				newT := time.Date(year, time.Month(month), clamped, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}, func(v int) string { return fmt.Sprintf("%02d", v) },
		canvas, tokens, ix, style, borderColor)

	// Month column.
	monthX := area.X + dateInputDrumW + dateInputSepW
	d.drawDateDrumColumn(ctx, monthX, area.Y, dateInputDrumW, totalH, vis,
		month, 1, 12, func(v int) {
			if onChange != nil {
				newDays := DaysInMonth(year, time.Month(v))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(year, time.Month(v), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}, func(v int) string {
			if v >= 1 && v <= 12 {
				return MonthNames[v]
			}
			return fmt.Sprintf("%d", v)
		},
		canvas, tokens, ix, style, borderColor)

	// Year column.
	yearX := monthX + dateInputDrumW + dateInputSepW
	minYear := year - 10
	maxYear := year + 10
	d.drawDateDrumColumn(ctx, yearX, area.Y, dateInputDrumW, totalH, vis,
		year, minYear, maxYear, func(v int) {
			if onChange != nil {
				newDays := DaysInMonth(v, time.Month(month))
				newDay := day
				if newDay > newDays {
					newDay = newDays
				}
				newT := time.Date(v, time.Month(month), newDay, 0, 0, 0, 0, val.Location())
				onChange(newT)
			}
		}, func(v int) string { return fmt.Sprintf("%d", v) },
		canvas, tokens, ix, style, borderColor)

	if focused {
		outerRect := draw.R(float32(area.X), float32(area.Y), float32(totalW), float32(totalH))
		ui.DrawFocusRing(canvas, outerRect, tokens.Radii.Input, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

func (d DateInput) drawDateDrumColumn(_ *ui.LayoutContext, x, y, w, h, vis int,
	selected, minVal, maxVal int,
	onSelect func(int), format func(int) string,
	canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor,
	style draw.TextStyle, borderColor draw.Color) {

	outerRect := draw.R(float32(x), float32(y), float32(w), float32(h))
	fillColor := tokens.Colors.Surface.Elevated
	if d.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(outerRect, tokens.Radii.Input, draw.SolidPaint(borderColor))
	canvas.FillRoundRect(
		draw.R(float32(x+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

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

	for i := 0; i < vis; i++ {
		val := selected - half + i
		if val < minVal || val > maxVal {
			continue
		}

		itemY := y + i*drumItemH
		itemRect := draw.R(float32(x), float32(itemY), float32(w), float32(drumItemH))

		if !d.Disabled && onSelect != nil {
			v := val
			fn := onSelect
			ho := ix.RegisterHit(itemRect, func() { fn(v) })
			if ho > 0 && i != half {
				canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{A: ho * 0.05}))
			}
		}

		textColor := tokens.Colors.Text.Primary
		if d.Disabled {
			textColor = tokens.Colors.Text.Disabled
		} else if i != half {
			textColor = tokens.Colors.Text.Secondary
		}

		label := format(val)
		m := canvas.MeasureText(label, style)
		textX := float32(x) + float32(w)/2 - m.Width/2
		textY := float32(itemY) + float32(drumItemH)/2 - style.Size/2
		canvas.DrawText(label, draw.Pt(textX, textY), style, textColor)
	}
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

	// Focus management.
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

	// Date text.
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

// TreeEqual implements ui.TreeEqualizer.
func (d DateInput) TreeEqual(other ui.Element) bool {
	db, ok := other.(DateInput)
	return ok && d.Value.Equal(db.Value) && d.Format == db.Format && d.Mode == db.Mode
}

// ResolveChildren implements ui.ChildResolver. DateInput is a leaf.
func (d DateInput) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return d
}

// WalkAccess implements ui.AccessWalker.
func (d DateInput) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  d.Value.Format("2006-01-02"),
		Label:  "Date",
		States: a11y.AccessStates{Disabled: d.Disabled},
	}, parentIdx, a11y.Rect{})
}

// Compile-time interface checks.
var _ osk.OSKRequester = DateInput{}
