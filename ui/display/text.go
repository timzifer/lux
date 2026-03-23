// Package display provides read-only, non-interactive display elements
// for the Lux UI framework (text, icons, images, badges, chips, cards, etc.).
package display

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// TextElement renders a single run of styled text.
type TextElement struct {
	ui.BaseElement
	Content string
	Style   draw.TextStyle // zero value = use tokens.Typography.Body
}

// Text creates a text element using the theme's body style.
func Text(content string) ui.Element { return TextElement{Content: content} }

// TextStyled creates a text element with a specific text style.
// Use this for headings or other non-Body text.
func TextStyled(content string, style draw.TextStyle) ui.Element {
	return TextElement{Content: content, Style: style}
}

// LayoutSelf implements ui.Layouter.
func (n TextElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	style := ctx.Tokens.Typography.Body
	if n.Style.Size > 0 {
		style = n.Style
	}
	metrics := ctx.Canvas.MeasureText(n.Content, style)
	w := int(math.Ceil(float64(metrics.Width)))
	h := int(math.Ceil(float64(metrics.Ascent)))
	ctx.Canvas.DrawText(n.Content, draw.Pt(float32(ctx.Area.X), float32(ctx.Area.Y)), style, ctx.Tokens.Colors.Text.Primary)
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n TextElement) TreeEqual(other ui.Element) bool {
	nb, ok := other.(TextElement)
	return ok && n.Content == nb.Content && n.Style == nb.Style
}

// ResolveChildren implements ui.ChildResolver. TextElement is a leaf.
func (n TextElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n TextElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: n.Content}, parentIdx, a11y.Rect{})
}
