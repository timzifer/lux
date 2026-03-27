package richtext

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── NewAttributedString ─────────────────────────────────────────

func TestNewAttributedString_Empty(t *testing.T) {
	as := NewAttributedString("")
	if !as.IsEmpty() {
		t.Fatal("expected empty")
	}
	if len(as.Attrs) != 0 {
		t.Fatalf("expected 0 attrs, got %d", len(as.Attrs))
	}
}

func TestNewAttributedString_Plain(t *testing.T) {
	as := NewAttributedString("Hello World")
	if as.Text != "Hello World" {
		t.Fatalf("unexpected text: %q", as.Text)
	}
	if len(as.Attrs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(as.Attrs))
	}
	if as.Attrs[0].End != 11 {
		t.Fatalf("expected End=11, got %d", as.Attrs[0].End)
	}
}

func TestNewAttributedString_Multiline(t *testing.T) {
	as := NewAttributedString("Hello\nWorld\nFoo")
	if as.Text != "Hello\nWorld\nFoo" {
		t.Fatalf("unexpected text: %q", as.Text)
	}
	if as.Attrs[0].End != len(as.Text) {
		t.Fatalf("run should cover full text")
	}
}

// ── PlainText ───────────────────────────────────────────────────

func TestPlainText_Empty(t *testing.T) {
	as := AttributedString{}
	if got := as.PlainText(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPlainText_Roundtrip(t *testing.T) {
	original := "Line one\nLine two\nLine three"
	as := NewAttributedString(original)
	if got := as.PlainText(); got != original {
		t.Fatalf("roundtrip failed: expected %q, got %q", original, got)
	}
}

// ── Len / IsEmpty ───────────────────────────────────────────────

func TestLen(t *testing.T) {
	as := NewAttributedString("Hello")
	if as.Len() != 5 {
		t.Fatalf("expected 5, got %d", as.Len())
	}
}

func TestIsEmpty(t *testing.T) {
	if !NewAttributedString("").IsEmpty() {
		t.Fatal("expected empty")
	}
	if NewAttributedString("x").IsEmpty() {
		t.Fatal("expected not empty")
	}
}

// ── Styled / Build / S ─────────────────────────────────────────

func TestStyled(t *testing.T) {
	as := Styled("Bold", SpanStyle{Bold: true})
	if as.Text != "Bold" {
		t.Fatalf("unexpected text: %q", as.Text)
	}
	if !as.Attrs[0].Style.Bold {
		t.Fatal("expected Bold style")
	}
}

func TestBuild(t *testing.T) {
	as := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("World"),
	)
	if as.Text != "Hello World" {
		t.Fatalf("unexpected text: %q", as.Text)
	}
	if len(as.Attrs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(as.Attrs))
	}
	if as.Attrs[0].End != 6 || !as.Attrs[0].Style.Bold {
		t.Fatalf("first run: End=%d Bold=%v", as.Attrs[0].End, as.Attrs[0].Style.Bold)
	}
	if as.Attrs[1].End != 11 || as.Attrs[1].Style.Bold {
		t.Fatalf("second run: End=%d Bold=%v", as.Attrs[1].End, as.Attrs[1].Style.Bold)
	}
}

func TestBuild_Empty(t *testing.T) {
	as := Build()
	if !as.IsEmpty() {
		t.Fatal("expected empty from Build()")
	}
}

// ── RunAt ───────────────────────────────────────────────────────

func TestRunAt_SingleRun(t *testing.T) {
	as := Styled("Hello", SpanStyle{Italic: true})
	s := as.RunAt(2)
	if !s.Italic {
		t.Fatal("expected Italic at offset 2")
	}
}

func TestRunAt_MultiRun(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	if !as.RunAt(0).Bold {
		t.Fatal("expected Bold at offset 0")
	}
	if !as.RunAt(2).Bold {
		t.Fatal("expected Bold at offset 2")
	}
	if !as.RunAt(3).Italic {
		t.Fatal("expected Italic at offset 3")
	}
	if !as.RunAt(5).Italic {
		t.Fatal("expected Italic at offset 5")
	}
}

func TestRunAt_OutOfRange(t *testing.T) {
	as := NewAttributedString("Hi")
	s := as.RunAt(-1)
	if s.Bold || s.Italic {
		t.Fatal("out-of-range should return zero style")
	}
}

