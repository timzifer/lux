package data

import "testing"

// Compile-time interface check.
var _ Dataset[int] = (*SliceDataset[int])(nil)
var _ Dataset[string] = (*SliceDataset[string])(nil)

func TestSliceDatasetLen(t *testing.T) {
	d := NewSliceDataset([]int{10, 20, 30})
	if got := d.Len(); got != 3 {
		t.Fatalf("Len() = %d, want 3", got)
	}
}

func TestSliceDatasetEmpty(t *testing.T) {
	d := NewSliceDataset[int](nil)
	if got := d.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
}

func TestSliceDatasetGet(t *testing.T) {
	d := NewSliceDataset([]string{"a", "b", "c"})
	for i, want := range []string{"a", "b", "c"} {
		id, loaded := d.Get(i)
		if !loaded {
			t.Fatalf("Get(%d) loaded=false, want true", i)
		}
		if id != want {
			t.Fatalf("Get(%d) = %q, want %q", i, id, want)
		}
	}
}

func TestSliceDatasetGetPanicsOutOfBounds(t *testing.T) {
	d := NewSliceDataset([]int{1, 2})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Get(5) should panic for out-of-bounds index")
		}
	}()
	d.Get(5)
}

func TestSliceDatasetAlwaysLoaded(t *testing.T) {
	d := NewSliceDataset([]int{42})
	_, loaded := d.Get(0)
	if !loaded {
		t.Fatal("SliceDataset.Get should always return loaded=true")
	}
}
