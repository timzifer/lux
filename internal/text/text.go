// Package text provides text measurement and shaping (RFC §16).
//
// For M2 this uses the embedded 5×7 bitmap font.
// Later milestones will integrate go-text/typesetting for full
// OpenType shaping and Unicode support.
package text

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
)

// ShapedGlyph describes a single positioned glyph produced by shaping.
type ShapedGlyph struct {
	Rune     rune
	Advance  float32 // horizontal advance in dp
	BearingX float32 // left-side bearing in dp
	BearingY float32 // top bearing from baseline in dp
	Width    float32 // glyph bounding box width in dp
	Height   float32 // glyph bounding box height in dp
}

// Shaper shapes a run of text into positioned glyphs (RFC-003 §3.3).
type Shaper interface {
	Measure(text string, style draw.TextStyle) draw.TextMetrics
	Shape(text string, style draw.TextStyle) []ShapedGlyph
}

// BitmapShaper implements Shaper using the embedded 5×7 bitmap font.
type BitmapShaper struct{}

// Measure returns the text dimensions at the given style's size.
func (BitmapShaper) Measure(text string, style draw.TextStyle) draw.TextMetrics {
	scale := bitmapScale(style.Size)
	runes := []rune(text)
	if len(runes) == 0 {
		return draw.TextMetrics{}
	}
	w := float32(len(runes)) * float32(fonts.BitmapCharWidth) * float32(scale)
	h := float32(fonts.BitmapCharHeight) * float32(scale)
	return draw.TextMetrics{
		Width:   w,
		Ascent:  h,
		Descent: 0,
		Leading: 0,
	}
}

// bitmapScale derives the integer pixel scale from a dp size.
// The bitmap font is 7px tall, so scale = round(size / 7).
func bitmapScale(size float32) int {
	s := int(size / float32(fonts.BitmapCharHeight))
	if s < 1 {
		s = 1
	}
	return s
}

// Shape returns one ShapedGlyph per rune using bitmap metrics.
func (BitmapShaper) Shape(text string, style draw.TextStyle) []ShapedGlyph {
	scale := bitmapScale(style.Size)
	adv := float32(fonts.BitmapCharWidth) * float32(scale)
	h := float32(fonts.BitmapCharHeight) * float32(scale)
	runes := []rune(text)
	out := make([]ShapedGlyph, len(runes))
	for i, r := range runes {
		out[i] = ShapedGlyph{
			Rune:     r,
			Advance:  adv,
			BearingX: 0,
			BearingY: h,
			Width:    adv,
			Height:   h,
		}
	}
	return out
}

// BitmapScale is exported for the renderer.
func BitmapScale(size float32) int { return bitmapScale(size) }
