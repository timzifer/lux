// Package web runs standardised web conformance tests (Acid1, Acid2, Acid3
// fragments) against the lux DOM parser, CSS cascade, and richtext pipeline.
//
// These tests are "just for the lolz" — a cheerful reality check of how far
// the young renderer has come, not a production benchmark.
package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/timzifer/lux/richtext"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// ── Acid 1 ────────────────────────────────────────────────────────
// The original 1998 Acid1 test by Todd Fahrner exercises CSS1 basics:
// background colours, text colours, fonts, inline formatting.

const acid1HTML = `<!DOCTYPE html>
<html>
<head>
<title>Acid1 — CSS1 Test</title>
<style>
  body        { background: white; color: black; font-family: serif }
  .header     { background: #ff0; color: #000; font-size: 24px; font-weight: bold; text-align: center }
  .pass       { color: green }
  .fail       { color: red }
  .inline-fmt { font-weight: bold; font-style: italic; text-decoration: underline }
  .nested     { color: blue }
  .nested em  { color: red; font-weight: bold }
  code        { font-family: monospace }
  h1          { font-size: 2em; font-weight: bold }
  h2          { font-size: 1.5em; font-weight: bold }
  .inherit    { color: navy }
  .inherit span { /* should inherit navy from parent */ }
</style>
</head>
<body>
  <h1>Acid1 Lite</h1>
  <div class="header">CSS1 Conformance Test</div>
  <p class="pass">This text should be green.</p>
  <p class="fail">This text should be red.</p>
  <p class="inline-fmt">Bold italic underlined.</p>
  <div class="nested"><em>Red bold inside blue</em> and blue text</div>
  <p>Normal paragraph with <code>inline code</code> inside.</p>
  <div class="inherit">Navy parent <span>inherits navy</span></div>
</body>
</html>`

func TestAcid1_DOMParsing(t *testing.T) {
	sc := &scorecard{name: "Acid1/DOM"}

	doc, err := dom.ParseHTML(acid1HTML)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// T1: Document structure — ParseHTML uses fragment parsing, so
	// html/head/body are NOT preserved as explicit elements. We verify
	// the parser correctly flattens them and exposes their children.
	sc.check(t, "fragment parsing: no <html> wrapper",
		len(doc.GetElementsByTagName("html")) == 0)

	sc.check(t, "fragment parsing: <style> accessible",
		len(doc.GetElementsByTagName("style")) == 1)

	// T2: Element counts
	sc.check(t, "has <h1>",
		len(doc.GetElementsByTagName("h1")) == 1)

	ps := doc.GetElementsByTagName("p")
	sc.check(t, "≥3 <p> elements",
		len(ps) >= 3)

	sc.check(t, "2x <div> (header + nested)",
		len(doc.GetElementsByTagName("div")) >= 2)

	// T3: Class selectors
	pass := doc.QuerySelector(".pass")
	sc.check(t, "QuerySelector(.pass) finds element",
		pass != nil)

	fail := doc.QuerySelector(".fail")
	sc.check(t, "QuerySelector(.fail) finds element",
		fail != nil)

	// T4: Text content extraction
	if pass != nil {
		sc.check(t, ".pass text content",
			pass.TextContent() == "This text should be green.")
	}

	// T5: Nested selectors
	nested := doc.QuerySelector(".nested em")
	sc.check(t, "descendant selector .nested em",
		nested != nil)

	// T6: Code element
	codes := doc.GetElementsByTagName("code")
	sc.check(t, "<code> element found",
		len(codes) == 1)

	if len(codes) > 0 {
		sc.check(t, "<code> text content",
			codes[0].TextContent() == "inline code")
	}

	// T7: Attribute access
	header := doc.QuerySelector(".header")
	sc.check(t, ".header class attribute",
		header != nil && header.Attr("class") == "header")

	sc.report(t)
}

