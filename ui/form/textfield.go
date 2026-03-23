package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
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

		// Text or placeholder
		textX := area.X + textFieldPadX
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

		// Cursor when focused
		if focused {
			metrics := canvas.MeasureText(n.Value, style)
			cursorX := float32(textX) + metrics.Width
			canvas.FillRect(draw.R(cursorX, float32(textY), 2, style.Size),
				draw.SolidPaint(tokens.Colors.Text.Primary))
		}
	}

	// Store input state for the focused TextField so the framework can
	// handle KeyMsg/CharMsg internally (no userland boilerplate needed).
	if focused && n.OnChange != nil && focus != nil {
		focus.Input = &ui.InputState{
			Value:    n.Value,
			OnChange: n.OnChange,
			FocusUID: focusUID,
		}
	}

	// Hit target for focus acquisition.
	if n.OnChange != nil && focus != nil && !n.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			func() { fm.SetFocusedUID(uid) })
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
