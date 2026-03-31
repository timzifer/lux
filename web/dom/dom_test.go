package dom

import (
	"strings"
	"testing"
)

func TestParseHTML_PlainText(t *testing.T) {
	doc, err := ParseHTML("hello world")
	if err != nil {
		t.Fatal(err)
	}
	got := doc.TextContent()
	if got != "hello world" {
		t.Errorf("TextContent() = %q, want %q", got, "hello world")
	}
}

func TestParseHTML_Elements(t *testing.T) {
	doc, err := ParseHTML(`<p>Hello <b>bold</b> world</p>`)
	if err != nil {
		t.Fatal(err)
	}

	ps := doc.GetElementsByTagName("p")
	if len(ps) != 1 {
		t.Fatalf("expected 1 <p>, got %d", len(ps))
	}
	got := ps[0].TextContent()
	if got != "Hello bold world" {
		t.Errorf("p.TextContent() = %q, want %q", got, "Hello bold world")
	}

	bs := doc.GetElementsByTagName("b")
	if len(bs) != 1 {
		t.Fatalf("expected 1 <b>, got %d", len(bs))
	}
	if bs[0].TextContent() != "bold" {
		t.Errorf("b.TextContent() = %q, want %q", bs[0].TextContent(), "bold")
	}
}

func TestParseHTML_Attributes(t *testing.T) {
	doc, err := ParseHTML(`<span class="highlight" style="color:red">text</span>`)
	if err != nil {
		t.Fatal(err)
	}
	spans := doc.GetElementsByTagName("span")
	if len(spans) != 1 {
		t.Fatalf("expected 1 <span>, got %d", len(spans))
	}
	if spans[0].Attr("class") != "highlight" {
		t.Errorf("class = %q, want %q", spans[0].Attr("class"), "highlight")
	}
	if spans[0].Attr("style") != "color:red" {
		t.Errorf("style = %q, want %q", spans[0].Attr("style"), "color:red")
	}
}

func TestSerialize_Roundtrip(t *testing.T) {
	input := `<p>Hello <b>bold</b> world</p>`
	doc, err := ParseHTML(input)
	if err != nil {
		t.Fatal(err)
	}
	got := Serialize(doc)
	// The serialized output should contain the same structure.
	if !strings.Contains(got, "<b>bold</b>") {
		t.Errorf("Serialize missing <b>bold</b>, got: %s", got)
	}
	if !strings.Contains(got, "Hello") {
		t.Errorf("Serialize missing Hello, got: %s", got)
	}
}

func TestAppendChild_RemoveChild(t *testing.T) {
	parent := NewElement("div")
	child1 := NewElement("p")
	child2 := NewElement("span")

	parent.AppendChild(child1)
	parent.AppendChild(child2)

	children := parent.Children()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	parent.RemoveChild(child1)
	children = parent.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0] != child2 {
		t.Error("remaining child should be child2")
	}
}

func TestQuerySelector(t *testing.T) {
	doc, err := ParseHTML(`<div><p class="first">one</p><p class="second">two</p></div>`)
	if err != nil {
		t.Fatal(err)
	}

	found := doc.QuerySelector(".second")
	if found == nil {
		t.Fatal("QuerySelector(.second) returned nil")
	}
	if found.TextContent() != "two" {
		t.Errorf("TextContent() = %q, want %q", found.TextContent(), "two")
	}
}

func TestQuerySelectorAll(t *testing.T) {
	doc, err := ParseHTML(`<ul><li>a</li><li>b</li><li>c</li></ul>`)
	if err != nil {
		t.Fatal(err)
	}

	items := doc.QuerySelectorAll("li")
	if len(items) != 3 {
		t.Fatalf("expected 3 <li> elements, got %d", len(items))
	}
}
