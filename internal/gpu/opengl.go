//go:build !nogui && !windows && !(darwin && arm64)

package gpu

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
)

// OpenGLRenderer implements Renderer using OpenGL 3.3 Core.
type OpenGLRenderer struct {
	width   int
	height  int
	bgColor draw.Color

	// Text rendering resources.
	textProgram  uint32
	textVAO      uint32
	textVBO      uint32
	atlasTexture uint32
	projUniform  int32
	colorUniform int32
	atlasUniform int32
	atlas        *text.GlyphAtlas
	textInited   bool

	// MSDF text rendering resources.
	msdfProgram        uint32
	msdfVAO            uint32
	msdfVBO            uint32
	msdfAtlasTexture   uint32
	msdfProjUniform    int32
	msdfColorUniform   int32
	msdfAtlasUniform   int32
	msdfPxRangeUniform int32
	msdfInited         bool

	// Rounded rect rendering resources.
	rectProgram      uint32
	rectVAO          uint32
	rectVBO          uint32
	rectInstanceVBO  uint32
	rectProjUniform  int32
	rectGrainUniform int32 // RFC-008 §10.5
	rectInited       bool
	grain            float32 // current grain intensity from scene

	// Surface texture-blit rendering resources (RFC §8).
	surfProgram     uint32
	surfVAO         uint32
	surfVBO         uint32
	surfProjUniform int32
	surfTexUniform  int32
	surfInited      bool
}

// NewOpenGL creates an OpenGL-based renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
}

// Init initializes OpenGL.
func (r *OpenGLRenderer) Init(cfg Config) error {
	if err := gl.Init(); err != nil {
		return fmt.Errorf("opengl init: %w", err)
	}

	r.width = cfg.Width
	r.height = cfg.Height

	gl.Viewport(0, 0, int32(r.width), int32(r.height))
	gl.Enable(gl.SCISSOR_TEST)

	return nil
}

// SetBackgroundColor sets the clear color for BeginFrame.
func (r *OpenGLRenderer) SetBackgroundColor(c draw.Color) {
	r.bgColor = c
}

// SetAtlas sets the glyph atlas for textured glyph rendering.
func (r *OpenGLRenderer) SetAtlas(a *text.GlyphAtlas) {
	r.atlas = a
}

