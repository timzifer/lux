//go:build !(js && wasm)

package app

import "github.com/timzifer/lux/platform"

// wirePlatformCanvas is a no-op outside browser/WASM builds.
func wirePlatformCanvas(platform.Platform) {}
