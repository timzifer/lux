package vellum

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/timzifer/lux/draw"
)

// wire.go provides low-level binary encoding helpers for the Vellum protocol.
// All multi-byte values are little-endian. Strings are length-prefixed with
// a varint. See RFC-012 §7.3 for the complete encoding table.

// ── Writer helpers ──────────────────────────────────────────────────────────

// WireWriter wraps an io.Writer with Vellum encoding helpers.
type WireWriter struct {
	w   io.Writer
	buf [8]byte // scratch buffer for fixed-size writes
	err error   // sticky error
}

// NewWireWriter creates a WireWriter.
func NewWireWriter(w io.Writer) *WireWriter {
	return &WireWriter{w: w}
}

// Err returns the first error encountered during writes.
func (w *WireWriter) Err() error { return w.err }

func (w *WireWriter) write(b []byte) {
	if w.err != nil {
		return
	}
	_, w.err = w.w.Write(b)
}

func (w *WireWriter) writeByte(v byte) {
	w.buf[0] = v
	w.write(w.buf[:1])
}

func (w *WireWriter) WriteBool(v bool) {
	if v {
		w.writeByte(1)
	} else {
		w.writeByte(0)
	}
}

func (w *WireWriter) WriteUint8(v uint8) {
	w.writeByte(v)
}

func (w *WireWriter) WriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(w.buf[:4], v)
	w.write(w.buf[:4])
}

func (w *WireWriter) WriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(w.buf[:8], v)
	w.write(w.buf[:8])
}

func (w *WireWriter) WriteInt64(v int64) {
	binary.LittleEndian.PutUint64(w.buf[:8], uint64(v))
	w.write(w.buf[:8])
}

func (w *WireWriter) WriteFloat32(v float32) {
	binary.LittleEndian.PutUint32(w.buf[:4], math.Float32bits(v))
	w.write(w.buf[:4])
}

// WriteVarint writes a LEB128-encoded unsigned integer (1–5 bytes).
func (w *WireWriter) WriteVarint(v uint32) {
	for v >= 0x80 {
		w.writeByte(byte(v) | 0x80)
		v >>= 7
	}
	w.writeByte(byte(v))
}

func (w *WireWriter) WriteString(s string) {
	w.WriteVarint(uint32(len(s)))
	if len(s) > 0 {
		w.write([]byte(s))
	}
}

func (w *WireWriter) WriteRect(r draw.Rect) {
	w.WriteFloat32(r.X)
	w.WriteFloat32(r.Y)
	w.WriteFloat32(r.W)
	w.WriteFloat32(r.H)
}

func (w *WireWriter) WritePoint(p draw.Point) {
	w.WriteFloat32(p.X)
	w.WriteFloat32(p.Y)
}

// WriteColor encodes a Color as 4 bytes (R, G, B, A mapped from 0–1 to 0–255).
func (w *WireWriter) WriteColor(c draw.Color) {
	w.writeByte(byte(c.R * 255))
	w.writeByte(byte(c.G * 255))
	w.writeByte(byte(c.B * 255))
	w.writeByte(byte(c.A * 255))
}

func (w *WireWriter) WriteCornerRadii(cr draw.CornerRadii) {
	w.WriteFloat32(cr.TopLeft)
	w.WriteFloat32(cr.TopRight)
	w.WriteFloat32(cr.BottomRight)
	w.WriteFloat32(cr.BottomLeft)
}

func (w *WireWriter) WriteInsets(ins draw.Insets) {
	w.WriteFloat32(ins.Top)
	w.WriteFloat32(ins.Right)
	w.WriteFloat32(ins.Bottom)
	w.WriteFloat32(ins.Left)
	w.WriteFloat32(ins.Start)
	w.WriteFloat32(ins.End)
}

func (w *WireWriter) WriteTransform(t draw.Transform) {
	for _, v := range t {
		w.WriteFloat32(v)
	}
}

func (w *WireWriter) WriteTextStyle(ts draw.TextStyle) {
	w.WriteString(ts.FontFamily)
	w.WriteFloat32(ts.Size)
	w.WriteUint32(uint32(ts.Weight))
	w.WriteFloat32(ts.LineHeight)
	w.WriteFloat32(ts.Tracking)
	w.WriteBool(ts.Raster)
}

func (w *WireWriter) WriteTextLayout(tl draw.TextLayout) {
	w.WriteString(tl.Text)
	w.WriteTextStyle(tl.Style)
	w.WriteFloat32(tl.MaxWidth)
	w.WriteUint8(uint8(tl.Alignment))
}

func (w *WireWriter) WriteImageOptions(opts draw.ImageOptions) {
	w.WriteFloat32(opts.Opacity)
}

func (w *WireWriter) WriteImageSlice(s draw.ImageSlice) {
	w.WriteUint64(uint64(s.Image))
	w.WriteInsets(s.Insets)
}

