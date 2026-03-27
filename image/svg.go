package image

import (
	"encoding/xml"
	"fmt"
	goimage "image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/timzifer/lux/draw"
	"golang.org/x/image/vector"
)

// SVGImage represents a vector image that can be rasterized at any resolution.
type SVGImage struct {
	ID     draw.ImageID
	source []byte
}

// svgDocument holds a parsed SVG tree for later rasterization.
type svgDocument struct {
	Width    float32
	Height   float32
	ViewBox  [4]float32 // minX, minY, width, height
	Elements []svgElement
}

type svgElementKind uint8

const (
	svgPath svgElementKind = iota
	svgRect
	svgCircle
	svgEllipse
	svgLine
	svgPolygon
	svgPolyline
	svgGroup
)

type svgElement struct {
	Kind        svgElementKind
	Path        draw.Path
	Fill        draw.Color
	HasFill     bool
	Stroke      draw.Color
	HasStroke   bool
	StrokeWidth float32
	Children    []svgElement
}

// LoadSVG parses SVG source data and registers it for later rasterization.
// Returns an ImageID handle that can be passed to RasterizeSVG.
func (s *Store) LoadSVG(data []byte) (draw.ImageID, error) {
	doc, err := parseSVGDocument(data)
	if err != nil {
		return 0, fmt.Errorf("image: SVG parse: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++
	s.svgImages[id] = doc
	return id, nil
}

// RasterizeSVG rasterizes a previously loaded SVG at the given resolution.
// Returns a new ImageID with the rasterized bitmap.
func (s *Store) RasterizeSVG(id draw.ImageID, width, height int) (draw.ImageID, error) {
	s.mu.RLock()
	doc := s.svgImages[id]
	s.mu.RUnlock()

	if doc == nil {
		return 0, fmt.Errorf("image: SVG id %d not found", id)
	}
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("image: invalid rasterization size %dx%d", width, height)
	}

	rgba := rasterizeSVGDoc(doc, width, height)
	return s.LoadFromRGBA(width, height, rgba)
}

// parseSVGDocument parses SVG XML into an svgDocument.
func parseSVGDocument(data []byte) (*svgDocument, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	doc := &svgDocument{
		Width:  100,
		Height: 100,
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "svg" {
				parseSVGRoot(se, doc)
				doc.Elements = parseSVGChildren(decoder)
				return doc, nil
			}
		}
	}

	return doc, nil
}

func parseSVGRoot(se xml.StartElement, doc *svgDocument) {
	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "width":
			doc.Width = parseLength(attr.Value)
		case "height":
			doc.Height = parseLength(attr.Value)
		case "viewBox":
			parts := strings.Fields(attr.Value)
			if len(parts) == 4 {
				for i, p := range parts {
					v, _ := strconv.ParseFloat(p, 32)
					doc.ViewBox[i] = float32(v)
				}
			}
		}
	}
	// If no viewBox specified, use width/height.
	if doc.ViewBox[2] == 0 && doc.ViewBox[3] == 0 {
		doc.ViewBox[2] = doc.Width
		doc.ViewBox[3] = doc.Height
	}
}

func parseSVGChildren(decoder *xml.Decoder) []svgElement {
	var elements []svgElement
	depth := 1

	for depth > 0 {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := tok.(type) {
		case xml.StartElement:
			elem, ok := parseSVGElement(se, decoder)
			if ok {
				elements = append(elements, elem)
			} else {
				depth++ // unknown element, track depth
			}
		case xml.EndElement:
			depth--
		}
	}
	return elements
}

