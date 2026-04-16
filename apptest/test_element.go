package apptest

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/ui"
)

// TestElement is a handle to an accessibility tree node with methods
// for programmatic interaction and state inspection.
type TestElement struct {
	node         *a11y.AccessTreeNode
	sendFn       func(any)
	stepFn       func()
	hitMapFn     func() *hit.Map
	fmFn         func() *ui.FocusManager
	dispatcherFn func() *ui.EventDispatcher
	accessTreeFn func() *a11y.AccessTree
	queryFn      func(Selector) *TestElement
	queryAllFn   func(Selector) []*TestElement
}

// ── State Queries ───────────────────────────────────────────────

// Text returns the node's label.
func (e *TestElement) Text() string {
	if e == nil || e.node == nil {
		return ""
	}
	return e.node.Node.Label
}

// Value returns the node's value (e.g. text field content).
func (e *TestElement) Value() string {
	if e == nil || e.node == nil {
		return ""
	}
	return e.node.Node.Value
}

// Role returns the node's accessibility role.
func (e *TestElement) Role() a11y.AccessRole {
	if e == nil || e.node == nil {
		return 0
	}
	return e.node.Node.Role
}

// ID returns the node's accessibility ID.
func (e *TestElement) ID() a11y.AccessNodeID {
	if e == nil || e.node == nil {
		return 0
	}
	return e.node.ID
}

// Bounds returns the node's bounding rectangle.
func (e *TestElement) Bounds() a11y.Rect {
	if e == nil || e.node == nil {
		return a11y.Rect{}
	}
	return e.node.Bounds
}

// IsEnabled reports whether the element is not disabled.
func (e *TestElement) IsEnabled() bool {
	if e == nil || e.node == nil {
		return false
	}
	return !e.node.Node.States.Disabled
}

// IsFocused reports whether this element currently has focus.
func (e *TestElement) IsFocused() bool {
	if e == nil || e.node == nil {
		return false
	}
	tree := e.accessTreeFn()
	return tree.FocusedID == e.node.ID
}

// IsChecked reports whether the element is checked (checkbox, toggle).
func (e *TestElement) IsChecked() bool {
	if e == nil || e.node == nil {
		return false
	}
	return e.node.Node.States.Checked
}

// IsSelected reports whether the element is selected.
func (e *TestElement) IsSelected() bool {
	if e == nil || e.node == nil {
		return false
	}
	return e.node.Node.States.Selected
}

// IsExpanded reports whether the element is expanded.
func (e *TestElement) IsExpanded() bool {
	if e == nil || e.node == nil {
		return false
	}
	return e.node.Node.States.Expanded
}

// Exists reports whether the element was found (non-nil).
func (e *TestElement) Exists() bool {
	return e != nil && e.node != nil
}

// Children returns child elements.
func (e *TestElement) Children() []*TestElement {
	if e == nil || e.node == nil {
		return nil
	}
	tree := e.accessTreeFn()
	children := tree.Children(e.node)
	result := make([]*TestElement, len(children))
	for i, child := range children {
		result[i] = &TestElement{
			node:         child,
			sendFn:       e.sendFn,
			stepFn:       e.stepFn,
			hitMapFn:     e.hitMapFn,
			fmFn:         e.fmFn,
			dispatcherFn: e.dispatcherFn,
			accessTreeFn: e.accessTreeFn,
			queryFn:      e.queryFn,
			queryAllFn:   e.queryAllFn,
		}
	}
	return result
}

// ── Interactions ────────────────────────────────────────────────

// Click simulates activating the element.
// It first tries the accessibility action ("activate"), which is the most
// reliable path for buttons and checkboxes. Falls back to hit-map
// coordinate-based click for elements without a11y actions.
func (e *TestElement) Click() {
	if e == nil || e.node == nil {
		return
	}

	// Prefer accessibility action trigger (same mechanism screen readers use).
	for _, action := range e.node.Node.Actions {
		if action.Name == "activate" && action.Trigger != nil {
			action.Trigger()
			e.stepFn()
			return
		}
	}

	// Fallback: coordinate-based click via hit map.
	cx, cy := e.center()
	hm := e.hitMapFn()
	fm := e.fmFn()
	dispatcher := e.dispatcherFn()

	// Blur current focus (matches run.go OnMouseButton behavior).
	oldUID := fm.FocusedUID()
	if oldUID != 0 {
		fm.Blur()
		dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceClick)
	}

	if target := hm.HitTest(cx, cy); target != nil {
		if target.OnClickAt != nil {
			target.OnClickAt(cx, cy)
		} else if target.OnClick != nil {
			target.OnClick()
		}
	}

	e.sendFn(input.MouseMsg{X: cx, Y: cy, Button: input.MouseButtonLeft, Action: input.MousePress})
	e.stepFn()
	e.sendFn(input.MouseMsg{X: cx, Y: cy, Button: input.MouseButtonLeft, Action: input.MouseRelease})
	e.stepFn()
}

// Type types the given text into the element. It focuses the element first,
// then sends CharMsg for each rune.
func (e *TestElement) Type(text string) {
	if e == nil || e.node == nil {
		return
	}
	e.Focus()
	for _, r := range text {
		e.sendFn(input.CharMsg{Char: r})
		e.stepFn()
	}
}

