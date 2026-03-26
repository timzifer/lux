package ui

import (
	"testing"
	"time"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
)

// ── Nested widget resolution ──────────────────────────────────────

// outerWidget renders an innerWidget, testing widget-inside-widget resolution.
type outerWidget struct{ Name string }
type outerState struct{ Expanded bool }

func (w outerWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[outerState](raw)
	return Component(innerWidget{Name: w.Name}), s
}

type innerWidget struct{ Name string }
type innerState struct{ Count int }

func (w innerWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[innerState](raw)
	s.Count++
	return Text("inner:" + w.Name), s
}

func TestReconcileNestedWidgets(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("outer", outerWidget{Name: "deep"})

	// First reconcile — both outer and inner should be resolved.
	resolved, changed := r.Reconcile(tree, th, noopSend, nil, nil, "")
	if !changed {
		t.Error("first reconcile should report changed")
	}

	// Resolved tree should be: WidgetBounds(outer) → WidgetBounds(inner) → Text
	wb, ok := resolved.(WidgetBoundsElement)
	if !ok {
		t.Fatalf("expected WidgetBoundsElement at root, got %T", resolved)
	}
	wb2, ok := wb.Child.(WidgetBoundsElement)
	if !ok {
		t.Fatalf("expected nested WidgetBoundsElement, got %T", wb.Child)
	}
	te, ok := wb2.Child.(TextElement)
	if !ok {
		t.Fatalf("expected TextElement, got %T", wb2.Child)
	}
	if te.Content != "inner:deep" {
		t.Errorf("content = %q, want %q", te.Content, "inner:deep")
	}

	// Both widgets should have state.
	if r.StateCount() != 2 {
		t.Errorf("expected 2 states (outer + inner), got %d", r.StateCount())
	}

	// Second reconcile — inner state persists.
	r.Reconcile(tree, th, noopSend, nil, nil, "")
	if r.StateCount() != 2 {
		t.Errorf("expected 2 states after 2nd reconcile, got %d", r.StateCount())
	}
}

// ── Widget reordering with keys ──────────────────────────────────

func TestReconcileWidgetReorderingPreservesState(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	// Render A, B in order.
	tree1 := Column(
		ComponentWithKey("A", counterWidget{}),
		ComponentWithKey("B", counterWidget{}),
	)
	r.Reconcile(tree1, th, noopSend, nil, nil, "")
	r.Reconcile(tree1, th, noopSend, nil, nil, "")

	uidA := MakeUID(0, "A", 0)
	uidB := MakeUID(0, "B", 0)
	sA := r.StateFor(uidA).(*counterState)
	sB := r.StateFor(uidB).(*counterState)
	if sA.RenderCount != 2 || sB.RenderCount != 2 {
		t.Fatalf("A.Count=%d, B.Count=%d, both should be 2", sA.RenderCount, sB.RenderCount)
	}

	// Swap order to B, A — keys ensure state is preserved.
	tree2 := Column(
		ComponentWithKey("B", counterWidget{}),
		ComponentWithKey("A", counterWidget{}),
	)
	r.Reconcile(tree2, th, noopSend, nil, nil, "")

	sA = r.StateFor(uidA).(*counterState)
	sB = r.StateFor(uidB).(*counterState)
	if sA.RenderCount != 3 {
		t.Errorf("A.RenderCount = %d, want 3 (persisted through reorder)", sA.RenderCount)
	}
	if sB.RenderCount != 3 {
		t.Errorf("B.RenderCount = %d, want 3 (persisted through reorder)", sB.RenderCount)
	}
}

// ── Multiple Equatable widgets (mixed equal/not-equal) ──────────

func TestEquatableMixedSkipAndRerender(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree1 := Column(
		ComponentWithKey("x", eqWidget{Label: "same"}),
		ComponentWithKey("y", eqWidget{Label: "v1"}),
	)
	r.Reconcile(tree1, th, noopSend, nil, nil, "")

	// Change only "y" label — "x" should skip Render.
	tree2 := Column(
		ComponentWithKey("x", eqWidget{Label: "same"}),
		ComponentWithKey("y", eqWidget{Label: "v2"}),
	)
	r.Reconcile(tree2, th, noopSend, nil, nil, "")

	uidX := MakeUID(0, "x", 0)
	uidY := MakeUID(0, "y", 0)
	sX := r.StateFor(uidX).(*counterState)
	sY := r.StateFor(uidY).(*counterState)

	if sX.RenderCount != 1 {
		t.Errorf("x.RenderCount = %d, want 1 (should have been skipped)", sX.RenderCount)
	}
	if sY.RenderCount != 2 {
		t.Errorf("y.RenderCount = %d, want 2 (props changed)", sY.RenderCount)
	}
}

