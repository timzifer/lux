package text

import (
	"image"
	"image/color"
	"math"
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
	"golang.org/x/image/font/sfnt"
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
	msdfGens map[msdfCacheKey]*msdfGenEntry
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

	ascent := fixedToFloat(output.LineBounds.Ascent)
	descent := fixedToFloat(-output.LineBounds.Descent) // Descent is typically negative
	leading := fixedToFloat(output.LineBounds.Gap)

	// Check if any glyphs are .notdef (missing in primary font).
	// If so, use Shape() to get accurate per-glyph fallback advances.
	hasNotdef := false
	for _, g := range output.Glyphs {
		if g.GlyphID == 0 {
			hasNotdef = true
			break
		}
	}

	width := fixedToFloat(output.Advance)
	if hasNotdef {
		shaped := s.Shape(text, style)
		width = 0
		for _, sg := range shaped {
			width += sg.Advance
		}
	}

	return draw.TextMetrics{
		Width:   width,
		Ascent:  ascent,
		Descent: descent,
		Leading: leading,
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

	family := s.resolveFontFamily(style)

	// Collect runs of .notdef glyphs so that multi-codepoint sequences
	// (flags, ZWJ families, skin-tone modifiers) can be shaped as a
	// single run with the fallback font, preserving GSUB composition.
	type notdefRun struct {
		startIdx int // index into output.Glyphs where the run starts in result
		runStart int // first rune index in runes[]
		runEnd   int // one-past-last rune index in runes[]
	}

	result := make([]ShapedGlyph, 0, len(output.Glyphs))
	var runs []notdefRun
	var curRun *notdefRun

	for gi, g := range output.Glyphs {
		clusterIdx := g.TextIndex()
		r := rune(0)
		if clusterIdx >= 0 && clusterIdx < len(runes) {
			r = runes[clusterIdx]
		}

		if g.GlyphID == 0 && r != 0 {
			// Start or extend a .notdef run.
			if curRun == nil {
				runs = append(runs, notdefRun{startIdx: len(result), runStart: clusterIdx})
				curRun = &runs[len(runs)-1]
			}
			// Determine how far this .notdef run extends in the rune array.
			// Multiple .notdef glyphs can share the same cluster (e.g., regional
			// indicators for flags). The actual rune extent is determined by
			// the cluster index of the NEXT non-.notdef glyph, or len(runes).
			nextCluster := len(runes)
			for k := gi + 1; k < len(output.Glyphs); k++ {
				if output.Glyphs[k].GlyphID != 0 {
					nextCluster = output.Glyphs[k].TextIndex()
					break
				}
			}
			if nextCluster > curRun.runEnd {
				curRun.runEnd = nextCluster
			}
			// Placeholder — will be replaced by fallbackShapeRun.
			result = append(result, ShapedGlyph{Rune: r, Cluster: clusterIdx})
		} else {
			curRun = nil
			result = append(result, ShapedGlyph{
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
			})
		}
	}

	// Replace .notdef runs with fallback-shaped runs.
	// Process in reverse so that splicing doesn't shift earlier indices.
	for i := len(runs) - 1; i >= 0; i-- {
		run := runs[i]
		if run.runEnd <= run.runStart {
			continue // degenerate run (same cluster as next glyph)
		}
		shaped := s.fallbackShapeRun(runes[run.runStart:run.runEnd], run.runStart, family, style, sizePx)
		// Count how many placeholder glyphs this run occupies.
		runLen := 0
		for j := run.startIdx; j < len(result); j++ {
			if j == run.startIdx || (result[j].GlyphID == 0 && result[j].Font == nil) {
				runLen++
			} else {
				break
			}
		}
		// Splice: replace runLen placeholders with the shaped glyphs.
		tail := append([]ShapedGlyph{}, result[run.startIdx+runLen:]...)
		result = append(result[:run.startIdx], shaped...)
		result = append(result, tail...)
	}

	return result
}

// fallbackShapeRun shapes a contiguous run of runes using the fallback chain.
// The entire run is shaped together so that OpenType GSUB rules (flag
// composition, ZWJ joining, skin-tone modifiers) can combine multi-codepoint
// sequences into single glyphs.
func (s *GoTextShaper) fallbackShapeRun(runes []rune, clusterOffset int, family *fonts.FontFamily, style draw.TextStyle, sizePx int) []ShapedGlyph {
	if len(runes) == 0 {
		return nil
	}
	weight := int(style.Weight)
	if weight == 0 {
		weight = 400
	}

	// Find a fallback font that covers the first rune of the run.
	var fbFont *fonts.Font
	if family != nil {
		fbFont = family.FindGlyphFont(runes[0], weight)
	}
	if fbFont == nil && s.fallback != nil && s.fallback != family {
		fbFont = s.fallback.FindGlyphFont(runes[0], weight)
	}

	if fbFont != nil {
		if shaped := s.shapeRunWithFont(runes, clusterOffset, fbFont, sizePx); len(shaped) > 0 {
			return shaped
		}
	}

	// Per-rune fallback as last resort (for runs where no single font covers all runes).
	result := make([]ShapedGlyph, 0, len(runes))
	for i, r := range runes {
		cluster := clusterOffset + i
		sg := s.fallbackShapeSingle(r, cluster, family, style, sizePx)
		result = append(result, sg)
	}
	return result
}

// fallbackShapeSingle attempts to find and shape a single glyph using the fallback chain.
func (s *GoTextShaper) fallbackShapeSingle(r rune, cluster int, family *fonts.FontFamily, style draw.TextStyle, sizePx int) ShapedGlyph {
	weight := int(style.Weight)
	if weight == 0 {
		weight = 400
	}

	if family != nil {
		if fbFont := family.FindGlyphFont(r, weight); fbFont != nil {
			if sg, ok := s.shapeSingleGlyph(r, cluster, fbFont, sizePx); ok {
				return sg
			}
		}
	}

	if s.fallback != nil && s.fallback != family {
		if fbFont := s.fallback.FindGlyphFont(r, weight); fbFont != nil {
			if sg, ok := s.shapeSingleGlyph(r, cluster, fbFont, sizePx); ok {
				return sg
			}
		}
	}

	// Ultimate fallback: U+FFFD replacement character.
	if r != '\uFFFD' {
		return s.fallbackShapeSingle('\uFFFD', cluster, family, style, sizePx)
	}

	return ShapedGlyph{Rune: r, Cluster: cluster}
}

// shapeRunWithFont shapes a slice of runes as a single run using the given font.
// This preserves OpenType GSUB/GPOS rules for multi-codepoint sequences.
func (s *GoTextShaper) shapeRunWithFont(runes []rune, clusterOffset int, f *fonts.Font, sizePx int) []ShapedGlyph {
	gtFace := f.GoTextFace()
	if gtFace == nil {
		return nil
	}

	inp := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      gtFace,
		Size:      fixed.I(sizePx),
		Script:    language.Common,
		Language:  language.NewLanguage("en"),
	}

	output := s.hbShaper.Shape(inp)
	if len(output.Glyphs) == 0 {
		return nil
	}

	result := make([]ShapedGlyph, 0, len(output.Glyphs))
	for _, g := range output.Glyphs {
		if g.GlyphID == 0 {
			continue // skip .notdef in fallback output
		}
		clusterIdx := g.TextIndex() + clusterOffset
		r := rune(0)
		if g.TextIndex() >= 0 && g.TextIndex() < len(runes) {
			r = runes[g.TextIndex()]
		}
		result = append(result, ShapedGlyph{
			Rune:     r,
			GlyphID:  GlyphID(g.GlyphID),
			Font:     f,
			Advance:  fixedToFloat(g.Advance),
			OffsetX:  fixedToFloat(g.XOffset),
			OffsetY:  fixedToFloat(g.YOffset),
			BearingX: fixedToFloat(g.XBearing),
			BearingY: fixedToFloat(g.YBearing),
			Width:    fixedToFloat(g.Width),
			Height:   fixedToFloat(g.Height),
			Cluster:  clusterIdx,
		})
	}
	return result
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
		Font:     f,
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
// It uses the OpenType GlyphID directly (post-GSUB) so that ligature glyphs
// such as "ff" are rendered correctly instead of the base rune.
func (s *GoTextShaper) RasterizeGlyph(id GlyphID, style draw.TextStyle) *RasterizedGlyph {
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

// RasterizeGlyphWithFont draws a single glyph using an explicit font (for per-glyph fallback).
func (s *GoTextShaper) RasterizeGlyphWithFont(id GlyphID, f *fonts.Font, style draw.TextStyle) *RasterizedGlyph {
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

// RasterizeMSDFGlyph renders a single glyph as an MSDF image using pierrec/msdf.
// The msdf library works with runes, so hintRune is used. If hintRune's cmap
// GlyphIndex doesn't match id (e.g., ligature), returns nil so the caller
// can fall back to bitmap rasterization.
func (s *GoTextShaper) RasterizeMSDFGlyph(id GlyphID, hintRune rune, f *fonts.Font, atlasSize int, pxRange float32) *MSDFRasterizedGlyph {
	sf := f.SfntFont()
	if sf == nil {
		return nil
	}

	// Verify hintRune maps to the expected GlyphID via cmap.
	var buf sfnt.Buffer
	cmapIdx, err := sf.GlyphIndex(&buf, hintRune)
	if err != nil || GlyphID(cmapIdx) != id {
		return nil
	}

	s.mu.Lock()
	cacheKey := msdfCacheKey{fontID: f.ID(), size: atlasSize}
	entry, ok := s.msdfGens[cacheKey]
	if !ok {
		entry = &msdfGenEntry{
			gen: msdf.NewFromFont(sf, &msdf.Config{
				Size:          float64(atlasSize),
				DistanceField: float64(pxRange),
			}),
		}
		if s.msdfGens == nil {
			s.msdfGens = make(map[msdfCacheKey]*msdfGenEntry)
		}
		s.msdfGens[cacheKey] = entry
	}
	s.mu.Unlock()

	entry.mu.Lock()
	glyph, err := entry.gen.Get(hintRune)
	entry.mu.Unlock()
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
// Metrics are returned at the CBDT ppem scale (unscaled); the caller
// scales to the requested size using atlas.ColorPPEM.
func (s *GoTextShaper) RasterizeColorGlyph(id GlyphID, f *fonts.Font, sizePx int) *ColorRasterizedGlyph {
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

