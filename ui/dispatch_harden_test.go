package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

// ── Focus change events during reconciliation ───────────────────

func TestDispatchFocusGainedLostOnTabNavigation(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widgetA := UID(100)
	widgetB := UID(200)

	fm.RegisterFocusable(widgetA, FocusOpts{Focusable: true, TabIndex: 0})
	fm.RegisterFocusable(widgetB, FocusOpts{Focusable: true, TabIndex: 0})
	fm.SortOrder()
	fm.SetFocusedUID(widgetA)

	// Simulate Tab: A loses focus, B gains.
	d.QueueFocusChange(widgetA, widgetB, FocusSourceTab)
	d.Dispatch()

	evA := d.EventsFor(widgetA)
	evB := d.EventsFor(widgetB)

	if len(evA) != 1 || evA[0].Kind != EventFocusLost {
		t.Errorf("widgetA should get 1 FocusLost, got %v", evA)
	}
	if evA[0].FocusLost.Source != FocusSourceTab {
		t.Errorf("FocusLost source = %d, want FocusSourceTab", evA[0].FocusLost.Source)
	}
	if len(evB) != 1 || evB[0].Kind != EventFocusGained {
		t.Errorf("widgetB should get 1 FocusGained, got %v", evB)
	}
	if evB[0].FocusGained.Source != FocusSourceTab {
		t.Errorf("FocusGained source = %d, want FocusSourceTab", evB[0].FocusGained.Source)
	}
}

// ── Focus change via click (programmatic) ───────────────────────

func TestDispatchFocusChangeViaClick(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	old := UID(1)
	new := UID(2)
	fm.SetFocusedUID(old)

	d.QueueFocusChange(old, new, FocusSourceClick)
	d.Dispatch()

	evOld := d.EventsFor(old)
	if len(evOld) != 1 || evOld[0].Kind != EventFocusLost {
		t.Fatalf("old widget should get FocusLost, got %v", evOld)
	}
	if evOld[0].FocusLost.Source != FocusSourceClick {
		t.Errorf("source = %d, want FocusSourceClick", evOld[0].FocusLost.Source)
	}

	evNew := d.EventsFor(new)
	if len(evNew) != 1 || evNew[0].Kind != EventFocusGained {
		t.Fatalf("new widget should get FocusGained, got %v", evNew)
	}
}

// ── Widget bounds persistence across frames ─────────────────────

func TestDispatchBoundsPersistAcrossSwap(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)

	// Frame 1: register bounds.
	d.RegisterWidgetBounds(widget, draw.R(10, 10, 100, 50))
	d.SwapBounds()

	// Frame 2: register NEW bounds.
	d.RegisterWidgetBounds(widget, draw.R(20, 20, 150, 60))

	// Before swap, mouse should hit using OLD (frame 1) bounds.
	d.Collect(input.MouseMsg{X: 15, Y: 15, Action: input.MousePress})
	d.Dispatch()
	ev := d.EventsFor(widget)
	if len(ev) != 1 {
		t.Fatalf("expected 1 event using old bounds, got %d", len(ev))
	}

	// After swap, new bounds are active.
	d.SwapBounds()
	d.ResetEvents()
	d.Collect(input.MouseMsg{X: 25, Y: 25, Action: input.MousePress})
	d.Dispatch()
	ev = d.EventsFor(widget)
	if len(ev) != 1 {
		t.Fatalf("expected 1 event using new bounds, got %d", len(ev))
	}
}

// ── Multiple mouse events to overlapping widgets ────────────────