func TestRunAt_PastEnd(t *testing.T) {
	as := Styled("Hi", SpanStyle{Bold: true})
	s := as.RunAt(100)
	if !s.Bold {
		t.Fatal("past-end should return last run style")
	}
}

// ── InsertText ──────────────────────────────────────────────────

func TestInsertText_Middle(t *testing.T) {
	as := NewAttributedString("HelloWorld")
	as2 := as.InsertText(5, " ")
	if as2.Text != "Hello World" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
	if as2.Attrs[len(as2.Attrs)-1].End != 11 {
		t.Fatal("attrs should cover full text after insert")
	}
}

func TestInsertText_PreservesStyle(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	// Insert at boundary of first run — should inherit bold.
	as2 := as.InsertText(3, "X")
	if as2.Text != "AAAXBBB" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
	if !as2.RunAt(3).Bold {
		t.Fatal("inserted char should inherit Bold from preceding run")
	}
}

func TestInsertText_AtStart(t *testing.T) {
	as := Styled("Hello", SpanStyle{Bold: true})
	as2 := as.InsertText(0, "XX")
	if as2.Text != "XXHello" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
}

func TestInsertText_AtEnd(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.InsertText(5, "!")
	if as2.Text != "Hello!" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
}

func TestInsertText_Empty(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.InsertText(2, "")
	if as2.Text != "Hello" {
		t.Fatal("inserting empty should be no-op")
	}
}

// ── DeleteRange ─────────────────────────────────────────────────

func TestDeleteRange_Middle(t *testing.T) {
	as := NewAttributedString("Hello World")
	as2 := as.DeleteRange(5, 6)
	if as2.Text != "HelloWorld" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
}

func TestDeleteRange_PreservesStyle(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	// Delete middle of first run.
	as2 := as.DeleteRange(1, 2)
	if as2.Text != "AABBB" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
	if !as2.RunAt(0).Bold {
		t.Fatal("first char should still be Bold")
	}
	if !as2.RunAt(2).Italic {
		t.Fatal("char at offset 2 should be Italic")
	}
}

func TestDeleteRange_All(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.DeleteRange(0, 5)
	if !as2.IsEmpty() {
		t.Fatal("deleting all should yield empty")
	}
}

func TestDeleteRange_NoOp(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.DeleteRange(3, 3)
	if as2.Text != "Hello" {
		t.Fatal("empty range should be no-op")
	}
}

func TestDeleteRange_CrossRun(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
		S("CCC"),
	)
	// Delete across first two runs.
	as2 := as.DeleteRange(2, 5)
	if as2.Text != "AABCCC" {
		t.Fatalf("unexpected text: %q", as2.Text)
	}
	if !as2.RunAt(0).Bold {
		t.Fatal("start should be Bold")
	}
}

// ── ApplyStyle ──────────────────────────────────────────────────

func TestApplyStyle_SubRange(t *testing.T) {
	as := NewAttributedString("Hello World")
	as2 := as.ApplyStyle(0, 5, SpanStyle{Bold: true})
	if !as2.RunAt(0).Bold {
		t.Fatal("expected Bold at offset 0")
	}
	if as2.RunAt(5).Bold {
		t.Fatal("offset 5 should not be Bold")
	}
}

func TestApplyStyle_Overlapping(t *testing.T) {
	as := Styled("Hello World", SpanStyle{Bold: true})
	as2 := as.ApplyStyle(6, 11, SpanStyle{Italic: true})
	if !as2.RunAt(0).Bold {
		t.Fatal("first part should remain Bold")
	}
	if !as2.RunAt(6).Italic {
		t.Fatal("second part should be Italic")
	}
	if as2.RunAt(6).Bold {
		t.Fatal("ApplyStyle replaces, not merges")
	}
}

func TestApplyStyle_Full(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.ApplyStyle(0, 5, SpanStyle{Underline: true})
	if len(as2.Attrs) != 1 {
		t.Fatalf("expected 1 run after full ApplyStyle, got %d", len(as2.Attrs))
	}
	if !as2.Attrs[0].Style.Underline {
		t.Fatal("expected Underline")
	}
}

// ── Normalized ──────────────────────────────────────────────────

func TestNormalized_MergesAdjacentSameStyle(t *testing.T) {
	as := AttributedString{
		Text: "Hello World",
		Attrs: []AttrRun{
			{End: 5, Style: SpanStyle{Bold: true}},
			{End: 11, Style: SpanStyle{Bold: true}},
		},
	}
	n := as.Normalized()
	if len(n.Attrs) != 1 {
		t.Fatalf("expected 1 run after normalize, got %d", len(n.Attrs))
	}
	if n.Attrs[0].End != 11 {
		t.Fatalf("expected End=11, got %d", n.Attrs[0].End)
	}
}

