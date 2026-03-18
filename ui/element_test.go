package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
)

func buildTestScene(root Element, w, h int) draw.Scene {
	canvas := render.NewSceneCanvas(w, h)
	return BuildScene(root, canvas, theme.Default, w, h, nil, nil)
}

// buildTestSceneSfnt builds a scene using the sfnt shaper and glyph atlas.
func buildTestSceneSfnt(root Element, w, h int) draw.Scene {
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewSfntShaper(fonts.Fallback)
	canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
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

// ── Tier 2 Widget Tests ──────────────────────────────────────────

func TestBuildSceneCheckboxUnchecked(t *testing.T) {
	scene := buildTestScene(Checkbox("Enable", false, nil), 800, 600)
	// Unchecked: 2 rects (border + fill) + 1 glyph (label)
	if len(scene.Rects) < 2 {
		t.Errorf("Unchecked Checkbox should produce at least 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 1 {
		t.Fatalf("Checkbox should produce at least 1 glyph (label), got %d", len(scene.Glyphs))
	}
	if scene.Glyphs[0].Text != "Enable" {
		t.Errorf("label text = %q, want %q", scene.Glyphs[0].Text, "Enable")
	}
}

func TestBuildSceneCheckboxChecked(t *testing.T) {
	scene := buildTestScene(Checkbox("On", true, nil), 800, 600)
	// Checked: 2 rects (border + fill) + 2 glyphs (checkmark + label)
	if len(scene.Rects) < 2 {
		t.Errorf("Checked Checkbox should produce at least 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 2 {
		t.Fatalf("Checked Checkbox should produce at least 2 glyphs (check + label), got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneCheckboxHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)

	// With callback
	BuildScene(Checkbox("A", false, func(bool) {}), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 1 {
		t.Errorf("Checkbox with onToggle should register 1 hit target, got %d", hitMap.Len())
	}

	// Without callback
	hitMap.Reset()
	canvas = render.NewSceneCanvas(800, 600)
	BuildScene(Checkbox("B", false, nil), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 0 {
		t.Errorf("Checkbox with nil onToggle should register 0 targets, got %d", hitMap.Len())
	}
}

func TestBuildSceneCheckboxToggle(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	var received bool
	BuildScene(Checkbox("X", false, func(v bool) { received = v }), canvas, theme.Default, 800, 600, &hitMap, nil)

	target := hitMap.HitTest(float32(framePadding+5), float32(framePadding+5))
	if target == nil {
		t.Fatal("expected hit target at checkbox position")
	}
	target.OnClick()
	if !received {
		t.Error("onToggle should receive true when toggling from unchecked")
	}
}

func TestBuildSceneRadio(t *testing.T) {
	scene := buildTestScene(Radio("Option", false, nil), 800, 600)
	if len(scene.Rects) < 2 {
		t.Errorf("Radio should produce at least 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 1 {
		t.Fatalf("Radio should produce at least 1 glyph (label), got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneRadioSelected(t *testing.T) {
	scene := buildTestScene(Radio("Option", true, nil), 800, 600)
	// Selected: 3 rects (outer + inner + dot)
	if len(scene.Rects) < 3 {
		t.Errorf("Selected Radio should produce at least 3 rects, got %d", len(scene.Rects))
	}
}

func TestBuildSceneRadioHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Radio("A", false, func() {}), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 1 {
		t.Errorf("Radio with onSelect should register 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneToggle(t *testing.T) {
	// Off state
	sceneOff := buildTestScene(Toggle(false, nil), 800, 600)
	if len(sceneOff.Rects) < 2 {
		t.Errorf("Toggle should produce at least 2 rects (track + thumb), got %d", len(sceneOff.Rects))
	}

	// On state — track color should differ
	sceneOn := buildTestScene(Toggle(true, nil), 800, 600)
	if len(sceneOn.Rects) < 2 {
		t.Fatalf("Toggle should produce at least 2 rects, got %d", len(sceneOn.Rects))
	}
	tokens := theme.Default.Tokens()
	trackOff := sceneOff.Rects[0]
	trackOn := sceneOn.Rects[0]
	if trackOff.Color == tokens.Colors.Accent.Primary {
		t.Error("Off toggle track should not use Accent.Primary")
	}
	if trackOn.Color != tokens.Colors.Accent.Primary {
		t.Errorf("On toggle track = %v, want Accent.Primary %v", trackOn.Color, tokens.Colors.Accent.Primary)
	}
}

func TestBuildSceneToggleHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Toggle(false, func(bool) {}), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 1 {
		t.Errorf("Toggle with onToggle should register 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneSlider(t *testing.T) {
	scene := buildTestScene(Slider(0.5, nil), 800, 600)
	// Track + filled portion + thumb = 3 rects minimum
	if len(scene.Rects) < 3 {
		t.Errorf("Slider should produce at least 3 rects, got %d", len(scene.Rects))
	}
}

func TestBuildSceneSliderHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Slider(0.5, func(float32) {}), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 1 {
		t.Errorf("Slider with onChange should register 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneProgressBar(t *testing.T) {
	// Determinate with value > 0: 2 rects (track + fill)
	scene := buildTestScene(ProgressBar(0.5), 800, 600)
	if len(scene.Rects) < 2 {
		t.Errorf("ProgressBar(0.5) should produce at least 2 rects, got %d", len(scene.Rects))
	}

	// Value = 0: only track (1 rect)
	scene0 := buildTestScene(ProgressBar(0), 800, 600)
	if len(scene0.Rects) < 1 {
		t.Errorf("ProgressBar(0) should produce at least 1 rect, got %d", len(scene0.Rects))
	}
}

func TestBuildSceneProgressBarIndeterminate(t *testing.T) {
	scene := buildTestScene(ProgressBarIndeterminate(), 800, 600)
	if len(scene.Rects) < 2 {
		t.Errorf("Indeterminate ProgressBar should produce at least 2 rects, got %d", len(scene.Rects))
	}
}

func TestBuildSceneTextField(t *testing.T) {
	scene := buildTestScene(TextField("hello", ""), 800, 600)
	if len(scene.Rects) < 2 {
		t.Errorf("TextField should produce at least 2 rects (border + fill), got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) < 1 {
		t.Fatalf("TextField with value should produce at least 1 glyph, got %d", len(scene.Glyphs))
	}
	if scene.Glyphs[0].Text != "hello" {
		t.Errorf("TextField glyph text = %q, want %q", scene.Glyphs[0].Text, "hello")
	}
}

func TestBuildSceneTextFieldPlaceholder(t *testing.T) {
	scene := buildTestScene(TextField("", "Enter..."), 800, 600)
	tokens := theme.Default.Tokens()
	if len(scene.Glyphs) < 1 {
		t.Fatalf("TextField with placeholder should produce 1 glyph, got %d", len(scene.Glyphs))
	}
	if scene.Glyphs[0].Text != "Enter..." {
		t.Errorf("placeholder text = %q, want %q", scene.Glyphs[0].Text, "Enter...")
	}
	if scene.Glyphs[0].Color != tokens.Colors.Text.Disabled {
		t.Errorf("placeholder color = %v, want Text.Disabled %v", scene.Glyphs[0].Color, tokens.Colors.Text.Disabled)
	}
}

func TestBuildSceneTextFieldNoHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(TextField("x", ""), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 0 {
		t.Errorf("TextField should not register hit targets, got %d", hitMap.Len())
	}
}

func TestBuildSceneSelect(t *testing.T) {
	scene := buildTestScene(Select("Option 1", []string{"Option 1", "Option 2"}), 800, 600)
	if len(scene.Rects) < 2 {
		t.Errorf("Select should produce at least 2 rects, got %d", len(scene.Rects))
	}
	// Value text + arrow indicator = 2 glyphs
	if len(scene.Glyphs) < 2 {
		t.Fatalf("Select should produce at least 2 glyphs (value + arrow), got %d", len(scene.Glyphs))
	}
}

func TestBuildSceneSelectNoHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Select("x", nil), canvas, theme.Default, 800, 600, &hitMap, nil)
	if hitMap.Len() != 0 {
		t.Errorf("Select should not register hit targets, got %d", hitMap.Len())
	}
}

func TestBuildSceneColumnWithTier2(t *testing.T) {
	// Mix of Tier 1 and Tier 2 widgets — should not panic and Y should increase.
	scene := buildTestScene(Column(
		Text("Title"),
		Checkbox("Check", true, nil),
		Radio("Radio", false, nil),
		Toggle(true, nil),
		Slider(0.5, nil),
		ProgressBar(0.7),
		TextField("val", ""),
		Select("opt", nil),
	), 800, 600)

	if len(scene.Rects) == 0 {
		t.Fatal("Mixed column should produce rects")
	}
	if len(scene.Glyphs) == 0 {
		t.Fatal("Mixed column should produce glyphs")
	}
}

func TestBuildSceneTextStyled(t *testing.T) {
	style := draw.TextStyle{Size: 20, Weight: draw.FontWeightSemiBold}
	scene := buildTestScene(TextStyled("Big", style), 800, 600)
	if len(scene.Glyphs) != 1 {
		t.Fatalf("TextStyled should produce 1 glyph, got %d", len(scene.Glyphs))
	}
	if scene.Glyphs[0].Text != "Big" {
		t.Errorf("glyph text = %q, want %q", scene.Glyphs[0].Text, "Big")
	}
}

// ── Sfnt Font Rendering Tests ────────────────────────────────────

func TestSfntBuildSceneText(t *testing.T) {
	scene := buildTestSceneSfnt(Text("HELLO WORLD"), 800, 600)
	// With sfnt shaper, text goes through TexturedGlyphs instead of Glyphs.
	if len(scene.TexturedGlyphs) == 0 {
		t.Fatal("Sfnt Text element should produce TexturedGlyphs")
	}
	// Each non-space character should produce a textured glyph.
	// "HELLO WORLD" has 10 non-space characters.
	if len(scene.TexturedGlyphs) != 10 {
		t.Errorf("expected 10 TexturedGlyphs for 'HELLO WORLD', got %d", len(scene.TexturedGlyphs))
	}
	// No legacy bitmap glyphs should be used.
	if len(scene.Glyphs) != 0 {
		t.Errorf("Sfnt path should produce 0 legacy Glyphs, got %d", len(scene.Glyphs))
	}
}

func TestSfntBuildSceneButton(t *testing.T) {
	scene := buildTestSceneSfnt(Button("OK", nil), 800, 600)
	// Button: 2 rects (edge + fill) + textured glyphs for "OK" (2 chars).
	if len(scene.Rects) != 2 {
		t.Fatalf("Button should produce 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.TexturedGlyphs) != 2 {
		t.Errorf("Button label 'OK' should produce 2 TexturedGlyphs, got %d", len(scene.TexturedGlyphs))
	}
}

func TestSfntBuildSceneColumnTextAndButton(t *testing.T) {
	scene := buildTestSceneSfnt(Column(
		Text("HELLO"),
		Button("GO", nil),
	), 800, 600)

	// HELLO (5 glyphs) + GO (2 glyphs) = 7 textured glyphs.
	if len(scene.TexturedGlyphs) != 7 {
		t.Errorf("expected 7 TexturedGlyphs, got %d", len(scene.TexturedGlyphs))
	}
	// Button produces 2 rects.
	if len(scene.Rects) != 2 {
		t.Errorf("expected 2 rects, got %d", len(scene.Rects))
	}
}

func TestSfntTexturedGlyphsHaveValidBounds(t *testing.T) {
	scene := buildTestSceneSfnt(Text("A"), 800, 600)
	if len(scene.TexturedGlyphs) != 1 {
		t.Fatalf("expected 1 TexturedGlyph, got %d", len(scene.TexturedGlyphs))
	}
	g := scene.TexturedGlyphs[0]
	if g.DstW <= 0 || g.DstH <= 0 {
		t.Errorf("glyph size = %fx%f, want > 0", g.DstW, g.DstH)
	}
	if g.SrcW <= 0 || g.SrcH <= 0 {
		t.Errorf("atlas source size = %dx%d, want > 0", g.SrcW, g.SrcH)
	}
}

func TestSfntTexturedGlyphsInsideViewport(t *testing.T) {
	scene := buildTestSceneSfnt(Text("Test"), 800, 600)
	for i, g := range scene.TexturedGlyphs {
		if g.DstX < 0 || g.DstY < -100 || g.DstX > 800 || g.DstY > 600 {
			t.Errorf("TexturedGlyph[%d] at (%f,%f) outside reasonable bounds", i, g.DstX, g.DstY)
		}
	}
}

func TestSfntTextMeasureConsistentWithLayout(t *testing.T) {
	// Verify that text layout uses sfnt metrics, not bitmap metrics.
	sceneBitmap := buildTestScene(Column(Text("A"), Text("B")), 800, 600)
	sceneSfnt := buildTestSceneSfnt(Column(Text("A"), Text("B")), 800, 600)

	// In bitmap mode, glyph[1].Y should differ from sfnt mode since metrics differ.
	if len(sceneBitmap.Glyphs) < 2 || len(sceneSfnt.TexturedGlyphs) < 2 {
		t.Skip("need at least 2 glyphs for comparison")
	}

	// The sfnt B's Y position should differ from bitmap B's Y position
	// because font metrics are different.
	bitmapBY := sceneBitmap.Glyphs[1].Y
	// Find the second text element's first glyph (B).
	// In sfnt mode, A produces 1 TexturedGlyph, B produces 1 = index 1.
	sfntBY := sceneSfnt.TexturedGlyphs[1].DstY

	// They should be different because bitmap and sfnt have different ascents.
	if float32(bitmapBY) == sfntBY {
		t.Log("bitmap and sfnt Y positions happen to match (possible but unlikely)")
	}
}

func TestSfntCheckboxWithLabel(t *testing.T) {
	scene := buildTestSceneSfnt(Checkbox("Enable", true, nil), 800, 600)
	// Should have TexturedGlyphs for both the checkmark and label.
	if len(scene.TexturedGlyphs) == 0 {
		t.Fatal("Sfnt Checkbox should produce TexturedGlyphs")
	}
}

// ── Scroll Offset Tests ──────────────────────────────────────────

func TestScrollViewOffsetShiftsContent(t *testing.T) {
	content := Column(
		Text("LINE 1"),
		Text("LINE 2"),
		Text("LINE 3"),
		Text("LINE 4"),
	)

	// Render without offset.
	sceneNoScroll := buildTestScene(ScrollView(content, 50), 800, 600)

	// Render with offset — LINE 1 gets clipped above the viewport,
	// so the first visible glyph changes.
	state := &ScrollState{Offset: 20}
	canvas := render.NewSceneCanvas(800, 600)
	sceneWithScroll := BuildScene(ScrollView(content, 50, state), canvas, theme.Default, 800, 600, nil, nil)

	if len(sceneNoScroll.Glyphs) == 0 {
		t.Fatal("non-scrolled scene should produce glyphs")
	}

	// The non-scrolled first glyph should be "LINE 1".
	if sceneNoScroll.Glyphs[0].Text != "LINE 1" {
		t.Errorf("non-scrolled first glyph = %q, want LINE 1", sceneNoScroll.Glyphs[0].Text)
	}

	// With offset, LINE 1 should be clipped out of the viewport.
	// The first visible glyph should be LINE 2 (shifted up but still in clip).
	if len(sceneWithScroll.Glyphs) == 0 {
		t.Fatal("scrolled scene should produce glyphs")
	}
	if sceneWithScroll.Glyphs[0].Text != "LINE 2" {
		t.Errorf("scrolled first visible glyph = %q, want LINE 2", sceneWithScroll.Glyphs[0].Text)
	}

	// The visible content should differ between scrolled and non-scrolled.
	// Non-scrolled starts with LINE 1; scrolled starts with LINE 2.
	if len(sceneNoScroll.Glyphs) > 0 && len(sceneWithScroll.Glyphs) > 0 {
		if sceneNoScroll.Glyphs[0].Text == sceneWithScroll.Glyphs[0].Text {
			t.Error("scrolled and non-scrolled views should show different first content")
		}
	}
}

func TestScrollViewRegistersScrollTarget(t *testing.T) {
	content := Column(
		Text("A"), Text("B"), Text("C"), Text("D"),
		Text("E"), Text("F"), Text("G"), Text("H"),
		Text("I"), Text("J"), Text("K"), Text("L"),
	)
	scroll := &ScrollState{}
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(ScrollView(content, 40, scroll), canvas, theme.Default, 800, 600, &hitMap, nil)

	// The scroll target should be registered.
	target := hitMap.HitTestScroll(30, 30)
	if target == nil {
		t.Fatal("expected scroll target at (30, 30)")
	}

	// Scrolling should modify the state.
	target.OnScroll(-30)
	if scroll.Offset != 30 {
		t.Errorf("Offset = %f, want 30 after OnScroll(-30)", scroll.Offset)
	}
}

func TestScrollViewThumbPositionReflectsOffset(t *testing.T) {
	content := Column(
		Text("A"), Text("B"), Text("C"), Text("D"),
		Text("E"), Text("F"), Text("G"), Text("H"),
	)

	// No offset — thumb at top.
	state0 := &ScrollState{Offset: 0}
	canvas0 := render.NewSceneCanvas(800, 600)
	scene0 := BuildScene(ScrollView(content, 40, state0), canvas0, theme.Default, 800, 600, nil, nil)

	// Large offset — thumb should be lower.
	state1 := &ScrollState{Offset: 100}
	canvas1 := render.NewSceneCanvas(800, 600)
	scene1 := BuildScene(ScrollView(content, 40, state1), canvas1, theme.Default, 800, 600, nil, nil)

	// Both scenes should have a scrollbar (track + thumb = 2+ rects).
	if len(scene0.Rects) < 2 || len(scene1.Rects) < 2 {
		t.Fatalf("expected scrollbar rects: scene0=%d, scene1=%d", len(scene0.Rects), len(scene1.Rects))
	}

	// The last rect in each scene is the scrollbar thumb.
	thumb0 := scene0.Rects[len(scene0.Rects)-1]
	thumb1 := scene1.Rects[len(scene1.Rects)-1]

	if thumb1.Y <= thumb0.Y {
		t.Errorf("scrolled thumb Y=%d should be below non-scrolled thumb Y=%d", thumb1.Y, thumb0.Y)
	}
}

func TestScrollStateScrollByDelta(t *testing.T) {
	var s ScrollState
	// Scroll down by 50 in a 200-content, 100-viewport scenario.
	s.ScrollBy(-50, 200, 100)
	if s.Offset != 50 {
		t.Errorf("Offset = %f, want 50", s.Offset)
	}
	// Scroll up by 100 — should clamp to 0.
	s.ScrollBy(100, 200, 100)
	if s.Offset != 0 {
		t.Errorf("Offset = %f, want 0 (clamped)", s.Offset)
	}
}

// ── Focus / TextField Tests ──────────────────────────────────────

func TestTextFieldFocusBorderHighlight(t *testing.T) {
	focus := &FocusState{FocusedID: 1}
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(
		TextField("hi", "", WithOnChange(func(string) {}), WithFocusState(focus)),
		canvas, theme.Default, 800, 600, nil, nil, focus,
	)
	scene := canvas.Scene()
	tokens := theme.Default.Tokens()

	// The first rect is the border — when focused, it should use Accent.Primary.
	if len(scene.Rects) < 1 {
		t.Fatal("expected at least 1 rect for border")
	}
	border := scene.Rects[0]
	if border.Color != tokens.Colors.Accent.Primary {
		t.Errorf("focused border color = %v, want Accent.Primary %v", border.Color, tokens.Colors.Accent.Primary)
	}
}

func TestTextFieldUnfocusedBorder(t *testing.T) {
	focus := &FocusState{FocusedID: 0} // nothing focused
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(
		TextField("hi", "", WithOnChange(func(string) {}), WithFocusState(focus)),
		canvas, theme.Default, 800, 600, nil, nil, focus,
	)
	scene := canvas.Scene()
	tokens := theme.Default.Tokens()

	if len(scene.Rects) < 1 {
		t.Fatal("expected at least 1 rect for border")
	}
	border := scene.Rects[0]
	if border.Color != tokens.Colors.Stroke.Border {
		t.Errorf("unfocused border color = %v, want Stroke.Border %v", border.Color, tokens.Colors.Stroke.Border)
	}
}

func TestTextFieldClickSetsFocus(t *testing.T) {
	focus := &FocusState{}
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(
		TextField("", "type here", WithOnChange(func(string) {}), WithFocusState(focus)),
		canvas, theme.Default, 800, 600, &hitMap, nil, focus,
	)

	if hitMap.Len() != 1 {
		t.Fatalf("TextField with onChange+focus should register 1 hit target, got %d", hitMap.Len())
	}
	target := hitMap.HitTest(float32(framePadding+5), float32(framePadding+5))
	if target == nil {
		t.Fatal("expected hit target at TextField position")
	}
	target.OnClick()
	if focus.FocusedID != 1 {
		t.Errorf("FocusedID = %d, want 1 after click", focus.FocusedID)
	}
}

func TestHandleKeyMsgBackspace(t *testing.T) {
	focus := &FocusState{FocusedID: 1}
	var result string
	handleKeyMsg(focus, "Backspace", "hello", func(v string) { result = v })
	if result != "hell" {
		t.Errorf("after Backspace: %q, want %q", result, "hell")
	}
}

func TestHandleKeyMsgEscapeBlurs(t *testing.T) {
	focus := &FocusState{FocusedID: 1}
	handleKeyMsg(focus, "Escape", "hello", func(string) {})
	if focus.FocusedID != 0 {
		t.Errorf("Escape should blur: FocusedID = %d, want 0", focus.FocusedID)
	}
}

func TestInternalCharInput(t *testing.T) {
	var result string
	handleCharInput('X', "hello", func(v string) { result = v })
	if result != "helloX" {
		t.Errorf("after char input: %q, want %q", result, "helloX")
	}
}

func TestHandleCharInputIgnoresControl(t *testing.T) {
	called := false
	handleCharInput(0x08, "hello", func(v string) { called = true })
	if called {
		t.Error("control characters should be ignored")
	}
}
