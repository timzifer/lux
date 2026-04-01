package html

import (
	"strings"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// builder converts DOM nodes into ui.Element trees.
type builder struct {
	sheets []*css.StyleSheet
	onLink func(href string)
}

// buildElement converts a single DOM node (and its subtree) into a
// ui.Element. Returns nil for nodes that produce no visual output.
func (b *builder) buildElement(node *dom.Node) ui.Element {
	switch node.Type {
	case dom.DocumentNode:
		return b.buildDocumentNode(node)
	case dom.TextNode:
		return b.buildTextNode(node)
	case dom.ElementNode:
		return b.buildElementNode(node)
	default:
		return nil
	}
}

// buildDocumentNode builds the children of a document root.
func (b *builder) buildDocumentNode(node *dom.Node) ui.Element {
	children := b.buildChildren(node)
	switch len(children) {
	case 0:
		return nil
	case 1:
		return children[0]
	default:
		return layout.Column(children...)
	}
}

// buildTextNode converts a text node to a text element.
func (b *builder) buildTextNode(node *dom.Node) ui.Element {
	text := node.Data
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return display.Text(text)
}

// buildElementNode converts an element node based on its tag and
// computed CSS display value.
func (b *builder) buildElementNode(node *dom.Node) ui.Element {
	tag := strings.ToLower(node.Tag)
	style := css.Resolve(node, b.sheets)
	disp := resolveDisplay(node, style)

	// Skip hidden elements.
	if disp == "none" {
		return nil
	}

	// Handle special elements first.
	switch tag {
	case "style", "script", "head", "title", "meta", "link", "template":
		return nil
	case "br":
		return nil // handled by inline collector
	case "hr":
		return display.Divider()
	case "table":
		el := b.buildTable(node, style)
		return applyBoxStyle(el, style)
	case "input", "select", "textarea", "button", "progress":
		el := b.buildFormControl(node, style)
		if el == nil {
			return nil
		}
		return applyBoxStyle(el, style)
	case "img":
		return b.buildImg(node, style)
	}

	// Table sub-elements — handled by buildTable, skip if orphaned.
	if isTableDisplay(disp) && disp != "table" {
		return nil
	}

	// Dispatch based on display type.
	switch {
	case disp == "flex":
		return b.buildFlexContainer(node, style)
	case disp == "grid":
		return b.buildGridContainer(node, style)
	case disp == "inline-block":
		return b.buildInlineBlock(node, style)
	case disp == "list-item":
		return b.buildListItem(node, style)
	case isBlockDisplay(disp):
		return b.buildBlockElement(node, style, tag)
	default:
		// Inline elements — return nil here; they're handled by the
		// inline collector in the parent's buildChildren.
		return nil
	}
}

// buildChildren converts all children of a node into a list of
// block-level elements. This handles the critical inline/block mixing:
// consecutive inline nodes are collected into RichParagraphs.
func (b *builder) buildChildren(node *dom.Node) []ui.Element {
	var result []ui.Element
	ic := &inlineCollector{
		sheets: b.sheets,
		onLink: b.onLink,
	}

	for child := node.FirstChild; child != nil; child = child.NextSib {
		switch child.Type {
		case dom.TextNode:
			text := child.Data
			if strings.TrimSpace(text) == "" && !ic.hasContent() {
				continue // skip leading whitespace
			}
			parentStyle := css.Resolve(node, b.sheets)
			ic.addTextNode(child, parentStyle)

		case dom.ElementNode:
			childStyle := css.Resolve(child, b.sheets)
			childDisplay := resolveDisplay(child, childStyle)

			if childDisplay == "none" {
				continue
			}

			tag := strings.ToLower(child.Tag)

			// Skip non-visual.
			if tag == "style" || tag == "script" || tag == "head" || tag == "title" || tag == "meta" || tag == "link" || tag == "template" {
				continue
			}

			if tag == "br" {
				ic.items = append(ic.items, display.Span{Text: "\n"})
				continue
			}

			if isInlineDisplay(childDisplay) {
				ic.addInlineElement(child, childStyle)
			} else {
				// Block element encountered — flush inline content first.
				if para := ic.flush(); para != nil {
					result = append(result, display.RichText(*para))
				}

				el := b.buildElementNode(child)
				if el != nil {
					result = append(result, el)
				}
			}
		}
	}

	// Flush remaining inline content.
	if para := ic.flush(); para != nil {
		result = append(result, display.RichText(*para))
	}

	return result
}

// buildBlockElement builds a generic block-level element.
func (b *builder) buildBlockElement(node *dom.Node, style css.StyleDeclaration, tag string) ui.Element {
	children := b.buildChildren(node)

	var el ui.Element
	switch len(children) {
	case 0:
		// Empty block — may still have box styling.
		el = display.Text("")
	case 1:
		el = children[0]
	default:
		el = layout.Column(children...)
	}

	// Apply heading styles.
	if isHeading(tag) {
		el = applyHeadingStyle(el, tag, style)
	}

	return applyBoxStyle(el, style)
}

// buildFlexContainer builds a flex layout from a node with display:flex.
func (b *builder) buildFlexContainer(node *dom.Node, style css.StyleDeclaration) ui.Element {
	var children []ui.Element
	for child := node.FirstChild; child != nil; child = child.NextSib {
		el := b.buildElement(child)
		if el != nil {
			children = append(children, el)
		}
	}

	flex := toFlexContainer(style, children)
	return applyBoxStyle(flex, style)
}

// buildGridContainer builds a grid layout from a node with display:grid.
func (b *builder) buildGridContainer(node *dom.Node, style css.StyleDeclaration) ui.Element {
	var children []ui.Element
	for child := node.FirstChild; child != nil; child = child.NextSib {
		el := b.buildElement(child)
		if el != nil {
			children = append(children, el)
		}
	}

	// For now, use a simple column layout as grid placeholder.
	// Full CSS Grid property mapping can be added incrementally.
	el := layout.Column(children...)
	return applyBoxStyle(el, style)
}

// buildInlineBlock builds an inline-block element (rendered as a block
// but positioned inline).
func (b *builder) buildInlineBlock(node *dom.Node, style css.StyleDeclaration) ui.Element {
	children := b.buildChildren(node)

	var el ui.Element
	switch len(children) {
	case 0:
		el = display.Text("")
	case 1:
		el = children[0]
	default:
		el = layout.Column(children...)
	}

	return applyBoxStyle(el, style)
}

// buildListItem builds a <li> or list-item display element.
func (b *builder) buildListItem(node *dom.Node, style css.StyleDeclaration) ui.Element {
	children := b.buildChildren(node)

	// Determine list type from parent.
	listType := draw.ListTypeUnordered
	if node.Parent != nil {
		parentTag := strings.ToLower(node.Parent.Tag)
		if parentTag == "ol" {
			listType = draw.ListTypeOrdered
		}
	}

	// Determine list level.
	level := 0
	for p := node.Parent; p != nil; p = p.Parent {
		pt := strings.ToLower(p.Tag)
		if pt == "ul" || pt == "ol" {
			level++
		}
	}
	if level > 0 {
		level-- // Top-level list is level 0.
	}

	// Build paragraph style.
	ps := toParagraphStyle(style)
	ps.ListType = listType
	ps.ListLevel = level

	// Collect content: if children are simple text elements, merge
	// them into a single RichParagraph with list styling.
	if len(children) == 0 {
		para := display.RichParagraph{
			Content: []display.ParagraphContent{display.Span{Text: ""}},
			Style:   ps,
		}
		return display.RichText(para)
	}

	// If there's inline content, wrap it in a paragraph with list style.
	// Otherwise wrap the block children in a column with a marker.
	// For simplicity in Phase 1, collect inline text from children.
	ic := &inlineCollector{sheets: b.sheets, onLink: b.onLink}
	ic.paraStyle = ps
	var blockChildren []ui.Element

	for child := node.FirstChild; child != nil; child = child.NextSib {
		switch child.Type {
		case dom.TextNode:
			parentStyle := css.Resolve(node, b.sheets)
			ic.addTextNode(child, parentStyle)
		case dom.ElementNode:
			childStyle := css.Resolve(child, b.sheets)
			childDisplay := resolveDisplay(child, childStyle)
			if isInlineDisplay(childDisplay) {
				ic.addInlineElement(child, childStyle)
			} else {
				if para := ic.flush(); para != nil {
					blockChildren = append(blockChildren, display.RichText(*para))
				}
				el := b.buildElementNode(child)
				if el != nil {
					blockChildren = append(blockChildren, el)
				}
			}
		}
	}

	if para := ic.flush(); para != nil {
		// Set list style on the flushed paragraph.
		para.Style = ps
		return display.RichText(*para)
	}

	if len(blockChildren) == 1 {
		return blockChildren[0]
	}
	if len(blockChildren) > 1 {
		return layout.Column(blockChildren...)
	}

	// Fallback: use the already-built children.
	if len(children) == 1 {
		return children[0]
	}
	return layout.Column(children...)
}

// buildImg builds an <img> element.
func (b *builder) buildImg(node *dom.Node, style css.StyleDeclaration) ui.Element {
	alt := node.Attr("alt")
	// Image loading is deferred to future phases. For now, render
	// a placeholder with alt text.
	if alt != "" {
		return display.Text("[" + alt + "]")
	}
	return display.Text("[image]")
}

// ── Heading helpers ─────────────────────────────────────────────────

func isHeading(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

// applyHeadingStyle wraps content in a RichParagraph with heading-sized text.
func applyHeadingStyle(el ui.Element, tag string, style css.StyleDeclaration) ui.Element {
	size := headingSize(tag)

	// If the element is already a RichTextElement, apply style to its paragraphs.
	if rte, ok := el.(display.RichTextElement); ok {
		for i := range rte.Paragraphs {
			for j := range rte.Paragraphs[i].Spans {
				if rte.Paragraphs[i].Spans[j].Style.Style.Size == 0 {
					rte.Paragraphs[i].Spans[j].Style.Style.Size = size
				}
				if rte.Paragraphs[i].Spans[j].Style.Style.Weight == 0 {
					rte.Paragraphs[i].Spans[j].Style.Style.Weight = draw.FontWeightBold
				}
			}
			// Also update Content spans.
			for j, item := range rte.Paragraphs[i].Content {
				if span, ok := item.(display.Span); ok {
					if span.Style.Style.Size == 0 {
						span.Style.Style.Size = size
					}
					if span.Style.Style.Weight == 0 {
						span.Style.Style.Weight = draw.FontWeightBold
					}
					rte.Paragraphs[i].Content[j] = span
				}
			}
		}
		return rte
	}

	// Non-RichText child — return it as-is (heading style applied
	// via the UA stylesheet cascade in css.Resolve).
	return el
}

func headingSize(tag string) float32 {
	switch tag {
	case "h1":
		return 28
	case "h2":
		return 21
	case "h3":
		return 16.4
	case "h4":
		return 14
	case "h5":
		return 12
	case "h6":
		return 10
	}
	return 14
}
