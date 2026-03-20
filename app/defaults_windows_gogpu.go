//go:build !nogui && windows && gogpu

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	windowsplatform "github.com/timzifer/lux/platform/windows"
)

func defaultPlatformFactory() platform.Platform {
	return windowsplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
