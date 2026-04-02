package ui

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// findByRoleAndLabel returns nodes matching both role and label.
func findByRoleAndLabel(tree a11y.AccessTree, role a11y.AccessRole, label string) []*a11y.AccessTreeNode {
	var result []*a11y.AccessTreeNode
	for i := range tree.Nodes {
		if tree.Nodes[i].Node.Role == role && tree.Nodes[i].Node.Label == label {
			result = append(result, &tree.Nodes[i])
		}
	}
	return result
}

// ── FocusTrap + AccessTree integration (RFC-001 §11.7) ───────────

func TestAccessTreeFocusTrapHidesBackground(t *testing.T) {
	// Build a tree with background content and a modal overlay.
	reconciler := NewReconciler()
	th := theme.LuxLight

	bgContent := Column(
		ButtonText("Background Button", func() {}),
		Text("Background Text"),
	)
	overlayContent := Column(
		ButtonText("Dialog OK", func() {}),
		ButtonText("Dialog Cancel", func() {}),
	)

	tree := Column(
		bgContent,
		Overlay{
			ID:        "modal-dialog",
			Content:   overlayContent,
			Backdrop:  true,
			FocusTrap: &FocusTrap{TrapID: "modal-dialog", RestoreFocus: true},
		},
	)

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)

	// Build access tree WITH active trap — background should be excluded.
	b := AccessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: a11y.Rect{Width: 800, Height: 600},
		ActiveTrapID: "modal-dialog",
	}
	rootIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, b.windowBounds)
	b.Walk(resolved, int32(rootIdx))
	accessTree := a11y.AccessTree{Nodes: b.nodes}
	accessTree.EnsureIndex()

	// Background buttons should NOT appear.
	bgButtons := findByRoleAndLabel(accessTree, a11y.RoleButton, "Background Button")
	if len(bgButtons) != 0 {
		t.Errorf("background button should be hidden by focus trap, found %d", len(bgButtons))
	}
	bgText := accessTree.FindByLabel("Background Text")
	if len(bgText) != 0 {
		t.Errorf("background text should be hidden by focus trap, found %d", len(bgText))
	}

	// Dialog buttons SHOULD appear.
	dialogOK := findByRoleAndLabel(accessTree, a11y.RoleButton, "Dialog OK")
	if len(dialogOK) != 1 {
		t.Errorf("expected 1 'Dialog OK' button, got %d", len(dialogOK))
	}
	dialogCancel := findByRoleAndLabel(accessTree, a11y.RoleButton, "Dialog Cancel")
	if len(dialogCancel) != 1 {
		t.Errorf("expected 1 'Dialog Cancel' button, got %d", len(dialogCancel))
	}
}

func TestAccessTreeNoTrapShowsEverything(t *testing.T) {
	// Without active trap, all content should be visible.
	reconciler := NewReconciler()
	th := theme.LuxLight

	tree := Column(
		ButtonText("Background", func() {}),
		Overlay{
			ID:      "popup",
			Content: ButtonText("Popup", func() {}),
		},
	)

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	bg := findByRoleAndLabel(accessTree, a11y.RoleButton, "Background")
	if len(bg) != 1 {
		t.Errorf("expected 1 'Background' button without trap, got %d", len(bg))
	}
	popup := findByRoleAndLabel(accessTree, a11y.RoleButton, "Popup")
	if len(popup) != 1 {
		t.Errorf("expected 1 'Popup' button without trap, got %d", len(popup))
	}
}

// ── Modal overlay produces RoleDialog node ──────────────────────

func TestAccessTreeModalOverlayRoleDialog(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	tree := Overlay{
		ID:       "confirm",
		Content:  Text("Are you sure?"),
		Backdrop: true,
	}

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	dialogs := accessTree.FindByRole(a11y.RoleDialog)
	if len(dialogs) != 1 {
		t.Fatalf("expected 1 RoleDialog for modal overlay, got %d", len(dialogs))
	}
	if dialogs[0].Node.Label != "confirm" {
		t.Errorf("dialog label = %q, want %q", dialogs[0].Node.Label, "confirm")
	}
}

func TestAccessTreeNonModalOverlayRoleGroup(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	tree := Overlay{
		ID:       "dropdown",
		Content:  Text("Option A"),
		Backdrop: false,
	}

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	// Non-modal overlay should use RoleGroup, not RoleDialog.
	dialogs := accessTree.FindByRole(a11y.RoleDialog)
	if len(dialogs) != 0 {
		t.Errorf("non-modal overlay should not be RoleDialog, got %d", len(dialogs))
	}
	groups := accessTree.FindByLabel("dropdown")
	if len(groups) != 1 {
		t.Fatalf("expected 1 node labeled 'dropdown', got %d", len(groups))
	}
	if groups[0].Node.Role != a11y.RoleGroup {
		t.Errorf("non-modal overlay role = %d, want RoleGroup", groups[0].Node.Role)
	}
}