// Resize updates the viewport.
func (r *OpenGLRenderer) Resize(width, height int) {
	r.width = width
	r.height = height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// BeginFrame clears the screen.
func (r *OpenGLRenderer) BeginFrame() {
	gl.Disable(gl.SCISSOR_TEST)
	gl.ClearColor(r.bgColor.R, r.bgColor.G, r.bgColor.B, r.bgColor.A)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.Enable(gl.SCISSOR_TEST)
}

// Draw renders the scene via gl.Scissor + gl.Clear for sharp rects,
// shader-based SDF for rounded rects, bitmap glyphs via per-pixel scissor,
// and textured quads for atlas glyphs.
func (r *OpenGLRenderer) Draw(scene draw.Scene) {
	r.grain = scene.Grain // RFC-008 §10.5
	// Render rects preserving scene order: batch consecutive rounded rects,
	// flush before each sharp rect to maintain correct z-ordering.
	var roundedBatch []draw.DrawRect
	for _, rect := range scene.Rects {
		if rect.Radius > 0 {
			roundedBatch = append(roundedBatch, rect)
		} else {
			if len(roundedBatch) > 0 {
				r.drawRoundedRects(roundedBatch)
				roundedBatch = roundedBatch[:0]
			}
			r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
		}
	}
	if len(roundedBatch) > 0 {
		r.drawRoundedRects(roundedBatch)
	}
	if len(scene.Surfaces) > 0 {
		r.drawSurfaces(scene.Surfaces)
	}
	for _, glyph := range scene.Glyphs {
		r.drawGlyph(glyph)
	}
	if len(scene.TexturedGlyphs) > 0 {
		r.drawTexturedGlyphs(scene.TexturedGlyphs)
	}
	if len(scene.MSDFGlyphs) > 0 {
		r.drawMSDFGlyphs(scene.MSDFGlyphs)
	}

	// Overlay pass — rendered after all main content so overlays
	// (dropdowns, tooltips, context menus) fully cover underlying text.
	if len(scene.OverlayRects) > 0 {
		var overlayRounded []draw.DrawRect
		for _, rect := range scene.OverlayRects {
			if rect.Radius > 0 {
				overlayRounded = append(overlayRounded, rect)
			} else {
				if len(overlayRounded) > 0 {
					r.drawRoundedRects(overlayRounded)
					overlayRounded = overlayRounded[:0]
				}
				r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
			}
		}
		if len(overlayRounded) > 0 {
			r.drawRoundedRects(overlayRounded)
		}
	}
	for _, glyph := range scene.OverlayGlyphs {
		r.drawGlyph(glyph)
	}
	if len(scene.OverlayTexturedGlyphs) > 0 {
		r.drawTexturedGlyphs(scene.OverlayTexturedGlyphs)
	}
	if len(scene.OverlayMSDFGlyphs) > 0 {
		r.drawMSDFGlyphs(scene.OverlayMSDFGlyphs)
	}
}

// EndFrame is a no-op for OpenGL (swap is handled by GLFW).
func (r *OpenGLRenderer) EndFrame() {}

// Destroy releases OpenGL resources.
func (r *OpenGLRenderer) Destroy() {
	if r.textInited {
		gl.DeleteProgram(r.textProgram)
		gl.DeleteVertexArrays(1, &r.textVAO)
		gl.DeleteBuffers(1, &r.textVBO)
		if r.atlasTexture != 0 {
			gl.DeleteTextures(1, &r.atlasTexture)
		}
	}
	if r.msdfInited {
		gl.DeleteProgram(r.msdfProgram)
		gl.DeleteVertexArrays(1, &r.msdfVAO)
		gl.DeleteBuffers(1, &r.msdfVBO)
		if r.msdfAtlasTexture != 0 {
			gl.DeleteTextures(1, &r.msdfAtlasTexture)
		}
	}
	if r.rectInited {
		gl.DeleteProgram(r.rectProgram)
		gl.DeleteVertexArrays(1, &r.rectVAO)
		gl.DeleteBuffers(1, &r.rectVBO)
		gl.DeleteBuffers(1, &r.rectInstanceVBO)
	}
	if r.surfInited {
		gl.DeleteProgram(r.surfProgram)
		gl.DeleteVertexArrays(1, &r.surfVAO)
		gl.DeleteBuffers(1, &r.surfVBO)
	}
}

func (r *OpenGLRenderer) fillRect(x, y, w, h int, color draw.Color) {
	if w <= 0 || h <= 0 || r.width <= 0 || r.height <= 0 {
		return
	}
	gl.ClearColor(color.R, color.G, color.B, color.A)
	gl.Scissor(int32(x), int32(r.height-y-h), int32(w), int32(h))
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (r *OpenGLRenderer) drawGlyph(cmd draw.DrawGlyph) {
	if cmd.Scale <= 0 {
		cmd.Scale = 1
	}
	render.RenderBitmapGlyph(cmd.Text, cmd.X, cmd.Y, cmd.Scale, func(px, py, w, h int) {
		r.fillRect(px, py, w, h, cmd.Color)
	})
}

// ── Rounded rect rendering ───────────────────────────────────────

func (r *OpenGLRenderer) initRectRendering() {
	if r.rectInited {
		return
	}

	program, err := compileProgram(rectVertexShader, rectFragmentShader)
	if err != nil {
		return
	}

	r.rectProgram = program
	r.rectProjUniform = gl.GetUniformLocation(program, gl.Str("uProj\x00"))
	r.rectGrainUniform = gl.GetUniformLocation(program, gl.Str("uGrain\x00"))

	// Unit quad vertices: 6 vertices (2 triangles).
	quadVerts := []float32{
		0, 0,
		1, 0,
		0, 1,
		1, 0,
		1, 1,
		0, 1,
	}

	gl.GenVertexArrays(1, &r.rectVAO)
	gl.GenBuffers(1, &r.rectVBO)
	gl.GenBuffers(1, &r.rectInstanceVBO)

	gl.BindVertexArray(r.rectVAO)

	// Attribute 0: unit quad vertex (per-vertex).
	gl.BindBuffer(gl.ARRAY_BUFFER, r.rectVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadVerts)*4, gl.Ptr(quadVerts), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 2*4, 0)

	// Instance buffer: rect (x,y,w,h), color (r,g,b,a), radius — 9 floats per instance.
	gl.BindBuffer(gl.ARRAY_BUFFER, r.rectInstanceVBO)

	stride := int32(9 * 4)
	// Attribute 1: aRect (vec4) = x, y, w, h
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 4, gl.FLOAT, false, stride, 0)
	gl.VertexAttribDivisor(1, 1)

	// Attribute 2: aColor (vec4) = r, g, b, a
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, stride, uintptr(4*4))
	gl.VertexAttribDivisor(2, 1)

	// Attribute 3: aRadius (float)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 1, gl.FLOAT, false, stride, uintptr(8*4))
	gl.VertexAttribDivisor(3, 1)

	gl.BindVertexArray(0)
	r.rectInited = true
}

