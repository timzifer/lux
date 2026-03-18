//go:build nogui

package app

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
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

// m2HelloView is the M2 hello world view.
func m2HelloView(m testModel) ui.Element {
	return ui.Column(
		ui.Text("HELLO WORLD"),
		ui.Button("CLICK ME", nil),
	)
}

func TestRunHeadless(t *testing.T) {
	err := Run(testModel{}, testUpdate, testView,
		WithTitle("test"),
		WithSize(320, 240),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestSendBeforeRun(t *testing.T) {
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

	err := Run(testModel{Count: 0}, update, testView,
		WithTitle("msg-test"),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	_ = finalCount
}

func TestM2HelloWorldScene(t *testing.T) {
	canvas := render.NewSceneCanvas(800, 600)
	scene := ui.BuildScene(m2HelloView(testModel{}), canvas, theme.Default, 800, 600)

	if len(scene.Rects) < 2 {
		t.Errorf("M2 scene should have at least 2 rects (button edge+fill), got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 2 {
		t.Fatalf("M2 scene should have at least 2 glyphs (label+button), got %d", len(scene.Glyphs))
	}

	foundLabel := false
	foundButton := false
	for _, g := range scene.Glyphs {
		if g.Text == "HELLO WORLD" {
			foundLabel = true
		}
		if g.Text == "CLICK ME" {
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
	if frameCount < 1 {
		t.Error("view was never called")
	}
}

func TestWithThemeOption(t *testing.T) {
	err := Run(testModel{}, testUpdate, testView,
		WithTheme(theme.Default),
		WithTitle("theme-test"),
	)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestCanvasInterface(t *testing.T) {
	// Verify that SceneCanvas implements draw.Canvas.
	canvas := render.NewSceneCanvas(800, 600)
	var _ draw.Canvas = canvas

	// Draw some primitives and verify scene output.
	canvas.FillRect(draw.R(10, 20, 100, 50), draw.SolidPaint(draw.RGBA(255, 0, 0, 255)))
	canvas.DrawText("TEST", draw.Pt(10, 20), draw.TextStyle{Size: 21}, draw.RGBA(255, 255, 255, 255))

	scene := canvas.Scene()
	if len(scene.Rects) != 1 {
		t.Errorf("expected 1 rect, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 1 {
		t.Errorf("expected 1 glyph, got %d", len(scene.Glyphs))
	}
}
