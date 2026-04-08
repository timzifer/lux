package draw

// PaintKind identifies the variant of a Paint value.
type PaintKind uint8

const (
	PaintSolid          PaintKind = iota
	PaintLinearGradient
	PaintRadialGradient
	PaintPattern
	PaintImage       // stretched/fitted image fill
	PaintShader      // custom WGSL or predefined effect fill
	PaintShaderImage // shader with an image texture input
)

// GradientStop defines a color at a specific position along a gradient.
type GradientStop struct {
	Offset float32 // 0.0–1.0
	Color  Color
}

// LinearGradient describes a linear color gradient.
type LinearGradient struct {
	Start Point
	End   Point
	Stops []GradientStop
}

// RadialGradient describes a radial color gradient.
type RadialGradient struct {
	Center Point
	Radius float32
	Stops  []GradientStop
}

// PatternDesc describes a repeating image pattern fill.
type PatternDesc struct {
	Image    ImageID
	TileSize Size
}

// ImageScaleMode controls how an image is fitted into a rectangle.
type ImageScaleMode uint8

const (
	ImageScaleFit     ImageScaleMode = iota // maintain aspect ratio, letterbox
	ImageScaleFill                          // maintain aspect ratio, crop
	ImageScaleStretch                       // distort to fill
)

// ImageFill describes an image used as a background fill.
type ImageFill struct {
	Image     ImageID
	ScaleMode ImageScaleMode
}

// ShaderEffect identifies a built-in shader effect.
type ShaderEffect uint8

const (
	ShaderEffectNone    ShaderEffect = iota
	ShaderEffectNoise                // Simplex/Perlin noise pattern
	ShaderEffectPlasma               // Animated plasma effect
	ShaderEffectVoronoi              // Voronoi cell pattern
)

// ShaderDesc describes a shader-based fill — either a built-in effect
// or custom WGSL fragment code.
type ShaderDesc struct {
	Source string       // custom WGSL fragment code (empty = use Effect)
	Effect ShaderEffect // built-in effect (used when Source is empty)
	Params [8]float32   // user-defined uniforms passed to the shader
	Image  ImageID      // optional texture input (for PaintShaderImage)
}

// Paint describes how a surface is filled (RFC §6.2.3).
// Tagged union — exactly one variant is active, determined by Kind.
type Paint struct {
	Kind    PaintKind
	Color   Color            // used when Kind == PaintSolid
	Linear  *LinearGradient  // used when Kind == PaintLinearGradient
	Radial  *RadialGradient  // used when Kind == PaintRadialGradient
	Pattern *PatternDesc     // used when Kind == PaintPattern
	Image   *ImageFill       // used when Kind == PaintImage
	Shader  *ShaderDesc      // used when Kind == PaintShader or PaintShaderImage
}

// SolidPaint creates a solid-color Paint.
func SolidPaint(c Color) Paint {
	return Paint{Kind: PaintSolid, Color: c}
}

// LinearGradientPaint creates a linear gradient Paint.
func LinearGradientPaint(start, end Point, stops ...GradientStop) Paint {
	return Paint{
		Kind:   PaintLinearGradient,
		Linear: &LinearGradient{Start: start, End: end, Stops: stops},
	}
}

// RadialGradientPaint creates a radial gradient Paint.
func RadialGradientPaint(center Point, radius float32, stops ...GradientStop) Paint {
	return Paint{
		Kind:   PaintRadialGradient,
		Radial: &RadialGradient{Center: center, Radius: radius, Stops: stops},
	}
}

// PatternPaint creates a repeating pattern Paint.
func PatternPaint(img ImageID, tileSize Size) Paint {
	return Paint{
		Kind:    PaintPattern,
		Pattern: &PatternDesc{Image: img, TileSize: tileSize},
	}
}

// ImagePaint creates an image-fill Paint.
func ImagePaint(img ImageID, mode ImageScaleMode) Paint {
	return Paint{
		Kind:  PaintImage,
		Image: &ImageFill{Image: img, ScaleMode: mode},
	}
}

// ShaderPaint creates a custom-shader Paint from WGSL fragment source.
func ShaderPaint(source string, params ...float32) Paint {
	var p [8]float32
	copy(p[:], params)
	return Paint{
		Kind:   PaintShader,
		Shader: &ShaderDesc{Source: source, Params: p},
	}
}

// ShaderEffectPaint creates a Paint using a built-in shader effect.
func ShaderEffectPaint(effect ShaderEffect, params ...float32) Paint {
	var p [8]float32
	copy(p[:], params)
	return Paint{
		Kind:   PaintShader,
		Shader: &ShaderDesc{Effect: effect, Params: p},
	}
}

// ShaderImagePaint creates a shader Paint that receives an image as texture input.
func ShaderImagePaint(img ImageID, source string, params ...float32) Paint {
	var p [8]float32
	copy(p[:], params)
	return Paint{
		Kind:   PaintShaderImage,
		Shader: &ShaderDesc{Source: source, Params: p, Image: img},
	}
}

// FallbackColor returns the effective solid color for a Paint.
// For gradients, it returns the first stop's color as a fallback.
// For patterns, images, shaders, and empty gradients, it returns transparent black.
func (p Paint) FallbackColor() Color {
	switch p.Kind {
	case PaintSolid:
		return p.Color
	case PaintLinearGradient:
		if p.Linear != nil && len(p.Linear.Stops) > 0 {
			return p.Linear.Stops[0].Color
		}
	case PaintRadialGradient:
		if p.Radial != nil && len(p.Radial.Stops) > 0 {
			return p.Radial.Stops[0].Color
		}
	}
	return Color{}
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
	Radius       float32 // corner radius of the shadow shape
	Inset        bool    // when true, shadow is drawn inside the rect (inner shadow)
}

// Extent returns how far the shadow extends beyond the source rect on each
// side.  Layout code should reserve this space so the shadow is not clipped.
// Inset shadows return zero extents (they are drawn inside the rect).
func (s Shadow) Extent() Insets {
	if s.Inset {
		return Insets{}
	}
	spread := s.BlurRadius + s.SpreadRadius
	return Insets{
		Top:    max(spread-s.OffsetY, 0),
		Right:  max(spread+s.OffsetX, 0),
		Bottom: max(spread+s.OffsetY, 0),
		Left:   max(spread-s.OffsetX, 0),
	}
}

// MaxExtent returns the component-wise maximum of the extents of all given
// shadows.  Useful when the shadow interpolates between several presets.
func MaxExtent(shadows ...Shadow) Insets {
	var out Insets
	for _, s := range shadows {
		e := s.Extent()
		out.Top = max(out.Top, e.Top)
		out.Right = max(out.Right, e.Right)
		out.Bottom = max(out.Bottom, e.Bottom)
		out.Left = max(out.Left, e.Left)
	}
	return out
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
