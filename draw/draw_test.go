package draw

import (
	"math"
	"testing"
)

func almostEqual(a, b, eps float32) bool {
	return float32(math.Abs(float64(a-b))) < eps
}

func TestHex6(t *testing.T) {
	c := Hex("#3b82f6")
	if !almostEqual(c.R, 0x3b/255.0, 0.002) {
		t.Errorf("R = %f, want %f", c.R, float32(0x3b)/255)
	}
	if !almostEqual(c.G, 0x82/255.0, 0.002) {
		t.Errorf("G = %f, want %f", c.G, float32(0x82)/255)
	}
	if !almostEqual(c.B, 0xf6/255.0, 0.002) {
		t.Errorf("B = %f, want %f", c.B, float32(0xf6)/255)
	}
	if !almostEqual(c.A, 1.0, 0.002) {
		t.Errorf("A = %f, want 1.0", c.A)
	}
}

func TestHex8(t *testing.T) {
	c := Hex("#ffffff80")
	if !almostEqual(c.R, 1.0, 0.002) {
		t.Errorf("R = %f, want 1.0", c.R)
	}
	if !almostEqual(c.A, 0x80/255.0, 0.002) {
		t.Errorf("A = %f, want %f", c.A, float32(0x80)/255)
	}
}

func TestHex3(t *testing.T) {
	c := Hex("#fff")
	if c.R != 1.0 || c.G != 1.0 || c.B != 1.0 {
		t.Errorf("Hex(#fff) = %v, want white", c)
	}
}

func TestHexPanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on invalid hex")
		}
	}()
	Hex("#gg")
}

func TestFontWeightSemiBold(t *testing.T) {
	if FontWeightSemiBold != 600 {
		t.Errorf("FontWeightSemiBold = %d, want 600", FontWeightSemiBold)
	}
	if FontWeightSemiBold <= FontWeightMedium || FontWeightSemiBold >= FontWeightBold {
		t.Error("FontWeightSemiBold should be between Medium and Bold")
	}
}
