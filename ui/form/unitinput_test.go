package form

import (
	"math"
	"testing"
)

func TestUnitInput_DisplayValue(t *testing.T) {
	units := []UnitDef{
		{Symbol: "mm", Label: "Millimeter", Factor: 1},
		{Symbol: "cm", Label: "Centimeter", Factor: 10},
		{Symbol: "m", Label: "Meter", Factor: 1000},
	}

	// Base unit: mm (Factor=1), value stored as mm.
	u := UnitInput{Value: 100, Unit: "mm", Units: units}
	if got := u.DisplayValue(); got != 100 {
		t.Errorf("display mm: got %v, want 100", got)
	}

	// cm (Factor=10): 100mm = 10cm.
	u.Unit = "cm"
	if got := u.DisplayValue(); got != 10 {
		t.Errorf("display cm: got %v, want 10", got)
	}

	// m (Factor=1000): 100mm = 0.1m.
	u.Unit = "m"
	if got := u.DisplayValue(); math.Abs(got-0.1) > 1e-12 {
		t.Errorf("display m: got %v, want 0.1", got)
	}
}

func TestUnitInput_ConvertToBase(t *testing.T) {
	units := []UnitDef{
		{Symbol: "mm", Factor: 1},
		{Symbol: "cm", Factor: 10},
		{Symbol: "in", Factor: 25.4},
	}

	u := UnitInput{Units: units}

	// 5 cm → 50 mm.
	if got := u.ConvertToBase(5, "cm"); math.Abs(got-50) > 1e-12 {
		t.Errorf("5 cm to base: got %v, want 50", got)
	}

	// 1 in → 25.4 mm.
	if got := u.ConvertToBase(1, "in"); math.Abs(got-25.4) > 1e-12 {
		t.Errorf("1 in to base: got %v, want 25.4", got)
	}

	// Unknown unit → identity.
	if got := u.ConvertToBase(7, "unknown"); got != 7 {
		t.Errorf("unknown unit: got %v, want 7", got)
	}
}

func TestUnitInput_MinMaxVal(t *testing.T) {
	min, max := 0.0, 100.0
	u := UnitInput{Min: &min, Max: &max}

	if got := u.minVal(); got != 0 {
		t.Errorf("minVal() = %v, want 0", got)
	}
	if got := u.maxVal(); got != 100 {
		t.Errorf("maxVal() = %v, want 100", got)
	}

	u2 := UnitInput{Min: nil, Max: nil}
	if got := u2.minVal(); got > -1e300 {
		t.Errorf("nil min: minVal() = %v, want -Inf", got)
	}
	if got := u2.maxVal(); got < 1e300 {
		t.Errorf("nil max: maxVal() = %v, want +Inf", got)
	}
}
