package richtext

import (
	"testing"

	"github.com/timzifer/lux/draw"
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

// ── EditorToolbar ───────────────────────────────────────────────

func TestEditorToolbar(t *testing.T) {
	tb := &EditorToolbar{Bold: true, Italic: true, Underline: false}
	if !tb.Bold || !tb.Italic || tb.Underline {
		t.Fatal("unexpected toolbar state")
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
