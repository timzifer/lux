package richtext

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ── makeOnChange ────────────────────────────────────────────────

func TestMakeOnChange_PreservesStyles(t *testing.T) {
	var received AttributedString
	editor := RichTextEditor{
		Value: Build(
			S("Hello\n", SpanStyle{Bold: true}),
			S("World", SpanStyle{Italic: true}),
		),
		OnChange: func(as AttributedString) { received = as },
	}
	fn := editor.makeOnChange()
	fn("Hello!\nWorld")

	if received.Text != "Hello!\nWorld" {
		t.Fatalf("unexpected text: %q", received.Text)
	}
	// "Hello" part should still be Bold.
	if !received.RunAt(0).Bold {
		t.Error("expected Bold to be preserved at start")
	}
	// "World" part should still be Italic.
	if !received.RunAt(received.Len() - 1).Italic {
		t.Error("expected Italic to be preserved at end")
	}
}

func TestMakeOnChange_MultiSpanPreservesStyles(t *testing.T) {
	var received AttributedString
	editor := RichTextEditor{
		Value: Build(
			S("Hello ", SpanStyle{Bold: true}),
			S("World", SpanStyle{Italic: true}),
		),
		OnChange: func(as AttributedString) { received = as },
	}
	fn := editor.makeOnChange()
	// Type "beautiful " inside the bold span (after "Hello ").
	fn("Hello beautiful World")

	if received.Text != "Hello beautiful World" {
		t.Fatalf("unexpected text: %q", received.Text)
	}
	// "Hello beautiful " should be bold (inserted text inherits preceding style).
	if !received.RunAt(0).Bold {
		t.Error("expected Bold at start")
	}
	if !received.RunAt(10).Bold {
		t.Error("expected Bold at 'beautiful' offset")
	}
	// "World" should still be italic.
	if !received.RunAt(received.Len() - 1).Italic {
		t.Error("expected Italic at end")
	}
}

func TestMakeOnChange_NewParagraphs(t *testing.T) {
	var received AttributedString
	editor := RichTextEditor{
		Value:    NewAttributedString("Hello"),
		OnChange: func(as AttributedString) { received = as },
	}
	fn := editor.makeOnChange()
	fn("Hello\nNew line\nAnother")

	if received.Text != "Hello\nNew line\nAnother" {
		t.Fatalf("unexpected text: %q", received.Text)
	}
}

func TestMakeOnChange_NilOnChange(t *testing.T) {
	editor := RichTextEditor{
		Value: NewAttributedString("Hello"),
	}
	fn := editor.makeOnChange()
	if fn != nil {
		t.Fatal("expected nil when OnChange is nil")
	}
}

func TestMakeOnChange_NoChange(t *testing.T) {
	callCount := 0
	orig := Styled("Hello", SpanStyle{Bold: true})
	editor := RichTextEditor{
		Value:    orig,
		OnChange: func(as AttributedString) { callCount++ },
	}
	fn := editor.makeOnChange()
	fn("Hello") // same text
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
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

// ── DefaultCommands ─────────────────────────────────────────────

func TestDefaultCommands(t *testing.T) {
	cmds := DefaultCommands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 default commands, got %d", len(cmds))
	}
}

// ── Complex AttributedString Scenarios ──────────────────────────

func TestAttributedString_MultipleStyles(t *testing.T) {
	as := Build(
		S("Hello ", SpanStyle{Bold: true}),
		S("beautiful ", SpanStyle{Italic: true}),
		S("World", SpanStyle{Color: draw.Color{R: 1, A: 1}}),
	)

	if as.Text != "Hello beautiful World" {
		t.Fatalf("unexpected text: %q", as.Text)
	}
	if len(as.Attrs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(as.Attrs))
	}
}

func TestAttributedString_InsertDeleteRoundtrip(t *testing.T) {
	as := Build(
		S("AAA", SpanStyle{Bold: true}),
		S("BBB", SpanStyle{Italic: true}),
	)
	// Insert then delete should restore original.
	as2 := as.InsertText(3, "XXX")
	as3 := as2.DeleteRange(3, 6)
	if as3.Text != "AAABBB" {
		t.Fatalf("roundtrip failed: %q", as3.Text)
	}
}

// ── commonPrefixLen / commonSuffixLen ───────────────────────────

