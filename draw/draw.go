// Package draw provides the 2D rendering types and Canvas interface (RFC §6).
//
// This package is the stable public API for all rendering operations.
// Widgets, themes, and custom draw functions operate through these types.
package draw

// Color is an RGBA color with float32 components in the range [0, 1].
type Color struct {
	R float32
	G float32
	B float32
	A float32
}

// RGBA creates a Color from 8-bit RGBA values.
func RGBA(r, g, b, a uint8) Color {
	return Color{
		R: float32(r) / 255,
		G: float32(g) / 255,
		B: float32(b) / 255,
		A: float32(a) / 255,
	}
}

// Point is a 2D point in dp (density-independent pixels).
type Point struct {
	X float32
	Y float32
}

// Pt is a convenience constructor for Point.
func Pt(x, y float32) Point { return Point{X: x, Y: y} }

// Size is a 2D extent in dp.
type Size struct {
	W float32
	H float32
}

// Rect is an axis-aligned rectangle defined by origin and size.
type Rect struct {
	X float32
	Y float32
	W float32
	H float32
}

// R is a convenience constructor for Rect.
func R(x, y, w, h float32) Rect { return Rect{X: x, Y: y, W: w, H: h} }

// Contains reports whether p is inside r.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// CornerRadii specifies per-corner radii for rounded rectangles.
type CornerRadii struct {
	TopLeft     float32
	TopRight    float32
	BottomRight float32
	BottomLeft  float32
}

// UniformRadii returns CornerRadii with all corners equal.
func UniformRadii(r float32) CornerRadii {
	return CornerRadii{r, r, r, r}
}

// Insets defines padding/margins on all four sides.
type Insets struct {
	Top    float32
	Right  float32
	Bottom float32
	Left   float32
}

// Transform is a 2D affine transformation matrix.
// Layout: [a, b, c, d, tx, ty] representing:
//
//	| a  b  tx |
//	| c  d  ty |
//	| 0  0   1 |
type Transform [6]float32

// Identity returns the identity transform.
func Identity() Transform { return Transform{1, 0, 0, 1, 0, 0} }

// TextMetrics describes the measured dimensions of a text string.
type TextMetrics struct {
	Width   float32
	Ascent  float32
	Descent float32
	Leading float32
}
