//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"log"
	"runtime"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
	"github.com/timzifer/lux/a11y"
)

// AXBridge implements a11y.A11yBridge for macOS using the NSAccessibility protocol.
type AXBridge struct {
	view       uintptr // LuxMetalView*
	mu         sync.RWMutex
	tree       a11y.AccessTree
	rootNodeID a11y.AccessNodeID
	elements   map[a11y.AccessNodeID]*axElement
	send       func(any) // routes actions to the app loop
	configured bool      // true after first configureViewAccessibility call
}

// axElement tracks a LuxAccessibilityElement ObjC object (subclass of NSAccessibilityElement).
type axElement struct {
	obj    uintptr // LuxAccessibilityElement*
	nodeID a11y.AccessNodeID
}

// NewAXBridge creates a macOS accessibility bridge for the given view.
func NewAXBridge(view uintptr, send func(any)) *AXBridge {
	log.Printf("[AX-BRIDGE] NewAXBridge: view=%#x", view)
	b := &AXBridge{
		view:     view,
		elements: make(map[a11y.AccessNodeID]*axElement),
		send:     send,
	}
	viewAXBridges.Store(view, b)
	log.Printf("[AX-BRIDGE] NewAXBridge: stored in viewAXBridges, view=%#x", view)
	return b
}

// UpdateTree replaces the current access tree and manages element lifecycle.
func (b *AXBridge) UpdateTree(tree a11y.AccessTree) {
	log.Printf("[AX-BRIDGE] UpdateTree: incoming nodes=%d, current elements=%d, view=%#x, configured=%v",
		len(tree.Nodes), len(b.elements), b.view, b.configured)
	b.mu.Lock()

	tree.EnsureIndex()
	oldTree := b.tree
	b.tree = tree

	if len(tree.Nodes) > 0 {
		b.rootNodeID = tree.Nodes[0].ID
		log.Printf("[AX-BRIDGE] UpdateTree: rootNodeID=%d", b.rootNodeID)
	}

	// Log all incoming nodes.
	for i, n := range tree.Nodes {
		log.Printf("[AX-BRIDGE] UpdateTree: node[%d] id=%d role=%v label=%q value=%q bounds={%.0f,%.0f,%.0f,%.0f} parent=%d children=%d",
			i, n.ID, n.Node.Role, n.Node.Label, n.Node.Value,
			n.Bounds.X, n.Bounds.Y, n.Bounds.Width, n.Bounds.Height,
			n.ParentIndex, n.ChildCount)
	}

	// Prune elements for removed nodes.
	pruned := 0
	for id, el := range b.elements {
		if tree.FindByID(id) == nil {
			log.Printf("[AX-BRIDGE] UpdateTree: pruning element id=%d obj=%#x", id, el.obj)
			releaseAXElement(el)
			delete(b.elements, id)
			pruned++
		}
	}
	log.Printf("[AX-BRIDGE] UpdateTree: pruned %d elements", pruned)

	// Create elements for new nodes and update properties for existing ones
	// (skip synthetic root at index 0).
	created, updated := 0, 0
	for i := 1; i < len(tree.Nodes); i++ {
		id := tree.Nodes[i].ID
		if el, exists := b.elements[id]; exists {
			updateAXElementFrame(el, tree.Nodes[i].Bounds, b.view)
			updateAXElementProperties(el, b, &tree.Nodes[i])
			updated++
		} else {
			b.elements[id] = newAXElement(b, id)
			created++
		}
	}
	log.Printf("[AX-BRIDGE] UpdateTree: created=%d updated=%d total_elements=%d", created, updated, len(b.elements))

	// Snapshot elements for use after unlock.
	elementsCopy := make([]*axElement, 0, len(b.elements))
	for _, el := range b.elements {
		elementsCopy = append(elementsCopy, el)
	}

	changed := len(oldTree.Nodes) != len(tree.Nodes) || structureChanged(oldTree, tree)
	log.Printf("[AX-BRIDGE] UpdateTree: structureChanged=%v (old=%d new=%d)", changed, len(oldTree.Nodes), len(tree.Nodes))
	b.mu.Unlock()

	// Rebuild the accessibility children arrays via property setters.
	for _, el := range elementsCopy {
		updateElementAccessibilityChildren(el, b)
	}
	updateViewAccessibilityChildren(b)

	// Configure accessibility on the view AFTER children are populated.
	if !b.configured && len(elementsCopy) > 0 && b.view != 0 {
		log.Printf("[AX-BRIDGE] UpdateTree: FIRST CONFIGURE — calling configureViewAccessibility view=%#x", b.view)
		configureViewAccessibility(b.view)
		b.configured = true
	}

	// Post layout changed notification if structure changed.
	if changed {
		log.Printf("[AX-BRIDGE] UpdateTree: posting AXLayoutChanged on view=%#x", b.view)
		axPostNotification(b.view, axNotificationLayoutChanged)
	}
	log.Printf("[AX-BRIDGE] UpdateTree: DONE")
}

