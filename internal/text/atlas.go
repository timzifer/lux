package text

import (
	"image"
	"sync/atomic"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
)

// msdfResult holds a completed MSDF glyph rasterized in a background goroutine.
type msdfResult struct {
	Key     GlyphKey
	Image   *image.NRGBA
	Bearing draw.Point
	Advance float32
	PxRange float32
}

// GlyphKey uniquely identifies a rasterized glyph in the atlas.
// It uses the OpenType GlyphID (post-GSUB) so that ligature glyphs
// like "ff" get their own atlas entry distinct from a regular "f".
// Rune is kept as a hint for rasterizers that need a cmap rune (e.g., MSDF).
// When MSDF is true, SizePx is fixed at MSDFAtlasSize.
type GlyphKey struct {
	FontID  uint64
	GlyphID GlyphID
	Rune    rune // hint rune for MSDF rasterization; GlyphID is the real key
	SizePx  uint16
	MSDF    bool
}

// AtlasEntry describes the location and metrics of a glyph in the atlas.
type AtlasEntry struct {
	X, Y, W, H          int     // position and size in the atlas image
	BearingX, BearingY   float32 // glyph bearing (offset from cursor)
	Advance              float32 // horizontal advance
	PxRange              float32 // MSDF pixel range (0 for bitmap glyphs)
}

// MSDFAtlasSize is the default pixel size at which MSDF glyphs are rasterized.
// Use MSDFBucketSize to pick an adaptive size based on display size.
const MSDFAtlasSize = 32

// MSDFPxRange is the SDF distance field range in pixels.
const MSDFPxRange = 4.0

// MSDFMinSize is the minimum requested text size (in px) for using MSDF.
// Below this threshold the hinted bitmap rasterizer is used instead,
// which produces sharper results at small sizes thanks to pixel-grid hinting.
const MSDFMinSize = 24

// MSDFMaxBucket is the largest atlas bucket size.
const MSDFMaxBucket = 256

// MSDFBucketSize returns the MSDF atlas rasterization size for a given
// display size in pixels. The bucket strategy limits upscaling to ~2x:
//
//	24-47px  → 32px   (max 1.47x)
//	48-95px  → 64px   (max 1.48x)
//	96-191px → 128px  (max 1.49x)
//	192px+   → 256px  (max ~2x for typical sizes)
func MSDFBucketSize(displayPx uint16) uint16 {
	switch {
	case displayPx >= 192:
		return MSDFMaxBucket
	case displayPx >= 96:
		return 128
	case displayPx >= 48:
		return 64
	default:
		return MSDFAtlasSize
	}
}

// GlyphAtlas caches rasterized glyphs in a grayscale texture atlas.
// The atlas uses simple row-based packing: glyphs are placed left to right,
// wrapping to the next row when the current row is full.
// A secondary NRGBA atlas is used for MSDF glyphs.
type GlyphAtlas struct {
	Image   *image.Gray
	Entries map[GlyphKey]AtlasEntry
	Width   int
	Height  int
	Dirty   bool // set when the atlas image has been modified since last GPU upload

	cursorX   int
	cursorY   int
	rowHeight int

	// MSDF atlas (RGB channels encode signed distance fields).
	MSDFImage     *image.NRGBA
	MSDFWidth     int
	MSDFHeight    int
	MSDFDirty     bool
	msdfCursorX   int
	msdfCursorY   int
	msdfRowHeight int

	// Async MSDF rasterization state.
	msdfPending map[GlyphKey]struct{} // keys currently being rasterized in background
	msdfResults chan msdfResult       // completed MSDF glyphs from goroutines
	msdfNotify  func()               // callback to trigger repaint (set via SetMSDFNotify)
	msdfSignal  atomic.Bool          // debounce flag: true while a notify is in flight

	// Color emoji atlas (NRGBA with actual color data).
	ColorImage     *image.NRGBA
	ColorWidth     int
	ColorHeight    int
	ColorDirty     bool
	ColorPPEM      int // pixels-per-em of the CBDT strike (for scaling)
	colorCursorX   int
	colorCursorY   int
	colorRowHeight int
}

