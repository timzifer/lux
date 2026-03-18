package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
)

func TestEventDispatcherKeyboardToFocused(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widgetA := UID(100)
	widgetB := UID(200)
	fm.SetFocusedUID(widgetA)

	// Collect keyboard events.
	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})
	d.Collect(input.CharMsg{Char: 'a'})
	d.Collect(input.TextInputMsg{Text: "hello"})

	d.Dispatch()

	// Events should be routed to the focused widget.
	evA := d.EventsFor(widgetA)
	if len(evA) != 3 {
		t.Fatalf("focused widget should get 3 events, got %d", len(evA))
	}
	if evA[0].Kind != EventKey {
		t.Errorf("event[0].Kind = %d, want EventKey", evA[0].Kind)
	}
	if evA[1].Kind != EventChar {
		t.Errorf("event[1].Kind = %d, want EventChar", evA[1].Kind)
	}
	if evA[2].Kind != EventTextInput {
		t.Errorf("event[2].Kind = %d, want EventTextInput", evA[2].Kind)
	}

	// Unfocused widget gets nothing.
	evB := d.EventsFor(widgetB)
	if len(evB) != 0 {
		t.Errorf("unfocused widget should get 0 events, got %d", len(evB))
	}
}

func TestEventDispatcherMouseToHitTested(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widgetA := UID(100)
	widgetB := UID(200)

	// Register widget bounds from "previous frame".
	d.RegisterWidgetBounds(widgetA, draw.R(0, 0, 100, 100))
	d.RegisterWidgetBounds(widgetB, draw.R(200, 200, 100, 100))
	d.SwapBounds()

	// Mouse click inside widgetA bounds.
	d.Collect(input.MouseMsg{X: 50, Y: 50, Action: input.MousePress})

	d.Dispatch()

	evA := d.EventsFor(widgetA)
	if len(evA) != 1 {
		t.Fatalf("widgetA should get 1 mouse event, got %d", len(evA))
	}
	if evA[0].Kind != EventMouse {
		t.Errorf("event kind = %d, want EventMouse", evA[0].Kind)
	}

	evB := d.EventsFor(widgetB)
	if len(evB) != 0 {
		t.Errorf("widgetB should get 0 events, got %d", len(evB))
	}
}

func TestEventDispatcherScrollToHitTested(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)
	d.RegisterWidgetBounds(widget, draw.R(10, 10, 200, 200))
	d.SwapBounds()

	d.Collect(input.ScrollMsg{X: 50, Y: 50, DeltaY: 10})
	d.Dispatch()

	ev := d.EventsFor(widget)
	if len(ev) != 1 {
		t.Fatalf("expected 1 scroll event, got %d", len(ev))
	}
	if ev[0].Kind != EventScroll {
		t.Errorf("event kind = %d, want EventScroll", ev[0].Kind)
	}
}

func TestEventDispatcherTouchToHitTested(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)
	d.RegisterWidgetBounds(widget, draw.R(10, 10, 200, 200))
	d.SwapBounds()

	d.Collect(input.TouchMsg{X: 50, Y: 50, Phase: input.TouchBegan})
	d.Dispatch()

	ev := d.EventsFor(widget)
	if len(ev) != 1 {
		t.Fatalf("expected 1 touch event, got %d", len(ev))
	}
	if ev[0].Kind != EventTouch {
		t.Errorf("event kind = %d, want EventTouch", ev[0].Kind)
	}
}

func TestEventDispatcherFocusTransitions(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	oldWidget := UID(100)
	newWidget := UID(200)

	d.QueueFocusChange(oldWidget, newWidget, FocusSourceTab)
	d.Dispatch()

	evOld := d.EventsFor(oldWidget)
	if len(evOld) != 1 || evOld[0].Kind != EventFocusLost {
		t.Errorf("old widget should get FocusLost, got %v", evOld)
	}
	if evOld[0].FocusLost.Source != FocusSourceTab {
		t.Errorf("FocusLost source = %d, want FocusSourceTab", evOld[0].FocusLost.Source)
	}

	evNew := d.EventsFor(newWidget)
	if len(evNew) != 1 || evNew[0].Kind != EventFocusGained {
		t.Errorf("new widget should get FocusGained, got %v", evNew)
	}
}

func TestEventDispatcherNoFocusNoKeyboardEvents(t *testing.T) {
	fm := NewFocusManager() // nothing focused
	d := NewEventDispatcher(fm)

	d.Collect(input.KeyMsg{Key: input.KeyA})
	d.Dispatch()

	// No widget should receive the key event.
	if len(d.EventsFor(UID(0))) != 0 {
		t.Error("UID(0) should not receive events")
	}
}

func TestEventDispatcherMouseMissesAllWidgets(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)
	d.RegisterWidgetBounds(widget, draw.R(0, 0, 50, 50))
	d.SwapBounds()

	// Click outside all widget bounds.
	d.Collect(input.MouseMsg{X: 999, Y: 999, Action: input.MousePress})
	d.Dispatch()

	ev := d.EventsFor(widget)
	if len(ev) != 0 {
		t.Errorf("widget should get 0 events for miss, got %d", len(ev))
	}
}

func TestEventDispatcherResetEvents(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)
	fm.SetFocusedUID(UID(1))

	d.Collect(input.KeyMsg{Key: input.KeyA})
	d.QueueFocusChange(0, UID(1), FocusSourceProgram)
	d.Dispatch()

	// After reset, new dispatch should produce no events.
	d.ResetEvents()
	d.Dispatch()

	ev := d.EventsFor(UID(1))
	if len(ev) != 0 {
		t.Errorf("after reset, should get 0 events, got %d", len(ev))
	}
}

func TestEventDispatcherSwapBounds(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	// Register bounds for current frame.
	d.RegisterWidgetBounds(UID(1), draw.R(0, 0, 100, 100))

	// Before swap, mouse hits should NOT work (bounds are in nextBounds).
	d.Collect(input.MouseMsg{X: 50, Y: 50})
	d.Dispatch()
	if len(d.EventsFor(UID(1))) != 0 {
		t.Error("before swap, events should not route to nextBounds")
	}

	// After swap, bounds are promoted.
	d.SwapBounds()
	d.ResetEvents()
	d.Collect(input.MouseMsg{X: 50, Y: 50})
	d.Dispatch()
	if len(d.EventsFor(UID(1))) != 1 {
		t.Error("after swap, events should route to promoted bounds")
	}
}

func TestEventDispatcherSmallestBoundsWins(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	// Overlapping widgets — smaller one should win.
	d.RegisterWidgetBounds(UID(1), draw.R(0, 0, 200, 200))    // large
	d.RegisterWidgetBounds(UID(2), draw.R(10, 10, 50, 50))     // small, nested
	d.SwapBounds()

	d.Collect(input.MouseMsg{X: 20, Y: 20, Action: input.MousePress})
	d.Dispatch()

	ev1 := d.EventsFor(UID(1))
	ev2 := d.EventsFor(UID(2))
	if len(ev2) != 1 {
		t.Errorf("smaller widget should get 1 event, got %d", len(ev2))
	}
	if len(ev1) != 0 {
		t.Errorf("larger widget should get 0 events, got %d", len(ev1))
	}
}
