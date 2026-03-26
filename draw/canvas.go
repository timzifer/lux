package draw

// Canvas is the 2D rendering interface (RFC §6.2).
//
// Coordinates are relative to the current transform stack.
// Origin is the top-left corner of the widget bounds.
type Canvas interface {
	// ── Primitives ───────────────────────────────────────────────

	FillRect(r Rect, paint Paint)
	FillRoundRect(r Rect, radius float32, paint Paint)
	FillRoundRectCorners(r Rect, radii CornerRadii, paint Paint)
	FillEllipse(r Rect, paint Paint)

	StrokeRect(r Rect, stroke Stroke)
	StrokeRoundRect(r Rect, radius float32, stroke Stroke)
	StrokeRoundRectCorners(r Rect, radii CornerRadii, stroke Stroke)
	StrokeEllipse(r Rect, stroke Stroke)
	StrokeLine(a, b Point, stroke Stroke)

	// ── Paths ────────────────────────────────────────────────────

	FillPath(p Path, paint Paint)
	StrokePath(p Path, stroke Stroke)

	// ── Text ─────────────────────────────────────────────────────

	DrawText(text string, origin Point, style TextStyle, color Color)
	MeasureText(text string, style TextStyle) TextMetrics
	DrawTextLayout(layout TextLayout, origin Point, color Color)

	// ── Images & Textures ────────────────────────────────────────

	DrawImage(img ImageID, dst Rect, opts ImageOptions)
	DrawImageScaled(img ImageID, dst Rect, mode ImageScaleMode, opts ImageOptions)
	DrawImageSlice(slice ImageSlice, dst Rect, opts ImageOptions)
	DrawTexture(tex TextureID, dst Rect)

	// ── Shadows ──────────────────────────────────────────────────

	DrawShadow(r Rect, shadow Shadow)

	// ── Clipping & Transform ─────────────────────────────────────

	PushClip(r Rect)
	PushClipRoundRect(r Rect, radius float32)
	PushClipPath(p Path)
	PopClip()
	PushTransform(t Transform)
	PopTransform()
	PushOffset(dx, dy float32)
	PushScale(sx, sy float32)

	// ── Effects ──────────────────────────────────────────────────

	PushOpacity(alpha float32)
	PopOpacity()
	PushBlur(radius float32)
	PopBlur()
	PushLayer(opts LayerOptions)
	PopLayer()

	// ── State ────────────────────────────────────────────────────

	Bounds() Rect
	DPR() float32
	Save()
	Restore()
}

// TextStyle describes how text is rendered.
type TextStyle struct {
	FontFamily string
	Size       float32    // dp
	Weight     FontWeight // 100–900; 400 = Regular
	LineHeight float32    // multiplier
	Tracking   float32    // em
	Raster     bool       // force bitmap rasterization, skip MSDF
}

// FontWeight represents a CSS-like font weight (100–900).
type FontWeight int

const (
	FontWeightThin       FontWeight = 100
	FontWeightLight      FontWeight = 300
	FontWeightRegular    FontWeight = 400
	FontWeightMedium     FontWeight = 500
	FontWeightSemiBold   FontWeight = 600
	FontWeightBold       FontWeight = 700
	FontWeightBlack      FontWeight = 900
)

// TextAlign controls horizontal text alignment within a TextLayout.
type TextAlign uint8

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

// TextLayout describes a block of text with layout constraints.
type TextLayout struct {
	Text      string
	Style     TextStyle
	MaxWidth  float32   // 0 = unbounded
	Alignment TextAlign // Left, Center, Right
}

// ImageID is a handle to a loaded image/texture.
type ImageID uint64

// TextureID is a handle to a GPU texture (for Surface slots).
type TextureID uint64

// SurfaceID identifies a surface slot in the widget tree (RFC §8).
type SurfaceID uint64

// DrawSurface is a GPU texture blit for an external surface (RFC §8).
type DrawSurface struct {
	X, Y, W, H  int
	TextureID    TextureID
	SurfaceID    SurfaceID
	ClipX, ClipY int // scissor clip origin (from scene clip stack)
	ClipW, ClipH int // scissor clip size; 0 = full viewport
}

// ImageSlice describes a 9-slice image for scalable borders/backgrounds.
type ImageSlice struct {
	Image  ImageID
	Insets Insets // defines the 9-slice border regions
}

// ImageOptions controls image rendering.
type ImageOptions struct {
	Opacity float32 // 0.0–1.0
}

// ── Scene graph ──────────────────────────────────────────────────
// The scene is the flat, fully laid-out draw list that the GPU
// renderer consumes.  It is produced by a Canvas implementation
// (internal/render) and consumed by the GPU backend (internal/gpu).

// DrawRect is a filled rectangle in framebuffer coordinates.
type DrawRect struct {
	X      int
	Y      int
	W      int
	H      int
	Color  Color
	Radius float32 // corner radius in dp; 0 = sharp corners
}

// DrawGlyph is a single text glyph in framebuffer coordinates.
type DrawGlyph struct {
	X     int
	Y     int
	Scale int
	Text  string
	Color Color
}

