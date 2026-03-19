package text

import (
	"image"
	"image/color"
	"sync"

	"github.com/go-text/typesetting/di"
	_ "github.com/go-text/typesetting/font" // transitive dep for shaping
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	msdf "github.com/pierrec/msdf/pkg"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// GoTextShaper implements Shaper using go-text/typesetting for full
// OpenType GSUB/GPOS shaping (RFC-003 §3.2). Rasterization still uses
// golang.org/x/image/font/sfnt for bitmap and MSDF glyph rendering.
type GoTextShaper struct {
	fallback *fonts.FontFamily
	families map[string]*fonts.FontFamily // keyed by family name

	hbShaper shaping.HarfbuzzShaper // reusable shaper instance

	mu       sync.Mutex
	faces    map[faceCacheKey]font.Face
	msdfGens map[msdfCacheKey]*msdf.Msdf
}

// NewGoTextShaper creates a shaper backed by the given fallback family.
func NewGoTextShaper(fallback *fonts.FontFamily) *GoTextShaper {
	return &GoTextShaper{
		fallback: fallback,
		families: make(map[string]*fonts.FontFamily),
		faces:    make(map[faceCacheKey]font.Face),
	}
}

// RegisterFamily adds a named font family to the shaper's registry.
func (s *GoTextShaper) RegisterFamily(family *fonts.FontFamily) {
	if family != nil && family.Name != "" {
		s.families[family.Name] = family
	}
}

// resolveFont picks the best font for the given style.
func (s *GoTextShaper) resolveFont(style draw.TextStyle) *fonts.Font {
	family := s.fallback
	if style.FontFamily != "" {
		if fam, ok := s.families[style.FontFamily]; ok {
			family = fam
		}
	}
	if family == nil {
		return nil
	}
	key := fonts.FontFaceKey{Weight: int(style.Weight), Style: fonts.StyleNormal}
	if f, ok := family.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	key.Weight = 400
	if f, ok := family.Faces[key]; ok && !f.IsBitmap() {
		return f
	}
	for _, f := range family.Faces {
		if !f.IsBitmap() {
			return f
		}
	}
	return nil
}

// resolveFontFamily returns the FontFamily for the given style.
func (s *GoTextShaper) resolveFontFamily(style draw.TextStyle) *fonts.FontFamily {
	if style.FontFamily != "" {
		if fam, ok := s.families[style.FontFamily]; ok {
			return fam
		}
	}
	return s.fallback
}

// ResolveFont returns the font that would be used for the given style.
func (s *GoTextShaper) ResolveFont(style draw.TextStyle) *fonts.Font {
	return s.resolveFont(style)
}

// getFace returns a cached sfnt font.Face for rasterization.
func (s *GoTextShaper) getFace(f *fonts.Font, sizePx int) font.Face {
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
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	s.faces[key] = face
	return face
}

// Measure returns text metrics using go-text shaping.
func (s *GoTextShaper) Measure(text string, style draw.TextStyle) draw.TextMetrics {
	f := s.resolveFont(style)
	if f == nil {
		return BitmapShaper{}.Measure(text, style)
	}

	gtFace := f.GoTextFace()
	if gtFace == nil {
		// Fallback to sfnt measurement if go-text face unavailable.
		return s.measureSfnt(text, style, f)
	}

	if len(text) == 0 {
		return draw.TextMetrics{}
	}

	runes := []rune(text)
	sizePx := DpToPixels(style.Size)
	sizeFixed := fixed.I(sizePx)

	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      gtFace,
		Size:      sizeFixed,
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}

	output := s.hbShaper.Shape(input)

	width := fixedToFloat(output.Advance)
	ascent := fixedToFloat(output.LineBounds.Ascent)
	descent := fixedToFloat(-output.LineBounds.Descent) // Descent is typically negative

	return draw.TextMetrics{
		Width:   width,
		Ascent:  ascent,
		Descent: descent,
		Leading: fixedToFloat(output.LineBounds.Gap),
	}
}

// measureSfnt is a fallback measurement path using golang.org/x/image/font.
func (s *GoTextShaper) measureSfnt(text string, style draw.TextStyle, f *fonts.Font) draw.TextMetrics {
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
	}
}

// Shape returns positioned glyphs for each rune using go-text/typesetting.
// Implements per-glyph fallback per RFC-003 §3.4.
func (s *GoTextShaper) Shape(text string, style draw.TextStyle) []ShapedGlyph {
	f := s.resolveFont(style)
	if f == nil {
		return BitmapShaper{}.Shape(text, style)
	}

	gtFace := f.GoTextFace()
	if gtFace == nil {
		// Fallback to sfnt shaping if go-text face unavailable.
		return s.shapeSfnt(text, style, f)
	}

	if len(text) == 0 {
		return nil
	}

	runes := []rune(text)
	sizePx := DpToPixels(style.Size)
	sizeFixed := fixed.I(sizePx)

	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      gtFace,
		Size:      sizeFixed,
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}

	output := s.hbShaper.Shape(input)

	result := make([]ShapedGlyph, 0, len(output.Glyphs))
	family := s.resolveFontFamily(style)

	for _, g := range output.Glyphs {
		clusterIdx := g.TextIndex()
		r := rune(0)
		if clusterIdx >= 0 && clusterIdx < len(runes) {
			r = runes[clusterIdx]
		}

		sg := ShapedGlyph{
			Rune:     r,
			GlyphID:  GlyphID(g.GlyphID),
			Advance:  fixedToFloat(g.Advance),
			OffsetX:  fixedToFloat(g.XOffset),
			OffsetY:  fixedToFloat(g.YOffset),
			BearingX: fixedToFloat(g.XBearing),
			BearingY: fixedToFloat(g.YBearing),
			Width:    fixedToFloat(g.Width),
			Height:   fixedToFloat(g.Height),
			Cluster:  clusterIdx,
		}

		// Per-glyph fallback (RFC-003 §3.4):
		// If the glyph is .notdef (GlyphID 0), try the fallback chain.
		if g.GlyphID == 0 && r != 0 {
			sg = s.fallbackShape(r, clusterIdx, family, style, sizePx)
		}

		result = append(result, sg)
	}

	return result
}

