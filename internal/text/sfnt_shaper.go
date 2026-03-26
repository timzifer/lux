package text

import (
	"image"
	"image/color"
	"math"
	"sync"

	msdf "github.com/pierrec/msdf/pkg"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// SfntShaper implements Shaper using golang.org/x/image/font/sfnt.
// It caches font.Face instances keyed by (font ID, pixel size).
type SfntShaper struct {
	fallback *fonts.FontFamily
	families map[string]*fonts.FontFamily // keyed by family name

	mu       sync.Mutex
	faces    map[faceCacheKey]font.Face
	msdfGens map[msdfCacheKey]*msdf.Msdf
}

type faceCacheKey struct {
	fontID uint64
	sizePx int
}

// NewSfntShaper creates a shaper backed by the given fallback family.
func NewSfntShaper(fallback *fonts.FontFamily) *SfntShaper {
	return &SfntShaper{
		fallback: fallback,
		families: make(map[string]*fonts.FontFamily),
		faces:    make(map[faceCacheKey]font.Face),
	}
}

// RegisterFamily adds a named font family to the shaper's registry.
func (s *SfntShaper) RegisterFamily(family *fonts.FontFamily) {
	if family != nil && family.Name != "" {
		s.families[family.Name] = family
	}
}

// resolveFont picks the best font for the given style.
// If style.FontFamily matches a registered family, that family is used;
// otherwise the default fallback family is used.
func (s *SfntShaper) resolveFont(style draw.TextStyle) *fonts.Font {
	family := s.fallback
	if style.FontFamily != "" {
		if fam, ok := s.families[style.FontFamily]; ok {
			family = fam
		}
	}
	if family == nil {
		return nil
	}
	// Try exact weight match.
	key := fonts.FontFaceKey{Weight: int(style.Weight), Style: fonts.StyleNormal}
	if f, ok := family.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	// Fall back to Regular.
	key.Weight = 400
	if f, ok := family.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	// Try any sfnt face.
	for _, f := range family.Faces {
		if !f.IsBitmap() {
			return f
		}
	}
	return nil
}

// getFace returns a cached font.Face for the given font and pixel size.
func (s *SfntShaper) getFace(f *fonts.Font, sizePx int) font.Face {
	if sizePx < 1 {
		sizePx = 1
	}

	key := faceCacheKey{fontID: f.ID(), sizePx: sizePx}

	s.mu.Lock()
	defer s.mu.Unlock()

	if face, ok := s.faces[key]; ok {
		return face
	}

	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

	face, err := opentype.NewFace(sf, &opentype.FaceOptions{
		Size:    float64(sizePx),
		DPI:     72, // 1 dp = 1 px at DPR 1.0
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}

	s.faces[key] = face
	return face
}

// ResolveFont returns the font that would be used for the given style.
// Exported for use by the rendering pipeline (atlas font ID lookup).
func (s *SfntShaper) ResolveFont(style draw.TextStyle) *fonts.Font {
	return s.resolveFont(style)
}

// Measure returns text metrics using the sfnt font face.
func (s *SfntShaper) Measure(text string, style draw.TextStyle) draw.TextMetrics {
	f := s.resolveFont(style)
	if f == nil {
		return BitmapShaper{}.Measure(text, style)
	}

	sizePx := DpToPixels(style.Size)
	face := s.getFace(f, sizePx)
	if face == nil {
		return BitmapShaper{}.Measure(text, style)
	}

	metrics := face.Metrics()
	ascent := fixedToFloat(metrics.Ascent)
	descent := fixedToFloat(metrics.Descent)

	var width fixed.Int26_6
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			width += face.Kern(prev, r)
		}
		adv, ok := face.GlyphAdvance(r)
		if !ok {
			adv, _ = face.GlyphAdvance('?')
		}
		width += adv
		prev = r
	}

	return draw.TextMetrics{
		Width:   fixedToFloat(width),
		Ascent:  ascent,
		Descent: descent,
		Leading: 0,
	}
}

