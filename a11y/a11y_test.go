package a11y

import (
	"testing"

	"golang.org/x/text/language"
)

func TestAccessRoleConstants(t *testing.T) {
	// Verify roles are distinct and ordered.
	roles := []AccessRole{
		RoleButton, RoleCheckbox, RoleCombobox, RoleDialog, RoleGrid,
		RoleGroup, RoleHeading, RoleImage, RoleLink, RoleListbox,
		RoleMenu, RoleProgressBar, RoleScrollBar, RoleSlider,
		RoleSpinButton, RoleTab, RoleTable, RoleTextInput,
		RoleToggle, RoleTree,
	}
	seen := make(map[AccessRole]bool)
	for _, r := range roles {
		if seen[r] {
			t.Errorf("duplicate role value: %d", r)
		}
		seen[r] = true
	}

	// All standard roles must be below RoleCustomBase.
	for _, r := range roles {
		if r >= RoleCustomBase {
			t.Errorf("standard role %d >= RoleCustomBase (%d)", r, RoleCustomBase)
		}
	}
}

func TestRoleCustomBase(t *testing.T) {
	custom1 := RoleCustomBase + 1
	custom2 := RoleCustomBase + 2
	if custom1 == custom2 {
		t.Error("custom roles should be distinct")
	}
	if custom1 < RoleCustomBase {
		t.Error("custom role should be >= RoleCustomBase")
	}
}

func TestAccessLiveRegion(t *testing.T) {
	if LiveOff != 0 {
		t.Errorf("LiveOff should be 0, got %d", LiveOff)
	}
	if LivePolite == LiveAssertive {
		t.Error("LivePolite and LiveAssertive should be distinct")
	}
}

func TestAccessStatesZeroValue(t *testing.T) {
	var s AccessStates
	if s.Focused || s.Checked || s.Selected || s.Expanded ||
		s.Disabled || s.ReadOnly || s.Required || s.Invalid || s.Busy {
		t.Error("zero-value AccessStates should have all booleans false")
	}
	if s.Live != LiveOff {
		t.Error("zero-value Live should be LiveOff")
	}
}

func TestAccessNode(t *testing.T) {
	node := AccessNode{
		Role:        RoleButton,
		Label:       "Save",
		Description: "Saves the current document",
		Value:       "",
		Lang:        language.German,
		States: AccessStates{
			Focused: true,
		},
		Actions: []AccessAction{
			{Name: "activate"},
		},
		Relations: []AccessRelation{
			{Kind: RelationLabelledBy, TargetID: 42},
		},
	}

	if node.Role != RoleButton {
		t.Errorf("expected RoleButton, got %d", node.Role)
	}
	if node.Label != "Save" {
		t.Errorf("expected label 'Save', got %q", node.Label)
	}
	if !node.States.Focused {
		t.Error("expected Focused=true")
	}
	if node.Lang != language.German {
		t.Errorf("expected German language tag, got %v", node.Lang)
	}
	if len(node.Actions) != 1 || node.Actions[0].Name != "activate" {
		t.Error("expected one action named 'activate'")
	}
	if len(node.Relations) != 1 || node.Relations[0].Kind != RelationLabelledBy {
		t.Error("expected one LabelledBy relation")
	}
}

func TestAccessActionDesc(t *testing.T) {
	desc := AccessActionDesc{Name: "increment"}
	if desc.Name != "increment" {
		t.Errorf("expected 'increment', got %q", desc.Name)
	}
}

func TestAccessRelationDesc(t *testing.T) {
	desc := AccessRelationDesc{
		Kind:     RelationControls,
		TargetID: 99,
	}
	if desc.Kind != RelationControls {
		t.Errorf("expected RelationControls, got %d", desc.Kind)
	}
	if desc.TargetID != 99 {
		t.Errorf("expected target 99, got %d", desc.TargetID)
	}
}

func TestAccessRelationKindConstants(t *testing.T) {
	kinds := []AccessRelationKind{
		RelationLabelledBy, RelationDescribedBy, RelationControls, RelationFlowsTo,
	}
	seen := make(map[AccessRelationKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate relation kind: %d", k)
		}
		seen[k] = true
	}
}
