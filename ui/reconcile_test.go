package ui

import (
	"testing"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

// ── Test helpers ─────────────────────────────────────────────────

// counterWidget is a stateful widget that counts how many times Render is called.
type counterWidget struct{}

type counterState struct {
	RenderCount int
}

func (counterWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[counterState](raw)
	s.RenderCount++
	return Text("counter"), s
}

// greetWidget returns a greeting whose content depends on state.
type greetWidget struct{ Name string }

type greetState struct {
	Greeted bool
}

func (w greetWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[greetState](raw)
	if !s.Greeted {
		s.Greeted = true
		return Text("hello " + w.Name), s
	}
	return Text("welcome back " + w.Name), s
}

func noopSend(_ any) {}

// ── UID generation ───────────────────────────────────────────────

func TestMakeUIDDeterministic(t *testing.T) {
	a := MakeUID(0, "key", 0)
	b := MakeUID(0, "key", 0)
	if a != b {
		t.Errorf("same inputs should produce same UID: %d != %d", a, b)
	}
}

func TestMakeUIDDiffersForDifferentKeys(t *testing.T) {
	a := MakeUID(0, "alpha", 0)
	b := MakeUID(0, "beta", 0)
	if a == b {
		t.Error("different keys should produce different UIDs")
	}
}

func TestMakeUIDDiffersForDifferentIndices(t *testing.T) {
	a := MakeUID(0, "", 0)
	b := MakeUID(0, "", 1)
	if a == b {
		t.Error("different indices should produce different UIDs")
	}
}

func TestMakeUIDKeyOverridesIndex(t *testing.T) {
	a := MakeUID(0, "x", 0)
	b := MakeUID(0, "x", 99)
	if a != b {
		t.Error("key should override index: same key at different indices should produce same UID")
	}
}

// ── Reconciler: state persistence ────────────────────────────────

func TestReconcileWidgetStatePreserved(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Component(counterWidget{})

	// First reconciliation — state initialised.
	resolved, changed := r.Reconcile(tree, th, noopSend, nil, nil, "")
	if !changed {
		t.Error("first reconcile should report changed")
	}
	if resolved == nil {
		t.Fatal("resolved tree should not be nil")
	}
	if r.StateCount() != 1 {
		t.Fatalf("expected 1 state entry, got %d", r.StateCount())
	}

	// Second reconciliation — state carried forward.
	_, _ = r.Reconcile(tree, th, noopSend, nil, nil, "")
	if r.StateCount() != 1 {
		t.Fatalf("expected 1 state entry after 2nd reconcile, got %d", r.StateCount())
	}

	// The counter should have been incremented twice.
	uid := MakeUID(0, "__widget__", 0)
	raw := r.StateFor(uid)
	s, ok := raw.(*counterState)
	if !ok {
		t.Fatalf("expected *counterState, got %T", raw)
	}
	if s.RenderCount != 2 {
		t.Errorf("RenderCount = %d, want 2", s.RenderCount)
	}
}

func TestReconcileWidgetStateWithStableKey(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	// Two widgets with explicit keys.
	tree := Column(
		ComponentWithKey("a", counterWidget{}),
		ComponentWithKey("b", counterWidget{}),
	)

	r.Reconcile(tree, th, noopSend, nil, nil, "")
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	if r.StateCount() != 2 {
		t.Fatalf("expected 2 states, got %d", r.StateCount())
	}
}

func TestReconcileStateResetOnKeyChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree1 := ComponentWithKey("v1", counterWidget{})
	r.Reconcile(tree1, th, noopSend, nil, nil, "")
	r.Reconcile(tree1, th, noopSend, nil, nil, "") // RenderCount = 2

	// Change the key — should get fresh state.
	tree2 := ComponentWithKey("v2", counterWidget{})
	r.Reconcile(tree2, th, noopSend, nil, nil, "")

	uid := MakeUID(0, "v2", 0)
	raw := r.StateFor(uid)
	s, ok := raw.(*counterState)
	if !ok {
		t.Fatalf("expected *counterState, got %T", raw)
	}
	if s.RenderCount != 1 {
		t.Errorf("RenderCount after key change = %d, want 1 (fresh state)", s.RenderCount)
	}

	// Old key's state should be purged.
	oldUID := MakeUID(0, "v1", 0)
	if r.StateFor(oldUID) != nil {
		t.Error("old key's state should have been purged")
	}
}

// ── Reconciler: dirty tracking ───────────────────────────────────

func TestReconcileNoChangeReturnsFalse(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Column(Text("hello"), ButtonText("ok", nil))

	// First call — always changed.
	_, changed := r.Reconcile(tree, th, noopSend, nil, nil, "")
	if !changed {
		t.Error("first reconcile should be changed")
	}

	// Same tree again — no change.
	_, changed = r.Reconcile(tree, th, noopSend, nil, nil, "")
	if changed {
		t.Error("identical tree should not report changed")
	}
}

func TestReconcileDetectsTextChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Text("v1"), th, noopSend, nil, nil, "")
	_, changed := r.Reconcile(Text("v2"), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("different text content should report changed")
	}
}

func TestReconcileDetectsStructuralChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Column(Text("a")), th, noopSend, nil, nil, "")
	_, changed := r.Reconcile(Column(Text("a"), Text("b")), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("adding a child should report changed")
	}
}

// ── Reconciler: state cleanup ────────────────────────────────────

func TestReconcileRemovesOrphanedState(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Column(
		ComponentWithKey("keep", counterWidget{}),
		ComponentWithKey("drop", counterWidget{}),
	)
	r.Reconcile(tree, th, noopSend, nil, nil, "")
	if r.StateCount() != 2 {
		t.Fatalf("expected 2 states, got %d", r.StateCount())
	}

	// Remove one widget.
	tree2 := Column(
		ComponentWithKey("keep", counterWidget{}),
	)
	r.Reconcile(tree2, th, noopSend, nil, nil, "")
	if r.StateCount() != 1 {
		t.Errorf("expected 1 state after removal, got %d", r.StateCount())
	}
}

// ── Reconciler: Widget.Render integration ────────────────────────

func TestReconcileExpandsWidgetToElement(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Component(greetWidget{Name: "lux"})

	resolved, _ := r.Reconcile(tree, th, noopSend, nil, nil, "")
	// Resolved widgets are wrapped in widgetBoundsElement.
	wb, ok := resolved.(widgetBoundsElement)
	if !ok {
		t.Fatalf("expected widgetBoundsElement, got %T", resolved)
	}
	te, ok := wb.Child.(textElement)
	if !ok {
		t.Fatalf("expected textElement child, got %T", wb.Child)
	}
	if te.Content != "hello lux" {
		t.Errorf("content = %q, want %q", te.Content, "hello lux")
	}

	// Second call — state was persisted, so greeting changes.
	resolved, _ = r.Reconcile(tree, th, noopSend, nil, nil, "")
	wb, ok = resolved.(widgetBoundsElement)
	if !ok {
		t.Fatalf("expected widgetBoundsElement, got %T", resolved)
	}
	te, ok = wb.Child.(textElement)
	if !ok {
		t.Fatalf("expected textElement child, got %T", wb.Child)
	}
	if te.Content != "welcome back lux" {
		t.Errorf("content = %q, want %q", te.Content, "welcome back lux")
	}
}

// ── treeEqual ────────────────────────────────────────────────────

func TestTreeEqualTier2Elements(t *testing.T) {
	tests := []struct {
		name  string
		a, b  Element
		equal bool
	}{
		{"checkbox same", Checkbox("x", true, nil), Checkbox("x", true, nil), true},
		{"checkbox differ", Checkbox("x", true, nil), Checkbox("x", false, nil), false},
		{"toggle same", Toggle(true, nil), Toggle(true, nil), true},
		{"toggle differ", Toggle(true, nil), Toggle(false, nil), false},
		{"slider same", Slider(0.5, nil), Slider(0.5, nil), true},
		{"slider differ", Slider(0.5, nil), Slider(0.7, nil), false},
		{"divider", Divider(), Divider(), true},
		{"spacer same", Spacer(10), Spacer(10), true},
		{"spacer differ", Spacer(10), Spacer(20), false},
		{"icon same", Icon("★"), Icon("★"), true},
		{"icon differ", Icon("★"), Icon("→"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := treeEqual(tc.a, tc.b)
			if got != tc.equal {
				t.Errorf("treeEqual = %v, want %v", got, tc.equal)
			}
		})
	}
}

func TestTreeEqualNested(t *testing.T) {
	a := Column(Row(Text("x")), ButtonText("ok", nil))
	b := Column(Row(Text("x")), ButtonText("ok", nil))
	if !treeEqual(a, b) {
		t.Error("structurally identical nested trees should be equal")
	}

	c := Column(Row(Text("y")), ButtonText("ok", nil))
	if treeEqual(a, c) {
		t.Error("trees with different text should not be equal")
	}
}

// ── Event dispatch integration ──────────────────────────────────

// eventCapture is a widget that records which events it received.
type eventCapture struct{}
type eventCaptureState struct {
	Events []InputEvent
}

func (eventCapture) Render(ctx RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[eventCaptureState](raw)
	s.Events = ctx.Events
	return Text("capture"), s
}

func TestReconcileDeliversEventsToWidget(t *testing.T) {
	r := NewReconciler()
	th := theme.Default
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	tree := ComponentWithKey("cap", eventCapture{})
	uid := MakeUID(0, "cap", 0)

	// First reconcile to establish the widget.
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	// Set focus and collect events.
	fm.SetFocusedUID(uid)
	d.Collect(input.KeyMsg{Key: input.KeyA, Action: input.KeyPress})
	d.Dispatch()

	// Reconcile with dispatcher — widget should receive events.
	r.Reconcile(tree, th, noopSend, d, fm, "")

	state := r.StateFor(uid)
	s, ok := state.(*eventCaptureState)
	if !ok {
		t.Fatalf("expected *eventCaptureState, got %T", state)
	}
	if len(s.Events) != 1 {
		t.Fatalf("widget should receive 1 event, got %d", len(s.Events))
	}
	if s.Events[0].Kind != EventKey {
		t.Errorf("event kind = %d, want EventKey", s.Events[0].Kind)
	}
}

