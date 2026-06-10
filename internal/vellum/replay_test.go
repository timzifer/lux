package vellum

import (
	"testing"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// TestReplayOps verifies that a DecodedFrame can be replayed onto a target
// canvas via ReplayOps, producing the same operations as a raw Decode.
func TestReplayOps(t *testing.T) {
	buf := NewFrameBuffer()
	encoder := NewCanvasEncoder(nil, buf)
	encoder.BeginFrame(1, draw.R(0, 0, 320, 240), 1.0)
	encoder.FillRect(draw.R(0, 0, 320, 240), draw.SolidPaint(draw.RGBA(10, 20, 30, 255)))
	encoder.FillRoundRect(draw.R(8, 8, 100, 40), 6, draw.SolidPaint(draw.RGBA(40, 80, 120, 255)))
	encoder.DrawText("hi", draw.Pt(12, 24), draw.TextStyle{FontFamily: "Inter", Size: 12}, draw.RGBA(255, 255, 255, 255))
	encoder.EndFrame(nil)

	frame, err := DecodeFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}

	rec := &recordCanvas{}
	decoder := NewCanvasDecoder(rec)
	if err := decoder.ReplayOps(frame.Ops); err != nil {
		t.Fatalf("replay ops: %v", err)
	}

	// BeginFrame/EndFrame are informational and produce no canvas call.
	want := []string{"FillRect", "FillRoundRect", "DrawText"}
	if len(rec.ops) != len(want) {
		t.Fatalf("op count: got %d (%v), want %d", len(rec.ops), rec.ops, len(want))
	}
	for i, w := range want {
		if rec.ops[i] != w {
			t.Errorf("op[%d]: got %q, want %q", i, rec.ops[i], w)
		}
	}
}

// TestWidgetTreeFromAccess verifies the access-tree → widget-tree conversion,
// including role naming, label/value props, and dirty flagging.
func TestWidgetTreeFromAccess(t *testing.T) {
	tree := a11y.AccessTree{
		Nodes: []a11y.AccessTreeNode{
			{
				ID:     1,
				Bounds: a11y.Rect{X: 0, Y: 0, Width: 800, Height: 600},
				Node:   a11y.AccessNode{Role: a11y.RoleGroup},
			},
			{
				ID:     2,
				Bounds: a11y.Rect{X: 10, Y: 20, Width: 120, Height: 40},
				Node:   a11y.AccessNode{Role: a11y.RoleButton, Label: "Click me"},
			},
			{
				ID:     3,
				Bounds: a11y.Rect{X: 10, Y: 80, Width: 200, Height: 24},
				Node:   a11y.AccessNode{Role: a11y.RoleSlider, Value: "0.5"},
			},
		},
	}

	wt := WidgetTreeFromAccess(tree, 99, []uint64{2})
	if wt.Version != 99 {
		t.Errorf("version: got %d, want 99", wt.Version)
	}
	if len(wt.Nodes) != 3 {
		t.Fatalf("node count: got %d, want 3", len(wt.Nodes))
	}

	if wt.Nodes[0].TypeName != "Group" {
		t.Errorf("node 0 type: got %q, want Group", wt.Nodes[0].TypeName)
	}

	btn := wt.Nodes[1]
	if btn.TypeName != "Button" {
		t.Errorf("node 1 type: got %q, want Button", btn.TypeName)
	}
	if btn.Props["label"] != "Click me" {
		t.Errorf("node 1 label prop: got %q, want %q", btn.Props["label"], "Click me")
	}
	if !btn.Dirty {
		t.Error("node 1 should be flagged dirty")
	}
	if btn.Bounds.W != 120 || btn.Bounds.H != 40 {
		t.Errorf("node 1 bounds: got %v, want W=120 H=40", btn.Bounds)
	}

	slider := wt.Nodes[2]
	if slider.TypeName != "Slider" {
		t.Errorf("node 2 type: got %q, want Slider", slider.TypeName)
	}
	if slider.Props["value"] != "0.5" {
		t.Errorf("node 2 value prop: got %q, want %q", slider.Props["value"], "0.5")
	}
	if slider.Dirty {
		t.Error("node 2 should not be dirty")
	}
}
