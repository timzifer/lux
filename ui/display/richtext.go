package display

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// SpanStyle overrides text style and color for a Span within a Paragraph.
// Zero values inherit from the theme's Body style.
type SpanStyle struct {
	Style draw.TextStyle // zero = inherit from theme Body
	Color draw.Color     // zero = use theme Text.Primary
}

// Span is a styled run of text within a Paragraph.
type Span struct {
	Text  string
	Style SpanStyle
}

// ── Inline Widgets (RFC-003 §5.5) ──────────────────────────────

// ParagraphContent is the sealed interface for items that can appear
// in a RichParagraph's Content slice: text spans and inline widgets.
type ParagraphContent interface{ isParagraphContent() }

func (Span) isParagraphContent()         {}
func (InlineWidget) isParagraphContent() {}

// InlineWidget embeds an arbitrary Element in the text flow.
// Width and height are determined by the element itself via intrinsic
// measurement (MeasureChild). Baseline alignment: the bottom edge of
// the widget sits on the text baseline, shifted up by Baseline dp.
type InlineWidget struct {
	Element  ui.Element
	Baseline float32 // 0 = bottom on text baseline; positive = shift up
}

// InlineElement creates an InlineWidget with default baseline alignment.
func InlineElement(el ui.Element) InlineWidget {
	return InlineWidget{Element: el}
}

// InlineElementWithBaseline creates an InlineWidget with a custom baseline offset.
func InlineElementWithBaseline(el ui.Element, baseline float32) InlineWidget {
	return InlineWidget{Element: el, Baseline: baseline}
}

// ── RichParagraph & RichTextElement ─────────────────────────────

// RichParagraph is a block-level text unit containing styled spans
// and/or inline widgets.
type RichParagraph struct {
	Spans   []Span             // legacy text-only content
	Content []ParagraphContent // mixed content (takes precedence over Spans)
}

// RichTextElement renders read-only rich-formatted text.
type RichTextElement struct {
	ui.BaseElement
	Paragraphs []RichParagraph
}

// RichText creates a read-only rich-formatted text element.
func RichText(paragraphs ...RichParagraph) ui.Element {
	return RichTextElement{Paragraphs: paragraphs}
}

// RichTextSpans is a convenience for single-paragraph rich text.
func RichTextSpans(spans ...Span) ui.Element {
	return RichTextElement{Paragraphs: []RichParagraph{{Spans: spans}}}
}

// paragraphContent returns the effective content slice for a paragraph,
// preferring Content if non-empty, otherwise converting Spans.
func paragraphContent(p RichParagraph) []ParagraphContent {
	if len(p.Content) > 0 {
		return p.Content
	}
	out := make([]ParagraphContent, len(p.Spans))
	for i, s := range p.Spans {
		out[i] = s
	}
	return out
}

// ── Layout ──────────────────────────────────────────────────────

// lineItem holds measured data for a single content item on a line.
type lineItem struct {
	content  ParagraphContent
	w, h     int     // measured width and height
	ascent   int     // text ascent (0 for widgets)
	baseline float32 // InlineWidget.Baseline offset (0 for spans)
	style    draw.TextStyle
	color    draw.Color
}

// LayoutSelf implements ui.Layouter.
func (n RichTextElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if len(n.Paragraphs) == 0 {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	bodyStyle := ctx.Tokens.Typography.Body
	primaryColor := ctx.Tokens.Colors.Text.Primary
	paraSpacing := int(ctx.Tokens.Spacing.S)
	if paraSpacing <= 0 {
		paraSpacing = 8
	}

	cursorY := ctx.Area.Y
	maxW := 0

	for pIdx, para := range n.Paragraphs {
		content := paragraphContent(para)
		if len(content) == 0 {
			if pIdx < len(n.Paragraphs)-1 {
				cursorY += paraSpacing
			}
			continue
		}

		// Measure all items in this paragraph.
		items := make([]lineItem, len(content))
		for i, c := range content {
			switch v := c.(type) {
			case Span:
				style := resolveSpanStyle(bodyStyle, v.Style)
				color := primaryColor
				if v.Style.Color.A > 0 {
					color = v.Style.Color
				}
				metrics := ctx.Canvas.MeasureText(v.Text, style)
				w := int(math.Ceil(float64(metrics.Width)))
				h := int(math.Ceil(float64(metrics.Ascent)))
				items[i] = lineItem{content: v, w: w, h: h, ascent: h, style: style, color: color}

			case InlineWidget:
				measured := ctx.MeasureChild(v.Element, ui.Bounds{
					X: 0, Y: 0,
					W: ctx.Area.W, H: ctx.Area.H,
				})
				items[i] = lineItem{
					content:  v,
					w:        measured.W,
					h:        measured.H,
					baseline: v.Baseline,
				}
			}
		}

		// Break items into lines and paint each line with baseline alignment.
		availW := ctx.Area.W
		lineStart := 0

		for lineStart < len(items) {
			// Determine which items fit on this line.
			lineEnd := lineStart
			lineW := 0
			for lineEnd < len(items) {
				itemW := items[lineEnd].w
				if lineEnd > lineStart && lineW+itemW > availW {
					break
				}
				lineW += itemW
				lineEnd++
			}
			// Safety: at least one item per line to avoid infinite loops.
			if lineEnd == lineStart {
				lineEnd = lineStart + 1
				lineW = items[lineStart].w
			}

			// Compute line metrics: max ascent (text baseline) and max descent/widget overshoot.
			lineAscent := 0  // max text ascent = baseline position from line top
			lineDescent := 0 // max space below baseline

			for i := lineStart; i < lineEnd; i++ {
				it := items[i]
				switch it.content.(type) {
				case Span:
					if it.ascent > lineAscent {
						lineAscent = it.ascent
					}
					// For text, descent is 0 in this simplified model (ascent ≈ line height).

				case InlineWidget:
					// Widget bottom sits on baseline, shifted up by it.baseline.
					// Space above baseline = widget height - baseline offset (but at least 0).
					aboveBaseline := it.h - int(math.Round(float64(it.baseline)))
					if aboveBaseline < 0 {
						aboveBaseline = 0
					}
					belowBaseline := int(math.Round(float64(it.baseline)))
					if belowBaseline < 0 {
						belowBaseline = 0
					}
					if aboveBaseline > lineAscent {
						lineAscent = aboveBaseline
					}
					if belowBaseline > lineDescent {
						lineDescent = belowBaseline
					}
				}
			}

			// Fallback: ensure minimum line height from body style.
			defaultLineH := int(bodyStyle.Size)
			if defaultLineH <= 0 {
				defaultLineH = 14
			}
			if lineAscent < defaultLineH {
				lineAscent = defaultLineH
			}

			lineH := lineAscent + lineDescent

			// Paint items on this line.
			cursorX := ctx.Area.X
			for i := lineStart; i < lineEnd; i++ {
				it := items[i]
				switch v := it.content.(type) {
				case Span:
					// Text is drawn at the top of the line (baseline-aligned
					// via the text renderer's own ascent handling).
					drawY := cursorY + (lineAscent - it.ascent)
					ctx.Canvas.DrawText(v.Text,
						draw.Pt(float32(cursorX), float32(drawY)),
						it.style, it.color)

				case InlineWidget:
					// Position widget so its bottom edge sits on the text baseline,
					// then shift up by the Baseline offset.
					// Baseline is at cursorY + lineAscent.
					widgetTop := cursorY + lineAscent - it.h + int(math.Round(float64(it.baseline)))
					if widgetTop < cursorY {
						widgetTop = cursorY
					}
					ctx.LayoutChild(v.Element, ui.Bounds{
						X: cursorX, Y: widgetTop,
						W: it.w, H: it.h,
					})
				}
				cursorX += it.w
			}

			if lineW > maxW {
				maxW = lineW
			}

			cursorY += lineH
			lineStart = lineEnd
		}

		if pIdx < len(n.Paragraphs)-1 {
			cursorY += paraSpacing
		}
	}

	totalH := cursorY - ctx.Area.Y
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: maxW, H: totalH}
}