func parseSVGElement(se xml.StartElement, decoder *xml.Decoder) (svgElement, bool) {
	var elem svgElement
	elem.Fill = draw.Color{R: 0, G: 0, B: 0, A: 1} // default black fill
	elem.HasFill = true
	elem.StrokeWidth = 1

	// Parse common attributes.
	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "fill":
			if attr.Value == "none" {
				elem.HasFill = false
			} else {
				elem.Fill = parseSVGColor(attr.Value)
				elem.HasFill = true
			}
		case "stroke":
			if attr.Value == "none" {
				elem.HasStroke = false
			} else {
				elem.Stroke = parseSVGColor(attr.Value)
				elem.HasStroke = true
			}
		case "stroke-width":
			v, _ := strconv.ParseFloat(attr.Value, 32)
			elem.StrokeWidth = float32(v)
		}
	}

	switch se.Name.Local {
	case "path":
		elem.Kind = svgPath
		for _, attr := range se.Attr {
			if attr.Name.Local == "d" {
				p, err := parseSVGPath(attr.Value)
				if err == nil {
					elem.Path = p
				}
			}
		}
		skipElement(decoder)
		return elem, true

	case "rect":
		elem.Kind = svgRect
		var x, y, w, h float32
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "x":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				x = float32(v)
			case "y":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				y = float32(v)
			case "width":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				w = float32(v)
			case "height":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				h = float32(v)
			}
		}
		elem.Path = draw.PathFromRect(draw.R(x, y, w, h))
		skipElement(decoder)
		return elem, true

	case "circle":
		elem.Kind = svgCircle
		var cx, cy, r float32
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "cx":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				cx = float32(v)
			case "cy":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				cy = float32(v)
			case "r":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				r = float32(v)
			}
		}
		elem.Path = circlePath(cx, cy, r)
		skipElement(decoder)
		return elem, true

	case "ellipse":
		elem.Kind = svgEllipse
		var cx, cy, rx, ry float32
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "cx":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				cx = float32(v)
			case "cy":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				cy = float32(v)
			case "rx":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				rx = float32(v)
			case "ry":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				ry = float32(v)
			}
		}
		elem.Path = ellipsePath(cx, cy, rx, ry)
		skipElement(decoder)
		return elem, true

	case "line":
		elem.Kind = svgLine
		var x1, y1, x2, y2 float32
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "x1":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				x1 = float32(v)
			case "y1":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				y1 = float32(v)
			case "x2":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				x2 = float32(v)
			case "y2":
				v, _ := strconv.ParseFloat(attr.Value, 32)
				y2 = float32(v)
			}
		}
		elem.Path = draw.NewPath().MoveTo(draw.Pt(x1, y1)).LineTo(draw.Pt(x2, y2)).Build()
		elem.HasFill = false // lines are stroked by default
		if !elem.HasStroke {
			elem.HasStroke = true
			elem.Stroke = draw.Color{R: 0, G: 0, B: 0, A: 1}
		}
		skipElement(decoder)
		return elem, true

	case "polygon", "polyline":
		if se.Name.Local == "polygon" {
			elem.Kind = svgPolygon
		} else {
			elem.Kind = svgPolyline
		}
		for _, attr := range se.Attr {
			if attr.Name.Local == "points" {
				elem.Path = parsePoints(attr.Value, se.Name.Local == "polygon")
			}
		}
		skipElement(decoder)
		return elem, true

	case "g":
		elem.Kind = svgGroup
		elem.Children = parseSVGChildren(decoder)
		return elem, true
	}

	return svgElement{}, false
}

