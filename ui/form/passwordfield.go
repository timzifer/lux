package form

import (
	"strings"
	"unicode/utf8"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Layout constants for the reveal toggle button inside the password field.
const (
	passwordMask   = "•"
	revealIconSize = 14 // icon glyph size in dp
	revealBtnPad   = 6  // horizontal padding around the icon
)

// PasswordField is a text input that masks its content with bullet characters.
// When Revealed is true the actual text is displayed instead.
// If OnRevealChange is set a toggle button (eye icon) is rendered on the right.
type PasswordField struct {
	ui.BaseElement
	Value          string
	Placeholder    string
	OnChange       func(string)
	Focus          *ui.FocusManager
	Disabled       bool
	Revealed       bool
	OnRevealChange func(bool)
}

// PasswordFieldOption configures a PasswordField.
type PasswordFieldOption func(*PasswordField)

// PasswordOnChange sets the callback invoked when the text value changes.
func PasswordOnChange(fn func(string)) PasswordFieldOption {
	return func(e *PasswordField) { e.OnChange = fn }
}

// PasswordFocus links the PasswordField to a FocusManager for keyboard input.
func PasswordFocus(fm *ui.FocusManager) PasswordFieldOption {
	return func(e *PasswordField) { e.Focus = fm }
}

// PasswordDisabled marks the PasswordField as disabled.
func PasswordDisabled() PasswordFieldOption {
	return func(e *PasswordField) { e.Disabled = true }
}

// PasswordReveal enables the reveal toggle with the given state and callback.
func PasswordReveal(revealed bool, onChange func(bool)) PasswordFieldOption {
	return func(e *PasswordField) {
		e.Revealed = revealed
		e.OnRevealChange = onChange
	}
}

// NewPasswordField creates a password input field.
func NewPasswordField(value, placeholder string, opts ...PasswordFieldOption) ui.Element {
	el := PasswordField{Value: value, Placeholder: placeholder}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// displayText returns masked or clear text depending on Revealed state.
func (n PasswordField) displayText() string {
	if n.Revealed {
		return n.Value
	}
	return strings.Repeat(passwordMask, utf8.RuneCountInString(n.Value))
}

// LayoutSelf implements ui.Layouter.
func (n PasswordField) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
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

	// Reserve space for reveal button when present.
	btnW := 0
	if n.OnRevealChange != nil {
		btnW = int(revealIconSize) + revealBtnPad*2
	}
	textAreaW := w - btnW

	// Focus management.
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

	// Custom theme DrawFunc dispatch.
	if df := th.DrawFunc(theme.WidgetKindTextField); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			Focused:  focused,
			Disabled: n.Disabled,
		}, tokens, n)
	} else {
		tfRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

		// Border.
		borderColor := tokens.Colors.Stroke.Border
		if n.Disabled {
			borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(tfRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

		// Fill.
		fillColor := tokens.Colors.Surface.Elevated
		if n.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(
			draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Focus ring.
		if focused {
			ui.DrawFocusRing(canvas, tfRect, tokens.Radii.Input, tokens)
		}

		// Text or placeholder.
		displayStr := n.displayText()
		textX := area.X + textFieldPadX
		textY := area.Y + textFieldPadY
		textColor := tokens.Colors.Text.Primary
		if n.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if n.Value != "" {
			canvas.DrawText(displayStr, draw.Pt(float32(textX), float32(textY)), style, textColor)
		} else if n.Placeholder != "" {
			canvas.DrawText(n.Placeholder, draw.Pt(float32(textX), float32(textY)), style, tokens.Colors.Text.Disabled)
		}

		// Cursor when focused.
		if focused {
			metrics := canvas.MeasureText(displayStr, style)
			cursorX := float32(textX) + metrics.Width
			maxCursorX := float32(area.X+textAreaW) - float32(textFieldPadX)
			if cursorX > maxCursorX {
				cursorX = maxCursorX
			}
			canvas.FillRect(draw.R(cursorX, float32(textY), 2, style.Size),
				draw.SolidPaint(tokens.Colors.Text.Primary))
		}
	}

	// Store input state for framework key handling.
	if focused && n.OnChange != nil && focus != nil {
		cursorOff := len(n.Value)
		selStart := -1
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			selStart = focus.Input.SelectionStart
			if cursorOff > len(n.Value) {
				cursorOff = len(n.Value)
			}
		}
		focus.Input = &ui.InputState{
			Value:          n.Value,
			OnChange:       n.OnChange,
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: selStart,
		}
	}

	// Hit target for focus acquisition (full field).
	if n.OnChange != nil && focus != nil && !n.Disabled {
		uid := focusUID
		fm := focus
		ix.RegisterHit(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			func() { fm.SetFocusedUID(uid) })
	}

	// Reveal button — registered after focus target so it wins in hit testing.
	if n.OnRevealChange != nil && !n.Disabled {
		btnX := area.X + w - btnW
		btnRect := draw.R(float32(btnX), float32(area.Y), float32(btnW), float32(h))

		revealed := n.Revealed
		onChange := n.OnRevealChange
		var onClick func()
		if focus != nil && n.OnChange != nil {
			uid := focusUID
			fm := focus
			onClick = func() {
				fm.SetFocusedUID(uid)
				onChange(!revealed)
			}
		} else {
			onClick = func() { onChange(!revealed) }
		}
		hoverOpacity := ix.RegisterHit(btnRect, onClick)

		// Draw the eye icon.
		iconStyle := draw.TextStyle{
			FontFamily: "Phosphor",
			Size:       float32(revealIconSize),
			Weight:     draw.FontWeightRegular,
			LineHeight: 1.0,
			Raster:     true,
		}

		iconName := icons.EyeSlash
		if n.Revealed {
			iconName = icons.Eye
		}

		iconColor := tokens.Colors.Text.Secondary
		if hoverOpacity > 0.1 {
			iconColor = tokens.Colors.Text.Primary
		}

		metrics := canvas.MeasureText(iconName, iconStyle)
		offsetX := (float32(btnW) - metrics.Width) / 2
		offsetY := (float32(h) - metrics.Ascent) / 2
		canvas.DrawText(iconName,
			draw.Pt(float32(btnX)+offsetX, float32(area.Y)+offsetY),
			iconStyle, iconColor)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n PasswordField) TreeEqual(other ui.Element) bool {
	nb, ok := other.(PasswordField)
	return ok && n.Value == nb.Value && n.Placeholder == nb.Placeholder && n.Revealed == nb.Revealed
}

// ResolveChildren implements ui.ChildResolver. PasswordField is a leaf.
func (n PasswordField) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n PasswordField) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	an := a11y.AccessNode{
		Role:   a11y.RoleTextInput,
		Label:  n.Placeholder,
		States: a11y.AccessStates{Disabled: n.Disabled},
		TextState: &a11y.AccessTextState{
			Length:      len([]rune(n.Value)),
			CaretOffset: -1,
		},
	}
	// Don't expose the actual password value in the accessibility tree.
	b.AddNode(an, parentIdx, a11y.Rect{})
}
