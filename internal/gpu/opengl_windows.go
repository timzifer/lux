//go:build windows && !nogui

package gpu

import (
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

// OpenGLRenderer implements the Windows M2 GDI software renderer.
type OpenGLRenderer struct {
	hwnd    uintptr
	width   int
	height  int
	pixels  []byte
	bgColor draw.Color
	atlas   *text.GlyphAtlas
}

// NewOpenGL creates the Windows software renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
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
		r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
	}
	for _, glyph := range scene.Glyphs {
		r.drawGlyph(glyph)
	}
	for _, tg := range scene.TexturedGlyphs {
		r.drawTexturedGlyph(tg)
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

	dstX := int(g.DstX)
	dstY := int(g.DstY)
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

func toBGRA(c draw.Color) [4]byte {
	return [4]byte{
		byte(c.B * 255),
		byte(c.G * 255),
		byte(c.R * 255),
		byte(c.A * 255),
	}
}