// Focus moves keyboard focus to this element.
func (e *TestElement) Focus() {
	if e == nil || e.node == nil {
		return
	}
	// Use click-based focusing: click at center to trigger FocusOnClick.
	cx, cy := e.center()
	hm := e.hitMapFn()
	fm := e.fmFn()
	dispatcher := e.dispatcherFn()

	oldUID := fm.FocusedUID()
	if oldUID != 0 {
		fm.Blur()
		dispatcher.QueueFocusChange(oldUID, 0, ui.FocusSourceClick)
	}

	if target := hm.HitTest(cx, cy); target != nil {
		if target.OnClickAt != nil {
			target.OnClickAt(cx, cy)
		} else if target.OnClick != nil {
			target.OnClick()
		}
	}
	e.stepFn()
}

// Blur removes focus from this element.
func (e *TestElement) Blur() {
	if e == nil || e.node == nil {
		return
	}
	e.sendFn(ui.ReleaseFocusMsg{})
	e.stepFn()
}

// Press simulates a key press.
func (e *TestElement) Press(key input.Key, mods ...input.ModifierSet) {
	if e == nil || e.node == nil {
		return
	}
	var m input.ModifierSet
	if len(mods) > 0 {
		m = mods[0]
	}
	e.sendFn(input.KeyMsg{Key: key, Action: input.KeyPress, Modifiers: m})
	e.stepFn()
}

// Scroll simulates a scroll event at the center of the element.
func (e *TestElement) Scroll(deltaY float32) {
	if e == nil || e.node == nil {
		return
	}
	cx, cy := e.center()
	hm := e.hitMapFn()
	if target := hm.HitTestScroll(cx, cy); target != nil {
		target.OnScroll(deltaY * 30) // 30dp per scroll unit, matching run.go
	}
	e.sendFn(input.ScrollMsg{X: cx, Y: cy, DeltaY: deltaY})
	e.stepFn()
}

// DragTo simulates a drag from this element to the target element.
func (e *TestElement) DragTo(target *TestElement) {
	if e == nil || e.node == nil || target == nil || target.node == nil {
		return
	}
	sx, sy := e.center()
	tx, ty := target.center()
	hm := e.hitMapFn()

	// Press at source.
	if ht := hm.HitTest(sx, sy); ht != nil {
		if ht.OnClickAt != nil {
			ht.OnClickAt(sx, sy)
		}
	}
	e.sendFn(input.MouseMsg{X: sx, Y: sy, Button: input.MouseButtonLeft, Action: input.MousePress})
	e.stepFn()

	// Move to target.
	e.sendFn(input.MouseMsg{X: tx, Y: ty, Action: input.MouseMove})
	e.stepFn()

	// Release at target.
	e.sendFn(input.MouseMsg{X: tx, Y: ty, Button: input.MouseButtonLeft, Action: input.MouseRelease})
	e.stepFn()
}

// center returns the center point of the element's bounds.
func (e *TestElement) center() (float32, float32) {
	b := e.node.Bounds
	return float32(b.X + b.Width/2), float32(b.Y + b.Height/2)
}

// ── Assertions ──────────────────────────────────────────────────

// AssertExists fails the test if the element was not found.
func (e *TestElement) AssertExists(t testing.TB) *TestElement {
	t.Helper()
	if !e.Exists() {
		t.Fatal("expected element to exist, but it was not found")
	}
	return e
}

// AssertText fails the test if the element's label doesn't match.
func (e *TestElement) AssertText(t testing.TB, expected string) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if got := e.Text(); got != expected {
		t.Errorf("expected text %q, got %q", expected, got)
	}
	return e
}

// AssertValue fails the test if the element's value doesn't match.
func (e *TestElement) AssertValue(t testing.TB, expected string) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if got := e.Value(); got != expected {
		t.Errorf("expected value %q, got %q", expected, got)
	}
	return e
}

// AssertRole fails the test if the element's role doesn't match.
func (e *TestElement) AssertRole(t testing.TB, expected a11y.AccessRole) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if got := e.Role(); got != expected {
		t.Errorf("expected role %v, got %v", expected, got)
	}
	return e
}

// AssertEnabled fails the test if the element is disabled.
func (e *TestElement) AssertEnabled(t testing.TB) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if !e.IsEnabled() {
		t.Error("expected element to be enabled, but it is disabled")
	}
	return e
}

// AssertDisabled fails the test if the element is enabled.
func (e *TestElement) AssertDisabled(t testing.TB) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if e.IsEnabled() {
		t.Error("expected element to be disabled, but it is enabled")
	}
	return e
}

// AssertFocused fails the test if the element does not have focus.
func (e *TestElement) AssertFocused(t testing.TB) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if !e.IsFocused() {
		t.Error("expected element to be focused")
	}
	return e
}

// AssertChecked fails the test if the element is not checked.
func (e *TestElement) AssertChecked(t testing.TB) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if !e.IsChecked() {
		t.Error("expected element to be checked")
	}
	return e
}

// AssertNotChecked fails the test if the element is checked.
func (e *TestElement) AssertNotChecked(t testing.TB) *TestElement {
	t.Helper()
	e.AssertExists(t)
	if e.IsChecked() {
		t.Error("expected element to not be checked")
	}
	return e
}
