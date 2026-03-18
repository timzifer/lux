package ui

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// SpanStyle overrides text style and color for a Span within a Paragraph.
// Zero values inherit from the theme's Body style.
type SpanStyle struct {
	Style draw.TextStyle // zero = inherit from theme Body
	Color draw.Color     // zero = use theme Text.Primary
}

// Span is a styled run of text within a Paragraph (RFC-003 §5.2).
type Span struct {
	Text  string
	Style SpanStyle
}

// RichParagraph is a block-level text unit containing styled spans (RFC-003 §5.2).
type RichParagraph struct {
	Spans []Span
}

// RichText creates a read-only rich-formatted text element (RFC-003 §5.4, M5).
func RichText(paragraphs ...RichParagraph) Element {
	return richTextElement{Paragraphs: paragraphs}
}

// RichTextSpans is a convenience for single-paragraph rich text.
func RichTextSpans(spans ...Span) Element {
	return richTextElement{Paragraphs: []RichParagraph{{Spans: spans}}}
}

type richTextElement struct {
	Paragraphs []RichParagraph
}

func (richTextElement) isElement() {}

func layoutRichText(node richTextElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet) bounds {
	if len(node.Paragraphs) == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	bodyStyle := tokens.Typography.Body
	primaryColor := tokens.Colors.Text.Primary
	paraSpacing := int(tokens.Spacing.S)
	if paraSpacing <= 0 {
		paraSpacing = 8
	}

	cursorY := area.Y
	maxW := 0

	for pIdx, para := range node.Paragraphs {
		cursorX := area.X
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

			metrics := canvas.MeasureText(span.Text, style)
			spanW := int(math.Ceil(float64(metrics.Width)))
			spanH := int(math.Ceil(float64(metrics.Ascent)))
			if spanH > lineH {
				lineH = spanH
			}

			// Wrap at span boundary if exceeds available width.
			if cursorX > area.X && cursorX+spanW > area.X+area.W {
				cursorX = area.X
				cursorY += lineH
			}

			canvas.DrawText(span.Text,
				draw.Pt(float32(cursorX), float32(cursorY)),
				style, color)

			cursorX += spanW
			if cursorX-area.X > maxW {
				maxW = cursorX - area.X
			}
		}

		cursorY += lineH
		if pIdx < len(node.Paragraphs)-1 {
			cursorY += paraSpacing
		}
	}

	totalH := cursorY - area.Y
	return bounds{X: area.X, Y: area.Y, W: maxW, H: totalH}
}
