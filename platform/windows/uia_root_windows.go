//go:build windows && !nogui

package windows

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

// hresult converts an HRESULT (int32) to a uintptr for COM return values.
func hresult(hr win32.HRESULT) uintptr {
	return uintptr(uint32(hr))
}

// rootProvider implements IRawElementProviderSimple, IRawElementProviderFragment,
// and IRawElementProviderFragmentRoot for the root window element.
type rootProvider struct {
	refCount int32
	vtblSimple       *win32.IRawElementProviderSimpleVtbl
	vtblFragment     *win32.IRawElementProviderFragmentVtbl
	vtblFragmentRoot *win32.IRawElementProviderFragmentRootVtbl
	bridge           *UIABridge
}

var (
	rootSimpleVtbl       *win32.IRawElementProviderSimpleVtbl
	rootFragmentVtbl     *win32.IRawElementProviderFragmentVtbl
	rootFragmentRootVtbl *win32.IRawElementProviderFragmentRootVtbl
)

func initRootVtables() {
	if rootSimpleVtbl != nil {
		return
	}

	rootSimpleVtbl = &win32.IRawElementProviderSimpleVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(rootQueryInterface),
			AddRef:         syscall.NewCallback(rootAddRef),
			Release:        syscall.NewCallback(rootRelease),
		},
		Get_ProviderOptions:        syscall.NewCallback(rootGetProviderOptions),
		GetPatternProvider:         syscall.NewCallback(rootGetPatternProvider),
		GetPropertyValue:           syscall.NewCallback(rootGetPropertyValue),
		Get_HostRawElementProvider: syscall.NewCallback(rootGetHostProvider),
	}

	rootFragmentVtbl = &win32.IRawElementProviderFragmentVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(rootQueryInterface),
			AddRef:         syscall.NewCallback(rootAddRef),
			Release:        syscall.NewCallback(rootRelease),
		},
		Navigate:                 syscall.NewCallback(rootNavigate),
		GetRuntimeId:             syscall.NewCallback(rootGetRuntimeId),
		Get_BoundingRectangle:    syscall.NewCallback(rootGetBoundingRectangle),
		GetEmbeddedFragmentRoots: syscall.NewCallback(rootGetEmbeddedFragmentRoots),
		SetFocus:                 syscall.NewCallback(rootSetFocus),
		Get_FragmentRoot:         syscall.NewCallback(rootGetFragmentRoot),
	}

	rootFragmentRootVtbl = &win32.IRawElementProviderFragmentRootVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(rootQueryInterface),
			AddRef:         syscall.NewCallback(rootAddRef),
			Release:        syscall.NewCallback(rootRelease),
		},
		ElementProviderFromPoint: syscall.NewCallback(rootElementProviderFromPoint),
		GetFocus:                 syscall.NewCallback(rootGetFocusProvider),
	}
}

func newRootProvider(bridge *UIABridge) *rootProvider {
	initRootVtables()
	return &rootProvider{
		refCount:         1,
		vtblSimple:       rootSimpleVtbl,
		vtblFragment:     rootFragmentVtbl,
		vtblFragmentRoot: rootFragmentRootVtbl,
		bridge:           bridge,
	}
}

// simplePtr returns the IRawElementProviderSimple pointer for UIA.
func (p *rootProvider) simplePtr() unsafe.Pointer {
	return unsafe.Pointer(&p.vtblSimple)
}

// rootFromSimple recovers the rootProvider from a vtbl pointer.
func rootFromSimple(this uintptr) *rootProvider {
	return (*rootProvider)(unsafe.Pointer(this - unsafe.Offsetof(rootProvider{}.vtblSimple)))
}

func rootFromFragment(this uintptr) *rootProvider {
	return (*rootProvider)(unsafe.Pointer(this - unsafe.Offsetof(rootProvider{}.vtblFragment)))
}

func rootFromFragmentRoot(this uintptr) *rootProvider {
	return (*rootProvider)(unsafe.Pointer(this - unsafe.Offsetof(rootProvider{}.vtblFragmentRoot)))
}

// --- IUnknown ---

