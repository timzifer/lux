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

// fontSize returns the resolved font-size for a node.
func (b *builder) fontSize(node *dom.Node) float32 {
	style := css.Resolve(node, b.sheets)
	parentFS := b.parentFontSize(node)
	return resolvedFontSize(style, parentFS)
}

// parentFontSize returns the inherited font-size from a node's parent.
func (b *builder) parentFontSize(node *dom.Node) float32 {
	if node.Parent != nil && node.Parent.Type == dom.ElementNode {
		return b.fontSize(node.Parent)
	}
	return css.DefaultFontSize
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
	// Collapse whitespace (tabs, newlines) per CSS white-space: normal.
	text = collapseWhitespace(text)
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
		return applyBoxStyle(el, style, b.fontSize(node))
	case "input", "select", "textarea", "button", "progress":
		el := b.buildFormControl(node, style)
		if el == nil {
			return nil
		}
		return applyBoxStyle(el, style, b.fontSize(node))
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

// floatChild pairs an element with its float/clear metadata.
type floatChild struct {
	element ui.Element
	float   FloatSide
	clear   ClearSide
}

// resolveFloat returns the FloatSide for a CSS float value.
func resolveFloat(style css.StyleDeclaration) FloatSide {
	switch strings.TrimSpace(style.Get("float")) {
	case "left":
		return FloatLeft
	case "right":
		return FloatRight
	}
	return FloatNone
}

// resolveClear returns the ClearSide for a CSS clear value.
func resolveClear(style css.StyleDeclaration) ClearSide {
	switch strings.TrimSpace(style.Get("clear")) {
	case "left":
		return ClearLeft
	case "right":
		return ClearRight
	case "both":
		return ClearBoth
	}
	return ClearNone
}

// buildChildren converts all children of a node into a list of
// block-level elements. This handles the critical inline/block mixing:
// consecutive inline nodes are collected into RichParagraphs.
// If any children have float or clear set, a FloatLayout is returned.
func (b *builder) buildChildren(node *dom.Node) []ui.Element {
	children := b.collectChildren(node)

	// Check if any children use float or clear.
	hasFloat := false
	for _, c := range children {
		if c.float != FloatNone || c.clear != ClearNone {
			hasFloat = true
			break
		}
	}

	if !hasFloat {
		// No floats — return plain elements as before.
		result := make([]ui.Element, 0, len(children))
		for _, c := range children {
			result = append(result, c.element)
		}
		return result
	}

	// Wrap in a FloatLayout.
	floatChildren := make([]FloatChild, 0, len(children))
	for _, c := range children {
		floatChildren = append(floatChildren, FloatChild{
			Element: c.element,
			Float:   c.float,
			Clear:   c.clear,
		})
	}

	// Contain floats when the container establishes a BFC or has visual
	// styling (padding/border/background) that would visually frame the
	// floated children. Containers without any visual styling (like a
	// bare <ul> with margin:0 border:0 padding:0) let their floats
	// "collapse" — the container height is 0, allowing sibling elements
	// to start at the same Y position as the floats.
	containFloats := b.nodeEstablishesBFC(node) || b.hasVisualBoxStyle(node)

	return []ui.Element{FloatLayout{Children: floatChildren, ContainFloats: containFloats}}
}

// nodeEstablishesBFC returns true if a DOM node establishes a new
// Block Formatting Context. BFC containers contain their floated
// children (expand to fit them). Elements that establish a BFC:
// floated elements, overflow != visible, display:flex/grid/flow-root,
// and root elements (html, body).
func (b *builder) nodeEstablishesBFC(node *dom.Node) bool {
	if node.Type != dom.ElementNode {
		return false
	}
	tag := strings.ToLower(node.Tag)
	if tag == "html" || tag == "body" {
		return true
	}
	style := css.Resolve(node, b.sheets)
	if resolveFloat(style) != FloatNone {
		return true
	}
	overflow := strings.TrimSpace(style.Get("overflow"))
	if overflow == "hidden" || overflow == "auto" || overflow == "scroll" {
		return true
	}
	disp := resolveDisplay(node, style)
	if disp == "flex" || disp == "grid" || disp == "flow-root" {
		return true
	}
	return false
}

// hasVisualBoxStyle returns true if a node has CSS box styling (padding,
// border, or background) that would visually frame its content.
func (b *builder) hasVisualBoxStyle(node *dom.Node) bool {
	if node.Type != dom.ElementNode {
		return false
	}
	style := css.Resolve(node, b.sheets)
	if v := style.Get("padding"); v != "" && v != "0" && v != "0px" {
		return true
	}
	for _, side := range []string{"padding-top", "padding-right", "padding-bottom", "padding-left"} {
		if v := style.Get(side); v != "" && v != "0" && v != "0px" {
			return true
		}
	}
	if v := style.Get("border-width"); v != "" && v != "0" && v != "0px" {
		return true
	}
	if v := style.Get("border-style"); v != "" && v != "none" {
		if bw := style.Get("border-width"); bw != "" && bw != "0" && bw != "0px" {
			return true
		}
	}
	if v := style.Get("background-color"); v != "" {
		return true
	}
	if v := style.Get("background"); v != "" {
		return true
	}
	return false
}

// collectChildren builds all child elements with float/clear metadata.
// For non-BFC block children that contain floats, the floats are
// "hoisted" to the current level (CSS float propagation: floats escape
// non-BFC containers to the nearest BFC ancestor).
func (b *builder) collectChildren(node *dom.Node) []floatChild {
	var result []floatChild
	ic := &inlineCollector{
		sheets:         b.sheets,
		onLink:         b.onLink,
		parentFontSize: b.fontSize(node),
		builder:        b,
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
					result = append(result, floatChild{element: display.RichText(*para)})
				}

				childFloat := resolveFloat(childStyle)
				childClear := resolveClear(childStyle)

				// CSS float propagation: floats escape non-BFC containers
				// to the nearest BFC ancestor. If this child is a non-BFC,
				// non-visual block with floated descendants, hoist them.
				if childFloat == FloatNone && childClear == ClearNone &&
					!b.nodeEstablishesBFC(child) && !b.hasVisualBoxStyle(child) &&
					b.hasFloatedDescendants(child) {
					hoisted := b.hoistFloats(child)
					result = append(result, hoisted...)
				} else {
					el := b.buildElementNode(child)
					if el != nil {
						fc := floatChild{
							element: el,
							float:   childFloat,
							clear:   childClear,
						}
						result = append(result, fc)
					}
				}
			}
		}
	}

	// Flush remaining inline content.
	if para := ic.flush(); para != nil {
		result = append(result, floatChild{element: display.RichText(*para)})
	}

	return result
}

