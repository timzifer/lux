package richtext

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

// ── NewDocument ─────────────────────────────────────────────────

func TestNewDocument_Empty(t *testing.T) {
	doc := NewDocument("")
	if len(doc.Paragraphs) != 1 {
		t.Fatalf("expected 1 paragraph, got %d", len(doc.Paragraphs))
	}
}

func TestNewDocument_SingleLine(t *testing.T) {
	doc := NewDocument("Hello World")
	if len(doc.Paragraphs) != 1 {
		t.Fatalf("expected 1 paragraph, got %d", len(doc.Paragraphs))
	}
	if doc.Paragraphs[0].Spans[0].Text != "Hello World" {
		t.Fatalf("unexpected text: %q", doc.Paragraphs[0].Spans[0].Text)
	}
}

func TestNewDocument_MultiLine(t *testing.T) {
	doc := NewDocument("Hello\nWorld\nFoo")
	if len(doc.Paragraphs) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(doc.Paragraphs))
	}
	expected := []string{"Hello", "World", "Foo"}
	for i, want := range expected {
		got := doc.Paragraphs[i].Spans[0].Text
		if got != want {
			t.Errorf("paragraph %d: want %q, got %q", i, want, got)
		}
	}
}

func TestNewDocument_TrailingNewline(t *testing.T) {
	doc := NewDocument("Hello\n")
	if len(doc.Paragraphs) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(doc.Paragraphs))
	}
	if doc.Paragraphs[1].Spans[0].Text != "" {
		t.Fatalf("expected empty trailing paragraph, got %q", doc.Paragraphs[1].Spans[0].Text)
	}
}

// ── PlainText ───────────────────────────────────────────────────

func TestPlainText_Empty(t *testing.T) {
	doc := Document{}
	if got := doc.PlainText(); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestPlainText_SingleParagraph(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "Hello World"}}},
	}}
	if got := doc.PlainText(); got != "Hello World" {
		t.Fatalf("expected %q, got %q", "Hello World", got)
	}
}

func TestPlainText_MultiParagraph(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "Hello"}}},
		{Spans: []Span{{Text: "World"}}},
	}}
	if got := doc.PlainText(); got != "Hello\nWorld" {
		t.Fatalf("expected %q, got %q", "Hello\nWorld", got)
	}
}

func TestPlainText_MultiSpan(t *testing.T) {
	doc := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "Hello "}, {Text: "World"}}},
	}}
	if got := doc.PlainText(); got != "Hello World" {
		t.Fatalf("expected %q, got %q", "Hello World", got)
	}
}

func TestPlainText_Roundtrip(t *testing.T) {
	original := "Line one\nLine two\nLine three"
	doc := NewDocument(original)
	if got := doc.PlainText(); got != original {
		t.Fatalf("roundtrip failed: expected %q, got %q", original, got)
	}
}

// ── paragraphText / paragraphLen ────────────────────────────────

func TestParagraphText(t *testing.T) {
	p := Paragraph{Spans: []Span{{Text: "Hello "}, {Text: "World"}}}
	if got := paragraphText(p); got != "Hello World" {
		t.Fatalf("expected %q, got %q", "Hello World", got)
	}
}

func TestParagraphLen(t *testing.T) {
	p := Paragraph{Spans: []Span{{Text: "Hello "}, {Text: "World"}}}
	if got := paragraphLen(p); got != 11 {
		t.Fatalf("expected 11, got %d", got)
	}
}

