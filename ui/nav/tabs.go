// Package nav provides navigation-oriented UI components.
package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// TabPosition controls where the tab header strip is placed relative to the content.
type TabPosition int

const (
	// TabPositionTop renders headers in a horizontal row above the content (default).
	TabPositionTop TabPosition = iota
	// TabPositionBottom renders headers in a horizontal row below the content.
	TabPositionBottom
	// TabPositionLeft renders headers in a vertical column to the left of the content.
	TabPositionLeft
	// TabPositionRight renders headers in a vertical column to the right of the content.
	TabPositionRight
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
	Position TabPosition
}

// New creates a Tabs element with headers at the top (default position).
func New(items []TabItem, selected int, onSelect func(int)) ui.Element {
	return Tabs{Items: items, Selected: selected, OnSelect: onSelect, Position: TabPositionTop}
}

// NewWithPosition creates a Tabs element with headers at the given position.
func NewWithPosition(items []TabItem, selected int, onSelect func(int), pos TabPosition) ui.Element {
	return Tabs{Items: items, Selected: selected, OnSelect: onSelect, Position: pos}
}

// tabAdaptivePadding returns touch-adapted padding and minimum tab size.
func tabAdaptivePadding(ctx *ui.LayoutContext) (padX, padY, minTabSize int) {
	padX = tabHeaderPadX
	padY = tabHeaderPadY
	if ctx.IsTouch() && ctx.Profile != nil {
		minT := int(ctx.Profile.MinTouchTarget)
		padY = (minT - int(ctx.Tokens.Typography.Label.Size)) / 2
		if padY < tabHeaderPadY {
			padY = tabHeaderPadY
		}
		minTabSize = minT * 2
	}
	return
}

// layoutHorizontalHeaders measures and renders tab headers in a horizontal row.
// headerY is the Y coordinate of the header row; indicatorEdge controls whether
// the selection indicator is at the bottom (top position) or top (bottom position).
func (n Tabs) layoutHorizontalHeaders(ctx *ui.LayoutContext, headerX, headerY, areaW, areaH int, selected int, padX, padY, minTabW int, indicatorAtBottom bool) (totalHeaderW, headerH int) {
	type tabMeasure struct{ w, h int }
	measures := make([]tabMeasure, len(n.Items))
	for i, item := range n.Items {
		cb := ctx.MeasureChild(item.Header, ui.Bounds{X: 0, Y: 0, W: areaW, H: areaH})
		w := cb.W + padX*2
		if w < minTabW {
			w = minTabW
		}
		h := cb.H + padY*2
		measures[i] = tabMeasure{w: w, h: h}
		if h > headerH {
			headerH = h
		}
	}

	cursorX := headerX
	for i, item := range n.Items {
		tw := measures[i].w

		var hoverOpacity float32
		if n.OnSelect != nil {
			idx := i
			onSelect := n.OnSelect
			hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(cursorX), float32(headerY), float32(tw), float32(headerH)),
				func() { onSelect(idx) })
		}

		if i == selected {
			tonalBg := ui.LerpColor(ctx.Tokens.Colors.Surface.Base, ctx.Tokens.Colors.Accent.Primary, 0.08)
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(headerY), float32(tw), float32(headerH)),
				draw.SolidPaint(tonalBg))
		} else if hoverOpacity > 0 {
			hc := ctx.Tokens.Colors.Surface.Hovered
			hc.A *= hoverOpacity
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(headerY), float32(tw), float32(headerH)),
				draw.SolidPaint(hc))
		}

		hdrArea := ui.Bounds{X: cursorX + padX, Y: headerY + padY, W: max(tw-padX*2, 0), H: max(headerH-padY*2, 0)}
		ctx.LayoutChild(item.Header, hdrArea)

		if i == selected {
			var indY int
			if indicatorAtBottom {
				indY = headerY + headerH - tabIndicatorH
			} else {
				indY = headerY
			}
			ctx.Canvas.FillRect(
				draw.R(float32(cursorX), float32(indY), float32(tw), float32(tabIndicatorH)),
				draw.SolidPaint(ctx.Tokens.Colors.Accent.Primary))
		}

		cursorX += tw
	}

	totalHeaderW = cursorX - headerX
	return
}

// measureVerticalHeaders returns the total width and height of vertical tab headers
// without rendering. Used to reserve space before positioning.
func (n Tabs) measureVerticalHeaders(ctx *ui.LayoutContext, areaW, areaH, padX, padY, minTabH int) (headerW, totalHeaderH int) {
	for _, item := range n.Items {
		cb := ctx.MeasureChild(item.Header, ui.Bounds{X: 0, Y: 0, W: areaW, H: areaH})
		w := cb.W + padX*2
		h := cb.H + padY*2
		if h < minTabH {
			h = minTabH
		}
		if w > headerW {
			headerW = w
		}
		totalHeaderH += h
	}
	return
}

