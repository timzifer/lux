// Package draw — lerp.go provides LerpFunc implementations for draw types.
//
// These functions are designed to be used with anim.LerpAnim[T] so that
// Color, Point, Size, Rect, and CornerRadii can be animated without a
// cyclic dependency between anim/ and draw/ (RFC-002 §1.4).
package draw

// LerpColor interpolates between two Colors component-wise.
func LerpColor(a, b Color, t float32) Color {
	return Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

// LerpPoint interpolates between two Points.
func LerpPoint(a, b Point, t float32) Point {
	return Point{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
	}
}

// LerpSize interpolates between two Sizes.
func LerpSize(a, b Size, t float32) Size {
	return Size{
		W: a.W + (b.W-a.W)*t,
		H: a.H + (b.H-a.H)*t,
	}
}

// LerpRect interpolates between two Rects (origin and size).
func LerpRect(a, b Rect, t float32) Rect {
	return Rect{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		W: a.W + (b.W-a.W)*t,
		H: a.H + (b.H-a.H)*t,
	}
}

// LerpCornerRadii interpolates between two CornerRadii.
func LerpCornerRadii(a, b CornerRadii, t float32) CornerRadii {
	return CornerRadii{
		TopLeft:     a.TopLeft + (b.TopLeft-a.TopLeft)*t,
		TopRight:    a.TopRight + (b.TopRight-a.TopRight)*t,
		BottomRight: a.BottomRight + (b.BottomRight-a.BottomRight)*t,
		BottomLeft:  a.BottomLeft + (b.BottomLeft-a.BottomLeft)*t,
	}
}
