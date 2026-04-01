package dom

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ParseHTML parses an HTML string and returns a DOM tree.
// If the input contains a full document (<!DOCTYPE, <html>, etc.), it is
// parsed as a complete document preserving <html>, <head>, and <body>
// elements. Otherwise it is parsed as a fragment inside <body>.
// The function always returns a DocumentNode root.
func ParseHTML(input string) (*Node, error) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
	isFullDoc := strings.HasPrefix(lower, "<!doctype") ||
		strings.HasPrefix(lower, "<html")

	if isFullDoc {
		return parseFullDocument(input)
	}
	return parseFragment(input)
}

// parseFullDocument parses a complete HTML document, preserving
// <html>, <head>, and <body> elements in the DOM tree.
func parseFullDocument(input string) (*Node, error) {
	root, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	return fromHTMLNode(root), nil
}

// parseFragment parses an HTML fragment inside a <body> context.
func parseFragment(input string) (*Node, error) {
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(input), ctx)
	if err != nil {
		return nil, err
	}
	doc := NewDocument()
	for _, hn := range nodes {
		doc.AppendChild(fromHTMLNode(hn))
	}
	return doc, nil
}

// fromHTMLNode recursively converts an x/net/html node tree to our DOM model.
func fromHTMLNode(hn *html.Node) *Node {
	n := &Node{}
	switch hn.Type {
	case html.ElementNode:
		n.Type = ElementNode
		n.Tag = hn.Data
		if len(hn.Attr) > 0 {
			n.Attrs = make(map[string]string, len(hn.Attr))
			for _, a := range hn.Attr {
				n.Attrs[a.Key] = a.Val
			}
		}
	case html.TextNode:
		n.Type = TextNode
		n.Data = hn.Data
	case html.CommentNode:
		n.Type = CommentNode
		n.Data = hn.Data
	case html.DocumentNode:
		n.Type = DocumentNode
	default:
		n.Type = TextNode
		n.Data = hn.Data
	}
	for c := hn.FirstChild; c != nil; c = c.NextSibling {
		n.AppendChild(fromHTMLNode(c))
	}
	return n
}

// ToHTMLNode recursively converts our DOM node to an x/net/html node.
// This is needed for cascadia selector matching.
func ToHTMLNode(n *Node) *html.Node {
	hn := &html.Node{}
	switch n.Type {
	case DocumentNode:
		hn.Type = html.DocumentNode
	case ElementNode:
		hn.Type = html.ElementNode
		hn.Data = n.Tag
		hn.DataAtom = atom.Lookup([]byte(n.Tag))
		for k, v := range n.Attrs {
			hn.Attr = append(hn.Attr, html.Attribute{Key: k, Val: v})
		}
	case TextNode:
		hn.Type = html.TextNode
		hn.Data = n.Data
	case CommentNode:
		hn.Type = html.CommentNode
		hn.Data = n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSib {
		hc := ToHTMLNode(c)
		hn.AppendChild(hc)
	}
	return hn
}

// HTMLNodeLookup finds the DOM node corresponding to an html.Node match
// by walking both trees in parallel (pre-order DFS).
func HTMLNodeLookup(hn *html.Node, dn *Node, target *html.Node) *Node {
	if hn == target {
		return dn
	}
	hc := hn.FirstChild
	dc := dn.FirstChild
	for hc != nil && dc != nil {
		if result := HTMLNodeLookup(hc, dc, target); result != nil {
			return result
		}
		hc = hc.NextSibling
		dc = dc.NextSib
	}
	return nil
}