// NotifyFocus raises a focus changed notification for the given node.
func (b *AXBridge) NotifyFocus(nodeID a11y.AccessNodeID) {
	log.Printf("[AX-BRIDGE] NotifyFocus: nodeID=%d", nodeID)
	b.mu.Lock()
	oldFocusID := b.tree.FocusedID
	b.tree.FocusedID = nodeID

	var oldObj, newObj uintptr
	if oldFocusID != 0 {
		if oldEl := b.elementFor(oldFocusID); oldEl != nil {
			oldObj = oldEl.obj
		}
	}
	el := b.ensureElement(nodeID)
	if el != nil {
		newObj = el.obj
	}
	b.mu.Unlock()

	log.Printf("[AX-BRIDGE] NotifyFocus: oldFocusID=%d oldObj=%#x → newFocusID=%d newObj=%#x", oldFocusID, oldObj, nodeID, newObj)

	// Set focused state outside the lock to avoid deadlock with ObjC callbacks.
	if oldObj != 0 {
		msgSendVoid(oldObj, sel("setAccessibilityFocused:"), argBool(false))
	}
	if newObj != 0 {
		msgSendVoid(newObj, sel("setAccessibilityFocused:"), argBool(true))
		log.Printf("[AX-BRIDGE] NotifyFocus: posting AXFocusedUIElementChanged for obj=%#x", newObj)
		axPostNotification(newObj, axNotificationFocusedUIElementChanged)
	}
}

// NotifyLiveRegion posts an announcement notification for live region changes.
func (b *AXBridge) NotifyLiveRegion(nodeID a11y.AccessNodeID, text string) {
	log.Printf("[AX-BRIDGE] NotifyLiveRegion: nodeID=%d text=%q", nodeID, text)
	b.mu.RLock()
	b.mu.RUnlock()

	// Post announcement via NSApp with userInfo dictionary containing the text.
	pool := newAutoreleasePool()
	defer drainPool(pool)

	app := msgSendPtr(getClass("NSApplication"), sel("sharedApplication"))
	axPostAnnouncementNotification(app, text)
}

// Destroy releases all ObjC elements and cleans up.
func (b *AXBridge) Destroy() {
	log.Printf("[AX-BRIDGE] Destroy: view=%#x, elements=%d", b.view, len(b.elements))
	b.mu.Lock()
	defer b.mu.Unlock()

	for id, el := range b.elements {
		log.Printf("[AX-BRIDGE] Destroy: releasing element id=%d obj=%#x", id, el.obj)
		releaseAXElement(el)
	}
	b.elements = nil
	viewAXBridges.Delete(b.view)
	log.Printf("[AX-BRIDGE] Destroy: DONE")
}

// elementFor returns the element for the given node ID.
// Must be called with at least a read lock held.
func (b *AXBridge) elementFor(nodeID a11y.AccessNodeID) *axElement {
	el := b.elements[nodeID]
	if el != nil {
		log.Printf("[AX-BRIDGE] elementFor: nodeID=%d → obj=%#x", nodeID, el.obj)
	} else {
		log.Printf("[AX-BRIDGE] elementFor: nodeID=%d → nil", nodeID)
	}
	return el
}