// hasFloatedDescendants checks if a DOM node has any direct children
// with float: left or float: right.
func (b *builder) hasFloatedDescendants(node *dom.Node) bool {
	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type == dom.ElementNode {
			childStyle := css.Resolve(child, b.sheets)
			if resolveFloat(childStyle) != FloatNone {
				return true
			}
		}
	}
	return false
}

// hoistFloats processes a non-BFC block element and returns its children
// as floatChild entries at the caller's level. Floated children are
// returned with their float metadata. Non-floated content is built as
// a normal-flow element wrapped in the parent's box style.
func (b *builder) hoistFloats(node *dom.Node) []floatChild {
	style := css.Resolve(node, b.sheets)
	fontSize := b.fontSize(node)

	var result []floatChild
	var normalFlowChildren []ui.Element

	ic := &inlineCollector{
		sheets:         b.sheets,
		onLink:         b.onLink,
		parentFontSize: fontSize,
		builder:        b,
	}

	flushNormal := func() {
		if para := ic.flush(); para != nil {
			normalFlowChildren = append(normalFlowChildren, display.RichText(*para))
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSib {
		switch child.Type {
		case dom.TextNode:
			text := child.Data
			if strings.TrimSpace(text) == "" && !ic.hasContent() {
				continue
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
			if tag == "style" || tag == "script" || tag == "head" || tag == "title" || tag == "meta" || tag == "link" || tag == "template" {
				continue
			}

			if isInlineDisplay(childDisplay) {
				ic.addInlineElement(child, childStyle)
				continue
			}

			flushNormal()

			childFloat := resolveFloat(childStyle)
			if childFloat != FloatNone {
				// Floated child — hoist to parent's level.
				el := b.buildElementNode(child)
				if el != nil {
					result = append(result, floatChild{
						element: el,
						float:   childFloat,
						clear:   resolveClear(childStyle),
					})
				}
			} else {
				// Non-floated child — stays as normal flow.
				el := b.buildElementNode(child)
				if el != nil {
					normalFlowChildren = append(normalFlowChildren, el)
				}
			}
		}
	}

	flushNormal()

	// Build the non-floated content as a normal-flow element with
	// the parent's box style (padding, margin, border, background).
	if len(normalFlowChildren) > 0 {
		var el ui.Element
		if len(normalFlowChildren) == 1 {
			el = normalFlowChildren[0]
		} else {
			el = layout.Column(normalFlowChildren...)
		}
		el = applyBoxStyle(el, style, fontSize)
		result = append(result, floatChild{element: el})
	} else if b.hasVisualBoxStyle(node) {
		// The parent has padding/border/background that needs rendering
		// even without non-floated content.
		el := applyBoxStyle(display.Text(""), style, fontSize)
		if _, ok := el.(StyledBox); ok {
			result = append(result, floatChild{element: el})
		}
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

	return applyBoxStyle(el, style, b.fontSize(node))
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
	return applyBoxStyle(flex, style, b.fontSize(node))
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
	return applyBoxStyle(el, style, b.fontSize(node))
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

	return applyBoxStyle(el, style, b.fontSize(node))
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
	ic := &inlineCollector{sheets: b.sheets, onLink: b.onLink, parentFontSize: b.fontSize(node), builder: b}
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
