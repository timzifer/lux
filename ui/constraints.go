package ui

import "math"

// Constraints define the allowed size range for a widget (RFC-002 §4.2).
type Constraints struct {
	MinWidth, MaxWidth   float32
	MinHeight, MaxHeight float32
}

// TightConstraints returns Constraints where min == max, forcing exact size.
func TightConstraints(w, h float32) Constraints {
	return Constraints{MinWidth: w, MaxWidth: w, MinHeight: h, MaxHeight: h}
}

// LooseConstraints returns Constraints allowing any size up to the given max.
func LooseConstraints(maxW, maxH float32) Constraints {
	return Constraints{MaxWidth: maxW, MaxHeight: maxH}
}

// UnboundedConstraints returns Constraints with no upper bound.
func UnboundedConstraints() Constraints {
	inf := float32(math.Inf(1))
	return Constraints{MaxWidth: inf, MaxHeight: inf}
}

// Constrain clamps a (w, h) pair to lie within the constraint range.
func (c Constraints) Constrain(w, h float32) (float32, float32) {
	if w < c.MinWidth {
		w = c.MinWidth
	}
	if w > c.MaxWidth {
		w = c.MaxWidth
	}
	if h < c.MinHeight {
		h = c.MinHeight
	}
	if h > c.MaxHeight {
		h = c.MaxHeight
	}
	return w, h
}