func TestDispatchMultipleEventsRouting(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widgetA := UID(1)
	widgetB := UID(2)
	fm.SetFocusedUID(widgetA)

	d.RegisterWidgetBounds(widgetA, draw.R(0, 0, 100, 100))
	d.RegisterWidgetBounds(widgetB, draw.R(200, 0, 100, 100))
	d.SwapBounds()

	// Key goes to focused (A), mouse to B.
	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})
	d.Collect(input.MouseMsg{X: 250, Y: 50, Action: input.MousePress})
	d.Dispatch()

	evA := d.EventsFor(widgetA)
	evB := d.EventsFor(widgetB)

	if len(evA) != 1 || evA[0].Kind != EventKey {
		t.Errorf("widgetA should get key event, got %v", evA)
	}
	if len(evB) != 1 || evB[0].Kind != EventMouse {
		t.Errorf("widgetB should get mouse event, got %v", evB)
	}
}

// ── Overlay dismiss only fires for dismissable overlays ─────────

func TestDispatchOverlayClickInsideNoDismiss(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	var dismissed OverlayID
	d.SetDismissHandler(func(id OverlayID) { dismissed = id })

	d.RegisterOverlay("popup", draw.R(100, 100, 200, 200), true)

	// Click INSIDE the overlay — should NOT dismiss.
	d.Collect(input.MouseMsg{X: 150, Y: 150, Action: input.MousePress})
	d.Dispatch()

	if dismissed != "" {
		t.Errorf("click inside overlay should not dismiss, but dismissed %q", dismissed)
	}
}

// ── Overlay dismiss and no-dismiss scenarios ────────────────────

func TestDispatchOverlayDismissStack(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	var dismissed []OverlayID
	d.SetDismissHandler(func(id OverlayID) {
		dismissed = append(dismissed, id)
	})

	// Register two overlapping overlays.
	d.RegisterOverlay("bottom", draw.R(50, 50, 200, 200), true)
	d.RegisterOverlay("top", draw.R(100, 100, 100, 100), true)

	// Click outside both overlays.
	d.Collect(input.MouseMsg{X: 10, Y: 10, Action: input.MousePress})
	d.Dispatch()

	// At least one overlay should be dismissed.
	if len(dismissed) == 0 {
		t.Error("clicking outside all overlays should dismiss at least one")
	}
}

// ── IME events routed to focused widget ─────────────────────────

func TestDispatchIMEEventsToFocused(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)
	fm.SetFocusedUID(widget)

	d.Collect(input.IMEComposeMsg{Text: "にほ"})
	d.Collect(input.IMECommitMsg{Text: "日本"})
	d.Dispatch()

	ev := d.EventsFor(widget)
	if len(ev) != 2 {
		t.Fatalf("expected 2 IME events, got %d", len(ev))
	}
	if ev[0].Kind != EventIMECompose {
		t.Errorf("event[0] = %d, want EventIMECompose", ev[0].Kind)
	}
	if ev[1].Kind != EventIMECommit {
		t.Errorf("event[1] = %d, want EventIMECommit", ev[1].Kind)
	}
}

// ── FilterCollectedEvents with mixed event types ────────────────

func TestFilterCollectedEventsSelectiveConsume(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)
	fm.SetFocusedUID(UID(1))

	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})
	d.Collect(input.KeyMsg{Key: input.KeyEnter, Action: input.KeyPress})
	d.Collect(input.KeyMsg{Key: input.KeyB, Action: input.KeyPress})

	// Consume only Enter key events.
	d.FilterCollectedEvents(func(ev InputEvent) bool {
		return ev.Kind == EventKey && ev.Key != nil && ev.Key.Key == input.KeyEnter
	})
	d.Dispatch()

	ev := d.EventsFor(UID(1))
	if len(ev) != 2 {
		t.Fatalf("expected 2 events after filtering Enter, got %d", len(ev))
	}
	for _, e := range ev {
		if e.Key.Key == input.KeyEnter {
			t.Error("Enter key should have been consumed by filter")
		}
	}
}

// ── FocusTrap constrains tab navigation with dispatch ───────────