func TestCommonPrefixLen(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"Hello", "Hello World", 5},
		{"abc", "xyz", 0},
		{"", "Hello", 0},
		{"same", "same", 4},
	}
	for _, tt := range tests {
		got := commonPrefixLen(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("commonPrefixLen(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCommonSuffixLen(t *testing.T) {
	tests := []struct {
		a, b string
		pfx  int
		want int
	}{
		{"Hello", "Hello World", 5, 0},
		{"Hello World", "Hello Brave World", 6, 5},
		{"abc", "abc", 3, 0},
	}
	for _, tt := range tests {
		got := commonSuffixLen(tt.a, tt.b, tt.pfx)
		if got != tt.want {
			t.Errorf("commonSuffixLen(%q, %q, %d) = %d, want %d", tt.a, tt.b, tt.pfx, got, tt.want)
		}
	}
}

// ── Image support ────────────────────────────────────────────────

func TestEditorWithImage_ElementCreation(t *testing.T) {
	// Build a document with an inline image using InsertImage.
	doc := NewAttributedString("before after")
	img := ImageAttachment{
		ImageID: 42,
		Alt:     "test image",
		Width:   24,
		Height:  24,
	}
	doc = doc.InsertImage(len("before "), img)

	el := New(doc, WithReadOnly(), WithRows(3))
	if el == nil {
		t.Fatal("expected non-nil element")
	}
	// Should be a RichTextEditor.
	if _, ok := el.(RichTextEditor); !ok {
		t.Fatalf("expected RichTextEditor, got %T", el)
	}
}

func TestEditorWithImage_RunAt(t *testing.T) {
	doc := NewAttributedString("hello")
	img := ImageAttachment{ImageID: 7, Width: 32, Height: 32, ScaleMode: draw.ImageScaleStretch}
	doc = doc.InsertImage(5, img)

	// The run at the placeholder should contain the image.
	s := doc.RunAt(5)
	if s.Image.ImageID != 7 {
		t.Errorf("ImageID = %d, want 7", s.Image.ImageID)
	}
	if s.Image.Width != 32 {
		t.Errorf("Width = %g, want 32", s.Image.Width)
	}
	if s.Image.ScaleMode != draw.ImageScaleStretch {
		t.Errorf("ScaleMode = %d, want Stretch", s.Image.ScaleMode)
	}
}

func TestEditorWithImage_OnlyImageDoc(t *testing.T) {
	// Document consisting solely of an image placeholder.
	doc := AttributedString{}
	img := ImageAttachment{ImageID: 1, Width: 48, Height: 48}
	doc = doc.InsertImage(0, img)

	el := New(doc, WithReadOnly())
	if _, ok := el.(RichTextEditor); !ok {
		t.Fatalf("expected RichTextEditor, got %T", el)
	}
	// Verify the editor holds the image data.
	editor := el.(RichTextEditor)
	s := editor.Value.RunAt(0)
	if s.Image.ImageID != 1 {
		t.Errorf("ImageID = %d, want 1", s.Image.ImageID)
	}
}

func TestEditorWithImage_WithToolbar(t *testing.T) {
	doc := Build(
		S("Hello "),
		S("\uFFFC", SpanStyle{Image: ImageAttachment{ImageID: 99, Width: 20, Height: 20}}),
		S(" World"),
	)
	el := NewEditorWithToolbar(doc)
	if _, ok := el.(ui.WidgetElement); !ok {
		t.Fatalf("expected WidgetElement, got %T", el)
	}
}

// ── Extended Style Fields ──────────────────────────────────────

func TestEditorDoc_StrikethroughPreserved(t *testing.T) {
	doc := Build(
		S("normal "),
		S("struck", SpanStyle{Strikethrough: true}),
	)
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	if editor.Value.RunAt(7).Strikethrough != true {
		t.Error("strikethrough should be preserved in editor value")
	}
}

func TestEditorDoc_FontFamilyPreserved(t *testing.T) {
	doc := Build(
		S("default "),
		S("mono", SpanStyle{FontFamily: "Monospace"}),
	)
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	if editor.Value.RunAt(9).FontFamily != "Monospace" {
		t.Errorf("FontFamily = %q, want Monospace", editor.Value.RunAt(9).FontFamily)
	}
}

func TestEditorDoc_WeightPreserved(t *testing.T) {
	doc := Styled("light", SpanStyle{Weight: draw.FontWeightLight})
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	if editor.Value.RunAt(0).Weight != draw.FontWeightLight {
		t.Errorf("Weight = %d, want Light (300)", editor.Value.RunAt(0).Weight)
	}
}

func TestEditorDoc_TrackingPreserved(t *testing.T) {
	doc := Styled("spaced", SpanStyle{Tracking: 0.15})
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	if editor.Value.RunAt(0).Tracking != 0.15 {
		t.Errorf("Tracking = %g, want 0.15", editor.Value.RunAt(0).Tracking)
	}
}

func TestEditorDoc_BgColorPreserved(t *testing.T) {
	bg := draw.Hex("#ffff00")
	doc := Styled("highlight", SpanStyle{BgColor: bg})
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	if editor.Value.RunAt(0).BgColor != bg {
		t.Error("BgColor should be preserved in editor value")
	}
}

func TestEditorDoc_AllNewFieldsCombined(t *testing.T) {
	style := SpanStyle{
		Bold:          true,
		Italic:        true,
		Underline:     true,
		Strikethrough: true,
		FontFamily:    "Serif",
		Weight:        draw.FontWeightBlack,
		Color:         draw.Hex("#ff0000"),
		BgColor:       draw.Hex("#00ff00"),
		Size:          24,
		Tracking:      0.1,
		LineHeight:    2.0,
		WhiteSpace:    WhiteSpacePreWrap,
	}
	doc := Styled("all", style)
	el := New(doc, WithReadOnly())
	editor := el.(RichTextEditor)
	got := editor.Value.RunAt(0)

	if !got.Strikethrough {
		t.Error("Strikethrough lost")
	}
	if got.FontFamily != "Serif" {
		t.Errorf("FontFamily = %q", got.FontFamily)
	}
	if got.Weight != draw.FontWeightBlack {
		t.Errorf("Weight = %d", got.Weight)
	}
	if got.BgColor != draw.Hex("#00ff00") {
		t.Error("BgColor lost")
	}
	if got.Tracking != 0.1 {
		t.Errorf("Tracking = %g", got.Tracking)
	}
	if got.LineHeight != 2.0 {
		t.Errorf("LineHeight = %g", got.LineHeight)
	}
	if got.WhiteSpace != WhiteSpacePreWrap {
		t.Errorf("WhiteSpace = %d", got.WhiteSpace)
	}
}
