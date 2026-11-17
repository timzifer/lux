package app

import (
	"image"

	"github.com/timzifer/lux/internal/gpu"
)

// WithSoftwareRenderer renders on the CPU with the pure-Go stilus backend
// instead of the GPU (RFC-001 §6.1, "Software (CI, Headless)").
//
// present, if non-nil, receives every finished frame as premultiplied RGBA.
// The image is reused for the next frame; copy it to keep it. With a nil
// present the frame is shown in the window where the platform supports
// CPU presentation (Windows, via GDI) and otherwise only rendered — which
// is what headless runs (-tags nogui) need to test real pixels.
func WithSoftwareRenderer(present func(img *image.RGBA)) Option {
	return func(o *options) {
		o.rendererFactory = func() gpu.Renderer {
			r := gpu.NewStilus()
			r.Present = present
			return r
		}
	}
}
