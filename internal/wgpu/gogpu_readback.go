//go:build (gogpu || (darwin && arm64) || windows) && !nogui && !(js && wasm)

package wgpu

import (
	"fmt"

	gpuwgpu "github.com/gogpu/wgpu"
)

var _ TextureReader = (*goDevice)(nil)

// WaitIdle blocks until the GPU has finished all submitted work.
func (d *goDevice) WaitIdle() error { return d.device.WaitIdle() }

// ReadTexture copies a texture to a staging buffer and reads it back.
func (d *goDevice) ReadTexture(tex Texture, width, height int, dst []byte) error {
	gt, ok := tex.(*goTexture)
	if !ok || gt.texture == nil {
		return fmt.Errorf("wgpu/gogpu: ReadTexture: not a gogpu texture")
	}
	if width <= 0 || height <= 0 || len(dst) < width*height*4 {
		return fmt.Errorf("wgpu/gogpu: ReadTexture: bad size %dx%d for %d bytes", width, height, len(dst))
	}
	// Rows of a buffer-texture copy are aligned to 256 bytes.
	stride := (width*4 + 255) &^ 255
	size := uint64(stride * height)
	buf, err := d.device.CreateBuffer(&gpuwgpu.BufferDescriptor{
		Label: "readback",
		Size:  size,
		Usage: gpuwgpu.BufferUsageMapRead | gpuwgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return fmt.Errorf("wgpu/gogpu: readback buffer: %w", err)
	}
	defer buf.Release()

	enc, err := d.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("wgpu/gogpu: readback encoder: %w", err)
	}
	enc.CopyTextureToBuffer(gt.texture, buf, []gpuwgpu.BufferTextureCopy{{
		BufferLayout: gpuwgpu.ImageDataLayout{BytesPerRow: uint32(stride), RowsPerImage: uint32(height)},
		TextureBase:  gpuwgpu.ImageCopyTexture{Texture: gt.texture},
		Size:         gpuwgpu.Extent3D{Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1},
	}})
	cb, err := enc.Finish()
	if err != nil {
		return fmt.Errorf("wgpu/gogpu: readback finish: %w", err)
	}
	if err := d.queue.Submit(cb); err != nil {
		return fmt.Errorf("wgpu/gogpu: readback submit: %w", err)
	}
	if err := d.device.WaitIdle(); err != nil {
		return fmt.Errorf("wgpu/gogpu: readback wait: %w", err)
	}
	staged := make([]byte, size)
	if err := d.queue.ReadBuffer(buf, 0, staged); err != nil {
		return fmt.Errorf("wgpu/gogpu: readback read: %w", err)
	}
	for y := 0; y < height; y++ {
		copy(dst[y*width*4:(y+1)*width*4], staged[y*stride:y*stride+width*4])
	}
	return nil
}
