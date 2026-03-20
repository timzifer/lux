//go:build windows && !nogui

package windows

import (
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/a11y"
	"github.com/zzl/go-win32api/v2/win32"
)

// --- Pattern provider types ---

// invokeProvider implements IInvokeProvider for buttons.
type invokeProvider struct {
	refCount int32
	vtbl     *win32.IInvokeProviderVtbl
	ep       *elementProvider
}

// toggleProvider implements IToggleProvider for checkboxes/toggles.
type toggleProvider struct {
	refCount int32
	vtbl     *win32.IToggleProviderVtbl
	ep       *elementProvider
}

// valueProvider implements IValueProvider for text inputs.
type valueProvider struct {
	refCount int32
	vtbl     *win32.IValueProviderVtbl
	ep       *elementProvider
}

// rangeValueProvider implements IRangeValueProvider for sliders.
type rangeValueProvider struct {
	refCount int32
	vtbl     *win32.IRangeValueProviderVtbl
	ep       *elementProvider
}

// expandCollapseProvider implements IExpandCollapseProvider for comboboxes.
type expandCollapseProvider struct {
	refCount int32
	vtbl     *win32.IExpandCollapseProviderVtbl
	ep       *elementProvider
}

// --- Vtable singletons ---

var (
	invokeVtbl         *win32.IInvokeProviderVtbl
	toggleVtbl         *win32.IToggleProviderVtbl
	valueVtbl          *win32.IValueProviderVtbl
	rangeValueVtbl     *win32.IRangeValueProviderVtbl
	expandCollapseVtbl *win32.IExpandCollapseProviderVtbl
)

func initPatternVtables() {
	if invokeVtbl != nil {
		return
	}

	invokeVtbl = &win32.IInvokeProviderVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(invokeQI),
			AddRef:         syscall.NewCallback(invokeAddRef),
			Release:        syscall.NewCallback(invokeReleaseRef),
		},
		Invoke: syscall.NewCallback(invokeInvoke),
	}

	toggleVtbl = &win32.IToggleProviderVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(toggleQI),
			AddRef:         syscall.NewCallback(toggleAddRef),
			Release:        syscall.NewCallback(toggleReleaseRef),
		},
		Toggle:          syscall.NewCallback(toggleToggle),
		Get_ToggleState: syscall.NewCallback(toggleGetState),
	}

	valueVtbl = &win32.IValueProviderVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(valueQI),
			AddRef:         syscall.NewCallback(valueAddRef),
			Release:        syscall.NewCallback(valueReleaseRef),
		},
		SetValue:       syscall.NewCallback(valueSetValue),
		Get_Value:      syscall.NewCallback(valueGetValue),
		Get_IsReadOnly: syscall.NewCallback(valueGetIsReadOnly),
	}

	rangeValueVtbl = &win32.IRangeValueProviderVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(rangeQI),
			AddRef:         syscall.NewCallback(rangeAddRef),
			Release:        syscall.NewCallback(rangeReleaseRef),
		},
		SetValue:        syscall.NewCallback(rangeSetValue),
		Get_Value:       syscall.NewCallback(rangeGetValue),
		Get_IsReadOnly:  syscall.NewCallback(rangeGetIsReadOnly),
		Get_Maximum:     syscall.NewCallback(rangeGetMaximum),
		Get_Minimum:     syscall.NewCallback(rangeGetMinimum),
		Get_LargeChange: syscall.NewCallback(rangeGetLargeChange),
		Get_SmallChange: syscall.NewCallback(rangeGetSmallChange),
	}

	expandCollapseVtbl = &win32.IExpandCollapseProviderVtbl{
		IUnknownVtbl: win32.IUnknownVtbl{
			QueryInterface: syscall.NewCallback(ecQI),
			AddRef:         syscall.NewCallback(ecAddRef),
			Release:        syscall.NewCallback(ecReleaseRef),
		},
		Expand:                   syscall.NewCallback(ecExpand),
		Collapse:                 syscall.NewCallback(ecCollapse),
		Get_ExpandCollapseState:  syscall.NewCallback(ecGetState),
	}
}

// --- Factory helpers ---

