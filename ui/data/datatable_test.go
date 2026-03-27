package data

import (
	"testing"

	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
)

// Compile-time interface check.
var _ ui.Element = DataTable{}

func TestDataTableStateDefaults(t *testing.T) {
	s := NewDataTableState()
	if s.Scroll == nil {
		t.Fatal("Scroll should be initialized")
	}
	if s.Scroll.Offset != 0 {
		t.Fatalf("Scroll.Offset = %f, want 0", s.Scroll.Offset)
	}
	if s.SortColumn != "" {
		t.Fatalf("SortColumn = %q, want empty", s.SortColumn)
	}
	if s.SortDir != SortNone {
		t.Fatalf("SortDir = %d, want SortNone", s.SortDir)
	}
	if s.FilterText != "" {
		t.Fatalf("FilterText = %q, want empty", s.FilterText)
	}
	if s.CurrentPage != 0 {
		t.Fatalf("CurrentPage = %d, want 0", s.CurrentPage)
	}
}

func TestDataTableSortCycle(t *testing.T) {
	tests := []struct {
		input SortDirection
		want  SortDirection
	}{
		{SortNone, SortAsc},
		{SortAsc, SortDesc},
		{SortDesc, SortNone},
	}
	for _, tt := range tests {
		got := NextSortDirection(tt.input)
		if got != tt.want {
			t.Errorf("NextSortDirection(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDataTableSortNoneIsZero(t *testing.T) {
	if SortNone != 0 {
		t.Fatalf("SortNone = %d, want 0 (must be zero value)", SortNone)
	}
}

func TestDataTableTreeEqual(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2, 3})
	cols := []DataTableColumn{
		{Key: "a", Header: "A"},
		{Key: "b", Header: "B"},
	}
	state := &DataTableState{}

	dt1 := NewDataTable(ds, cols, state)
	dt2 := NewDataTable(ds, cols, state)

	if !dt1.(DataTable).TreeEqual(dt2) {
		t.Fatal("identical DataTables should be TreeEqual")
	}
}

func TestDataTableTreeEqualDifferent(t *testing.T) {
	ds1 := NewSliceDataset([]int{1, 2, 3})
	ds2 := NewSliceDataset([]int{4, 5, 6})
	cols := []DataTableColumn{{Key: "a", Header: "A"}}
	state := &DataTableState{}

	dt1 := NewDataTable(ds1, cols, state)
	dt2 := NewDataTable(ds2, cols, state)

	if dt1.(DataTable).TreeEqual(dt2) {
		t.Fatal("DataTables with different datasets should not be TreeEqual")
	}
}

func TestDataTableTreeEqualDifferentColumnCount(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2})
	cols1 := []DataTableColumn{{Key: "a", Header: "A"}}
	cols2 := []DataTableColumn{{Key: "a", Header: "A"}, {Key: "b", Header: "B"}}
	state := &DataTableState{}

	dt1 := NewDataTable(ds, cols1, state)
	dt2 := NewDataTable(ds, cols2, state)

	if dt1.(DataTable).TreeEqual(dt2) {
		t.Fatal("DataTables with different column counts should not be TreeEqual")
	}
}

func TestDataTablePaginationDetect(t *testing.T) {
	pd := NewPagedDataset[int](20)
	_, ok := isPaged(pd)
	if !ok {
		t.Fatal("isPaged should detect PagedDataset")
	}

	sd := NewSliceDataset([]int{1})
	_, ok = isPaged(sd)
	if ok {
		t.Fatal("isPaged should not detect SliceDataset")
	}
}

func TestDataTableStreamDetect(t *testing.T) {
	sd := NewStreamDataset[int](StreamAppend)
	_, ok := isStream(sd)
	if !ok {
		t.Fatal("isStream should detect StreamDataset")
	}

	slice := NewSliceDataset([]int{1})
	_, ok = isStream(slice)
	if ok {
		t.Fatal("isStream should not detect SliceDataset")
	}
}

