package data

import "testing"

// Compile-time interface check.
var _ Dataset[int] = (*StreamDataset[int])(nil)
var _ Dataset[string] = (*StreamDataset[string])(nil)

func TestStreamDatasetNew(t *testing.T) {
	d := NewStreamDataset[int](StreamAppend)
	if d.Len() != -1 {
		t.Fatalf("Len() = %d, want -1", d.Len())
	}
	if d.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", d.Count())
	}
}

func TestStreamDatasetAppend(t *testing.T) {
	d := NewStreamDataset[string](StreamAppend)
	d.Append("a", "b")
	d.Append("c")

	if d.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", d.Count())
	}
	for i, want := range []string{"a", "b", "c"} {
		id, loaded := d.Get(i)
		if !loaded {
			t.Fatalf("Get(%d) loaded=false", i)
		}
		if id != want {
			t.Fatalf("Get(%d) = %q, want %q", i, id, want)
		}
	}
}

func TestStreamDatasetPrepend(t *testing.T) {
	d := NewStreamDataset[int](StreamPrepend)
	d.Append(3)         // [3]
	d.Prepend(1, 2)     // [1, 2, 3]

	if d.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", d.Count())
	}
	expected := []int{1, 2, 3}
	for i, want := range expected {
		id, loaded := d.Get(i)
		if !loaded || id != want {
			t.Fatalf("Get(%d) = (%d, %v), want (%d, true)", i, id, loaded, want)
		}
	}
}

func TestStreamDatasetGetOutOfRange(t *testing.T) {
	d := NewStreamDataset[int](StreamAppend)
	d.Append(1)

	id, loaded := d.Get(-1)
	if loaded {
		t.Fatal("Get(-1) loaded=true, want false")
	}
	if id != 0 {
		t.Fatalf("Get(-1) = %d, want 0", id)
	}

	id, loaded = d.Get(1)
	if loaded {
		t.Fatal("Get(1) loaded=true, want false (only 1 item)")
	}
	if id != 0 {
		t.Fatalf("Get(1) = %d, want 0", id)
	}
}

func TestStreamDatasetLenAlwaysMinusOne(t *testing.T) {
	d := NewStreamDataset[int](StreamAppend)
	d.Append(1, 2, 3, 4, 5)
	if d.Len() != -1 {
		t.Fatalf("Len() = %d, want -1 (always unknown for streams)", d.Len())
	}
}

func TestStreamDatasetMode(t *testing.T) {
	a := NewStreamDataset[int](StreamAppend)
	if a.Mode() != StreamAppend {
		t.Fatalf("Mode() = %d, want StreamAppend", a.Mode())
	}
	p := NewStreamDataset[int](StreamPrepend)
	if p.Mode() != StreamPrepend {
		t.Fatalf("Mode() = %d, want StreamPrepend", p.Mode())
	}
}

func TestStreamDatasetEmpty(t *testing.T) {
	d := NewStreamDataset[string](StreamAppend)
	_, loaded := d.Get(0)
	if loaded {
		t.Fatal("Get(0) on empty stream should return loaded=false")
	}
}

func TestStreamDatasetPrependMultipleBatches(t *testing.T) {
	d := NewStreamDataset[string](StreamPrepend)
	d.Prepend("c")          // [c]
	d.Prepend("b")          // [b, c]
	d.Prepend("a")          // [a, b, c]

	expected := []string{"a", "b", "c"}
	for i, want := range expected {
		id, _ := d.Get(i)
		if id != want {
			t.Fatalf("Get(%d) = %q, want %q", i, id, want)
		}
	}
}
