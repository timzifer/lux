//go:build !windows || nogui

package gpu

import "image"

// windowPresenter returns nil: this platform has no CPU presentation path
// yet, so frames are only rendered (and handed to StilusRenderer.Present).
func windowPresenter(Config) func(*image.RGBA) { return nil }
