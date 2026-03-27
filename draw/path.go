package draw

import "math"

// FillRule determines how a path interior is computed.
type FillRule uint8

const (
	FillRuleNonZero FillRule = iota
	FillRuleEvenOdd
)

// PathSegmentKind identifies the type of a path segment.
type PathSegmentKind uint8

const (
	SegMoveTo  PathSegmentKind = iota
	SegLineTo
	SegQuadTo
	SegCubicTo
	SegArcTo
	SegClose
)

// ArcParams holds the parameters for an SVG-style elliptical arc.
type ArcParams struct {
	RX, RY float32
	XRot   float32
	Large  bool
	Sweep  bool
}

// PathSegment is an exported path command for iteration via Walk.
type PathSegment struct {
	Kind   PathSegmentKind
	Points [3]Point  // endpoint(s) and control points
	Arc    ArcParams // valid only when Kind == SegArcTo
}

// pathCmd is an internal path segment.
type pathCmd struct {
	kind    pathCmdKind
	points  [3]Point // up to 3 control/end points
	hasArc  bool
	arcDesc arcDesc
}

type pathCmdKind uint8

const (
	pathCmdMoveTo pathCmdKind = iota
	pathCmdLineTo
	pathCmdQuadTo
	pathCmdCubicTo
	pathCmdArcTo
	pathCmdClose
)

type arcDesc struct {
	R     Size
	XRot  float32
	Large bool
	Sweep bool
}

// Path is an immutable geometric path.
type Path struct {
	FillRule FillRule
	cmds     []pathCmd
}

// PathBuilder constructs a Path incrementally.
type PathBuilder struct {
	fill FillRule
	cmds []pathCmd
}

// NewPath creates a PathBuilder with the default fill rule (NonZero).
func NewPath() *PathBuilder { return &PathBuilder{fill: FillRuleNonZero} }

func (b *PathBuilder) MoveTo(p Point) *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{kind: pathCmdMoveTo, points: [3]Point{p}})
	return b
}

func (b *PathBuilder) LineTo(p Point) *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{kind: pathCmdLineTo, points: [3]Point{p}})
	return b
}

func (b *PathBuilder) QuadTo(ctrl, end Point) *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{kind: pathCmdQuadTo, points: [3]Point{ctrl, end}})
	return b
}

func (b *PathBuilder) CubicTo(c1, c2, end Point) *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{kind: pathCmdCubicTo, points: [3]Point{c1, c2, end}})
	return b
}

// ArcTo appends an elliptical arc segment (SVG-style arc parameters).
// rx, ry are the ellipse radii, xRot is the X-axis rotation in degrees,
// large selects the large arc, sweep selects clockwise direction,
// and end is the arc endpoint.
func (b *PathBuilder) ArcTo(rx, ry, xRot float32, large, sweep bool, end Point) *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{
		kind:   pathCmdArcTo,
		points: [3]Point{end},
		hasArc: true,
		arcDesc: arcDesc{
			R:     Size{W: rx, H: ry},
			XRot:  xRot,
			Large: large,
			Sweep: sweep,
		},
	})
	return b
}

func (b *PathBuilder) Close() *PathBuilder {
	b.cmds = append(b.cmds, pathCmd{kind: pathCmdClose})
	return b
}

func (b *PathBuilder) Build() Path {
	out := Path{FillRule: b.fill, cmds: make([]pathCmd, len(b.cmds))}
	copy(out.cmds, b.cmds)
	return out
}

// SetFillRule sets the fill rule for the path being built.
func (b *PathBuilder) SetFillRule(r FillRule) *PathBuilder {
	b.fill = r
	return b
}

// PathFromRect creates a rectangular path.
func PathFromRect(r Rect) Path {
	return NewPath().
		MoveTo(Pt(r.X, r.Y)).
		LineTo(Pt(r.X+r.W, r.Y)).
		LineTo(Pt(r.X+r.W, r.Y+r.H)).
		LineTo(Pt(r.X, r.Y+r.H)).
		Close().Build()
}

// Empty reports whether the path contains no commands.
func (p Path) Empty() bool { return len(p.cmds) == 0 }

// Walk iterates over each segment of the path, calling fn for each.
func (p Path) Walk(fn func(PathSegment)) {
	for _, c := range p.cmds {
		seg := PathSegment{Points: c.points}
		switch c.kind {
		case pathCmdMoveTo:
			seg.Kind = SegMoveTo
		case pathCmdLineTo:
			seg.Kind = SegLineTo
		case pathCmdQuadTo:
			seg.Kind = SegQuadTo
		case pathCmdCubicTo:
			seg.Kind = SegCubicTo
		case pathCmdArcTo:
			seg.Kind = SegArcTo
			seg.Arc = ArcParams{
				RX:    c.arcDesc.R.W,
				RY:    c.arcDesc.R.H,
				XRot:  c.arcDesc.XRot,
				Large: c.arcDesc.Large,
				Sweep: c.arcDesc.Sweep,
			}
		case pathCmdClose:
			seg.Kind = SegClose
		}
		fn(seg)
	}
}

// Bounds computes the axis-aligned bounding box of the path.
// For curves, this uses control-point bounds (conservative estimate).
func (p Path) Bounds() Rect {
	if len(p.cmds) == 0 {
		return Rect{}
	}
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(-math.MaxFloat32)
	maxY := float32(-math.MaxFloat32)

	update := func(pt Point) {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}

	for _, c := range p.cmds {
		switch c.kind {
		case pathCmdMoveTo, pathCmdLineTo, pathCmdArcTo:
			update(c.points[0])
		case pathCmdQuadTo:
			update(c.points[0]) // control
			update(c.points[1]) // end
		case pathCmdCubicTo:
			update(c.points[0]) // control 1
			update(c.points[1]) // control 2
			update(c.points[2]) // end
		}
	}

	if minX > maxX {
		return Rect{}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}
