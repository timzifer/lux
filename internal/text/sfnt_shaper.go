package text

import (
	"image"
	"math"
	"sync"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// SfntShaper implements Shaper using golang.org/x/image/font/sfnt.
// It caches font.Face instances keyed by (font ID, pixel size).
type SfntShaper struct {
	fallback *fonts.FontFamily

	mu    sync.Mutex
	faces map[faceCacheKey]font.Face
}

type faceCacheKey struct {
	fontID uint64
	sizePx int
}

// NewSfntShaper creates a shaper backed by the given fallback family.
func NewSfntShaper(fallback *fonts.FontFamily) *SfntShaper {
	return &SfntShaper{
		fallback: fallback,
		faces:    make(map[faceCacheKey]font.Face),
	}
}

// resolveFont picks the best font for the given style from the fallback family.
func (s *SfntShaper) resolveFont(style draw.TextStyle) *fonts.Font {
	if s.fallback == nil {
		return nil
	}
	// Try exact weight match.
	key := fonts.FontFaceKey{Weight: int(style.Weight), Style: fonts.StyleNormal}
	if f, ok := s.fallback.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	// Fall back to Regular.
	key.Weight = 400
	if f, ok := s.fallback.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	// Try any sfnt face.
	for _, f := range s.fallback.Faces {
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
// The returned bearings use Floor/Ceil to match the rasterized pixel bounds
// exactly, preventing sub-pixel baseline misalignment.
func (s *SfntShaper) RasterizeGlyph(r rune, style draw.TextStyle) *RasterizedGlyph {
	f := s.resolveFont(style)
	if f == nil {
		return nil
	}

	sizePx := DpToPixels(style.Size)
	face := s.getFace(f, sizePx)
	if face == nil {
		return nil
	}

	bounds, _, ok := face.GlyphBounds(r)
	if !ok {
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

	img := image.NewGray(image.Rect(0, 0, w, h))

	d := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(-minX), Y: fixed.I(-minY)},
	}
	d.DrawString(string(r))

	adv, ok := face.GlyphAdvance(r)
	if !ok {
		adv, _ = face.GlyphAdvance('?')
	}

	return &RasterizedGlyph{
		Image:    img,
		Font:     f,
		BearingX: float32(minX),
		BearingY: float32(-minY), // negate: minY is negative for ascenders
		Advance:  fixedToFloat(adv),
	}
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
