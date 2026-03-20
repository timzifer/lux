// Package ui — dialog.go provides fallback dialog elements rendered as overlays.
//
// These are used when the platform does not implement NativeDialogProvider,
// or when native dialogs fail. Each function returns an Overlay element
// with Backdrop: true and PlacementCenter positioning.
package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/platform"
)

// MessageDialog returns an overlay element displaying a message with an OK button.
func MessageDialog(id OverlayID, title, message string, kind platform.DialogKind, onClose func()) Element {
	icon := dialogIcon(kind)
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onClose,
		Backdrop:    true,
		Content: SizedBox(320, 0,
			Column(
				Row(
					TextStyled(icon, draw.TextStyle{Size: 20, Weight: draw.FontWeightBold}),
					Spacer(8),
					TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				),
				Spacer(12),
				Text(message),
				Spacer(16),
				Row(
					Spacer(0),
					ButtonText("OK", onClose),
				),
			),
		),
	}
}

// ConfirmDialog returns an overlay element with Confirm/Cancel buttons.
func ConfirmDialog(id OverlayID, title, message string, onConfirm, onCancel func()) Element {
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		Content: SizedBox(320, 0,
			Column(
				TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				Spacer(12),
				Text(message),
				Spacer(16),
				Row(
					Spacer(0),
					ButtonOutlinedText("Cancel", onCancel),
					Spacer(8),
					ButtonText("Confirm", onConfirm),
				),
			),
		),
	}
}

// InputDialog returns an overlay element with a text field and OK/Cancel buttons.
func InputDialog(id OverlayID, title, message, value, placeholder string, onValueChange func(string), onConfirm, onCancel func()) Element {
	return Overlay{
		ID:          id,
		Placement:   PlacementCenter,
		Dismissable: true,
		OnDismiss:   onCancel,
		Backdrop:    true,
		Content: SizedBox(360, 0,
			Column(
				TextStyled(title, draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold}),
				Spacer(12),
				Text(message),
				Spacer(12),
				TextField(value, placeholder, WithOnChange(onValueChange)),
				Spacer(16),
				Row(
					Spacer(0),
					ButtonOutlinedText("Cancel", onCancel),
					Spacer(8),
					ButtonText("OK", onConfirm),
				),
			),
		),
	}
}

// dialogIcon returns a text symbol for the given dialog kind.
func dialogIcon(kind platform.DialogKind) string {
	switch kind {
	case platform.DialogWarning:
		return "⚠"
	case platform.DialogError:
		return "✖"
	default:
		return "ℹ"
	}
}
