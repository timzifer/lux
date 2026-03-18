package ui

import (
	"testing"

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
	resolved, changed := r.Reconcile(tree, th, noopSend)
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
	_, _ = r.Reconcile(tree, th, noopSend)
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

	r.Reconcile(tree, th, noopSend)
	r.Reconcile(tree, th, noopSend)

	if r.StateCount() != 2 {
		t.Fatalf("expected 2 states, got %d", r.StateCount())
	}
}

func TestReconcileStateResetOnKeyChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree1 := ComponentWithKey("v1", counterWidget{})
	r.Reconcile(tree1, th, noopSend)
	r.Reconcile(tree1, th, noopSend) // RenderCount = 2

	// Change the key — should get fresh state.
	tree2 := ComponentWithKey("v2", counterWidget{})
	r.Reconcile(tree2, th, noopSend)

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

	tree := Column(Text("hello"), Button("ok", nil))

	// First call — always changed.
	_, changed := r.Reconcile(tree, th, noopSend)
	if !changed {
		t.Error("first reconcile should be changed")
	}

	// Same tree again — no change.
	_, changed = r.Reconcile(tree, th, noopSend)
	if changed {
		t.Error("identical tree should not report changed")
	}
}

func TestReconcileDetectsTextChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Text("v1"), th, noopSend)
	_, changed := r.Reconcile(Text("v2"), th, noopSend)
	if !changed {
		t.Error("different text content should report changed")
	}
}

func TestReconcileDetectsStructuralChange(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	r.Reconcile(Column(Text("a")), th, noopSend)
	_, changed := r.Reconcile(Column(Text("a"), Text("b")), th, noopSend)
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
	r.Reconcile(tree, th, noopSend)
	if r.StateCount() != 2 {
		t.Fatalf("expected 2 states, got %d", r.StateCount())
	}

	// Remove one widget.
	tree2 := Column(
		ComponentWithKey("keep", counterWidget{}),
	)
	r.Reconcile(tree2, th, noopSend)
	if r.StateCount() != 1 {
		t.Errorf("expected 1 state after removal, got %d", r.StateCount())
	}
}

// ── Reconciler: Widget.Render integration ────────────────────────

func TestReconcileExpandsWidgetToElement(t *testing.T) {
	r := NewReconciler()
	th := theme.Default

	tree := Component(greetWidget{Name: "lux"})

	resolved, _ := r.Reconcile(tree, th, noopSend)
	te, ok := resolved.(textElement)
	if !ok {
		t.Fatalf("expected textElement, got %T", resolved)
	}
	if te.Content != "hello lux" {
		t.Errorf("content = %q, want %q", te.Content, "hello lux")
	}

	// Second call — state was persisted, so greeting changes.
	resolved, _ = r.Reconcile(tree, th, noopSend)
	te, ok = resolved.(textElement)
	if !ok {
		t.Fatalf("expected textElement, got %T", resolved)
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
	a := Column(Row(Text("x")), Button("ok", nil))
	b := Column(Row(Text("x")), Button("ok", nil))
	if !treeEqual(a, b) {
		t.Error("structurally identical nested trees should be equal")
	}

	c := Column(Row(Text("y")), Button("ok", nil))
	if treeEqual(a, c) {
		t.Error("trees with different text should not be equal")
	}
}
