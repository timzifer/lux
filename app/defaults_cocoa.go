//go:build darwin && cocoa && !nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	cocoaplatform "github.com/timzifer/lux/platform/cocoa"
)

func defaultPlatformFactory() platform.Platform {
	return cocoaplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
