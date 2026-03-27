package image

import (
	"image/color"
	"image/png"
	"bytes"
	goimage "image"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
}

func TestLoadFromBytes_PNG(t *testing.T) {
	// Create a 2x2 red PNG in memory.
	img := goimage.NewNRGBA(goimage.Rect(0, 0, 2, 2))
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	img.Set(0, 0, red)
	img.Set(1, 0, red)
	img.Set(0, 1, red)
	img.Set(1, 1, red)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	s := NewStore()
	id, err := s.LoadFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	if id == 0 {
		t.Fatal("got zero ImageID")
	}

	w, h := s.Size(id)
	if w != 2 || h != 2 {
		t.Fatalf("Size = (%d, %d), want (2, 2)", w, h)
	}

	entry := s.Get(id)
	if entry == nil {
		t.Fatal("Get returned nil")
	}
	if len(entry.RGBA) != 2*2*4 {
		t.Fatalf("RGBA length = %d, want %d", len(entry.RGBA), 2*2*4)
	}

	// Check first pixel is red (RGBA).
	if entry.RGBA[0] != 255 || entry.RGBA[1] != 0 || entry.RGBA[2] != 0 || entry.RGBA[3] != 255 {
		t.Fatalf("pixel 0 = %v, want [255 0 0 255]", entry.RGBA[:4])
	}

	if !entry.Dirty() {
		t.Fatal("expected dirty=true for new image")
	}
	entry.ClearDirty()
	if entry.Dirty() {
		t.Fatal("expected dirty=false after ClearDirty")
	}
}

func TestLoadFromRGBA(t *testing.T) {
	s := NewStore()
	rgba := []byte{255, 0, 0, 255, 0, 255, 0, 255}
	id, err := s.LoadFromRGBA(2, 1, rgba)
	if err != nil {
		t.Fatalf("LoadFromRGBA: %v", err)
	}
	w, h := s.Size(id)
	if w != 2 || h != 1 {
		t.Fatalf("Size = (%d, %d), want (2, 1)", w, h)
	}
}

func TestLoadFromRGBA_BadSize(t *testing.T) {
	s := NewStore()
	_, err := s.LoadFromRGBA(2, 2, []byte{0, 0, 0, 0})
	if err == nil {
		t.Fatal("expected error for wrong data size")
	}
}

func TestRemove(t *testing.T) {
	s := NewStore()
	rgba := []byte{0, 0, 0, 255, 0, 0, 0, 255, 0, 0, 0, 255, 0, 0, 0, 255}
	id, _ := s.LoadFromRGBA(2, 2, rgba)
	s.Remove(id)
	if s.Get(id) != nil {
		t.Fatal("Get should return nil after Remove")
	}
	w, h := s.Size(id)
	if w != 0 || h != 0 {
		t.Fatalf("Size after Remove = (%d, %d), want (0, 0)", w, h)
	}
}

func TestLoadFromBytes_Empty(t *testing.T) {
	s := NewStore()
	_, err := s.LoadFromBytes(nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
}

func TestLoadSVG(t *testing.T) {
	s := NewStore()
	id, err := s.LoadSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect x="0" y="0" width="100" height="100" fill="red"/></svg>`))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if id == 0 {
		t.Fatal("LoadSVG returned zero ID")
	}
}

func TestMultipleImages(t *testing.T) {
	s := NewStore()
	rgba := make([]byte, 4*4*4)
	id1, _ := s.LoadFromRGBA(4, 4, rgba)
	id2, _ := s.LoadFromRGBA(4, 4, rgba)
	if id1 == id2 {
		t.Fatal("expected different IDs for different images")
	}
	if s.Get(id1) == nil || s.Get(id2) == nil {
		t.Fatal("both images should be accessible")
	}
}