func rootQueryInterface(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	// Determine which interface pointer was called (Simple, Fragment, or FragmentRoot).
	// We use the vtblSimple as the canonical "this" for offset calculation.
	// However, the call could come via any of the three vtable pointers.
	// We need to figure out which rootProvider this belongs to.
	// Since all three vtables have the same QI callback, we check which vtable
	// the 'this' pointer points to.
	vptr := *(*uintptr)(unsafe.Pointer(this))

	var rp *rootProvider
	switch vptr {
	case uintptr(unsafe.Pointer(rootSimpleVtbl)):
		rp = rootFromSimple(this)
	case uintptr(unsafe.Pointer(rootFragmentVtbl)):
		rp = rootFromFragment(this)
	case uintptr(unsafe.Pointer(rootFragmentRootVtbl)):
		rp = rootFromFragmentRoot(this)
	default:
		*ppv = nil
		return hresult(win32.E_NOINTERFACE)
	}

	if *riid == win32.IID_IUnknown || *riid == win32.IID_IRawElementProviderSimple {
		*ppv = unsafe.Pointer(&rp.vtblSimple)
		atomic.AddInt32(&rp.refCount, 1)
		return uintptr(win32.S_OK)
	}
	if *riid == win32.IID_IRawElementProviderFragment {
		*ppv = unsafe.Pointer(&rp.vtblFragment)
		atomic.AddInt32(&rp.refCount, 1)
		return uintptr(win32.S_OK)
	}
	if *riid == win32.IID_IRawElementProviderFragmentRoot {
		*ppv = unsafe.Pointer(&rp.vtblFragmentRoot)
		atomic.AddInt32(&rp.refCount, 1)
		return uintptr(win32.S_OK)
	}

	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func rootAddRef(this uintptr) uintptr {
	vptr := *(*uintptr)(unsafe.Pointer(this))
	var rp *rootProvider
	switch vptr {
	case uintptr(unsafe.Pointer(rootSimpleVtbl)):
		rp = rootFromSimple(this)
	case uintptr(unsafe.Pointer(rootFragmentVtbl)):
		rp = rootFromFragment(this)
	case uintptr(unsafe.Pointer(rootFragmentRootVtbl)):
		rp = rootFromFragmentRoot(this)
	default:
		return 0
	}
	return uintptr(atomic.AddInt32(&rp.refCount, 1))
}

func rootRelease(this uintptr) uintptr {
	vptr := *(*uintptr)(unsafe.Pointer(this))
	var rp *rootProvider
	switch vptr {
	case uintptr(unsafe.Pointer(rootSimpleVtbl)):
		rp = rootFromSimple(this)
	case uintptr(unsafe.Pointer(rootFragmentVtbl)):
		rp = rootFromFragment(this)
	case uintptr(unsafe.Pointer(rootFragmentRootVtbl)):
		rp = rootFromFragmentRoot(this)
	default:
		return 0
	}
	ref := atomic.AddInt32(&rp.refCount, -1)
	return uintptr(ref)
}

// --- IRawElementProviderSimple ---

func rootGetProviderOptions(this uintptr, pRetVal *win32.ProviderOptions) uintptr {
	*pRetVal = win32.ProviderOptions_ServerSideProvider | win32.ProviderOptions_UseComThreading
	return uintptr(win32.S_OK)
}

func rootGetPatternProvider(this uintptr, patternID win32.UIA_PATTERN_ID, pRetVal *unsafe.Pointer) uintptr {
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

func rootGetPropertyValue(this uintptr, propertyID win32.UIA_PROPERTY_ID, pRetVal *win32.VARIANT) uintptr {
	rp := rootFromSimple(this)
	switch propertyID {
	case win32.UIA_ControlTypePropertyId:
		*pRetVal = variantInt32(int32(win32.UIA_WindowControlTypeId))
	case win32.UIA_NamePropertyId:
		*pRetVal = variantString("Lux Application")
	case win32.UIA_IsKeyboardFocusablePropertyId:
		*pRetVal = variantBool(true)
	case win32.UIA_HasKeyboardFocusPropertyId:
		*pRetVal = variantBool(true)
	case win32.UIA_AutomationIdPropertyId:
		*pRetVal = variantString(fmt.Sprintf("lux_root_%d", rp.bridge.hwnd))
	default:
		*pRetVal = variantEmpty()
	}
	return uintptr(win32.S_OK)
}

func rootGetHostProvider(this uintptr, pRetVal **win32.IRawElementProviderSimple) uintptr {
	rp := rootFromSimple(this)
	host, hr := uiaHostProviderFromHwnd(rp.bridge.hwnd)
	if win32.SUCCEEDED(hr) {
		*pRetVal = host
	} else {
		*pRetVal = nil
	}
	return uintptr(win32.S_OK)
}

// --- IRawElementProviderFragment ---

func rootNavigate(this uintptr, direction win32.NavigateDirection, pRetVal **win32.IRawElementProviderFragment) uintptr {
	rp := rootFromFragment(this)
	*pRetVal = nil

	rp.bridge.mu.RLock()
	defer rp.bridge.mu.RUnlock()

	tree := &rp.bridge.tree
	if len(tree.Nodes) == 0 {
		return uintptr(win32.S_OK)
	}

	switch direction {
	case win32.NavigateDirection_FirstChild:
		root := tree.Root()
		if root != nil && root.FirstChild >= 0 {
			ep := rp.bridge.providerFor(tree.Nodes[root.FirstChild].ID)
			if ep != nil {
				atomic.AddInt32(&ep.refCount, 1)
				*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&ep.vtblFragment))
			}
		}
	case win32.NavigateDirection_LastChild:
		root := tree.Root()
		if root != nil {
			children := tree.Children(root)
			if len(children) > 0 {
				last := children[len(children)-1]
				ep := rp.bridge.providerFor(last.ID)
				if ep != nil {
					atomic.AddInt32(&ep.refCount, 1)
					*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&ep.vtblFragment))
				}
			}
		}
	}
	return uintptr(win32.S_OK)
}

