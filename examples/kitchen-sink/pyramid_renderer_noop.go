//go:build nogui || (windows && !gogpu) || (darwin && arm64)

package main

import (
	"github.com/timzifer/lux/internal/gpu"
)

func pyramidRendererFactory(_ *PyramidSurface) func() gpu.Renderer {
	return nil
}
