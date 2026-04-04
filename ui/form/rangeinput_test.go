package form

import (
	"math"
	"testing"
)

func TestRangeInput_Snap(t *testing.T) {
	r := RangeInput{Min: 0, Max: 100, Step: 5}

	tests := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{2.4, 0},
		{2.5, 5},
		{7.5, 10},
		{100, 100},
		{97.5, 100},
	}
	for _, tt := range tests {
		if got := r.snap(tt.in); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("snap(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestRangeInput_ClampLow(t *testing.T) {
	r := RangeInput{Low: 20, High: 80, Min: 0, Max: 100}

	// Within range.
	if got := r.clampLow(50); got != 50 {
		t.Errorf("clampLow(50) = %v, want 50", got)
	}

	// Below min.
	if got := r.clampLow(-5); got != 0 {
		t.Errorf("clampLow(-5) = %v, want 0", got)
	}

	// Above high (invariant: Low <= High).
	if got := r.clampLow(90); got != 80 {
		t.Errorf("clampLow(90) = %v, want 80", got)
	}
}

func TestRangeInput_ClampHigh(t *testing.T) {
	r := RangeInput{Low: 20, High: 80, Min: 0, Max: 100}

	// Within range.
	if got := r.clampHigh(50); got != 50 {
		t.Errorf("clampHigh(50) = %v, want 50", got)
	}

	// Above max.
	if got := r.clampHigh(110); got != 100 {
		t.Errorf("clampHigh(110) = %v, want 100", got)
	}

	// Below low (invariant: Low <= High).
	if got := r.clampHigh(10); got != 20 {
		t.Errorf("clampHigh(10) = %v, want 20", got)
	}
}

func TestRangeInput_FormatVal(t *testing.T) {
	r := RangeInput{}
	if got := r.formatVal(42.7); got != "43" {
		t.Errorf("default formatVal(42.7) = %q, want %q", got, "43")
	}

	r.Format = func(v float64) string { return "val" }
	if got := r.formatVal(42.7); got != "val" {
		t.Errorf("custom formatVal = %q, want %q", got, "val")
	}
}
