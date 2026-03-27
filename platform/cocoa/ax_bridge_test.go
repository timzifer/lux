//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"testing"

	"github.com/timzifer/lux/a11y"
)

func TestAXRoleMapping(t *testing.T) {
	tests := []struct {
		role     a11y.AccessRole
		expected string
	}{
		{a11y.RoleButton, "AXButton"},
		{a11y.RoleCheckbox, "AXCheckBox"},
		{a11y.RoleCombobox, "AXComboBox"},
		{a11y.RoleDialog, "AXGroup"},
		{a11y.RoleGrid, "AXTable"},
		{a11y.RoleGroup, "AXGroup"},
		{a11y.RoleHeading, "AXHeading"},
		{a11y.RoleImage, "AXImage"},
		{a11y.RoleLink, "AXLink"},
		{a11y.RoleListbox, "AXList"},
		{a11y.RoleMenu, "AXMenu"},
		{a11y.RoleProgressBar, "AXProgressIndicator"},
		{a11y.RoleScrollBar, "AXScrollBar"},
		{a11y.RoleSlider, "AXSlider"},
		{a11y.RoleSpinButton, "AXIncrementor"},
		{a11y.RoleTab, "AXRadioButton"},
		{a11y.RoleTable, "AXTable"},
		{a11y.RoleTextInput, "AXTextField"},
		{a11y.RoleToggle, "AXCheckBox"},
		{a11y.RoleTree, "AXOutline"},
	}
	for _, tt := range tests {
		got := roleToAXRole(tt.role)
		if got != tt.expected {
			t.Errorf("roleToAXRole(%d) = %q, want %q", tt.role, got, tt.expected)
		}
	}
}

func TestAXRoleMappingDefault(t *testing.T) {
	got := roleToAXRole(a11y.AccessRole(9999))
	if got != "AXGroup" {
		t.Errorf("roleToAXRole(9999) = %q, want %q", got, "AXGroup")
	}
}

func TestAXActionsForRole(t *testing.T) {
	tests := []struct {
		role     a11y.AccessRole
		expected []string
	}{
		{a11y.RoleButton, []string{"AXPress"}},
		{a11y.RoleCheckbox, []string{"AXPress"}},
		{a11y.RoleSlider, []string{"AXIncrement", "AXDecrement"}},
		{a11y.RoleSpinButton, []string{"AXIncrement", "AXDecrement"}},
		{a11y.RoleGroup, nil},
		{a11y.RoleImage, nil},
	}
	for _, tt := range tests {
		got := actionsForRole(tt.role)
		if len(got) != len(tt.expected) {
			t.Errorf("actionsForRole(%d) returned %d actions, want %d", tt.role, len(got), len(tt.expected))
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("actionsForRole(%d)[%d] = %q, want %q", tt.role, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestSubroleForRole(t *testing.T) {
	if sr := subroleForRole(a11y.RoleDialog); sr != "AXDialog" {
		t.Errorf("subroleForRole(RoleDialog) = %q, want %q", sr, "AXDialog")
	}
	if sr := subroleForRole(a11y.RoleButton); sr != "" {
		t.Errorf("subroleForRole(RoleButton) = %q, want empty", sr)
	}
}

func TestAXBridgeUpdateTree(t *testing.T) {
	// Create a bridge without a real view (won't call ObjC).
	bridge := &AXBridge{
		view:     0,
		elements: make(map[a11y.AccessNodeID]*axElement),
	}

	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 1,
				Node: a11y.AccessNode{Role: a11y.RoleGroup, Label: "Root"}},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: -1, ChildCount: 0,
				Node: a11y.AccessNode{Role: a11y.RoleButton, Label: "Click Me"}},
		},
	}

	bridge.UpdateTree(tree)

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if len(bridge.tree.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(bridge.tree.Nodes))
	}

	node := bridge.tree.FindByID(2)
	if node == nil {
		t.Fatal("expected to find node with ID 2")
	}
	if node.Node.Label != "Click Me" {
		t.Errorf("expected label 'Click Me', got %q", node.Node.Label)
	}

	// Elements are created for non-root nodes. With view=0, ObjC alloc returns 0,
	// so element.obj will be 0, but the element struct should exist.
	if len(bridge.elements) != 1 {
		t.Errorf("expected 1 element (node 2), got %d", len(bridge.elements))
	}
}

func TestAXBridgeElementPruning(t *testing.T) {
	bridge := &AXBridge{
		view:     0,
		elements: make(map[a11y.AccessNodeID]*axElement),
	}

	tree1 := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 2},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: 2, PrevSibling: -1, ChildCount: 0},
			{ID: 3, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: 1, ChildCount: 0},
		},
	}
	bridge.UpdateTree(tree1)

	if len(bridge.elements) != 2 {
		t.Fatalf("expected 2 elements (nodes 2,3), got %d", len(bridge.elements))
	}

	tree2 := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 1},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: -1, ChildCount: 0},
		},
	}
	bridge.UpdateTree(tree2)

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if _, exists := bridge.elements[3]; exists {
		t.Error("expected element for node 3 to be pruned")
	}
	if len(bridge.elements) != 1 {
		t.Errorf("expected 1 element after pruning, got %d", len(bridge.elements))
	}
}

func TestStructureChanged(t *testing.T) {
	a := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1},
		},
	}
	b := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1},
		},
	}
	if structureChanged(a, b) {
		t.Error("identical trees should not be considered changed")
	}

	c := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1},
			{ID: 2, ParentIndex: 0, FirstChild: -1},
		},
	}
	if !structureChanged(a, c) {
		t.Error("different trees should be considered changed")
	}
}
