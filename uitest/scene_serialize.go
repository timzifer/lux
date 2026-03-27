// Package uitest provides golden-file snapshot testing for Lux UI scenes.
//
// Usage:
//
//	func TestMyWidget(t *testing.T) {
//	    scene := buildTestScene(myWidget(), 800, 600)
//	    uitest.AssertScene(t, scene, "testdata/my_widget.golden")
//	}
//
// Run with -update to regenerate golden files:
//
//	go test ./... -update
package uitest

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/timzifer/lux/draw"
)

// SerializeScene converts a Scene to a stable, human-readable text format.
// The output is deterministic: identical scenes always produce identical text.
func SerializeScene(s draw.Scene) string {
	var b strings.Builder

	b.WriteString("=== Scene ===\n")
	if s.Grain != 0 {
		b.WriteString(fmt.Sprintf("grain: %.4f\n", s.Grain))
	}
	b.WriteString("\n")

	writeRects(&b, "Rects", s.Rects)
	writeGlyphs(&b, "Glyphs", s.Glyphs)
	writeTexturedGlyphs(&b, "TexturedGlyphs", s.TexturedGlyphs)
	writeTexturedGlyphs(&b, "MSDFGlyphs", s.MSDFGlyphs)
	writeTexturedGlyphs(&b, "EmojiGlyphs", s.EmojiGlyphs)
	writeShadowRects(&b, "ShadowRects", s.ShadowRects)
	writeGradientRects(&b, "GradientRects", s.GradientRects)
	writeImageRects(&b, "ImageRects", s.ImageRects)
	writeShaderRects(&b, "ShaderRects", s.ShaderRects)
	writeSurfaces(&b, "Surfaces", s.Surfaces)

	// Overlay layers
	writeRects(&b, "OverlayRects", s.OverlayRects)
	writeGlyphs(&b, "OverlayGlyphs", s.OverlayGlyphs)
	writeTexturedGlyphs(&b, "OverlayTexturedGlyphs", s.OverlayTexturedGlyphs)
	writeTexturedGlyphs(&b, "OverlayMSDFGlyphs", s.OverlayMSDFGlyphs)
	writeTexturedGlyphs(&b, "OverlayEmojiGlyphs", s.OverlayEmojiGlyphs)
	writeShadowRects(&b, "OverlayShadowRects", s.OverlayShadowRects)
	writeGradientRects(&b, "OverlayGradientRects", s.OverlayGradientRects)
	writeImageRects(&b, "OverlayImageRects", s.OverlayImageRects)
	writeShaderRects(&b, "OverlayShaderRects", s.OverlayShaderRects)

	// Clip batches
	writeClipBatches(&b, "ClipBatches", s.ClipBatches)
	writeClipBatches(&b, "OverlayClipBatches", s.OverlayClipBatches)

	// Blur regions
	writeBlurRegions(&b, s.BlurRegions)

	return b.String()
}

// --- Section writers ---

func writeRects(b *strings.Builder, name string, rects []draw.DrawRect) {
	if len(rects) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(rects)))
	for i, r := range rects {
		b.WriteString(fmt.Sprintf("  %d: rect(%d,%d %dx%d) color%s",
			i, r.X, r.Y, r.W, r.H, fmtColor(r.Color)))
		if r.Radius != 0 {
			b.WriteString(fmt.Sprintf(" r=%.1f", r.Radius))
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func writeGlyphs(b *strings.Builder, name string, glyphs []draw.DrawGlyph) {
	if len(glyphs) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(glyphs)))
	for i, g := range glyphs {
		b.WriteString(fmt.Sprintf("  %d: glyph(%d,%d) scale=%d color%s %q\n",
			i, g.X, g.Y, g.Scale, fmtColor(g.Color), g.Text))
	}
	b.WriteByte('\n')
}

func writeTexturedGlyphs(b *strings.Builder, name string, glyphs []draw.TexturedGlyph) {
	if len(glyphs) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(glyphs)))
	for i, g := range glyphs {
		b.WriteString(fmt.Sprintf("  %d: dst(%.1f,%.1f %.1fx%.1f) src(%d,%d %dx%d) color%s\n",
			i, g.DstX, g.DstY, g.DstW, g.DstH,
			g.SrcX, g.SrcY, g.SrcW, g.SrcH, fmtColor(g.Color)))
	}
	b.WriteByte('\n')
}

func writeShadowRects(b *strings.Builder, name string, rects []draw.DrawShadowRect) {
	if len(rects) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(rects)))
	for i, r := range rects {
		inset := ""
		if r.Inset {
			inset = " inset"
		}
		b.WriteString(fmt.Sprintf("  %d: shadow(%d,%d %dx%d) color%s r=%.1f blur=%.1f%s\n",
			i, r.X, r.Y, r.W, r.H, fmtColor(r.Color), r.Radius, r.BlurRadius, inset))
	}
	b.WriteByte('\n')
}

func writeGradientRects(b *strings.Builder, name string, rects []draw.DrawGradientRect) {
	if len(rects) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(rects)))
	for i, r := range rects {
		kind := "linear"
		if r.Kind == draw.PaintRadialGradient {
			kind = "radial"
		}
		b.WriteString(fmt.Sprintf("  %d: gradient-%s(%d,%d %dx%d) r=%.1f stops=%d\n",
			i, kind, r.X, r.Y, r.W, r.H, r.Radius, r.StopCount))
		for j := 0; j < r.StopCount && j < len(r.Stops); j++ {
			s := r.Stops[j]
			b.WriteString(fmt.Sprintf("      stop[%d]: %.3f color%s\n",
				j, s.Offset, fmtColor(s.Color)))
		}
	}
	b.WriteByte('\n')
}

