//go:build !nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	glfwplatform "github.com/timzifer/lux/platform/glfw"
	"github.com/timzifer/lux/platform"
)

func defaultPlatformFactory() platform.Platform {
	return glfwplatform.New()
}

func defaultRendererFactory() gpu.Renderer {
	return gpu.NewOpenGL()
}
