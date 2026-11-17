//go:build nogui

package app

import (
	"image"
	"testing"
)

func TestRunHeadlessSoftwareRenderer(t *testing.T) {
	var frames int
	var last *image.RGBA
	err := Run(testModel{}, testUpdate, m2HelloView,
		WithSize(320, 240),
		WithHeadlessFrames(3),
		WithSoftwareRenderer(func(img *image.RGBA) {
			frames++
			last = image.NewRGBA(img.Rect)
			copy(last.Pix, img.Pix)
		}),
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if frames == 0 || last == nil {
		t.Fatal("no frame presented")
	}
	if b := last.Bounds(); b.Dx() != 320 || b.Dy() != 240 {
		t.Fatalf("frame size = %v, want 320x240", b)
	}
	// The view draws text and a button: the frame must not be a flat fill.
	bg := last.RGBAAt(319, 239)
	differ := 0
	for y := 0; y < 240; y++ {
		for x := 0; x < 320; x++ {
			if last.RGBAAt(x, y) != bg {
				differ++
			}
		}
	}
	if differ < 200 {
		t.Errorf("only %d pixels differ from the background; text/button not rendered", differ)
	}
}