// ensureElement returns or creates an element under write lock.
func (b *AXBridge) ensureElement(nodeID a11y.AccessNodeID) *axElement {
	if el, ok := b.elements[nodeID]; ok {
		log.Printf("[AX-BRIDGE] ensureElement: nodeID=%d found existing obj=%#x", nodeID, el.obj)
		return el
	}
	log.Printf("[AX-BRIDGE] ensureElement: nodeID=%d creating new element", nodeID)
	el := newAXElement(b, nodeID)
	b.elements[nodeID] = el
	return el
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

// ── NSAccessibilityPostNotification helpers ──

var (
	axPostNotifOnce sync.Once
	fnAXPostNotif   unsafe.Pointer
	cifAXPostNotif  types.CallInterface
)

const (
	axNotificationLayoutChanged              = "AXLayoutChanged"
	axNotificationFocusedUIElementChanged    = "AXFocusedUIElementChanged"
	axNotificationAnnouncementRequested      = "AXAnnouncementRequested"
	axNotificationUIElementDestroyed         = "AXUIElementDestroyed"
)

func ensureAXPostNotif() {
	axPostNotifOnce.Do(func() {
		var err error
		fnAXPostNotif, err = ffi.GetSymbol(rt.appKit, "NSAccessibilityPostNotification")
		if err != nil {
			return
		}
		_ = ffi.PrepareCallInterface(&cifAXPostNotif, types.DefaultCall, types.VoidTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor})
	})
}

// axPostNotification posts an NSAccessibility notification.
func axPostNotification(element uintptr, notifName string) {
	log.Printf("[AX-NOTIF] axPostNotification: element=%#x notif=%s", element, notifName)
	if rt == nil {
		log.Printf("[AX-NOTIF] axPostNotification: SKIP (rt==nil)")
		return
	}
	ensureAXPostNotif()
	if fnAXPostNotif == nil || element == 0 {
		log.Printf("[AX-NOTIF] axPostNotification: SKIP (fn=%v element=%#x)", fnAXPostNotif != nil, element)
		return
	}

	pool := newAutoreleasePool()
	defer drainPool(pool)

	nsNotif := newNSString(notifName)
	_ = ffi.CallFunction(&cifAXPostNotif, fnAXPostNotif, nil,
		[]unsafe.Pointer{unsafe.Pointer(&element), unsafe.Pointer(&nsNotif)})
	runtime.KeepAlive(nsNotif)
}

// axPostAnnouncementNotification posts an announcement with text via userInfo dict.
func axPostAnnouncementNotification(element uintptr, text string) {
	ensureAXPostNotif()
	if fnAXPostNotif == nil || element == 0 {
		return
	}

	pool := newAutoreleasePool()
	defer drainPool(pool)

	// Create userInfo: @{NSAccessibilityAnnouncementKey: text}
	nsKey := newNSString("AXAnnouncementKey")
	nsValue := newNSString(text)

	nsDictClass := getClass("NSDictionary")
	dict := msgSendPtr(nsDictClass, sel("dictionaryWithObject:forKey:"),
		argPtr(nsValue), argPtr(nsKey))

	nsNotif := newNSString(axNotificationAnnouncementRequested)

	// NSAccessibilityPostNotificationWithUserInfo is not always available;
	// use the two-arg version and pass dict as the notification name's userInfo
	// via a helper: post the notification on the element.
	// Actually, we need the 3-arg variant. Let's load it.
	fnPostWithUserInfo, err := ffi.GetSymbol(rt.appKit, "NSAccessibilityPostNotificationWithUserInfo")
	if err == nil {
		var cifPostUI types.CallInterface
		_ = ffi.PrepareCallInterface(&cifPostUI, types.DefaultCall, types.VoidTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor, types.PointerTypeDescriptor})
		_ = ffi.CallFunction(&cifPostUI, fnPostWithUserInfo, nil,
			[]unsafe.Pointer{unsafe.Pointer(&element), unsafe.Pointer(&nsNotif), unsafe.Pointer(&dict)})
	} else {
		// Fallback: post without userInfo.
		_ = ffi.CallFunction(&cifAXPostNotif, fnAXPostNotif, nil,
			[]unsafe.Pointer{unsafe.Pointer(&element), unsafe.Pointer(&nsNotif)})
	}
	runtime.KeepAlive(nsNotif)
	runtime.KeepAlive(dict)
}

// Verify AXBridge implements a11y.A11yBridge.
var _ a11y.A11yBridge = (*AXBridge)(nil)
