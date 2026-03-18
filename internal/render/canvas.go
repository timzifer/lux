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
	shaper text.BitmapShaper
}

// NewSceneCanvas creates a canvas for the given framebuffer size.
func NewSceneCanvas(width, height int) *SceneCanvas {
	return &SceneCanvas{width: width, height: height}
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
	scale := text.BitmapScale(style.Size)
	c.scene.Glyphs = append(c.scene.Glyphs, draw.DrawGlyph{
		X:     int(origin.X),
		Y:     int(origin.Y),
		Scale: scale,
		Text:  txt,
		Color: color,
	})
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
