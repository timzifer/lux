package dialog

import (
	"testing"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/platform"
)

// mockPlatform implements platform.Platform with NativeDialogProvider.
type mockPlatform struct {
	platform.Platform // embed to satisfy interface
	messageErr        error
	confirmResult     bool
	confirmErr        error
	inputValue        string
	inputConfirmed    bool
	inputErr          error
}

func (m *mockPlatform) ShowMessageDialog(_, _ string, _ platform.DialogKind) error {
	return m.messageErr
}

func (m *mockPlatform) ShowConfirmDialog(_, _ string) (bool, error) {
	return m.confirmResult, m.confirmErr
}

func (m *mockPlatform) ShowInputDialog(_, _, _ string) (string, bool, error) {
	return m.inputValue, m.inputConfirmed, m.inputErr
}

// stubPlatform does NOT implement NativeDialogProvider.
type stubPlatform struct {
	platform.Platform
}

func TestShowMessageNativePath(t *testing.T) {
	app.SetActivePlatformForTest(&mockPlatform{})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowMessage("Title", "Msg", platform.DialogInfo)
	result := cmd()

	if _, ok := result.(MessageResultMsg); !ok {
		t.Fatalf("expected MessageResultMsg, got %T", result)
	}
}

func TestShowMessageFallback(t *testing.T) {
	app.SetActivePlatformForTest(&stubPlatform{})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowMessage("Title", "Msg", platform.DialogWarning)
	result := cmd()

	msg, ok := result.(ShowFallbackMessageMsg)
	if !ok {
		t.Fatalf("expected ShowFallbackMessageMsg, got %T", result)
	}
	if msg.Title != "Title" {
		t.Errorf("Title = %q, want %q", msg.Title, "Title")
	}
	if msg.Kind != platform.DialogWarning {
		t.Errorf("Kind = %d, want %d", msg.Kind, platform.DialogWarning)
	}
}

func TestShowConfirmNativePath(t *testing.T) {
	app.SetActivePlatformForTest(&mockPlatform{confirmResult: true})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowConfirm("Title", "Msg")
	result := cmd()

	msg, ok := result.(ConfirmResultMsg)
	if !ok {
		t.Fatalf("expected ConfirmResultMsg, got %T", result)
	}
	if !msg.Confirmed {
		t.Error("Confirmed should be true")
	}
}

func TestShowConfirmFallback(t *testing.T) {
	app.SetActivePlatformForTest(&stubPlatform{})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowConfirm("Title", "Msg")
	result := cmd()

	if _, ok := result.(ShowFallbackConfirmMsg); !ok {
		t.Fatalf("expected ShowFallbackConfirmMsg, got %T", result)
	}
}

func TestShowInputNativePath(t *testing.T) {
	app.SetActivePlatformForTest(&mockPlatform{inputValue: "hello", inputConfirmed: true})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowInput("Title", "Msg", "default")
	result := cmd()

	msg, ok := result.(InputResultMsg)
	if !ok {
		t.Fatalf("expected InputResultMsg, got %T", result)
	}
	if msg.Value != "hello" {
		t.Errorf("Value = %q, want %q", msg.Value, "hello")
	}
	if !msg.Confirmed {
		t.Error("Confirmed should be true")
	}
}

func TestShowInputFallback(t *testing.T) {
	app.SetActivePlatformForTest(&stubPlatform{})
	defer app.SetActivePlatformForTest(nil)

	cmd := ShowInput("Title", "Msg", "default")
	result := cmd()

	msg, ok := result.(ShowFallbackInputMsg)
	if !ok {
		t.Fatalf("expected ShowFallbackInputMsg, got %T", result)
	}
	if msg.DefaultValue != "default" {
		t.Errorf("DefaultValue = %q, want %q", msg.DefaultValue, "default")
	}
}

func TestShowMessageNilPlatform(t *testing.T) {
	app.SetActivePlatformForTest(nil)

	cmd := ShowMessage("Title", "Msg", platform.DialogInfo)
	result := cmd()

	if _, ok := result.(ShowFallbackMessageMsg); !ok {
		t.Fatalf("expected ShowFallbackMessageMsg with nil platform, got %T", result)
	}
}