func TestAcid1_CSSCascade(t *testing.T) {
	sc := &scorecard{name: "Acid1/CSS"}

	doc, err := dom.ParseHTML(acid1HTML)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Extract stylesheets.
	var sheets []*css.StyleSheet
	for _, el := range doc.GetElementsByTagName("style") {
		if sheet, err := css.ParseStyleSheet(el.TextContent()); err == nil {
			sheets = append(sheets, sheet)
		}
	}
	sc.check(t, "parsed author stylesheet", len(sheets) > 0)

	// T1: .pass → color: green
	if pass := doc.QuerySelector(".pass"); pass != nil {
		style := css.Resolve(pass, sheets)
		sc.check(t, ".pass color=green",
			style.Get("color") == "green")
	}

	// T2: .fail → color: red
	if fail := doc.QuerySelector(".fail"); fail != nil {
		style := css.Resolve(fail, sheets)
		sc.check(t, ".fail color=red",
			style.Get("color") == "red")
	}

	// T3: .header → background + color + bold + center
	if header := doc.QuerySelector(".header"); header != nil {
		style := css.Resolve(header, sheets)
		sc.check(t, ".header background=#ff0",
			style.Get("background") == "#ff0")
		sc.check(t, ".header font-weight=bold",
			style.Get("font-weight") == "bold")
		sc.check(t, ".header text-align=center",
			style.Get("text-align") == "center")
		sc.check(t, ".header font-size=24px",
			style.Get("font-size") == "24px")
	}

	// T4: .inline-fmt → bold + italic + underline
	if inl := doc.QuerySelector(".inline-fmt"); inl != nil {
		style := css.Resolve(inl, sheets)
		sc.check(t, ".inline-fmt font-weight=bold",
			style.Get("font-weight") == "bold")
		sc.check(t, ".inline-fmt font-style=italic",
			style.Get("font-style") == "italic")
		sc.check(t, ".inline-fmt text-decoration=underline",
			style.Get("text-decoration") == "underline")
	}

	// T5: Descendant selector — .nested em gets color: red; font-weight: bold
	if em := doc.QuerySelector(".nested em"); em != nil {
		style := css.Resolve(em, sheets)
		sc.check(t, ".nested em color=red",
			style.Get("color") == "red")
		sc.check(t, ".nested em font-weight=bold",
			style.Get("font-weight") == "bold")
	}

	// T6: UA stylesheet — <code> gets font-family: monospace
	if code := doc.QuerySelector("code"); code != nil {
		style := css.Resolve(code, sheets)
		sc.check(t, "<code> font-family=monospace",
			style.Get("font-family") == "monospace")
	}

	// T7: <h1> gets font-size: 2em, font-weight: bold
	if h1 := doc.QuerySelector("h1"); h1 != nil {
		style := css.Resolve(h1, sheets)
		sc.check(t, "<h1> font-weight=bold",
			style.Get("font-weight") == "bold")
		sc.check(t, "<h1> font-size=2em",
			style.Get("font-size") == "2em")
	}

	// T8: Inheritance — .inherit span should inherit navy
	if span := doc.QuerySelector(".inherit span"); span != nil {
		style := css.Resolve(span, sheets)
		sc.check(t, ".inherit span inherits color=navy",
			style.Get("color") == "navy")
	}

	sc.report(t)
}

func TestAcid1_Richtext(t *testing.T) {
	sc := &scorecard{name: "Acid1/Richtext"}

	as, err := richtext.FromHTML(acid1HTML)
	if err != nil {
		t.Fatalf("FromHTML failed: %v", err)
	}

	sc.check(t, "produces non-empty attributed string",
		!as.IsEmpty())

	sc.check(t, "contains 'CSS1 Conformance Test'",
		strings.Contains(as.Text, "CSS1 Conformance Test"))

	sc.check(t, "contains 'green' text",
		strings.Contains(as.Text, "This text should be green."))

	sc.check(t, "contains 'red' text",
		strings.Contains(as.Text, "This text should be red."))

	sc.check(t, "contains inline code",
		strings.Contains(as.Text, "inline code"))

	sc.check(t, "preserves heading text",
		strings.Contains(as.Text, "Acid1 Lite"))

	sc.report(t)
}

// ── Acid 2 ────────────────────────────────────────────────────────
// Acid2 tests CSS2.1: box model, positioning, generated content,
// table layout, and the smiley face. We test the fragments we can
// handle — and cheerfully document what explodes.