// ── Nested overlays a11y tree ───────────────────────────────────

func TestAccessTreeNestedOverlays(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	innerOverlay := Overlay{
		ID:       "inner-dialog",
		Content:  ButtonText("Inner Action", func() {}),
		Backdrop: true,
	}
	outerOverlay := Overlay{
		ID:       "outer-dialog",
		Content:  Column(ButtonText("Outer Action", func() {}), innerOverlay),
		Backdrop: true,
	}

	tree := Column(Text("Background"), outerOverlay)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)

	// Build with trap on inner dialog — only inner content should be visible.
	b := AccessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: a11y.Rect{Width: 800, Height: 600},
		ActiveTrapID: "inner-dialog",
	}
	rootIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, b.windowBounds)
	b.Walk(resolved, int32(rootIdx))
	accessTree := a11y.AccessTree{Nodes: b.nodes}
	accessTree.EnsureIndex()

	innerActions := findByRoleAndLabel(accessTree, a11y.RoleButton, "Inner Action")
	if len(innerActions) != 1 {
		t.Errorf("expected 1 'Inner Action', got %d", len(innerActions))
	}

	// Background should be hidden.
	bg := accessTree.FindByLabel("Background")
	if len(bg) != 0 {
		t.Errorf("background should be hidden with inner trap active, found %d", len(bg))
	}
}

// ── AccessibleWidget with all states ────────────────────────────

type testFullAccessWidget struct {
	label    string
	role     a11y.AccessRole
	states   a11y.AccessStates
	value    string
	desc     string
	numeric  *a11y.AccessNumericValue
	actions  []a11y.AccessAction
}

func (w testFullAccessWidget) Render(_ RenderCtx, state WidgetState) (Element, WidgetState) {
	return Text(w.label), state
}

func (w testFullAccessWidget) Accessibility(_ WidgetState) a11y.AccessNode {
	return a11y.AccessNode{
		Role:         w.role,
		Label:        w.label,
		Description:  w.desc,
		Value:        w.value,
		States:       w.states,
		NumericValue: w.numeric,
		Actions:      w.actions,
	}
}

func TestAccessTreeWidgetWithAllStates(t *testing.T) {
	w := testFullAccessWidget{
		label: "Volume",
		role:  a11y.RoleSlider,
		desc:  "Adjust volume level",
		value: "50%",
		states: a11y.AccessStates{
			Focused:  true,
			Disabled: false,
		},
		numeric: &a11y.AccessNumericValue{Current: 50, Min: 0, Max: 100, Step: 1},
		actions: []a11y.AccessAction{
			{Name: "increment"},
			{Name: "decrement"},
		},
	}

	reconciler := NewReconciler()
	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, theme.LuxLight, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	sliders := accessTree.FindByRole(a11y.RoleSlider)
	if len(sliders) != 1 {
		t.Fatalf("expected 1 slider, got %d", len(sliders))
	}
	node := sliders[0].Node
	if node.Label != "Volume" {
		t.Errorf("label = %q, want %q", node.Label, "Volume")
	}
	if node.Description != "Adjust volume level" {
		t.Errorf("description = %q, want %q", node.Description, "Adjust volume level")
	}
	if node.Value != "50%" {
		t.Errorf("value = %q, want %q", node.Value, "50%")
	}
	if !node.States.Focused {
		t.Error("expected Focused state")
	}
	if node.NumericValue == nil {
		t.Fatal("expected NumericValue to be non-nil")
	}
	if node.NumericValue.Current != 50 {
		t.Errorf("NumericValue.Current = %f, want 50", node.NumericValue.Current)
	}
	if node.NumericValue.Max != 100 {
		t.Errorf("NumericValue.Max = %f, want 100", node.NumericValue.Max)
	}
	if len(node.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(node.Actions))
	}
	if node.Actions[0].Name != "increment" {
		t.Errorf("action[0] = %q, want %q", node.Actions[0].Name, "increment")
	}
}

// ── Deep nesting a11y tree structure ────────────────────────────

