//go:build windows && !nogui

package windows

import (
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

var (
	uiautomationcore = syscall.NewLazyDLL("uiautomationcore.dll")

	procUiaReturnRawElementProvider   = uiautomationcore.NewProc("UiaReturnRawElementProvider")
	procUiaRaiseAutomationEvent       = uiautomationcore.NewProc("UiaRaiseAutomationEvent")
	procUiaRaiseStructureChangedEvent = uiautomationcore.NewProc("UiaRaiseStructureChangedEvent")
	procUiaHostProviderFromHwnd       = uiautomationcore.NewProc("UiaHostProviderFromHwnd")
)

const (
	uiaRootObjectId = 0xFFFFFFE7 // -25 as uint32 (OBJID_NATIVEOM = -16, UiaRootObjectId = -25)
	wmGetObject     = 0x003D
)

// uiaReturnRawElementProvider wraps the UiaReturnRawElementProvider call.
// Returns the LRESULT to return from WM_GETOBJECT.
func uiaReturnRawElementProvider(hwnd, wParam, lParam uintptr, provider unsafe.Pointer) uintptr {
	ret, _, _ := procUiaReturnRawElementProvider.Call(hwnd, wParam, lParam, uintptr(provider))
	return ret
}

// uiaRaiseAutomationEvent raises a UIA event on the given provider.
func uiaRaiseAutomationEvent(provider unsafe.Pointer, eventID win32.UIA_EVENT_ID) win32.HRESULT {
	ret, _, _ := procUiaRaiseAutomationEvent.Call(uintptr(provider), uintptr(eventID))
	return win32.HRESULT(ret)
}

// uiaRaiseStructureChangedEvent raises a structure-changed event.
func uiaRaiseStructureChangedEvent(provider unsafe.Pointer, structureChangeType int32, runtimeID *int32, runtimeIDLen int) win32.HRESULT {
	ret, _, _ := procUiaRaiseStructureChangedEvent.Call(
		uintptr(provider),
		uintptr(structureChangeType),
		uintptr(unsafe.Pointer(runtimeID)),
		uintptr(runtimeIDLen),
	)
	return win32.HRESULT(ret)
}

// uiaHostProviderFromHwnd returns the host provider for the given HWND.
func uiaHostProviderFromHwnd(hwnd uintptr) (*win32.IRawElementProviderSimple, win32.HRESULT) {
	var provider *win32.IRawElementProviderSimple
	ret, _, _ := procUiaHostProviderFromHwnd.Call(hwnd, uintptr(unsafe.Pointer(&provider)))
	return provider, win32.HRESULT(ret)
}
