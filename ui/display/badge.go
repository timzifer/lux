package display

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// Badge layout constants.
const (
	badgePadX    = 6
	badgePadY    = 2
	badgeMinSize = 20
)

// BadgeElement renders a small pill-shaped indicator with arbitrary content.
type BadgeElement struct {
	ui.BaseElement
	Content ui.Element
	Color   draw.Color // optional custom color; zero = Accent.Primary
}

// Badge creates a small pill-shaped indicator with arbitrary Element content.
func Badge(content ui.Element) ui.Element {
	return BadgeElement{Content: content}
}

// BadgeText is a convenience for text-only badges.
func BadgeText(label string) ui.Element {
	return BadgeElement{Content: Text(label)}
}

// BadgeColor creates a badge with a custom background color.
func BadgeColor(content ui.Element, color draw.Color) ui.Element {
	return BadgeElement{Content: content, Color: color}
}

// LayoutSelf implements ui.Layouter.
func (n BadgeElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Measure content.
	cb := ctx.MeasureChild(n.Content, ui.Bounds{X: 0, Y: 0, W: ctx.Area.W, H: ctx.Area.H})

	w := cb.W + badgePadX*2
	h := cb.H + badgePadY*2
	// Ensure minimum size for circle shape with single characters.
	if w < badgeMinSize {
		w = badgeMinSize
	}
	if h < badgeMinSize {
		h = badgeMinSize
	}

	// Pill background — tonal but still readable.
	bgColor := ui.LerpColor(ctx.Tokens.Colors.Surface.Elevated, ctx.Tokens.Colors.Accent.Primary, 0.75)
	if n.Color.A > 0 {
		bgColor = n.Color
	}
	radius := min(ctx.Tokens.Radii.Pill, float32(min(w, h))/2)
	ctx.Canvas.FillRoundRect(
		draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h)),
		radius, draw.SolidPaint(bgColor))

	// Content (centered).
	contentX := ctx.Area.X + (w-cb.W)/2
	contentY := ctx.Area.Y + (h-cb.H)/2
	ctx.LayoutChild(n.Content, ui.Bounds{X: contentX, Y: contentY, W: cb.W, H: cb.H})

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
// Returns false because BadgeElement has a child element.
func (n BadgeElement) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
// Badge is a leaf in reconciliation.
func (n BadgeElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n BadgeElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Content, parentIdx)
}