func TestParagraphLen_Empty(t *testing.T) {
	p := Paragraph{}
	if got := paragraphLen(p); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

// ── CursorPosition & Selection ──────────────────────────────────

func TestSelection_HasSelection(t *testing.T) {
	sel := Selection{
		Anchor: CursorPosition{Paragraph: 0, Offset: 0},
		Focus:  CursorPosition{Paragraph: 0, Offset: 5},
	}
	if !sel.HasSelection() {
		t.Fatal("expected HasSelection to be true")
	}
}

func TestSelection_NoSelection(t *testing.T) {
	sel := Selection{
		Anchor: CursorPosition{Paragraph: 0, Offset: 3},
		Focus:  CursorPosition{Paragraph: 0, Offset: 3},
	}
	if sel.HasSelection() {
		t.Fatal("expected HasSelection to be false")
	}
}

func TestSelection_Ordered_Forward(t *testing.T) {
	sel := Selection{
		Anchor: CursorPosition{Paragraph: 0, Offset: 2},
		Focus:  CursorPosition{Paragraph: 1, Offset: 3},
	}
	start, end := sel.Ordered()
	if start.Paragraph != 0 || start.Offset != 2 {
		t.Fatalf("unexpected start: %+v", start)
	}
	if end.Paragraph != 1 || end.Offset != 3 {
		t.Fatalf("unexpected end: %+v", end)
	}
}

func TestSelection_Ordered_Backward(t *testing.T) {
	sel := Selection{
		Anchor: CursorPosition{Paragraph: 1, Offset: 3},
		Focus:  CursorPosition{Paragraph: 0, Offset: 2},
	}
	start, end := sel.Ordered()
	if start.Paragraph != 0 || start.Offset != 2 {
		t.Fatalf("unexpected start: %+v", start)
	}
	if end.Paragraph != 1 || end.Offset != 3 {
		t.Fatalf("unexpected end: %+v", end)
	}
}

func TestSelection_Ordered_SameParagraph_Backward(t *testing.T) {
	sel := Selection{
		Anchor: CursorPosition{Paragraph: 0, Offset: 5},
		Focus:  CursorPosition{Paragraph: 0, Offset: 2},
	}
	start, end := sel.Ordered()
	if start.Offset != 2 || end.Offset != 5 {
		t.Fatalf("unexpected order: start=%d, end=%d", start.Offset, end.Offset)
	}
}

// ── DocumentEdit ────────────────────────────────────────────────

func TestDocumentEdit_UndoRedo(t *testing.T) {
	before := NewDocument("Hello")
	after := NewDocument("Hello World")
	edit := DocumentEdit{
		Before: before,
		After:  after,
		Cursor: CursorPosition{Paragraph: 0, Offset: 11},
	}
	if edit.After.PlainText() != "Hello World" {
		t.Fatalf("unexpected after text: %q", edit.After.PlainText())
	}
	if edit.Before.PlainText() != "Hello" {
		t.Fatalf("unexpected before text: %q", edit.Before.PlainText())
	}
}

// ── SpanStyle ───────────────────────────────────────────────────

func TestSpanStyle_Defaults(t *testing.T) {
	s := SpanStyle{}
	if s.Bold || s.Italic || s.Underline {
		t.Fatal("default SpanStyle should have no formatting")
	}
	if s.Color.A != 0 {
		t.Fatal("default color should be zero (inherit)")
	}
	if s.Size != 0 {
		t.Fatal("default size should be zero (inherit)")
	}
}

func TestSpanStyle_Bold(t *testing.T) {
	s := SpanStyle{Bold: true}
	if !s.Bold {
		t.Fatal("expected Bold to be true")
	}
}

func TestSpanStyle_WithColor(t *testing.T) {
	s := SpanStyle{Color: draw.Color{R: 1, G: 0, B: 0, A: 1}}
	if s.Color.R != 1 || s.Color.A != 1 {
		t.Fatal("unexpected color")
	}
}

// ── documentsEqual ──────────────────────────────────────────────

func TestDocumentsEqual_Identical(t *testing.T) {
	a := NewDocument("Hello\nWorld")
	b := NewDocument("Hello\nWorld")
	if !documentsEqual(a, b) {
		t.Fatal("identical documents should be equal")
	}
}

func TestDocumentsEqual_Different(t *testing.T) {
	a := NewDocument("Hello")
	b := NewDocument("World")
	if documentsEqual(a, b) {
		t.Fatal("different documents should not be equal")
	}
}

func TestDocumentsEqual_DifferentParagraphCount(t *testing.T) {
	a := NewDocument("Hello")
	b := NewDocument("Hello\nWorld")
	if documentsEqual(a, b) {
		t.Fatal("documents with different paragraph counts should not be equal")
	}
}

func TestDocumentsEqual_DifferentStyles(t *testing.T) {
	a := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "Hello", Style: SpanStyle{Bold: true}}}},
	}}
	b := Document{Paragraphs: []Paragraph{
		{Spans: []Span{{Text: "Hello", Style: SpanStyle{Bold: false}}}},
	}}
	if documentsEqual(a, b) {
		t.Fatal("documents with different styles should not be equal")
	}
}

// ── closestBoundary ─────────────────────────────────────────────