func newInvokeProvider(ep *elementProvider) *invokeProvider {
	initPatternVtables()
	return &invokeProvider{refCount: 1, vtbl: invokeVtbl, ep: ep}
}

func newToggleProvider(ep *elementProvider) *toggleProvider {
	initPatternVtables()
	return &toggleProvider{refCount: 1, vtbl: toggleVtbl, ep: ep}
}

func newValueProvider(ep *elementProvider) *valueProvider {
	initPatternVtables()
	return &valueProvider{refCount: 1, vtbl: valueVtbl, ep: ep}
}

func newRangeValueProvider(ep *elementProvider) *rangeValueProvider {
	initPatternVtables()
	return &rangeValueProvider{refCount: 1, vtbl: rangeValueVtbl, ep: ep}
}

func newExpandCollapseProvider(ep *elementProvider) *expandCollapseProvider {
	initPatternVtables()
	return &expandCollapseProvider{refCount: 1, vtbl: expandCollapseVtbl, ep: ep}
}

// getPatternProvider returns the pattern provider for a node, or nil.
func getPatternProvider(ep *elementProvider, node *a11y.AccessTreeNode, patternID win32.UIA_PATTERN_ID) unsafe.Pointer {
	patterns := patternsForRole(node.Node.Role)
	for _, p := range patterns {
		if p == patternID {
			switch patternID {
			case win32.UIA_InvokePatternId:
				ip := newInvokeProvider(ep)
				return unsafe.Pointer(&ip.vtbl)
			case win32.UIA_TogglePatternId:
				tp := newToggleProvider(ep)
				return unsafe.Pointer(&tp.vtbl)
			case win32.UIA_ValuePatternId:
				vp := newValueProvider(ep)
				return unsafe.Pointer(&vp.vtbl)
			case win32.UIA_RangeValuePatternId:
				rp := newRangeValueProvider(ep)
				return unsafe.Pointer(&rp.vtbl)
			case win32.UIA_ExpandCollapsePatternId:
				ec := newExpandCollapseProvider(ep)
				return unsafe.Pointer(&ec.vtbl)
			}
		}
	}
	return nil
}

// getPatternProviderForIID checks if a QI for a pattern interface should succeed.
func getPatternProviderForIID(ep *elementProvider, node *a11y.AccessTreeNode, riid *syscall.GUID) unsafe.Pointer {
	patterns := patternsForRole(node.Node.Role)
	for _, p := range patterns {
		switch p {
		case win32.UIA_InvokePatternId:
			if *riid == win32.IID_IInvokeProvider {
				ip := newInvokeProvider(ep)
				return unsafe.Pointer(&ip.vtbl)
			}
		case win32.UIA_TogglePatternId:
			if *riid == win32.IID_IToggleProvider {
				tp := newToggleProvider(ep)
				return unsafe.Pointer(&tp.vtbl)
			}
		case win32.UIA_ValuePatternId:
			if *riid == win32.IID_IValueProvider {
				vp := newValueProvider(ep)
				return unsafe.Pointer(&vp.vtbl)
			}
		case win32.UIA_RangeValuePatternId:
			if *riid == win32.IID_IRangeValueProvider {
				rp := newRangeValueProvider(ep)
				return unsafe.Pointer(&rp.vtbl)
			}
		case win32.UIA_ExpandCollapsePatternId:
			if *riid == win32.IID_IExpandCollapseProvider {
				ec := newExpandCollapseProvider(ep)
				return unsafe.Pointer(&ec.vtbl)
			}
		}
	}
	return nil
}

// --- Helper to find and trigger actions ---

func findAction(ep *elementProvider, name string) *a11y.AccessAction {
	ep.bridge.mu.RLock()
	node := ep.bridge.tree.FindByID(ep.nodeID)
	ep.bridge.mu.RUnlock()
	if node == nil {
		return nil
	}
	for i := range node.Node.Actions {
		if node.Node.Actions[i].Name == name {
			return &node.Node.Actions[i]
		}
	}
	return nil
}

func triggerAction(ep *elementProvider, name string) {
	action := findAction(ep, name)
	if action != nil && action.Trigger != nil && ep.bridge.send != nil {
		trigger := action.Trigger
		ep.bridge.send(uiaActionMsg{trigger: trigger})
	}
}

