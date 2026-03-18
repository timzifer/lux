// Package ui – reconcile.go implements the widget reconciliation runtime.
//
// The Reconciler walks element trees produced by view functions, assigns
// stable UIDs to widget nodes, persists WidgetState across frames, and
// expands Widget.Render calls. It also provides structural tree comparison
// so the app loop can skip redundant scene rebuilds.
package ui

import (
	"encoding/binary"
	"hash/fnv"
	"time"

	"github.com/timzifer/lux/theme"
)

// Reconciler manages persistent widget state across frames.
type Reconciler struct {
	states   map[UID]WidgetState
	prevTree Element
}

// NewReconciler creates a ready-to-use Reconciler.
func NewReconciler() *Reconciler {
	return &Reconciler{states: make(map[UID]WidgetState)}
}

// Reconcile processes a new element tree: expands widgetElement nodes by
// calling Widget.Render with persisted state, cleans up removed states,
// and reports whether the resolved tree differs from the previous frame.
//
// If dispatcher is non-nil, each widget's RenderCtx.Events is populated
// from the dispatcher's per-UID event buffers (RFC-002 §2.6). If fm is
// non-nil, Focusable widgets are registered for tab-order tracking.
func (r *Reconciler) Reconcile(newTree Element, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager) (Element, bool) {
	seen := make(map[UID]bool)
	resolved := r.resolveTree(newTree, 0, 0, seen, th, send, dispatcher, fm)

	changed := !treeEqual(r.prevTree, resolved)
	r.prevTree = resolved

	// Purge state for widgets no longer in the tree.
	for uid := range r.states {
		if !seen[uid] {
			delete(r.states, uid)
		}
	}
	return resolved, changed
}

// StateFor returns the persisted state for a given UID (test helper).
func (r *Reconciler) StateFor(uid UID) WidgetState {
	return r.states[uid]
}

// StateCount returns the number of tracked widget states.
func (r *Reconciler) StateCount() int {
	return len(r.states)
}

// TickAnimators calls Tick(dt) on every WidgetState that implements
// the Animator interface (RFC-002 §1.3). Returns true if any animation
// is still running (i.e. at least one Tick returned true), signalling
// that the widget tree should be repainted.
func (r *Reconciler) TickAnimators(dt time.Duration) bool {
	anyRunning := false
	for _, state := range r.states {
		if a, ok := state.(Animator); ok {
			if a.Tick(dt) {
				anyRunning = true
			}
		}
	}
	return anyRunning
}

// MakeUID computes a deterministic UID from parent, key, and child index.
// Exported so tests and widgets can predict UIDs.
func MakeUID(parent UID, key string, index int) UID {
	h := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(parent))
	h.Write(buf[:])
	if key != "" {
		h.Write([]byte(key))
	} else {
		binary.LittleEndian.PutUint64(buf[:], uint64(index))
		h.Write(buf[:])
	}
	return UID(h.Sum64())
}

// resolveTree recursively walks the element tree, expanding widgets.
func (r *Reconciler) resolveTree(el Element, parentUID UID, index int, seen map[UID]bool, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager) Element {
	switch node := el.(type) {
	case widgetElement:
		key := node.Key
		if key == "" {
			key = "__widget__"
		}
		uid := MakeUID(parentUID, key, index)
		seen[uid] = true
		state := r.states[uid]

		// Build RenderCtx with dispatched events.
		ctx := RenderCtx{UID: uid, Theme: th, Send: send}
		if dispatcher != nil {
			ctx.Events = dispatcher.EventsFor(uid)
		}

		// Register Focusable widgets with FocusManager.
		if fm != nil {
			if fw, ok := node.W.(Focusable); ok {
				fm.RegisterFocusable(uid, fw.FocusOptions())
			}
		}

		child, newState := node.W.Render(ctx, state)
		r.states[uid] = newState

		// Recursively resolve the widget's output (it may contain more widgets).
		resolved := r.resolveTree(child, uid, 0, seen, th, send, dispatcher, fm)

		// Wrap in widgetBoundsElement so layout can track screen bounds.
		return widgetBoundsElement{WidgetUID: uid, Child: resolved}

	case keyedElement:
		uid := MakeUID(parentUID, node.Key, index)
		child := r.resolveTree(node.Child, uid, 0, seen, th, send, dispatcher, fm)
		return keyedElement{Key: node.Key, Child: child}

	case boxElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm)
		}
		return boxElement{Axis: node.Axis, Children: children}

	case stackElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm)
		}
		return stackElement{Children: children}

	case scrollViewElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm)
		return scrollViewElement{Child: child, MaxHeight: node.MaxHeight, State: node.State}

	case paddingElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm)
		return paddingElement{Insets: node.Insets, Child: child}

	case sizedBoxElement:
		if node.Child != nil {
			child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm)
			return sizedBoxElement{Width: node.Width, Height: node.Height, Child: child}
		}
		return el

	case expandedElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm)
		return expandedElement{Child: child, Grow: node.Grow}

	case flexElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm)
		}
		return flexElement{Direction: node.Direction, Justify: node.Justify, Align: node.Align, Gap: node.Gap, Children: children}

	case gridElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm)
		}
		return gridElement{Columns: node.Columns, RowGap: node.RowGap, ColGap: node.ColGap, Children: children}

	default:
		// Leaf elements (text, button, divider, virtualList, tree, richText, etc.) pass through unchanged.
		return el
	}
}

