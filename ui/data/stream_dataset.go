package data

// StreamMode controls where new items are inserted.
type StreamMode uint8

const (
	StreamAppend  StreamMode = iota // New items at bottom (log, chat history)
	StreamPrepend                   // New items at top (social feed, inbox)
)

// StreamDataset is for append-only real-time streams (chat, log, feed).
// Len() always returns -1 — the total is never known.
// New items are added via Append or Prepend. (RFC-002 §6.3)
type StreamDataset[ID comparable] struct {
	items []ID
	mode  StreamMode
}

// NewStreamDataset creates an empty StreamDataset with the given mode.
func NewStreamDataset[ID comparable](mode StreamMode) *StreamDataset[ID] {
	return &StreamDataset[ID]{mode: mode}
}

// Len returns -1 — a stream has no known total length.
func (d *StreamDataset[ID]) Len() int { return -1 }

// Count returns the number of currently loaded items.
func (d *StreamDataset[ID]) Count() int { return len(d.items) }

// Get returns the item at the given index.
// Returns (zero, false) if the index is out of range.
func (d *StreamDataset[ID]) Get(i int) (ID, bool) {
	if i < 0 || i >= len(d.items) {
		var zero ID
		return zero, false
	}
	return d.items[i], true
}

// Append adds items at the end of the stream.
func (d *StreamDataset[ID]) Append(ids ...ID) {
	d.items = append(d.items, ids...)
}

// Prepend adds items at the beginning of the stream.
func (d *StreamDataset[ID]) Prepend(ids ...ID) {
	d.items = append(ids, d.items...)
}

// Mode returns the stream mode.
func (d *StreamDataset[ID]) Mode() StreamMode { return d.mode }
