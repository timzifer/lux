// Package link provides a clickable link element for the Lux UI framework.
//
// A Link renders as underlined, accent-colored text with hover and focus
// states — HTML <a> semantics without button chrome. Links can be embedded
// inline in a RichTextWidget via display.InlineElement.
package link

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
)

// Link is a clickable inline link element with arbitrary content.
type Link struct {
	ui.BaseElement
	Content  ui.Element
	OnClick  func()
	URL      string // semantic href for accessibility
	Disabled bool
}

// Text creates a link with a text label.
func Text(label string, onClick func()) ui.Element {
	return Link{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
	}
}

// New creates a link wrapping arbitrary content.
func New(content ui.Element, onClick func()) ui.Element {
	return Link{Content: content, OnClick: onClick}
}

// WithURL creates a text link that carries a URL for accessibility.
func WithURL(label, url string, onClick func()) ui.Element {
	return Link{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		URL:     url,
	}
}

// TextDisabled creates a disabled text link.
func TextDisabled(label string) ui.Element {
	return Link{
		Content:  display.TextElement{Content: label},
		Disabled: true,
	}
}

// LayoutSelf implements ui.Layouter.
func (n Link) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	th := ctx.Theme
	ix := ctx.IX
	fs := ctx.Focus

	// Determine link color.
	linkColor := tokens.Colors.Accent.Primary
	if n.Disabled {
		linkColor = tokens.Colors.Text.Disabled
	}

	// Custom theme DrawFunc dispatch.
	if df := th.DrawFunc(theme.WidgetKindLink); df != nil {
		// Measure content for bounds.
		cb := ctx.MeasureChild(n.Content, ui.Bounds{X: 0, Y: 0, W: area.W, H: area.H})
		rect := draw.R(float32(area.X), float32(area.Y), float32(cb.W), float32(cb.H))
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   rect,
			Hovered:  false,
			Focused:  false,
			Disabled: n.Disabled,
		}, tokens, n)
		return ui.Bounds{X: area.X, Y: area.Y, W: cb.W, H: cb.H, Baseline: cb.Baseline}
	}

	// Fast path: TextElement content — draw text directly with link styling.
	if txt, ok := n.Content.(display.TextElement); ok {
		style := tokens.Typography.Body
		if txt.Style != (draw.TextStyle{}) {
			style = txt.Style
		}
		metrics := canvas.MeasureText(txt.Content, style)
		w := int(math.Ceil(float64(metrics.Width)))
		h := int(math.Ceil(float64(metrics.Ascent)))

		linkRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

		// Hit target and hover.
		var hoverOpacity float32
		if n.Disabled {
			ix.RegisterHit(linkRect, nil)
		} else {
			hoverOpacity = ix.RegisterHit(linkRect, n.OnClick)
		}

		// Focus management.
		var focused bool
		if fs != nil && !n.Disabled {
			uid := fs.NextElementUID()
			fs.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
			focused = fs.IsElementFocused(uid)
		}

		// Hover: blend toward a lighter/darker shade.
		drawColor := linkColor
		if hoverOpacity > 0 {
			drawColor = ui.LerpColor(linkColor, tokens.Colors.Text.Primary, hoverOpacity*0.3)
		}

		// Draw text.
		canvas.DrawText(txt.Content,
			draw.Pt(float32(area.X), float32(area.Y)),
			style, drawColor)

		// Draw underline (1dp below text baseline).
		underlineY := float32(area.Y+h) + 1
		canvas.StrokeLine(
			draw.Pt(float32(area.X), underlineY),
			draw.Pt(float32(area.X+w), underlineY),
			draw.Stroke{Paint: draw.SolidPaint(drawColor), Width: 1},
		)

		// Focus ring.
		if focused {
			ui.DrawFocusRing(canvas, linkRect, 2, tokens)
		}

		return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h + 2, Baseline: h}
	}

	// Generic content path: measure, register hit, layout child.
	cb := ctx.MeasureChild(n.Content, ui.Bounds{X: 0, Y: 0, W: area.W, H: area.H})
	w := cb.W
	h := cb.H

	linkRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(linkRect, nil)
	} else {
		hoverOpacity = ix.RegisterHit(linkRect, n.OnClick)
	}

	var focused bool
	if fs != nil && !n.Disabled {
		uid := fs.NextElementUID()
		fs.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = fs.IsElementFocused(uid)
	}

	_ = hoverOpacity // hover effect handled by child rendering

	// Layout child content.
	subCtx := &ui.LayoutContext{
		Area:     ui.Bounds{X: area.X, Y: area.Y, W: w, H: h},
		Canvas:   canvas,
		Theme:    th,
		Tokens:   tokens,
		IX:       ix,
		Overlays: ctx.Overlays,
		Focus:    fs,
	}
	subCtx.LayoutChild(n.Content, subCtx.Area)

	// Draw underline beneath content.
	underlineY := float32(area.Y+h) + 1
	drawColor := linkColor
	canvas.StrokeLine(
		draw.Pt(float32(area.X), underlineY),
		draw.Pt(float32(area.X+w), underlineY),
		draw.Stroke{Paint: draw.SolidPaint(drawColor), Width: 1},
	)

	if focused {
		ui.DrawFocusRing(canvas, linkRect, 2, tokens)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h + 2, Baseline: cb.Baseline}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Link) TreeEqual(other ui.Element) bool {
	o, ok := other.(Link)
	return ok && n.URL == o.URL && n.Disabled == o.Disabled
}

// ResolveChildren implements ui.ChildResolver. Link is treated as a leaf
// for widget resolution purposes when content is a TextElement; otherwise
// the content child is resolved.
func (n Link) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	if _, ok := n.Content.(display.TextElement); ok {
		return n
	}
	n.Content = resolve(n.Content, 0)
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Link) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := extractLabel(n.Content)
	node := a11y.AccessNode{
		Role:  a11y.RoleLink,
		Label: label,
		Value: n.URL,
	}
	if n.OnClick != nil && !n.Disabled {
		node.Actions = []a11y.AccessAction{
			{Name: "activate", Trigger: n.OnClick},
		}
	}
	if n.Disabled {
		node.States.Disabled = true
	}
	b.AddNode(node, parentIdx, a11y.Rect{})
}

// extractLabel tries to get a text label from a link's content element.
func extractLabel(el ui.Element) string {
	if txt, ok := el.(display.TextElement); ok {
		return txt.Content
	}
	return ""
}
