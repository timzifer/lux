package gpu

import (
	"image"
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/stilus"
)

// drawSurface rasterizes a CPU surface straight into the framebuffer.
func (r *StilusRenderer) drawSurface(s *draw.DrawSurface) {
	if s.Raster == nil || s.W <= 0 || s.H <= 0 {
		return
	}
	rect := image.Rect(s.X, s.Y, s.X+s.W, s.Y+s.H)
	clip := rect.Intersect(r.img.Rect)
	if s.ClipW > 0 && s.ClipH > 0 {
		clip = clip.Intersect(image.Rect(s.ClipX, s.ClipY, s.ClipX+s.ClipW, s.ClipY+s.ClipH))
	}
	if clip.Empty() {
		return
	}
	s.Raster.RasterizeSurface(r.img, rect, clip)
}

// drawShaderRect fills a (rounded) rect with a built-in effect or the
// CPU function of a custom shader.
func (r *stilusWorker) drawShaderRect(d *draw.DrawShaderRect) {
	if d.W <= 0 || d.H <= 0 {
		return
	}
	fn := d.CPU
	if fn == nil {
		fn = builtinEffect(d.ShaderKey)
	}
	if fn == nil {
		return // custom WGSL without a CPU implementation
	}
	r.fx = effectShader{
		fn: fn, x: float32(d.X), y: float32(d.Y), w: float32(d.W), h: float32(d.H),
		t: d.Time, params: d.Params,
	}
	r.path.Reset()
	roundRect(&r.path, float32(d.X), float32(d.Y), float32(d.W), float32(d.H), d.Radius)
	r.paint = stilus.Paint{Shader: &r.fx}
	r.canvas.Fill(&r.path, stilus.Identity, stilus.NonZero, &r.paint)
}

// effectShader evaluates a ShaderFunc at pixel centres, uv over the rect.
type effectShader struct {
	fn         draw.ShaderFunc
	x, y, w, h float32
	t          float32
	params     [8]float32
}

func (s *effectShader) ShadeSpan(y, x int, dst []uint32) {
	v := (float32(y) + 0.5 - s.y) / s.h
	for i := range dst {
		u := (float32(x+i) + 0.5 - s.x) / s.w
		c := s.fn(u, v, s.t, &s.params)
		dst[i] = packStraight(c.R, c.G, c.B, c.A)
	}
}

// builtinEffect returns the Go port of a built-in WGSL effect
// (internal/gpu/wgpu_shaders.go), or nil.
func builtinEffect(key string) draw.ShaderFunc {
	switch key {
	case "_builtin:noise":
		return EffectNoise
	case "_builtin:plasma":
		return EffectPlasma
	case "_builtin:voronoi":
		return EffectVoronoi
	}
	return nil
}

func fract(x float32) float32 { return x - float32(math.Floor(float64(x))) }

func sin32(x float32) float32 { return float32(math.Sin(float64(x))) }

func mix(a, b, t float32) float32 { return a + (b-a)*t }

// hash22 matches the noise shader's hash22.
func hash22(px, py float32) (float32, float32) {
	p3x, p3y, p3z := fract(px*0.1031), fract(py*0.1030), fract(px*0.0973)
	d := p3x*(p3y+33.33) + p3y*(p3z+33.33) + p3z*(p3x+33.33)
	p3x, p3y, p3z = p3x+d, p3y+d, p3z+d
	return fract((p3x + p3y) * p3z), fract((p3x + p3z) * p3y)
}

func gradNoise(px, py float32) float32 {
	ix, iy := float32(math.Floor(float64(px))), float32(math.Floor(float64(py)))
	fx, fy := px-ix, py-iy
	ux, uy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	corner := func(ox, oy float32) float32 {
		hx, hy := hash22(ix+ox, iy+oy)
		return (hx-0.5)*(fx-ox) + (hy-0.5)*(fy-oy)
	}
	a, b, c, d := corner(0, 0), corner(1, 0), corner(0, 1), corner(1, 1)
	return mix(mix(a, b, ux), mix(c, d, ux), uy) + 0.5
}

// EffectNoise is the CPU form of draw.ShaderEffectNoise.
func EffectNoise(u, v, t float32, p *[8]float32) draw.Color {
	scale := p[0] + 4
	n := gradNoise(u*scale+t*0.1, v*scale)
	return draw.Color{R: n * p[1], G: n * p[2], B: n * p[3], A: 1}
}

// EffectPlasma is the CPU form of draw.ShaderEffectPlasma.
func EffectPlasma(u, v, t float32, p *[8]float32) draw.Color {
	s := p[0] + 4
	x, y := u*s, v*s
	val := (sin32(x+t) + sin32(y+t) + sin32(x+y+t) + sin32(float32(math.Sqrt(float64(x*x+y*y)))+t)) * 0.25
	return draw.Color{
		R: sin32(val*3.14159)*0.5 + 0.5,
		G: sin32(val*3.14159+2.094)*0.5 + 0.5,
		B: sin32(val*3.14159+4.189)*0.5 + 0.5,
		A: 1,
	}
}

func hash2(px, py float32) (float32, float32) {
	return fract(sin32(px*127.1+py*311.7) * 43758.5453), fract(sin32(px*269.5+py*183.3) * 43758.5453)
}

// EffectVoronoi is the CPU form of draw.ShaderEffectVoronoi.
func EffectVoronoi(u, v, t float32, p *[8]float32) draw.Color {
	scale := p[0] + 4
	x, y := u*scale, v*scale
	ix, iy := float32(math.Floor(float64(x))), float32(math.Floor(float64(y)))
	fx, fy := x-ix, y-iy
	minDist := float32(1)
	var cell float32
	for j := float32(-1); j <= 1; j++ {
		for i := float32(-1); i <= 1; i++ {
			hx, hy := hash2(ix+i, iy+j)
			ax := i + sin32(hx*6.28+t)*0.5 + 0.5
			ay := j + sin32(hy*6.28+t)*0.5 + 0.5
			dx, dy := ax-fx, ay-fy
			if d := dx*dx + dy*dy; d < minDist {
				minDist, cell = d, hx
			}
		}
	}
	c := cell*0.5 + 0.25
	return draw.Color{R: c, G: c * 0.8, B: c * 1.2, A: 1}
}
