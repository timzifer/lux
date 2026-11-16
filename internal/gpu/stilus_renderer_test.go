package gpu

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/text"
)

func newTestStilus(t *testing.T, w, h int) *StilusRenderer {
	t.Helper()
	r := NewStilus()
	if err := r.Init(Config{Width: w, Height: h}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	r.SetBackgroundColor(draw.Color{R: 1, G: 1, B: 1, A: 1})
	return r
}

func renderScene(r *StilusRenderer, s draw.Scene) *image.RGBA {
	r.BeginFrame()
	r.Draw(s)
	r.EndFrame()
	return r.Image()
}

func px(img *image.RGBA, x, y int) color.RGBA { return img.RGBAAt(x, y) }

func near(a, b uint8, tol int) bool {
	d := int(a) - int(b)
	return d >= -tol && d <= tol
}

func wantColor(t *testing.T, img *image.RGBA, x, y int, want color.RGBA, tol int) {
	t.Helper()
	got := px(img, x, y)
	if !near(got.R, want.R, tol) || !near(got.G, want.G, tol) || !near(got.B, want.B, tol) || !near(got.A, want.A, tol) {
		t.Errorf("pixel (%d,%d) = %v, want %v (±%d)", x, y, got, want, tol)
	}
}

var (
	white = color.RGBA{255, 255, 255, 255}
	red   = color.RGBA{255, 0, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
)

func TestStilusLifecycle(t *testing.T) {
	r := NewStilus()
	if err := r.Init(Config{}); err != nil {
		t.Fatal(err)
	}
	if b := r.Image().Bounds(); b.Dx() != 800 || b.Dy() != 600 {
		t.Fatalf("default size = %v, want 800x600", b)
	}
	r.Resize(320, 200)
	if b := r.Image().Bounds(); b.Dx() != 320 || b.Dy() != 200 {
		t.Fatalf("resized = %v", b)
	}
	r.Resize(0, 0) // ignored
	var presented *image.RGBA
	r.Present = func(img *image.RGBA) { presented = img }
	r.BeginFrame()
	r.Draw(draw.Scene{})
	r.EndFrame()
	if presented != r.Image() {
		t.Fatal("Present not called with framebuffer")
	}
	r.Destroy()
	r.BeginFrame() // must not panic after Destroy
	r.Draw(draw.Scene{})
}

func TestStilusClear(t *testing.T) {
	r := newTestStilus(t, 7, 5)
	r.SetBackgroundColor(draw.Color{R: 0, G: 0.5, B: 1, A: 1})
	img := renderScene(r, draw.Scene{})
	for y := 0; y < 5; y++ {
		for x := 0; x < 7; x++ {
			wantColor(t, img, x, y, color.RGBA{0, 128, 255, 255}, 0)
		}
	}
}

func TestStilusRect(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	img := renderScene(r, draw.Scene{Rects: []draw.DrawRect{
		{X: 10, Y: 10, W: 20, H: 20, Color: draw.Color{R: 1, A: 1}},
	}})
	wantColor(t, img, 10, 10, red, 0)
	wantColor(t, img, 29, 29, red, 0)
	wantColor(t, img, 9, 15, white, 0)
	wantColor(t, img, 30, 15, white, 0)
	wantColor(t, img, 15, 30, white, 0)
}

func TestStilusRectTranslucent(t *testing.T) {
	r := newTestStilus(t, 10, 10)
	img := renderScene(r, draw.Scene{Rects: []draw.DrawRect{
		{X: 0, Y: 0, W: 10, H: 10, Color: draw.Color{R: 0, G: 0, B: 0, A: 0.5}},
	}})
	wantColor(t, img, 5, 5, color.RGBA{128, 128, 128, 255}, 2)
}

func TestStilusRoundedRect(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	img := renderScene(r, draw.Scene{Rects: []draw.DrawRect{
		{X: 0, Y: 0, W: 40, H: 40, Radius: 10, Color: draw.Color{B: 1, A: 1}},
	}})
	wantColor(t, img, 0, 0, white, 0)  // corner cut away
	wantColor(t, img, 20, 20, blue, 0) // centre
	wantColor(t, img, 20, 0, blue, 0)  // edge midpoint
	// Antialiased arc: some pixel of the top row near the corner is partially covered.
	partial := false
	for x := 0; x < 10; x++ {
		c := px(img, x, 0)
		if c.B == 255 && c.R > 10 && c.R < 245 {
			partial = true
		}
	}
	if !partial {
		t.Error("rounded corner has no antialiased pixels")
	}
}

func TestStilusClipBatch(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	s := draw.Scene{
		Rects: []draw.DrawRect{
			{X: 0, Y: 0, W: 40, H: 40, Color: draw.Color{R: 1, A: 1}}, // batch 0: full viewport
			{X: 0, Y: 0, W: 40, H: 40, Color: draw.Color{B: 1, A: 1}}, // batch 1: clipped
		},
		ClipBatches: []draw.ClipBatch{
			{RectIdx: 0, FullViewport: true},
			{RectIdx: 1, Clip: draw.R(10, 10, 10, 10)},
		},
	}
	img := renderScene(r, s)
	wantColor(t, img, 5, 5, red, 0)
	wantColor(t, img, 15, 15, blue, 0)
	wantColor(t, img, 25, 25, red, 0)
}

func TestStilusClipBatchOverlayBase(t *testing.T) {
	// Overlay batches index into the overlay lists with their own base.
	r := newTestStilus(t, 20, 20)
	s := draw.Scene{
		OverlayRects: []draw.DrawRect{{X: 0, Y: 0, W: 20, H: 20, Color: draw.Color{R: 1, A: 1}}},
		OverlayClipBatches: []draw.ClipBatch{
			{RectIdx: 0, Clip: draw.R(0, 0, 10, 20)},
		},
	}
	img := renderScene(r, s)
	wantColor(t, img, 5, 5, red, 0)
	wantColor(t, img, 15, 5, white, 0)
}

func TestStilusDrawOrder(t *testing.T) {
	// Within a batch: shadows < rects < paths < gradients; overlay last.
	r := newTestStilus(t, 30, 10)
	s := draw.Scene{
		Rects: []draw.DrawRect{{X: 0, Y: 0, W: 30, H: 10, Color: draw.Color{R: 1, A: 1}}},
		GradientRects: []draw.DrawGradientRect{{
			X: 10, Y: 0, W: 20, H: 10, Kind: draw.PaintLinearGradient,
			StartX: 10, EndX: 30,
			Stops:     [8]draw.GradientStop{{Offset: 0, Color: draw.Color{G: 1, A: 1}}, {Offset: 1, Color: draw.Color{G: 1, A: 1}}},
			StopCount: 2,
		}},
		OverlayRects: []draw.DrawRect{{X: 20, Y: 0, W: 10, H: 10, Color: draw.Color{B: 1, A: 1}}},
	}
	img := renderScene(r, s)
	wantColor(t, img, 5, 5, red, 0)
	wantColor(t, img, 15, 5, color.RGBA{0, 255, 0, 255}, 1)
	wantColor(t, img, 25, 5, blue, 0)
}

func TestStilusLinearGradient(t *testing.T) {
	r := newTestStilus(t, 100, 10)
	img := renderScene(r, draw.Scene{GradientRects: []draw.DrawGradientRect{{
		X: 0, Y: 0, W: 100, H: 10, Kind: draw.PaintLinearGradient,
		StartX: 0, StartY: 0, EndX: 100, EndY: 0,
		Stops: [8]draw.GradientStop{
			{Offset: 0, Color: draw.Color{R: 1, A: 1}},
			{Offset: 1, Color: draw.Color{B: 1, A: 1}},
		},
		StopCount: 2,
	}}})
	left, mid, right := px(img, 0, 5), px(img, 50, 5), px(img, 99, 5)
	if left.R < 245 || left.B > 10 {
		t.Errorf("left = %v, want ≈ red", left)
	}
	if right.B < 245 || right.R > 10 {
		t.Errorf("right = %v, want ≈ blue", right)
	}
	if !near(mid.R, 128, 6) || !near(mid.B, 128, 6) {
		t.Errorf("mid = %v, want ≈ (128,0,128)", mid)
	}
}

func TestStilusRadialGradient(t *testing.T) {
	r := newTestStilus(t, 41, 41)
	img := renderScene(r, draw.Scene{GradientRects: []draw.DrawGradientRect{{
		X: 0, Y: 0, W: 41, H: 41, Kind: draw.PaintRadialGradient,
		CenterX: 20.5, CenterY: 20.5, GradRadius: 20,
		Stops: [8]draw.GradientStop{
			{Offset: 0, Color: draw.Color{R: 1, A: 1}},
			{Offset: 1, Color: draw.Color{B: 1, A: 1}},
		},
		StopCount: 2,
	}}})
	c, e := px(img, 20, 20), px(img, 0, 0)
	if c.R < 240 {
		t.Errorf("centre = %v, want ≈ red", c)
	}
	if e.B < 245 || e.R > 10 {
		t.Errorf("corner = %v, want blue (extended)", e)
	}
}

func TestStilusShadow(t *testing.T) {
	r := newTestStilus(t, 60, 60)
	// CPU side expands outer shadows by BlurRadius; the fade starts at the
	// original edge (20..40).
	img := renderScene(r, draw.Scene{ShadowRects: []draw.DrawShadowRect{{
		X: 10, Y: 10, W: 40, H: 40, BlurRadius: 10, Color: draw.Color{A: 1},
	}}})
	core := px(img, 30, 30)
	if core.R > 2 {
		t.Errorf("shadow core = %v, want black", core)
	}
	// Monotonic falloff outward from the original edge.
	prev := -1
	for x := 40; x < 60; x++ {
		v := int(px(img, x, 30).R)
		if v < prev {
			t.Fatalf("shadow not monotonic at x=%d: %d < %d", x, v, prev)
		}
		prev = v
	}
	if far := px(img, 59, 30); far.R < 200 {
		t.Errorf("far = %v, want mostly background", far)
	}
	if mid := px(img, 45, 30); mid.R < 20 || mid.R > 235 {
		t.Errorf("falloff pixel = %v, want partial", mid)
	}
}

func TestStilusInsetShadow(t *testing.T) {
	r := newTestStilus(t, 40, 40)
	img := renderScene(r, draw.Scene{ShadowRects: []draw.DrawShadowRect{{
		X: 0, Y: 0, W: 40, H: 40, BlurRadius: 8, Inset: true, Color: draw.Color{A: 1},
	}}})
	edge, centre := px(img, 0, 20), px(img, 20, 20)
	if edge.R >= centre.R {
		t.Errorf("inset shadow: edge %v should be darker than centre %v", edge, centre)
	}
	if centre.R < 250 {
		t.Errorf("inset centre = %v, want background", centre)
	}
}

func TestStilusPathBatch(t *testing.T) {
	r := newTestStilus(t, 20, 20)
	c := draw.Color{G: 1, A: 1}
	// A square from two triangles of opposite winding; shared diagonal must be seamless.
	verts := []draw.PathVertex{
		{X: 2, Y: 2}, {X: 18, Y: 2}, {X: 18, Y: 18},
		{X: 2, Y: 2}, {X: 2, Y: 18}, {X: 18, Y: 18},
	}
	img := renderScene(r, draw.Scene{
		PathVertices: verts,
		PathBatches:  []draw.DrawPathBatch{{VertexOffset: 0, VertexCount: 6, Color: c}},
	})
	for i := 3; i < 17; i++ {
		wantColor(t, img, i, i, color.RGBA{0, 255, 0, 255}, 0)
	}
	wantColor(t, img, 1, 10, white, 0)
}

func TestStilusPathBatchTranslucentOverlap(t *testing.T) {
	// Overlapping triangles of one batch blend once, not twice.
	r := newTestStilus(t, 20, 20)
	verts := []draw.PathVertex{
		{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 0, Y: 20},
		{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 0, Y: 20},
	}
	img := renderScene(r, draw.Scene{
		PathVertices: verts,
		PathBatches:  []draw.DrawPathBatch{{VertexCount: 6, Color: draw.Color{A: 0.5}}},
	})
	wantColor(t, img, 3, 3, color.RGBA{128, 128, 128, 255}, 2)
}

func TestStilusImage(t *testing.T) {
	r := newTestStilus(t, 20, 20)
	// 2×1 image: left red, right blue (straight alpha).
	r.UploadImage(7, 2, 1, []byte{255, 0, 0, 255, 0, 0, 255, 255})
	if w, h := r.ImageSize(7); w != 2 || h != 1 {
		t.Fatalf("ImageSize = %d,%d", w, h)
	}
	img := renderScene(r, draw.Scene{ImageRects: []draw.DrawImageRect{{
		X: 0, Y: 0, W: 20, H: 10, ImageID: 7, Opacity: 1, U1: 1, V1: 1, ScaleMode: draw.ImageScaleStretch,
	}}})
	wantColor(t, img, 1, 5, red, 3)
	wantColor(t, img, 18, 5, blue, 3)
	wantColor(t, img, 10, 15, white, 0)

	// Clip limits the image.
	img = renderScene(r, draw.Scene{ImageRects: []draw.DrawImageRect{{
		X: 0, Y: 0, W: 20, H: 10, ImageID: 7, Opacity: 1, U1: 1, V1: 1, ScaleMode: draw.ImageScaleStretch,
		ClipX: 0, ClipY: 0, ClipW: 5, ClipH: 10,
	}}})
	wantColor(t, img, 1, 5, red, 3)
	wantColor(t, img, 18, 5, white, 0)

	r.RemoveImage(7)
	img = renderScene(r, draw.Scene{ImageRects: []draw.DrawImageRect{{
		X: 0, Y: 0, W: 20, H: 10, ImageID: 7, Opacity: 1, U1: 1, V1: 1,
	}}})
	wantColor(t, img, 1, 5, white, 0)
}

func TestStilusImageFit(t *testing.T) {
	r := newTestStilus(t, 20, 10)
	r.UploadImage(1, 1, 1, []byte{0, 0, 255, 255})
	img := renderScene(r, draw.Scene{ImageRects: []draw.DrawImageRect{{
		X: 0, Y: 0, W: 20, H: 10, ImageID: 1, Opacity: 1, U1: 1, V1: 1, ScaleMode: draw.ImageScaleFit,
	}}})
	wantColor(t, img, 10, 5, blue, 0) // letterboxed square in the middle
	wantColor(t, img, 1, 5, white, 0)
	wantColor(t, img, 18, 5, white, 0)
}

func TestStilusTexturedGlyph(t *testing.T) {
	r := newTestStilus(t, 20, 20)
	atlas := text.NewGlyphAtlas(16, 16)
	// 4×4 opaque coverage block at (2,2) in the atlas.
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			atlas.Image.Pix[y*atlas.Image.Stride+x] = 255
		}
	}
	r.SetAtlas(atlas)
	img := renderScene(r, draw.Scene{TexturedGlyphs: []draw.TexturedGlyph{{
		DstX: 5, DstY: 5, DstW: 4, DstH: 4, SrcX: 2, SrcY: 2, SrcW: 4, SrcH: 4,
		Color: draw.Color{R: 1, A: 1},
	}}})
	wantColor(t, img, 5, 5, red, 0)
	wantColor(t, img, 8, 8, red, 0)
	wantColor(t, img, 9, 9, white, 0)
	wantColor(t, img, 4, 4, white, 0)
}

