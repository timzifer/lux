package text

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// correctMSDFCorners applies Chlumsky-style error correction to an MSDF glyph image.
// It detects texels where median3(R,G,B) disagrees with the true inside/outside
// classification (determined via winding number on the glyph outline) and replaces
// those texels' RGB channels with the true signed distance. This eliminates the
// white/black corner artifacts that occur at sharp MSDF channel transitions.
//
// segments: glyph outline from sfnt.LoadGlyph at the same ppem as the MSDF.
// planeLeft/Top/Right/Bottom: PlaneBounds from the MSDF generator (pixel coords).
// pxRange: SDF distance field range (typically 4.0).
func correctMSDFCorners(img *image.NRGBA, segments sfnt.Segments,
	planeLeft, planeTop, planeRight, planeBottom float32, pxRange float32) {

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 || len(segments) == 0 {
		return
	}

	// Convert segments to float32 points once.
	segs := convertSegments(segments)
	if len(segs) == 0 {
		return
	}

	scaleX := (planeRight - planeLeft) / float32(w)
	scaleY := (planeBottom - planeTop) / float32(h)

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			// Map pixel center to outline coordinates.
			ox := planeLeft + (float32(px)+0.5)*scaleX
			oy := planeTop + (float32(py)+0.5)*scaleY

			// Read current MSDF channels (normalized to [0,1]).
			c := img.NRGBAAt(bounds.Min.X+px, bounds.Min.Y+py)
			r := float32(c.R) / 255.0
			g := float32(c.G) / 255.0
			b := float32(c.B) / 255.0

			med := median3f(r, g, b)

			// Corner artifacts have a unique signature: the three MSDF channels
			// diverge strongly because the edge-channel assignment changes at
			// the corner. Normal pixels (even at edges) have channels that
			// roughly agree. Only correct pixels with high channel spread.
			spread := max32(max32(r, g), b) - min32(min32(r, g), b)
			if spread < 0.5 {
				continue // Channels agree — not a corner artifact.
			}

			// Skip pixels in the anti-aliasing transition zone.
			threshold := 0.5 / pxRange
			if absf(med-0.5) < threshold {
				continue
			}

			medianInside := med >= 0.5

			// Check true inside/outside via winding number.
			trueInside := windingNumber(ox, oy, segs) != 0

			if medianInside == trueInside {
				continue // No artifact — skip.
			}

			// Artifact detected: compute true signed distance and correct.
			// SDF convention: inside > 0.5, outside < 0.5, edge = 0.5.
			// Signed distance: positive = inside, negative = outside.
			dist := minDistToSegments(ox, oy, segs)
			if !trueInside {
				dist = -dist
			}

			// Encode: 0.5 = edge, >0.5 = inside, <0.5 = outside.
			encoded := clampf(dist/pxRange+0.5, 0, 1)
			v := uint8(encoded * 255)
			img.SetNRGBA(bounds.Min.X+px, bounds.Min.Y+py,
				color.NRGBA{R: v, G: v, B: v, A: c.A})
		}
	}
}

// segment types for float32-converted outline data.
type segType int

const (
	segLine  segType = iota
	segQuad          // quadratic Bézier
	segCubic         // cubic Bézier
)

// seg holds a single outline segment with float32 control points.
type seg struct {
	typ        segType
	p0, p1, p2 [2]float32 // p2 used only for cubic; p1 for quad+cubic
	p3         [2]float32 // only for cubic
}

// convertSegments converts sfnt.Segments to float32 seg slices, tracking the
// current pen position via MoveTo/LineTo/QuadTo/CubeTo operations.
func convertSegments(segments sfnt.Segments) []seg {
	var result []seg
	var pen [2]float32

	for _, s := range segments {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			pen = [2]float32{f26_6ToF32(s.Args[0].X), f26_6ToF32(s.Args[0].Y)}

		case sfnt.SegmentOpLineTo:
			to := [2]float32{f26_6ToF32(s.Args[0].X), f26_6ToF32(s.Args[0].Y)}
			result = append(result, seg{typ: segLine, p0: pen, p1: to})
			pen = to

		case sfnt.SegmentOpQuadTo:
			ctrl := [2]float32{f26_6ToF32(s.Args[0].X), f26_6ToF32(s.Args[0].Y)}
			to := [2]float32{f26_6ToF32(s.Args[1].X), f26_6ToF32(s.Args[1].Y)}
			result = append(result, seg{typ: segQuad, p0: pen, p1: ctrl, p2: to})
			pen = to

		case sfnt.SegmentOpCubeTo:
			c1 := [2]float32{f26_6ToF32(s.Args[0].X), f26_6ToF32(s.Args[0].Y)}
			c2 := [2]float32{f26_6ToF32(s.Args[1].X), f26_6ToF32(s.Args[1].Y)}
			to := [2]float32{f26_6ToF32(s.Args[2].X), f26_6ToF32(s.Args[2].Y)}
			result = append(result, seg{typ: segCubic, p0: pen, p1: c1, p2: c2, p3: to})
			pen = to
		}
	}
	return result
}

