package cerasurface

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/timzifer/cera"
)

// drawingPDF builds an A4 landscape "technical drawing": a grid, 3000
// stroked segments, circles and 120 lines of Helvetica text.
func drawingPDF(tb testing.TB) *cera.Page {
	tb.Helper()
	var c bytes.Buffer
	c.WriteString("0.2 w 0.6 g\n")
	for x := 0; x <= 842; x += 10 {
		fmt.Fprintf(&c, "%d 0 m %d 595 l\n", x, x)
	}
	for y := 0; y <= 595; y += 10 {
		fmt.Fprintf(&c, "0 %d m 842 %d l\n", y, y)
	}
	c.WriteString("S 0.6 w 0 0 0.5 RG\n")
	for i := 0; i < 3000; i++ {
		a := float64(i) * 0.37
		x, y := 421+300*math.Cos(a)*math.Sin(float64(i)*0.011), 297+250*math.Sin(a*1.3)
		fmt.Fprintf(&c, "%.2f %.2f m %.2f %.2f l\n", x, y, x+20*math.Cos(a*3), y+20*math.Sin(a*2))
	}
	c.WriteString("S 1 0 0 RG 1 w\n")
	for i := 0; i < 60; i++ {
		cx, cy, r := float64(60+(i%12)*65), float64(60+(i/12)*110), 18.0
		k := 0.5523 * r
		fmt.Fprintf(&c, "%.1f %.1f m %.1f %.1f %.1f %.1f %.1f %.1f c %.1f %.1f %.1f %.1f %.1f %.1f c %.1f %.1f %.1f %.1f %.1f %.1f c %.1f %.1f %.1f %.1f %.1f %.1f c S\n",
			cx+r, cy, cx+r, cy+k, cx+k, cy+r, cx, cy+r, cx-k, cy+r, cx-r, cy+k, cx-r, cy, cx-r, cy-k, cx-k, cy-r, cx, cy-r, cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	}
	c.WriteString("BT /F1 7 Tf 0 g\n")
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&c, "1 0 0 1 %d %d Tm (Part %04d  M6x20  DIN 912  qty %d  rev C) Tj\n", 20+(i/60)*420, 585-(i%60)*9, i, i%9+1)
	}
	c.WriteString("ET\n")
	content := c.String()
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 842 595] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
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
		tb.Fatal(err)
	}
	p, err := doc.Page(0)
	if err != nil {
		tb.Fatal(err)
	}
	return p
}

// BenchmarkSurfaceFrame is the per-frame cost of a PDF surface filling a
// 1600×1130 view (the page fitted, ~140 dpi) inside a 1920×1200 framebuffer:
// what the CPU renderer pays for it, drawn in place.
func BenchmarkSurfaceFrame(b *testing.B) {
	page := drawingPDF(b)
	for _, workers := range []int{1, 0} {
		name := "workers=all"
		if workers == 1 {
			name = "workers=1"
		}
		b.Run(name, func(b *testing.B) {
			dst := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
			p := &Page{Page: page, Workers: workers}
			rect := image.Rect(160, 35, 1760, 1165)
			p.RasterizeSurface(dst, rect, rect) // display list
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.RasterizeSurface(dst, rect, rect)
			}
			if p.Err != nil {
				b.Fatal(p.Err)
			}
		})
	}
}
