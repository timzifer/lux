package data

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

const virtualListOverscan = 3

// VirtualList displays a virtualized list that only renders visible items.
//
// When Dataset is set, it takes priority over ItemCount. The BuildItemDS
// callback receives a loaded flag so unloaded slots can show placeholders.
// If Dataset is nil, the legacy ItemCount/BuildItem API is used. (RFC-002 §6.2)
type VirtualList struct {
	ui.BaseElement

	// Dataset provides items with dynamic length support. Takes priority over ItemCount.
	Dataset Dataset[int]

	// BuildItemDS builds the element for a given index with a loaded flag.
	// Used when Dataset is set. loaded=false means the slot is not yet available.
	BuildItemDS func(index int, loaded bool) ui.Element

	// Legacy API — used when Dataset is nil.
	ItemCount int
	ItemHeight float32
	BuildItem  func(int) ui.Element
	MaxHeight  float32
	State      *ui.ScrollState
}

// NewVirtualList creates a VirtualList element from a VirtualListConfig.
// This uses the legacy API (ItemCount/BuildItem).
func NewVirtualList(config ui.VirtualListConfig) ui.Element {
	return VirtualList{
		ItemCount:  config.ItemCount,
		ItemHeight: config.ItemHeight,
		BuildItem:  config.BuildItem,
		MaxHeight:  config.MaxHeight,
		State:      config.State,
	}
}

// resolvedItemCount returns the effective item count, considering Dataset.
func (n VirtualList) resolvedItemCount() int {
	if n.Dataset != nil {
		l := n.Dataset.Len()
		if l >= 0 {
			return l
		}
		// Unknown length: use currently known items.
		// StreamDataset implements Count(); PagedDataset uses TotalCount.
		// For truly unknown lengths, return a large estimate to enable overscan loading.
		type counter interface{ Count() int }
		if c, ok := n.Dataset.(counter); ok {
			return c.Count()
		}
		return 0
	}
	return n.ItemCount
}

// resolvedBuildItem returns a build function that always passes loaded status.
func (n VirtualList) resolvedBuildItem() func(int, bool) ui.Element {
	if n.Dataset != nil && n.BuildItemDS != nil {
		return n.BuildItemDS
	}
	if n.BuildItem != nil {
		fn := n.BuildItem
		return func(i int, _ bool) ui.Element { return fn(i) }
	}
	return nil
}

// LayoutSelf implements ui.Layouter.
func (n VirtualList) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	itemCount := n.resolvedItemCount()
	buildItem := n.resolvedBuildItem()

	if itemCount <= 0 || buildItem == nil {
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

	contentH := float32(itemCount * itemH)

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
	if lastVisible >= itemCount {
		lastVisible = itemCount - 1
	}

	// Clip to viewport (including scrollbar space).
	ctx.Canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)))

	// Track which pages need loading (for Dataset mode).
	var loadPages map[int]bool

	// Render visible items.
	for i := firstVisible; i <= lastVisible; i++ {
		loaded := true
		if n.Dataset != nil {
			_, loaded = n.Dataset.Get(i)
			// Track unloaded pages for load requests.
			if !loaded && loadPages == nil {
				loadPages = make(map[int]bool)
			}
			if !loaded {
				if pd, ok := n.Dataset.(*PagedDataset[int]); ok {
					pg := pd.PageForIndex(i)
					if !pd.IsPageLoading(pg) && pd.PageState(pg) != SlotLoaded {
						loadPages[pg] = true
					}
				}
			}
		}

		itemY := area.Y + i*itemH - int(offset)
		child := buildItem(i, loaded)
		childArea := ui.Bounds{X: area.X, Y: itemY, W: contentW, H: itemH}
		ctx.LayoutChild(child, childArea)
	}

	// Send load requests for unloaded pages (RFC-002 §6.5).
	if n.Dataset != nil && len(loadPages) > 0 {
		if pd, ok := n.Dataset.(*PagedDataset[int]); ok {
			for pg := range loadPages {
				start := pg * pd.PageSize
				end := start + pd.PageSize - 1
				if n.Dataset.Len() >= 0 && end >= n.Dataset.Len() {
					end = n.Dataset.Len() - 1
				}
				app.Send(DatasetLoadRequestMsg{
					PageIndex:  pg,
					StartIndex: start,
					EndIndex:   end,
				})
			}
		}
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
	// When using Dataset, compare by dataset identity and item height/max height.
	if n.Dataset != nil || o.Dataset != nil {
		return n.Dataset == o.Dataset && n.ItemHeight == o.ItemHeight && n.MaxHeight == o.MaxHeight
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
	itemCount := n.resolvedItemCount()
	buildItem := n.resolvedBuildItem()
	if buildItem != nil {
		for i := 0; i < itemCount; i++ {
			loaded := true
			if n.Dataset != nil {
				_, loaded = n.Dataset.Get(i)
			}
			item := buildItem(i, loaded)
			b.Walk(item, int32(listIdx))
		}
	}
}