// Shape returns positioned glyphs for each rune in the text.
func (s *SfntShaper) Shape(text string, style draw.TextStyle) []ShapedGlyph {
	f := s.resolveFont(style)
	if f == nil {
		return BitmapShaper{}.Shape(text, style)
	}

	sizePx := DpToPixels(style.Size)
	face := s.getFace(f, sizePx)
	if face == nil {
		return BitmapShaper{}.Shape(text, style)
	}

	runes := []rune(text)
	out := make([]ShapedGlyph, 0, len(runes))
	prev := rune(-1)

	for _, r := range runes {
		var kern fixed.Int26_6
		if prev >= 0 {
			kern = face.Kern(prev, r)
		}

		adv, ok := face.GlyphAdvance(r)
		if !ok {
			adv, _ = face.GlyphAdvance('?')
		}

		var bearingX, bearingY, gw, gh float32
		bounds, _, ok := face.GlyphBounds(r)
		if ok {
			bearingX = fixedToFloat(bounds.Min.X)
			bearingY = -fixedToFloat(bounds.Min.Y) // flip: Min.Y is negative for ascenders
			gw = fixedToFloat(bounds.Max.X - bounds.Min.X)
			gh = fixedToFloat(bounds.Max.Y - bounds.Min.Y)
		}

		out = append(out, ShapedGlyph{
			Rune:     r,
			Advance:  fixedToFloat(adv + kern),
			BearingX: bearingX,
			BearingY: bearingY,
			Width:    gw,
			Height:   gh,
		})
		prev = r
	}

	return out
}

// RasterizedGlyph holds the rasterized image plus the pixel-aligned metrics
// that match the rasterization bounds exactly.
type RasterizedGlyph struct {
	Image    *image.Gray
	Font     *fonts.Font
	BearingX float32 // pixel-aligned horizontal bearing (matches Floor of bounds.Min.X)
	BearingY float32 // pixel-aligned vertical bearing (matches Floor of bounds.Min.Y, negated)
	Advance  float32
}

// RasterizeGlyph draws a single glyph into an image.Gray at the given size.
// It uses the OpenType GlyphID directly (post-GSUB) so that ligature glyphs
// such as "ff" are rendered correctly instead of the base rune.
// The returned bearings use Floor/Ceil to match the rasterized pixel bounds
// exactly, preventing sub-pixel baseline misalignment.
func (s *SfntShaper) RasterizeGlyph(id GlyphID, style draw.TextStyle) *RasterizedGlyph {
	f := s.resolveFont(style)
	if f == nil {
		return nil
	}
	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

	sizePx := DpToPixels(style.Size)
	ppem := fixed.I(sizePx)

	glyphIdx := sfnt.GlyphIndex(id)
	var buf sfnt.Buffer

	bounds, advance, err := sf.GlyphBounds(&buf, glyphIdx, ppem, font.HintingFull)
	if err != nil {
		return nil
	}

	minX := bounds.Min.X.Floor()
	minY := bounds.Min.Y.Floor()
	maxX := bounds.Max.X.Ceil()
	maxY := bounds.Max.Y.Ceil()

	w := maxX - minX
	h := maxY - minY
	if w <= 0 || h <= 0 {
		return nil
	}

	// Load glyph outline segments by index — this handles ligatures correctly.
	segments, err := sf.LoadGlyph(&buf, glyphIdx, ppem, &sfnt.LoadGlyphOptions{})
	if err != nil {
		return nil
	}

	// Rasterize the outline segments into a grayscale image.
	img := rasterizeSegments(segments, w, h, -minX, -minY)

	return &RasterizedGlyph{
		Image:    img,
		Font:     f,
		BearingX: float32(minX),
		BearingY: float32(-minY),
		Advance:  fixedToFloat(advance),
	}
}

