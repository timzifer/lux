//go:build js && wasm

package app

import (
	"syscall/js"

	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/wgpu"
)

// wirePlatformCanvas hands the platform's HTML canvas to the GPU backend.
// Probed via an optional interface so the platform layer stays free of
// renderer dependencies. Must run after plat.Init (which selects the
// canvas) and before renderer.Init (which creates the surface from it).
func wirePlatformCanvas(plat platform.Platform) {
	if cp, ok := plat.(interface{ WASMCanvas() js.Value }); ok {
		wgpu.SetWASMCanvas(cp.WASMCanvas())
	}
}
