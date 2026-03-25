// Package dialog provides fallback dialog elements rendered as overlays.
//
// These are used when the platform does not implement NativeDialogProvider,
// or when native dialogs fail. Each function returns an Overlay element
// with Backdrop: true and PlacementCenter positioning.
package dialog

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/platform"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
)

// MessageDialog returns an overlay element displaying a message with an OK button.
func MessageDialog(id ui.OverlayID, title, message string, kind platform.DialogKind, onClose func()) ui.Element {
	return ui.Overlay{
		ID:          id,
		Placement:   ui.PlacementCenter,
		Dismissable: true,
		OnDismiss:   onClose,
		Backdrop:    true,
		Content: ui.DialogLayout(kind,
			layout.Column(
				display.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				display.Spacer(12),
				display.Text(message),
				display.Spacer(16),
				layout.Row(
					display.Spacer(0),
					button.Text("OK", onClose),
				),
			),
		),
	}
}

// ConfirmDialog returns an overlay element with Confirm/Cancel buttons.
func ConfirmDialog(id ui.OverlayID, title, message string, onConfirm, onCancel func()) ui.Element {
	return ui.Overlay{
		ID:          id,
		Placement:   ui.PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		Content: ui.DialogLayout(platform.DialogInfo,
			layout.Column(
				display.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				display.Spacer(12),
				display.Text(message),
				display.Spacer(16),
				layout.Row(
					display.Spacer(0),
					button.OutlinedText("Cancel", onCancel),
					display.Spacer(8),
					button.Text("Confirm", onConfirm),
				),
			),
		),
	}
}

// InputDialog returns an overlay element with a text field and OK/Cancel buttons.
func InputDialog(id ui.OverlayID, title, message, value, placeholder string, onValueChange func(string), onConfirm, onCancel func()) ui.Element {
	return ui.Overlay{
		ID:          id,
		Placement:   ui.PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		Content: ui.DialogLayout(platform.DialogInfo,
			layout.Column(
				display.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				display.Spacer(12),
				display.Text(message),
				display.Spacer(12),
				form.NewTextField(value, placeholder, form.WithOnChange(onValueChange)),
				display.Spacer(16),
				layout.Row(
					display.Spacer(0),
					button.OutlinedText("Cancel", onCancel),
					display.Spacer(8),
					button.Text("OK", onConfirm),
				),
			),
		),
	}
}