func rootGetRuntimeId(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	// Root element returns NULL runtime ID — UIA uses HWND-based identification.
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

func rootGetBoundingRectangle(this uintptr, pRetVal *win32.UiaRect) uintptr {
	rp := rootFromFragment(this)
	var r rect
	procGetWindowRect.Call(rp.bridge.hwnd, uintptr(unsafe.Pointer(&r)))
	pRetVal.Left = float64(r.Left)
	pRetVal.Top = float64(r.Top)
	pRetVal.Width = float64(r.Right - r.Left)
	pRetVal.Height = float64(r.Bottom - r.Top)
	return uintptr(win32.S_OK)
}

func rootGetEmbeddedFragmentRoots(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

func rootSetFocus(this uintptr) uintptr {
	rp := rootFromFragment(this)
	// Bring window to front.
	procSetFocus.Call(rp.bridge.hwnd)
	return uintptr(win32.S_OK)
}

func rootGetFragmentRoot(this uintptr, pRetVal *unsafe.Pointer) uintptr {
	rp := rootFromFragment(this)
	// Root's fragment root is itself.
	atomic.AddInt32(&rp.refCount, 1)
	*pRetVal = unsafe.Pointer(&rp.vtblFragmentRoot)
	return uintptr(win32.S_OK)
}

// --- IRawElementProviderFragmentRoot ---

// rootElementProviderFromPoint receives float64 x,y as pairs of uintptr
// because syscall.NewCallback does not support float arguments.
// On amd64, each float64 occupies one 8-byte register passed as uintptr.
func rootElementProviderFromPoint(this, xBits, yBits uintptr, pRetVal **win32.IRawElementProviderFragment) uintptr {
	// For now, return nil — UIA will fall back to the root.
	// A full implementation would hit-test the access tree.
	*pRetVal = nil
	return uintptr(win32.S_OK)
}

func rootGetFocusProvider(this uintptr, pRetVal **win32.IRawElementProviderFragment) uintptr {
	rp := rootFromFragmentRoot(this)
	*pRetVal = nil

	rp.bridge.mu.RLock()
	defer rp.bridge.mu.RUnlock()

	if rp.bridge.tree.FocusedID != 0 {
		ep := rp.bridge.providerFor(rp.bridge.tree.FocusedID)
		if ep != nil {
			atomic.AddInt32(&ep.refCount, 1)
			*pRetVal = (*win32.IRawElementProviderFragment)(unsafe.Pointer(&ep.vtblFragment))
		}
	}
	return uintptr(win32.S_OK)
}

var procSetFocus = user32.NewProc("SetFocus")