const acid2HTML = `<!DOCTYPE html>
<html>
<head>
<title>Acid2 Fragments</title>
<style>
  /* Box model basics */
  .box { width: 100px; height: 100px; margin: 10px; padding: 5px;
         border: 2px solid black; background: yellow }

  /* Table layout */
  table       { border-collapse: collapse }
  td          { border: 1px solid #ccc; padding: 8px }
  .highlight  { background: #ffc; color: #333 }

  /* Float (not yet supported — expect failure) */
  .left       { float: left; width: 50px }
  .right      { float: right; width: 50px }

  /* Positioning (not yet supported — expect failure) */
  .absolute   { position: absolute; top: 0; left: 0 }
  .relative   { position: relative; top: 10px }
  .fixed      { position: fixed; bottom: 0 }

  /* Generated content (not yet supported — expect failure) */
  .before::before { content: "BEFORE" }
  .after::after   { content: "AFTER" }

  /* Overflow (not yet supported) */
  .hidden     { overflow: hidden }
  .scroll     { overflow: scroll }

  /* z-index (not yet supported) */
  .front      { z-index: 10 }
  .back       { z-index: -1 }

  /* Opacity */
  .translucent { opacity: 0.5 }

  /* Visibility */
  .invisible  { visibility: hidden }

  /* Inline-block (not yet supported) */
  .ib         { display: inline-block; width: 80px }

  /* Multi-selector */
  h2, h3      { font-weight: bold; color: #333 }
</style>
</head>
<body>
  <h2>Acid2 Fragments</h2>

  <div class="box">Yellow box</div>

  <table>
    <tr><td class="highlight">A1</td><td>B1</td></tr>
    <tr><td>A2</td><td class="highlight">B2</td></tr>
  </table>

  <div class="left">Float L</div>
  <div class="right">Float R</div>
  <div class="absolute">Absolute</div>
  <div class="relative">Relative</div>
  <div class="before">Content</div>
  <p>Paragraph with <b>bold</b> and <i>italic</i> and <em>emphasis</em>.</p>
  <p>Multi <strong>strong</strong> text.</p>
</body>
</html>`

func TestAcid2_DOMParsing(t *testing.T) {
	sc := &scorecard{name: "Acid2/DOM"}

	doc, err := dom.ParseHTML(acid2HTML)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sc.check(t, "has <table>",
		len(doc.GetElementsByTagName("table")) == 1)

	sc.check(t, "has 2x <tr>",
		len(doc.GetElementsByTagName("tr")) == 2)

	sc.check(t, "has 4x <td>",
		len(doc.GetElementsByTagName("td")) == 4)

	sc.check(t, "has .box",
		doc.QuerySelector(".box") != nil)

	sc.check(t, "has .highlight cells",
		len(doc.QuerySelectorAll(".highlight")) == 2)

	sc.check(t, "has <b>",
		len(doc.GetElementsByTagName("b")) == 1)

	sc.check(t, "has <i>",
		len(doc.GetElementsByTagName("i")) == 1)

	sc.check(t, "has <em>",
		len(doc.GetElementsByTagName("em")) == 1)

	sc.check(t, "has <strong>",
		len(doc.GetElementsByTagName("strong")) == 1)

	sc.report(t)
}

