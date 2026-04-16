// Package ui — input_handler.go extracts framework-internal TextField
// keyboard handling from app/run.go so that both the live run-loop and
// the headless TestApp can share the same logic.
package ui

import (
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/text"
)

// ClipboardProvider abstracts clipboard access for testability.
// The live app uses platform clipboard; tests can use a simple string buffer.
type ClipboardProvider interface {
	GetClipboard() (string, error)
	SetClipboard(string) error
}

// HandleTabNavigation processes Tab / Shift+Tab for focus navigation.
// Returns true if the key was consumed (i.e. it was a Tab key).
func HandleTabNavigation(fm *FocusManager, msg input.KeyMsg, dispatcher *EventDispatcher) bool {
	if msg.Action != input.KeyPress && msg.Action != input.KeyRepeat {
		return false
	}
	if msg.Key != input.KeyTab {
		return false
	}
	oldUID := fm.FocusedUID()
	var newUID UID
	if msg.Modifiers.Has(input.ModShift) {
		newUID = fm.FocusPrev()
	} else {
		newUID = fm.FocusNext()
	}
	if newUID != oldUID && dispatcher != nil {
		dispatcher.QueueFocusChange(oldUID, newUID, FocusSourceTab)
	}
	return true
}

// HandleTextFieldKeyMsg processes a KeyMsg for the currently focused TextField.
// It handles platform shortcuts (Ctrl+C/V/X/A), navigation keys
// (arrows, Home/End), editing keys (Backspace/Delete/Enter), and Escape blur.
//
// Returns true if the InputState was modified (caller should mark model dirty).
// The msg is NOT collected into the dispatcher — the caller must do that.
func HandleTextFieldKeyMsg(fm *FocusManager, msg input.KeyMsg, dispatcher *EventDispatcher, clipboard ClipboardProvider) bool {
	if msg.Action != input.KeyPress && msg.Action != input.KeyRepeat {
		return false
	}
	is := fm.Input
	if is == nil {
		return false
	}

	shift := msg.Modifiers.Has(input.ModShift)
	ctrl := msg.Modifiers.Has(input.ModCtrl) || msg.Modifiers.Has(input.ModSuper)
	dirty := false

	// Platform shortcuts: Ctrl+C/V/X/A.
	if ctrl {
		switch msg.Key {
		case input.KeyC:
			if is.HasSelection() && clipboard != nil {
				_ = clipboard.SetClipboard(is.SelectedText())
			}
			return true

		case input.KeyX:
			if is.HasSelection() {
				if clipboard != nil {
					_ = clipboard.SetClipboard(is.SelectedText())
				}
				is.DeleteSelection()
				is.OnChange(is.Value)
				return true
			}

		case input.KeyV:
			if clipboard != nil {
				if clip, err := clipboard.GetClipboard(); err == nil && clip != "" {
					is.DeleteSelection()
					v := is.Value[:is.CursorOffset] + clip + is.Value[is.CursorOffset:]
					is.CursorOffset += len(clip)
					is.Value = v
					is.ClearSelection()
					is.OnChange(v)
					return true
				}
			}

		case input.KeyA:
			is.SelectionStart = 0
			is.CursorOffset = len(is.Value)
			return true
		}
	}

	// Navigation and editing keys.
	switch msg.Key {
	case input.KeyEnter:
		if is.Multiline {
			is.DeleteSelection()
			v := is.Value[:is.CursorOffset] + "\n" + is.Value[is.CursorOffset:]
			is.CursorOffset++
			is.Value = v
			is.ClearSelection()
			is.OnChange(v)
			dirty = true
		}
	case input.KeyBackspace:
		if is.HasSelection() {
			is.DeleteSelection()
			is.OnChange(is.Value)
			dirty = true
		} else if is.CursorOffset > 0 {
			v, newOff := text.DeleteBackward(is.Value, is.CursorOffset)
			is.Value = v
			is.CursorOffset = newOff
			is.OnChange(v)
			dirty = true
		}
	case input.KeyDelete:
		if is.HasSelection() {
			is.DeleteSelection()
			is.OnChange(is.Value)
			dirty = true
		} else if is.CursorOffset < len(is.Value) {
			v, newOff := text.DeleteForward(is.Value, is.CursorOffset)
			is.Value = v
			is.CursorOffset = newOff
			is.OnChange(v)
			dirty = true
		}
	case input.KeyLeft:
		if shift {
			if is.SelectionStart < 0 {
				is.SelectionStart = is.CursorOffset
			}
		} else if is.HasSelection() {
			a, _ := is.SelectionRange()
			is.CursorOffset = a
			is.ClearSelection()
			return true
		} else {
			is.ClearSelection()
		}
		if ctrl {
			is.CursorOffset = text.PrevWordBoundary(is.Value, is.CursorOffset)
		} else {
			is.CursorOffset = text.PrevGraphemeCluster(is.Value, is.CursorOffset)
		}
		dirty = true
	case input.KeyRight:
		if shift {
			if is.SelectionStart < 0 {
				is.SelectionStart = is.CursorOffset
			}
		} else if is.HasSelection() {
			_, b := is.SelectionRange()
			is.CursorOffset = b
			is.ClearSelection()
			return true
		} else {
			is.ClearSelection()
		}
		if ctrl {
			is.CursorOffset = text.NextWordBoundary(is.Value, is.CursorOffset)
		} else {
			is.CursorOffset = text.NextGraphemeCluster(is.Value, is.CursorOffset)
		}
		dirty = true
	case input.KeyUp:
		if is.Multiline {
			if shift {
				if is.SelectionStart < 0 {
					is.SelectionStart = is.CursorOffset
				}
			} else {
				is.ClearSelection()
			}
			is.CursorOffset = text.CursorUp(is.Value, is.CursorOffset)
			dirty = true
		}
	case input.KeyDown:
		if is.Multiline {
			if shift {
				if is.SelectionStart < 0 {
					is.SelectionStart = is.CursorOffset
				}
			} else {
				is.ClearSelection()
			}
			is.CursorOffset = text.CursorDown(is.Value, is.CursorOffset)
			dirty = true
		}
	case input.KeyHome:
		if shift {
			if is.SelectionStart < 0 {
				is.SelectionStart = is.CursorOffset
			}
		} else {
			is.ClearSelection()
		}
		if is.Multiline && !ctrl {
			is.CursorOffset = text.LineStart(is.Value, is.CursorOffset)
		} else {
			is.CursorOffset = 0
		}
		dirty = true
	case input.KeyEnd:
		if shift {
			if is.SelectionStart < 0 {
				is.SelectionStart = is.CursorOffset
			}
		} else {
			is.ClearSelection()
		}
		if is.Multiline && !ctrl {
			is.CursorOffset = text.LineEnd(is.Value, is.CursorOffset)
		} else {
			is.CursorOffset = len(is.Value)
		}
		dirty = true
	case input.KeyEscape:
		oldUID := fm.FocusedUID()
		fm.Blur()
		if dispatcher != nil {
			dispatcher.QueueFocusChange(oldUID, 0, FocusSourceProgram)
		}
		dirty = true
	}
	return dirty
}

