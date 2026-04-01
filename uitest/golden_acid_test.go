package uitest

import (
	"testing"

	"github.com/timzifer/lux/richtext"
)

// ── Acid Visual Golden-File Tests ─────────────────────────────────
//
// These tests push real HTML through the full rendering pipeline:
//
//   HTML string
//     → richtext.FromHTML()   (DOM parse → CSS cascade → AttributedString)
//     → richtext.New(…)       (AttributedString → ui.Element, read-only)
//     → BuildScene()          (layout + paint → draw.Scene)
//     → AssertScene()         (serialize → golden-file diff)
//
// Run with -update to generate/update golden files:
//   go test ./uitest -run TestGoldenAcid -update
//
// The golden files capture the exact rendered scene (rects, glyphs,
// shadows, clips) so any regression in DOM parsing, CSS cascade,
// style mapping, text shaping, or layout is caught automatically.
//
// Programmatic verification: run without -update — any change in
// HTML parsing, CSS cascade, or rendering produces a diff failure.

const acidW = 600
const acidH = 800

// htmlElement creates a read-only richtext element from HTML, expanding
// the visible rows so all content is rendered (not clipped by the editor viewport).
func htmlElement(t *testing.T, html string) richtext.RichTextEditor {
	t.Helper()
	as, err := richtext.FromHTML(html)
	if err != nil {
		t.Fatalf("FromHTML: %v", err)
	}
	return richtext.RichTextEditor{
		Value:    as,
		ReadOnly: true,
		Rows:     40, // enough rows to render all content visually
	}
}

// ── Acid1: CSS1 Basics ───────────────────────────────────────────

const acid1Visual = `
<style>
  .header { background-color: #ffff00; color: #000; font-size: 24px; font-weight: bold; text-align: center }
  .pass   { color: green }
  .fail   { color: red }
  .fmt    { font-weight: bold; font-style: italic; text-decoration: underline }
  .nested { color: blue }
  .nested em { color: red; font-weight: bold }
</style>
<h1>Acid1 Lite</h1>
<div class="header">CSS1 Conformance</div>
<p class="pass">PASS: green text</p>
<p class="fail">FAIL: red text</p>
<p class="fmt">Bold italic underlined</p>
<div class="nested"><em>Red bold</em> in blue</div>
<p>Normal with <code>monospace</code> inline.</p>
`

func TestGoldenAcid1(t *testing.T) {
	el := htmlElement(t, acid1Visual)
	scene := BuildScene(el, acidW, acidH)
	AssertScene(t, scene, "testdata/acid1.golden")
}

// ── Acid2: CSS2.1 Fragments ─────────────────────────────────────
// The real Acid2 renders a smiley face with absolute positioning,
// ::before/::after, and clip. We test the subset lux can handle:
// inline formatting, colours, UA stylesheet, headings, lists.

const acid2Visual = `
<style>
  h2          { font-size: 1.5em; font-weight: bold; color: #333 }
  .eyes       { font-size: 32px; letter-spacing: 8px }
  .smile      { color: #cc6600; font-size: 20px; text-align: center }
  .label      { font-weight: bold; color: #336699 }
</style>
<h2>Acid2 Visual</h2>
<p class="eyes" style="text-align:center">O  O</p>
<p class="smile">\___/</p>
<p class="label">Features tested:</p>
<ul>
  <li><b>Bold</b> via UA stylesheet</li>
  <li><i>Italic</i> via UA stylesheet</li>
  <li><u>Underline</u> via UA stylesheet</li>
  <li>Inline <code>monospace</code></li>
  <li><span style="color:red">Inline style</span> override</li>
</ul>
`

func TestGoldenAcid2(t *testing.T) {
	el := htmlElement(t, acid2Visual)
	scene := BuildScene(el, acidW, acidH)
	AssertScene(t, scene, "testdata/acid2.golden")
}

// ── Acid3: CSS3 Selectors ────────────────────────────────────────
// The real Acid3 is mostly JavaScript. We exercise CSS3 selectors
// and verify that styled output reaches the golden file.

const acid3Visual = `
<style>
  [data-test]           { font-weight: bold }
  [data-test="hello"]   { color: purple }
  .a.b                  { color: teal; font-style: italic }
  div > p               { color: green }
  h3                    { font-size: 1.17em; font-weight: bold }
</style>
<h3>Acid3 Selectors</h3>
<div><p>Child combinator: green</p></div>
<p data-test="hello">Attribute selector: purple bold</p>
<p class="a b">Multiple classes: teal italic</p>
<ol>
  <li>Ordered item one</li>
  <li>Ordered item two</li>
  <li>Ordered item three</li>
</ol>
`

func TestGoldenAcid3(t *testing.T) {
	el := htmlElement(t, acid3Visual)
	scene := BuildScene(el, acidW, acidH)
	AssertScene(t, scene, "testdata/acid3.golden")
}
