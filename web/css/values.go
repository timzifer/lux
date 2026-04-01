package css

import (
	"strconv"
	"strings"

	"github.com/timzifer/lux/draw"
)

// DefaultFontSize is the default base font-size in dp when no inherited
// font-size is available.
const DefaultFontSize float32 = 14

// ParseFontSize parses a CSS font-size value and returns the size in dp.
// Supports named sizes (xx-small through xx-large), em units, and
// absolute units (px, dp, pt). Uses DefaultFontSize as the em base.
func ParseFontSize(v string) (float32, bool) {
	return ParseFontSizeWith(v, DefaultFontSize)
}

// ParseFontSizeWith parses a CSS font-size value using the given base
// font-size for resolving em/rem units.
func ParseFontSizeWith(v string, baseFontSize float32) (float32, bool) {
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
	if baseFontSize <= 0 {
		baseFontSize = DefaultFontSize
	}
	// Relative em/rem sizes.
	if strings.HasSuffix(v, "rem") {
		if f, err := strconv.ParseFloat(v[:len(v)-3], 32); err == nil {
			return float32(f) * DefaultFontSize, true
		}
	}
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(v[:len(v)-2], 32); err == nil {
			return float32(f) * baseFontSize, true
		}
	}
	return ParseDimension(v)
}

// ParseDimension parses a CSS length value (e.g. "12px", "1.5em", "16pt")
// and returns the numeric value in dp. Known unit suffixes (px, dp, pt,
// em, rem) are stripped; the raw number is returned.
// NOTE: em/rem units are NOT resolved here — use ResolveDimension for
// font-size-aware resolution.
func ParseDimension(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	for _, suffix := range []string{"px", "dp", "pt", "rem", "em"} {
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

// ResolveDimension parses a CSS length value and resolves em/rem units
// using the given base font-size.
func ResolveDimension(v string, baseFontSize float32) (float32, bool) {
	v = strings.TrimSpace(v)
	if baseFontSize <= 0 {
		baseFontSize = DefaultFontSize
	}
	if strings.HasSuffix(v, "rem") {
		if f, err := strconv.ParseFloat(v[:len(v)-3], 32); err == nil {
			return float32(f) * DefaultFontSize, true
		}
		return 0, false
	}
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(v[:len(v)-2], 32); err == nil {
			return float32(f) * baseFontSize, true
		}
		return 0, false
	}
	return ParseDimension(v)
}

// ResolveBoxDimensions parses CSS shorthand values with 1-4 components
// and resolves em/rem units using the given base font-size.
func ResolveBoxDimensions(v string, baseFontSize float32) ([4]float32, bool) {
	parts := strings.Fields(v)
	var vals [4]float32
	switch len(parts) {
	case 1:
		d, ok := ResolveDimension(parts[0], baseFontSize)
		if !ok {
			return vals, false
		}
		vals = [4]float32{d, d, d, d}
	case 2:
		tb, ok1 := ResolveDimension(parts[0], baseFontSize)
		lr, ok2 := ResolveDimension(parts[1], baseFontSize)
		if !ok1 || !ok2 {
			return vals, false
		}
		vals = [4]float32{tb, lr, tb, lr}
	case 3:
		t, ok1 := ResolveDimension(parts[0], baseFontSize)
		lr, ok2 := ResolveDimension(parts[1], baseFontSize)
		b, ok3 := ResolveDimension(parts[2], baseFontSize)
		if !ok1 || !ok2 || !ok3 {
			return vals, false
		}
		vals = [4]float32{t, lr, b, lr}
	case 4:
		t, ok1 := ResolveDimension(parts[0], baseFontSize)
		r, ok2 := ResolveDimension(parts[1], baseFontSize)
		b, ok3 := ResolveDimension(parts[2], baseFontSize)
		l, ok4 := ResolveDimension(parts[3], baseFontSize)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return vals, false
		}
		vals = [4]float32{t, r, b, l}
	default:
		return vals, false
	}
	return vals, true
}

// IsPercentage returns true if the value is a CSS percentage (e.g. "50%").
func IsPercentage(v string) bool {
	return strings.HasSuffix(strings.TrimSpace(v), "%")
}