func skipElement(decoder *xml.Decoder) {
	depth := 1
	for depth > 0 {
		tok, err := decoder.Token()
		if err != nil {
			return
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
}

// circlePath creates a path approximating a circle using 4 cubic Beziers.
func circlePath(cx, cy, r float32) draw.Path {
	return ellipsePath(cx, cy, r, r)
}

// ellipsePath creates a path approximating an ellipse using 4 cubic Beziers.
func ellipsePath(cx, cy, rx, ry float32) draw.Path {
	// Kappa constant for approximating a quarter-circle with a cubic Bezier.
	const k = 0.5522847498
	kx := float32(k) * rx
	ky := float32(k) * ry

	return draw.NewPath().
		MoveTo(draw.Pt(cx+rx, cy)).
		CubicTo(draw.Pt(cx+rx, cy+ky), draw.Pt(cx+kx, cy+ry), draw.Pt(cx, cy+ry)).
		CubicTo(draw.Pt(cx-kx, cy+ry), draw.Pt(cx-rx, cy+ky), draw.Pt(cx-rx, cy)).
		CubicTo(draw.Pt(cx-rx, cy-ky), draw.Pt(cx-kx, cy-ry), draw.Pt(cx, cy-ry)).
		CubicTo(draw.Pt(cx+kx, cy-ry), draw.Pt(cx+rx, cy-ky), draw.Pt(cx+rx, cy)).
		Close().Build()
}

func parsePoints(s string, close bool) draw.Path {
	b := draw.NewPath()
	fields := strings.Fields(s)
	first := true
	for _, f := range fields {
		parts := strings.Split(f, ",")
		if len(parts) != 2 {
			continue
		}
		x, err1 := strconv.ParseFloat(parts[0], 32)
		y, err2 := strconv.ParseFloat(parts[1], 32)
		if err1 != nil || err2 != nil {
			continue
		}
		if first {
			b.MoveTo(draw.Pt(float32(x), float32(y)))
			first = false
		} else {
			b.LineTo(draw.Pt(float32(x), float32(y)))
		}
	}
	if close {
		b.Close()
	}
	return b.Build()
}

func parseSVGColor(s string) draw.Color {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		switch len(hex) {
		case 3:
			r := parseHexByte(hex[0:1] + hex[0:1])
			g := parseHexByte(hex[1:2] + hex[1:2])
			b := parseHexByte(hex[2:3] + hex[2:3])
			return draw.RGBA(r, g, b, 255)
		case 6:
			r := parseHexByte(hex[0:2])
			g := parseHexByte(hex[2:4])
			b := parseHexByte(hex[4:6])
			return draw.RGBA(r, g, b, 255)
		}
	}
	// Named colors (common subset).
	switch strings.ToLower(s) {
	case "black":
		return draw.RGBA(0, 0, 0, 255)
	case "white":
		return draw.RGBA(255, 255, 255, 255)
	case "red":
		return draw.RGBA(255, 0, 0, 255)
	case "green":
		return draw.RGBA(0, 128, 0, 255)
	case "blue":
		return draw.RGBA(0, 0, 255, 255)
	case "yellow":
		return draw.RGBA(255, 255, 0, 255)
	case "orange":
		return draw.RGBA(255, 165, 0, 255)
	case "gray", "grey":
		return draw.RGBA(128, 128, 128, 255)
	}
	return draw.Color{A: 1} // default: black
}

func parseHexByte(s string) uint8 {
	v, _ := strconv.ParseUint(s, 16, 8)
	return uint8(v)
}

func parseLength(s string) float32 {
	// Strip common units.
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	s = strings.TrimSuffix(s, "em")
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 32)
	return float32(v)
}

// rasterizeSVGDoc renders all SVG elements onto an RGBA bitmap.
// Uses golang.org/x/image/vector for anti-aliased path rasterization.
func rasterizeSVGDoc(doc *svgDocument, width, height int) []byte {
	dst := goimage.NewNRGBA(goimage.Rect(0, 0, width, height))

	// Compute scale from viewBox to target dimensions.
	scaleX := float32(width) / doc.ViewBox[2]
	scaleY := float32(height) / doc.ViewBox[3]
	offsetX := -doc.ViewBox[0] * scaleX
	offsetY := -doc.ViewBox[1] * scaleY

	rasterizeElements(dst, doc.Elements, scaleX, scaleY, offsetX, offsetY)

	return dst.Pix
}

