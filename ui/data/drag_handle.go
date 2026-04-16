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

	// When inside a HandleOnly DragSource, register the actual drag
	// interaction on the handle bounds (not just a cursor).
	if ctx.IX != nil && activeDragHandleConfig != nil && ctx.IX.DnD != nil {
		cfg := *activeDragHandleConfig
		sourceBounds := bounds
		dnd := ctx.IX.DnD

		var startX, startY float32
		var pressing, dragSent bool

		ctx.IX.RegisterSurfaceDrag(bounds,
			func(x, y float32) {
				if dragSent {
					return
				}
				if !pressing {
					startX, startY = x, y
					pressing = true
					return
				}
				dx := x - startX
				dy := y - startY
				if dx*dx+dy*dy > mouseDragThreshold*mouseDragThreshold {
					data := cfg.DataFn()
					if data == nil {
						return
					}
					ops := cfg.Operations
					if ops == 0 {
						ops = input.DragOperationMove
					}
					data.AllowedOps = ops

					var preview ui.Element
					if cfg.PreviewFn != nil {
						preview = cfg.PreviewFn()
					}

					offset := draw.Point{
						X: sourceBounds.X - startX,
						Y: sourceBounds.Y - startY,
					}

					dnd.StartDrag(
						cfg.WidgetUID, data,
						input.GesturePoint{X: startX, Y: startY},
						sourceBounds, preview, offset,
						cfg.Placeholder,
					)

					if cfg.OnDragStart != nil {
						cfg.OnDragStart()
					}
					dragSent = true
				}
			},
			func(x, y float32) {
				pressing = false
				dragSent = false
			},
		)
	} else if ctx.IX != nil {
		// Register grab cursor for the handle region (visual only).
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
