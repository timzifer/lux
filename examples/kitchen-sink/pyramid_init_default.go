//go:build nogui || !gogpu

package main

import "github.com/timzifer/lux/internal/gpu"

// pyramidRendererFactory returns nil on non-WGPU builds (use default renderer).
func pyramidRendererFactory(_ *PyramidSurface) func() gpu.Renderer {
	return nil
}