// focusableWidget implements the Focusable interface.
type focusableWidget struct{}
type focusableState struct{}

func (focusableWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[focusableState](raw)
	return Text("focusable"), s
}

func (focusableWidget) FocusOptions() FocusOpts {
	return FocusOpts{Focusable: true, TabIndex: 0}
}

func TestReconcileRegistersFocusableWidgets(t *testing.T) {
	r := NewReconciler()
	th := theme.Default
	fm := NewFocusManager()

	tree := Column(
		ComponentWithKey("a", focusableWidget{}),
		ComponentWithKey("b", focusableWidget{}),
	)

	r.Reconcile(tree, th, noopSend, nil, fm, "")

	if fm.OrderLen() != 2 {
		t.Errorf("expected 2 focusable widgets registered, got %d", fm.OrderLen())
	}

	// Tab navigation should work.
	uid := fm.FocusNext()
	expected := MakeUID(0, "a", 0)
	if uid != expected {
		t.Errorf("first FocusNext = %d, want %d (widget 'a')", uid, expected)
	}
}

// ── Equatable short-circuit ───────────────────────────────────────

// eqWidget implements Equatable — Equal compares the Label field.
type eqWidget struct{ Label string }

func (w eqWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[counterState](raw)
	s.RenderCount++
	return Text(w.Label), s
}

func (w eqWidget) Equal(other Widget) bool {
	o := other.(eqWidget)
	return w.Label == o.Label
}

func TestEquatableSkipsRenderWhenEqual(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("eq", eqWidget{Label: "hello"})

	// First reconcile — always renders.
	r.Reconcile(tree, th, noopSend, nil, nil, "")
	uid := MakeUID(0, "eq", 0)
	s := r.StateFor(uid).(*counterState)
	if s.RenderCount != 1 {
		t.Fatalf("RenderCount after 1st reconcile = %d, want 1", s.RenderCount)
	}

	// Second reconcile with same Label — Equatable should skip Render.
	r.Reconcile(tree, th, noopSend, nil, nil, "")
	s = r.StateFor(uid).(*counterState)
	if s.RenderCount != 1 {
		t.Errorf("RenderCount after equal reconcile = %d, want 1 (should skip)", s.RenderCount)
	}
}

func TestEquatableReRendersWhenNotEqual(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(ComponentWithKey("eq", eqWidget{Label: "v1"}), th, noopSend, nil, nil, "")
	uid := MakeUID(0, "eq", 0)

	// Change label — Equatable.Equal should return false, Render called again.
	r.Reconcile(ComponentWithKey("eq", eqWidget{Label: "v2"}), th, noopSend, nil, nil, "")
	s := r.StateFor(uid).(*counterState)
	if s.RenderCount != 2 {
		t.Errorf("RenderCount after changed props = %d, want 2", s.RenderCount)
	}
}

// ── DirtyTracker ─────────────────────────────────────────────────

type dirtyState struct {
	dirty bool
}

func (d *dirtyState) IsDirty() bool  { return d.dirty }
func (d *dirtyState) ClearDirty()    { d.dirty = false }

// dirtyWidget produces a state that implements DirtyTracker.
type dirtyWidget struct{}

func (dirtyWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[dirtyState](raw)
	return Text("dirty"), s
}

func TestCheckDirtyTrackersReturnsTrueAndClears(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("d", dirtyWidget{})
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	uid := MakeUID(0, "d", 0)
	s := r.StateFor(uid).(*dirtyState)
	s.dirty = true

	if !r.CheckDirtyTrackers() {
		t.Error("CheckDirtyTrackers should return true when a state is dirty")
	}
	if s.dirty {
		t.Error("ClearDirty should have been called")
	}

	// Second call — already cleared.
	if r.CheckDirtyTrackers() {
		t.Error("CheckDirtyTrackers should return false after clearing")
	}
}

func TestCheckDirtyTrackersReturnsFalseWhenClean(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("d", dirtyWidget{})
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	if r.CheckDirtyTrackers() {
		t.Error("CheckDirtyTrackers should return false when no state is dirty")
	}
}

func TestTreeEqualWidgetBoundsElement(t *testing.T) {
	a := widgetBoundsElement{WidgetUID: 42, Child: Text("hello")}
	b := widgetBoundsElement{WidgetUID: 42, Child: Text("hello")}
	if !treeEqual(a, b) {
		t.Error("identical widgetBoundsElements should be equal")
	}

	c := widgetBoundsElement{WidgetUID: 42, Child: Text("world")}
	if treeEqual(a, c) {
		t.Error("widgetBoundsElements with different children should not be equal")
	}

	d := widgetBoundsElement{WidgetUID: 99, Child: Text("hello")}
	if treeEqual(a, d) {
		t.Error("widgetBoundsElements with different UIDs should not be equal")
	}
}
