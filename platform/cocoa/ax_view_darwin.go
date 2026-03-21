//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"fmt"
	"log"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
	"github.com/timzifer/lux/a11y"
)

// viewAXBridges maps NSView pointers to their *AXBridge.
var viewAXBridges sync.Map

func init() {
	registerLuxViewClassHooks = append(registerLuxViewClassHooks, axViewHook)
}

// axViewHook adds accessibility method overrides to LuxMetalView.
func axViewHook(cls uintptr, fnAddMethod unsafe.Pointer, cifAddMethod *types.CallInterface) {
	addMethod := func(selName string, imp uintptr, typeEncoding string) {
		s := sel(selName)
		enc := append([]byte(typeEncoding), 0)
		encPtr := unsafe.Pointer(&enc[0])
		var result uint8
		_ = ffi.CallFunction(cifAddMethod, fnAddMethod, unsafe.Pointer(&result),
			[]unsafe.Pointer{unsafe.Pointer(&cls), unsafe.Pointer(&s),
				unsafe.Pointer(&imp), unsafe.Pointer(&encPtr)})
	}

	// Modern protocol overrides.
	addMethod("accessibilityChildren", ffi.NewCallback(axViewChildren), "@@:")
	addMethod("accessibilityHitTest:", ffi.NewCallback(axViewHitTest), "@@:{CGPoint=dd}")
	addMethod("accessibilityFocusedUIElement", ffi.NewCallback(axViewFocusedElement), "@@:")
	addMethod("accessibilityRole", ffi.NewCallback(axViewRole), "@@:")
	addMethod("isAccessibilityElement", ffi.NewCallback(axViewYES), "B@:")
	addMethod("accessibilityIsIgnored", ffi.NewCallback(axViewNO), "B@:")
}

// configureViewAccessibility sets properties on the view via setters.
func configureViewAccessibility(view uintptr) {
	log.Printf("[AX-VIEW] configureViewAccessibility: view=%#x", view)
	msgSendVoid(view, sel("setAccessibilityElement:"), argBool(true))
	nsRole := newNSString("AXGroup")
	msgSendVoid(view, sel("setAccessibilityRole:"), argPtr(nsRole))
	nsLabel := newNSString("application")
	msgSendVoid(view, sel("setAccessibilityLabel:"), argPtr(nsLabel))
	log.Printf("[AX-VIEW] configureViewAccessibility: done (role=AXGroup, label=application)")
}

// updateViewAccessibilityChildren sets the view's children array.
func updateViewAccessibilityChildren(bridge *AXBridge) {
	log.Printf("[AX-VIEW] updateViewAccessibilityChildren: view=%#x, rt=%v, nodes=%d", bridge.view, rt != nil, len(bridge.tree.Nodes))
	if rt == nil || bridge.view == 0 {
		log.Printf("[AX-VIEW] updateViewAccessibilityChildren: EARLY RETURN (rt==nil: %v, view==0: %v)", rt == nil, bridge.view == 0)
		return
	}
	if len(bridge.tree.Nodes) == 0 {
		log.Printf("[AX-VIEW] updateViewAccessibilityChildren: EARLY RETURN (no nodes)")
		return
	}
	root := &bridge.tree.Nodes[0]
	children := bridge.tree.Children(root)
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
			objs = append(objs, el.obj)
		}
	}
	log.Printf("[AX-VIEW] updateViewAccessibilityChildren: setting %d children (of %d tree children) on view=%#x", len(objs), len(children), bridge.view)
	arr := newNSArray(objs)
	msgSendVoid(bridge.view, sel("setAccessibilityChildren:"), argPtr(arr))
}

// updateElementAccessibilityChildren sets children on NSAccessibilityElement instances.
func updateElementAccessibilityChildren(el *axElement, bridge *AXBridge) {
	if rt == nil || el.obj == 0 {
		return
	}
	node := bridge.tree.FindByID(el.nodeID)
	if node == nil || node.FirstChild < 0 {
		return
	}
	children := bridge.tree.Children(node)
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
			objs = append(objs, childEl.obj)
		}
	}
	if len(objs) > 0 {
		log.Printf("[AX-VIEW] updateElementAccessibilityChildren: nodeID=%d obj=%#x → %d children", el.nodeID, el.obj, len(objs))
		arr := newNSArray(objs)
		msgSendVoid(el.obj, sel("setAccessibilityChildren:"), argPtr(arr))
	}
}

