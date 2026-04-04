package form

import "testing"

func TestStepper_Clamp(t *testing.T) {
	s := Stepper{Min: 0, Max: 10, Step: 1}

	tests := []struct {
		name string
		in   int
		want int
	}{
		{"within range", 5, 5},
		{"at min", 0, 0},
		{"at max", 10, 10},
		{"below min", -1, 0},
		{"above max", 11, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.clamp(tt.in); got != tt.want {
				t.Errorf("clamp(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestStepper_FormatValue(t *testing.T) {
	// Default formatting.
	s := Stepper{Value: 42}
	if got := s.formatValue(); got != "42" {
		t.Errorf("formatValue() = %q, want %q", got, "42")
	}

	// With Label.
	s.Label = "Day 42"
	if got := s.formatValue(); got != "Day 42" {
		t.Errorf("formatValue() with Label = %q, want %q", got, "Day 42")
	}

	// With custom Format function.
	s.Format = func(v int) string { return "Step " + string(rune('0'+v%10)) }
	s.Value = 3
	if got := s.formatValue(); got != "Step 3" {
		t.Errorf("formatValue() with Format = %q, want %q", got, "Step 3")
	}
}

func TestStepper_IncrementDecrement(t *testing.T) {
	s := Stepper{Value: 5, Min: 0, Max: 10, Step: 2}

	up := s.clamp(s.Value + s.Step)
	if up != 7 {
		t.Errorf("increment: got %d, want 7", up)
	}

	down := s.clamp(s.Value - s.Step)
	if down != 3 {
		t.Errorf("decrement: got %d, want 3", down)
	}

	// Clamp at boundaries.
	s.Value = 9
	up = s.clamp(s.Value + s.Step)
	if up != 10 {
		t.Errorf("increment at boundary: got %d, want 10", up)
	}

	s.Value = 1
	down = s.clamp(s.Value - s.Step)
	if down != 0 {
		t.Errorf("decrement at boundary: got %d, want 0", down)
	}
}
