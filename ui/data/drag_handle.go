// Package data — drag_handle.go implements the DragHandle element
// that provides a visual grab indicator for drag-and-drop (RFC-005 §9).
package data

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// DragHandle is a visual grip icon that indicates an element can be
// dragged. When used inside a DragSource with HandleOnly=true, only
// the handle region initiates the drag gesture.
type DragHandle struct {
	ui.BaseElement

	// Size is the width and height of the handle in dp. Default: 24.
	Size float32
}

// LayoutSelf implements ui.Layouter.
func (dh DragHandle) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	size := dh.Size
	if size <= 0 {
		size = 24
	}
	w := int(size)
	h := int(size)
	if w > area.W {
		w = area.W
	}
	if h > area.H {
		h = area.H
	}

	bounds := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Register grab cursor for the handle region.
	if ctx.IX != nil {
		ctx.IX.RegisterDragCursor(bounds, input.CursorGrab, nil)
	}

	// Draw the 6-dot grip pattern (2 columns x 3 rows).
	tokens := ctx.Tokens
	dotColor := tokens.Colors.Text.Secondary
	dotColor.A = 0.5
	dotRadius := float32(2)
	dotSpacing := float32(5)
	centerX := float32(area.X) + float32(w)/2
	centerY := float32(area.Y) + float32(h)/2
	startX := centerX - dotSpacing/2
	startY := centerY - dotSpacing

	for col := 0; col < 2; col++ {
		for row := 0; row < 3; row++ {
			cx := startX + float32(col)*dotSpacing
			cy := startY + float32(row)*dotSpacing
			ctx.Canvas.FillEllipse(
				draw.R(cx-dotRadius, cy-dotRadius, dotRadius*2, dotRadius*2),
				draw.SolidPaint(dotColor),
			)
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}
