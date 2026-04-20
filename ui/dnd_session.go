// Package ui — dnd_session.go implements the framework-internal DnD session
// manager (RFC-005 §4). It tracks active drag sessions, manages drop zone
// registration, and dispatches DragEnter/Over/Leave/Drop events.
//
// The DnDManager lives alongside EventDispatcher and FocusManager as a
// framework-owned coordinator. It is NOT user-space state — drag sessions
// are inherently cross-widget and must be tracked at the framework level.
package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
)

// ── DragSession ─────────────────────────────────────────────────

// DragSessionPhase describes the lifecycle of a drag-and-drop session.
type DragSessionPhase uint8

const (
	DragSessionIdle      DragSessionPhase = iota // no active drag
	DragSessionActive                            // drag in progress
	DragSessionCompleted                         // drag just finished, data available for one BuildScene pass
)

// DragSession holds the state of an active drag-and-drop operation.
// It is created by DnDManager.StartDrag and cleared on EndDrag/CancelDrag.
type DragSession struct {
	Phase           DragSessionPhase
	Data            *input.DragData
	SourceUID       UID
	SourceBounds    draw.Rect
	StartPos        input.GesturePoint
	CurrentPos      input.GesturePoint
	Modifiers       input.ModifierSet
	Operation       input.DragOperation // current operation based on modifiers
	Preview         Element             // element to render as drag ghost
	PreviewBounds   draw.Rect           // size of the preview element
	PreviewOffset   draw.Point          // offset from cursor to preview origin
	ShowPlaceholder bool                // whether to show placeholder at source
}

// ── DropZone ────────────────────────────────────────────────────

// DropZone describes a registered drop target area. Drop zones are
// re-registered every frame during layout (like hit targets) and
// do NOT consume hover animation slots.
type DropZone struct {
	UID      UID
	Bounds   draw.Rect
	Accept   func(data *input.DragData, op input.DragOperation) bool
	Priority int // higher priority wins for nested targets
}

// ── DnDManager ──────────────────────────────────────────────────

// DnDManager coordinates drag-and-drop sessions at the framework level.
// It is created once per window and lives for the entire application
// lifetime (like EventDispatcher).
type DnDManager struct {
	session        DragSession
	dropZones      []DropZone
	hoveredZoneUID UID // UID of hovered zone, 0 = none
	prevHoveredUID UID // for enter/leave detection
}

// NewDnDManager creates a new DnD manager in idle state.
func NewDnDManager() *DnDManager {
	return &DnDManager{}
}

// ── Session Lifecycle ───────────────────────────────────────────

// StartDrag initiates a new drag-and-drop session.
// Called when a DragSource widget detects DragMsg.DragBegan.
func (m *DnDManager) StartDrag(sourceUID UID, data *input.DragData, startPos input.GesturePoint, sourceBounds draw.Rect, preview Element, previewOffset draw.Point, showPlaceholder bool) {
	m.session = DragSession{
		Phase:           DragSessionActive,
		Data:            data,
		SourceUID:       sourceUID,
		SourceBounds:    sourceBounds,
		StartPos:        startPos,
		CurrentPos:      startPos,
		Operation:       input.ResolveOperation(data.AllowedOps, 0),
		Preview:         preview,
		PreviewOffset:   previewOffset,
		ShowPlaceholder: showPlaceholder,
	}
	m.hoveredZoneUID = 0
	m.prevHoveredUID = 0
}

// UpdateDrag updates the cursor position and keyboard modifiers during
// an active drag. Returns false if no session is active.
func (m *DnDManager) UpdateDrag(pos input.GesturePoint, mods input.ModifierSet) bool {
	if m.session.Phase != DragSessionActive {
		return false
	}
	m.session.CurrentPos = pos
	m.session.Modifiers = mods
	m.session.Operation = input.ResolveOperation(m.session.Data.AllowedOps, mods)

	// Hit-test against drop zones.
	m.hoveredZoneUID = m.hitTestDropZone(pos.X, pos.Y)
	return true
}

