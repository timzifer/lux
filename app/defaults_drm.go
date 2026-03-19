//go:build drm && !nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	drmplatform "github.com/timzifer/lux/platform/drm"
)

func defaultPlatformFactory() platform.Platform {
	return drmplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