func (r *OpenGLRenderer) drawRoundedRects(rects []draw.DrawRect) {
	r.initRectRendering()
	if !r.rectInited {
		return
	}

	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.UseProgram(r.rectProgram)

	proj := orthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	gl.UniformMatrix4fv(r.rectProjUniform, 1, false, &proj[0])
	gl.Uniform1f(r.rectGrainUniform, r.grain)

	// Build instance data: 9 floats per rect.
	instances := make([]float32, 0, len(rects)*9)
	for _, rect := range rects {
		instances = append(instances,
			float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
			rect.Color.R, rect.Color.G, rect.Color.B, rect.Color.A,
			rect.Radius,
		)
	}

	gl.BindVertexArray(r.rectVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.rectInstanceVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(instances)*4, gl.Ptr(instances), gl.DYNAMIC_DRAW)

	gl.DrawArraysInstanced(gl.TRIANGLES, 0, 6, int32(len(rects)))

	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)
}

// ── Textured glyph rendering ────────────────────────────────────

func (r *OpenGLRenderer) initTextRendering() {
	if r.textInited {
		return
	}

	program, err := compileProgram(textVertexShader, textFragmentShader)
	if err != nil {
		// If shaders fail to compile, fall back silently.
		return
	}

	r.textProgram = program
	r.projUniform = gl.GetUniformLocation(program, gl.Str("uProj\x00"))
	r.colorUniform = gl.GetUniformLocation(program, gl.Str("uColor\x00"))
	r.atlasUniform = gl.GetUniformLocation(program, gl.Str("uAtlas\x00"))

	// Create VAO and VBO.
	gl.GenVertexArrays(1, &r.textVAO)
	gl.GenBuffers(1, &r.textVBO)

	gl.BindVertexArray(r.textVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.textVBO)

	// Vertex layout: [posX, posY, uvX, uvY] per vertex.
	stride := int32(4 * 4) // 4 floats * 4 bytes
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, uintptr(2*4))

	gl.BindVertexArray(0)

	// Create atlas texture.
	gl.GenTextures(1, &r.atlasTexture)
	gl.BindTexture(gl.TEXTURE_2D, r.atlasTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	// GL_RED for single-channel grayscale.
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_SWIZZLE_R, gl.RED)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_SWIZZLE_G, gl.RED)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_SWIZZLE_B, gl.RED)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_SWIZZLE_A, gl.RED)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	r.textInited = true
}

func (r *OpenGLRenderer) uploadAtlas() {
	if r.atlas == nil || !r.atlas.Dirty {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, r.atlasTexture)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RED,
		int32(r.atlas.Width), int32(r.atlas.Height),
		0, gl.RED, gl.UNSIGNED_BYTE,
		gl.Ptr(r.atlas.Image.Pix),
	)
	r.atlas.Dirty = false
}