func (w *WireWriter) WriteShadow(s draw.Shadow) {
	w.WriteColor(s.Color)
	w.WriteFloat32(s.BlurRadius)
	w.WriteFloat32(s.SpreadRadius)
	w.WriteFloat32(s.OffsetX)
	w.WriteFloat32(s.OffsetY)
	w.WriteFloat32(s.Radius)
	w.WriteBool(s.Inset)
}

func (w *WireWriter) WriteLayerOptions(opts draw.LayerOptions) {
	w.WriteUint8(uint8(opts.BlendMode))
	w.WriteFloat32(opts.Opacity)
	w.WriteBool(opts.CacheHint)
}

// WritePaint encodes a Paint tagged union.
func (w *WireWriter) WritePaint(p draw.Paint) {
	w.WriteUint8(uint8(p.Kind))
	switch p.Kind {
	case draw.PaintSolid:
		w.WriteColor(p.Color)
	case draw.PaintLinearGradient:
		if p.Linear == nil {
			w.WritePoint(draw.Point{})
			w.WritePoint(draw.Point{})
			w.WriteVarint(0)
			return
		}
		w.WritePoint(p.Linear.Start)
		w.WritePoint(p.Linear.End)
		w.WriteVarint(uint32(len(p.Linear.Stops)))
		for _, s := range p.Linear.Stops {
			w.WriteFloat32(s.Offset)
			w.WriteColor(s.Color)
		}
	case draw.PaintRadialGradient:
		if p.Radial == nil {
			w.WritePoint(draw.Point{})
			w.WriteFloat32(0)
			w.WriteVarint(0)
			return
		}
		w.WritePoint(p.Radial.Center)
		w.WriteFloat32(p.Radial.Radius)
		w.WriteVarint(uint32(len(p.Radial.Stops)))
		for _, s := range p.Radial.Stops {
			w.WriteFloat32(s.Offset)
			w.WriteColor(s.Color)
		}
	case draw.PaintPattern:
		if p.Pattern == nil {
			w.WriteUint64(0)
			w.WriteFloat32(0)
			w.WriteFloat32(0)
			return
		}
		w.WriteUint64(uint64(p.Pattern.Image))
		w.WriteFloat32(p.Pattern.TileSize.W)
		w.WriteFloat32(p.Pattern.TileSize.H)
	case draw.PaintImage:
		if p.Image == nil {
			w.WriteUint64(0)
			w.WriteUint8(0)
			return
		}
		w.WriteUint64(uint64(p.Image.Image))
		w.WriteUint8(uint8(p.Image.ScaleMode))
	case draw.PaintShader, draw.PaintShaderImage:
		if p.Shader == nil {
			w.WriteString("")
			w.WriteUint8(0)
			for i := 0; i < 8; i++ {
				w.WriteFloat32(0)
			}
			w.WriteUint64(0)
			return
		}
		w.WriteString(p.Shader.Source)
		w.WriteUint8(uint8(p.Shader.Effect))
		for _, v := range p.Shader.Params {
			w.WriteFloat32(v)
		}
		w.WriteUint64(uint64(p.Shader.Image))
	}
}

// WriteStroke encodes a Stroke.
func (w *WireWriter) WriteStroke(s draw.Stroke) {
	w.WritePaint(s.Paint)
	w.WriteFloat32(s.Width)
	w.WriteUint8(uint8(s.Cap))
	w.WriteUint8(uint8(s.Join))
	w.WriteFloat32(s.MiterLimit)
	w.WriteVarint(uint32(len(s.Dash)))
	for _, d := range s.Dash {
		w.WriteFloat32(d)
	}
	w.WriteFloat32(s.DashOffset)
}

// WritePath encodes a Path by walking its segments.
func (w *WireWriter) WritePath(p draw.Path) {
	w.WriteUint8(uint8(p.FillRule))
	// Count segments first.
	var count uint32
	p.Walk(func(draw.PathSegment) { count++ })
	w.WriteVarint(count)
	p.Walk(func(seg draw.PathSegment) {
		w.WriteUint8(uint8(seg.Kind))
		switch seg.Kind {
		case draw.SegMoveTo, draw.SegLineTo:
			w.WritePoint(seg.Points[0])
		case draw.SegQuadTo:
			w.WritePoint(seg.Points[0]) // control
			w.WritePoint(seg.Points[1]) // end
		case draw.SegCubicTo:
			w.WritePoint(seg.Points[0]) // c1
			w.WritePoint(seg.Points[1]) // c2
			w.WritePoint(seg.Points[2]) // end
		case draw.SegArcTo:
			w.WriteFloat32(seg.Arc.RX)
			w.WriteFloat32(seg.Arc.RY)
			w.WriteFloat32(seg.Arc.XRot)
			w.WriteBool(seg.Arc.Large)
			w.WriteBool(seg.Arc.Sweep)
			w.WritePoint(seg.Points[0]) // end
		case draw.SegClose:
			// no data
		}
	})
}

