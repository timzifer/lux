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
// The Document lives in the user model; internal state (cursor,
// selection, undo/redo) lives in the framework's WidgetState.
type RichTextEditor struct {
	ui.BaseElement

	// Value is the current document content (user-model owned).
	Value Document

	// OnChange is called when the user edits the document.
	OnChange func(Document)

	// ReadOnly accepts no input but allows selection and copy.
	ReadOnly bool

	// Rows controls the visible row count (default 4).
	Rows int

	// Focus links the editor to a FocusManager for keyboard input.
	Focus *ui.FocusManager

	// Scroll links the editor to a ScrollState for internal scrolling.
	Scroll *ui.ScrollState

	// Placeholder is shown when the document is empty.
	Placeholder string

	// Toolbar is an optional formatting toolbar configuration.
	Toolbar *EditorToolbar
}

// EditorToolbar configures the optional formatting toolbar.
type EditorToolbar struct {
	Bold      bool // show bold toggle
	Italic    bool // show italic toggle
	Underline bool // show underline toggle
}

// Option configures a RichTextEditor.
type Option func(*RichTextEditor)

// WithOnChange sets the document change callback.
func WithOnChange(fn func(Document)) Option {
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

// WithToolbar enables the formatting toolbar.
func WithToolbar(tb *EditorToolbar) Option {
	return func(e *RichTextEditor) { e.Toolbar = tb }
}

// New creates a RichTextEditor with the given document and options.
func New(doc Document, opts ...Option) ui.Element {
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

	// Flatten document to plain text for cursor/selection operations.
	plainText := n.Value.PlainText()
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
		n.drawSelection(canvas, tokens, plainText, lines, lineH, bodyStyle,
			contentX, contentY, contentH, scrollOff, focus.Input)
	}

	// Draw rich text content with per-span styling.
	textColor := tokens.Colors.Text.Primary
	if n.ReadOnly {
		textColor = tokens.Colors.Text.Disabled
	}
	if len(n.Value.Paragraphs) > 0 && plainText != "" {
		n.drawRichContent(canvas, tokens, lines, lineH, bodyStyle, textColor,
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

// drawRichContent renders paragraphs with per-span styling.
func (n RichTextEditor) drawRichContent(canvas draw.Canvas, tokens *theme.TokenSet,
	lines []text.LineSpan, lineH float32, bodyStyle draw.TextStyle, defaultColor draw.Color,
	contentX, contentY, contentH, scrollOff float32) {

	// Build a flat list of styled runs from paragraphs, accounting for \n separators.
	type styledRun struct {
		start int // byte offset in plain text
		end   int
		style SpanStyle
	}
	var runs []styledRun
	offset := 0
	for i, p := range n.Value.Paragraphs {
		if i > 0 {
			offset++ // \n separator
		}
		for _, s := range p.Spans {
			runs = append(runs, styledRun{
				start: offset,
				end:   offset + len(s.Text),
				style: s.Style,
			})
			offset += len(s.Text)
		}
		if len(p.Spans) == 0 {
			// Empty paragraph — no runs to add.
		}
	}

	plainText := n.Value.PlainText()

	for i, span := range lines {
		y := contentY + lineH*float32(i) - scrollOff
		if y+lineH < contentY || y > contentY+contentH {
			continue // skip lines outside viewport
		}

		// Find styled runs that overlap this line.
		x := contentX
		for _, run := range runs {
			// Compute overlap of this run with the line.
			rStart := run.start
			if rStart < span.Start {
				rStart = span.Start
			}
			rEnd := run.end
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
			if run.style.Bold {
				style.Weight = draw.FontWeightBold
			}
			// Note: Italic is tracked in SpanStyle but draw.TextStyle
			// does not yet support italic; style will be applied when
			// the text stack gains italic support.
			if run.style.Size > 0 {
				style.Size = run.style.Size
			}
			if run.style.Color.A > 0 {
				color = run.style.Color
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
			x += m.Width
		}

		// If no runs covered this line (e.g. all default), draw with body style.
		if len(runs) == 0 {
			lineText := plainText[span.Start:span.End]
			canvas.DrawText(lineText, draw.Pt(contentX, y), bodyStyle, defaultColor)
		}
	}
}

// makeOnChange wraps the Document-based OnChange into a string-based callback
// compatible with InputState.
func (n RichTextEditor) makeOnChange() func(string) {
	onChange := n.OnChange
	if onChange == nil {
		return nil
	}
	return func(newText string) {
		doc := NewDocument(newText)
		// Preserve span styles from the original document where possible.
		// For simple edits, we copy styles from corresponding original paragraphs.
		orig := n.Value
		for i := range doc.Paragraphs {
			if i < len(orig.Paragraphs) && len(orig.Paragraphs[i].Spans) > 0 {
				// If the original had styled spans, apply the first span's style
				// to the new single-span paragraph as a best-effort preservation.
				if len(doc.Paragraphs[i].Spans) == 1 && len(orig.Paragraphs[i].Spans) == 1 {
					doc.Paragraphs[i].Spans[0].Style = orig.Paragraphs[i].Spans[0].Style
				}
			}
		}
		onChange(doc)
	}
}

// ── Interface implementations ───────────────────────────────────

// TreeEqual implements ui.TreeEqualizer.
func (n RichTextEditor) TreeEqual(other ui.Element) bool {
	nb, ok := other.(RichTextEditor)
	if !ok {
		return false
	}
	return documentsEqual(n.Value, nb.Value) &&
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
	an := a11y.AccessNode{
		Role:  a11y.RoleTextInput,
		Label: label,
		Value: n.Value.PlainText(),
		States: a11y.AccessStates{
			Disabled: n.ReadOnly,
		},
		TextState: &a11y.AccessTextState{
			Length:      len([]rune(n.Value.PlainText())),
			CaretOffset: -1,
		},
	}
	b.AddNode(an, parentIdx, a11y.Rect{})
}

// ── Helpers ─────────────────────────────────────────────────────

func documentsEqual(a, b Document) bool {
	if len(a.Paragraphs) != len(b.Paragraphs) {
		return false
	}
	for i, pa := range a.Paragraphs {
		pb := b.Paragraphs[i]
		if len(pa.Spans) != len(pb.Spans) {
			return false
		}
		for j, sa := range pa.Spans {
			sb := pb.Spans[j]
			if sa.Text != sb.Text || sa.Style != sb.Style {
				return false
			}
		}
	}
	return true
}

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
