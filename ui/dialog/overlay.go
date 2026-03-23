// Package dialog — overlay.go provides the Overlay element for rendering
// content above the normal layout flow (menus, tooltips, dropdowns, dialogs).
// This is the sub-package version of ui.Overlay (RFC-002 §5.3).
package dialog

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// OverlayID stably identifies an overlay across frames.
// Overlays with the same ID are diffed, not recreated.
type OverlayID = ui.OverlayID

// OverlayPlacement determines how the overlay is positioned relative to its anchor.
type OverlayPlacement = ui.OverlayPlacement

// Re-export placement constants for convenience.
const (
	PlacementBelow  = ui.PlacementBelow
	PlacementAbove  = ui.PlacementAbove
	PlacementRight  = ui.PlacementRight
	PlacementLeft   = ui.PlacementLeft
	PlacementCenter = ui.PlacementCenter
	PlacementCursor = ui.PlacementCursor
)

// OverlayAnimation describes enter/exit animation behavior.
type OverlayAnimation = ui.OverlayAnimation

// Re-export animation constants for convenience.
const (
	OverlayAnimNone      = ui.OverlayAnimNone
	OverlayAnimFade      = ui.OverlayAnimFade
	OverlayAnimFadeScale = ui.OverlayAnimFadeScale
)

// DismissOverlayMsg is sent when a dismissable overlay is closed
// by clicking outside or pressing Escape (RFC-002 §5.3).
type DismissOverlayMsg = ui.DismissOverlayMsg

// OverlayElement is an Element that renders above the normal layout flow.
// Position is relative to the window origin, not the parent.
type OverlayElement struct {
	ui.BaseElement

	// ID stably identifies this overlay across frames for diffing.
	ID OverlayID

	// Anchor is the position of the triggering widget in window coordinates (dp).
	Anchor draw.Rect

	// Placement determines how the overlay is positioned relative to the anchor.
	Placement OverlayPlacement

	// Content is the element tree to render inside the overlay.
	Content ui.Element

	// Dismissable: if true, clicking outside closes the overlay
	// and sends DismissOverlayMsg{ID} to the user loop.
	Dismissable bool

	// Animation defines enter/exit behavior.
	Animation OverlayAnimation

	// OnDismiss is called when a dismissable overlay's backdrop is clicked.
	OnDismiss func()

	// Backdrop draws a semi-transparent scrim behind the overlay when true.
	Backdrop bool
}

// LayoutSelf implements ui.Layouter. It pushes an overlay entry onto the
// overlay stack and takes no space in the normal layout flow.
func (o OverlayElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if o.Content == nil || ctx.Overlays == nil {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	content := o.Content
	anchor := o.Anchor
	placement := o.Placement
	dismissable := o.Dismissable
	onDismiss := o.OnDismiss
	backdrop := o.Backdrop
	animation := o.Animation
	winW, winH := ctx.Overlays.WindowW, ctx.Overlays.WindowH

	// Resolve animation duration from theme tokens (RFC-008 §9.5).
	overlayDuration := ctx.Tokens.Motion.Standard
	if animation == OverlayAnimFadeScale {
		overlayDuration = ctx.Tokens.Motion.Emphasized
	}

	// Capture theme and canvas for the deferred render closure.
	th := ctx.Theme
	tokens := ctx.Tokens

	ctx.Overlays.Push(ui.OverlayEntry{
		Animation: animation,
		Duration:  overlayDuration,
		Render: func(canvas draw.Canvas, tokens2 theme.TokenSet, ix *ui.Interactor) {
			// Draw semi-transparent scrim behind the overlay for modal dialogs.
			if backdrop {
				canvas.FillRect(draw.R(0, 0, float32(winW), float32(winH)),
					draw.SolidPaint(tokens2.Colors.Surface.Scrim))
			}

			// If dismissable, register a full-window backdrop hit target.
			if dismissable && onDismiss != nil {
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), onDismiss)
			}

			// Measure content with null canvas.
			nc := ui.NullCanvas{Delegate: canvas}
			measureCtx := &ui.LayoutContext{
				Area:   ui.Bounds{X: 0, Y: 0, W: 400, H: 300},
				Canvas: nc,
				Theme:  th,
				Tokens: tokens,
			}
			cb := measureCtx.LayoutChild(content, measureCtx.Area)

			pad := 8
			contentSize := draw.Size{W: float32(cb.W + pad*2), H: float32(cb.H + pad*2)}

			// Compute position using the overlay placement logic.
			pos := ui.ComputeOverlayPosition(anchor, placement, contentSize, winW, winH)

			// Draw border.
			overlayRect := draw.R(pos.X, pos.Y, contentSize.W, contentSize.H)
			canvas.FillRoundRect(overlayRect, tokens2.Radii.Card, draw.SolidPaint(tokens2.Colors.Stroke.Border))

			// Draw elevated surface fill.
			inner := draw.R(pos.X+1, pos.Y+1, contentSize.W-2, contentSize.H-2)
			r := tokens2.Radii.Card - 1
			if r < 0 {
				r = 0
			}
			canvas.FillRoundRect(inner, r, draw.SolidPaint(tokens2.Colors.Surface.Elevated))

			// Layout content inside the overlay.
			renderCtx := &ui.LayoutContext{
				Area: ui.Bounds{
					X: int(pos.X) + pad, Y: int(pos.Y) + pad,
					W: max(int(contentSize.W)-pad*2, 0), H: max(int(contentSize.H)-pad*2, 0),
				},
				Canvas: canvas,
				Theme:  th,
				Tokens: tokens,
				IX:     ix,
			}
			renderCtx.LayoutChild(content, renderCtx.Area)
		},
	})

	// Overlays take no space in normal layout flow.
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}

// TreeEqual implements ui.TreeEqualizer.
func (o OverlayElement) TreeEqual(other ui.Element) bool {
	ob, ok := other.(OverlayElement)
	if !ok {
		return false
	}
	return o.ID == ob.ID &&
		o.Anchor == ob.Anchor &&
		o.Placement == ob.Placement &&
		o.Dismissable == ob.Dismissable &&
		o.Animation == ob.Animation &&
		o.Backdrop == ob.Backdrop
}

// ResolveChildren implements ui.ChildResolver.
func (o OverlayElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	if o.Content != nil {
		o.Content = resolve(o.Content, 0)
	}
	return o
}