func TestDataTableResolveColumnWidthsFixed(t *testing.T) {
	cols := []DataTableColumn{
		{Key: "a", Width: layout.Px(100)},
		{Key: "b", Width: layout.Px(200)},
		{Key: "c", Width: layout.Px(150)},
	}
	widths := resolveColumnWidths(cols, 500)
	if len(widths) != 3 {
		t.Fatalf("len(widths) = %d, want 3", len(widths))
	}
	if widths[0] != 100 {
		t.Errorf("widths[0] = %d, want 100", widths[0])
	}
	if widths[1] != 200 {
		t.Errorf("widths[1] = %d, want 200", widths[1])
	}
	if widths[2] != 150 {
		t.Errorf("widths[2] = %d, want 150", widths[2])
	}
}

func TestDataTableResolveColumnWidthsFr(t *testing.T) {
	cols := []DataTableColumn{
		{Key: "a", Width: layout.Fr(1)},
		{Key: "b", Width: layout.Fr(2)},
		{Key: "c", Width: layout.Fr(1)},
	}
	widths := resolveColumnWidths(cols, 400)
	// Total fr = 4; frUnit = 100
	if widths[0] != 100 {
		t.Errorf("widths[0] = %d, want 100", widths[0])
	}
	if widths[1] != 200 {
		t.Errorf("widths[1] = %d, want 200", widths[1])
	}
	if widths[2] != 100 {
		t.Errorf("widths[2] = %d, want 100", widths[2])
	}
}

func TestDataTableResolveColumnWidthsMixed(t *testing.T) {
	cols := []DataTableColumn{
		{Key: "a", Width: layout.Px(100)},
		{Key: "b", Width: layout.Fr(1)},
		{Key: "c", Width: layout.Fr(1)},
	}
	widths := resolveColumnWidths(cols, 500)
	// Fixed = 100; remaining = 400; frUnit = 200
	if widths[0] != 100 {
		t.Errorf("widths[0] = %d, want 100", widths[0])
	}
	if widths[1] != 200 {
		t.Errorf("widths[1] = %d, want 200", widths[1])
	}
	if widths[2] != 200 {
		t.Errorf("widths[2] = %d, want 200", widths[2])
	}
}

func TestDataTableResolveColumnWidthsMinmax(t *testing.T) {
	cols := []DataTableColumn{
		{Key: "a", Width: layout.Minmax(50, 150)},
		{Key: "b", Width: layout.Fr(1)},
	}
	widths := resolveColumnWidths(cols, 300)
	// Minmax takes min=50 initially; remaining = 250 → Fr(1) = 250
	if widths[0] != 50 {
		t.Errorf("widths[0] = %d, want 50", widths[0])
	}
	if widths[1] != 250 {
		t.Errorf("widths[1] = %d, want 250", widths[1])
	}
}

func TestDataTableResolveColumnWidthsAuto(t *testing.T) {
	cols := []DataTableColumn{
		{Key: "a", Width: layout.Px(100)},
		{Key: "b", Width: layout.AutoTrack()},
		{Key: "c", Width: layout.AutoTrack()},
	}
	widths := resolveColumnWidths(cols, 400)
	// Fixed = 100; remaining = 300; 2 auto columns → 150 each
	if widths[0] != 100 {
		t.Errorf("widths[0] = %d, want 100", widths[0])
	}
	if widths[1] != 150 {
		t.Errorf("widths[1] = %d, want 150", widths[1])
	}
	if widths[2] != 150 {
		t.Errorf("widths[2] = %d, want 150", widths[2])
	}
}

func TestDataTableResolveColumnWidthsZeroValue(t *testing.T) {
	// Zero-value TrackSize should be treated as Fr(1).
	cols := []DataTableColumn{
		{Key: "a"},
		{Key: "b"},
	}
	widths := resolveColumnWidths(cols, 300)
	if widths[0] != 150 {
		t.Errorf("widths[0] = %d, want 150", widths[0])
	}
	if widths[1] != 150 {
		t.Errorf("widths[1] = %d, want 150", widths[1])
	}
}

func TestDataTableResolveColumnWidthsEmpty(t *testing.T) {
	widths := resolveColumnWidths(nil, 500)
	if widths != nil {
		t.Fatalf("expected nil for empty columns, got %v", widths)
	}
}

