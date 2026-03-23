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
	if rt == nil {
		return &axElement{nodeID: nodeID}
	}

	cls := registerLuxAccessibilityElementClass()
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

		// Set frame via property (returning CGRect from ffi callback is unreliable).
		msgSendVoid(obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
			Origin: nsPoint{X: node.Bounds.X, Y: node.Bounds.Y},
			Size:   nsSize{Width: node.Bounds.Width, Height: node.Bounds.Height},
		}))
		if bridge.view != 0 {
			screenFrame := axFrameFromBounds(node.Bounds, bridge.view)
			msgSendVoid(obj, sel("setAccessibilityFrame:"), argRect(screenFrame))
		}

		// Mark as accessibility element. NSAccessibilityElement may return
		// YES from the isAccessibilityElement method, but the backing ivar
		// defaults to NO. The AX server reads the ivar for hierarchy
		// traversal, so we must set it explicitly.
		msgSendVoid(obj, sel("setAccessibilityElement:"), argBool(true))

		// Set instance properties so the AX server can discover elements
		// via ivar access. The LuxAccessibilityElement subclass overrides
		// handle dynamic queries; these property setters prime the ivars
		// for initial hierarchy discovery.
		role := roleToAXRole(node.Node.Role)
		msgSendVoid(obj, sel("setAccessibilityRole:"), argPtr(newNSString(role)))
		if node.Node.Label != "" {
			msgSendVoid(obj, sel("setAccessibilityLabel:"), argPtr(newNSString(node.Node.Label)))
		}
		if node.Node.Value != "" {
			msgSendVoid(obj, sel("setAccessibilityValue:"), argPtr(newNSString(node.Node.Value)))
		}

		// Set parent.
		if node.ParentIndex >= 0 {
			parent := bridge.tree.NodeByIndex(int(node.ParentIndex))
			if parent != nil && parent.ID == bridge.rootNodeID {
				msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(bridge.view))
			} else if parent != nil {
				if parentEl := bridge.elementFor(parent.ID); parentEl != nil && parentEl.obj != 0 {
					msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(parentEl.obj))
				}
			}
		}

		// Set window and top-level element.
		if bridge.view != 0 {
			win := msgSendPtr(bridge.view, sel("window"))
			if win != 0 {
				msgSendVoid(obj, sel("setAccessibilityWindow:"), argPtr(win))
				msgSendVoid(obj, sel("setAccessibilityTopLevelUIElement:"), argPtr(win))
			}
		}
	}
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

// updateAXElementProperties updates the property backing ivars on an existing
// element so the AX server sees current values via both ivar access and
// method overrides.
func updateAXElementProperties(el *axElement, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if el.obj == 0 {
		return
	}
	role := roleToAXRole(node.Node.Role)
	msgSendVoid(el.obj, sel("setAccessibilityRole:"), argPtr(newNSString(role)))
	if node.Node.Label != "" {
		msgSendVoid(el.obj, sel("setAccessibilityLabel:"), argPtr(newNSString(node.Node.Label)))
	}
	if node.Node.Value != "" {
		msgSendVoid(el.obj, sel("setAccessibilityValue:"), argPtr(newNSString(node.Node.Value)))
	}
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
