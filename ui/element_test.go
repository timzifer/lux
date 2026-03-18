package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
)

func buildTestScene(root Element, w, h int) draw.Scene {
	canvas := render.NewSceneCanvas(w, h)
	return BuildScene(root, canvas, theme.Default, w, h, nil)
}

func TestBuildSceneEmpty(t *testing.T) {
	scene := buildTestScene(Empty(), 800, 600)
	if len(scene.Rects) != 0 {
		t.Errorf("Empty element should produce 0 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 0 {
		t.Errorf("Empty element should produce 0 glyphs, got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneText(t *testing.T) {
	scene := buildTestScene(Text("HELLO WORLD"), 800, 600)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("Text element should produce 1 glyph entry, got %d", len(scene.Glyphs))
	}
	glyph := scene.Glyphs[0]
	if glyph.Text != "HELLO WORLD" {
		t.Errorf("glyph text = %q, want %q", glyph.Text, "HELLO WORLD")
	}
	if glyph.X != framePadding {
		t.Errorf("glyph X = %d, want %d", glyph.X, framePadding)
	}
	if glyph.Y != framePadding {
		t.Errorf("glyph Y = %d, want %d", glyph.Y, framePadding)
	}
}

func TestBuildSceneButton(t *testing.T) {
	scene := buildTestScene(Button("OK", nil), 800, 600)

	// Button: 2 rects (edge + fill) + 1 glyph (label).
	if len(scene.Rects) != 2 {
		t.Fatalf("Button should produce 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 1 {
		t.Fatalf("Button should produce 1 glyph, got %d", len(scene.Glyphs))
	}

	edge := scene.Rects[0]
	fill := scene.Rects[1]
	label := scene.Glyphs[0]

	if edge.X != framePadding || edge.Y != framePadding {
		t.Errorf("edge origin = (%d,%d), want (%d,%d)", edge.X, edge.Y, framePadding, framePadding)
	}

	if fill.X != framePadding+buttonBorder || fill.Y != framePadding+buttonBorder {
		t.Errorf("fill origin = (%d,%d), want (%d,%d)", fill.X, fill.Y, framePadding+buttonBorder, framePadding+buttonBorder)
	}
	if fill.W != edge.W-buttonBorder*2 || fill.H != edge.H-buttonBorder*2 {
		t.Errorf("fill size = %dx%d, want %dx%d", fill.W, fill.H, edge.W-buttonBorder*2, edge.H-buttonBorder*2)
	}

	if label.Text != "OK" {
		t.Errorf("label text = %q, want %q", label.Text, "OK")
	}

	// Label inside button bounds.
	if label.X < edge.X || label.X >= edge.X+edge.W {
		t.Errorf("label X=%d outside button [%d, %d)", label.X, edge.X, edge.X+edge.W)
	}
	if label.Y < edge.Y || label.Y >= edge.Y+edge.H {
		t.Errorf("label Y=%d outside button [%d, %d)", label.Y, edge.Y, edge.Y+edge.H)
	}
}

func TestBuildSceneColumnTextAndButton(t *testing.T) {
	scene := buildTestScene(Column(
		Text("HELLO WORLD"),
		Button("CLICK ME", nil),
	), 800, 600)

	if len(scene.Rects) != 2 {
		t.Errorf("M2 scene should have 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 2 {
		t.Errorf("M2 scene should have 2 glyphs, got %d", len(scene.Glyphs))
	}
	if len(scene.Glyphs) < 2 {
		t.FailNow()
	}

	hello := scene.Glyphs[0]
	click := scene.Glyphs[1]

	if hello.Text != "HELLO WORLD" {
		t.Errorf("first text = %q, want %q", hello.Text, "HELLO WORLD")
	}
	if click.Text != "CLICK ME" {
		t.Errorf("second text = %q, want %q", click.Text, "CLICK ME")
	}

	if click.Y <= hello.Y {
		t.Errorf("button label Y=%d should be below text Y=%d", click.Y, hello.Y)
	}

	for _, g := range scene.Glyphs {
		if g.X < 0 || g.Y < 0 || g.X >= 800 || g.Y >= 600 {
			t.Errorf("glyph %q at (%d,%d) outside 800x600", g.Text, g.X, g.Y)
		}
	}
	for _, r := range scene.Rects {
		if r.X < 0 || r.Y < 0 || r.X+r.W > 800 || r.Y+r.H > 600 {
			t.Errorf("rect at (%d,%d) %dx%d outside 800x600", r.X, r.Y, r.W, r.H)
		}
	}
}

func TestBuildSceneRow(t *testing.T) {
	scene := buildTestScene(Row(
		Text("A"),
		Text("B"),
	), 800, 600)

	if len(scene.Glyphs) != 2 {
		t.Fatalf("Row with 2 texts should produce 2 glyphs, got %d", len(scene.Glyphs))
	}

	a := scene.Glyphs[0]
	b := scene.Glyphs[1]

	if a.Y != b.Y {
		t.Errorf("Row children should share Y: a.Y=%d, b.Y=%d", a.Y, b.Y)
	}
	if b.X <= a.X {
		t.Errorf("b.X=%d should be > a.X=%d", b.X, a.X)
	}
}

func TestBuildSceneDefaultSize(t *testing.T) {
	scene := buildTestScene(Text("X"), 0, 0)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("expected 1 glyph, got %d", len(scene.Glyphs))
	}
	if scene.Glyphs[0].X != framePadding {
		t.Errorf("X = %d, want %d", scene.Glyphs[0].X, framePadding)
	}
}

func TestWithKey(t *testing.T) {
	scene := buildTestScene(WithKey("test", Text("KEYED")), 800, 600)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("WithKey should render child: got %d glyphs", len(scene.Glyphs))
	}
	if scene.Glyphs[0].Text != "KEYED" {
		t.Errorf("glyph text = %q, want %q", scene.Glyphs[0].Text, "KEYED")
	}
}

func TestBuildSceneCollectsHitTargets(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Column(
		Text("Label"),
		Button("OK", func() {}),
	), canvas, theme.Default, 800, 600, &hitMap)

	if hitMap.Len() != 1 {
		t.Fatalf("expected 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneHitTargetNilOnClick(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Button("X", nil), canvas, theme.Default, 800, 600, &hitMap)

	if hitMap.Len() != 0 {
		t.Errorf("nil OnClick should not register hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneMultipleHitTargets(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Row(
		Button("A", func() {}),
		Button("B", func() {}),
	), canvas, theme.Default, 800, 600, &hitMap)

	if hitMap.Len() != 2 {
		t.Fatalf("expected 2 hit targets, got %d", hitMap.Len())
	}
}

func TestHitTargetClickable(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	clicked := false
	BuildScene(Button("OK", func() { clicked = true }), canvas, theme.Default, 800, 600, &hitMap)

	// Button is at framePadding (24) position, with buttonMinWidth (180) and some height.
	target := hitMap.HitTest(float32(framePadding+10), float32(framePadding+5))
	if target == nil {
		t.Fatal("expected hit target at button position")
	}
	target.OnClick()
	if !clicked {
		t.Error("OnClick was not invoked")
	}
}

func TestThemeColorsUsed(t *testing.T) {
	scene := buildTestScene(Button("X", nil), 800, 600)
	tokens := theme.Default.Tokens()

	if len(scene.Rects) < 2 {
		t.Fatal("need at least 2 rects")
	}

	// Edge should use Outline color.
	edge := scene.Rects[0]
	if edge.Color != tokens.Colors.Outline {
		t.Errorf("edge color = %v, want Outline %v", edge.Color, tokens.Colors.Outline)
	}

	// Fill should use Primary color.
	fill := scene.Rects[1]
	if fill.Color != tokens.Colors.Primary {
		t.Errorf("fill color = %v, want Primary %v", fill.Color, tokens.Colors.Primary)
	}

	// Label should use OnPrimary color.
	if len(scene.Glyphs) < 1 {
		t.Fatal("need at least 1 glyph")
	}
	label := scene.Glyphs[0]
	if label.Color != tokens.Colors.OnPrimary {
		t.Errorf("label color = %v, want OnPrimary %v", label.Color, tokens.Colors.OnPrimary)
	}
}
