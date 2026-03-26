package text

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"golang.org/x/image/font/sfnt"
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
	f := s.ResolveFont(bodyStyle)
	if f == nil {
		t.Fatal("ResolveFont returned nil")
	}
	var buf sfnt.Buffer
	glyphIdx, err := f.SfntFont().GlyphIndex(&buf, 'A')
	if err != nil {
		t.Fatalf("GlyphIndex('A') failed: %v", err)
	}
	rg := s.RasterizeGlyph(GlyphID(glyphIdx), bodyStyle)
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

	// Verify some pixels are non-zero (the glyph was actually drawn).
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

func TestMSDFBearingYAlignment(t *testing.T) {
	// Flat-top capital letters share the same cap-height in well-formed fonts.
	// Their MSDF BearingY values must be identical to prevent vertical jitter.
	// Round letters (S, O, C, …) are excluded because they have intentional
	// typographic overshoots that extend slightly above the cap line.
	capLetters := []rune{'A', 'B', 'E', 'H', 'T'}
	const atlasSize = 32
	const pxRange = float32(MSDFPxRange)

	s := newTestShaper()
	f := s.ResolveFont(bodyStyle)
	if f == nil {
		t.Fatal("ResolveFont returned nil")
	}
	sf := f.SfntFont()
	if sf == nil {
		t.Fatal("SfntFont returned nil")
	}

	var buf sfnt.Buffer
	var bearings []float32
	for _, r := range capLetters {
		glyphIdx, err := sf.GlyphIndex(&buf, r)
		if err != nil {
			t.Fatalf("GlyphIndex(%q) failed: %v", r, err)
		}
		rg := s.RasterizeMSDFGlyph(GlyphID(glyphIdx), r, f, atlasSize, pxRange)
		if rg == nil {
			t.Fatalf("RasterizeMSDFGlyph(%q) returned nil", r)
		}
		bearings = append(bearings, rg.BearingY)
	}

	for i := 1; i < len(bearings); i++ {
		if bearings[i] != bearings[0] {
			t.Errorf("BearingY mismatch: %q has %g but %q has %g",
				capLetters[0], bearings[0], capLetters[i], bearings[i])
		}
	}
}
