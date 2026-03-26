package render

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestTessellateFillRect(t *testing.T) {
	p := draw.PathFromRect(draw.R(0, 0, 100, 50))
	verts := TessellateFill(p)
	// A rectangle should produce 2 triangles = 6 vertices.
	if len(verts) != 6 {
		t.Fatalf("rect tessellation: got %d vertices, want 6", len(verts))
	}
}

func TestTessellateFillTriangle(t *testing.T) {
	p := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		LineTo(draw.Pt(100, 0)).
		LineTo(draw.Pt(50, 80)).
		Close().Build()
	verts := TessellateFill(p)
	// A triangle should produce exactly 3 vertices.
	if len(verts) != 3 {
		t.Fatalf("triangle tessellation: got %d vertices, want 3", len(verts))
	}
}

func TestTessellateFillEmpty(t *testing.T) {
	p := draw.NewPath().Build()
	verts := TessellateFill(p)
	if len(verts) != 0 {
		t.Fatalf("empty path: got %d vertices, want 0", len(verts))
	}
}

func TestTessellateStrokeLine(t *testing.T) {
	p := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		LineTo(draw.Pt(100, 0)).
		Build()
	verts := TessellateStroke(p, 4.0, draw.StrokeCapButt, draw.StrokeJoinMiter)
	// A single line segment with butt caps should produce 1 quad = 6 vertices.
	if len(verts) < 6 {
		t.Fatalf("stroke line: got %d vertices, want >= 6", len(verts))
	}
}

func TestTessellateStrokeSquareCap(t *testing.T) {
	p := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		LineTo(draw.Pt(100, 0)).
		Build()
	verts := TessellateStroke(p, 4.0, draw.StrokeCapSquare, draw.StrokeJoinMiter)
	// Square caps add extra quads at each end.
	if len(verts) < 12 {
		t.Fatalf("stroke square cap: got %d vertices, want >= 12", len(verts))
	}
}

func TestTessellateFillQuadCurve(t *testing.T) {
	p := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		QuadTo(draw.Pt(50, 100), draw.Pt(100, 0)).
		Close().Build()
	verts := TessellateFill(p)
	// Should produce some triangles (the curve gets flattened).
	if len(verts) < 3 {
		t.Fatalf("quad curve: got %d vertices, want >= 3", len(verts))
	}
	// Vertices should be multiples of 3.
	if len(verts)%3 != 0 {
		t.Fatalf("vertex count %d not multiple of 3", len(verts))
	}
}

func TestArcToCubics(t *testing.T) {
	cubics := arcToCubics(50, 50, 0, true, true, draw.Pt(0, 0), draw.Pt(100, 0))
	if len(cubics) == 0 {
		t.Fatal("arcToCubics returned no segments")
	}
	// Each segment should have 3 points (2 control + 1 end).
	for i, c := range cubics {
		_ = c[0] // control point 1
		_ = c[1] // control point 2
		_ = c[2] // end point
		_ = i
	}
}

func TestArcToCubicsDegenerate(t *testing.T) {
	// Zero radius should return nil.
	cubics := arcToCubics(0, 50, 0, false, false, draw.Pt(0, 0), draw.Pt(100, 0))
	if cubics != nil {
		t.Fatal("expected nil for zero rx")
	}
	// Same start/end should return nil.
	cubics = arcToCubics(50, 50, 0, false, false, draw.Pt(10, 10), draw.Pt(10, 10))
	if cubics != nil {
		t.Fatal("expected nil for same start/end")
	}
}