// HandleCharMsg processes a CharMsg for the currently focused TextField.
// Returns true if the InputState was modified.
func HandleCharMsg(fm *FocusManager, msg input.CharMsg) bool {
	is := fm.Input
	if is == nil {
		return false
	}
	// Skip CR and LF -- Enter is already handled by KeyMsg(KeyEnter).
	// On Windows/GLFW both a KeyMsg and a CharMsg fire for Enter,
	// which would insert a double newline without this guard.
	if msg.Char < 32 {
		return false
	}
	is.DeleteSelection()
	ch := string(msg.Char)
	v := is.Value[:is.CursorOffset] + ch + is.Value[is.CursorOffset:]
	is.CursorOffset += len(ch)
	is.Value = v
	is.ClearSelection()
	is.OnChange(v)
	return true
}

// HandleTextInputMsg processes a TextInputMsg (IME batch input) for the
// currently focused TextField. Returns true if the InputState was modified.
func HandleTextInputMsg(fm *FocusManager, msg input.TextInputMsg) bool {
	is := fm.Input
	if is == nil || msg.Text == "" {
		return false
	}
	is.DeleteSelection()
	v := is.Value[:is.CursorOffset] + msg.Text + is.Value[is.CursorOffset:]
	is.CursorOffset += len(msg.Text)
	is.Value = v
	is.ClearSelection()
	is.OnChange(v)
	return true
}

// HandleIMEComposeMsg updates the composition state on the focused TextField.
// Returns true if the InputState was modified.
func HandleIMEComposeMsg(fm *FocusManager, msg input.IMEComposeMsg) bool {
	is := fm.Input
	if is == nil {
		return false
	}
	is.ComposeText = msg.Text
	is.ComposeCursorStart = msg.CursorStart
	is.ComposeCursorEnd = msg.CursorEnd
	return true
}

// HandleIMECommitMsg inserts committed IME text into the focused TextField.
// Returns true if the InputState was modified.
func HandleIMECommitMsg(fm *FocusManager, msg input.IMECommitMsg) bool {
	is := fm.Input
	if is == nil || msg.Text == "" {
		return false
	}
	is.ComposeText = "" // clear composition
	is.DeleteSelection()
	v := is.Value[:is.CursorOffset] + msg.Text + is.Value[is.CursorOffset:]
	is.CursorOffset += len(msg.Text)
	is.Value = v
	is.ClearSelection()
	is.OnChange(v)
	return true
}
