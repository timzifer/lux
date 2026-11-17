package uitest

import (
	"image"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// msdfSettle bounds how long RenderImage waits for asynchronously
// rasterized MSDF glyphs (large text) before drawing the final frame.
const msdfSettle = 2 * time.Second

// RenderImage lays out root with the default theme, shapes its text with the
// bundled fonts and rasterizes the resulting scene on the CPU with the stilus
// backend. The returned image is premultiplied RGBA of width × height pixels.
//
// Unlike BuildScene, which yields the draw list, RenderImage yields pixels,
// so it covers the backend as well: text atlases, clipping, shadows,
// gradients, blur and overlays.
func RenderImage(root ui.Element, width, height int) *image.RGBA {
	return RenderImageWithTheme(root, theme.Default, width, height)
}

// RenderImageWithTheme is RenderImage with an explicit theme.
func RenderImageWithTheme(root ui.Element, th theme.Theme, width, height int) *image.RGBA {
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewGoTextShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)
	shaper.RegisterFamily(fonts.NotoEmojiFamily)

	ready := make(chan struct{}, 1)
	atlas.SetMSDFNotify(func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})

	build := func() draw.Scene {
		canvas := render.NewSceneCanvas(width, height, render.WithShaper(shaper), render.WithAtlas(atlas))
		return ui.BuildScene(root, canvas, th, width, height, nil)
	}

	// Large text is rasterized as MSDF in the background; rebuild until no
	// further glyphs arrive so the frame is complete and deterministic.
	scene := build()
	deadline := time.Now().Add(msdfSettle)
	for time.Now().Before(deadline) {
		select {
		case <-ready:
			// Let the rest of the batch finish before draining.
			time.Sleep(20 * time.Millisecond)
			atlas.DrainMSDFResults()
			scene = build()
			continue
		case <-time.After(100 * time.Millisecond):
		}
		if !atlas.DrainMSDFResults() {
			break
		}
		scene = build()
	}

	r := gpu.NewStilus()
	_ = r.Init(gpu.Config{Width: width, Height: height})
	r.SetAtlas(atlas)
	r.SetBackgroundColor(th.Tokens().Colors.Surface.Base)
	r.BeginFrame()
	r.Draw(scene)
	r.EndFrame()
	return r.Image()
}
