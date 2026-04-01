package html

import (
	"testing"
)

func TestParseDocument(t *testing.T) {
	doc, err := Parse("<p>Hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root == nil {
		t.Fatal("expected non-nil root")
	}
}

func TestParseDocumentWithStyle(t *testing.T) {
	doc, err := Parse(`<style>.x { color: red; }</style><p class="x">Red</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sheets) == 0 {
		t.Fatal("expected extracted stylesheet")
	}
}

func TestAddCSS(t *testing.T) {
	doc, err := Parse("<p>Test</p>")
	if err != nil {
		t.Fatal(err)
	}
	err = doc.AddCSS("p { font-weight: bold; }")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(doc.Sheets))
	}
}

func TestViewConvenience(t *testing.T) {
	el := View("<h1>Hello</h1><p>World</p>")
	if el == nil {
		t.Fatal("expected non-nil element from View()")
	}
}

func TestViewWithOptions(t *testing.T) {
	var linkHref string
	el := View("<a href=\"/test\">Link</a>",
		WithOnLink(func(href string) { linkHref = href }),
		WithMaxWidth(600),
		WithScrollable(400),
	)
	if el == nil {
		t.Fatal("expected non-nil element with options")
	}
	_ = linkHref
}

func TestViewFromDocument(t *testing.T) {
	doc, err := Parse("<p>Test</p>")
	if err != nil {
		t.Fatal(err)
	}
	el := ViewFromDocument(doc)
	if el == nil {
		t.Fatal("expected non-nil element from ViewFromDocument()")
	}
}

func TestViewEmptyHTML(t *testing.T) {
	el := View("")
	if el == nil {
		t.Fatal("expected non-nil element for empty HTML")
	}
}

func TestViewInvalidHTML(t *testing.T) {
	// HTML parser is lenient — even malformed HTML should produce output.
	el := View("<div><p>Unclosed")
	if el == nil {
		t.Fatal("expected non-nil element for malformed HTML")
	}
}

func TestViewComplexHTML(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Test Page</title></head>
	<body>
		<style>
			body { font-family: Arial; }
			.highlight { background-color: yellow; }
			.flex { display: flex; gap: 10px; }
		</style>
		<h1>Page Title</h1>
		<p>Intro <b>bold</b> text with <a href="/link">a link</a>.</p>
		<div class="flex">
			<div>Column 1</div>
			<div>Column 2</div>
		</div>
		<table>
			<caption>Data Table</caption>
			<thead><tr><th>Header 1</th><th>Header 2</th></tr></thead>
			<tbody>
				<tr><td>Cell 1</td><td>Cell 2</td></tr>
				<tr><td colspan="2">Merged</td></tr>
			</tbody>
		</table>
		<ul>
			<li>Item 1</li>
			<li>Item 2
				<ul><li>Nested</li></ul>
			</li>
		</ul>
		<ol><li>First</li><li>Second</li></ol>
		<hr>
		<form>
			<input type="text" placeholder="Name">
			<input type="checkbox" checked>
			<select>
				<option>A</option>
				<option selected>B</option>
			</select>
			<textarea>Multi-line</textarea>
			<button>Submit</button>
		</form>
		<pre><code>code block</code></pre>
	</body>
	</html>`

	el := View(html, WithOnLink(func(string) {}), WithScrollable(600))
	if el == nil {
		t.Fatal("expected non-nil element for complex HTML")
	}
}