// NewGlyphAtlas creates an atlas with the given initial dimensions.
func NewGlyphAtlas(w, h int) *GlyphAtlas {
	return &GlyphAtlas{
		Image:       image.NewGray(image.Rect(0, 0, w, h)),
		Entries:     make(map[GlyphKey]AtlasEntry),
		Width:       w,
		Height:      h,
		MSDFImage:   image.NewNRGBA(image.Rect(0, 0, 512, 512)),
		MSDFWidth:   512,
		MSDFHeight:  512,
		msdfPending: make(map[GlyphKey]struct{}),
		msdfResults: make(chan msdfResult, 256),
		ColorImage:  image.NewNRGBA(image.Rect(0, 0, 512, 512)),
		ColorWidth:  512,
		ColorHeight: 512,
	}
}

// Lookup returns the atlas entry for the given key, if it exists.
func (a *GlyphAtlas) Lookup(key GlyphKey) (AtlasEntry, bool) {
	e, ok := a.Entries[key]
	return e, ok
}

// Insert rasterizes and inserts a glyph into the atlas.
// The glyphImg must be a grayscale image of the rasterized glyph.
// Returns the atlas entry for the newly inserted glyph.
func (a *GlyphAtlas) Insert(key GlyphKey, glyphImg *image.Gray, bearing draw.Point, advance float32) AtlasEntry {
	bounds := glyphImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Check if we need to wrap to the next row.
	if a.cursorX+w > a.Width {
		a.cursorY += a.rowHeight + 1 // 1px padding between rows
		a.cursorX = 0
		a.rowHeight = 0
	}

	// Check if we need to grow the atlas.
	if a.cursorY+h > a.Height {
		a.grow()
	}

	// Copy glyph pixels into the atlas.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a.Image.SetGray(a.cursorX+x, a.cursorY+y, glyphImg.GrayAt(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

	entry := AtlasEntry{
		X: a.cursorX, Y: a.cursorY,
		W: w, H: h,
		BearingX: bearing.X,
		BearingY: bearing.Y,
		Advance:  advance,
	}
	a.Entries[key] = entry

	a.cursorX += w + 1 // 1px padding between glyphs
	if h > a.rowHeight {
		a.rowHeight = h
	}
	a.Dirty = true

	return entry
}

// grow doubles the atlas height and copies existing data.
func (a *GlyphAtlas) grow() {
	newH := a.Height * 2
	if newH < 256 {
		newH = 256
	}
	newImg := image.NewGray(image.Rect(0, 0, a.Width, newH))

	// Copy old atlas into new image.
	for y := 0; y < a.Height; y++ {
		copy(
			newImg.Pix[y*newImg.Stride:y*newImg.Stride+a.Width],
			a.Image.Pix[y*a.Image.Stride:y*a.Image.Stride+a.Width],
		)
	}

	a.Image = newImg
	a.Height = newH
}

// LookupOrInsert looks up a glyph in the atlas, inserting it via the shaper if missing.
func (a *GlyphAtlas) LookupOrInsert(key GlyphKey, shaper GlyphRasterizer, style draw.TextStyle) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	// Rasterize the glyph by its OpenType GlyphID (post-GSUB), so that
	// ligature glyphs are rendered correctly instead of the base rune.
	rg := shaper.RasterizeGlyph(key.GlyphID, style)
	if rg == nil {
		return AtlasEntry{}, false
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.Insert(key, rg.Image, bearing, rg.Advance)
	return entry, true
}