// windingNumber computes the winding number for point (px, py) relative to the
// outline defined by segs. Non-zero means inside.
func windingNumber(px, py float32, segs []seg) int {
	wn := 0
	for i := range segs {
		switch segs[i].typ {
		case segLine:
			wn += windingLine(px, py, segs[i].p0, segs[i].p1)
		case segQuad:
			wn += windingQuad(px, py, segs[i].p0, segs[i].p1, segs[i].p2)
		case segCubic:
			wn += windingCubic(px, py, segs[i].p0, segs[i].p1, segs[i].p2, segs[i].p3)
		}
	}
	return wn
}

// windingLine computes the winding contribution of a line segment a→b
// for a horizontal ray from (px, py) to +∞.
func windingLine(px, py float32, a, b [2]float32) int {
	if a[1] <= py {
		if b[1] > py {
			if cross2d(b[0]-a[0], b[1]-a[1], px-a[0], py-a[1]) > 0 {
				return 1
			}
		}
	} else {
		if b[1] <= py {
			if cross2d(b[0]-a[0], b[1]-a[1], px-a[0], py-a[1]) < 0 {
				return -1
			}
		}
	}
	return 0
}

// windingQuad approximates the winding contribution of a quadratic Bézier by
// subdividing it into line segments.
func windingQuad(px, py float32, p0, p1, p2 [2]float32) int {
	const steps = 8
	wn := 0
	prev := p0
	for i := 1; i <= steps; i++ {
		t := float32(i) / steps
		cur := evalQuad(p0, p1, p2, t)
		wn += windingLine(px, py, prev, cur)
		prev = cur
	}
	return wn
}

// windingCubic approximates the winding contribution of a cubic Bézier.
func windingCubic(px, py float32, p0, p1, p2, p3 [2]float32) int {
	const steps = 12
	wn := 0
	prev := p0
	for i := 1; i <= steps; i++ {
		t := float32(i) / steps
		cur := evalCubic(p0, p1, p2, p3, t)
		wn += windingLine(px, py, prev, cur)
		prev = cur
	}
	return wn
}

// minDistToSegments returns the minimum unsigned distance from (px, py) to
// any segment in the outline.
func minDistToSegments(px, py float32, segs []seg) float32 {
	best := float32(math.MaxFloat32)
	for i := range segs {
		var d float32
		switch segs[i].typ {
		case segLine:
			d = distToLine(px, py, segs[i].p0, segs[i].p1)
		case segQuad:
			d = distToQuadBezier(px, py, segs[i].p0, segs[i].p1, segs[i].p2)
		case segCubic:
			d = distToCubicBezier(px, py, segs[i].p0, segs[i].p1, segs[i].p2, segs[i].p3)
		}
		if d < best {
			best = d
		}
	}
	return best
}

// distToLine returns the minimum distance from point (px, py) to line segment a→b.
func distToLine(px, py float32, a, b [2]float32) float32 {
	dx := b[0] - a[0]
	dy := b[1] - a[1]
	lenSq := dx*dx + dy*dy
	if lenSq < 1e-12 {
		return hypot(px-a[0], py-a[1])
	}
	t := ((px-a[0])*dx + (py-a[1])*dy) / lenSq
	t = clampf(t, 0, 1)
	cx := a[0] + t*dx
	cy := a[1] + t*dy
	return hypot(px-cx, py-cy)
}

// distToQuadBezier returns the minimum distance from (px, py) to quadratic Bézier p0,p1,p2.
// Uses sampling + Newton refinement for robustness.
func distToQuadBezier(px, py float32, p0, p1, p2 [2]float32) float32 {
	const samples = 16
	bestDist := float32(math.MaxFloat32)
	bestT := float32(0)

	// Coarse sampling to find approximate closest point.
	for i := 0; i <= samples; i++ {
		t := float32(i) / samples
		pt := evalQuad(p0, p1, p2, t)
		d := distSq(px, py, pt[0], pt[1])
		if d < bestDist {
			bestDist = d
			bestT = t
		}
	}

	// Newton-Raphson refinement (3 iterations).
	bestT = refineQuad(px, py, p0, p1, p2, bestT)
	pt := evalQuad(p0, p1, p2, bestT)
	return hypot(px-pt[0], py-pt[1])
}

