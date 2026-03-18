package draw

import "testing"

func TestSolidPaintKind(t *testing.T) {
	p := SolidPaint(Hex("#ff0000"))
	if p.Kind != PaintSolid {
		t.Errorf("Kind = %d, want PaintSolid (%d)", p.Kind, PaintSolid)
	}
	if p.Color != Hex("#ff0000") {
		t.Error("SolidPaint color mismatch")
	}
}

func TestLinearGradientPaint(t *testing.T) {
	p := LinearGradientPaint(
		Pt(0, 0), Pt(100, 0),
		GradientStop{Offset: 0, Color: Hex("#ff0000")},
		GradientStop{Offset: 1, Color: Hex("#0000ff")},
	)
	if p.Kind != PaintLinearGradient {
		t.Errorf("Kind = %d, want PaintLinearGradient (%d)", p.Kind, PaintLinearGradient)
	}
	if p.Linear == nil {
		t.Fatal("Linear should not be nil")
	}
	if len(p.Linear.Stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(p.Linear.Stops))
	}
	if p.Linear.Start != (Point{X: 0, Y: 0}) {
		t.Errorf("Start = %v, want (0,0)", p.Linear.Start)
	}
	if p.Linear.End != (Point{X: 100, Y: 0}) {
		t.Errorf("End = %v, want (100,0)", p.Linear.End)
	}
}

func TestRadialGradientPaint(t *testing.T) {
	p := RadialGradientPaint(
		Pt(50, 50), 100,
		GradientStop{Offset: 0, Color: Hex("#ffffff")},
		GradientStop{Offset: 1, Color: Hex("#000000")},
	)
	if p.Kind != PaintRadialGradient {
		t.Errorf("Kind = %d, want PaintRadialGradient (%d)", p.Kind, PaintRadialGradient)
	}
	if p.Radial == nil {
		t.Fatal("Radial should not be nil")
	}
	if p.Radial.Radius != 100 {
		t.Errorf("Radius = %f, want 100", p.Radial.Radius)
	}
	if p.Radial.Center != (Point{X: 50, Y: 50}) {
		t.Errorf("Center = %v, want (50,50)", p.Radial.Center)
	}
}

func TestPatternPaint(t *testing.T) {
	p := PatternPaint(ImageID(42), Size{W: 16, H: 16})
	if p.Kind != PaintPattern {
		t.Errorf("Kind = %d, want PaintPattern (%d)", p.Kind, PaintPattern)
	}
	if p.Pattern == nil {
		t.Fatal("Pattern should not be nil")
	}
	if p.Pattern.Image != 42 {
		t.Errorf("Image = %d, want 42", p.Pattern.Image)
	}
	if p.Pattern.TileSize.W != 16 || p.Pattern.TileSize.H != 16 {
		t.Errorf("TileSize = %v, want (16,16)", p.Pattern.TileSize)
	}
}

func TestFallbackColorSolid(t *testing.T) {
	p := SolidPaint(Hex("#ff0000"))
	c := p.FallbackColor()
	if c != Hex("#ff0000") {
		t.Error("FallbackColor for solid should return the color")
	}
}

func TestFallbackColorLinearGradient(t *testing.T) {
	red := Hex("#ff0000")
	p := LinearGradientPaint(Pt(0, 0), Pt(100, 0),
		GradientStop{Offset: 0, Color: red},
		GradientStop{Offset: 1, Color: Hex("#0000ff")},
	)
	c := p.FallbackColor()
	if c != red {
		t.Error("FallbackColor for linear gradient should return first stop color")
	}
}

func TestFallbackColorRadialGradient(t *testing.T) {
	white := Hex("#ffffff")
	p := RadialGradientPaint(Pt(50, 50), 100,
		GradientStop{Offset: 0, Color: white},
	)
	c := p.FallbackColor()
	if c != white {
		t.Error("FallbackColor for radial gradient should return first stop color")
	}
}

func TestFallbackColorEmptyGradient(t *testing.T) {
	p := LinearGradientPaint(Pt(0, 0), Pt(100, 0))
	c := p.FallbackColor()
	if c != (Color{}) {
		t.Error("FallbackColor for empty gradient should return zero color")
	}
}

func TestGradientStopOrder(t *testing.T) {
	stops := []GradientStop{
		{Offset: 0.0, Color: Hex("#ff0000")},
		{Offset: 0.5, Color: Hex("#00ff00")},
		{Offset: 1.0, Color: Hex("#0000ff")},
	}
	p := LinearGradientPaint(Pt(0, 0), Pt(100, 0), stops...)
	for i, s := range p.Linear.Stops {
		if s != stops[i] {
			t.Errorf("stop[%d] mismatch", i)
		}
	}
}
