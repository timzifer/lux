package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
)

func TestDnDManagerInitialState(t *testing.T) {
	m := NewDnDManager()
	if m.IsActive() {
		t.Error("new manager should not be active")
	}
	if m.Session() != nil {
		t.Error("new manager should have nil session")
	}
	if m.HoveredZoneUID() != 0 {
		t.Error("new manager should have no hovered zone")
	}
}

func testDragData() *input.DragData {
	return input.NewTextDragData("hello")
}

func TestDnDManagerStartDrag(t *testing.T) {
	m := NewDnDManager()
	data := testDragData()
	startPos := input.GesturePoint{X: 100, Y: 100}
	bounds := draw.R(90, 90, 40, 20)

	m.StartDrag(42, data, startPos, bounds, nil, draw.Point{}, false)

	if !m.IsActive() {
		t.Fatal("manager should be active after StartDrag")
	}
	s := m.Session()
	if s == nil {
		t.Fatal("session should not be nil")
	}
	if s.SourceUID != 42 {
		t.Errorf("SourceUID = %d, want 42", s.SourceUID)
	}
	if s.Data != data {
		t.Error("session data should match")
	}
	if s.StartPos != startPos {
		t.Errorf("StartPos = %v, want %v", s.StartPos, startPos)
	}
	if s.Phase != DragSessionActive {
		t.Errorf("Phase = %d, want DragSessionActive", s.Phase)
	}
}

func TestDnDManagerUpdateDrag(t *testing.T) {
	m := NewDnDManager()
	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 50, Y: 50}, draw.Rect{}, nil, draw.Point{}, false)

	newPos := input.GesturePoint{X: 150, Y: 200}
	ok := m.UpdateDrag(newPos, input.ModCtrl)
	if !ok {
		t.Fatal("UpdateDrag should return true when active")
	}

	s := m.Session()
	if s.CurrentPos != newPos {
		t.Errorf("CurrentPos = %v, want %v", s.CurrentPos, newPos)
	}
	if s.Modifiers != input.ModCtrl {
		t.Errorf("Modifiers = %d, want ModCtrl", s.Modifiers)
	}
	if s.Operation != input.DragOperationCopy {
		t.Errorf("Operation = %d, want DragOperationCopy (Ctrl held)", s.Operation)
	}
}

func TestDnDManagerUpdateDragInactive(t *testing.T) {
	m := NewDnDManager()
	ok := m.UpdateDrag(input.GesturePoint{}, 0)
	if ok {
		t.Error("UpdateDrag should return false when inactive")
	}
}

func TestDnDManagerCancelDrag(t *testing.T) {
	m := NewDnDManager()
	m.StartDrag(1, testDragData(), input.GesturePoint{}, draw.Rect{}, nil, draw.Point{}, false)
	m.CancelDrag()

	if m.IsActive() {
		t.Error("should not be active after CancelDrag")
	}
	if m.Session() != nil {
		t.Error("session should be nil after CancelDrag")
	}
}

func TestDnDManagerDropZoneRegistration(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})
	m.RegisterDropZone(DropZone{
		UID:    200,
		Bounds: draw.R(300, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	if m.DropZoneCount() != 2 {
		t.Fatalf("DropZoneCount = %d, want 2", m.DropZoneCount())
	}

	m.ResetDropZones()
	if m.DropZoneCount() != 0 {
		t.Fatalf("DropZoneCount = %d after reset, want 0", m.DropZoneCount())
	}
}

func TestDnDManagerHitTestDropZone(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})
	m.RegisterDropZone(DropZone{
		UID:    200,
		Bounds: draw.R(300, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 50, Y: 50}, draw.Rect{}, nil, draw.Point{}, false)

	// Over first zone.
	m.UpdateDrag(input.GesturePoint{X: 100, Y: 100}, 0)
	if m.HoveredZoneUID() != 100 {
		t.Errorf("HoveredZoneUID = %d, want 100", m.HoveredZoneUID())
	}

	// Over second zone.
	m.UpdateDrag(input.GesturePoint{X: 400, Y: 100}, 0)
	if m.HoveredZoneUID() != 200 {
		t.Errorf("HoveredZoneUID = %d, want 200", m.HoveredZoneUID())
	}

	// Over no zone.
	m.UpdateDrag(input.GesturePoint{X: 250, Y: 100}, 0)
	if m.HoveredZoneUID() != 0 {
		t.Errorf("HoveredZoneUID = %d, want 0 (no zone)", m.HoveredZoneUID())
	}
}

func TestDnDManagerNestedDropZonePriority(t *testing.T) {
	m := NewDnDManager()
	// Outer zone (low priority).
	m.RegisterDropZone(DropZone{
		UID:      100,
		Bounds:   draw.R(0, 0, 400, 400),
		Accept:   func(d *input.DragData, op input.DragOperation) bool { return true },
		Priority: 0,
	})
	// Inner zone (high priority).
	m.RegisterDropZone(DropZone{
		UID:      200,
		Bounds:   draw.R(100, 100, 100, 100),
		Accept:   func(d *input.DragData, op input.DragOperation) bool { return true },
		Priority: 1,
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 50, Y: 50}, draw.Rect{}, nil, draw.Point{}, false)

	// Point inside both zones → inner wins (higher priority).
	m.UpdateDrag(input.GesturePoint{X: 150, Y: 150}, 0)
	if m.HoveredZoneUID() != 200 {
		t.Errorf("HoveredZoneUID = %d, want 200 (inner/higher priority)", m.HoveredZoneUID())
	}

	// Point inside outer only.
	m.UpdateDrag(input.GesturePoint{X: 50, Y: 50}, 0)
	if m.HoveredZoneUID() != 100 {
		t.Errorf("HoveredZoneUID = %d, want 100 (outer)", m.HoveredZoneUID())
	}
}

func TestDnDManagerSmallestAreaTiebreak(t *testing.T) {
	m := NewDnDManager()
	// Two zones with same priority, overlapping.
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 400, 400),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})
	m.RegisterDropZone(DropZone{
		UID:    200,
		Bounds: draw.R(50, 50, 100, 100),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 10, Y: 10}, draw.Rect{}, nil, draw.Point{}, false)

	// Point inside both → smaller area wins.
	m.UpdateDrag(input.GesturePoint{X: 75, Y: 75}, 0)
	if m.HoveredZoneUID() != 200 {
		t.Errorf("HoveredZoneUID = %d, want 200 (smaller area)", m.HoveredZoneUID())
	}
}

