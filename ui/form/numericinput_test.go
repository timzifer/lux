package form

import (
	"math"
	"testing"
)

func TestSnapToStep_IntegerStep(t *testing.T) {
	n := NumericInput{Min: 0, Max: 100, Step: 1}
	// Simulate 10 increments from 0.
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
	n := NumericInput{Min: 0, Max: 10, Step: 0.1}
	// Simulate 30 increments from 0 — classic floating-point drift scenario.
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
	n := NumericInput{Min: 0, Max: 10, Step: 0.1}
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
	n := NumericInput{Min: 5, Max: 15, Step: 0.5}
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
	n := NumericInput{Min: 0, Max: 100, Step: 1}
	// Simulate drag leaving value off-grid.
	v := 7.4999999
	v = n.snapToStep(v + n.Step)
	want := 8.0
	if v != want {
		t.Errorf("drag then step: got %v, want %v", v, want)
	}
}

func TestSnapToStep_ZeroStep(t *testing.T) {
	n := NumericInput{Min: 0, Max: 100, Step: 0}
	v := 3.7
	if got := n.snapToStep(v); got != v {
		t.Errorf("zero step: got %v, want %v (unchanged)", got, v)
	}
}

func TestClampWithSnap(t *testing.T) {
	n := NumericInput{Min: 0, Max: 10, Step: 1}
	// Increment past max should clamp.
	v := n.clamp(n.snapToStep(10 + n.Step))
	if v != 10 {
		t.Errorf("clamp at max: got %v, want 10", v)
	}
	// Decrement past min should clamp.
	v = n.clamp(n.snapToStep(0 - n.Step))
	if v != 0 {
		t.Errorf("clamp at min: got %v, want 0", v)
	}
}