// EndDrag completes the active drag session. If the cursor is over an
// accepting drop target, the drop is executed and the resolved DropEffect
// is returned. Otherwise DropEffectNone is returned.
//
// The session transitions to DragSessionCompleted (instead of immediately
// resetting) so that widgets like SortableList can detect the drop during
// the subsequent BuildScene/LayoutSelf pass. Call ClearCompletedDrag after
// BuildScene to finalize the reset.
func (m *DnDManager) EndDrag(pos input.GesturePoint, mods input.ModifierSet) input.DropEffect {
	if m.session.Phase != DragSessionActive {
		return input.DropEffectNone
	}

	m.session.CurrentPos = pos
	m.session.Modifiers = mods
	m.session.Operation = input.ResolveOperation(m.session.Data.AllowedOps, mods)

	m.hoveredZoneUID = m.hitTestDropZone(pos.X, pos.Y)

	var effect input.DropEffect
	if zone := m.zoneByUID(m.hoveredZoneUID); zone != nil {
		if zone.Accept != nil && zone.Accept(m.session.Data, m.session.Operation) {
			effect = operationToEffect(m.session.Operation)
		}
	}

	// Transition to Completed instead of resetting — keeps Data and
	// CurrentPos available for LayoutSelf to read during BuildScene.
	m.session.Phase = DragSessionCompleted
	m.hoveredZoneUID = 0
	m.prevHoveredUID = 0
	return effect
}

// CancelDrag aborts the active drag session without dropping.
func (m *DnDManager) CancelDrag() {
	m.reset()
}

func (m *DnDManager) reset() {
	m.session = DragSession{}
	m.hoveredZoneUID = 0
	m.prevHoveredUID = 0
}

// ── Drop Zone Registration ──────────────────────────────────────

// RegisterDropZone adds a drop zone for the current frame. Called during
// layout by DropTarget widgets. Drop zones are cleared each frame via
// ResetDropZones.
func (m *DnDManager) RegisterDropZone(zone DropZone) {
	m.dropZones = append(m.dropZones, zone)
}

// ResetDropZones clears all registered drop zones. Called at the start
// of each BuildScene pass.
func (m *DnDManager) ResetDropZones() {
	m.dropZones = m.dropZones[:0]
	// Don't reset session or hovered state — those persist across frames.
}

// ── Completed Drag ──────────────────────────────────────────────

// CompletedDrag returns the session data from a drag that just finished,
// or nil if no drag completed this frame. The data persists for one
// BuildScene pass so LayoutSelf-based widgets (e.g. SortableList) can
// detect the drop and act on it.
func (m *DnDManager) CompletedDrag() *DragSession {
	if m != nil && m.session.Phase == DragSessionCompleted {
		return &m.session
	}
	return nil
}

// ClearCompletedDrag resets a completed drag session to idle. Called by
// the framework after BuildScene so the data is available for exactly
// one layout pass.
func (m *DnDManager) ClearCompletedDrag() {
	if m != nil && m.session.Phase == DragSessionCompleted {
		m.session = DragSession{}
	}
}

// ── Queries ─────────────────────────────────────────────────────

// IsActive reports whether a drag session is currently in progress.
func (m *DnDManager) IsActive() bool {
	return m != nil && m.session.Phase == DragSessionActive
}

// Session returns a pointer to the current drag session.
// Returns nil if no session is active.
func (m *DnDManager) Session() *DragSession {
	if !m.IsActive() {
		return nil
	}
	return &m.session
}

// HoveredZoneUID returns the UID of the currently hovered drop zone,
// or 0 if no zone is hovered.
func (m *DnDManager) HoveredZoneUID() UID {
	return m.hoveredZoneUID
}

// HoveredZoneAccepts reports whether the currently hovered drop zone
// accepts the active drag data with the current operation.
func (m *DnDManager) HoveredZoneAccepts() bool {
	if !m.IsActive() || m.hoveredZoneUID == 0 {
		return false
	}
	zone := m.zoneByUID(m.hoveredZoneUID)
	return zone != nil && zone.Accept != nil && zone.Accept(m.session.Data, m.session.Operation)
}

// IsDropHovered reports whether the given UID is the currently hovered
// drop zone during an active drag.
func (m *DnDManager) IsDropHovered(uid UID) bool {
	return m.IsActive() && m.hoveredZoneUID == uid
}

// DropZoneCount returns the number of registered drop zones.
func (m *DnDManager) DropZoneCount() int {
	if m == nil {
		return 0
	}
	return len(m.dropZones)
}

// DropZoneUIDs returns the UIDs of all registered drop zones.
// Used by keyboard DnD to cycle through targets.
func (m *DnDManager) DropZoneUIDs() []UID {
	uids := make([]UID, len(m.dropZones))
	for i, z := range m.dropZones {
		uids[i] = z.UID
	}
	return uids
}

// ── Event Dispatch ──────────────────────────────────────────────