func TestDnDManagerEndDragWithDrop(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEText)
		},
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 300, Y: 300}, draw.Rect{}, nil, draw.Point{}, false)

	// End drag over drop zone.
	effect := m.EndDrag(input.GesturePoint{X: 100, Y: 100}, 0)
	if effect != input.DropEffectMove {
		t.Errorf("EndDrag effect = %v, want DropEffectMove", effect)
	}
	if m.IsActive() {
		t.Error("should not be active after EndDrag")
	}
}

func TestDnDManagerEndDragNoTarget(t *testing.T) {
	m := NewDnDManager()
	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 100, Y: 100}, draw.Rect{}, nil, draw.Point{}, false)

	effect := m.EndDrag(input.GesturePoint{X: 500, Y: 500}, 0)
	if effect != input.DropEffectNone {
		t.Errorf("EndDrag effect = %v, want DropEffectNone", effect)
	}
}

func TestDnDManagerEndDragRejected(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return false // always reject
		},
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 300, Y: 300}, draw.Rect{}, nil, draw.Point{}, false)

	effect := m.EndDrag(input.GesturePoint{X: 100, Y: 100}, 0)
	if effect != input.DropEffectNone {
		t.Errorf("EndDrag effect = %v, want DropEffectNone (rejected)", effect)
	}
}

func TestDnDManagerEndDragCopy(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	data := input.NewTextDragData("test")
	m.StartDrag(1, data, input.GesturePoint{X: 300, Y: 300}, draw.Rect{}, nil, draw.Point{}, false)

	// Ctrl held → Copy.
	effect := m.EndDrag(input.GesturePoint{X: 100, Y: 100}, input.ModCtrl)
	if effect != input.DropEffectCopy {
		t.Errorf("EndDrag effect = %v, want DropEffectCopy (Ctrl held)", effect)
	}
}

func TestDnDManagerDispatchEvents(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})
	m.RegisterDropZone(DropZone{
		UID:    200,
		Bounds: draw.R(300, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 600, Y: 600}, draw.Rect{}, nil, draw.Point{}, false)

	type event struct {
		uid  UID
		kind InputEventKind
	}
	var events []event
	append := func(uid UID, ev InputEvent) {
		events = append(events, event{uid, ev.Kind})
	}

	// Move over zone 100.
	m.UpdateDrag(input.GesturePoint{X: 100, Y: 100}, 0)
	m.DispatchDnDEvents(append)
	if len(events) != 2 {
		t.Fatalf("expected 2 events (Enter + Over), got %d", len(events))
	}
	if events[0].kind != EventDragEnter || events[0].uid != 100 {
		t.Errorf("event[0] = (%d, %d), want (100, EventDragEnter)", events[0].uid, events[0].kind)
	}
	if events[1].kind != EventDragOver || events[1].uid != 100 {
		t.Errorf("event[1] = (%d, %d), want (100, EventDragOver)", events[1].uid, events[1].kind)
	}

	// Stay over zone 100 → only Over.
	events = events[:0]
	m.UpdateDrag(input.GesturePoint{X: 110, Y: 110}, 0)
	m.DispatchDnDEvents(append)
	if len(events) != 1 {
		t.Fatalf("expected 1 event (Over), got %d", len(events))
	}
	if events[0].kind != EventDragOver {
		t.Errorf("event[0].kind = %d, want EventDragOver", events[0].kind)
	}

	// Move to zone 200 → Leave(100) + Enter(200) + Over(200).
	events = events[:0]
	m.UpdateDrag(input.GesturePoint{X: 400, Y: 100}, 0)
	m.DispatchDnDEvents(append)
	if len(events) != 3 {
		t.Fatalf("expected 3 events (Leave + Enter + Over), got %d", len(events))
	}
	if events[0].kind != EventDragLeave || events[0].uid != 100 {
		t.Errorf("event[0] = (%d, %d), want (100, EventDragLeave)", events[0].uid, events[0].kind)
	}
	if events[1].kind != EventDragEnter || events[1].uid != 200 {
		t.Errorf("event[1] = (%d, %d), want (200, EventDragEnter)", events[1].uid, events[1].kind)
	}
	if events[2].kind != EventDragOver || events[2].uid != 200 {
		t.Errorf("event[2] = (%d, %d), want (200, EventDragOver)", events[2].uid, events[2].kind)
	}

	// Move outside all zones → Leave(200).
	events = events[:0]
	m.UpdateDrag(input.GesturePoint{X: 600, Y: 600}, 0)
	m.DispatchDnDEvents(append)
	if len(events) != 1 {
		t.Fatalf("expected 1 event (Leave), got %d", len(events))
	}
	if events[0].kind != EventDragLeave || events[0].uid != 200 {
		t.Errorf("event[0] = (%d, %d), want (200, EventDragLeave)", events[0].uid, events[0].kind)
	}
}

