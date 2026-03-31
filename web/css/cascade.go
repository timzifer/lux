package css

import (
	"github.com/timzifer/lux/web/dom"
)

// inheritableProperties lists CSS properties that inherit from parent elements.
var inheritableProperties = map[string]bool{
	"color":           true,
	"font-family":     true,
	"font-size":       true,
	"font-weight":     true,
	"font-style":      true,
	"letter-spacing":  true,
	"line-height":     true,
	"white-space":     true,
	"text-align":      true,
	"list-style-type": true,
}

// uaStyleSheet is the built-in user-agent stylesheet.
var uaStyleSheet *StyleSheet

func init() {
	uaStyleSheet = &StyleSheet{
		Rules: []StyleRule{
			{Selector: "b, strong", Decl: declFrom("font-weight", "bold")},
			{Selector: "i, em", Decl: declFrom("font-style", "italic")},
			{Selector: "u", Decl: declFrom("text-decoration", "underline")},
			{Selector: "s, strike, del", Decl: declFrom("text-decoration", "line-through")},
			{Selector: "pre", Decl: declFrom2("white-space", "pre", "font-family", "monospace")},
			{Selector: "code", Decl: declFrom("font-family", "monospace")},
			{Selector: "h1", Decl: declFrom2("font-size", "2em", "font-weight", "bold")},
			{Selector: "h2", Decl: declFrom2("font-size", "1.5em", "font-weight", "bold")},
			{Selector: "h3", Decl: declFrom2("font-size", "1.17em", "font-weight", "bold")},
			{Selector: "h4", Decl: declFrom("font-weight", "bold")},
			{Selector: "h5", Decl: declFrom("font-weight", "bold")},
			{Selector: "h6", Decl: declFrom("font-weight", "bold")},
			{Selector: "ul", Decl: declFrom("list-style-type", "disc")},
			{Selector: "ol", Decl: declFrom("list-style-type", "decimal")},
		},
	}
}

func declFrom(k, v string) StyleDeclaration {
	d := NewDecl()
	d.Set(k, v)
	return d
}

func declFrom2(k1, v1, k2, v2 string) StyleDeclaration {
	d := NewDecl()
	d.Set(k1, v1)
	d.Set(k2, v2)
	return d
}

// Resolve computes the final style for a DOM node by applying the cascade:
// 1. UA defaults (user-agent stylesheet)
// 2. Author stylesheet rules (sorted by specificity)
// 3. Inline style attribute
// 4. Inheritance from parent for inheritable properties
func Resolve(node *dom.Node, sheets []*StyleSheet) StyleDeclaration {
	if node.Type != dom.ElementNode {
		// For text nodes, inherit from parent.
		if node.Parent != nil {
			return Resolve(node.Parent, sheets)
		}
		return NewDecl()
	}

	result := NewDecl()

	// 1. Inherit from parent.
	if node.Parent != nil && node.Parent.Type == dom.ElementNode {
		parentStyle := Resolve(node.Parent, sheets)
		for prop, val := range parentStyle.Properties {
			if inheritableProperties[prop] {
				result.Set(prop, val)
			}
		}
	}

	// 2. UA stylesheet.
	for _, rule := range MatchingRules(node, uaStyleSheet) {
		result.Merge(rule.Decl)
	}

	// 3. Author stylesheets.
	for _, sheet := range sheets {
		for _, rule := range MatchingRules(node, sheet) {
			result.Merge(rule.Decl)
		}
	}

	// 4. Inline style (highest specificity).
	if style := node.Attr("style"); style != "" {
		inline := ParseStyleAttribute(style)
		result.Merge(inline)
	}

	return result
}