// LookupOrInsertWithFont looks up a glyph in the atlas, inserting it via the
// shaper with an explicit font if missing. Used for per-glyph fallback rendering.
func (a *GlyphAtlas) LookupOrInsertWithFont(key GlyphKey, shaper GlyphRasterizer, f *fonts.Font, style draw.TextStyle) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	rg := shaper.RasterizeGlyphWithFont(key.GlyphID, f, style)
	if rg == nil {
		return AtlasEntry{}, false
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.Insert(key, rg.Image, bearing, rg.Advance)
	return entry, true
}

// InsertMSDF inserts an MSDF glyph into the NRGBA atlas.
func (a *GlyphAtlas) InsertMSDF(key GlyphKey, glyphImg *image.NRGBA, bearing draw.Point, advance, pxRange float32) AtlasEntry {
	bounds := glyphImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if a.msdfCursorX+w > a.MSDFWidth {
		a.msdfCursorY += a.msdfRowHeight + 1
		a.msdfCursorX = 0
		a.msdfRowHeight = 0
	}

	if a.msdfCursorY+h > a.MSDFHeight {
		a.growMSDF()
	}

	// Copy glyph pixels into the MSDF atlas.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a.MSDFImage.SetNRGBA(a.msdfCursorX+x, a.msdfCursorY+y,
				glyphImg.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

	entry := AtlasEntry{
		X: a.msdfCursorX, Y: a.msdfCursorY,
		W: w, H: h,
		BearingX: bearing.X,
		BearingY: bearing.Y,
		Advance:  advance,
		PxRange:  pxRange,
	}
	a.Entries[key] = entry

	a.msdfCursorX += w + 1
	if h > a.msdfRowHeight {
		a.msdfRowHeight = h
	}
	a.MSDFDirty = true

	return entry
}

// growMSDF doubles the MSDF atlas height and copies existing data.
func (a *GlyphAtlas) growMSDF() {
	newH := a.MSDFHeight * 2
	if newH < 512 {
		newH = 512
	}
	newImg := image.NewNRGBA(image.Rect(0, 0, a.MSDFWidth, newH))

	for y := 0; y < a.MSDFHeight; y++ {
		srcOff := y * a.MSDFImage.Stride
		dstOff := y * newImg.Stride
		copy(newImg.Pix[dstOff:dstOff+a.MSDFWidth*4], a.MSDFImage.Pix[srcOff:srcOff+a.MSDFWidth*4])
	}

	a.MSDFImage = newImg
	a.MSDFHeight = newH
}

// LookupOrInsertMSDF looks up an MSDF glyph, rasterizing it via the shaper if missing.
// The atlas size is taken from key.SizePx (set via MSDFBucketSize).
func (a *GlyphAtlas) LookupOrInsertMSDF(key GlyphKey, shaper GlyphRasterizer, f *fonts.Font) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	atlasSize := int(key.SizePx)
	if atlasSize <= 0 {
		atlasSize = MSDFAtlasSize
	}
	rg := shaper.RasterizeMSDFGlyph(key.GlyphID, key.Rune, f, atlasSize, MSDFPxRange)
	if rg == nil {
		return AtlasEntry{}, false
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.InsertMSDF(key, rg.Image, bearing, rg.Advance, rg.PxRange)
	return entry, true
}

// SetMSDFNotify sets a callback that is invoked (at most once per batch) when
// background MSDF rasterization completes. The callback is called from a
// goroutine and should be safe to call concurrently (e.g. appLoop.Send).
func (a *GlyphAtlas) SetMSDFNotify(fn func()) {
	a.msdfNotify = fn
}

