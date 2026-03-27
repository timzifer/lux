package data

import "testing"

// Compile-time interface check.
var _ Dataset[int] = (*PagedDataset[int])(nil)
var _ Dataset[string] = (*PagedDataset[string])(nil)

func TestPagedDatasetNew(t *testing.T) {
	d := NewPagedDataset[int](20)
	if d.TotalCount != -1 {
		t.Fatalf("TotalCount = %d, want -1", d.TotalCount)
	}
	if d.Len() != -1 {
		t.Fatalf("Len() = %d, want -1", d.Len())
	}
	if d.PageSize != 20 {
		t.Fatalf("PageSize = %d, want 20", d.PageSize)
	}
}

func TestPagedDatasetDefaultPageSize(t *testing.T) {
	d := NewPagedDataset[int](0)
	if d.PageSize != 50 {
		t.Fatalf("PageSize = %d, want 50 (default)", d.PageSize)
	}
}

func TestPagedDatasetSetPage(t *testing.T) {
	d := NewPagedDataset[string](3)
	d.SetPage(0, []string{"a", "b", "c"}, 9)

	if d.TotalCount != 9 {
		t.Fatalf("TotalCount = %d, want 9", d.TotalCount)
	}
	if d.Len() != 9 {
		t.Fatalf("Len() = %d, want 9", d.Len())
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

func TestPagedDatasetGetUnloaded(t *testing.T) {
	d := NewPagedDataset[int](10)
	id, loaded := d.Get(5)
	if loaded {
		t.Fatalf("Get(5) loaded=true, want false (page not loaded)")
	}
	if id != 0 {
		t.Fatalf("Get(5) = %d, want 0 (zero value)", id)
	}
}

func TestPagedDatasetSetLoading(t *testing.T) {
	d := NewPagedDataset[int](10)
	d.SetLoading(2)
	if !d.IsPageLoading(2) {
		t.Fatal("IsPageLoading(2) = false, want true")
	}
	if d.IsPageLoading(0) {
		t.Fatal("IsPageLoading(0) = true, want false")
	}
}

func TestPagedDatasetSetError(t *testing.T) {
	d := NewPagedDataset[int](10)
	d.SetError(1)
	if d.PageState(1) != SlotError {
		t.Fatalf("PageState(1) = %d, want SlotError", d.PageState(1))
	}
}

func TestPagedDatasetMultiplePages(t *testing.T) {
	d := NewPagedDataset[int](2)
	d.SetPage(0, []int{10, 20}, 6)
	d.SetPage(1, []int{30, 40}, -1) // keep TotalCount unchanged
	d.SetPage(2, []int{50, 60}, -1)

	if d.TotalCount != 6 {
		t.Fatalf("TotalCount = %d, want 6", d.TotalCount)
	}

	expected := []int{10, 20, 30, 40, 50, 60}
	for i, want := range expected {
		id, loaded := d.Get(i)
		if !loaded {
			t.Fatalf("Get(%d) loaded=false", i)
		}
		if id != want {
			t.Fatalf("Get(%d) = %d, want %d", i, id, want)
		}
	}
}

func TestPagedDatasetPageForIndex(t *testing.T) {
	d := NewPagedDataset[int](25)
	tests := []struct{ index, wantPage int }{
		{0, 0}, {24, 0}, {25, 1}, {49, 1}, {50, 2}, {100, 4},
	}
	for _, tt := range tests {
		if got := d.PageForIndex(tt.index); got != tt.wantPage {
			t.Errorf("PageForIndex(%d) = %d, want %d", tt.index, got, tt.wantPage)
		}
	}
}

func TestPagedDatasetPartialLastPage(t *testing.T) {
	d := NewPagedDataset[string](3)
	d.SetPage(0, []string{"a", "b", "c"}, 5)
	d.SetPage(1, []string{"d", "e"}, -1) // partial page (2 of 3)

	id, loaded := d.Get(4) // index 4 = page 1, offset 1
	if !loaded || id != "e" {
		t.Fatalf("Get(4) = (%q, %v), want (\"e\", true)", id, loaded)
	}

	// Index 5 is beyond the partial page data even though TotalCount=5
	_, loaded = d.Get(5)
	if loaded {
		t.Fatal("Get(5) loaded=true, want false (beyond partial page)")
	}
}

func TestPagedDatasetSetPageOverwrite(t *testing.T) {
	d := NewPagedDataset[int](2)
	d.SetPage(0, []int{1, 2}, 10)
	d.SetPage(0, []int{3, 4}, -1) // overwrite page 0

	id, loaded := d.Get(0)
	if !loaded || id != 3 {
		t.Fatalf("Get(0) after overwrite = (%d, %v), want (3, true)", id, loaded)
	}
}

func TestPagedDatasetLoadingThenLoaded(t *testing.T) {
	d := NewPagedDataset[int](10)
	d.SetLoading(0)
	if !d.IsPageLoading(0) {
		t.Fatal("after SetLoading: IsPageLoading=false")
	}

	d.SetPage(0, []int{1, 2, 3}, 100)
	if d.IsPageLoading(0) {
		t.Fatal("after SetPage: IsPageLoading=true, want false")
	}
	if d.PageState(0) != SlotLoaded {
		t.Fatalf("after SetPage: PageState = %d, want SlotLoaded", d.PageState(0))
	}
}

func TestPagedDatasetErrorThenRetry(t *testing.T) {
	d := NewPagedDataset[int](10)
	d.SetError(0)
	if d.PageState(0) != SlotError {
		t.Fatal("expected SlotError after SetError")
	}

	// Retry: set loading again, then load
	d.SetLoading(0)
	if d.PageState(0) != SlotLoading {
		t.Fatal("expected SlotLoading after retry")
	}

	d.SetPage(0, []int{42}, 1)
	id, loaded := d.Get(0)
	if !loaded || id != 42 {
		t.Fatalf("after retry: Get(0) = (%d, %v), want (42, true)", id, loaded)
	}
}
