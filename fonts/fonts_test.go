package fonts

import "testing"

func TestLoadBytesEmbedded(t *testing.T) {
	data, err := notoFS.ReadFile("noto/NotoSans-Regular.ttf")
	if err != nil {
		t.Fatalf("reading embedded font: %v", err)
	}
	f, err := LoadBytes(data)
	if err != nil {
		t.Fatalf("LoadBytes: %v", err)
	}
	if f.SfntFont() == nil {
		t.Fatal("SfntFont() should be non-nil for loaded font")
	}
	if f.IsBitmap() {
		t.Fatal("IsBitmap() should be false for loaded font")
	}
}

func TestLoadBytesInvalid(t *testing.T) {
	_, err := LoadBytes([]byte("not a font"))
	if err == nil {
		t.Fatal("LoadBytes with invalid data should return error")
	}
}

func TestDefaultFont(t *testing.T) {
	f := DefaultFont()
	if f == nil {
		t.Fatal("DefaultFont() should not be nil")
	}
	if f.IsBitmap() {
		t.Fatal("DefaultFont should not be bitmap")
	}
	if f.Name() != "Noto Sans" {
		t.Errorf("DefaultFont name = %q, want %q", f.Name(), "Noto Sans")
	}
}

func TestFallbackHasSfntFace(t *testing.T) {
	key := FontFaceKey{Weight: 400, Style: StyleNormal}
	face, ok := Fallback.Faces[key]
	if !ok {
		t.Fatal("Fallback should have a Regular face")
	}
	if face.IsBitmap() {
		t.Fatal("Fallback Regular face should be an sfnt font")
	}
}

func TestBitmapGlyphStillWorks(t *testing.T) {
	g := BitmapGlyph('A')
	if g[0] != "01110" {
		t.Errorf("BitmapGlyph('A')[0] = %q, want %q", g[0], "01110")
	}
}

// ── Phase 4.1 go-text/typesetting tests ──────────────────────────

func TestFontGoTextFace(t *testing.T) {
	f := DefaultFont()
	if f == nil {
		t.Fatal("DefaultFont() should not be nil")
	}
	if f.GoTextFace() == nil {
		t.Fatal("GoTextFace() should be non-nil for loaded font")
	}
}

func TestFontHasGlyph(t *testing.T) {
	f := DefaultFont()
	if f == nil {
		t.Fatal("DefaultFont() should not be nil")
	}
	if !f.HasGlyph('A') {
		t.Error("HasGlyph('A') should return true for Noto Sans")
	}
	if !f.HasGlyph('0') {
		t.Error("HasGlyph('0') should return true for Noto Sans")
	}
}

func TestFontHasGlyphMissing(t *testing.T) {
	f := DefaultFont()
	if f == nil {
		t.Fatal("DefaultFont() should not be nil")
	}
	// U+E000 is a Private Use Area codepoint — unlikely in Noto Sans.
	if f.HasGlyph('\uE000') {
		t.Error("HasGlyph(U+E000) should return false for Noto Sans")
	}
}

func TestFindGlyphFont(t *testing.T) {
	f := Fallback.FindGlyphFont('A', 400)
	if f == nil {
		t.Fatal("FindGlyphFont('A', 400) should return non-nil")
	}
}

func TestFindGlyphFontMissing(t *testing.T) {
	// U+FFFF is permanently unassigned — no font should have it.
	f := Fallback.FindGlyphFont('\uFFFF', 400)
	if f != nil {
		t.Errorf("FindGlyphFont(U+FFFF) should return nil, got %q", f.Name())
	}
}

func TestFindGlyphFontNilFamily(t *testing.T) {
	var ff *FontFamily
	f := ff.FindGlyphFont('A', 400)
	if f != nil {
		t.Error("FindGlyphFont on nil family should return nil")
	}
}