func TestAcid2_CSSCascade(t *testing.T) {
	sc := &scorecard{name: "Acid2/CSS"}

	doc, err := dom.ParseHTML(acid2HTML)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	var sheets []*css.StyleSheet
	for _, el := range doc.GetElementsByTagName("style") {
		if sheet, err := css.ParseStyleSheet(el.TextContent()); err == nil {
			sheets = append(sheets, sheet)
		}
	}

	// ── Supported features ──────────────────────────────────

	// Box model properties
	if box := doc.QuerySelector(".box"); box != nil {
		style := css.Resolve(box, sheets)
		sc.check(t, ".box background=yellow",
			style.Get("background") == "yellow")
		sc.check(t, ".box width=100px",
			style.Get("width") == "100px")
		sc.check(t, ".box height=100px",
			style.Get("height") == "100px")
		sc.check(t, ".box margin=10px",
			style.Get("margin") == "10px")
		sc.check(t, ".box padding=5px",
			style.Get("padding") == "5px")
	}

	// Table cell styles
	if td := doc.QuerySelector(".highlight"); td != nil {
		style := css.Resolve(td, sheets)
		sc.check(t, ".highlight background=#ffc",
			style.Get("background") == "#ffc")
		sc.check(t, ".highlight color=#333",
			style.Get("color") == "#333")
	}

	// Multi-selector (h2, h3)
	if h2 := doc.QuerySelector("h2"); h2 != nil {
		style := css.Resolve(h2, sheets)
		sc.check(t, "h2 font-weight=bold",
			style.Get("font-weight") == "bold")
		sc.check(t, "h2 color=#333",
			style.Get("color") == "#333")
	}

	// UA stylesheet semantics
	if b := doc.QuerySelector("b"); b != nil {
		style := css.Resolve(b, sheets)
		sc.check(t, "<b> UA font-weight=bold",
			style.Get("font-weight") == "bold")
	}

	if em := doc.QuerySelector("em"); em != nil {
		style := css.Resolve(em, sheets)
		sc.check(t, "<em> UA font-style=italic",
			style.Get("font-style") == "italic")
	}

	// ── Expected failures (not yet supported) ───────────────

	if abs := doc.QuerySelector(".absolute"); abs != nil {
		style := css.Resolve(abs, sheets)
		sc.checkExpectFail(t, "position: absolute (not implemented)",
			style.Get("position") == "absolute")
	}

	if rel := doc.QuerySelector(".relative"); rel != nil {
		style := css.Resolve(rel, sheets)
		sc.checkExpectFail(t, "position: relative (not implemented)",
			style.Get("position") == "relative")
	}

	if bef := doc.QuerySelector(".before"); bef != nil {
		style := css.Resolve(bef, sheets)
		sc.checkExpectFail(t, "::before content (not implemented)",
			style.Get("content") == `"BEFORE"`)
	}

	if fl := doc.QuerySelector(".left"); fl != nil {
		style := css.Resolve(fl, sheets)
		sc.checkExpectFail(t, "float: left (not implemented)",
			style.Get("float") == "left")
	}

	sc.report(t)
}

// ── Acid 3 ────────────────────────────────────────────────────────
// Acid3 is heavily JavaScript-dependent. We can only test the
// DOM/CSS subset. Expected score: very low (and that's fine!).

const acid3HTML = `<!DOCTYPE html>
<html>
<head>
<title>Acid3 Fragments</title>
<style>
  /* Selector combinators */
  div > p             { color: green }
  div + p             { color: blue }
  div ~ p             { font-style: italic }

  /* Attribute selectors */
  [data-test]         { font-weight: bold }
  [data-test="hello"] { color: purple }
  [data-test^="he"]   { text-decoration: underline }
  [data-test$="lo"]   { font-family: monospace }
  [data-test*="ell"]  { font-size: 18px }

  /* Pseudo-classes (limited support expected) */
  p:first-child       { color: orange }
  li:nth-child(2)     { color: red }
  a:link              { color: blue }
  a:visited           { color: purple }

  /* CSS3 selectors */
  :not(.skip)         { /* matches everything without .skip */ }
  .items > :last-child { font-weight: bold }

  /* Multiple classes */
  .a.b                { color: teal }
</style>
</head>
<body>
  <div>
    <p>Direct child</p>
  </div>
  <p>Adjacent sibling</p>
  <p>General sibling</p>

  <p data-test="hello">Attribute selector target</p>

  <ul class="items">
    <li>First</li>
    <li>Second (should be red)</li>
    <li>Third (should be bold)</li>
  </ul>

  <p class="a b">Multiple classes</p>
</body>
</html>`

