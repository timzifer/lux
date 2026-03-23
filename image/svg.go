package image

import (
	"fmt"

	"github.com/timzifer/lux/draw"
)

// SVGImage represents a vector image that can be rasterized at any resolution.
// Currently a stub — the interface is defined so that widgets can prepare
// dynamic paths that adapt to widget size without a later API rewrite.
type SVGImage struct {
	ID     draw.ImageID
	source []byte
}

// LoadSVG registers SVG source data for later rasterization.
// Currently returns an error — SVG decoding is not yet implemented.
// The API is provided so that callers can prepare for dynamic
// re-rasterization at different widget sizes.
func (s *Store) LoadSVG(data []byte) (draw.ImageID, error) {
	return 0, fmt.Errorf("image: SVG loading not yet implemented")
}

// RasterizeSVG rasterizes a previously loaded SVG at the given resolution.
// Returns a new ImageID with the rasterized bitmap.
// Currently returns an error — not yet implemented.
func (s *Store) RasterizeSVG(id draw.ImageID, width, height int) (draw.ImageID, error) {
	return 0, fmt.Errorf("image: SVG rasterization not yet implemented")
}
