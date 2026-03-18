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
	return BuildScene(root, canvas, theme.Default, w, h, nil, nil)
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
	), canvas, theme.Default, 800, 600, &hitMap, nil)

	if hitMap.Len() != 1 {
		t.Fatalf("expected 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneHitTargetNilOnClick(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Button("X", nil), canvas, theme.Default, 800, 600, &hitMap, nil)

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
	), canvas, theme.Default, 800, 600, &hitMap, nil)

	if hitMap.Len() != 2 {
		t.Fatalf("expected 2 hit targets, got %d", hitMap.Len())
	}
}

func TestHitTargetClickable(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	clicked := false
	BuildScene(Button("OK", func() { clicked = true }), canvas, theme.Default, 800, 600, &hitMap, nil)

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

	// Edge should use Stroke.Border color.
	edge := scene.Rects[0]
	if edge.Color != tokens.Colors.Stroke.Border {
		t.Errorf("edge color = %v, want Stroke.Border %v", edge.Color, tokens.Colors.Stroke.Border)
	}

	// Fill should use Accent.Primary color.
	fill := scene.Rects[1]
	if fill.Color != tokens.Colors.Accent.Primary {
		t.Errorf("fill color = %v, want Accent.Primary %v", fill.Color, tokens.Colors.Accent.Primary)
	}

	// Label should use Text.OnAccent color.
	if len(scene.Glyphs) < 1 {
		t.Fatal("need at least 1 glyph")
	}
	label := scene.Glyphs[0]
	if label.Color != tokens.Colors.Text.OnAccent {
		t.Errorf("label color = %v, want Text.OnAccent %v", label.Color, tokens.Colors.Text.OnAccent)
	}
}

// ── M4 Hover Tests ──────────────────────────────────────────────

func TestBuildSceneWithHoverState(t *testing.T) {
	// Simulate a fully hovered button: fill color should differ from Primary.
	var hover HoverState
	hover.SetHovered(0, 0) // instant
	hover.Tick(0)          // ensure the anim completes

	canvas := render.NewSceneCanvas(800, 600)
	scene := BuildScene(Button("OK", nil), canvas, theme.Default, 800, 600, nil, &hover)

	tokens := theme.Default.Tokens()

	// With hover at 1.0, the fill color should be a lightened Primary, not raw Primary.
	if len(scene.Rects) < 2 {
		t.Fatal("need at least 2 rects for button")
	}
	fill := scene.Rects[1]
	if fill.Color == tokens.Colors.Accent.Primary {
		t.Error("hovered button fill should differ from raw Accent.Primary")
	}
}

func TestBuildSceneNilHoverState(t *testing.T) {
	// nil HoverState should render normally without panic.
	canvas := render.NewSceneCanvas(800, 600)
	scene := BuildScene(Button("OK", nil), canvas, theme.Default, 800, 600, nil, nil)

	tokens := theme.Default.Tokens()
	if len(scene.Rects) < 2 {
		t.Fatal("need at least 2 rects")
	}
	fill := scene.Rects[1]
	if fill.Color != tokens.Colors.Accent.Primary {
		t.Errorf("non-hovered fill = %v, want Accent.Primary %v", fill.Color, tokens.Colors.Accent.Primary)
	}
}

func TestBuildSceneWithLightTheme(t *testing.T) {
	canvas := render.NewSceneCanvas(800, 600)
	scene := BuildScene(Text("HELLO"), canvas, theme.Light, 800, 600, nil, nil)

	if len(scene.Glyphs) != 1 {
		t.Fatalf("expected 1 glyph, got %d", len(scene.Glyphs))
	}
	glyph := scene.Glyphs[0]
	lightTextPrimary := theme.Light.Tokens().Colors.Text.Primary
	if glyph.Color != lightTextPrimary {
		t.Errorf("light theme text color = %v, want %v", glyph.Color, lightTextPrimary)
	}
}

// ── Tier 1 Widget Tests ─────────────────────────────────────────

func TestBuildSceneDivider(t *testing.T) {
	scene := buildTestScene(Divider(), 800, 600)
	tokens := theme.Default.Tokens()

	if len(scene.Rects) != 1 {
		t.Fatalf("Divider should produce 1 rect, got %d", len(scene.Rects))
	}
	r := scene.Rects[0]
	if r.H != 1 {
		t.Errorf("Divider height = %d, want 1", r.H)
	}
	if r.Color != tokens.Colors.Stroke.Divider {
		t.Errorf("Divider color = %v, want Stroke.Divider %v", r.Color, tokens.Colors.Stroke.Divider)
	}
	// Divider should span available width (800 - 2*framePadding = 752)
	expectedW := 800 - 2*framePadding
	if r.W != expectedW {
		t.Errorf("Divider width = %d, want %d", r.W, expectedW)
	}
}

