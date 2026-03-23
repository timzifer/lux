package button

import (
	"math"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// SegmentedItem describes one segment in a segmented button group.
type SegmentedItem struct {
	Label   string
	Icon    string // optional icon (from icons package)
	OnClick func()
}

// Segmented is a group of connected buttons with one selected.
type Segmented struct {
	ui.BaseElement
	Items    []SegmentedItem
	Selected int
}

// NewSegmented creates a segmented button group.
func NewSegmented(items []SegmentedItem, selected int) ui.Element {
	return Segmented{Items: items, Selected: selected}
}

// LayoutSelf implements ui.Layouter.
func (n Segmented) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	ix := ctx.IX

	numItems := len(n.Items)
	if numItems == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	style := tokens.Typography.Label
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       tokens.Typography.Label.Size * 1.5,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}

	// Pass 1: measure each segment.
	type segInfo struct {
		labelW, labelH int
		iconW          int
		totalW         int
	}
	infos := make([]segInfo, numItems)
	maxH := 0
	for i, item := range n.Items {
		var info segInfo
		if item.Label != "" {
			m := canvas.MeasureText(item.Label, style)
			info.labelW = int(math.Ceil(float64(m.Width)))
			info.labelH = int(math.Ceil(float64(m.Ascent)))
		}
		if item.Icon != "" {
			m := canvas.MeasureText(item.Icon, iconStyle)
			info.iconW = int(math.Ceil(float64(m.Width)))
			if info.labelH == 0 {
				info.labelH = int(math.Ceil(float64(m.Ascent)))
			}
		}
		info.totalW = ui.SegmentPadX*2 + info.labelW
		if item.Icon != "" {
			info.totalW += info.iconW
			if item.Label != "" {
				info.totalW += 6 // gap between icon and label
			}
		}
		infos[i] = info
		h := info.labelH + ui.SegmentPadY*2
		if h > maxH {
			maxH = h
		}
	}

	radius := tokens.Radii.Button

	// Pass 2: render segments.
	cursorX := area.X
	for i, item := range n.Items {
		info := infos[i]
		w := info.totalW

		segRect := draw.R(float32(cursorX), float32(area.Y), float32(w), float32(maxH))
		hoverOpacity := ix.RegisterHit(segRect, item.OnClick)

		selected := i == n.Selected

		// Determine colors.
		var fillColor, textColor draw.Color
		if selected {
			fillColor = tokens.Colors.Accent.Primary
			if hoverOpacity > 0 {
				fillColor = ui.LerpColor(fillColor, ui.HoverHighlight(fillColor), hoverOpacity)
			}
			textColor = tokens.Colors.Text.OnAccent
		} else {
			fillColor = tokens.Colors.Surface.Elevated
			if hoverOpacity > 0 {
				fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
			}
			textColor = tokens.Colors.Text.Primary
		}

		// Draw segment background with appropriate corner rounding.
		if i == 0 && numItems > 1 {
			// Left-rounded segment.
			canvas.FillRoundRect(segRect, radius, draw.SolidPaint(fillColor))
			// Square off right side.
			canvas.FillRect(draw.R(float32(cursorX+w-int(radius)), float32(area.Y), float32(int(radius)), float32(maxH)),
				draw.SolidPaint(fillColor))
		} else if i == numItems-1 && numItems > 1 {
			// Right-rounded segment.
			canvas.FillRoundRect(segRect, radius, draw.SolidPaint(fillColor))
			// Square off left side.
			canvas.FillRect(draw.R(float32(cursorX), float32(area.Y), float32(int(radius)), float32(maxH)),
				draw.SolidPaint(fillColor))
		} else if numItems == 1 {
			canvas.FillRoundRect(segRect, radius, draw.SolidPaint(fillColor))
		} else {
			// Middle segment — no rounding.
			canvas.FillRect(segRect, draw.SolidPaint(fillColor))
		}

		// Draw border between segments (not after last).
		if i < numItems-1 {
			canvas.FillRect(draw.R(float32(cursorX+w), float32(area.Y+2), 1, float32(maxH-4)),
				draw.SolidPaint(tokens.Colors.Stroke.Border))
		}

		// Render content centered.
		contentX := cursorX + ui.SegmentPadX
		centerY := area.Y + (maxH-info.labelH)/2
		if item.Icon != "" {
			canvas.DrawText(item.Icon, draw.Pt(float32(contentX), float32(centerY)), iconStyle, textColor)
			contentX += info.iconW
			if item.Label != "" {
				contentX += 6
			}
		}
		if item.Label != "" {
			canvas.DrawText(item.Label, draw.Pt(float32(contentX), float32(centerY)), style, textColor)
		}

		cursorX += w
	}

	totalW := cursorX - area.X
	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: maxH, Baseline: ui.SegmentPadY + maxH/2}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Segmented) TreeEqual(other ui.Element) bool {
	_, ok := other.(Segmented)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver. Segmented buttons are leaves.
func (n Segmented) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}