// ParsePercentage parses a CSS percentage value (e.g. "10.638%") and
// returns the fractional value (e.g. 0.10638).
func ParsePercentage(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	if !strings.HasSuffix(v, "%") {
		return 0, false
	}
	f, err := strconv.ParseFloat(v[:len(v)-1], 32)
	if err != nil {
		return 0, false
	}
	return float32(f) / 100, true
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

// FontShorthand holds the parsed components of a CSS font shorthand value.
type FontShorthand struct {
	Style      string // e.g. "italic", "oblique", or ""
	Weight     string // e.g. "bold", "700", or ""
	Size       string // e.g. "10px"
	LineHeight string // e.g. "1", "1.5em", or ""
	Family     string // e.g. "Verdana, sans-serif"
}

// ParseFontShorthand parses a CSS font shorthand value.
// Format: [style] [variant] [weight] size[/line-height] family
func ParseFontShorthand(v string) (FontShorthand, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return FontShorthand{}, false
	}

	var result FontShorthand

	// Split on the first comma to separate size/line-height part from family list.
	// But first we need to find the font-size token which is the key delimiter.
	// Everything before font-size is style/variant/weight.
	// Font-size is the first token that looks like a dimension or named size.
	// Everything after font-size (and optional /line-height) is font-family.

	tokens := strings.Fields(v)
	sizeIdx := -1
	for i, t := range tokens {
		// Check if this token starts with a digit, or is a named size,
		// or starts with a '.' (like .5em).
		clean := t
		if idx := strings.Index(t, "/"); idx >= 0 {
			clean = t[:idx]
		}
		if isCSSSizeToken(clean) {
			sizeIdx = i
			break
		}
	}

	if sizeIdx < 0 {
		return FontShorthand{}, false
	}

	// Parse pre-size tokens as style/variant/weight.
	for i := 0; i < sizeIdx; i++ {
		t := strings.ToLower(tokens[i])
		switch t {
		case "italic", "oblique":
			result.Style = t
		case "small-caps", "normal":
			// variant or reset — skip
		case "bold", "bolder", "lighter":
			result.Weight = t
		default:
			// Numeric weight?
			if _, err := strconv.Atoi(t); err == nil {
				result.Weight = t
			}
		}
	}

	// Parse size[/line-height].
	sizeToken := tokens[sizeIdx]
	if idx := strings.Index(sizeToken, "/"); idx >= 0 {
		result.Size = sizeToken[:idx]
		result.LineHeight = sizeToken[idx+1:]
	} else {
		result.Size = sizeToken
		// Check if next token starts with "/" (space around slash).
		if sizeIdx+1 < len(tokens) && strings.HasPrefix(tokens[sizeIdx+1], "/") {
			lh := strings.TrimPrefix(tokens[sizeIdx+1], "/")
			if lh != "" {
				result.LineHeight = lh
				sizeIdx++
			} else if sizeIdx+2 < len(tokens) {
				result.LineHeight = tokens[sizeIdx+2]
				sizeIdx += 2
			}
		}
	}

	// Everything after is font-family.
	if sizeIdx+1 < len(tokens) {
		result.Family = strings.Join(tokens[sizeIdx+1:], " ")
		// Clean up trailing semicolons or commas.
		result.Family = strings.TrimRight(result.Family, ";")
		result.Family = strings.TrimSpace(result.Family)
	}

	return result, result.Size != ""
}

// isCSSSizeToken returns true if the token looks like a CSS font-size value.
func isCSSSizeToken(s string) bool {
	s = strings.ToLower(s)
	// Named sizes.
	switch s {
	case "xx-small", "x-small", "small", "medium", "large", "x-large", "xx-large":
		return true
	}
	// Starts with digit or dot (dimension value).
	if len(s) > 0 && (s[0] >= '0' && s[0] <= '9' || s[0] == '.') {
		return true
	}
	return false
}

// ExpandFontShorthand expands a font shorthand value into individual
// property declarations.
func ExpandFontShorthand(v string, decl *StyleDeclaration) {
	fs, ok := ParseFontShorthand(v)
	if !ok {
		return
	}
	if fs.Size != "" {
		decl.Set("font-size", fs.Size)
	}
	if fs.LineHeight != "" {
		decl.Set("line-height", fs.LineHeight)
	}
	if fs.Family != "" {
		decl.Set("font-family", fs.Family)
	}
	if fs.Style != "" {
		decl.Set("font-style", fs.Style)
	}
	if fs.Weight != "" {
		decl.Set("font-weight", fs.Weight)
	}
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
