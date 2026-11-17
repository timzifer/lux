package gpu

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/stilus"
)

// StilusRenderer is a pure-Go CPU renderer backed by github.com/timzifer/stilus.
//
// It rasterizes a draw.Scene into a caller-visible *image.RGBA (premultiplied)
// without any GPU, cgo or window system, so it runs on every GOOS/GOARCH,
// including CI machines and js/wasm. It is the "Software (CI, Headless)"
// backend of RFC-001 §6.1.
//
// The draw order matches WGPURenderer: per clip batch shadows, rects, paths,
// bitmap text, MSDF text, emoji, gradients; then images, legacy bitmap glyphs,
// blur regions and finally the overlay layer with the same order.
//
// External surfaces are drawn when their provider rasterizes on the CPU
// (draw.SurfaceRasterizer), directly into the framebuffer; GPU-texture
// surfaces are skipped. Shader rects run the built-in effects in Go, and
// custom WGSL through its ShaderDesc.CPU function (skipped without one).
//
// The frame is drawn in horizontal bands by Workers goroutines, each with its
// own stilus.Canvas (stilus draws a band byte for byte as it draws the whole
// frame). Surfaces are drawn between the band phases, one at a time: their
// providers parallelize on their own.
//
// Presentation: set Present to receive the finished frame in EndFrame
// (e.g. to encode it as PNG). Left nil, Init presents into the native window
// where the platform supports it (Windows, via GDI).
type StilusRenderer struct {
	// Present, when non-nil, is called from EndFrame with the finished frame.
	// The image is reused for the next frame; copy it to retain it.
	Present func(img *image.RGBA)

	// Workers is the number of goroutines drawing bands of the frame;
	// 0 means GOMAXPROCS, 1 draws on the calling goroutine.
	Workers int

	width, height int
	img           *image.RGBA
	bg            draw.Color
	atlas         *text.GlyphAtlas
	images        map[draw.ImageID]*stilusImage
	workers       []*stilusWorker
	blurTmp       []float32
}

// minBandHeight keeps bands tall enough that per-band setup stays small.
const minBandHeight = 32

// stilusWorker draws one band of the frame. All scratch state is per worker,
// so bands draw concurrently without locks; after warm-up nothing allocates.
type stilusWorker struct {
	canvas *stilus.Canvas
	band   image.Rectangle

	// Shared, read-only during Draw.
	atlas  *text.GlyphAtlas
	images map[draw.ImageID]*stilusImage
	grain  float32

	path   stilus.Path
	paint  stilus.Paint
	shadow shadowShader
	fx     effectShader
	glyph  glyphShader
	solid  grainShader
	lin    stilus.LinearGradient
	rad    stilus.RadialGradient
	imgShd stilus.ImageShader
	gradSh gradientShader
	ramp   stilus.Ramp
	knots  []float32
}

type stilusImage struct {
	w, h int
	tex  *stilus.Texture
}

// NewStilus creates a CPU renderer backed by stilus.
func NewStilus() *StilusRenderer {
	return &StilusRenderer{images: make(map[draw.ImageID]*stilusImage)}
}

// Init allocates the framebuffer.
func (r *StilusRenderer) Init(cfg Config) error {
	w, h := cfg.Width, cfg.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}
	r.resize(w, h)
	if r.Present == nil {
		r.Present = windowPresenter(cfg)
	}
	return nil
}

func (r *StilusRenderer) resize(w, h int) {
	r.width, r.height = w, h
	r.img = image.NewRGBA(image.Rect(0, 0, w, h))
	r.layoutBands()
}

// layoutBands splits the frame into one band per worker.
func (r *StilusRenderer) layoutBands() {
	n := r.Workers
	if n <= 0 {
		n = runtime.GOMAXPROCS(0)
	}
	if m := r.height / minBandHeight; n > m {
		n = m
	}
	if n < 1 {
		n = 1
	}
	for len(r.workers) < n {
		r.workers = append(r.workers, &stilusWorker{canvas: stilus.NewCanvas(r.img)})
	}
	r.workers = r.workers[:n]
	bh := (r.height + n - 1) / n
	for i, w := range r.workers {
		y0 := i * bh
		y1 := min(y0+bh, r.height)
		w.band = image.Rect(0, y0, r.width, y1)
	}
}

// parallel runs fn for every band, on Workers goroutines.
func (r *StilusRenderer) parallel(fn func(w *stilusWorker)) {
	run := func(w *stilusWorker) {
		w.canvas.Reset(r.img, w.band)
		fn(w)
	}
	if len(r.workers) == 1 {
		run(r.workers[0])
		return
	}
	var wg sync.WaitGroup
	for _, w := range r.workers[1:] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(w)
		}()
	}
	run(r.workers[0])
	wg.Wait()
}

// Resize reallocates the framebuffer.
func (r *StilusRenderer) Resize(width, height int) {
	if width <= 0 || height <= 0 || (width == r.width && height == r.height && r.img != nil) {
		return
	}
	r.resize(width, height)
}

// Image returns the framebuffer (premultiplied RGBA). Valid until the next Resize.
func (r *StilusRenderer) Image() *image.RGBA { return r.img }

// SetBackgroundColor sets the clear color used by BeginFrame.
func (r *StilusRenderer) SetBackgroundColor(c draw.Color) { r.bg = c }