// fallbackShape attempts to find and shape a single glyph using the fallback chain.
func (s *GoTextShaper) fallbackShape(r rune, cluster int, family *fonts.FontFamily, style draw.TextStyle, sizePx int) ShapedGlyph {
	weight := int(style.Weight)
	if weight == 0 {
		weight = 400
	}

	// Search family fallback chain.
	if fbFont := family.FindGlyphFont(r, weight); fbFont != nil {
		if sg, ok := s.shapeSingleGlyph(r, cluster, fbFont, sizePx); ok {
			return sg
		}
	}

	// Try the global embedded fallback.
	if s.fallback != nil && s.fallback != family {
		if fbFont := s.fallback.FindGlyphFont(r, weight); fbFont != nil {
			if sg, ok := s.shapeSingleGlyph(r, cluster, fbFont, sizePx); ok {
				return sg
			}
		}
	}

	// Ultimate fallback: U+FFFD replacement character.
	if r != '\uFFFD' {
		return s.fallbackShape('\uFFFD', cluster, family, style, sizePx)
	}

	// Even U+FFFD not found — return zero-advance placeholder.
	return ShapedGlyph{Rune: r, Cluster: cluster}
}

// shapeSingleGlyph shapes a single rune using a specific font.
func (s *GoTextShaper) shapeSingleGlyph(r rune, cluster int, f *fonts.Font, sizePx int) (ShapedGlyph, bool) {
	gtFace := f.GoTextFace()
	if gtFace == nil {
		return ShapedGlyph{}, false
	}

	runes := []rune{r}
	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    1,
		Direction: di.DirectionLTR,
		Face:      gtFace,
		Size:      fixed.I(sizePx),
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}

	output := s.hbShaper.Shape(input)
	if len(output.Glyphs) == 0 {
		return ShapedGlyph{}, false
	}

	g := output.Glyphs[0]
	if g.GlyphID == 0 {
		return ShapedGlyph{}, false
	}

	return ShapedGlyph{
		Rune:     r,
		GlyphID:  GlyphID(g.GlyphID),
		Advance:  fixedToFloat(g.Advance),
		OffsetX:  fixedToFloat(g.XOffset),
		OffsetY:  fixedToFloat(g.YOffset),
		BearingX: fixedToFloat(g.XBearing),
		BearingY: fixedToFloat(g.YBearing),
		Width:    fixedToFloat(g.Width),
		Height:   fixedToFloat(g.Height),
		Cluster:  cluster,
	}, true
}

// shapeSfnt is a fallback shaping path using golang.org/x/image/font (no go-text).
func (s *GoTextShaper) shapeSfnt(text string, style draw.TextStyle, f *fonts.Font) []ShapedGlyph {
	sizePx := DpToPixels(style.Size)
	face := s.getFace(f, sizePx)
	if face == nil {
		return BitmapShaper{}.Shape(text, style)
	}

	runes := []rune(text)
	out := make([]ShapedGlyph, 0, len(runes))
	prev := rune(-1)

	for i, r := range runes {
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
			bearingY = -fixedToFloat(bounds.Min.Y)
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
			Cluster:  i,
		})
		prev = r
	}
	return out
}

// ── Rasterization (shared with SfntShaper) ──────────────────────

// RasterizeGlyph draws a single glyph into an image.Gray at the given size.
func (s *GoTextShaper) RasterizeGlyph(r rune, style draw.TextStyle) *RasterizedGlyph {
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
		BearingY: float32(-minY),
		Advance:  fixedToFloat(adv),
	}
}

// RasterizeMSDFGlyph renders a single glyph as an MSDF image using pierrec/msdf.
func (s *GoTextShaper) RasterizeMSDFGlyph(r rune, f *fonts.Font, atlasSize int, pxRange float32) *MSDFRasterizedGlyph {
	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

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

	glyph, err := gen.Get(r)
	if err != nil || glyph == nil || glyph.Canvas == nil {
		return nil
	}

	rgba := glyph.Canvas.Image()
	if rgba == nil {
		return nil
	}
	imgBounds := rgba.Bounds()
	w, h := imgBounds.Dx(), imgBounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgba.RGBAAt(imgBounds.Min.X+x, imgBounds.Min.Y+y)
			nrgba.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}

	opts := glyph.Options
	bearingX := float32(opts.PlaneBounds.Left)
	bearingY := float32(-opts.PlaneBounds.Top)
	advance := float32(opts.Advance)

	return &MSDFRasterizedGlyph{
		Image:    nrgba,
		BearingX: bearingX,
		BearingY: bearingY,
		Advance:  advance,
		PxRange:  pxRange,
	}
}