// RequestMSDFAsync queues an MSDF glyph for background rasterization.
// If the glyph is already cached or already in flight, this is a no-op.
// The caller should fall back to bitmap rendering for the current frame.
func (a *GlyphAtlas) RequestMSDFAsync(key GlyphKey, shaper GlyphRasterizer, f *fonts.Font) {
	if _, ok := a.Entries[key]; ok {
		return
	}
	if _, ok := a.msdfPending[key]; ok {
		return
	}
	a.msdfPending[key] = struct{}{}

	atlasSize := int(key.SizePx)
	if atlasSize <= 0 {
		atlasSize = MSDFAtlasSize
	}

	resultCh := a.msdfResults
	notify := a.msdfNotify
	signal := &a.msdfSignal

	go func() {
		rg := shaper.RasterizeMSDFGlyph(key.GlyphID, key.Rune, f, atlasSize, MSDFPxRange)
		if rg != nil {
			resultCh <- msdfResult{
				Key:     key,
				Image:   rg.Image,
				Bearing: draw.Pt(rg.BearingX, rg.BearingY),
				Advance: rg.Advance,
				PxRange: rg.PxRange,
			}
		} else {
			resultCh <- msdfResult{Key: key}
		}
		// Debounced notify: only send one repaint signal per batch.
		if notify != nil && signal.CompareAndSwap(false, true) {
			notify()
		}
	}()
}

// DrainMSDFResults inserts all completed MSDF glyphs into the atlas.
// Must be called on the main thread. Returns true if any glyphs were inserted.
func (a *GlyphAtlas) DrainMSDFResults() bool {
	anyInserted := false
	for {
		select {
		case r := <-a.msdfResults:
			delete(a.msdfPending, r.Key)
			if r.Image != nil {
				a.InsertMSDF(r.Key, r.Image, r.Bearing, r.Advance, r.PxRange)
				anyInserted = true
			}
		default:
			// Reset debounce flag so the next batch can signal again.
			a.msdfSignal.Store(false)
			return anyInserted
		}
	}
}

// InsertColor inserts a color emoji glyph into the color NRGBA atlas.
func (a *GlyphAtlas) InsertColor(key GlyphKey, glyphImg *image.NRGBA, bearing draw.Point, advance float32) AtlasEntry {
	bounds := glyphImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if a.colorCursorX+w > a.ColorWidth {
		a.colorCursorY += a.colorRowHeight + 1
		a.colorCursorX = 0
		a.colorRowHeight = 0
	}

	if a.colorCursorY+h > a.ColorHeight {
		a.growColor()
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a.ColorImage.SetNRGBA(a.colorCursorX+x, a.colorCursorY+y,
				glyphImg.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

	entry := AtlasEntry{
		X: a.colorCursorX, Y: a.colorCursorY,
		W: w, H: h,
		BearingX: bearing.X,
		BearingY: bearing.Y,
		Advance:  advance,
		PxRange:  -1, // sentinel: negative PxRange marks color emoji entries
	}
	a.Entries[key] = entry

	a.colorCursorX += w + 1
	if h > a.colorRowHeight {
		a.colorRowHeight = h
	}
	a.ColorDirty = true

	return entry
}

func (a *GlyphAtlas) growColor() {
	newH := a.ColorHeight * 2
	if newH < 512 {
		newH = 512
	}
	newImg := image.NewNRGBA(image.Rect(0, 0, a.ColorWidth, newH))

	for y := 0; y < a.ColorHeight; y++ {
		srcOff := y * a.ColorImage.Stride
		dstOff := y * newImg.Stride
		copy(newImg.Pix[dstOff:dstOff+a.ColorWidth*4], a.ColorImage.Pix[srcOff:srcOff+a.ColorWidth*4])
	}

	a.ColorImage = newImg
	a.ColorHeight = newH
}

// LookupOrInsertColor looks up a color emoji glyph, extracting it from CBDT if missing.
func (a *GlyphAtlas) LookupOrInsertColor(key GlyphKey, shaper GlyphRasterizer, f *fonts.Font, sizePx int) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	rg := shaper.RasterizeColorGlyph(key.GlyphID, f, sizePx)
	if rg == nil {
		return AtlasEntry{}, false
	}

	// Record PPEM once (all CBDT glyphs share the same strike size).
	if a.ColorPPEM == 0 && rg.PPEM > 0 {
		a.ColorPPEM = rg.PPEM
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.InsertColor(key, rg.Image, bearing, rg.Advance)
	return entry, true
}
