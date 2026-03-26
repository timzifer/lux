package form

import (
	"fmt"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Layout constants for date picker.
const (
	datePickerW      = 200
	calCellSize      = 30
	calHeaderH       = 32
	calDayLabelH     = 20
	calPad           = 8
	calCols          = 7
)

// DatePickerState holds the open/closed state and the currently viewed month.
type DatePickerState struct {
	Open      bool
	ViewYear  int
	ViewMonth time.Month
}

// DatePicker is a date selection widget with a calendar dropdown.
type DatePicker struct {
	ui.BaseElement
	Value    time.Time
	OnChange func(time.Time)
	State    *DatePickerState
	Disabled bool
}

// DatePickerOption configures a DatePicker element.
type DatePickerOption func(*DatePicker)

// WithDatePickerState links the DatePicker to state for dropdown behaviour.
func WithDatePickerState(s *DatePickerState) DatePickerOption {
	return func(e *DatePicker) { e.State = s }
}

// WithOnDateChange sets the callback invoked when a date is chosen.
func WithOnDateChange(fn func(time.Time)) DatePickerOption {
	return func(e *DatePicker) { e.OnChange = fn }
}

// WithDatePickerDisabled marks the DatePicker as disabled.
func WithDatePickerDisabled() DatePickerOption {
	return func(e *DatePicker) { e.Disabled = true }
}

// NewDatePicker creates a date picker element.
func NewDatePicker(value time.Time, opts ...DatePickerOption) ui.Element {
	el := DatePicker{Value: value}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n DatePicker) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := datePickerW
	if area.W < w {
		w = area.W
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Hit target: click toggles dropdown.
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(fieldRect, nil)
	} else {
		var clickFn func()
		if n.State != nil {
			state := n.State
			val := n.Value
			clickFn = func() {
				state.Open = !state.Open
				if state.Open && state.ViewYear == 0 {
					state.ViewYear = val.Year()
					state.ViewMonth = val.Month()
				}
			}
		}
		hoverOpacity = ix.RegisterHit(fieldRect, clickFn)
	}

	isOpen := n.State != nil && n.State.Open && !n.Disabled

	// Focus management.
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Border.
	borderColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill.
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	if n.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Date text.
	dateText := n.Value.Format("2006-01-02")
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(dateText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Down arrow indicator.
	arrowStyle := tokens.Typography.LabelSmall
	arrowStyle.FontFamily = "Phosphor"
	arrowX := area.X + w - textFieldPadX - int(arrowStyle.Size)
	arrowColor := tokens.Colors.Text.Secondary
	if n.Disabled {
		arrowColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(icons.CaretDown, draw.Pt(float32(arrowX), float32(textY)), arrowStyle, arrowColor)

	// Focus glow.
	if focused || isOpen {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// Calendar overlay.
	if isOpen && n.State != nil && overlays != nil {
		state := n.State
		onChange := n.OnChange
		selectedDate := n.Value
		winW := overlays.WindowW
		winH := overlays.WindowH

		viewYear := state.ViewYear
		viewMonth := state.ViewMonth
		if viewYear == 0 {
			viewYear = selectedDate.Year()
			viewMonth = selectedDate.Month()
		}

		calW := calCols*calCellSize + calPad*2
		calRows := calendarRows(viewYear, viewMonth)
		calH := calHeaderH + calDayLabelH + calRows*calCellSize + calPad*2

		dropX := area.X
		dropY := area.Y + h
		if dropY+calH > winH && area.Y-calH >= 0 {
			dropY = area.Y - calH
		}

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Backdrop.
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					state.Open = false
				})

				// Calendar background.
				calRect := draw.R(float32(dropX), float32(dropY), float32(calW), float32(calH))
				canvas.FillRoundRect(calRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				canvas.StrokeRoundRect(calRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				bodyStyle := tokens.Typography.Body
				labelStyle := tokens.Typography.Label

				// Header: < Month Year >
				headerY := dropY + calPad
				prevRect := draw.R(float32(dropX+calPad), float32(headerY), float32(calCellSize), float32(calHeaderH))
				nextRect := draw.R(float32(dropX+calW-calPad-calCellSize), float32(headerY), float32(calCellSize), float32(calHeaderH))

				prevHo := ix.RegisterHit(prevRect, func() {
					if viewMonth == time.January {
						state.ViewMonth = time.December
						state.ViewYear = viewYear - 1
					} else {
						state.ViewMonth = viewMonth - 1
					}
				})
				if prevHo > 0 {
					canvas.FillRoundRect(prevRect, 4, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				navStyle := labelStyle
				navStyle.FontFamily = "Phosphor"
				canvas.DrawText(icons.CaretLeft,
					draw.Pt(float32(dropX+calPad+calCellSize/2-int(navStyle.Size)/2),
						float32(headerY+calHeaderH/2-int(navStyle.Size)/2)),
					navStyle, tokens.Colors.Text.Primary)

				nextHo := ix.RegisterHit(nextRect, func() {
					if viewMonth == time.December {
						state.ViewMonth = time.January
						state.ViewYear = viewYear + 1
					} else {
						state.ViewMonth = viewMonth + 1
					}
				})
				if nextHo > 0 {
					canvas.FillRoundRect(nextRect, 4, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				canvas.DrawText(icons.CaretRight,
					draw.Pt(float32(dropX+calW-calPad-calCellSize/2-int(navStyle.Size)/2),
						float32(headerY+calHeaderH/2-int(navStyle.Size)/2)),
					navStyle, tokens.Colors.Text.Primary)

				// Month Year label (centered).
				monthLabel := fmt.Sprintf("%s %d", viewMonth.String()[:3], viewYear)
				monthMetrics := canvas.MeasureText(monthLabel, bodyStyle)
				monthX := float32(dropX) + float32(calW)/2 - monthMetrics.Width/2
				canvas.DrawText(monthLabel,
					draw.Pt(monthX, float32(headerY+calHeaderH/2-int(bodyStyle.Size)/2)),
					bodyStyle, tokens.Colors.Text.Primary)

				// Day-of-week labels.
				dayLabelY := headerY + calHeaderH
				dayNames := [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
				for i, dn := range dayNames {
					dx := dropX + calPad + i*calCellSize
					m := canvas.MeasureText(dn, labelStyle)
					cx := float32(dx) + float32(calCellSize)/2 - m.Width/2
					canvas.DrawText(dn, draw.Pt(cx, float32(dayLabelY+2)), labelStyle, tokens.Colors.Text.Secondary)
				}

				// Day grid.
				gridY := dayLabelY + calDayLabelH
				firstDay := time.Date(viewYear, viewMonth, 1, 0, 0, 0, 0, time.Local)
				startWeekday := int(firstDay.Weekday()) // 0=Sun
				daysInMonth := daysIn(viewYear, viewMonth)

				selY := selectedDate.Year()
				selM := selectedDate.Month()
				selD := selectedDate.Day()

				for d := 1; d <= daysInMonth; d++ {
					idx := startWeekday + d - 1
					col := idx % calCols
					row := idx / calCols
					cx := dropX + calPad + col*calCellSize
					cy := gridY + row*calCellSize
					cellRect := draw.R(float32(cx), float32(cy), float32(calCellSize), float32(calCellSize))

					day := d
					yr := viewYear
					mo := viewMonth
					var cellClick func()
					if onChange != nil || state != nil {
						cellClick = func() {
							if onChange != nil {
								onChange(time.Date(yr, mo, day, 0, 0, 0, 0, time.Local))
							}
							state.Open = false
						}
					}
					ho := ix.RegisterHit(cellRect, cellClick)

					// Highlight selected day.
					isSelected := viewYear == selY && viewMonth == selM && d == selD
					if isSelected {
						canvas.FillEllipse(cellRect, draw.SolidPaint(tokens.Colors.Accent.Primary))
					} else if ho > 0 {
						canvas.FillEllipse(cellRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}

					dayStr := fmt.Sprintf("%d", d)
					dm := canvas.MeasureText(dayStr, labelStyle)
					dayTextX := float32(cx) + float32(calCellSize)/2 - dm.Width/2
					dayTextY := float32(cy) + float32(calCellSize)/2 - labelStyle.Size/2

					dayColor := tokens.Colors.Text.Primary
					if isSelected {
						dayColor = tokens.Colors.Text.OnAccent
					}
					canvas.DrawText(dayStr, draw.Pt(dayTextX, dayTextY), labelStyle, dayColor)
				}
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// calendarRows returns the number of rows needed for the given month.
func calendarRows(year int, month time.Month) int {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	days := daysIn(year, month)
	totalCells := startWeekday + days
	rows := totalCells / calCols
	if totalCells%calCols != 0 {
		rows++
	}
	return rows
}

// daysIn returns the number of days in the given month.
func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// TreeEqual implements ui.TreeEqualizer.
func (n DatePicker) TreeEqual(other ui.Element) bool {
	nb, ok := other.(DatePicker)
	return ok && n.Value.Equal(nb.Value)
}

// ResolveChildren implements ui.ChildResolver. DatePicker is a leaf.
func (n DatePicker) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n DatePicker) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  n.Value.Format("2006-01-02"),
		States: a11y.AccessStates{Disabled: n.Disabled},
	}, parentIdx, a11y.Rect{})
}
