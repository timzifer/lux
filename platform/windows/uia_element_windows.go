//go:build windows && !nogui

package windows

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/a11y"
	"github.com/zzl/go-win32api/v2/win32"
)

// elementProvider implements IRawElementProviderSimple and IRawElementProviderFragment
// for a single AccessTreeNode.
type elementProvider struct {
	refCount     int32
	vtblSimple   *win32.IRawElementProviderSimpleVtbl
	vtblFragment *win32.IRawElementProviderFragmentVtbl
	bridge       *UIABridge
	nodeID       a11y.AccessNodeID
}

var (
	elemSimpleVtbl   *win32.IRawElementProviderSimpleVtbl
	elemFragmentVtbl *win32.IRawElementProviderFragmentVtbl
)

func initElementVtables() {
	if elemSimpleVtbl != nil {
		return
	}

	elemSimpleVtbl = &win32.IRawElementProviderSimpleVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(elemQueryInterface),
			AddRef:         syscall.NewCallback(elemAddRef),
			Release:        syscall.NewCallback(elemRelease),
		},
		Get_ProviderOptions:        syscall.NewCallback(elemGetProviderOptions),
		GetPatternProvider:         syscall.NewCallback(elemGetPatternProvider),
		GetPropertyValue:           syscall.NewCallback(elemGetPropertyValue),
		Get_HostRawElementProvider: syscall.NewCallback(elemGetHostProvider),
	}

	elemFragmentVtbl = &win32.IRawElementProviderFragmentVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(elemQueryInterface),
			AddRef:         syscall.NewCallback(elemAddRef),
			Release:        syscall.NewCallback(elemRelease),
		},
		Navigate:                 syscall.NewCallback(elemNavigate),
		GetRuntimeId:             syscall.NewCallback(elemGetRuntimeId),
		Get_BoundingRectangle:    syscall.NewCallback(elemGetBoundingRectangle),
		GetEmbeddedFragmentRoots: syscall.NewCallback(elemGetEmbeddedFragmentRoots),
		SetFocus:                 syscall.NewCallback(elemSetFocus),
		Get_FragmentRoot:         syscall.NewCallback(elemGetFragmentRoot),
	}
}

func newElementProvider(bridge *UIABridge, nodeID a11y.AccessNodeID) *elementProvider {
	initElementVtables()
	return &elementProvider{
		refCount:     1,
		vtblSimple:   elemSimpleVtbl,
		vtblFragment: elemFragmentVtbl,
		bridge:       bridge,
		nodeID:       nodeID,
	}
}

func elemFromSimple(this uintptr) *elementProvider {
	return (*elementProvider)(unsafe.Pointer(this - unsafe.Offsetof(elementProvider{}.vtblSimple)))
}

func elemFromFragment(this uintptr) *elementProvider {
	return (*elementProvider)(unsafe.Pointer(this - unsafe.Offsetof(elementProvider{}.vtblFragment)))
}

func resolveElemProvider(this uintptr) *elementProvider {
	vptr := *(*uintptr)(unsafe.Pointer(this))
	switch vptr {
	case uintptr(unsafe.Pointer(elemSimpleVtbl)):
		return elemFromSimple(this)
	case uintptr(unsafe.Pointer(elemFragmentVtbl)):
		return elemFromFragment(this)
	}
	return nil
}

// --- IUnknown ---