// ── Equatable skip still allows DirtyTracker ────────────────────

// eqDirtyWidget combines Equatable (on widget) and DirtyTracker (on state).
type eqDirtyWidget struct{ Label string }
type eqDirtyState struct {
	renderCount int
	dirty       bool
}

func (s *eqDirtyState) IsDirty() bool { return s.dirty }
func (s *eqDirtyState) ClearDirty()   { s.dirty = false }

func (w eqDirtyWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[eqDirtyState](raw)
	s.renderCount++
	return Text(w.Label), s
}

func (w eqDirtyWidget) Equal(other Widget) bool {
	o := other.(eqDirtyWidget)
	return w.Label == o.Label
}

func TestEquatableSkipDoesNotBlockDirtyTracker(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("ed", eqDirtyWidget{Label: "hello"})
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	uid := MakeUID(0, "ed", 0)
	s := r.StateFor(uid).(*eqDirtyState)
	s.dirty = true

	// Equatable skip (same label), but DirtyTracker should still detect dirty.
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	if !r.CheckDirtyTrackers() {
		// State was marked dirty before reconcile. Since Equatable reused the
		// previous output, Render didn't run and s.dirty was preserved.
		// CheckDirtyTrackers should still find it.
		t.Error("CheckDirtyTrackers should detect dirty state even after Equatable skip")
	}
}

// ── Animator + DirtyTracker on same state ──────────────────────

type animDirtyState struct {
	pos    float32
	target float32
	dirty  bool
}

func (s *animDirtyState) Tick(dt time.Duration) bool {
	s.pos += float32(dt.Seconds()) * 100
	if s.pos >= s.target {
		s.pos = s.target
		return false
	}
	return true
}

func (s *animDirtyState) IsDirty() bool { return s.dirty }
func (s *animDirtyState) ClearDirty()   { s.dirty = false }

type animDirtyWidget struct{}

func (animDirtyWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[animDirtyState](raw)
	if s.target == 0 {
		s.target = 100
	}
	return Text("animdirty"), s
}

func TestAnimatorAndDirtyTrackerOnSameState(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("ad", animDirtyWidget{})
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	uid := MakeUID(0, "ad", 0)
	s := r.StateFor(uid).(*animDirtyState)

	// Mark dirty and tick animation.
	s.dirty = true
	running := r.TickAnimators(100 * time.Millisecond)
	dirty := r.CheckDirtyTrackers()

	if !running {
		t.Error("animation should still be running after 100ms")
	}
	if !dirty {
		t.Error("dirty tracker should report dirty")
	}
	if s.dirty {
		t.Error("dirty flag should have been cleared")
	}
}

// ── Empty/nil tree reconciliation ───────────────────────────────

func TestReconcileEmptyTree(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	resolved, changed := r.Reconcile(Empty(), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("first reconcile should report changed")
	}
	if resolved == nil {
		t.Error("resolved tree should not be nil for Empty()")
	}
}

func TestReconcileNilTree(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	// nil → nil: prevTree starts as nil, so treeEqual(nil, nil) = true → changed=false.
	resolved, changed := r.Reconcile(nil, th, noopSend, nil, nil, "")
	if changed {
		t.Error("nil → nil should not report changed")
	}
	if resolved != nil {
		t.Errorf("resolved should be nil for nil input, got %T", resolved)
	}

	// nil → Text: should report changed.
	_, changed = r.Reconcile(Text("hello"), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("nil → Text should report changed")
	}
}

// ── State replaced by different widget at same key ─────────────

// alphaWidget and betaWidget share the same key but have incompatible states.
type alphaWidget struct{}
type alphaState struct{ Alpha int }

func (alphaWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[alphaState](raw)
	s.Alpha++
	return Text("alpha"), s
}

type betaWidget struct{}
type betaState struct{ Beta string }

func (betaWidget) Render(_ RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[betaState](raw)
	s.Beta += "b"
	return Text("beta"), s
}

func TestReconcileWidgetReplacementAtSameKey(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	// First frame with alphaWidget.
	tree1 := ComponentWithKey("slot", alphaWidget{})
	r.Reconcile(tree1, th, noopSend, nil, nil, "")

	uid := MakeUID(0, "slot", 0)
	s1 := r.StateFor(uid)
	if _, ok := s1.(*alphaState); !ok {
		t.Fatalf("expected *alphaState, got %T", s1)
	}

	// Second frame with betaWidget at same key — state should be fresh.
	tree2 := ComponentWithKey("slot", betaWidget{})
	r.Reconcile(tree2, th, noopSend, nil, nil, "")

	s2 := r.StateFor(uid)
	bs, ok := s2.(*betaState)
	if !ok {
		t.Fatalf("expected *betaState after replacement, got %T", s2)
	}
	if bs.Beta != "b" {
		t.Errorf("Beta = %q, want %q (fresh state)", bs.Beta, "b")
	}
}