func TestAccessTreeDeepNestingParentChild(t *testing.T) {
	tree := Column(
		Row(
			Column(
				ButtonText("Deep Button", func() {}),
			),
		),
	)

	accessTree := RenderToAccessTree(tree)

	// The button should exist regardless of nesting depth.
	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button in deep nesting, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Deep Button" {
		t.Errorf("label = %q, want %q", buttons[0].Node.Label, "Deep Button")
	}

	// Verify parent chain reaches root.
	node := buttons[0]
	depth := 0
	for node.ParentIndex >= 0 {
		parent := accessTree.Parent(node)
		if parent == nil {
			t.Fatal("parent should not be nil for non-root node")
		}
		node = parent
		depth++
		if depth > 20 {
			t.Fatal("infinite parent loop detected")
		}
	}
	// Should reach root (index 0).
	if node != accessTree.Root() {
		t.Error("parent chain should lead to root")
	}
}

// ── Mixed widget/non-widget tree a11y ───────────────────────────

func TestAccessTreeMixedWidgetAndLeafElements(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	w := testAccessibleWidget{
		label: "Custom Widget",
		role:  a11y.RoleButton,
	}

	tree := Column(
		Text("Static Text"),
		Component(w),
		Checkbox("Accept", true, nil),
		Divider(), // Divider has no a11y node
		Slider(0.5, nil),
	)

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	// Check all expected nodes exist.
	texts := accessTree.FindByLabel("Static Text")
	if len(texts) != 1 {
		t.Errorf("expected 1 'Static Text', got %d", len(texts))
	}
	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Errorf("expected 1 button (Custom Widget), got %d", len(buttons))
	}
	checkboxes := accessTree.FindByRole(a11y.RoleCheckbox)
	if len(checkboxes) != 1 {
		t.Errorf("expected 1 checkbox, got %d", len(checkboxes))
	}
	sliders := accessTree.FindByRole(a11y.RoleSlider)
	if len(sliders) != 1 {
		t.Errorf("expected 1 slider, got %d", len(sliders))
	}

	// Verify unique IDs across all nodes.
	ids := make(map[a11y.AccessNodeID]bool)
	for _, n := range accessTree.Nodes {
		if ids[n.ID] {
			t.Errorf("duplicate access node ID: %d", n.ID)
		}
		ids[n.ID] = true
	}
}

// ── Widget bounds in access tree via dispatcher ─────────────────

func TestAccessTreeWidgetBoundsFromDispatcher(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight
	fm := NewFocusManager()
	d := NewEventDispatcher(fm)

	w := testAccessibleWidget{
		label: "Positioned",
		role:  a11y.RoleButton,
	}

	tree := ComponentWithKey("pos", w)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, d, fm, "", nil)

	// Register bounds for the widget.
	uid := MakeUID(0, "pos", 0)
	d.RegisterWidgetBounds(uid, draw.R(50, 100, 200, 40))
	d.SwapBounds()

	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600}, d)

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	b := buttons[0].Bounds
	if b.X != 50 || b.Y != 100 || b.Width != 200 || b.Height != 40 {
		t.Errorf("bounds = %+v, want {50 100 200 40}", b)
	}
}

// ── AccessTree with ThemedElement ───────────────────────────────

func TestAccessTreeThemedSubtree(t *testing.T) {
	reconciler := NewReconciler()

	tree := ThemedElement{
		Theme: theme.LuxLight,
		Children: []Element{
			ButtonText("Themed Button", func() {}),
			Text("Themed Text"),
		},
	}

	resolved, _ := reconciler.Reconcile(tree, theme.Default, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button in themed subtree, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Themed Button" {
		t.Errorf("label = %q, want %q", buttons[0].Node.Label, "Themed Button")
	}
	texts := accessTree.FindByLabel("Themed Text")
	if len(texts) != 1 {
		t.Errorf("expected 1 'Themed Text', got %d", len(texts))
	}
}

// ── AccessTree with KeyedElement ────────────────────────────────

func TestAccessTreeKeyedElements(t *testing.T) {
	tree := Column(
		KeyedElement{Key: "k1", Child: ButtonText("First", func() {})},
		KeyedElement{Key: "k2", Child: ButtonText("Second", func() {})},
	)

	accessTree := RenderToAccessTree(tree)

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons through keyed elements, got %d", len(buttons))
	}
}

// ── AccessTree sibling navigation ───────────────────────────────