func elemQueryInterface(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	ep := resolveElemProvider(this)
	if ep == nil {
		*ppv = nil
		return hresult(win32.E_NOINTERFACE)
	}

	if *riid == win32.IID_IUnknown || *riid == win32.IID_IRawElementProviderSimple {
		*ppv = unsafe.Pointer(&ep.vtblSimple)
		atomic.AddInt32(&ep.refCount, 1)
		return uintptr(win32.S_OK)
	}
	if *riid == win32.IID_IRawElementProviderFragment {
		*ppv = unsafe.Pointer(&ep.vtblFragment)
		atomic.AddInt32(&ep.refCount, 1)
		return uintptr(win32.S_OK)
	}

	// Check for pattern provider interfaces.
	ep.bridge.mu.RLock()
	node := ep.bridge.tree.FindByID(ep.nodeID)
	ep.bridge.mu.RUnlock()

	if node != nil {
		patternPtr := getPatternProviderForIID(ep, node, riid)
		if patternPtr != nil {
			*ppv = patternPtr
			return uintptr(win32.S_OK)
		}
	}

	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func elemAddRef(this uintptr) uintptr {
	ep := resolveElemProvider(this)
	if ep == nil {
		return 0
	}
	return uintptr(atomic.AddInt32(&ep.refCount, 1))
}

func elemRelease(this uintptr) uintptr {
	ep := resolveElemProvider(this)
	if ep == nil {
		return 0
	}
	ref := atomic.AddInt32(&ep.refCount, -1)
	return uintptr(ref)
}

// --- IRawElementProviderSimple ---

func elemGetProviderOptions(this uintptr, pRetVal *win32.ProviderOptions) uintptr {
	*pRetVal = win32.ProviderOptions_ServerSideProvider | win32.ProviderOptions_UseComThreading
	return uintptr(win32.S_OK)
}

func elemGetPatternProvider(this uintptr, patternID win32.UIA_PATTERN_ID, pRetVal *unsafe.Pointer) uintptr {
	ep := elemFromSimple(this)
	*pRetVal = nil

	ep.bridge.mu.RLock()
	node := ep.bridge.tree.FindByID(ep.nodeID)
	ep.bridge.mu.RUnlock()

	if node == nil {
		return uintptr(win32.S_OK)
	}

	*pRetVal = getPatternProvider(ep, node, patternID)
	return uintptr(win32.S_OK)
}

func elemGetPropertyValue(this uintptr, propertyID win32.UIA_PROPERTY_ID, pRetVal *win32.VARIANT) uintptr {
	ep := elemFromSimple(this)

	ep.bridge.mu.RLock()
	node := ep.bridge.tree.FindByID(ep.nodeID)
	ep.bridge.mu.RUnlock()

	if node == nil {
		*pRetVal = variantEmpty()
		return uintptr(win32.S_OK)
	}

	switch propertyID {
	case win32.UIA_ControlTypePropertyId:
		*pRetVal = variantInt32(int32(roleToControlType(node.Node.Role)))
	case win32.UIA_NamePropertyId:
		*pRetVal = variantString(node.Node.Label)
	case win32.UIA_IsEnabledPropertyId:
		*pRetVal = variantBool(!node.Node.States.Disabled)
	case win32.UIA_IsKeyboardFocusablePropertyId:
		*pRetVal = variantBool(!node.Node.States.Disabled)
	case win32.UIA_HasKeyboardFocusPropertyId:
		*pRetVal = variantBool(node.Node.States.Focused)
	case win32.UIA_AutomationIdPropertyId:
		*pRetVal = variantString(fmt.Sprintf("lux_%d", ep.nodeID))
	default:
		*pRetVal = variantEmpty()
	}
	return uintptr(win32.S_OK)
}

func elemGetHostProvider(this uintptr, pRetVal **win32.IRawElementProviderSimple) uintptr {
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

// --- IRawElementProviderFragment ---

func elemNavigate(this uintptr, direction win32.NavigateDirection, pRetVal **win32.IRawElementProviderFragment) uintptr {
	ep := elemFromFragment(this)
	*pRetVal = nil

	ep.bridge.mu.RLock()
	defer ep.bridge.mu.RUnlock()

	tree := &ep.bridge.tree
	node := tree.FindByID(ep.nodeID)
	if node == nil {
		return uintptr(win32.S_OK)
	}

	switch direction {
	case win32.NavigateDirection_Parent:
		if node.ParentIndex < 0 {
			// Should not happen (only the synthetic root has parentIndex=-1,
			// and we don't create element providers for it). Safety fallback.
			return uintptr(win32.S_OK)
		}
		parent := tree.NodeByIndex(int(node.ParentIndex))
		if parent != nil && parent.ID == ep.bridge.rootNodeID {
			// Parent is the synthetic root → return the root UIA provider.
			rp := ep.bridge.root
			atomic.AddInt32(&rp.refCount, 1)
			*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&rp.vtblFragment))
		} else if parent != nil {
			pp := ep.bridge.providerFor(parent.ID)
			if pp != nil {
				atomic.AddInt32(&pp.refCount, 1)
				*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&pp.vtblFragment))
			}
		}

	case win32.NavigateDirection_NextSibling:
		if node.NextSibling >= 0 {
			sibling := tree.NodeByIndex(int(node.NextSibling))
			if sibling != nil {
				sp := ep.bridge.providerFor(sibling.ID)
				if sp != nil {
					atomic.AddInt32(&sp.refCount, 1)
					*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&sp.vtblFragment))
				}
			}
		}

	case win32.NavigateDirection_PreviousSibling:
		if node.PrevSibling >= 0 {
			sibling := tree.NodeByIndex(int(node.PrevSibling))
			if sibling != nil {
				sp := ep.bridge.providerFor(sibling.ID)
				if sp != nil {
					atomic.AddInt32(&sp.refCount, 1)
					*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&sp.vtblFragment))
				}
			}
		}

	case win32.NavigateDirection_FirstChild:
		if node.FirstChild >= 0 {
			child := tree.NodeByIndex(int(node.FirstChild))
			if child != nil {
				cp := ep.bridge.providerFor(child.ID)
				if cp != nil {
					atomic.AddInt32(&cp.refCount, 1)
					*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&cp.vtblFragment))
				}
			}
		}

	case win32.NavigateDirection_LastChild:
		children := tree.Children(node)
		if len(children) > 0 {
			last := children[len(children)-1]
			lp := ep.bridge.providerFor(last.ID)
			if lp != nil {
				atomic.AddInt32(&lp.refCount, 1)
				*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&lp.vtblFragment))
			}
		}
	}

	return uintptr(win32.S_OK)
}

