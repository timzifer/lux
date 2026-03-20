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

	// Scissor clip-batch tracking.
	lastClip    draw.Rect
	lastClipSet bool

	blurStack    []float32
	opacityStack []float32
}

// SetOverlayMode switches between main and overlay draw lists.
// Overlay content is rendered after all main content, ensuring
// dropdowns/tooltips fully cover underlying text.
func (c *SceneCanvas) SetOverlayMode(on bool) {
	c.overlayMode = on
	c.lastClipSet = false // reset clip tracking on mode switch
}

// emitClipIfChanged emits a new ClipBatch if the current clip rect
// differs from the last emitted one. This allows the GPU renderer to
// set scissor rects per batch.
func (c *SceneCanvas) emitClipIfChanged() {
	cur := c.clipRect()
	if c.lastClipSet && cur == c.lastClip {
		return
	}
	fullViewport := len(c.clips) == 0
	batch := draw.ClipBatch{
		Clip:         cur,
		RectIdx:      len(c.scene.Rects),
		TextIdx:      len(c.scene.TexturedGlyphs),
		MSDFIdx:      len(c.scene.MSDFGlyphs),
		GradientIdx:  len(c.scene.GradientRects),
		ShadowIdx:    len(c.scene.ShadowRects),
		FullViewport: fullViewport,
	}
	if c.overlayMode {
		batch.RectIdx = len(c.scene.OverlayRects)
		batch.TextIdx = len(c.scene.OverlayTexturedGlyphs)
		batch.MSDFIdx = len(c.scene.OverlayMSDFGlyphs)
		batch.GradientIdx = len(c.scene.OverlayGradientRects)
		batch.ShadowIdx = len(c.scene.OverlayShadowRects)
		c.scene.OverlayClipBatches = append(c.scene.OverlayClipBatches, batch)
	} else {
		c.scene.ClipBatches = append(c.scene.ClipBatches, batch)
	}
	c.lastClip = cur
	c.lastClipSet = true
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
	if c.isClipped(r.X, r.Y, r.W, r.H) {
		return
	}
	// Route gradient paints to the gradient pipeline.
	if paint.Kind == draw.PaintLinearGradient || paint.Kind == draw.PaintRadialGradient {
		c.appendGradientRect(r, 0, paint)
		return
	}
	c.emitClipIfChanged()
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
	color := paint.FallbackColor()
	color.A *= c.effectiveOpacity()
	if color.A < 0.001 {
		return
	}
	rect := draw.DrawRect{X: int(x), Y: int(y), W: int(w), H: int(h), Color: color}
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
	// Route gradient paints to the gradient pipeline.
	if paint.Kind == draw.PaintLinearGradient || paint.Kind == draw.PaintRadialGradient {
		c.appendGradientRect(r, radius, paint)
		return
	}
	c.emitClipIfChanged()
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
	color := paint.FallbackColor()
	color.A *= c.effectiveOpacity()
	if color.A < 0.001 {
		return
	}
	rect := draw.DrawRect{X: int(x), Y: int(y), W: int(w), H: int(h), Color: color, Radius: radius}
	if c.overlayMode {
		c.scene.OverlayRects = append(c.scene.OverlayRects, rect)
	} else {
		c.scene.Rects = append(c.scene.Rects, rect)
	}
}

