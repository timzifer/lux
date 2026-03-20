//go:build windows && !nogui

package windows

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/zzl/go-win32api/v2/win32"
)

func TestRoleMapping(t *testing.T) {
	tests := []struct {
		role     a11y.AccessRole
		expected win32.UIA_CONTROLTYPE_ID
	}{
		{a11y.RoleButton, win32.UIA_ButtonControlTypeId},
		{a11y.RoleCheckbox, win32.UIA_CheckBoxControlTypeId},
		{a11y.RoleTextInput, win32.UIA_EditControlTypeId},
		{a11y.RoleSlider, win32.UIA_SliderControlTypeId},
		{a11y.RoleCombobox, win32.UIA_ComboBoxControlTypeId},
		{a11y.RoleGroup, win32.UIA_GroupControlTypeId},
		{a11y.RoleImage, win32.UIA_ImageControlTypeId},
		{a11y.RoleLink, win32.UIA_HyperlinkControlTypeId},
		{a11y.RoleToggle, win32.UIA_CheckBoxControlTypeId},
		{a11y.RoleTree, win32.UIA_TreeControlTypeId},
	}
	for _, tt := range tests {
		got := roleToControlType(tt.role)
		if got != tt.expected {
			t.Errorf("roleToControlType(%d) = %d, want %d", tt.role, got, tt.expected)
		}
	}
}

func TestPatternsForRole(t *testing.T) {
	tests := []struct {
		role     a11y.AccessRole
		expected []win32.UIA_PATTERN_ID
	}{
		{a11y.RoleButton, []win32.UIA_PATTERN_ID{win32.UIA_InvokePatternId}},
		{a11y.RoleCheckbox, []win32.UIA_PATTERN_ID{win32.UIA_TogglePatternId}},
		{a11y.RoleTextInput, []win32.UIA_PATTERN_ID{win32.UIA_ValuePatternId}},
		{a11y.RoleSlider, []win32.UIA_PATTERN_ID{win32.UIA_RangeValuePatternId}},
		{a11y.RoleCombobox, []win32.UIA_PATTERN_ID{win32.UIA_ExpandCollapsePatternId}},
		{a11y.RoleGroup, nil},
	}
	for _, tt := range tests {
		got := patternsForRole(tt.role)
		if len(got) != len(tt.expected) {
			t.Errorf("patternsForRole(%d) returned %d patterns, want %d", tt.role, len(got), len(tt.expected))
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("patternsForRole(%d)[%d] = %d, want %d", tt.role, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestVariantHelpers(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		v := variantEmpty()
		if v.Vt != win32.VT_EMPTY {
			t.Errorf("expected VT_EMPTY, got %d", v.Vt)
		}
	})

	t.Run("Int32", func(t *testing.T) {
		v := variantInt32(42)
		if v.Vt != win32.VT_I4 {
			t.Errorf("expected VT_I4, got %d", v.Vt)
		}
		if v.LValVal() != 42 {
			t.Errorf("expected 42, got %d", v.LValVal())
		}
	})

	t.Run("Bool", func(t *testing.T) {
		v := variantBool(true)
		if v.Vt != win32.VT_BOOL {
			t.Errorf("expected VT_BOOL, got %d", v.Vt)
		}
		if v.BoolValVal() != win32.VARIANT_TRUE {
			t.Errorf("expected VARIANT_TRUE, got %d", v.BoolValVal())
		}

		v2 := variantBool(false)
		if v2.BoolValVal() != win32.VARIANT_FALSE {
			t.Errorf("expected VARIANT_FALSE, got %d", v2.BoolValVal())
		}
	})

	t.Run("String", func(t *testing.T) {
		v := variantString("hello")
		if v.Vt != win32.VT_BSTR {
			t.Errorf("expected VT_BSTR, got %d", v.Vt)
		}
		// BSTR should not be nil.
		if v.BstrValVal() == nil {
			t.Error("expected non-nil BSTR")
		}
		// Clean up.
		win32.SysFreeString(v.BstrValVal())
	})

	t.Run("Float64", func(t *testing.T) {
		v := variantFloat64(3.14)
		if v.Vt != win32.VT_R8 {
			t.Errorf("expected VT_R8, got %d", v.Vt)
		}
		if v.DblValVal() != 3.14 {
			t.Errorf("expected 3.14, got %f", v.DblValVal())
		}
	})
}

func TestUIABridgeUpdateTree(t *testing.T) {
	// Create a bridge with a fake HWND (won't actually call UIA).
	bridge := &UIABridge{
		hwnd:      0,
		providers: make(map[a11y.AccessNodeID]*elementProvider),
	}
	bridge.root = newRootProvider(bridge)

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
}

func TestUIABridgeProviderPruning(t *testing.T) {
	bridge := &UIABridge{
		hwnd:      0,
		providers: make(map[a11y.AccessNodeID]*elementProvider),
	}
	bridge.root = newRootProvider(bridge)

	// First tree with 2 children.
	tree1 := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 2},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: 2, PrevSibling: -1, ChildCount: 0},
			{ID: 3, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: 1, ChildCount: 0},
		},
	}
	bridge.UpdateTree(tree1)

	// Force provider creation for node 3.
	bridge.mu.Lock()
	bridge.getOrCreateProviderLocked(3)
	bridge.mu.Unlock()

	if len(bridge.providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(bridge.providers))
	}

	// Second tree without node 3.
	tree2 := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 1},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: -1, ChildCount: 0},
		},
	}
	bridge.UpdateTree(tree2)

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if _, exists := bridge.providers[3]; exists {
		t.Error("expected provider for node 3 to be pruned")
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
