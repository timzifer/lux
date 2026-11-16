//go:build !nogui && ((darwin && arm64) || windows || gogpu)

// CPU (stilus) against GPU (wgpu, offscreen) on the same scenes.
//
//	CGO_ENABLED=0 go test -run x -bench Compare -benchtime 2s ./internal/gpu/
//
// GPU numbers come in three flavours:
//   - submit: Draw only — CPU time the app thread spends encoding a frame.
//   - frame:  Draw + WaitIdle — until the GPU has finished the frame.
//   - pixels: Draw + ReadPixels — the frame as host memory (tests, CI,
//     screenshots), comparable to what the CPU renderer delivers.
package gpu_test

import (
	"fmt"
	"image"
	"io"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/fonts"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/effects"
	"github.com/timzifer/lux/ui/layout"
)

type benchScene struct {
	name  string
	scene draw.Scene
	atlas *text.GlyphAtlas
}

func buildScene(root ui.Element, w, h int) (draw.Scene, *text.GlyphAtlas) {
	atlas := text.NewGlyphAtlas(512, 512)
	shaper := text.NewGoTextShaper(fonts.Fallback)
	shaper.RegisterFamily(fonts.PhosphorFamily)
	build := func() draw.Scene {
		c := render.NewSceneCanvas(w, h, render.WithShaper(shaper), render.WithAtlas(atlas))
		return ui.BuildScene(root, c, theme.Default, w, h, nil)
	}
	s := build()
	// Let async MSDF glyphs arrive, then rebuild with them.
	for i := 0; i < 20; i++ {
		time.Sleep(25 * time.Millisecond)
		if atlas.DrainMSDFResults() {
			s = build()
			i = 0
		}
	}
	return s, atlas
}

func card(i int) ui.Element {
	return display.Card(layout.Column(
		display.TextStyled(fmt.Sprintf("Sensor %d", i), draw.TextStyle{Size: 28}),
		display.Text("Temperature 21.4 °C · Humidity 48 % · Pressure 1013 hPa"),
		display.Text("Last update 12:04:33, all systems nominal."),
		layout.Row(button.Text("Details", nil), button.Text("Acknowledge", nil)),
		display.GradientRect(260, 12, 6, draw.LinearGradientPaint(draw.Pt(0, 0), draw.Pt(260, 0),
			draw.GradientStop{Offset: 0, Color: draw.Hex("#22c55e")},
			draw.GradientStop{Offset: 1, Color: draw.Hex("#ef4444")})),
	))
}

func dashboard(cols, rows int) ui.Element {
	var cs []ui.Element
	for c := 0; c < cols; c++ {
		var rs []ui.Element
		for r := 0; r < rows; r++ {
			rs = append(rs, card(c*rows+r))
		}
		cs = append(cs, layout.Column(rs...))
	}
	return layout.Row(cs...)
}

func scenes(w, h int) []benchScene {
	var out []benchScene
	add := func(name string, root ui.Element) {
		s, a := buildScene(root, w, h)
		out = append(out, benchScene{name, s, a})
	}
	cols, rows := w/330, h/230
	add("dashboard", dashboard(cols, rows))
	add("dashboard+blur", layout.NewStack(dashboard(cols, rows),
		effects.FrostedGlass(16, draw.Color{R: 1, G: 1, B: 1, A: 0.1}, display.Spacer(float32(h)))))

	var rects draw.Scene
	for i := 0; i < 2000; i++ {
		rects.Rects = append(rects.Rects, draw.DrawRect{
			X: (i * 37) % (w - 40), Y: (i * 53) % (h - 30), W: 40, H: 30, Radius: 6,
			Color: draw.Color{R: 0.2, G: 0.4, B: 0.8, A: 0.8},
		})
	}
	out = append(out, benchScene{name: "2000-rects", scene: rects})

	var shadows draw.Scene
	for i := 0; i < 100; i++ {
		x, y := (i%10)*(w/10), (i/10)*(h/10)
		shadows.ShadowRects = append(shadows.ShadowRects, draw.DrawShadowRect{
			X: x, Y: y, W: w/10 - 8, H: h/10 - 8, Radius: 8, BlurRadius: 12, Color: draw.Color{A: 0.35},
		})
		shadows.Rects = append(shadows.Rects, draw.DrawRect{X: x + 12, Y: y + 12, W: w/10 - 32, H: h/10 - 32, Radius: 8, Color: draw.Color{R: 1, G: 1, B: 1, A: 1}})
	}
	out = append(out, benchScene{name: "100-shadows", scene: shadows})
	return out
}

