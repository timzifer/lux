package button

import "github.com/timzifer/lux/ui"

// maxRippleRadius is a convenience wrapper used by Hold and Confirm buttons.
func maxRippleRadius(cx, cy, x, y, w, h float32) float32 {
	return ui.MaxRippleRadius(cx, cy, x, y, w, h)
}
