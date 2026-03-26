package draw

import "testing"

func TestPathBuilderBasic(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		LineTo(Pt(100, 0)).
		LineTo(Pt(100, 100)).
		Close().
		Build()

	if len(p.cmds) != 4 {
		t.Fatalf("expected 4 cmds, got %d", len(p.cmds))
	}
	if p.cmds[0].kind != pathCmdMoveTo {
		t.Error("first cmd should be MoveTo")
	}
	if p.cmds[3].kind != pathCmdClose {
		t.Error("last cmd should be Close")
	}
}

func TestPathBuilderArcTo(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		ArcTo(50, 50, 0, true, true, Pt(100, 0)).
		Close().
		Build()

	if len(p.cmds) != 3 {
		t.Fatalf("expected 3 cmds, got %d", len(p.cmds))
	}

	arc := p.cmds[1]
	if arc.kind != pathCmdArcTo {
		t.Fatalf("cmd[1] kind = %d, want pathCmdArcTo (%d)", arc.kind, pathCmdArcTo)
	}
	if !arc.hasArc {
		t.Error("arc cmd should have hasArc = true")
	}
	if arc.arcDesc.R.W != 50 || arc.arcDesc.R.H != 50 {
		t.Errorf("arc radii = (%f, %f), want (50, 50)", arc.arcDesc.R.W, arc.arcDesc.R.H)
	}
	if !arc.arcDesc.Large {
		t.Error("arc should have Large = true")
	}
	if !arc.arcDesc.Sweep {
		t.Error("arc should have Sweep = true")
	}
	if arc.points[0] != (Point{X: 100, Y: 0}) {
		t.Errorf("arc endpoint = %v, want (100, 0)", arc.points[0])
	}
}

func TestPathBuilderQuadAndCubic(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		QuadTo(Pt(50, 50), Pt(100, 0)).
		CubicTo(Pt(120, 40), Pt(180, 40), Pt(200, 0)).
		Build()

	if len(p.cmds) != 3 {
		t.Fatalf("expected 3 cmds, got %d", len(p.cmds))
	}
	if p.cmds[1].kind != pathCmdQuadTo {
		t.Error("cmd[1] should be QuadTo")
	}
	if p.cmds[2].kind != pathCmdCubicTo {
		t.Error("cmd[2] should be CubicTo")
	}
}

func TestPathFromRect(t *testing.T) {
	p := PathFromRect(R(10, 20, 100, 50))
	// MoveTo + 3 LineTo + Close = 5
	if len(p.cmds) != 5 {
		t.Fatalf("expected 5 cmds, got %d", len(p.cmds))
	}
}

func TestPathBuilderFillRule(t *testing.T) {
	p := NewPath().Build()
	if p.FillRule != FillRuleNonZero {
		t.Errorf("default FillRule = %d, want FillRuleNonZero (%d)", p.FillRule, FillRuleNonZero)
	}
}

func TestPathEmpty(t *testing.T) {
	p := NewPath().Build()
	if !p.Empty() {
		t.Fatal("expected empty path")
	}
	p2 := NewPath().MoveTo(Pt(0, 0)).Build()
	if p2.Empty() {
		t.Fatal("expected non-empty path")
	}
}

func TestSetFillRule(t *testing.T) {
	p := NewPath().SetFillRule(FillRuleEvenOdd).
		MoveTo(Pt(0, 0)).LineTo(Pt(10, 10)).Close().Build()
	if p.FillRule != FillRuleEvenOdd {
		t.Fatalf("FillRule = %d, want EvenOdd(%d)", p.FillRule, FillRuleEvenOdd)
	}
}

func TestPathWalk(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		LineTo(Pt(10, 0)).
		QuadTo(Pt(15, 5), Pt(10, 10)).
		CubicTo(Pt(5, 15), Pt(0, 15), Pt(0, 10)).
		Close().
		Build()

	var kinds []PathSegmentKind
	p.Walk(func(seg PathSegment) {
		kinds = append(kinds, seg.Kind)
	})

	expected := []PathSegmentKind{SegMoveTo, SegLineTo, SegQuadTo, SegCubicTo, SegClose}
	if len(kinds) != len(expected) {
		t.Fatalf("Walk: got %d segments, want %d", len(kinds), len(expected))
	}
	for i, k := range kinds {
		if k != expected[i] {
			t.Errorf("segment %d: got kind %d, want %d", i, k, expected[i])
		}
	}
}

func TestPathWalkArc(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		ArcTo(10, 10, 0, false, true, Pt(20, 0)).
		Build()

	var arcSeen bool
	p.Walk(func(seg PathSegment) {
		if seg.Kind == SegArcTo {
			arcSeen = true
			if seg.Arc.RX != 10 || seg.Arc.RY != 10 {
				t.Errorf("arc: RX=%f, RY=%f, want 10, 10", seg.Arc.RX, seg.Arc.RY)
			}
			if seg.Arc.Large {
				t.Error("arc: expected large=false")
			}
			if !seg.Arc.Sweep {
				t.Error("arc: expected sweep=true")
			}
		}
	})
	if !arcSeen {
		t.Fatal("no ArcTo segment found")
	}
}

func TestPathBoundsEmpty(t *testing.T) {
	p := NewPath().Build()
	b := p.Bounds()
	if b != (Rect{}) {
		t.Fatalf("empty path bounds = %v, want zero", b)
	}
}

func TestPathBoundsRect(t *testing.T) {
	p := PathFromRect(R(10, 20, 30, 40))
	b := p.Bounds()
	if b.X != 10 || b.Y != 20 || b.W != 30 || b.H != 40 {
		t.Fatalf("rect path bounds = %v, want {10, 20, 30, 40}", b)
	}
}

func TestPathBoundsTriangle(t *testing.T) {
	p := NewPath().
		MoveTo(Pt(0, 0)).
		LineTo(Pt(100, 0)).
		LineTo(Pt(50, 80)).
		Close().Build()
	b := p.Bounds()
	if b.X != 0 || b.Y != 0 || b.W != 100 || b.H != 80 {
		t.Fatalf("triangle bounds = %v, want {0, 0, 100, 80}", b)
	}
}