// appendGradientRect appends a gradient-filled rect to the scene.
func (c *SceneCanvas) appendGradientRect(r draw.Rect, radius float32, paint draw.Paint) {
	c.emitClipIfChanged()
	gr := draw.DrawGradientRect{
		X: int(r.X), Y: int(r.Y), W: int(r.W), H: int(r.H),
		Radius: radius,
		Kind:   paint.Kind,
	}
	// Gradient coordinates in Paint are element-local; offset to screen space.
	ox, oy := r.X, r.Y
	switch paint.Kind {
	case draw.PaintLinearGradient:
		if paint.Linear != nil {
			gr.StartX = paint.Linear.Start.X + ox
			gr.StartY = paint.Linear.Start.Y + oy
			gr.EndX = paint.Linear.End.X + ox
			gr.EndY = paint.Linear.End.Y + oy
			for i, s := range paint.Linear.Stops {
				if i >= 8 {
					break
				}
				gr.Stops[i] = s
				gr.StopCount = i + 1
			}
		}
	case draw.PaintRadialGradient:
		if paint.Radial != nil {
			gr.CenterX = paint.Radial.Center.X + ox
			gr.CenterY = paint.Radial.Center.Y + oy
			gr.GradRadius = paint.Radial.Radius
			for i, s := range paint.Radial.Stops {
				if i >= 8 {
					break
				}
				gr.Stops[i] = s
				gr.StopCount = i + 1
			}
		}
	}
	opacity := c.effectiveOpacity()
	if opacity < 1.0 {
		for i := 0; i < gr.StopCount; i++ {
			gr.Stops[i].Color.A *= opacity
		}
	}
	if c.overlayMode {
		c.scene.OverlayGradientRects = append(c.scene.OverlayGradientRects, gr)
	} else {
		c.scene.GradientRects = append(c.scene.GradientRects, gr)
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

	// If we have an atlas and a glyph rasterizer, use the textured glyph path.
	if c.atlas != nil {
		if rasterizer, ok := c.shaper.(text.GlyphRasterizer); ok {
			c.drawTextTextured(txt, origin, style, color, rasterizer)
			return
		}
	}

	// Legacy bitmap path — strict clip-cull (no per-pixel clipping available).
	approxW := style.Size * float32(len(txt))
	if c.isClipped(origin.X, origin.Y, approxW, style.Size) {
		return
	}
	// Skip glyphs whose top edge is at or below the clip bottom.
	// Legacy glyphs can't be pixel-clipped, so we cull runs that start
	// outside the visible region. Runs that partially overlap are kept
	// (minor bleed at the boundary is acceptable for the legacy path).
	if len(c.clips) > 0 {
		clip := c.clips[len(c.clips)-1]
		if origin.Y >= clip.Y+clip.H || origin.Y+style.Size <= clip.Y {
			return
		}
	}
	scale := text.BitmapScale(style.Size)
	glyph := draw.DrawGlyph{X: int(origin.X), Y: int(origin.Y), Scale: scale, Text: txt, Color: color}
	if c.overlayMode {
		c.scene.OverlayGlyphs = append(c.scene.OverlayGlyphs, glyph)
	} else {
		c.scene.Glyphs = append(c.scene.Glyphs, glyph)
	}
}

// drawTextTextured shapes text and emits TexturedGlyphs (or MSDFGlyphs) from the atlas.
// origin.Y is the top-left of the text bounding box (not the baseline).
func (c *SceneCanvas) drawTextTextured(txt string, origin draw.Point, style draw.TextStyle, color draw.Color, shaper text.GlyphRasterizer) {
	color.A *= c.effectiveOpacity()
	if color.A < 0.001 {
		return
	}
	c.emitClipIfChanged()
	shaped := shaper.Shape(txt, style)
	cursorX := origin.X

	f := shaper.ResolveFont(style)
	if f == nil {
		return
	}
	fontID := f.ID()

	sizePx := uint16(text.DpToPixels(style.Size))

	// Use MSDF only for large sizes where the scalable SDF sharpness
	// advantage outweighs the lack of hinting. Below the threshold,
	// the hinted bitmap rasterizer produces crisper results.
	useMSDF := f.SfntFont() != nil && sizePx >= text.MSDFMinSize && !style.Raster

	// Compute the font ascent so we can convert the top-left origin to
	// a baseline for glyph placement: baseline = origin.Y + ascent.
	// Snap baseline to integer pixels to prevent glyphs from jumping
	// between sub-pixel rows (especially visible on macOS / Retina).
	metrics := shaper.Measure(txt, style)
	baseline := float32(math.Round(float64(origin.Y + metrics.Ascent)))

	// Scale factor for MSDF glyphs: atlas is rendered at MSDFAtlasSize,
	// we scale to the requested size.
	msdfScale := float32(sizePx) / float32(text.MSDFAtlasSize)

	for _, sg := range shaped {
		if sg.Rune == ' ' {
			cursorX += sg.Advance
			continue
		}

		var entry text.AtlasEntry
		var ok bool

		if useMSDF {
			key := text.GlyphKey{FontID: fontID, Rune: sg.Rune, SizePx: uint16(text.MSDFAtlasSize), MSDF: true}
			entry, ok = c.atlas.LookupOrInsertMSDF(key, shaper, f)
		} else {
			key := text.GlyphKey{FontID: fontID, Rune: sg.Rune, SizePx: sizePx}
			entry, ok = c.atlas.LookupOrInsert(key, shaper, style)
		}
		if !ok {
			cursorX += sg.Advance
			continue
		}

		var dstX, dstY, dstW, dstH float32
		if useMSDF {
			dstW = float32(entry.W) * msdfScale
			dstH = float32(entry.H) * msdfScale
			dstX = float32(math.Round(float64(cursorX + entry.BearingX*msdfScale)))
			dstY = float32(math.Round(float64(baseline - entry.BearingY*msdfScale)))
		} else {
			dstW = float32(entry.W)
			dstH = float32(entry.H)
			dstX = float32(math.Round(float64(cursorX + entry.BearingX)))
			dstY = float32(math.Round(float64(baseline - entry.BearingY)))
		}

		if c.isClipped(dstX, dstY, dstW, dstH) {
			cursorX += sg.Advance
			continue
		}
		tg := draw.TexturedGlyph{
			DstX: dstX,
			DstY: dstY,
			DstW: dstW,
			DstH: dstH,
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
				if useMSDF {
					srcD := d / msdfScale
					tg.SrcX += int(srcD)
					tg.SrcW -= int(srcD)
				} else {
					tg.SrcX += int(d)
					tg.SrcW -= int(d)
				}
				tg.DstW -= d
				tg.DstX = clip.X
			}
			if tg.DstY < clip.Y {
				d := clip.Y - tg.DstY
				if useMSDF {
					srcD := d / msdfScale
					tg.SrcY += int(srcD)
					tg.SrcH -= int(srcD)
				} else {
					tg.SrcY += int(d)
					tg.SrcH -= int(d)
				}
				tg.DstH -= d
				tg.DstY = clip.Y
			}
			if tg.DstX+tg.DstW > clip.X+clip.W {
				tg.DstW = clip.X + clip.W - tg.DstX
				if !useMSDF {
					tg.SrcW = int(tg.DstW)
				} else {
					tg.SrcW = int(tg.DstW / msdfScale)
				}
			}
			if tg.DstY+tg.DstH > clip.Y+clip.H {
				tg.DstH = clip.Y + clip.H - tg.DstY
				if !useMSDF {
					tg.SrcH = int(tg.DstH)
				} else {
					tg.SrcH = int(tg.DstH / msdfScale)
				}
			}
			if tg.DstW <= 0 || tg.DstH <= 0 {
				cursorX += sg.Advance
				continue
			}
		}
		if useMSDF {
			if c.overlayMode {
				c.scene.OverlayMSDFGlyphs = append(c.scene.OverlayMSDFGlyphs, tg)
			} else {
				c.scene.MSDFGlyphs = append(c.scene.MSDFGlyphs, tg)
			}
		} else {
			if c.overlayMode {
				c.scene.OverlayTexturedGlyphs = append(c.scene.OverlayTexturedGlyphs, tg)
			} else {
				c.scene.TexturedGlyphs = append(c.scene.TexturedGlyphs, tg)
			}
		}

		cursorX += sg.Advance
	}
}