// RasterizeGlyphWithFont draws a single glyph using an explicit font (for per-glyph fallback).
func (s *SfntShaper) RasterizeGlyphWithFont(id GlyphID, f *fonts.Font, style draw.TextStyle) *RasterizedGlyph {
	if f == nil {
		return nil
	}
	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

	sizePx := DpToPixels(style.Size)
	ppem := fixed.I(sizePx)

	glyphIdx := sfnt.GlyphIndex(id)
	var buf sfnt.Buffer

	bounds, advance, err := sf.GlyphBounds(&buf, glyphIdx, ppem, font.HintingFull)
	if err != nil {
		return nil
	}

	minX := bounds.Min.X.Floor()
	minY := bounds.Min.Y.Floor()
	maxX := bounds.Max.X.Ceil()
	maxY := bounds.Max.Y.Ceil()

	w := maxX - minX
	h := maxY - minY
	if w <= 0 || h <= 0 {
		return nil
	}

	segments, err := sf.LoadGlyph(&buf, glyphIdx, ppem, &sfnt.LoadGlyphOptions{})
	if err != nil {
		return nil
	}

	img := rasterizeSegments(segments, w, h, -minX, -minY)

	return &RasterizedGlyph{
		Image:    img,
		Font:     f,
		BearingX: float32(minX),
		BearingY: float32(-minY),
		Advance:  fixedToFloat(advance),
	}
}

// rasterizeSegments converts sfnt outline segments into a grayscale image
// using golang.org/x/image/vector.Rasterizer.
func rasterizeSegments(segments sfnt.Segments, w, h, offsetX, offsetY int) *image.Gray {
	r := vector.NewRasterizer(w, h)

	ox := float32(offsetX)
	oy := float32(offsetY)

	for _, seg := range segments {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			r.MoveTo(
				fixedToFloat(seg.Args[0].X)+ox,
				fixedToFloat(seg.Args[0].Y)+oy,
			)
		case sfnt.SegmentOpLineTo:
			r.LineTo(
				fixedToFloat(seg.Args[0].X)+ox,
				fixedToFloat(seg.Args[0].Y)+oy,
			)
		case sfnt.SegmentOpQuadTo:
			r.QuadTo(
				fixedToFloat(seg.Args[0].X)+ox,
				fixedToFloat(seg.Args[0].Y)+oy,
				fixedToFloat(seg.Args[1].X)+ox,
				fixedToFloat(seg.Args[1].Y)+oy,
			)
		case sfnt.SegmentOpCubeTo:
			r.CubeTo(
				fixedToFloat(seg.Args[0].X)+ox,
				fixedToFloat(seg.Args[0].Y)+oy,
				fixedToFloat(seg.Args[1].X)+ox,
				fixedToFloat(seg.Args[1].Y)+oy,
				fixedToFloat(seg.Args[2].X)+ox,
				fixedToFloat(seg.Args[2].Y)+oy,
			)
		}
	}

	dst := image.NewGray(image.Rect(0, 0, w, h))
	r.Draw(dst, dst.Bounds(), image.White, image.Point{})
	return dst
}

// MSDFRasterizedGlyph holds the MSDF-rendered image and metrics.
type MSDFRasterizedGlyph struct {
	Image    *image.NRGBA
	BearingX float32
	BearingY float32
	Advance  float32
	PxRange  float32
}

