package ui

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/validation"
)

// testAccessibleWidget is a test widget that implements AccessibleWidget.
type testAccessibleWidget struct {
	label   string
	role    a11y.AccessRole
	onClick func()
}

func (w testAccessibleWidget) Render(ctx RenderCtx, state WidgetState) (Element, WidgetState) {
	return Text(w.label), state
}

func (w testAccessibleWidget) Accessibility(state WidgetState) a11y.AccessNode {
	node := a11y.AccessNode{
		Role:  w.role,
		Label: w.label,
	}
	if w.onClick != nil {
		node.Actions = []a11y.AccessAction{{Name: "activate", Trigger: w.onClick}}
	}
	return node
}

// testPlainWidget is a widget that does NOT implement AccessibleWidget.
type testPlainWidget struct{}

func (testPlainWidget) Render(ctx RenderCtx, state WidgetState) (Element, WidgetState) {
	return Text("plain"), state
}

func TestBuildAccessTree_Button(t *testing.T) {
	clicked := false
	w := testAccessibleWidget{
		label:   "OK",
		role:    a11y.RoleButton,
		onClick: func() { clicked = true },
	}

	th := theme.LuxLight
	reconciler := NewReconciler()
	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "")

	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{})
	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "OK" {
		t.Errorf("expected label 'OK', got %q", buttons[0].Node.Label)
	}
	if len(buttons[0].Node.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(buttons[0].Node.Actions))
	}
	buttons[0].Node.Actions[0].Trigger()
	if !clicked {
		t.Error("expected action trigger to set clicked=true")
	}
}

func TestBuildAccessTree_PlainWidgetFallback(t *testing.T) {
	w := testPlainWidget{}

	th := theme.LuxLight
	reconciler := NewReconciler()
	tree := Component(w)
	resolved, _ := reconciler.Reconcile(tree, th, func(any) {}, nil, nil, "")

	accessTree := BuildAccessTree(resolved, reconciler, a11y.Rect{})
	groups := accessTree.FindByRole(a11y.RoleGroup)
	if len(groups) == 0 {
		t.Fatal("expected at least 1 group node for non-accessible widget")
	}
}

func TestBuildAccessTree_ButtonElement(t *testing.T) {
	tree := ButtonText("Submit", func() {})

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Submit" {
		t.Errorf("expected label 'Submit', got %q", buttons[0].Node.Label)
	}
}

func TestBuildAccessTree_NestedBox(t *testing.T) {
	tree := Column(
		ButtonText("A", func() {}),
		ButtonText("B", func() {}),
	)

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(buttons))
	}
}

func TestBuildAccessTree_TreeNavigation(t *testing.T) {
	tree := Column(
		Text("Header"),
		ButtonText("Action", func() {}),
	)

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})

	// Synthetic root + text "Header" + button "Action" + button's text content.
	if len(accessTree.Nodes) < 3 {
		t.Fatalf("expected at least 3 nodes, got %d", len(accessTree.Nodes))
	}

	// Root should have children (the text and button are under the synthetic root).
	root := accessTree.Root()
	if root == nil {
		t.Fatal("expected root node")
	}
	children := accessTree.Children(root)
	if len(children) < 2 {
		t.Fatalf("expected at least 2 children of root, got %d", len(children))
	}

	// Verify we can find both by role/label.
	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Action" {
		t.Errorf("expected label 'Action', got %q", buttons[0].Node.Label)
	}

	headers := accessTree.FindByLabel("Header")
	if len(headers) != 1 {
		t.Fatalf("expected 1 node with label 'Header', got %d", len(headers))
	}
}

func TestBuildAccessTree_EmptyTree(t *testing.T) {
	tree := Empty()
	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})
	// Always has the synthetic root node at index 0.
	if len(accessTree.Nodes) != 1 {
		t.Errorf("expected 1 node (synthetic root) for empty tree, got %d", len(accessTree.Nodes))
	}
	if accessTree.Root().Node.Role != a11y.RoleGroup {
		t.Errorf("expected root role RoleGroup, got %d", accessTree.Root().Node.Role)
	}
}

// ── Surface-Subtree Merge Tests (RFC-006 §6) ────────────────────

