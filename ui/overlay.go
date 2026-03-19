// Package ui — overlay.go implements the Overlay system (RFC-002 §5.3).
//
// Overlays render above normal layout flow for menus, tooltips, dropdowns
// and dialogs. They are declared in the view tree but positioned in a
// separate layer by the renderer.
package ui

import "github.com/timzifer/lux/draw"

// OverlayID stably identifies an overlay across frames.
// Overlays with the same ID are diffed, not recreated.
type OverlayID string

// OverlayPlacement determines how the overlay is positioned relative to its anchor.
type OverlayPlacement uint8

const (
	PlacementBelow  OverlayPlacement = iota // Below the anchor
	PlacementAbove                          // Above the anchor
	PlacementRight                          // Right of the anchor
	PlacementLeft                           // Left of the anchor
	PlacementCenter                         // Centered in the window (for dialogs)
	PlacementCursor                         // At the current mouse position
)

// OverlayAnimation describes enter/exit animation behavior.
type OverlayAnimation uint8

const (
	OverlayAnimNone      OverlayAnimation = iota
	OverlayAnimFade                       // Fade in/out
	OverlayAnimFadeScale                  // Fade + scale from anchor
)

// DismissOverlayMsg is sent when a dismissable overlay is closed
// by clicking outside or pressing Escape (RFC-002 §5.3).
type DismissOverlayMsg struct {
	ID OverlayID
}

// Overlay is an Element that renders above the normal layout flow (RFC-002 §5.3).
// Position is relative to the window origin, not the parent.
type Overlay struct {
	// ID stably identifies this overlay across frames for diffing.
	ID OverlayID

	// Anchor is the position of the triggering widget in window coordinates (dp).
	Anchor draw.Rect

	// Placement determines how the overlay is positioned relative to the anchor.
	Placement OverlayPlacement

	// Content is the element tree to render inside the overlay.
	Content Element

	// Dismissable: if true, clicking outside closes the overlay
	// and sends DismissOverlayMsg{ID} to the user loop.
	Dismissable bool

	// Animation defines enter/exit behavior.
	Animation OverlayAnimation

	// OnDismiss is called when a dismissable overlay's backdrop is clicked.
	// Required for dismiss-on-click-outside to work when Dismissable is true.
	OnDismiss func()

	// Backdrop draws a semi-transparent scrim behind the overlay when true.
	// Used by modal dialogs to dim the background content.
	Backdrop bool
}

// isElement marks Overlay as an Element.
func (o Overlay) isElement() {}

// ComputeOverlayPosition calculates the overlay's top-left position based on
// anchor rect, placement, content size, and window bounds.
func ComputeOverlayPosition(anchor draw.Rect, placement OverlayPlacement, contentSize draw.Size, windowW, windowH int) draw.Point {
	ww := float32(windowW)
	wh := float32(windowH)

	var x, y float32

	switch placement {
	case PlacementBelow:
		x = anchor.X
		y = anchor.Y + anchor.H
		// Flip above if it would go off-screen bottom.
		if y+contentSize.H > wh && anchor.Y-contentSize.H >= 0 {
			y = anchor.Y - contentSize.H
		}
	case PlacementAbove:
		x = anchor.X
		y = anchor.Y - contentSize.H
		// Flip below if it would go off-screen top.
		if y < 0 && anchor.Y+anchor.H+contentSize.H <= wh {
			y = anchor.Y + anchor.H
		}
	case PlacementRight:
		x = anchor.X + anchor.W
		y = anchor.Y
		if x+contentSize.W > ww && anchor.X-contentSize.W >= 0 {
			x = anchor.X - contentSize.W
		}
	case PlacementLeft:
		x = anchor.X - contentSize.W
		if x < 0 && anchor.X+anchor.W+contentSize.W <= ww {
			x = anchor.X + anchor.W
		}
		y = anchor.Y
	case PlacementCenter:
		x = (ww - contentSize.W) / 2
		y = (wh - contentSize.H) / 2
	case PlacementCursor:
		// Anchor.X/Y is the cursor position.
		x = anchor.X
		y = anchor.Y
	}

	// Final clamp to window bounds.
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+contentSize.W > ww {
		x = ww - contentSize.W
	}
	if y+contentSize.H > wh {
		y = wh - contentSize.H
	}

	return draw.Pt(x, y)
}
