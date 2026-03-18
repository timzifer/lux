package text

import (
	"image"
	"image/color"
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
)

func TestAtlasInsertAndLookup(t *testing.T) {
	atlas := NewGlyphAtlas(256, 256)

	// Create a small test glyph image.
	img := image.NewGray(image.Rect(0, 0, 10, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 200})
		}
	}

	key := GlyphKey{FontID: 1, Rune: 'A', SizePx: 13}
	entry := atlas.Insert(key, img, draw.Pt(1, 10), 8.0)

	if entry.W != 10 || entry.H != 12 {
		t.Errorf("entry size = %dx%d, want 10x12", entry.W, entry.H)
	}
	if entry.Advance != 8.0 {
		t.Errorf("entry Advance = %f, want 8.0", entry.Advance)
	}

	// Lookup should return the same entry.
	found, ok := atlas.Lookup(key)
	if !ok {
		t.Fatal("Lookup should find the inserted entry")
	}
	if found != entry {
		t.Errorf("Lookup returned %+v, want %+v", found, entry)
	}

	if !atlas.Dirty {
		t.Error("atlas should be dirty after insert")
	}
}

func TestAtlasCaching(t *testing.T) {
	atlas := NewGlyphAtlas(256, 256)

	img := image.NewGray(image.Rect(0, 0, 8, 10))
	key := GlyphKey{FontID: 1, Rune: 'B', SizePx: 13}

	entry1 := atlas.Insert(key, img, draw.Pt(0, 8), 7.0)
	entry2, ok := atlas.Lookup(key)
	if !ok {
		t.Fatal("second lookup failed")
	}
	if entry1 != entry2 {
		t.Error("cached entry should match original")
	}
}

func TestAtlasGrowth(t *testing.T) {
	atlas := NewGlyphAtlas(64, 64)

	img := image.NewGray(image.Rect(0, 0, 10, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	// Insert enough glyphs to force growth.
	for i := 0; i < 100; i++ {
		key := GlyphKey{FontID: 1, Rune: rune('A' + i%26), SizePx: uint16(10 + i)}
		atlas.Insert(key, img, draw.Pt(0, 10), 8.0)
	}

	if atlas.Height <= 64 {
		t.Errorf("atlas should have grown, height = %d", atlas.Height)
	}

	// Verify the first entry is still accessible.
	key := GlyphKey{FontID: 1, Rune: 'A', SizePx: 10}
	_, ok := atlas.Lookup(key)
	if !ok {
		t.Error("first entry should still be in atlas after growth")
	}
}

func TestAtlasLookupOrInsert(t *testing.T) {
	atlas := NewGlyphAtlas(512, 512)
	shaper := NewSfntShaper(fonts.Fallback)

	style := draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular}
	f := fonts.DefaultFont()
	if f == nil {
		t.Skip("no default font available")
	}

	key := GlyphKey{FontID: f.ID(), Rune: 'H', SizePx: 13}
	entry, ok := atlas.LookupOrInsert(key, shaper, style)
	if !ok {
		t.Fatal("LookupOrInsert should succeed for 'H'")
	}
	if entry.W <= 0 || entry.H <= 0 {
		t.Errorf("entry size = %dx%d, want > 0", entry.W, entry.H)
	}

	// Second call should return cached.
	entry2, ok := atlas.LookupOrInsert(key, shaper, style)
	if !ok {
		t.Fatal("second LookupOrInsert should succeed")
	}
	if entry2 != entry {
		t.Error("cached entry should match original")
	}
}

func TestAtlasDirtyFlag(t *testing.T) {
	atlas := NewGlyphAtlas(256, 256)
	if atlas.Dirty {
		t.Error("new atlas should not be dirty")
	}

	img := image.NewGray(image.Rect(0, 0, 5, 5))
	atlas.Insert(GlyphKey{FontID: 1, Rune: 'X', SizePx: 10}, img, draw.Pt(0, 5), 5.0)
	if !atlas.Dirty {
		t.Error("atlas should be dirty after insert")
	}

	atlas.Dirty = false
	if atlas.Dirty {
		t.Error("clearing dirty flag should work")
	}
}
