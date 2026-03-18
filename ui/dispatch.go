// Package ui – dispatch.go implements the framework-internal input event
// dispatcher (RFC-002 §2.6). It collects raw input events per frame and
// routes them to widgets via RenderCtx.Events:
//
//   Mouse/Scroll/Touch → hit-test against previous-frame widget bounds → Widget UID
//   Keyboard/Char/Text → FocusManager → focused Widget UID
//   Focus gained/lost  → delivered to affected Widget UIDs
package ui

import (
	"sort"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
)

// EventDispatcher routes input events to widgets by UID (RFC-002 §2.6).
//
// The dispatch cycle works with one-frame latency for mouse routing:
//  1. Layout pass registers widget bounds via RegisterWidgetBounds.
//  2. Next frame: raw input events are collected via Collect.
//  3. Before reconciliation, Dispatch routes events to per-UID buffers.
//  4. Reconciler reads EventsFor(uid) to populate RenderCtx.Events.
type EventDispatcher struct {
	fm *FocusManager

	// widgetBounds maps UID → screen bounds from the previous layout pass.
	widgetBounds map[UID]draw.Rect

	// nextBounds accumulates bounds during the current layout pass.
	nextBounds map[UID]draw.Rect

	// Per-frame raw event buffers.
	keyEvents    []input.KeyMsg
	charEvents   []input.CharMsg
	textEvents   []input.TextInputMsg
	mouseEvents  []input.MouseMsg
	scrollEvents []input.ScrollMsg
	touchEvents  []input.TouchMsg

	// Computed per-UID event buffers (populated by Dispatch).
	events map[UID][]InputEvent

	// Focus transitions queued during this frame.
	focusGained map[UID]FocusGainedMsg
	focusLost   map[UID]FocusLostMsg

	// Overlay tracking for input priority (RFC-002 §5.3).
	overlays       []dispatchOverlayEntry
	dismissHandler func(id OverlayID) // callback to send DismissOverlayMsg
}

// dispatchOverlayEntry tracks an active overlay's bounds and properties.
type dispatchOverlayEntry struct {
	id          OverlayID
	bounds      draw.Rect
	dismissable bool
}

// NewEventDispatcher creates a dispatcher bound to the given FocusManager.
func NewEventDispatcher(fm *FocusManager) *EventDispatcher {
	return &EventDispatcher{
		fm:           fm,
		widgetBounds: make(map[UID]draw.Rect),
		nextBounds:   make(map[UID]draw.Rect),
		events:       make(map[UID][]InputEvent),
		focusGained:  make(map[UID]FocusGainedMsg),
		focusLost:    make(map[UID]FocusLostMsg),
	}
}

// Collect adds a raw input event to the per-frame buffer.
func (d *EventDispatcher) Collect(msg any) {
	switch m := msg.(type) {
	case input.KeyMsg:
		d.keyEvents = append(d.keyEvents, m)
	case input.CharMsg:
		d.charEvents = append(d.charEvents, m)
	case input.TextInputMsg:
		d.textEvents = append(d.textEvents, m)
	case input.MouseMsg:
		d.mouseEvents = append(d.mouseEvents, m)
	case input.ScrollMsg:
		d.scrollEvents = append(d.scrollEvents, m)
	case input.TouchMsg:
		d.touchEvents = append(d.touchEvents, m)
	}
}

// QueueFocusChange records a focus transition. The old UID gets a
// FocusLostMsg and the new UID gets a FocusGainedMsg, delivered
// during the next Dispatch.
func (d *EventDispatcher) QueueFocusChange(oldUID, newUID UID, source FocusSource) {
	if oldUID != 0 {
		d.focusLost[oldUID] = FocusLostMsg{Source: source}
	}
	if newUID != 0 {
		d.focusGained[newUID] = FocusGainedMsg{Source: source}
	}
}

