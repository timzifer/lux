//go:build wayland && !nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	waylandplatform "github.com/timzifer/lux/platform/wayland"
)

func defaultPlatformFactory() platform.Platform {
	return waylandplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