// distToCubicBezier returns the minimum distance from (px, py) to cubic Bézier p0..p3.
func distToCubicBezier(px, py float32, p0, p1, p2, p3 [2]float32) float32 {
	const samples = 16
	bestDist := float32(math.MaxFloat32)
	bestT := float32(0)

	for i := 0; i <= samples; i++ {
		t := float32(i) / samples
		pt := evalCubic(p0, p1, p2, p3, t)
		d := distSq(px, py, pt[0], pt[1])
		if d < bestDist {
			bestDist = d
			bestT = t
		}
	}

	bestT = refineCubic(px, py, p0, p1, p2, p3, bestT)
	pt := evalCubic(p0, p1, p2, p3, bestT)
	return hypot(px-pt[0], py-pt[1])
}

// evalQuad evaluates a quadratic Bézier at parameter t.
func evalQuad(p0, p1, p2 [2]float32, t float32) [2]float32 {
	u := 1 - t
	return [2]float32{
		u*u*p0[0] + 2*u*t*p1[0] + t*t*p2[0],
		u*u*p0[1] + 2*u*t*p1[1] + t*t*p2[1],
	}
}

// evalQuadDeriv evaluates the derivative of a quadratic Bézier at t.
func evalQuadDeriv(p0, p1, p2 [2]float32, t float32) [2]float32 {
	u := 1 - t
	return [2]float32{
		2*u*(p1[0]-p0[0]) + 2*t*(p2[0]-p1[0]),
		2*u*(p1[1]-p0[1]) + 2*t*(p2[1]-p1[1]),
	}
}

// evalCubic evaluates a cubic Bézier at parameter t.
func evalCubic(p0, p1, p2, p3 [2]float32, t float32) [2]float32 {
	u := 1 - t
	u2 := u * u
	t2 := t * t
	return [2]float32{
		u2*u*p0[0] + 3*u2*t*p1[0] + 3*u*t2*p2[0] + t2*t*p3[0],
		u2*u*p0[1] + 3*u2*t*p1[1] + 3*u*t2*p2[1] + t2*t*p3[1],
	}
}

// evalCubicDeriv evaluates the derivative of a cubic Bézier at t.
func evalCubicDeriv(p0, p1, p2, p3 [2]float32, t float32) [2]float32 {
	u := 1 - t
	return [2]float32{
		3*u*u*(p1[0]-p0[0]) + 6*u*t*(p2[0]-p1[0]) + 3*t*t*(p3[0]-p2[0]),
		3*u*u*(p1[1]-p0[1]) + 6*u*t*(p2[1]-p1[1]) + 3*t*t*(p3[1]-p2[1]),
	}
}

// refineQuad performs Newton-Raphson refinement to find the closest point on
// a quadratic Bézier to (px, py), starting from parameter t0.
func refineQuad(px, py float32, p0, p1, p2 [2]float32, t0 float32) float32 {
	t := t0
	for i := 0; i < 4; i++ {
		pt := evalQuad(p0, p1, p2, t)
		dt := evalQuadDeriv(p0, p1, p2, t)
		// Minimize f(t) = |B(t) - P|^2, f'(t) = 2 * dot(B(t)-P, B'(t)).
		dx := pt[0] - px
		dy := pt[1] - py
		num := dx*dt[0] + dy*dt[1]
		// f''(t) ≈ dot(B'(t), B'(t)) + dot(B(t)-P, B''(t)); approximate with just |B'|^2.
		den := dt[0]*dt[0] + dt[1]*dt[1]
		if den < 1e-12 {
			break
		}
		t -= num / den
		t = clampf(t, 0, 1)
	}
	return t
}

// refineCubic performs Newton-Raphson refinement for a cubic Bézier.
func refineCubic(px, py float32, p0, p1, p2, p3 [2]float32, t0 float32) float32 {
	t := t0
	for i := 0; i < 4; i++ {
		pt := evalCubic(p0, p1, p2, p3, t)
		dt := evalCubicDeriv(p0, p1, p2, p3, t)
		dx := pt[0] - px
		dy := pt[1] - py
		num := dx*dt[0] + dy*dt[1]
		den := dt[0]*dt[0] + dt[1]*dt[1]
		if den < 1e-12 {
			break
		}
		t -= num / den
		t = clampf(t, 0, 1)
	}
	return t
}

// Helper functions.

func f26_6ToF32(v fixed.Int26_6) float32 {
	return float32(v) / 64.0
}

func cross2d(ax, ay, bx, by float32) float32 {
	return ax*by - ay*bx
}

func median3f(a, b, c float32) float32 {
	return max32(min32(a, b), min32(max32(a, b), c))
}

func hypot(x, y float32) float32 {
	return float32(math.Sqrt(float64(x*x + y*y)))
}

func distSq(ax, ay, bx, by float32) float32 {
	dx := ax - bx
	dy := ay - by
	return dx*dx + dy*dy
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
