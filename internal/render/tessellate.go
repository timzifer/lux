// Package render provides the scene-building Canvas implementation.
//
// This file implements CPU-side path tessellation: converting draw.Path
// into triangle meshes ([]draw.PathVertex) for GPU rendering.
package render

import (
	"math"

	"github.com/timzifer/lux/draw"
)

// flattenTolerance controls the maximum error when flattening curves to
// line segments. Smaller values produce smoother curves with more vertices.
const flattenTolerance = 0.25

// TessellateFill converts a filled path into triangles using ear-clipping
// on the flattened polygon. The path is first flattened (curves → lines),
// then triangulated.
func TessellateFill(p draw.Path) []draw.PathVertex {
	polys := flattenPath(p)
	var verts []draw.PathVertex
	for _, poly := range polys {
		verts = append(verts, triangulatePoly(poly)...)
	}
	return verts
}

// TessellateStroke converts a stroked path into triangles by expanding
// the path outline by the stroke width.
func TessellateStroke(p draw.Path, width float32, cap draw.StrokeCap, join draw.StrokeJoin) []draw.PathVertex {
	polys := flattenPath(p)
	var verts []draw.PathVertex
	halfW := width / 2
	for _, poly := range polys {
		if len(poly) < 2 {
			continue
		}
		verts = append(verts, strokePoly(poly, halfW, cap, join)...)
	}
	return verts
}

// flattenPath converts all path segments into polylines (one per sub-path).
func flattenPath(p draw.Path) [][]draw.Point {
	var polys [][]draw.Point
	var current []draw.Point
	var cursor draw.Point
	var subStart draw.Point

	p.Walk(func(seg draw.PathSegment) {
		switch seg.Kind {
		case draw.SegMoveTo:
			if len(current) > 1 {
				polys = append(polys, current)
			}
			cursor = seg.Points[0]
			subStart = cursor
			current = []draw.Point{cursor}

		case draw.SegLineTo:
			cursor = seg.Points[0]
			current = append(current, cursor)

		case draw.SegQuadTo:
			pts := flattenQuad(cursor, seg.Points[0], seg.Points[1], flattenTolerance)
			current = append(current, pts...)
			cursor = seg.Points[1]

		case draw.SegCubicTo:
			pts := flattenCubic(cursor, seg.Points[0], seg.Points[1], seg.Points[2], flattenTolerance)
			current = append(current, pts...)
			cursor = seg.Points[2]

		case draw.SegArcTo:
			cubics := arcToCubics(
				seg.Arc.RX, seg.Arc.RY, seg.Arc.XRot,
				seg.Arc.Large, seg.Arc.Sweep,
				cursor, seg.Points[0],
			)
			for _, c := range cubics {
				pts := flattenCubic(cursor, c[0], c[1], c[2], flattenTolerance)
				current = append(current, pts...)
				cursor = c[2]
			}

		case draw.SegClose:
			if len(current) > 0 {
				cursor = subStart
				current = append(current, cursor)
			}
		}
	})

	if len(current) > 1 {
		polys = append(polys, current)
	}
	return polys
}

// flattenQuad flattens a quadratic Bezier into line segments.
func flattenQuad(p0, p1, p2 draw.Point, tol float32) []draw.Point {
	// Check if curve is flat enough.
	dx := p0.X - 2*p1.X + p2.X
	dy := p0.Y - 2*p1.Y + p2.Y
	if dx*dx+dy*dy <= tol*tol {
		return []draw.Point{p2}
	}
	// Subdivide at t=0.5.
	m01 := mid(p0, p1)
	m12 := mid(p1, p2)
	m := mid(m01, m12)
	var pts []draw.Point
	pts = append(pts, flattenQuad(p0, m01, m, tol)...)
	pts = append(pts, flattenQuad(m, m12, p2, tol)...)
	return pts
}

// flattenCubic flattens a cubic Bezier into line segments.
func flattenCubic(p0, p1, p2, p3 draw.Point, tol float32) []draw.Point {
	// Check flatness: max distance of control points from line p0→p3.
	dx := p3.X - p0.X
	dy := p3.Y - p0.Y
	d1 := abs32((p1.X-p3.X)*dy - (p1.Y-p3.Y)*dx)
	d2 := abs32((p2.X-p3.X)*dy - (p2.Y-p3.Y)*dx)
	if (d1+d2)*(d1+d2) <= tol*tol*(dx*dx+dy*dy) {
		return []draw.Point{p3}
	}
	// Subdivide at t=0.5 (de Casteljau).
	m01 := mid(p0, p1)
	m12 := mid(p1, p2)
	m23 := mid(p2, p3)
	m012 := mid(m01, m12)
	m123 := mid(m12, m23)
	m := mid(m012, m123)
	var pts []draw.Point
	pts = append(pts, flattenCubic(p0, m01, m012, m, tol)...)
	pts = append(pts, flattenCubic(m, m123, m23, p3, tol)...)
	return pts
}

