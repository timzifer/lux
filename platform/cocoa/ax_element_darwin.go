//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
	"github.com/timzifer/lux/a11y"
)

// axElementInfo associates an ObjC element pointer with its bridge and node ID.
type axElementInfo struct {
	bridge *AXBridge
	nodeID a11y.AccessNodeID
}

// elementInfoMap maps ObjC element pointers to axElementInfo.
var elementInfoMap sync.Map

// newAXElement creates an NSAccessibilityElement and configures it via property setters.
// This is Apple's recommended approach for virtual (non-view-backed) elements.
// No subclass or method overrides — NSAccessibilityElement handles all protocol queries
// from its internal property storage.
func newAXElement(bridge *AXBridge, nodeID a11y.AccessNodeID) *axElement {
	if rt == nil {
		return &axElement{nodeID: nodeID}
	}

	cls := getClass("NSAccessibilityElement")
	if cls == 0 {
		return &axElement{nodeID: nodeID}
	}

	obj := msgSendPtr(cls, sel("alloc"))
	obj = msgSendPtr(obj, sel("init"))
	if obj == 0 {
		return &axElement{nodeID: nodeID}
	}

	elementInfoMap.Store(obj, axElementInfo{bridge: bridge, nodeID: nodeID})

	node := bridge.tree.FindByID(nodeID)
	if node != nil {
		// Role.
		nsRole := newNSString(roleToAXRole(node.Node.Role))
		msgSendVoid(obj, sel("setAccessibilityRole:"), argPtr(nsRole))

		// Subrole.
		if subrole := subroleForRole(node.Node.Role); subrole != "" {
			nsSubrole := newNSString(subrole)
			msgSendOptionalPtr(obj, "setAccessibilitySubrole:", nsSubrole)
		}

		// Label/title.
		nsLabel := newNSString(node.Node.Label)
		msgSendVoid(obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
		msgSendOptionalPtr(obj, "setAccessibilityTitle:", nsLabel)

		// Value.
		nsVal := newNSString(node.Node.Value)
		msgSendVoid(obj, sel("setAccessibilityValue:"), argPtr(nsVal))

		// Enabled.
		msgSendVoid(obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))

		// Identifier.
		nsIdentifier := newNSString(fmt.Sprintf("lux_%d", nodeID))
		msgSendOptionalPtr(obj, "setAccessibilityIdentifier:", nsIdentifier)

		// Frame in parent space (relative to parent, not screen).
		// Also set the absolute accessibility frame so Inspector/AX clients can
		// highlight the element even if parent-space conversion is not enough.
		msgSendOptionalRect(obj, "setAccessibilityFrame:", axFrameFromBounds(node.Bounds, bridge.view))
		msgSendVoid(obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
			Origin: nsPoint{X: node.Bounds.X, Y: node.Bounds.Y},
			Size:   nsSize{Width: node.Bounds.Width, Height: node.Bounds.Height},
		}))

		// Supported actions.
		if actions := actionsForRole(node.Node.Role); len(actions) > 0 {
			actionNames := make([]uintptr, 0, len(actions))
			for _, action := range actions {
				actionNames = append(actionNames, newNSString(action))
			}
			msgSendOptionalPtr(obj, "setAccessibilityActionNames:", newNSArray(actionNames))
		}

		// Initialize children to an empty array so AppKit never sees stale/nil state.
		msgSendVoid(obj, sel("setAccessibilityChildren:"), argPtr(newNSArray(nil)))

		// Parent.
		axSetElementParent(obj, bridge, node)
	}

	return &axElement{obj: obj, nodeID: nodeID}
}

func axSetElementParent(obj uintptr, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if node.ParentIndex < 0 {
		return
	}
	parent := bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		return
	}
	if parent.ID == bridge.rootNodeID {
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(axUnignoredAncestor(bridge.view)))
	} else if parentEl := bridge.elementFor(parent.ID); parentEl != nil && parentEl.obj != 0 {
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(parentEl.obj))
	}
}

func updateAXElementFrame(el *axElement, bounds a11y.Rect, view uintptr) {
	if el.obj == 0 {
		return
	}
	msgSendOptionalRect(el.obj, "setAccessibilityFrame:", axFrameFromBounds(bounds, view))
	msgSendVoid(el.obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
		Origin: nsPoint{X: bounds.X, Y: bounds.Y},
		Size:   nsSize{Width: bounds.Width, Height: bounds.Height},
	}))
}

