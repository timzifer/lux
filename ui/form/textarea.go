package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Layout constants for text area.
const (
	textAreaPadX    = 8
	textAreaPadY    = 8
	textAreaMinW    = 100
	textAreaDefRows = 4
)

// TextArea is a multiline text input field.
type TextArea struct {
	ui.BaseElement
	Value       string
	Placeholder string
	Rows        int // visible rows; default 4
	OnChange    func(string)
	Focus       *ui.FocusManager
	FocusUID    ui.UID
	Disabled    bool
	Scroll      *ui.ScrollState
}

// TextAreaOption configures a TextArea.
type TextAreaOption func(*TextArea)

// TextAreaOnChange sets the callback invoked when the text value changes.
func TextAreaOnChange(fn func(string)) TextAreaOption {
	return func(e *TextArea) { e.OnChange = fn }
}

// TextAreaFocus links the TextArea to a FocusManager for keyboard input.
func TextAreaFocus(fm *ui.FocusManager) TextAreaOption {
	return func(e *TextArea) { e.Focus = fm }
}

// TextAreaRows sets the number of visible rows.
func TextAreaRows(n int) TextAreaOption {
	return func(e *TextArea) { e.Rows = n }
}

// TextAreaDisabled marks the TextArea as disabled.
func TextAreaDisabled() TextAreaOption {
	return func(e *TextArea) { e.Disabled = true }
}

// TextAreaScroll links the TextArea to a ScrollState for internal scrolling.
func TextAreaScroll(s *ui.ScrollState) TextAreaOption {
	return func(e *TextArea) { e.Scroll = s }
}