func rasterizeElements(dst *goimage.NRGBA, elements []svgElement, sx, sy, ox, oy float32) {
	for _, elem := range elements {
		if elem.Kind == svgGroup {
			rasterizeElements(dst, elem.Children, sx, sy, ox, oy)
			continue
		}

		if elem.HasFill {
			rasterizePathFill(dst, elem.Path, elem.Fill, sx, sy, ox, oy)
		}
		if elem.HasStroke {
			rasterizePathStroke(dst, elem.Path, elem.Stroke, elem.StrokeWidth*sx, sx, sy, ox, oy)
		}
	}
}

// rasterizePathFill renders a filled path onto the destination image.
func rasterizePathFill(dst *goimage.NRGBA, p draw.Path, c draw.Color, sx, sy, ox, oy float32) {
	if p.Empty() {
		return
	}
	w := dst.Bounds().Dx()
	h := dst.Bounds().Dy()
	r := vector.NewRasterizer(w, h)

	var cursor draw.Point
	p.Walk(func(seg draw.PathSegment) {
		switch seg.Kind {
		case draw.SegMoveTo:
			pt := transformPt(seg.Points[0], sx, sy, ox, oy)
			r.MoveTo(pt.X, pt.Y)
			cursor = seg.Points[0]
		case draw.SegLineTo:
			pt := transformPt(seg.Points[0], sx, sy, ox, oy)
			r.LineTo(pt.X, pt.Y)
			cursor = seg.Points[0]
		case draw.SegQuadTo:
			c1 := transformPt(seg.Points[0], sx, sy, ox, oy)
			end := transformPt(seg.Points[1], sx, sy, ox, oy)
			r.QuadTo(c1.X, c1.Y, end.X, end.Y)
			cursor = seg.Points[1]
		case draw.SegCubicTo:
			c1 := transformPt(seg.Points[0], sx, sy, ox, oy)
			c2 := transformPt(seg.Points[1], sx, sy, ox, oy)
			end := transformPt(seg.Points[2], sx, sy, ox, oy)
			r.CubeTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
			cursor = seg.Points[2]
		case draw.SegArcTo:
			// Convert arc to cubics, then rasterize.
			arcCubics := arcToCubicsForRaster(
				seg.Arc.RX, seg.Arc.RY, seg.Arc.XRot,
				seg.Arc.Large, seg.Arc.Sweep,
				cursor, seg.Points[0],
			)
			for _, cb := range arcCubics {
				c1 := transformPt(cb[0], sx, sy, ox, oy)
				c2 := transformPt(cb[1], sx, sy, ox, oy)
				end := transformPt(cb[2], sx, sy, ox, oy)
				r.CubeTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
			}
			cursor = seg.Points[0]
		case draw.SegClose:
			r.ClosePath()
		}
	})

	// Rasterize into an alpha mask, then composite with the fill color.
	mask := goimage.NewAlpha(goimage.Rect(0, 0, w, h))
	r.Draw(mask, mask.Bounds(), goimage.White, goimage.Point{})

	cr := uint8(c.R * 255)
	cg := uint8(c.G * 255)
	cb := uint8(c.B * 255)
	ca := c.A

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := mask.AlphaAt(x, y).A
			if a == 0 {
				continue
			}
			alpha := float32(a) / 255 * ca
			existing := dst.NRGBAAt(x, y)
			// Alpha-composite over existing pixel.
			srcR := float32(cr) * alpha
			srcG := float32(cg) * alpha
			srcB := float32(cb) * alpha
			srcA := alpha * 255
			invA := 1 - alpha
			dst.SetNRGBA(x, y, color.NRGBA{
				R: clampByte(srcR + float32(existing.R)*invA),
				G: clampByte(srcG + float32(existing.G)*invA),
				B: clampByte(srcB + float32(existing.B)*invA),
				A: clampByte(srcA + float32(existing.A)*invA),
			})
		}
	}
}