func TestStilusMSDFGlyph(t *testing.T) {
	r := newTestStilus(t, 20, 20)
	atlas := text.NewGlyphAtlas(16, 16)
	// Distance field: inside (> 0.5) on the left half, outside on the right.
	m := atlas.MSDFImage
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := uint8(255)
			if x >= 4 {
				v = 0
			}
			o := y*m.Stride + x*4
			m.Pix[o], m.Pix[o+1], m.Pix[o+2], m.Pix[o+3] = v, v, v, 255
		}
	}
	r.SetAtlas(atlas)
	img := renderScene(r, draw.Scene{MSDFGlyphs: []draw.TexturedGlyph{{
		DstX: 0, DstY: 0, DstW: 16, DstH: 16, SrcX: 0, SrcY: 0, SrcW: 8, SrcH: 8,
		Color: draw.Color{B: 1, A: 1},
	}}})
	wantColor(t, img, 2, 8, blue, 0)
	wantColor(t, img, 13, 8, white, 0)
}

func TestStilusEmojiGlyph(t *testing.T) {
	r := newTestStilus(t, 10, 10)
	atlas := text.NewGlyphAtlas(16, 16)
	c := atlas.ColorImage
	for i := 0; i < 4*4; i++ {
		x, y := i%4, i/4
		o := y*c.Stride + x*4
		c.Pix[o], c.Pix[o+1], c.Pix[o+2], c.Pix[o+3] = 0, 255, 0, 255
	}
	r.SetAtlas(atlas)
	img := renderScene(r, draw.Scene{EmojiGlyphs: []draw.TexturedGlyph{{
		DstX: 2, DstY: 2, DstW: 4, DstH: 4, SrcW: 4, SrcH: 4, Color: draw.Color{A: 1},
	}}})
	wantColor(t, img, 3, 3, color.RGBA{0, 255, 0, 255}, 0)
	wantColor(t, img, 7, 7, white, 0)
}

