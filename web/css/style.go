// Package css provides CSS parsing, cascade resolution, and selector
// matching. It is designed to be reusable as the foundation for a
// browser engine (see RFC-998).
package css

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

// Set sets a property value.
func (d *StyleDeclaration) Set(prop, value string) {
	if d.Properties == nil {
		d.Properties = make(map[string]string)
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
