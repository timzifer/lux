// Package cerasurface embeds PDF pages rendered by github.com/timzifer/cera in a
// lux widget tree as external surfaces (RFC §8).
//
// A *Page is a ui.SurfaceProvider that rasterizes on the CPU
// (draw.SurfaceRasterizer): with the stilus CPU renderer the page is drawn
// straight into the framebuffer, without a copy, in bands on all cores;
// the GPU renderer has it drawn into a scratch image and uploads that.
//
//	doc, _ := cera.Open(pdf)
//	pdfPage, _ := doc.Page(0)
//	page := &cerasurface.Page{Page: pdfPage}
//	ui.Surface(1, page, 600, 800)
package cerasurface

import (
	"context"
	"image"
	"image/color"
	"time"

	"github.com/timzifer/cera"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Page shows one PDF page in a lux surface.
type Page struct {
	// Page is the PDF page to draw.
	Page *cera.Page
	// Scale is device pixels per PDF point; 0 fits the page width into the
	// surface.
	Scale float64
	// Scroll is the device pixel of the rendered page shown at the top left
	// corner of the surface.
	Scroll image.Point
	// Background is the paper; the zero value is white. Pixels of the surface
	// outside the page are left as they are.
	Background color.RGBA
	// Workers bounds the goroutines drawing bands of the page; 0 is
	// GOMAXPROCS (see cera.RenderOptions).
	Workers int
	// Budget bounds the time one frame may spend on the page; 0 is none.
	// A page cut short shows what was drawn and sets Err to cera.ErrDeadline.
	Budget time.Duration
	// OnMsg, if set, receives the surface's input messages
	// (ui.SurfaceMouseMsg, ui.SurfaceKeyMsg …) and reports whether it used them.
	OnMsg func(msg any) bool

	// Err is the error of the last render, if any (the page was drawn partly).
	Err error
}

var (
	_ ui.SurfaceProvider     = (*Page)(nil)
	_ draw.SurfaceRasterizer = (*Page)(nil)
)

// RasterizeSurface draws the page into dst over rect, limited to clip.
func (p *Page) RasterizeSurface(dst *image.RGBA, rect, clip image.Rectangle) {
	if p == nil || p.Page == nil || rect.Empty() {
		return
	}
	scale := p.Scale
	if scale <= 0 {
		w, _ := p.Page.Size()
		if w <= 0 {
			return
		}
		scale = float64(rect.Dx()) / w
	}
	// The page's device pixel (0, 0) sits at origin in dst.
	origin := rect.Min.Sub(p.Scroll)
	region := clip.Intersect(dst.Rect).Sub(origin).Intersect(p.Page.Bounds(scale))
	if region.Empty() {
		return
	}
	// cera addresses dst in page pixels: alias the framebuffer so that
	// page pixel q is dst pixel q+origin. No copy, no allocation.
	start := region.Min.Add(origin)
	page := &image.RGBA{
		Pix:    dst.Pix[dst.PixOffset(start.X, start.Y):],
		Stride: dst.Stride,
		Rect:   region,
	}
	bg := p.Background
	if bg == (color.RGBA{}) {
		bg = color.RGBA{255, 255, 255, 255}
	}
	opt := cera.RenderOptions{Scale: scale, Region: region, Background: bg, Workers: p.Workers}
	if p.Budget > 0 {
		opt.Deadline = time.Now().Add(p.Budget)
	}
	p.Err = p.Page.Render(context.Background(), page, opt)
}

// AcquireFrame is not used for drawing: the page is rasterized
// (RasterizeSurface). It returns no texture.
func (p *Page) AcquireFrame(draw.Rect) (draw.TextureID, ui.FrameToken) { return 0, 0 }

// ReleaseFrame does nothing.
func (p *Page) ReleaseFrame(ui.FrameToken) {}

// HandleMsg forwards input to OnMsg.
func (p *Page) HandleMsg(msg any) bool {
	if p.OnMsg != nil {
		return p.OnMsg(msg)
	}
	return false
}
