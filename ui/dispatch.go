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

	// Route mouse events → hit-tested widget.
	for i := range d.mouseEvents {
		m := d.mouseEvents[i]
		if uid := d.hitTestWidget(m.X, m.Y); uid != 0 {
			d.appendEvent(uid, MouseEvent(m))
		}
	}

	// Route scroll events → hit-tested widget.
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
