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

// axViewHook adds NSAccessibility method overrides to LuxMetalView.
// We need BOTH method overrides AND property setters because:
// - Property setters populate NSView's internal storage (needed for some code paths)
// - Method overrides provide dynamic dispatch (needed for accessibilityHitTest,
//   accessibilityChildren, etc. where the default NSView getter computes from subviews)
func axViewHook(cls uintptr, fnAddMethod unsafe.Pointer, cifAddMethod *types.CallInterface) {
	addMethod := func(selName string, imp uintptr, typeEncoding string) {
		s := sel(selName)
		enc := append([]byte(typeEncoding), 0)
		encPtr := unsafe.Pointer(&enc[0])
		var result uint8
		_ = ffi.CallFunction(cifAddMethod, fnAddMethod, unsafe.Pointer(&result),
			[]unsafe.Pointer{unsafe.Pointer(&cls), unsafe.Pointer(&s),
				unsafe.Pointer(&imp), unsafe.Pointer(&encPtr)})
		if result == 0 {
			println("[a11y-hook] class_addMethod FAILED for", selName)
		} else {
			println("[a11y-hook] class_addMethod OK for", selName)
		}
	}

	// ── Modern protocol overrides ──
	addMethod("accessibilityChildren", ffi.NewCallback(axViewChildren), "@@:")
	addMethod("accessibilityHitTest:", ffi.NewCallback(axViewHitTest), "@@:{CGPoint=dd}")
	addMethod("accessibilityFocusedUIElement", ffi.NewCallback(axViewFocusedElement), "@@:")
	addMethod("accessibilityRole", ffi.NewCallback(axViewRole), "@@:")
	addMethod("isAccessibilityElement", ffi.NewCallback(axViewIsElement), "B@:")
	addMethod("accessibilityIsIgnored", ffi.NewCallback(axViewIsNotIgnored), "B@:")

	// ── Legacy protocol: override both attribute names and values.
	// accessibilityAttributeValue: calls super for all attributes we don't handle,
	// ensuring NSView's defaults for AXParent, AXPosition, AXSize, etc. still work.
	addMethod("accessibilityAttributeNames", ffi.NewCallback(axViewAttrNames), "@@:")
	addMethod("accessibilityAttributeValue:", ffi.NewCallback(axViewAttrValue), "@@:@")
}

// configureViewAccessibility sets accessibility properties on a view instance
// using property setters (belt-and-suspenders with method overrides).
func configureViewAccessibility(view uintptr) {
	msgSendVoid(view, sel("setAccessibilityElement:"), argBool(true))

	nsRole := newNSString("AXGroup")
	msgSendVoid(view, sel("setAccessibilityRole:"), argPtr(nsRole))

	nsLabel := newNSString("application")
	msgSendVoid(view, sel("setAccessibilityLabel:"), argPtr(nsLabel))
}

// updateViewAccessibilityChildren sets children via property setter AND logs
// the result for diagnostics.
func updateViewAccessibilityChildren(bridge *AXBridge) {
	if rt == nil || bridge.view == 0 {
		return
	}

	if len(bridge.tree.Nodes) == 0 {
		emptyArr := newNSArray(nil)
		msgSendVoid(bridge.view, sel("setAccessibilityChildren:"), argPtr(emptyArr))
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
	arr := newNSArray(objs)
	msgSendVoid(bridge.view, sel("setAccessibilityChildren:"), argPtr(arr))
}

// updateElementAccessibilityChildren sets the children property on an element.
func updateElementAccessibilityChildren(el *axElement, bridge *AXBridge) {
	if rt == nil || el.obj == 0 {
		return
	}
	node := bridge.tree.FindByID(el.nodeID)
	if node == nil || node.FirstChild < 0 {
		emptyArr := newNSArray(nil)
		msgSendVoid(el.obj, sel("setAccessibilityChildren:"), argPtr(emptyArr))
		return
	}

	children := bridge.tree.Children(node)
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
			objs = append(objs, childEl.obj)
		}
	}
	arr := newNSArray(objs)
	msgSendVoid(el.obj, sel("setAccessibilityChildren:"), argPtr(arr))
}

// axViewAttrValue handles AXChildren and AXFocusedUIElement, delegates everything else to super.
func axViewAttrValue(self, _cmd, attrName uintptr) uintptr {
	attr := goString(attrName)
	println("[a11y-val]", attr)

	window := msgSendPtr(self, sel("window"))

	switch attr {
	case "AXRole":
		return newNSString("AXGroup")
	case "AXRoleDescription":
		return newNSString("group")
	case "AXChildren":
		children := axViewChildren(self, _cmd)
		count := msgSendUInt64(children, sel("count"))
		println("[a11y-val] AXChildren returning", count, "items")
		return children
	case "AXParent":
		println("[a11y-val] AXParent -> window", window)
		return window
	case "AXWindow":
		return window
	case "AXTopLevelUIElement":
		return window
	case "AXPosition":
		// Content area origin in screen coords.
		origin := axWindowOrigin(self)
		screenH := axScreenHeight()
		winH := axWindowHeight(self)
		// Convert to top-left screen coords for AX (bottom-left origin).
		return axNSValueWithPoint(origin.X, screenH-origin.Y-winH)
	case "AXSize":
		winH := axWindowHeight(self)
		frame := msgSendRect(window, sel("frame"))
		contentRect := msgSendRect(window, sel("contentRectForFrameRect:"), argRect(frame))
		return axNSValueWithSize(contentRect.Size.Width, winH)
	case "AXFocused":
		return msgSendPtr(getClass("NSNumber"), sel("numberWithBool:"), argBool(false))
	case "AXFocusedUIElement":
		return axViewFocusedElement(self, _cmd)
	default:
		// Unknown attribute — call super.
		return callSuperPtrWithArg(self, _cmd, attrName, getClass("NSView"))
	}
}

