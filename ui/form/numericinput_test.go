package form

import (
	"math"
	"testing"
)

func ptrF(v float64) *float64 { return &v }

func TestSnapToStep_IntegerStep(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(100), Step: 1}
	v := 0.0
	for i := 0; i < 10; i++ {
		v = n.snapToStep(v + n.Step)
		expected := float64(i + 1)
		if v != expected {
			t.Errorf("increment %d: got %v, want %v", i+1, v, expected)
		}
	}
}

func TestSnapToStep_FractionalStep(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(10), Step: 0.1}
	v := 0.0
	for i := 0; i < 30; i++ {
		v = n.snapToStep(v + n.Step)
	}
	want := 3.0
	if math.Abs(v-want) > 1e-12 {
		t.Errorf("after 30 increments of 0.1: got %.17f, want %v", v, want)
	}
}

func TestSnapToStep_DecrementNoSkip(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(10), Step: 0.1}
	v := 3.0
	for i := 0; i < 30; i++ {
		v = n.snapToStep(v - n.Step)
	}
	want := 0.0
	if math.Abs(v-want) > 1e-12 {
		t.Errorf("after 30 decrements of 0.1: got %.17f, want %v", v, want)
	}
}

func TestSnapToStep_NonZeroMin(t *testing.T) {
	n := NumericInput{Min: ptrF(5), Max: ptrF(15), Step: 0.5}
	v := 5.0
	for i := 0; i < 20; i++ {
		v = n.snapToStep(v + n.Step)
	}
	want := 15.0
	if math.Abs(v-want) > 1e-12 {
		t.Errorf("after 20 increments of 0.5 from 5: got %.17f, want %v", v, want)
	}
}

func TestSnapToStep_DragThenStep(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(100), Step: 1}
	v := 7.4999999
	v = n.snapToStep(v + n.Step)
	want := 8.0
	if v != want {
		t.Errorf("drag then step: got %v, want %v", v, want)
	}
}

func TestSnapToStep_ZeroStep(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(100), Step: 0}
	v := 3.7
	if got := n.snapToStep(v); got != v {
		t.Errorf("zero step: got %v, want %v (unchanged)", got, v)
	}
}

func TestClampWithSnap(t *testing.T) {
	n := NumericInput{Min: ptrF(0), Max: ptrF(10), Step: 1}
	v := n.clamp(n.snapToStep(10 + n.Step))
	if v != 10 {
		t.Errorf("clamp at max: got %v, want 10", v)
	}
	v = n.clamp(n.snapToStep(0 - n.Step))
	if v != 0 {
		t.Errorf("clamp at min: got %v, want 0", v)
	}
}

func TestNumericInput_IsValidChar(t *testing.T) {
	intInput := NumericInput{Kind: NumericInteger}
	floatInput := NumericInput{Kind: NumericFloat}

	// Digits always allowed.
	for _, ch := range "0123456789" {
		if !intInput.IsValidChar(ch, "", 0) {
			t.Errorf("digit %c should be valid for integer", ch)
		}
	}

	// Sign only at position 0.
	if !intInput.IsValidChar('-', "", 0) {
		t.Error("minus at pos 0 should be valid")
	}
	if intInput.IsValidChar('-', "1", 1) {
		t.Error("minus at pos 1 should be invalid")
	}

	// Decimal only for float, and only once.
	if intInput.IsValidChar('.', "1", 1) {
		t.Error("dot should be invalid for integer")
	}
	if !floatInput.IsValidChar('.', "1", 1) {
		t.Error("dot should be valid for float")
	}
	if floatInput.IsValidChar('.', "1.2", 3) {
		t.Error("second dot should be invalid")
	}

	// Letters not allowed.
	if intInput.IsValidChar('a', "", 0) {
		t.Error("letter should be invalid")
	}
}

func TestNumericInput_FormatValue(t *testing.T) {
	tests := []struct {
		name string
		n    NumericInput
		want string
	}{
		{"integer step", NumericInput{Value: 42, Step: 1}, "42"},
		{"float default", NumericInput{Value: 3.14, Step: 0.1}, "3.14"},
		{"float precision 1", NumericInput{Value: 3.14, Kind: NumericFloat, Precision: 1}, "3.1"},
		{"float precision 3", NumericInput{Value: 1.0, Kind: NumericFloat, Precision: 3}, "1.000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.n.formatValue()
			if got != tt.want {
				t.Errorf("formatValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNumericInput_NilBounds(t *testing.T) {
	n := NumericInput{Value: 50, Min: nil, Max: nil, Step: 1}
	// With nil bounds, clamp should not restrict.
	if got := n.clamp(1000); got != 1000 {
		t.Errorf("nil max: clamp(1000) = %v, want 1000", got)
	}
	if got := n.clamp(-1000); got != -1000 {
		t.Errorf("nil min: clamp(-1000) = %v, want -1000", got)
	}
}