func mid(a, b draw.Point) draw.Point {
	return draw.Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// triangulatePoly triangulates a simple polygon using ear-clipping.
func triangulatePoly(poly []draw.Point) []draw.PathVertex {
	n := len(poly)
	// Remove duplicate closing point if present.
	if n >= 2 && poly[0] == poly[n-1] {
		poly = poly[:n-1]
		n--
	}
	if n < 3 {
		return nil
	}

	// Build index list.
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// Ensure winding is counter-clockwise for ear-clipping.
	if signedArea(poly) < 0 {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			indices[i], indices[j] = indices[j], indices[i]
		}
	}

	var verts []draw.PathVertex
	for len(indices) > 3 {
		earFound := false
		nIdx := len(indices)
		for i := 0; i < nIdx; i++ {
			prev := indices[(i+nIdx-1)%nIdx]
			curr := indices[i]
			next := indices[(i+1)%nIdx]

			if !isConvex(poly[prev], poly[curr], poly[next]) {
				continue
			}
			if anyPointInTriangle(poly, indices, prev, curr, next) {
				continue
			}
			// Found an ear — emit triangle.
			verts = append(verts,
				draw.PathVertex{X: poly[prev].X, Y: poly[prev].Y},
				draw.PathVertex{X: poly[curr].X, Y: poly[curr].Y},
				draw.PathVertex{X: poly[next].X, Y: poly[next].Y},
			)
			// Remove curr from indices.
			indices = append(indices[:i], indices[i+1:]...)
			earFound = true
			break
		}
		if !earFound {
			// Degenerate polygon — emit remaining as fan.
			break
		}
	}
	// Emit last triangle.
	if len(indices) == 3 {
		verts = append(verts,
			draw.PathVertex{X: poly[indices[0]].X, Y: poly[indices[0]].Y},
			draw.PathVertex{X: poly[indices[1]].X, Y: poly[indices[1]].Y},
			draw.PathVertex{X: poly[indices[2]].X, Y: poly[indices[2]].Y},
		)
	}
	return verts
}

func signedArea(poly []draw.Point) float32 {
	var area float32
	n := len(poly)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += poly[i].X * poly[j].Y
		area -= poly[j].X * poly[i].Y
	}
	return area / 2
}

func isConvex(a, b, c draw.Point) bool {
	return cross2D(b.X-a.X, b.Y-a.Y, c.X-b.X, c.Y-b.Y) > 0
}

func cross2D(ax, ay, bx, by float32) float32 {
	return ax*by - ay*bx
}

func anyPointInTriangle(poly []draw.Point, indices []int, a, b, c int) bool {
	for _, idx := range indices {
		if idx == a || idx == b || idx == c {
			continue
		}
		if pointInTriangle(poly[idx], poly[a], poly[b], poly[c]) {
			return true
		}
	}
	return false
}

func pointInTriangle(p, a, b, c draw.Point) bool {
	d1 := cross2D(b.X-a.X, b.Y-a.Y, p.X-a.X, p.Y-a.Y)
	d2 := cross2D(c.X-b.X, c.Y-b.Y, p.X-b.X, p.Y-b.Y)
	d3 := cross2D(a.X-c.X, a.Y-c.Y, p.X-c.X, p.Y-c.Y)
	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(hasNeg && hasPos)
}

