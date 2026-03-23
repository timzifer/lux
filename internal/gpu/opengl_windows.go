//go:build windows && !nogui

package gpu

import (
	"math"
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
)

var (
	gdi32                 = syscall.NewLazyDLL("gdi32.dll")
	user32gpu             = syscall.NewLazyDLL("user32.dll")
	procGetDC             = user32gpu.NewProc("GetDC")
	procReleaseDC         = user32gpu.NewProc("ReleaseDC")
	procSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// softImage holds CPU-side image data for the software renderer.
type softImage struct {
	width, height int
	rgba          []byte // RGBA8 pre-multiplied, row-major
}

// OpenGLRenderer implements the Windows M2 GDI software renderer.
type OpenGLRenderer struct {
	hwnd          uintptr
	width         int
	height        int
	pixels        []byte
	bgColor       draw.Color
	atlas         *text.GlyphAtlas
	imageTextures map[draw.ImageID]*softImage
}

// NewOpenGL creates the Windows software renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{
		imageTextures: make(map[draw.ImageID]*softImage),
	}
}

func (r *OpenGLRenderer) Init(cfg Config) error {
	r.hwnd = cfg.NativeHandle
	r.width = cfg.Width
	r.height = cfg.Height
	if r.width <= 0 {
		r.width = 800
	}
	if r.height <= 0 {
		r.height = 600
	}
	r.pixels = make([]byte, r.width*r.height*4)
	return nil
}

func (r *OpenGLRenderer) SetBackgroundColor(c draw.Color) {
	r.bgColor = c
}

// SetAtlas sets the glyph atlas for textured glyph rendering.
func (r *OpenGLRenderer) SetAtlas(a *text.GlyphAtlas) {
	r.atlas = a
}

func (r *OpenGLRenderer) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	r.width = width
	r.height = height
	r.pixels = make([]byte, r.width*r.height*4)
}

func (r *OpenGLRenderer) BeginFrame() {
	b := toBGRA(r.bgColor)
	for i := 0; i < len(r.pixels); i += 4 {
		r.pixels[i] = b[0]
		r.pixels[i+1] = b[1]
		r.pixels[i+2] = b[2]
		r.pixels[i+3] = b[3]
	}
}

func (r *OpenGLRenderer) Draw(scene draw.Scene) {
	for _, rect := range scene.Rects {
		if rect.Radius > 0 {
			r.fillRoundRect(rect.X, rect.Y, rect.W, rect.H, rect.Radius, rect.Color)
		} else {
			r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
		}
	}
	for _, glyph := range scene.Glyphs {
		r.drawGlyph(glyph)
	}
	for _, tg := range scene.TexturedGlyphs {
		r.drawTexturedGlyph(tg)
	}
	for _, img := range scene.ImageRects {
		r.drawImageRect(img)
	}
	// Overlay pass
	for _, rect := range scene.OverlayRects {
		if rect.Radius > 0 {
			r.fillRoundRect(rect.X, rect.Y, rect.W, rect.H, rect.Radius, rect.Color)
		} else {
			r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
		}
	}
	for _, glyph := range scene.OverlayGlyphs {
		r.drawGlyph(glyph)
	}
	for _, tg := range scene.OverlayTexturedGlyphs {
		r.drawTexturedGlyph(tg)
	}
	for _, img := range scene.OverlayImageRects {
		r.drawImageRect(img)
	}
}

