//go:build nogui

package app

import (
	"testing"

	"github.com/timzifer/lux/ui"
)

type testModel struct {
	Count int
}

type incrMsg struct{}

func testUpdate(m testModel, msg Msg) testModel {
	switch msg.(type) {
	case incrMsg:
		m.Count++
	}
	return m
}

func testView(m testModel) ui.Element {
	return ui.Empty()
}

// m2HelloView is the M2 hello world view — must produce visible output.
func m2HelloView(m testModel) ui.Element {
	return ui.Column(
		ui.Text("HELLO WORLD"),
		ui.Button("CLICK ME", nil),
	)
}

func TestRunHeadless(t *testing.T) {
	// In headless mode, Run executes one frame and returns.
	err := Run(testModel{}, testUpdate, testView,
		WithTitle("test"),
		WithSize(320, 240),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestSendBeforeRun(t *testing.T) {
	// Send before Run should not panic.
	Send(incrMsg{})
	TrySend(incrMsg{})
}

func TestRunWithMessages(t *testing.T) {
	var finalCount int

	update := func(m testModel, msg Msg) testModel {
		switch msg.(type) {
		case incrMsg:
			m.Count++
		}
		finalCount = m.Count
		return m
	}

	// Send a message before Run — it will be picked up in the first frame.
	err := Run(testModel{Count: 0}, update, testView,
		WithTitle("msg-test"),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// finalCount may be 0 since we can't easily send messages
	// into the headless run. This test verifies no panics/crashes.
	_ = finalCount
}

func TestM2HelloWorldScene(t *testing.T) {
	// Verify the M2 view produces a non-empty scene with label and button.
	scene := ui.BuildScene(m2HelloView(testModel{}), 800, 600)

	if len(scene.Rects) < 2 {
		t.Errorf("M2 scene should have at least 2 rects (button edge+fill), got %d", len(scene.Rects))
	}
	if len(scene.Texts) < 2 {
		t.Fatalf("M2 scene should have at least 2 texts (label+button), got %d", len(scene.Texts))
	}

	foundLabel := false
	foundButton := false
	for _, txt := range scene.Texts {
		if txt.Text == "HELLO WORLD" {
			foundLabel = true
		}
		if txt.Text == "CLICK ME" {
			foundButton = true
		}
	}
	if !foundLabel {
		t.Error("scene is missing the HELLO WORLD label")
	}
	if !foundButton {
		t.Error("scene is missing the CLICK ME button label")
	}
}

func TestM2RunHeadlessRendersScene(t *testing.T) {
	// Verify the full pipeline: Run with the M2 view should not crash
	// and should execute at least one frame.
	var frameCount int
	view := func(m testModel) ui.Element {
		frameCount++
		return m2HelloView(m)
	}

	err := Run(testModel{}, testUpdate, view,
		WithTitle("M2 test"),
		WithSize(800, 600),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// view is called once during init + potentially once in OnFrame.
	if frameCount < 1 {
		t.Error("view was never called — no frame was rendered")
	}
}
