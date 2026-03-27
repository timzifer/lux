package nav

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Toolbar layout constants.
const (
	toolbarHeight     = 36
	toolbarItemPadX   = 6
	toolbarItemPadY   = 4
	toolbarSepWidth   = 1
	toolbarSepPadY    = 6
	toolbarDefaultGap = 2
)

// ToolbarItem represents one item in a Toolbar.
type ToolbarItem struct {
	// Element is the visual content of this item (e.g. a Text or Icon element).
	// Ignored for separator items.
	Element ui.Element

	// OnClick is invoked when the item is clicked.
	OnClick func()

	// Toggle marks this item as a toggle button.
	Toggle bool

	// Active is the toggle state; when true an accent tint is drawn behind
	// the item.  Only meaningful when Toggle is true.
	Active bool

	// separator is true for items created via ToolbarSeparator().
	separator bool
}

// ToolbarSeparator returns a ToolbarItem that renders as a vertical divider.
func ToolbarSeparator() ToolbarItem {
	return ToolbarItem{separator: true}
}

// Toolbar renders a horizontal bar of interactive items with optional
// separators and toggle highlights.
type Toolbar struct {
	ui.BaseElement
	Items []ToolbarItem
	Gap   int // gap between items; 0 uses toolbarDefaultGap
}

// NewToolbar creates a Toolbar element with default gap.
func NewToolbar(items []ToolbarItem) ui.Element {
	return Toolbar{Items: items}
}

// NewToolbarWithGap creates a Toolbar element with a custom inter-item gap.
func NewToolbarWithGap(items []ToolbarItem, gap int) ui.Element {
	return Toolbar{Items: items, Gap: gap}
}

// LayoutSelf implements ui.Layouter.
func (n Toolbar) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if len(n.Items) == 0 {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}

	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	gap := n.Gap
	if gap <= 0 {
		gap = toolbarDefaultGap
	}

	// Background strip.
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(toolbarHeight)),
		draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Bottom border.
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+toolbarHeight-1), float32(area.W), 1),
		draw.SolidPaint(tokens.Colors.Stroke.Border))

	cursorX := area.X

	for i, item := range n.Items {
		if item.separator {
			// Vertical divider line.
			if i > 0 {
				cursorX += gap
			}
			sepX := float32(cursorX)
			sepY := float32(area.Y + toolbarSepPadY)
			sepH := float32(toolbarHeight - toolbarSepPadY*2)
			canvas.FillRect(
				draw.R(sepX, sepY, float32(toolbarSepWidth), sepH),
				draw.SolidPaint(tokens.Colors.Stroke.Divider))
			cursorX += toolbarSepWidth
			continue
		}

		if i > 0 && !n.Items[i-1].separator {
			cursorX += gap
		}

		// Measure child.
		cb := ctx.MeasureChild(item.Element, ui.Bounds{X: 0, Y: 0, W: area.W, H: toolbarHeight})
		itemW := cb.W + toolbarItemPadX*2
		itemH := toolbarHeight

		itemRect := draw.R(float32(cursorX), float32(area.Y), float32(itemW), float32(itemH))

		// Register hit target.
		var hoverOpacity float32
		if item.OnClick != nil {
			hoverOpacity = ctx.IX.RegisterHit(itemRect, item.OnClick)
		}

		// Active toggle tint.
		if item.Toggle && item.Active {
			tonalBg := ui.LerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Accent.Primary, 0.15)
			canvas.FillRoundRect(itemRect, tokens.Radii.Button, draw.SolidPaint(tonalBg))
		}

		// Hover highlight.
		if hoverOpacity > 0 {
			hc := ui.LerpColor(tokens.Colors.Surface.Elevated, tokens.Colors.Surface.Hovered, hoverOpacity)
			canvas.FillRoundRect(itemRect, tokens.Radii.Button, draw.SolidPaint(hc))
		}

		// Layout child centered.
		childX := cursorX + toolbarItemPadX
		childY := area.Y + (itemH-cb.H)/2
		ctx.LayoutChild(item.Element, ui.Bounds{X: childX, Y: childY, W: cb.W, H: cb.H})

		cursorX += itemW
	}

	totalW := cursorX - area.X
	if area.W > totalW {
		totalW = area.W
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: toolbarHeight}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Toolbar) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver. Toolbar is a leaf in resolution.
func (n Toolbar) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Toolbar) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	idx := b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleToolbar,
		Label: "Toolbar",
	}, parentIdx, a11y.Rect{})
	for _, item := range n.Items {
		if item.separator || item.Element == nil {
			continue
		}
		b.Walk(item.Element, int32(idx))
	}
}
