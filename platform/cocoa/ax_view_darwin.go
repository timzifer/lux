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
		log.Printf("[AX-VIEW] class_addMethod %q → result=%d", selName, result)
	}

	// Override accessibility methods on LuxMetalView so macOS treats it as
	// a real accessibility container. Without isAccessibilityElement returning YES,
	// macOS skips the view entirely (NSView defaults to NO). Without accessibilityRole
	// returning AXGroup, macOS doesn't know the view is a container.
	// Modern NSAccessibility protocol overrides.
	addMethod("isAccessibilityElement", ffi.NewCallback(axViewIsElement), "B@:")
	addMethod("accessibilityRole", ffi.NewCallback(axViewRole), "@@:")
	addMethod("accessibilityParent", ffi.NewCallback(axViewParent), "@@:")
	addMethod("accessibilityWindow", ffi.NewCallback(axViewWindow), "@@:")
	addMethod("accessibilityTopLevelUIElement", ffi.NewCallback(axViewWindow), "@@:")
	addMethod("accessibilityChildren", ffi.NewCallback(axViewChildren), "@@:")
	addMethod("accessibilityHitTest:", ffi.NewCallback(axViewHitTest), "@@:{CGPoint=dd}")
	addMethod("accessibilityFocusedUIElement", ffi.NewCallback(axViewFocusedElement), "@@:")

}

// configureViewAccessibility sets instance-level accessibility properties on
// the view. These property setters prime the backing ivars so the AX server
// can discover the view. The method overrides (registered by axViewHook)
// handle dynamic queries.
func configureViewAccessibility(view uintptr) {
	msgSendVoid(view, sel("setAccessibilityElement:"), argBool(true))
	msgSendVoid(view, sel("setAccessibilityRole:"), argPtr(newNSString("AXGroup")))

	win := msgSendPtr(view, sel("window"))
	if win != 0 {
		msgSendVoid(view, sel("setAccessibilityParent:"), argPtr(win))
		msgSendVoid(view, sel("setAccessibilityWindow:"), argPtr(win))
		msgSendVoid(view, sel("setAccessibilityTopLevelUIElement:"), argPtr(win))
	}

	log.Printf("[AX-VIEW] configureViewAccessibility: set properties on view=%#x window=%#x", view, win)
}

// updateViewAccessibilityChildren sets the children property on the view.
func updateViewAccessibilityChildren(bridge *AXBridge) {
	if bridge.view == 0 {
		return
	}
	bridge.mu.RLock()
	var objs []uintptr
	if len(bridge.tree.Nodes) > 0 {
		root := &bridge.tree.Nodes[0]
		children := bridge.tree.Children(root)
		objs = make([]uintptr, 0, len(children))
		for _, child := range children {
			if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
				objs = append(objs, el.obj)
			}
		}
	}
	bridge.mu.RUnlock()

	arr := newNSArray(objs)
	msgSendVoid(bridge.view, sel("setAccessibilityChildren:"), argPtr(arr))
}

// updateElementAccessibilityChildren sets the children property on an element
// so the AX server can discover child elements via ivar access. The method
// override (luxAXChildren) handles dynamic queries.
func updateElementAccessibilityChildren(el *axElement, bridge *AXBridge) {
	if el.obj == 0 {
		return
	}
	bridge.mu.RLock()
	node := bridge.tree.FindByID(el.nodeID)
	var objs []uintptr
	if node != nil && node.FirstChild >= 0 {
		children := bridge.tree.Children(node)
		objs = make([]uintptr, 0, len(children))
		for _, child := range children {
			if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
				objs = append(objs, childEl.obj)
			}
		}
	}
	bridge.mu.RUnlock()

	arr := newNSArray(objs)
	msgSendVoid(el.obj, sel("setAccessibilityChildren:"), argPtr(arr))
}

func bridgeForView(view uintptr) *AXBridge {
	v, ok := viewAXBridges.Load(view)
	if !ok {
		return nil
	}
	return v.(*AXBridge)
}

// ── Callbacks ──

func axViewIsElement(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] isAccessibilityElement: self=%#x → YES", self)
	return 1 // YES — required so macOS doesn't skip this view
}

func axViewRole(self, _cmd uintptr) uintptr {
	log.Printf("[AX-CB] accessibilityRole: self=%#x → AXGroup", self)
	return newNSString("AXGroup")
}

func axViewParent(self, _cmd uintptr) uintptr {
	win := msgSendPtr(self, sel("window"))
	log.Printf("[AX-CB] accessibilityParent(view): self=%#x → window=%#x", self, win)
	return win
}

func axViewWindow(self, _cmd uintptr) uintptr {
	win := msgSendPtr(self, sel("window"))
	log.Printf("[AX-CB] accessibilityWindow(view): self=%#x → window=%#x", self, win)
	return win
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

