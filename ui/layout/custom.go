package layout

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// CustomLayout creates an Element that uses a user-provided ui.Layout
// to arrange its children.
type CustomLayout struct {
	ui.BaseElement
	Layout   ui.Layout
	Children []ui.Element
}

// NewCustomLayout creates a CustomLayout element.
func NewCustomLayout(layout ui.Layout, children ...ui.Element) ui.Element {
	return CustomLayout{
		Layout:   layout,
		Children: children,
	}
}

// LayoutSelf implements ui.Layouter.
func (n CustomLayout) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if n.Layout == nil || len(n.Children) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Track placements: child index -> offset.
	type placement struct {
		offset draw.Point
		placed bool
	}
	placements := make([]placement, len(n.Children))

	// Build an identity map for fast child->index lookup.
	// We match by index in the children slice.
	childIndex := make(map[ui.Element]int, len(n.Children))
	for i, ch := range n.Children {
		childIndex[ch] = i
	}

	// Measure callback: layout children with MeasureChild to get their size.
	measureFn := func(child ui.Element, c ui.Constraints) ui.Size {
		measureArea := ui.Bounds{X: 0, Y: 0, W: int(c.MaxWidth), H: int(c.MaxHeight)}
		cb := ctx.MeasureChild(child, measureArea)
		return ui.Size{Width: float32(cb.W), Height: float32(cb.H)}
	}

	// Place callback: record offset for later painting.
	placeFn := func(child ui.Element, offset draw.Point) {
		if idx, ok := childIndex[child]; ok {
			placements[idx] = placement{offset: offset, placed: true}
		}
	}

	lctx := ui.LayoutCtx{
		Constraints: ui.Constraints{
			MaxWidth:  float32(area.W),
			MaxHeight: float32(area.H),
		},
		Measure: measureFn,
		Place:   placeFn,
		Theme:   ctx.Theme,
	}

	size := n.Layout.LayoutChildren(lctx, n.Children)

	// Paint pass: draw each placed child at its offset.
	maxW, maxH := 0, 0
	for i, child := range n.Children {
		p := placements[i]
		if !p.placed {
			continue
		}
		childArea := ui.Bounds{
			X: area.X + int(p.offset.X),
			Y: area.Y + int(p.offset.Y),
			W: area.W,
			H: area.H,
		}
		cb := ctx.LayoutChild(child, childArea)
		endX := int(p.offset.X) + cb.W
		endY := int(p.offset.Y) + cb.H
		if endX > maxW {
			maxW = endX
		}
		if endY > maxH {
			maxH = endY
		}
	}

	w := int(size.Width)
	h := int(size.Height)
	if w == 0 {
		w = maxW
	}
	if h == 0 {
		h = maxH
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n CustomLayout) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n CustomLayout) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n CustomLayout) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}