// SetAtlas sets the glyph atlas used for textured, MSDF and emoji glyphs.
func (r *StilusRenderer) SetAtlas(a *text.GlyphAtlas) { r.atlas = a }

// UploadImage stores straight-alpha RGBA8 pixels for DrawImageRect (implements ImageUploader).
func (r *StilusRenderer) UploadImage(id draw.ImageID, width, height int, rgba []byte) {
	if width <= 0 || height <= 0 || len(rgba) < width*height*4 {
		return
	}
	pix := make([]uint32, width*height)
	for i := range pix {
		o := i * 4
		a := uint32(rgba[o+3])
		c := color.RGBA{
			R: uint8((uint32(rgba[o])*a + 127) / 255),
			G: uint8((uint32(rgba[o+1])*a + 127) / 255),
			B: uint8((uint32(rgba[o+2])*a + 127) / 255),
			A: uint8(a),
		}
		pix[i] = stilus.PackRGBA(c)
	}
	plane := stilus.Plane{Kind: stilus.PlaneRGBA, W: width, H: height, Stride: width, Pix32: pix}
	r.images[id] = &stilusImage{w: width, h: height, tex: stilus.NewTexture(plane)}
}

// RemoveImage drops an uploaded image.
func (r *StilusRenderer) RemoveImage(id draw.ImageID) { delete(r.images, id) }

// ImageSize returns the size of an uploaded image, or 0,0.
func (r *StilusRenderer) ImageSize(id draw.ImageID) (w, h int) {
	if im, ok := r.images[id]; ok {
		return im.w, im.h
	}
	return 0, 0
}

// BeginFrame clears the framebuffer to the background color.
func (r *StilusRenderer) BeginFrame() {
	if r.img == nil {
		return
	}
	c := premul(r.bg, 1)
	pix := r.img.Pix
	if len(pix) == 0 {
		return
	}
	pix[0], pix[1], pix[2], pix[3] = c.R, c.G, c.B, c.A
	for n := 4; n < len(pix); n *= 2 {
		copy(pix[n:], pix[:n])
	}
}

// Draw rasterizes the scene into the framebuffer.
func (r *StilusRenderer) Draw(scene draw.Scene) {
	if r.img == nil {
		return
	}
	if want := r.Workers; want > 0 && want != len(r.workers) && r.height/minBandHeight >= want {
		r.layoutBands()
	}
	for _, w := range r.workers {
		w.atlas, w.images, w.grain = r.atlas, r.images, scene.Grain
	}

	main := layer{
		rects: scene.Rects, text: scene.TexturedGlyphs, msdf: scene.MSDFGlyphs,
		emoji: scene.EmojiGlyphs, grads: scene.GradientRects, shadows: scene.ShadowRects,
		images: scene.ImageRects, verts: scene.PathVertices, paths: scene.PathBatches,
		glyphs: scene.Glyphs, batches: scene.ClipBatches,
		shaders: scene.ShaderRects, surfaces: scene.Surfaces,
	}
	r.drawLayer(&main)

	for _, br := range scene.BlurRegions {
		r.blur(br)
	}

	overlay := layer{
		rects: scene.OverlayRects, text: scene.OverlayTexturedGlyphs, msdf: scene.OverlayMSDFGlyphs,
		emoji: scene.OverlayEmojiGlyphs, grads: scene.OverlayGradientRects, shadows: scene.OverlayShadowRects,
		images: scene.OverlayImageRects, verts: scene.OverlayPathVertices, paths: scene.OverlayPathBatches,
		glyphs: scene.OverlayGlyphs, batches: scene.OverlayClipBatches,
		shaders: scene.OverlayShaderRects,
	}
	r.drawLayer(&overlay)
}

// EndFrame hands the finished frame to Present, if set.
func (r *StilusRenderer) EndFrame() {
	if r.Present != nil && r.img != nil {
		r.Present(r.img)
	}
}

// Destroy releases the framebuffer and images.
func (r *StilusRenderer) Destroy() {
	r.img = nil
	r.images = make(map[draw.ImageID]*stilusImage)
}

// ── Layers and clip batches ─────────────────────────────────────

type layer struct {
	rects    []draw.DrawRect
	text     []draw.TexturedGlyph
	msdf     []draw.TexturedGlyph
	emoji    []draw.TexturedGlyph
	grads    []draw.DrawGradientRect
	shadows  []draw.DrawShadowRect
	images   []draw.DrawImageRect
	verts    []draw.PathVertex
	paths    []draw.DrawPathBatch
	glyphs   []draw.DrawGlyph
	shaders  []draw.DrawShaderRect
	surfaces []draw.DrawSurface
	batches  []draw.ClipBatch
}

func (r *StilusRenderer) drawLayer(l *layer) {
	r.parallel(func(w *stilusWorker) { w.drawBatches(l) })
	for i := range l.surfaces {
		r.drawSurface(&l.surfaces[i])
	}
	if len(l.images) > 0 || len(l.glyphs) > 0 {
		r.parallel(func(w *stilusWorker) {
			for i := range l.images {
				w.drawImage(&l.images[i])
			}
			for _, g := range l.glyphs {
				w.drawBitmapGlyph(g)
			}
		})
	}
}

