package richtext

import (
	"github.com/timzifer/lux/draw"
)

// ── Document Model (user-facing, serializable) ─────────────────

// Document is the serializable document content.
// It lives in the user model and can be persisted (RFC-003 §5.6).
type Document struct {
	Paragraphs []Paragraph
}

// Paragraph is a block-level unit containing styled text spans.
type Paragraph struct {
	Spans []Span
}

// Span is a styled run of text within a Paragraph.
type Span struct {
	Text  string
	Style SpanStyle
}

// SpanStyle overrides text style for a Span.
// Zero values inherit from the theme's Body style.
type SpanStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Color     draw.Color // zero = theme Text.Primary
	Size      float32    // zero = inherit from theme Body
}

// DocumentChangedMsg is sent when the user edits the document.
// The user model should replace its Document value with this.
type DocumentChangedMsg struct {
	Document Document
}

// ── Document helpers ────────────────────────────────────────────

// NewDocument creates a Document from plain text, splitting on newlines.
func NewDocument(text string) Document {
	if text == "" {
		return Document{Paragraphs: []Paragraph{{}}}
	}
	var paragraphs []Paragraph
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == '\n' {
			paragraphs = append(paragraphs, Paragraph{
				Spans: []Span{{Text: text[start:i]}},
			})
			start = i + 1
		}
	}
	return Document{Paragraphs: paragraphs}
}

// PlainText returns the concatenated plain text of the document.
func (d Document) PlainText() string {
	if len(d.Paragraphs) == 0 {
		return ""
	}
	var buf []byte
	for i, p := range d.Paragraphs {
		if i > 0 {
			buf = append(buf, '\n')
		}
		for _, s := range p.Spans {
			buf = append(buf, s.Text...)
		}
	}
	return string(buf)
}

// paragraphText returns the plain text of a single paragraph.
func paragraphText(p Paragraph) string {
	if len(p.Spans) == 1 {
		return p.Spans[0].Text
	}
	var buf []byte
	for _, s := range p.Spans {
		buf = append(buf, s.Text...)
	}
	return string(buf)
}

// paragraphLen returns the byte length of the paragraph's text.
func paragraphLen(p Paragraph) int {
	n := 0
	for _, s := range p.Spans {
		n += len(s.Text)
	}
	return n
}

// ── Cursor & Selection ──────────────────────────────────────────

// CursorPosition addresses a byte offset within a paragraph.
type CursorPosition struct {
	Paragraph int // index into Document.Paragraphs
	Offset    int // byte offset within the paragraph's plain text
}

// Selection represents a text selection via anchor and focus positions.
// When nil / zero, no selection is active.
type Selection struct {
	Anchor CursorPosition
	Focus  CursorPosition
}

// HasSelection returns true if the selection covers a non-empty range.
func (s Selection) HasSelection() bool {
	return s.Anchor != s.Focus
}

// Ordered returns (start, end) with start <= end.
func (s Selection) Ordered() (CursorPosition, CursorPosition) {
	if s.Anchor.Paragraph < s.Focus.Paragraph ||
		(s.Anchor.Paragraph == s.Focus.Paragraph && s.Anchor.Offset <= s.Focus.Offset) {
		return s.Anchor, s.Focus
	}
	return s.Focus, s.Anchor
}

// ── Undo / Redo ─────────────────────────────────────────────────

// DocumentEdit captures a reversible editing operation.
type DocumentEdit struct {
	Before Document
	After  Document
	Cursor CursorPosition // cursor position after the edit
}
