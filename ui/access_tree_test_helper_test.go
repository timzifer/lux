package ui

import (
	"testing"

	"github.com/timzifer/lux/a11y"
)

func TestRenderToAccessTree_SingleButton(t *testing.T) {
	tree := RenderToAccessTree(ButtonText("Save", func() {}))

	buttons := tree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	if buttons[0].Node.Label != "Save" {
		t.Errorf("expected label 'Save', got %q", buttons[0].Node.Label)
	}
}

func TestRenderToAccessTree_MultipleWidgets(t *testing.T) {
	tree := RenderToAccessTree(Column(
		ButtonText("A", func() {}),
		Checkbox("Accept", true, nil),
		Slider(0.5, nil),
	))

	buttons := tree.FindByRole(a11y.RoleButton)
	if len(buttons) != 1 {
		t.Fatalf("expected 1 button, got %d", len(buttons))
	}
	checkboxes := tree.FindByRole(a11y.RoleCheckbox)
	if len(checkboxes) != 1 {
		t.Fatalf("expected 1 checkbox, got %d", len(checkboxes))
	}
	if !checkboxes[0].Node.States.Checked {
		t.Error("expected checkbox to be checked")
	}
	sliders := tree.FindByRole(a11y.RoleSlider)
	if len(sliders) != 1 {
		t.Fatalf("expected 1 slider, got %d", len(sliders))
	}
}

func TestRenderToAccessTree_EmptyView(t *testing.T) {
	tree := RenderToAccessTree(Empty())

	// Always has synthetic root.
	if len(tree.Nodes) != 1 {
		t.Errorf("expected 1 node (synthetic root), got %d", len(tree.Nodes))
	}
	if tree.Root().Node.Role != a11y.RoleGroup {
		t.Errorf("expected root role RoleGroup, got %d", tree.Root().Node.Role)
	}
}

func TestRenderToAccessTree_NestedLayout(t *testing.T) {
	tree := RenderToAccessTree(Column(
		Row(
			ButtonText("X", func() {}),
			ButtonText("Y", func() {}),
		),
		Text("Footer"),
	))

	buttons := tree.FindByRole(a11y.RoleButton)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(buttons))
	}
	footers := tree.FindByLabel("Footer")
	if len(footers) != 1 {
		t.Fatalf("expected 1 node with label 'Footer', got %d", len(footers))
	}
}

func TestRenderToAccessTree_TextFieldWithValue(t *testing.T) {
	tree := RenderToAccessTree(TextField("hello", "Enter text..."))

	inputs := tree.FindByRole(a11y.RoleTextInput)
	if len(inputs) != 1 {
		t.Fatalf("expected 1 text input, got %d", len(inputs))
	}
	if inputs[0].Node.Value != "hello" {
		t.Errorf("expected value 'hello', got %q", inputs[0].Node.Value)
	}
	if inputs[0].Node.TextState == nil {
		t.Fatal("expected TextState to be non-nil")
	}
	if inputs[0].Node.TextState.Length != 5 {
		t.Errorf("expected text length 5, got %d", inputs[0].Node.TextState.Length)
	}
}

func TestRenderToAccessTree_ProgressBar(t *testing.T) {
	tree := RenderToAccessTree(ProgressBar(0.75))

	bars := tree.FindByRole(a11y.RoleProgressBar)
	if len(bars) != 1 {
		t.Fatalf("expected 1 progress bar, got %d", len(bars))
	}
	if bars[0].Node.NumericValue == nil {
		t.Fatal("expected NumericValue to be non-nil")
	}
	if bars[0].Node.NumericValue.Current != 0.75 {
		t.Errorf("expected current value 0.75, got %f", bars[0].Node.NumericValue.Current)
	}
}

func TestRenderToAccessTree_Toggle(t *testing.T) {
	tree := RenderToAccessTree(Toggle(true, nil))

	toggles := tree.FindByRole(a11y.RoleToggle)
	if len(toggles) != 1 {
		t.Fatalf("expected 1 toggle, got %d", len(toggles))
	}
	if !toggles[0].Node.States.Checked {
		t.Error("expected toggle to be checked")
	}
}
