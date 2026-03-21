//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
	"github.com/timzifer/lux/a11y"
)

// luxAXElementClass is the registered LuxAccessibilityElement ObjC class.
var (
	luxAXElementClass uintptr
	luxAXClassOnce    sync.Once
)

// registerLuxAccessibilityElementClass creates a custom ObjC subclass of
// NSAccessibilityElement that overrides the NSAccessibility protocol methods.
// Unlike plain NSAccessibilityElement with property setters, a subclass with
// method overrides is properly resolved by the macOS accessibility server for
// cross-process queries (e.g. Accessibility Inspector).
func registerLuxAccessibilityElementClass() uintptr {
	luxAXClassOnce.Do(func() {
		existing := getClass("LuxAccessibilityElement")
		if existing != 0 {
			luxAXElementClass = existing
			return
		}

		fnAllocClassPair, err := ffi.GetSymbol(rt.libobjc, "objc_allocateClassPair")
		if err != nil {
			log.Printf("[AX-SUBCLASS] failed to load objc_allocateClassPair: %v", err)
			return
		}
		fnRegisterClassPair, err := ffi.GetSymbol(rt.libobjc, "objc_registerClassPair")
		if err != nil {
			log.Printf("[AX-SUBCLASS] failed to load objc_registerClassPair: %v", err)
			return
		}
		fnAddMethod, err := ffi.GetSymbol(rt.libobjc, "class_addMethod")
		if err != nil {
			log.Printf("[AX-SUBCLASS] failed to load class_addMethod: %v", err)
			return
		}

		superClass := getClass("NSAccessibilityElement")
		if superClass == 0 {
			log.Printf("[AX-SUBCLASS] NSAccessibilityElement class not found")
			return
		}

		className := append([]byte("LuxAccessibilityElement"), 0)
		namePtr := unsafe.Pointer(&className[0])
		var extraBytes uint64

		var cifAllocClass types.CallInterface
		_ = ffi.PrepareCallInterface(&cifAllocClass, types.DefaultCall, types.PointerTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor, types.UInt64TypeDescriptor})

		var newClass uintptr
		_ = ffi.CallFunction(&cifAllocClass, fnAllocClassPair, unsafe.Pointer(&newClass),
			[]unsafe.Pointer{unsafe.Pointer(&superClass), unsafe.Pointer(&namePtr), unsafe.Pointer(&extraBytes)})
		runtime.KeepAlive(className)

		if newClass == 0 {
			log.Printf("[AX-SUBCLASS] objc_allocateClassPair returned nil")
			return
		}

		var cifAddMethod types.CallInterface
		_ = ffi.PrepareCallInterface(&cifAddMethod, types.DefaultCall, types.UInt8TypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor,
				types.PointerTypeDescriptor, types.PointerTypeDescriptor})

		addMethod := func(selName string, imp uintptr, typeEncoding string) {
			s := sel(selName)
			enc := append([]byte(typeEncoding), 0)
			encPtr := unsafe.Pointer(&enc[0])
			var result uint8
			_ = ffi.CallFunction(&cifAddMethod, fnAddMethod, unsafe.Pointer(&result),
				[]unsafe.Pointer{unsafe.Pointer(&newClass), unsafe.Pointer(&s),
					unsafe.Pointer(&imp), unsafe.Pointer(&encPtr)})
		}

		// Override NSAccessibility protocol methods so the AX server can query
		// element properties cross-process.
		addMethod("accessibilityRole", ffi.NewCallback(luxAXRole), "@@:")
		addMethod("accessibilityLabel", ffi.NewCallback(luxAXLabel), "@@:")
		addMethod("accessibilityValue", ffi.NewCallback(luxAXValue), "@@:")
		addMethod("accessibilityParent", ffi.NewCallback(luxAXParent), "@@:")
		addMethod("accessibilityChildren", ffi.NewCallback(luxAXChildren), "@@:")
		addMethod("accessibilityWindow", ffi.NewCallback(luxAXWindow), "@@:")
		addMethod("accessibilityTopLevelUIElement", ffi.NewCallback(luxAXTopLevel), "@@:")
		addMethod("accessibilitySubrole", ffi.NewCallback(luxAXSubrole), "@@:")
		addMethod("accessibilityRoleDescription", ffi.NewCallback(luxAXRoleDescription), "@@:")
		addMethod("accessibilityIdentifier", ffi.NewCallback(luxAXIdentifier), "@@:")
		addMethod("isAccessibilityElement", ffi.NewCallback(luxAXIsElement), "B@:")
		addMethod("isAccessibilityEnabled", ffi.NewCallback(luxAXIsEnabled), "B@:")
		addMethod("isAccessibilityFocused", ffi.NewCallback(luxAXIsFocused), "B@:")
		addMethod("accessibilityHitTest:", ffi.NewCallback(luxAXHitTest), "@@:{CGPoint=dd}")

		var cifRegister types.CallInterface
		_ = ffi.PrepareCallInterface(&cifRegister, types.DefaultCall, types.VoidTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor})
		_ = ffi.CallFunction(&cifRegister, fnRegisterClassPair, nil,
			[]unsafe.Pointer{unsafe.Pointer(&newClass)})

		classCache.Store("LuxAccessibilityElement", newClass)
		luxAXElementClass = newClass
		log.Printf("[AX-SUBCLASS] registered LuxAccessibilityElement class=%#x (super=%#x)", newClass, superClass)
	})
	return luxAXElementClass
}

// ── Helper to look up element info from the ObjC self pointer ──

