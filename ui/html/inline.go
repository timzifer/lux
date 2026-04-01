package html

import (
	"regexp"
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/link"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// collapseWhitespaceRe matches runs of whitespace (spaces, tabs, newlines).
var collapseWhitespaceRe = regexp.MustCompile(`[\s]+`)

// collapseWhitespace collapses runs of whitespace (tabs, newlines, spaces)
// into a single space, per CSS white-space: normal behavior.
func collapseWhitespace(s string) string {
	return collapseWhitespaceRe.ReplaceAllString(s, " ")
}

// inlineCollector accumulates inline content (text spans, inline widgets,
// images) and produces a display.RichParagraph when flushed.
//
// This handles the critical inline content assembly phase: consecutive
// inline-level DOM nodes are merged into a single RichParagraph with
// properly styled spans and embedded inline widgets.
type inlineCollector struct {
	items          []display.ParagraphContent
	paraStyle      display.ParagraphStyle
	sheets         []*css.StyleSheet
	onLink         func(href string)
	parentFontSize float32  // inherited font-size for em resolution
	builder        *builder // reference to the builder for block-in-inline handling
}

// hasContent returns true if the collector has accumulated any content.
func (ic *inlineCollector) hasContent() bool {
	return len(ic.items) > 0
}

// addTextNode adds a text node's content as a styled span.
func (ic *inlineCollector) addTextNode(node *dom.Node, style css.StyleDeclaration) {
	text := node.Data
	if text == "" {
		return
	}
	// Collapse whitespace unless white-space: pre.
	ws := style.Get("white-space")
	if ws != "pre" && ws != "pre-wrap" && ws != "pre-line" {
		text = collapseWhitespace(text)
		if text == " " && !ic.hasContent() {
			return // skip leading whitespace-only spans
		}
	}
	ss := toSpanStyle(style, ic.parentFontSize)

	// Split text into word-level spans so the RichText line-breaking
	// algorithm can wrap between words. Each word includes its trailing
	// whitespace to maintain correct spacing.
	words := splitIntoWords(text)
	for _, w := range words {
		ic.items = append(ic.items, display.Span{Text: w, Style: ss})
	}
}

// splitIntoWords splits text into word-level chunks, each including
// trailing whitespace. E.g. "the way" → ["the ", "way"].
// Leading whitespace is preserved as a separate entry.
func splitIntoWords(s string) []string {
	if s == "" {
		return nil
	}
	var words []string
	i := 0
	for i < len(s) {
		// Skip to start of word (consume leading spaces as their own chunk).
		if s[i] == ' ' {
			j := i
			for j < len(s) && s[j] == ' ' {
				j++
			}
			// Attach leading spaces to the next word if possible.
			if j < len(s) {
				// Find end of next word.
				k := j
				for k < len(s) && s[k] != ' ' {
					k++
				}
				words = append(words, s[i:k])
				i = k
			} else {
				// Trailing spaces only.
				words = append(words, s[i:j])
				i = j
			}
		} else {
			// Find end of word.
			j := i
			for j < len(s) && s[j] != ' ' {
				j++
			}
			words = append(words, s[i:j])
			i = j
		}
	}
	return words
}

// addInlineElement adds an inline-level element node to the collector.
// It handles text formatting elements (<b>, <i>, <a>, etc.) by
// recursively collecting their children with resolved styles.
func (ic *inlineCollector) addInlineElement(node *dom.Node, style css.StyleDeclaration) {
	tag := strings.ToLower(node.Tag)

	switch tag {
	case "br":
		ic.items = append(ic.items, display.Span{Text: "\n"})
		return
	case "img":
		ic.addImage(node)
		return
	case "a":
		ic.addLink(node, style)
		return
	case "input", "select", "textarea", "button", "progress":
		// Form controls: build via the builder and embed as inline widget.
		if ic.builder != nil {
			el := ic.builder.buildElementNode(node)
			if el != nil {
				ic.items = append(ic.items, display.InlineElement(el))
			}
		}
		return
	}

	// For inline-block elements without special handling, build via
	// the builder and embed as inline widget.
	childDisplay := resolveDisplay(node, style)
	if childDisplay == "inline-block" && ic.builder != nil {
		el := ic.builder.buildElementNode(node)
		if el != nil {
			ic.items = append(ic.items, display.InlineElement(el))
		}
		return
	}

	// For text formatting elements (<b>, <i>, <span>, <code>, etc.),
	// recurse into children with the resolved style.
	for child := node.FirstChild; child != nil; child = child.NextSib {
		switch child.Type {
		case dom.TextNode:
			childStyle := css.Resolve(node, ic.sheets) // style from the formatting element
			ic.addTextNode(child, childStyle)
		case dom.ElementNode:
			childStyle := css.Resolve(child, ic.sheets)
			childDisplay := resolveDisplay(child, childStyle)
			if isInlineDisplay(childDisplay) {
				ic.addInlineElement(child, childStyle)
			} else {
				// Block-in-inline: wrap as block widget.
				// This is unusual HTML but we handle it gracefully.
				ic.addBlockWidget(child, childStyle)
			}
		}
	}
}

// addLink adds an <a> element as an inline link widget.
func (ic *inlineCollector) addLink(node *dom.Node, style css.StyleDeclaration) {
	href := node.Attr("href")

	// Collect link text content.
	var linkText strings.Builder
	collectText(node, &linkText)
	label := linkText.String()

	onClick := func() {}
	if ic.onLink != nil && href != "" {
		capturedHref := href
		onClick = func() { ic.onLink(capturedHref) }
	}

	var linkEl ui.Element
	if href != "" {
		linkEl = link.WithURL(label, href, onClick)
	} else {
		linkEl = link.Text(label, onClick)
	}

	ic.items = append(ic.items, display.InlineElement(linkEl))
}

// addImage adds an <img> element as an inline image.
func (ic *inlineCollector) addImage(node *dom.Node) {
	alt := node.Attr("alt")
	// We create a placeholder ImageSpan. The actual image loading
	// would be handled by a resource loader in later phases.
	var opts []display.ImageSpanOption
	if w := node.Attr("width"); w != "" {
		if wv, ok := css.ParseDimension(w); ok {
			if h := node.Attr("height"); h != "" {
				if hv, ok := css.ParseDimension(h); ok {
					opts = append(opts, display.WithImageSpanSize(wv, hv))
				}
			} else {
				opts = append(opts, display.WithImageSpanSize(wv, wv))
			}
		}
	}
	if alt != "" {
		opts = append(opts, display.WithImageSpanAlt(alt))
	}
	ic.items = append(ic.items, display.InlineImage(draw.ImageID(0), opts...))
}

// addBlockWidget adds a block-level element as an inline block widget.
// This is used when a block element appears inside an inline context.
func (ic *inlineCollector) addBlockWidget(node *dom.Node, style css.StyleDeclaration) {
	// If we have a builder reference, properly build the block element
	// and embed it as an inline widget.
	if ic.builder != nil {
		el := ic.builder.buildElementNode(node)
		if el != nil {
			ic.items = append(ic.items, display.InlineElement(el))
			return
		}
	}
	// Fallback: extract text content only.
	var text strings.Builder
	collectText(node, &text)
	if text.Len() > 0 {
		ss := toSpanStyle(style, ic.parentFontSize)
		ic.items = append(ic.items, display.Span{Text: text.String(), Style: ss})
	}
}

// flush returns a RichParagraph from the accumulated content and resets
// the collector. Returns nil if there is no content.
func (ic *inlineCollector) flush() *display.RichParagraph {
	if len(ic.items) == 0 {
		return nil
	}
	para := &display.RichParagraph{
		Content: ic.items,
		Style:   ic.paraStyle,
	}
	ic.items = nil
	ic.paraStyle = display.ParagraphStyle{}
	return para
}

// setParagraphStyle sets the paragraph-level style for the next flush.
func (ic *inlineCollector) setParagraphStyle(style css.StyleDeclaration) {
	ic.paraStyle = toParagraphStyle(style)
}

// collectText recursively collects all text content from a DOM subtree.
func collectText(node *dom.Node, b *strings.Builder) {
	if node.Type == dom.TextNode {
		b.WriteString(node.Data)
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSib {
		collectText(child, b)
	}
}
