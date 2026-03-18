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

	// ── Images & Textures ────────────────────────────────────────

	DrawImage(img ImageID, dst Rect, opts ImageOptions)

	// ── Shadows ──────────────────────────────────────────────────

	DrawShadow(r Rect, shadow Shadow)

	// ── Clipping & Transform ─────────────────────────────────────

	PushClip(r Rect)
	PopClip()
	PushTransform(t Transform)
	PopTransform()
	PushOffset(dx, dy float32)

	// ── Effects ──────────────────────────────────────────────────

	PushOpacity(alpha float32)
	PopOpacity()

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

// ImageID is a handle to a loaded image/texture.
type ImageID uint64

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

// Scene is the fully laid-out draw list for one frame.
type Scene struct {
	Rects          []DrawRect
	Glyphs         []DrawGlyph      // legacy bitmap glyphs
	TexturedGlyphs []TexturedGlyph  // atlas-based glyphs
}