// rasterizePathStroke renders a stroked path by expanding it and filling.
// For SVG rasterization, we use a simple approach: draw the stroke using
// the vector rasterizer with a thick line approximation.
func rasterizePathStroke(dst *goimage.NRGBA, p draw.Path, c draw.Color, strokeW, sx, sy, ox, oy float32) {
	if p.Empty() || strokeW <= 0 {
		return
	}
	// For strokes, we rasterize each segment as a filled rectangle
	// (line segments) or delegate to the fill path with expanded outline.
	// Simple approximation: use the fill rasterizer with the path offset by ±halfW.
	// For a proper implementation, we would generate the stroke outline.
	// For now, draw each line segment as a filled rectangle.
	w := dst.Bounds().Dx()
	h := dst.Bounds().Dy()
	halfW := strokeW / 2

	var segments []lineSeg
	var cursor, subStart draw.Point

	p.Walk(func(seg draw.PathSegment) {
		switch seg.Kind {
		case draw.SegMoveTo:
			cursor = seg.Points[0]
			subStart = cursor
		case draw.SegLineTo:
			segments = append(segments, lineSeg{cursor, seg.Points[0]})
			cursor = seg.Points[0]
		case draw.SegQuadTo, draw.SegCubicTo, draw.SegArcTo:
			// Approximate: just draw line to endpoint.
			end := seg.Points[0]
			if seg.Kind == draw.SegCubicTo {
				end = seg.Points[2]
			} else if seg.Kind == draw.SegQuadTo {
				end = seg.Points[1]
			}
			segments = append(segments, lineSeg{cursor, end})
			cursor = end
		case draw.SegClose:
			if cursor != subStart {
				segments = append(segments, lineSeg{cursor, subStart})
			}
			cursor = subStart
		}
	})

	// Rasterize each segment as a thick line.
	r := vector.NewRasterizer(w, h)
	for _, seg := range segments {
		a := transformPt(seg.a, sx, sy, ox, oy)
		b := transformPt(seg.b, sx, sy, ox, oy)
		dx := b.X - a.X
		dy := b.Y - a.Y
		ln := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if ln < 1e-6 {
			continue
		}
		nx := -dy / ln * halfW
		ny := dx / ln * halfW

		r.MoveTo(a.X+nx, a.Y+ny)
		r.LineTo(b.X+nx, b.Y+ny)
		r.LineTo(b.X-nx, b.Y-ny)
		r.LineTo(a.X-nx, a.Y-ny)
		r.ClosePath()
	}

	mask := goimage.NewAlpha(goimage.Rect(0, 0, w, h))
	r.Draw(mask, mask.Bounds(), goimage.White, goimage.Point{})

	cr := uint8(c.R * 255)
	cg := uint8(c.G * 255)
	cb := uint8(c.B * 255)
	ca := c.A

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := mask.AlphaAt(x, y).A
			if a == 0 {
				continue
			}
			alpha := float32(a) / 255 * ca
			existing := dst.NRGBAAt(x, y)
			srcR := float32(cr) * alpha
			srcG := float32(cg) * alpha
			srcB := float32(cb) * alpha
			srcA := alpha * 255
			invA := 1 - alpha
			dst.SetNRGBA(x, y, color.NRGBA{
				R: clampByte(srcR + float32(existing.R)*invA),
				G: clampByte(srcG + float32(existing.G)*invA),
				B: clampByte(srcB + float32(existing.B)*invA),
				A: clampByte(srcA + float32(existing.A)*invA),
			})
		}
	}
}

type lineSeg struct {
	a, b draw.Point
}

func transformPt(p draw.Point, sx, sy, ox, oy float32) draw.Point {
	return draw.Point{X: p.X*sx + ox, Y: p.Y*sy + oy}
}

