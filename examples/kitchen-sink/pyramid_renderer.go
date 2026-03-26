//go:build !nogui && !windows && !(darwin && arm64)

package main

import (
	"github.com/timzifer/lux/internal/gpu"
)

// pyramidRendererFactory returns a renderer factory if the pyramid surface
// requires a custom renderer, or nil to use the default.
func pyramidRendererFactory(_ *PyramidSurface) func() gpu.Renderer {
	return nil
}
