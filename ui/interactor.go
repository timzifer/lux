package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
)

// Interactor couples hit-target registration with hover animation tracking.
// Every call that adds a clickable hit target also consumes exactly one hover
// animation slot, so hitMap indices and hover animation indices can never
// diverge. All interactive widgets must use Interactor methods instead of
// directly touching hit.Map or HoverState.
//
// A nil *Interactor is safe to call — all methods return zero values.
type Interactor struct {
	hitMap *hit.Map
	hover  *HoverState
}

// NewInteractor creates an Interactor for use during a single BuildScene pass.
// Either parameter may be nil (e.g. during measure-only passes).
func NewInteractor(hitMap *hit.Map, hover *HoverState) *Interactor {
	if hitMap == nil && hover == nil {
		return nil
	}
	return &Interactor{hitMap: hitMap, hover: hover}
}

// RegisterHit registers a clickable hit target and returns the current hover
// opacity for the element. If onClick is nil, a no-op target is still
// registered so that the hover index stays aligned.
func (ix *Interactor) RegisterHit(bounds draw.Rect, onClick func()) float32 {
	if ix == nil {
		return 0
	}
	var opacity float32
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
	}
	if ix.hitMap != nil {
		if onClick == nil {
			onClick = func() {}
		}
		ix.hitMap.Add(bounds, onClick)
	}
	return opacity
}

// RegisterDrag registers a draggable hit target (fires continuously while
// mouse is held) and returns the current hover opacity. If onDrag is nil,
// a no-op click target is registered instead.
func (ix *Interactor) RegisterDrag(bounds draw.Rect, onDrag func(x, y float32)) float32 {
	if ix == nil {
		return 0
	}
	var opacity float32
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
	}
	if ix.hitMap != nil {
		if onDrag != nil {
			ix.hitMap.AddDrag(bounds, onDrag)
		} else {
			ix.hitMap.Add(bounds, func() {})
		}
	}
	return opacity
}

// RegisterClickAt registers a positional click target (e.g. scrollbar track)
// and returns the current hover opacity. If onClick is nil, a no-op click
// target is registered instead.
func (ix *Interactor) RegisterClickAt(bounds draw.Rect, onClick func(x, y float32)) float32 {
	if ix == nil {
		return 0
	}
	var opacity float32
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
	}
	if ix.hitMap != nil {
		if onClick != nil {
			ix.hitMap.AddAt(bounds, onClick)
		} else {
			ix.hitMap.Add(bounds, func() {})
		}
	}
	return opacity
}

// RegisterDragCursor registers a draggable hit target with a custom hover
// cursor and returns the current hover opacity. If onDrag is nil, a no-op
// click target is registered instead.
func (ix *Interactor) RegisterDragCursor(bounds draw.Rect, cursor input.CursorKind, onDrag func(x, y float32)) float32 {
	if ix == nil {
		return 0
	}
	var opacity float32
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
	}
	if ix.hitMap != nil {
		if onDrag != nil {
			ix.hitMap.AddDragCursor(bounds, cursor, onDrag)
		} else {
			ix.hitMap.Add(bounds, func() {})
		}
	}
	return opacity
}

// RegisterScroll registers a scrollable viewport region. Scroll targets use
// a separate target list in hit.Map, so they do NOT consume a hover slot and
// cannot cause index misalignment.
func (ix *Interactor) RegisterScroll(bounds draw.Rect, contentH, viewportH float32, onScroll func(deltaY float32)) {
	if ix == nil || ix.hitMap == nil || onScroll == nil {
		return
	}
	ix.hitMap.AddScroll(bounds, contentH, viewportH, onScroll)
}

// resetCounter resets the hover animation counter for a new BuildScene pass.
func (ix *Interactor) resetCounter() {
	if ix != nil && ix.hover != nil {
		ix.hover.resetCounter()
	}
}
