package vellum

import (
	"github.com/timzifer/lux/draw"
)

// CanvasEncoder implements draw.Canvas and records all operations into a
// FrameBuffer while forwarding them to an optional inner Canvas (RFC-012 §5.2).
//
// When inner is nil, the encoder operates in record-only mode (useful for tests).
type CanvasEncoder struct {
	inner  draw.Canvas  // real canvas (may be nil for record-only)
	buf    *FrameBuffer // serialized stream
	bounds draw.Rect    // cached from BeginFrame or inner
	dpr    float32      // cached from BeginFrame or inner
}

var _ draw.Canvas = (*CanvasEncoder)(nil)

// NewCanvasEncoder creates a CanvasEncoder. If inner is nil, calls are
// recorded but not forwarded.
func NewCanvasEncoder(inner draw.Canvas, buf *FrameBuffer) *CanvasEncoder {
	e := &CanvasEncoder{inner: inner, buf: buf, dpr: 1.0}
	if inner != nil {
		e.bounds = inner.Bounds()
		e.dpr = inner.DPR()
	}
	return e
}

// Buffer returns the underlying FrameBuffer.
func (e *CanvasEncoder) Buffer() *FrameBuffer { return e.buf }

// BeginFrame records a frame start marker with metadata.
func (e *CanvasEncoder) BeginFrame(frameID uint64, bounds draw.Rect, dpr float32) {
	e.bounds = bounds
	e.dpr = dpr
	e.buf.WriteOp(OpBeginFrame, func(w *WireWriter) {
		w.WriteUint64(frameID)
		w.WriteRect(bounds)
		w.WriteFloat32(dpr)
	})
}

// EndFrame records a frame end marker, optionally including DebugFrameInfo.
func (e *CanvasEncoder) EndFrame(info *DebugFrameInfo) {
	if info != nil {
		e.buf.WriteOp(OpDebugFrameInfo, func(w *WireWriter) {
			writeDebugFrameInfo(w, info)
		})
	}
	e.buf.WriteOp(OpEndFrame, nil)
}

// ── Primitives ──────────────────────────────────────────────────────────────

func (e *CanvasEncoder) FillRect(r draw.Rect, paint draw.Paint) {
	if e.inner != nil {
		e.inner.FillRect(r, paint)
	}
	e.buf.WriteOp(OpFillRect, func(w *WireWriter) {
		w.WriteRect(r)
		w.WritePaint(paint)
	})
}

func (e *CanvasEncoder) FillRoundRect(r draw.Rect, radius float32, paint draw.Paint) {
	if e.inner != nil {
		e.inner.FillRoundRect(r, radius, paint)
	}
	e.buf.WriteOp(OpFillRoundRect, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteFloat32(radius)
		w.WritePaint(paint)
	})
}

func (e *CanvasEncoder) FillRoundRectCorners(r draw.Rect, radii draw.CornerRadii, paint draw.Paint) {
	if e.inner != nil {
		e.inner.FillRoundRectCorners(r, radii, paint)
	}
	e.buf.WriteOp(OpFillRoundRectCorners, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteCornerRadii(radii)
		w.WritePaint(paint)
	})
}

func (e *CanvasEncoder) FillEllipse(r draw.Rect, paint draw.Paint) {
	if e.inner != nil {
		e.inner.FillEllipse(r, paint)
	}
	e.buf.WriteOp(OpFillEllipse, func(w *WireWriter) {
		w.WriteRect(r)
		w.WritePaint(paint)
	})
}

func (e *CanvasEncoder) StrokeRect(r draw.Rect, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokeRect(r, stroke)
	}
	e.buf.WriteOp(OpStrokeRect, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteStroke(stroke)
	})
}

func (e *CanvasEncoder) StrokeRoundRect(r draw.Rect, radius float32, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokeRoundRect(r, radius, stroke)
	}
	e.buf.WriteOp(OpStrokeRoundRect, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteFloat32(radius)
		w.WriteStroke(stroke)
	})
}

func (e *CanvasEncoder) StrokeRoundRectCorners(r draw.Rect, radii draw.CornerRadii, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokeRoundRectCorners(r, radii, stroke)
	}
	e.buf.WriteOp(OpStrokeRoundRectCorners, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteCornerRadii(radii)
		w.WriteStroke(stroke)
	})
}

func (e *CanvasEncoder) StrokeEllipse(r draw.Rect, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokeEllipse(r, stroke)
	}
	e.buf.WriteOp(OpStrokeEllipse, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteStroke(stroke)
	})
}