// luxAXLookupInfo returns the bridge and nodeID for an element pointer.
// The caller must acquire the bridge lock before accessing tree data.
func luxAXLookupInfo(self uintptr) (*AXBridge, a11y.AccessNodeID) {
	v, ok := elementInfoMap.Load(self)
	if !ok {
		return nil, 0
	}
	info := v.(axElementInfo)
	return info.bridge, info.nodeID
}

// ── Protocol method overrides ──

func luxAXRole(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return newNSString("AXGroup")
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return newNSString("AXGroup")
	}
	role := roleToAXRole(node.Node.Role)
	log.Printf("[AX-SUB-CB] accessibilityRole: self=%#x nodeID=%d → %s", self, node.ID, role)
	return newNSString(role)
}

func luxAXLabel(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil || node.Node.Label == "" {
		return 0
	}
	log.Printf("[AX-SUB-CB] accessibilityLabel: self=%#x nodeID=%d → %q", self, node.ID, node.Node.Label)
	return newNSString(node.Node.Label)
}

func luxAXValue(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil || node.Node.Value == "" {
		return 0
	}
	log.Printf("[AX-SUB-CB] accessibilityValue: self=%#x nodeID=%d → %q", self, node.ID, node.Node.Value)
	return newNSString(node.Node.Value)
}

func luxAXParent(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return 0
	}

	if node.ParentIndex < 0 {
		return 0
	}
	parent := bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		return 0
	}
	if parent.ID == bridge.rootNodeID {
		log.Printf("[AX-SUB-CB] accessibilityParent: self=%#x nodeID=%d → view=%#x", self, node.ID, bridge.view)
		return bridge.view
	}
	if parentEl := bridge.elementFor(parent.ID); parentEl != nil && parentEl.obj != 0 {
		log.Printf("[AX-SUB-CB] accessibilityParent: self=%#x nodeID=%d → parent nodeID=%d obj=%#x", self, node.ID, parent.ID, parentEl.obj)
		return parentEl.obj
	}
	return 0
}

func luxAXChildren(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return newNSArray(nil)
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return newNSArray(nil)
	}

	if node.FirstChild < 0 {
		return newNSArray(nil)
	}
	children := bridge.tree.Children(node)
	objs := make([]uintptr, 0, len(children))
	for _, child := range children {
		if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
			objs = append(objs, childEl.obj)
		}
	}
	log.Printf("[AX-SUB-CB] accessibilityChildren: self=%#x nodeID=%d → %d children", self, node.ID, len(objs))
	return newNSArray(objs)
}

func luxAXWindow(self, _cmd uintptr) uintptr {
	bridge, _ := luxAXLookupInfo(self)
	if bridge == nil || bridge.view == 0 {
		return 0
	}
	win := msgSendPtr(bridge.view, sel("window"))
	return win
}

func luxAXTopLevel(self, _cmd uintptr) uintptr {
	return luxAXWindow(self, _cmd)
}

func luxAXSubrole(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return 0
	}
	sr := subroleForRole(node.Node.Role)
	if sr == "" {
		return 0
	}
	return newNSString(sr)
}

func luxAXRoleDescription(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return 0
	}
	// Use AppKit's standard role description function.
	role := newNSString(roleToAXRole(node.Node.Role))
	sr := subroleForRole(node.Node.Role)
	var nsSub uintptr
	if sr != "" {
		nsSub = newNSString(sr)
	}
	fn, err := ffi.GetSymbol(rt.appKit, "NSAccessibilityRoleDescription")
	if err != nil {
		return role // fallback: return role as description
	}
	var cif types.CallInterface
	_ = ffi.PrepareCallInterface(&cif, types.DefaultCall, types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor})
	var result uintptr
	_ = ffi.CallFunction(&cif, fn, unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&role), unsafe.Pointer(&nsSub)})
	return result
}

func luxAXIdentifier(self, _cmd uintptr) uintptr {
	_, nodeID := luxAXLookupInfo(self)
	if nodeID == 0 {
		return 0
	}
	return newNSString(fmt.Sprintf("lux-ax-%d", nodeID))
}

func luxAXIsElement(self, _cmd uintptr) uintptr {
	return 1 // YES
}

func luxAXIsEnabled(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 1
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return 1
	}
	if node.Node.States.Disabled {
		return 0
	}
	return 1
}

func luxAXIsFocused(self, _cmd uintptr) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return 0
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	if bridge.tree.FocusedID == nodeID {
		return 1
	}
	return 0
}

func luxAXHitTest(self, _cmd uintptr, pointX, pointY float64) uintptr {
	bridge, nodeID := luxAXLookupInfo(self)
	if bridge == nil {
		return self
	}
	bridge.mu.RLock()
	defer bridge.mu.RUnlock()
	node := bridge.tree.FindByID(nodeID)
	if node == nil {
		return self
	}

	// If this element has no children, it is the deepest hit.
	if node.FirstChild < 0 {
		return self
	}

	// Convert screen point to local coordinates.
	winOrigin := axWindowOrigin(bridge.view)
	winHeight := axWindowHeight(bridge.view)
	localX := pointX - winOrigin.X
	localY := winHeight - (pointY - winOrigin.Y)

	// Check children (last match = deepest).
	children := bridge.tree.Children(node)
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		b := child.Bounds
		if localX >= b.X && localX < b.X+b.Width &&
			localY >= b.Y && localY < b.Y+b.Height {
			if childEl := bridge.elementFor(child.ID); childEl != nil && childEl.obj != 0 {
				log.Printf("[AX-SUB-CB] accessibilityHitTest: self=%#x → child nodeID=%d", self, child.ID)
				return childEl.obj
			}
		}
	}
	return self
}