// layoutVerticalHeaders measures and renders tab headers in a vertical column.
// headerX is the X coordinate of the header column; indicatorAtRight controls whether
// the selection indicator is at the right edge (left position) or left edge (right position).
func (n Tabs) layoutVerticalHeaders(ctx *ui.LayoutContext, headerX, headerY, areaW, areaH int, selected int, padX, padY, minTabH int, indicatorAtRight bool) (headerW, totalHeaderH int) {
	type tabMeasure struct{ w, h int }
	measures := make([]tabMeasure, len(n.Items))
	for i, item := range n.Items {
		cb := ctx.MeasureChild(item.Header, ui.Bounds{X: 0, Y: 0, W: areaW, H: areaH})
		w := cb.W + padX*2
		h := cb.H + padY*2
		if h < minTabH {
			h = minTabH
		}
		measures[i] = tabMeasure{w: w, h: h}
		if w > headerW {
			headerW = w
		}
	}

	cursorY := headerY
	for i, item := range n.Items {
		th := measures[i].h

		var hoverOpacity float32
		if n.OnSelect != nil {
			idx := i
			onSelect := n.OnSelect
			hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(headerX), float32(cursorY), float32(headerW), float32(th)),
				func() { onSelect(idx) })
		}

		if i == selected {
			tonalBg := ui.LerpColor(ctx.Tokens.Colors.Surface.Base, ctx.Tokens.Colors.Accent.Primary, 0.08)
			ctx.Canvas.FillRect(
				draw.R(float32(headerX), float32(cursorY), float32(headerW), float32(th)),
				draw.SolidPaint(tonalBg))
		} else if hoverOpacity > 0 {
			hc := ctx.Tokens.Colors.Surface.Hovered
			hc.A *= hoverOpacity
			ctx.Canvas.FillRect(
				draw.R(float32(headerX), float32(cursorY), float32(headerW), float32(th)),
				draw.SolidPaint(hc))
		}

		hdrArea := ui.Bounds{X: headerX + padX, Y: cursorY + padY, W: max(headerW-padX*2, 0), H: max(th-padY*2, 0)}
		ctx.LayoutChild(item.Header, hdrArea)

		if i == selected {
			var indX int
			if indicatorAtRight {
				indX = headerX + headerW - tabIndicatorH
			} else {
				indX = headerX
			}
			ctx.Canvas.FillRect(
				draw.R(float32(indX), float32(cursorY), float32(tabIndicatorH), float32(th)),
				draw.SolidPaint(ctx.Tokens.Colors.Accent.Primary))
		}

		cursorY += th
	}

	totalHeaderH = cursorY - headerY
	return
}

// LayoutSelf implements ui.Layouter.
func (n Tabs) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if len(n.Items) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	padX, padY, minTabSize := tabAdaptivePadding(ctx)

	selected := n.Selected
	if selected < 0 || selected >= len(n.Items) {
		selected = 0
	}

	switch n.Position {
	case TabPositionBottom:
		// Content first, then divider, then headers at bottom.
		contentArea := ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: max(area.H, 0)}
		cb := ctx.LayoutChild(n.Items[selected].Content, contentArea)

		dividerY := area.Y + cb.H + columnGap
		headerY := dividerY + 1

		totalHeaderW, headerH := n.layoutHorizontalHeaders(ctx, area.X, headerY, area.W, area.H, selected, padX, padY, minTabSize, false)

		// Divider above headers.
		ctx.Canvas.FillRect(
			draw.R(float32(area.X), float32(dividerY), float32(max(totalHeaderW, area.W)), 1),
			draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))

		totalH := cb.H + columnGap + 1 + headerH
		totalW := max(totalHeaderW, cb.W)
		return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}

	case TabPositionLeft:
		// Headers on the left, divider, content on the right.
		headerW, totalHeaderH := n.layoutVerticalHeaders(ctx, area.X, area.Y, area.W, area.H, selected, padX, padY, minTabSize, true)

		dividerX := area.X + headerW
		contentX := dividerX + 1 + columnGap

		// Vertical divider.
		ctx.Canvas.FillRect(
			draw.R(float32(dividerX), float32(area.Y), 1, float32(max(totalHeaderH, area.H))),
			draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))

		contentArea := ui.Bounds{X: contentX, Y: area.Y, W: max(area.W-headerW-1-columnGap, 0), H: area.H}
		cb := ctx.LayoutChild(n.Items[selected].Content, contentArea)

		totalW := headerW + 1 + columnGap + cb.W
		totalH := max(totalHeaderH, cb.H)
		return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}

	case TabPositionRight:
		// Content on the left, divider, headers on the right.
		// Measure headers first to know their width (no rendering).
		headerW, _ := n.measureVerticalHeaders(ctx, area.W, area.H, padX, padY, minTabSize)

		contentArea := ui.Bounds{X: area.X, Y: area.Y, W: max(area.W-headerW-1-columnGap, 0), H: area.H}
		cb := ctx.LayoutChild(n.Items[selected].Content, contentArea)

		dividerX := area.X + cb.W + columnGap
		headerX := dividerX + 1

		// Render headers at the computed position.
		_, totalHeaderH := n.layoutVerticalHeaders(ctx, headerX, area.Y, area.W, area.H, selected, padX, padY, minTabSize, false)

		// Vertical divider.
		ctx.Canvas.FillRect(
			draw.R(float32(dividerX), float32(area.Y), 1, float32(max(totalHeaderH, area.H))),
			draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))

		totalW := cb.W + columnGap + 1 + headerW
		totalH := max(totalHeaderH, cb.H)
		return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}

	default: // TabPositionTop
		// Headers at top, divider, content below (original behavior).
		totalHeaderW, headerH := n.layoutHorizontalHeaders(ctx, area.X, area.Y, area.W, area.H, selected, padX, padY, minTabSize, true)

		// Divider below headers.
		ctx.Canvas.FillRect(
			draw.R(float32(area.X), float32(area.Y+headerH), float32(max(totalHeaderW, area.W)), 1),
			draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))

		contentY := area.Y + headerH + 1 + columnGap
		contentArea := ui.Bounds{X: area.X, Y: contentY, W: area.W, H: max(area.H-headerH-1-columnGap, 0)}
		cb := ctx.LayoutChild(n.Items[selected].Content, contentArea)

		totalH := headerH + 1 + columnGap + cb.H
		totalW := max(totalHeaderW, cb.W)
		return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
	}
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
