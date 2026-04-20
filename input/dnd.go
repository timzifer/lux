// Package input — dnd.go defines the data-transfer model and operation
// types for drag-and-drop operations (RFC-005 §3).
//
// These are pure value types with no framework dependencies, matching the
// pattern of gesture.go (TapMsg, DragMsg, etc.).
package input

// ── Drag Operation ──────────────────────────────────────────────

// DragOperation is a bitfield describing what a drag source supports.
// Multiple operations can be combined (e.g. DragOperationMove | DragOperationCopy).
type DragOperation uint8

const (
	// DragOperationNone indicates no drag operation is allowed.
	DragOperationNone DragOperation = 0
	// DragOperationMove indicates the source data will be moved to the target.
	DragOperationMove DragOperation = 1 << iota
	// DragOperationCopy indicates the source data will be copied to the target.
	DragOperationCopy
	// DragOperationLink indicates the target will receive a reference to the source.
	DragOperationLink
)

// Has reports whether op includes the given flag.
func (op DragOperation) Has(flag DragOperation) bool {
	return op&flag != 0
}

// ── Drop Effect ─────────────────────────────────────────────────

// DropEffect describes the resolved operation at a drop target —
// what actually happened (or would happen) when data is dropped.
type DropEffect uint8

const (
	DropEffectNone DropEffect = iota
	DropEffectMove
	DropEffectCopy
	DropEffectLink
)

// String returns a human-readable name for the drop effect.
func (e DropEffect) String() string {
	switch e {
	case DropEffectMove:
		return "move"
	case DropEffectCopy:
		return "copy"
	case DropEffectLink:
		return "link"
	default:
		return "none"
	}
}

// ── Drag Item ───────────────────────────────────────────────────

// DragItem represents a single piece of dragged content identified
// by a MIME type. Multiple items with different MIME types can be
// carried in a single DragData to support format negotiation.
type DragItem struct {
	// MIMEType identifies the data format.
	// Standard types: "text/plain", "text/uri-list", "application/json".
	// Lux-internal types: "application/x-lux-sortable-key",
	// "application/x-lux-widget-id".
	MIMEType string

	// Data is the payload. The concrete Go type depends on MIMEType.
	// For "text/plain" this is a string, for structured types it may
	// be any serializable value.
	Data any
}

// ── Drag Data ───────────────────────────────────────────────────

// DragData is the transfer model for a drag-and-drop operation.
// It carries one or more typed items and declares the allowed
// operations the source supports.
type DragData struct {
	// Items contains the dragged payload(s).
	Items []DragItem

	// AllowedOps declares which operations the source supports.
	// Default: DragOperationMove.
	AllowedOps DragOperation

	// SourceID is an opaque identifier of the drag source.
	// Used by drop targets to detect same-source drops (e.g. to
	// prevent dropping a list item onto its own list position).
	SourceID string
}

// HasType reports whether the drag data contains an item with the
// given MIME type.
func (d *DragData) HasType(mime string) bool {
	if d == nil {
		return false
	}
	for _, item := range d.Items {
		if item.MIMEType == mime {
			return true
		}
	}
	return false
}

// Get returns the data of the first item matching the MIME type.
// Returns (nil, false) if no item matches.
func (d *DragData) Get(mime string) (any, bool) {
	if d == nil {
		return nil, false
	}
	for _, item := range d.Items {
		if item.MIMEType == mime {
			return item.Data, true
		}
	}
	return nil, false
}

// SetText adds or replaces a "text/plain" item with the given string.
func (d *DragData) SetText(s string) {
	if d == nil {
		return
	}
	for i, item := range d.Items {
		if item.MIMEType == "text/plain" {
			d.Items[i].Data = s
			return
		}
	}
	d.Items = append(d.Items, DragItem{MIMEType: "text/plain", Data: s})
}

// Text returns the "text/plain" content, or "" if none exists.
func (d *DragData) Text() string {
	v, ok := d.Get("text/plain")
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Types returns the MIME types of all items.
func (d *DragData) Types() []string {
	if d == nil {
		return nil
	}
	types := make([]string, len(d.Items))
	for i, item := range d.Items {
		types[i] = item.MIMEType
	}
	return types
}

// ── Well-Known MIME Types ───────────────────────────────────────

const (
	// MIMEText is the standard MIME type for plain text.
	MIMEText = "text/plain"
	// MIMEURIList is the standard MIME type for a list of URIs.
	MIMEURIList = "text/uri-list"
	// MIMEJSON is the standard MIME type for JSON data.
	MIMEJSON = "application/json"
	// MIMESortableKey is a Lux-internal type for sortable list item keys.
	MIMESortableKey = "application/x-lux-sortable-key"
	// MIMESortableGroup is a Lux-internal type carrying the GroupID of a SortableList.
	MIMESortableGroup = "application/x-lux-sortable-group"
	// MIMEWidgetID is a Lux-internal type carrying a widget identifier.
	MIMEWidgetID = "application/x-lux-widget-id"
)

// ── Convenience Constructors ────────────────────────────────────

// NewDragData creates a DragData with a single item and default Move operation.
func NewDragData(mime string, data any) *DragData {
	return &DragData{
		Items:      []DragItem{{MIMEType: mime, Data: data}},
		AllowedOps: DragOperationMove,
	}
}

// NewTextDragData creates a DragData carrying plain text with Move + Copy operations.
func NewTextDragData(text string) *DragData {
	return &DragData{
		Items:      []DragItem{{MIMEType: MIMEText, Data: text}},
		AllowedOps: DragOperationMove | DragOperationCopy,
	}
}

// ── Modifier-to-Operation Resolution ────────────────────────────

// ResolveOperation determines the drag operation based on allowed
// operations and the keyboard modifiers held during the drag.
//
//	No modifier  → Move (if allowed), else Copy
//	Ctrl         → Copy (if allowed)
//	Ctrl+Shift   → Link (if allowed)
//	Shift        → Move (if allowed, explicit)
func ResolveOperation(allowed DragOperation, mods ModifierSet) DragOperation {
	switch {
	case mods.Has(ModCtrl) && mods.Has(ModShift):
		if allowed.Has(DragOperationLink) {
			return DragOperationLink
		}
	case mods.Has(ModCtrl):
		if allowed.Has(DragOperationCopy) {
			return DragOperationCopy
		}
	case mods.Has(ModShift):
		if allowed.Has(DragOperationMove) {
			return DragOperationMove
		}
	}
	// Default: prefer Move, fallback to Copy, then Link.
	if allowed.Has(DragOperationMove) {
		return DragOperationMove
	}
	if allowed.Has(DragOperationCopy) {
		return DragOperationCopy
	}
	if allowed.Has(DragOperationLink) {
		return DragOperationLink
	}
	return DragOperationNone
}
