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
	// Plain text has no attrs (all defaults).
	if len(as.Attrs) != 0 {
		t.Fatalf("expected 0 attrs for plain text, got %d", len(as.Attrs))
	}
}

func TestNewAttributedString_Multiline(t *testing.T) {
	as := NewAttributedString("Hello\nWorld\nFoo")
	if as.Text != "Hello\nWorld\nFoo" {
		t.Fatalf("unexpected text: %q", as.Text)
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
	if !as.RunAt(0).Bold {
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
	if !as.RunAt(0).Bold {
		t.Fatal("expected Bold at offset 0")
	}
	if !as.RunAt(5).Bold {
		t.Fatal("expected Bold at offset 5")
	}
	if as.RunAt(6).Bold {
		t.Fatal("expected non-Bold at offset 6")
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
	// With tagged ranges, the Bold attr still covers [0,11),
	// and Italic covers [6,11), so Bold is ALSO present at offset 6.
	if !as2.RunAt(6).Bold {
		t.Fatal("tagged ranges: Bold should still be active at offset 6")
	}
}

func TestApplyStyle_Full(t *testing.T) {
	as := NewAttributedString("Hello")
	as2 := as.ApplyStyle(0, 5, SpanStyle{Underline: true})
	if !as2.RunAt(0).Underline {
		t.Fatal("expected Underline")
	}
}

// ── Normalized ──────────────────────────────────────────────────

func TestNormalized_RemovesEmptyRanges(t *testing.T) {
	as := AttributedString{
		Text: "Hello",
		Attrs: []Attr{
			{Start: 0, End: 5, Value: BoldAttr(true)},
			{Start: 3, End: 3, Value: ItalicAttr(true)}, // empty range
		},
	}
	n := as.Normalized()
	if len(n.Attrs) != 1 {
		t.Fatalf("expected 1 attr after normalize, got %d", len(n.Attrs))
	}
}

func TestNormalized_KeepsValidAttrs(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	n := as.Normalized()
	if !n.RunAt(0).Bold {
		t.Fatal("expected Bold preserved")
	}
	if !n.RunAt(3).Italic {
		t.Fatal("expected Italic preserved")
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

func TestEqual_SameResolution(t *testing.T) {
	// Two different Attr slices but same resolved output.
	a := Styled("Hello", SpanStyle{Bold: true})
	b := AttributedString{
		Text: "Hello",
		Attrs: []Attr{
			{Start: 0, End: 3, Value: BoldAttr(true)},
			{Start: 3, End: 5, Value: BoldAttr(true)},
		},
	}
	if !a.Equal(b) {
		t.Fatal("same resolved styles should be Equal")
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

// ── InsertImage ─────────────────────────────────────────────────

func TestInsertImage_AtStart(t *testing.T) {
	as := NewAttributedString("hello")
	img := ImageAttachment{ImageID: 42, Alt: "icon", Width: 16, Height: 16}
	result := as.InsertImage(0, img)

	const placeholder = "\uFFFC"
	want := placeholder + "hello"
	if result.Text != want {
		t.Fatalf("text = %q, want %q", result.Text, want)
	}
	s := result.RunAt(0)
	if s.Image.ImageID != 42 {
		t.Errorf("ImageID = %d, want 42", s.Image.ImageID)
	}
	if s.Image.Alt != "icon" {
		t.Errorf("Alt = %q, want %q", s.Image.Alt, "icon")
	}
	if result.RunAt(len(placeholder)).Image.ImageID != 0 {
		t.Error("expected no image on text portion")
	}
}

func TestInsertImage_InMiddle(t *testing.T) {
	as := Build(
		S("before", SpanStyle{Bold: true}),
		S("after", SpanStyle{Italic: true}),
	)
	img := ImageAttachment{ImageID: 7, Width: 24, Height: 24}
	result := as.InsertImage(len("before"), img)

	const placeholder = "\uFFFC"
	want := "before" + placeholder + "after"
	if result.Text != want {
		t.Fatalf("text = %q, want %q", result.Text, want)
	}
	if !result.RunAt(0).Bold {
		t.Error("expected Bold before image")
	}
	mid := len("before")
	if result.RunAt(mid).Image.ImageID != 7 {
		t.Errorf("image run: ImageID = %d, want 7", result.RunAt(mid).Image.ImageID)
	}
	if !result.RunAt(mid + len(placeholder)).Italic {
		t.Error("expected Italic after image")
	}
}

func TestInsertImage_AtEnd(t *testing.T) {
	as := NewAttributedString("hello")
	img := ImageAttachment{ImageID: 99}
	result := as.InsertImage(len("hello"), img)

	const placeholder = "\uFFFC"
	want := "hello" + placeholder
	if result.Text != want {
		t.Fatalf("text = %q, want %q", result.Text, want)
	}
	if result.RunAt(len("hello")).Image.ImageID != 99 {
		t.Errorf("expected image at end")
	}
}

func TestInsertImage_StylePreservation(t *testing.T) {
	as := Build(
		S("red text", SpanStyle{Color: draw.Hex("#ff0000")}),
	)
	img := ImageAttachment{ImageID: 5}
	result := as.InsertImage(4, img) // "red " + image + "text"

	if result.RunAt(0).Color != draw.Hex("#ff0000") {
		t.Error("color before image should be preserved")
	}
	endOff := len("red ") + len("\uFFFC")
	if result.RunAt(endOff).Color != draw.Hex("#ff0000") {
		t.Error("color after image should be preserved")
	}
}

func TestInsertImage_TwoImagesNotMerged(t *testing.T) {
	as := NewAttributedString("")
	img1 := ImageAttachment{ImageID: 1, Width: 16, Height: 16}
	img2 := ImageAttachment{ImageID: 2, Width: 32, Height: 32}
	const placeholder = "\uFFFC"

	as = as.InsertImage(0, img1)
	as = as.InsertImage(len(placeholder), img2)

	if as.RunAt(0).Image.ImageID != 1 {
		t.Errorf("first image ImageID = %d, want 1", as.RunAt(0).Image.ImageID)
	}
	if as.RunAt(len(placeholder)).Image.ImageID != 2 {
		t.Errorf("second image ImageID = %d, want 2", as.RunAt(len(placeholder)).Image.ImageID)
	}
}

func TestInsertImage_DeleteRemovesPlaceholder(t *testing.T) {
	as := NewAttributedString("hello")
	img := ImageAttachment{ImageID: 42}
	const placeholder = "\uFFFC"
	as = as.InsertImage(5, img)

	as = as.DeleteRange(5, 5+len(placeholder))
	if as.Text != "hello" {
		t.Fatalf("after delete: text = %q, want %q", as.Text, "hello")
	}
	if as.RunAt(0).Image.ImageID != 0 {
		t.Error("expected no image after placeholder deleted")
	}
}

// ── Strikethrough ──────────────────────────────────────────────

func TestStyled_Strikethrough(t *testing.T) {
	as := Styled("Strike", SpanStyle{Strikethrough: true})
	if !as.RunAt(0).Strikethrough {
		t.Fatal("expected Strikethrough style")
	}
}

func TestBuild_Strikethrough(t *testing.T) {
	as := Build(
		S("normal "),
		S("struck", SpanStyle{Strikethrough: true}),
	)
	if as.RunAt(0).Strikethrough {
		t.Error("first run should not be strikethrough")
	}
	if !as.RunAt(8).Strikethrough {
		t.Error("second run should be strikethrough")
	}
}

func TestToggleStyleFunc_Strikethrough(t *testing.T) {
	as := NewAttributedString("Hello World")
	as = as.ToggleStyleFunc(0, 5, func(s SpanStyle) SpanStyle {
		s.Strikethrough = true
		return s
	})
	if !as.RunAt(0).Strikethrough {
		t.Error("expected strikethrough at offset 0")
	}
	if as.RunAt(6).Strikethrough {
		t.Error("expected no strikethrough at offset 6")
	}
}

// ── FontFamily ─────────────────────────────────────────────────

func TestStyled_FontFamily(t *testing.T) {
	as := Styled("Mono", SpanStyle{FontFamily: "Monospace"})
	if as.RunAt(0).FontFamily != "Monospace" {
		t.Fatalf("expected FontFamily=Monospace, got %q", as.RunAt(0).FontFamily)
	}
}

func TestBuild_FontFamily(t *testing.T) {
	as := Build(
		S("sans "),
		S("mono", SpanStyle{FontFamily: "Monospace"}),
	)
	if as.RunAt(0).FontFamily != "" {
		t.Error("first run should inherit font family")
	}
	if as.RunAt(6).FontFamily != "Monospace" {
		t.Errorf("second run FontFamily = %q, want Monospace", as.RunAt(6).FontFamily)
	}
}

// ── Weight ─────────────────────────────────────────────────────

func TestStyled_Weight(t *testing.T) {
	as := Styled("Light", SpanStyle{Weight: draw.FontWeightLight})
	if as.RunAt(0).Weight != draw.FontWeightLight {
		t.Fatalf("expected Weight=300, got %d", as.RunAt(0).Weight)
	}
}

func TestBuild_MultipleWeights(t *testing.T) {
	as := Build(
		S("thin ", SpanStyle{Weight: draw.FontWeightThin}),
		S("bold ", SpanStyle{Weight: draw.FontWeightBold}),
		S("black", SpanStyle{Weight: draw.FontWeightBlack}),
	)
	if as.RunAt(0).Weight != draw.FontWeightThin {
		t.Errorf("expected Thin at 0, got %d", as.RunAt(0).Weight)
	}
	if as.RunAt(5).Weight != draw.FontWeightBold {
		t.Errorf("expected Bold at 5, got %d", as.RunAt(5).Weight)
	}
	if as.RunAt(10).Weight != draw.FontWeightBlack {
		t.Errorf("expected Black at 10, got %d", as.RunAt(10).Weight)
	}
}

// ── BgColor ────────────────────────────────────────────────────

func TestStyled_BgColor(t *testing.T) {
	bg := draw.Hex("#ffff00")
	as := Styled("Highlight", SpanStyle{BgColor: bg})
	if as.RunAt(0).BgColor != bg {
		t.Fatal("expected yellow background color")
	}
}

// ── Tracking ───────────────────────────────────────────────────

func TestStyled_Tracking(t *testing.T) {
	as := Styled("Spaced", SpanStyle{Tracking: 0.1})
	if as.RunAt(0).Tracking != 0.1 {
		t.Fatalf("expected Tracking=0.1, got %g", as.RunAt(0).Tracking)
	}
}

// ── LineHeight ─────────────────────────────────────────────────

func TestStyled_LineHeight(t *testing.T) {
	as := Styled("Tall", SpanStyle{LineHeight: 2.0})
	if as.RunAt(0).LineHeight != 2.0 {
		t.Fatalf("expected LineHeight=2.0, got %g", as.RunAt(0).LineHeight)
	}
}

// ── WhiteSpace ─────────────────────────────────────────────────

func TestWhiteSpace_Enum(t *testing.T) {
	tests := []struct {
		ws   WhiteSpace
		want WhiteSpace
	}{
		{WhiteSpaceNormal, 0},
		{WhiteSpacePre, 1},
		{WhiteSpaceNoWrap, 2},
		{WhiteSpacePreWrap, 3},
		{WhiteSpacePreLine, 4},
	}
	for _, tt := range tests {
		if tt.ws != tt.want {
			t.Errorf("WhiteSpace %d != %d", tt.ws, tt.want)
		}
	}
}

func TestStyled_WhiteSpace(t *testing.T) {
	as := Styled("pre", SpanStyle{WhiteSpace: WhiteSpacePre})
	if as.RunAt(0).WhiteSpace != WhiteSpacePre {
		t.Fatalf("expected WhiteSpacePre, got %d", as.RunAt(0).WhiteSpace)
	}
}

// ── Combined styles ────────────────────────────────────────────

func TestBuild_AllInlineStyles(t *testing.T) {
	style := SpanStyle{
		Bold:          true,
		Italic:        true,
		Underline:     true,
		Strikethrough: true,
		FontFamily:    "Serif",
		Weight:        draw.FontWeightSemiBold,
		Color:         draw.Hex("#ff0000"),
		BgColor:       draw.Hex("#00ff00"),
		Size:          18,
		Tracking:      0.05,
		LineHeight:    1.6,
		WhiteSpace:    WhiteSpacePreWrap,
	}
	as := Styled("all styles", style)
	got := as.RunAt(0)
	if !got.Bold || !got.Italic || !got.Underline || !got.Strikethrough {
		t.Error("expected all boolean flags set")
	}
	if got.FontFamily != "Serif" {
		t.Errorf("FontFamily = %q, want Serif", got.FontFamily)
	}
	if got.Weight != draw.FontWeightSemiBold {
		t.Errorf("Weight = %d, want SemiBold", got.Weight)
	}
	if got.Size != 18 {
		t.Errorf("Size = %g, want 18", got.Size)
	}
	if got.Tracking != 0.05 {
		t.Errorf("Tracking = %g, want 0.05", got.Tracking)
	}
	if got.LineHeight != 1.6 {
		t.Errorf("LineHeight = %g, want 1.6", got.LineHeight)
	}
	if got.WhiteSpace != WhiteSpacePreWrap {
		t.Errorf("WhiteSpace = %d, want PreWrap", got.WhiteSpace)
	}
}

func TestInsertText_InheritsNewFields(t *testing.T) {
	as := Styled("AB", SpanStyle{
		Strikethrough: true,
		FontFamily:    "Mono",
		Weight:        draw.FontWeightLight,
		BgColor:       draw.Hex("#ff0000"),
		Tracking:      0.1,
	})
	as = as.InsertText(1, "X")
	got := as.RunAt(1)
	if !got.Strikethrough {
		t.Error("inserted text should inherit Strikethrough")
	}
	if got.FontFamily != "Mono" {
		t.Errorf("inserted text FontFamily = %q, want Mono", got.FontFamily)
	}
	if got.Weight != draw.FontWeightLight {
		t.Errorf("inserted text Weight = %d, want Light", got.Weight)
	}
	if got.BgColor != draw.Hex("#ff0000") {
		t.Error("inserted text should inherit BgColor")
	}
	if got.Tracking != 0.1 {
		t.Errorf("inserted text Tracking = %g, want 0.1", got.Tracking)
	}
}

func TestAllMatch_Strikethrough(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Strikethrough: true}),
		S("World", SpanStyle{Strikethrough: true}),
	)
	if !doc.AllMatch(0, 11, func(s SpanStyle) bool { return s.Strikethrough }) {
		t.Error("expected all strikethrough")
	}
}

func TestAllMatch_MixedStrikethrough(t *testing.T) {
	doc := Build(
		S("Hello ", SpanStyle{Strikethrough: true}),
		S("World"),
	)
	if doc.AllMatch(0, 11, func(s SpanStyle) bool { return s.Strikethrough }) {
		t.Error("expected not all strikethrough")
	}
}

// ── Tagged Ranges: Overlapping Attributes ──────────────────────

func TestResolveAt_OverlappingAttrs(t *testing.T) {
	// Paragraph-level Align + overlapping Span-level Bold.
	as := AttributedString{
		Text: "Hello World",
		Attrs: []Attr{
			{Start: 0, End: 11, Value: AlignAttr(draw.TextAlignCenter)},
			{Start: 0, End: 5, Value: BoldAttr(true)},
		},
	}
	s0 := as.ResolveAt(0)
	if s0.Align != draw.TextAlignCenter {
		t.Error("expected Center align at offset 0")
	}
	if !s0.Bold {
		t.Error("expected Bold at offset 0")
	}
	s6 := as.ResolveAt(6)
	if s6.Align != draw.TextAlignCenter {
		t.Error("expected Center align at offset 6")
	}
	if s6.Bold {
		t.Error("expected non-Bold at offset 6")
	}
}

func TestApply_NoSplit(t *testing.T) {
	as := NewAttributedString("Hello World")
	as = as.Apply(0, 11, AlignAttr(draw.TextAlignCenter))
	as = as.Apply(3, 7, BoldAttr(true))
	// Should have exactly 2 attrs (no splitting).
	if len(as.Attrs) != 2 {
		t.Fatalf("expected 2 attrs (no split), got %d", len(as.Attrs))
	}
}

func TestStyleRuns_TransitionPoints(t *testing.T) {
	as := AttributedString{
		Text: "AABBCC",
		Attrs: []Attr{
			{Start: 0, End: 6, Value: AlignAttr(draw.TextAlignRight)},
			{Start: 2, End: 4, Value: BoldAttr(true)},
		},
	}
	runs := as.StyleRuns(0, 6)
	if len(runs) != 3 {
		t.Fatalf("expected 3 style runs, got %d", len(runs))
	}
	if runs[0].Start != 0 || runs[0].End != 2 {
		t.Errorf("run 0: [%d,%d) want [0,2)", runs[0].Start, runs[0].End)
	}
	if runs[1].Start != 2 || runs[1].End != 4 || !runs[1].Style.Bold {
		t.Errorf("run 1: [%d,%d) Bold=%v want [2,4) Bold=true", runs[1].Start, runs[1].End, runs[1].Style.Bold)
	}
	if runs[2].Start != 4 || runs[2].End != 6 || runs[2].Style.Bold {
		t.Errorf("run 2: [%d,%d) Bold=%v want [4,6) Bold=false", runs[2].Start, runs[2].End, runs[2].Style.Bold)
	}
	// All runs should have Right alignment.
	for i, r := range runs {
		if r.Style.Align != draw.TextAlignRight {
			t.Errorf("run %d: Align=%d, want Right", i, r.Style.Align)
		}
	}
}

func TestParagraphRange(t *testing.T) {
	text := "Hello\nWorld\nFoo"
	tests := []struct {
		offset    int
		wantStart int
		wantEnd   int
	}{
		{0, 0, 5},
		{3, 0, 5},
		{6, 6, 11},
		{12, 12, 15},
	}
	for _, tt := range tests {
		start, end := ParagraphRange(text, tt.offset)
		if start != tt.wantStart || end != tt.wantEnd {
			t.Errorf("ParagraphRange(%q, %d) = (%d, %d), want (%d, %d)",
				text, tt.offset, start, end, tt.wantStart, tt.wantEnd)
		}
	}
}

// ── Paragraph-level attributes ─────────────────────────────────

func TestStyled_Align(t *testing.T) {
	as := Styled("centered", SpanStyle{Align: draw.TextAlignCenter})
	if as.RunAt(0).Align != draw.TextAlignCenter {
		t.Fatal("expected Center alignment")
	}
}

func TestStyled_Indent(t *testing.T) {
	as := Styled("indented", SpanStyle{Indent: 24})
	if as.RunAt(0).Indent != 24 {
		t.Fatalf("expected Indent=24, got %g", as.RunAt(0).Indent)
	}
}

func TestStyled_ParaSpacing(t *testing.T) {
	as := Styled("spaced", SpanStyle{ParaSpacing: 16})
	if as.RunAt(0).ParaSpacing != 16 {
		t.Fatalf("expected ParaSpacing=16, got %g", as.RunAt(0).ParaSpacing)
	}
}

// ── List Attributes ─────────────────────────────────────────────

func TestListTypeAttr_Apply(t *testing.T) {
	as := NewAttributedString("Item one\nItem two")
	as = as.Apply(0, 8, ListTypeAttr(draw.ListTypeUnordered))

	got := as.ResolveAt(0).ListType
	if got != draw.ListTypeUnordered {
		t.Fatalf("expected ListTypeUnordered, got %d", got)
	}
	// Second paragraph should not have list type.
	got2 := as.ResolveAt(9).ListType
	if got2 != draw.ListTypeNone {
		t.Fatalf("expected ListTypeNone at offset 9, got %d", got2)
	}
}

func TestListLevelAttr_Apply(t *testing.T) {
	as := NewAttributedString("Nested item")
	as = as.Apply(0, len(as.Text), ListLevelAttr(2))

	got := as.ResolveAt(0).ListLevel
	if got != 2 {
		t.Fatalf("expected ListLevel=2, got %d", got)
	}
}

func TestListStartAttr_Apply(t *testing.T) {
	as := NewAttributedString("Item five")
	as = as.Apply(0, len(as.Text), ListStartAttr(5))

	got := as.ResolveAt(0).ListStart
	if got != 5 {
		t.Fatalf("expected ListStart=5, got %d", got)
	}
}

func TestListMarkerAttr_Apply(t *testing.T) {
	as := NewAttributedString("Roman item")
	as = as.Apply(0, len(as.Text), ListMarkerAttr(draw.ListMarkerLowerRoman))

	got := as.ResolveAt(0).ListMarker
	if got != draw.ListMarkerLowerRoman {
		t.Fatalf("expected ListMarkerLowerRoman, got %d", got)
	}
}

func TestListAttr_ParagraphScope(t *testing.T) {
	as := NewAttributedString("Para one\nPara two\nPara three")
	// Apply list only to second paragraph.
	as = as.Apply(9, 17, ListTypeAttr(draw.ListTypeOrdered))

	if as.ResolveAt(0).ListType != draw.ListTypeNone {
		t.Fatal("first paragraph should not be a list")
	}
	if as.ResolveAt(9).ListType != draw.ListTypeOrdered {
		t.Fatal("second paragraph should be ordered list")
	}
	if as.ResolveAt(18).ListType != draw.ListTypeNone {
		t.Fatal("third paragraph should not be a list")
	}
}

func TestListAttr_ToggleOff(t *testing.T) {
	as := NewAttributedString("List item")
	as = as.Apply(0, len(as.Text), ListTypeAttr(draw.ListTypeUnordered))
	// Toggle off.
	as = as.Apply(0, len(as.Text), ListTypeAttr(draw.ListTypeNone))

	got := as.ResolveAt(0).ListType
	if got != draw.ListTypeNone {
		t.Fatalf("expected ListTypeNone after toggle off, got %d", got)
	}
}

func TestListAttr_SpanStyleToAttrs(t *testing.T) {
	as := Styled("List styled", SpanStyle{
		ListType:  draw.ListTypeOrdered,
		ListLevel: 1,
		ListStart: 3,
	})
	s := as.ResolveAt(0)
	if s.ListType != draw.ListTypeOrdered {
		t.Fatalf("expected ListTypeOrdered, got %d", s.ListType)
	}
	if s.ListLevel != 1 {
		t.Fatalf("expected ListLevel=1, got %d", s.ListLevel)
	}
	if s.ListStart != 3 {
		t.Fatalf("expected ListStart=3, got %d", s.ListStart)
	}
}

// ── Paragraph-level attribute handling in InsertText / DeleteRange ──

func TestInsertText_ParagraphAttr_CharInMiddle(t *testing.T) {
	// Typing a character in the middle of a list item should keep the attr.
	doc := NewAttributedString("Item A\nItem B")
	doc = doc.Apply(0, 7, ListTypeAttr(draw.ListTypeUnordered))
	doc = doc.Apply(7, 13, ListTypeAttr(draw.ListTypeUnordered))

	doc = doc.InsertText(4, "X") // "ItemX A\nItem B"

	s := doc.ResolveAt(4)
	if s.ListType != draw.ListTypeUnordered {
		t.Errorf("inserted char should be Unordered, got %d", s.ListType)
	}
	// Second paragraph should still be Unordered.
	s2 := doc.ResolveAt(8)
	if s2.ListType != draw.ListTypeUnordered {
		t.Errorf("second line should remain Unordered, got %d", s2.ListType)
	}
}

func TestInsertText_ParagraphAttr_CharAtStart(t *testing.T) {
	// Typing at the start of a list item should keep the attr (not shift it away).
	doc := NewAttributedString("Item A\nItem B")
	doc = doc.Apply(0, 7, ListTypeAttr(draw.ListTypeUnordered))
	doc = doc.Apply(7, 13, ListTypeAttr(draw.ListTypeUnordered))

	doc = doc.InsertText(7, "X") // "Item A\nXItem B"

	s := doc.ResolveAt(7)
	if s.ListType != draw.ListTypeUnordered {
		t.Errorf("char at start of second paragraph should be Unordered, got %d", s.ListType)
	}
}

func TestInsertText_ParagraphAttr_NewlineInMiddle(t *testing.T) {
	// Pressing Enter in the middle of a list item should give both halves the attr.
	doc := NewAttributedString("Item AB")
	doc = doc.Apply(0, 7, ListTypeAttr(draw.ListTypeUnordered))

	doc = doc.InsertText(4, "\n") // "Item\nAB"

	s1 := doc.ResolveAt(0) // "Item" portion
	if s1.ListType != draw.ListTypeUnordered {
		t.Errorf("first half should be Unordered, got %d", s1.ListType)
	}
	s2 := doc.ResolveAt(5) // "AB" portion
	if s2.ListType != draw.ListTypeUnordered {
		t.Errorf("second half should be Unordered, got %d", s2.ListType)
	}
}

func TestInsertText_ParagraphAttr_NewlineAtEnd(t *testing.T) {
	// Pressing Enter at the end of a list item should create a new list item.
	doc := NewAttributedString("Item A\nItem B")
	doc = doc.Apply(0, 7, ListTypeAttr(draw.ListTypeUnordered))
	doc = doc.Apply(7, 13, ListTypeAttr(draw.ListTypeUnordered))

	doc = doc.InsertText(6, "\n") // "Item A\n\nItem B"

	// The new empty line (position 7) should be Unordered.
	s := doc.ResolveAt(7)
	if s.ListType != draw.ListTypeUnordered {
		t.Errorf("new line after Enter should be Unordered, got %d", s.ListType)
	}
	// Original second line should remain Unordered.
	s2 := doc.ResolveAt(8)
	if s2.ListType != draw.ListTypeUnordered {
		t.Errorf("original second line should remain Unordered, got %d", s2.ListType)
	}
}

func TestInsertText_ParagraphAttr_LevelPreserved(t *testing.T) {
	// Pressing Enter in a nested list item should preserve the level.
	doc := NewAttributedString("Sub A")
	doc = doc.Apply(0, 5, ListTypeAttr(draw.ListTypeUnordered))
	doc = doc.Apply(0, 5, ListLevelAttr(1))

	doc = doc.InsertText(3, "\n") // "Sub\nA"

	s1 := doc.ResolveAt(0)
	if s1.ListLevel != 1 {
		t.Errorf("first half level = %d, want 1", s1.ListLevel)
	}
	s2 := doc.ResolveAt(4)
	if s2.ListLevel != 1 {
		t.Errorf("second half level = %d, want 1", s2.ListLevel)
	}
}

func TestInsertText_SpanAttr_NotAffectedByParagraphLogic(t *testing.T) {
	// Bold (span attr) should still extend through insertion as before.
	doc := NewAttributedString("Hello")
	doc = doc.Apply(0, 5, BoldAttr(true))
	doc = doc.Apply(0, 5, ListTypeAttr(draw.ListTypeUnordered))

	doc = doc.InsertText(3, "\n") // "Hel\nlo"

	// Bold should extend through both halves (span attr behavior).
	s1 := doc.ResolveAt(0)
	if !s1.Bold {
		t.Error("first half should still be bold")
	}
	s2 := doc.ResolveAt(4)
	if !s2.Bold {
		t.Error("second half should still be bold")
	}
}
