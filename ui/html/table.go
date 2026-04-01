package html

import (
	"strconv"
	"strings"

	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// buildTable converts a <table> DOM node into a layout.Table element.
func (b *builder) buildTable(node *dom.Node, style css.StyleDeclaration) ui.Element {
	var children []ui.Element

	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type != dom.ElementNode {
			continue
		}
		tag := strings.ToLower(child.Tag)
		switch tag {
		case "caption":
			children = append(children, b.buildCaption(child))
		case "colgroup":
			children = append(children, b.buildColGroup(child))
		case "col":
			children = append(children, b.buildCol(child))
		case "thead":
			children = append(children, b.buildSection(child, layout.SectionHead))
		case "tbody":
			children = append(children, b.buildSection(child, layout.SectionBody))
		case "tfoot":
			children = append(children, b.buildSection(child, layout.SectionFoot))
		case "tr":
			// Bare <tr> outside of a section — treat as body row.
			children = append(children, b.buildRow(child))
		}
	}

	var opts []layout.TableOption
	if v := style.Get("border-collapse"); v == "collapse" {
		opts = append(opts, layout.WithBorderCollapse(layout.BorderCollapsed))
	}
	if v := style.Get("border-spacing"); v != "" {
		if d, ok := css.ParseDimension(v); ok {
			opts = append(opts, layout.WithBorderSpacing(d, d))
		}
	}

	return layout.NewTable(children, opts...)
}

// buildCaption converts a <caption> element.
func (b *builder) buildCaption(node *dom.Node) ui.Element {
	content := b.buildBlockContent(node)
	return layout.NewTableCaption(content)
}

// buildColGroup converts a <colgroup> element.
func (b *builder) buildColGroup(node *dom.Node) ui.Element {
	var cols []layout.TableCol
	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type == dom.ElementNode && strings.ToLower(child.Tag) == "col" {
			cols = append(cols, b.buildColDef(child))
		}
	}
	return layout.NewTableColGroup(cols...)
}

// buildCol converts a standalone <col> element.
func (b *builder) buildCol(node *dom.Node) ui.Element {
	def := b.buildColDef(node)
	return layout.NewTableColGroup(def)
}

// buildColDef converts a <col> node to a TableCol definition.
func (b *builder) buildColDef(node *dom.Node) layout.TableCol {
	span := 1
	if s := node.Attr("span"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			span = v
		}
	}

	width := layout.AutoTrack()
	if w := node.Attr("width"); w != "" {
		if d, ok := css.ParseDimension(w); ok {
			width = layout.Px(d)
		}
	}

	return layout.Col(width, span)
}

// buildSection converts a <thead>, <tbody>, or <tfoot> element.
func (b *builder) buildSection(node *dom.Node, sectionType layout.SectionType) ui.Element {
	var rows []ui.Element
	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type == dom.ElementNode && strings.ToLower(child.Tag) == "tr" {
			rows = append(rows, b.buildRow(child))
		}
	}
	return layout.NewTableSection(sectionType, rows...)
}

// buildRow converts a <tr> element.
func (b *builder) buildRow(node *dom.Node) ui.Element {
	var cells []ui.Element
	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type != dom.ElementNode {
			continue
		}
		tag := strings.ToLower(child.Tag)
		if tag == "td" || tag == "th" {
			cells = append(cells, b.buildCell(child, tag == "th"))
		}
	}
	return layout.TR(cells...)
}

// buildCell converts a <td> or <th> element.
func (b *builder) buildCell(node *dom.Node, isHead bool) ui.Element {
	content := b.buildBlockContent(node)

	var opts []layout.CellOption
	if isHead {
		opts = append(opts, func(c *layout.TableCell) { c.IsHead = true })
	}

	if s := node.Attr("colspan"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 1 {
			opts = append(opts, layout.WithColSpan(v))
		}
	}
	if s := node.Attr("rowspan"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 1 {
			opts = append(opts, layout.WithRowSpan(v))
		}
	}

	// Vertical alignment.
	style := css.Resolve(node, b.sheets)
	if v := style.Get("vertical-align"); v != "" {
		switch v {
		case "top":
			opts = append(opts, layout.WithVAlign(layout.VAlignTop))
		case "middle":
			opts = append(opts, layout.WithVAlign(layout.VAlignMiddle))
		case "bottom":
			opts = append(opts, layout.WithVAlign(layout.VAlignBottom))
		case "baseline":
			opts = append(opts, layout.WithVAlign(layout.VAlignBaseline))
		}
	}

	if isHead {
		return layout.TH(content, opts...)
	}
	return layout.TD(content, opts...)
}

// buildBlockContent builds the children of a node as a single block
// element. If there is only one child element, returns it directly.
// Otherwise wraps in a Column.
func (b *builder) buildBlockContent(node *dom.Node) ui.Element {
	children := b.buildChildren(node)
	switch len(children) {
	case 0:
		// Empty element — return empty text.
		return display.Text("")
	case 1:
		return children[0]
	default:
		return layout.Column(children...)
	}
}
