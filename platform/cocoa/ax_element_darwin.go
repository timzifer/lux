//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"log"
	"runtime"
	"sync"
	"unsafe"

	"github.com/timzifer/lux/a11y"
)

// axElementInfo associates an ObjC element pointer with its bridge and node ID.
type axElementInfo struct {
	bridge *AXBridge
	nodeID a11y.AccessNodeID
}

// elementInfoMap maps ObjC element pointers to axElementInfo.
var elementInfoMap sync.Map

// newAXElement creates a LuxAccessibilityElement (custom subclass of NSAccessibilityElement)
// and configures it via property setters. The subclass overrides NSAccessibility protocol
// methods so the macOS accessibility server can query element properties cross-process
// (required for Accessibility Inspector, VoiceOver, etc.).
func newAXElement(bridge *AXBridge, nodeID a11y.AccessNodeID) *axElement {
	log.Printf("[AX-ELEM] newAXElement: nodeID=%d, rt=%v", nodeID, rt != nil)
	if rt == nil {
		log.Printf("[AX-ELEM] newAXElement: EARLY RETURN (rt==nil)")
		return &axElement{nodeID: nodeID}
	}

	cls := registerLuxAccessibilityElementClass()
	log.Printf("[AX-ELEM] newAXElement: LuxAccessibilityElement class=%#x", cls)
	if cls == 0 {
		log.Printf("[AX-ELEM] newAXElement: EARLY RETURN (class==0)")
		return &axElement{nodeID: nodeID}
	}

	obj := msgSendPtr(cls, sel("alloc"))
	obj = msgSendPtr(obj, sel("init"))
	log.Printf("[AX-ELEM] newAXElement: allocated obj=%#x", obj)
	if obj == 0 {
		log.Printf("[AX-ELEM] newAXElement: EARLY RETURN (obj==0)")
		return &axElement{nodeID: nodeID}
	}

	elementInfoMap.Store(obj, axElementInfo{bridge: bridge, nodeID: nodeID})

	node := bridge.tree.FindByID(nodeID)
	if node != nil {
		log.Printf("[AX-ELEM] newAXElement: nodeID=%d role=%v label=%q bounds={%.0f,%.0f,%.0f,%.0f}",
			nodeID, node.Node.Role, node.Node.Label,
			node.Bounds.X, node.Bounds.Y, node.Bounds.Width, node.Bounds.Height)
		// Most properties (role, label, value, parent, children, window, etc.)
		// are resolved dynamically by the LuxAccessibilityElement subclass overrides.
		// Frame is set via property since returning CGRect from ffi callback is unreliable.
		msgSendVoid(obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
			Origin: nsPoint{X: node.Bounds.X, Y: node.Bounds.Y},
			Size:   nsSize{Width: node.Bounds.Width, Height: node.Bounds.Height},
		}))
		if bridge.view != 0 {
			screenFrame := axFrameFromBounds(node.Bounds, bridge.view)
			msgSendVoid(obj, sel("setAccessibilityFrame:"), argRect(screenFrame))
		}
	} else {
		log.Printf("[AX-ELEM] WARNING: newAXElement nodeID=%d — node NOT FOUND in tree!", nodeID)
	}

	log.Printf("[AX-ELEM] newAXElement: DONE nodeID=%d obj=%#x", nodeID, obj)
	return &axElement{obj: obj, nodeID: nodeID}
}

func updateAXElementFrame(el *axElement, bounds a11y.Rect, view uintptr) {
	if el.obj == 0 {
		return
	}
	msgSendVoid(el.obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
		Origin: nsPoint{X: bounds.X, Y: bounds.Y},
		Size:   nsSize{Width: bounds.Width, Height: bounds.Height},
	}))
	if view != 0 {
		screenFrame := axFrameFromBounds(bounds, view)
		msgSendVoid(el.obj, sel("setAccessibilityFrame:"), argRect(screenFrame))
	}
}

// updateAXElementProperties is now a no-op: all properties are resolved
// dynamically by the LuxAccessibilityElement subclass method overrides.
func updateAXElementProperties(el *axElement, bridge *AXBridge, node *a11y.AccessTreeNode) {
	// No-op: properties are resolved dynamically via subclass callbacks.
}

func releaseAXElement(el *axElement) {
	if el.obj != 0 {
		log.Printf("[AX-ELEM] releaseAXElement: nodeID=%d obj=%#x", el.nodeID, el.obj)
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
	result := nsRect{
		Origin: nsPoint{X: screenX, Y: screenY},
		Size:   nsSize{Width: bounds.Width, Height: bounds.Height},
	}
	log.Printf("[AX-COORD] axFrameFromBounds: bounds={%.0f,%.0f,%.0f,%.0f} screenH=%.0f winOrigin=(%.0f,%.0f) → screen={%.0f,%.0f,%.0f,%.0f}",
		bounds.X, bounds.Y, bounds.Width, bounds.Height,
		screenHeight, windowOrigin.X, windowOrigin.Y,
		result.Origin.X, result.Origin.Y, result.Size.Width, result.Size.Height)
	return result
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
