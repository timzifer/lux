package cerasurface

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/timzifer/cera"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/uitest"
)

// testPDF is a 100×100 pt page: white, with a red square at x 0..50,
// y 50..100 in PDF space (the top-left quarter on screen) and a blue one
// at the bottom right quarter.
func testPDF(t testing.TB) *cera.Page {
	t.Helper()
	content := "1 0 0 rg 0 50 50 50 re f 0 0 1 rg 50 0 50 50 re f"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	doc, err := cera.Open(b.Bytes())
	if err != nil {
		t.Fatalf("cera.Open: %v", err)
	}
	p, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var (
	red   = color.RGBA{255, 0, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
	white = color.RGBA{255, 255, 255, 255}
	grey  = color.RGBA{9, 9, 9, 255}
)

func fill(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
}

func want(t *testing.T, img *image.RGBA, x, y int, c color.RGBA) {
	t.Helper()
	if got := img.RGBAAt(x, y); got != c {
		t.Errorf("(%d,%d) = %v, want %v", x, y, got, c)
	}
}

func TestRasterizeFitsPageAtOffset(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 300, 300))
	fill(dst, grey)
	p := &Page{Page: testPDF(t)}
	rect := image.Rect(50, 100, 250, 300) // 200 px wide: scale 2
	p.RasterizeSurface(dst, rect, rect)
	if p.Err != nil {
		t.Fatal(p.Err)
	}
	want(t, dst, 60, 110, red)    // top-left quarter
	want(t, dst, 240, 290, blue)  // bottom-right quarter
	want(t, dst, 240, 110, white) // paper
	want(t, dst, 40, 110, grey)   // outside the surface untouched
	want(t, dst, 60, 90, grey)
}

func TestRasterizeRespectsClipAndScroll(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	fill(dst, grey)
	p := &Page{Page: testPDF(t), Scale: 2, Scroll: image.Pt(100, 100)}
	rect := image.Rect(0, 0, 200, 200)
	clip := image.Rect(0, 0, 50, 200)
	p.RasterizeSurface(dst, rect, clip)
	// Scrolled by half the page: the blue quarter is at the top left.
	want(t, dst, 10, 10, blue)
	want(t, dst, 60, 10, grey) // clipped away
	// Below the page (scrolled 100 px of 200): untouched.
	want(t, dst, 10, 150, grey)
}

func TestRasterizeNoAlloc(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	p := &Page{Page: testPDF(t), Workers: 1}
	r := dst.Rect
	p.RasterizeSurface(dst, r, r) // warm up: display list
	allocs := testing.AllocsPerRun(10, func() { p.RasterizeSurface(dst, r, r) })
	t.Logf("allocs per frame: %.1f", allocs)
	if allocs > 4 {
		t.Errorf("%.1f allocations per frame", allocs)
	}
}

func TestSurfaceInWidgetTree(t *testing.T) {
	p := &Page{Page: testPDF(t)}
	root := layout.Column(
		display.Text("PDF"),
		ui.Surface(1, p, 200, 200),
	)
	img := uitest.RenderImage(root, 300, 300)
	// The surface starts below the text; find the red quarter.
	found := false
	for y := 0; y < 300 && !found; y++ {
		for x := 0; x < 300; x++ {
			if img.RGBAAt(x, y) == red {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("PDF page not drawn through the widget tree")
	}
	if p.Err != nil {
		t.Fatal(p.Err)
	}
}
