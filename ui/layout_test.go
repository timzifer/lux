//go:build nogui

package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
)

// diagonalLayout places children diagonally with a fixed step.
type diagonalLayout struct {
	Step float32
}

func (d diagonalLayout) LayoutChildren(ctx LayoutCtx, children []Element) Size {
	x, y := float32(0), float32(0)
	maxW, maxH := float32(0), float32(0)

	for _, child := range children {
		size := ctx.Measure(child, LooseConstraints(ctx.Constraints.MaxWidth, ctx.Constraints.MaxHeight))
		ctx.Place(child, draw.Pt(x, y))
		endX := x + size.Width
		endY := y + size.Height
		if endX > maxW {
			maxW = endX
		}
		if endY > maxH {
			maxH = endY
		}
		x += d.Step
		y += d.Step
	}

	return Size{Width: maxW, Height: maxH}
}

func buildLayoutTestScene(root Element, w, h int) draw.Scene {
	canvas := render.NewSceneCanvas(w, h)
	return BuildScene(root, canvas, theme.Default, w, h, nil)
}

func TestCustomLayoutDiagonal(t *testing.T) {
	layout := diagonalLayout{Step: 30}
	el := CustomLayout(layout,
		Text("A"),
		Text("B"),
		Text("C"),
	)

	scene := buildLayoutTestScene(el, 800, 600)
	// Should produce glyph entries for the 3 text elements.
	if len(scene.Glyphs) < 3 {
		t.Errorf("custom layout should render 3 text glyphs, got %d", len(scene.Glyphs))
	}
}

func TestCustomLayoutEmpty(t *testing.T) {
	layout := diagonalLayout{Step: 10}
	el := CustomLayout(layout)

	scene := buildLayoutTestScene(el, 800, 600)
	if len(scene.Glyphs) != 0 {
		t.Errorf("empty custom layout should produce no glyphs, got %d", len(scene.Glyphs))
	}
}

func TestCustomLayoutNilLayout(t *testing.T) {
	// Should not panic with nil layout.
	el := CustomLayout(nil, Text("A"))
	scene := buildLayoutTestScene(el, 800, 600)
	// Nil layout produces no output.
	if len(scene.Glyphs) != 0 {
		t.Errorf("nil layout should produce no glyphs, got %d", len(scene.Glyphs))
	}
}

// ── Layout Cache Tests ───────────────────────────────────────────

func TestLayoutCacheStoreAndRetrieve(t *testing.T) {
	var cache LayoutCache

	c := Constraints{MaxWidth: 800, MaxHeight: 600}
	s := Size{Width: 100, Height: 50}
	rects := []draw.Rect{draw.R(0, 0, 50, 25), draw.R(50, 0, 50, 25)}

	cache.Store(c, s, rects)

	if !cache.IsValid(c) {
		t.Error("cache should be valid for same constraints")
	}
	if cache.CachedSize() != s {
		t.Errorf("cached size = %v, want %v", cache.CachedSize(), s)
	}
	if len(cache.CachedChildRects()) != 2 {
		t.Errorf("cached rects count = %d, want 2", len(cache.CachedChildRects()))
	}
}

func TestLayoutCacheInvalidateOnConstraintChange(t *testing.T) {
	var cache LayoutCache

	c1 := Constraints{MaxWidth: 800, MaxHeight: 600}
	c2 := Constraints{MaxWidth: 400, MaxHeight: 300}
	cache.Store(c1, Size{Width: 100, Height: 50}, nil)

	if cache.IsValid(c2) {
		t.Error("cache should not be valid for different constraints")
	}
}

func TestLayoutCacheInvalidate(t *testing.T) {
	var cache LayoutCache

	c := Constraints{MaxWidth: 800, MaxHeight: 600}
	cache.Store(c, Size{Width: 100, Height: 50}, nil)
	cache.Invalidate()

	if cache.IsValid(c) {
		t.Error("cache should not be valid after Invalidate()")
	}
}

func TestLayoutCacheZeroValue(t *testing.T) {
	var cache LayoutCache

	if cache.IsValid(Constraints{}) {
		t.Error("zero-value cache should not be valid")
	}
}
