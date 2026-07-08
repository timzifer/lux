//go:build js && wasm

package app

import (
	"github.com/timzifer/lux/gpu"
	"github.com/timzifer/lux/platform"
	webplatform "github.com/timzifer/lux/platform/web"
)

func defaultPlatformFactory() platform.Platform {
	return webplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewWGPU()
}
