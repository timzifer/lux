package vellum

import (
	"testing"
	"time"

	"github.com/timzifer/lux/draw"
)

// TestCanvasRoundtrip records a sequence of Canvas operations through the
// CanvasEncoder, then decodes them with CanvasDecoder onto a RecordCanvas,
// and verifies the same operations were replayed.
func TestCanvasRoundtrip(t *testing.T) {
	buf := NewFrameBuffer()
	encoder := NewCanvasEncoder(nil, buf) // nil inner = record-only

	// Record a variety of operations.
	encoder.BeginFrame(42, draw.R(0, 0, 800, 600), 2.0)
	encoder.FillRect(draw.R(10, 20, 100, 50), draw.SolidPaint(draw.RGBA(255, 0, 0, 255)))
	encoder.FillRoundRect(draw.R(0, 0, 200, 100), 8, draw.SolidPaint(draw.RGBA(0, 255, 0, 255)))
	encoder.FillEllipse(draw.R(50, 50, 60, 60), draw.SolidPaint(draw.RGBA(0, 0, 255, 255)))
	encoder.StrokeLine(draw.Pt(0, 0), draw.Pt(100, 100), draw.Stroke{
		Paint: draw.SolidPaint(draw.RGBA(128, 128, 128, 255)),
		Width: 2,
	})
	encoder.DrawText("Hello", draw.Pt(10, 10), draw.TextStyle{
		FontFamily: "Inter",
		Size:       14,
		Weight:     draw.FontWeightRegular,
	}, draw.RGBA(0, 0, 0, 255))
	encoder.PushClip(draw.R(0, 0, 400, 300))
	encoder.PushOpacity(0.5)
	encoder.FillRect(draw.R(20, 20, 80, 40), draw.SolidPaint(draw.RGBA(255, 255, 0, 255)))
	encoder.PopOpacity()
	encoder.PopClip()
	encoder.PushOffset(10, 20)
	encoder.PopTransform()
	encoder.Save()
	encoder.Restore()
	encoder.DrawShadow(draw.R(0, 0, 100, 100), draw.Shadow{
		Color:      draw.RGBA(0, 0, 0, 64),
		BlurRadius: 8,
		OffsetY:    4,
	})
	encoder.EndFrame(nil)

	// Decode onto a recording canvas.
	rec := &recordCanvas{}
	decoder := NewCanvasDecoder(rec)
	if err := decoder.Decode(buf.Bytes()); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Verify operations were replayed.
	expected := []string{
		"FillRect", "FillRoundRect", "FillEllipse", "StrokeLine",
		"DrawText", "PushClip", "PushOpacity", "FillRect", "PopOpacity",
		"PopClip", "PushOffset", "PopTransform", "Save", "Restore", "DrawShadow",
	}
	if len(rec.ops) != len(expected) {
		t.Fatalf("op count: got %d, want %d\ngot: %v", len(rec.ops), len(expected), rec.ops)
	}
	for i, want := range expected {
		if rec.ops[i] != want {
			t.Errorf("op[%d]: got %q, want %q", i, rec.ops[i], want)
		}
	}
}

func TestDecodeFrame(t *testing.T) {
	buf := NewFrameBuffer()
	encoder := NewCanvasEncoder(nil, buf)
	encoder.BeginFrame(7, draw.R(0, 0, 1920, 1080), 1.0)
	encoder.FillRect(draw.R(0, 0, 100, 100), draw.SolidPaint(draw.RGBA(255, 0, 0, 255)))
	encoder.EndFrame(&DebugFrameInfo{
		FrameID:   7,
		FrameTime: 2 * time.Millisecond,
		PaintTime: 1 * time.Millisecond,
	})

	frame, err := DecodeFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if frame.FrameID != 7 {
		t.Errorf("frameID: got %d, want 7", frame.FrameID)
	}
	if frame.Bounds.W != 1920 {
		t.Errorf("bounds.W: got %v, want 1920", frame.Bounds.W)
	}
	if frame.FrameInfo == nil {
		t.Fatal("frameInfo is nil")
	}
	if frame.FrameInfo.FrameTime != 2*time.Millisecond {
		t.Errorf("frameTime: got %v, want 2ms", frame.FrameInfo.FrameTime)
	}
}