func TestAccessTreeSiblingNavigation(t *testing.T) {
	tree := Column(
		ButtonText("A", func() {}),
		ButtonText("B", func() {}),
		ButtonText("C", func() {}),
	)

	accessTree := RenderToAccessTree(tree)
	root := accessTree.Root()
	children := accessTree.Children(root)

	if len(children) < 3 {
		t.Fatalf("expected at least 3 children of root, got %d", len(children))
	}

	// First child should have no prev sibling.
	if children[0].PrevSibling >= 0 {
		t.Error("first child should have PrevSibling = -1")
	}
	// Last child should have no next sibling.
	last := children[len(children)-1]
	if last.NextSibling >= 0 {
		t.Error("last child should have NextSibling = -1")
	}
	// Middle children should have both siblings.
	for i := 1; i < len(children)-1; i++ {
		if children[i].PrevSibling < 0 {
			t.Errorf("child[%d] should have PrevSibling", i)
		}
		if children[i].NextSibling < 0 {
			t.Errorf("child[%d] should have NextSibling", i)
		}
	}
}

// ── Zero-bounds fallback ────────────────────────────────────────

func TestAccessTreeZeroBoundsFallsBackToWindow(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	w := testAccessibleWidget{
		label: "No Bounds",
		role:  a11y.RoleButton,
	}

	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)

	windowBounds := a11y.Rect{Width: 1024, Height: 768}
	accessTree := BuildAccessTree(resolved, reconciler, windowBounds)

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}

	// Without dispatcher, zero bounds should fall back to window bounds.
	b := buttons[0].Bounds
	if b.Width != 1024 || b.Height != 768 {
		t.Errorf("zero-size bounds should fall back to window: got %+v", b)
	}
}

// ── Disabled widget a11y state ──────────────────────────────────

func TestAccessTreeDisabledButton(t *testing.T) {
	tree := ButtonTextDisabled("Disabled")
	accessTree := RenderToAccessTree(tree)

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if !buttons[0].Node.States.Disabled {
		t.Error("disabled button should have Disabled state")
	}
}

// ── Checked checkbox a11y state ────────────────────────────────

func TestAccessTreeCheckedCheckbox(t *testing.T) {
	tree := Checkbox("Terms", true, nil)
	accessTree := RenderToAccessTree(tree)

	cbs := accessTree.FindByRole(a11y.RoleCheckbox)
	if len(cbs) != 1 {
		t.Fatalf("expected 1 checkbox, got %d", len(cbs))
	}
	if !cbs[0].Node.States.Checked {
		t.Error("checked checkbox should have Checked state")
	}
}

func TestAccessTreeUncheckedCheckbox(t *testing.T) {
	tree := Checkbox("Terms", false, nil)
	accessTree := RenderToAccessTree(tree)

	cbs := accessTree.FindByRole(a11y.RoleCheckbox)
	if len(cbs) != 1 {
		t.Fatalf("expected 1 checkbox, got %d", len(cbs))
	}
	if cbs[0].Node.States.Checked {
		t.Error("unchecked checkbox should not have Checked state")
	}
}

// ── Surface with parent→child relationships ─────────────────────

func TestAccessTreeSurfaceNestedNodes(t *testing.T) {
	sem := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleGroup, Label: "Container", Bounds: draw.Rect{W: 300, H: 200}},
			{ID: 2, Parent: 1, Role: a11y.RoleButton, Label: "Surface Button", Bounds: draw.Rect{X: 10, Y: 10, W: 80, H: 30}},
		},
		version: 1,
	}

	tree := SurfaceElement{ID: 1, Provider: sem, Width: 300, Height: 200}
	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{Width: 800, Height: 600})

	containers := accessTree.FindByLabel("Container")
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	surfButtons := accessTree.FindByLabel("Surface Button")
	if len(surfButtons) != 1 {
		t.Fatalf("expected 1 surface button, got %d", len(surfButtons))
	}

	// Surface button should be a child of container.
	parent := accessTree.Parent(surfButtons[0])
	if parent == nil || parent.Node.Label != "Container" {
		t.Error("surface button should be a child of container")
	}
}

// ── AccessTree root invariants ──────────────────────────────────

func TestAccessTreeRootInvariants(t *testing.T) {
	tree := Column(ButtonText("A", func() {}))
	accessTree := RenderToAccessTree(tree)

	root := accessTree.Root()
	if root == nil {
		t.Fatal("root should never be nil")
	}
	if root.Node.Role != a11y.RoleGroup {
		t.Errorf("root role = %d, want RoleGroup", root.Node.Role)
	}
	if root.Node.Label != "Application" {
		t.Errorf("root label = %q, want %q", root.Node.Label, "Application")
	}
	if root.ParentIndex != -1 {
		t.Errorf("root parent index = %d, want -1", root.ParentIndex)
	}
	if root.ChildCount == 0 {
		t.Error("root should have at least 1 child")
	}
}

