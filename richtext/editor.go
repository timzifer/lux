package richtext

import (
	"math"
	"strconv"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Layout constants for the editor.
const (
	editorPadX    = 8
	editorPadY    = 8
	editorMinW    = 200
	editorMinRows = 4
)

// ── RichTextEditor ──────────────────────────────────────────────

// RichTextEditor is an editable rich-text widget (RFC-003 §5.6).
// The AttributedString lives in the user model; internal state
// (cursor, selection, undo/redo) lives in the framework's WidgetState.
type RichTextEditor struct {
	ui.BaseElement

	// Value is the current content (user-model owned).
	Value AttributedString

	// OnChange is called when the user edits the content.
	OnChange func(AttributedString)

	// ReadOnly accepts no input but allows selection and copy.
	ReadOnly bool

	// Rows controls the visible row count (default 4).
	Rows int

	// Focus links the editor to a FocusManager for keyboard input.
	Focus *ui.FocusManager

	// Scroll links the editor to a ScrollState for internal scrolling.
	Scroll *ui.ScrollState

	// Placeholder is shown when the content is empty.
	Placeholder string
}

// Option configures a RichTextEditor.
type Option func(*RichTextEditor)

// WithOnChange sets the change callback.
func WithOnChange(fn func(AttributedString)) Option {
	return func(e *RichTextEditor) { e.OnChange = fn }
}

// WithReadOnly sets the editor to read-only mode.
func WithReadOnly() Option {
	return func(e *RichTextEditor) { e.ReadOnly = true }
}

// WithRows sets the number of visible rows.
func WithRows(n int) Option {
	return func(e *RichTextEditor) { e.Rows = n }
}

// WithFocus links the editor to a FocusManager.
func WithFocus(fm *ui.FocusManager) Option {
	return func(e *RichTextEditor) { e.Focus = fm }
}

// WithScroll links the editor to a ScrollState.
func WithScroll(s *ui.ScrollState) Option {
	return func(e *RichTextEditor) { e.Scroll = s }
}

// WithPlaceholder sets the placeholder text.
func WithPlaceholder(p string) Option {
	return func(e *RichTextEditor) { e.Placeholder = p }
}

// New creates a RichTextEditor with the given content and options.
func New(doc AttributedString, opts ...Option) ui.Element {
	e := RichTextEditor{
		Value: doc,
		Rows:  editorMinRows,
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// ── Layout ──────────────────────────────────────────────────────

// LayoutSelf implements ui.Layouter.
func (n RichTextEditor) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	th := ctx.Theme
	tokens := ctx.Tokens
	ix := ctx.IX
	focus := ctx.Focus

	bodyStyle := tokens.Typography.Body
	rows := n.Rows
	if rows < 1 {
		rows = editorMinRows
	}

	// Compute line height.
	metrics := canvas.MeasureText("Mg", bodyStyle)
	lineH := metrics.Ascent + metrics.Descent + metrics.Leading
	if bodyStyle.LineHeight > 0 {
		lineH = bodyStyle.Size * bodyStyle.LineHeight
	}

	w := area.W
	if w < editorMinW {
		w = editorMinW
	}
	viewportH := int(lineH*float32(rows)) + editorPadY*2
	h := viewportH

	// Focus management.
	var focusUID ui.UID
	if focus != nil && !n.ReadOnly {
		focusUID = focus.NextElementUID()
		focus.RegisterFocusable(focusUID, ui.FocusOpts{
			Focusable:    true,
			TabIndex:     0,
			FocusOnClick: true,
		})
	}
	focused := !n.ReadOnly && focus != nil && focus.IsElementFocused(focusUID)

	// Scroll offset.
	scrollOff := float32(0)
	if n.Scroll != nil {
		scrollOff = n.Scroll.Offset
	}

	// Plain text and lines for layout (direct from AttributedString).
	plainText := n.Value.Text
	lines := text.Lines(plainText)

	// Custom theme DrawFunc dispatch.
	if df := th.DrawFunc(theme.WidgetKindRichTextEditor); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			Focused:  focused,
			Disabled: n.ReadOnly,
		}, tokens, n)
	} else {
		n.drawDefault(ctx, area, w, h, focused, scrollOff, plainText, lines, lineH, bodyStyle)
	}

	// Store input state for the focused editor.
	if focused && n.OnChange != nil && focus != nil {
		cursorOff := len(plainText)
		selStart := -1
		if focus.Input != nil && focus.Input.FocusUID == focusUID {
			cursorOff = focus.Input.CursorOffset
			selStart = focus.Input.SelectionStart
			if cursorOff > len(plainText) {
				cursorOff = len(plainText)
			}
		}
		if focus.PendingCursorOffset >= 0 {
			cursorOff = focus.PendingCursorOffset
			if cursorOff > len(plainText) {
				cursorOff = len(plainText)
			}
			selStart = -1
			focus.PendingCursorOffset = -1
		}
		focus.Input = &ui.InputState{
			Value:          plainText,
			OnChange:       n.makeOnChange(),
			FocusUID:       focusUID,
			CursorOffset:   cursorOff,
			SelectionStart: selStart,
			Multiline:      true,
		}
	}

	// Register scroll for mouse wheel.
	if n.Scroll != nil {
		totalH := lineH * float32(len(lines))
		vpH := float32(h - editorPadY*2)
		scroll := n.Scroll
		ix.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			totalH, vpH,
			func(deltaY float32) { scroll.ScrollBy(deltaY, totalH, vpH) },
		)
	}

	// Hit target for focus acquisition and click-to-position cursor.
	if n.OnChange != nil && focus != nil && !n.ReadOnly {
		uid := focusUID
		fm := focus
		cX := float32(area.X + editorPadX)
		cY := float32(area.Y + editorPadY)
		lH := lineH
		sOff := scrollOff
		val := plainText
		ls := lines
		sty := bodyStyle

		type lineBounds struct {
			xs   []float32
			offs []int
		}
		lineBoundsArr := make([]lineBounds, len(ls))
		for i, span := range ls {
			lineText := val[span.Start:span.End]
			clusters := text.GraphemeClusters(lineText)
			xs := make([]float32, len(clusters))
			offs := make([]int, len(clusters))
			for j, boff := range clusters {
				offs[j] = span.Start + boff
				if boff == 0 {
					xs[j] = cX
				} else {
					m := canvas.MeasureText(lineText[:boff], sty)
					xs[j] = cX + m.Width
				}
			}
			lineBoundsArr[i] = lineBounds{xs: xs, offs: offs}
		}

		dragAnchor := -1
		multiClickHandled := false
		ix.RegisterDrag(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			func(mx, my float32) {
				fm.SetFocusedUID(uid)
				relY := my - cY + sOff
				line := int(relY / lH)
				if line < 0 {
					line = 0
				}
				if line >= len(ls) {
					line = len(ls) - 1
				}
				if line < 0 {
					return
				}
				lb := lineBoundsArr[line]
				off := closestBoundary(lb.xs, lb.offs, mx)
				if dragAnchor < 0 {
					clickCount := fm.TrackClick(mx, my)
					if clickCount >= 3 {
						// Triple-click: select entire line.
						multiClickHandled = true
						span := ls[line]
						dragAnchor = span.Start
						lineEnd := span.End
						// Include the newline if not the last line.
						if line < len(ls)-1 && lineEnd < len(val) {
							lineEnd++ // include '\n'
						}
						if fm.Input != nil {
							fm.Input.SelectionStart = span.Start
							fm.Input.CursorOffset = lineEnd
						}
					} else if clickCount == 2 {
						// Double-click: select word.
						multiClickHandled = true
						wStart, wEnd := text.WordAt(val, off)
						dragAnchor = wStart
						if fm.Input != nil {
							fm.Input.SelectionStart = wStart
							fm.Input.CursorOffset = wEnd
						}
					} else {
						multiClickHandled = false
						dragAnchor = off
						if fm.Input != nil {
							fm.Input.CursorOffset = off
							fm.Input.ClearSelection()
						} else {
							fm.PendingCursorOffset = off
						}
					}
				} else if !multiClickHandled {
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

// drawDefault renders the editor with the default theme appearance.
func (n RichTextEditor) drawDefault(ctx *ui.LayoutContext, area ui.Bounds, w, h int,
	focused bool, scrollOff float32, plainText string, lines []text.LineSpan,
	lineH float32, bodyStyle draw.TextStyle) {

	canvas := ctx.Canvas
	tokens := ctx.Tokens
	focus := ctx.Focus

	editorRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Border.
	borderColor := tokens.Colors.Stroke.Border
	if n.ReadOnly {
		borderColor = disabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(editorRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill.
	fillColor := tokens.Colors.Surface.Elevated
	if n.ReadOnly {
		fillColor = disabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Focus glow + ring.
	if focused {
		ui.DrawFocusRing(canvas, editorRect, tokens.Radii.Input, tokens)
	}

	// Content area.
	contentX := float32(area.X + editorPadX)
	contentY := float32(area.Y + editorPadY)
	contentW := float32(w - editorPadX*2)
	contentH := float32(h - editorPadY*2)

	// Determine cursor position.
	cursorLine := 0
	cursorX := contentX
	cursorOff := len(plainText)
	if focused && focus != nil && focus.Input != nil {
		cursorOff = focus.Input.CursorOffset
	}
	for i, span := range lines {
		if cursorOff >= span.Start && (cursorOff <= span.End || i == len(lines)-1) {
			cursorLine = i
			lineText := plainText[span.Start:span.End]
			offInLine := cursorOff - span.Start
			if offInLine > len(lineText) {
				offInLine = len(lineText)
			}
			m := canvas.MeasureText(lineText[:offInLine], bodyStyle)
			cursorX = contentX + m.Width
			break
		}
	}

	// Auto-scroll cursor into view.
	if focused && n.Scroll != nil {
		cursorTop := lineH * float32(cursorLine)
		cursorBottom := cursorTop + lineH
		totalContentH := lineH * float32(len(lines))
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
		n.drawSelection(canvas, &tokens, plainText, lines, lineH, bodyStyle,
			contentX, contentY, contentH, scrollOff, focus.Input)
	}

	// Draw rich text content with per-run styling.
	textColor := tokens.Colors.Text.Primary
	if n.ReadOnly {
		textColor = tokens.Colors.Text.Disabled
	}
	if len(plainText) > 0 {
		n.drawRichContent(canvas, plainText, lines, lineH, bodyStyle, textColor,
			contentX, contentY, contentW, contentH, scrollOff)
	} else if n.Placeholder != "" {
		canvas.DrawText(n.Placeholder, draw.Pt(contentX, contentY), bodyStyle, tokens.Colors.Text.Disabled)
	}

	// Draw cursor when focused.
	if focused {
		cy := contentY + lineH*float32(cursorLine) - scrollOff
		canvas.FillRect(draw.R(cursorX, cy, 2, lineH),
			draw.SolidPaint(tokens.Colors.Text.Primary))
	}

	canvas.PopClip()
}

// drawSelection renders the selection highlight across lines.
func (n RichTextEditor) drawSelection(canvas draw.Canvas, tokens *theme.TokenSet,
	plainText string, lines []text.LineSpan, lineH float32, bodyStyle draw.TextStyle,
	contentX, contentY, contentH, scrollOff float32, input *ui.InputState) {

	selA, selB := input.SelectionRange()
	if selA > len(plainText) {
		selA = len(plainText)
	}
	if selB > len(plainText) {
		selB = len(plainText)
	}
	selColor := tokens.Colors.Accent.Primary
	selColor.A = 0.3

	for i, span := range lines {
		y := contentY + lineH*float32(i) - scrollOff
		if y+lineH < contentY || y > contentY+contentH {
			continue
		}
		lineSelStart := selA
		if lineSelStart < span.Start {
			lineSelStart = span.Start
		}
		lineSelEnd := selB
		if lineSelEnd > span.End {
			lineSelEnd = span.End
		}
		if lineSelStart >= lineSelEnd {
			if selA <= span.End && selB > span.End && i < len(lines)-1 {
				lineText := plainText[span.Start:span.End]
				mEnd := canvas.MeasureText(lineText, bodyStyle)
				canvas.FillRect(draw.R(contentX+mEnd.Width, y, 4, lineH),
					draw.SolidPaint(selColor))
			}
			continue
		}
		lineText := plainText[span.Start:span.End]
		offA := lineSelStart - span.Start
		offB := lineSelEnd - span.Start
		mA := canvas.MeasureText(lineText[:offA], bodyStyle)
		mB := canvas.MeasureText(lineText[:offB], bodyStyle)
		sx := contentX + mA.Width
		sw := mB.Width - mA.Width
		canvas.FillRect(draw.R(sx, y, sw, lineH),
			draw.SolidPaint(selColor))
		if selB > span.End && i < len(lines)-1 {
			canvas.FillRect(draw.R(contentX+mB.Width, y, 4, lineH),
				draw.SolidPaint(selColor))
		}
	}
}

// drawRichContent renders text with per-run styling using resolved
// style runs from the AttributedString's tagged-range attributes.
func (n RichTextEditor) drawRichContent(canvas draw.Canvas,
	plainText string, lines []text.LineSpan, lineH float32,
	bodyStyle draw.TextStyle, defaultColor draw.Color,
	contentX, contentY, contentW, contentH, scrollOff float32) {

	for i, span := range lines {
		y := contentY + lineH*float32(i) - scrollOff
		if y+lineH < contentY || y > contentY+contentH {
			continue // skip lines outside viewport
		}

		if len(n.Value.Attrs) == 0 {
			lineText := plainText[span.Start:span.End]
			canvas.DrawText(lineText, draw.Pt(contentX, y), bodyStyle, defaultColor)
			continue
		}

		// Resolve styled runs for this line.
		runs := n.Value.StyleRuns(span.Start, span.End)

		// Read paragraph-level alignment from the first byte of this line.
		paraStyle := n.Value.ResolveAt(span.Start)

		// ── List marker + indent ──
		lineContentX := contentX
		lineContentW := contentW
		if paraStyle.ListType != draw.ListTypeNone {
			level := paraStyle.ListLevel
			indent := float32((level + 1) * editorListIndentStep)

			// Draw marker on first line of paragraph only.
			if isFirstLineOfParagraph(i, lines, plainText) {
				itemIdx := countEditorListItemIndex(n.Value, span.Start)
				marker := resolveEditorListMarker(paraStyle, itemIdx)
				if marker != "" {
					markerX := contentX + float32(level*editorListIndentStep)
					canvas.DrawText(marker, draw.Pt(markerX, y), bodyStyle, defaultColor)
				}
			}

			lineContentX += indent
			lineContentW -= indent
		}

		// ── Pre-pass: measure total line width for alignment ──
		type runMeasure struct {
			style   draw.TextStyle
			color   draw.Color
			ss      SpanStyle
			text    string
			width   float32
			isImage bool
		}
		measured := make([]runMeasure, len(runs))
		lineWidth := float32(0)

		for j, run := range runs {
			rStart := run.Start
			if rStart < span.Start {
				rStart = span.Start
			}
			rEnd := run.End
			if rEnd > span.End {
				rEnd = span.End
			}
			if rStart >= rEnd {
				continue
			}

			rm := runMeasure{ss: run.Style}

			// Image run.
			if run.Style.Image.ImageID != 0 {
				img := run.Style.Image
				imgH := img.Height
				if imgH == 0 {
					imgH = lineH
				}
				imgW := img.Width
				if imgW == 0 {
					imgW = imgH
				}
				rm.isImage = true
				rm.width = imgW
				lineWidth += imgW
				measured[j] = rm
				continue
			}

			// Text run — resolve draw.TextStyle.
			rm.text = plainText[rStart:rEnd]
			style := bodyStyle
			color := defaultColor

			if run.Style.FontFamily != "" {
				style.FontFamily = run.Style.FontFamily
			}
			if run.Style.Weight > 0 {
				style.Weight = run.Style.Weight
			} else if run.Style.Bold {
				style.Weight = draw.FontWeightBold
			}
			if run.Style.Italic {
				style.Style = draw.FontStyleItalic
			}
			if run.Style.Size > 0 {
				style.Size = run.Style.Size
			}
			if run.Style.Tracking != 0 {
				style.Tracking = run.Style.Tracking
			}
			if run.Style.LineHeight > 0 {
				style.LineHeight = run.Style.LineHeight
			}
			if run.Style.Color.A > 0 {
				color = run.Style.Color
			}

			m := canvas.MeasureText(rm.text, style)
			rm.style = style
			rm.color = color
			rm.width = m.Width
			lineWidth += m.Width
			measured[j] = rm
		}

		// ── Compute alignment offset ──
		x := lineContentX

		// First-line indent.
		if paraStyle.Indent > 0 && isFirstLineOfParagraph(i, lines, plainText) {
			x += paraStyle.Indent
		}

		isLastLine := i == len(lines)-1 || (i < len(lines)-1 && span.End < len(plainText) && plainText[span.End] == '\n')
		justifyGap := float32(0)

		switch paraStyle.Align {
		case draw.TextAlignCenter:
			x += (lineContentW - lineWidth) / 2
		case draw.TextAlignRight:
			x += lineContentW - lineWidth
		case draw.TextAlignJustify:
			if !isLastLine && len(runs) > 1 {
				extra := lineContentW - lineWidth
				if extra > 0 {
					justifyGap = extra / float32(len(runs)-1)
				}
			}
		}

		// ── Paint runs ──
		for j, rm := range measured {
			if rm.width == 0 && rm.text == "" && !rm.isImage {
				continue
			}

			if rm.isImage {
				img := rm.ss.Image
				imgH := img.Height
				if imgH == 0 {
					imgH = lineH
				}
				imgW := img.Width
				if imgW == 0 {
					imgW = imgH
				}
				op := img.Opacity
				if op == 0 {
					op = 1.0
				}
				imgY := y + (lineH-imgH)/2
				canvas.DrawImageScaled(img.ImageID,
					draw.R(x, imgY, imgW, imgH),
					img.ScaleMode,
					draw.ImageOptions{Opacity: op})
				x += imgW
			} else {
				m := canvas.MeasureText(rm.text, rm.style)

				if rm.ss.BgColor.A > 0 {
					canvas.FillRect(draw.R(x, y, m.Width, lineH),
						draw.SolidPaint(rm.ss.BgColor))
				}

				canvas.DrawText(rm.text, draw.Pt(x, y), rm.style, rm.color)

				if rm.ss.Bold {
					canvas.DrawText(rm.text, draw.Pt(x+1, y), rm.style, rm.color)
				}

				if rm.ss.Underline {
					ulY := y + m.Ascent + 1
					canvas.FillRect(draw.R(x, ulY, m.Width, 1),
						draw.SolidPaint(rm.color))
				}

				if rm.ss.Strikethrough {
					stY := y + m.Ascent*0.65
					canvas.FillRect(draw.R(x, stY, m.Width, 1),
						draw.SolidPaint(rm.color))
				}

				x += m.Width
			}

			if justifyGap > 0 && j < len(measured)-1 {
				x += justifyGap
			}
		}
	}
}

// isFirstLineOfParagraph returns true if lines[i] is the first line
// of a paragraph (i.e., preceded by \n or at the start of text).
func isFirstLineOfParagraph(i int, lines []text.LineSpan, plainText string) bool {
	if i == 0 {
		return true
	}
	prevEnd := lines[i-1].End
	return prevEnd > 0 && prevEnd <= len(plainText) && plainText[prevEnd-1] == '\n'
}

// ── List Helpers ────────────────────────────────────────────────

const editorListIndentStep = 24 // dp per nesting level

// countEditorListItemIndex counts preceding list items of the same
// type and level by scanning paragraphs backwards from offset.
func countEditorListItemIndex(doc AttributedString, offset int) int {
	curStyle := doc.ResolveAt(offset)
	count := 0
	pos := offset
	for {
		start, _ := ParagraphRange(doc.Text, pos)
		if start == 0 {
			break
		}
		pos = start - 1 // move to \n before this paragraph
		prevStyle := doc.ResolveAt(pos)
		if prevStyle.ListType != curStyle.ListType || prevStyle.ListLevel != curStyle.ListLevel {
			break
		}
		count++
	}
	return count
}

// resolveEditorListMarker returns the marker string for a list item
// in the editor. Uses the same CSS-like logic as the display layer.
func resolveEditorListMarker(style SpanStyle, itemIndex int) string {
	marker := style.ListMarker
	if marker == draw.ListMarkerDefault {
		if style.ListType == draw.ListTypeUnordered {
			switch style.ListLevel % 3 {
			case 0:
				return "\u2022" // •
			case 1:
				return "\u25E6" // ◦
			case 2:
				return "\u25AA" // ▪
			}
		}
		marker = draw.ListMarkerDecimal
	}

	switch marker {
	case draw.ListMarkerDisc:
		return "\u2022"
	case draw.ListMarkerCircle:
		return "\u25E6"
	case draw.ListMarkerSquare:
		return "\u25AA"
	case draw.ListMarkerDecimal:
		start := style.ListStart
		if start == 0 {
			start = 1
		}
		return strconv.Itoa(start+itemIndex) + "."
	case draw.ListMarkerLowerAlpha:
		return string(rune('a'+itemIndex%26)) + "."
	case draw.ListMarkerUpperAlpha:
		return string(rune('A'+itemIndex%26)) + "."
	case draw.ListMarkerLowerRoman:
		return toLowerRoman(itemIndex+1) + "."
	case draw.ListMarkerUpperRoman:
		return strings.ToUpper(toLowerRoman(itemIndex+1)) + "."
	case draw.ListMarkerNone:
		return ""
	}
	return "\u2022"
}

// toLowerRoman converts n (1-based) to a lowercase Roman numeral string.
func toLowerRoman(n int) string {
	if n <= 0 || n > 3999 {
		return strconv.Itoa(n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

// makeOnChange wraps the AttributedString-based OnChange into a
// string-based callback compatible with InputState. The key advantage
// of AttributedString: we diff the text edit and use InsertText /
// DeleteRange to update runs — no information is lost.
func (n RichTextEditor) makeOnChange() func(string) {
	onChange := n.OnChange
	if onChange == nil {
		return nil
	}
	return func(newText string) {
		orig := n.Value
		oldText := orig.Text
		if oldText == newText {
			onChange(orig)
			return
		}

		// Find edit region via common prefix/suffix.
		pfx := commonPrefixLen(oldText, newText)
		sfx := commonSuffixLen(oldText, newText, pfx)
		delStart := pfx
		delEnd := len(oldText) - sfx
		ins := newText[pfx : len(newText)-sfx]

		// Apply the edit: delete then insert.
		result := orig
		if delEnd > delStart {
			result = result.DeleteRange(delStart, delEnd)
		}
		if len(ins) > 0 {
			result = result.InsertText(delStart, ins)
		}

		onChange(result)
	}
}

func commonPrefixLen(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func commonSuffixLen(a, b string, pfx int) int {
	n := 0
	for n < len(a)-pfx && n < len(b)-pfx &&
		a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	return n
}

// ── Interface implementations ───────────────────────────────────

// TreeEqual implements ui.TreeEqualizer.
func (n RichTextEditor) TreeEqual(other ui.Element) bool {
	nb, ok := other.(RichTextEditor)
	if !ok {
		return false
	}
	return n.Value.Equal(nb.Value) &&
		n.ReadOnly == nb.ReadOnly &&
		n.Rows == nb.Rows &&
		n.Placeholder == nb.Placeholder
}

// ResolveChildren implements ui.ChildResolver. RichTextEditor is a leaf.
func (n RichTextEditor) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n RichTextEditor) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := n.Placeholder
	if label == "" {
		label = "Rich text editor"
	}
	plainText := n.Value.Text
	an := a11y.AccessNode{
		Role:  a11y.RoleTextInput,
		Label: label,
		Value: plainText,
		States: a11y.AccessStates{
			Disabled: n.ReadOnly,
		},
		TextState: &a11y.AccessTextState{
			Length:      len([]rune(plainText)),
			CaretOffset: -1,
		},
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

// ── Helpers ─────────────────────────────────────────────────────

func closestBoundary(xs []float32, offsets []int, mx float32) int {
	best := 0
	bestDist := float32(math.MaxFloat32)
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
	if len(offsets) > 0 {
		return offsets[len(offsets)-1]
	}
	return 0
}

func disabledColor(fg, bg draw.Color) draw.Color {
	return ui.DisabledColor(fg, bg)
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
