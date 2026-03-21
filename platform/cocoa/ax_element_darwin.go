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

// newAXElement creates an NSAccessibilityElement and configures it via property setters.
// This is Apple's recommended approach for virtual (non-view-backed) elements.
// No subclass or method overrides — NSAccessibilityElement handles all protocol queries
// from its internal property storage.
func newAXElement(bridge *AXBridge, nodeID a11y.AccessNodeID) *axElement {
	log.Printf("[AX-ELEM] newAXElement: nodeID=%d, rt=%v", nodeID, rt != nil)
	if rt == nil {
		log.Printf("[AX-ELEM] newAXElement: EARLY RETURN (rt==nil)")
		return &axElement{nodeID: nodeID}
	}

	cls := getClass("NSAccessibilityElement")
	log.Printf("[AX-ELEM] newAXElement: NSAccessibilityElement class=%#x", cls)
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
		axRole := roleToAXRole(node.Node.Role)
		log.Printf("[AX-ELEM] newAXElement: nodeID=%d role=%v→%s label=%q value=%q disabled=%v bounds={%.0f,%.0f,%.0f,%.0f}",
			nodeID, node.Node.Role, axRole, node.Node.Label, node.Node.Value, node.Node.States.Disabled,
			node.Bounds.X, node.Bounds.Y, node.Bounds.Width, node.Bounds.Height)

		// Role.
		nsRole := newNSString(axRole)
		msgSendVoid(obj, sel("setAccessibilityRole:"), argPtr(nsRole))

		// Label.
		if node.Node.Label != "" {
			nsLabel := newNSString(node.Node.Label)
			msgSendVoid(obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
		}

		// Value.
		if node.Node.Value != "" {
			nsVal := newNSString(node.Node.Value)
			msgSendVoid(obj, sel("setAccessibilityValue:"), argPtr(nsVal))
		}

		// Enabled.
		msgSendVoid(obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))

		// Frame – set both parent-space and screen-coordinate frames.
		msgSendVoid(obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
			Origin: nsPoint{X: node.Bounds.X, Y: node.Bounds.Y},
			Size:   nsSize{Width: node.Bounds.Width, Height: node.Bounds.Height},
		}))
		if bridge.view != 0 {
			screenFrame := axFrameFromBounds(node.Bounds, bridge.view)
			log.Printf("[AX-ELEM] newAXElement: nodeID=%d screenFrame={%.0f,%.0f,%.0f,%.0f}",
				nodeID, screenFrame.Origin.X, screenFrame.Origin.Y, screenFrame.Size.Width, screenFrame.Size.Height)
			msgSendVoid(obj, sel("setAccessibilityFrame:"), argRect(screenFrame))
		}

		// Subrole (e.g. AXDialog for dialog groups).
		if sr := subroleForRole(node.Node.Role); sr != "" {
			log.Printf("[AX-ELEM] newAXElement: nodeID=%d subrole=%s", nodeID, sr)
			nsSub := newNSString(sr)
			msgSendVoid(obj, sel("setAccessibilitySubrole:"), argPtr(nsSub))
		}

		// Parent.
		axSetElementParent(obj, bridge, node)

		// Window & top-level element – required for cross-process AX queries
		if bridge.view != 0 {
			win := msgSendPtr(bridge.view, sel("window"))
			log.Printf("[AX-ELEM] newAXElement: nodeID=%d view=%#x window=%#x", nodeID, bridge.view, win)
			if win != 0 {
				msgSendVoid(obj, sel("setAccessibilityWindow:"), argPtr(win))
				msgSendVoid(obj, sel("setAccessibilityTopLevelUIElement:"), argPtr(win))
			} else {
				log.Printf("[AX-ELEM] WARNING: nodeID=%d window is nil!", nodeID)
			}
		} else {
			log.Printf("[AX-ELEM] WARNING: nodeID=%d bridge.view is 0!", nodeID)
		}
	} else {
		log.Printf("[AX-ELEM] WARNING: newAXElement nodeID=%d — node NOT FOUND in tree!", nodeID)
	}

	log.Printf("[AX-ELEM] newAXElement: DONE nodeID=%d obj=%#x", nodeID, obj)
	return &axElement{obj: obj, nodeID: nodeID}
}

