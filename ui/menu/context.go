package menu

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// ContextMenuState holds optional state for touch-mode scroll support.
type ContextMenuState struct {
	TouchScroll ui.ScrollState
}

// ContextMenu renders a floating context menu at a given position.
type ContextMenu struct {
	ui.BaseElement
	Items     []MenuItem
	Visible   bool
	PosX      float32
	PosY      float32
	Blur      bool                // optional frosted-glass backdrop (RFC-008 §11.5)
	State     *ContextMenuState   // optional state for touch scroll
	OnDismiss func()              // called when backdrop is tapped (touch mode)
}

// NewContextMenu creates a context menu at the given position.
func NewContextMenu(items []MenuItem, visible bool, x, y float32) ui.Element {
	return ContextMenu{Items: items, Visible: visible, PosX: x, PosY: y}
}

// NewContextMenuBlur creates a context menu with frosted-glass backdrop (RFC-008 §11.5).
func NewContextMenuBlur(items []MenuItem, visible bool, x, y float32) ui.Element {
	return ContextMenu{Items: items, Visible: visible, PosX: x, PosY: y, Blur: true}
}

// ContextMenuOption configures a ContextMenu element.
type ContextMenuOption func(*ContextMenu)

// WithContextMenuState links a ContextMenuState for touch scroll support.
func WithContextMenuState(s *ContextMenuState) ContextMenuOption {
	return func(c *ContextMenu) { c.State = s }
}

// WithContextMenuOnDismiss sets the dismiss callback for touch-mode backdrop taps.
func WithContextMenuOnDismiss(fn func()) ContextMenuOption {
	return func(c *ContextMenu) { c.OnDismiss = fn }
}

// NewContextMenuOpts creates a context menu with functional options.
func NewContextMenuOpts(items []MenuItem, visible bool, x, y float32, opts ...ContextMenuOption) ui.Element {
	cm := ContextMenu{Items: items, Visible: visible, PosX: x, PosY: y}
	for _, o := range opts {
		o(&cm)
	}
	return cm
}

// LayoutSelf implements ui.Layouter.
func (n ContextMenu) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if !n.Visible || len(n.Items) == 0 || ctx.Overlays == nil {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	if ctx.IsTouch() {
		return n.layoutTouch(ctx)
	}
	return n.layoutDesktop(ctx)
}

// layoutDesktop renders the classic floating context menu (unchanged legacy behavior).
func (n ContextMenu) layoutDesktop(ctx *ui.LayoutContext) ui.Bounds {
	items := n.Items
	// Anchor relative to the element's layout area.
	posX := ctx.Area.X + int(n.PosX)
	posY := ctx.Area.Y + int(n.PosY)
	winW, winH := ctx.Overlays.WindowW, ctx.Overlays.WindowH
	th := ctx.Theme

	// Push overlay for context menu rendering.
	ctx.Overlays.Push(ui.OverlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
			nc := ui.NullCanvas{Delegate: canvas}

			// Measure dropdown size for clamping.
			measureCtx := &ui.LayoutContext{
				Canvas: nc,
				Theme:  th,
				Tokens: tokens,
			}
			maxItemW := 0
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
			menuW := maxItemW
			menuH := len(items) * menuItemHeight

			// Clamp to window bounds so the menu stays fully visible.
			clampedX := posX
			clampedY := posY
			if clampedX+menuW > winW {
				clampedX = winW - menuW
			}
			if clampedX < 0 {
				clampedX = 0
			}
			if clampedY+menuH > winH {
				clampedY = winH - menuH
			}
			if clampedY < 0 {
				clampedY = 0
			}

			layoutMenuDropdown(items, clampedX, clampedY, nc, canvas, th, tokens, ix)
		},
	})

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}

// layoutTouch renders a centralized action sheet overlay for touch/HMI profiles.
func (n ContextMenu) layoutTouch(ctx *ui.LayoutContext) ui.Bounds {
	profile := ctx.Profile
	items := n.Items
	winW := ctx.Overlays.WindowW
	winH := ctx.Overlays.WindowH
	state := n.State
	onDismiss := n.OnDismiss
	th := ctx.Theme

	ctx.Overlays.Push(ui.OverlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
			asItems := make([]ActionSheetItem, len(items))
			for i, item := range items {
				asItems[i] = ActionSheetItem{
					Element: item.Label,
					OnClick: item.OnClick,
				}
			}
			var scroll *ui.ScrollState
			if state != nil {
				scroll = &state.TouchScroll
			}
			RenderActionSheet(ActionSheetConfig{
				Items:       asItems,
				Profile:     profile,
				WinW:        winW,
				WinH:        winH,
				OnDismiss:   onDismiss,
				ScrollState: scroll,
				Theme:       th,
			}, canvas, tokens, ix)
		},
	})

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}

// TreeEqual implements ui.TreeEqualizer.
func (n ContextMenu) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n ContextMenu) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
// Walk all menu item labels for accessibility.
func (n ContextMenu) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, item := range n.Items {
		b.Walk(item.Label, parentIdx)
	}
}
