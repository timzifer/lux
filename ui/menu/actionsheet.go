// Package menu — actionsheet.go provides a shared touch-optimized overlay
// renderer ("Action Sheet") used by Select, ContextMenu, and MenuBar when
// the active InteractionProfile indicates touch or HMI input.
//
// The Action Sheet replaces small, anchor-relative dropdown overlays with a
// centralized, bottom-aligned panel featuring large touch targets, scrim
// backdrop, and optional scroll support for long option lists.
package menu

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/interaction"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Action sheet layout constants.
const (
	// actionSheetMaxWidthFrac: the sheet occupies at most 90% of window width.
	actionSheetMaxWidthFrac = 0.9
	// actionSheetMaxHeightFrac: max 70% of window height for content area.
	actionSheetMaxHeightFrac = 0.70
	// actionSheetBottomMargin: spacing from bottom window edge.
	actionSheetBottomMargin = 12
	// actionSheetPadY: vertical padding inside the sheet (top/bottom).
	actionSheetPadY = 8
	// actionSheetItemPadX: horizontal padding inside each item.
	actionSheetItemPadX = 16
	// actionSheetCornerRadius: rounded corners for the sheet.
	actionSheetCornerRadius float32 = 16
	// actionSheetHandleW/H: drag-handle pill dimensions.
	actionSheetHandleW float32 = 36
	actionSheetHandleH float32 = 4
	// actionSheetHandleMarginY: spacing above and below the handle.
	actionSheetHandleMarginY = 8
)

// ActionSheetItem describes a single selectable entry in the action sheet.
type ActionSheetItem struct {
	// Label is a plain-text label (used by Select).
	Label string
	// Element is a custom label element (used by MenuItem). Takes precedence
	// over Label when non-nil.
	Element ui.Element
	// OnClick is called when the item is tapped.
	OnClick func()
	// Selected highlights this item as the current selection.
	Selected bool
}

// ActionSheetConfig configures the action sheet overlay rendering.
type ActionSheetConfig struct {
	// Title is an optional header displayed above the item list.
	Title string
	// Items are the selectable entries.
	Items []ActionSheetItem
	// Profile is the active interaction profile (must not be nil).
	Profile *interaction.InteractionProfile
	// WinW, WinH are the window dimensions in dp.
	WinW, WinH int
	// OnDismiss is called when the scrim backdrop is tapped.
	OnDismiss func()
	// ScrollState persists the scroll offset for long lists across frames.
	// May be nil if scrolling is not needed.
	ScrollState *ui.ScrollState
	// Theme is the active theme, needed for rendering Element labels.
	Theme theme.Theme
}

// TouchItemHeight returns the per-item height for the given interaction
// profile: MinTouchTarget + TouchTargetSpacing.
//   - ProfileTouch: 48 + 8 = 56dp
//   - ProfileHMI:   64 + 12 = 76dp
func TouchItemHeight(profile *interaction.InteractionProfile) int {
	if profile == nil {
		return 56 // sensible fallback
	}
	return int(profile.MinTouchTarget + profile.TouchTargetSpacing)
}