func (c *SceneCanvas) MeasureText(txt string, style draw.TextStyle) draw.TextMetrics {
	return c.shaper.Measure(txt, style)
}

// ── Text Layout ─────────────────────────────────────────────────

func (c *SceneCanvas) DrawTextLayout(layout draw.TextLayout, origin draw.Point, color draw.Color) {
	// Fallback: delegate to DrawText (no line breaking or alignment yet).
	c.DrawText(layout.Text, origin, layout.Style, color)
}

// ── Images ───────────────────────────────────────────────────────

func (c *SceneCanvas) DrawImage(_ draw.ImageID, _ draw.Rect, _ draw.ImageOptions)       {}
func (c *SceneCanvas) DrawImageSlice(_ draw.ImageSlice, _ draw.Rect, _ draw.ImageOptions) {}
func (c *SceneCanvas) DrawTexture(tex draw.TextureID, dst draw.Rect) {
	if c.isClipped(dst.X, dst.Y, dst.W, dst.H) {
		return
	}
	c.emitClipIfChanged()
	c.scene.Surfaces = append(c.scene.Surfaces, draw.DrawSurface{
		X: int(dst.X), Y: int(dst.Y), W: int(dst.W), H: int(dst.H),
		TextureID: tex,
	})
}

// ── Shadows ──────────────────────────────────────────────────────

