package main

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/vellum"
	"github.com/timzifer/lux/ui"
)

// CanvasReplayElement replays a decoded Vellum frame onto the inspector's own
// canvas, clipped to its layout area (RFC-012 §6.3 — the left "Canvas Replay"
// panel showing the live remote surface).
//
// The Lux renderer's transform stack only supports translation — PushScale and
// PushTransform are no-ops on the scene canvas — so the replay is rendered 1:1:
// the remote frame's origin is pinned to the panel's top-left and anything that
// extends past the panel is clipped away.
type CanvasReplayElement struct {
	ui.BaseElement
	Frame         *vellum.DecodedFrame
	Width, Height float32
}

// canvasReplay creates a CanvasReplayElement sized at most w×h dp.
func canvasReplay(frame *vellum.DecodedFrame, w, h float32) ui.Element {
	return CanvasReplayElement{Frame: frame, Width: w, Height: h}
}

// LayoutSelf implements ui.Layouter.
func (n CanvasReplayElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w := int(n.Width)
	h := int(n.Height)
	if w <= 0 || w > ctx.Area.W {
		w = ctx.Area.W
	}
	if h <= 0 || h > ctx.Area.H {
		h = ctx.Area.H
	}
	dst := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))

	// Paint a backdrop so the replay region is always visible, even before the
	// first frame arrives or when the remote surface is smaller than the panel.
	ctx.Canvas.FillRect(dst, draw.SolidPaint(ctx.Tokens.Colors.Surface.Base))

	if n.Frame != nil && len(n.Frame.Ops) > 0 {
		replayInto(ctx.Canvas, n.Frame, dst)
	}

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h, Baseline: h}
}

// replayInto paints the decoded frame into dst, clipped to dst and translated so
// the frame origin maps to dst's origin. It is guarded against malformed streams
// so a single bad frame cannot crash the inspector.
func replayInto(canvas draw.Canvas, frame *vellum.DecodedFrame, dst draw.Rect) {
	defer func() { _ = recover() }()

	canvas.PushClip(dst)
	canvas.PushOffset(dst.X-frame.Bounds.X, dst.Y-frame.Bounds.Y)

	decoder := vellum.NewCanvasDecoder(canvas)
	_ = decoder.ReplayOps(frame.Ops)

	canvas.PopTransform() // balances PushOffset (offset stack)
	canvas.PopClip()
}

// TreeEqual implements ui.TreeEqualizer. A replay element never compares equal
// so the panel repaints whenever a new frame arrives.
func (n CanvasReplayElement) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver. CanvasReplayElement is a leaf.
func (n CanvasReplayElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op — the replay is decorative.
func (n CanvasReplayElement) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
