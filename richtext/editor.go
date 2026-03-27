package richtext

import (
	"math"

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
			contentX, contentY, contentH, scrollOff)
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

// drawRichContent renders text with per-run styling directly from
// the AttributedString's Attrs — no intermediate conversion needed.
func (n RichTextEditor) drawRichContent(canvas draw.Canvas,
	plainText string, lines []text.LineSpan, lineH float32,
	bodyStyle draw.TextStyle, defaultColor draw.Color,
	contentX, contentY, contentH, scrollOff float32) {

	attrs := n.Value.Attrs

	for i, span := range lines {
		y := contentY + lineH*float32(i) - scrollOff
		if y+lineH < contentY || y > contentY+contentH {
			continue // skip lines outside viewport
		}

		if len(attrs) == 0 {
			// No attrs — draw with body style.
			lineText := plainText[span.Start:span.End]
			canvas.DrawText(lineText, draw.Pt(contentX, y), bodyStyle, defaultColor)
			continue
		}

		// Walk attribute runs that overlap this line.
		x := contentX
		prevEnd := 0
		for _, run := range attrs {
			runStart := prevEnd
			runEnd := run.End
			prevEnd = runEnd

			// Clip to line bounds.
			rStart := runStart
			if rStart < span.Start {
				rStart = span.Start
			}
			rEnd := runEnd
			if rEnd > span.End {
				rEnd = span.End
			}
			if rStart >= rEnd {
				continue
			}

			runText := plainText[rStart:rEnd]

			// Resolve style.
			style := bodyStyle
			color := defaultColor
			if run.Style.Bold {
				style.Weight = draw.FontWeightBold
			}
			if run.Style.Size > 0 {
				style.Size = run.Style.Size
			}
			if run.Style.Color.A > 0 {
				color = run.Style.Color
			}

			// Measure prefix to get correct X position.
			lineText := plainText[span.Start:span.End]
			prefixLen := rStart - span.Start
			if prefixLen > 0 {
				m := canvas.MeasureText(lineText[:prefixLen], bodyStyle)
				x = contentX + m.Width
			}

			canvas.DrawText(runText, draw.Pt(x, y), style, color)

			m := canvas.MeasureText(runText, style)

			// Synthetic bold: draw text a second time offset by 1dp.
			// This thickens strokes when no dedicated bold font
			// face is available in the font family.
			if run.Style.Bold {
				canvas.DrawText(runText, draw.Pt(x+1, y), style, color)
			}

			// Underline: draw a 1dp line below the text baseline.
			if run.Style.Underline {
				ulY := y + lineH - 2
				canvas.FillRect(draw.R(x, ulY, m.Width, 1),
					draw.SolidPaint(color))
			}

			x += m.Width
		}
	}
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
