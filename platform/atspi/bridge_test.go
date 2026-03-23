package atspi

import (
	"testing"

	"github.com/timzifer/lux/a11y"
)

func TestMapRole(t *testing.T) {
	tests := []struct {
		role     a11y.AccessRole
		expected uint32
	}{
		{a11y.RoleButton, rolePushButton},
		{a11y.RoleCheckbox, roleCheckBox},
		{a11y.RoleCombobox, roleComboBox},
		{a11y.RoleDialog, roleDialog},
		{a11y.RoleGrid, roleTable},
		{a11y.RoleGroup, rolePanel},
		{a11y.RoleHeading, roleHeading},
		{a11y.RoleImage, roleImage},
		{a11y.RoleLink, roleLink},
		{a11y.RoleListbox, roleList},
		{a11y.RoleMenu, roleMenu},
		{a11y.RoleProgressBar, roleProgressBar},
		{a11y.RoleScrollBar, roleScrollBar},
		{a11y.RoleSlider, roleSlider},
		{a11y.RoleSpinButton, roleSpinButton},
		{a11y.RoleTab, rolePageTab},
		{a11y.RoleTable, roleTable},
		{a11y.RoleTextInput, roleText},
		{a11y.RoleToggle, roleToggleButton},
		{a11y.RoleTree, roleTreeView},
		{a11y.RoleCustomBase, roleUnknown},
		{a11y.RoleCustomBase + 5, roleUnknown},
	}
	for _, tt := range tests {
		got := mapRole(tt.role)
		if got != tt.expected {
			t.Errorf("mapRole(%d) = %d, want %d", tt.role, got, tt.expected)
		}
	}
}

func TestMapStates(t *testing.T) {
	// Default (empty states): visible, showing, sensitive, enabled.
	bits := mapStates(a11y.AccessStates{})
	if bits[stateVisible/32]&(1<<(stateVisible%32)) == 0 {
		t.Error("expected visible state set")
	}
	if bits[stateShowing/32]&(1<<(stateShowing%32)) == 0 {
		t.Error("expected showing state set")
	}
	if bits[stateEnabled/32]&(1<<(stateEnabled%32)) == 0 {
		t.Error("expected enabled state set")
	}

	// Focused state.
	bits = mapStates(a11y.AccessStates{Focused: true})
	if bits[stateFocused/32]&(1<<(stateFocused%32)) == 0 {
		t.Error("expected focused state set")
	}

	// Checked state.
	bits = mapStates(a11y.AccessStates{Checked: true})
	if bits[stateChecked/32]&(1<<(stateChecked%32)) == 0 {
		t.Error("expected checked state set")
	}

	// Disabled clears enabled and sensitive.
	bits = mapStates(a11y.AccessStates{Disabled: true})
	if bits[stateEnabled/32]&(1<<(stateEnabled%32)) != 0 {
		t.Error("expected enabled state cleared when disabled")
	}
	if bits[stateSensitive/32]&(1<<(stateSensitive%32)) != 0 {
		t.Error("expected sensitive state cleared when disabled")
	}

	// Expanded sets both expandable and expanded.
	bits = mapStates(a11y.AccessStates{Expanded: true})
	if bits[stateExpanded/32]&(1<<(stateExpanded%32)) == 0 {
		t.Error("expected expanded state set")
	}
	if bits[stateExpandable/32]&(1<<(stateExpandable%32)) == 0 {
		t.Error("expected expandable state set when expanded")
	}
}

func TestObjectPath(t *testing.T) {
	path := objectPath(42)
	expected := "/org/lux/accessible/42"
	if string(path) != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestAccessibleObject_LookupNil(t *testing.T) {
	b := &ATSPIBridge{
		tree:    a11y.AccessTree{},
		objects: make(map[a11y.AccessNodeID]*accessibleObject),
	}
	obj := &accessibleObject{bridge: b, nodeID: 99}
	node := obj.lookupNode()
	if node != nil {
		t.Error("expected nil for non-existent node")
	}
}

func TestAccessibleObject_GetRole(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node: a11y.AccessNode{Role: a11y.RoleButton, Label: "OK"}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{
		tree:    tree,
		objects: make(map[a11y.AccessNodeID]*accessibleObject),
	}
	obj := &accessibleObject{bridge: b, nodeID: 1, path: objectPath(1)}

	role, err := obj.GetRole()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != rolePushButton {
		t.Errorf("expected rolePushButton (%d), got %d", rolePushButton, role)
	}

	name, err := obj.GetName()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "OK" {
		t.Errorf("expected 'OK', got %q", name)
	}
}

func TestAccessibleObject_GetState(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node: a11y.AccessNode{
					Role:   a11y.RoleCheckbox,
					States: a11y.AccessStates{Checked: true, Focused: true},
				}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}
	obj := &accessibleObject{bridge: b, nodeID: 1}

	state, err := obj.GetState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state[stateChecked/32]&(1<<(stateChecked%32)) == 0 {
		t.Error("expected checked state")
	}
	if state[stateFocused/32]&(1<<(stateFocused%32)) == 0 {
		t.Error("expected focused state")
	}
}