// strokePoly generates triangles for a stroked polyline.
func strokePoly(poly []draw.Point, halfW float32, cap draw.StrokeCap, join draw.StrokeJoin) []draw.PathVertex {
	n := len(poly)
	if n < 2 {
		return nil
	}

	closed := poly[0] == poly[n-1]
	var verts []draw.PathVertex

	// Generate offset points on both sides of each segment.
	type seg struct {
		l0, l1, r0, r1 draw.Point
	}
	segs := make([]seg, 0, n-1)
	for i := 0; i < n-1; i++ {
		dx := poly[i+1].X - poly[i].X
		dy := poly[i+1].Y - poly[i].Y
		ln := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if ln < 1e-6 {
			continue
		}
		nx := -dy / ln * halfW
		ny := dx / ln * halfW
		segs = append(segs, seg{
			l0: draw.Point{X: poly[i].X + nx, Y: poly[i].Y + ny},
			l1: draw.Point{X: poly[i+1].X + nx, Y: poly[i+1].Y + ny},
			r0: draw.Point{X: poly[i].X - nx, Y: poly[i].Y - ny},
			r1: draw.Point{X: poly[i+1].X - nx, Y: poly[i+1].Y - ny},
		})
	}
	if len(segs) == 0 {
		return nil
	}

	// Emit quads (2 triangles each) for each segment.
	for _, s := range segs {
		verts = append(verts,
			draw.PathVertex{X: s.l0.X, Y: s.l0.Y},
			draw.PathVertex{X: s.r0.X, Y: s.r0.Y},
			draw.PathVertex{X: s.l1.X, Y: s.l1.Y},
			draw.PathVertex{X: s.l1.X, Y: s.l1.Y},
			draw.PathVertex{X: s.r0.X, Y: s.r0.Y},
			draw.PathVertex{X: s.r1.X, Y: s.r1.Y},
		)
	}

	// Emit join triangles between consecutive segments.
	for i := 1; i < len(segs); i++ {
		prev := segs[i-1]
		curr := segs[i]
		center := poly[i] // approximate — works for the segment index mapping
		// Find the right index: segs may have skipped degenerate segments.
		// For simplicity, use the join point as the segment endpoint.
		_ = center
		verts = append(verts, joinTriangles(prev.l1, prev.r1, curr.l0, curr.r0, join)...)
	}

	// Join last↔first for closed paths.
	if closed && len(segs) > 1 {
		prev := segs[len(segs)-1]
		curr := segs[0]
		verts = append(verts, joinTriangles(prev.l1, prev.r1, curr.l0, curr.r0, join)...)
	}

	// Emit caps for open paths.
	if !closed {
		verts = append(verts, capTriangles(poly[0], segs[0].l0, segs[0].r0, halfW, cap, true)...)
		last := segs[len(segs)-1]
		verts = append(verts, capTriangles(poly[n-1], last.l1, last.r1, halfW, cap, false)...)
	}

	return verts
}

// joinTriangles emits triangles for a line join between two consecutive segments.
func joinTriangles(prevL, prevR, currL, currR draw.Point, join draw.StrokeJoin) []draw.PathVertex {
	// Simple bevel join: connect the four offset points with two triangles.
	_ = join // miter/round can be added later; bevel is the baseline
	return []draw.PathVertex{
		{X: prevL.X, Y: prevL.Y},
		{X: prevR.X, Y: prevR.Y},
		{X: currL.X, Y: currL.Y},
		{X: currL.X, Y: currL.Y},
		{X: prevR.X, Y: prevR.Y},
		{X: currR.X, Y: currR.Y},
	}
}

// capTriangles emits triangles for a line cap.
func capTriangles(center, left, right draw.Point, halfW float32, cap draw.StrokeCap, isStart bool) []draw.PathVertex {
	switch cap {
	case draw.StrokeCapSquare:
		// Extend by halfW in the line direction.
		dx := left.X - right.X
		dy := left.Y - right.Y
		ln := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if ln < 1e-6 {
			return nil
		}
		// Perpendicular to left-right direction = along the line.
		nx := -dy / ln * halfW
		ny := dx / ln * halfW
		if isStart {
			nx, ny = -nx, -ny
		}
		el := draw.Point{X: left.X + nx, Y: left.Y + ny}
		er := draw.Point{X: right.X + nx, Y: right.Y + ny}
		return []draw.PathVertex{
			{X: left.X, Y: left.Y},
			{X: right.X, Y: right.Y},
			{X: el.X, Y: el.Y},
			{X: el.X, Y: el.Y},
			{X: right.X, Y: right.Y},
			{X: er.X, Y: er.Y},
		}

	case draw.StrokeCapRound:
		// Approximate with a fan of triangles.
		var verts []draw.PathVertex
		steps := 8
		for i := 0; i < steps; i++ {
			a0 := float64(i) / float64(steps) * math.Pi
			a1 := float64(i+1) / float64(steps) * math.Pi
			if !isStart {
				a0 += math.Pi
				a1 += math.Pi
			}
			// Rotate around center using perpendicular to left-right.
			dx := left.X - right.X
			dy := left.Y - right.Y
			ln := float64(math.Sqrt(float64(dx*dx + dy*dy)))
			if ln < 1e-6 {
				break
			}
			// Base angle from left-right direction.
			baseAngle := math.Atan2(float64(dy), float64(dx))
			r := float64(halfW)
			p0 := draw.Point{
				X: center.X + float32(math.Cos(baseAngle+a0)*r),
				Y: center.Y + float32(math.Sin(baseAngle+a0)*r),
			}
			p1 := draw.Point{
				X: center.X + float32(math.Cos(baseAngle+a1)*r),
				Y: center.Y + float32(math.Sin(baseAngle+a1)*r),
			}
			verts = append(verts,
				draw.PathVertex{X: center.X, Y: center.Y},
				draw.PathVertex{X: p0.X, Y: p0.Y},
				draw.PathVertex{X: p1.X, Y: p1.Y},
			)
		}
		return verts

	default: // StrokeCapButt — no cap needed
		return nil
	}
}

