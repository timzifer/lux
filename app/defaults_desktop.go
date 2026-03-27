//go:build !nogui && !windows && !wayland && !x11 && !cocoa && !drm

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
	glfwplatform "github.com/timzifer/lux/platform/glfw"
)

func defaultPlatformFactory() platform.Platform {
	return glfwplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewOpenGL()
}
