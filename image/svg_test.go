package image

import (
	"testing"
)

func TestLoadSVG_Basic(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
		<rect x="10" y="10" width="80" height="80" fill="red"/>
	</svg>`)
	id, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if id == 0 {
		t.Fatal("LoadSVG returned zero ID")
	}
}

func TestRasterizeSVG(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
		<rect x="0" y="0" width="100" height="100" fill="red"/>
	</svg>`)
	svgID, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}

	rasterID, err := s.RasterizeSVG(svgID, 50, 50)
	if err != nil {
		t.Fatalf("RasterizeSVG: %v", err)
	}
	if rasterID == 0 {
		t.Fatal("RasterizeSVG returned zero ID")
	}

	w, h := s.Size(rasterID)
	if w != 50 || h != 50 {
		t.Fatalf("rasterized size = (%d, %d), want (50, 50)", w, h)
	}

	entry := s.Get(rasterID)
	if entry == nil {
		t.Fatal("Get returned nil for rasterized image")
	}
	if len(entry.RGBA) != 50*50*4 {
		t.Fatalf("RGBA length = %d, want %d", len(entry.RGBA), 50*50*4)
	}
}

func TestRasterizeSVG_Circle(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
		<circle cx="50" cy="50" r="40" fill="blue"/>
	</svg>`)
	svgID, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	rasterID, err := s.RasterizeSVG(svgID, 100, 100)
	if err != nil {
		t.Fatalf("RasterizeSVG: %v", err)
	}

	entry := s.Get(rasterID)
	if entry == nil {
		t.Fatal("Get returned nil")
	}
	// The center pixel should have some blue color.
	cx, cy := 50, 50
	offset := (cy*100 + cx) * 4
	if entry.RGBA[offset+2] == 0 {
		t.Error("expected blue component at center to be non-zero")
	}
}

func TestRasterizeSVG_Path(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
		<path d="M 10 10 L 90 10 L 50 90 Z" fill="green"/>
	</svg>`)
	svgID, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	_, err = s.RasterizeSVG(svgID, 100, 100)
	if err != nil {
		t.Fatalf("RasterizeSVG: %v", err)
	}
}

func TestRasterizeSVG_NotFound(t *testing.T) {
	s := NewStore()
	_, err := s.RasterizeSVG(999, 100, 100)
	if err == nil {
		t.Fatal("expected error for non-existent SVG")
	}
}

func TestRasterizeSVG_InvalidSize(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect x="0" y="0" width="100" height="100" fill="red"/></svg>`)
	svgID, _ := s.LoadSVG(svg)
	_, err := s.RasterizeSVG(svgID, 0, 100)
	if err == nil {
		t.Fatal("expected error for zero width")
	}
}

func TestLoadSVG_Group(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
		<g>
			<rect x="0" y="0" width="50" height="50" fill="red"/>
			<rect x="50" y="50" width="50" height="50" fill="blue"/>
		</g>
	</svg>`)
	svgID, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	_, err = s.RasterizeSVG(svgID, 100, 100)
	if err != nil {
		t.Fatalf("RasterizeSVG: %v", err)
	}
}

func TestLoadSVG_Stroke(t *testing.T) {
	s := NewStore()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
		<line x1="10" y1="10" x2="90" y2="90" stroke="black" stroke-width="2"/>
	</svg>`)
	svgID, err := s.LoadSVG(svg)
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	_, err = s.RasterizeSVG(svgID, 100, 100)
	if err != nil {
		t.Fatalf("RasterizeSVG: %v", err)
	}
}
