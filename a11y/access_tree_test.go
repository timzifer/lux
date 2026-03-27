package a11y

import "testing"

// buildTestTree creates a tree with the following structure:
//
//	[0] Window (root)
//	  [1] Button "OK"
//	  [2] Group
//	    [3] Checkbox "Accept"
//	    [4] TextInput "Name"
//	  [5] Slider "Volume"
func buildTestTree() AccessTree {
	nodes := []AccessTreeNode{
		{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 3,
			Node: AccessNode{Role: RoleGroup, Label: "Window"}, Bounds: Rect{0, 0, 800, 600}},
		{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: 2, PrevSibling: -1, ChildCount: 0,
			Node: AccessNode{Role: RoleButton, Label: "OK"}, Bounds: Rect{10, 10, 80, 30}},
		{ID: 3, ParentIndex: 0, FirstChild: 3, NextSibling: 5, PrevSibling: 1, ChildCount: 2,
			Node: AccessNode{Role: RoleGroup, Label: "Form"}, Bounds: Rect{10, 50, 400, 200}},
		{ID: 4, ParentIndex: 2, FirstChild: -1, NextSibling: 4, PrevSibling: -1, ChildCount: 0,
			Node: AccessNode{Role: RoleCheckbox, Label: "Accept"}, Bounds: Rect{20, 60, 100, 20}},
		{ID: 5, ParentIndex: 2, FirstChild: -1, NextSibling: -1, PrevSibling: 3, ChildCount: 0,
			Node: AccessNode{Role: RoleTextInput, Label: "Name"}, Bounds: Rect{20, 90, 200, 30}},
		{ID: 6, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: 2, ChildCount: 0,
			Node: AccessNode{Role: RoleSlider, Label: "Volume", Value: "50"}, Bounds: Rect{10, 260, 300, 20}},
	}
	tree := AccessTree{Nodes: nodes, FocusedID: 2}
	return tree
}

func TestFindByID(t *testing.T) {
	tree := buildTestTree()

	n := tree.FindByID(4)
	if n == nil {
		t.Fatal("expected to find node with ID 4")
	}
	if n.Node.Label != "Accept" {
		t.Errorf("expected label 'Accept', got %q", n.Node.Label)
	}

	if tree.FindByID(999) != nil {
		t.Error("expected nil for unknown ID")
	}
}

func TestFindByRole(t *testing.T) {
	tree := buildTestTree()
	groups := tree.FindByRole(RoleGroup)
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	buttons := tree.FindByRole(RoleButton)
	if len(buttons) != 1 {
		t.Errorf("expected 1 button, got %d", len(buttons))
	}
}

func TestFindByLabel(t *testing.T) {
	tree := buildTestTree()
	nodes := tree.FindByLabel("Volume")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node with label 'Volume', got %d", len(nodes))
	}
	if nodes[0].Node.Role != RoleSlider {
		t.Errorf("expected RoleSlider, got %d", nodes[0].Node.Role)
	}
}

func TestParent(t *testing.T) {
	tree := buildTestTree()
	child := tree.FindByID(4) // Accept checkbox
	parent := tree.Parent(child)
	if parent == nil {
		t.Fatal("expected parent")
	}
	if parent.Node.Label != "Form" {
		t.Errorf("expected parent label 'Form', got %q", parent.Node.Label)
	}

	root := tree.Root()
	if tree.Parent(root) != nil {
		t.Error("root should have no parent")
	}
}

func TestChildren(t *testing.T) {
	tree := buildTestTree()
	root := tree.Root()
	children := tree.Children(root)
	if len(children) != 3 {
		t.Fatalf("expected 3 children of root, got %d", len(children))
	}
	if children[0].Node.Label != "OK" {
		t.Errorf("first child should be 'OK', got %q", children[0].Node.Label)
	}
	if children[1].Node.Label != "Form" {
		t.Errorf("second child should be 'Form', got %q", children[1].Node.Label)
	}
	if children[2].Node.Label != "Volume" {
		t.Errorf("third child should be 'Volume', got %q", children[2].Node.Label)
	}
}

func TestFocusedNode(t *testing.T) {
	tree := buildTestTree()
	focused := tree.FocusedNode()
	if focused == nil {
		t.Fatal("expected focused node")
	}
	if focused.Node.Label != "OK" {
		t.Errorf("expected focused node 'OK', got %q", focused.Node.Label)
	}
}

func TestNodeByIndex(t *testing.T) {
	tree := buildTestTree()
	if tree.NodeByIndex(-1) != nil {
		t.Error("expected nil for negative index")
	}
	if tree.NodeByIndex(100) != nil {
		t.Error("expected nil for out-of-range index")
	}
	n := tree.NodeByIndex(0)
	if n == nil || n.Node.Label != "Window" {
		t.Error("expected root node at index 0")
	}
}

func TestIndexByID(t *testing.T) {
	tree := buildTestTree()
	if tree.IndexByID(999) != -1 {
		t.Error("expected -1 for unknown ID")
	}
	idx := tree.IndexByID(2)
	if idx != 1 {
		t.Errorf("expected index 1 for ID 2, got %d", idx)
	}
}

func TestEmptyTree(t *testing.T) {
	tree := AccessTree{}
	if tree.Root() != nil {
		t.Error("empty tree should have nil root")
	}
	if tree.FocusedNode() != nil {
		t.Error("empty tree should have nil focused")
	}
	if tree.FindByID(1) != nil {
		t.Error("empty tree FindByID should return nil")
	}
}
