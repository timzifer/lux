package gpu

import (
	"image"
	"image/color"
	"testing"

	"github.com/timzifer/lux/draw"
)

// checkerSurface paints red where (x+y) is even and blue elsewhere, in
// device coordinates, and records what it was asked for.
type checkerSurface struct {
	calls      int
	rect, clip image.Rectangle
}

func (s *checkerSurface) RasterizeSurface(dst *image.RGBA, rect, clip image.Rectangle) {
	s.calls++
	s.rect, s.clip = rect, clip
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if (x+y)%2 != 0 {
				c = color.RGBA{0, 0, 255, 255}
			}
			dst.SetRGBA(x, y, c)
		}
	}
}

func TestStilusRasterSurface(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	src := &checkerSurface{}
	img := renderScene(r, draw.Scene{Surfaces: []draw.DrawSurface{{
		X: 10, Y: 10, W: 20, H: 20, Raster: src,
		ClipX: 0, ClipY: 0, ClipW: 25, ClipH: 40,
	}}})
	if src.calls != 1 {
		t.Fatalf("RasterizeSurface called %d times", src.calls)
	}
	if want := image.Rect(10, 10, 25, 30); src.clip != want {
		t.Errorf("clip = %v, want %v", src.clip, want)
	}
	wantColor(t, img, 10, 10, red, 0)
	wantColor(t, img, 11, 10, blue, 0)
	wantColor(t, img, 26, 15, white, 0) // clipped
	wantColor(t, img, 5, 5, white, 0)
}

func TestStilusSurfaceOrder(t *testing.T) {
	// Surfaces sit above main content and below images and overlays.
	r := newTestStilus(t, 20, 20)
	img := renderScene(r, draw.Scene{
		Rects:        []draw.DrawRect{{X: 0, Y: 0, W: 20, H: 20, Color: draw.Color{G: 1, A: 1}}},
		Surfaces:     []draw.DrawSurface{{X: 0, Y: 0, W: 20, H: 20, Raster: &checkerSurface{}}},
		OverlayRects: []draw.DrawRect{{X: 0, Y: 0, W: 5, H: 5, Color: draw.Color{A: 1}}},
	})
	wantColor(t, img, 10, 10, red, 0)
	wantColor(t, img, 2, 2, color.RGBA{0, 0, 0, 255}, 0)
}

func TestStilusTextureSurfaceSkipped(t *testing.T) {
	r := newTestStilus(t, 10, 10)
	img := renderScene(r, draw.Scene{Surfaces: []draw.DrawSurface{{X: 0, Y: 0, W: 10, H: 10, TextureID: 5}}})
	wantColor(t, img, 5, 5, white, 0)
}

func TestStilusBuiltinEffects(t *testing.T) {
	for _, key := range []string{"_builtin:noise", "_builtin:plasma", "_builtin:voronoi"} {
		t.Run(key, func(t *testing.T) {
			r := newTestStilus(t, 64, 64)
			img := renderScene(r, draw.Scene{ShaderRects: []draw.DrawShaderRect{{
				X: 0, Y: 0, W: 64, H: 64, ShaderKey: key,
				Params: [8]float32{0, 1, 1, 1},
			}}})
			seen := map[color.RGBA]bool{}
			for y := 0; y < 64; y += 4 {
				for x := 0; x < 64; x += 4 {
					c := img.RGBAAt(x, y)
					if c.A != 255 {
						t.Fatalf("(%d,%d) not opaque: %v", x, y, c)
					}
					seen[c] = true
				}
			}
			if len(seen) < 8 {
				t.Errorf("effect produced only %d distinct colors", len(seen))
			}
		})
	}
}

func TestStilusEffectAnimates(t *testing.T) {
	r := newTestStilus(t, 32, 32)
	frame := func(time float32) []byte {
		img := renderScene(r, draw.Scene{ShaderRects: []draw.DrawShaderRect{{
			X: 0, Y: 0, W: 32, H: 32, ShaderKey: "_builtin:plasma", Time: time,
		}}})
		return append([]byte(nil), img.Pix...)
	}
	a, b := frame(0), frame(1)
	if string(a) == string(b) {
		t.Error("plasma does not change with time")
	}
}

func TestStilusCustomShaderCPU(t *testing.T) {
	r := newTestStilus(t, 20, 20)
	// Horizontal ramp from the CPU function; uv covers the rect.
	fn := func(u, v, tm float32, p *[8]float32) draw.Color { return draw.Color{R: u, G: p[0], B: 0, A: 1} }
	img := renderScene(r, draw.Scene{ShaderRects: []draw.DrawShaderRect{
		{X: 0, Y: 0, W: 20, H: 10, ShaderKey: "custom-hash", CPU: fn, Params: [8]float32{1}},
		{X: 0, Y: 10, W: 20, H: 10, ShaderKey: "custom-without-cpu"},
	}})
	left, right := img.RGBAAt(0, 5), img.RGBAAt(19, 5)
	if left.R > 10 || right.R < 245 || left.G != 255 {
		t.Errorf("ramp: left %v right %v", left, right)
	}
	wantColor(t, img, 10, 15, white, 0) // no CPU function: skipped
}

func TestStilusShaderRectRoundedAndClipped(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	solid := func(u, v, tm float32, p *[8]float32) draw.Color { return draw.Color{B: 1, A: 1} }
	img := renderScene(r, draw.Scene{
		ShaderRects: []draw.DrawShaderRect{{X: 0, Y: 0, W: 40, H: 40, Radius: 12, CPU: solid}},
		ClipBatches: []draw.ClipBatch{{ShaderIdx: 0, Clip: draw.R(0, 0, 30, 40)}},
	})
	wantColor(t, img, 0, 0, white, 0)   // rounded corner
	wantColor(t, img, 20, 20, blue, 0)  // inside
	wantColor(t, img, 35, 20, white, 0) // clipped
}
