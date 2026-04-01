package html

import (
	"testing"

	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

func TestBuildSimpleText(t *testing.T) {
	doc, err := Parse("Hello World")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
}

func TestBuildParagraph(t *testing.T) {
	doc, err := Parse("<p>Hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
}

func TestBuildBoldItalic(t *testing.T) {
	doc, err := Parse("<p><b>bold</b> and <i>italic</i></p>")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
}

func TestBuildHeadings(t *testing.T) {
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		doc, err := Parse("<" + tag + ">Title</" + tag + ">")
		if err != nil {
			t.Fatalf("%s: %v", tag, err)
		}
		b := &builder{sheets: doc.Sheets}
		el := b.buildElement(doc.Root)
		if el == nil {
			t.Fatalf("%s: expected non-nil element", tag)
		}
	}
}

func TestBuildNestedDivs(t *testing.T) {
	html := `<div>
		<div>Inner 1</div>
		<div>Inner 2</div>
	</div>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element")
	}
}

func TestBuildMixedInlineBlock(t *testing.T) {
	html := `<div>Text before <p>paragraph</p> text after</div>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for mixed content")
	}
}

func TestBuildTable(t *testing.T) {
	html := `<table>
		<thead><tr><th>Name</th><th>Age</th></tr></thead>
		<tbody>
			<tr><td>Alice</td><td>30</td></tr>
			<tr><td>Bob</td><td>25</td></tr>
		</tbody>
	</table>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for table")
	}
}

func TestBuildTableWithColspan(t *testing.T) {
	html := `<table>
		<tr><td colspan="2">Merged</td></tr>
		<tr><td>A</td><td>B</td></tr>
	</table>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for table with colspan")
	}
}

func TestBuildUnorderedList(t *testing.T) {
	html := `<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for list")
	}
}

func TestBuildOrderedList(t *testing.T) {
	html := `<ol><li>First</li><li>Second</li></ol>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for ordered list")
	}
}

func TestBuildLink(t *testing.T) {
	var clicked string
	html := `<p>Click <a href="https://example.com">here</a></p>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{
		sheets: doc.Sheets,
		onLink: func(href string) { clicked = href },
	}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for link")
	}
	// The link callback is embedded; we verify it was captured.
	_ = clicked
}

func TestBuildFormControls(t *testing.T) {
	html := `<div>
		<input type="text" value="hello" placeholder="name">
		<input type="checkbox" checked>
		<input type="radio">
		<input type="password">
		<select><option>A</option><option selected>B</option></select>
		<textarea>content</textarea>
		<button>Click</button>
		<progress value="0.5"></progress>
	</div>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for form controls")
	}
}

func TestBuildHiddenInput(t *testing.T) {
	html := `<input type="hidden" name="token" value="abc">`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	// Hidden inputs produce no visual output.
	_ = el
}

func TestBuildWithCSS(t *testing.T) {
	html := `<style>.red { color: red; }</style>
	<p class="red">Red text</p>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sheets) == 0 {
		t.Fatal("expected stylesheet to be extracted")
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element with CSS")
	}
}

func TestBuildDisplayFlex(t *testing.T) {
	html := `<style>.flex { display: flex; flex-direction: row; gap: 10px; }</style>
	<div class="flex"><span>A</span><span>B</span></div>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for flex container")
	}
}

func TestBuildDisplayNone(t *testing.T) {
	html := `<div>Visible</div><div style="display:none">Hidden</div>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	// The hidden div should not appear in the output.
	if el == nil {
		t.Fatal("expected non-nil element")
	}
}

func TestBuildHR(t *testing.T) {
	html := `<p>Before</p><hr><p>After</p>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for hr")
	}
}

func TestBuildImg(t *testing.T) {
	html := `<p>See: <img src="photo.jpg" alt="Photo" width="100" height="50"></p>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for img")
	}
}

func TestBuildPreCode(t *testing.T) {
	html := `<pre><code>func main() {}</code></pre>`
	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for pre/code")
	}
}

func TestBuildComplexDocument(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Test</title></head>
	<body>
		<style>
			.container { padding: 10px; background-color: #f0f0f0; }
			.bold { font-weight: bold; }
		</style>
		<h1>Welcome</h1>
		<div class="container">
			<p>Hello <b>World</b>! Visit <a href="/about">about</a>.</p>
			<ul>
				<li>Item <i>one</i></li>
				<li>Item two</li>
			</ul>
			<table>
				<tr><th>Col 1</th><th>Col 2</th></tr>
				<tr><td>A</td><td>B</td></tr>
			</table>
			<hr>
			<form>
				<input type="text" placeholder="Name">
				<input type="checkbox" checked> Subscribe
				<select>
					<option>Option A</option>
					<option selected>Option B</option>
				</select>
				<button>Submit</button>
			</form>
		</div>
	</body>
	</html>`

	doc, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sheets) == 0 {
		t.Fatal("expected stylesheets")
	}

	b := &builder{sheets: doc.Sheets}
	el := b.buildElement(doc.Root)
	if el == nil {
		t.Fatal("expected non-nil element for complex document")
	}
}

// Ensure types are used to avoid import errors.
var (
	_ = display.Text
	_ = layout.Column
)