func axSetElementParent(obj uintptr, bridge *AXBridge, node *a11y.AccessTreeNode) {
	log.Printf("[AX-ELEM] axSetElementParent: obj=%#x nodeID=%d parentIndex=%d", obj, node.ID, node.ParentIndex)
	if node.ParentIndex < 0 {
		log.Printf("[AX-ELEM] axSetElementParent: no parent (parentIndex<0)")
		return
	}
	parent := bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		log.Printf("[AX-ELEM] axSetElementParent: parent node at index %d is nil", node.ParentIndex)
		return
	}
	if parent.ID == bridge.rootNodeID {
		log.Printf("[AX-ELEM] axSetElementParent: parent is root → setting parent to view=%#x", bridge.view)
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(bridge.view))
	} else if parentEl := bridge.elementFor(parent.ID); parentEl != nil && parentEl.obj != 0 {
		log.Printf("[AX-ELEM] axSetElementParent: parent nodeID=%d obj=%#x", parent.ID, parentEl.obj)
		msgSendVoid(obj, sel("setAccessibilityParent:"), argPtr(parentEl.obj))
	} else {
		log.Printf("[AX-ELEM] WARNING: axSetElementParent: parent nodeID=%d has no element!", parent.ID)
	}
}

func updateAXElementFrame(el *axElement, bounds a11y.Rect, view uintptr) {
	if el.obj == 0 {
		return
	}
	log.Printf("[AX-ELEM] updateAXElementFrame: nodeID=%d obj=%#x bounds={%.0f,%.0f,%.0f,%.0f}",
		el.nodeID, el.obj, bounds.X, bounds.Y, bounds.Width, bounds.Height)
	msgSendVoid(el.obj, sel("setAccessibilityFrameInParentSpace:"), argRect(nsRect{
		Origin: nsPoint{X: bounds.X, Y: bounds.Y},
		Size:   nsSize{Width: bounds.Width, Height: bounds.Height},
	}))
	if view != 0 {
		screenFrame := axFrameFromBounds(bounds, view)
		log.Printf("[AX-ELEM] updateAXElementFrame: nodeID=%d screenFrame={%.0f,%.0f,%.0f,%.0f}",
			el.nodeID, screenFrame.Origin.X, screenFrame.Origin.Y, screenFrame.Size.Width, screenFrame.Size.Height)
		msgSendVoid(el.obj, sel("setAccessibilityFrame:"), argRect(screenFrame))
	}
}

func updateAXElementProperties(el *axElement, bridge *AXBridge, node *a11y.AccessTreeNode) {
	if el.obj == 0 {
		return
	}
	log.Printf("[AX-ELEM] updateAXElementProperties: nodeID=%d obj=%#x label=%q value=%q disabled=%v",
		el.nodeID, el.obj, node.Node.Label, node.Node.Value, node.Node.States.Disabled)
	if node.Node.Label != "" {
		nsLabel := newNSString(node.Node.Label)
		msgSendVoid(el.obj, sel("setAccessibilityLabel:"), argPtr(nsLabel))
	}
	if node.Node.Value != "" {
		nsVal := newNSString(node.Node.Value)
		msgSendVoid(el.obj, sel("setAccessibilityValue:"), argPtr(nsVal))
	}
	msgSendVoid(el.obj, sel("setAccessibilityEnabled:"), argBool(!node.Node.States.Disabled))
	axSetElementParent(el.obj, bridge, node)

	// Keep window/top-level references current for AX inspector queries.
	if bridge.view != 0 {
		win := msgSendPtr(bridge.view, sel("window"))
		log.Printf("[AX-ELEM] updateAXElementProperties: nodeID=%d view=%#x window=%#x", el.nodeID, bridge.view, win)
		if win != 0 {
			msgSendVoid(el.obj, sel("setAccessibilityWindow:"), argPtr(win))
			msgSendVoid(el.obj, sel("setAccessibilityTopLevelUIElement:"), argPtr(win))
		}
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
