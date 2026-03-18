// Package render provides the scene-building Canvas implementation.
//
// SceneCanvas implements draw.Canvas and accumulates DrawRects and
// DrawGlyphs into a draw.Scene that the GPU backend can consume.
package render

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/internal/text"
)

// SceneCanvas implements draw.Canvas by accumulating draw commands
// into a flat draw.Scene.
type SceneCanvas struct {
	width       int
	height      int
	scene       draw.Scene
	shaper      text.Shaper
	atlas       *text.GlyphAtlas
	clips       []draw.Rect // clip rect stack
	overlayMode bool        // when true, draw commands go to overlay lists
}

// SetOverlayMode switches between main and overlay draw lists.
// Overlay content is rendered after all main content, ensuring
// dropdowns/tooltips fully cover underlying text.
func (c *SceneCanvas) SetOverlayMode(on bool) { c.overlayMode = on }

// CanvasOption configures a SceneCanvas.
type CanvasOption func(*SceneCanvas)

// WithShaper sets the text shaper. Default is BitmapShaper.
func WithShaper(s text.Shaper) CanvasOption {
	return func(c *SceneCanvas) { c.shaper = s }
}

// WithAtlas sets the glyph atlas for textured glyph rendering.
func WithAtlas(a *text.GlyphAtlas) CanvasOption {
	return func(c *SceneCanvas) { c.atlas = a }
}

