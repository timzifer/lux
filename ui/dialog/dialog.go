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
)

// MessageDialog returns an overlay element displaying a message with an OK button.
func MessageDialog(id ui.OverlayID, title, message string, kind platform.DialogKind, onClose func()) ui.Element {
	return ui.Overlay{
		ID:          id,
		Placement:   ui.PlacementCenter,
		Dismissable: true,
		OnDismiss:   onClose,
		Backdrop:    true,
		Content: ui.SizedBox(420, 0,
			ui.DialogLayout(kind,
				ui.Column(
					ui.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					ui.Spacer(12),
					ui.Text(message),
					ui.Spacer(16),
					ui.Row(
						ui.Spacer(0),
						ui.ButtonText("OK", onClose),
					),
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
		Content: ui.SizedBox(420, 0,
			ui.DialogLayout(platform.DialogInfo,
				ui.Column(
					ui.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					ui.Spacer(12),
					ui.Text(message),
					ui.Spacer(16),
					ui.Row(
						ui.Spacer(0),
						ui.ButtonOutlinedText("Cancel", onCancel),
						ui.Spacer(8),
						ui.ButtonText("Confirm", onConfirm),
					),
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
		Content: ui.SizedBox(460, 0,
			ui.DialogLayout(platform.DialogInfo,
				ui.Column(
					ui.TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
					ui.Spacer(12),
					ui.Text(message),
					ui.Spacer(12),
					ui.TextField(value, placeholder, ui.WithOnChange(onValueChange)),
					ui.Spacer(16),
					ui.Row(
						ui.Spacer(0),
						ui.ButtonOutlinedText("Cancel", onCancel),
						ui.Spacer(8),
						ui.ButtonText("OK", onConfirm),
					),
				),
			),
		),
	}
}
