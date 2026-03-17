// Package ui defines the Element types for the virtual tree.
// For M1 this is minimal — just the Element interface and a no-op element.
package ui

// Element is the base interface for all virtual tree nodes (RFC §4.3).
type Element interface {
	isElement()
}

// emptyElement is a no-op element for views that render nothing.
type emptyElement struct{}

func (emptyElement) isElement() {}

// Empty returns an Element that renders nothing.
// Used in M1 where the view produces no visible content.
func Empty() Element {
	return emptyElement{}
}

// Box is a layout container element (RFC §4.3).
// Stub for M1 — will be fleshed out in M2/M3.
type Box struct {
	Children []Element
}

func (Box) isElement() {}
