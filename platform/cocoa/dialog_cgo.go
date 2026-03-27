//go:build darwin && cocoa && !nogui && !arm64

package cocoa

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

static int luxShowMessageDialog(const char *title, const char *message, int kind) {
	__block int result = 0;
	dispatch_sync(dispatch_get_main_queue(), ^{
		NSAlert *alert = [[NSAlert alloc] init];
		[alert setMessageText:[NSString stringWithUTF8String:title]];
		[alert setInformativeText:[NSString stringWithUTF8String:message]];

		switch (kind) {
		case 1: // Warning
			[alert setAlertStyle:NSAlertStyleWarning];
			break;
		case 2: // Error
			[alert setAlertStyle:NSAlertStyleCritical];
			break;
		default: // Info
			[alert setAlertStyle:NSAlertStyleInformational];
			break;
		}

		[alert runModal];
	});
	return result;
}

static int luxShowConfirmDialog(const char *title, const char *message) {
	__block int result = 0;
	dispatch_sync(dispatch_get_main_queue(), ^{
		NSAlert *alert = [[NSAlert alloc] init];
		[alert setMessageText:[NSString stringWithUTF8String:title]];
		[alert setInformativeText:[NSString stringWithUTF8String:message]];
		[alert addButtonWithTitle:@"Confirm"];
		[alert addButtonWithTitle:@"Cancel"];
		[alert setAlertStyle:NSAlertStyleInformational];

		NSModalResponse response = [alert runModal];
		result = (response == NSAlertFirstButtonReturn) ? 1 : 0;
	});
	return result;
}

typedef struct {
	int confirmed;
	char value[1024];
} InputResult;

static InputResult luxShowInputDialog(const char *title, const char *message, const char *defaultValue) {
	__block InputResult result = {0, ""};
	dispatch_sync(dispatch_get_main_queue(), ^{
		NSAlert *alert = [[NSAlert alloc] init];
		[alert setMessageText:[NSString stringWithUTF8String:title]];
		[alert setInformativeText:[NSString stringWithUTF8String:message]];
		[alert addButtonWithTitle:@"OK"];
		[alert addButtonWithTitle:@"Cancel"];

		NSTextField *input = [[NSTextField alloc] initWithFrame:NSMakeRect(0, 0, 300, 24)];
		[input setStringValue:[NSString stringWithUTF8String:defaultValue]];
		[alert setAccessoryView:input];

		NSModalResponse response = [alert runModal];
		if (response == NSAlertFirstButtonReturn) {
			result.confirmed = 1;
			const char *text = [[input stringValue] UTF8String];
			if (text) {
				strncpy(result.value, text, sizeof(result.value) - 1);
				result.value[sizeof(result.value) - 1] = '\0';
			}
		}
	});
	return result;
}
*/
import "C"
import (
	"unsafe"

	"github.com/timzifer/lux/platform"
)

// ShowMessageDialog displays an NSAlert message dialog.
func (p *Platform) ShowMessageDialog(title, message string, kind platform.DialogKind) error {
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))

	C.luxShowMessageDialog(ct, cm, C.int(kind))
	return nil
}

// ShowConfirmDialog displays an NSAlert with Confirm/Cancel buttons.
func (p *Platform) ShowConfirmDialog(title, message string) (bool, error) {
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))

	result := C.luxShowConfirmDialog(ct, cm)
	return result == 1, nil
}

// ShowInputDialog displays an NSAlert with a text field accessory view.
func (p *Platform) ShowInputDialog(title, message, defaultValue string) (string, bool, error) {
	ct := C.CString(title)
	cm := C.CString(message)
	cd := C.CString(defaultValue)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))
	defer C.free(unsafe.Pointer(cd))

	result := C.luxShowInputDialog(ct, cm, cd)
	if result.confirmed == 1 {
		return C.GoString(&result.value[0]), true, nil
	}
	return "", false, nil
}
