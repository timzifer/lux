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
