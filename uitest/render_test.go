package uitest

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

// -render-out writes the images rendered by these tests as PNG files into
// the given directory, for looking at them.
var renderOut = flag.String("render-out", "", "directory to write rendered PNGs to")

func savePNG(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	if *renderOut == "" {
		return
	}
	if err := os.MkdirAll(*renderOut, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*renderOut, name+".png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// inkIn counts pixels in r that differ from bg by more than tol in any channel.
func inkIn(img *image.RGBA, r image.Rectangle, bg [4]uint8, tol int) int {
	n := 0
	r = r.Intersect(img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			o := img.PixOffset(x, y)
			for c := 0; c < 4; c++ {
				d := int(img.Pix[o+c]) - int(bg[c])
				if d > tol || d < -tol {
					n++
					break
				}
			}
		}
	}
	return n
}

func bgOf(th theme.Theme) [4]uint8 {
	c := th.Tokens().Colors.Surface.Base
	f := func(v float32) uint8 { return uint8(v*c.A*255 + 0.5) }
	return [4]uint8{f(c.R), f(c.G), f(c.B), uint8(c.A*255 + 0.5)}
}

func TestRenderImageBackground(t *testing.T) {
	img := RenderImage(display.Empty(), 64, 32)
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 32 {
		t.Fatalf("bounds = %v", b)
	}
	if n := inkIn(img, img.Rect, bgOf(theme.Default), 1); n != 0 {
		t.Errorf("empty element: %d pixels differ from the theme background", n)
	}
}

func TestRenderImageText(t *testing.T) {
	img := RenderImage(display.Text("Hello World"), 400, 100)
	savePNG(t, "text", img)
	bg := bgOf(theme.Default)
	// Text is laid out at the top left; it must leave ink there and nowhere else.
	if n := inkIn(img, image.Rect(0, 0, 200, 40), bg, 8); n < 50 {
		t.Errorf("text area has only %d ink pixels", n)
	}
	if n := inkIn(img, image.Rect(0, 60, 400, 100), bg, 8); n != 0 {
		t.Errorf("%d ink pixels below the text", n)
	}
}

func TestRenderImageLargeTextMSDF(t *testing.T) {
	// Large sizes take the MSDF path, rasterized asynchronously.
	img := RenderImage(display.TextStyled("Lux", draw.TextStyle{Size: 64}), 300, 120)
	savePNG(t, "msdf", img)
	if n := inkIn(img, image.Rect(0, 0, 300, 120), bgOf(theme.Default), 16); n < 500 {
		t.Errorf("MSDF text has only %d ink pixels", n)
	}
}

func TestRenderImageWidgets(t *testing.T) {
	root := layout.Column(
		display.Text("Settings"),
		button.Text("Save", func() {}),
		display.Divider(),
		display.GradientRect(200, 40, 8, draw.LinearGradientPaint(
			draw.Pt(0, 0), draw.Pt(200, 0),
			draw.GradientStop{Offset: 0, Color: draw.Hex("#ff0000")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#0000ff")},
		)),
		display.Card(display.Text("Inside a card")),
	)
	img := RenderImage(root, 400, 300)
	savePNG(t, "widgets", img)

	// The gradient must show both of its ends somewhere in the frame.
	var reddish, bluish bool
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 230 && c.B < 30 && c.G < 30 {
				reddish = true
			}
			if c.B > 230 && c.R < 30 && c.G < 30 {
				bluish = true
			}
		}
	}
	if !reddish || !bluish {
		t.Errorf("gradient ends missing: red=%v blue=%v", reddish, bluish)
	}
}

func TestRenderImageDeterministic(t *testing.T) {
	root := layout.Column(
		display.Text("Determinism"),
		button.Text("OK", func() {}),
		display.TextStyled("Big", draw.TextStyle{Size: 48}),
	)
	a := RenderImage(root, 320, 200)
	b := RenderImage(root, 320, 200)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("two renders of the same tree differ")
	}
}

func TestRenderImageMatchesScene(t *testing.T) {
	// Every rect of the scene lands in the frame: sample the centre of each
	// opaque, unclipped rect drawn last at that point.
	root := layout.Column(button.Text("A", func() {}), button.Text("B", func() {}))
	img := RenderImage(root, 200, 120)
	scene := BuildScene(root, 200, 120)
	checked := 0
	for i, r := range scene.Rects {
		if r.Color.A < 1 || r.W < 6 || r.H < 6 {
			continue
		}
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		covered := false
		for _, s := range scene.Rects[i+1:] {
			if cx >= s.X && cx < s.X+s.W && cy >= s.Y && cy < s.Y+s.H {
				covered = true
			}
		}
		if covered {
			continue
		}
		// Text glyphs may sit on top of the rect centre; sample a corner-ish
		// point inside the rect instead, clear of the radius.
		sx, sy := r.X+int(r.Radius)+2, r.Y+r.H/2
		if sx >= r.X+r.W {
			continue
		}
		got := img.RGBAAt(sx, sy)
		want := color8(r.Color)
		// The default theme adds film grain of ±Grain/2 to rect colors.
		tol := int(scene.Grain*255/2) + 2
		if absDiff(got.R, want[0]) > tol || absDiff(got.G, want[1]) > tol || absDiff(got.B, want[2]) > tol {
			t.Errorf("rect %d at (%d,%d): pixel %v, want %v", i, sx, sy, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no rects checked")
	}
}

func color8(c draw.Color) [3]uint8 {
	f := func(v float32) uint8 { return uint8(v*255 + 0.5) }
	return [3]uint8{f(c.R), f(c.G), f(c.B)}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