func (e *CanvasEncoder) StrokeLine(a, b draw.Point, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokeLine(a, b, stroke)
	}
	e.buf.WriteOp(OpStrokeLine, func(w *WireWriter) {
		w.WritePoint(a)
		w.WritePoint(b)
		w.WriteStroke(stroke)
	})
}

// ── Paths ───────────────────────────────────────────────────────────────────

func (e *CanvasEncoder) FillPath(p draw.Path, paint draw.Paint) {
	if e.inner != nil {
		e.inner.FillPath(p, paint)
	}
	e.buf.WriteOp(OpFillPath, func(w *WireWriter) {
		w.WritePath(p)
		w.WritePaint(paint)
	})
}

func (e *CanvasEncoder) StrokePath(p draw.Path, stroke draw.Stroke) {
	if e.inner != nil {
		e.inner.StrokePath(p, stroke)
	}
	e.buf.WriteOp(OpStrokePath, func(w *WireWriter) {
		w.WritePath(p)
		w.WriteStroke(stroke)
	})
}

// ── Text ────────────────────────────────────────────────────────────────────

func (e *CanvasEncoder) DrawText(text string, origin draw.Point, style draw.TextStyle, color draw.Color) {
	if e.inner != nil {
		e.inner.DrawText(text, origin, style, color)
	}
	e.buf.WriteOp(OpDrawText, func(w *WireWriter) {
		w.WriteString(text)
		w.WritePoint(origin)
		w.WriteTextStyle(style)
		w.WriteColor(color)
	})
}

func (e *CanvasEncoder) MeasureText(text string, style draw.TextStyle) draw.TextMetrics {
	// MeasureText is a query — record it but delegate to inner for the result.
	e.buf.WriteOp(OpMeasureText, func(w *WireWriter) {
		w.WriteString(text)
		w.WriteTextStyle(style)
	})
	if e.inner != nil {
		return e.inner.MeasureText(text, style)
	}
	return draw.TextMetrics{}
}

func (e *CanvasEncoder) DrawTextLayout(layout draw.TextLayout, origin draw.Point, color draw.Color) {
	if e.inner != nil {
		e.inner.DrawTextLayout(layout, origin, color)
	}
	e.buf.WriteOp(OpDrawTextLayout, func(w *WireWriter) {
		w.WriteTextLayout(layout)
		w.WritePoint(origin)
		w.WriteColor(color)
	})
}

// ── Images & Textures ───────────────────────────────────────────────────────

func (e *CanvasEncoder) DrawImage(img draw.ImageID, dst draw.Rect, opts draw.ImageOptions) {
	if e.inner != nil {
		e.inner.DrawImage(img, dst, opts)
	}
	e.buf.WriteOp(OpDrawImage, func(w *WireWriter) {
		w.WriteUint64(uint64(img))
		w.WriteRect(dst)
		w.WriteImageOptions(opts)
	})
}

func (e *CanvasEncoder) DrawImageScaled(img draw.ImageID, dst draw.Rect, mode draw.ImageScaleMode, opts draw.ImageOptions) {
	if e.inner != nil {
		e.inner.DrawImageScaled(img, dst, mode, opts)
	}
	e.buf.WriteOp(OpDrawImageScaled, func(w *WireWriter) {
		w.WriteUint64(uint64(img))
		w.WriteRect(dst)
		w.WriteUint8(uint8(mode))
		w.WriteImageOptions(opts)
	})
}

func (e *CanvasEncoder) DrawImageSlice(slice draw.ImageSlice, dst draw.Rect, opts draw.ImageOptions) {
	if e.inner != nil {
		e.inner.DrawImageSlice(slice, dst, opts)
	}
	e.buf.WriteOp(OpDrawImageSlice, func(w *WireWriter) {
		w.WriteImageSlice(slice)
		w.WriteRect(dst)
		w.WriteImageOptions(opts)
	})
}

func (e *CanvasEncoder) DrawTexture(tex draw.TextureID, dst draw.Rect) {
	if e.inner != nil {
		e.inner.DrawTexture(tex, dst)
	}
	e.buf.WriteOp(OpDrawTexture, func(w *WireWriter) {
		w.WriteUint64(uint64(tex))
		w.WriteRect(dst)
	})
}

// ── Shadows ─────────────────────────────────────────────────────────────────

func (e *CanvasEncoder) DrawShadow(r draw.Rect, shadow draw.Shadow) {
	if e.inner != nil {
		e.inner.DrawShadow(r, shadow)
	}
	e.buf.WriteOp(OpDrawShadow, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteShadow(shadow)
	})
}

