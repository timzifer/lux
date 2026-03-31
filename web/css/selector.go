package css

import (
	"sort"

	"github.com/andybalholm/cascadia"
	xhtml "golang.org/x/net/html"

	"github.com/timzifer/lux/web/dom"
)

// MatchingRules returns all rules from sheet that match the given node,
// sorted by specificity (ascending — last wins in the cascade).
func MatchingRules(node *dom.Node, sheet *StyleSheet) []StyleRule {
	if sheet == nil || node.Type != dom.ElementNode {
		return nil
	}

	// Build an html.Node tree for cascadia matching and find the target.
	root := findRoot(node)
	hn := dom.ToHTMLNode(root)
	target := findHTMLTarget(hn, root, node)
	if target == nil {
		return nil
	}

	var matched []StyleRule
	for _, rule := range sheet.Rules {
		// Use ParseGroup to support comma-separated selectors (e.g. "b, strong").
		group, err := cascadia.ParseGroup(rule.Selector)
		if err != nil {
			continue
		}
		for _, sel := range group {
			if sel.Match(target) {
				r := rule
				r.Specificity = specificityFromCascadia(sel)
				matched = append(matched, r)
				break // one match per rule is enough
			}
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		return compareSpecificity(matched[i].Specificity, matched[j].Specificity) < 0
	})
	return matched
}

func findRoot(n *dom.Node) *dom.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}

// findHTMLTarget walks both trees in parallel to find the *html.Node
// corresponding to the given *dom.Node target.
func findHTMLTarget(hn *xhtml.Node, dn *dom.Node, target *dom.Node) *xhtml.Node {
	if dn == target {
		return hn
	}
	hc := hn.FirstChild
	dc := dn.FirstChild
	for hc != nil && dc != nil {
		if result := findHTMLTarget(hc, dc, target); result != nil {
			return result
		}
		hc = hc.NextSibling
		dc = dc.NextSib
	}
	return nil
}

func specificityFromCascadia(sel cascadia.Sel) [3]int {
	s := sel.Specificity()
	return [3]int{s[0], s[1], s[2]}
}

func compareSpecificity(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return 0
}
