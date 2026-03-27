package data

import (
	"testing"

	"github.com/timzifer/lux/ui"
)

// testElement is a minimal ui.Element for testing.
type testElement struct{ ui.BaseElement }

func TestVirtualListResolvedItemCountLegacy(t *testing.T) {
	vl := VirtualList{ItemCount: 42}
	if got := vl.resolvedItemCount(); got != 42 {
		t.Fatalf("resolvedItemCount() = %d, want 42", got)
	}
}

func TestVirtualListResolvedItemCountSliceDataset(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2, 3, 4, 5})
	vl := VirtualList{Dataset: ds}
	if got := vl.resolvedItemCount(); got != 5 {
		t.Fatalf("resolvedItemCount() = %d, want 5", got)
	}
}

func TestVirtualListResolvedItemCountPagedDataset(t *testing.T) {
	ds := NewPagedDataset[int](10)
	ds.SetPage(0, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 100)
	vl := VirtualList{Dataset: ds}
	if got := vl.resolvedItemCount(); got != 100 {
		t.Fatalf("resolvedItemCount() = %d, want 100", got)
	}
}

func TestVirtualListResolvedItemCountPagedUnknown(t *testing.T) {
	ds := NewPagedDataset[int](10)
	// TotalCount is -1, no Count() method on PagedDataset
	vl := VirtualList{Dataset: ds}
	if got := vl.resolvedItemCount(); got != 0 {
		t.Fatalf("resolvedItemCount() = %d, want 0 (unknown length, no counter)", got)
	}
}

func TestVirtualListResolvedItemCountStreamDataset(t *testing.T) {
	ds := NewStreamDataset[int](StreamAppend)
	ds.Append(10, 20, 30)
	vl := VirtualList{Dataset: ds}
	// Len()=-1 but Count()=3
	if got := vl.resolvedItemCount(); got != 3 {
		t.Fatalf("resolvedItemCount() = %d, want 3", got)
	}
}

func TestVirtualListResolvedBuildItemLegacy(t *testing.T) {
	called := false
	vl := VirtualList{BuildItem: func(i int) ui.Element {
		called = true
		return testElement{}
	}}
	fn := vl.resolvedBuildItem()
	if fn == nil {
		t.Fatal("resolvedBuildItem() = nil")
	}
	fn(0, true)
	if !called {
		t.Fatal("legacy BuildItem was not called")
	}
}

func TestVirtualListResolvedBuildItemDS(t *testing.T) {
	var gotIndex int
	var gotLoaded bool
	vl := VirtualList{
		Dataset: NewSliceDataset([]int{1}),
		BuildItemDS: func(i int, loaded bool) ui.Element {
			gotIndex = i
			gotLoaded = loaded
			return testElement{}
		},
	}
	fn := vl.resolvedBuildItem()
	if fn == nil {
		t.Fatal("resolvedBuildItem() = nil")
	}
	fn(7, false)
	if gotIndex != 7 || gotLoaded != false {
		t.Fatalf("BuildItemDS called with (%d, %v), want (7, false)", gotIndex, gotLoaded)
	}
}

func TestVirtualListResolvedBuildItemNil(t *testing.T) {
	vl := VirtualList{}
	if fn := vl.resolvedBuildItem(); fn != nil {
		t.Fatal("resolvedBuildItem() should be nil when no build function is set")
	}
}

func TestVirtualListTreeEqualLegacy(t *testing.T) {
	a := VirtualList{ItemCount: 10, ItemHeight: 24, MaxHeight: 200}
	b := VirtualList{ItemCount: 10, ItemHeight: 24, MaxHeight: 200}
	if !a.TreeEqual(b) {
		t.Fatal("identical legacy VirtualLists should be TreeEqual")
	}

	c := VirtualList{ItemCount: 20, ItemHeight: 24, MaxHeight: 200}
	if a.TreeEqual(c) {
		t.Fatal("different ItemCount should not be TreeEqual")
	}
}

func TestVirtualListTreeEqualDataset(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2, 3})
	a := VirtualList{Dataset: ds, ItemHeight: 24, MaxHeight: 200}
	b := VirtualList{Dataset: ds, ItemHeight: 24, MaxHeight: 200}
	if !a.TreeEqual(b) {
		t.Fatal("same dataset pointer should be TreeEqual")
	}

	ds2 := NewSliceDataset([]int{1, 2, 3})
	c := VirtualList{Dataset: ds2, ItemHeight: 24, MaxHeight: 200}
	if a.TreeEqual(c) {
		t.Fatal("different dataset pointer should not be TreeEqual")
	}
}

func TestVirtualListTreeEqualTypeMismatch(t *testing.T) {
	vl := VirtualList{ItemCount: 10}
	if vl.TreeEqual(testElement{}) {
		t.Fatal("should not be TreeEqual to a different type")
	}
}

func TestVirtualListDatasetBuildItemReceivesLoadedFlag(t *testing.T) {
	ds := NewPagedDataset[int](3)
	ds.SetPage(0, []int{100, 200, 300}, 6)
	// Page 1 (indices 3-5) is not loaded.

	type call struct {
		index  int
		loaded bool
	}
	var calls []call
	vl := VirtualList{
		Dataset: ds,
		BuildItemDS: func(i int, loaded bool) ui.Element {
			calls = append(calls, call{i, loaded})
			return testElement{}
		},
	}

	build := vl.resolvedBuildItem()
	count := vl.resolvedItemCount()

	// Simulate rendering all items
	for i := 0; i < count; i++ {
		_, loaded := ds.Get(i)
		build(i, loaded)
	}

	if len(calls) != 6 {
		t.Fatalf("expected 6 calls, got %d", len(calls))
	}
	// First 3 should be loaded
	for i := 0; i < 3; i++ {
		if !calls[i].loaded {
			t.Fatalf("call[%d].loaded = false, want true (page 0 is loaded)", i)
		}
	}
	// Last 3 should be unloaded
	for i := 3; i < 6; i++ {
		if calls[i].loaded {
			t.Fatalf("call[%d].loaded = true, want false (page 1 is not loaded)", i)
		}
	}
}

func TestVirtualListBackwardCompat(t *testing.T) {
	var calledWith int
	vl := VirtualList{
		ItemCount:  5,
		ItemHeight: 30,
		BuildItem: func(i int) ui.Element {
			calledWith = i
			return testElement{}
		},
	}

	if got := vl.resolvedItemCount(); got != 5 {
		t.Fatalf("resolvedItemCount() = %d, want 5", got)
	}

	fn := vl.resolvedBuildItem()
	fn(3, true)
	if calledWith != 3 {
		t.Fatalf("BuildItem called with %d, want 3", calledWith)
	}
}

func TestVirtualListDatasetPriority(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2, 3})
	vl := VirtualList{
		Dataset:   ds,
		ItemCount: 999, // should be ignored when Dataset is set
	}
	if got := vl.resolvedItemCount(); got != 3 {
		t.Fatalf("Dataset should take priority, got %d, want 3", got)
	}
}

func TestVirtualListResolveChildren(t *testing.T) {
	vl := VirtualList{ItemCount: 5}
	result := vl.ResolveChildren(func(el ui.Element, i int) ui.Element { return el })
	if _, ok := result.(VirtualList); !ok {
		t.Fatal("ResolveChildren should return the same VirtualList")
	}
}
