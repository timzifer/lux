package form

import (
	"fmt"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Layout constants for time picker.
const (
	timePickerW       = 200
	timeColumnW       = 60
	timeColumnGap     = 8
	timeItemH         = 28
	timeDropdownPad   = 8
	timeMaxVisRows    = 6
)

// TimePickerState holds the open/closed state for a TimePicker dropdown.
type TimePickerState struct {
	Open bool
}

// TimePicker is a time selection widget showing HH:MM.
type TimePicker struct {
	ui.BaseElement
	Hour     int
	Minute   int
	OnChange func(hour, minute int)
	State    *TimePickerState
	Disabled bool
}

// TimePickerOption configures a TimePicker element.
type TimePickerOption func(*TimePicker)

// WithTimePickerState links the TimePicker to state for dropdown behaviour.
func WithTimePickerState(s *TimePickerState) TimePickerOption {
	return func(e *TimePicker) { e.State = s }
}

// WithOnTimeChange sets the callback invoked when the time changes.
func WithOnTimeChange(fn func(hour, minute int)) TimePickerOption {
	return func(e *TimePicker) { e.OnChange = fn }
}

// WithTimePickerDisabled marks the TimePicker as disabled.
func WithTimePickerDisabled() TimePickerOption {
	return func(e *TimePicker) { e.Disabled = true }
}

// NewTimePicker creates a time picker element.
func NewTimePicker(hour, minute int, opts ...TimePickerOption) ui.Element {
	el := TimePicker{Hour: hour, Minute: minute}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n TimePicker) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := timePickerW
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
			clickFn = func() { state.Open = !state.Open }
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

	// Time text.
	timeText := fmt.Sprintf("%02d:%02d", n.Hour, n.Minute)
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	textColor := tokens.Colors.Text.Primary
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(timeText, draw.Pt(float32(textX), float32(textY)), style, textColor)

	// Clock icon.
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

	// Dropdown overlay: two columns (hours, minutes).
	if isOpen && overlays != nil {
		dropX := area.X
		dropY := area.Y + h
		state := n.State
		onChange := n.OnChange
		curHour := n.Hour
		curMinute := n.Minute
		winW := overlays.WindowW
		winH := overlays.WindowH

		colH := timeMaxVisRows * timeItemH
		dropW := timeColumnW*2 + timeColumnGap + timeDropdownPad*2
		dropH := colH + timeDropdownPad*2

		if dropY+dropH > winH && area.Y-dropH >= 0 {
			dropY = area.Y - dropH
		}

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Backdrop.
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					if state != nil {
						state.Open = false
					}
				})

				// Background.
				bgRect := draw.R(float32(dropX), float32(dropY), float32(dropW), float32(dropH))
				canvas.FillRoundRect(bgRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				canvas.StrokeRoundRect(bgRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				bodyStyle := tokens.Typography.Body

				// Hours column (0-23).
				hColX := dropX + timeDropdownPad
				for i := 0; i < 24 && i < timeMaxVisRows*4; i++ {
					// Show a window of hours around the current hour.
					startH := curHour - timeMaxVisRows/2
					if startH < 0 {
						startH = 0
					}
					if startH+timeMaxVisRows > 24 {
						startH = 24 - timeMaxVisRows
					}
					hour := startH + i
					if i >= timeMaxVisRows || hour < 0 || hour >= 24 {
						break
					}
					itemY := dropY + timeDropdownPad + i*timeItemH
					itemRect := draw.R(float32(hColX), float32(itemY), float32(timeColumnW), float32(timeItemH))
					hr := hour
					var itemClick func()
					if onChange != nil || state != nil {
						itemClick = func() {
							if onChange != nil {
								onChange(hr, curMinute)
							}
							if state != nil {
								state.Open = false
							}
						}
					}
					ho := ix.RegisterHit(itemRect, itemClick)
					if ho > 0 {
						canvas.FillRect(itemRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}
					if hour == curHour {
						canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{
							R: tokens.Colors.Accent.Primary.R,
							G: tokens.Colors.Accent.Primary.G,
							B: tokens.Colors.Accent.Primary.B,
							A: 0.15,
						}))
					}
					canvas.DrawText(fmt.Sprintf("%02d", hour),
						draw.Pt(float32(hColX+8), float32(itemY+6)), bodyStyle, tokens.Colors.Text.Primary)
				}

				// Minutes column (0, 5, 10, ..., 55).
				mColX := dropX + timeDropdownPad + timeColumnW + timeColumnGap
				minuteSteps := []int{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55}
				// Show window around current minute.
				startIdx := 0
				for j, m := range minuteSteps {
					if m <= curMinute {
						startIdx = j
					}
				}
				startIdx -= timeMaxVisRows / 2
				if startIdx < 0 {
					startIdx = 0
				}
				if startIdx+timeMaxVisRows > len(minuteSteps) {
					startIdx = len(minuteSteps) - timeMaxVisRows
				}
				if startIdx < 0 {
					startIdx = 0
				}
				for i := 0; i < timeMaxVisRows && startIdx+i < len(minuteSteps); i++ {
					minute := minuteSteps[startIdx+i]
					itemY := dropY + timeDropdownPad + i*timeItemH
					itemRect := draw.R(float32(mColX), float32(itemY), float32(timeColumnW), float32(timeItemH))
					min := minute
					var itemClick func()
					if onChange != nil || state != nil {
						itemClick = func() {
							if onChange != nil {
								onChange(curHour, min)
							}
							if state != nil {
								state.Open = false
							}
						}
					}
					ho := ix.RegisterHit(itemRect, itemClick)
					if ho > 0 {
						canvas.FillRect(itemRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}
					if minute == curMinute {
						canvas.FillRect(itemRect, draw.SolidPaint(draw.Color{
							R: tokens.Colors.Accent.Primary.R,
							G: tokens.Colors.Accent.Primary.G,
							B: tokens.Colors.Accent.Primary.B,
							A: 0.15,
						}))
					}
					canvas.DrawText(fmt.Sprintf("%02d", minute),
						draw.Pt(float32(mColX+8), float32(itemY+6)), bodyStyle, tokens.Colors.Text.Primary)
				}
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n TimePicker) TreeEqual(other ui.Element) bool {
	nb, ok := other.(TimePicker)
	return ok && n.Hour == nb.Hour && n.Minute == nb.Minute
}

// ResolveChildren implements ui.ChildResolver. TimePicker is a leaf.
func (n TimePicker) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n TimePicker) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Value:  fmt.Sprintf("%02d:%02d", n.Hour, n.Minute),
		States: a11y.AccessStates{Disabled: n.Disabled},
	}, parentIdx, a11y.Rect{})
}
