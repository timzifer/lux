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

	// Dispatcher is the EventDispatcher used during layout to register
	// widget bounds for hit-testing. May be nil during measure-only passes.
	Dispatcher *EventDispatcher

	// NeedsFrame is set by widgets that have active animations not managed
	// by the reconciler (e.g. button.HoldButtonState). When non-nil and
	// set to true during BuildScene, the framework requests another frame
	// so animations keep rendering.
	NeedsFrame *bool

	// DnD is the drag-and-drop manager for drop zone registration (RFC-005 §4).
	DnD *DnDManager
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

// RegisterSurfaceDrag registers a draggable hit target with a release callback.
// Used by surface slots to forward press/move/release to SurfaceProvider.HandleMsg.
func (ix *Interactor) RegisterSurfaceDrag(bounds draw.Rect, onDrag func(x, y float32), onRelease func(x, y float32)) float32 {
	if ix == nil {
		return 0
	}
	var opacity float32
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
	}
	if ix.hitMap != nil {
		if onDrag != nil {
			ix.hitMap.AddDragRelease(bounds, onDrag, onRelease)
		} else {
			ix.hitMap.Add(bounds, func() {})
		}
	}
	return opacity
}

// RegisterHitRipple registers a clickable hit target with automatic touch
// ripple feedback. Returns hover opacity and the ripple state for drawing.
// The ripple is triggered on click at the touch position.
func (ix *Interactor) RegisterHitRipple(bounds draw.Rect, onClick func()) (float32, *RippleState) {
	if ix == nil {
		return 0, nil
	}
	var opacity float32
	var ripple *RippleState
	if ix.hover != nil {
		opacity = ix.hover.nextButtonHoverOpacity()
		ripple = ix.hover.currentButtonRipple()
	}
	if ix.hitMap != nil {
		rs := ripple // capture for closure
		ix.hitMap.AddAt(bounds, func(x, y float32) {
			if rs != nil {
				rs.Trigger(x, y, MaxRippleRadius(x, y, bounds.X, bounds.Y, bounds.W, bounds.H))
			}
			if onClick != nil {
				onClick()
			}
		})
	}
	return opacity, ripple
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

// SetNeedsFrame signals that the current widget has active animations and
// the framework should schedule another render frame.
func (ix *Interactor) SetNeedsFrame() {
	if ix != nil && ix.NeedsFrame != nil {
		*ix.NeedsFrame = true
	}
}

// RegisterDropZone registers a drop zone for drag-and-drop targeting (RFC-005 §4).
// Unlike hit targets, drop zones do NOT consume hover animation slots and
// are stored in a separate list in the DnDManager.
func (ix *Interactor) RegisterDropZone(bounds draw.Rect, uid UID, accept func(*input.DragData, input.DragOperation) bool, priority int) {
	if ix == nil || ix.DnD == nil {
		return
	}
	ix.DnD.RegisterDropZone(DropZone{
		UID:      uid,
		Bounds:   bounds,
		Accept:   accept,
		Priority: priority,
	})
}

// resetCounter resets the hover animation counter for a new BuildScene pass.
func (ix *Interactor) resetCounter() {
	if ix != nil && ix.hover != nil {
		ix.hover.resetCounter()
	}
}