func (c *SceneCanvas) DrawShadow(r draw.Rect, s draw.Shadow) {
	// Compute expanded bounds: shadow extends by offset + spread + blur.
	expand := s.SpreadRadius + s.BlurRadius
	x := r.X + s.OffsetX - expand
	y := r.Y + s.OffsetY - expand
	w := r.W + 2*expand
	h := r.H + 2*expand
	if c.isClipped(x, y, w, h) {
		return
	}
	color := s.Color
	color.A *= c.effectiveOpacity()
	if color.A < 0.001 {
		return
	}
	c.emitClipIfChanged()
	sr := draw.DrawShadowRect{
		X: int(x), Y: int(y), W: int(w), H: int(h),
		Color:      color,
		Radius:     s.Radius,
		BlurRadius: s.BlurRadius,
	}
	if c.overlayMode {
		c.scene.OverlayShadowRects = append(c.scene.OverlayShadowRects, sr)
	} else {
		c.scene.ShadowRects = append(c.scene.ShadowRects, sr)
	}
}

// ── Clipping & Transform ─────────────────────────────────────────

func (c *SceneCanvas) PushClip(r draw.Rect) {
	// Intersect with the current clip so nested clips accumulate correctly.
	if len(c.clips) > 0 {
		parent := c.clips[len(c.clips)-1]
		r = intersectRect(parent, r)
	}
	c.clips = append(c.clips, r)
}

func (c *SceneCanvas) PopClip() {
	if len(c.clips) > 0 {
		c.clips = c.clips[:len(c.clips)-1]
	}
}

// intersectRect returns the intersection of two rects (zero-area if disjoint).
func intersectRect(a, b draw.Rect) draw.Rect {
	x := max32(a.X, b.X)
	y := max32(a.Y, b.Y)
	x2 := min32(a.X+a.W, b.X+b.W)
	y2 := min32(a.Y+a.H, b.Y+b.H)
	w := x2 - x
	h := y2 - y
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return draw.R(x, y, w, h)
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
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
// PushClipRoundRect pushes a rounded-rect clip (bounding-box approximation).
func (c *SceneCanvas) PushClipRoundRect(r draw.Rect, _ float32) {
	c.PushClip(r) // approximate as axis-aligned rect
}

// PushClipPath is a no-op stub (requires GPU path clipping support).
func (c *SceneCanvas) PushClipPath(_ draw.Path) {}

func (c *SceneCanvas) PushTransform(_ draw.Transform) {}
func (c *SceneCanvas) PopTransform()                  {}
func (c *SceneCanvas) PushOffset(_, _ float32)        {}
func (c *SceneCanvas) PushScale(_, _ float32)         {}

// ── Effects ──────────────────────────────────────────────────────

func (c *SceneCanvas) PushOpacity(alpha float32) {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	c.opacityStack = append(c.opacityStack, c.effectiveOpacity()*alpha)
}

func (c *SceneCanvas) PopOpacity() {
	if len(c.opacityStack) > 0 {
		c.opacityStack = c.opacityStack[:len(c.opacityStack)-1]
	}
}

// effectiveOpacity returns the cumulative opacity from the stack, or 1.0 if empty.
func (c *SceneCanvas) effectiveOpacity() float32 {
	if len(c.opacityStack) == 0 {
		return 1.0
	}
	return c.opacityStack[len(c.opacityStack)-1]
}
func (c *SceneCanvas) PushBlur(radius float32) {
	if radius <= 0 {
		return
	}
	if radius > 64 {
		radius = 64
	}
	c.blurStack = append(c.blurStack, radius)
	clip := c.clipRect()
	c.scene.BlurRegions = append(c.scene.BlurRegions, draw.BlurRegion{
		X:      int(clip.X),
		Y:      int(clip.Y),
		W:      int(clip.W),
		H:      int(clip.H),
		Radius: radius,
	})
}

func (c *SceneCanvas) PopBlur() {
	if len(c.blurStack) > 0 {
		c.blurStack = c.blurStack[:len(c.blurStack)-1]
	}
}
func (c *SceneCanvas) PushLayer(_ draw.LayerOptions) {}
func (c *SceneCanvas) PopLayer()                     {}

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