// ── Reader helpers ──────────────────────────────────────────────────────────

// WireReader wraps an io.Reader with Vellum decoding helpers.
type WireReader struct {
	r   io.Reader
	buf [8]byte
	err error // sticky error
}

// NewWireReader creates a WireReader.
func NewWireReader(r io.Reader) *WireReader {
	return &WireReader{r: r}
}

// Err returns the first error encountered during reads.
func (r *WireReader) Err() error { return r.err }

func (r *WireReader) read(b []byte) {
	if r.err != nil {
		return
	}
	_, r.err = io.ReadFull(r.r, b)
}

func (r *WireReader) readByte() byte {
	r.read(r.buf[:1])
	return r.buf[0]
}

func (r *WireReader) ReadBool() bool {
	return r.readByte() != 0
}

func (r *WireReader) ReadUint8() uint8 {
	return r.readByte()
}

func (r *WireReader) ReadUint32() uint32 {
	r.read(r.buf[:4])
	return binary.LittleEndian.Uint32(r.buf[:4])
}

func (r *WireReader) ReadUint64() uint64 {
	r.read(r.buf[:8])
	return binary.LittleEndian.Uint64(r.buf[:8])
}

func (r *WireReader) ReadInt64() int64 {
	return int64(r.ReadUint64())
}

func (r *WireReader) ReadFloat32() float32 {
	r.read(r.buf[:4])
	return math.Float32frombits(binary.LittleEndian.Uint32(r.buf[:4]))
}

func (r *WireReader) ReadVarint() uint32 {
	var result uint32
	var shift uint
	for {
		b := r.readByte()
		if r.err != nil {
			return 0
		}
		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result
		}
		shift += 7
		if shift >= 35 {
			r.err = fmt.Errorf("vellum: varint overflow")
			return 0
		}
	}
}

func (r *WireReader) ReadString() string {
	n := r.ReadVarint()
	if r.err != nil || n == 0 {
		return ""
	}
	buf := make([]byte, n)
	r.read(buf)
	return string(buf)
}

func (r *WireReader) ReadRect() draw.Rect {
	return draw.Rect{
		X: r.ReadFloat32(),
		Y: r.ReadFloat32(),
		W: r.ReadFloat32(),
		H: r.ReadFloat32(),
	}
}

func (r *WireReader) ReadPoint() draw.Point {
	return draw.Point{
		X: r.ReadFloat32(),
		Y: r.ReadFloat32(),
	}
}

func (r *WireReader) ReadColor() draw.Color {
	return draw.Color{
		R: float32(r.readByte()) / 255,
		G: float32(r.readByte()) / 255,
		B: float32(r.readByte()) / 255,
		A: float32(r.readByte()) / 255,
	}
}

func (r *WireReader) ReadCornerRadii() draw.CornerRadii {
	return draw.CornerRadii{
		TopLeft:     r.ReadFloat32(),
		TopRight:    r.ReadFloat32(),
		BottomRight: r.ReadFloat32(),
		BottomLeft:  r.ReadFloat32(),
	}
}

func (r *WireReader) ReadInsets() draw.Insets {
	return draw.Insets{
		Top:    r.ReadFloat32(),
		Right:  r.ReadFloat32(),
		Bottom: r.ReadFloat32(),
		Left:   r.ReadFloat32(),
		Start:  r.ReadFloat32(),
		End:    r.ReadFloat32(),
	}
}

func (r *WireReader) ReadTransform() draw.Transform {
	var t draw.Transform
	for i := range t {
		t[i] = r.ReadFloat32()
	}
	return t
}

func (r *WireReader) ReadTextStyle() draw.TextStyle {
	return draw.TextStyle{
		FontFamily: r.ReadString(),
		Size:       r.ReadFloat32(),
		Weight:     draw.FontWeight(r.ReadUint32()),
		LineHeight: r.ReadFloat32(),
		Tracking:   r.ReadFloat32(),
		Raster:     r.ReadBool(),
	}
}

func (r *WireReader) ReadTextLayout() draw.TextLayout {
	return draw.TextLayout{
		Text:      r.ReadString(),
		Style:     r.ReadTextStyle(),
		MaxWidth:  r.ReadFloat32(),
		Alignment: draw.TextAlign(r.ReadUint8()),
	}
}

func (r *WireReader) ReadImageOptions() draw.ImageOptions {
	return draw.ImageOptions{Opacity: r.ReadFloat32()}
}

func (r *WireReader) ReadImageSlice() draw.ImageSlice {
	return draw.ImageSlice{
		Image: draw.ImageID(r.ReadUint64()),
		Insets: r.ReadInsets(),
	}
}