func TestAccessibleObject_GetText(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node: a11y.AccessNode{
					Role:  a11y.RoleTextInput,
					Value: "Hello World",
					TextState: &a11y.AccessTextState{
						Length:      11,
						CaretOffset: 5,
					},
				}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}
	obj := &accessibleObject{bridge: b, nodeID: 1}

	text, err := obj.GetText(0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello" {
		t.Errorf("expected 'Hello', got %q", text)
	}

	caret, err := obj.GetCaretOffset()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caret != 5 {
		t.Errorf("expected caret 5, got %d", caret)
	}

	count, err := obj.GetCharacterCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 11 {
		t.Errorf("expected count 11, got %d", count)
	}
}

func TestAccessibleObject_ChildNavigation(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: 1, NextSibling: -1, PrevSibling: -1, ChildCount: 2,
				Node: a11y.AccessNode{Role: a11y.RoleGroup}},
			{ID: 2, ParentIndex: 0, FirstChild: -1, NextSibling: 2, PrevSibling: -1,
				Node: a11y.AccessNode{Role: a11y.RoleButton, Label: "A"}},
			{ID: 3, ParentIndex: 0, FirstChild: -1, NextSibling: -1, PrevSibling: 1,
				Node: a11y.AccessNode{Role: a11y.RoleButton, Label: "B"}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}

	root := &accessibleObject{bridge: b, nodeID: 1}
	count, err := root.GetChildCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 children, got %d", count)
	}

	childPath, err := root.GetChildAtIndex(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(childPath) != "/org/lux/accessible/2" {
		t.Errorf("expected child path /org/lux/accessible/2, got %s", childPath)
	}

	child := &accessibleObject{bridge: b, nodeID: 2}
	parentPath, err := child.GetParent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(parentPath) != "/org/lux/accessible/1" {
		t.Errorf("expected parent path /org/lux/accessible/1, got %s", parentPath)
	}

	idx, err := child.GetIndexInParent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestAccessibleObject_ValueInterface(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node: a11y.AccessNode{
					Role: a11y.RoleSlider,
					NumericValue: &a11y.AccessNumericValue{
						Current: 0.5, Min: 0, Max: 1, Step: 0.1,
					},
				}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}
	obj := &accessibleObject{bridge: b, nodeID: 1}

	cur, _ := obj.GetCurrentValue()
	if cur != 0.5 {
		t.Errorf("expected current value 0.5, got %f", cur)
	}
	min, _ := obj.GetMinimumValue()
	if min != 0 {
		t.Errorf("expected min 0, got %f", min)
	}
	max, _ := obj.GetMaximumValue()
	if max != 1 {
		t.Errorf("expected max 1, got %f", max)
	}
}

func TestAccessibleObject_ActionInterface(t *testing.T) {
	triggered := false
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node: a11y.AccessNode{
					Role: a11y.RoleButton,
					Actions: []a11y.AccessAction{
						{Name: "activate", Trigger: func() { triggered = true }},
					},
				}},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}
	obj := &accessibleObject{bridge: b, nodeID: 1}

	n, _ := obj.GetNActions()
	if n != 1 {
		t.Fatalf("expected 1 action, got %d", n)
	}

	name, _ := obj.GetActionName(0)
	if name != "activate" {
		t.Errorf("expected 'activate', got %q", name)
	}

	ok, _ := obj.DoAction(0)
	if !ok {
		t.Error("expected DoAction to succeed")
	}
	if !triggered {
		t.Error("expected action trigger to fire")
	}
}

func TestAccessibleObject_ComponentInterface(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{ID: 1, ParentIndex: -1, FirstChild: -1, NextSibling: -1, PrevSibling: -1,
				Node:   a11y.AccessNode{Role: a11y.RoleButton},
				Bounds: a11y.Rect{X: 10, Y: 20, Width: 100, Height: 50},
			},
		},
	}
	tree.EnsureIndex()

	b := &ATSPIBridge{tree: tree, objects: make(map[a11y.AccessNodeID]*accessibleObject)}
	obj := &accessibleObject{bridge: b, nodeID: 1}

	x, y, w, h, _ := obj.GetExtents(0)
	if x != 10 || y != 20 || w != 100 || h != 50 {
		t.Errorf("expected extents (10,20,100,50), got (%d,%d,%d,%d)", x, y, w, h)
	}

	ok, _ := obj.Contains(50, 30, 0)
	if !ok {
		t.Error("expected point (50,30) to be contained")
	}

	ok, _ = obj.Contains(200, 200, 0)
	if ok {
		t.Error("expected point (200,200) to not be contained")
	}
}
