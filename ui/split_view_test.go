package ui

import (
	"testing"

	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
)

func TestSplitViewDefaults(t *testing.T) {
	el := SplitView(Text("A"), Text("B"), 0.5, nil)
	sv, ok := el.(splitViewElement)
	if !ok {
		t.Fatal("SplitView should return splitViewElement")
	}
	if sv.Axis != AxisRow {
		t.Errorf("default axis = %v, want AxisRow", sv.Axis)
	}
	if sv.Ratio != 0.5 {
		t.Errorf("ratio = %v, want 0.5", sv.Ratio)
	}
	if sv.DividerSize != 0 {
		t.Errorf("divider size = %v, want 0 (use default)", sv.DividerSize)
	}
}

func TestSplitViewOptions(t *testing.T) {
	el := SplitView(Text("A"), Text("B"), 0.3, nil,
		WithSplitAxis(AxisColumn),
		WithDividerSize(10),
	)
	sv := el.(splitViewElement)
	if sv.Axis != AxisColumn {
		t.Errorf("axis = %v, want AxisColumn", sv.Axis)
	}
	if sv.DividerSize != 10 {
		t.Errorf("divider size = %v, want 10", sv.DividerSize)
	}
}

func TestSplitViewRendersChildren(t *testing.T) {
	scene := buildTestScene(
		SplitView(Text("LEFT"), Text("RIGHT"), 0.5, nil),
		800, 600,
	)
	// Both text children should produce glyphs.
	if len(scene.Glyphs) < 2 {
		t.Fatalf("expected at least 2 glyph entries, got %d", len(scene.Glyphs))
	}
}

func TestSplitViewDividerRect(t *testing.T) {
	scene := buildTestScene(
		SplitView(Empty(), Empty(), 0.5, nil),
		800, 600,
	)
	// The divider line should produce exactly 1 rect.
	if len(scene.Rects) != 1 {
		t.Fatalf("expected 1 rect (divider line), got %d", len(scene.Rects))
	}
}

func TestSplitViewVerticalAxis(t *testing.T) {
	scene := buildTestScene(
		SplitView(Text("TOP"), Text("BOTTOM"), 0.5, nil, WithSplitAxis(AxisColumn)),
		800, 600,
	)
	if len(scene.Glyphs) < 2 {
		t.Fatalf("expected at least 2 glyph entries, got %d", len(scene.Glyphs))
	}
	// TOP should be above BOTTOM.
	top := scene.Glyphs[0]
	bottom := scene.Glyphs[1]
	if top.Y >= bottom.Y {
		t.Errorf("TOP glyph Y (%d) should be above BOTTOM glyph Y (%d)", top.Y, bottom.Y)
	}
}

func TestSplitViewHorizontalChildPositions(t *testing.T) {
	scene := buildTestScene(
		SplitView(Text("LEFT"), Text("RIGHT"), 0.5, nil),
		800, 600,
	)
	if len(scene.Glyphs) < 2 {
		t.Fatalf("expected at least 2 glyph entries, got %d", len(scene.Glyphs))
	}
	left := scene.Glyphs[0]
	right := scene.Glyphs[1]
	if left.X >= right.X {
		t.Errorf("LEFT glyph X (%d) should be before RIGHT glyph X (%d)", left.X, right.X)
	}
}

func TestSplitViewDragTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	var resized float32
	root := SplitView(Empty(), Empty(), 0.5, func(r float32) { resized = r })
	BuildScene(root, canvas, theme.Default, 800, 600, &hitMap, nil)

	if hitMap.Len() == 0 {
		t.Fatal("expected at least 1 hit target for the divider drag area")
	}

	// Hit-test the center of the window (where the divider should be).
	target := hitMap.HitTest(400, 300)
	if target == nil {
		t.Fatal("expected hit target at window center")
	}
	if !target.Draggable {
		t.Error("divider target should be draggable")
	}
	if target.OnClickAt == nil {
		t.Fatal("divider target should have OnClickAt callback")
	}

	// Simulate a drag and verify callback fires.
	target.OnClickAt(300, 300)
	if resized == 0 {
		t.Error("OnResize callback should have been called")
	}
}

func TestSplitViewNilOnResizeNoDragTarget(t *testing.T) {
	var hitMap hit.Map
	canvas := render.NewSceneCanvas(800, 600)
	root := SplitView(Empty(), Empty(), 0.5, nil)
	BuildScene(root, canvas, theme.Default, 800, 600, &hitMap, nil)

	if hitMap.Len() != 0 {
		t.Errorf("expected 0 hit targets when OnResize is nil, got %d", hitMap.Len())
	}
}

func TestSplitViewRatioClamp(t *testing.T) {
	// Ratio < 0 should not panic.
	scene := buildTestScene(SplitView(Text("A"), Text("B"), -0.5, nil), 800, 600)
	if len(scene.Glyphs) < 2 {
		t.Fatal("expected 2 glyph entries for clamped negative ratio")
	}

	// Ratio > 1 should not panic.
	scene = buildTestScene(SplitView(Text("A"), Text("B"), 1.5, nil), 800, 600)
	if len(scene.Glyphs) < 2 {
		t.Fatal("expected 2 glyph entries for clamped ratio > 1")
	}
}
