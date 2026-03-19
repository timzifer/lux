//go:build x11 && !nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	x11platform "github.com/timzifer/lux/platform/x11"
)

func defaultPlatformFactory() platform.Platform {
	return x11platform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
