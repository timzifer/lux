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