func TestNormalized_KeepsDifferentStyles(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	n := as.Normalized()
	if len(n.Attrs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(n.Attrs))
	}
}

// ── Equal ───────────────────────────────────────────────────────

func TestEqual_Identical(t *testing.T) {
	a := NewAttributedString("Hello")
	b := NewAttributedString("Hello")
	if !a.Equal(b) {
		t.Fatal("identical should be Equal")
	}
}

func TestEqual_DifferentText(t *testing.T) {
	a := NewAttributedString("Hello")
	b := NewAttributedString("World")
	if a.Equal(b) {
		t.Fatal("different text should not be Equal")
	}
}

func TestEqual_DifferentStyles(t *testing.T) {
	a := Styled("Hello", SpanStyle{Bold: true})
	b := NewAttributedString("Hello")
	if a.Equal(b) {
		t.Fatal("different styles should not be Equal")
	}
}

func TestEqual_DifferentRunCount(t *testing.T) {
	a := NewAttributedString("Hello")
	b := Build(S("Hel", SpanStyle{Bold: true}), S("lo"))
	if a.Equal(b) {
		t.Fatal("different run count should not be Equal")
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
	el := New(NewAttributedString("test"))
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
	el := New(NewAttributedString("test"),
		WithRows(8),
		WithReadOnly(),
		WithPlaceholder("Type here..."),
		WithOnChange(func(AttributedString) { called = true }),
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
	if editor.OnChange == nil {
		t.Fatal("expected OnChange to be set")
	}
	editor.OnChange(NewAttributedString("new"))
	if !called {
		t.Fatal("OnChange was not called")
	}
}

// ── TreeEqual ───────────────────────────────────────────────────

func TestTreeEqual_Same(t *testing.T) {
	a := New(NewAttributedString("Hello"))
	b := New(NewAttributedString("Hello"))
	ea := a.(RichTextEditor)
	if !ea.TreeEqual(b) {
		t.Fatal("same editors should be TreeEqual")
	}
}

func TestTreeEqual_DifferentDoc(t *testing.T) {
	a := New(NewAttributedString("Hello"))
	b := New(NewAttributedString("World"))
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different docs should not be TreeEqual")
	}
}

func TestTreeEqual_DifferentReadOnly(t *testing.T) {
	a := New(NewAttributedString("Hello"))
	b := New(NewAttributedString("Hello"), WithReadOnly())
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different ReadOnly should not be TreeEqual")
	}
}

func TestTreeEqual_DifferentRows(t *testing.T) {
	a := New(NewAttributedString("Hello"), WithRows(4))
	b := New(NewAttributedString("Hello"), WithRows(8))
	ea := a.(RichTextEditor)
	if ea.TreeEqual(b) {
		t.Fatal("different Rows should not be TreeEqual")
	}
}

// ── ResolveChildren ─────────────────────────────────────────────

func TestResolveChildren_IsLeaf(t *testing.T) {
	el := New(NewAttributedString("Hello"))
	editor := el.(RichTextEditor)
	resolved := editor.ResolveChildren(func(e ui.Element, i int) ui.Element {
		t.Fatal("should not be called on leaf")
		return e
	})
	if resolved.(RichTextEditor).Value.Text != editor.Value.Text {
		t.Fatal("leaf should return self")
	}
}

// ── Unicode Content ─────────────────────────────────────────────

func TestNewAttributedString_Unicode(t *testing.T) {
	as := NewAttributedString("Hëllo Wörld 🌍")
	if got := as.PlainText(); got != "Hëllo Wörld 🌍" {
		t.Fatalf("unicode roundtrip failed: %q", got)
	}
}

func TestNewAttributedString_MultilineUnicode(t *testing.T) {
	text := "日本語\nالعربية\nDeutsch"
	as := NewAttributedString(text)
	if got := as.PlainText(); got != text {
		t.Fatalf("roundtrip failed: %q", got)
	}
}

// ── DocumentChangedMsg ──────────────────────────────────────────

func TestDocumentChangedMsg(t *testing.T) {
	msg := DocumentChangedMsg{
		Value: NewAttributedString("Hello World"),
	}
	if msg.Value.PlainText() != "Hello World" {
		t.Fatal("unexpected document text")
	}
}
