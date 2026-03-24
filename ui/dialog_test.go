package ui

import (
	"testing"

	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/ui/icons"
)

func TestMessageDialogIsOverlay(t *testing.T) {
	el := MessageDialog("test-msg", "Title", "Hello", platform.DialogInfo, nil)
	ov, ok := el.(Overlay)
	if !ok {
		t.Fatalf("MessageDialog returned %T, want Overlay", el)
	}
	if ov.ID != "test-msg" {
		t.Errorf("ID = %q, want %q", ov.ID, "test-msg")
	}
	if ov.Placement != PlacementCenter {
		t.Errorf("Placement = %d, want PlacementCenter", ov.Placement)
	}
	if !ov.Backdrop {
		t.Error("Backdrop should be true")
	}
	if !ov.Dismissable {
		t.Error("Dismissable should be true")
	}
}

func TestConfirmDialogIsOverlay(t *testing.T) {
	el := ConfirmDialog("test-confirm", "Confirm?", "Are you sure?", nil, nil)
	ov, ok := el.(Overlay)
	if !ok {
		t.Fatalf("ConfirmDialog returned %T, want Overlay", el)
	}
	if ov.ID != "test-confirm" {
		t.Errorf("ID = %q, want %q", ov.ID, "test-confirm")
	}
	if !ov.Backdrop {
		t.Error("Backdrop should be true")
	}
}

func TestInputDialogIsOverlay(t *testing.T) {
	el := InputDialog("test-input", "Enter", "Value:", "default", "placeholder", nil, nil, nil)
	ov, ok := el.(Overlay)
	if !ok {
		t.Fatalf("InputDialog returned %T, want Overlay", el)
	}
	if ov.ID != "test-input" {
		t.Errorf("ID = %q, want %q", ov.ID, "test-input")
	}
	if !ov.Backdrop {
		t.Error("Backdrop should be true")
	}
}

func TestMessageDialogRenders(t *testing.T) {
	el := MessageDialog("msg", "Alert", "Something happened", platform.DialogInfo, func() {})
	// Should not panic when building a scene.
	scene := buildTestScene(el, 800, 600)
	// Overlay content is rendered in a deferred pass, so scene may have zero
	// items in the main pass. Just verify it completes without panic.
	_ = scene
}

func TestConfirmDialogRenders(t *testing.T) {
	el := ConfirmDialog("cfm", "Confirm", "Proceed?", func() {}, func() {})
	scene := buildTestScene(el, 800, 600)
	_ = scene
}

func TestInputDialogRenders(t *testing.T) {
	el := InputDialog("inp", "Input", "Name:", "", "Enter name", func(string) {}, func() {}, func() {})
	scene := buildTestScene(el, 800, 600)
	_ = scene
}

func TestDialogPhosphorIconInfo(t *testing.T) {
	if icon := dialogPhosphorIcon(platform.DialogInfo); icon != icons.Info {
		t.Errorf("DialogInfo icon = %q, want icons.Info %q", icon, icons.Info)
	}
}

func TestDialogPhosphorIconWarning(t *testing.T) {
	if icon := dialogPhosphorIcon(platform.DialogWarning); icon != icons.Warning {
		t.Errorf("DialogWarning icon = %q, want icons.Warning %q", icon, icons.Warning)
	}
}

func TestDialogPhosphorIconError(t *testing.T) {
	if icon := dialogPhosphorIcon(platform.DialogError); icon != icons.X {
		t.Errorf("DialogError icon = %q, want icons.X %q", icon, icons.X)
	}
}