// ── Deep orphan cleanup ─────────────────────────────────────────

func TestReconcileDeepOrphanCleanup(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	// Tree with deeply nested widget.
	tree1 := Column(
		ComponentWithKey("top", outerWidget{Name: "deep"}),
	)
	r.Reconcile(tree1, th, noopSend, nil, nil, "")

	// outerWidget + innerWidget = 2 states.
	if r.StateCount() != 2 {
		t.Fatalf("expected 2 states for nested widgets, got %d", r.StateCount())
	}

	// Remove the entire subtree.
	tree2 := Column(Text("simple"))
	r.Reconcile(tree2, th, noopSend, nil, nil, "")

	if r.StateCount() != 0 {
		t.Errorf("expected 0 states after removing all widgets, got %d", r.StateCount())
	}
}

// ── ThemedElement with widgets ─────────────────────────────────

func TestReconcileThemedSubtree(t *testing.T) {
	r := NewReconciler()

	tree := ThemedElement{
		Theme: theme.LuxLight,
		Children: []Element{
			ComponentWithKey("themed", counterWidget{}),
		},
	}

	r.Reconcile(tree, theme.Default, noopSend, nil, nil, "")
	r.Reconcile(tree, theme.Default, noopSend, nil, nil, "")

	if r.StateCount() != 1 {
		t.Errorf("expected 1 state for themed widget, got %d", r.StateCount())
	}
}

// ── UID collision resistance ────────────────────────────────────

func TestMakeUIDDifferentParentsProduceDifferentUIDs(t *testing.T) {
	a := MakeUID(100, "child", 0)
	b := MakeUID(200, "child", 0)
	if a == b {
		t.Error("same key under different parents should produce different UIDs")
	}
}

func TestMakeUIDEmptyKeyDifferentFromNonEmpty(t *testing.T) {
	a := MakeUID(0, "", 0)
	b := MakeUID(0, "explicit", 0)
	if a == b {
		t.Error("empty key (index-based) should differ from explicit key")
	}
}

// ── Reconcile with locale change ────────────────────────────────

type localeWidget struct{}
type localeState struct{ Locale string }

func (localeWidget) Render(ctx RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[localeState](raw)
	s.Locale = ctx.Locale
	return Text("locale:" + ctx.Locale), s
}

func TestReconcilePassesLocaleToWidget(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := ComponentWithKey("loc", localeWidget{})

	r.Reconcile(tree, th, noopSend, nil, nil, "de")
	uid := MakeUID(0, "loc", 0)
	s := r.StateFor(uid).(*localeState)
	if s.Locale != "de" {
		t.Errorf("Locale = %q, want %q", s.Locale, "de")
	}

	r.Reconcile(tree, th, noopSend, nil, nil, "en")
	s = r.StateFor(uid).(*localeState)
	if s.Locale != "en" {
		t.Errorf("Locale after change = %q, want %q", s.Locale, "en")
	}
}

// ── Send function integration ───────────────────────────────────

type sendWidget struct{}
type sendState struct{ Sent bool }

func (sendWidget) Render(ctx RenderCtx, raw WidgetState) (Element, WidgetState) {
	s := AdoptState[sendState](raw)
	if !s.Sent {
		ctx.Send("msg")
		s.Sent = true
	}
	return Text("send"), s
}

func TestReconcileWidgetCanCallSend(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	var received []any
	send := func(msg any) { received = append(received, msg) }

	tree := ComponentWithKey("s", sendWidget{})
	r.Reconcile(tree, th, send, nil, nil, "")

	if len(received) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(received))
	}
	if received[0] != "msg" {
		t.Errorf("message = %v, want %q", received[0], "msg")
	}

	// Second reconcile — already sent, no new message.
	r.Reconcile(tree, th, send, nil, nil, "")
	if len(received) != 1 {
		t.Errorf("expected still 1 message after 2nd reconcile, got %d", len(received))
	}
}

// ── Event dispatch through nested widgets ───────────────────────

type eventNestedOuter struct{}

func (eventNestedOuter) Render(ctx RenderCtx, raw WidgetState) (Element, WidgetState) {
	return ComponentWithKey("inner", eventCapture{}), raw
}