// arcToCubics converts an SVG-style elliptical arc to cubic Bezier segments.
// Uses the standard SVG arc → center parameterization algorithm.
func arcToCubics(rx, ry, xRotDeg float32, largeArc, sweep bool, from, to draw.Point) [][3]draw.Point {
	// Handle degenerate cases.
	if rx == 0 || ry == 0 {
		return nil // degenerate — treat as line
	}
	if from == to {
		return nil
	}

	rx = abs32(rx)
	ry = abs32(ry)

	xRot := float64(xRotDeg) * math.Pi / 180.0
	cosR := float32(math.Cos(xRot))
	sinR := float32(math.Sin(xRot))

	// Step 1: Compute (x1', y1') — rotated midpoint.
	dx := (from.X - to.X) / 2
	dy := (from.Y - to.Y) / 2
	x1p := cosR*dx + sinR*dy
	y1p := -sinR*dx + cosR*dy

	// Step 2: Compute center point (cx', cy').
	x1p2 := x1p * x1p
	y1p2 := y1p * y1p
	rx2 := rx * rx
	ry2 := ry * ry

	// Scale radii if too small.
	lambda := x1p2/rx2 + y1p2/ry2
	if lambda > 1 {
		s := float32(math.Sqrt(float64(lambda)))
		rx *= s
		ry *= s
		rx2 = rx * rx
		ry2 = ry * ry
	}

	num := rx2*ry2 - rx2*y1p2 - ry2*x1p2
	den := rx2*y1p2 + ry2*x1p2
	if den < 1e-10 {
		return nil
	}
	sq := float32(0)
	if num > 0 {
		sq = float32(math.Sqrt(float64(num / den)))
	}
	if largeArc == sweep {
		sq = -sq
	}
	cxp := sq * rx * y1p / ry
	cyp := -sq * ry * x1p / rx

	// Step 3: Compute center (cx, cy) in original coords.
	mx := (from.X + to.X) / 2
	my := (from.Y + to.Y) / 2
	cx := cosR*cxp - sinR*cyp + mx
	cy := sinR*cxp + cosR*cyp + my

	// Step 4: Compute start and sweep angles.
	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := (-x1p - cxp) / rx
	vy := (-y1p - cyp) / ry

	theta1 := angle(1, 0, float64(ux), float64(uy))
	dtheta := angle(float64(ux), float64(uy), float64(vx), float64(vy))

	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	// Step 5: Split into cubic segments (max 90° each).
	nSegs := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	if nSegs == 0 {
		return nil
	}
	segAngle := dtheta / float64(nSegs)

	var cubics [][3]draw.Point
	for i := 0; i < nSegs; i++ {
		a1 := theta1 + float64(i)*segAngle
		a2 := theta1 + float64(i+1)*segAngle
		cubics = append(cubics, arcSegToCubic(cx, cy, rx, ry, cosR, sinR, float32(a1), float32(a2)))
	}
	return cubics
}

// arcSegToCubic converts a single arc segment (≤90°) to a cubic Bezier.
func arcSegToCubic(cx, cy, rx, ry, cosR, sinR, a1, a2 float32) [3]draw.Point {
	da := a2 - a1
	alpha := float32(math.Sin(float64(da))) * (float32(math.Sqrt(float64(4+3*tan32(da/2)*tan32(da/2)))) - 1) / 3

	cos1 := float32(math.Cos(float64(a1)))
	sin1 := float32(math.Sin(float64(a1)))
	cos2 := float32(math.Cos(float64(a2)))
	sin2 := float32(math.Sin(float64(a2)))

	// Endpoint 1 derivatives.
	ex1 := rx * cos1
	ey1 := ry * sin1
	dx1 := -rx * sin1
	dy1 := ry * cos1

	// Endpoint 2 derivatives.
	ex2 := rx * cos2
	ey2 := ry * sin2
	dx2 := -rx * sin2
	dy2 := ry * cos2

	// Control points in rotated space.
	cp1x := ex1 + alpha*dx1
	cp1y := ey1 + alpha*dy1
	cp2x := ex2 - alpha*dx2
	cp2y := ey2 - alpha*dy2
	endx := ex2
	endy := ey2

	// Transform back to original space.
	return [3]draw.Point{
		{X: cosR*cp1x - sinR*cp1y + cx, Y: sinR*cp1x + cosR*cp1y + cy},
		{X: cosR*cp2x - sinR*cp2y + cx, Y: sinR*cp2x + cosR*cp2y + cy},
		{X: cosR*endx - sinR*endy + cx, Y: sinR*endx + cosR*endy + cy},
	}
}

func angle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	det := ux*vy - uy*vx
	return math.Atan2(det, dot)
}

func tan32(x float32) float32 {
	return float32(math.Tan(float64(x)))
}
