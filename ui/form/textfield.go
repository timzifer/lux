package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Layout constants for text field.
const (
	textFieldW    = 200
	textFieldPadX = 8
	textFieldPadY = 8
)

// TextField is a text input field.
type TextField struct {
	ui.BaseElement
	Value       string
	Placeholder string
	OnChange    func(string)
	Focus       *ui.FocusManager
	FocusUID    ui.UID // assigned during layout
	Disabled    bool
}

// TextFieldOption configures a TextField.
type TextFieldOption func(*TextField)

// WithOnChange sets the callback invoked when the text value changes.
func WithOnChange(fn func(string)) TextFieldOption {
	return func(e *TextField) { e.OnChange = fn }
}

// WithFocus links the TextField to a FocusManager for keyboard input.
func WithFocus(fm *ui.FocusManager) TextFieldOption {
	return func(e *TextField) { e.Focus = fm }
}

// WithDisabled marks the TextField as disabled.
func WithDisabled() TextFieldOption {
	return func(e *TextField) { e.Disabled = true }
}

// NewTextField creates a text input field.
func NewTextField(value string, placeholder string, opts ...TextFieldOption) ui.Element {
	el := TextField{Value: value, Placeholder: placeholder}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n TextField) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	th := ctx.Theme
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Assign a focus UID if focus manager is provided.
	var focusUID ui.UID
	if focus != nil && !n.Disabled {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{
			Focusable:    true,
			TabIndex:     0,
			FocusOnClick: true,
		})
	}
	focused := !n.Disabled && focus != nil && focus.IsElementFocused(focusUID)

	textX := area.X + textFieldPadX

	// Custom theme DrawFunc dispatch (RFC §5.3).
	if df := th.DrawFunc(theme.WidgetKindTextField); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			Focused:  focused,
			Disabled: n.Disabled,
		}, tokens, n)
	} else {
		tfRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

		// Border
		borderColor := tokens.Colors.Stroke.Border
		if n.Disabled {
			borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(tfRect,
			tokens.Radii.Input, draw.SolidPaint(borderColor))

		// Fill
		fillColor := tokens.Colors.Surface.Elevated
		if n.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(
			draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Focus glow + ring (RFC-008 §9.4)
		if focused {
			ui.DrawFocusRing(canvas, tfRect, tokens.Radii.Input, tokens)
		}

		// Clip text content to the field's inner area.
		clipRect := draw.R(float32(area.X+textFieldPadX), float32(area.Y), float32(w-textFieldPadX*2), float32(h))
		canvas.PushClip(clipRect)

		// Text or placeholder
		textY := area.Y + textFieldPadY
		textColor := tokens.Colors.Text.Primary
		if n.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if n.Value != "" {
			canvas.DrawText(n.Value, draw.Pt(float32(textX), float32(textY)), style, textColor)
		} else if n.Placeholder != "" {
			canvas.DrawText(n.Placeholder, draw.Pt(float32(textX), float32(textY)), style, tokens.Colors.Text.Disabled)
		}

		// Selection highlight + cursor when focused.
		if focused {
			cursorOff := len(n.Value) // default: end
			if focus != nil && focus.Input != nil {
				cursorOff = focus.Input.CursorOffset
				if cursorOff > len(n.Value) {
					cursorOff = len(n.Value)
				}
			}

			// Draw selection highlight.
			if focus != nil && focus.Input != nil && focus.Input.HasSelection() {
				selA, selB := focus.Input.SelectionRange()
				if selA > len(n.Value) {
					selA = len(n.Value)
				}
				if selB > len(n.Value) {
					selB = len(n.Value)
				}
				mA := canvas.MeasureText(n.Value[:selA], style)
				mB := canvas.MeasureText(n.Value[:selB], style)
				selX := float32(textX) + mA.Width
				selW := mB.Width - mA.Width
				selColor := tokens.Colors.Accent.Primary
				selColor.A = 0.3
				canvas.FillRect(draw.R(selX, float32(textY), selW, style.Size),
					draw.SolidPaint(selColor))
			}

			metrics := canvas.MeasureText(n.Value[:cursorOff], style)
			cursorX := float32(textX) + metrics.Width
			canvas.FillRect(draw.R(cursorX, float32(textY), 2, style.Size),
				draw.SolidPaint(tokens.Colors.Text.Primary))
		}

		canvas.PopClip()
	}

	// Store input state for the focused TextField so the framework can
	// handle KeyMsg/CharMsg internally (no userland boilerplate needed).
	if focused && n.OnChange != nil && focus != nil {
		cursorOff := len(n.Value)
		selStart := -1
		// Preserve cursor/selection from previous frame if this field was already focused.
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			selStart = focus.Input.SelectionStart
			if cursorOff > len(n.Value) {
				cursorOff = len(n.Value)
			}
		}
		// Apply pending cursor offset from a click that occurred before
		// InputState existed (first click to focus).
		if focus.PendingCursorOffset >= 0 {
			cursorOff = focus.PendingCursorOffset
			if cursorOff > len(n.Value) {
				cursorOff = len(n.Value)
			}
			selStart = -1
			focus.PendingCursorOffset = -1
		}
		focus.Input = &ui.InputState{
			Value:          n.Value,
			OnChange:       n.OnChange,
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: selStart,
		}
	}

	// Pre-compute grapheme boundary X positions for click-to-cursor.
	boundaries := text.GraphemeClusters(n.Value)
	boundaryXs := make([]float32, len(boundaries))
	for i, boff := range boundaries {
		if boff == 0 {
			boundaryXs[i] = float32(textX)
		} else {
			m := canvas.MeasureText(n.Value[:boff], style)
			boundaryXs[i] = float32(textX) + m.Width
		}
	}

	// Hit target for focus acquisition and click-to-position cursor.
	if n.OnChange != nil && focus != nil && !n.Disabled {
		uid := focusUID
		fm := focus
		bXs := boundaryXs
		bOffs := boundaries
		dragAnchor := -1
		ix.RegisterDrag(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			func(mx, _ float32) {
				fm.SetFocusedUID(uid)
				off := closestBoundary(bXs, bOffs, mx)
				if dragAnchor < 0 {
					dragAnchor = off
					if fm.Input != nil {
						fm.Input.CursorOffset = off
						fm.Input.ClearSelection()
					} else {
						fm.PendingCursorOffset = off
					}
				} else {
					if fm.Input != nil {
						fm.Input.CursorOffset = off
						if off != dragAnchor {
							fm.Input.SelectionStart = dragAnchor
						} else {
							fm.Input.ClearSelection()
						}
					}
				}
			})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n TextField) TreeEqual(other ui.Element) bool {
	nb, ok := other.(TextField)
	return ok && n.Value == nb.Value && n.Placeholder == nb.Placeholder
}

// ResolveChildren implements ui.ChildResolver. TextField is a leaf.
func (n TextField) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n TextField) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleTextInput,
		Label:  n.Placeholder,
		Value:  n.Value,
		States: a11y.AccessStates{Disabled: n.Disabled},
		TextState: &a11y.AccessTextState{
			Length:      len([]rune(n.Value)),
			CaretOffset: -1,
		},
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

// closestBoundary returns the byte offset of the grapheme boundary
// closest to pixel position mx.
func closestBoundary(xs []float32, offsets []int, mx float32) int {
	best := 0
	bestDist := float32(1e9)
	for i, x := range xs {
		d := mx - x
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	if best < len(offsets) {
		return offsets[best]
	}
	return 0
}