func clampByte(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// arcToCubicsForRaster converts an SVG elliptical arc to cubic Bezier segments
// for the rasterizer. Same algorithm as in internal/render/tessellate.go.
func arcToCubicsForRaster(rx, ry, xRotDeg float32, largeArc, sweep bool, from, to draw.Point) [][3]draw.Point {
	if rx == 0 || ry == 0 || from == to {
		return nil
	}
	if rx < 0 {
		rx = -rx
	}
	if ry < 0 {
		ry = -ry
	}

	xRot := float64(xRotDeg) * math.Pi / 180.0
	cosR := float32(math.Cos(xRot))
	sinR := float32(math.Sin(xRot))

	dx := (from.X - to.X) / 2
	dy := (from.Y - to.Y) / 2
	x1p := cosR*dx + sinR*dy
	y1p := -sinR*dx + cosR*dy

	x1p2 := x1p * x1p
	y1p2 := y1p * y1p
	rx2 := rx * rx
	ry2 := ry * ry

	lambda := x1p2/rx2 + y1p2/ry2
	if lambda > 1 {
		s := float32(math.Sqrt(float64(lambda)))
		rx *= s
		ry *= s
		rx2 = rx * rx
		ry2 = ry * ry
	}

	num := rx2*ry2 - rx2*y1p2 - ry2*x1p2
	den := rx2*y1p2 + ry2*x1p2
	if den < 1e-10 {
		return nil
	}
	sq := float32(0)
	if num > 0 {
		sq = float32(math.Sqrt(float64(num / den)))
	}
	if largeArc == sweep {
		sq = -sq
	}
	cxp := sq * rx * y1p / ry
	cyp := -sq * ry * x1p / rx

	mx := (from.X + to.X) / 2
	my := (from.Y + to.Y) / 2
	cx := cosR*cxp - sinR*cyp + mx
	cy := sinR*cxp + cosR*cyp + my

	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := (-x1p - cxp) / rx
	vy := (-y1p - cyp) / ry

	theta1 := math.Atan2(float64(uy), float64(ux))
	dtheta := math.Atan2(float64(ux*vy-uy*vx), float64(ux*vx+uy*vy))

	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	nSegs := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	if nSegs == 0 {
		return nil
	}
	segAngle := dtheta / float64(nSegs)

	var cubics [][3]draw.Point
	for i := 0; i < nSegs; i++ {
		a1 := float32(theta1 + float64(i)*segAngle)
		a2 := float32(theta1 + float64(i+1)*segAngle)
		cubics = append(cubics, arcSegToCubicForRaster(cx, cy, rx, ry, cosR, sinR, a1, a2))
	}
	return cubics
}

func arcSegToCubicForRaster(cx, cy, rx, ry, cosR, sinR, a1, a2 float32) [3]draw.Point {
	da := a2 - a1
	t := float32(math.Tan(float64(da / 2)))
	alpha := float32(math.Sin(float64(da))) * (float32(math.Sqrt(float64(4+3*t*t))) - 1) / 3

	cos1 := float32(math.Cos(float64(a1)))
	sin1 := float32(math.Sin(float64(a1)))
	cos2 := float32(math.Cos(float64(a2)))
	sin2 := float32(math.Sin(float64(a2)))

	ex1 := rx * cos1
	ey1 := ry * sin1
	dx1 := -rx * sin1
	dy1 := ry * cos1

	ex2 := rx * cos2
	ey2 := ry * sin2
	dx2 := -rx * sin2
	dy2 := ry * cos2

	cp1x := ex1 + alpha*dx1
	cp1y := ey1 + alpha*dy1
	cp2x := ex2 - alpha*dx2
	cp2y := ey2 - alpha*dy2

	return [3]draw.Point{
		{X: cosR*cp1x - sinR*cp1y + cx, Y: sinR*cp1x + cosR*cp1y + cy},
		{X: cosR*cp2x - sinR*cp2y + cx, Y: sinR*cp2x + cosR*cp2y + cy},
		{X: cosR*ex2 - sinR*ey2 + cx, Y: sinR*ex2 + cosR*ey2 + cy},
	}
}