// RasterizeMSDFGlyph renders a single glyph as an MSDF image using pierrec/msdf.
// atlasSize is the ppem size (typically 32), pxRange is the SDF distance range.
// The msdf library works with runes (cmap lookup), so hintRune is used.
// If hintRune's cmap GlyphIndex doesn't match id (e.g., ligature), returns nil
// so the caller can fall back to bitmap rasterization.
func (s *SfntShaper) RasterizeMSDFGlyph(id GlyphID, hintRune rune, f *fonts.Font, atlasSize int, pxRange float32) *MSDFRasterizedGlyph {
	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

	// Verify hintRune maps to the expected GlyphID via cmap.
	// If not (ligature glyph), we can't use the rune-based msdf library.
	var buf sfnt.Buffer
	cmapIdx, err := sf.GlyphIndex(&buf, hintRune)
	if err != nil || GlyphID(cmapIdx) != id {
		return nil
	}

	// Get or create the MSDF generator for this font+size combination.
	s.mu.Lock()
	cacheKey := msdfCacheKey{fontID: f.ID(), size: atlasSize}
	gen, ok := s.msdfGens[cacheKey]
	if !ok {
		gen = msdf.NewFromFont(sf, &msdf.Config{
			Size:          float64(atlasSize),
			DistanceField: float64(pxRange),
		})
		if s.msdfGens == nil {
			s.msdfGens = make(map[msdfCacheKey]*msdf.Msdf)
		}
		s.msdfGens[cacheKey] = gen
	}
	s.mu.Unlock()

	glyph, err := gen.Get(hintRune)
	if err != nil || glyph == nil || glyph.Canvas == nil {
		return nil
	}

	rgba := glyph.Canvas.Image()
	if rgba == nil {
		return nil
	}
	bounds := rgba.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	// Convert *image.RGBA to *image.NRGBA.
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgba.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			nrgba.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}

	opts := glyph.Options

	// Chlumsky error correction: detect and fix corner artifacts where
	// median3(R,G,B) disagrees with the true inside/outside classification.
	glyphIdx := sfnt.GlyphIndex(id)
	segments, segErr := sf.LoadGlyph(&buf, glyphIdx, fixed.I(atlasSize), &sfnt.LoadGlyphOptions{})
	if segErr == nil && len(segments) > 0 {
		correctMSDFCorners(nrgba, segments,
			float32(opts.PlaneBounds.Left), float32(opts.PlaneBounds.Top),
			float32(opts.PlaneBounds.Right), float32(opts.PlaneBounds.Bottom),
			pxRange)
	}

	// PlaneBounds and Advance are already in pixel units at the given ppem
	// (Config.Size). BearingX = left edge offset from cursor.
	// BearingY = distance from baseline to top of glyph (negate Min.Y/Top
	// since Min.Y is negative for glyphs above baseline in sfnt coords).
	bearingX := float32(math.Floor(float64(opts.PlaneBounds.Left)))
	bearingY := float32(-math.Floor(float64(opts.PlaneBounds.Top)))
	advance := float32(opts.Advance)

	return &MSDFRasterizedGlyph{
		Image:    nrgba,
		BearingX: bearingX,
		BearingY: bearingY,
		Advance:  advance,
		PxRange:  pxRange,
	}
}

// RasterizeColorGlyph extracts a color bitmap glyph from a CBDT font.
func (s *SfntShaper) RasterizeColorGlyph(id GlyphID, f *fonts.Font, sizePx int) *ColorRasterizedGlyph {
	if f == nil || !f.HasCBDT() {
		return nil
	}
	cbdt := ParseCBDT(f.RawData())
	if cbdt == nil {
		return nil
	}
	img, metrics := cbdt.RasterizeGlyph(uint16(id))
	if img == nil {
		return nil
	}
	return &ColorRasterizedGlyph{
		Image:    img,
		BearingX: float32(metrics.BearingX),
		BearingY: float32(metrics.BearingY),
		Advance:  float32(metrics.Advance),
		PPEM:     cbdt.PPEM(),
	}
}

type msdfCacheKey struct {
	fontID uint64
	size   int
}

// ── Helpers ──────────────────────────────────────────────────────

// DpToPixels converts dp size to pixel size (at DPR 1.0, 1 dp = 1 px).
func DpToPixels(dp float32) int {
	px := int(math.Round(float64(dp)))
	if px < 1 {
		px = 1
	}
	return px
}

func fixedToFloat(v fixed.Int26_6) float32 {
	return float32(v) / 64.0
}
