package vellum

import (
	"bytes"
	"testing"

	"github.com/timzifer/lux/draw"
)

func roundtripWire(t *testing.T, name string, writeFn func(*WireWriter), readFn func(*WireReader), checkFn func()) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var buf bytes.Buffer
		ww := NewWireWriter(&buf)
		writeFn(ww)
		if ww.Err() != nil {
			t.Fatalf("write error: %v", ww.Err())
		}
		rr := NewWireReader(bytes.NewReader(buf.Bytes()))
		readFn(rr)
		if rr.Err() != nil {
			t.Fatalf("read error: %v", rr.Err())
		}
		checkFn()
	})
}

func TestWireRoundtripFloat32(t *testing.T) {
	var got float32
	roundtripWire(t, "float32",
		func(w *WireWriter) { w.WriteFloat32(3.14) },
		func(r *WireReader) { got = r.ReadFloat32() },
		func() {
			if got != 3.14 {
				t.Errorf("got %v, want 3.14", got)
			}
		},
	)
}

func TestWireRoundtripVarint(t *testing.T) {
	cases := []uint32{0, 1, 127, 128, 16383, 16384, 1<<21 - 1, 1 << 21}
	for _, want := range cases {
		var got uint32
		roundtripWire(t, "varint",
			func(w *WireWriter) { w.WriteVarint(want) },
			func(r *WireReader) { got = r.ReadVarint() },
			func() {
				if got != want {
					t.Errorf("got %d, want %d", got, want)
				}
			},
		)
	}
}

func TestWireRoundtripString(t *testing.T) {
	var got string
	want := "Hello, Vellum!"
	roundtripWire(t, "string",
		func(w *WireWriter) { w.WriteString(want) },
		func(r *WireReader) { got = r.ReadString() },
		func() {
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		},
	)
}

func TestWireRoundtripRect(t *testing.T) {
	var got draw.Rect
	want := draw.R(10, 20, 300, 400)
	roundtripWire(t, "rect",
		func(w *WireWriter) { w.WriteRect(want) },
		func(r *WireReader) { got = r.ReadRect() },
		func() {
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		},
	)
}

func TestWireRoundtripColor(t *testing.T) {
	var got draw.Color
	want := draw.RGBA(128, 64, 200, 255)
	roundtripWire(t, "color",
		func(w *WireWriter) { w.WriteColor(want) },
		func(r *WireReader) { got = r.ReadColor() },
		func() {
			// Allow for 1/255 rounding.
			if absDiff(got.R, want.R) > 0.005 || absDiff(got.G, want.G) > 0.005 ||
				absDiff(got.B, want.B) > 0.005 || absDiff(got.A, want.A) > 0.005 {
				t.Errorf("got %v, want %v", got, want)
			}
		},
	)
}

func TestWireRoundtripPaintSolid(t *testing.T) {
	var got draw.Paint
	want := draw.SolidPaint(draw.RGBA(255, 0, 0, 255))
	roundtripWire(t, "paint-solid",
		func(w *WireWriter) { w.WritePaint(want) },
		func(r *WireReader) { got = r.ReadPaint() },
		func() {
			if got.Kind != want.Kind {
				t.Errorf("kind: got %v, want %v", got.Kind, want.Kind)
			}
		},
	)
}

func TestWireRoundtripPaintLinearGradient(t *testing.T) {
	var got draw.Paint
	want := draw.LinearGradientPaint(
		draw.Pt(0, 0), draw.Pt(100, 100),
		draw.GradientStop{Offset: 0, Color: draw.RGBA(255, 0, 0, 255)},
		draw.GradientStop{Offset: 1, Color: draw.RGBA(0, 0, 255, 255)},
	)
	roundtripWire(t, "paint-linear-gradient",
		func(w *WireWriter) { w.WritePaint(want) },
		func(r *WireReader) { got = r.ReadPaint() },
		func() {
			if got.Kind != draw.PaintLinearGradient {
				t.Errorf("kind: got %v, want PaintLinearGradient", got.Kind)
			}
			if got.Linear == nil {
				t.Fatal("Linear is nil")
			}
			if len(got.Linear.Stops) != 2 {
				t.Errorf("stops: got %d, want 2", len(got.Linear.Stops))
			}
		},
	)
}

