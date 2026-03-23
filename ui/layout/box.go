// Package layout provides layout element types (Box, Stack, Padding, Flex, Grid, etc.)
// for the Lux UI framework.
package layout

import (
	"github.com/timzifer/lux/ui"
)

// Box arranges children along a single axis (column or row).
type Box struct {
	ui.BaseElement
	Axis     ui.LayoutAxis
	Children []ui.Element
}

// Column creates a Box that stacks children vertically.
func Column(children ...ui.Element) ui.Element {
	return Box{Axis: ui.AxisColumn, Children: children}
}

// Row creates a Box that stacks children horizontally.
func Row(children ...ui.Element) ui.Element {
	return Box{Axis: ui.AxisRow, Children: children}
}

// LayoutSelf implements ui.Layouter.
func (n Box) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if n.Axis == ui.AxisRow {
		return n.layoutRow(ctx)
	}
	return n.layoutColumn(ctx)
}

func (n Box) layoutColumn(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	cursorY := area.Y
	maxW := 0
	maxH := 0
	count := 0
	firstBaseline := 0

	for _, child := range n.Children {
		childBounds := ctx.LayoutChild(child, ui.Bounds{X: area.X, Y: cursorY, W: area.W, H: area.H})
		if childBounds.W == 0 && childBounds.H == 0 {
			continue
		}
		if count == 0 {
			firstBaseline = childBounds.Baseline
		}
		count++
		cursorY += childBounds.H + ui.ColumnGap
		if childBounds.W > maxW {
			maxW = childBounds.W
		}
		if h := cursorY - area.Y - ui.ColumnGap; h > maxH {
			maxH = h
		}
	}

	if count == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}
	if firstBaseline == 0 {
		firstBaseline = maxH
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: firstBaseline}
}

func (n Box) layoutRow(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	nn := len(n.Children)
	if nn == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Pass 1: measure with NullCanvas (MeasureChild).
	type childInfo struct {
		w, h, baseline int
	}
	infos := make([]childInfo, nn)
	cursorX := area.X
	maxH := 0
	hasContent := false

	for i, child := range n.Children {
		childW := area.X + area.W - cursorX
		if childW < 0 {
			childW = 0
		}
		cb := ctx.MeasureChild(child, ui.Bounds{X: cursorX, Y: area.Y, W: childW, H: area.H})
		if cb.W == 0 && cb.H == 0 {
			continue
		}
		infos[i] = childInfo{w: cb.W, h: cb.H, baseline: cb.Baseline}
		if cb.H > maxH {
			maxH = cb.H
		}
		cursorX += cb.W + ui.RowGap
		hasContent = true
	}

	if !hasContent {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Pass 2: render with vertical centering.
	cursorX = area.X
	maxW := 0

	for i, child := range n.Children {
		info := infos[i]
		if info.w == 0 && info.h == 0 {
			continue
		}
		yOffset := (maxH - info.h) / 2
		childW := area.X + area.W - cursorX
		if childW < 0 {
			childW = 0
		}
		ctx.LayoutChild(child, ui.Bounds{X: cursorX, Y: area.Y + yOffset, W: childW, H: area.H})
		cursorX += info.w + ui.RowGap
		if w := cursorX - area.X - ui.RowGap; w > maxW {
			maxW = w
		}
	}

	// Baseline: use the tallest child's baseline + its centering offset.
	baseline := maxH
	for _, info := range infos {
		if info.h > 0 && info.baseline > 0 {
			bl := (maxH-info.h)/2 + info.baseline
			if bl > 0 {
				baseline = bl
				break
			}
		}
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: maxH, Baseline: baseline}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Box) TreeEqual(other ui.Element) bool {
	o, ok := other.(Box)
	return ok && n.Axis == o.Axis && len(n.Children) == len(o.Children)
}

// ResolveChildren implements ui.ChildResolver.
func (n Box) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Box) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}
