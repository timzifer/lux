//go:build windows && !nogui

package windows

import (
	"syscall"
	"unsafe"

	"github.com/timzifer/lux/platform"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

const (
	mbOK             = 0x00000000
	mbYesNo          = 0x00000004
	mbIconInfo       = 0x00000040
	mbIconWarning    = 0x00000030
	mbIconError      = 0x00000010
	idYes            = 6
)

// ShowMessageDialog displays a Win32 MessageBox.
func (p *Platform) ShowMessageDialog(title, message string, kind platform.DialogKind) error {
	flags := mbOK
	switch kind {
	case platform.DialogWarning:
		flags |= mbIconWarning
	case platform.DialogError:
		flags |= mbIconError
	default:
		flags |= mbIconInfo
	}

	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)

	procMessageBox.Call(
		p.hwnd,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(flags),
	)
	return nil
}

// ShowConfirmDialog displays a Yes/No MessageBox.
func (p *Platform) ShowConfirmDialog(title, message string) (bool, error) {
	flags := mbYesNo | mbIconInfo

	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)

	ret, _, _ := procMessageBox.Call(
		p.hwnd,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(flags),
	)
	return ret == idYes, nil
}

// ShowInputDialog is not natively supported on Windows without a custom dialog.
// Returns an error to trigger the framework fallback.
func (p *Platform) ShowInputDialog(title, message, defaultValue string) (string, bool, error) {
	return "", false, &dialogError{"native input dialog not supported on Windows; use fallback"}
}

type dialogError struct{ msg string }

func (e *dialogError) Error() string { return e.msg }
