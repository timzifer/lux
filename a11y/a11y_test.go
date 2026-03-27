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

func TestAccessNumericValue(t *testing.T) {
	nv := AccessNumericValue{
		Current: 0.5,
		Min:     0,
		Max:     1,
		Step:    0.1,
	}
	if nv.Current != 0.5 {
		t.Errorf("expected Current=0.5, got %f", nv.Current)
	}
	if nv.Step != 0.1 {
		t.Errorf("expected Step=0.1, got %f", nv.Step)
	}

	// Continuous slider: Step == 0
	continuous := AccessNumericValue{Current: 42, Min: 0, Max: 100, Step: 0}
	if continuous.Step != 0 {
		t.Error("expected Step=0 for continuous value")
	}
}

func TestAccessTextState(t *testing.T) {
	// Single selection
	ts := AccessTextState{
		Length:      12,
		CaretOffset: 5,
		Selections:  []TextSelection{{Start: 3, End: 8}},
	}
	if ts.Length != 12 {
		t.Errorf("expected Length=12, got %d", ts.Length)
	}
	if ts.CaretOffset != 5 {
		t.Errorf("expected CaretOffset=5, got %d", ts.CaretOffset)
	}
	if len(ts.Selections) != 1 {
		t.Fatalf("expected 1 selection, got %d", len(ts.Selections))
	}
	if ts.Selections[0].Start != 3 || ts.Selections[0].End != 8 {
		t.Errorf("expected selection 3..8, got %d..%d", ts.Selections[0].Start, ts.Selections[0].End)
	}

	// No selection / no caret
	noSel := AccessTextState{Length: 0, CaretOffset: -1}
	if noSel.CaretOffset != -1 {
		t.Error("expected CaretOffset=-1 for no caret")
	}
	if len(noSel.Selections) != 0 {
		t.Error("expected empty selections for no selection")
	}
}

func TestAccessTextStateMultiSelection(t *testing.T) {
	// Multi-cursor / column selection: three separate ranges
	ts := AccessTextState{
		Length:      100,
		CaretOffset: 75,
		Selections: []TextSelection{
			{Start: 10, End: 15},
			{Start: 30, End: 35},
			{Start: 70, End: 75},
		},
	}
	if len(ts.Selections) != 3 {
		t.Fatalf("expected 3 selections, got %d", len(ts.Selections))
	}
	// Each range should be 5 runes wide
	for i, sel := range ts.Selections {
		if sel.End-sel.Start != 5 {
			t.Errorf("selection %d: expected width 5, got %d", i, sel.End-sel.Start)
		}
	}
}

func TestAccessNodeNumericValueNil(t *testing.T) {
	node := AccessNode{Role: RoleButton, Label: "Click"}
	if node.NumericValue != nil {
		t.Error("expected nil NumericValue for button")
	}
	if node.TextState != nil {
		t.Error("expected nil TextState for button")
	}
}

func TestAccessNodeWithNumericValue(t *testing.T) {
	node := AccessNode{
		Role: RoleSlider,
		NumericValue: &AccessNumericValue{
			Current: 75,
			Min:     0,
			Max:     100,
			Step:    5,
		},
	}
	if node.NumericValue == nil {
		t.Fatal("expected non-nil NumericValue for slider")
	}
	if node.NumericValue.Current != 75 {
		t.Errorf("expected Current=75, got %f", node.NumericValue.Current)
	}
}

func TestAccessNodeWithTextState(t *testing.T) {
	node := AccessNode{
		Role:  RoleTextInput,
		Value: "Hello",
		TextState: &AccessTextState{
			Length:      5,
			CaretOffset: 5,
		},
	}
	if node.TextState == nil {
		t.Fatal("expected non-nil TextState for text input")
	}
	if node.TextState.Length != 5 {
		t.Errorf("expected Length=5, got %d", node.TextState.Length)
	}
	if len(node.TextState.Selections) != 0 {
		t.Error("expected no selections for caret-only state")
	}
}