func (r *OpenGLRenderer) drawTexturedGlyphs(glyphs []draw.TexturedGlyph) {
	r.initTextRendering()
	if !r.textInited || r.atlas == nil {
		return
	}

	r.uploadAtlas()

	// Disable scissor for textured rendering.
	gl.Disable(gl.SCISSOR_TEST)

	// Enable alpha blending.
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.UseProgram(r.textProgram)

	// Set orthographic projection matrix.
	proj := orthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	gl.UniformMatrix4fv(r.projUniform, 1, false, &proj[0])

	// Bind atlas texture.
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.atlasTexture)
	gl.Uniform1i(r.atlasUniform, 0)

	gl.BindVertexArray(r.textVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.textVBO)

	atlasW := float32(r.atlas.Width)
	atlasH := float32(r.atlas.Height)

	// Build vertex data: 6 vertices per glyph (2 triangles).
	vertices := make([]float32, 0, len(glyphs)*6*4)

	var prevColor draw.Color
	firstGlyph := true

	for _, g := range glyphs {
		// If color changes, flush the current batch.
		if !firstGlyph && g.Color != prevColor {
			r.flushTextBatch(vertices, prevColor)
			vertices = vertices[:0]
		}
		prevColor = g.Color
		firstGlyph = false

		x0 := g.DstX
		y0 := g.DstY
		x1 := g.DstX + g.DstW
		y1 := g.DstY + g.DstH

		u0 := float32(g.SrcX) / atlasW
		v0 := float32(g.SrcY) / atlasH
		u1 := float32(g.SrcX+g.SrcW) / atlasW
		v1 := float32(g.SrcY+g.SrcH) / atlasH

		// Two triangles forming a quad.
		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,

			x1, y0, u1, v0,
			x1, y1, u1, v1,
			x0, y1, u0, v1,
		)
	}

	if len(vertices) > 0 {
		r.flushTextBatch(vertices, prevColor)
	}

	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)
}

func (r *OpenGLRenderer) flushTextBatch(vertices []float32, color draw.Color) {
	gl.Uniform4f(r.colorUniform, color.R, color.G, color.B, color.A)
	gl.BufferData(gl.ARRAY_BUFFER,
		len(vertices)*4,
		gl.Ptr(vertices),
		gl.DYNAMIC_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)/4))
}

// ── MSDF glyph rendering ────────────────────────────────────────

func (r *OpenGLRenderer) initMSDFRendering() {
	if r.msdfInited {
		return
	}

	program, err := compileProgram(textVertexShader, msdfFragmentShader)
	if err != nil {
		return
	}

	r.msdfProgram = program
	r.msdfProjUniform = gl.GetUniformLocation(program, gl.Str("uProj\x00"))
	r.msdfColorUniform = gl.GetUniformLocation(program, gl.Str("uColor\x00"))
	r.msdfAtlasUniform = gl.GetUniformLocation(program, gl.Str("uAtlas\x00"))
	r.msdfPxRangeUniform = gl.GetUniformLocation(program, gl.Str("uPxRange\x00"))

	gl.GenVertexArrays(1, &r.msdfVAO)
	gl.GenBuffers(1, &r.msdfVBO)

	gl.BindVertexArray(r.msdfVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.msdfVBO)

	stride := int32(4 * 4)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, uintptr(2*4))

	gl.BindVertexArray(0)

	// Create MSDF atlas texture (RGBA, no swizzle).
	gl.GenTextures(1, &r.msdfAtlasTexture)
	gl.BindTexture(gl.TEXTURE_2D, r.msdfAtlasTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	r.msdfInited = true
}

func (r *OpenGLRenderer) uploadMSDFAtlas() {
	if r.atlas == nil || !r.atlas.MSDFDirty {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, r.msdfAtlasTexture)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA,
		int32(r.atlas.MSDFWidth), int32(r.atlas.MSDFHeight),
		0, gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(r.atlas.MSDFImage.Pix),
	)
	r.atlas.MSDFDirty = false
}

func (r *OpenGLRenderer) drawMSDFGlyphs(glyphs []draw.TexturedGlyph) {
	r.initMSDFRendering()
	if !r.msdfInited || r.atlas == nil {
		return
	}

	r.uploadMSDFAtlas()

	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.UseProgram(r.msdfProgram)

	proj := orthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	gl.UniformMatrix4fv(r.msdfProjUniform, 1, false, &proj[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.msdfAtlasTexture)
	gl.Uniform1i(r.msdfAtlasUniform, 0)
	gl.Uniform1f(r.msdfPxRangeUniform, float32(text.MSDFPxRange))

	gl.BindVertexArray(r.msdfVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.msdfVBO)

	atlasW := float32(r.atlas.MSDFWidth)
	atlasH := float32(r.atlas.MSDFHeight)

	vertices := make([]float32, 0, len(glyphs)*6*4)
	var prevColor draw.Color
	firstGlyph := true

	for _, g := range glyphs {
		if !firstGlyph && g.Color != prevColor {
			r.flushMSDFBatch(vertices, prevColor)
			vertices = vertices[:0]
		}
		prevColor = g.Color
		firstGlyph = false

		x0 := g.DstX
		y0 := g.DstY
		x1 := g.DstX + g.DstW
		y1 := g.DstY + g.DstH

		u0 := float32(g.SrcX) / atlasW
		v0 := float32(g.SrcY) / atlasH
		u1 := float32(g.SrcX+g.SrcW) / atlasW
		v1 := float32(g.SrcY+g.SrcH) / atlasH

		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,

			x1, y0, u1, v0,
			x1, y1, u1, v1,
			x0, y1, u0, v1,
		)
	}

	if len(vertices) > 0 {
		r.flushMSDFBatch(vertices, prevColor)
	}

	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)
}

