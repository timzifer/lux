//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
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

// elementInfoMap maps ObjC NSAccessibilityElement pointers to axElementInfo.
var elementInfoMap sync.Map

// newAXElement creates a plain NSAccessibilityElement instance and configures
// it entirely via property setters. No subclass, no method overrides — on modern
// macOS the accessibility framework reads from NSAccessibilityElement's internal
// property storage, not from dynamically dispatched getter methods.
func newAXElement(bridge *AXBridge, nodeID a11y.AccessNodeID) *axElement {
	// Guard: if the ObjC runtime isn't loaded (e.g. in unit tests), skip.
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

	// Mark as an accessibility element.
	msgSendVoid(obj, sel("setAccessibilityElement:"), argBool(true))

	node := bridge.tree.FindByID(nodeID)
	if node != nil {
		// Set role.
		nsRole := newNSString(roleToAXRole(node.Node.Role))
		msgSendVoid(obj, sel("setAccessibilityRole:"), argPtr(nsRole))

		// Set subrole if applicable.
		if sr := subroleForRole(node.Node.Role); sr != "" {
			nsSR := newNSString(sr)
			msgSendVoid(obj, sel("setAccessibilitySubrole:"), argPtr(nsSR))
		}

		// Set label.
		if node.Node.Label != "" {
			nsLabel := newNSString(node.Node.Label)
			msgSendVoid(obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
		}

		// Set value.
		if node.Node.Value != "" {
			nsValue := newNSString(node.Node.Value)
			msgSendVoid(obj, sel("setAccessibilityValue:"), argPtr(nsValue))
		}

		// Set enabled state.
		msgSendVoid(obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))

		// Set frame in screen coordinates.
		if bridge.view != 0 {
			frame := axFrameFromBounds(node.Bounds, bridge.view)
			msgSendVoid(obj, sel("setAccessibilityFrame:"), argRect(frame))
		}

		// Set parent: synthetic root's children → view, otherwise → parent element.
		axSetElementParent(obj, bridge, node)
	}

	return &axElement{obj: obj, nodeID: nodeID}
}

// axSetElementParent sets the accessibilityParent property on an ObjC element.
func axSetElementParent(obj uintptr, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if node.ParentIndex < 0 {
		return
	}
	parent := bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		return
	}
	if parent.ID == bridge.rootNodeID {
		// Top-level element: parent is the view.
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(bridge.view))
	} else if parentEl := bridge.elementFor(parent.ID); parentEl != nil && parentEl.obj != 0 {
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(parentEl.obj))
	}
}

// updateAXElementFrame updates the accessibility frame for an element.
func updateAXElementFrame(el *axElement, bounds a11y.Rect, view uintptr) {
	if el.obj == 0 || view == 0 {
		return
	}
	frame := axFrameFromBounds(bounds, view)
	msgSendVoid(el.obj, sel("setAccessibilityFrame:"), argRect(frame))
}

// updateAXElementProperties refreshes label, value, enabled, and parent properties.
func updateAXElementProperties(el *axElement, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if el.obj == 0 {
		return
	}
	// Update label.
	if node.Node.Label != "" {
		nsLabel := newNSString(node.Node.Label)
		msgSendVoid(el.obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
	}

	// Update value.
	if node.Node.Value != "" {
		nsValue := newNSString(node.Node.Value)
		msgSendVoid(el.obj, sel("setAccessibilityValue:"), argPtr(nsValue))
	}

	// Update enabled state.
	msgSendVoid(el.obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))

	// Update parent relationship.
	axSetElementParent(el.obj, bridge, node)
}

// releaseAXElement releases the ObjC element.
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

// axFrameFromBounds converts a11y.Rect to screen-space nsRect with Y-flip.
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

// axScreenHeight returns the height of the main screen.
func axScreenHeight() float64 {
	mainScreen := msgSendPtr(getClass("NSScreen"), sel("mainScreen"))
	if mainScreen == 0 {
		return 900 // fallback
	}
	frame := msgSendRect(mainScreen, sel("frame"))
	return frame.Size.Height
}

// axWindowOrigin returns the window content area origin in screen coords (bottom-left).
func axWindowOrigin(view uintptr) nsPoint {
	window := msgSendPtr(view, sel("window"))
	if window == 0 {
		return nsPoint{}
	}
	frame := msgSendRect(window, sel("frame"))
	contentRect := msgSendRect(window, sel("contentRectForFrameRect:"), argRect(frame))
	return contentRect.Origin
}

// axWindowHeight returns the window content area height.
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

// newNSArray creates an NSArray from a slice of ObjC object pointers.
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