func (r *OpenGLRenderer) EndFrame() {
	if r.hwnd == 0 {
		return
	}
	hdc, _, _ := procGetDC.Call(r.hwnd)
	if hdc == 0 {
		return
	}
	defer procReleaseDC.Call(r.hwnd, hdc)

	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(r.width),
		Height:   int32(r.height),
		Planes:   1,
		BitCount: 32,
	}
	procSetDIBitsToDevice.Call(
		hdc, 0, 0,
		uintptr(r.width), uintptr(r.height),
		0, 0, 0, uintptr(r.height),
		uintptr(unsafe.Pointer(&r.pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0,
	)
}

func (r *OpenGLRenderer) Destroy() { r.pixels = nil }

func (r *OpenGLRenderer) fillRect(x, y, w, h int, color draw.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	bgra := toBGRA(color)
	for row := 0; row < h; row++ {
		py := y + row
		if py < 0 || py >= r.height {
			continue
		}
		flippedY := r.height - 1 - py
		for col := 0; col < w; col++ {
			px := x + col
			if px < 0 || px >= r.width {
				continue
			}
			off := (flippedY*r.width + px) * 4
			r.pixels[off] = bgra[0]
			r.pixels[off+1] = bgra[1]
			r.pixels[off+2] = bgra[2]
			r.pixels[off+3] = bgra[3]
		}
	}
}

func (r *OpenGLRenderer) fillRoundRect(x, y, w, h int, radius float32, color draw.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	bgra := toBGRA(color)
	rad := radius
	// Clamp radius to half of the smaller dimension.
	halfW := float32(w) / 2
	halfH := float32(h) / 2
	if rad > halfW {
		rad = halfW
	}
	if rad > halfH {
		rad = halfH
	}

	for row := 0; row < h; row++ {
		py := y + row
		if py < 0 || py >= r.height {
			continue
		}
		flippedY := r.height - 1 - py
		for col := 0; col < w; col++ {
			px := x + col
			if px < 0 || px >= r.width {
				continue
			}

			// Check corner distance using pixel-center coordinates
			// (col+0.5, row+0.5) for symmetric top/bottom rounding.
			fx := float32(col) + 0.5
			fy := float32(row) + 0.5
			fw := float32(w)
			fh := float32(h)

			// Distance from the nearest corner circle center.
			var dx, dy float32
			if fx < rad {
				dx = rad - fx
			} else if fx > fw-rad {
				dx = fx - (fw - rad)
			}
			if fy < rad {
				dy = rad - fy
			} else if fy > fh-rad {
				dy = fy - (fh - rad)
			}

			// If in a corner region, check if outside the rounded corner.
			if dx > 0 && dy > 0 {
				if dx*dx+dy*dy > rad*rad {
					continue
				}
			}

			off := (flippedY*r.width + px) * 4
			r.pixels[off] = bgra[0]
			r.pixels[off+1] = bgra[1]
			r.pixels[off+2] = bgra[2]
			r.pixels[off+3] = bgra[3]
		}
	}
}

func (r *OpenGLRenderer) drawGlyph(cmd draw.DrawGlyph) {
	if cmd.Scale <= 0 {
		cmd.Scale = 1
	}
	render.RenderBitmapGlyph(cmd.Text, cmd.X, cmd.Y, cmd.Scale, func(px, py, w, h int) {
		r.fillRect(px, py, w, h, cmd.Color)
	})
}

// drawTexturedGlyph renders a single atlas-based glyph by alpha-blending
// atlas pixels into the software framebuffer.
func (r *OpenGLRenderer) drawTexturedGlyph(g draw.TexturedGlyph) {
	if r.atlas == nil {
		return
	}

	dstX := int(math.Round(float64(g.DstX)))
	dstY := int(math.Round(float64(g.DstY)))
	srcR := byte(g.Color.R * 255)
	srcG := byte(g.Color.G * 255)
	srcB := byte(g.Color.B * 255)

	for row := 0; row < g.SrcH; row++ {
		py := dstY + row
		if py < 0 || py >= r.height {
			continue
		}
		flippedY := r.height - 1 - py
		atlasRow := g.SrcY + row
		if atlasRow < 0 || atlasRow >= r.atlas.Height {
			continue
		}

		for col := 0; col < g.SrcW; col++ {
			px := dstX + col
			if px < 0 || px >= r.width {
				continue
			}
			atlasCol := g.SrcX + col
			if atlasCol < 0 || atlasCol >= r.atlas.Width {
				continue
			}

			// Read alpha from atlas.
			alpha := r.atlas.Image.Pix[atlasRow*r.atlas.Image.Stride+atlasCol]
			if alpha == 0 {
				continue
			}

			off := (flippedY*r.width + px) * 4
			a := uint16(alpha) * uint16(g.Color.A*255) / 255
			invA := 255 - a

			// Alpha-blend into framebuffer (BGRA order).
			r.pixels[off] = byte((uint16(srcB)*a + uint16(r.pixels[off])*invA) / 255)
			r.pixels[off+1] = byte((uint16(srcG)*a + uint16(r.pixels[off+1])*invA) / 255)
			r.pixels[off+2] = byte((uint16(srcR)*a + uint16(r.pixels[off+2])*invA) / 255)
			r.pixels[off+3] = 255
		}
	}
}

// UploadImage stores image data for software rendering (implements gpu.ImageUploader).
func (r *OpenGLRenderer) UploadImage(id draw.ImageID, width, height int, rgba []byte) {
	buf := make([]byte, len(rgba))
	copy(buf, rgba)
	r.imageTextures[id] = &softImage{width: width, height: height, rgba: buf}
}

// drawImageRect blits an image into the framebuffer with scale mode, UV clipping, and opacity.
func (r *OpenGLRenderer) drawImageRect(img draw.DrawImageRect) {
	si := r.imageTextures[img.ImageID]
	if si == nil || img.ImageID == 0 {
		return
	}
	dstW, dstH := img.W, img.H
	if dstW <= 0 || dstH <= 0 || si.width <= 0 || si.height <= 0 {
		return
	}
	opacity := img.Opacity
	if opacity <= 0 {
		return
	}
	if opacity > 1 {
		opacity = 1
	}

	// Clip rect from scene (0 means full viewport).
	clipX0, clipY0 := 0, 0
	clipX1, clipY1 := r.width, r.height
	if img.ClipW > 0 && img.ClipH > 0 {
		clipX0 = img.ClipX
		clipY0 = img.ClipY
		clipX1 = img.ClipX + img.ClipW
		clipY1 = img.ClipY + img.ClipH
	}

	// UV range from scene (may be sub-region due to clip intersection).
	u0, v0, u1, v1 := img.U0, img.V0, img.U1, img.V1

	// Compute draw rect and source sampling based on ScaleMode.
	drawX, drawY, drawW, drawH := img.X, img.Y, dstW, dstH
	// Source sampling region in source-image pixels.
	srcX0 := int(float64(si.width) * float64(u0))
	srcY0 := int(float64(si.height) * float64(v0))
	srcW := int(float64(si.width) * float64(u1-u0))
	srcH := int(float64(si.height) * float64(v1-v0))

	srcAspect := float64(si.width) / float64(si.height)
	dstAspect := float64(dstW) / float64(dstH)

	switch img.ScaleMode {
	case draw.ImageScaleFit:
		// Letterbox: scale to fit inside dst, preserving aspect ratio.
		var fitW, fitH int
		if srcAspect > dstAspect {
			fitW = dstW
			fitH = int(float64(dstW) / srcAspect)
		} else {
			fitH = dstH
			fitW = int(float64(dstH) * srcAspect)
		}
		drawX = img.X + (dstW-fitW)/2
		drawY = img.Y + (dstH-fitH)/2
		drawW = fitW
		drawH = fitH
		// Use full source for Fit.
		srcX0 = 0
		srcY0 = 0
		srcW = si.width
		srcH = si.height

	case draw.ImageScaleFill:
		// Crop: scale to cover dst, preserving aspect ratio.
		if srcAspect > dstAspect {
			cropW := int(float64(si.height) * dstAspect)
			srcX0 = (si.width - cropW) / 2
			srcW = cropW
		} else {
			cropH := int(float64(si.width) / dstAspect)
			srcY0 = (si.height - cropH) / 2
			srcH = cropH
		}
		srcY0 = 0
		srcH = si.height

	case draw.ImageScaleStretch:
		// Stretch uses the UV-defined sub-region as-is.
	}

	if drawW <= 0 || drawH <= 0 || srcW <= 0 || srcH <= 0 {
		return
	}

	for row := 0; row < drawH; row++ {
		py := drawY + row
		if py < clipY0 || py >= clipY1 || py < 0 || py >= r.height {
			continue
		}
		flippedY := r.height - 1 - py
		sy := srcY0 + row*srcH/drawH
		if sy < 0 {
			sy = 0
		}
		if sy >= si.height {
			sy = si.height - 1
		}
		for col := 0; col < drawW; col++ {
			px := drawX + col
			if px < clipX0 || px >= clipX1 || px < 0 || px >= r.width {
				continue
			}
			sx := srcX0 + col*srcW/drawW
			if sx < 0 {
				sx = 0
			}
			if sx >= si.width {
				sx = si.width - 1
			}
			srcOff := (sy*si.width + sx) * 4
			srcR := si.rgba[srcOff]
			srcG := si.rgba[srcOff+1]
			srcB := si.rgba[srcOff+2]
			srcA := si.rgba[srcOff+3]

			a := uint16(float32(srcA) * opacity)
			if a == 0 {
				continue
			}
			invA := 255 - a

			off := (flippedY*r.width + px) * 4
			r.pixels[off] = byte((uint16(srcB)*a + uint16(r.pixels[off])*invA) / 255)
			r.pixels[off+1] = byte((uint16(srcG)*a + uint16(r.pixels[off+1])*invA) / 255)
			r.pixels[off+2] = byte((uint16(srcR)*a + uint16(r.pixels[off+2])*invA) / 255)
			r.pixels[off+3] = 255
		}
	}
}

func toBGRA(c draw.Color) [4]byte {
	return [4]byte{
		byte(c.B * 255),
		byte(c.G * 255),
		byte(c.R * 255),
		byte(c.A * 255),
	}
}
