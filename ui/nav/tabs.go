// Package nav provides navigation-oriented UI components.
package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// TabItem defines a single tab with an arbitrary header Element and content.
type TabItem struct {
	Header  ui.Element
	Content ui.Element
}

// Layout constants matching the core ui package values.
const (
	tabHeaderPadX = 16
	tabHeaderPadY = 10
	tabIndicatorH = 2
	columnGap     = ui.ColumnGap
)

// Tabs displays a row of tab headers with selectable content panels.
type Tabs struct {
	ui.BaseElement
	Items    []TabItem
	Selected int
	OnSelect func(int)
}

// New creates a Tabs element.
func New(items []TabItem, selected int, onSelect func(int)) ui.Element {
	return Tabs{Items: items, Selected: selected, OnSelect: onSelect}
}

// LayoutSelf implements ui.Layouter.
func (n Tabs) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if len(n.Items) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Pass 1: measure all headers to determine tab widths.
	type tabMeasure struct{ w, h int }
	measures := make([]tabMeasure, len(n.Items))
	headerH := 0
	for i, item := range n.Items {
		cb := ctx.MeasureChild(item.Header, ui.Bounds{X: 0, Y: 0, W: area.W, H: area.H})
		w := cb.W + tabHeaderPadX*2
		h := cb.H + tabHeaderPadY*2
		measures[i] = tabMeasure{w: w, h: h}
		if h > headerH {
			headerH = h
		}
	}

	// Pass 2: draw tab header row.
	cursorX := area.X
	selected := n.Selected
	if selected < 0 || selected >= len(n.Items) {
		selected = 0
	}

	for i, item := range n.Items {
		tw := measures[i].w

		// Register tab hit target and get hover opacity.
		var hoverOpacity float32
		if n.OnSelect != nil {
			idx := i
			onSelect := n.OnSelect
			hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				func() { onSelect(idx) })
		}

		// Tab background — selected tab gets tonal accent tint; hover blends on top.
		if i == selected {
			tonalBg := ui.LerpColor(ctx.Tokens.Colors.Surface.Base, ctx.Tokens.Colors.Accent.Primary, 0.08)
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				draw.SolidPaint(tonalBg))
		} else if hoverOpacity > 0 {
			hc := ctx.Tokens.Colors.Surface.Hovered
			hc.A *= hoverOpacity
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH)),
				draw.SolidPaint(hc))
		}

		// Tab header content
		headerArea := ui.Bounds{X: cursorX + tabHeaderPadX, Y: area.Y + tabHeaderPadY, W: max(tw-tabHeaderPadX*2, 0), H: max(headerH-tabHeaderPadY*2, 0)}
		ctx.LayoutChild(item.Header, headerArea)

		// Selection indicator (underline)
		if i == selected {
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y+headerH-tabIndicatorH), float32(tw), float32(tabIndicatorH)),
				draw.SolidPaint(ctx.Tokens.Colors.Accent.Primary))
		}

		cursorX += tw
	}

	totalHeaderW := cursorX - area.X

	// Divider below headers
	ctx.Canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+headerH), float32(max(totalHeaderW, area.W)), 1),
		draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))

	// Selected tab content
	contentY := area.Y + headerH + 1 + columnGap
	contentArea := ui.Bounds{X: area.X, Y: contentY, W: area.W, H: max(area.H-headerH-1-columnGap, 0)}
	cb := ctx.LayoutChild(n.Items[selected].Content, contentArea)

	totalH := headerH + 1 + columnGap + cb.H
	totalW := max(totalHeaderW, cb.W)
	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer. Tabs are always unequal (dynamic content).
func (n Tabs) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver.
func (n Tabs) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	out.Items = make([]TabItem, len(n.Items))
	sel := n.Selected
	if sel < 0 || sel >= len(n.Items) {
		sel = 0
	}
	for i, item := range n.Items {
		out.Items[i] = TabItem{
			Header: resolve(item.Header, i*2),
		}
		if i == sel {
			out.Items[i].Content = resolve(item.Content, i*2+1)
		} else {
			out.Items[i].Content = item.Content
		}
	}
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Tabs) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	sel := n.Selected
	if sel < 0 || sel >= len(n.Items) {
		sel = 0
	}
	for i, item := range n.Items {
		b.Walk(item.Header, parentIdx)
		if i == sel {
			b.Walk(item.Content, parentIdx)
		}
	}
}