// drawBatches draws the clip batches of a layer into the worker's band.
func (r *stilusWorker) drawBatches(l *layer) {
	if len(l.batches) == 0 {
		r.drawRange(l, 0, len(l.shadows), 0, len(l.rects), 0, len(l.paths),
			0, len(l.text), 0, len(l.msdf), 0, len(l.emoji), 0, len(l.grads), 0, len(l.shaders))
	} else {
		// Batch indices address the scene lists; the first batch gives
		// the base for this layer (see WGPURenderer.drawClipBatches).
		b0 := l.batches[0]
		for i, b := range l.batches {
			var eS, eR, eP, eT, eM, eE, eG, eX int
			if i+1 < len(l.batches) {
				n := l.batches[i+1]
				eS, eR, eP, eT, eM, eE, eG, eX = n.ShadowIdx, n.RectIdx, n.PathIdx, n.TextIdx, n.MSDFIdx, n.EmojiIdx, n.GradientIdx, n.ShaderIdx
			} else {
				eX = b0.ShaderIdx + len(l.shaders)
				eS = b0.ShadowIdx + len(l.shadows)
				eR = b0.RectIdx + len(l.rects)
				eP = b0.PathIdx + len(l.paths)
				eT = b0.TextIdx + len(l.text)
				eM = b0.MSDFIdx + len(l.msdf)
				eE = b0.EmojiIdx + len(l.emoji)
				eG = b0.GradientIdx + len(l.grads)
			}
			clipped := !b.FullViewport
			if clipped {
				if b.Clip.W <= 0 || b.Clip.H <= 0 {
					continue
				}
				r.canvas.ClipRect(stilus.Rect{
					X0: float64(b.Clip.X), Y0: float64(b.Clip.Y),
					X1: float64(b.Clip.X + b.Clip.W), Y1: float64(b.Clip.Y + b.Clip.H),
				}, stilus.Identity)
			}
			r.drawRange(l,
				b.ShadowIdx-b0.ShadowIdx, eS-b0.ShadowIdx,
				b.RectIdx-b0.RectIdx, eR-b0.RectIdx,
				b.PathIdx-b0.PathIdx, eP-b0.PathIdx,
				b.TextIdx-b0.TextIdx, eT-b0.TextIdx,
				b.MSDFIdx-b0.MSDFIdx, eM-b0.MSDFIdx,
				b.EmojiIdx-b0.EmojiIdx, eE-b0.EmojiIdx,
				b.GradientIdx-b0.GradientIdx, eG-b0.GradientIdx,
				b.ShaderIdx-b0.ShaderIdx, eX-b0.ShaderIdx)
			if clipped {
				r.canvas.PopClip()
			}
		}
	}
}

func clampRange(lo, hi, n int) (int, int) {
	if lo < 0 {
		lo = 0
	}
	if hi > n {
		hi = n
	}
	return lo, hi
}

func (r *stilusWorker) drawRange(l *layer, s0, s1, r0, r1, p0, p1, t0, t1, m0, m1, e0, e1, g0, g1, x0, x1 int) {
	s0, s1 = clampRange(s0, s1, len(l.shadows))
	for i := s0; i < s1; i++ {
		r.drawShadow(&l.shadows[i])
	}
	r0, r1 = clampRange(r0, r1, len(l.rects))
	for i := r0; i < r1; i++ {
		r.drawRect(&l.rects[i])
	}
	p0, p1 = clampRange(p0, p1, len(l.paths))
	for i := p0; i < p1; i++ {
		r.drawPathBatch(l.verts, &l.paths[i])
	}
	if r.atlas != nil {
		t0, t1 = clampRange(t0, t1, len(l.text))
		for i := t0; i < t1; i++ {
			r.drawGlyph(&l.text[i], glyphAlpha)
		}
		m0, m1 = clampRange(m0, m1, len(l.msdf))
		for i := m0; i < m1; i++ {
			r.drawGlyph(&l.msdf[i], glyphMSDF)
		}
		e0, e1 = clampRange(e0, e1, len(l.emoji))
		for i := e0; i < e1; i++ {
			r.drawGlyph(&l.emoji[i], glyphColor)
		}
	}
	g0, g1 = clampRange(g0, g1, len(l.grads))
	for i := g0; i < g1; i++ {
		r.drawGradient(&l.grads[i])
	}
	x0, x1 = clampRange(x0, x1, len(l.shaders))
	for i := x0; i < x1; i++ {
		r.drawShaderRect(&l.shaders[i])
	}
}

// ── Primitives ───────────────────────────────────────────────────

