package vellum

import (
	"bytes"
	"fmt"

	"github.com/timzifer/lux/draw"
)

// CanvasDecoder reads a Vellum frame buffer and replays the recorded
// draw.Canvas operations on a target Canvas (RFC-012 §5.2).
type CanvasDecoder struct {
	target draw.Canvas
}

// NewCanvasDecoder creates a decoder that replays operations on target.
func NewCanvasDecoder(target draw.Canvas) *CanvasDecoder {
	return &CanvasDecoder{target: target}
}

// DecodedFrame holds a deserialized frame with its metadata.
type DecodedFrame struct {
	FrameID   uint64
	Bounds    draw.Rect
	DPR       float32
	Ops       []DecodedOp
	FrameInfo *DebugFrameInfo
}

// DecodedOp is a single decoded canvas operation (opcode + raw payload).
type DecodedOp struct {
	Opcode  byte
	Payload []byte
}

// Decode replays all operations from data onto the target Canvas.
// Unknown opcodes are skipped (forward-compatibility via TLV length field).
func (d *CanvasDecoder) Decode(data []byte) error {
	offset := 0
	for offset < len(data) {
		opcode, payload, consumed, err := ReadOp(data, offset)
		if err != nil {
			return err
		}
		offset += consumed

		if err := d.dispatchOp(opcode, payload); err != nil {
			return fmt.Errorf("vellum: opcode 0x%02X at offset %d: %w", opcode, offset-consumed, err)
		}
	}
	return nil
}

// DecodeFrame parses data into a DecodedFrame without replaying.
func DecodeFrame(data []byte) (*DecodedFrame, error) {
	frame := &DecodedFrame{}
	offset := 0
	for offset < len(data) {
		opcode, payload, consumed, err := ReadOp(data, offset)
		if err != nil {
			return nil, err
		}
		offset += consumed

		switch opcode {
		case OpBeginFrame:
			r := NewWireReader(bytes.NewReader(payload))
			frame.FrameID = r.ReadUint64()
			frame.Bounds = r.ReadRect()
			frame.DPR = r.ReadFloat32()
			if r.Err() != nil {
				return nil, r.Err()
			}
		case OpDebugFrameInfo:
			r := NewWireReader(bytes.NewReader(payload))
			info := readDebugFrameInfo(r)
			if r.Err() != nil {
				return nil, r.Err()
			}
			frame.FrameInfo = &info
		}

		frame.Ops = append(frame.Ops, DecodedOp{Opcode: opcode, Payload: payload})
	}
	return frame, nil
}

