//go:build !nogui && ((darwin && arm64) || windows || gogpu)

package gpu

import (
	"image"
	"io"
	"log"
	"runtime"
	"testing"

	"github.com/timzifer/lux/draw"
)

func newOffscreenWGPU(tb testing.TB, w, h int) *WGPURenderer {
	tb.Helper()
	// Metal autorelease pools belong to the thread that made them; the
	// app's render loop is pinned the same way.
	runtime.LockOSThread()
	tb.Cleanup(runtime.UnlockOSThread)
	log.SetOutput(io.Discard)
	tb.Cleanup(func() { log.SetOutput(nil) })
	r := NewWGPU()
	if err := r.Init(Config{Width: w, Height: h, DRMfd: -1, Offscreen: true}); err != nil {
		tb.Skipf("no GPU: %v", err)
	}
	if err := r.WaitIdle(); err != nil {
		tb.Skipf("no readback: %v", err)
	}
	tb.Cleanup(r.Destroy)
	return r
}

func TestWGPUOffscreenReadback(t *testing.T) {
	r := newOffscreenWGPU(t, 64, 32)
	r.SetBackgroundColor(draw.Color{R: 1, G: 1, B: 1, A: 1})
	r.BeginFrame()
	r.Draw(draw.Scene{Rects: []draw.DrawRect{{X: 8, Y: 8, W: 16, H: 16, Color: draw.Color{R: 1, A: 1}}}})
	r.EndFrame()
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	if err := r.ReadPixels(img); err != nil {
		t.Fatal(err)
	}
	t.Logf("inside=%v outside=%v", img.RGBAAt(15, 15), img.RGBAAt(40, 20))
	if c := img.RGBAAt(15, 15); c.R < 240 || c.G > 15 {
		t.Errorf("inside = %v, want red", c)
	}
	if c := img.RGBAAt(40, 20); c.G < 240 {
		t.Errorf("outside = %v, want white background", c)
	}
}

// halfSurface paints its left half opaque red and its right half 50 % blue.
type halfSurface struct{}

func (halfSurface) RasterizeSurface(dst *image.RGBA, rect, clip image.Rectangle) {
	mid := rect.Min.X + rect.Dx()/2
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			src := [4]uint8{255, 0, 0, 255} // premultiplied
			if x >= mid {
				src = [4]uint8{0, 0, 128, 128}
			}
			o := dst.PixOffset(x, y)
			inv := 255 - uint32(src[3])
			for c := 0; c < 4; c++ { // source-over
				dst.Pix[o+c] = uint8(uint32(src[c]) + (uint32(dst.Pix[o+c])*inv+127)/255)
			}
		}
	}
}

func TestWGPURasterSurfaceMatchesStilus(t *testing.T) {
	const w, h = 64, 48
	scene := draw.Scene{
		Rects: []draw.DrawRect{{X: 0, Y: 0, W: w, H: h, Color: draw.Color{G: 1, A: 1}}},
		Surfaces: []draw.DrawSurface{
			{X: 4, Y: 4, W: 20, H: 20, Raster: halfSurface{}},
			{X: 30, Y: 20, W: 20, H: 20, Raster: halfSurface{}, ClipX: 0, ClipY: 0, ClipW: 45, ClipH: h},
		},
	}
	g := newOffscreenWGPU(t, w, h)
	g.BeginFrame()
	g.Draw(scene)
	g.EndFrame()
	gi := image.NewRGBA(image.Rect(0, 0, w, h))
	if err := g.ReadPixels(gi); err != nil {
		t.Fatal(err)
	}
	c := NewStilus()
	_ = c.Init(Config{Width: w, Height: h})
	c.BeginFrame()
	c.Draw(scene)
	ci := c.Image()
	for _, p := range []image.Point{{6, 10}, {20, 10}, {32, 30}, {44, 30}, {48, 30}, {60, 5}} {
		a, b := gi.RGBAAt(p.X, p.Y), ci.RGBAAt(p.X, p.Y)
		if !near(a.R, b.R, 3) || !near(a.G, b.G, 3) || !near(a.B, b.B, 3) {
			t.Errorf("%v: gpu %v, cpu %v", p, a, b)
		}
	}
}