func premul(c draw.Color, alpha float32) color.RGBA {
	a := clamp01(c.A * alpha)
	return color.RGBA{
		R: to8(clamp01(c.R) * a),
		G: to8(clamp01(c.G) * a),
		B: to8(clamp01(c.B) * a),
		A: to8(a),
	}
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func to8(v float32) uint8 { return uint8(v*255 + 0.5) }

// roundRect appends a rounded rectangle to p (radius clamped to half the short side).
func roundRect(p *stilus.Path, x, y, w, h, rad float32) {
	if rad > w/2 {
		rad = w / 2
	}
	if rad > h/2 {
		rad = h / 2
	}
	if rad <= 0 {
		p.Rect(x, y, w, h)
		return
	}
	const k = 0.5522847498 // cubic Bézier circle constant
	c := rad * (1 - k)
	x1, y1 := x+w, y+h
	p.MoveTo(x+rad, y)
	p.LineTo(x1-rad, y)
	p.CubicTo(x1-c, y, x1, y+c, x1, y+rad)
	p.LineTo(x1, y1-rad)
	p.CubicTo(x1, y1-c, x1-c, y1, x1-rad, y1)
	p.LineTo(x+rad, y1)
	p.CubicTo(x+c, y1, x, y1-c, x, y1-rad)
	p.LineTo(x, y+rad)
	p.CubicTo(x, y+c, x+c, y, x+rad, y)
	p.Close()
}

func (r *stilusWorker) drawRect(d *draw.DrawRect) {
	if d.W <= 0 || d.H <= 0 || d.Color.A <= 0 {
		return
	}
	r.path.Reset()
	roundRect(&r.path, float32(d.X), float32(d.Y), float32(d.W), float32(d.H), d.Radius)
	r.paint = stilus.Paint{}
	if r.grain > 0 {
		r.solid = grainShader{c: d.Color, grain: r.grain}
		r.paint.Shader = &r.solid
	} else {
		r.paint.Color = premul(d.Color, 1)
	}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

func (r *stilusWorker) drawShadow(d *draw.DrawShadowRect) {
	if d.W <= 0 || d.H <= 0 || d.Color.A <= 0 {
		return
	}
	expand := float32(0.5)
	if !d.Inset {
		expand = d.BlurRadius + 0.5
	}
	x := float32(d.X) - expand
	y := float32(d.Y) - expand
	w := float32(d.W) + 2*expand
	h := float32(d.H) + 2*expand
	hw, hh := float32(d.W)/2, float32(d.H)/2
	if !d.Inset {
		hw -= d.BlurRadius
		hh -= d.BlurRadius
	}
	r.shadow = shadowShader{
		cx: float32(d.X) + float32(d.W)/2, cy: float32(d.Y) + float32(d.H)/2,
		hw: hw, hh: hh, radius: d.Radius, blur: d.BlurRadius, inset: d.Inset, c: d.Color,
		lut: r.shadow.lut,
	}
	r.shadow.setup()
	r.path.Reset()
	r.path.Rect(x, y, w, h)
	r.paint = stilus.Paint{Shader: &r.shadow}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

func (r *stilusWorker) drawPathBatch(verts []draw.PathVertex, b *draw.DrawPathBatch) {
	lo, hi := b.VertexOffset, b.VertexOffset+b.VertexCount
	if lo < 0 || hi > len(verts) || b.VertexCount < 3 || b.Color.A <= 0 {
		return
	}
	// Tessellated triangles share edges; filling them as one NonZero path
	// (consistently wound) covers shared edges seamlessly and blends
	// overlapping triangles once, unlike per-triangle compositing.
	r.path.Reset()
	for i := lo; i+2 < hi; i += 3 {
		a, b2, c := verts[i], verts[i+1], verts[i+2]
		cross := (b2.X-a.X)*(c.Y-a.Y) - (b2.Y-a.Y)*(c.X-a.X)
		if cross == 0 {
			continue
		}
		if cross < 0 {
			b2, c = c, b2
		}
		r.path.MoveTo(a.X, a.Y)
		r.path.LineTo(b2.X, b2.Y)
		r.path.LineTo(c.X, c.Y)
		r.path.Close()
	}
	if r.path.Empty() {
		return
	}
	r.paint = stilus.Paint{Color: premul(b.Color, 1)}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

func (r *stilusWorker) drawGradient(d *draw.DrawGradientRect) {
	if d.W <= 0 || d.H <= 0 || d.StopCount <= 0 {
		return
	}
	n := d.StopCount
	if n > len(d.Stops) {
		n = len(d.Stops)
	}
	r.ramp = r.ramp[:0]
	r.knots = r.knots[:0]
	prev := float32(0)
	for i := 0; i < n; i++ {
		s := d.Stops[i]
		off := clamp01(s.Offset)
		if off < prev {
			off = prev
		}
		prev = off
		r.ramp = append(r.ramp, stilus.PackRGBA(premul(s.Color, 1)))
		r.knots = append(r.knots, off)
	}
	if n == 1 {
		// A single stop paints its color everywhere.
		r.ramp = append(r.ramp, r.ramp[0])
		r.knots = append(r.knots[:0], 0, 1)
	}

	var sh stilus.Shader
	ok := false
	switch d.Kind {
	case draw.PaintRadialGradient:
		g := &r.rad
		g.Gradient = stilus.Gradient{Ramp: r.ramp, Knots: r.knots, Extend: [2]bool{true, true}, Alpha: 255}
		rad := d.GradRadius
		if rad < 1e-4 {
			rad = 1e-4
		}
		ok = g.Set(float64(d.CenterX), float64(d.CenterY), 0, float64(d.CenterX), float64(d.CenterY), float64(rad), stilus.Identity)
		sh = g
	default:
		g := &r.lin
		g.Gradient = stilus.Gradient{Ramp: r.ramp, Knots: r.knots, Extend: [2]bool{true, true}, Alpha: 255}
		ok = g.Set(float64(d.StartX), float64(d.StartY), float64(d.EndX), float64(d.EndY), stilus.Identity)
		sh = g
	}
	if !ok {
		// Degenerate axis: the GPU shader samples t = 0 (the first stop).
		r.solid = grainShader{c: d.Stops[0].Color, grain: r.grain}
		sh = &r.solid
	} else if r.grain > 0 {
		r.gradSh = gradientShader{inner: sh, grain: r.grain}
		sh = &r.gradSh
	}
	r.path.Reset()
	roundRect(&r.path, float32(d.X), float32(d.Y), float32(d.W), float32(d.H), d.Radius)
	r.paint = stilus.Paint{Shader: sh}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

func (r *stilusWorker) drawImage(d *draw.DrawImageRect) {
	im, ok := r.images[d.ImageID]
	if !ok || d.ImageID == 0 || d.W <= 0 || d.H <= 0 {
		return
	}
	x, y, w, h := d.X, d.Y, d.W, d.H
	u0, v0, u1, v1 := d.U0, d.V0, d.U1, d.V1
	if u0 == 0 && v0 == 0 && u1 == 0 && v1 == 0 {
		u1, v1 = 1, 1
	}
	srcAspect := float64(im.w) / float64(im.h)
	dstAspect := float64(w) / float64(h)
	switch d.ScaleMode {
	case draw.ImageScaleFit:
		fw, fh := w, h
		if srcAspect > dstAspect {
			fh = int(float64(w) / srcAspect)
		} else {
			fw = int(float64(h) * srcAspect)
		}
		x += (w - fw) / 2
		y += (h - fh) / 2
		w, h = fw, fh
	case draw.ImageScaleFill:
		if srcAspect > dstAspect {
			m := (1 - float32(dstAspect/srcAspect)) / 2
			u0, u1 = m, 1-m
		} else {
			m := (1 - float32(srcAspect/dstAspect)) / 2
			v0, v1 = m, 1-m
		}
	}
	if w <= 0 || h <= 0 || u1 == u0 || v1 == v0 {
		return
	}
	alpha := d.Opacity
	if alpha <= 0 {
		return
	}
	// Texture pixel space → device: texel (tu, tv) maps to
	// x + (tu/imW - u0)/(u1-u0)*w.
	sx := float64(w) / (float64(u1-u0) * float64(im.w))
	sy := float64(h) / (float64(v1-v0) * float64(im.h))
	m := stilus.Matrix{sx, 0, 0, sy, float64(x) - float64(u0)*float64(im.w)*sx, float64(y) - float64(v0)*float64(im.h)*sy}
	r.imgShd.Reset()
	if !r.imgShd.SetImage(im.tex, m, true, to8(clamp01(alpha))) {
		return
	}
	clipped := d.ClipW > 0 && d.ClipH > 0
	if clipped {
		r.canvas.ClipRect(stilus.Rect{
			X0: float64(d.ClipX), Y0: float64(d.ClipY),
			X1: float64(d.ClipX + d.ClipW), Y1: float64(d.ClipY + d.ClipH),
		}, stilus.Identity)
	}
	r.path.Reset()
	r.path.Rect(float32(x), float32(y), float32(w), float32(h))
	r.paint = stilus.Paint{Shader: &r.imgShd}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
	if clipped {
		r.canvas.PopClip()
	}
}

func (r *stilusWorker) drawBitmapGlyph(g draw.DrawGlyph) {
	if g.Scale <= 0 {
		g.Scale = 1
	}
	// All cells wind the same way, so one NonZero fill covers abutting
	// cells without seams.
	r.path.Reset()
	render.RenderBitmapGlyph(g.Text, g.X, g.Y, g.Scale, func(px, py, w, h int) {
		r.path.Rect(float32(px), float32(py), float32(w), float32(h))
	})
	if r.path.Empty() {
		return
	}
	r.paint = stilus.Paint{Color: premul(g.Color, 1)}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

type glyphKind uint8

const (
	glyphAlpha glyphKind = iota // gray atlas coverage, tinted
	glyphMSDF                   // multi-channel SDF atlas, tinted
	glyphColor                  // color emoji atlas
)

func (r *stilusWorker) drawGlyph(g *draw.TexturedGlyph, kind glyphKind) {
	if g.DstW <= 0 || g.DstH <= 0 || g.SrcW <= 0 || g.SrcH <= 0 || g.Color.A <= 0 {
		return
	}
	sh := &r.glyph
	*sh = glyphShader{kind: kind, g: g, c: g.Color, pc: toPM(g.Color)}
	switch kind {
	case glyphAlpha:
		if r.atlas.Image == nil {
			return
		}
		sh.gray = r.atlas.Image
	case glyphMSDF:
		if r.atlas.MSDFImage == nil {
			return
		}
		sh.nrgba = r.atlas.MSDFImage
		sh.pxRange = float32(math.Max(0.5*float64(text.MSDFPxRange)*(float64(g.DstW)/float64(g.SrcW)+float64(g.DstH)/float64(g.SrcH)), 1))
	case glyphColor:
		if r.atlas.ColorImage == nil {
			return
		}
		sh.nrgba = r.atlas.ColorImage
	}
	sh.scaleX = float32(g.SrcW) / g.DstW
	sh.scaleY = float32(g.SrcH) / g.DstH
	r.path.Reset()
	r.path.Rect(g.DstX, g.DstY, g.DstW, g.DstH)
	r.paint = stilus.Paint{Shader: sh}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

// ── Blur ─────────────────────────────────────────────────────────

// blur applies the separable Gaussian of the GPU blur pass (sigma = r/3,
// taps clamped to the framebuffer edge) to the region, rows split over the
// workers.
func (r *StilusRenderer) blur(br draw.BlurRegion) {
	if br.W <= 0 || br.H <= 0 || br.Radius <= 0 {
		return
	}
	rad := int(br.Radius)
	if rad > 64 {
		rad = 64
	}
	if rad < 1 {
		return
	}
	reg := image.Rect(br.X, br.Y, br.X+br.W, br.Y+br.H).Intersect(r.img.Rect)
	if reg.Empty() {
		return
	}
	sigma := math.Max(float64(rad)/3, 1)
	var wbuf [129]float32
	weights := wbuf[:2*rad+1]
	var sum float32
	for i := -rad; i <= rad; i++ {
		w := float32(math.Exp(-float64(i*i) / (2 * sigma * sigma)))
		weights[i+rad] = w
		sum += w
	}
	for i := range weights {
		weights[i] /= sum
	}

	W := r.width
	// The horizontal pass covers rad rows beyond the region, so the
	// vertical pass sees blurred neighbours as on the GPU.
	y0, y1 := max(reg.Min.Y-rad, 0), min(reg.Max.Y+rad, r.height)
	x0, x1 := reg.Min.X, reg.Max.X
	cw := x1 - x0
	need := cw * (y1 - y0) * 4
	if cap(r.blurTmp) < need {
		r.blurTmp = make([]float32, need)
	}
	tmp := r.blurTmp[:need]
	pix, stride := r.img.Pix, r.img.Stride

	r.parallelRows(y0, y1, func(ya, yb int) {
		for y := ya; y < yb; y++ {
			row := pix[y*stride : y*stride+W*4]
			t := (y - y0) * cw * 4
			for x := x0; x < x1; x++ {
				var a0, a1, a2, a3 float32
				if x-rad >= 0 && x+rad < W {
					taps := row[(x-rad)*4 : (x+rad+1)*4]
					for k, w := range weights {
						o := k * 4
						a0 += w * float32(taps[o])
						a1 += w * float32(taps[o+1])
						a2 += w * float32(taps[o+2])
						a3 += w * float32(taps[o+3])
					}
				} else {
					for k, w := range weights {
						sx := min(max(x+k-rad, 0), W-1) * 4
						a0 += w * float32(row[sx])
						a1 += w * float32(row[sx+1])
						a2 += w * float32(row[sx+2])
						a3 += w * float32(row[sx+3])
					}
				}
				tmp[t], tmp[t+1], tmp[t+2], tmp[t+3] = a0, a1, a2, a3
				t += 4
			}
		}
	})
	r.parallelRows(reg.Min.Y, reg.Max.Y, func(ya, yb int) {
		rowLen := cw * 4
		for y := ya; y < yb; y++ {
			o := y*stride + x0*4
			for t := 0; t < rowLen; t += 4 {
				var a0, a1, a2, a3 float32
				for k, w := range weights {
					sy := min(max(y+k-rad, y0), y1-1)
					q := (sy-y0)*rowLen + t
					a0 += w * tmp[q]
					a1 += w * tmp[q+1]
					a2 += w * tmp[q+2]
					a3 += w * tmp[q+3]
				}
				pix[o+t], pix[o+t+1], pix[o+t+2], pix[o+t+3] = u8(a0), u8(a1), u8(a2), u8(a3)
			}
		}
	})
}

// parallelRows splits [y0, y1) into one range per worker and waits.
func (r *StilusRenderer) parallelRows(y0, y1 int, fn func(ya, yb int)) {
	n := len(r.workers)
	if n <= 1 || y1-y0 < 2*minBandHeight {
		fn(y0, y1)
		return
	}
	step := (y1 - y0 + n - 1) / n
	var wg sync.WaitGroup
	for ya := y0 + step; ya < y1; ya += step {
		wg.Add(1)
		go func(ya int) {
			defer wg.Done()
			fn(ya, min(ya+step, y1))
		}(ya)
	}
	fn(y0, min(y0+step, y1))
	wg.Wait()
}

func u8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// ── Shaders ──────────────────────────────────────────────────────

func packStraight(r, g, b, a float32) uint32 {
	a = clamp01(a)
	return stilus.PackRGBA(color.RGBA{
		R: to8(clamp01(r) * a), G: to8(clamp01(g) * a), B: to8(clamp01(b) * a), A: to8(a),
	})
}

// pmColor is a premultiplied color with 8-bit channels widened for arithmetic.
type pmColor struct{ r, g, b, a uint32 }

func toPM(c draw.Color) pmColor {
	p := premul(c, 1)
	return pmColor{uint32(p.R), uint32(p.G), uint32(p.B), uint32(p.A)}
}

// packLE is true where stilus.PackRGBA puts R in the low byte, which the
// integer shader paths write directly; elsewhere they go through PackRGBA.
var packLE = stilus.PackRGBA(color.RGBA{R: 1}) == 1

// pack packs premultiplied 8-bit channels in stilus's layout.
func pack(r, g, b, a uint32) uint32 {
	if packLE {
		return r | g<<8 | b<<16 | a<<24
	}
	return stilus.PackRGBA(color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
}

// scale returns c times coverage a (0..255), packed for stilus.
func (c pmColor) scale(a uint32) uint32 {
	if a >= 255 {
		return pack(c.r, c.g, c.b, c.a)
	}
	return pack(mul255(c.r, a), mul255(c.g, a), mul255(c.b, a), mul255(c.a, a))
}

// mul255 is x*a/255 rounded, for x, a in 0..255.
func mul255(x, a uint32) uint32 {
	t := x*a + 128
	return (t + t>>8) >> 8
}

// noiseHash matches noise_hash in the WGSL rect/gradient shaders.
func noiseHash(x, y int) float32 {
	n := uint32(x)*1597334673 ^ uint32(y)*3812015801
	n = (n ^ (n >> 16)) * 2246822519
	n = (n ^ (n >> 13)) * 3266489917
	n ^= n >> 16
	return float32(n) / 4294967295.0
}

// grainShader paints a solid color with the theme's film grain (RFC-008 §10.5).
type grainShader struct {
	c     draw.Color
	grain float32
}

func (s *grainShader) ShadeSpan(y, x int, dst []uint32) {
	// Straight channels in 0..255; grain shifts all three alike.
	sr, sg, sb := clamp01(s.c.R)*255, clamp01(s.c.G)*255, clamp01(s.c.B)*255
	a := uint32(to8(clamp01(s.c.A)))
	amp := s.grain * 255
	ch := func(v float32) uint32 {
		if v <= 0 {
			return 0
		}
		if v >= 255 {
			return 255
		}
		return uint32(v + 0.5)
	}
	for i := range dst {
		n := float32(0)
		if amp > 0 {
			n = (noiseHash(x+i, y) - 0.5) * amp
		}
		c := pmColor{ch(sr + n), ch(sg + n), ch(sb + n), 255}
		if a < 255 {
			c.r, c.g, c.b = mul255(c.r, a), mul255(c.g, a), mul255(c.b, a)
			c.a = a
		}
		dst[i] = pack(c.r, c.g, c.b, c.a)
	}
}

// gradientShader adds grain on top of a stilus gradient.
type gradientShader struct {
	inner stilus.Shader
	grain float32
}

func (s *gradientShader) ShadeSpan(y, x int, dst []uint32) {
	s.inner.ShadeSpan(y, x, dst)
	for i, v := range dst {
		c := stilus.UnpackRGBA(v)
		if c.A == 0 {
			continue
		}
		a := int32(c.A)
		n := int32((noiseHash(x+i, y) - 0.5) * s.grain * float32(a))
		add := func(ch uint8) uint32 {
			return uint32(min(max(int32(ch)+n, 0), a))
		}
		dst[i] = pack(add(c.R), add(c.G), add(c.B), uint32(a))
	}
}

// shadowShader evaluates the WGSL shadow SDF per pixel. Outer shadows look
// their Gaussian falloff up in a table (1/8 px steps).
type shadowShader struct {
	cx, cy, hw, hh float32
	radius, blur   float32
	inset          bool
	c              draw.Color
	pc             pmColor
	lut            []uint8 // outer: coverage by distance*shadowLUTScale
}

const shadowLUTScale = 8

func (s *shadowShader) setup() {
	s.pc = toPM(s.c)
	if s.inset {
		return
	}
	sigma := math.Max(float64(s.blur)*0.5, 0.001)
	// exp(-d²/2σ²) < 0.001 beyond 3.72σ.
	n := int(3.72*sigma*shadowLUTScale) + 2
	if cap(s.lut) < n {
		s.lut = make([]uint8, n)
	}
	s.lut = s.lut[:n]
	for i := range s.lut {
		d := float64(i) / shadowLUTScale
		a := math.Exp(-0.5 * d * d / (sigma * sigma))
		if a < 0.001 {
			a = 0
		}
		s.lut[i] = uint8(a*255 + 0.5)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func roundedBoxSDF(px, py, bx, by, r float32) float32 {
	qx := max(abs32(px)-bx+r, 0)
	qy := max(abs32(py)-by+r, 0)
	switch {
	case qx == 0:
		return qy - r
	case qy == 0:
		return qx - r
	}
	return float32(math.Sqrt(float64(qx*qx+qy*qy))) - r
}

func smoothstep(e0, e1, x float32) float32 {
	if e1 == e0 {
		if x < e0 {
			return 0
		}
		return 1
	}
	t := clamp01((x - e0) / (e1 - e0))
	return t * t * (3 - 2*t)
}

func (s *shadowShader) ShadeSpan(y, x int, dst []uint32) {
	py := float32(y) + 0.5 - s.cy
	if !s.inset {
		n := len(s.lut)
		for i := range dst {
			px := float32(x+i) + 0.5 - s.cx
			d := max(roundedBoxSDF(px, py, s.hw, s.hh, s.radius), 0)
			k := int(d*shadowLUTScale + 0.5)
			if k >= n {
				dst[i] = 0
				continue
			}
			dst[i] = s.pc.scale(uint32(s.lut[k]))
		}
		return
	}
	fr := min(s.blur, s.hw, s.hh)
	for i := range dst {
		px := float32(x+i) + 0.5 - s.cx
		dist := roundedBoxSDF(px, py, s.hw, s.hh, s.radius)
		fade := smoothstep(-fr, 0, roundedBoxSDF(px, py, s.hw, s.hh, fr))
		a := fade * (1 - smoothstep(0, 0.5, dist))
		if a < 0.001 {
			dst[i] = 0
			continue
		}
		dst[i] = s.pc.scale(uint32(a*255 + 0.5))
	}
}

// glyphShader samples one of the glyph atlases for a glyph quad.
type glyphShader struct {
	kind           glyphKind
	g              *draw.TexturedGlyph
	c              draw.Color
	pc             pmColor
	gray           *image.Gray
	nrgba          *image.NRGBA
	scaleX, scaleY float32 // atlas texels per device pixel
	pxRange        float32 // MSDF screen pixel range
}

func (s *glyphShader) ShadeSpan(y, x int, dst []uint32) {
	g := s.g
	// Atlas coordinate of the pixel centre.
	v := float32(g.SrcY) + (float32(y)+0.5-g.DstY)*s.scaleY
	u0 := float32(g.SrcX) + (float32(x)+0.5-g.DstX)*s.scaleX
	switch s.kind {
	case glyphAlpha:
		iy := min(max(int(math.Floor(float64(v))), g.SrcY), g.SrcY+g.SrcH-1)
		if iy < s.gray.Rect.Min.Y || iy >= s.gray.Rect.Max.Y {
			clear(dst)
			return
		}
		row := s.gray.Pix[(iy-s.gray.Rect.Min.Y)*s.gray.Stride:]
		for i := range dst {
			u := u0 + float32(i)*s.scaleX
			ix := min(max(int(math.Floor(float64(u))), g.SrcX), g.SrcX+g.SrcW-1) - s.gray.Rect.Min.X
			if ix < 0 || ix >= s.gray.Rect.Dx() {
				dst[i] = 0
				continue
			}
			dst[i] = s.pc.scale(uint32(row[ix]))
		}
	case glyphMSDF:
		for i := range dst {
			u := u0 + float32(i)*s.scaleX
			r, gg, b, _ := sampleNRGBABilinear(s.nrgba, u, v, g)
			a := clamp01(s.pxRange*(median3(r, gg, b)-0.5) + 0.5)
			if a < 0.01 {
				dst[i] = 0
				continue
			}
			dst[i] = s.pc.scale(uint32(a*255 + 0.5))
		}
	case glyphColor:
		ta := uint32(to8(clamp01(s.c.A)))
		for i := range dst {
			u := u0 + float32(i)*s.scaleX
			c := sampleNRGBANearest(s.nrgba, u, v, g)
			a := mul255(uint32(c.A), ta)
			dst[i] = pack(mul255(uint32(c.R), a), mul255(uint32(c.G), a), mul255(uint32(c.B), a), a)
		}
	}
}

func median3(r, g, b float32) float32 {
	return max(min(r, g), min(max(r, g), b))
}

// clampToGlyph keeps samples inside the glyph's atlas cell so neighbours never bleed in.
func clampToGlyph(ix, iy int, g *draw.TexturedGlyph) (int, int) {
	if ix < g.SrcX {
		ix = g.SrcX
	} else if ix >= g.SrcX+g.SrcW {
		ix = g.SrcX + g.SrcW - 1
	}
	if iy < g.SrcY {
		iy = g.SrcY
	} else if iy >= g.SrcY+g.SrcH {
		iy = g.SrcY + g.SrcH - 1
	}
	return ix, iy
}

func sampleGrayNearest(img *image.Gray, u, v float32, g *draw.TexturedGlyph) uint8 {
	ix, iy := clampToGlyph(int(math.Floor(float64(u))), int(math.Floor(float64(v))), g)
	if !(image.Point{ix, iy}.In(img.Rect)) {
		return 0
	}
	return img.Pix[(iy-img.Rect.Min.Y)*img.Stride+(ix-img.Rect.Min.X)]
}

func sampleNRGBANearest(img *image.NRGBA, u, v float32, g *draw.TexturedGlyph) color.NRGBA {
	ix, iy := clampToGlyph(int(math.Floor(float64(u))), int(math.Floor(float64(v))), g)
	if !(image.Point{ix, iy}.In(img.Rect)) {
		return color.NRGBA{}
	}
	o := (iy-img.Rect.Min.Y)*img.Stride + (ix-img.Rect.Min.X)*4
	p := img.Pix[o : o+4 : o+4]
	return color.NRGBA{p[0], p[1], p[2], p[3]}
}

func sampleNRGBABilinear(img *image.NRGBA, u, v float32, g *draw.TexturedGlyph) (r, gg, b, a float32) {
	fu, fv := float64(u)-0.5, float64(v)-0.5
	x0, y0 := int(math.Floor(fu)), int(math.Floor(fv))
	tx, ty := float32(fu-float64(x0)), float32(fv-float64(y0))
	var acc [4]float32
	for j := 0; j < 2; j++ {
		wy := ty
		if j == 0 {
			wy = 1 - ty
		}
		for i := 0; i < 2; i++ {
			wx := tx
			if i == 0 {
				wx = 1 - tx
			}
			c := sampleNRGBANearest(img, float32(x0+i)+0.5, float32(y0+j)+0.5, g)
			w := wx * wy
			acc[0] += w * float32(c.R)
			acc[1] += w * float32(c.G)
			acc[2] += w * float32(c.B)
			acc[3] += w * float32(c.A)
		}
	}
	return acc[0] / 255, acc[1] / 255, acc[2] / 255, acc[3] / 255
}

var (
	_ Renderer      = (*StilusRenderer)(nil)
	_ ImageUploader = (*StilusRenderer)(nil)
)
