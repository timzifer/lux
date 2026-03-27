package draw

import "testing"

func TestBlurRegionFields(t *testing.T) {
	br := BlurRegion{X: 10, Y: 20, W: 100, H: 50, Radius: 16}
	if br.X != 10 || br.Y != 20 || br.W != 100 || br.H != 50 {
		t.Errorf("unexpected blur region bounds: %+v", br)
	}
	if br.Radius != 16 {
		t.Errorf("expected radius 16, got %f", br.Radius)
	}
}

func TestSceneBlurRegions(t *testing.T) {
	scene := Scene{
		BlurRegions: []BlurRegion{
			{X: 0, Y: 0, W: 800, H: 600, Radius: 8},
			{X: 100, Y: 100, W: 200, H: 200, Radius: 64},
		},
	}
	if len(scene.BlurRegions) != 2 {
		t.Fatalf("expected 2, got %d", len(scene.BlurRegions))
	}
	if scene.BlurRegions[1].Radius != 64 {
		t.Errorf("expected 64, got %f", scene.BlurRegions[1].Radius)
	}
}