// Dispatch routes all collected events to per-UID buffers.
// Call this before reconciliation each frame.
func (d *EventDispatcher) Dispatch() {
	// Clear per-UID event buffers.
	for uid := range d.events {
		delete(d.events, uid)
	}

	focusedUID := d.fm.FocusedUID()

	// Route keyboard events → focused widget.
	if focusedUID != 0 {
		for i := range d.keyEvents {
			d.appendEvent(focusedUID, KeyEvent(d.keyEvents[i]))
		}
		for i := range d.charEvents {
			d.appendEvent(focusedUID, CharEvent(d.charEvents[i]))
		}
		for i := range d.textEvents {
			d.appendEvent(focusedUID, TextInputEvent(d.textEvents[i]))
		}
	}

	// Route mouse events → overlay (priority) or hit-tested widget.
	for i := range d.mouseEvents {
		m := d.mouseEvents[i]
		if d.routeToOverlayOrDismiss(m.X, m.Y, m.Action == input.MousePress) {
			continue // consumed by overlay logic
		}
		if uid := d.hitTestWidget(m.X, m.Y); uid != 0 {
			d.appendEvent(uid, MouseEvent(m))
		}
	}

	// Route scroll events → hit-tested widget (overlays don't consume scrolls by default).
	for i := range d.scrollEvents {
		m := d.scrollEvents[i]
		if uid := d.hitTestWidget(m.X, m.Y); uid != 0 {
			d.appendEvent(uid, ScrollEvent(m))
		}
	}

	// Route touch events → hit-tested widget.
	for i := range d.touchEvents {
		m := d.touchEvents[i]
		if uid := d.hitTestWidget(m.X, m.Y); uid != 0 {
			d.appendEvent(uid, TouchEvent(m))
		}
	}

	// Deliver focus transition events.
	for uid, msg := range d.focusGained {
		d.appendEvent(uid, FocusGainedEvent(msg))
	}
	for uid, msg := range d.focusLost {
		d.appendEvent(uid, FocusLostEvent(msg))
	}
}

// EventsFor returns the dispatched events for a widget UID.
func (d *EventDispatcher) EventsFor(uid UID) []InputEvent {
	return d.events[uid]
}

// RegisterWidgetBounds records a widget's screen bounds during layout.
// These bounds are used for hit-testing in the next frame.
func (d *EventDispatcher) RegisterWidgetBounds(uid UID, bounds draw.Rect) {
	d.nextBounds[uid] = bounds
}

// SwapBounds promotes nextBounds to widgetBounds and clears nextBounds.
// Call this after the layout pass completes.
func (d *EventDispatcher) SwapBounds() {
	d.widgetBounds, d.nextBounds = d.nextBounds, d.widgetBounds
	for uid := range d.nextBounds {
		delete(d.nextBounds, uid)
	}
}

// ResetEvents clears per-frame event buffers for the next frame.
func (d *EventDispatcher) ResetEvents() {
	d.keyEvents = d.keyEvents[:0]
	d.charEvents = d.charEvents[:0]
	d.textEvents = d.textEvents[:0]
	d.mouseEvents = d.mouseEvents[:0]
	d.scrollEvents = d.scrollEvents[:0]
	d.touchEvents = d.touchEvents[:0]

	for uid := range d.focusGained {
		delete(d.focusGained, uid)
	}
	for uid := range d.focusLost {
		delete(d.focusLost, uid)
	}
}

// RegisterOverlay tracks an active overlay for input priority (RFC-002 §5.3).
func (d *EventDispatcher) RegisterOverlay(id OverlayID, bounds draw.Rect, dismissable bool) {
	d.overlays = append(d.overlays, dispatchOverlayEntry{id: id, bounds: bounds, dismissable: dismissable})
}

// ClearOverlays removes all registered overlays (called each frame before re-registration).
func (d *EventDispatcher) ClearOverlays() {
	d.overlays = d.overlays[:0]
}

// SetDismissHandler sets the callback invoked when a dismissable overlay
// should be closed (click outside).
func (d *EventDispatcher) SetDismissHandler(fn func(id OverlayID)) {
	d.dismissHandler = fn
}

// HasOverlays reports whether any overlays are currently registered.
func (d *EventDispatcher) HasOverlays() bool {
	return len(d.overlays) > 0
}

