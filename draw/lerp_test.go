package draw

import (
	"math"
	"testing"
)

func approx(a, b, eps float32) bool {
	return float32(math.Abs(float64(a-b))) <= eps
}

func TestLerpColor(t *testing.T) {
	a := Color{R: 0, G: 0, B: 0, A: 1}
	b := Color{R: 1, G: 0.5, B: 0.2, A: 0.8}

	// t=0 → a
	c := LerpColor(a, b, 0)
	if c != a {
		t.Errorf("LerpColor(a,b,0) = %v, want %v", c, a)
	}

	// t=1 → b
	c = LerpColor(a, b, 1)
	if c != b {
		t.Errorf("LerpColor(a,b,1) = %v, want %v", c, b)
	}

	// t=0.5 → midpoint
	c = LerpColor(a, b, 0.5)
	if !approx(c.R, 0.5, 0.01) || !approx(c.G, 0.25, 0.01) ||
		!approx(c.B, 0.1, 0.01) || !approx(c.A, 0.9, 0.01) {
		t.Errorf("LerpColor midpoint = %v", c)
	}
}

func TestLerpPoint(t *testing.T) {
	a := Pt(0, 0)
	b := Pt(100, 200)
	p := LerpPoint(a, b, 0.5)
	if !approx(p.X, 50, 0.01) || !approx(p.Y, 100, 0.01) {
		t.Errorf("LerpPoint midpoint = %v", p)
	}
}

func TestLerpSize(t *testing.T) {
	a := Size{W: 10, H: 20}
	b := Size{W: 110, H: 220}
	s := LerpSize(a, b, 0.5)
	if !approx(s.W, 60, 0.01) || !approx(s.H, 120, 0.01) {
		t.Errorf("LerpSize midpoint = %v", s)
	}
}

func TestLerpRect(t *testing.T) {
	a := R(0, 0, 100, 100)
	b := R(100, 200, 200, 300)
	r := LerpRect(a, b, 0.5)
	if !approx(r.X, 50, 0.01) || !approx(r.Y, 100, 0.01) ||
		!approx(r.W, 150, 0.01) || !approx(r.H, 200, 0.01) {
		t.Errorf("LerpRect midpoint = %v", r)
	}
}

func TestLerpCornerRadii(t *testing.T) {
	a := CornerRadii{0, 0, 0, 0}
	b := CornerRadii{10, 20, 30, 40}
	c := LerpCornerRadii(a, b, 0.5)
	if !approx(c.TopLeft, 5, 0.01) || !approx(c.TopRight, 10, 0.01) ||
		!approx(c.BottomRight, 15, 0.01) || !approx(c.BottomLeft, 20, 0.01) {
		t.Errorf("LerpCornerRadii midpoint = %v", c)
	}
}

func TestLerpColorEndpoints(t *testing.T) {
	red := Color{R: 1, G: 0, B: 0, A: 1}
	blue := Color{R: 0, G: 0, B: 1, A: 1}

	if c := LerpColor(red, blue, 0); c != red {
		t.Errorf("t=0 should return first color, got %v", c)
	}
	if c := LerpColor(red, blue, 1); c != blue {
		t.Errorf("t=1 should return second color, got %v", c)
	}
}