func updateAXElementProperties(el *axElement, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if el.obj == 0 {
		return
	}
	nsRole := newNSString(roleToAXRole(node.Node.Role))
	msgSendVoid(el.obj, sel("setAccessibilityRole:"), argPtr(nsRole))
	if subrole := subroleForRole(node.Node.Role); subrole != "" {
		nsSubrole := newNSString(subrole)
		msgSendOptionalPtr(el.obj, "setAccessibilitySubrole:", nsSubrole)
	}
	nsLabel := newNSString(node.Node.Label)
	msgSendVoid(el.obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
	msgSendOptionalPtr(el.obj, "setAccessibilityTitle:", nsLabel)
	nsVal := newNSString(node.Node.Value)
	msgSendVoid(el.obj, sel("setAccessibilityValue:"), argPtr(nsVal))
	msgSendVoid(el.obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))
	if actions := actionsForRole(node.Node.Role); len(actions) > 0 {
		actionNames := make([]uintptr, 0, len(actions))
		for _, action := range actions {
			actionNames = append(actionNames, newNSString(action))
		}
		msgSendOptionalPtr(el.obj, "setAccessibilityActionNames:", newNSArray(actionNames))
	} else if respondsToSelector(el.obj, sel("setAccessibilityActionNames:")) {
		msgSendVoid(el.obj, sel("setAccessibilityActionNames:"), argPtr(newNSArray(nil)))
	}
	axSetElementParent(el.obj, bridge, node)
}

func msgSendOptionalPtr(self uintptr, selectorName string, value uintptr) bool {
	cmd := sel(selectorName)
	if !respondsToSelector(self, cmd) {
		return false
	}
	msgSendVoid(self, cmd, argPtr(value))
	return true
}

func msgSendOptionalRect(self uintptr, selectorName string, value nsRect) bool {
	cmd := sel(selectorName)
	if !respondsToSelector(self, cmd) {
		return false
	}
	msgSendVoid(self, cmd, argRect(value))
	return true
}

var (
	axUnignoredAncestorOnce sync.Once
	fnAXUnignoredAncestor   unsafe.Pointer
	cifAXUnignoredAncestor  types.CallInterface
)

func ensureAXUnignoredAncestor() {
	axUnignoredAncestorOnce.Do(func() {
		if rt == nil {
			return
		}
		var err error
		fnAXUnignoredAncestor, err = ffi.GetSymbol(rt.appKit, "NSAccessibilityUnignoredAncestor")
		if err != nil {
			return
		}
		_ = ffi.PrepareCallInterface(&cifAXUnignoredAncestor, types.DefaultCall, types.PointerTypeDescriptor,
			[]*types.TypeDescriptor{types.PointerTypeDescriptor})
	})
}

func axUnignoredAncestor(obj uintptr) uintptr {
	if obj == 0 {
		return 0
	}
	ensureAXUnignoredAncestor()
	if fnAXUnignoredAncestor == nil {
		return obj
	}
	var result uintptr
	_ = ffi.CallFunction(&cifAXUnignoredAncestor, fnAXUnignoredAncestor, unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&obj)})
	if result == 0 {
		return obj
	}
	return result
}

func releaseAXElement(el *axElement) {
	if el.obj != 0 {
		elementInfoMap.Delete(el.obj)
		msgSendVoid(el.obj, sel("release"))
		el.obj = 0
	}
}

// axAction is sent via app.Send when accessibility requests an action.
type axAction struct {
	trigger func()
}

// ── Coordinate conversion ──

func axFrameFromBounds(bounds a11y.Rect, view uintptr) nsRect {
	screenHeight := axScreenHeight()
	windowOrigin := axWindowOrigin(view)
	screenX := bounds.X + windowOrigin.X
	screenY := screenHeight - (bounds.Y + windowOrigin.Y) - bounds.Height
	return nsRect{
		Origin: nsPoint{X: screenX, Y: screenY},
		Size:   nsSize{Width: bounds.Width, Height: bounds.Height},
	}
}

func axScreenHeight() float64 {
	mainScreen := msgSendPtr(getClass("NSScreen"), sel("mainScreen"))
	if mainScreen == 0 {
		return 900
	}
	frame := msgSendRect(mainScreen, sel("frame"))
	return frame.Size.Height
}

func axWindowOrigin(view uintptr) nsPoint {
	window := msgSendPtr(view, sel("window"))
	if window == 0 {
		return nsPoint{}
	}
	frame := msgSendRect(window, sel("frame"))
	contentRect := msgSendRect(window, sel("contentRectForFrameRect:"), argRect(frame))
	return contentRect.Origin
}

func axWindowHeight(view uintptr) float64 {
	window := msgSendPtr(view, sel("window"))
	if window == 0 {
		return 600
	}
	frame := msgSendRect(window, sel("frame"))
	contentRect := msgSendRect(window, sel("contentRectForFrameRect:"), argRect(frame))
	return contentRect.Size.Height
}

// ── NSArray helper ──

func newNSArray(objs []uintptr) uintptr {
	if len(objs) == 0 {
		return msgSendPtr(getClass("NSArray"), sel("array"))
	}
	count := uint64(len(objs))
	objsPtr := unsafe.Pointer(&objs[0])
	arr := msgSendPtr(getClass("NSArray"), sel("arrayWithObjects:count:"),
		argPtr(uintptr(objsPtr)), argUInt64(count))
	runtime.KeepAlive(objs)
	return arr
}