func TestDataTableNewDefaults(t *testing.T) {
	ds := NewSliceDataset([]int{1, 2, 3})
	cols := []DataTableColumn{{Key: "id", Header: "ID"}}
	state := &DataTableState{}

	dt := NewDataTable(ds, cols, state).(DataTable)
	if dt.SelectedRow != -1 {
		t.Fatalf("SelectedRow = %d, want -1", dt.SelectedRow)
	}
	if dt.RowHeight != 0 {
		t.Fatalf("RowHeight = %f, want 0 (uses default)", dt.RowHeight)
	}
	if dt.Filterable {
		t.Fatal("Filterable should be false by default")
	}
}

func TestDataTableNewWithOptions(t *testing.T) {
	ds := NewSliceDataset([]int{1})
	cols := []DataTableColumn{{Key: "id", Header: "ID"}}
	state := &DataTableState{}
	var sortCalled bool

	dt := NewDataTable(ds, cols, state,
		WithDTRowHeight(40),
		WithDTMaxHeight(300),
		WithDTFilterable(true),
		WithDTOnSort(func(col string, dir SortDirection) { sortCalled = true }),
	).(DataTable)

	if dt.RowHeight != 40 {
		t.Errorf("RowHeight = %f, want 40", dt.RowHeight)
	}
	if dt.MaxHeight != 300 {
		t.Errorf("MaxHeight = %f, want 300", dt.MaxHeight)
	}
	if !dt.Filterable {
		t.Error("Filterable should be true")
	}
	if dt.OnSort == nil {
		t.Fatal("OnSort should be set")
	}
	dt.OnSort("col", SortAsc)
	if !sortCalled {
		t.Fatal("OnSort callback not invoked")
	}
}

