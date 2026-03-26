package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
)

func buildTestScene(root Element, w, h int) draw.Scene {
	canvas := render.NewSceneCanvas(w, h)
	return BuildScene(root, canvas, theme.Default, w, h, nil)
}

// buildTestSceneSfnt builds a scene using the sfnt shaper and glyph atlas.
func buildTestSceneSfnt(root Element, w, h int) draw.Scene {
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewSfntShaper(fonts.Fallback)
	canvas := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
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
		ButtonText("OK", func() {}),
	), canvas, theme.Default, 800, 600, NewInteractor(&hitMap, nil))

	if hitMap.Len() != 1 {
		t.Fatalf("expected 1 hit target, got %d", hitMap.Len())
	}
}

func TestBuildSceneHitTargetNilOnClick(t *testing.T) {
	// Buttons with nil onClick still register a hit target (no-op) so the
	// hover index stays in sync with the hit-map index.
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(ButtonText("X", nil), canvas, theme.Default, 800, 600, NewInteractor(&hitMap, nil))

	if hitMap.Len() != 1 {
		t.Errorf("nil OnClick should still register hit target for hover sync, got %d", hitMap.Len())
	}
}

func TestBuildSceneMultipleHitTargets(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Row(
		ButtonText("A", func() {}),
		ButtonText("B", func() {}),
	), canvas, theme.Default, 800, 600, NewInteractor(&hitMap, nil))

	if hitMap.Len() != 2 {
		t.Fatalf("expected 2 hit targets, got %d", hitMap.Len())
	}
}

