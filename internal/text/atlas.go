package text

import (
	"image"

	"github.com/timzifer/lux/draw"
)

// GlyphKey uniquely identifies a rasterized glyph in the atlas.
type GlyphKey struct {
	FontID uint64
	Rune   rune
	SizePx uint16
}

// AtlasEntry describes the location and metrics of a glyph in the atlas.
type AtlasEntry struct {
	X, Y, W, H          int     // position and size in the atlas image
	BearingX, BearingY   float32 // glyph bearing (offset from cursor)
	Advance              float32 // horizontal advance
}

// GlyphAtlas caches rasterized glyphs in a grayscale texture atlas.
// The atlas uses simple row-based packing: glyphs are placed left to right,
// wrapping to the next row when the current row is full.
type GlyphAtlas struct {
	Image   *image.Gray
	Entries map[GlyphKey]AtlasEntry
	Width   int
	Height  int
	Dirty   bool // set when the atlas image has been modified since last GPU upload

	cursorX   int
	cursorY   int
	rowHeight int
}

// NewGlyphAtlas creates an atlas with the given initial dimensions.
func NewGlyphAtlas(w, h int) *GlyphAtlas {
	return &GlyphAtlas{
		Image:   image.NewGray(image.Rect(0, 0, w, h)),
		Entries: make(map[GlyphKey]AtlasEntry),
		Width:   w,
		Height:  h,
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
func (a *GlyphAtlas) LookupOrInsert(key GlyphKey, shaper *SfntShaper, style draw.TextStyle) (AtlasEntry, bool) {
	if entry, ok := a.Lookup(key); ok {
		return entry, true
	}

	// Rasterize the glyph. The returned bearings are pixel-aligned
	// (using Floor/Ceil) to match the rasterized bitmap exactly.
	rg := shaper.RasterizeGlyph(key.Rune, style)
	if rg == nil {
		return AtlasEntry{}, false
	}

	bearing := draw.Pt(rg.BearingX, rg.BearingY)
	entry := a.Insert(key, rg.Image, bearing, rg.Advance)
	return entry, true
}
