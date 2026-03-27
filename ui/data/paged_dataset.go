package data

// PagedDataset manages loaded pages and triggers reloading via
// DatasetLoadRequestMsg when unloaded slots become visible. (RFC-002 §6.3)
type PagedDataset[ID comparable] struct {
	// TotalCount is -1 when unknown, set after the first load.
	TotalCount int

	// PageSize is the number of items per page.
	PageSize int

	pages     map[int][]ID       // pageIndex -> loaded IDs
	pageState map[int]SlotState  // pageIndex -> loading state
}

// NewPagedDataset creates a PagedDataset with the given page size.
// TotalCount starts at -1 (unknown).
func NewPagedDataset[ID comparable](pageSize int) *PagedDataset[ID] {
	if pageSize <= 0 {
		pageSize = 50
	}
	return &PagedDataset[ID]{
		TotalCount: -1,
		PageSize:   pageSize,
		pages:      make(map[int][]ID),
		pageState:  make(map[int]SlotState),
	}
}

// Len returns the known total count, or -1 if unknown.
func (d *PagedDataset[ID]) Len() int { return d.TotalCount }

// Get returns the item at the given index.
// Returns (id, true) if the item's page is loaded, (zero, false) otherwise.
func (d *PagedDataset[ID]) Get(index int) (ID, bool) {
	pageIdx := index / d.PageSize
	offset := index % d.PageSize
	page, ok := d.pages[pageIdx]
	if !ok || offset >= len(page) {
		var zero ID
		return zero, false
	}
	return page[offset], true
}

// SetPage inserts a loaded page. totalCount updates TotalCount (pass -1 to keep unchanged).
func (d *PagedDataset[ID]) SetPage(pageIndex int, ids []ID, totalCount int) {
	d.pages[pageIndex] = ids
	d.pageState[pageIndex] = SlotLoaded
	if totalCount >= 0 {
		d.TotalCount = totalCount
	}
}

// SetLoading marks a page as currently loading.
func (d *PagedDataset[ID]) SetLoading(pageIndex int) {
	d.pageState[pageIndex] = SlotLoading
}

// SetError marks a page as failed to load.
func (d *PagedDataset[ID]) SetError(pageIndex int) {
	d.pageState[pageIndex] = SlotError
}

// IsPageLoading reports whether a page is currently being loaded.
func (d *PagedDataset[ID]) IsPageLoading(pageIndex int) bool {
	return d.pageState[pageIndex] == SlotLoading
}

// PageState returns the SlotState for a page.
func (d *PagedDataset[ID]) PageState(pageIndex int) SlotState {
	return d.pageState[pageIndex]
}

// PageForIndex returns the page index that contains the given item index.
func (d *PagedDataset[ID]) PageForIndex(index int) int {
	return index / d.PageSize
}

// LoadedCount returns the number of items across all loaded pages.
// Useful when TotalCount is unknown (-1) to determine how many items
// are currently available for rendering.
func (d *PagedDataset[ID]) LoadedCount() int {
	n := 0
	for _, page := range d.pages {
		n += len(page)
	}
	return n
}