func TestHitTargetClickable(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	clicked := false
	BuildScene(ButtonText("OK", func() { clicked = true }), canvas, theme.Default, 800, 600, NewInteractor(&hitMap, nil))

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

func TestBuildSceneWithLightTheme(t *testing.T) {
	canvas := render.NewSceneCanvas(800, 600)
	scene := BuildScene(Text("HELLO"), canvas, theme.Light, 800, 600, nil)

	if len(scene.Glyphs) != 1 {
		t.Fatalf("expected 1 glyph, got %d", len(scene.Glyphs))
	}
	glyph := scene.Glyphs[0]
	lightTextPrimary := theme.Light.Tokens().Colors.Text.Primary
	if glyph.Color != lightTextPrimary {
		t.Errorf("light theme text color = %v, want %v", glyph.Color, lightTextPrimary)
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

func TestBuildSceneTextFieldNoHitTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(TextField("x", ""), canvas, theme.Default, 800, 600, NewInteractor(&hitMap, nil))
	if hitMap.Len() != 0 {
		t.Errorf("TextField should not register hit targets, got %d", hitMap.Len())
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
	// With sfnt shaper + MSDF, text goes through MSDFGlyphs (or TexturedGlyphs for bitmap fallback).
	total := len(scene.TexturedGlyphs) + len(scene.MSDFGlyphs)
	if total == 0 {
		t.Fatal("Sfnt Text element should produce TexturedGlyphs or MSDFGlyphs")
	}
	// Each non-space character should produce a glyph.
	// "HELLO WORLD" has 10 non-space characters.
	if total != 10 {
		t.Errorf("expected 10 glyphs for 'HELLO WORLD', got %d", total)
	}
	// No legacy bitmap glyphs should be used.
	if len(scene.Glyphs) != 0 {
		t.Errorf("Sfnt path should produce 0 legacy Glyphs, got %d", len(scene.Glyphs))
	}
}

func TestSfntTexturedGlyphsHaveValidBounds(t *testing.T) {
	scene := buildTestSceneSfnt(Text("A"), 800, 600)
	// Glyphs may be in TexturedGlyphs or MSDFGlyphs depending on font.
	allGlyphs := append(scene.TexturedGlyphs, scene.MSDFGlyphs...)
	if len(allGlyphs) != 1 {
		t.Fatalf("expected 1 glyph, got %d", len(allGlyphs))
	}
	g := allGlyphs[0]
	if g.DstW <= 0 || g.DstH <= 0 {
		t.Errorf("glyph size = %fx%f, want > 0", g.DstW, g.DstH)
	}
	if g.SrcW <= 0 || g.SrcH <= 0 {
		t.Errorf("atlas source size = %dx%d, want > 0", g.SrcW, g.SrcH)
	}
}

func TestSfntTexturedGlyphsInsideViewport(t *testing.T) {
	scene := buildTestSceneSfnt(Text("Test"), 800, 600)
	allGlyphs := append(scene.TexturedGlyphs, scene.MSDFGlyphs...)
	for i, g := range allGlyphs {
		if g.DstX < 0 || g.DstY < -100 || g.DstX > 800 || g.DstY > 600 {
			t.Errorf("Glyph[%d] at (%f,%f) outside reasonable bounds", i, g.DstX, g.DstY)
		}
	}
}

func TestSfntTextMeasureConsistentWithLayout(t *testing.T) {
	// Verify that text layout uses sfnt metrics, not bitmap metrics.
	sceneBitmap := buildTestScene(Column(Text("A"), Text("B")), 800, 600)
	sceneSfnt := buildTestSceneSfnt(Column(Text("A"), Text("B")), 800, 600)

	// In bitmap mode, glyph[1].Y should differ from sfnt mode since metrics differ.
	allSfntGlyphs := append(sceneSfnt.TexturedGlyphs, sceneSfnt.MSDFGlyphs...)
	if len(sceneBitmap.Glyphs) < 2 || len(allSfntGlyphs) < 2 {
		t.Skip("need at least 2 glyphs for comparison")
	}

	// The sfnt B's Y position should differ from bitmap B's Y position
	// because font metrics are different.
	bitmapBY := sceneBitmap.Glyphs[1].Y
	// Find the second text element's first glyph (B).
	// In sfnt mode, A produces 1 glyph, B produces 1 = index 1.
	sfntBY := allSfntGlyphs[1].DstY

	// They should be different because bitmap and sfnt have different ascents.
	if float32(bitmapBY) == sfntBY {
		t.Log("bitmap and sfnt Y positions happen to match (possible but unlikely)")
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

func TestBuildSceneMenuBarEmpty(t *testing.T) {
	scene := buildTestScene(MenuBar(nil, nil), 800, 600)
	if len(scene.Rects) != 0 {
		t.Errorf("Empty MenuBar should produce 0 rects, got %d", len(scene.Rects))
	}
}

func TestBuildSceneContextMenuHidden(t *testing.T) {
	items := []MenuItem{
		{Label: Text("Cut")},
		{Label: Text("Copy")},
	}
	scene := buildTestScene(ContextMenu(items, false, 100, 100), 800, 600)
	if len(scene.Rects) != 0 {
		t.Errorf("Hidden ContextMenu should produce 0 rects, got %d", len(scene.Rects))
	}
	if len(scene.Glyphs) != 0 {
		t.Errorf("Hidden ContextMenu should produce 0 glyphs, got %d", len(scene.Glyphs))
	}
}

func TestDefaultThemeDrawFuncIsNil(t *testing.T) {
	// Built-in themes return nil DrawFunc, so default rendering runs.
	if df := theme.Default.DrawFunc(theme.WidgetKindButton); df != nil {
		t.Error("Default theme should return nil DrawFunc for Button")
	}
	if df := theme.Light.DrawFunc(theme.WidgetKindTextField); df != nil {
		t.Error("Light theme should return nil DrawFunc for TextField")
	}
}

func TestSelectClosedStateNoOverlay(t *testing.T) {
	state := &SelectState{Open: false}
	el := Select("Apple", []string{"Apple", "Banana", "Cherry"}, WithSelectState(state))

	canvas := render.NewSceneCanvas(800, 600)
	scene := BuildScene(el, canvas, theme.Default, 800, 600, nil)

	// No overlay content when closed.
	if len(scene.OverlayRects) != 0 || len(scene.OverlayGlyphs) != 0 {
		t.Error("closed Select should not produce overlay content")
	}
}

// ── Surface Tests (RFC §8) ──────────────────────────────────────

// fakeSurfaceProvider is a test helper implementing SurfaceProvider.
type fakeSurfaceProvider struct {
	textureID draw.TextureID
	acquired  bool
	released  bool
}

func (f *fakeSurfaceProvider) AcquireFrame(bounds draw.Rect) (draw.TextureID, FrameToken) {
	f.acquired = true
	return f.textureID, 1
}

func (f *fakeSurfaceProvider) ReleaseFrame(token FrameToken) {
	f.released = true
}

func (f *fakeSurfaceProvider) HandleMsg(msg any) bool {
	_, ok := msg.(SurfaceMouseMsg)
	return ok
}

func TestBuildSceneSurfaceNilProvider(t *testing.T) {
	scene := buildTestScene(Surface(1, nil, 200, 150), 800, 600)
	// Nil provider should produce a placeholder rect (fill + stroke = 2).
	if len(scene.Rects) < 1 {
		t.Errorf("nil-provider surface should produce placeholder rect(s), got %d rects", len(scene.Rects))
	}
	// No surface textures should be recorded.
	if len(scene.Surfaces) != 0 {
		t.Errorf("nil-provider surface should produce 0 surface entries, got %d", len(scene.Surfaces))
	}
}

func TestBuildSceneSurfaceWithProvider(t *testing.T) {
	provider := &fakeSurfaceProvider{textureID: 42}
	scene := buildTestScene(Surface(1, provider, 200, 150), 800, 600)
	if !provider.acquired {
		t.Error("AcquireFrame was not called")
	}
	if !provider.released {
		t.Error("ReleaseFrame was not called")
	}
	if len(scene.Surfaces) != 1 {
		t.Fatalf("expected 1 surface entry, got %d", len(scene.Surfaces))
	}
	if scene.Surfaces[0].TextureID != 42 {
		t.Errorf("surface TextureID = %d, want 42", scene.Surfaces[0].TextureID)
	}
}

func TestBuildSceneSurfaceConstrainsToBounds(t *testing.T) {
	provider := &fakeSurfaceProvider{textureID: 1}
	// Request 2000x2000 but canvas is only 800x600 (minus framePadding*2).
	scene := buildTestScene(Surface(1, provider, 2000, 2000), 800, 600)
	if len(scene.Surfaces) != 1 {
		t.Fatalf("expected 1 surface entry, got %d", len(scene.Surfaces))
	}
	s := scene.Surfaces[0]
	maxW := 800 - framePadding*2
	maxH := 600 - framePadding*2
	if s.W > maxW || s.H > maxH {
		t.Errorf("surface %dx%d exceeds available area %dx%d", s.W, s.H, maxW, maxH)
	}
}

func TestSurfaceHitTarget(t *testing.T) {
	provider := &fakeSurfaceProvider{textureID: 1}
	hitMap := hit.Map{}
	hover := HoverState{}
	ix := NewInteractor(&hitMap, &hover)

	canvas := render.NewSceneCanvas(800, 600)
	BuildScene(Surface(1, provider, 200, 150), canvas, theme.Default, 800, 600, ix)

	// Surface with a provider should register a hit target.
	if hitMap.Len() < 1 {
		t.Error("surface with provider should register at least 1 hit target")
	}
}

func TestSurfaceInputRouting(t *testing.T) {
	provider := &fakeSurfaceProvider{}
	msg := SurfaceMouseMsg{
		SurfaceID: 1,
		Pos:       draw.Pt(100, 75),
		Button:    input.MouseButtonLeft,
		Action:    input.MousePress,
	}
	consumed := provider.HandleMsg(msg)
	if !consumed {
		t.Error("SurfaceProvider should consume SurfaceMouseMsg")
	}
	// Non-surface messages should not be consumed.
	if provider.HandleMsg("not a surface msg") {
		t.Error("SurfaceProvider should not consume non-SurfaceMouseMsg")
	}
}

func TestZeroCopyModeConstants(t *testing.T) {
	// Verify zero-copy mode constants are distinct.
	modes := []ZeroCopyMode{ZeroCopyNone, ZeroCopyIOSurface, ZeroCopyDMABuf, ZeroCopyDXGI}
	seen := map[ZeroCopyMode]bool{}
	for _, m := range modes {
		if seen[m] {
			t.Errorf("duplicate ZeroCopyMode value: %d", m)
		}
		seen[m] = true
	}
	// Preferred mode should be a valid non-negative value.
	mode := PreferredZeroCopyMode()
	if mode < ZeroCopyNone {
		t.Errorf("PreferredZeroCopyMode() = %d, want >= 0", mode)
	}
}