func TestFrameBufferTLV(t *testing.T) {
	fb := NewFrameBuffer()

	// Write a few ops.
	fb.WriteOp(OpFillRect, func(w *WireWriter) {
		w.WriteRect(draw.R(1, 2, 3, 4))
	})
	fb.WriteOp(OpPopClip, nil)
	fb.WriteOp(OpPushOpacity, func(w *WireWriter) {
		w.WriteFloat32(0.75)
	})

	// Read them back.
	data := fb.Bytes()
	offset := 0
	ops := []byte{}
	for offset < len(data) {
		opcode, _, consumed, err := ReadOp(data, offset)
		if err != nil {
			t.Fatalf("ReadOp at offset %d: %v", offset, err)
		}
		ops = append(ops, opcode)
		offset += consumed
	}

	if len(ops) != 3 {
		t.Fatalf("op count: got %d, want 3", len(ops))
	}
	if ops[0] != OpFillRect {
		t.Errorf("ops[0]: got 0x%02X, want 0x%02X", ops[0], OpFillRect)
	}
	if ops[1] != OpPopClip {
		t.Errorf("ops[1]: got 0x%02X, want 0x%02X", ops[1], OpPopClip)
	}
	if ops[2] != OpPushOpacity {
		t.Errorf("ops[2]: got 0x%02X, want 0x%02X", ops[2], OpPushOpacity)
	}
}