func elemGetRuntimeId(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	ep := elemFromFragment(this)
	// Runtime ID: [UiaAppendRuntimeId, nodeID]
	// UiaAppendRuntimeId = 3
	ids := [2]int32{3, int32(ep.nodeID)}
	// Allocate SAFEARRAY for the runtime ID.
	sa := safeArrayFromInt32s(ids[:])
	*pRetVal = unsafe.Pointer(sa)
	return uintptr(win32.S_OK)
}

func elemGetBoundingRectangle(this uintptr, pRetVal *win32.UiaRect) uintptr {
	ep := elemFromFragment(this)

	ep.bridge.mu.RLock()
	node := ep.bridge.tree.FindByID(ep.nodeID)
	ep.bridge.mu.RUnlock()

	if node == nil {
		*pRetVal = win32.UiaRect{}
		return uintptr(win32.S_OK)
	}

	pRetVal.Left = node.Bounds.X
	pRetVal.Top = node.Bounds.Y
	pRetVal.Width = node.Bounds.Width
	pRetVal.Height = node.Bounds.Height
	return uintptr(win32.S_OK)
}

func elemGetEmbeddedFragmentRoots(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

func elemSetFocus(this uintptr) uintptr {
	ep := elemFromFragment(this)
	// Trigger focus action via app.Send.
	if ep.bridge.send != nil {
		ep.bridge.mu.RLock()
		node := ep.bridge.tree.FindByID(ep.nodeID)
		ep.bridge.mu.RUnlock()
		if node != nil {
			for _, action := range node.Node.Actions {
				if action.Name == "focus" && action.Trigger != nil {
					ep.bridge.send(uiaFocusAction{trigger: action.Trigger})
					break
				}
			}
		}
	}
	return uintptr(win32.S_OK)
}

func elemGetFragmentRoot(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	ep := elemFromFragment(this)
	rp := ep.bridge.root
	atomic.AddInt32(&rp.refCount, 1)
	*pRetVal = unsafe.Pointer(&rp.vtblFragmentRoot)
	return uintptr(win32.S_OK)
}

// uiaFocusAction is sent via app.Send when UIA requests focus.
type uiaFocusAction struct {
	trigger func()
}

// --- SAFEARRAY helper ---

// safeArrayFromInt32s creates a SAFEARRAY of VT_I4 for runtime IDs.
func safeArrayFromInt32s(vals []int32) *win32.SAFEARRAY {
	sa := win32.SafeArrayCreateVector(win32.VT_I4, 0, uint32(len(vals)))
	if sa == nil {
		return nil
	}
	for i, v := range vals {
		idx := int32(i)
		win32.SafeArrayPutElement(sa, &idx, unsafe.Pointer(&v))
	}
	return sa
}
