//go:build nogui

package app

import "testing"

func TestWindowTypes(t *testing.T) {
	if MainWindow != 0 {
		t.Errorf("MainWindow should be 0, got %d", MainWindow)
	}

	cfg := WindowConfig{Title: "Test", Width: 640, Height: 480}
	if cfg.Title != "Test" || cfg.Width != 640 || cfg.Height != 480 {
		t.Errorf("unexpected WindowConfig: %+v", cfg)
	}
}

func TestOpenWindowCmd(t *testing.T) {
	cmd := OpenWindow(1, WindowConfig{Title: "Second", Width: 400, Height: 300})
	if cmd == nil {
		t.Fatal("OpenWindow should return a non-nil Cmd")
	}
	msg := cmd()
	owm, ok := msg.(OpenWindowMsg)
	if !ok {
		t.Fatalf("expected OpenWindowMsg, got %T", msg)
	}
	if owm.ID != 1 {
		t.Errorf("expected window ID 1, got %d", owm.ID)
	}
	if owm.Config.Title != "Second" {
		t.Errorf("expected title 'Second', got %q", owm.Config.Title)
	}
}

func TestCloseWindowCmd(t *testing.T) {
	cmd := CloseWindow(2)
	if cmd == nil {
		t.Fatal("CloseWindow should return a non-nil Cmd")
	}
	msg := cmd()
	cwm, ok := msg.(CloseWindowMsg)
	if !ok {
		t.Fatalf("expected CloseWindowMsg, got %T", msg)
	}
	if cwm.ID != 2 {
		t.Errorf("expected window ID 2, got %d", cwm.ID)
	}
}

func TestWindowOpenedClosedMsg(t *testing.T) {
	opened := WindowOpenedMsg{Window: 3}
	if opened.Window != 3 {
		t.Errorf("expected 3, got %d", opened.Window)
	}
	closed := WindowClosedMsg{Window: 3}
	if closed.Window != 3 {
		t.Errorf("expected 3, got %d", closed.Window)
	}
}

func TestMainWindowIsZero(t *testing.T) {
	var id WindowID = MainWindow
	if id != 0 {
		t.Errorf("MainWindow should equal 0")
	}
}