func axNSValueWithPoint(x, y float64) uintptr {
	return msgSendPtr(getClass("NSValue"), sel("valueWithPoint:"),
		argPtr(uintptr(unsafe.Pointer(&nsPoint{X: x, Y: y}))))
}

func axNSValueWithSize(w, h float64) uintptr {
	return msgSendPtr(getClass("NSValue"), sel("valueWithSize:"),
		argPtr(uintptr(unsafe.Pointer(&nsSize{Width: w, Height: h}))))
}

// callSuperPtr calls objc_msgSendSuper to invoke the superclass implementation.
func callSuperPtr(self, cmd, superClass uintptr) uintptr {
	// struct objc_super { id receiver; Class super_class; }
	type objcSuper struct {
		receiver   uintptr
		superClass uintptr
	}
	sup := objcSuper{receiver: self, superClass: superClass}

	fnMsgSendSuper, _ := ffi.GetSymbol(rt.libobjc, "objc_msgSendSuper")
	var cif types.CallInterface
	_ = ffi.PrepareCallInterface(&cif, types.DefaultCall, types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor})

	supPtr := uintptr(unsafe.Pointer(&sup))
	var result uintptr
	_ = ffi.CallFunction(&cif, fnMsgSendSuper, unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&supPtr), unsafe.Pointer(&cmd)})
	return result
}

// callSuperPtrWithArg calls objc_msgSendSuper with one pointer argument.
func callSuperPtrWithArg(self, cmd, arg, superClass uintptr) uintptr {
	type objcSuper struct {
		receiver   uintptr
		superClass uintptr
	}
	sup := objcSuper{receiver: self, superClass: superClass}

	fnMsgSendSuper, _ := ffi.GetSymbol(rt.libobjc, "objc_msgSendSuper")
	var cif types.CallInterface
	_ = ffi.PrepareCallInterface(&cif, types.DefaultCall, types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor, types.PointerTypeDescriptor})

	supPtr := uintptr(unsafe.Pointer(&sup))
	var result uintptr
	_ = ffi.CallFunction(&cif, fnMsgSendSuper, unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&supPtr), unsafe.Pointer(&cmd), unsafe.Pointer(&arg)})
	return result
}

// bridgeForView returns the AXBridge associated with a view, or nil.
func bridgeForView(view uintptr) *AXBridge {
	v, ok := viewAXBridges.Load(view)
	if !ok {
		return nil
	}
	return v.(*AXBridge)
}

// ── ObjC callback implementations for LuxMetalView ──

func axViewIsElement(self, _cmd uintptr) uintptr {
	return 1 // YES
}

func axViewIsNotIgnored(self, _cmd uintptr) uintptr {
	println("[a11y-cb] accessibilityIsIgnored -> NO")
	return 0 // NO — don't skip this view
}

// ── Legacy accessibility protocol ──

// axViewAttrNames returns the list of supported accessibility attribute names.
// axViewAttrNames returns the super's attribute names with AXChildren ensured.
// This tells the legacy accessibility protocol that our view supports AXChildren,
// causing the system to call accessibilityAttributeValue:@"AXChildren" which
// NSView's default implementation dispatches to our accessibilityChildren override.
func axViewAttrNames(self, _cmd uintptr) uintptr {
	names := []uintptr{
		newNSString("AXRole"),
		newNSString("AXRoleDescription"),
		newNSString("AXChildren"),
		newNSString("AXParent"),
		newNSString("AXWindow"),
		newNSString("AXTopLevelUIElement"),
		newNSString("AXPosition"),
		newNSString("AXSize"),
		newNSString("AXFocused"),
		newNSString("AXFocusedUIElement"),
	}
	return newNSArray(names)
}

func axViewRole(self, _cmd uintptr) uintptr {
	return newNSString("AXGroup")
}

func axViewChildren(self, _cmd uintptr) uintptr {
	println("[a11y-CHILDREN] accessibilityChildren CALLED!")
	bridge := bridgeForView(self)
	if bridge == nil {
		return newNSArray(nil)
	}

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	if len(bridge.tree.Nodes) == 0 {
		return newNSArray(nil)
	}

	root := &bridge.tree.Nodes[0]
	children := bridge.tree.Children(root)
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if el := bridge.elementFor(child.ID); el != nil && el.obj != 0 {
			objs = append(objs, el.obj)
		}
	}

	return newNSArray(objs)
}

// axViewHitTest receives CGPoint as two float64 args (ffi.NewCallback limitation).
func axViewHitTest(self, _cmd uintptr, pointX, pointY float64) uintptr {
	bridge := bridgeForView(self)
	if bridge == nil {
		return self
	}

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	// Convert screen-space point (bottom-left origin) to window-local (top-left origin).
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
		return deepest.obj
	}
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
		return self
	}

	el := bridge.elementFor(bridge.tree.FocusedID)
	if el != nil && el.obj != 0 {
		return el.obj
	}
	return self
}