// ── Widget action trigger round-trip ───────────────────────────

func TestAccessTreeActionTriggerRoundTrip(t *testing.T) {
	var triggered int
	w := testAccessibleWidget{
		label:   "Trigger Me",
		role:    a11y.RoleButton,
		onClick: func() { triggered++ },
	}

	reconciler := NewReconciler()
	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, theme.LuxLight, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if len(buttons[0].Node.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(buttons[0].Node.Actions))
	}

	// Trigger the action multiple times.
	buttons[0].Node.Actions[0].Trigger()
	buttons[0].Node.Actions[0].Trigger()
	if triggered != 2 {
		t.Errorf("expected triggered=2, got %d", triggered)
	}
}

// ── Stability regression tests ──────────────────────────────────
// See milestone "Stability: Reconciler / Scene / AccessTree"

// TestAccessTreeFocusTrapPushPopSameFrame verifies that after a focus trap
// is pushed (hides background) and then popped within the same logical frame,
// the resulting access tree contains all content. (Issue #93)
func TestAccessTreeFocusTrapPushPopSameFrame(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	bgContent := ButtonText("Background", func() {})
	overlayContent := ButtonText("Dialog OK", func() {})

	tree := Column(
		bgContent,
		Overlay{
			ID:        "trap-dialog",
			Content:   overlayContent,
			Backdrop:  true,
			FocusTrap: &FocusTrap{TrapID: "trap-dialog", RestoreFocus: true},
		},
	)

	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)

	// Phase 1: trap active → background hidden.
	b1 := AccessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: a11y.Rect{Width: 800, Height: 600},
		ActiveTrapID: "trap-dialog",
	}
	root1 := b1.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, b1.windowBounds)
	b1.Walk(resolved, int32(root1))
	tree1 := a11y.AccessTree{Nodes: b1.nodes}
	tree1.EnsureIndex()

	bgInTrap := findByRoleAndLabel(tree1, a11y.RoleButton, "Background")
	if len(bgInTrap) != 0 {
		t.Errorf("with trap active: background should be hidden, found %d", len(bgInTrap))
	}
	dialogInTrap := findByRoleAndLabel(tree1, a11y.RoleButton, "Dialog OK")
	if len(dialogInTrap) != 1 {
		t.Errorf("with trap active: expected 1 dialog button, got %d", len(dialogInTrap))
	}

	// Phase 2: trap popped (ActiveTrapID="") → everything visible.
	b2 := AccessTreeBuilder{
		reconciler:   reconciler,
		windowBounds: a11y.Rect{Width: 800, Height: 600},
		ActiveTrapID: "",
	}
	root2 := b2.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Application"}, -1, b2.windowBounds)
	b2.Walk(resolved, int32(root2))
	tree2 := a11y.AccessTree{Nodes: b2.nodes}
	tree2.EnsureIndex()

	bgAfterPop := findByRoleAndLabel(tree2, a11y.RoleButton, "Background")
	if len(bgAfterPop) != 1 {
		t.Errorf("after trap pop: expected 1 background button, got %d", len(bgAfterPop))
	}
	dialogAfterPop := findByRoleAndLabel(tree2, a11y.RoleButton, "Dialog OK")
	if len(dialogAfterPop) != 1 {
		t.Errorf("after trap pop: expected 1 dialog button, got %d", len(dialogAfterPop))
	}
}

// testEmptyAccessWidget returns Empty() but provides accessibility metadata.
type testEmptyAccessWidget struct {
	label string
	role  a11y.AccessRole
}

func (w testEmptyAccessWidget) Render(_ RenderCtx, state WidgetState) (Element, WidgetState) {
	return Empty(), state
}

func (w testEmptyAccessWidget) Accessibility(_ WidgetState) a11y.AccessNode {
	return a11y.AccessNode{Role: w.role, Label: w.label}
}

// TestAccessTreeEmptyWidgetWithAccessibility verifies that a widget returning
// Empty() from Render still produces an accessible node if it implements
// AccessibleWidget. (Issue #93)
func TestAccessTreeEmptyWidgetWithAccessibility(t *testing.T) {
	reconciler := NewReconciler()
	th := theme.LuxLight

	w := testEmptyAccessWidget{label: "Hidden Control", role: a11y.RoleButton}
	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "", nil)
	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{Width: 800, Height: 600})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button from empty-rendering widget, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Hidden Control" {
		t.Errorf("label = %q, want %q", buttons[0].Node.Label, "Hidden Control")
	}
}