// dispatchOp dispatches a single opcode+payload to the target Canvas.
func (d *CanvasDecoder) dispatchOp(opcode byte, payload []byte) error {
	r := NewWireReader(bytes.NewReader(payload))

	switch opcode {
	// Frame control — informational only, no Canvas call.
	case OpBeginFrame, OpEndFrame:
		return nil

	// Primitives.
	case OpFillRect:
		rect := r.ReadRect()
		paint := r.ReadPaint()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.FillRect(rect, paint)

	case OpFillRoundRect:
		rect := r.ReadRect()
		radius := r.ReadFloat32()
		paint := r.ReadPaint()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.FillRoundRect(rect, radius, paint)

	case OpFillRoundRectCorners:
		rect := r.ReadRect()
		radii := r.ReadCornerRadii()
		paint := r.ReadPaint()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.FillRoundRectCorners(rect, radii, paint)

	case OpFillEllipse:
		rect := r.ReadRect()
		paint := r.ReadPaint()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.FillEllipse(rect, paint)

	case OpStrokeRect:
		rect := r.ReadRect()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokeRect(rect, stroke)

	case OpStrokeRoundRect:
		rect := r.ReadRect()
		radius := r.ReadFloat32()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokeRoundRect(rect, radius, stroke)

	case OpStrokeRoundRectCorners:
		rect := r.ReadRect()
		radii := r.ReadCornerRadii()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokeRoundRectCorners(rect, radii, stroke)

	case OpStrokeEllipse:
		rect := r.ReadRect()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokeEllipse(rect, stroke)

	case OpStrokeLine:
		a := r.ReadPoint()
		b := r.ReadPoint()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokeLine(a, b, stroke)

	// Paths.
	case OpFillPath:
		path := r.ReadPath()
		paint := r.ReadPaint()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.FillPath(path, paint)

	case OpStrokePath:
		path := r.ReadPath()
		stroke := r.ReadStroke()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.StrokePath(path, stroke)

	// Text.
	case OpDrawText:
		text := r.ReadString()
		origin := r.ReadPoint()
		style := r.ReadTextStyle()
		color := r.ReadColor()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawText(text, origin, style, color)

	case OpDrawTextLayout:
		layout := r.ReadTextLayout()
		origin := r.ReadPoint()
		color := r.ReadColor()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawTextLayout(layout, origin, color)

	case OpMeasureText:
		// MeasureText is a query — skip on replay (RFC-012 §10.1).
		return nil

	// Images & Textures.
	case OpDrawImage:
		img := draw.ImageID(r.ReadUint64())
		dst := r.ReadRect()
		opts := r.ReadImageOptions()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawImage(img, dst, opts)

	case OpDrawImageScaled:
		img := draw.ImageID(r.ReadUint64())
		dst := r.ReadRect()
		mode := draw.ImageScaleMode(r.ReadUint8())
		opts := r.ReadImageOptions()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawImageScaled(img, dst, mode, opts)

	case OpDrawImageSlice:
		slice := r.ReadImageSlice()
		dst := r.ReadRect()
		opts := r.ReadImageOptions()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawImageSlice(slice, dst, opts)

	case OpDrawTexture:
		tex := draw.TextureID(r.ReadUint64())
		dst := r.ReadRect()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawTexture(tex, dst)

	// Shadows.
	case OpDrawShadow:
		rect := r.ReadRect()
		shadow := r.ReadShadow()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.DrawShadow(rect, shadow)

	// Clipping & Transform.
	case OpPushClip:
		rect := r.ReadRect()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushClip(rect)

	case OpPushClipRoundRect:
		rect := r.ReadRect()
		radius := r.ReadFloat32()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushClipRoundRect(rect, radius)

	case OpPushClipPath:
		path := r.ReadPath()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushClipPath(path)

	case OpPopClip:
		d.target.PopClip()

	case OpPushTransform:
		t := r.ReadTransform()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushTransform(t)

	case OpPopTransform:
		d.target.PopTransform()

	case OpPushOffset:
		dx := r.ReadFloat32()
		dy := r.ReadFloat32()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushOffset(dx, dy)

	case OpPushScale:
		sx := r.ReadFloat32()
		sy := r.ReadFloat32()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushScale(sx, sy)

	// Effects.
	case OpPushOpacity:
		alpha := r.ReadFloat32()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushOpacity(alpha)

	case OpPopOpacity:
		d.target.PopOpacity()

	case OpPushBlur:
		radius := r.ReadFloat32()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushBlur(radius)

	case OpPopBlur:
		d.target.PopBlur()

	case OpPushLayer:
		opts := r.ReadLayerOptions()
		if r.Err() != nil {
			return r.Err()
		}
		d.target.PushLayer(opts)

	case OpPopLayer:
		d.target.PopLayer()

	// State.
	case OpSave:
		d.target.Save()

	case OpRestore:
		d.target.Restore()

	// Debug & Control — informational, no Canvas call.
	case OpDebugFrameInfo, OpDebugWidgetTree, OpDebugEventLog,
		OpHandshake, OpAccessTreeUpdate:
		return nil

	default:
		// Unknown opcode — skip (forward-compatible via TLV length).
		return nil
	}

	return nil
}
