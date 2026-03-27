package menu

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// MenuItem defines an item in a MenuBar or ContextMenu.
type MenuItem struct {
	Label   ui.Element
	OnClick func()
	Items   []MenuItem // sub-items (nested menus)
}

// MenuBarState tracks which top-level menu is open (-1 = all closed).
type MenuBarState struct {
	OpenIndex int
}

// NewMenuBarState creates a MenuBarState with all menus closed.
func NewMenuBarState() *MenuBarState {
	return &MenuBarState{OpenIndex: -1}
}

// MenuBar layout constants.
const (
	menuBarHeight   = 32
	menuBarItemPadX = 12
	menuItemHeight  = 32
	menuItemPadX    = 12
)

// MenuBar renders a horizontal menu bar with dropdown submenus.
type MenuBar struct {
	ui.BaseElement
	Items []MenuItem
	State *MenuBarState
}

// NewMenuBar creates a horizontal menu bar element.
func NewMenuBar(items []MenuItem, state *MenuBarState) ui.Element {
	return MenuBar{Items: items, State: state}
}

// LayoutSelf implements ui.Layouter.
func (n MenuBar) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if len(n.Items) == 0 {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	// Backdrop: when a dropdown is open, a full-screen hit target closes it
	// on any click outside menu bar items or dropdown items.
	if n.State != nil && n.State.OpenIndex >= 0 {
		state := n.State
		ctx.IX.RegisterHit(draw.R(0, 0, 9999, 9999), func() {
			state.OpenIndex = -1
		})
	}

	// Background strip.
	ctx.Canvas.FillRect(
		draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(ctx.Area.W), float32(menuBarHeight)),
		draw.SolidPaint(ctx.Tokens.Colors.Surface.Elevated))

	// Bottom border.
	ctx.Canvas.FillRect(
		draw.R(float32(ctx.Area.X), float32(ctx.Area.Y+menuBarHeight-1), float32(ctx.Area.W), 1),
		draw.SolidPaint(ctx.Tokens.Colors.Stroke.Border))

	cursorX := ctx.Area.X

	for i, item := range n.Items {
		// Measure label.
		cb := ctx.MeasureChild(item.Label, ui.Bounds{X: 0, Y: 0, W: ctx.Area.W, H: menuBarHeight})
		itemW := cb.W + menuBarItemPadX*2

		hasAction := item.OnClick != nil || len(item.Items) > 0

		// Register hit target and get hover opacity atomically.
		var hoverOpacity float32
		if hasAction {
			idx := i
			state := n.State
			subItems := item.Items
			onClick := item.OnClick
			hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(cursorX), float32(ctx.Area.Y), float32(itemW), float32(menuBarHeight)),
				func() {
					if len(subItems) > 0 && state != nil {
						if state.OpenIndex == idx {
							state.OpenIndex = -1
						} else {
							state.OpenIndex = idx
						}
					}
					if onClick != nil {
						onClick()
					}
				})
		}

		// Active highlight for open menu.
		isOpen := n.State != nil && n.State.OpenIndex == i
		if isOpen || hoverOpacity > 0 {
			op := hoverOpacity
			if isOpen {
				op = 1.0
			}
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(ctx.Area.Y), float32(itemW), float32(menuBarHeight)),
				draw.SolidPaint(ui.LerpColor(ctx.Tokens.Colors.Surface.Elevated, ctx.Tokens.Colors.Surface.Hovered, op)))
		}

		// Draw label.
		labelArea := ui.Bounds{X: cursorX + menuBarItemPadX, Y: ctx.Area.Y + (menuBarHeight-cb.H)/2, W: cb.W, H: cb.H}
		ctx.LayoutChild(item.Label, labelArea)

		// Dropdown overlay for open submenu.
		if isOpen && len(item.Items) > 0 {
			dropdownX := cursorX
			dropdownY := ctx.Area.Y + menuBarHeight
			subItems := item.Items
			th := ctx.Theme
			ctx.Overlays.Push(ui.OverlayEntry{
				Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
					nc := ui.NullCanvas{Delegate: canvas}
					layoutMenuDropdown(subItems, dropdownX, dropdownY, nc, canvas, th, tokens, ix)
				},
			})
		}

		cursorX += itemW
	}

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: ctx.Area.W, H: menuBarHeight}
}

// layoutMenuDropdown renders a dropdown menu at the given position.
// Shared by MenuBar dropdowns and context menus.
func layoutMenuDropdown(items []MenuItem, posX, posY int, nc ui.NullCanvas, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *ui.Interactor) {
	// Measure all items.
	maxItemW := 0
	measureCtx := &ui.LayoutContext{
		Canvas: nc,
		Theme:  th,
		Tokens: tokens,
	}
	for _, item := range items {
		cb := measureCtx.LayoutChild(item.Label, ui.Bounds{X: 0, Y: 0, W: 300, H: menuItemHeight})
		w := cb.W + menuItemPadX*2
		if w > maxItemW {
			maxItemW = w
		}
	}
	if maxItemW < 120 {
		maxItemW = 120
	}

	totalH := len(items) * menuItemHeight
	menuW := maxItemW
	menuH := totalH

	// Border.
	canvas.FillRoundRect(
		draw.R(float32(posX), float32(posY), float32(menuW), float32(menuH)),
		tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Stroke.Border))

	// Fill.
	canvas.FillRoundRect(
		draw.R(float32(posX+1), float32(posY+1), float32(max(menuW-2, 0)), float32(max(menuH-2, 0))),
		maxf(tokens.Radii.Card-1, 0), draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Items.
	cursorY := posY
	cornerR := maxf(tokens.Radii.Card-1, 0)
	for itemIdx, item := range items {
		// Register hit target and get hover opacity atomically.
		hoverOpacity := ix.RegisterHit(draw.R(float32(posX), float32(cursorY), float32(menuW), float32(menuItemHeight)),
			item.OnClick)
		if hoverOpacity > 0 {
			hoverColor := draw.SolidPaint(ui.LerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Surface.Hovered, hoverOpacity))
			hoverRect := draw.R(float32(posX+1), float32(cursorY), float32(max(menuW-2, 0)), float32(menuItemHeight))
			if itemIdx == 0 || itemIdx == len(items)-1 {
				canvas.FillRoundRect(hoverRect, cornerR, hoverColor)
			} else {
				canvas.FillRect(hoverRect, hoverColor)
			}
		}

		labelArea := ui.Bounds{X: posX + menuItemPadX, Y: cursorY + (menuItemHeight-16)/2, W: max(menuW-menuItemPadX*2, 0), H: 16}
		itemCtx := &ui.LayoutContext{
			Area:   labelArea,
			Canvas: canvas,
			Theme:  th,
			Tokens: tokens,
			IX:     ix,
		}
		itemCtx.LayoutChild(item.Label, labelArea)

		cursorY += menuItemHeight
	}
}

// TreeEqual implements ui.TreeEqualizer.
func (n MenuBar) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n MenuBar) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
// Walk all top-level menu item labels for accessibility.
func (n MenuBar) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, item := range n.Items {
		b.Walk(item.Label, parentIdx)
	}
}
