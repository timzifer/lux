package data

import "github.com/timzifer/lux/ui"

// Dataset[ID] abstracts over all length scenarios for data-driven widgets.
// It replaces ItemCount int and RootIDs []ID in VirtualList and Tree.
// (RFC-002 §6.2)
type Dataset[ID comparable] interface {
	// Len returns the known length.
	// -1 = length unknown (not yet loaded or never known).
	Len() int

	// Get returns the item at index i.
	// loaded=false means the item is not yet available (loading).
	// The widget shows a skeleton/placeholder for unloaded slots.
	Get(index int) (id ID, loaded bool)
}

// DatasetSlot describes the state of a single index.
// Used internally by Dataset implementations.
type DatasetSlot[ID comparable] struct {
	ID    ID
	State SlotState
}

// SlotState represents the loading state of a dataset slot.
type SlotState uint8

const (
	SlotLoaded  SlotState = iota // ID is available
	SlotLoading                  // Request in progress
	SlotError                    // Loading failed
)

// DatasetLoadRequestMsg is sent automatically by VirtualList when an unloaded
// slot scrolls into the viewport. The user's update function decides whether
// and how to load the data. (RFC-002 §6.5)
type DatasetLoadRequestMsg struct {
	WidgetUID  ui.UID // which widget triggered the request
	PageIndex  int    // which page is needed
	StartIndex int    // first unloaded index
	EndIndex   int    // last unloaded index (inclusive)
}