func (r *WireReader) ReadShadow() draw.Shadow {
	return draw.Shadow{
		Color:        r.ReadColor(),
		BlurRadius:   r.ReadFloat32(),
		SpreadRadius: r.ReadFloat32(),
		OffsetX:      r.ReadFloat32(),
		OffsetY:      r.ReadFloat32(),
		Radius:       r.ReadFloat32(),
		Inset:        r.ReadBool(),
	}
}

func (r *WireReader) ReadLayerOptions() draw.LayerOptions {
	return draw.LayerOptions{
		BlendMode: draw.BlendMode(r.ReadUint8()),
		Opacity:   r.ReadFloat32(),
		CacheHint: r.ReadBool(),
	}
}

func (r *WireReader) ReadPaint() draw.Paint {
	kind := draw.PaintKind(r.ReadUint8())
	p := draw.Paint{Kind: kind}
	switch kind {
	case draw.PaintSolid:
		p.Color = r.ReadColor()
	case draw.PaintLinearGradient:
		lg := &draw.LinearGradient{
			Start: r.ReadPoint(),
			End:   r.ReadPoint(),
		}
		n := r.ReadVarint()
		lg.Stops = make([]draw.GradientStop, n)
		for i := range lg.Stops {
			lg.Stops[i].Offset = r.ReadFloat32()
			lg.Stops[i].Color = r.ReadColor()
		}
		p.Linear = lg
	case draw.PaintRadialGradient:
		rg := &draw.RadialGradient{
			Center: r.ReadPoint(),
			Radius: r.ReadFloat32(),
		}
		n := r.ReadVarint()
		rg.Stops = make([]draw.GradientStop, n)
		for i := range rg.Stops {
			rg.Stops[i].Offset = r.ReadFloat32()
			rg.Stops[i].Color = r.ReadColor()
		}
		p.Radial = rg
	case draw.PaintPattern:
		p.Pattern = &draw.PatternDesc{
			Image:    draw.ImageID(r.ReadUint64()),
			TileSize: draw.Size{W: r.ReadFloat32(), H: r.ReadFloat32()},
		}
	case draw.PaintImage:
		p.Image = &draw.ImageFill{
			Image:     draw.ImageID(r.ReadUint64()),
			ScaleMode: draw.ImageScaleMode(r.ReadUint8()),
		}
	case draw.PaintShader, draw.PaintShaderImage:
		sd := &draw.ShaderDesc{
			Source: r.ReadString(),
			Effect: draw.ShaderEffect(r.ReadUint8()),
		}
		for i := range sd.Params {
			sd.Params[i] = r.ReadFloat32()
		}
		sd.Image = draw.ImageID(r.ReadUint64())
		p.Shader = sd
	}
	return p
}

func (r *WireReader) ReadStroke() draw.Stroke {
	s := draw.Stroke{
		Paint:      r.ReadPaint(),
		Width:      r.ReadFloat32(),
		Cap:        draw.StrokeCap(r.ReadUint8()),
		Join:       draw.StrokeJoin(r.ReadUint8()),
		MiterLimit: r.ReadFloat32(),
	}
	n := r.ReadVarint()
	if n > 0 {
		s.Dash = make([]float32, n)
		for i := range s.Dash {
			s.Dash[i] = r.ReadFloat32()
		}
	}
	s.DashOffset = r.ReadFloat32()
	return s
}

func (r *WireReader) ReadPath() draw.Path {
	fillRule := draw.FillRule(r.ReadUint8())
	n := r.ReadVarint()
	if r.err != nil || n == 0 {
		return draw.Path{}
	}
	b := draw.NewPath()
	b.SetFillRule(fillRule)
	for i := uint32(0); i < n; i++ {
		kind := draw.PathSegmentKind(r.ReadUint8())
		switch kind {
		case draw.SegMoveTo:
			b.MoveTo(r.ReadPoint())
		case draw.SegLineTo:
			b.LineTo(r.ReadPoint())
		case draw.SegQuadTo:
			ctrl := r.ReadPoint()
			end := r.ReadPoint()
			b.QuadTo(ctrl, end)
		case draw.SegCubicTo:
			c1 := r.ReadPoint()
			c2 := r.ReadPoint()
			end := r.ReadPoint()
			b.CubicTo(c1, c2, end)
		case draw.SegArcTo:
			rx := r.ReadFloat32()
			ry := r.ReadFloat32()
			xRot := r.ReadFloat32()
			large := r.ReadBool()
			sweep := r.ReadBool()
			end := r.ReadPoint()
			b.ArcTo(rx, ry, xRot, large, sweep, end)
		case draw.SegClose:
			b.Close()
		}
		if r.err != nil {
			return draw.Path{}
		}
	}
	return b.Build()
}