func TestAcid3_Selectors(t *testing.T) {
	sc := &scorecard{name: "Acid3/Selectors"}

	doc, err := dom.ParseHTML(acid3HTML)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	var sheets []*css.StyleSheet
	for _, el := range doc.GetElementsByTagName("style") {
		if sheet, err := css.ParseStyleSheet(el.TextContent()); err == nil {
			sheets = append(sheets, sheet)
		}
	}

	// Child combinator: div > p
	// Note: p:first-child also matches this element with equal specificity
	// but appears later in the stylesheet, so it wins the cascade with
	// color: orange. We test that the selector MATCHES (not the final color).
	divP := doc.QuerySelector("div > p")
	sc.check(t, "div > p   selector matches", divP != nil)
	if divP != nil {
		style := css.Resolve(divP, sheets)
		// p:first-child overrides div>p (same specificity, later in source)
		sc.check(t, "div > p   cascade: p:first-child wins (orange)",
			style.Get("color") == "orange")
	}

	// Adjacent sibling: div + p
	if p := doc.QuerySelector("div + p"); p != nil {
		style := css.Resolve(p, sheets)
		sc.check(t, "div + p   color=blue",
			style.Get("color") == "blue")
	} else {
		sc.check(t, "div + p   selector match", false)
	}

	// General sibling: div ~ p
	allP := doc.QuerySelectorAll("div ~ p")
	sc.check(t, "div ~ p   matches ≥1 element",
		len(allP) >= 1)

	// Attribute selector: [data-test]
	if el := doc.QuerySelector("[data-test]"); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, "[data-test] font-weight=bold",
			style.Get("font-weight") == "bold")
	} else {
		sc.check(t, "[data-test] selector match", false)
	}

	// Attribute value: [data-test="hello"]
	if el := doc.QuerySelector(`[data-test="hello"]`); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, `[data-test="hello"] color=purple`,
			style.Get("color") == "purple")
	} else {
		sc.check(t, `[data-test="hello"] selector match`, false)
	}

	// Attribute prefix: [data-test^="he"]
	if el := doc.QuerySelector(`[data-test^="he"]`); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, `[data-test^="he"] underline`,
			style.Get("text-decoration") == "underline")
	} else {
		sc.check(t, `[data-test^="he"] selector match`, false)
	}

	// Attribute suffix: [data-test$="lo"]
	if el := doc.QuerySelector(`[data-test$="lo"]`); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, `[data-test$="lo"] font-family=monospace`,
			style.Get("font-family") == "monospace")
	} else {
		sc.check(t, `[data-test$="lo"] selector match`, false)
	}

	// Attribute substring: [data-test*="ell"]
	if el := doc.QuerySelector(`[data-test*="ell"]`); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, `[data-test*="ell"] font-size=18px`,
			style.Get("font-size") == "18px")
	} else {
		sc.check(t, `[data-test*="ell"] selector match`, false)
	}

	// :first-child
	if p := doc.QuerySelector("p:first-child"); p != nil {
		style := css.Resolve(p, sheets)
		sc.check(t, "p:first-child color=orange",
			style.Get("color") == "orange")
	} else {
		sc.checkExpectFail(t, "p:first-child (pseudo-class)", false)
	}

	// :nth-child(2)
	if li := doc.QuerySelector("li:nth-child(2)"); li != nil {
		style := css.Resolve(li, sheets)
		sc.check(t, "li:nth-child(2) color=red",
			style.Get("color") == "red")
	} else {
		sc.checkExpectFail(t, "li:nth-child(2) (pseudo-class)", false)
	}

	// Multiple classes: .a.b
	if el := doc.QuerySelector(".a.b"); el != nil {
		style := css.Resolve(el, sheets)
		sc.check(t, ".a.b color=teal",
			style.Get("color") == "teal")
	} else {
		sc.check(t, ".a.b selector match", false)
	}

	sc.report(t)
}

func TestAcid3_Richtext(t *testing.T) {
	sc := &scorecard{name: "Acid3/Richtext"}

	as, err := richtext.FromHTML(acid3HTML)
	if err != nil {
		t.Fatalf("FromHTML failed: %v", err)
	}

	sc.check(t, "non-empty output", !as.IsEmpty())
	sc.check(t, "contains 'Direct child'",
		strings.Contains(as.Text, "Direct child"))
	sc.check(t, "contains 'Attribute selector target'",
		strings.Contains(as.Text, "Attribute selector target"))
	sc.check(t, "contains list items",
		strings.Contains(as.Text, "First") &&
			strings.Contains(as.Text, "Second") &&
			strings.Contains(as.Text, "Third"))
	sc.check(t, "contains 'Multiple classes'",
		strings.Contains(as.Text, "Multiple classes"))

	sc.report(t)
}