func bridgeForView(view uintptr) *AXBridge {
	v, ok := viewAXBridges.Load(view)
	if !ok {
		return nil
	}
	return v.(*AXBridge)
}

// ── Callbacks ──

func axViewYES(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] isAccessibilityElement: self=%#x → YES", self)
	return 1
}
func axViewNO(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] accessibilityIsIgnored: self=%#x → NO", self)
	return 0
}

func axViewRole(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] accessibilityRole: self=%#x → AXGroup", self)
	return newNSString("AXGroup")
}

func axViewChildren(self, _cmd uintptr) uintptr {
	bridge := bridgeForView(self)
	if bridge == nil {
		log.Printf("[AX-CB] accessibilityChildren: self=%#x → NO BRIDGE, returning empty", self)
		return newNSArray(nil)
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if len(bridge.tree.Nodes) == 0 {
		log.Printf("[AX-CB] accessibilityChildren: self=%#x → no nodes, returning empty", self)
		return newNSArray(nil)
	}
	root := &bridge.tree.Nodes[0]
	children := bridge.tree.Children(root)
	objs := make([]uintptr, 0, len(children))
	childInfo := make([]string, 0, len(children))
	for _, child := range children {
		if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
			objs = append(objs, el.obj)
			childInfo = append(childInfo, fmt.Sprintf("id=%d/obj=%#x/role=%v/label=%q", child.ID, el.obj, child.Node.Role, child.Node.Label))
		}
	}
	log.Printf("[AX-CB] accessibilityChildren: self=%#x → %d children: %v", self, len(objs), childInfo)
	return newNSArray(objs)
}

func axViewHitTest(self, _cmd uintptr, pointX, pointY float64) uintptr {
	log.Printf("[AX-CB] accessibilityHitTest: self=%#x point=(%.1f, %.1f)", self, pointX, pointY)
	bridge := bridgeForView(self)
	if bridge == nil {
		log.Printf("[AX-CB] accessibilityHitTest: NO BRIDGE → returning self=%#x", self)
		return self
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	winOrigin := axWindowOrigin(self)
	winHeight := axWindowHeight(self)
	localX := pointX - winOrigin.X
	localY := winHeight - (pointY - winOrigin.Y)
	log.Printf("[AX-CB] accessibilityHitTest: winOrigin=(%.1f,%.1f) winHeight=%.1f → local=(%.1f, %.1f)", winOrigin.X, winOrigin.Y, winHeight, localX, localY)

	var deepest *axElement
	var deepestNode *a11y.AccessTreeNode
	for i := 1; i < len(bridge.tree.Nodes); i++ {
		node := &bridge.tree.Nodes[i]
		b := node.Bounds
		if localX >= b.X && localX < b.X+b.Width &&
			localY >= b.Y && localY < b.Y+b.Height {
			if el := bridge.elementFor(node.ID); el != nil && el.obj != 0 {
				deepest = el
				deepestNode = node
			}
		}
	}
	if deepest != nil {
		log.Printf("[AX-CB] accessibilityHitTest: HIT nodeID=%d obj=%#x role=%v label=%q bounds={%.0f,%.0f,%.0f,%.0f}",
			deepestNode.ID, deepest.obj, deepestNode.Node.Role, deepestNode.Node.Label,
			deepestNode.Bounds.X, deepestNode.Bounds.Y, deepestNode.Bounds.Width, deepestNode.Bounds.Height)
		return deepest.obj
	}
	log.Printf("[AX-CB] accessibilityHitTest: NO HIT (tested %d nodes) → returning self=%#x", len(bridge.tree.Nodes)-1, self)
	return self
}

func axViewFocusedElement(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] accessibilityFocusedUIElement: self=%#x", self)
	bridge := bridgeForView(self)
	if bridge == nil {
		log.Printf("[AX-CB] accessibilityFocusedUIElement: NO BRIDGE → self")
		return self
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if bridge.tree.FocusedID == 0 {
		log.Printf("[AX-CB] accessibilityFocusedUIElement: focusedID=0 → self")
		return self
	}
	el := bridge.elementFor(bridge.tree.FocusedID)
	if el != nil && el.obj != 0 {
		log.Printf("[AX-CB] accessibilityFocusedUIElement: focusedID=%d → obj=%#x", bridge.tree.FocusedID, el.obj)
		return el.obj
	}
	log.Printf("[AX-CB] accessibilityFocusedUIElement: focusedID=%d but no element → self", bridge.tree.FocusedID)
	return self
}
