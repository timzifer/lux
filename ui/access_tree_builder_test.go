package ui

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/theme"
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