func TestBuildAccessTree_SurfaceWithSemanticProvider(t *testing.T) {
	// Create a surface with semantic content (e.g. a PDF viewer).
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleHeading, Label: "Chapter 1", Bounds: draw.Rect{X: 0, Y: 0, W: 200, H: 30}},
			{ID: 2, Role: a11y.RoleLink, Label: "More info", Bounds: draw.Rect{X: 0, Y: 30, W: 100, H: 20}},
		},
		version: 1,
	}

	tree := SurfaceElement{
		ID:       1,
		Provider: pdf,
		Width:    300,
		Height:   400,
	}

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{Width: 800, Height: 600})

	// Should have: synthetic root + heading + link = at least 3 nodes.
	if len(accessTree.Nodes) < 3 {
		t.Fatalf("expected at least 3 nodes, got %d", len(accessTree.Nodes))
	}

	// Verify surface nodes were merged into the tree.
	headings := accessTree.FindByRole(a11y.RoleHeading)
	if len(headings) != 1 {
		t.Fatalf("expected 1 heading from surface, got %d", len(headings))
	}
	if headings[0].Node.Label != "Chapter 1" {
		t.Errorf("expected heading label 'Chapter 1', got %q", headings[0].Node.Label)
	}

	links := accessTree.FindByRole(a11y.RoleLink)
	if len(links) != 1 {
		t.Fatalf("expected 1 link from surface, got %d", len(links))
	}
	if links[0].Node.Label != "More info" {
		t.Errorf("expected link label 'More info', got %q", links[0].Node.Label)
	}
}

func TestBuildAccessTree_SurfaceWithoutSemanticProvider(t *testing.T) {
	// Surface without SemanticProvider gets a fallback group node.
	plain := &fakeSurface{}

	tree := SurfaceElement{
		ID:       2,
		Provider: plain,
		Width:    640,
		Height:   480,
	}

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{Width: 800, Height: 600})

	// Synthetic root + fallback "Surface" group = 2 nodes.
	if len(accessTree.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(accessTree.Nodes))
	}

	surfaces := accessTree.FindByLabel("Surface")
	if len(surfaces) != 1 {
		t.Fatalf("expected 1 fallback Surface node, got %d", len(surfaces))
	}
}

func TestBuildAccessTree_SurfaceMixedWithWidgets(t *testing.T) {
	// Mix regular widgets with a semantic surface.
	pdf := &fakeSemantic{
		nodes: []SurfaceAccessNode{
			{ID: 1, Role: a11y.RoleHeading, Label: "Document Title", Bounds: draw.Rect{W: 300, H: 30}},
		},
		version: 1,
	}

	tree := Column(
		ButtonText("Back", func() {}),
		SurfaceElement{ID: 3, Provider: pdf, Width: 300, Height: 400},
		ButtonText("Next", func() {}),
	)

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{Width: 800, Height: 600})

	buttons := accessTree.FindByRole(a11y.RoleButton)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(buttons))
	}

	headings := accessTree.FindByRole(a11y.RoleHeading)
	if len(headings) != 1 {
		t.Fatalf("expected 1 heading from surface, got %d", len(headings))
	}

	// All nodes should have unique IDs.
	ids := make(map[a11y.AccessNodeID]bool)
	for _, n := range accessTree.Nodes {
		if ids[n.ID] {
			t.Errorf("duplicate access node ID: %d", n.ID)
		}
		ids[n.ID] = true
	}
}

func TestBuildAccessTree_FormFieldValid(t *testing.T) {
	tree := FormField(
		TextField("hello", "Type here..."),
		WithFormLabel("Username"),
		WithFormHint("Pick a unique name."),
	)

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})

	// Should have a Group node for the FormField wrapper.
	groups := accessTree.FindByRole(a11y.RoleGroup)
	var formGroup *a11y.AccessTreeNode
	for _, g := range groups {
		if g.Node.Label == "Username" {
			formGroup = g
			break
		}
	}
	if formGroup == nil {
		t.Fatal("expected a Group node with label 'Username'")
	}
	if formGroup.Node.Description != "Pick a unique name." {
		t.Errorf("expected description 'Pick a unique name.', got %q", formGroup.Node.Description)
	}
	if formGroup.Node.States.Invalid {
		t.Error("valid FormField should not have Invalid state")
	}

	// The TextField child should appear as a TextInput under the group.
	inputs := accessTree.FindByRole(a11y.RoleTextInput)
	if len(inputs) != 1 {
		t.Fatalf("expected 1 TextInput, got %d", len(inputs))
	}
}

func TestBuildAccessTree_FormFieldInvalid(t *testing.T) {
	tree := FormField(
		TextField("", "Email"),
		WithFormLabel("Email"),
		WithFormHint("We'll never share it."),
		WithFormValidation(validation.FieldResult{Error: "This field is required"}),
	)

	reconciler := NewReconciler()
	accessTree := BuildAccessTree(tree, reconciler, a11y.Rect{})

	groups := accessTree.FindByRole(a11y.RoleGroup)
	var formGroup *a11y.AccessTreeNode
	for _, g := range groups {
		if g.Node.Label == "Email" {
			formGroup = g
			break
		}
	}
	if formGroup == nil {
		t.Fatal("expected a Group node with label 'Email'")
	}
	if !formGroup.Node.States.Invalid {
		t.Error("invalid FormField should have Invalid state")
	}
	// When invalid, Description should be the error message (overrides hint).
	if formGroup.Node.Description != "This field is required" {
		t.Errorf("expected error description, got %q", formGroup.Node.Description)
	}
}
