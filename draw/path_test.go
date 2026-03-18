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
