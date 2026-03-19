package ui

import "github.com/timzifer/lux/draw"

// UniformInsets creates Insets with equal spacing on all sides.
func UniformInsets(all float32) draw.Insets {
	return draw.Insets{Top: all, Right: all, Bottom: all, Left: all}
}

// SymmetricInsets creates Insets with equal horizontal and vertical spacing.
func SymmetricInsets(horizontal, vertical float32) draw.Insets {
	return draw.Insets{Top: vertical, Right: horizontal, Bottom: vertical, Left: horizontal}
}

// HorizontalInsets creates Insets with only left and right spacing.
func HorizontalInsets(left, right float32) draw.Insets {
	return draw.Insets{Left: left, Right: right}
}

// VerticalInsets creates Insets with only top and bottom spacing.
func VerticalInsets(top, bottom float32) draw.Insets {
	return draw.Insets{Top: top, Bottom: bottom}
}

// InlineInsets creates Insets using logical Start/End values (RFC-002 §4.6).
// Start maps to Left in LTR and Right in RTL; End maps to the opposite.
func InlineInsets(start, end float32) draw.Insets {
	return draw.Insets{Start: start, End: end}
}

// BlockInsets creates Insets using logical Top/Bottom (block-axis) values.
// Equivalent to VerticalInsets; provided for symmetry with InlineInsets.
func BlockInsets(top, bottom float32) draw.Insets {
	return draw.Insets{Top: top, Bottom: bottom}
}

// LogicalInsets creates Insets using all four logical directions (RFC-002 §4.6).
// Block-axis: Top/Bottom are physical. Inline-axis: Start/End are direction-aware.
func LogicalInsets(top, end, bottom, start float32) draw.Insets {
	return draw.Insets{Top: top, Bottom: bottom, Start: start, End: end}
}