func TestStilusBitmapGlyph(t *testing.T) {
	r := newTestStilus(t, 40, 20)
	img := renderScene(r, draw.Scene{Glyphs: []draw.DrawGlyph{{
		X: 2, Y: 2, Scale: 2, Text: "H", Color: draw.Color{A: 1},
	}}})
	dark := 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			if px(img, x, y).R < 128 {
				dark++
			}
		}
	}
	if dark == 0 {
		t.Fatal("bitmap glyph drew nothing")
	}
}

func TestStilusBlur(t *testing.T) {
	r := newTestStilus(t, 40, 10)
	img := renderScene(r, draw.Scene{
		Rects:       []draw.DrawRect{{X: 0, Y: 0, W: 20, H: 10, Color: draw.Color{A: 1}}},
		BlurRegions: []draw.BlurRegion{{X: 0, Y: 0, W: 40, H: 10, Radius: 6}},
	})
	// The hard edge at x = 20 becomes a ramp.
	a, b, c := px(img, 17, 5).R, px(img, 20, 5).R, px(img, 23, 5).R
	if !(a < b && b < c) {
		t.Errorf("blur ramp not increasing: %d %d %d", a, b, c)
	}
	if b < 60 || b > 200 {
		t.Errorf("edge pixel = %d, want mid grey", b)
	}
	if px(img, 2, 5).R > 5 || px(img, 38, 5).R < 250 {
		t.Error("blur changed pixels far from the edge")
	}
}

