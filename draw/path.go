package draw

// FillRule determines how a path interior is computed.
type FillRule uint8

const (
	FillRuleNonZero FillRule = iota
	FillRuleEvenOdd
)

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

// PathFromRect creates a rectangular path.
func PathFromRect(r Rect) Path {
	return NewPath().
		MoveTo(Pt(r.X, r.Y)).
		LineTo(Pt(r.X+r.W, r.Y)).
		LineTo(Pt(r.X+r.W, r.Y+r.H)).
		LineTo(Pt(r.X, r.Y+r.H)).
		Close().Build()
}
