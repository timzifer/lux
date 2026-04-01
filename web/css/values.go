package css

import (
	"strconv"
	"strings"

	"github.com/timzifer/lux/draw"
)

// ParseFontSize parses a CSS font-size value and returns the size in dp.
// Supports named sizes (xx-small through xx-large), em units, and
// absolute units (px, dp, pt).
func ParseFontSize(v string) (float32, bool) {
	switch v {
	case "xx-small":
		return 8, true
	case "x-small":
		return 10, true
	case "small":
		return 12, true
	case "medium":
		return 14, true
	case "large":
		return 16, true
	case "x-large":
		return 20, true
	case "xx-large":
		return 24, true
	}
	// Relative em sizes (multiply by a base of 14dp).
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(v[:len(v)-2], 32); err == nil {
			return float32(f) * 14, true
		}
	}
	return ParseDimension(v)
}

// ParseDimension parses a CSS length value (e.g. "12px", "1.5em", "16pt")
// and returns the numeric value in dp. Known unit suffixes (px, dp, pt,
// em, rem) are stripped; the raw number is returned.
func ParseDimension(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	for _, suffix := range []string{"px", "dp", "pt", "em", "rem"} {
		if strings.HasSuffix(v, suffix) {
			v = v[:len(v)-len(suffix)]
			break
		}
	}
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return 0, false
	}
	return float32(f), true
}

// ParseLineHeight parses a CSS line-height value.
// Returns the multiplier (unitless or dimension-derived).
// "normal" returns (0, false) to signal inheritance.
func ParseLineHeight(v string) (float32, bool) {
	if v == "normal" {
		return 0, false
	}
	// Unitless number is a multiplier.
	if f, err := strconv.ParseFloat(v, 32); err == nil {
		return float32(f), true
	}
	return ParseDimension(v)
}

// ParseTextAlign parses a CSS text-align value.
func ParseTextAlign(v string) (draw.TextAlign, bool) {
	switch v {
	case "left", "start":
		return draw.TextAlignLeft, true
	case "center":
		return draw.TextAlignCenter, true
	case "right", "end":
		return draw.TextAlignRight, true
	case "justify":
		return draw.TextAlignJustify, true
	}
	return 0, false
}

// ParseFontWeight parses a CSS font-weight value into a draw.FontWeight
// and a boolean indicating whether the weight should be considered bold.
func ParseFontWeight(v string) (weight draw.FontWeight, bold bool, ok bool) {
	switch v {
	case "bold":
		return draw.FontWeightBold, true, true
	case "normal":
		return draw.FontWeightRegular, false, true
	case "lighter":
		return draw.FontWeightLight, false, true
	case "bolder":
		return draw.FontWeightBold, true, true
	default:
		if w, err := strconv.Atoi(v); err == nil {
			return draw.FontWeight(w), w >= 700, true
		}
	}
	return 0, false, false
}

// ParseBoxDimensions parses CSS shorthand values with 1-4 components
// (used by margin, padding, border-width). Returns [top, right, bottom, left].
func ParseBoxDimensions(v string) ([4]float32, bool) {
	parts := strings.Fields(v)
	var vals [4]float32
	switch len(parts) {
	case 1:
		d, ok := ParseDimension(parts[0])
		if !ok {
			return vals, false
		}
		vals = [4]float32{d, d, d, d}
	case 2:
		tb, ok1 := ParseDimension(parts[0])
		lr, ok2 := ParseDimension(parts[1])
		if !ok1 || !ok2 {
			return vals, false
		}
		vals = [4]float32{tb, lr, tb, lr}
	case 3:
		t, ok1 := ParseDimension(parts[0])
		lr, ok2 := ParseDimension(parts[1])
		b, ok3 := ParseDimension(parts[2])
		if !ok1 || !ok2 || !ok3 {
			return vals, false
		}
		vals = [4]float32{t, lr, b, lr}
	case 4:
		t, ok1 := ParseDimension(parts[0])
		r, ok2 := ParseDimension(parts[1])
		b, ok3 := ParseDimension(parts[2])
		l, ok4 := ParseDimension(parts[3])
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return vals, false
		}
		vals = [4]float32{t, r, b, l}
	default:
		return vals, false
	}
	return vals, true
}
