package dom

import (
	"strings"

	"github.com/andybalholm/cascadia"
)

// Walk calls fn for every node in the subtree rooted at n (pre-order DFS).
// If fn returns false the subtree of that node is skipped.
func Walk(n *Node, fn func(*Node) bool) {
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSib {
		Walk(c, fn)
	}
}

// QuerySelector returns the first descendant element matching the CSS
// selector, or nil if none match.
func (n *Node) QuerySelector(sel string) *Node {
	compiled, err := cascadia.Parse(sel)
	if err != nil {
		return nil
	}
	hn := ToHTMLNode(n)
	match := cascadia.Query(hn, compiled)
	if match == nil {
		return nil
	}
	return HTMLNodeLookup(hn, n, match)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (n *Node) QuerySelectorAll(sel string) []*Node {
	compiled, err := cascadia.Parse(sel)
	if err != nil {
		return nil
	}
	hn := ToHTMLNode(n)
	matches := cascadia.QueryAll(hn, compiled)
	out := make([]*Node, 0, len(matches))
	for _, m := range matches {
		if dn := HTMLNodeLookup(hn, n, m); dn != nil {
			out = append(out, dn)
		}
	}
	return out
}

// GetElementsByTagName returns all descendant elements with the given tag.
func (n *Node) GetElementsByTagName(tag string) []*Node {
	tag = strings.ToLower(tag)
	var out []*Node
	Walk(n, func(node *Node) bool {
		if node != n && node.Type == ElementNode && node.Tag == tag {
			out = append(out, node)
		}
		return true
	})
	return out
}