// treeEqual performs a structural comparison of two element trees.
// Function fields (callbacks) are ignored since they are recreated each frame.
func treeEqual(a, b Element) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch na := a.(type) {
	case emptyElement:
		_, ok := b.(emptyElement)
		return ok
	case textElement:
		nb, ok := b.(textElement)
		return ok && na.Content == nb.Content && na.Style == nb.Style
	case buttonElement:
		nb, ok := b.(buttonElement)
		return ok && na.Label == nb.Label
	case keyedElement:
		nb, ok := b.(keyedElement)
		return ok && na.Key == nb.Key && treeEqual(na.Child, nb.Child)
	case widgetBoundsElement:
		nb, ok := b.(widgetBoundsElement)
		return ok && na.WidgetUID == nb.WidgetUID && treeEqual(na.Child, nb.Child)
	case boxElement:
		nb, ok := b.(boxElement)
		if !ok || na.Axis != nb.Axis || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case stackElement:
		nb, ok := b.(stackElement)
		if !ok || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case scrollViewElement:
		nb, ok := b.(scrollViewElement)
		return ok && na.MaxHeight == nb.MaxHeight && treeEqual(na.Child, nb.Child)
	case dividerElement:
		_, ok := b.(dividerElement)
		return ok
	case spacerElement:
		nb, ok := b.(spacerElement)
		return ok && na.Size == nb.Size
	case iconElement:
		nb, ok := b.(iconElement)
		return ok && na.Name == nb.Name && na.Size == nb.Size
	case checkboxElement:
		nb, ok := b.(checkboxElement)
		return ok && na.Label == nb.Label && na.Checked == nb.Checked
	case radioElement:
		nb, ok := b.(radioElement)
		return ok && na.Label == nb.Label && na.Selected == nb.Selected
	case toggleElement:
		nb, ok := b.(toggleElement)
		return ok && na.On == nb.On
	case sliderElement:
		nb, ok := b.(sliderElement)
		return ok && na.Value == nb.Value
	case progressBarElement:
		nb, ok := b.(progressBarElement)
		return ok && na.Value == nb.Value && na.Indeterminate == nb.Indeterminate
	case textFieldElement:
		nb, ok := b.(textFieldElement)
		return ok && na.Value == nb.Value && na.Placeholder == nb.Placeholder
	case selectElement:
		nb, ok := b.(selectElement)
		if !ok || na.Value != nb.Value || len(na.Options) != len(nb.Options) {
			return false
		}
		for i := range na.Options {
			if na.Options[i] != nb.Options[i] {
				return false
			}
		}
		return true
	case paddingElement:
		nb, ok := b.(paddingElement)
		return ok && na.Insets == nb.Insets && treeEqual(na.Child, nb.Child)
	case sizedBoxElement:
		nb, ok := b.(sizedBoxElement)
		return ok && na.Width == nb.Width && na.Height == nb.Height && treeEqual(na.Child, nb.Child)
	case expandedElement:
		nb, ok := b.(expandedElement)
		return ok && na.Grow == nb.Grow && treeEqual(na.Child, nb.Child)
	case flexElement:
		nb, ok := b.(flexElement)
		if !ok || na.Direction != nb.Direction || na.Justify != nb.Justify || na.Align != nb.Align || na.Gap != nb.Gap || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case gridElement:
		nb, ok := b.(gridElement)
		if !ok || na.Columns != nb.Columns || na.RowGap != nb.RowGap || na.ColGap != nb.ColGap || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case virtualListElement:
		nb, ok := b.(virtualListElement)
		return ok && na.ItemCount == nb.ItemCount && na.ItemHeight == nb.ItemHeight && na.MaxHeight == nb.MaxHeight
	case treeElement:
		// Tree content is dynamic — always re-render.
		_, ok := b.(treeElement)
		return ok && false
	case richTextElement:
		nb, ok := b.(richTextElement)
		if !ok || len(na.Paragraphs) != len(nb.Paragraphs) {
			return false
		}
		for i := range na.Paragraphs {
			pa, pb := na.Paragraphs[i], nb.Paragraphs[i]
			if len(pa.Spans) != len(pb.Spans) {
				return false
			}
			for j := range pa.Spans {
				if pa.Spans[j].Text != pb.Spans[j].Text || pa.Spans[j].Style != pb.Spans[j].Style {
					return false
				}
			}
		}
		return true
	case widgetElement:
		// Unresolved widget elements — should not appear in resolved trees.
		return false
	default:
		return false
	}
}
