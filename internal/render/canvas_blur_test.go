//go:build nogui

package render

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestPushBlurPopBlur(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	c.PushBlur(16)
	c.PopBlur()

	scene := c.Scene()
	if len(scene.BlurRegions) != 1 {
		t.Fatalf("expected 1 blur region, got %d", len(scene.BlurRegions))
	}
	br := scene.BlurRegions[0]
	if br.Radius != 16 {
		t.Errorf("expected radius 16, got %f", br.Radius)
	}
	if br.W != 800 || br.H != 600 {
		t.Errorf("expected full viewport blur region (800x600), got %dx%d", br.W, br.H)
	}
}

func TestNestedBlur(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	c.PushBlur(8)
	c.PushBlur(32)
	c.PopBlur()
	c.PopBlur()

	scene := c.Scene()
	if len(scene.BlurRegions) != 2 {
		t.Fatalf("expected 2 blur regions, got %d", len(scene.BlurRegions))
	}
	if scene.BlurRegions[0].Radius != 8 {
		t.Errorf("first region radius: expected 8, got %f", scene.BlurRegions[0].Radius)
	}
	if scene.BlurRegions[1].Radius != 32 {
		t.Errorf("second region radius: expected 32, got %f", scene.BlurRegions[1].Radius)
	}
}

func TestBlurWithClip(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	c.PushClip(draw.R(100, 100, 200, 200))
	c.PushBlur(16)
	c.PopBlur()
	c.PopClip()

	scene := c.Scene()
	if len(scene.BlurRegions) != 1 {
		t.Fatalf("expected 1 blur region, got %d", len(scene.BlurRegions))
	}
	br := scene.BlurRegions[0]
	if br.X != 100 || br.Y != 100 || br.W != 200 || br.H != 200 {
		t.Errorf("expected clipped blur region (100,100,200,200), got (%d,%d,%d,%d)", br.X, br.Y, br.W, br.H)
	}
}

func TestBlurRadiusClamping(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	c.PushBlur(100) // should be clamped to 64
	c.PopBlur()

	scene := c.Scene()
	if len(scene.BlurRegions) != 1 {
		t.Fatalf("expected 1 blur region, got %d", len(scene.BlurRegions))
	}
	if scene.BlurRegions[0].Radius != 64 {
		t.Errorf("expected clamped radius 64, got %f", scene.BlurRegions[0].Radius)
	}
}

func TestBlurZeroRadius(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	c.PushBlur(0)
	c.PopBlur()

	scene := c.Scene()
	if len(scene.BlurRegions) != 0 {
		t.Errorf("expected 0 blur regions for zero radius, got %d", len(scene.BlurRegions))
	}
}

func TestEmptyBlurStack(t *testing.T) {
	c := NewSceneCanvas(800, 600)
	// PopBlur on empty stack should not panic.
	c.PopBlur()

	scene := c.Scene()
	if len(scene.BlurRegions) != 0 {
		t.Errorf("expected 0 blur regions, got %d", len(scene.BlurRegions))
	}
}