func TestDataTableBuildViewIndicesNoFilter(t *testing.T) {
	ds := NewSliceDataset([]int{10, 20, 30})
	state := &DataTableState{}
	dt := DataTable{Dataset: ds, State: state}

	indices := dt.buildViewIndices()
	if len(indices) != 3 {
		t.Fatalf("len(indices) = %d, want 3", len(indices))
	}
	for i, want := range []int{0, 1, 2} {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTableBuildViewIndicesWithFilter(t *testing.T) {
	ds := NewSliceDataset([]int{10, 20, 30})
	state := &DataTableState{FilterText: "twenty"}
	dt := DataTable{
		Dataset: ds,
		State:   state,
		Columns: []DataTableColumn{
			{
				Key:    "val",
				Header: "Value",
				FilterValue: func(id int) string {
					switch id {
					case 0:
						return "ten"
					case 1:
						return "twenty"
					case 2:
						return "thirty"
					}
					return ""
				},
			},
		},
	}

	indices := dt.buildViewIndices()
	if len(indices) != 1 {
		t.Fatalf("len(indices) = %d, want 1", len(indices))
	}
	if indices[0] != 1 {
		t.Errorf("indices[0] = %d, want 1 (index of 'twenty')", indices[0])
	}
}

func TestDataTableBuildViewIndicesWithSort(t *testing.T) {
	ds := NewSliceDataset([]int{30, 10, 20})
	state := &DataTableState{SortColumn: "val", SortDir: SortAsc}
	dt := DataTable{
		Dataset: ds,
		State:   state,
		Columns: []DataTableColumn{
			{
				Key:      "val",
				Header:   "Value",
				Sortable: true,
				SortLess: func(i, j int) bool {
					vi, _ := ds.Get(i)
					vj, _ := ds.Get(j)
					return vi < vj
				},
			},
		},
	}

	indices := dt.buildViewIndices()
	if len(indices) != 3 {
		t.Fatalf("len(indices) = %d, want 3", len(indices))
	}
	// Original: [30, 10, 20] → sorted ascending by value → [10, 20, 30] → indices [1, 2, 0]
	expected := []int{1, 2, 0}
	for i, want := range expected {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTableBuildViewIndicesWithSortDesc(t *testing.T) {
	ds := NewSliceDataset([]int{30, 10, 20})
	state := &DataTableState{SortColumn: "val", SortDir: SortDesc}
	dt := DataTable{
		Dataset: ds,
		State:   state,
		Columns: []DataTableColumn{
			{
				Key:      "val",
				Header:   "Value",
				Sortable: true,
				SortLess: func(i, j int) bool {
					vi, _ := ds.Get(i)
					vj, _ := ds.Get(j)
					return vi < vj
				},
			},
		},
	}

	indices := dt.buildViewIndices()
	// Desc: [30, 20, 10] → indices [0, 2, 1]
	expected := []int{0, 2, 1}
	for i, want := range expected {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTableResolvedItemCount(t *testing.T) {
	// SliceDataset
	dt := DataTable{Dataset: NewSliceDataset([]int{1, 2, 3, 4, 5})}
	if got := dt.resolvedItemCount(); got != 5 {
		t.Errorf("SliceDataset: resolvedItemCount() = %d, want 5", got)
	}

	// PagedDataset with known total
	pd := NewPagedDataset[int](10)
	pd.SetPage(0, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 50)
	dt2 := DataTable{Dataset: pd}
	if got := dt2.resolvedItemCount(); got != 50 {
		t.Errorf("PagedDataset: resolvedItemCount() = %d, want 50", got)
	}

	// StreamDataset
	sd := NewStreamDataset[int](StreamAppend)
	sd.Append(1, 2, 3)
	dt3 := DataTable{Dataset: sd}
	if got := dt3.resolvedItemCount(); got != 3 {
		t.Errorf("StreamDataset: resolvedItemCount() = %d, want 3", got)
	}

	// Nil dataset
	dt4 := DataTable{}
	if got := dt4.resolvedItemCount(); got != 0 {
		t.Errorf("nil dataset: resolvedItemCount() = %d, want 0", got)
	}
}

func TestDataTableResolveChildren(t *testing.T) {
	ds := NewSliceDataset([]int{1})
	dt := DataTable{Dataset: ds}
	resolved := dt.ResolveChildren(func(el ui.Element, i int) ui.Element { return el })
	if _, ok := resolved.(DataTable); !ok {
		t.Fatal("ResolveChildren should return DataTable (leaf)")
	}
}

func TestDataTablePagedViewIndicesPage0(t *testing.T) {
	pd := NewPagedDataset[int](5)
	pd.SetPage(0, []int{10, 20, 30, 40, 50}, 15)
	state := NewDataTableState()
	state.CurrentPage = 0

	dt := DataTable{Dataset: pd, State: state}
	indices := dt.buildViewIndices()
	if len(indices) != 5 {
		t.Fatalf("len(indices) = %d, want 5", len(indices))
	}
	for i, want := range []int{0, 1, 2, 3, 4} {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTablePagedViewIndicesPage1(t *testing.T) {
	pd := NewPagedDataset[int](5)
	pd.SetPage(0, []int{10, 20, 30, 40, 50}, 15)
	pd.SetPage(1, []int{60, 70, 80, 90, 100}, -1)
	state := NewDataTableState()
	state.CurrentPage = 1

	dt := DataTable{Dataset: pd, State: state}
	indices := dt.buildViewIndices()
	if len(indices) != 5 {
		t.Fatalf("len(indices) = %d, want 5", len(indices))
	}
	for i, want := range []int{5, 6, 7, 8, 9} {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTablePagedViewIndicesLastPage(t *testing.T) {
	pd := NewPagedDataset[int](5)
	pd.SetPage(0, []int{10, 20, 30, 40, 50}, 13)
	pd.SetPage(2, []int{110, 120, 130}, -1) // partial last page
	state := NewDataTableState()
	state.CurrentPage = 2

	dt := DataTable{Dataset: pd, State: state}
	indices := dt.buildViewIndices()
	if len(indices) != 3 {
		t.Fatalf("len(indices) = %d, want 3 (partial last page)", len(indices))
	}
	for i, want := range []int{10, 11, 12} {
		if indices[i] != want {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], want)
		}
	}
}

func TestDataTableNewDataTableState(t *testing.T) {
	s := NewDataTableState()
	if s == nil {
		t.Fatal("NewDataTableState should return non-nil")
	}
	if s.Scroll == nil {
		t.Fatal("Scroll should be initialized")
	}
}
