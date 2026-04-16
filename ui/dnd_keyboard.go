// Package ui — dnd_keyboard.go implements keyboard-based drag-and-drop
// as an accessible alternative to mouse/touch dragging (RFC-005 §11).
//
// Keyboard DnD workflow:
//  1. User focuses a DragSource and presses Space/Enter → enters drag mode
//  2. Tab/Shift+Tab cycles through available DropTargets (highlighted)
//  3. Enter confirms the drop at the focused target
//  4. Escape cancels the drag
//  5. Arrow keys in SortableList move the item up/down
//
// The keyboard DnD mode is driven by FilterCollectedEvents (Global Handler
// Layer, RFC-002 §2.8) so it intercepts keys before normal focus handling.
package ui

import (
	"github.com/timzifer/lux/input"
)

// DnDKeyboardState tracks the keyboard-based drag mode.
type DnDKeyboardState struct {
	Active        bool
	SourceUID     UID
	Data          *input.DragData
	TargetUIDs    []UID // available drop targets
	FocusedTarget int   // index in TargetUIDs (-1 = none)
}

// NewDnDKeyboardState creates a ready-to-use keyboard DnD state.
func NewDnDKeyboardState() *DnDKeyboardState {
	return &DnDKeyboardState{
		FocusedTarget: -1,
	}
}

// StartKeyboardDrag initiates keyboard-based drag mode.
func (ks *DnDKeyboardState) StartKeyboardDrag(sourceUID UID, data *input.DragData, targets []UID) {
	ks.Active = true
	ks.SourceUID = sourceUID
	ks.Data = data
	ks.TargetUIDs = targets
	ks.FocusedTarget = -1
	if len(targets) > 0 {
		ks.FocusedTarget = 0
	}
}

// Cancel ends keyboard drag mode without dropping.
func (ks *DnDKeyboardState) Cancel() {
	ks.Active = false
	ks.SourceUID = 0
	ks.Data = nil
	ks.TargetUIDs = nil
	ks.FocusedTarget = -1
}

// NextTarget moves focus to the next drop target.
func (ks *DnDKeyboardState) NextTarget() {
	if !ks.Active || len(ks.TargetUIDs) == 0 {
		return
	}
	ks.FocusedTarget = (ks.FocusedTarget + 1) % len(ks.TargetUIDs)
}

// PrevTarget moves focus to the previous drop target.
func (ks *DnDKeyboardState) PrevTarget() {
	if !ks.Active || len(ks.TargetUIDs) == 0 {
		return
	}
	ks.FocusedTarget--
	if ks.FocusedTarget < 0 {
		ks.FocusedTarget = len(ks.TargetUIDs) - 1
	}
}

// FocusedTargetUID returns the UID of the currently focused drop target,
// or 0 if no target is focused.
func (ks *DnDKeyboardState) FocusedTargetUID() UID {
	if !ks.Active || ks.FocusedTarget < 0 || ks.FocusedTarget >= len(ks.TargetUIDs) {
		return 0
	}
	return ks.TargetUIDs[ks.FocusedTarget]
}

// HandleKeyboardDnD processes keyboard events during an active keyboard
// drag session. Returns true if the event was consumed.
//
// This function is intended to be called from FilterCollectedEvents
// in the EventDispatcher to intercept keys before normal focus handling.
func HandleKeyboardDnD(ks *DnDKeyboardState, dnd *DnDManager, ev InputEvent, appendEvent func(uid UID, ev InputEvent)) bool {
	if ks == nil || !ks.Active {
		return false
	}

	if ev.Kind != EventKey || ev.Key == nil {
		return false
	}

	key := ev.Key
	if key.Action != input.KeyPress && key.Action != input.KeyRepeat {
		return false
	}

	switch key.Key {
	case input.KeyEscape:
		// Cancel the keyboard drag.
		ks.Cancel()
		dnd.CancelDrag()
		return true

	case input.KeyTab:
		// Cycle through drop targets.
		if key.Modifiers.Has(input.ModShift) {
			ks.PrevTarget()
		} else {
			ks.NextTarget()
		}
		// Highlight the new target by moving the DnD hover.
		if targetUID := ks.FocusedTargetUID(); targetUID != 0 {
			// Update DnD manager to hover over the keyboard-selected zone.
			dnd.hoveredZoneUID = targetUID
			dnd.DispatchDnDEvents(appendEvent)
			dnd.prevHoveredUID = dnd.hoveredZoneUID
		}
		return true

	case input.KeyEnter, input.KeySpace:
		// Confirm drop at the focused target.
		if targetUID := ks.FocusedTargetUID(); targetUID != 0 {
			if dnd.HoveredZoneAccepts() {
				effect := operationToEffect(dnd.session.Operation)
				dnd.DispatchDropEvent(appendEvent, effect)
				dnd.EndDrag(dnd.session.CurrentPos, dnd.session.Modifiers)
			}
		}
		ks.Cancel()
		return true

	case input.KeyUp:
		ks.PrevTarget()
		return true

	case input.KeyDown:
		ks.NextTarget()
		return true
	}

	return false
}