// DispatchDnDEvents generates DragEnter, DragOver, and DragLeave events
// based on hover state changes. Call this during Dispatch() after updating
// the drag position.
//
// The appendEvent callback is typically EventDispatcher.appendEvent.
func (m *DnDManager) DispatchDnDEvents(appendEvent func(uid UID, ev InputEvent)) {
	if !m.IsActive() {
		return
	}

	currentUID := m.hoveredZoneUID
	prevUID := m.prevHoveredUID

	// DragLeave for the previous zone (if changed).
	if prevUID != 0 && prevUID != currentUID {
		appendEvent(prevUID, DragLeaveEvent(DragLeaveMsg{
			Data: m.session.Data,
		}))
	}

	// DragEnter for the new zone (if changed).
	if currentUID != 0 && currentUID != prevUID {
		appendEvent(currentUID, DragEnterEvent(DragEnterMsg{
			Data:      m.session.Data,
			Pos:       m.session.CurrentPos,
			Modifiers: m.session.Modifiers,
			Operation: m.session.Operation,
		}))
	}

	// DragOver for the current zone (continuous).
	if currentUID != 0 {
		appendEvent(currentUID, DragOverEvent(DragOverMsg{
			Data:      m.session.Data,
			Pos:       m.session.CurrentPos,
			Modifiers: m.session.Modifiers,
			Operation: m.session.Operation,
		}))
	}

	m.prevHoveredUID = currentUID
}

// DispatchDropEvent generates a Drop event for the currently hovered zone.
// Call this when the drag ends over an accepting target.
func (m *DnDManager) DispatchDropEvent(appendEvent func(uid UID, ev InputEvent), effect input.DropEffect) {
	if !m.IsActive() || m.hoveredZoneUID == 0 {
		return
	}
	appendEvent(m.hoveredZoneUID, DropEvent(DropMsg{
		Data:      m.session.Data,
		Pos:       m.session.CurrentPos,
		Effect:    effect,
		Modifiers: m.session.Modifiers,
	}))
}

// ── Hit-Testing ─────────────────────────────────────────────────

// hitTestDropZone returns the UID of the top-most drop zone containing
// point (x, y), or 0 if none match. When multiple zones overlap, the
// one with the highest priority wins; on tie, the smallest area wins.
func (m *DnDManager) hitTestDropZone(x, y float32) UID {
	pt := draw.Pt(x, y)
	var bestUID UID
	bestPriority := -1
	var bestArea float32

	for i := range m.dropZones {
		z := &m.dropZones[i]
		if !z.Bounds.Contains(pt) {
			continue
		}
		area := z.Bounds.W * z.Bounds.H
		if bestUID == 0 ||
			z.Priority > bestPriority ||
			(z.Priority == bestPriority && area < bestArea) {
			bestUID = z.UID
			bestPriority = z.Priority
			bestArea = area
		}
	}
	return bestUID
}

// zoneByUID returns a pointer to the drop zone with the given UID, or nil.
func (m *DnDManager) zoneByUID(uid UID) *DropZone {
	for i := range m.dropZones {
		if m.dropZones[i].UID == uid {
			return &m.dropZones[i]
		}
	}
	return nil
}

// ── Cursor Resolution ───────────────────────────────────────────

// DragCursor returns the appropriate cursor kind for the current drag state.
func (m *DnDManager) DragCursor() input.CursorKind {
	if !m.IsActive() {
		return input.CursorDefault
	}
	if m.hoveredZoneUID == 0 {
		return input.CursorGrabbing
	}
	if !m.HoveredZoneAccepts() {
		return input.CursorNotAllowed
	}
	// CursorMove is used for all accepted drop operations.
	// The framework does not distinguish copy/link cursors visually
	// beyond the drop highlight — this matches the available CursorKind set.
	return input.CursorMove
}

// ── App-Level Messages ──────────────────────────────────────────

// StartDragSessionMsg initiates a drag-and-drop session (RFC-005 §4).
// Sent by DragSource widgets via ctx.Send when a drag gesture is detected.
// The framework intercepts this message in the app loop and calls
// DnDManager.StartDrag.
type StartDragSessionMsg struct {
	SourceUID       UID
	Data            *input.DragData
	StartPos        input.GesturePoint
	SourceBounds    draw.Rect
	Preview         Element
	PreviewOffset   draw.Point
	ShowPlaceholder bool
}

// DragCompletedMsg is sent to the user model when a drag completes (RFC-005 §4).
type DragCompletedMsg struct {
	Effect   input.DropEffect
	TargetID string
}

// DragCancelledMsg is sent when a drag is cancelled without dropping.
type DragCancelledMsg struct{}

// ── Helpers ─────────────────────────────────────────────────────

// operationToEffect converts a DragOperation to the corresponding DropEffect.
func operationToEffect(op input.DragOperation) input.DropEffect {
	switch {
	case op.Has(input.DragOperationMove):
		return input.DropEffectMove
	case op.Has(input.DragOperationCopy):
		return input.DropEffectCopy
	case op.Has(input.DragOperationLink):
		return input.DropEffectLink
	default:
		return input.DropEffectNone
	}
}
