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

// RichParagraph is a block-level text unit containing styled spans.
type RichParagraph struct {
	Spans []Span
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
		cursorX := ctx.Area.X
		lineH := int(bodyStyle.Size)
		if lineH <= 0 {
			lineH = 14
		}

		for _, span := range para.Spans {
			// Resolve style: use span override or fallback to theme Body.
			style := bodyStyle
			if span.Style.Style.Size > 0 {
				style = span.Style.Style
			} else {
				// Merge non-zero fields from span style.
				if span.Style.Style.Weight > 0 {
					style.Weight = span.Style.Style.Weight
				}
				if span.Style.Style.FontFamily != "" {
					style.FontFamily = span.Style.Style.FontFamily
				}
			}

			color := primaryColor
			if span.Style.Color.A > 0 {
				color = span.Style.Color
			}

			metrics := ctx.Canvas.MeasureText(span.Text, style)
			spanW := int(math.Ceil(float64(metrics.Width)))
			spanH := int(math.Ceil(float64(metrics.Ascent)))
			if spanH > lineH {
				lineH = spanH
			}

			// Wrap at span boundary if exceeds available width.
			if cursorX > ctx.Area.X && cursorX+spanW > ctx.Area.X+ctx.Area.W {
				cursorX = ctx.Area.X
				cursorY += lineH
			}

			ctx.Canvas.DrawText(span.Text,
				draw.Pt(float32(cursorX), float32(cursorY)),
				style, color)

			cursorX += spanW
			if cursorX-ctx.Area.X > maxW {
				maxW = cursorX - ctx.Area.X
			}
		}

		cursorY += lineH
		if pIdx < len(n.Paragraphs)-1 {
			cursorY += paraSpacing
		}
	}

	totalH := cursorY - ctx.Area.Y
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: maxW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n RichTextElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(RichTextElement)
	if !ok || len(n.Paragraphs) != len(nb.Paragraphs) {
		return false
	}
	for i, p := range n.Paragraphs {
		op := nb.Paragraphs[i]
		if len(p.Spans) != len(op.Spans) {
			return false
		}
		for j, s := range p.Spans {
			os := op.Spans[j]
			if s.Text != os.Text || s.Style != os.Style {
				return false
			}
		}
	}
	return true
}

// ResolveChildren implements ui.ChildResolver. RichTextElement is a leaf.
func (n RichTextElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op for rich text.
func (n RichTextElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
