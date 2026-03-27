package image

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
)

// decodeRaster decodes PNG or JPEG image data into RGBA8 pixels.
// Format is auto-detected via Go's image.Decode registry.
// Returns width, height, RGBA pixel data (row-major, 4 bytes per pixel).
func decodeRaster(data []byte) (width, height int, rgba []byte, err error) {
	if len(data) == 0 {
		return 0, 0, nil, fmt.Errorf("empty data")
	}

	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return 0, 0, nil, fmt.Errorf("zero-size image (%dx%d)", w, h)
	}

	// Fast path: if already NRGBA, convert in bulk.
	if nrgba, ok := img.(*goimage.NRGBA); ok {
		return w, h, nrgbaToRGBA(nrgba), nil
	}

	// Generic path: pixel-by-pixel conversion.
	out := make([]byte, w*h*4)
	i := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// RGBA() returns pre-multiplied 16-bit values.
			// Convert to 8-bit non-premultiplied for GPU upload (RGBA8Unorm).
			if a == 0 {
				out[i] = 0
				out[i+1] = 0
				out[i+2] = 0
				out[i+3] = 0
			} else {
				out[i] = uint8(r >> 8)
				out[i+1] = uint8(g >> 8)
				out[i+2] = uint8(b >> 8)
				out[i+3] = uint8(a >> 8)
			}
			i += 4
		}
	}
	return w, h, out, nil
}

// nrgbaToRGBA converts an NRGBA image to a flat RGBA byte slice.
func nrgbaToRGBA(img *goimage.NRGBA) []byte {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	out := make([]byte, w*h*4)

	for y := 0; y < h; y++ {
		srcOff := img.PixOffset(bounds.Min.X, bounds.Min.Y+y)
		dstOff := y * w * 4
		copy(out[dstOff:dstOff+w*4], img.Pix[srcOff:srcOff+w*4])
	}
	return out
}

// rgbaModel is used for color conversion.
var _ color.Model = color.NRGBAModel