// NewTextArea creates a multiline text input field.
func NewTextArea(value, placeholder string, opts ...TextAreaOption) ui.Element {
	el := TextArea{
		Value:       value,
		Placeholder: placeholder,
		Rows:        textAreaDefRows,
	}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n TextArea) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	th := ctx.Theme
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	style := tokens.Typography.Body
	rows := n.Rows
	if rows < 1 {
		rows = textAreaDefRows
	}

	// Compute line height using the same formula as DrawTextLayout.
	metrics := canvas.MeasureText("Mg", style)
	lineH := metrics.Ascent + metrics.Descent + metrics.Leading
	if style.LineHeight > 0 {
		lineH = style.Size * style.LineHeight
	}

	w := area.W
	if w < textAreaMinW {
		w = textAreaMinW
	}
	viewportH := int(lineH*float32(rows)) + textAreaPadY*2
	h := viewportH

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

	// Custom theme DrawFunc dispatch.
	if df := th.DrawFunc(theme.WidgetKindTextArea); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			Focused:  focused,
			Disabled: n.Disabled,
		}, tokens, n)
	} else {
		taRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

		// Border.
		borderColor := tokens.Colors.Stroke.Border
		if n.Disabled {
			borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(taRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

		// Fill.
		fillColor := tokens.Colors.Surface.Elevated
		if n.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(
			draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Focus glow + ring.
		if focused {
			ui.DrawFocusRing(canvas, taRect, tokens.Radii.Input, tokens)
		}

		// Content area.
		contentX := float32(area.X + textAreaPadX)
		contentY := float32(area.Y + textAreaPadY)
		contentW := float32(w - textAreaPadX*2)
		contentH := float32(h - textAreaPadY*2)

		// Scroll offset.
		scrollOff := float32(0)
		if n.Scroll != nil {
			scrollOff = n.Scroll.Offset
		}

		// Compute lines and find cursor position.
		lines := text.Lines(n.Value)
		totalContentH := lineH * float32(len(lines))

		// Determine which line the cursor is on and the cursor X position.
		cursorLine := 0
		cursorX := contentX
		cursorOff := len(n.Value)
		if focused && focus != nil && focus.Input != nil {
			cursorOff = focus.Input.CursorOffset
		}
		for i, span := range lines {
			if cursorOff >= span.Start && (cursorOff <= span.End || i == len(lines)-1) {
				cursorLine = i
				lineText := n.Value[span.Start:span.End]
				offInLine := cursorOff - span.Start
				if offInLine > len(lineText) {
					offInLine = len(lineText)
				}
				m := canvas.MeasureText(lineText[:offInLine], style)
				cursorX = contentX + m.Width
				break
			}
		}

		// Auto-scroll cursor into view.
		if focused && n.Scroll != nil {
			cursorTop := lineH * float32(cursorLine)
			cursorBottom := cursorTop + lineH
			if cursorTop < scrollOff {
				scrollOff = cursorTop
			}
			if cursorBottom > scrollOff+contentH {
				scrollOff = cursorBottom - contentH
			}
			maxScroll := totalContentH - contentH
			if maxScroll < 0 {
				maxScroll = 0
			}
			if scrollOff < 0 {
				scrollOff = 0
			}
			if scrollOff > maxScroll {
				scrollOff = maxScroll
			}
			n.Scroll.Offset = scrollOff
		}

		// Clip to content area.
		clipRect := draw.R(contentX, contentY, contentW, contentH)
		canvas.PushClip(clipRect)

		// Draw selection highlight.
		if focused && focus != nil && focus.Input != nil && focus.Input.HasSelection() {
			selA, selB := focus.Input.SelectionRange()
			if selA > len(n.Value) {
				selA = len(n.Value)
			}
			if selB > len(n.Value) {
				selB = len(n.Value)
			}
			selColor := tokens.Colors.Accent.Primary
			selColor.A = 0.3
			for i, span := range lines {
				y := contentY + lineH*float32(i) - scrollOff
				if y+lineH < contentY || y > contentY+contentH {
					continue
				}
				// Compute overlap of selection with this line.
				lineSelStart := selA
				if lineSelStart < span.Start {
					lineSelStart = span.Start
				}
				lineSelEnd := selB
				if lineSelEnd > span.End {
					lineSelEnd = span.End
				}
				if lineSelStart >= lineSelEnd {
					// Check if selection extends past line end (newline selected).
					if selA <= span.End && selB > span.End && i < len(lines)-1 {
						// Highlight to end of text on this line.
						lineText := n.Value[span.Start:span.End]
						mEnd := canvas.MeasureText(lineText, style)
						canvas.FillRect(draw.R(contentX+mEnd.Width, y, 4, lineH),
							draw.SolidPaint(selColor))
					}
					continue
				}
				lineText := n.Value[span.Start:span.End]
				offA := lineSelStart - span.Start
				offB := lineSelEnd - span.Start
				mA := canvas.MeasureText(lineText[:offA], style)
				mB := canvas.MeasureText(lineText[:offB], style)
				sx := contentX + mA.Width
				sw := mB.Width - mA.Width
				canvas.FillRect(draw.R(sx, y, sw, lineH),
					draw.SolidPaint(selColor))
				// If selection continues past line end, show newline selection.
				if selB > span.End && i < len(lines)-1 {
					canvas.FillRect(draw.R(contentX+mB.Width, y, 4, lineH),
						draw.SolidPaint(selColor))
				}
			}
		}

		// Draw text or placeholder.
		textColor := tokens.Colors.Text.Primary
		if n.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if n.Value != "" {
			for i, span := range lines {
				y := contentY + lineH*float32(i) - scrollOff
				if y+lineH < contentY || y > contentY+contentH {
					continue // skip lines outside viewport
				}
				lineText := n.Value[span.Start:span.End]
				canvas.DrawText(lineText, draw.Pt(contentX, y), style, textColor)
			}
		} else if n.Placeholder != "" {
			canvas.DrawText(n.Placeholder, draw.Pt(contentX, contentY), style, tokens.Colors.Text.Disabled)
		}

		// Draw cursor when focused.
		if focused {
			cy := contentY + lineH*float32(cursorLine) - scrollOff
			canvas.FillRect(draw.R(cursorX, cy, 2, lineH),
				draw.SolidPaint(tokens.Colors.Text.Primary))
		}

		canvas.PopClip()
	}

	// Store input state for the focused TextArea.
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
			Multiline:      true,
		}
	}

	// Register scroll for mouse wheel.
	if n.Scroll != nil {
		lines := text.Lines(n.Value)
		lineH2 := lineH
		totalH := lineH2 * float32(len(lines))
		vpH := float32(h - textAreaPadY*2)
		scroll := n.Scroll
		ix.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			totalH, vpH,
			func(deltaY float32) { scroll.ScrollBy(deltaY, totalH, vpH) },
		)
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
func (n TextArea) TreeEqual(other ui.Element) bool {
	nb, ok := other.(TextArea)
	return ok && n.Value == nb.Value && n.Placeholder == nb.Placeholder && n.Rows == nb.Rows
}

// ResolveChildren implements ui.ChildResolver. TextArea is a leaf.
func (n TextArea) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n TextArea) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
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
