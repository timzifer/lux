package form

import "testing"

func TestDrumPicker_WrapIndex(t *testing.T) {
	items := []DrumItem{
		{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}, {Label: "E"},
	}

	// Non-looping: clamp.
	d := DrumPicker{Items: items, Looping: false}
	if got := d.WrapIndex(-1); got != 0 {
		t.Errorf("non-looping WrapIndex(-1) = %d, want 0", got)
	}
	if got := d.WrapIndex(5); got != 4 {
		t.Errorf("non-looping WrapIndex(5) = %d, want 4", got)
	}
	if got := d.WrapIndex(2); got != 2 {
		t.Errorf("non-looping WrapIndex(2) = %d, want 2", got)
	}

	// Looping: wrap around.
	d.Looping = true
	if got := d.WrapIndex(-1); got != 4 {
		t.Errorf("looping WrapIndex(-1) = %d, want 4", got)
	}
	if got := d.WrapIndex(5); got != 0 {
		t.Errorf("looping WrapIndex(5) = %d, want 0", got)
	}
	if got := d.WrapIndex(7); got != 2 {
		t.Errorf("looping WrapIndex(7) = %d, want 2", got)
	}
}

func TestDrumPicker_WrapIndex_Empty(t *testing.T) {
	d := DrumPicker{Items: nil, Looping: true}
	if got := d.WrapIndex(0); got != 0 {
		t.Errorf("empty WrapIndex(0) = %d, want 0", got)
	}
}

func TestIntItems(t *testing.T) {
	items := IntItems(0, 5)
	if len(items) != 6 {
		t.Fatalf("IntItems(0,5) len = %d, want 6", len(items))
	}
	for i, item := range items {
		if item.Value != i {
			t.Errorf("IntItems[%d].Value = %v, want %d", i, item.Value, i)
		}
	}

	// Reverse range.
	rev := IntItems(3, 0)
	if len(rev) != 4 {
		t.Fatalf("IntItems(3,0) len = %d, want 4", len(rev))
	}
	if rev[0].Value != 3 || rev[3].Value != 0 {
		t.Errorf("IntItems(3,0) = [%v..%v], want [3..0]", rev[0].Value, rev[3].Value)
	}
}