// NewSceneCanvas creates a canvas for the given framebuffer size.
// Without options it defaults to the bitmap shaper for backward compatibility.
func NewSceneCanvas(width, height int, opts ...CanvasOption) *SceneCanvas {
	c := &SceneCanvas{
		width:  width,
		height: height,
		shaper: text.BitmapShaper{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Scene returns the accumulated draw commands.
func (c *SceneCanvas) Scene() draw.Scene { return c.scene }

// ── Primitives ───────────────────────────────────────────────────

func (c *SceneCanvas) FillRect(r draw.Rect, paint draw.Paint) {
	if c.isClipped(r.X, r.Y, r.W, r.H) {
		return
	}
	// Intersect with clip rect to clamp visible portion.
	x, y, w, h := r.X, r.Y, r.W, r.H
	if len(c.clips) > 0 {
		clip := c.clips[len(c.clips)-1]
		if x < clip.X {
			w -= clip.X - x
			x = clip.X
		}
		if y < clip.Y {
			h -= clip.Y - y
			y = clip.Y
		}
		if x+w > clip.X+clip.W {
			w = clip.X + clip.W - x
		}
		if y+h > clip.Y+clip.H {
			h = clip.Y + clip.H - y
		}
		if w <= 0 || h <= 0 {
			return
		}
	}
	rect := draw.DrawRect{X: int(x), Y: int(y), W: int(w), H: int(h), Color: paint.Color}
	if c.overlayMode {
		c.scene.OverlayRects = append(c.scene.OverlayRects, rect)
	} else {
		c.scene.Rects = append(c.scene.Rects, rect)
	}
}

func (c *SceneCanvas) FillRoundRect(r draw.Rect, radius float32, paint draw.Paint) {
	if c.isClipped(r.X, r.Y, r.W, r.H) {
		return
	}
	x, y, w, h := r.X, r.Y, r.W, r.H
	if len(c.clips) > 0 {
		clip := c.clips[len(c.clips)-1]
		if x < clip.X {
			w -= clip.X - x
			x = clip.X
		}
		if y < clip.Y {
			h -= clip.Y - y
			y = clip.Y
		}
		if x+w > clip.X+clip.W {
			w = clip.X + clip.W - x
		}
		if y+h > clip.Y+clip.H {
			h = clip.Y + clip.H - y
		}
		if w <= 0 || h <= 0 {
			return
		}
	}
	rect := draw.DrawRect{X: int(x), Y: int(y), W: int(w), H: int(h), Color: paint.Color, Radius: radius}
	if c.overlayMode {
		c.scene.OverlayRects = append(c.scene.OverlayRects, rect)
	} else {
		c.scene.Rects = append(c.scene.Rects, rect)
	}
}

func (c *SceneCanvas) FillRoundRectCorners(r draw.Rect, _ draw.CornerRadii, paint draw.Paint) {
	c.FillRect(r, paint)
}

func (c *SceneCanvas) FillEllipse(r draw.Rect, paint draw.Paint) {
	// Use a rounded rect with radius = min(w,h)/2 to produce a circle/ellipse
	// via the SDF shader.
	radius := r.W
	if r.H < radius {
		radius = r.H
	}
	c.FillRoundRect(r, radius/2, paint)
}

func (c *SceneCanvas) StrokeRect(r draw.Rect, stroke draw.Stroke) {
	w := stroke.Width
	c.FillRect(draw.R(r.X, r.Y, r.W, w), draw.SolidPaint(stroke.Paint.Color))
	c.FillRect(draw.R(r.X, r.Y+r.H-w, r.W, w), draw.SolidPaint(stroke.Paint.Color))
	c.FillRect(draw.R(r.X, r.Y+w, w, r.H-2*w), draw.SolidPaint(stroke.Paint.Color))
	c.FillRect(draw.R(r.X+r.W-w, r.Y+w, w, r.H-2*w), draw.SolidPaint(stroke.Paint.Color))
}

func (c *SceneCanvas) StrokeRoundRect(r draw.Rect, _ float32, stroke draw.Stroke) {
	c.StrokeRect(r, stroke)
}

func (c *SceneCanvas) StrokeRoundRectCorners(r draw.Rect, _ draw.CornerRadii, stroke draw.Stroke) {
	c.StrokeRect(r, stroke)
}

func (c *SceneCanvas) StrokeEllipse(r draw.Rect, stroke draw.Stroke) {
	c.StrokeRect(r, stroke)
}

func (c *SceneCanvas) StrokeLine(a, b draw.Point, stroke draw.Stroke) {
	// M2: lines not needed for text + button.
}

// ── Paths ────────────────────────────────────────────────────────

func (c *SceneCanvas) FillPath(_ draw.Path, _ draw.Paint)   {}
func (c *SceneCanvas) StrokePath(_ draw.Path, _ draw.Stroke) {}

// ── Text ─────────────────────────────────────────────────────────

func (c *SceneCanvas) DrawText(txt string, origin draw.Point, style draw.TextStyle, color draw.Color) {
	if len(txt) == 0 {
		return
	}

	// If we have an atlas and an sfnt shaper, use the textured glyph path.
	if c.atlas != nil {
		if sfntShaper, ok := c.shaper.(*text.SfntShaper); ok {
			c.drawTextTextured(txt, origin, style, color, sfntShaper)
			return
		}
	}

	// Legacy bitmap path.
	if c.isClipped(origin.X, origin.Y, style.Size*float32(len(txt)), style.Size) {
		return
	}
	scale := text.BitmapScale(style.Size)
	glyph := draw.DrawGlyph{X: int(origin.X), Y: int(origin.Y), Scale: scale, Text: txt, Color: color}
	if c.overlayMode {
		c.scene.OverlayGlyphs = append(c.scene.OverlayGlyphs, glyph)
	} else {
		c.scene.Glyphs = append(c.scene.Glyphs, glyph)
	}
}

// drawTextTextured shapes text and emits TexturedGlyphs from the atlas.
// origin.Y is the top-left of the text bounding box (not the baseline).
func (c *SceneCanvas) drawTextTextured(txt string, origin draw.Point, style draw.TextStyle, color draw.Color, shaper *text.SfntShaper) {
	shaped := shaper.Shape(txt, style)
	cursorX := origin.X
	sizePx := uint16(text.DpToPixels(style.Size))

	f := shaper.ResolveFont(style)
	if f == nil {
		return
	}
	fontID := f.ID()

	// Compute the font ascent so we can convert the top-left origin to
	// a baseline for glyph placement: baseline = origin.Y + ascent.
	// Snap baseline to integer pixels to prevent glyphs from jumping
	// between sub-pixel rows (especially visible on macOS / Retina).
	metrics := shaper.Measure(txt, style)
	baseline := float32(math.Round(float64(origin.Y + metrics.Ascent)))

	for _, sg := range shaped {
		if sg.Rune == ' ' {
			cursorX += sg.Advance
			continue
		}

		key := text.GlyphKey{FontID: fontID, Rune: sg.Rune, SizePx: sizePx}
		entry, ok := c.atlas.LookupOrInsert(key, shaper, style)
		if !ok {
			cursorX += sg.Advance
			continue
		}

		dstX := float32(math.Round(float64(cursorX + entry.BearingX)))
		dstY := float32(math.Round(float64(baseline - entry.BearingY)))
		if c.isClipped(dstX, dstY, float32(entry.W), float32(entry.H)) {
			cursorX += sg.Advance
			continue
		}
		tg := draw.TexturedGlyph{
			DstX: dstX,
			DstY: dstY,
			DstW: float32(entry.W),
			DstH: float32(entry.H),
			SrcX: entry.X, SrcY: entry.Y,
			SrcW: entry.W, SrcH: entry.H,
			Color: color,
		}
		// Clamp glyph quad to clip rect so glyphs at scroll
		// boundaries don't bleed past the viewport.
		if len(c.clips) > 0 {
			clip := c.clips[len(c.clips)-1]
			if tg.DstX < clip.X {
				d := clip.X - tg.DstX
				tg.SrcX += int(d)
				tg.SrcW -= int(d)
				tg.DstW -= d
				tg.DstX = clip.X
			}
			if tg.DstY < clip.Y {
				d := clip.Y - tg.DstY
				tg.SrcY += int(d)
				tg.SrcH -= int(d)
				tg.DstH -= d
				tg.DstY = clip.Y
			}
			if tg.DstX+tg.DstW > clip.X+clip.W {
				tg.DstW = clip.X + clip.W - tg.DstX
				tg.SrcW = int(tg.DstW)
			}
			if tg.DstY+tg.DstH > clip.Y+clip.H {
				tg.DstH = clip.Y + clip.H - tg.DstY
				tg.SrcH = int(tg.DstH)
			}
			if tg.DstW <= 0 || tg.DstH <= 0 {
				cursorX += sg.Advance
				continue
			}
		}
		if c.overlayMode {
			c.scene.OverlayTexturedGlyphs = append(c.scene.OverlayTexturedGlyphs, tg)
		} else {
			c.scene.TexturedGlyphs = append(c.scene.TexturedGlyphs, tg)
		}

		cursorX += sg.Advance
	}
}

func (c *SceneCanvas) MeasureText(txt string, style draw.TextStyle) draw.TextMetrics {
	return c.shaper.Measure(txt, style)
}

// ── Images ───────────────────────────────────────────────────────

func (c *SceneCanvas) DrawImage(_ draw.ImageID, _ draw.Rect, _ draw.ImageOptions) {}

// ── Shadows ──────────────────────────────────────────────────────

func (c *SceneCanvas) DrawShadow(_ draw.Rect, _ draw.Shadow) {}

// ── Clipping & Transform ─────────────────────────────────────────

func (c *SceneCanvas) PushClip(r draw.Rect) {
	c.clips = append(c.clips, r)
}

func (c *SceneCanvas) PopClip() {
	if len(c.clips) > 0 {
		c.clips = c.clips[:len(c.clips)-1]
	}
}

// clipRect returns the current clip rect, or the full canvas if none.
func (c *SceneCanvas) clipRect() draw.Rect {
	if len(c.clips) == 0 {
		return draw.R(0, 0, float32(c.width), float32(c.height))
	}
	return c.clips[len(c.clips)-1]
}

// isClipped returns true if the rect is fully outside the current clip.
func (c *SceneCanvas) isClipped(x, y, w, h float32) bool {
	if len(c.clips) == 0 {
		return false
	}
	clip := c.clips[len(c.clips)-1]
	return x+w <= clip.X || x >= clip.X+clip.W ||
		y+h <= clip.Y || y >= clip.Y+clip.H
}
func (c *SceneCanvas) PushTransform(_ draw.Transform) {}
func (c *SceneCanvas) PopTransform()                  {}
func (c *SceneCanvas) PushOffset(_, _ float32)        {}

// ── Effects ──────────────────────────────────────────────────────

func (c *SceneCanvas) PushOpacity(_ float32) {}
func (c *SceneCanvas) PopOpacity()           {}

// ── State ────────────────────────────────────────────────────────

func (c *SceneCanvas) Bounds() draw.Rect {
	return draw.R(0, 0, float32(c.width), float32(c.height))
}

func (c *SceneCanvas) DPR() float32 { return 1.0 }
func (c *SceneCanvas) Save()        {}
func (c *SceneCanvas) Restore()     {}

// ── Compile-time check ───────────────────────────────────────────

var _ draw.Canvas = (*SceneCanvas)(nil)

// ── Bitmap helpers (for GPU renderers that need glyphs) ──────────

// RenderBitmapGlyph calls fn for each "on" pixel of the glyph at
// the given position and scale.
func RenderBitmapGlyph(txt string, x, y, scale int, fn func(px, py, w, h int)) {
	cursorX := x
	for _, raw := range txt {
		if raw == ' ' {
			cursorX += fonts.BitmapCharWidth * scale
			continue
		}
		glyph := fonts.BitmapGlyph(raw)
		for row, bits := range glyph {
			for col := 0; col < len(bits); col++ {
				if bits[col] != '1' {
					continue
				}
				fn(cursorX+(col*scale), y+(row*scale), scale, scale)
			}
		}
		cursorX += fonts.BitmapCharWidth * scale
	}
}
