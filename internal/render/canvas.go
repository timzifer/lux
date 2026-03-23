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
	if paint.Kind == draw.PaintImage {
		c.appendImageFill(r, paint)
		return
	}
	if paint.Kind == draw.PaintShader || paint.Kind == draw.PaintShaderImage {
		c.appendShaderFill(r, 0, paint)
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
	if paint.Kind == draw.PaintImage {
		c.appendImageFill(r, paint)
		return
	}
	if paint.Kind == draw.PaintShader || paint.Kind == draw.PaintShaderImage {
		c.appendShaderFill(r, radius, paint)
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

// appendImageFill appends an image-filled rect to the scene.
func (c *SceneCanvas) appendImageFill(r draw.Rect, paint draw.Paint) {
	if paint.Image == nil || paint.Image.Image == 0 {
		return
	}
	c.emitClipIfChanged()
	opacity := c.effectiveOpacity()
	if opacity < 0.001 {
		return
	}
	ir := draw.DrawImageRect{
		X: int(r.X), Y: int(r.Y), W: int(r.W), H: int(r.H),
		ImageID: paint.Image.Image,
		Opacity: opacity,
		U0: 0, V0: 0, U1: 1, V1: 1,
	}
	if c.overlayMode {
		c.scene.OverlayImageRects = append(c.scene.OverlayImageRects, ir)
	} else {
		c.scene.ImageRects = append(c.scene.ImageRects, ir)
	}
}

// appendShaderFill appends a shader-filled rect to the scene.
func (c *SceneCanvas) appendShaderFill(r draw.Rect, radius float32, paint draw.Paint) {
	if paint.Shader == nil {
		return
	}
	c.emitClipIfChanged()

	// Build the cache key from effect name or source hash.
	key := paint.Shader.Source
	if key == "" {
		switch paint.Shader.Effect {
		case draw.ShaderEffectNoise:
			key = "_builtin:noise"
		case draw.ShaderEffectPlasma:
			key = "_builtin:plasma"
		case draw.ShaderEffectVoronoi:
			key = "_builtin:voronoi"
		default:
			return // no shader specified
		}
	}

	sr := draw.DrawShaderRect{
		X: int(r.X), Y: int(r.Y), W: int(r.W), H: int(r.H),
		Radius:    radius,
		ShaderKey: key,
		Params:    paint.Shader.Params,
		ImageID:   paint.Shader.Image,
	}
	if c.overlayMode {
		c.scene.OverlayShaderRects = append(c.scene.OverlayShaderRects, sr)
	} else {
		c.scene.ShaderRects = append(c.scene.ShaderRects, sr)
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
			key := text.GlyphKey{FontID: fontID, GlyphID: sg.GlyphID, Rune: sg.Rune, SizePx: uint16(text.MSDFAtlasSize), MSDF: true}
			entry, ok = c.atlas.LookupOrInsertMSDF(key, shaper, f)
		}
		if !ok {
			// Bitmap path (or MSDF fallback for ligature glyphs whose
			// GlyphID has no single-rune cmap entry).
			key := text.GlyphKey{FontID: fontID, GlyphID: sg.GlyphID, Rune: sg.Rune, SizePx: sizePx}
			entry, ok = c.atlas.LookupOrInsert(key, shaper, style)
		}
		if !ok {
			cursorX += sg.Advance
			continue
		}

		// Per-glyph MSDF check: ligature glyphs may fall back to bitmap
		// even when the text run uses MSDF, since the msdf library can't
		// render glyphs without a direct cmap rune mapping.
		glyphIsMSDF := entry.PxRange > 0

		var dstX, dstY, dstW, dstH float32
		if glyphIsMSDF {
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
				if glyphIsMSDF {
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
				if glyphIsMSDF {
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
				if !glyphIsMSDF {
					tg.SrcW = int(tg.DstW)
				} else {
					tg.SrcW = int(tg.DstW / msdfScale)
				}
			}
			if tg.DstY+tg.DstH > clip.Y+clip.H {
				tg.DstH = clip.Y + clip.H - tg.DstY
				if !glyphIsMSDF {
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
		if glyphIsMSDF {
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
	if len(layout.Text) == 0 {
		return
	}

	// If no max width is set, just draw as a single line.
	if layout.MaxWidth <= 0 {
		x := origin.X
		if layout.Alignment != draw.TextAlignLeft {
			m := c.shaper.Measure(layout.Text, layout.Style)
			if layout.Alignment == draw.TextAlignCenter {
				x += (layout.MaxWidth - m.Width) / 2
			}
		}
		c.DrawText(layout.Text, draw.Point{X: x, Y: origin.Y}, layout.Style, color)
		return
	}

	// Break text into lines using the UAX #14 line breaker.
	lines := breakTextIntoLines(layout.Text, layout.Style, layout.MaxWidth, c.shaper)

	metrics := c.shaper.Measure("Mg", layout.Style) // representative metrics
	lineHeight := metrics.Ascent + metrics.Descent + metrics.Leading
	if layout.Style.LineHeight > 0 {
		lineHeight = layout.Style.Size * layout.Style.LineHeight
	}

	y := origin.Y
	for _, line := range lines {
		x := origin.X
		switch layout.Alignment {
		case draw.TextAlignCenter:
			m := c.shaper.Measure(line, layout.Style)
			x += (layout.MaxWidth - m.Width) / 2
		case draw.TextAlignRight:
			m := c.shaper.Measure(line, layout.Style)
			x += layout.MaxWidth - m.Width
		}
		c.DrawText(line, draw.Point{X: x, Y: y}, layout.Style, color)
		y += lineHeight
	}
}

// breakTextIntoLines wraps text into lines that fit within maxWidth,
// using UAX #14 line break opportunities.
func breakTextIntoLines(txt string, style draw.TextStyle, maxWidth float32, shaper text.Shaper) []string {
	breaks := text.DefaultLineBreaker.Breaks(txt)

	if len(breaks) == 0 {
		return []string{txt}
	}

	var lines []string
	lineStart := 0

	for i := 0; i < len(breaks); i++ {
		b := breaks[i]
		candidate := txt[lineStart:b.Offset]

		if b.Kind == text.LineBreakMandatory {
			// Mandatory break: emit the line up to this point.
			lines = append(lines, trimTrailingNewline(candidate))
			lineStart = b.Offset
			continue
		}

		// Check if the text from lineStart to this break fits.
		m := shaper.Measure(candidate, style)
		if m.Width <= maxWidth {
			// Fits so far; continue looking for a wider segment.
			continue
		}

		// Doesn't fit. Break at the previous opportunity if there was one.
		if i > 0 && breaks[i-1].Offset > lineStart {
			prev := breaks[i-1]
			lines = append(lines, trimTrailingWhitespace(txt[lineStart:prev.Offset]))
			lineStart = prev.Offset
			// Re-check current break from the new line start.
			i--
		} else {
			// No previous break opportunity; force break here.
			lines = append(lines, trimTrailingWhitespace(candidate))
			lineStart = b.Offset
		}
	}

	// Remaining text after last break.
	if lineStart < len(txt) {
		lines = append(lines, txt[lineStart:])
	}

	return lines
}

func trimTrailingNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func trimTrailingWhitespace(s string) string {
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// ── Images ───────────────────────────────────────────────────────

func (c *SceneCanvas) DrawImage(img draw.ImageID, dst draw.Rect, opts draw.ImageOptions) {
	c.DrawImageScaled(img, dst, draw.ImageScaleStretch, opts)
}

func (c *SceneCanvas) DrawImageScaled(img draw.ImageID, dst draw.Rect, mode draw.ImageScaleMode, opts draw.ImageOptions) {
	if img == 0 || c.isClipped(dst.X, dst.Y, dst.W, dst.H) {
		return
	}
	c.emitClipIfChanged()
	opacity := opts.Opacity * c.effectiveOpacity()
	if opacity < 0.001 {
		return
	}

	ir := draw.DrawImageRect{
		X: int(dst.X), Y: int(dst.Y), W: int(dst.W), H: int(dst.H),
		ImageID:   img,
		Opacity:   opacity,
		U0: 0, V0: 0, U1: 1, V1: 1,
		ScaleMode: mode,
	}

	// Store current clip rect so the renderer can scissor-clip the image.
	if len(c.clips) > 0 {
		clip := c.clips[len(c.clips)-1]
		ir.ClipX = int(clip.X)
		ir.ClipY = int(clip.Y)
		ir.ClipW = int(clip.W)
		ir.ClipH = int(clip.H)
	}

	if c.overlayMode {
		c.scene.OverlayImageRects = append(c.scene.OverlayImageRects, ir)
	} else {
		c.scene.ImageRects = append(c.scene.ImageRects, ir)
	}
}

func (c *SceneCanvas) DrawImageSlice(slice draw.ImageSlice, dst draw.Rect, opts draw.ImageOptions) {
	if slice.Image == 0 || c.isClipped(dst.X, dst.Y, dst.W, dst.H) {
		return
	}
	// 9-slice rendering: split destination rect into 9 regions based on insets.
	// The insets define border widths; corners are fixed-size, edges stretch
	// in one direction, and the center stretches in both.
	ins := slice.Insets
	left, top, right, bottom := ins.Left, ins.Top, ins.Right, ins.Bottom

	// Clamp insets to destination size.
	if left+right > dst.W {
		scale := dst.W / (left + right)
		left *= scale
		right *= scale
	}
	if top+bottom > dst.H {
		scale := dst.H / (top + bottom)
		top *= scale
		bottom *= scale
	}

	// We need the image's natural size to compute UV coordinates.
	// Since we don't have access to the image store here, we use
	// normalized UV coordinates (0→1) and let the insets define
	// proportional regions. The caller should set insets in the
	// same coordinate space as the image dimensions.
	//
	// For now, insets are interpreted as fractions of the destination
	// rect when the image size is unknown. A future enhancement will
	// pass image dimensions to compute exact UVs.

	// UV boundaries (assuming insets are in pixel units of the source image
	// and the image maps to 0→1 UV space; the GPU will handle the mapping).
	// For correct 9-slice, we'd need image width/height. As a reasonable
	// approximation, we use the destination dimensions.
	uL := left / dst.W
	uR := 1.0 - right/dst.W
	vT := top / dst.H
	vB := 1.0 - bottom/dst.H

	type region struct {
		x, y, w, h         float32
		u0, v0, u1, v1     float32
	}

	regions := [9]region{
		// Top row
		{dst.X, dst.Y, left, top, 0, 0, uL, vT},
		{dst.X + left, dst.Y, dst.W - left - right, top, uL, 0, uR, vT},
		{dst.X + dst.W - right, dst.Y, right, top, uR, 0, 1, vT},
		// Middle row
		{dst.X, dst.Y + top, left, dst.H - top - bottom, 0, vT, uL, vB},
		{dst.X + left, dst.Y + top, dst.W - left - right, dst.H - top - bottom, uL, vT, uR, vB},
		{dst.X + dst.W - right, dst.Y + top, right, dst.H - top - bottom, uR, vT, 1, vB},
		// Bottom row
		{dst.X, dst.Y + dst.H - bottom, left, bottom, 0, vB, uL, 1},
		{dst.X + left, dst.Y + dst.H - bottom, dst.W - left - right, bottom, uL, vB, uR, 1},
		{dst.X + dst.W - right, dst.Y + dst.H - bottom, right, bottom, uR, vB, 1, 1},
	}

	opacity := opts.Opacity
	if opacity == 0 {
		opacity = 1
	}
	opacity *= c.effectiveOpacity()
	if opacity < 0.001 {
		return
	}

	c.emitClipIfChanged()
	for _, r := range regions {
		if r.w <= 0 || r.h <= 0 {
			continue
		}
		ir := draw.DrawImageRect{
			X: int(r.x), Y: int(r.y), W: int(r.w), H: int(r.h),
			ImageID: slice.Image, Opacity: opacity,
			U0: r.u0, V0: r.v0, U1: r.u1, V1: r.v1,
		}
		if c.overlayMode {
			c.scene.OverlayImageRects = append(c.scene.OverlayImageRects, ir)
		} else {
			c.scene.ImageRects = append(c.scene.ImageRects, ir)
		}
	}
}
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
	color := s.Color
	color.A *= c.effectiveOpacity()
	if color.A < 0.001 {
		return
	}

	var x, y, w, h float32
	if s.Inset {
		// Inset shadow: rendered inside the rect, no bound expansion.
		x, y, w, h = r.X, r.Y, r.W, r.H
	} else {
		// Outer shadow: expand bounds by offset + spread + blur.
		expand := s.SpreadRadius + s.BlurRadius
		x = r.X + s.OffsetX - expand
		y = r.Y + s.OffsetY - expand
		w = r.W + 2*expand
		h = r.H + 2*expand
	}
	if c.isClipped(x, y, w, h) {
		return
	}
	c.emitClipIfChanged()
	sr := draw.DrawShadowRect{
		X: int(x), Y: int(y), W: int(w), H: int(h),
		Color:      color,
		Radius:     s.Radius,
		BlurRadius: s.BlurRadius,
		Inset:      s.Inset,
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