func TestDnDManagerHoveredZoneAccepts(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEJSON) // only accepts JSON
		},
	})

	// Drag text data (not JSON).
	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 300, Y: 300}, draw.Rect{}, nil, draw.Point{}, false)
	m.UpdateDrag(input.GesturePoint{X: 100, Y: 100}, 0)

	if m.HoveredZoneAccepts() {
		t.Error("zone should NOT accept text/plain data when it only accepts JSON")
	}
}

func TestDnDManagerDragCursor(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})
	m.RegisterDropZone(DropZone{
		UID:    200,
		Bounds: draw.R(300, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return false },
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 600, Y: 600}, draw.Rect{}, nil, draw.Point{}, false)

	// No zone → CursorGrabbing.
	m.UpdateDrag(input.GesturePoint{X: 600, Y: 600}, 0)
	if c := m.DragCursor(); c != input.CursorGrabbing {
		t.Errorf("cursor = %d, want CursorGrabbing (no zone)", c)
	}

	// Accepting zone → CursorMove.
	m.UpdateDrag(input.GesturePoint{X: 100, Y: 100}, 0)
	if c := m.DragCursor(); c != input.CursorMove {
		t.Errorf("cursor = %d, want CursorMove (accepting zone)", c)
	}

	// Rejecting zone → CursorNotAllowed.
	m.UpdateDrag(input.GesturePoint{X: 400, Y: 100}, 0)
	if c := m.DragCursor(); c != input.CursorNotAllowed {
		t.Errorf("cursor = %d, want CursorNotAllowed (rejecting zone)", c)
	}
}

func TestDnDManagerDropZoneUIDs(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{UID: 10, Bounds: draw.R(0, 0, 50, 50)})
	m.RegisterDropZone(DropZone{UID: 20, Bounds: draw.R(60, 0, 50, 50)})
	m.RegisterDropZone(DropZone{UID: 30, Bounds: draw.R(120, 0, 50, 50)})

	uids := m.DropZoneUIDs()
	if len(uids) != 3 {
		t.Fatalf("DropZoneUIDs length = %d, want 3", len(uids))
	}
	if uids[0] != 10 || uids[1] != 20 || uids[2] != 30 {
		t.Errorf("DropZoneUIDs = %v, want [10 20 30]", uids)
	}
}

func TestDnDManagerIsDropHovered(t *testing.T) {
	m := NewDnDManager()
	m.RegisterDropZone(DropZone{
		UID:    100,
		Bounds: draw.R(0, 0, 200, 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool { return true },
	})

	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 300, Y: 300}, draw.Rect{}, nil, draw.Point{}, false)
	m.UpdateDrag(input.GesturePoint{X: 100, Y: 100}, 0)

	if !m.IsDropHovered(100) {
		t.Error("IsDropHovered(100) should be true")
	}
	if m.IsDropHovered(200) {
		t.Error("IsDropHovered(200) should be false")
	}
}

func TestDnDManagerResetDropZonesPreservesSession(t *testing.T) {
	m := NewDnDManager()
	data := testDragData()
	m.StartDrag(1, data, input.GesturePoint{X: 100, Y: 100}, draw.Rect{}, nil, draw.Point{}, false)

	m.RegisterDropZone(DropZone{UID: 100, Bounds: draw.R(0, 0, 200, 200)})
	m.ResetDropZones()

	// Session should still be active after ResetDropZones.
	if !m.IsActive() {
		t.Error("session should remain active after ResetDropZones")
	}
	if m.DropZoneCount() != 0 {
		t.Errorf("zones should be cleared, got %d", m.DropZoneCount())
	}
}