func TestWireRoundtripStroke(t *testing.T) {
	var got draw.Stroke
	want := draw.Stroke{
		Paint:      draw.SolidPaint(draw.RGBA(0, 255, 0, 255)),
		Width:      2.5,
		Cap:        draw.StrokeCapRound,
		Join:       draw.StrokeJoinBevel,
		MiterLimit: 4.0,
		Dash:       []float32{5, 3, 2},
		DashOffset: 1.0,
	}
	roundtripWire(t, "stroke",
		func(w *WireWriter) { w.WriteStroke(want) },
		func(r *WireReader) { got = r.ReadStroke() },
		func() {
			if got.Width != want.Width {
				t.Errorf("width: got %v, want %v", got.Width, want.Width)
			}
			if got.Cap != want.Cap {
				t.Errorf("cap: got %v, want %v", got.Cap, want.Cap)
			}
			if len(got.Dash) != len(want.Dash) {
				t.Errorf("dash len: got %d, want %d", len(got.Dash), len(want.Dash))
			}
		},
	)
}

func TestWireRoundtripPath(t *testing.T) {
	var got draw.Path
	want := draw.NewPath().
		MoveTo(draw.Pt(10, 20)).
		LineTo(draw.Pt(100, 20)).
		QuadTo(draw.Pt(150, 50), draw.Pt(100, 80)).
		CubicTo(draw.Pt(80, 100), draw.Pt(20, 100), draw.Pt(10, 80)).
		Close().
		Build()

	roundtripWire(t, "path",
		func(w *WireWriter) { w.WritePath(want) },
		func(r *WireReader) { got = r.ReadPath() },
		func() {
			// Compare by walking both paths.
			var wantSegs, gotSegs []draw.PathSegment
			want.Walk(func(s draw.PathSegment) { wantSegs = append(wantSegs, s) })
			got.Walk(func(s draw.PathSegment) { gotSegs = append(gotSegs, s) })
			if len(wantSegs) != len(gotSegs) {
				t.Fatalf("segment count: got %d, want %d", len(gotSegs), len(wantSegs))
			}
			for i, ws := range wantSegs {
				gs := gotSegs[i]
				if ws.Kind != gs.Kind {
					t.Errorf("seg[%d].Kind: got %v, want %v", i, gs.Kind, ws.Kind)
				}
			}
		},
	)
}

func TestWireRoundtripShadow(t *testing.T) {
	var got draw.Shadow
	want := draw.Shadow{
		Color:        draw.RGBA(0, 0, 0, 128),
		BlurRadius:   10,
		SpreadRadius: 2,
		OffsetX:      3,
		OffsetY:      4,
		Radius:       8,
		Inset:        true,
	}
	roundtripWire(t, "shadow",
		func(w *WireWriter) { w.WriteShadow(want) },
		func(r *WireReader) { got = r.ReadShadow() },
		func() {
			if got.BlurRadius != want.BlurRadius {
				t.Errorf("blur: got %v, want %v", got.BlurRadius, want.BlurRadius)
			}
			if got.Inset != want.Inset {
				t.Errorf("inset: got %v, want %v", got.Inset, want.Inset)
			}
		},
	)
}

func TestWireRoundtripTextStyle(t *testing.T) {
	var got draw.TextStyle
	want := draw.TextStyle{
		FontFamily: "Inter",
		Size:       16,
		Weight:     draw.FontWeightBold,
		LineHeight: 1.5,
		Tracking:   0.02,
		Raster:     true,
	}
	roundtripWire(t, "textstyle",
		func(w *WireWriter) { w.WriteTextStyle(want) },
		func(r *WireReader) { got = r.ReadTextStyle() },
		func() {
			if got.FontFamily != want.FontFamily {
				t.Errorf("family: got %q, want %q", got.FontFamily, want.FontFamily)
			}
			if got.Size != want.Size {
				t.Errorf("size: got %v, want %v", got.Size, want.Size)
			}
			if got.Weight != want.Weight {
				t.Errorf("weight: got %v, want %v", got.Weight, want.Weight)
			}
		},
	)
}

func absDiff(a, b float32) float32 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}
