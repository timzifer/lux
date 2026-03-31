package richtext

import (
	"strings"
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestFromHTML_PlainText(t *testing.T) {
	as, err := FromHTML("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "hello world" {
		t.Errorf("Text = %q, want %q", as.Text, "hello world")
	}
	if len(as.Attrs) != 0 {
		t.Errorf("expected 0 attrs, got %d", len(as.Attrs))
	}
}

func TestFromHTML_Bold(t *testing.T) {
	as, err := FromHTML("<b>bold text</b>")
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "bold text" {
		t.Errorf("Text = %q, want %q", as.Text, "bold text")
	}
	style := as.ResolveAt(0)
	if !style.Bold {
		t.Error("expected Bold at offset 0")
	}
}

func TestFromHTML_Strong(t *testing.T) {
	as, err := FromHTML("<strong>strong text</strong>")
	if err != nil {
		t.Fatal(err)
	}
	style := as.ResolveAt(0)
	if !style.Bold {
		t.Error("expected Bold for <strong>")
	}
}

func TestFromHTML_Nested(t *testing.T) {
	as, err := FromHTML("<b><i>bold italic</i></b>")
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "bold italic" {
		t.Errorf("Text = %q, want %q", as.Text, "bold italic")
	}
	style := as.ResolveAt(0)
	if !style.Bold {
		t.Error("expected Bold")
	}
	if !style.Italic {
		t.Error("expected Italic")
	}
}

func TestFromHTML_InlineStyle(t *testing.T) {
	as, err := FromHTML(`<span style="color:red;font-size:20px">styled</span>`)
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "styled" {
		t.Errorf("Text = %q, want %q", as.Text, "styled")
	}
	style := as.ResolveAt(0)
	if style.Color == (draw.Color{}) {
		t.Error("expected non-zero Color")
	}
	if style.Size == 0 {
		t.Error("expected non-zero Size")
	}
}

func TestFromHTML_StyleBlock(t *testing.T) {
	html := `<style>.red { color: red }</style><span class="red">colored</span>`
	as, err := FromHTML(html)
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "colored" {
		t.Errorf("Text = %q, want %q", as.Text, "colored")
	}
	style := as.ResolveAt(0)
	if style.Color == (draw.Color{}) {
		t.Error("expected non-zero Color from <style> block")
	}
}

func TestFromHTML_Paragraphs(t *testing.T) {
	as, err := FromHTML("<p>first</p><p>second</p>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(as.Text, "first") || !strings.Contains(as.Text, "second") {
		t.Errorf("Text = %q, expected both paragraphs", as.Text)
	}
	if !strings.Contains(as.Text, "\n") {
		t.Error("expected newline between paragraphs")
	}
}

func TestFromHTML_Lists(t *testing.T) {
	as, err := FromHTML("<ul><li>item one</li><li>item two</li></ul>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(as.Text, "item one") {
		t.Errorf("Text = %q, missing list content", as.Text)
	}
	// Check that list attributes are set.
	style := as.ResolveAt(0)
	if style.ListType != draw.ListTypeUnordered {
		t.Errorf("ListType = %v, want Unordered", style.ListType)
	}
}

func TestFromHTML_NestedLists(t *testing.T) {
	html := `<ul><li>outer</li><ul><li>inner</li></ul></ul>`
	as, err := FromHTML(html)
	if err != nil {
		t.Fatal(err)
	}
	// Find "inner" text offset.
	idx := strings.Index(as.Text, "inner")
	if idx < 0 {
		t.Fatalf("Text = %q, missing 'inner'", as.Text)
	}
	style := as.ResolveAt(idx)
	if style.ListLevel < 1 {
		t.Errorf("ListLevel = %d, want >= 1 for nested list", style.ListLevel)
	}
}

func TestFromHTML_Headings(t *testing.T) {
	as, err := FromHTML("<h1>Title</h1>")
	if err != nil {
		t.Fatal(err)
	}
	if as.Text != "Title" {
		t.Errorf("Text = %q, want %q", as.Text, "Title")
	}
	style := as.ResolveAt(0)
	if !style.Bold {
		t.Error("expected Bold for <h1>")
	}
	if style.Size == 0 {
		t.Error("expected non-zero Size for <h1>")
	}
}

func TestFromHTML_WhiteSpace(t *testing.T) {
	as, err := FromHTML(`<pre>  preserved  </pre>`)
	if err != nil {
		t.Fatal(err)
	}
	style := as.ResolveAt(0)
	if style.WhiteSpace != WhiteSpacePre {
		t.Errorf("WhiteSpace = %v, want Pre", style.WhiteSpace)
	}
}

func TestFromHTML_Br(t *testing.T) {
	as, err := FromHTML("line1<br>line2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(as.Text, "\n") {
		t.Errorf("Text = %q, expected newline for <br>", as.Text)
	}
}

func TestToHTML_Basic(t *testing.T) {
	as := NewAttributedString("hello world")
	html := as.ToHTML()
	if !strings.Contains(html, "hello world") {
		t.Errorf("ToHTML() = %q, missing text", html)
	}
}

func TestToHTML_SemanticTags(t *testing.T) {
	as := Styled("bold text", SpanStyle{Bold: true})
	html := as.ToHTML()
	if !strings.Contains(html, "<b>") {
		t.Errorf("ToHTML() = %q, expected <b> tag", html)
	}
}

func TestToHTML_Lists(t *testing.T) {
	as := NewAttributedString("item one\nitem two")
	as = as.Apply(0, len(as.Text), ListTypeAttr(draw.ListTypeUnordered))
	html := as.ToHTML()
	if !strings.Contains(html, "<ul>") {
		t.Errorf("ToHTML() = %q, expected <ul> tag", html)
	}
	if !strings.Contains(html, "<li>") {
		t.Errorf("ToHTML() = %q, expected <li> tag", html)
	}
}

func TestRoundtrip(t *testing.T) {
	input := `<b>bold</b> and <i>italic</i>`
	as1, err := FromHTML(input)
	if err != nil {
		t.Fatal(err)
	}
	html := as1.ToHTML()
	as2, err := FromHTML(html)
	if err != nil {
		t.Fatal(err)
	}
	if as1.Text != as2.Text {
		t.Errorf("roundtrip text mismatch: %q vs %q", as1.Text, as2.Text)
	}
}
