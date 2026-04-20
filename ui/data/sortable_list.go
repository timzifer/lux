// Package data — sortable_list.go implements a reorderable list widget
// where items can be rearranged by dragging (RFC-005 §8).
package data

import (
	"time"
	"unsafe"

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
	// When the parent allocates zero height (e.g. unconstrained Flex),
	// fall back to MaxHeight or content height so items are visible.
	if viewportH <= 0 {
		if sl.MaxHeight > 0 {
			viewportH = int(sl.MaxHeight)
		} else {
			viewportH = contentH
		}
	}

	// Register as drop target for reordering.
	if ctx.IX != nil {
		listBounds := draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH))

		groupID := sl.GroupID
		if groupID == "" {
			groupID = "sortable"
		}

		// Use the state pointer as a stable, unique UID so multiple
		// SortableLists don't share UID 0 (which breaks hover detection).
		zoneUID := ui.UID(uintptr(unsafe.Pointer(state)))
		ctx.IX.RegisterDropZone(listBounds, zoneUID, func(data *input.DragData, op input.DragOperation) bool {
			// Accept items with matching sortable key or same group.
			return data.HasType(input.MIMESortableKey)
		}, 0)
	}

	// Scroll offset (used by both drag tracking and item rendering).
	scrollOffset := int(state.Scroll.Offset)

	// Helper: check if cursor is within this list's viewport.
	cursorOverList := func(cx, cy float32) bool {
		return cx >= float32(area.X) && cx < float32(area.X+area.W) &&
			cy >= float32(area.Y) && cy < float32(area.Y+viewportH)
	}
	// Helper: compute insertion index from a cursor Y position.
	insertionIndex := func(cy float32) int {
		relY := cy - float32(area.Y) + float32(scrollOffset)
		idx := int((relY + float32(itemH)/2) / float32(itemH))
		if idx < 0 {
			idx = 0
		}
		if idx > itemCount {
			idx = itemCount
		}
		return idx
	}

	// Track active drag session for insertion index and reorder detection.
	// Handles same-list reorder AND cross-list insert (GroupID).
	if ctx.IX != nil && ctx.IX.DnD != nil {
		dnd := ctx.IX.DnD
		if dnd.IsActive() {
			sess := dnd.Session()
			if sess != nil && sess.Data != nil && sess.Data.HasType(input.MIMESortableKey) {
				dragKey := sortableDragKey(sess.Data)
				foundIdx := sortableFindKey(sl.Items, dragKey)

				if foundIdx >= 0 {
					// Same-list drag — item is in our Items.
					state.DragIndex = foundIdx
					state.SetInsertIndex(insertionIndex(sess.CurrentPos.Y), itemCount)
				} else if sl.groupMatches(sess.Data) &&
					cursorOverList(sess.CurrentPos.X, sess.CurrentPos.Y) {
					// Cross-list: foreign item hovering over this list.
					// DragIndex stays -1; InsertIndex shows the insertion gap.
					state.DragIndex = -1
					state.InsertIndex = insertionIndex(sess.CurrentPos.Y)
				} else {
					// Cursor outside this list or group mismatch — clear.
					state.DragIndex = -1
					state.InsertIndex = -1
				}
			}
		} else if completed := dnd.CompletedDrag(); completed != nil &&
			completed.Data != nil && completed.Data.HasType(input.MIMESortableKey) {
			// Drag just completed — handle same-list reorder OR cross-list transfer.
			dragKey := sortableDragKey(completed.Data)
			foundIdx := sortableFindKey(sl.Items, dragKey)
			overUs := cursorOverList(completed.CurrentPos.X, completed.CurrentPos.Y)

			if foundIdx >= 0 && overUs {
				// Same-list reorder: item originated here, dropped here.
				toIdx := insertionIndex(completed.CurrentPos.Y)
				if toIdx != foundIdx && sl.OnReorder != nil {
					sl.OnReorder(foundIdx, toIdx)
				}
			} else if foundIdx < 0 && overUs && sl.groupMatches(completed.Data) {
				// Foreign item dropped on this list — insert.
				toIdx := insertionIndex(completed.CurrentPos.Y)
				if sl.OnInsert != nil {
					sl.OnInsert(toIdx, completed.Data)
				}
			}
			state.DragIndex = -1
			state.InsertIndex = -1
		} else if state.DragIndex >= 0 || state.InsertIndex >= 0 {
			state.DragIndex = -1
			state.InsertIndex = -1
		}
	}

	// When a cross-list insertion gap is active, the effective content
	// is one item taller. Adjust heights so the viewport, scroll region,
	// and returned bounds all account for the extra slot.
	if state.DragIndex < 0 && state.InsertIndex >= 0 {
		contentH = (itemCount + 1) * itemH
		if sl.MaxHeight <= 0 || contentH < int(sl.MaxHeight) {
			viewportH = contentH
		}
	}

	// Render visible items.
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

		// Compute displacement directly from DragIndex/InsertIndex.
		// SortableListState.Tick() is not called by the framework for
		// non-Widget elements, so animation values stay at 0.
		var displacement float32
		if state.InsertIndex >= 0 {
			if state.DragIndex >= 0 && !dragging {
				// Same-list reorder: shift items between DragIndex and InsertIndex.
				if state.DragIndex < state.InsertIndex && i >= state.DragIndex && i < state.InsertIndex {
					displacement = -1
				} else if state.DragIndex > state.InsertIndex && i >= state.InsertIndex && i < state.DragIndex {
					displacement = 1
				}
			} else if state.DragIndex < 0 {
				// Cross-list insertion: open a gap at InsertIndex.
				if i >= state.InsertIndex {
					displacement = 1
				}
			}
		}

		yOffset := float32(area.Y) + float32(i*itemH) - float32(scrollOffset) + displacement*float32(itemH)

		itemArea := ui.Bounds{
			X: area.X,
			Y: int(yOffset),
			W: area.W,
			H: itemH,
		}

		// Show dashed placeholder for the item being dragged.
		if dragging && sl.ShowPlaceholder {
			placeholderColor := tokens.Colors.Stroke.Border
			placeholderColor.A = 0.3
			ctx.Canvas.StrokeRoundRect(
				draw.R(float32(itemArea.X), float32(itemArea.Y), float32(itemArea.W), float32(itemArea.H)),
				tokens.Radii.Card,
				draw.Stroke{
					Paint:      draw.SolidPaint(placeholderColor),
					Width:      2.0,
					Dash:       []float32{6, 4},
					DashOffset: 0,
				},
			)
			continue
		}

		el := sl.BuildItem(sl.Items[i], i, dragging)

		// In HandleOnly mode, expose a drag config to DragHandle children
		// so they initiate drags with the correct MIMESortableKey data.
		if sl.HandleOnly && ctx.IX != nil && ctx.IX.DnD != nil {
			itemKey := sl.Items[i]
			idx := i
			showPH := sl.ShowPlaceholder
			grp := sl.GroupID
			activeDragHandleConfig = &dragConfig{
				DataFn: func() *input.DragData {
					return sortableDragData(itemKey, grp)
				},
				Operations:  input.DragOperationMove,
				Placeholder: showPH,
				HandleOnly:  true,
				PreviewFn: func() ui.Element {
					if sl.BuildItem != nil {
						return sl.BuildItem(itemKey, idx, true)
					}
					return nil
				},
			}
		}

		childBounds := ctx.LayoutChild(el, itemArea)

		if sl.HandleOnly {
			activeDragHandleConfig = nil
		}

		// Register drag hit target for reordering (bypasses DragSource
		// widget which requires reconciliation not available in LayoutSelf).
		if !sl.HandleOnly && ctx.IX != nil && ctx.IX.DnD != nil {
			itemKey := sl.Items[i]
			idx := i
			itemRect := draw.R(float32(itemArea.X), float32(itemArea.Y),
				float32(childBounds.W), float32(childBounds.H))
			dnd := ctx.IX.DnD
			showPH := sl.ShowPlaceholder

			var startX, startY float32
			var pressing, dragSent bool

			ctx.IX.RegisterSurfaceDrag(itemRect,
				func(x, y float32) {
					if dragSent {
						return
					}
					if !pressing {
						startX, startY = x, y
						pressing = true
						return
					}
					dx := x - startX
					dy := y - startY
					if dx*dx+dy*dy > mouseDragThreshold*mouseDragThreshold {
						data := sortableDragData(itemKey, sl.GroupID)
						data.AllowedOps = input.DragOperationMove

						offset := draw.Point{
							X: itemRect.X - startX,
							Y: itemRect.Y - startY,
						}

						// Build a preview element from the item.
						var preview ui.Element
						if sl.BuildItem != nil {
							preview = sl.BuildItem(itemKey, idx, true)
						}

						dnd.StartDrag(0, data,
							input.GesturePoint{X: startX, Y: startY},
							itemRect, preview, offset, showPH)

						state.DragIndex = idx
						dragSent = true
					}
				},
				func(x, y float32) {
					pressing = false
					// Don't reset DragIndex/InsertIndex here — LayoutSelf
					// detects the drag-ended transition and calls OnReorder.
					dragSent = false
				},
			)
		}
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

// ── Sortable helpers ────────────────────────────────────────────

// sortableDragData creates drag data with MIMESortableKey and optional GroupID.
func sortableDragData(key, groupID string) *input.DragData {
	d := input.NewDragData(input.MIMESortableKey, key)
	if groupID != "" {
		d.Items = append(d.Items, input.DragItem{
			MIMEType: input.MIMESortableGroup,
			Data:     groupID,
		})
	}
	return d
}

// sortableDragKey extracts the item key from sortable drag data.
func sortableDragKey(d *input.DragData) string {
	if v, ok := d.Get(input.MIMESortableKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// sortableFindKey returns the index of key in items, or -1 if not found.
func sortableFindKey(items []string, key string) int {
	if key == "" {
		return -1
	}
	for i, k := range items {
		if k == key {
			return i
		}
	}
	return -1
}

// groupMatches checks if drag data's GroupID matches this list's GroupID.
func (sl SortableList) groupMatches(d *input.DragData) bool {
	if sl.GroupID == "" {
		return false // no group set → no cross-list
	}
	v, ok := d.Get(input.MIMESortableGroup)
	if !ok {
		return false
	}
	g, _ := v.(string)
	return g == sl.GroupID
}