func TestDispatchFocusTrapConstrainsKeyboardNavigation(t *testing.T) {
	fm := NewFocusManager()
	fm.Trap = NewFocusTrapManager()

	// Register global widgets.
	fm.RegisterFocusable(UID(1), FocusOpts{Focusable: true})
	fm.RegisterFocusable(UID(2), FocusOpts{Focusable: true})
	fm.RegisterFocusable(UID(3), FocusOpts{Focusable: true})
	fm.SortOrder()

	// Push trap containing only UIDs 2 and 3.
	initial := fm.Trap.PushTrap(FocusTrap{TrapID: "dialog", RestoreFocus: true}, UID(1), []UID{2, 3})
	fm.SetFocusedUID(initial)

	// Tab should cycle within {2, 3}, never reaching 1.
	visited := make(map[UID]bool)
	for i := 0; i < 10; i++ {
		next := fm.FocusNext()
		visited[next] = true
	}

	if visited[UID(1)] {
		t.Error("UID(1) should never be reached inside the trap")
	}
	if !visited[UID(2)] || !visited[UID(3)] {
		t.Error("UIDs 2 and 3 should both be visited inside the trap")
	}

	// Pop trap and verify we can reach UID(1) again.
	restored := fm.Trap.PopTrap("dialog")
	fm.SetFocusedUID(restored)

	visited2 := make(map[UID]bool)
	for i := 0; i < 5; i++ {
		next := fm.FocusNext()
		visited2[next] = true
	}
	if !visited2[UID(1)] {
		t.Error("UID(1) should be reachable after trap pop")
	}
}

// ── Reconciler + Dispatch end-to-end ───────────────────────────

func TestReconcileDispatchEndToEnd(t *testing.T) {
	r := NewReconciler()
	th := theme.Default
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	// Two focusable event-capture widgets.
	tree := Column(
		ComponentWithKey("w1", eventCapture{}),
		ComponentWithKey("w2", eventCapture{}),
	)

	// Frame 1: establish widgets.
	r.Reconcile(tree, th, noopSend, d, fm, "", nil)

	uid1 := MakeUID(0, "w1", 0)
	uid2 := MakeUID(0, "w2", 0)

	// Focus w1, send key, dispatch.
	fm.SetFocusedUID(uid1)
	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})
	d.Dispatch()

	// Frame 2: reconcile with dispatch.
	r.Reconcile(tree, th, noopSend, d, fm, "", nil)

	s1 := r.StateFor(uid1).(*eventCaptureState)
	s2 := r.StateFor(uid2).(*eventCaptureState)

	if len(s1.Events) != 1 {
		t.Errorf("w1 should get 1 event, got %d", len(s1.Events))
	}
	if len(s2.Events) != 0 {
		t.Errorf("w2 should get 0 events, got %d", len(s2.Events))
	}

	// Reset and send to w2.
	d.ResetEvents()
	fm.SetFocusedUID(uid2)
	d.Collect(input.KeyMsg{Key: input.KeyB, Action: input.KeyPress})
	d.Dispatch()

	r.Reconcile(tree, th, noopSend, d, fm, "", nil)

	s1 = r.StateFor(uid1).(*eventCaptureState)
	s2 = r.StateFor(uid2).(*eventCaptureState)

	if len(s1.Events) != 0 {
		t.Errorf("w1 should get 0 events in frame 3, got %d", len(s1.Events))
	}
	if len(s2.Events) != 1 {
		t.Errorf("w2 should get 1 event in frame 3, got %d", len(s2.Events))
	}
}

// ── Bounds for removed widgets are cleaned up ───────────────────

func TestDispatchBoundsCleanupOnSwap(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widget := UID(42)
	d.RegisterWidgetBounds(widget, draw.R(0, 0, 100, 100))
	d.SwapBounds()

	// Don't re-register in next frame — simulates widget removal.
	d.SwapBounds()

	// Mouse in old bounds should NOT route to removed widget.
	d.Collect(input.MouseMsg{X: 50, Y: 50, Action: input.MousePress})
	d.Dispatch()

	ev := d.EventsFor(widget)
	if len(ev) != 0 {
		t.Errorf("removed widget should get 0 events, got %d", len(ev))
	}
}

// ── Scroll events to nested widget ──────────────────────────────