func TestStilusGrain(t *testing.T) {
	r := newTestStilus(t, 32, 32)
	img := renderScene(r, draw.Scene{
		Grain: 0.2,
		Rects: []draw.DrawRect{{X: 0, Y: 0, W: 32, H: 32, Color: draw.Color{R: 0.5, G: 0.5, B: 0.5, A: 1}}},
	})
	seen := map[uint8]bool{}
	for x := 0; x < 32; x++ {
		seen[px(img, x, 3).R] = true
	}
	if len(seen) < 4 {
		t.Errorf("grain produced only %d distinct values", len(seen))
	}
}

func TestStilusUnsupportedIgnored(t *testing.T) {
	r := newTestStilus(t, 10, 10)
	img := renderScene(r, draw.Scene{
		Surfaces:    []draw.DrawSurface{{X: 0, Y: 0, W: 10, H: 10, TextureID: 1}},
		ShaderRects: []draw.DrawShaderRect{{X: 0, Y: 0, W: 10, H: 10, ShaderKey: "x"}},
		// Out-of-range indices and degenerate primitives must not panic.
		PathBatches:   []draw.DrawPathBatch{{VertexOffset: 5, VertexCount: 9, Color: draw.Color{A: 1}}},
		Rects:         []draw.DrawRect{{X: 0, Y: 0, W: -1, H: 5, Color: draw.Color{A: 1}}},
		GradientRects: []draw.DrawGradientRect{{X: 0, Y: 0, W: 10, H: 10, StopCount: 0}},
		ClipBatches:   []draw.ClipBatch{{RectIdx: 3}},
	})
	wantColor(t, img, 5, 5, white, 0)
}

