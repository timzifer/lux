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

	"github.com/timzifer/lux/interaction"
	"github.com/timzifer/lux/theme"
)

// Reconciler manages persistent widget state across frames.
type Reconciler struct {
	states      map[UID]WidgetState
	widgets     map[UID]Widget  // previous widget instance per UID (for Equatable)
	resolvedSub map[UID]Element // previous resolved subtree per UID (for Equatable skip)
	prevTree    Element
	themeCache  map[theme.Theme]*theme.CachedTheme // reuse CachedTheme across frames

	lastDirtyUIDs []uint64 // UIDs of widgets that were dirty in the last CheckDirtyTrackers call
}

// NewReconciler creates a ready-to-use Reconciler.
func NewReconciler() *Reconciler {
	return &Reconciler{
		states:      make(map[UID]WidgetState),
		widgets:     make(map[UID]Widget),
		resolvedSub: make(map[UID]Element),
		themeCache:  make(map[theme.Theme]*theme.CachedTheme),
	}
}

// Reconcile processes a new element tree: expands WidgetElement nodes by
// calling Widget.Render with persisted state, cleans up removed states,
// and reports whether the resolved tree differs from the previous frame.
//
// If dispatcher is non-nil, each widget's RenderCtx.Events is populated
// from the dispatcher's per-UID event buffers (RFC-002 §2.6). If fm is
// non-nil, Focusable widgets are registered for tab-order tracking.
func (r *Reconciler) Reconcile(newTree Element, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager, locale string, profile *interaction.InteractionProfile) (Element, bool) {
	seen := make(map[UID]bool)
	resolved := r.resolveTree(newTree, 0, 0, seen, th, send, dispatcher, fm, locale, profile)

	changed := !treeEqual(r.prevTree, resolved)
	r.prevTree = resolved

	// Purge state for widgets no longer in the tree.
	for uid := range r.states {
		if !seen[uid] {
			delete(r.states, uid)
			delete(r.widgets, uid)
			delete(r.resolvedSub, uid)
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

// CheckDirtyTrackers iterates all persisted WidgetState values and returns
// true if any implements DirtyTracker and reports IsDirty() == true
// (RFC-001 §6.4). Dirty flags are cleared after being consumed so that
// the next frame starts clean.
func (r *Reconciler) CheckDirtyTrackers() bool {
	r.lastDirtyUIDs = r.lastDirtyUIDs[:0]
	anyDirty := false
	for uid, state := range r.states {
		if dt, ok := state.(DirtyTracker); ok {
			if dt.IsDirty() {
				anyDirty = true
				r.lastDirtyUIDs = append(r.lastDirtyUIDs, uint64(uid))
				dt.ClearDirty()
			}
		}
	}
	return anyDirty
}

// DirtyUIDs returns the UIDs of widgets that were dirty in the most recent
// CheckDirtyTrackers call. Used by the Inspector to highlight repainted widgets.
func (r *Reconciler) DirtyUIDs() []uint64 {
	return r.lastDirtyUIDs
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
func (r *Reconciler) resolveTree(el Element, parentUID UID, index int, seen map[UID]bool, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager, locale string, profile *interaction.InteractionProfile) Element {
	// Interface-based dispatch for sub-package element types.
	if cr, ok := el.(ChildResolver); ok {
		return cr.ResolveChildren(func(child Element, childIndex int) Element {
			return r.resolveTree(child, parentUID, childIndex, seen, th, send, dispatcher, fm, locale, profile)
		})
	}
	switch node := el.(type) {
	case WidgetElement:
		key := node.Key
		if key == "" {
			key = "__widget__"
		}
		uid := MakeUID(parentUID, key, index)
		seen[uid] = true

		// Equatable short-circuit (RFC-001 §6.4): if the widget implements
		// Equatable and reports equal props, reuse previous output.
		if eq, ok := node.W.(Equatable); ok {
			if prev, hasPrev := r.widgets[uid]; hasPrev {
				if eq.Equal(prev) {
					if sub, hasSub := r.resolvedSub[uid]; hasSub {
						r.widgets[uid] = node.W
						return WidgetBoundsElement{WidgetUID: uid, Child: sub}
					}
				}
			}
		}

		state := r.states[uid]

		// Build RenderCtx with dispatched events.
		ctx := RenderCtx{UID: uid, Theme: th, Send: send, Locale: locale, InteractionProfile: profile}
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
		r.widgets[uid] = node.W

		// Recursively resolve the widget's output (it may contain more widgets).
		resolved := r.resolveTree(child, uid, 0, seen, th, send, dispatcher, fm, locale, profile)
		r.resolvedSub[uid] = resolved

		// Wrap in WidgetBoundsElement so layout can track screen bounds.
		return WidgetBoundsElement{WidgetUID: uid, Child: resolved}

	case KeyedElement:
		uid := MakeUID(parentUID, node.Key, index)
		child := r.resolveTree(node.Child, uid, 0, seen, th, send, dispatcher, fm, locale, profile)
		return KeyedElement{Key: node.Key, Child: child}

	case ThemedElement:
		// Replace the active theme for this subtree.
		// Cache the CachedTheme wrapper so repeated frames reuse it.
		sub, ok := r.themeCache[node.Theme]
		if !ok {
			sub = theme.NewCachedTheme(node.Theme)
			sub.WarmUp()
			r.themeCache[node.Theme] = sub
		}
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, sub, send, dispatcher, fm, locale, profile)
		}
		return ThemedElement{Theme: sub, Children: children}

	case CustomLayoutElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm, locale, profile)
		}
		return CustomLayoutElement{Layout: node.Layout, Children: children}

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
	// Interface-based dispatch for sub-package element types.
	if te, ok := a.(TreeEqualizer); ok {
		return te.TreeEqual(b)
	}
	switch na := a.(type) {
	case KeyedElement:
		nb, ok := b.(KeyedElement)
		return ok && na.Key == nb.Key && treeEqual(na.Child, nb.Child)
	case WidgetBoundsElement:
		nb, ok := b.(WidgetBoundsElement)
		return ok && na.WidgetUID == nb.WidgetUID && treeEqual(na.Child, nb.Child)
	case ThemedElement:
		nb, ok := b.(ThemedElement)
		if !ok || na.Theme != nb.Theme || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case CustomLayoutElement:
		nb, ok := b.(CustomLayoutElement)
		if !ok || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case WidgetElement:
		// Unresolved widget elements — should not appear in resolved trees.
		return false
	default:
		return false
	}
}