func TestDispatchScrollToNestedWidget(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	parent := UID(1)
	child := UID(2)

	d.RegisterWidgetBounds(parent, draw.R(0, 0, 300, 300))
	d.RegisterWidgetBounds(child, draw.R(10, 10, 100, 100))
	d.SwapBounds()

	// Scroll inside child bounds — child (smaller) should win.
	d.Collect(input.ScrollMsg{X: 50, Y: 50, DeltaY: 5})
	d.Dispatch()

	evParent := d.EventsFor(parent)
	evChild := d.EventsFor(child)

	if len(evChild) != 1 {
		t.Errorf("child should get 1 scroll event, got %d", len(evChild))
	}
	if len(evParent) != 0 {
		t.Errorf("parent should get 0 events (child wins hit test), got %d", len(evParent))
	}
}

// ── Stability regression tests ──────────────────────────────────
// See milestone "Stability: Reconciler / Scene / AccessTree"

// TestDispatchHitTestDeterministicWithIdenticalBounds verifies that hit-testing
// two widgets with identical bounds always returns the same winner, regardless
// of Go map iteration order. (Issue #94)
func TestDispatchHitTestDeterministicWithIdenticalBounds(t *testing.T) {
	fm := NewFocusManager()

	// Run 100 iterations — map order is randomized per Go spec.
	var firstWinner UID
	for i := 0; i < 100; i++ {
		d := NewEventDispatcher(fm)
		d.RegisterWidgetBounds(UID(1), draw.R(0, 0, 100, 100))
		d.RegisterWidgetBounds(UID(2), draw.R(0, 0, 100, 100))
		d.SwapBounds()

		d.Collect(input.MouseMsg{X: 50, Y: 50, Action: input.MousePress})
		d.Dispatch()

		ev1 := d.EventsFor(UID(1))
		ev2 := d.EventsFor(UID(2))

		var winner UID
		if len(ev1) == 1 && len(ev2) == 0 {
			winner = UID(1)
		} else if len(ev2) == 1 && len(ev1) == 0 {
			winner = UID(2)
		} else {
			t.Fatalf("iteration %d: expected exactly 1 event to exactly 1 widget, got ev1=%d ev2=%d", i, len(ev1), len(ev2))
		}

		if i == 0 {
			firstWinner = winner
		} else if winner != firstWinner {
			t.Fatalf("iteration %d: winner=%d but first winner=%d — hit-test is non-deterministic", i, winner, firstWinner)
		}
	}
}

// TestDispatchFocusChangeAndKeyEventSameFrame verifies that a key event
// collected before a focus change is still routed to the originally focused
// widget. (Issue #94)
func TestDispatchFocusChangeAndKeyEventSameFrame(t *testing.T) {
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	widgetA := UID(100)
	widgetB := UID(200)
	fm.SetFocusedUID(widgetA)

	// Collect key event (focus is on A).
	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})

	// Queue focus change A→B before Dispatch.
	d.QueueFocusChange(widgetA, widgetB, FocusSourceTab)
	d.Dispatch()

	evA := d.EventsFor(widgetA)
	evB := d.EventsFor(widgetB)

	// Key event should go to A (it was focused at dispatch time).
	hasKey := false
	for _, e := range evA {
		if e.Kind == EventKey {
			hasKey = true
		}
	}
	if !hasKey {
		t.Error("widgetA should receive the key event (it was focused at dispatch time)")
	}

	// A should also get FocusLost.
	hasFocusLost := false
	for _, e := range evA {
		if e.Kind == EventFocusLost {
			hasFocusLost = true
		}
	}
	if !hasFocusLost {
		t.Error("widgetA should receive FocusLost event")
	}

	// B should get FocusGained.
	hasFocusGained := false
	for _, e := range evB {
		if e.Kind == EventFocusGained {
			hasFocusGained = true
		}
	}
	if !hasFocusGained {
		t.Error("widgetB should receive FocusGained event")
	}
}