func TestClosestBoundary_ExactMatch(t *testing.T) {
	xs := []float32{10, 20, 30, 40}
	offs := []int{0, 3, 6, 9}
	got := closestBoundary(xs, offs, 20)
	if got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestClosestBoundary_Between(t *testing.T) {
	xs := []float32{10, 20, 30}
	offs := []int{0, 3, 6}
	got := closestBoundary(xs, offs, 24)
	if got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestClosestBoundary_BeforeFirst(t *testing.T) {
	xs := []float32{10, 20, 30}
	offs := []int{0, 3, 6}
	got := closestBoundary(xs, offs, 5)
	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestClosestBoundary_AfterLast(t *testing.T) {
	xs := []float32{10, 20, 30}
	offs := []int{0, 3, 6}
	got := closestBoundary(xs, offs, 50)
	if got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}

func TestClosestBoundary_Empty(t *testing.T) {
	got := closestBoundary(nil, nil, 10)
	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

// ── Constructor & Options ───────────────────────────────────────

func TestNew_DefaultRows(t *testing.T) {
	el := New(NewDocument("test"))
	editor, ok := el.(RichTextEditor)
	if !ok {
		t.Fatal("expected RichTextEditor")
	}
	if editor.Rows != editorMinRows {
		t.Fatalf("expected %d rows, got %d", editorMinRows, editor.Rows)
	}
}

func TestNew_WithOptions(t *testing.T) {
	called := false
	el := New(NewDocument("test"),
		WithRows(8),
		WithReadOnly(),
		WithPlaceholder("Type here..."),
		WithOnChange(func(Document) { called = true }),
		WithToolbar(&EditorToolbar{Bold: true, Italic: true}),
	)
	editor, ok := el.(RichTextEditor)
	if !ok {
		t.Fatal("expected RichTextEditor")
	}
	if editor.Rows != 8 {
		t.Fatalf("expected 8 rows, got %d", editor.Rows)
	}
	if !editor.ReadOnly {
		t.Fatal("expected ReadOnly")
	}
	if editor.Placeholder != "Type here..." {
		t.Fatalf("unexpected placeholder: %q", editor.Placeholder)
	}
	if editor.Toolbar == nil || !editor.Toolbar.Bold || !editor.Toolbar.Italic {
		t.Fatal("unexpected toolbar state")
	}
	if editor.OnChange == nil {
		t.Fatal("expected OnChange to be set")
	}
	// Call the OnChange to verify it works.
	editor.OnChange(NewDocument("new"))
	if !called {
		t.Fatal("OnChange was not called")
	}
}

// ── TreeEqual ───────────────────────────────────────────────────

func TestTreeEqual_Same(t *testing.T) {
	a := New(NewDocument("Hello"))
	b := New(NewDocument("Hello"))
	ea := a.(RichTextEditor)
	if !ea.TreeEqual(b) {
		t.Fatal("same editors should be TreeEqual")
	}
}

func TestTreeEqual_DifferentDoc(t *testing.T) {
	a := New(NewDocument("Hello"))
	b := New(NewDocument("World"))
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different docs should not be TreeEqual")
	}
}

func TestTreeEqual_DifferentReadOnly(t *testing.T) {
	a := New(NewDocument("Hello"))
	b := New(NewDocument("Hello"), WithReadOnly())
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different ReadOnly should not be TreeEqual")
	}
}

func TestTreeEqual_DifferentRows(t *testing.T) {
	a := New(NewDocument("Hello"), WithRows(4))
	b := New(NewDocument("Hello"), WithRows(8))
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different Rows should not be TreeEqual")
	}
}

// ── ResolveChildren ─────────────────────────────────────────────

func TestResolveChildren_IsLeaf(t *testing.T) {
	el := New(NewDocument("Hello"))
	editor := el.(RichTextEditor)
	resolved := editor.ResolveChildren(func(e ui.Element, i int) ui.Element {
		t.Fatal("should not be called on leaf")
		return e
	})
	if resolved != editor {
		t.Fatal("leaf should return self")
	}
}

// ── Unicode Content ─────────────────────────────────────────────

func TestNewDocument_Unicode(t *testing.T) {
	doc := NewDocument("Hëllo Wörld 🌍")
	if got := doc.PlainText(); got != "Hëllo Wörld 🌍" {
		t.Fatalf("unicode roundtrip failed: %q", got)
	}
}

func TestNewDocument_MultilineUnicode(t *testing.T) {
	text := "日本語\nالعربية\nDeutsch"
	doc := NewDocument(text)
	if len(doc.Paragraphs) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(doc.Paragraphs))
	}
	if got := doc.PlainText(); got != text {
		t.Fatalf("roundtrip failed: %q", got)
	}
}
