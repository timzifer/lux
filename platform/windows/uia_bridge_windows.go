//go:build windows && !nogui

package windows

import (
	"sync"
	"unsafe"

	"github.com/timzifer/lux/a11y"
	"github.com/zzl/go-win32api/v2/win32"
)

// UIABridge coordinates the Windows UIA accessibility bridge.
// It manages the root provider, element providers, and access tree state.
type UIABridge struct {
	hwnd       uintptr
	root       *rootProvider
	mu         sync.RWMutex
	tree       a11y.AccessTree
	rootNodeID a11y.AccessNodeID // ID of synthetic root node (tree.Nodes[0])
	providers  map[a11y.AccessNodeID]*elementProvider
	send       func(any) // routes actions to the app loop
}

// NewUIABridge creates a UIA bridge for the given window.
func NewUIABridge(hwnd uintptr, send func(any)) *UIABridge {
	b := &UIABridge{
		hwnd:      hwnd,
		providers: make(map[a11y.AccessNodeID]*elementProvider),
		send:      send,
	}
	b.root = newRootProvider(b)
	return b
}

// RootProvider returns the COM pointer for the root element provider.
// This is returned from WM_GETOBJECT.
func (b *UIABridge) RootProvider() unsafe.Pointer {
	return b.root.simplePtr()
}

// UpdateTree replaces the current access tree and notifies UIA of structural changes.
func (b *UIABridge) UpdateTree(tree a11y.AccessTree) {
	b.mu.Lock()

	tree.EnsureIndex()
	oldTree := b.tree
	b.tree = tree

	// Remember the root node ID (synthetic root at index 0).
	if len(tree.Nodes) > 0 {
		b.rootNodeID = tree.Nodes[0].ID
	}

	// Prune providers for nodes no longer in the tree.
	// Skip the root node (index 0) — it maps to the root UIA provider, not an element provider.
	for id := range b.providers {
		if tree.FindByID(id) == nil {
			delete(b.providers, id)
		}
	}

	// Pre-create providers for all nodes (except the root node at index 0)
	// so that navigation callbacks only need read access.
	for i := 1; i < len(tree.Nodes); i++ {
		id := tree.Nodes[i].ID
		if _, exists := b.providers[id]; !exists {
			b.providers[id] = newElementProvider(b, id)
		}
	}

	b.mu.Unlock()

	// Raise structure changed event if tree changed.
	if len(oldTree.Nodes) != len(tree.Nodes) || structureChanged(oldTree, tree) {
		uiaRaiseStructureChangedEvent(
			b.root.simplePtr(),
			0, // StructureChangeType_ChildrenInvalidated
			nil, 0,
		)
	}
}

// NotifyFocus raises a UIA focus event for the given node.
func (b *UIABridge) NotifyFocus(nodeID a11y.AccessNodeID) {
	b.mu.Lock()
	b.tree.FocusedID = nodeID
	ep := b.ensureProvider(nodeID)
	b.mu.Unlock()

	if ep != nil {
		uiaRaiseAutomationEvent(
			unsafe.Pointer(&ep.vtblSimple),
			win32.UIA_AutomationFocusChangedEventId,
		)
	}
}

// NotifyLiveRegion raises a live-region changed event.
func (b *UIABridge) NotifyLiveRegion(nodeID a11y.AccessNodeID, text string) {
	b.mu.RLock()
	ep := b.providerFor(nodeID)
	b.mu.RUnlock()

	if ep != nil {
		uiaRaiseAutomationEvent(
			unsafe.Pointer(&ep.vtblSimple),
			win32.UIA_LiveRegionChangedEventId,
		)
	}
}

// Destroy releases all UIA resources.
func (b *UIABridge) Destroy() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.providers = nil
	b.root = nil
}

// providerFor returns the pre-created provider for the given node ID.
// Must be called with at least a read lock held. Returns nil if not found
// (e.g. for the root node which maps to the root UIA provider).
func (b *UIABridge) providerFor(nodeID a11y.AccessNodeID) *elementProvider {
	return b.providers[nodeID]
}

// ensureProvider returns or creates a provider under write lock.
func (b *UIABridge) ensureProvider(nodeID a11y.AccessNodeID) *elementProvider {
	if ep, ok := b.providers[nodeID]; ok {
		return ep
	}
	ep := newElementProvider(b, nodeID)
	b.providers[nodeID] = ep
	return ep
}

// structureChanged compares two trees for structural equivalence.
func structureChanged(a, b a11y.AccessTree) bool {
	if len(a.Nodes) != len(b.Nodes) {
		return true
	}
	for i := range a.Nodes {
		an := &a.Nodes[i]
		bn := &b.Nodes[i]
		if an.ID != bn.ID || an.ParentIndex != bn.ParentIndex ||
			an.FirstChild != bn.FirstChild || an.NextSibling != bn.NextSibling ||
			an.ChildCount != bn.ChildCount {
			return true
		}
	}
	return false
}

// Verify UIABridge implements a11y.A11yBridge.
var _ a11y.A11yBridge = (*UIABridge)(nil)