// FilterCollectedEvents runs handler against all collected raw events.
// Events for which handler returns true are removed from the buffers
// (consumed). This implements the Global Handler Layer (RFC-002 §2.8).
func (d *EventDispatcher) FilterCollectedEvents(handler func(InputEvent) bool) {
	d.keyEvents = filterSlice(d.keyEvents, func(m input.KeyMsg) bool {
		return handler(KeyEvent(m))
	})
	d.charEvents = filterSlice(d.charEvents, func(m input.CharMsg) bool {
		return handler(CharEvent(m))
	})
	d.textEvents = filterSlice(d.textEvents, func(m input.TextInputMsg) bool {
		return handler(TextInputEvent(m))
	})
	d.mouseEvents = filterSlice(d.mouseEvents, func(m input.MouseMsg) bool {
		return handler(MouseEvent(m))
	})
	d.scrollEvents = filterSlice(d.scrollEvents, func(m input.ScrollMsg) bool {
		return handler(ScrollEvent(m))
	})
	d.touchEvents = filterSlice(d.touchEvents, func(m input.TouchMsg) bool {
		return handler(TouchEvent(m))
	})
}

// filterSlice removes elements for which consumed returns true (in-place).
func filterSlice[T any](s []T, consumed func(T) bool) []T {
	n := 0
	for _, v := range s {
		if !consumed(v) {
			s[n] = v
			n++
		}
	}
	return s[:n]
}

// hitTestWidget returns the UID of the top-most widget whose bounds
// contain point (x, y), or 0 if none match. Widgets registered later
// (higher Z-order) are tested first.
func (d *EventDispatcher) hitTestWidget(x, y float32) UID {
	pt := draw.Pt(x, y)
	// Build a sorted slice for deterministic iteration (map order is random).
	// We check all entries and return the last-registered match (highest Z).
	type entry struct {
		uid    UID
		bounds draw.Rect
	}
	var match UID
	for uid, b := range d.widgetBounds {
		if b.Contains(pt) {
			// Prefer the smallest containing bounds (most specific widget).
			if match == 0 {
				match = uid
			} else {
				prev := d.widgetBounds[match]
				if b.W*b.H < prev.W*prev.H {
					match = uid
				}
			}
		}
	}
	return match
}

// routeToOverlayOrDismiss checks if a click hits an overlay or should dismiss one.
// Returns true if the event was consumed by overlay logic.
func (d *EventDispatcher) routeToOverlayOrDismiss(x, y float32, isPress bool) bool {
	if len(d.overlays) == 0 {
		return false
	}
	pt := draw.Pt(x, y)

	// Check overlays in reverse order (newest = highest Z).
	for i := len(d.overlays) - 1; i >= 0; i-- {
		if d.overlays[i].bounds.Contains(pt) {
			return false // click inside overlay → let normal dispatch handle it
		}
	}

	// Click outside all overlays → dismiss the top-most dismissable overlay.
	if isPress && d.dismissHandler != nil {
		for i := len(d.overlays) - 1; i >= 0; i-- {
			if d.overlays[i].dismissable {
				d.dismissHandler(d.overlays[i].id)
				return true // consumed
			}
		}
	}

	return false
}

func (d *EventDispatcher) appendEvent(uid UID, ev InputEvent) {
	d.events[uid] = append(d.events[uid], ev)
}

// ── Widget bounds tracking (for layout registration) ────────────

// widgetBoundsEntry is used to maintain insertion order for hit-testing.
type widgetBoundsEntry struct {
	uid    UID
	bounds draw.Rect
	order  int // registration order (layout order = Z-order)
}

// widgetBoundsSlice implements sort.Interface by order descending
// (last registered = highest Z = checked first).
type widgetBoundsSlice []widgetBoundsEntry

func (s widgetBoundsSlice) Len() int           { return len(s) }
func (s widgetBoundsSlice) Less(i, j int) bool { return s[i].order > s[j].order }
func (s widgetBoundsSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

var _ sort.Interface = widgetBoundsSlice{}