func writeImageRects(b *strings.Builder, name string, rects []draw.DrawImageRect) {
	if len(rects) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(rects)))
	for i, r := range rects {
		b.WriteString(fmt.Sprintf("  %d: image(%d,%d %dx%d) id=%d opacity=%.2f uv(%.2f,%.2f→%.2f,%.2f)\n",
			i, r.X, r.Y, r.W, r.H, r.ImageID, r.Opacity,
			r.U0, r.V0, r.U1, r.V1))
	}
	b.WriteByte('\n')
}

func writeShaderRects(b *strings.Builder, name string, rects []draw.DrawShaderRect) {
	if len(rects) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(rects)))
	for i, r := range rects {
		b.WriteString(fmt.Sprintf("  %d: shader(%d,%d %dx%d) key=%q r=%.1f\n",
			i, r.X, r.Y, r.W, r.H, r.ShaderKey, r.Radius))
	}
	b.WriteByte('\n')
}

func writeSurfaces(b *strings.Builder, name string, surfaces []draw.DrawSurface) {
	if len(surfaces) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(surfaces)))
	for i, s := range surfaces {
		b.WriteString(fmt.Sprintf("  %d: surface(%d,%d %dx%d) tex=%d surf=%d\n",
			i, s.X, s.Y, s.W, s.H, s.TextureID, s.SurfaceID))
	}
	b.WriteByte('\n')
}

func writeClipBatches(b *strings.Builder, name string, batches []draw.ClipBatch) {
	if len(batches) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[%s] (%d)\n", name, len(batches)))
	for i, cb := range batches {
		vp := ""
		if cb.FullViewport {
			vp = " full-viewport"
		}
		b.WriteString(fmt.Sprintf("  %d: clip(%.0f,%.0f %.0fx%.0f)%s rects@%d text@%d msdf@%d emoji@%d grad@%d shadow@%d img@%d shader@%d\n",
			i, cb.Clip.X, cb.Clip.Y, cb.Clip.W, cb.Clip.H, vp,
			cb.RectIdx, cb.TextIdx, cb.MSDFIdx, cb.EmojiIdx,
			cb.GradientIdx, cb.ShadowIdx, cb.ImageIdx, cb.ShaderIdx))
	}
	b.WriteByte('\n')
}

func writeBlurRegions(b *strings.Builder, regions []draw.BlurRegion) {
	if len(regions) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("[BlurRegions] (%d)\n", len(regions)))
	for i, r := range regions {
		b.WriteString(fmt.Sprintf("  %d: blur(%d,%d %dx%d) radius=%.1f\n",
			i, r.X, r.Y, r.W, r.H, r.Radius))
	}
	b.WriteByte('\n')
}

// --- Helpers ---

// fmtColor formats a Color as (R,G,B,A) with 8-bit values for readability.
func fmtColor(c draw.Color) string {
	r := clampByte(c.R)
	g := clampByte(c.G)
	b := clampByte(c.B)
	a := clampByte(c.A)
	return fmt.Sprintf("(%d,%d,%d,%d)", r, g, b, a)
}

func clampByte(f float32) uint8 {
	v := math.Round(float64(f) * 255)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// --- Diff ---

// DiffScenes returns a human-readable diff between two serialized scenes.
// Returns empty string if they are identical.
func DiffScenes(got, want string) string {
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")

	maxLen := len(gotLines)
	if len(wantLines) > maxLen {
		maxLen = len(wantLines)
	}

	var diffs []string
	for i := 0; i < maxLen; i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			diffs = append(diffs, fmt.Sprintf("  line %d:\n    got:  %s\n    want: %s", i+1, g, w))
		}
	}

	if len(diffs) == 0 {
		return ""
	}

	// Summarize
	summary := summarizeDiffs(got, want)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Scene diff (%d lines differ):\n", len(diffs)))
	if summary != "" {
		b.WriteString(summary)
		b.WriteByte('\n')
	}
	// Show first 20 diffs to keep output manageable
	limit := 20
	if len(diffs) < limit {
		limit = len(diffs)
	}
	for _, d := range diffs[:limit] {
		b.WriteString(d)
		b.WriteByte('\n')
	}
	if len(diffs) > 20 {
		b.WriteString(fmt.Sprintf("  ... and %d more lines\n", len(diffs)-20))
	}
	return b.String()
}

// summarizeDiffs provides a high-level summary of what changed between two scenes.
func summarizeDiffs(got, want string) string {
	type counts struct {
		name  string
		got   int
		want  int
	}

	sections := []string{
		"Rects", "Glyphs", "TexturedGlyphs", "MSDFGlyphs", "EmojiGlyphs",
		"ShadowRects", "GradientRects", "ImageRects", "ShaderRects", "Surfaces",
		"OverlayRects", "OverlayGlyphs", "OverlayTexturedGlyphs",
		"ClipBatches", "BlurRegions",
	}

	extract := func(text, section string) int {
		prefix := fmt.Sprintf("[%s] (", section)
		idx := strings.Index(text, prefix)
		if idx < 0 {
			return 0
		}
		rest := text[idx+len(prefix):]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return 0
		}
		var n int
		fmt.Sscanf(rest[:end], "%d", &n)
		return n
	}

	var changed []counts
	for _, s := range sections {
		g := extract(got, s)
		w := extract(want, s)
		if g != w {
			changed = append(changed, counts{s, g, w})
		}
	}

	if len(changed) == 0 {
		return ""
	}

	sort.Slice(changed, func(i, j int) bool {
		return changed[i].name < changed[j].name
	})

	var parts []string
	for _, c := range changed {
		parts = append(parts, fmt.Sprintf("  %s: got %d, want %d", c.name, c.got, c.want))
	}
	return "Count changes:\n" + strings.Join(parts, "\n")
}
