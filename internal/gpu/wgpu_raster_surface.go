//go:build !nogui

package gpu

import (
	"image"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/wgpu"
)

// rasterSurfaceTexBase marks TextureIDs the renderer assigns to CPU surfaces,
// far above IDs handed out by applications.
const rasterSurfaceTexBase draw.TextureID = 1 << 62

// rasterSurfaceSlot is the scratch image and GPU texture for one CPU surface
// of a frame (slots are reused by position from frame to frame).
type rasterSurfaceSlot struct {
	img      *image.RGBA
	straight []byte
	tex      wgpu.Texture
	view     wgpu.TextureView
	w, h     int
}

// prepareSurfaces rasterizes CPU surfaces (draw.SurfaceRasterizer) into
// textures, assigns them TextureIDs and uploads one instance rect per
// surface. It must run before the render pass is recorded: buffer and
// texture writes take effect immediately, draws only at submit.
func (r *WGPURenderer) prepareSurfaces(surfaces []draw.DrawSurface) []draw.DrawSurface {
	if len(surfaces) == 0 {
		return nil
	}
	out := r.surfFrame[:0]
	slot := 0
	vp := image.Rect(0, 0, r.width, r.height)
	for _, s := range surfaces {
		if s.Raster != nil {
			rect := image.Rect(s.X, s.Y, s.X+s.W, s.Y+s.H)
			clip := rect.Intersect(vp)
			if s.ClipW > 0 && s.ClipH > 0 {
				clip = clip.Intersect(image.Rect(s.ClipX, s.ClipY, s.ClipX+s.ClipW, s.ClipY+s.ClipH))
			}
			if clip.Empty() {
				continue
			}
			s.TextureID = r.rasterSurface(slot, s.Raster, rect, clip)
			slot++
		}
		if s.TextureID == 0 {
			continue
		}
		if _, ok := r.surfaceTextures[s.TextureID]; !ok {
			continue
		}
		out = append(out, s)
	}
	r.surfFrame = out

	// One instance rect per surface, addressed by offset in drawSurfaces.
	r.surfInst = r.surfInst[:0]
	for _, s := range out {
		r.surfInst = append(r.surfInst, float32(s.X), float32(s.Y), float32(s.W), float32(s.H))
	}
	if len(r.surfInst) > 0 {
		r.ensureGPUBuffer(&r.surfDrawBuffer, &r.surfDrawBufCap, uint64(len(r.surfInst))*4,
			"surf-instances", wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst)
		r.surfDrawBuffer.Write(r.queue, float32SliceToBytes(r.surfInst))
	}
	return out
}

func (r *WGPURenderer) rasterSurface(i int, src draw.SurfaceRasterizer, rect, clip image.Rectangle) draw.TextureID {
	for len(r.rasterSlots) <= i {
		r.rasterSlots = append(r.rasterSlots, &rasterSurfaceSlot{})
	}
	sl := r.rasterSlots[i]
	w, h := rect.Dx(), rect.Dy()
	if sl.w != w || sl.h != h || sl.tex == nil {
		if sl.view != nil {
			sl.view.Destroy()
		}
		if sl.tex != nil {
			sl.tex.Destroy()
		}
		sl.tex = r.device.CreateTexture(&wgpu.TextureDescriptor{
			Label:  "raster-surface",
			Size:   wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
			Format: wgpu.TextureFormatRGBA8Unorm,
			Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
		})
		sl.view = sl.tex.CreateView()
		sl.img = image.NewRGBA(image.Rect(0, 0, w, h))
		sl.straight = make([]byte, w*h*4)
		sl.w, sl.h = w, h
	}
	// The scratch image lives at the surface's device position, so the
	// provider draws with the same coordinates as on the CPU renderer.
	img := &image.RGBA{Pix: sl.img.Pix, Stride: sl.img.Stride, Rect: rect}
	clear(img.Pix)
	src.RasterizeSurface(img, rect, clip)

	// The surface pipeline blends straight alpha.
	for p := 0; p < len(img.Pix); p += 4 {
		a := img.Pix[p+3]
		switch a {
		case 0:
			sl.straight[p], sl.straight[p+1], sl.straight[p+2], sl.straight[p+3] = 0, 0, 0, 0
		case 255:
			copy(sl.straight[p:p+4], img.Pix[p:p+4])
		default:
			un := func(c uint8) uint8 { return uint8((uint32(c)*255 + uint32(a)/2) / uint32(a)) }
			sl.straight[p], sl.straight[p+1], sl.straight[p+2], sl.straight[p+3] = un(img.Pix[p]), un(img.Pix[p+1]), un(img.Pix[p+2]), a
		}
	}
	sl.tex.Write(r.queue, sl.straight, uint32(w*4))

	id := rasterSurfaceTexBase + draw.TextureID(i)
	r.surfaceTextures[id] = sl.view
	return id
}

func (r *WGPURenderer) destroyRasterSurfaces() {
	for _, sl := range r.rasterSlots {
		if sl.view != nil {
			sl.view.Destroy()
		}
		if sl.tex != nil {
			sl.tex.Destroy()
		}
	}
	r.rasterSlots = nil
}
