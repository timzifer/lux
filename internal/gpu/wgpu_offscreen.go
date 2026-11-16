//go:build !nogui

package gpu

import (
	"errors"
	"image"

	"github.com/timzifer/lux/internal/wgpu"
)

// createOffscreenTarget (re)creates the BGRA render target used when the
// renderer was initialised with Config.Offscreen and no window.
func (r *WGPURenderer) createOffscreenTarget() {
	if r.offView != nil {
		r.offView.Destroy()
	}
	if r.offTex != nil {
		r.offTex.Destroy()
	}
	r.offTex = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "offscreen-target",
		Size:   wgpu.Extent3D{Width: uint32(r.width), Height: uint32(r.height), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageCopySrc | wgpu.TextureUsageTextureBinding,
	})
	r.offView = r.offTex.CreateView()
}

// ErrNoReadback reports a renderer or backend that cannot read pixels back.
var ErrNoReadback = errors.New("gpu: renderer has no offscreen readback")

// WaitIdle blocks until the GPU has finished all submitted work.
func (r *WGPURenderer) WaitIdle() error {
	tr, ok := r.device.(wgpu.TextureReader)
	if !ok {
		return ErrNoReadback
	}
	return tr.WaitIdle()
}

// ReadPixels reads the offscreen frame into dst (premultiplied RGBA, the
// renderer's size). It waits for the GPU.
func (r *WGPURenderer) ReadPixels(dst *image.RGBA) error {
	tr, ok := r.device.(wgpu.TextureReader)
	if !r.offscreen || !ok || r.offTex == nil {
		return ErrNoReadback
	}
	w, h := r.width, r.height
	if dst.Rect.Dx() != w || dst.Rect.Dy() != h {
		return errors.New("gpu: ReadPixels: size mismatch")
	}
	buf := make([]byte, w*h*4)
	if err := tr.ReadTexture(r.offTex, w, h, buf); err != nil {
		return err
	}
	for y := 0; y < h; y++ {
		row := dst.Pix[y*dst.Stride : y*dst.Stride+w*4]
		src := buf[y*w*4 : (y+1)*w*4]
		for x := 0; x < w*4; x += 4 {
			// BGRA → RGBA; the target is opaque, so straight == premultiplied.
			row[x], row[x+1], row[x+2], row[x+3] = src[x+2], src[x+1], src[x], src[x+3]
		}
	}
	return nil
}
