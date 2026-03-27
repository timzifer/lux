//go:build windows && !nogui

package windows

import (
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

// variantEmpty returns a VT_EMPTY VARIANT.
func variantEmpty() win32.VARIANT {
	var v win32.VARIANT
	v.Vt = win32.VT_EMPTY
	return v
}

// variantInt32 returns a VT_I4 VARIANT with the given value.
func variantInt32(val int32) win32.VARIANT {
	var v win32.VARIANT
	v.Vt = win32.VT_I4
	*v.LVal() = val
	return v
}

// variantBool returns a VT_BOOL VARIANT.
func variantBool(val bool) win32.VARIANT {
	var v win32.VARIANT
	v.Vt = win32.VT_BOOL
	if val {
		*v.BoolVal() = win32.VARIANT_TRUE
	} else {
		*v.BoolVal() = win32.VARIANT_FALSE
	}
	return v
}

// variantString returns a VT_BSTR VARIANT with the given string.
// The caller must free the BSTR when done (via SysFreeString).
func variantString(s string) win32.VARIANT {
	var v win32.VARIANT
	v.Vt = win32.VT_BSTR
	bstr := win32.SysAllocString(win32.StrToPwstr(s))
	*v.BstrVal() = bstr
	return v
}

// variantFloat64 returns a VT_R8 VARIANT with the given value.
func variantFloat64(val float64) win32.VARIANT {
	var v win32.VARIANT
	v.Vt = win32.VT_R8
	*v.DblVal() = val
	return v
}

// bstrFromString allocates a BSTR from a Go string.
func bstrFromString(s string) win32.BSTR {
	utf16, _ := syscall.UTF16FromString(s)
	return win32.SysAllocString(win32.PWSTR(unsafe.Pointer(&utf16[0])))
}
