package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// ── Custom Layout Interface (RFC-002 §4.3) ──────────────────────

// Size represents the dimensions of a widget.
type Size struct {
	Width  float32
	Height float32
}

// Layout is an optional interface for custom layout algorithms.
// Types implementing Layout can define how children are measured and placed.
type Layout interface {
	// LayoutChildren computes the size and position of all children.
	// Use ctx.Measure to measure a child, ctx.Place to position it.
	// Returns the total size of the layout.
	LayoutChildren(ctx LayoutCtx, children []Element) Size
}

// LayoutCtx provides measurement and placement primitives to Layout implementations.
type LayoutCtx struct {
	// Constraints from the parent.
	Constraints Constraints

	// Measure measures a child under the given constraints.
	// Returns the child's desired size.
	Measure func(child Element, c Constraints) Size

	// Place positions a child relative to the layout's origin.
	// Must be called after Measure for each child.
	Place func(child Element, offset draw.Point)

	// Theme for layout-relevant tokens (spacing, etc.)
	Theme theme.Theme

	// Direction is the inline layout direction (LTR or RTL) propagated
	// from the application locale (RFC-002 §4.6).
	Direction draw.LayoutDirection
}

// CustomLayout creates an Element that uses a user-provided Layout
// to arrange its children.
func CustomLayout(layout Layout, children ...Element) Element {
	return customLayoutElement{
		Layout:   layout,
		Children: children,
	}
}

type customLayoutElement struct {
	Layout   Layout
	Children []Element
}

func (customLayoutElement) isElement() {}