func BenchmarkStilusScene(b *testing.B) {
	r := NewStilus()
	_ = r.Init(Config{Width: 800, Height: 600})
	var s draw.Scene
	for i := 0; i < 200; i++ {
		x, y := (i*37)%760, (i*53)%560
		s.Rects = append(s.Rects, draw.DrawRect{X: x, Y: y, W: 40, H: 30, Radius: 6, Color: draw.Color{R: 0.2, G: 0.4, B: 0.8, A: 0.9}})
	}
	for i := 0; i < 20; i++ {
		s.ShadowRects = append(s.ShadowRects, draw.DrawShadowRect{X: i * 30, Y: 100, W: 60, H: 40, Radius: 8, BlurRadius: 8, Color: draw.Color{A: 0.3}})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.BeginFrame()
		r.Draw(s)
		r.EndFrame()
	}
}

// complexScene uses every primitive the CPU renderer draws.
func complexScene(t *testing.T, r *StilusRenderer) draw.Scene {
	t.Helper()
	atlas := text.NewGlyphAtlas(64, 64)
	for i := range atlas.Image.Pix {
		atlas.Image.Pix[i] = uint8(i * 7)
	}
	r.SetAtlas(atlas)
	r.UploadImage(3, 2, 2, []byte{255, 0, 0, 255, 0, 255, 0, 128, 0, 0, 255, 255, 255, 255, 0, 64})
	var s draw.Scene
	s.Grain = 0.05
	for i := 0; i < 60; i++ {
		x, y := (i*37)%280, (i*53)%280
		s.Rects = append(s.Rects, draw.DrawRect{X: x, Y: y, W: 40, H: 33, Radius: float32(i % 12), Color: draw.Color{R: 0.3, G: float32(i%5) / 5, B: 0.7, A: 0.7}})
		s.ShadowRects = append(s.ShadowRects, draw.DrawShadowRect{X: x, Y: y, W: 30, H: 30, Radius: 6, BlurRadius: 9, Inset: i%4 == 0, Color: draw.Color{A: 0.4}})
	}
	s.GradientRects = append(s.GradientRects, draw.DrawGradientRect{X: 10, Y: 100, W: 200, H: 70, Radius: 9, Kind: draw.PaintRadialGradient,
		CenterX: 110, CenterY: 135, GradRadius: 90, StopCount: 2,
		Stops: [8]draw.GradientStop{{Offset: 0, Color: draw.Color{R: 1, A: 1}}, {Offset: 1, Color: draw.Color{B: 1, A: 0.5}}}})
	s.PathVertices = []draw.PathVertex{{X: 5, Y: 5}, {X: 290, Y: 40}, {X: 40, Y: 290}}
	s.PathBatches = []draw.DrawPathBatch{{VertexCount: 3, Color: draw.Color{G: 0.5, A: 0.5}}}
	for i := 0; i < 40; i++ {
		s.TexturedGlyphs = append(s.TexturedGlyphs, draw.TexturedGlyph{DstX: float32(i*7) + 0.3, DstY: float32(i*7) + 0.6, DstW: 8, DstH: 12, SrcX: i % 50, SrcY: 3, SrcW: 8, SrcH: 12, Color: draw.Color{R: 1, G: 1, A: 1}})
	}
	s.ShaderRects = []draw.DrawShaderRect{{X: 200, Y: 200, W: 90, H: 90, Radius: 10, ShaderKey: "_builtin:voronoi", Time: 0.5}}
	s.ImageRects = []draw.DrawImageRect{{X: 150, Y: 20, W: 120, H: 90, ImageID: 3, Opacity: 0.8, U1: 1, V1: 1}}
	s.Surfaces = []draw.DrawSurface{{X: 20, Y: 230, W: 60, H: 60, Raster: &checkerSurface{}}}
	s.ClipBatches = []draw.ClipBatch{{FullViewport: true}, {RectIdx: 30, ShadowIdx: 30, Clip: draw.R(30, 30, 200, 210)}}
	s.BlurRegions = []draw.BlurRegion{{X: 100, Y: 60, W: 150, H: 120, Radius: 10}}
	s.OverlayRects = []draw.DrawRect{{X: 60, Y: 60, W: 80, H: 50, Radius: 8, Color: draw.Color{R: 1, G: 1, B: 1, A: 0.9}}}
	s.OverlayShadowRects = []draw.DrawShadowRect{{X: 50, Y: 50, W: 100, H: 70, Radius: 8, BlurRadius: 10, Color: draw.Color{A: 0.5}}}
	return s
}

func TestStilusBandsMatchSingleWorker(t *testing.T) {
	render := func(workers int) []byte {
		r := NewStilus()
		r.Workers = workers
		if err := r.Init(Config{Width: 300, Height: 300}); err != nil {
			t.Fatal(err)
		}
		r.SetBackgroundColor(draw.Color{R: 0.1, G: 0.1, B: 0.12, A: 1})
		s := complexScene(t, r)
		r.BeginFrame()
		r.Draw(s)
		return append([]byte(nil), r.Image().Pix...)
	}
	one := render(1)
	for _, n := range []int{2, 3, 7, 9} {
		if got := render(n); !bytes.Equal(one, got) {
			diff := 0
			for i := range one {
				if one[i] != got[i] {
					diff++
				}
			}
			t.Errorf("%d workers: %d bytes differ from 1 worker", n, diff)
		}
	}
}