func TestDebugFrameInfoRoundtrip(t *testing.T) {
	want := &DebugFrameInfo{
		FrameID:       42,
		FrameTime:     16 * time.Millisecond,
		UpdateTime:    1 * time.Millisecond,
		ReconcileTime: 2 * time.Millisecond,
		LayoutTime:    3 * time.Millisecond,
		PaintTime:     10 * time.Millisecond,
		WidgetCount:   123,
		DirtyWidgets:  []uint64{1, 2, 3, 99},
	}

	buf := NewFrameBuffer()
	buf.WriteOp(OpDebugFrameInfo, func(w *WireWriter) {
		writeDebugFrameInfo(w, want)
	})

	data := buf.Bytes()
	_, payload, _, err := ReadOp(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecodeDebugFrameInfo(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.FrameID != want.FrameID {
		t.Errorf("FrameID: got %d, want %d", got.FrameID, want.FrameID)
	}
	if got.FrameTime != want.FrameTime {
		t.Errorf("FrameTime: got %v, want %v", got.FrameTime, want.FrameTime)
	}
	if len(got.DirtyWidgets) != len(want.DirtyWidgets) {
		t.Errorf("DirtyWidgets len: got %d, want %d", len(got.DirtyWidgets), len(want.DirtyWidgets))
	}
}

func TestDebugWidgetTreeRoundtrip(t *testing.T) {
	want := &DebugWidgetTree{
		Version: 5,
		Nodes: []DebugWidgetNode{
			{
				UID:       100,
				TypeName:  "ui.Button",
				Props:     map[string]string{"label": "Click me", "disabled": "false"},
				StateDump: `{"pressed":false}`,
				Bounds:    draw.R(10, 20, 100, 40),
				Dirty:     true,
			},
			{
				UID:      200,
				TypeName: "ui.Text",
				Props:    map[string]string{"text": "Hello"},
				Bounds:   draw.R(10, 70, 80, 20),
			},
		},
	}

	buf := NewFrameBuffer()
	EncodeDebugWidgetTree(buf, want)

	data := buf.Bytes()
	_, payload, _, err := ReadOp(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecodeDebugWidgetTree(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != want.Version {
		t.Errorf("Version: got %d, want %d", got.Version, want.Version)
	}
	if len(got.Nodes) != len(want.Nodes) {
		t.Fatalf("Nodes len: got %d, want %d", len(got.Nodes), len(want.Nodes))
	}
	if got.Nodes[0].TypeName != "ui.Button" {
		t.Errorf("Nodes[0].TypeName: got %q, want %q", got.Nodes[0].TypeName, "ui.Button")
	}
	if got.Nodes[0].Props["label"] != "Click me" {
		t.Errorf("Nodes[0].Props[label]: got %q, want %q", got.Nodes[0].Props["label"], "Click me")
	}
	if !got.Nodes[0].Dirty {
		t.Error("Nodes[0].Dirty: got false, want true")
	}
}

func TestDebugEventLogRoundtrip(t *testing.T) {
	want := &DebugEventLog{
		FrameID: 10,
		Events: []DebugEvent{
			{
				Timestamp:  500 * time.Microsecond,
				Kind:       "KeyPress",
				TargetUID:  42,
				TargetType: "form.TextField",
				Detail:     "key=Enter",
				Consumed:   true,
			},
		},
	}

	buf := NewFrameBuffer()
	EncodeDebugEventLog(buf, want)

	data := buf.Bytes()
	_, payload, _, err := ReadOp(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecodeDebugEventLog(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.FrameID != want.FrameID {
		t.Errorf("FrameID: got %d, want %d", got.FrameID, want.FrameID)
	}
	if len(got.Events) != 1 {
		t.Fatalf("Events len: got %d, want 1", len(got.Events))
	}
	if got.Events[0].Kind != "KeyPress" {
		t.Errorf("Events[0].Kind: got %q, want %q", got.Events[0].Kind, "KeyPress")
	}
	if !got.Events[0].Consumed {
		t.Error("Events[0].Consumed: got false, want true")
	}
}

// recordCanvas is a minimal Canvas that records operation names.
type recordCanvas struct {
	ops []string
}

func (c *recordCanvas) record(name string) { c.ops = append(c.ops, name) }

func (c *recordCanvas) FillRect(draw.Rect, draw.Paint)                                  { c.record("FillRect") }
func (c *recordCanvas) FillRoundRect(draw.Rect, float32, draw.Paint)                    { c.record("FillRoundRect") }
func (c *recordCanvas) FillRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Paint)    { c.record("FillRoundRectCorners") }
func (c *recordCanvas) FillEllipse(draw.Rect, draw.Paint)                               { c.record("FillEllipse") }
func (c *recordCanvas) StrokeRect(draw.Rect, draw.Stroke)                               { c.record("StrokeRect") }
func (c *recordCanvas) StrokeRoundRect(draw.Rect, float32, draw.Stroke)                 { c.record("StrokeRoundRect") }
func (c *recordCanvas) StrokeRoundRectCorners(draw.Rect, draw.CornerRadii, draw.Stroke) { c.record("StrokeRoundRectCorners") }
func (c *recordCanvas) StrokeEllipse(draw.Rect, draw.Stroke)                            { c.record("StrokeEllipse") }
func (c *recordCanvas) StrokeLine(draw.Point, draw.Point, draw.Stroke)                  { c.record("StrokeLine") }
func (c *recordCanvas) FillPath(draw.Path, draw.Paint)                                  { c.record("FillPath") }
func (c *recordCanvas) StrokePath(draw.Path, draw.Stroke)                               { c.record("StrokePath") }
func (c *recordCanvas) DrawText(string, draw.Point, draw.TextStyle, draw.Color)         { c.record("DrawText") }
func (c *recordCanvas) DrawTextLayout(draw.TextLayout, draw.Point, draw.Color)          { c.record("DrawTextLayout") }
func (c *recordCanvas) DrawImage(draw.ImageID, draw.Rect, draw.ImageOptions)            { c.record("DrawImage") }
func (c *recordCanvas) DrawImageScaled(draw.ImageID, draw.Rect, draw.ImageScaleMode, draw.ImageOptions) {
	c.record("DrawImageScaled")
}
func (c *recordCanvas) DrawImageSlice(draw.ImageSlice, draw.Rect, draw.ImageOptions) { c.record("DrawImageSlice") }
func (c *recordCanvas) DrawTexture(draw.TextureID, draw.Rect)                        { c.record("DrawTexture") }
func (c *recordCanvas) DrawShadow(draw.Rect, draw.Shadow)                            { c.record("DrawShadow") }
func (c *recordCanvas) PushClip(draw.Rect)                                           { c.record("PushClip") }
func (c *recordCanvas) PushClipRoundRect(draw.Rect, float32)                         { c.record("PushClipRoundRect") }
func (c *recordCanvas) PushClipPath(draw.Path)                                       { c.record("PushClipPath") }
func (c *recordCanvas) PopClip()                                                     { c.record("PopClip") }
func (c *recordCanvas) PushTransform(draw.Transform)                                 { c.record("PushTransform") }
func (c *recordCanvas) PopTransform()                                                { c.record("PopTransform") }
func (c *recordCanvas) PushOffset(float32, float32)                                  { c.record("PushOffset") }
func (c *recordCanvas) PushScale(float32, float32)                                   { c.record("PushScale") }
func (c *recordCanvas) PushOpacity(float32)                                          { c.record("PushOpacity") }
func (c *recordCanvas) PopOpacity()                                                  { c.record("PopOpacity") }
func (c *recordCanvas) PushBlur(float32)                                             { c.record("PushBlur") }
func (c *recordCanvas) PopBlur()                                                     { c.record("PopBlur") }
func (c *recordCanvas) PushLayer(draw.LayerOptions)                                  { c.record("PushLayer") }
func (c *recordCanvas) PopLayer()                                                    { c.record("PopLayer") }
func (c *recordCanvas) Save()                                                        { c.record("Save") }
func (c *recordCanvas) Restore()                                                     { c.record("Restore") }
func (c *recordCanvas) MeasureText(string, draw.TextStyle) draw.TextMetrics          { return draw.TextMetrics{} }
func (c *recordCanvas) Bounds() draw.Rect                                            { return draw.R(0, 0, 800, 600) }
func (c *recordCanvas) DPR() float32                                                 { return 1.0 }
