package text

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"golang.org/x/image/font/sfnt"
)

func newGoTextTestShaper() *GoTextShaper {
	return NewGoTextShaper(fonts.Fallback)
}

var goTextBodyStyle = draw.TextStyle{
	Size:   13,
	Weight: draw.FontWeightRegular,
}

func TestGoTextShaperMeasure(t *testing.T) {
	s := newGoTextTestShaper()
	m := s.Measure("Hello", goTextBodyStyle)
	if m.Width <= 0 {
		t.Errorf("Width = %f, want > 0", m.Width)
	}
	if m.Ascent <= 0 {
		t.Errorf("Ascent = %f, want > 0", m.Ascent)
	}
}

func TestGoTextShaperMeasureEmpty(t *testing.T) {
	s := newGoTextTestShaper()
	m := s.Measure("", goTextBodyStyle)
	if m.Width != 0 {
		t.Errorf("empty string Width = %f, want 0", m.Width)
	}
}

func TestGoTextShaperShape(t *testing.T) {
	s := newGoTextTestShaper()
	glyphs := s.Shape("ABC", goTextBodyStyle)
	if len(glyphs) != 3 {
		t.Fatalf("Shape(\"ABC\") returned %d glyphs, want 3", len(glyphs))
	}
	for i, g := range glyphs {
		if g.Advance <= 0 {
			t.Errorf("glyph[%d] (%c) Advance = %f, want > 0", i, g.Rune, g.Advance)
		}
	}
}

func TestGoTextShaperShapeEmpty(t *testing.T) {
	s := newGoTextTestShaper()
	glyphs := s.Shape("", goTextBodyStyle)
	if len(glyphs) != 0 {
		t.Errorf("Shape(\"\") returned %d glyphs, want 0", len(glyphs))
	}
}

func TestGoTextShaperMeasureConsistency(t *testing.T) {
	s := newGoTextTestShaper()
	m := s.Measure("Hello World", goTextBodyStyle)
	glyphs := s.Shape("Hello World", goTextBodyStyle)

	var sumAdvance float32
	for _, g := range glyphs {
		sumAdvance += g.Advance
	}

	diff := m.Width - sumAdvance
	if diff < -2 || diff > 2 {
		t.Errorf("Measure width (%f) and sum of advances (%f) differ by %f",
			m.Width, sumAdvance, diff)
	}
}

func TestGoTextShaperRasterizeGlyph(t *testing.T) {
	s := newGoTextTestShaper()
	f := s.ResolveFont(goTextBodyStyle)
	if f == nil {
		t.Fatal("ResolveFont returned nil")
	}
	var buf sfnt.Buffer
	glyphIdx, err := f.SfntFont().GlyphIndex(&buf, 'A')
	if err != nil {
		t.Fatalf("GlyphIndex('A') failed: %v", err)
	}
	rg := s.RasterizeGlyph(GlyphID(glyphIdx), goTextBodyStyle)
	if rg == nil {
		t.Fatal("RasterizeGlyph('A') returned nil")
	}
	if rg.Image == nil {
		t.Fatal("RasterizeGlyph('A') returned nil image")
	}
	if rg.Font == nil {
		t.Fatal("RasterizeGlyph('A') returned nil font")
	}
	if rg.Image.Bounds().Dx() <= 0 || rg.Image.Bounds().Dy() <= 0 {
		t.Errorf("glyph image bounds = %v, want non-zero", rg.Image.Bounds())
	}

	hasContent := false
	for y := rg.Image.Bounds().Min.Y; y < rg.Image.Bounds().Max.Y; y++ {
		for x := rg.Image.Bounds().Min.X; x < rg.Image.Bounds().Max.X; x++ {
			if rg.Image.GrayAt(x, y).Y > 0 {
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

func TestGoTextShaperGlyphID(t *testing.T) {
	s := newGoTextTestShaper()
	glyphs := s.Shape("A", goTextBodyStyle)
	if len(glyphs) != 1 {
		t.Fatalf("Shape(\"A\") returned %d glyphs, want 1", len(glyphs))
	}
	if glyphs[0].GlyphID == 0 {
		t.Error("GlyphID for 'A' should be non-zero")
	}
}

func TestGoTextShaperCluster(t *testing.T) {
	s := newGoTextTestShaper()
	glyphs := s.Shape("Hello", goTextBodyStyle)
	if len(glyphs) != 5 {
		t.Fatalf("Shape(\"Hello\") returned %d glyphs, want 5", len(glyphs))
	}
	for i, g := range glyphs {
		if g.Cluster != i {
			t.Errorf("glyph[%d] Cluster = %d, want %d", i, g.Cluster, i)
		}
	}
}

func TestGoTextShaperFallback(t *testing.T) {
	s := newGoTextTestShaper()
	// U+FFFD is the replacement character — should exist in Noto Sans.
	glyphs := s.Shape("\uFFFD", goTextBodyStyle)
	if len(glyphs) != 1 {
		t.Fatalf("Shape(U+FFFD) returned %d glyphs, want 1", len(glyphs))
	}
	if glyphs[0].Advance <= 0 {
		t.Errorf("U+FFFD advance = %f, want > 0", glyphs[0].Advance)
	}
}

func TestGoTextShaperRegisterFamily(t *testing.T) {
	s := newGoTextTestShaper()
	s.RegisterFamily(fonts.PhosphorFamily)

	// The shaper should now be able to resolve fonts from the Phosphor family.
	f := s.ResolveFont(draw.TextStyle{FontFamily: "Phosphor", Size: 14})
	if f == nil {
		t.Error("ResolveFont with registered Phosphor family returned nil")
	}
}

func TestFontFamilyFindGlyphFont(t *testing.T) {
	f := fonts.Fallback.FindGlyphFont('A', 400)
	if f == nil {
		t.Fatal("FindGlyphFont('A', 400) returned nil")
	}
	if f.IsBitmap() {
		t.Error("FindGlyphFont returned bitmap font for 'A'")
	}
}

func TestFontFamilyFindGlyphFontMissing(t *testing.T) {
	// U+FFFF is not a valid Unicode character — no font should have it.
	f := fonts.Fallback.FindGlyphFont('\uFFFF', 400)
	if f != nil {
		t.Errorf("FindGlyphFont(U+FFFF) should return nil, got %v", f.Name())
	}
}
