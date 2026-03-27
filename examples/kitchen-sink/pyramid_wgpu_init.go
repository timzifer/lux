//go:build !nogui && gogpu

package main

import "github.com/timzifer/lux/internal/gpu"

// pyramidRendererFactory returns a renderer factory that also connects
// the PyramidSurface to the WGPU renderer for device/queue access.
func pyramidRendererFactory(pyramid *PyramidSurface) func() gpu.Renderer {
	return func() gpu.Renderer {
		r := gpu.NewWGPU()
		pyramid.SetRenderer(r)
		return r
	}
}