func (r *OpenGLRenderer) flushMSDFBatch(vertices []float32, color draw.Color) {
	gl.Uniform4f(r.msdfColorUniform, color.R, color.G, color.B, color.A)
	gl.BufferData(gl.ARRAY_BUFFER,
		len(vertices)*4,
		gl.Ptr(vertices),
		gl.DYNAMIC_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)/4))
}

// ── Shader helpers ──────────────────────────────────────────────

// ── Surface texture-blit rendering (RFC §8) ─────────────────────

func (r *OpenGLRenderer) initSurfaceRendering() {
	if r.surfInited {
		return
	}

	program, err := compileProgram(surfaceVertexShader, surfaceFragmentShader)
	if err != nil {
		return
	}

	r.surfProgram = program
	r.surfProjUniform = gl.GetUniformLocation(program, gl.Str("uProj\x00"))
	r.surfTexUniform = gl.GetUniformLocation(program, gl.Str("uTex\x00"))

	gl.GenVertexArrays(1, &r.surfVAO)
	gl.GenBuffers(1, &r.surfVBO)

	gl.BindVertexArray(r.surfVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.surfVBO)

	// Vertex layout: [posX, posY, uvX, uvY] per vertex.
	stride := int32(4 * 4)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, uintptr(2*4))

	gl.BindVertexArray(0)
	r.surfInited = true
}

func (r *OpenGLRenderer) drawSurfaces(surfaces []draw.DrawSurface) {
	r.initSurfaceRendering()
	if !r.surfInited {
		return
	}

	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.UseProgram(r.surfProgram)

	proj := orthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	gl.UniformMatrix4fv(r.surfProjUniform, 1, false, &proj[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.Uniform1i(r.surfTexUniform, 0)

	gl.BindVertexArray(r.surfVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.surfVBO)

	for _, s := range surfaces {
		if s.TextureID == 0 {
			continue
		}

		x0 := float32(s.X)
		y0 := float32(s.Y)
		x1 := float32(s.X + s.W)
		y1 := float32(s.Y + s.H)

		// UV: (0,0) bottom-left to (1,1) top-right in OpenGL, but FBO textures
		// are already in correct orientation, so flip V.
		vertices := []float32{
			x0, y0, 0, 1,
			x1, y0, 1, 1,
			x0, y1, 0, 0,

			x1, y0, 1, 1,
			x1, y1, 1, 0,
			x0, y1, 0, 0,
		}

		gl.BindTexture(gl.TEXTURE_2D, uint32(s.TextureID))
		gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.SCISSOR_TEST)
}

func compileProgram(vertSrc, fragSrc string) (uint32, error) {
	vert, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	frag, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vert)
		return 0, err
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		gl.DeleteProgram(prog)
		gl.DeleteShader(vert)
		gl.DeleteShader(frag)
		return 0, fmt.Errorf("program link: %s", log)
	}

	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	return prog, nil
}

func compileShader(src string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("shader compile: %s", log)
	}
	return shader, nil
}

// orthoMatrix returns a 4x4 orthographic projection matrix.
func orthoMatrix(left, right, bottom, top, near, far float32) [16]float32 {
	dx := right - left
	dy := top - bottom
	dz := far - near
	return [16]float32{
		2 / dx, 0, 0, 0,
		0, 2 / dy, 0, 0,
		0, 0, -2 / dz, 0,
		-(right + left) / dx, -(top + bottom) / dy, -(far + near) / dz, 1,
	}
}

// Ensure unsafe is used (for gl.Ptr).
var _ = unsafe.Pointer(nil)
