package ui

import "github.com/timzifer/lux/draw"

// ── Layout Cache (RFC-002 §4.9) ─────────────────────────────────
//
// The layout cache stores the result of a layout computation so that
// unchanged subtrees can skip re-layout. Currently used for
// CustomLayout elements; built-in layouts (Flex, Grid) are fast
// enough to re-compute each frame.

// LayoutCache stores cached layout results for a subtree.
type LayoutCache struct {
	constraints Constraints
	size        Size
	childRects  []draw.Rect
	valid       bool
}

// Invalidate marks the cache as dirty, forcing re-layout on next pass.
func (c *LayoutCache) Invalidate() {
	c.valid = false
}

// IsValid returns true if the cache holds a valid result for the given constraints.
func (c *LayoutCache) IsValid(constraints Constraints) bool {
	if !c.valid {
		return false
	}
	return c.constraints == constraints
}

// Store saves a layout result.
func (c *LayoutCache) Store(constraints Constraints, size Size, childRects []draw.Rect) {
	c.constraints = constraints
	c.size = size
	c.childRects = childRects
	c.valid = true
}

// CachedSize returns the cached size. Only valid if IsValid() returns true.
func (c *LayoutCache) CachedSize() Size {
	return c.size
}

// CachedChildRects returns the cached child positions.
func (c *LayoutCache) CachedChildRects() []draw.Rect {
	return c.childRects
}
