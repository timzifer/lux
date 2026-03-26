package image

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestParseSVGPath_MoveTo(t *testing.T) {
	p, err := parseSVGPath("M 10 20")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var count int
	p.Walk(func(seg draw.PathSegment) {
		count++
		if seg.Kind != draw.SegMoveTo {
			t.Errorf("expected MoveTo, got %d", seg.Kind)
		}
		if seg.Points[0].X != 10 || seg.Points[0].Y != 20 {
			t.Errorf("MoveTo point = %v, want (10, 20)", seg.Points[0])
		}
	})
	if count != 1 {
		t.Fatalf("expected 1 segment, got %d", count)
	}
}

func TestParseSVGPath_Triangle(t *testing.T) {
	p, err := parseSVGPath("M 0 0 L 100 0 L 50 80 Z")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var kinds []draw.PathSegmentKind
	p.Walk(func(seg draw.PathSegment) {
		kinds = append(kinds, seg.Kind)
	})
	expected := []draw.PathSegmentKind{draw.SegMoveTo, draw.SegLineTo, draw.SegLineTo, draw.SegClose}
	if len(kinds) != len(expected) {
		t.Fatalf("got %d segments, want %d", len(kinds), len(expected))
	}
	for i, k := range kinds {
		if k != expected[i] {
			t.Errorf("segment %d: got %d, want %d", i, k, expected[i])
		}
	}
}

func TestParseSVGPath_Relative(t *testing.T) {
	p, err := parseSVGPath("M 10 10 l 20 0 l 0 20 z")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var points []draw.Point
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegMoveTo || seg.Kind == draw.SegLineTo {
			points = append(points, seg.Points[0])
		}
	})
	if len(points) != 3 {
		t.Fatalf("got %d points, want 3", len(points))
	}
	// M 10,10  l 20,0 → L 30,10  l 0,20 → L 30,30
	if points[1].X != 30 || points[1].Y != 10 {
		t.Errorf("point 1 = %v, want (30, 10)", points[1])
	}
	if points[2].X != 30 || points[2].Y != 30 {
		t.Errorf("point 2 = %v, want (30, 30)", points[2])
	}
}

func TestParseSVGPath_HV(t *testing.T) {
	p, err := parseSVGPath("M 0 0 H 50 V 30")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var points []draw.Point
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegLineTo {
			points = append(points, seg.Points[0])
		}
	})
	if len(points) != 2 {
		t.Fatalf("got %d line points, want 2", len(points))
	}
	if points[0].X != 50 || points[0].Y != 0 {
		t.Errorf("H point = %v, want (50, 0)", points[0])
	}
	if points[1].X != 50 || points[1].Y != 30 {
		t.Errorf("V point = %v, want (50, 30)", points[1])
	}
}

func TestParseSVGPath_Cubic(t *testing.T) {
	p, err := parseSVGPath("M 0 0 C 10 20 30 40 50 60")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var cubicSeen bool
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegCubicTo {
			cubicSeen = true
			if seg.Points[2].X != 50 || seg.Points[2].Y != 60 {
				t.Errorf("cubic end = %v, want (50, 60)", seg.Points[2])
			}
		}
	})
	if !cubicSeen {
		t.Fatal("no CubicTo segment")
	}
}

func TestParseSVGPath_Quad(t *testing.T) {
	p, err := parseSVGPath("M 0 0 Q 25 50 50 0")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var quadSeen bool
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegQuadTo {
			quadSeen = true
		}
	})
	if !quadSeen {
		t.Fatal("no QuadTo segment")
	}
}

func TestParseSVGPath_Arc(t *testing.T) {
	p, err := parseSVGPath("M 10 80 A 25 25 0 0 1 50 80")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var arcSeen bool
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegArcTo {
			arcSeen = true
			if seg.Arc.RX != 25 || seg.Arc.RY != 25 {
				t.Errorf("arc radii = (%f, %f), want (25, 25)", seg.Arc.RX, seg.Arc.RY)
			}
		}
	})
	if !arcSeen {
		t.Fatal("no ArcTo segment")
	}
}

func TestParseSVGPath_SmoothCubic(t *testing.T) {
	p, err := parseSVGPath("M 0 0 C 10 20 30 40 50 60 S 70 80 90 100")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var cubicCount int
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegCubicTo {
			cubicCount++
		}
	})
	if cubicCount != 2 {
		t.Fatalf("expected 2 cubics, got %d", cubicCount)
	}
}

func TestParseSVGPath_ImplicitLineTo(t *testing.T) {
	// After M, subsequent coordinate pairs become implicit L.
	p, err := parseSVGPath("M 0 0 10 10 20 20")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var kinds []draw.PathSegmentKind
	p.Walk(func(seg draw.PathSegment) {
		kinds = append(kinds, seg.Kind)
	})
	expected := []draw.PathSegmentKind{draw.SegMoveTo, draw.SegLineTo, draw.SegLineTo}
	if len(kinds) != len(expected) {
		t.Fatalf("got %d segments, want %d", len(kinds), len(expected))
	}
}

func TestParseSVGPath_Empty(t *testing.T) {
	p, err := parseSVGPath("")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	if !p.Empty() {
		t.Fatal("expected empty path for empty string")
	}
}

func TestParseSVGPath_CompactNotation(t *testing.T) {
	// No spaces between numbers (negative sign as separator).
	p, err := parseSVGPath("M10-20L30-40")
	if err != nil {
		t.Fatalf("parseSVGPath: %v", err)
	}
	var points []draw.Point
	p.Walk(func(seg draw.PathSegment) {
		if seg.Kind == draw.SegMoveTo || seg.Kind == draw.SegLineTo {
			points = append(points, seg.Points[0])
		}
	})
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}
	if points[0].X != 10 || points[0].Y != -20 {
		t.Errorf("M = %v, want (10, -20)", points[0])
	}
	if points[1].X != 30 || points[1].Y != -40 {
		t.Errorf("L = %v, want (30, -40)", points[1])
	}
}
