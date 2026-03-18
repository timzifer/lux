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
func (r *Reconciler) Reconcile(newTree Element, th theme.Theme, send func(any)) (Element, bool) {
	seen := make(map[UID]bool)
	resolved := r.resolveTree(newTree, 0, 0, seen, th, send)

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
func (r *Reconciler) resolveTree(el Element, parentUID UID, index int, seen map[UID]bool, th theme.Theme, send func(any)) Element {
	switch node := el.(type) {
	case widgetElement:
		key := node.Key
		if key == "" {
			key = "__widget__"
		}
		uid := MakeUID(parentUID, key, index)
		seen[uid] = true
		state := r.states[uid]
		ctx := RenderCtx{UID: uid, Theme: th, Send: send}
		child, newState := node.W.Render(ctx, state)
		r.states[uid] = newState
		// Recursively resolve the widget's output (it may contain more widgets).
		return r.resolveTree(child, uid, 0, seen, th, send)

	case keyedElement:
		uid := MakeUID(parentUID, node.Key, index)
		child := r.resolveTree(node.Child, uid, 0, seen, th, send)
		return keyedElement{Key: node.Key, Child: child}

	case boxElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send)
		}
		return boxElement{Axis: node.Axis, Children: children}

	case stackElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send)
		}
		return stackElement{Children: children}

	case scrollViewElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send)
		return scrollViewElement{Child: child, MaxHeight: node.MaxHeight, State: node.State}

	default:
		// Leaf elements (text, button, divider, etc.) pass through unchanged.
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
	case widgetElement:
		// Unresolved widget elements — should not appear in resolved trees.
		return false
	default:
		return false
	}
}
