package text

import (
	"image"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
)

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

// MSDFAtlasSize is the fixed pixel size at which MSDF glyphs are rasterized.
const MSDFAtlasSize = 32

// MSDFPxRange is the SDF distance field range in pixels.
const MSDFPxRange = 4.0

// MSDFMinSize is the minimum requested text size (in px) for using MSDF.
// Below this threshold the hinted bitmap rasterizer is used instead,
// which produces sharper results at small sizes thanks to pixel-grid hinting.
const MSDFMinSize = 24

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
}

// NewGlyphAtlas creates an atlas with the given initial dimensions.
func NewGlyphAtlas(w, h int) *GlyphAtlas {
	return &GlyphAtlas{
		Image:     image.NewGray(image.Rect(0, 0, w, h)),
		Entries:   make(map[GlyphKey]AtlasEntry),
		Width:     w,
		Height:    h,
		MSDFImage:  image.NewNRGBA(image.Rect(0, 0, 512, 512)),
		MSDFWidth:  512,
		MSDFHeight: 512,
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
func (a *GlyphAtlas) LookupOrInsertMSDF(key GlyphKey, shaper GlyphRasterizer, f *fonts.Font) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	rg := shaper.RasterizeMSDFGlyph(key.GlyphID, key.Rune, f, MSDFAtlasSize, MSDFPxRange)
	if rg == nil {
		return AtlasEntry{}, false
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.InsertMSDF(key, rg.Image, bearing, rg.Advance, rg.PxRange)
	return entry, true
}