func TestReconcileEventsRoutedToNestedWidget(t *testing.T) {
	r := NewReconciler()
	th := theme.Default
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	tree := ComponentWithKey("outer", eventNestedOuter{})

	// First reconcile to establish both widgets.
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	// The inner widget's UID is nested under the outer's UID.
	outerUID := MakeUID(0, "outer", 0)
	innerUID := MakeUID(outerUID, "inner", 0)

	// Focus the inner widget and send a key event.
	fm.SetFocusedUID(innerUID)
	d.Collect(input.KeyMsg{Key: input.KeyEnter, Action: input.KeyPress})
	d.Dispatch()

	r.Reconcile(tree, th, noopSend, d, fm, "")

	state := r.StateFor(innerUID)
	s, ok := state.(*eventCaptureState)
	if !ok {
		t.Fatalf("expected *eventCaptureState, got %T", state)
	}
	if len(s.Events) != 1 {
		t.Fatalf("inner widget should receive 1 event, got %d", len(s.Events))
	}
	if s.Events[0].Kind != EventKey {
		t.Errorf("event kind = %d, want EventKey", s.Events[0].Kind)
	}
}

// ── Multiple DirtyTrackers — only some dirty ────────────────────

func TestCheckDirtyTrackersPartialDirty(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Column(
		ComponentWithKey("d1", dirtyWidget{}),
		ComponentWithKey("d2", dirtyWidget{}),
		ComponentWithKey("d3", dirtyWidget{}),
	)
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	// Only mark d2 as dirty.
	uid2 := MakeUID(0, "d2", 0)
	s2 := r.StateFor(uid2).(*dirtyState)
	s2.dirty = true

	if !r.CheckDirtyTrackers() {
		t.Error("should return true when at least one tracker is dirty")
	}

	// All should be clean now.
	if r.CheckDirtyTrackers() {
		t.Error("should return false after clearing")
	}
}

// ── TickAnimators with mixed states ─────────────────────────────

func TestTickAnimatorsMixedAnimatorAndNonAnimator(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Column(
		ComponentWithKey("anim", animWidget{}),
		ComponentWithKey("noanim", nonAnimWidget{}),
		ComponentWithKey("counter", counterWidget{}),
	)
	r.Reconcile(tree, th, noopSend, nil, nil, "")

	// Tick — only the animator should advance.
	running := r.TickAnimators(100 * time.Millisecond)
	if !running {
		t.Error("should still be running (animWidget hasn't reached target)")
	}

	// Non-animator states should be untouched.
	noanim := MakeUID(0, "noanim", 0)
	ns := r.StateFor(noanim).(*nonAnimState)
	if ns.count != 1 {
		t.Errorf("non-anim count = %d, want 1 (only from Render)", ns.count)
	}
}

// ── treeEqual edge cases ────────────────────────────────────────

func TestTreeEqualNilCases(t *testing.T) {
	if !treeEqual(nil, nil) {
		t.Error("nil == nil should be true")
	}
	if treeEqual(nil, Text("x")) {
		t.Error("nil != Text should be false")
	}
	if treeEqual(Text("x"), nil) {
		t.Error("Text != nil should be false")
	}
}

func TestTreeEqualTypeMismatch(t *testing.T) {
	if treeEqual(Text("x"), Spacer(10)) {
		t.Error("different element types should not be equal")
	}
}

func TestTreeEqualThemedElement(t *testing.T) {
	a := ThemedElement{Theme: theme.LuxLight, Children: []Element{Text("x")}}
	b := ThemedElement{Theme: theme.LuxLight, Children: []Element{Text("x")}}
	if !treeEqual(a, b) {
		t.Error("identical ThemedElements should be equal")
	}

	c := ThemedElement{Theme: theme.LuxLight, Children: []Element{Text("y")}}
	if treeEqual(a, c) {
		t.Error("ThemedElements with different children should not be equal")
	}

	d := ThemedElement{Theme: theme.LuxLight, Children: []Element{Text("x"), Text("y")}}
	if treeEqual(a, d) {
		t.Error("ThemedElements with different child counts should not be equal")
	}
}

func TestTreeEqualKeyedElement(t *testing.T) {
	a := KeyedElement{Key: "k", Child: Text("x")}
	b := KeyedElement{Key: "k", Child: Text("x")}
	if !treeEqual(a, b) {
		t.Error("identical KeyedElements should be equal")
	}

	c := KeyedElement{Key: "k2", Child: Text("x")}
	if treeEqual(a, c) {
		t.Error("KeyedElements with different keys should not be equal")
	}
}

// ── Reconcile changed detection across multiple frames ──────────

func TestReconcileChangedDetectsRemovedChild(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Column(Text("a"), Text("b"), Text("c")), th, noopSend, nil, nil, "")
	_, changed := r.Reconcile(Column(Text("a"), Text("c")), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("removing a child should report changed")
	}
}

func TestReconcileChangedDetectsReplacedElement(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Column(Text("a")), th, noopSend, nil, nil, "")
	_, changed := r.Reconcile(Column(Spacer(10)), th, noopSend, nil, nil, "")
	if !changed {
		t.Error("replacing element type should report changed")
	}
}