// TexturedGlyph is an atlas-based glyph drawn as a textured quad.
type TexturedGlyph struct {
	DstX, DstY float32 // screen position (dp)
	DstW, DstH float32 // screen size (dp)
	SrcX, SrcY int     // atlas position (pixels)
	SrcW, SrcH int     // atlas size (pixels)
	Color      Color
}

// DrawGradientRect is a gradient-filled rectangle in framebuffer coordinates.
type DrawGradientRect struct {
	X, Y, W, H int
	Radius      float32
	Kind        PaintKind // PaintLinearGradient or PaintRadialGradient

	// Linear gradient: Start/End in screen coords.
	StartX, StartY, EndX, EndY float32
	// Radial gradient: Center + Radius in screen coords.
	CenterX, CenterY, GradRadius float32

	Stops     [8]GradientStop
	StopCount int
}

// DrawShadowRect is a soft-shadow rectangle in framebuffer coordinates.
type DrawShadowRect struct {
	X, Y, W, H int
	Color       Color
	Radius      float32 // corner radius
	BlurRadius  float32 // shadow blur spread
	Inset       bool    // when true, shadow is drawn inside the rect
}

// DrawImageRect is a textured image rectangle in framebuffer coordinates.
type DrawImageRect struct {
	X, Y, W, H     int
	ImageID         ImageID
	Opacity         float32
	U0, V0, U1, V1 float32        // UV subregion (0,0 → 1,1 = full image)
	ScaleMode       ImageScaleMode // Fit/Fill/Stretch
	ClipX, ClipY    int            // scissor clip origin (from scene clip stack)
	ClipW, ClipH    int            // scissor clip size; 0 = full viewport
}

// DrawShaderRect is a shader-filled rectangle in framebuffer coordinates.
type DrawShaderRect struct {
	X, Y, W, H int
	Radius      float32
	ShaderKey   string     // cache key (source hash or effect name)
	Params      [8]float32 // user-defined uniforms
	Time        float32    // frame time for animation (seconds since app start)
	ImageID     ImageID    // optional texture input (0 = none)
}

// PathVertex is a single vertex in a tessellated path triangle mesh.
type PathVertex struct {
	X, Y float32
}

// DrawPathBatch describes a batch of triangles for a filled or stroked path.
type DrawPathBatch struct {
	VertexOffset int   // start index in PathVertices[]
	VertexCount  int   // number of vertices (multiple of 3)
	Color        Color // fill/stroke color
}

// ClipBatch groups draw commands that share the same scissor rectangle.
type ClipBatch struct {
	Clip         Rect
	RectIdx      int  // start index in Rects[]
	TextIdx      int  // start index in TexturedGlyphs[]
	MSDFIdx      int  // start index in MSDFGlyphs[]
	EmojiIdx     int  // start index in EmojiGlyphs[]
	GradientIdx  int  // start index in GradientRects[]
	ShadowIdx    int  // start index in ShadowRects[]
	ImageIdx     int  // start index in ImageRects[]
	ShaderIdx    int  // start index in ShaderRects[]
	PathIdx      int  // start index in PathBatches[]
	FullViewport bool // true = no scissor, full viewport
}

// BlurRegion describes a rectangular area to apply Gaussian blur.
type BlurRegion struct {
	X, Y, W, H int
	Radius      float32
}

// Scene is the fully laid-out draw list for one frame.
type Scene struct {
	Grain float32 // Noise/grain intensity from theme (RFC-008 §10.5); 0 = off

	Rects          []DrawRect
	Glyphs         []DrawGlyph      // legacy bitmap glyphs
	TexturedGlyphs []TexturedGlyph  // atlas-based glyphs
	MSDFGlyphs     []TexturedGlyph  // MSDF atlas-based glyphs
	EmojiGlyphs    []TexturedGlyph  // color emoji atlas-based glyphs

	// External surface texture blits (RFC §8).
	Surfaces []DrawSurface

	// Gradient-filled rectangles.
	GradientRects []DrawGradientRect

	// Soft-shadow rectangles (rendered before rects so shadows go behind content).
	ShadowRects []DrawShadowRect

	// Image-filled rectangles.
	ImageRects []DrawImageRect

	// Shader-filled rectangles.
	ShaderRects []DrawShaderRect

	// Tessellated path triangles (CPU-tessellated, GPU-rendered).
	PathVertices []PathVertex
	PathBatches  []DrawPathBatch

	// Overlay draw lists — rendered after main content so overlays
	// (tooltips, dropdowns, context menus) fully cover underlying text.
	OverlayRects          []DrawRect
	OverlayGlyphs         []DrawGlyph
	OverlayTexturedGlyphs []TexturedGlyph
	OverlayMSDFGlyphs     []TexturedGlyph
	OverlayEmojiGlyphs    []TexturedGlyph
	OverlayGradientRects  []DrawGradientRect
	OverlayShadowRects    []DrawShadowRect
	OverlayImageRects     []DrawImageRect
	OverlayShaderRects    []DrawShaderRect
	OverlayPathVertices   []PathVertex
	OverlayPathBatches    []DrawPathBatch

	// Scissor clip batches — each batch specifies a scissor rect and
	// index ranges into the main/overlay draw lists.
	ClipBatches        []ClipBatch
	OverlayClipBatches []ClipBatch

	// Blur regions — areas to apply Gaussian blur post-processing.
	BlurRegions []BlurRegion
}
