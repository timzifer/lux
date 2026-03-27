//go:build darwin && cocoa && !nogui && arm64

package cocoa

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
	"github.com/timzifer/lux/platform"
)

// NSAlert style constants.
const (
	nsAlertStyleInformational uint64 = 1
	nsAlertStyleWarning       uint64 = 0
	nsAlertStyleCritical      uint64 = 2
)

// ShowMessageDialog displays an NSAlert message dialog via FFI.
// Dispatches to the main thread because NSAlert creates an NSPanel internally.
func (p *Platform) ShowMessageDialog(title, message string, kind platform.DialogKind) error {
	var err error
	p.RunOnMainThread(func() {
		pool := newAutoreleasePool()
		defer drainPool(pool)

		nsAlert := getClass("NSAlert")
		alert := msgSendPtr(nsAlert, sel("new"))
		defer msgSendVoid(alert, sel("release"))

		nsTitle := newNSString(title)
		defer msgSendVoid(nsTitle, sel("release"))
		msgSendVoid(alert, sel("setMessageText:"), argPtr(nsTitle))

		nsMsg := newNSString(message)
		defer msgSendVoid(nsMsg, sel("release"))
		msgSendVoid(alert, sel("setInformativeText:"), argPtr(nsMsg))

		var style uint64
		switch kind {
		case 1: // Warning
			style = nsAlertStyleWarning
		case 2: // Error
			style = nsAlertStyleCritical
		default: // Info
			style = nsAlertStyleInformational
		}
		msgSendVoid(alert, sel("setAlertStyle:"), argUInt64(style))

		msgSendPtr(alert, sel("runModal"))
	})
	return err
}

// ShowConfirmDialog displays an NSAlert with Confirm/Cancel buttons via FFI.
func (p *Platform) ShowConfirmDialog(title, message string) (bool, error) {
	var confirmed bool
	var err error
	p.RunOnMainThread(func() {
		pool := newAutoreleasePool()
		defer drainPool(pool)

		nsAlert := getClass("NSAlert")
		alert := msgSendPtr(nsAlert, sel("new"))
		defer msgSendVoid(alert, sel("release"))

		nsTitle := newNSString(title)
		defer msgSendVoid(nsTitle, sel("release"))
		msgSendVoid(alert, sel("setMessageText:"), argPtr(nsTitle))

		nsMsg := newNSString(message)
		defer msgSendVoid(nsMsg, sel("release"))
		msgSendVoid(alert, sel("setInformativeText:"), argPtr(nsMsg))

		confirmStr := newNSString("Confirm")
		defer msgSendVoid(confirmStr, sel("release"))
		cancelStr := newNSString("Cancel")
		defer msgSendVoid(cancelStr, sel("release"))

		msgSendVoid(alert, sel("addButtonWithTitle:"), argPtr(confirmStr))
		msgSendVoid(alert, sel("addButtonWithTitle:"), argPtr(cancelStr))
		msgSendVoid(alert, sel("setAlertStyle:"), argUInt64(nsAlertStyleInformational))

		response := msgSendInt64(alert, sel("runModal"))
		confirmed = response == 1000
	})
	return confirmed, err
}

// ShowInputDialog displays an NSAlert with a text field accessory view via FFI.
func (p *Platform) ShowInputDialog(title, message, defaultValue string) (string, bool, error) {
	var value string
	var confirmed bool
	var err error
	p.RunOnMainThread(func() {
		pool := newAutoreleasePool()
		defer drainPool(pool)

		nsAlert := getClass("NSAlert")
		alert := msgSendPtr(nsAlert, sel("new"))
		defer msgSendVoid(alert, sel("release"))

		nsTitle := newNSString(title)
		defer msgSendVoid(nsTitle, sel("release"))
		msgSendVoid(alert, sel("setMessageText:"), argPtr(nsTitle))

		nsMsg := newNSString(message)
		defer msgSendVoid(nsMsg, sel("release"))
		msgSendVoid(alert, sel("setInformativeText:"), argPtr(nsMsg))

		okStr := newNSString("OK")
		defer msgSendVoid(okStr, sel("release"))
		cancelStr := newNSString("Cancel")
		defer msgSendVoid(cancelStr, sel("release"))

		msgSendVoid(alert, sel("addButtonWithTitle:"), argPtr(okStr))
		msgSendVoid(alert, sel("addButtonWithTitle:"), argPtr(cancelStr))

		nsTextField := getClass("NSTextField")
		inputField := msgSendPtr(nsTextField, sel("alloc"))
		frame := nsRect{Size: nsSize{Width: 300, Height: 24}}
		inputField = msgSendPtr(inputField, sel("initWithFrame:"), argRect(frame))
		defer msgSendVoid(inputField, sel("release"))

		nsDefault := newNSString(defaultValue)
		defer msgSendVoid(nsDefault, sel("release"))
		msgSendVoid(inputField, sel("setStringValue:"), argPtr(nsDefault))

		msgSendVoid(alert, sel("setAccessoryView:"), argPtr(inputField))

		response := msgSendInt64(alert, sel("runModal"))
		if response == 1000 {
			nsValue := msgSendPtr(inputField, sel("stringValue"))
			value = goString(nsValue)
			confirmed = true
		}
	})
	return value, confirmed, err
}

// msgSendInt64 calls objc_msgSend returning int64.
func msgSendInt64(self, cmd uintptr, args ...objcArg) int64 {
	var result int64
	msgSend(types.SInt64TypeDescriptor, unsafe.Pointer(&result), self, cmd, args...)
	return result
}