// ── Inline Style Override ─────────────────────────────────────────
// Verify that inline styles beat everything (CSS specificity).

func TestInlineStyleOverride(t *testing.T) {
	sc := &scorecard{name: "Specificity"}

	html := `<html><head><style>
		p { color: blue; font-size: 14px }
		.red { color: red }
		#unique { color: green }
	</style></head><body>
		<p class="red" id="unique" style="color: purple">Override</p>
	</body></html>`

	doc, err := dom.ParseHTML(html)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	var sheets []*css.StyleSheet
	for _, el := range doc.GetElementsByTagName("style") {
		if sheet, err := css.ParseStyleSheet(el.TextContent()); err == nil {
			sheets = append(sheets, sheet)
		}
	}

	if p := doc.QuerySelector("p"); p != nil {
		style := css.Resolve(p, sheets)
		sc.check(t, "inline style beats class+id: color=purple",
			style.Get("color") == "purple")
		sc.check(t, "non-overridden property preserved: font-size=14px",
			style.Get("font-size") == "14px")
	}

	sc.report(t)
}

// ── Scorecard ─────────────────────────────────────────────────────

type scorecard struct {
	name    string
	passed  int
	failed  int
	xfail   int // expected failures
	results []result
}

type result struct {
	label     string
	passed    bool
	expectFail bool
}

func (s *scorecard) check(t *testing.T, label string, ok bool) {
	t.Helper()
	s.results = append(s.results, result{label: label, passed: ok})
	if ok {
		s.passed++
	} else {
		s.failed++
		t.Errorf("  FAIL: %s", label)
	}
}

func (s *scorecard) checkExpectFail(t *testing.T, label string, ok bool) {
	t.Helper()
	s.results = append(s.results, result{label: label, passed: ok, expectFail: true})
	if ok {
		s.passed++
		t.Logf("  SURPRISE PASS: %s", label)
	} else {
		s.xfail++
	}
}

func (s *scorecard) report(t *testing.T) {
	t.Helper()
	total := s.passed + s.failed + s.xfail
	pct := 0
	if total > 0 {
		pct = (s.passed * 100) / total
	}

	bar := renderBar(s.passed, s.failed, s.xfail, 40)

	t.Logf("")
	t.Logf("╔══════════════════════════════════════════════╗")
	t.Logf("║  %-42s  ║", s.name+" Scorecard")
	t.Logf("╠══════════════════════════════════════════════╣")
	t.Logf("║  %s  ║", bar)
	t.Logf("║  Passed: %2d  Failed: %2d  XFail: %2d  (%2d%%)  ║",
		s.passed, s.failed, s.xfail, pct)
	t.Logf("╠══════════════════════════════════════════════╣")

	for _, r := range s.results {
		icon := "PASS"
		if !r.passed && r.expectFail {
			icon := "XFAIL"
			_ = icon
			t.Logf("║  XFAIL %-38s ║", r.label)
		} else if !r.passed {
			t.Logf("║  FAIL  %-38s ║", r.label)
		} else if r.expectFail {
			t.Logf("║  WOW!  %-38s ║", r.label)
		} else {
			_ = icon
			t.Logf("║  PASS  %-38s ║", r.label)
		}
	}

	t.Logf("╚══════════════════════════════════════════════╝")
}

func renderBar(passed, failed, xfail, width int) string {
	total := passed + failed + xfail
	if total == 0 {
		return strings.Repeat("░", width)
	}

	green := (passed * width) / total
	red := (failed * width) / total
	yellow := width - green - red

	return fmt.Sprintf("\033[32m%s\033[31m%s\033[33m%s\033[0m",
		strings.Repeat("█", green),
		strings.Repeat("█", red),
		strings.Repeat("░", yellow))
}