// RenderActionSheet draws a touch-optimized action sheet overlay.
// It is called inside an OverlayEntry.Render closure.
func RenderActionSheet(cfg ActionSheetConfig, canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
	winW := cfg.WinW
	winH := cfg.WinH
	profile := cfg.Profile
	items := cfg.Items
	if len(items) == 0 || profile == nil {
		return
	}

	// ── 1. Scrim backdrop ────────────────────────────────────────
	scrimRect := draw.R(0, 0, float32(winW), float32(winH))
	canvas.FillRect(scrimRect, draw.SolidPaint(tokens.Colors.Surface.Scrim))
	if cfg.OnDismiss != nil {
		ix.RegisterHit(scrimRect, cfg.OnDismiss)
	}

	// ── 2. Compute geometry ──────────────────────────────────────
	itemH := TouchItemHeight(profile)

	// Handle area (pill + margins).
	handleAreaH := actionSheetHandleMarginY*2 + int(actionSheetHandleH)

	// Optional title/header.
	headerH := 0
	if cfg.Title != "" {
		headerH = itemH // same height as an item
	}

	// Total content height.
	contentH := handleAreaH + headerH + len(items)*itemH + actionSheetPadY

	// Sheet dimensions.
	maxH := int(float32(winH) * actionSheetMaxHeightFrac)
	sheetH := contentH
	if sheetH > maxH {
		sheetH = maxH
	}
	sheetW := winW
	maxW := int(float32(winW) * actionSheetMaxWidthFrac)
	if sheetW > maxW {
		sheetW = maxW
	}

	sheetX := (winW - sheetW) / 2
	sheetY := winH - sheetH - actionSheetBottomMargin

	// ── 3. Sheet background ──────────────────────────────────────
	sheetRect := draw.R(float32(sheetX), float32(sheetY), float32(sheetW), float32(sheetH))
	// Eat clicks on the sheet body so they don't fall through to the scrim.
	ix.RegisterHit(sheetRect, func() {})
	canvas.FillRoundRect(sheetRect, actionSheetCornerRadius, draw.SolidPaint(tokens.Colors.Surface.Elevated))
	canvas.StrokeRoundRect(sheetRect, actionSheetCornerRadius, draw.Stroke{
		Paint: draw.SolidPaint(tokens.Colors.Stroke.Border),
		Width: 1,
	})

	// ── 4. Drag handle pill ──────────────────────────────────────
	handleX := float32(sheetX) + (float32(sheetW)-actionSheetHandleW)/2
	handleY := float32(sheetY) + float32(actionSheetHandleMarginY)
	canvas.FillRoundRect(
		draw.R(handleX, handleY, actionSheetHandleW, actionSheetHandleH),
		actionSheetHandleH/2, // fully rounded
		draw.SolidPaint(tokens.Colors.Stroke.Border))

	// ── 5. Title/header ──────────────────────────────────────────
	cursorY := sheetY + handleAreaH
	if cfg.Title != "" {
		titleStyle := tokens.Typography.Label
		titleX := sheetX + actionSheetItemPadX
		titleTextY := cursorY + (headerH-int(titleStyle.Size))/2
		canvas.DrawText(cfg.Title,
			draw.Pt(float32(titleX), float32(titleTextY)),
			titleStyle, tokens.Colors.Text.Secondary)
		// Separator line.
		sepY := float32(cursorY + headerH)
		canvas.FillRect(
			draw.R(float32(sheetX+actionSheetItemPadX), sepY, float32(sheetW-actionSheetItemPadX*2), 1),
			draw.SolidPaint(tokens.Colors.Stroke.Border))
		cursorY += headerH
	}

	// ── 6. Item list (with scroll support) ───────────────────────
	itemAreaY := cursorY
	itemAreaH := sheetH - (cursorY - sheetY) - actionSheetPadY
	totalItemsH := len(items) * itemH
	needsScroll := totalItemsH > itemAreaH

	var scrollOffset float32
	if needsScroll && cfg.ScrollState != nil {
		scrollOffset = cfg.ScrollState.Offset
		// Clamp scroll offset.
		maxScroll := float32(totalItemsH - itemAreaH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if scrollOffset > maxScroll {
			scrollOffset = maxScroll
			cfg.ScrollState.Offset = scrollOffset
		}
	}

	// Clip to item viewport.
	viewportRect := draw.R(float32(sheetX), float32(itemAreaY), float32(sheetW), float32(itemAreaH))
	canvas.PushClip(viewportRect)

	bodyStyle := tokens.Typography.Body
	th := cfg.Theme

	for i, item := range items {
		iy := itemAreaY + i*itemH - int(scrollOffset)

		// Skip items fully outside viewport.
		if iy+itemH < itemAreaY || iy > itemAreaY+itemAreaH {
			// Still register a hit target to keep hover indices aligned.
			ix.RegisterHit(draw.R(0, 0, 0, 0), nil)
			continue
		}

		itemRect := draw.R(float32(sheetX), float32(iy), float32(sheetW), float32(itemH))

		// Selected highlight.
		if item.Selected {
			canvas.FillRect(itemRect, draw.SolidPaint(tokens.Colors.Surface.Pressed))
		}

		// Hit target and hover (hover will be zero on touch since !HasHover).
		ho := ix.RegisterHit(itemRect, item.OnClick)
		if ho > 0 {
			canvas.FillRect(itemRect, draw.SolidPaint(
				ui.LerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Surface.Hovered, ho)))
		}

		// Separator between items (not after last).
		if i < len(items)-1 {
			sepY := float32(iy + itemH)
			canvas.FillRect(
				draw.R(float32(sheetX+actionSheetItemPadX), sepY, float32(sheetW-actionSheetItemPadX*2), 1),
				draw.SolidPaint(tokens.Colors.Stroke.Border))
		}

		// Label rendering.
		if item.Element != nil && th != nil {
			// Render custom element label.
			labelArea := ui.Bounds{
				X: sheetX + actionSheetItemPadX,
				Y: iy + (itemH-int(bodyStyle.Size))/2,
				W: sheetW - actionSheetItemPadX*2,
				H: int(bodyStyle.Size),
			}
			labelCtx := &ui.LayoutContext{
				Area:   labelArea,
				Canvas: canvas,
				Theme:  th,
				Tokens: tokens,
				IX:     ix,
			}
			labelCtx.LayoutChild(item.Element, labelArea)
		} else if item.Label != "" {
			// Render plain text label.
			textX := sheetX + actionSheetItemPadX
			textY := iy + (itemH-int(bodyStyle.Size))/2
			canvas.DrawText(item.Label,
				draw.Pt(float32(textX), float32(textY)),
				bodyStyle, tokens.Colors.Text.Primary)
		}
	}

	canvas.PopClip()

	// ── 7. Scrollbar ─────────────────────────────────────────────
	if needsScroll && cfg.ScrollState != nil {
		st := cfg.ScrollState
		cH := float32(totalItemsH)
		vH := float32(itemAreaH)
		// Register scroll region.
		ix.RegisterScroll(viewportRect, cH, vH, func(deltaY float32) {
			st.ScrollBy(deltaY, cH, vH)
		})
		// Draw scrollbar on right edge of sheet.
		trackX := sheetX + sheetW - 10
		ui.DrawScrollbar(canvas, tokens, ix, st, trackX, itemAreaY, itemAreaH, cH, scrollOffset)
	}
}
