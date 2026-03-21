//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
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
	addMethod("isAccessibilityElement", ffi.NewCallback(axViewNO), "B@:")
	addMethod("accessibilityIsIgnored", ffi.NewCallback(axViewYES), "B@:")
}

// configureViewAccessibility sets properties on the view via setters.
func configureViewAccessibility(view uintptr) {
	// The LuxMetalView is a container for virtual accessibility children, not a
	// user-facing control of its own. Let AppKit expose the window/application
	// hierarchy and tunnel through this ignored container to its children.
	axDebugf("configure view: view=%#x accessibilityElement=false", view)
	msgSendVoid(view, sel("setAccessibilityElement:"), argBool(false))
}

// updateViewAccessibilityChildren sets the view's children array.
func updateViewAccessibilityChildren(bridge *AXBridge) {
	if rt == nil || bridge.view == 0 {
		return
	}
	if len(bridge.tree.Nodes) == 0 {
		return
	}
	root := &bridge.tree.Nodes[0]
	children := bridge.tree.Children(root)
	axDebugf("set view children: view=%#x count=%d", bridge.view, len(children))
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
			objs = append(objs, el.obj)
		}
	}
	arr := newNSArray(objs)
	msgSendVoid(bridge.view, sel("setAccessibilityChildren:"), argPtr(arr))
}

// updateElementAccessibilityChildren sets children on NSAccessibilityElement instances.
func updateElementAccessibilityChildren(el *axElement, bridge *AXBridge) {
	if rt == nil || el.obj == 0 {
		return
	}
	node := bridge.tree.FindByID(el.nodeID)
	if node == nil {
		return
	}
	axDebugf("set element children: node=%d obj=%#x count=%d", el.nodeID, el.obj, node.ChildCount)
	objs := make([]uintptr, 0, node.ChildCount)
	if node.FirstChild >= 0 {
		children := bridge.tree.Children(node)
		for _, child := range children {
			if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
				objs = append(objs, childEl.obj)
			}
		}
	}
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

func axViewYES(self, _cmd uintptr) uintptr { return 1 }
func axViewNO(self, _cmd uintptr) uintptr  { return 0 }

func axViewChildren(self, _cmd uintptr) uintptr {
	bridge := bridgeForView(self)
	if bridge == nil {
		axDebugf("view children callback: view=%#x bridge=nil", self)
		return newNSArray(nil)
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if len(bridge.tree.Nodes) == 0 {
		axDebugf("view children callback: view=%#x nodes=0", self)
		return newNSArray(nil)
	}
	root := &bridge.tree.Nodes[0]
	children := bridge.tree.Children(root)
	axDebugf("view children callback: view=%#x count=%d", self, len(children))
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
			objs = append(objs, el.obj)
		}
	}
	return newNSArray(objs)
}

func axViewHitTest(self, _cmd uintptr, pointX, pointY float64) uintptr {
	bridge := bridgeForView(self)
	if bridge == nil {
		axDebugf("hit test: view=%#x bridge=nil point=(%.1f,%.1f)", self, pointX, pointY)
		return self
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	winOrigin := axWindowOrigin(self)
	winHeight := axWindowHeight(self)
	localX := pointX - winOrigin.X
	localY := winHeight - (pointY - winOrigin.Y)

	var deepest *axElement
	for i := 1; i < len(bridge.tree.Nodes); i++ {
		node := &bridge.tree.Nodes[i]
		b := node.Bounds
		if localX >= b.X && localX < b.X+b.Width &&
			localY >= b.Y && localY < b.Y+b.Height {
			if el := bridge.elementFor(node.ID); el != nil && el.obj != 0 {
				deepest = el
			}
		}
	}
	if deepest != nil {
		axDebugf("hit test: view=%#x point=(%.1f,%.1f) local=(%.1f,%.1f) -> node=%d obj=%#x", self, pointX, pointY, localX, localY, deepest.nodeID, deepest.obj)
		return deepest.obj
	}
	axDebugf("hit test: view=%#x point=(%.1f,%.1f) local=(%.1f,%.1f) -> self", self, pointX, pointY, localX, localY)
	return self
}

func axViewFocusedElement(self, _cmd uintptr) uintptr {
	bridge := bridgeForView(self)
	if bridge == nil {
		return self
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if bridge.tree.FocusedID == 0 {
		axDebugf("focused element callback: view=%#x focused=0 -> self", self)
		return self
	}
	el := bridge.elementFor(bridge.tree.FocusedID)
	if el != nil && el.obj != 0 {
		axDebugf("focused element callback: view=%#x focusedNode=%d obj=%#x", self, bridge.tree.FocusedID, el.obj)
		return el.obj
	}
	axDebugf("focused element callback: view=%#x focusedNode=%d missing -> self", self, bridge.tree.FocusedID)
	return self
}
