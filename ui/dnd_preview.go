// Package ui — dnd_preview.go renders the drag-and-drop preview ghost
// as an overlay during active drag sessions (RFC-005 §10).
package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// renderDnDPreview draws the drag preview (ghost) at the current cursor
// position with reduced opacity. Called from BuildScene/BuildSceneWithOSK
// while in overlay mode, so the preview renders above all content.
func renderDnDPreview(dnd *DnDManager, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, windowW, windowH int) {
	sess := dnd.Session()
	if sess == nil {
		return
	}

	// Calculate preview position: cursor + offset.
	x := sess.CurrentPos.X + sess.PreviewOffset.X
	y := sess.CurrentPos.Y + sess.PreviewOffset.Y

	if sess.Preview != nil {
		// Render the custom preview element at the cursor position with
		// 70% opacity for the ghost effect.
		canvas.PushOpacity(0.7)

		// Lay out the preview element in a small viewport.
		previewW := 200 // default preview width
		previewH := 60  // default preview height
		if sess.PreviewBounds.W > 0 {
			previewW = int(sess.PreviewBounds.W)
		}
		if sess.PreviewBounds.H > 0 {
			previewH = int(sess.PreviewBounds.H)
		}

		area := Bounds{X: int(x), Y: int(y), W: previewW, H: previewH}
		layoutElement(sess.Preview, area, canvas, th, tokens, nil, nil, nil)

		canvas.PopOpacity()
	} else {
		// Default preview: small semi-transparent rectangle with drag data indicator.
		previewW := float32(120)
		previewH := float32(40)
		accentColor := tokens.Colors.Accent.Primary
		accentColor.A = 0.6
		canvas.FillRoundRect(
			draw.R(x, y, previewW, previewH),
			tokens.Radii.Card,
			draw.SolidPaint(accentColor),
		)
	}

	// Drop zone highlighting is handled by DropTarget widgets themselves
	// (in dropTargetLayout.LayoutSelf) so the overlay only needs to draw
	// the rejection indicator for zones that explicitly reject the data.
	if zone := dnd.zoneByUID(dnd.hoveredZoneUID); zone != nil {
		if zone.Accept != nil && !zone.Accept(sess.Data, sess.Operation) {
			rejectColor := tokens.Colors.Status.Error
			rejectColor.A = 0.1
			canvas.FillRoundRect(
				zone.Bounds,
				tokens.Radii.Card,
				draw.SolidPaint(rejectColor),
			)
		}
	}
}
