package image

import (
	"fmt"
	"math"
	"strconv"

	"github.com/timzifer/lux/draw"
)

// parseSVGPath parses an SVG path "d" attribute string into a draw.Path.
// Supports all SVG path commands: M/m, L/l, H/h, V/v, C/c, S/s, Q/q, T/t, A/a, Z/z.
func parseSVGPath(d string) (draw.Path, error) {
	p := &svgPathParser{input: d}
	return p.parse()
}

type svgPathParser struct {
	input string
	pos   int
}

func (p *svgPathParser) parse() (draw.Path, error) {
	b := draw.NewPath()
	var cx, cy float32     // current point
	var sx, sy float32     // subpath start
	var lcx, lcy float32   // last control point (for S/T)
	var lastCmd byte

	for {
		p.skipWhitespaceAndCommas()
		if p.pos >= len(p.input) {
			break
		}

		ch := p.input[p.pos]
		if !isCommand(ch) {
			// Implicit repetition of previous command.
			if lastCmd == 0 {
				return draw.Path{}, fmt.Errorf("svgpath: unexpected character %q at position %d", ch, p.pos)
			}
			ch = lastCmd
			// M/m becomes L/l on repetition.
			if ch == 'M' {
				ch = 'L'
			} else if ch == 'm' {
				ch = 'l'
			}
		} else {
			p.pos++
		}

		rel := ch >= 'a' && ch <= 'z'
		cmd := upper(ch)

		switch cmd {
		case 'M':
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x += cx
				y += cy
			}
			b.MoveTo(draw.Pt(x, y))
			cx, cy = x, y
			sx, sy = x, y
			lastCmd = ch

		case 'L':
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x += cx
				y += cy
			}
			b.LineTo(draw.Pt(x, y))
			cx, cy = x, y
			lastCmd = ch

		case 'H':
			x, err := p.readFloat()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x += cx
			}
			b.LineTo(draw.Pt(x, cy))
			cx = x
			lastCmd = ch

		case 'V':
			y, err := p.readFloat()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				y += cy
			}
			b.LineTo(draw.Pt(cx, y))
			cy = y
			lastCmd = ch

		case 'C':
			x1, y1, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			x2, y2, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x1 += cx
				y1 += cy
				x2 += cx
				y2 += cy
				x += cx
				y += cy
			}
			b.CubicTo(draw.Pt(x1, y1), draw.Pt(x2, y2), draw.Pt(x, y))
			lcx, lcy = x2, y2
			cx, cy = x, y
			lastCmd = ch

		case 'S':
			// Smooth cubic: reflect last control point.
			x2, y2, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x2 += cx
				y2 += cy
				x += cx
				y += cy
			}
			x1 := 2*cx - lcx
			y1 := 2*cy - lcy
			if lastCmd != 'C' && lastCmd != 'c' && lastCmd != 'S' && lastCmd != 's' {
				x1, y1 = cx, cy
			}
			b.CubicTo(draw.Pt(x1, y1), draw.Pt(x2, y2), draw.Pt(x, y))
			lcx, lcy = x2, y2
			cx, cy = x, y
			lastCmd = ch

		case 'Q':
			x1, y1, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x1 += cx
				y1 += cy
				x += cx
				y += cy
			}
			b.QuadTo(draw.Pt(x1, y1), draw.Pt(x, y))
			lcx, lcy = x1, y1
			cx, cy = x, y
			lastCmd = ch

		case 'T':
			// Smooth quadratic: reflect last control point.
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x += cx
				y += cy
			}
			x1 := 2*cx - lcx
			y1 := 2*cy - lcy
			if lastCmd != 'Q' && lastCmd != 'q' && lastCmd != 'T' && lastCmd != 't' {
				x1, y1 = cx, cy
			}
			b.QuadTo(draw.Pt(x1, y1), draw.Pt(x, y))
			lcx, lcy = x1, y1
			cx, cy = x, y
			lastCmd = ch

		case 'A':
			rx, err := p.readFloat()
			if err != nil {
				return draw.Path{}, err
			}
			ry, err := p.readFloat()
			if err != nil {
				return draw.Path{}, err
			}
			xRot, err := p.readFloat()
			if err != nil {
				return draw.Path{}, err
			}
			largeArc, err := p.readFlag()
			if err != nil {
				return draw.Path{}, err
			}
			sweep, err := p.readFlag()
			if err != nil {
				return draw.Path{}, err
			}
			x, y, err := p.readCoordPair()
			if err != nil {
				return draw.Path{}, err
			}
			if rel {
				x += cx
				y += cy
			}
			b.ArcTo(float32(math.Abs(float64(rx))), float32(math.Abs(float64(ry))), xRot, largeArc, sweep, draw.Pt(x, y))
			cx, cy = x, y
			lastCmd = ch

		case 'Z':
			b.Close()
			cx, cy = sx, sy
			lastCmd = ch

		default:
			return draw.Path{}, fmt.Errorf("svgpath: unknown command %q at position %d", cmd, p.pos-1)
		}

		// Reset last control point for non-curve commands.
		if cmd != 'C' && cmd != 'S' && cmd != 'Q' && cmd != 'T' {
			lcx, lcy = cx, cy
		}
	}

	return b.Build(), nil
}

func (p *svgPathParser) skipWhitespaceAndCommas() {
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ',' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *svgPathParser) readFloat() (float32, error) {
	p.skipWhitespaceAndCommas()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("svgpath: unexpected end of input")
	}

	start := p.pos
	// Optional sign.
	if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
		p.pos++
	}
	// Integer part.
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	// Fractional part.
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.pos++
		}
	}
	// Exponent.
	if p.pos < len(p.input) && (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
			p.pos++
		}
		for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.pos++
		}
	}

	if p.pos == start {
		return 0, fmt.Errorf("svgpath: expected number at position %d", p.pos)
	}

	v, err := strconv.ParseFloat(p.input[start:p.pos], 32)
	if err != nil {
		return 0, fmt.Errorf("svgpath: invalid number %q: %w", p.input[start:p.pos], err)
	}
	return float32(v), nil
}

func (p *svgPathParser) readCoordPair() (float32, float32, error) {
	x, err := p.readFloat()
	if err != nil {
		return 0, 0, err
	}
	y, err := p.readFloat()
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func (p *svgPathParser) readFlag() (bool, error) {
	p.skipWhitespaceAndCommas()
	if p.pos >= len(p.input) {
		return false, fmt.Errorf("svgpath: unexpected end of input (expected flag)")
	}
	ch := p.input[p.pos]
	if ch == '0' {
		p.pos++
		return false, nil
	}
	if ch == '1' {
		p.pos++
		return true, nil
	}
	return false, fmt.Errorf("svgpath: expected flag (0 or 1) at position %d, got %q", p.pos, ch)
}

func isCommand(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func upper(ch byte) byte {
	if ch >= 'a' && ch <= 'z' {
		return ch - 32
	}
	return ch
}