type uiaActionMsg struct {
	trigger func()
}

// --- IInvokeProvider ---

func invokeQI(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IInvokeProvider {
		*ppv = unsafe.Pointer(this)
		ip := (*invokeProvider)(unsafe.Pointer(this - unsafe.Offsetof(invokeProvider{}.vtbl)))
		atomic.AddInt32(&ip.refCount, 1)
		return uintptr(win32.S_OK)
	}
	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func invokeAddRef(this uintptr) uintptr {
	ip := (*invokeProvider)(unsafe.Pointer(this - unsafe.Offsetof(invokeProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&ip.refCount, 1))
}

func invokeReleaseRef(this uintptr) uintptr {
	ip := (*invokeProvider)(unsafe.Pointer(this - unsafe.Offsetof(invokeProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&ip.refCount, -1))
}

func invokeInvoke(this uintptr) uintptr {
	ip := (*invokeProvider)(unsafe.Pointer(this - unsafe.Offsetof(invokeProvider{}.vtbl)))
	triggerAction(ip.ep, "activate")
	return uintptr(win32.S_OK)
}

// --- IToggleProvider ---

func toggleQI(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IToggleProvider {
		*ppv = unsafe.Pointer(this)
		tp := (*toggleProvider)(unsafe.Pointer(this - unsafe.Offsetof(toggleProvider{}.vtbl)))
		atomic.AddInt32(&tp.refCount, 1)
		return uintptr(win32.S_OK)
	}
	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func toggleAddRef(this uintptr) uintptr {
	tp := (*toggleProvider)(unsafe.Pointer(this - unsafe.Offsetof(toggleProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&tp.refCount, 1))
}

func toggleReleaseRef(this uintptr) uintptr {
	tp := (*toggleProvider)(unsafe.Pointer(this - unsafe.Offsetof(toggleProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&tp.refCount, -1))
}

func toggleToggle(this uintptr) uintptr {
	tp := (*toggleProvider)(unsafe.Pointer(this - unsafe.Offsetof(toggleProvider{}.vtbl)))
	triggerAction(tp.ep, "activate")
	return uintptr(win32.S_OK)
}

func toggleGetState(this uintptr, pRetVal *win32.ToggleState) uintptr {
	tp := (*toggleProvider)(unsafe.Pointer(this - unsafe.Offsetof(toggleProvider{}.vtbl)))
	tp.ep.bridge.mu.RLock()
	node := tp.ep.bridge.tree.FindByID(tp.ep.nodeID)
	tp.ep.bridge.mu.RUnlock()
	if node != nil && node.Node.States.Checked {
		*pRetVal = win32.ToggleState_On
	} else {
		*pRetVal = win32.ToggleState_Off
	}
	return uintptr(win32.S_OK)
}

// --- IValueProvider ---

func valueQI(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IValueProvider {
		*ppv = unsafe.Pointer(this)
		vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
		atomic.AddInt32(&vp.refCount, 1)
		return uintptr(win32.S_OK)
	}
	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func valueAddRef(this uintptr) uintptr {
	vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&vp.refCount, 1))
}

func valueReleaseRef(this uintptr) uintptr {
	vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&vp.refCount, -1))
}

func valueSetValue(this uintptr, val win32.PWSTR) uintptr {
	vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
	_ = vp // TODO: route setValue to widget via send
	return uintptr(win32.S_OK)
}

func valueGetValue(this uintptr, pRetVal *win32.BSTR) uintptr {
	vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
	vp.ep.bridge.mu.RLock()
	node := vp.ep.bridge.tree.FindByID(vp.ep.nodeID)
	vp.ep.bridge.mu.RUnlock()
	if node != nil {
		*pRetVal = bstrFromString(node.Node.Value)
	} else {
		*pRetVal = bstrFromString("")
	}
	return uintptr(win32.S_OK)
}

func valueGetIsReadOnly(this uintptr, pRetVal *win32.BOOL) uintptr {
	vp := (*valueProvider)(unsafe.Pointer(this - unsafe.Offsetof(valueProvider{}.vtbl)))
	vp.ep.bridge.mu.RLock()
	node := vp.ep.bridge.tree.FindByID(vp.ep.nodeID)
	vp.ep.bridge.mu.RUnlock()
	if node != nil && node.Node.States.ReadOnly {
		*pRetVal = win32.TRUE
	} else {
		*pRetVal = win32.FALSE
	}
	return uintptr(win32.S_OK)
}

// --- IRangeValueProvider ---

func rangeQI(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IRangeValueProvider {
		*ppv = unsafe.Pointer(this)
		rp := (*rangeValueProvider)(unsafe.Pointer(this - unsafe.Offsetof(rangeValueProvider{}.vtbl)))
		atomic.AddInt32(&rp.refCount, 1)
		return uintptr(win32.S_OK)
	}
	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func rangeAddRef(this uintptr) uintptr {
	rp := (*rangeValueProvider)(unsafe.Pointer(this - unsafe.Offsetof(rangeValueProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&rp.refCount, 1))
}

func rangeReleaseRef(this uintptr) uintptr {
	rp := (*rangeValueProvider)(unsafe.Pointer(this - unsafe.Offsetof(rangeValueProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&rp.refCount, -1))
}

// rangeSetValue receives float64 as uintptr because syscall.NewCallback
// does not support float arguments on Windows.
func rangeSetValue(this uintptr, valBits uintptr) uintptr {
	rp := (*rangeValueProvider)(unsafe.Pointer(this - unsafe.Offsetof(rangeValueProvider{}.vtbl)))
	_ = rp // TODO: route setValue
	return uintptr(win32.S_OK)
}

func rangeGetValue(this uintptr, pRetVal *float64) uintptr {
	*pRetVal = 0
	return uintptr(win32.S_OK)
}

func rangeGetIsReadOnly(this uintptr, pRetVal *win32.BOOL) uintptr {
	*pRetVal = win32.FALSE
	return uintptr(win32.S_OK)
}

func rangeGetMaximum(this uintptr, pRetVal *float64) uintptr {
	*pRetVal = 100
	return uintptr(win32.S_OK)
}

func rangeGetMinimum(this uintptr, pRetVal *float64) uintptr {
	*pRetVal = 0
	return uintptr(win32.S_OK)
}

func rangeGetLargeChange(this uintptr, pRetVal *float64) uintptr {
	*pRetVal = 10
	return uintptr(win32.S_OK)
}

func rangeGetSmallChange(this uintptr, pRetVal *float64) uintptr {
	*pRetVal = 1
	return uintptr(win32.S_OK)
}

// --- IExpandCollapseProvider ---

func ecQI(this uintptr, riid *syscall.GUID, ppv *unsafe.Pointer) uintptr {
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IExpandCollapseProvider {
		*ppv = unsafe.Pointer(this)
		ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
		atomic.AddInt32(&ec.refCount, 1)
		return uintptr(win32.S_OK)
	}
	*ppv = nil
	return hresult(win32.E_NOINTERFACE)
}

func ecAddRef(this uintptr) uintptr {
	ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&ec.refCount, 1))
}

func ecReleaseRef(this uintptr) uintptr {
	ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
	return uintptr(atomic.AddInt32(&ec.refCount, -1))
}

func ecExpand(this uintptr) uintptr {
	ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
	triggerAction(ec.ep, "expand")
	return uintptr(win32.S_OK)
}

func ecCollapse(this uintptr) uintptr {
	ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
	triggerAction(ec.ep, "collapse")
	return uintptr(win32.S_OK)
}

func ecGetState(this uintptr, pRetVal *win32.ExpandCollapseState) uintptr {
	ec := (*expandCollapseProvider)(unsafe.Pointer(this - unsafe.Offsetof(expandCollapseProvider{}.vtbl)))
	ec.ep.bridge.mu.RLock()
	node := ec.ep.bridge.tree.FindByID(ec.ep.nodeID)
	ec.ep.bridge.mu.RUnlock()
	if node != nil && node.Node.States.Expanded {
		*pRetVal = win32.ExpandCollapseState_Expanded
	} else {
		*pRetVal = win32.ExpandCollapseState_Collapsed
	}
	return uintptr(win32.S_OK)
}
