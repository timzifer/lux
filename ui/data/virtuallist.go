package data

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

const virtualListOverscan = 3

// VirtualList displays a virtualized list that only renders visible items.
type VirtualList struct {
	ui.BaseElement
	ItemCount  int
	ItemHeight float32
	BuildItem  func(int) ui.Element
	MaxHeight  float32
	State      *ui.ScrollState
}

// NewVirtualList creates a VirtualList element from a VirtualListConfig.
func NewVirtualList(config ui.VirtualListConfig) ui.Element {
	return VirtualList{
		ItemCount:  config.ItemCount,
		ItemHeight: config.ItemHeight,
		BuildItem:  config.BuildItem,
		MaxHeight:  config.MaxHeight,
		State:      config.State,
	}
}

// LayoutSelf implements ui.Layouter.
func (n VirtualList) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if n.ItemCount <= 0 || n.BuildItem == nil {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	itemH := int(n.ItemHeight)
	if itemH <= 0 {
		itemH = 24
	}

	viewportH := int(n.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	contentH := float32(n.ItemCount * itemH)

	// The list grows to fit its content, capped at viewportH.
	// Only scroll when content exceeds the viewport.
	needsScroll := contentH > float32(viewportH)
	actualH := viewportH
	if !needsScroll {
		actualH = int(contentH)
		if actualH <= 0 {
			actualH = itemH
		}
	}

	// Determine scrollbar width so we can reserve space inside the clip.
	scrollbarW := 0
	if needsScroll {
		scrollbarW = int(ctx.Tokens.Scroll.TrackWidth)
		if scrollbarW <= 0 {
			scrollbarW = 8
		}
	}

	// Content width excluding the scrollbar.
	contentW := area.W - scrollbarW

	var offset float32
	if n.State != nil {
		offset = n.State.Offset
	}

	// Determine visible range.
	firstVisible := int(offset) / itemH
	if firstVisible < 0 {
		firstVisible = 0
	}
	firstVisible -= virtualListOverscan
	if firstVisible < 0 {
		firstVisible = 0
	}

	lastVisible := (int(offset) + actualH) / itemH
	lastVisible += virtualListOverscan
	if lastVisible >= n.ItemCount {
		lastVisible = n.ItemCount - 1
	}

	// Clip to viewport (including scrollbar space).
	ctx.Canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)))

	// Render visible items.
	for i := firstVisible; i <= lastVisible; i++ {
		itemY := area.Y + i*itemH - int(offset)
		child := n.BuildItem(i)
		childArea := ui.Bounds{X: area.X, Y: itemY, W: contentW, H: itemH}
		ctx.LayoutChild(child, childArea)
	}

	// Draw scrollbar INSIDE the clip so it's visible even within a parent ScrollView.
	if needsScroll && n.State != nil {
		ui.DrawScrollbar(ctx.Canvas, ctx.Tokens, ctx.IX, n.State, area.X+contentW, area.Y, actualH, contentH, offset)
	}

	ctx.Canvas.PopClip()

	// Clamp scroll state.
	if n.State != nil {
		maxScroll := contentH - float32(actualH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if n.State.Offset > maxScroll {
			n.State.Offset = maxScroll
		}
		if n.State.Offset < 0 {
			n.State.Offset = 0
		}
	}

	// Register scroll target.
	if n.State != nil && needsScroll {
		state := n.State
		cH := contentH
		vH := float32(actualH)
		ctx.IX.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: actualH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n VirtualList) TreeEqual(other ui.Element) bool {
	o, ok := other.(VirtualList)
	if !ok {
		return false
	}
	return n.ItemCount == o.ItemCount && n.ItemHeight == o.ItemHeight && n.MaxHeight == o.MaxHeight
}

// ResolveChildren implements ui.ChildResolver. VirtualList is a leaf in resolution
// (children are built dynamically via BuildItem).
func (n VirtualList) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. Builds a11y tree nodes for VirtualList.
func (n VirtualList) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	listIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleListbox, Label: "List"}, parentIdx, a11y.Rect{})
	if n.BuildItem != nil {
		for i := 0; i < n.ItemCount; i++ {
			item := n.BuildItem(i)
			b.Walk(item, int32(listIdx))
		}
	}
}