func TestBuildSceneDividerInColumn(t *testing.T) {
	scene := buildTestScene(Column(
		Text("ABOVE"),
		Divider(),
		Text("BELOW"),
	), 800, 600)

	if len(scene.Glyphs) < 2 {
		t.Fatalf("expected 2 glyphs, got %d", len(scene.Glyphs))
	}
	above := scene.Glyphs[0]
	below := scene.Glyphs[1]
	if below.Y <= above.Y {
		t.Errorf("BELOW (Y=%d) should be below ABOVE (Y=%d)", below.Y, above.Y)
	}
	if len(scene.Rects) < 1 {
		t.Fatal("expected at least 1 rect for divider")
	}
}

func TestBuildSceneSpacer(t *testing.T) {
	scene := buildTestScene(Column(
		Text("A"),
		Spacer(40),
		Text("B"),
	), 800, 600)

	if len(scene.Glyphs) < 2 {
		t.Fatalf("expected 2 glyphs, got %d", len(scene.Glyphs))
	}
	a := scene.Glyphs[0]
	b := scene.Glyphs[1]
	// B should be pushed down by Spacer(40) + columnGap*2
	gap := b.Y - a.Y
	if gap < 40 {
		t.Errorf("Spacer(40) should push B at least 40px below A, got gap=%d", gap)
	}
}

func TestBuildSceneSpacerEmpty(t *testing.T) {
	scene := buildTestScene(Spacer(20), 800, 600)
	// Spacer should produce no draw commands
	if len(scene.Rects) != 0 {
		t.Errorf("Spacer should produce 0 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 0 {
		t.Errorf("Spacer should produce 0 glyphs, got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneIcon(t *testing.T) {
	scene := buildTestScene(Icon("★"), 800, 600)
	tokens := theme.Default.Tokens()

	if len(scene.Glyphs) != 1 {
		t.Fatalf("Icon should produce 1 glyph, got %d", len(scene.Glyphs))
	}
	g := scene.Glyphs[0]
	if g.Text != "★" {
		t.Errorf("Icon glyph text = %q, want %q", g.Text, "★")
	}
	if g.Color != tokens.Colors.Text.Primary {
		t.Errorf("Icon color = %v, want Text.Primary %v", g.Color, tokens.Colors.Text.Primary)
	}
}

func TestBuildSceneIconSize(t *testing.T) {
	scene := buildTestScene(IconSize("→", 24), 800, 600)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("IconSize should produce 1 glyph, got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneStack(t *testing.T) {
	scene := buildTestScene(Stack(
		Text("BOTTOM"),
		Text("TOP"),
	), 800, 600)

	if len(scene.Glyphs) != 2 {
		t.Fatalf("Stack should produce 2 glyphs, got %d", len(scene.Glyphs))
	}
	bottom := scene.Glyphs[0]
	top := scene.Glyphs[1]
	// Both children should share the same origin (stacked on top of each other)
	if bottom.X != top.X || bottom.Y != top.Y {
		t.Errorf("Stack children should share origin: bottom=(%d,%d), top=(%d,%d)",
			bottom.X, bottom.Y, top.X, top.Y)
	}
}

func TestBuildSceneStackEmpty(t *testing.T) {
	scene := buildTestScene(Stack(), 800, 600)
	if len(scene.Rects) != 0 || len(scene.Glyphs) != 0 {
		t.Error("Empty Stack should produce no draw commands")
	}
}

func TestBuildSceneScrollView(t *testing.T) {
	// Create content taller than the viewport
	content := Column(
		Text("LINE 1"),
		Text("LINE 2"),
		Text("LINE 3"),
		Text("LINE 4"),
	)
	scene := buildTestScene(ScrollView(content, 50), 800, 600)

	// Should render at least the visible glyphs
	if len(scene.Glyphs) == 0 {
		t.Fatal("ScrollView should render content glyphs")
	}
}

func TestBuildSceneScrollViewNoScrollbar(t *testing.T) {
	// Small content that fits within viewport — no scrollbar
	scene := buildTestScene(ScrollView(Text("SHORT"), 200), 800, 600)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("ScrollView should render 1 glyph, got %d", len(scene.Glyphs))
	}
}

func TestScrollStateClamp(t *testing.T) {
	var s ScrollState
	s.ScrollBy(-100, 500, 200) // scroll down 100
	if s.Offset != 100 {
		t.Errorf("Offset = %f, want 100", s.Offset)
	}
	s.ScrollBy(-500, 500, 200) // try to scroll past max
	if s.Offset != 300 {        // max = 500 - 200 = 300
		t.Errorf("Offset = %f, want 300 (clamped)", s.Offset)
	}
	s.ScrollBy(1000, 500, 200) // scroll back up past 0
	if s.Offset != 0 {
		t.Errorf("Offset = %f, want 0 (clamped)", s.Offset)
	}
}
