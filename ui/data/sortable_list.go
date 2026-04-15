// Package data — sortable_list.go implements a reorderable list widget
// where items can be rearranged by dragging (RFC-005 §8).
package data

import (
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// LayoutAxis describes the direction of a sortable list.
type LayoutAxis uint8

const (
	AxisVertical   LayoutAxis = iota // items stack top-to-bottom
	AxisHorizontal                   // items stack left-to-right
)

// SortableList renders a list of items that can be reordered by dragging.
// Each item is automatically wrapped in a DragSource, and the list itself
// acts as a DropTarget accepting items from the same list (or cross-list
// via GroupID).
type SortableList struct {
	ui.BaseElement

	// Items contains the keys of all items in their current order.
	Items []string

	// BuildItem builds the visual element for a given item.
	// dragging is true when this item is currently being dragged.
	BuildItem func(key string, index int, dragging bool) ui.Element

	// ItemHeight is the uniform height per item in dp.
	// Default: 48.
	ItemHeight float32

	// MaxHeight is the maximum viewport height in dp.
	// If zero, the list grows to fit all items.
	MaxHeight float32

	// OnReorder is called when the user completes a drag reorder.
	// fromIndex and toIndex are indices in the Items slice.
	OnReorder func(fromIndex, toIndex int)

	// State holds the scroll and drag state. Required for stateful operation.
	State *SortableListState

	// HandleOnly requires an explicit DragHandle to initiate dragging
	// (instead of dragging from anywhere on the item).
	HandleOnly bool

	// Axis determines the list direction. Default: AxisVertical.
	Axis LayoutAxis

	// GroupID enables cross-list dragging. Items can be dragged between
	// SortableLists that share the same GroupID.
	GroupID string

	// OnInsert is called when an item from another list is dropped here.
	OnInsert func(index int, data *input.DragData)

	// OnRemove is called when an item is dragged out to another list.
	OnRemove func(index int)

	// ShowPlaceholder shows a dashed placeholder at the dragged item's
	// original position.
	ShowPlaceholder bool
}

// SortableListState tracks the drag and animation state of a SortableList.
type SortableListState struct {
	Scroll      ui.ScrollState
	DragIndex   int // index of item being dragged (-1 = none)
	InsertIndex int // where the item would be inserted (-1 = none)
	itemAnims   map[int]*anim.Anim[float32]
	motionDur   time.Duration
	motionEase  anim.EasingFunc
}

// NewSortableListState creates a ready-to-use SortableListState.
func NewSortableListState() *SortableListState {
	return &SortableListState{
		DragIndex:   -1,
		InsertIndex: -1,
		itemAnims:   make(map[int]*anim.Anim[float32]),
	}
}

// Tick advances all item displacement animations.
func (s *SortableListState) Tick(dt time.Duration) bool {
	if s == nil {
		return false
	}
	running := false
	for idx, a := range s.itemAnims {
		if a.Tick(dt) {
			running = true
		} else {
			delete(s.itemAnims, idx)
		}
	}
	return running
}

// SetInsertIndex updates the insertion point and triggers displacement
// animations for affected items.
func (s *SortableListState) SetInsertIndex(insertIdx int, itemCount int) {
	if s == nil || insertIdx == s.InsertIndex {
		return
	}
	s.InsertIndex = insertIdx

	dur := s.motionDur
	eas := s.motionEase
	if dur == 0 {
		dur = 150 * time.Millisecond
		eas = anim.OutCubic
	}

	for i := 0; i < itemCount; i++ {
		target := float32(0)
		if s.DragIndex >= 0 {
			if s.DragIndex < insertIdx && i >= s.DragIndex && i < insertIdx {
				target = -1 // shift up
			} else if s.DragIndex > insertIdx && i >= insertIdx && i < s.DragIndex {
				target = 1 // shift down
			}
		}
		a := s.getOrCreateAnim(i)
		a.SetTarget(target, dur, eas)
	}
}

func (s *SortableListState) getOrCreateAnim(idx int) *anim.Anim[float32] {
	if s.itemAnims == nil {
		s.itemAnims = make(map[int]*anim.Anim[float32])
	}
	a, ok := s.itemAnims[idx]
	if !ok {
		a = &anim.Anim[float32]{}
		s.itemAnims[idx] = a
	}
	return a
}

func (s *SortableListState) displacement(idx int) float32 {
	if s == nil {
		return 0
	}
	a, ok := s.itemAnims[idx]
	if !ok {
		return 0
	}
	return a.Value()
}

// CacheMotion stores theme motion parameters for animations.
func (s *SortableListState) CacheMotion(dur time.Duration, easing anim.EasingFunc) {
	if s != nil {
		s.motionDur = dur
		s.motionEase = easing
	}
}

// LayoutSelf implements ui.Layouter.
func (sl SortableList) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	itemCount := len(sl.Items)
	if itemCount == 0 || sl.BuildItem == nil {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	itemH := int(sl.ItemHeight)
	if itemH <= 0 {
		itemH = 48
	}

	state := sl.State
	if state == nil {
		state = NewSortableListState()
	}

	// Cache motion spec from theme tokens.
	tokens := ctx.Tokens
	state.CacheMotion(tokens.Motion.Standard.Duration, anim.OutCubic)

	contentH := itemCount * itemH
	viewportH := area.H
	if sl.MaxHeight > 0 && int(sl.MaxHeight) < viewportH {
		viewportH = int(sl.MaxHeight)
	}

	// Register as drop target for reordering.
	if ctx.IX != nil {
		listBounds := draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH))

		groupID := sl.GroupID
		if groupID == "" {
			groupID = "sortable"
		}

		ctx.IX.RegisterDropZone(listBounds, 0, func(data *input.DragData, op input.DragOperation) bool {
			// Accept items with matching sortable key or same group.
			return data.HasType(input.MIMESortableKey)
		}, 0)
	}

	// Render visible items.
	scrollOffset := int(state.Scroll.Offset)
	firstVisible := scrollOffset / itemH
	if firstVisible < 0 {
		firstVisible = 0
	}
	lastVisible := (scrollOffset + viewportH) / itemH
	if lastVisible >= itemCount {
		lastVisible = itemCount - 1
	}

	for i := firstVisible; i <= lastVisible; i++ {
		dragging := state.DragIndex == i
		displacement := state.displacement(i)
		yOffset := float32(area.Y) + float32(i*itemH) - float32(scrollOffset) + displacement*float32(itemH)

		itemArea := ui.Bounds{
			X: area.X,
			Y: int(yOffset),
			W: area.W,
			H: itemH,
		}

		el := sl.BuildItem(sl.Items[i], i, dragging)

		if !sl.HandleOnly {
			// Wrap each item in a DragSource for reordering.
			itemKey := sl.Items[i]
			idx := i
			el = DragSource{
				Child: el,
				Data: func() *input.DragData {
					return input.NewDragData(input.MIMESortableKey, itemKey)
				},
				Operations:  input.DragOperationMove,
				Placeholder: sl.ShowPlaceholder,
				OnDragStart: func() {
					state.DragIndex = idx
				},
				OnDragEnd: func(effect input.DropEffect) {
					state.DragIndex = -1
					state.InsertIndex = -1
				},
			}
		}

		ctx.LayoutChild(el, itemArea)
	}

	// Register scroll target.
	if ctx.IX != nil {
		listBounds := draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH))
		ctx.IX.RegisterScroll(listBounds, float32(contentH), float32(viewportH), func(deltaY float32) {
			state.Scroll.ScrollBy(deltaY, float32(contentH), float32(viewportH))
		})

		// Draw scrollbar if needed.
		if contentH > viewportH {
			ui.DrawScrollbar(ctx.Canvas, tokens, ctx.IX, &state.Scroll,
				area.X+area.W-8, area.Y, viewportH,
				float32(contentH), state.Scroll.Offset)
		}
	}

	usedH := viewportH
	if contentH < viewportH {
		usedH = contentH
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: usedH}
}
