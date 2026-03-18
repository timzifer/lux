// Package render provides the scene-building Canvas implementation.
//
// SceneCanvas implements draw.Canvas and accumulates DrawRects and
// DrawGlyphs into a draw.Scene that the GPU backend can consume.
package render

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/internal/text"
)

// SceneCanvas implements draw.Canvas by accumulating draw commands
// into a flat draw.Scene.
type SceneCanvas struct {
	width  int
	height int
	scene  draw.Scene
	shaper text.Shaper
	atlas  *text.GlyphAtlas
}

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
	c.scene.Rects = append(c.scene.Rects, draw.DrawRect{
		X: int(r.X), Y: int(r.Y), W: int(r.W), H: int(r.H),
		Color: paint.Color,
	})
}

func (c *SceneCanvas) FillRoundRect(r draw.Rect, radius float32, paint draw.Paint) {
	// M2: rounded rects rendered as regular rects.
	c.FillRect(r, paint)
}

func (c *SceneCanvas) FillRoundRectCorners(r draw.Rect, _ draw.CornerRadii, paint draw.Paint) {
	c.FillRect(r, paint)
}

func (c *SceneCanvas) FillEllipse(r draw.Rect, paint draw.Paint) {
	c.FillRect(r, paint)
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
	scale := text.BitmapScale(style.Size)
	c.scene.Glyphs = append(c.scene.Glyphs, draw.DrawGlyph{
		X:     int(origin.X),
		Y:     int(origin.Y),
		Scale: scale,
		Text:  txt,
		Color: color,
	})
}

// drawTextTextured shapes text and emits TexturedGlyphs from the atlas.
func (c *SceneCanvas) drawTextTextured(txt string, origin draw.Point, style draw.TextStyle, color draw.Color, shaper *text.SfntShaper) {
	shaped := shaper.Shape(txt, style)
	cursorX := origin.X
	sizePx := uint16(text.DpToPixels(style.Size))

	f := fonts.DefaultFont()
	if f == nil {
		return
	}
	fontID := f.ID()

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

		c.scene.TexturedGlyphs = append(c.scene.TexturedGlyphs, draw.TexturedGlyph{
			DstX: cursorX + entry.BearingX,
			DstY: origin.Y - entry.BearingY,
			DstW: float32(entry.W),
			DstH: float32(entry.H),
			SrcX: entry.X, SrcY: entry.Y,
			SrcW: entry.W, SrcH: entry.H,
			Color: color,
		})

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

func (c *SceneCanvas) PushClip(_ draw.Rect)         {}
func (c *SceneCanvas) PopClip()                      {}
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
