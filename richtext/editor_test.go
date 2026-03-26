package richtext

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

// ── makeOnChange ────────────────────────────────────────────────

func TestMakeOnChange_PreservesStyles(t *testing.T) {
	var received Document
	editor := RichTextEditor{
		Value: Document{Paragraphs: []Paragraph{
			{Spans: []Span{{Text: "Hello", Style: SpanStyle{Bold: true}}}},
			{Spans: []Span{{Text: "World", Style: SpanStyle{Italic: true}}}},
		}},
		OnChange: func(doc Document) { received = doc },
	}
	fn := editor.makeOnChange()
	fn("Hello!\nWorld")

	if len(received.Paragraphs) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(received.Paragraphs))
	}
	// First paragraph should keep Bold style.
	if !received.Paragraphs[0].Spans[0].Style.Bold {
		t.Error("expected first paragraph to preserve Bold style")
	}
	// Second paragraph should keep Italic style.
	if !received.Paragraphs[1].Spans[0].Style.Italic {
		t.Error("expected second paragraph to preserve Italic style")
	}
}

func TestMakeOnChange_NewParagraphs(t *testing.T) {
	var received Document
	editor := RichTextEditor{
		Value: Document{Paragraphs: []Paragraph{
			{Spans: []Span{{Text: "Hello"}}},
		}},
		OnChange: func(doc Document) { received = doc },
	}
	fn := editor.makeOnChange()
	fn("Hello\nNew line\nAnother")

	if len(received.Paragraphs) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(received.Paragraphs))
	}
	if received.PlainText() != "Hello\nNew line\nAnother" {
		t.Fatalf("unexpected text: %q", received.PlainText())
	}
}

func TestMakeOnChange_NilOnChange(t *testing.T) {
	editor := RichTextEditor{
		Value: NewDocument("Hello"),
	}
	fn := editor.makeOnChange()
	if fn != nil {
		t.Fatal("expected nil when OnChange is nil")
	}
}

// ── maxf ────────────────────────────────────────────────────────

func TestMaxf(t *testing.T) {
	if maxf(1, 2) != 2 {
		t.Fatal("expected 2")
	}
	if maxf(3, 1) != 3 {
		t.Fatal("expected 3")
	}
	if maxf(5, 5) != 5 {
		t.Fatal("expected 5")
	}
}

// ── EditorToolbar ───────────────────────────────────────────────

func TestEditorToolbar(t *testing.T) {
	tb := &EditorToolbar{Bold: true, Italic: true, Underline: false}
	if !tb.Bold || !tb.Italic || tb.Underline {
		t.Fatal("unexpected toolbar state")
	}
}

// ── Complex Document Scenarios ──────────────────────────────────

func TestDocument_MultiSpanParagraphs(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Spans: []Span{
			{Text: "Hello ", Style: SpanStyle{Bold: true}},
			{Text: "beautiful ", Style: SpanStyle{Italic: true}},
			{Text: "World", Style: SpanStyle{Color: draw.Color{R: 1, A: 1}}},
		}},
		{Spans: []Span{
			{Text: "Second paragraph"},
		}},
	}}

	if got := doc.PlainText(); got != "Hello beautiful World\nSecond paragraph" {
		t.Fatalf("unexpected plain text: %q", got)
	}

	if len(doc.Paragraphs[0].Spans) != 3 {
		t.Fatalf("expected 3 spans in first paragraph")
	}
}

func TestDocument_EmptyParagraphs(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "First"}}},
		{}, // empty paragraph (no spans)
		{Spans: []Span{{Text: "Third"}}},
	}}
	if got := doc.PlainText(); got != "First\n\nThird" {
		t.Fatalf("expected %q, got %q", "First\n\nThird", got)
	}
}

func TestParagraphText_SingleSpan(t *testing.T) {
	p := Paragraph{Spans: []Span{{Text: "Hello"}}}
	if got := paragraphText(p); got != "Hello" {
		t.Fatalf("expected %q, got %q", "Hello", got)
	}
}

// ── DocumentChangedMsg ──────────────────────────────────────────

func TestDocumentChangedMsg(t *testing.T) {
	msg := DocumentChangedMsg{
		Document: NewDocument("Hello World"),
	}
	if msg.Document.PlainText() != "Hello World" {
		t.Fatal("unexpected document text")
	}
}
