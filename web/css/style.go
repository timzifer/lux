// Package css provides CSS parsing, cascade resolution, and selector
// matching. It is designed to be reusable as the foundation for a
// browser engine (see RFC-998).
package css

import "strings"

// StyleDeclaration is an ordered set of CSS property declarations.
type StyleDeclaration struct {
	Properties map[string]string // e.g. "font-weight" → "bold"
}

// NewDecl creates an empty StyleDeclaration.
func NewDecl() StyleDeclaration {
	return StyleDeclaration{Properties: make(map[string]string)}
}

// Get returns the value of a property, or "" if not set.
func (d StyleDeclaration) Get(prop string) string {
	if d.Properties == nil {
		return ""
	}
	return d.Properties[prop]
}

// Set sets a property value. Shorthand properties (e.g. "font") are
// automatically expanded into their longhand components.
func (d *StyleDeclaration) Set(prop, value string) {
	if d.Properties == nil {
		d.Properties = make(map[string]string)
	}
	// Expand shorthand properties.
	if prop == "font" {
		ExpandFontShorthand(value, d)
		return
	}
	if prop == "border" {
		expandBorderShorthand(value, d)
		return
	}
	d.Properties[prop] = value
}

// Merge copies all properties from other into d.
// Existing properties in d are overwritten.
func (d *StyleDeclaration) Merge(other StyleDeclaration) {
	if other.Properties == nil {
		return
	}
	if d.Properties == nil {
		d.Properties = make(map[string]string, len(other.Properties))
	}
	for k, v := range other.Properties {
		d.Properties[k] = v
	}
}

// Clone returns a deep copy of the declaration.
func (d StyleDeclaration) Clone() StyleDeclaration {
	if d.Properties == nil {
		return NewDecl()
	}
	out := StyleDeclaration{Properties: make(map[string]string, len(d.Properties))}
	for k, v := range d.Properties {
		out.Properties[k] = v
	}
	return out
}

// expandBorderShorthand expands a "border" shorthand value into individual
// longhand properties (border-width, border-style, border-color).
func expandBorderShorthand(value string, d *StyleDeclaration) {
	parts := strings.Fields(value)
	// Default values for reset.
	width := ""
	style := ""
	color := ""

	for _, part := range parts {
		// Check if it's a style keyword.
		switch strings.ToLower(part) {
		case "none", "hidden":
			style = "none"
			continue
		case "solid", "dashed", "dotted", "double", "groove", "ridge", "inset", "outset":
			style = part
			continue
		}
		// Try as color.
		if _, ok := ParseColor(part); ok {
			color = part
			continue
		}
		// Assume it's a width.
		width = part
	}

	if width != "" {
		d.Properties["border-width"] = width
	} else {
		// border: 0 or border: none implies zero width.
		d.Properties["border-width"] = "0"
	}
	if style != "" {
		d.Properties["border-style"] = style
	} else if width != "" {
		// If a width is specified without style, default to solid.
		d.Properties["border-style"] = "solid"
	} else {
		d.Properties["border-style"] = "none"
	}
	if color != "" {
		d.Properties["border-color"] = color
	}
}

// StyleRule is a CSS ruleset (selector + declarations).
type StyleRule struct {
	Selector    string
	Specificity [3]int // [A, B, C] per CSS spec
	Decl        StyleDeclaration
}

// StyleSheet is an ordered list of style rules.
type StyleSheet struct {
	Rules []StyleRule
}
