package text

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
)

func newTestShaper() *SfntShaper {
	return NewSfntShaper(fonts.Fallback)
}

var bodyStyle = draw.TextStyle{
	Size:   13,
	Weight: draw.FontWeightRegular,
}

func TestSfntShaperMeasure(t *testing.T) {
	s := newTestShaper()
	m := s.Measure("Hello", bodyStyle)
	if m.Width <= 0 {
		t.Errorf("Width = %f, want > 0", m.Width)
	}
	if m.Ascent <= 0 {
		t.Errorf("Ascent = %f, want > 0", m.Ascent)
	}
}

func TestSfntShaperMeasureEmpty(t *testing.T) {
	s := newTestShaper()
	m := s.Measure("", bodyStyle)
	if m.Width != 0 {
		t.Errorf("empty string Width = %f, want 0", m.Width)
	}
}

func TestSfntShaperShape(t *testing.T) {
	s := newTestShaper()
	glyphs := s.Shape("ABC", bodyStyle)
	if len(glyphs) != 3 {
		t.Fatalf("Shape(\"ABC\") returned %d glyphs, want 3", len(glyphs))
	}
	for i, g := range glyphs {
		if g.Advance <= 0 {
			t.Errorf("glyph[%d] (%c) Advance = %f, want > 0", i, g.Rune, g.Advance)
		}
	}
}

func TestSfntShaperShapeEmpty(t *testing.T) {
	s := newTestShaper()
	glyphs := s.Shape("", bodyStyle)
	if len(glyphs) != 0 {
		t.Errorf("Shape(\"\") returned %d glyphs, want 0", len(glyphs))
	}
}

func TestSfntShaperMeasureConsistency(t *testing.T) {
	s := newTestShaper()
	m := s.Measure("Hello World", bodyStyle)
	glyphs := s.Shape("Hello World", bodyStyle)

	var sumAdvance float32
	for _, g := range glyphs {
		sumAdvance += g.Advance
	}

	// Width from Measure and sum of advances should be close.
	diff := m.Width - sumAdvance
	if diff < -1 || diff > 1 {
		t.Errorf("Measure width (%f) and sum of advances (%f) differ by %f",
			m.Width, sumAdvance, diff)
	}
}

func TestSfntShaperRasterizeGlyph(t *testing.T) {
	s := newTestShaper()
	img, f := s.RasterizeGlyph('A', bodyStyle)
	if img == nil {
		t.Fatal("RasterizeGlyph('A') returned nil image")
	}
	if f == nil {
		t.Fatal("RasterizeGlyph('A') returned nil font")
	}
	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Errorf("glyph image bounds = %v, want non-zero", img.Bounds())
	}

	// Verify some pixels are non-zero (the glyph was actually drawn).
	hasContent := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.GrayAt(x, y).Y > 0 {
				hasContent = true
				break
			}
		}
		if hasContent {
			break
		}
	}
	if !hasContent {
		t.Error("rasterized glyph image has no content")
	}
}