// resolveSpanStyle merges a SpanStyle with the theme body style.
func resolveSpanStyle(body draw.TextStyle, ss SpanStyle) draw.TextStyle {
	if ss.Style.Size > 0 {
		return ss.Style
	}
	style := body
	if ss.Style.Weight > 0 {
		style.Weight = ss.Style.Weight
	}
	if ss.Style.FontFamily != "" {
		style.FontFamily = ss.Style.FontFamily
	}
	return style
}

// ── TreeEqual ───────────────────────────────────────────────────

// TreeEqual implements ui.TreeEqualizer.
func (n RichTextElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(RichTextElement)
	if !ok || len(n.Paragraphs) != len(nb.Paragraphs) {
		return false
	}
	for i, p := range n.Paragraphs {
		op := nb.Paragraphs[i]
		ac := paragraphContent(p)
		bc := paragraphContent(op)
		if len(ac) != len(bc) {
			return false
		}
		for j, ca := range ac {
			if !contentEqual(ca, bc[j]) {
				return false
			}
		}
	}
	return true
}

// contentEqual compares two ParagraphContent items for structural equality.
func contentEqual(a, b ParagraphContent) bool {
	switch va := a.(type) {
	case Span:
		vb, ok := b.(Span)
		return ok && va.Text == vb.Text && va.Style == vb.Style
	case InlineWidget:
		vb, ok := b.(InlineWidget)
		if !ok || va.Baseline != vb.Baseline {
			return false
		}
		return elementsEqual(va.Element, vb.Element)
	}
	return false
}

// elementsEqual compares two Elements for structural equality using
// the TreeEqualizer interface when available.
func elementsEqual(a, b ui.Element) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if te, ok := a.(ui.TreeEqualizer); ok {
		return te.TreeEqual(b)
	}
	return false
}

// ── ResolveChildren ─────────────────────────────────────────────

// ResolveChildren implements ui.ChildResolver.
// Inline widgets are resolved recursively; text spans are leaves.
func (n RichTextElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	changed := false
	paragraphs := make([]RichParagraph, len(n.Paragraphs))
	childIdx := 0

	for i, p := range n.Paragraphs {
		if len(p.Content) == 0 {
			paragraphs[i] = p
			continue
		}
		content := make([]ParagraphContent, len(p.Content))
		for j, c := range p.Content {
			if iw, ok := c.(InlineWidget); ok {
				resolved := resolve(iw.Element, childIdx)
				childIdx++
				if resolved != iw.Element {
					changed = true
					content[j] = InlineWidget{Element: resolved, Baseline: iw.Baseline}
				} else {
					content[j] = c
				}
			} else {
				content[j] = c
			}
		}
		paragraphs[i] = RichParagraph{Content: content, Spans: p.Spans}
	}

	if !changed {
		return n
	}
	return RichTextElement{BaseElement: n.BaseElement, Paragraphs: paragraphs}
}

// ── Accessibility ───────────────────────────────────────────────

// WalkAccess implements ui.AccessWalker.
// Walks inline widget children for accessibility tree building.
func (n RichTextElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, p := range n.Paragraphs {
		for _, c := range p.Content {
			if iw, ok := c.(InlineWidget); ok {
				if aw, ok := iw.Element.(ui.AccessWalker); ok {
					aw.WalkAccess(b, parentIdx)
				}
			}
		}
	}
}
