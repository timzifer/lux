package data

// SliceDataset wraps a slice — length immediately known, all items always loaded.
// Drop-in replacement for the old ItemCount + BuildItem approach. (RFC-002 §6.3)
type SliceDataset[ID comparable] struct {
	Items []ID
}

// NewSliceDataset creates a SliceDataset from the given items.
func NewSliceDataset[ID comparable](items []ID) *SliceDataset[ID] {
	return &SliceDataset[ID]{Items: items}
}

// Len returns the number of items.
func (d *SliceDataset[ID]) Len() int { return len(d.Items) }

// Get returns the item at index i. Always loaded.
func (d *SliceDataset[ID]) Get(i int) (ID, bool) { return d.Items[i], true }