// ── Clipping & Transform ────────────────────────────────────────────────────

func (e *CanvasEncoder) PushClip(r draw.Rect) {
	if e.inner != nil {
		e.inner.PushClip(r)
	}
	e.buf.WriteOp(OpPushClip, func(w *WireWriter) {
		w.WriteRect(r)
	})
}

func (e *CanvasEncoder) PushClipRoundRect(r draw.Rect, radius float32) {
	if e.inner != nil {
		e.inner.PushClipRoundRect(r, radius)
	}
	e.buf.WriteOp(OpPushClipRoundRect, func(w *WireWriter) {
		w.WriteRect(r)
		w.WriteFloat32(radius)
	})
}

func (e *CanvasEncoder) PushClipPath(p draw.Path) {
	if e.inner != nil {
		e.inner.PushClipPath(p)
	}
	e.buf.WriteOp(OpPushClipPath, func(w *WireWriter) {
		w.WritePath(p)
	})
}

func (e *CanvasEncoder) PopClip() {
	if e.inner != nil {
		e.inner.PopClip()
	}
	e.buf.WriteOp(OpPopClip, nil)
}

func (e *CanvasEncoder) PushTransform(t draw.Transform) {
	if e.inner != nil {
		e.inner.PushTransform(t)
	}
	e.buf.WriteOp(OpPushTransform, func(w *WireWriter) {
		w.WriteTransform(t)
	})
}

func (e *CanvasEncoder) PopTransform() {
	if e.inner != nil {
		e.inner.PopTransform()
	}
	e.buf.WriteOp(OpPopTransform, nil)
}

func (e *CanvasEncoder) PushOffset(dx, dy float32) {
	if e.inner != nil {
		e.inner.PushOffset(dx, dy)
	}
	e.buf.WriteOp(OpPushOffset, func(w *WireWriter) {
		w.WriteFloat32(dx)
		w.WriteFloat32(dy)
	})
}

func (e *CanvasEncoder) PushScale(sx, sy float32) {
	if e.inner != nil {
		e.inner.PushScale(sx, sy)
	}
	e.buf.WriteOp(OpPushScale, func(w *WireWriter) {
		w.WriteFloat32(sx)
		w.WriteFloat32(sy)
	})
}

// ── Effects ─────────────────────────────────────────────────────────────────

func (e *CanvasEncoder) PushOpacity(alpha float32) {
	if e.inner != nil {
		e.inner.PushOpacity(alpha)
	}
	e.buf.WriteOp(OpPushOpacity, func(w *WireWriter) {
		w.WriteFloat32(alpha)
	})
}

func (e *CanvasEncoder) PopOpacity() {
	if e.inner != nil {
		e.inner.PopOpacity()
	}
	e.buf.WriteOp(OpPopOpacity, nil)
}

func (e *CanvasEncoder) PushBlur(radius float32) {
	if e.inner != nil {
		e.inner.PushBlur(radius)
	}
	e.buf.WriteOp(OpPushBlur, func(w *WireWriter) {
		w.WriteFloat32(radius)
	})
}

func (e *CanvasEncoder) PopBlur() {
	if e.inner != nil {
		e.inner.PopBlur()
	}
	e.buf.WriteOp(OpPopBlur, nil)
}

func (e *CanvasEncoder) PushLayer(opts draw.LayerOptions) {
	if e.inner != nil {
		e.inner.PushLayer(opts)
	}
	e.buf.WriteOp(OpPushLayer, func(w *WireWriter) {
		w.WriteLayerOptions(opts)
	})
}

func (e *CanvasEncoder) PopLayer() {
	if e.inner != nil {
		e.inner.PopLayer()
	}
	e.buf.WriteOp(OpPopLayer, nil)
}

// ── State ───────────────────────────────────────────────────────────────────

func (e *CanvasEncoder) Bounds() draw.Rect {
	if e.inner != nil {
		return e.inner.Bounds()
	}
	return e.bounds
}

func (e *CanvasEncoder) DPR() float32 {
	if e.inner != nil {
		return e.inner.DPR()
	}
	return e.dpr
}

func (e *CanvasEncoder) Save() {
	if e.inner != nil {
		e.inner.Save()
	}
	e.buf.WriteOp(OpSave, nil)
}

func (e *CanvasEncoder) Restore() {
	if e.inner != nil {
		e.inner.Restore()
	}
	e.buf.WriteOp(OpRestore, nil)
}