// lockThread pins the benchmark goroutine to its OS thread for the GPU
// renderer's lifetime, as the app's render loop is: Metal autorelease pools
// belong to the thread that made them.
func lockThread(b *testing.B) {
	runtime.LockOSThread()
	b.Cleanup(runtime.UnlockOSThread)
}

func quietLog(b *testing.B) {
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(nil) })
}

func BenchmarkCompare(b *testing.B) {
	quietLog(b)
	for _, sz := range []image.Point{{800, 600}, {1920, 1080}} {
		w, h := sz.X, sz.Y
		for _, sc := range scenes(w, h) {
			prefix := fmt.Sprintf("%s/%dx%d", sc.name, w, h)

			for _, workers := range []int{0, 1} {
				name := "/cpu-stilus"
				if workers == 1 {
					name = "/cpu-stilus-1core"
				}
				b.Run(prefix+name, func(b *testing.B) {
					r := gpu.NewStilus()
					r.Workers = workers
					_ = r.Init(gpu.Config{Width: w, Height: h})
					r.SetAtlas(sc.atlas)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						r.BeginFrame()
						r.Draw(sc.scene)
						r.EndFrame()
					}
				})
			}

			newGPU := func(b *testing.B) *gpu.WGPURenderer {
				lockThread(b)
				r := gpu.NewWGPU()
				if err := r.Init(gpu.Config{Width: w, Height: h, DRMfd: -1, Offscreen: true}); err != nil {
					b.Skipf("no GPU: %v", err)
				}
				b.Cleanup(r.Destroy)
				r.SetAtlas(sc.atlas)
				// Warm up: pipelines, atlas upload.
				r.BeginFrame()
				r.Draw(sc.scene)
				r.EndFrame()
				if err := r.WaitIdle(); err != nil {
					b.Skipf("no readback: %v", err)
				}
				return r
			}

			b.Run(prefix+"/gpu-submit", func(b *testing.B) {
				r := newGPU(b)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r.BeginFrame()
					r.Draw(sc.scene)
					r.EndFrame()
					if i%8 == 7 { // keep the queue bounded, outside the timing
						b.StopTimer()
						_ = r.WaitIdle()
						b.StartTimer()
					}
				}
				b.StopTimer()
				_ = r.WaitIdle()
			})

			b.Run(prefix+"/gpu-frame", func(b *testing.B) {
				r := newGPU(b)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r.BeginFrame()
					r.Draw(sc.scene)
					r.EndFrame()
					_ = r.WaitIdle()
				}
			})

			b.Run(prefix+"/gpu-pixels", func(b *testing.B) {
				r := newGPU(b)
				img := image.NewRGBA(image.Rect(0, 0, w, h))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r.BeginFrame()
					r.Draw(sc.scene)
					r.EndFrame()
					if err := r.ReadPixels(img); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// heavySurface stands in for an external CPU surface (a PDF page): it
// fills its clip opaquely, row by row.
type heavySurface struct{}

func (heavySurface) RasterizeSurface(dst *image.RGBA, rect, clip image.Rectangle) {
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		row := dst.Pix[dst.PixOffset(clip.Min.X, y):dst.PixOffset(clip.Max.X, y)]
		for x := 0; x < len(row); x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = uint8(y), uint8(x), 128, 255
		}
	}
}

// BenchmarkCompareSurface measures a full-frame CPU surface: drawn in place
// by the CPU renderer, drawn to scratch + uploaded by the GPU renderer.
func BenchmarkCompareSurface(b *testing.B) {
	quietLog(b)
	for _, sz := range []image.Point{{800, 600}, {1920, 1080}} {
		w, h := sz.X, sz.Y
		scene := draw.Scene{Surfaces: []draw.DrawSurface{{X: 0, Y: 0, W: w, H: h, Raster: heavySurface{}}}}
		prefix := fmt.Sprintf("surface/%dx%d", w, h)
		b.Run(prefix+"/cpu-stilus", func(b *testing.B) {
			r := gpu.NewStilus()
			_ = r.Init(gpu.Config{Width: w, Height: h})
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.BeginFrame()
				r.Draw(scene)
			}
		})
		b.Run(prefix+"/gpu-frame", func(b *testing.B) {
			lockThread(b)
			r := gpu.NewWGPU()
			if err := r.Init(gpu.Config{Width: w, Height: h, DRMfd: -1, Offscreen: true}); err != nil {
				b.Skip(err)
			}
			defer r.Destroy()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.BeginFrame()
				r.Draw(scene)
				r.EndFrame()
				_ = r.WaitIdle()
			}
		})
	}
}
