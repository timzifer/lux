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
	states      map[UID]WidgetState
	widgets     map[UID]Widget  // previous widget instance per UID (for Equatable)
	resolvedSub map[UID]Element // previous resolved subtree per UID (for Equatable skip)
	prevTree    Element
	themeCache  map[theme.Theme]*theme.CachedTheme // reuse CachedTheme across frames
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
func (r *Reconciler) Reconcile(newTree Element, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager, locale string) (Element, bool) {
	seen := make(map[UID]bool)
	resolved := r.resolveTree(newTree, 0, 0, seen, th, send, dispatcher, fm, locale)

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
	anyDirty := false
	for _, state := range r.states {
		if dt, ok := state.(DirtyTracker); ok {
			if dt.IsDirty() {
				anyDirty = true
				dt.ClearDirty()
			}
		}
	}
	return anyDirty
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
func (r *Reconciler) resolveTree(el Element, parentUID UID, index int, seen map[UID]bool, th theme.Theme, send func(any), dispatcher *EventDispatcher, fm *FocusManager, locale string) Element {
	// Interface-based dispatch for sub-package element types.
	if cr, ok := el.(ChildResolver); ok {
		return cr.ResolveChildren(func(child Element, childIndex int) Element {
			return r.resolveTree(child, parentUID, childIndex, seen, th, send, dispatcher, fm, locale)
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
		ctx := RenderCtx{UID: uid, Theme: th, Send: send, Locale: locale}
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
		resolved := r.resolveTree(child, uid, 0, seen, th, send, dispatcher, fm, locale)
		r.resolvedSub[uid] = resolved

		// Wrap in WidgetBoundsElement so layout can track screen bounds.
		return WidgetBoundsElement{WidgetUID: uid, Child: resolved}

	case KeyedElement:
		uid := MakeUID(parentUID, node.Key, index)
		child := r.resolveTree(node.Child, uid, 0, seen, th, send, dispatcher, fm, locale)
		return KeyedElement{Key: node.Key, Child: child}

	case BoxElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm, locale)
		}
		return BoxElement{Axis: node.Axis, Children: children}

	case StackElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm, locale)
		}
		return StackElement{Children: children}

	case ScrollViewElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm, locale)
		return ScrollViewElement{Child: child, MaxHeight: node.MaxHeight, State: node.State}

	case PaddingElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm, locale)
		return PaddingElement{Insets: node.Insets, Child: child}

	case SizedBoxElement:
		if node.Child != nil {
			child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm, locale)
			return SizedBoxElement{Width: node.Width, Height: node.Height, Child: child}
		}
		return el

	case ExpandedElement:
		child := r.resolveTree(node.Child, parentUID, 0, seen, th, send, dispatcher, fm, locale)
		return ExpandedElement{Child: child, Grow: node.Grow}

	case FlexElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm, locale)
		}
		return FlexElement{Direction: node.Direction, Justify: node.Justify, Align: node.Align, Gap: node.Gap, Children: children}

	case GridElement:
		children := make([]Element, len(node.Children))
		for i, c := range node.Children {
			children[i] = r.resolveTree(c, parentUID, i, seen, th, send, dispatcher, fm, locale)
		}
		return GridElement{Columns: node.Columns, RowGap: node.RowGap, ColGap: node.ColGap, Children: children}

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
			children[i] = r.resolveTree(c, parentUID, i, seen, sub, send, dispatcher, fm, locale)
		}
		return ThemedElement{Theme: sub, Children: children}

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
	case EmptyElement:
		_, ok := b.(EmptyElement)
		return ok
	case TextElement:
		nb, ok := b.(TextElement)
		return ok && na.Content == nb.Content && na.Style == nb.Style
	case ButtonElement:
		nb, ok := b.(ButtonElement)
		return ok && treeEqual(na.Content, nb.Content)
	case KeyedElement:
		nb, ok := b.(KeyedElement)
		return ok && na.Key == nb.Key && treeEqual(na.Child, nb.Child)
	case WidgetBoundsElement:
		nb, ok := b.(WidgetBoundsElement)
		return ok && na.WidgetUID == nb.WidgetUID && treeEqual(na.Child, nb.Child)
	case BoxElement:
		nb, ok := b.(BoxElement)
		if !ok || na.Axis != nb.Axis || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case StackElement:
		nb, ok := b.(StackElement)
		if !ok || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case ScrollViewElement:
		nb, ok := b.(ScrollViewElement)
		return ok && na.MaxHeight == nb.MaxHeight && treeEqual(na.Child, nb.Child)
	case DividerElement:
		_, ok := b.(DividerElement)
		return ok
	case SpacerElement:
		nb, ok := b.(SpacerElement)
		return ok && na.Size == nb.Size
	case IconElement:
		nb, ok := b.(IconElement)
		return ok && na.Name == nb.Name && na.Size == nb.Size
	case CheckboxElement:
		nb, ok := b.(CheckboxElement)
		return ok && na.Label == nb.Label && na.Checked == nb.Checked
	case RadioElement:
		nb, ok := b.(RadioElement)
		return ok && na.Label == nb.Label && na.Selected == nb.Selected
	case ToggleElement:
		nb, ok := b.(ToggleElement)
		return ok && na.On == nb.On
	case SliderElement:
		nb, ok := b.(SliderElement)
		return ok && na.Value == nb.Value
	case ProgressBarElement:
		nb, ok := b.(ProgressBarElement)
		return ok && na.Value == nb.Value && na.Indeterminate == nb.Indeterminate
	case TextFieldElement:
		nb, ok := b.(TextFieldElement)
		return ok && na.Value == nb.Value && na.Placeholder == nb.Placeholder
	case SelectElement:
		nb, ok := b.(SelectElement)
		if !ok || na.Value != nb.Value || len(na.Options) != len(nb.Options) {
			return false
		}
		for i := range na.Options {
			if na.Options[i] != nb.Options[i] {
				return false
			}
		}
		return true
	case PaddingElement:
		nb, ok := b.(PaddingElement)
		return ok && na.Insets == nb.Insets && treeEqual(na.Child, nb.Child)
	case SizedBoxElement:
		nb, ok := b.(SizedBoxElement)
		return ok && na.Width == nb.Width && na.Height == nb.Height && treeEqual(na.Child, nb.Child)
	case ExpandedElement:
		nb, ok := b.(ExpandedElement)
		return ok && na.Grow == nb.Grow && treeEqual(na.Child, nb.Child)
	case FlexElement:
		nb, ok := b.(FlexElement)
		if !ok || na.Direction != nb.Direction || na.Justify != nb.Justify || na.Align != nb.Align || na.Gap != nb.Gap || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case GridElement:
		nb, ok := b.(GridElement)
		if !ok || na.Columns != nb.Columns || na.RowGap != nb.RowGap || na.ColGap != nb.ColGap || len(na.Children) != len(nb.Children) {
			return false
		}
		for i := range na.Children {
			if !treeEqual(na.Children[i], nb.Children[i]) {
				return false
			}
		}
		return true
	case VirtualListElement:
		nb, ok := b.(VirtualListElement)
		return ok && na.ItemCount == nb.ItemCount && na.ItemHeight == nb.ItemHeight && na.MaxHeight == nb.MaxHeight
	case TreeElement:
		// Tree content is dynamic — always re-render.
		_, ok := b.(TreeElement)
		return ok && false
	case RichTextElement:
		nb, ok := b.(RichTextElement)
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
	case WidgetElement:
		// Unresolved widget elements — should not appear in resolved trees.
		return false
	default:
		return false
	}
}
