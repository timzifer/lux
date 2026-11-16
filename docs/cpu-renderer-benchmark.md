# CPU renderer (stilus) vs. GPU renderer (wgpu)

Apple M4 (10 cores), macOS, Go 1.25.1, `CGO_ENABLED=0`, wgpu via gogpu/Metal,
2026-11-16. Median of 6 runs (`benchstat`), milliseconds per frame.

```sh
CGO_ENABLED=0 go test -run x -bench Compare -count 6 ./internal/gpu/
cd contrib/cerasurface && go test -run x -bench . -count 6
```

## What is measured

| Column | Meaning |
|---|---|
| CPU | `StilusRenderer`, bands on all cores (`Workers = 0`) — the frame is in host memory |
| CPU 1 core | `Workers = 1` (WASM, small embedded) |
| GPU submit | `WGPURenderer.Draw` only — CPU time of the render thread |
| GPU frame | `Draw` + wait until the GPU is done |
| GPU pixels | `Draw` + readback into an `*image.RGBA` (tests, CI, screenshots, remote) |

Scenes: `dashboard` — cards with text (bitmap + 28 px MSDF), buttons, gradients and
card shadows filling the frame; `+blur` adds a full-frame frosted-glass blur (r = 16);
`2000-rects` — rounded, translucent rects; `100-shadows` — soft shadows (r = 12) with
cards on top. The default theme's grain is on.

## Results

| Scene | Size | CPU | CPU 1 core | GPU submit | GPU frame | GPU pixels |
|---|---|---:|---:|---:|---:|---:|
| dashboard | 800×600 | 0.86 | 3.2 | 0.36 | 0.37 | 1.49 |
| dashboard + blur | 800×600 | 6.8 | 28.8 | 1.7 | 1.6 | 2.0 |
| 2000 rects | 800×600 | 2.05 | 7.6 | 0.55 | 0.64 | 1.83 |
| 100 shadows | 800×600 | 1.22 | 4.1 | 0.36 | 0.38 | 1.59 |
| dashboard | 1920×1080 | 4.3 | 15.8 | 0.83 | 0.88 | 6.0 |
| dashboard + blur | 1920×1080 | 24.1 | 100.2 | 2.4 | 4.2 | 7.3 |
| 2000 rects | 1920×1080 | 2.4 | 7.8 | 0.65 | 0.91 | 7.6 |
| 100 shadows | 1920×1080 | 3.3 | 11.8 | 0.56 | 0.55 | 5.9 |

External CPU surface covering the frame (`draw.SurfaceRasterizer`):

| Size | CPU (drawn in place) | GPU frame (scratch + upload) |
|---|---:|---:|
| 800×600 | 0.39 | 1.44 |
| 1920×1080 | 1.55 | 5.44 |

PDF page through `contrib/cerasurface` (synthetic technical drawing: grid, 3000
segments, 60 circles, 120 text lines; 1600×1130 view in a 1920×1200 frame, display list
cached): **1.27 ms** on all cores, 6.1 ms on one core, 11 / 2 allocations per frame.

GPU numbers vary more between runs (CI up to ±50 % for `submit`); CPU numbers within ±7 %.

## Reading

- **On screen the GPU stays faster**: a finished GPU frame takes 2–9× less time than the
  CPU on all cores. The CPU renderer is not a replacement for the GPU on machines that
  have one.
- **When the pixels are needed in memory**, the CPU renderer is on par or faster than GPU
  + readback without blur: 1080p dashboard 4.3 vs 6.0 ms, 2000 rects 2.4 vs 7.6 ms,
  shadows 3.3 vs 5.9 ms. Headless tests, CI, screenshots and remote/VNC frames take this
  path.
- **CPU surfaces** (PDF, software video, charts) are 3.5× cheaper on the CPU renderer,
  which draws them into the frame without a copy, than on the GPU, which has to upload.
- **Blur is the CPU's weak spot** (exact Gaussian, O(radius) per pixel): 24 ms at 1080p
  on all cores. A box-blur approximation (O(1)) would be the next step if it matters.
- Without a GPU (CI, VMs, embedded without a driver, WASM without WebGPU) the CPU renderer
  keeps a 1080p dashboard at 4 ms on 10 cores, 16 ms on one.

## Optimisation during the work

First version, one core, 1080p dashboard: 40.8 ms. Banded rendering on all cores
(`stilus.Canvas` per band, byte-identical to one band), a lookup table for shadow
falloff, integer paths for grain and glyphs, and a parallel blur brought it to 15.8 ms
on one core and 4.3 ms on ten.
