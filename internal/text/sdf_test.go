package text

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// fixedPt creates a fixed.Point26_6 from float32 values.
func fixedPt(x, y float32) fixed.Point26_6 {
	return fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}
}

// makeSquareOutline creates a square outline as sfnt.Segments:
// (x0,y0) → (x1,y0) → (x1,y1) → (x0,y1) → close.
func makeSquareOutline(x0, y0, x1, y1 float32) sfnt.Segments {
	return sfnt.Segments{
		{Op: sfnt.SegmentOpMoveTo, Args: [3]fixed.Point26_6{fixedPt(x0, y0)}},
		{Op: sfnt.SegmentOpLineTo, Args: [3]fixed.Point26_6{fixedPt(x1, y0)}},
		{Op: sfnt.SegmentOpLineTo, Args: [3]fixed.Point26_6{fixedPt(x1, y1)}},
		{Op: sfnt.SegmentOpLineTo, Args: [3]fixed.Point26_6{fixedPt(x0, y1)}},
		{Op: sfnt.SegmentOpLineTo, Args: [3]fixed.Point26_6{fixedPt(x0, y0)}},
	}
}

func TestWindingNumber(t *testing.T) {
	// Square from (2,2) to (8,8).
	segs := convertSegments(makeSquareOutline(2, 2, 8, 8))

	tests := []struct {
		name   string
		px, py float32
		inside bool
	}{
		{"center", 5, 5, true},
		{"inside near edge", 3, 3, true},
		{"outside left", 1, 5, false},
		{"outside right", 9, 5, false},
		{"outside above", 5, 1, false},
		{"outside below", 5, 9, false},
		{"outside corner", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wn := windingNumber(tt.px, tt.py, segs)
			got := wn != 0
			if got != tt.inside {
				t.Errorf("windingNumber(%g, %g) = %d, inside=%v; want inside=%v",
					tt.px, tt.py, wn, got, tt.inside)
			}
		})
	}
}

func TestDistToLine(t *testing.T) {
	tests := []struct {
		name string
		px, py float32
		a, b   [2]float32
		want   float32
	}{
		{"perpendicular", 0, 1, [2]float32{0, 0}, [2]float32{10, 0}, 1},
		{"at endpoint a", 0, 0, [2]float32{0, 0}, [2]float32{10, 0}, 0},
		{"beyond endpoint a", -1, 0, [2]float32{0, 0}, [2]float32{10, 0}, 1},
		{"midpoint offset", 5, 3, [2]float32{0, 0}, [2]float32{10, 0}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distToLine(tt.px, tt.py, tt.a, tt.b)
			if math.Abs(float64(got-tt.want)) > 0.01 {
				t.Errorf("distToLine(%g,%g, %v,%v) = %g; want %g",
					tt.px, tt.py, tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDistToQuadBezier(t *testing.T) {
	// A simple arch: p0=(0,0), p1=(5,10), p2=(10,0).
	p0 := [2]float32{0, 0}
	p1 := [2]float32{5, 10}
	p2 := [2]float32{10, 0}

	// Point at the apex of the curve: B(0.5) = (5, 5).
	// Distance from (5, 5) to the curve should be ~0.
	d := distToQuadBezier(5, 5, p0, p1, p2)
	if d > 0.1 {
		t.Errorf("distToQuadBezier(5,5) = %g; want ~0", d)
	}

	// Point well outside the curve.
	d = distToQuadBezier(5, 10, p0, p1, p2)
	if d < 4.5 {
		t.Errorf("distToQuadBezier(5,10) = %g; want > 4.5", d)
	}
}

func TestDistToCubicBezier(t *testing.T) {
	// S-curve: p0=(0,0), p1=(3,10), p2=(7,-5), p3=(10,0).
	p0 := [2]float32{0, 0}
	p1 := [2]float32{3, 10}
	p2 := [2]float32{7, -5}
	p3 := [2]float32{10, 0}

	// Start point should be on the curve.
	d := distToCubicBezier(0, 0, p0, p1, p2, p3)
	if d > 0.01 {
		t.Errorf("distToCubicBezier(0,0) = %g; want ~0", d)
	}

	// End point should be on the curve.
	d = distToCubicBezier(10, 0, p0, p1, p2, p3)
	if d > 0.01 {
		t.Errorf("distToCubicBezier(10,0) = %g; want ~0", d)
	}
}

func TestCorrectMSDFCorners(t *testing.T) {
	// Create a 10x10 MSDF image for a square outline from (2,2) to (8,8).
	// Fill all pixels with "inside" (R=G=B=200, which is > 128 = inside).
	// Then set corner pixel (1,1) to "inside" even though it's outside the square.
	// This simulates a corner artifact.
	w, h := 10, 10
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	pxRange := float32(4.0)
	outline := makeSquareOutline(2, 2, 8, 8)

	// Fill image: pixels inside the square get high values, outside get low values.
	// But deliberately set (1,1) to high value (artifact).
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			// Map pixel to outline space (identity: plane bounds = 0,0,10,10).
			ox := float32(px) + 0.5
			oy := float32(py) + 0.5
			inside := ox >= 2 && ox <= 8 && oy >= 2 && oy <= 8
			var v uint8
			if inside {
				v = 200
			} else {
				v = 50
			}
			img.SetNRGBA(px, py, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}

	// Inject corner artifact at pixel (1,1): outside the square but channels
	// strongly disagree (high spread) with median > 0.5 — classic MSDF corner artifact.
	// R=220 (inside), G=30 (outside), B=200 (inside) → median=200, spread=190/255≈0.75.
	img.SetNRGBA(1, 1, color.NRGBA{R: 220, G: 30, B: 200, A: 255})

	correctMSDFCorners(img, outline, 0, 0, 10, 10, pxRange)

	// After correction, pixel (1,1) should be corrected to outside (< 128).
	c := img.NRGBAAt(1, 1)
	if c.R >= 128 {
		t.Errorf("after correction, pixel (1,1) R=%d; want < 128 (outside)", c.R)
	}

	// Verify that a normal outside pixel with agreeing channels was NOT corrected.
	c = img.NRGBAAt(0, 0)
	if c.R != 50 {
		t.Errorf("normal outside pixel (0,0) was modified: R=%d; want 50", c.R)
	}

	// Verify a known-inside pixel (5,5) was NOT changed.
	c = img.NRGBAAt(5, 5)
	if c.R != 200 {
		t.Errorf("inside pixel (5,5) was modified: R=%d; want 200", c.R)
	}
}

func TestMedian3f(t *testing.T) {
	tests := []struct {
		a, b, c, want float32
	}{
		{0.1, 0.5, 0.9, 0.5},
		{0.9, 0.1, 0.5, 0.5},
		{0.3, 0.3, 0.3, 0.3},
		{0.0, 1.0, 0.5, 0.5},
	}
	for _, tt := range tests {
		got := median3f(tt.a, tt.b, tt.c)
		if math.Abs(float64(got-tt.want)) > 0.001 {
			t.Errorf("median3f(%g,%g,%g) = %g; want %g", tt.a, tt.b, tt.c, got, tt.want)
		}
	}
}

func BenchmarkCorrectMSDFCorners(b *testing.B) {
	// Simulate a 32x32 MSDF glyph with a square outline.
	w, h := 32, 32
	outline := makeSquareOutline(4, 4, 28, 28)
	pxRange := float32(4.0)

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			img.SetNRGBA(px, py, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		correctMSDFCorners(img, outline, 0, 0, 32, 32, pxRange)
	}
}
