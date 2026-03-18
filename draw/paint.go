package draw

// PaintKind identifies the variant of a Paint value.
type PaintKind uint8

const (
	PaintSolid          PaintKind = iota
	PaintLinearGradient           // reserved for later milestones
	PaintRadialGradient           // reserved for later milestones
	PaintPattern                  // reserved for later milestones
)

// Paint describes how a surface is filled (RFC §6.2.3).
// Tagged union — exactly one variant is active, determined by Kind.
type Paint struct {
	Kind  PaintKind
	Color Color // used when Kind == PaintSolid

	// Future: Linear, Radial, Pattern fields
}

// SolidPaint creates a solid-color Paint.
func SolidPaint(c Color) Paint {
	return Paint{Kind: PaintSolid, Color: c}
}

// StrokeCap controls how line endpoints are drawn.
type StrokeCap uint8

const (
	StrokeCapButt StrokeCap = iota
	StrokeCapRound
	StrokeCapSquare
)

// StrokeJoin controls how line segments are joined.
type StrokeJoin uint8

const (
	StrokeJoinMiter StrokeJoin = iota
	StrokeJoinRound
	StrokeJoinBevel
)

// Stroke describes a contour style.
type Stroke struct {
	Paint      Paint
	Width      float32
	Cap        StrokeCap
	Join       StrokeJoin
	MiterLimit float32
	Dash       []float32
	DashOffset float32
}

// Shadow describes an elevation/drop-shadow effect.
type Shadow struct {
	Color        Color
	BlurRadius   float32
	SpreadRadius float32
	OffsetX      float32
	OffsetY      float32
}

// BlendMode controls layer compositing.
type BlendMode uint8

const (
	BlendNormal BlendMode = iota
	BlendMultiply
	BlendScreen
	BlendOverlay
)

// LayerOptions controls PushLayer behaviour.
type LayerOptions struct {
	BlendMode BlendMode
	Opacity   float32

	// CacheHint is a promise from the widget author to the framework:
	// "This layer's content changes only when DirtyTracker.IsDirty()
	// returns true for the owning widget's state." (RFC-001 §6.2.3)
	//
	// When true, the framework may reuse the recorded GPU command buffer
	// between frames without re-invoking the DrawFunc.
	//
	// If the widget does NOT implement DirtyTracker, CacheHint is ignored
	// and the layer is always re-recorded (safe fallback).
	CacheHint bool
}
