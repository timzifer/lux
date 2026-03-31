package css

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/timzifer/lux/draw"
)

// ParseColor parses a CSS color value.
// Supports: #RGB, #RRGGBB, #RRGGBBAA, rgb(r,g,b), rgba(r,g,b,a),
// and all 148 CSS Named Colors (W3C CSS Color Level 4).
func ParseColor(s string) (draw.Color, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return draw.Color{}, false
	}

	// Named color lookup.
	if c, ok := namedColors[s]; ok {
		return c, true
	}

	// Hex notation.
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s[1:])
	}

	// rgb()/rgba() functional notation.
	if strings.HasPrefix(s, "rgba(") && strings.HasSuffix(s, ")") {
		return parseRGBFunc(s[5 : len(s)-1])
	}
	if strings.HasPrefix(s, "rgb(") && strings.HasSuffix(s, ")") {
		return parseRGBFunc(s[4 : len(s)-1])
	}

	return draw.Color{}, false
}

func parseHexColor(hex string) (draw.Color, bool) {
	switch len(hex) {
	case 3: // #RGB → RRGGBB
		r, _ := strconv.ParseUint(string([]byte{hex[0], hex[0]}), 16, 8)
		g, _ := strconv.ParseUint(string([]byte{hex[1], hex[1]}), 16, 8)
		b, _ := strconv.ParseUint(string([]byte{hex[2], hex[2]}), 16, 8)
		return draw.RGBA(uint8(r), uint8(g), uint8(b), 255), true
	case 6: // #RRGGBB
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return draw.RGBA(uint8(r), uint8(g), uint8(b), 255), true
	case 8: // #RRGGBBAA
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		a, _ := strconv.ParseUint(hex[6:8], 16, 8)
		return draw.RGBA(uint8(r), uint8(g), uint8(b), uint8(a)), true
	}
	return draw.Color{}, false
}

func parseRGBFunc(args string) (draw.Color, bool) {
	// Support both comma-separated and space-separated syntax.
	// Also support / for alpha: rgb(255 0 0 / 0.5)
	args = strings.ReplaceAll(args, "/", ",")
	args = strings.ReplaceAll(args, "  ", " ")
	var parts []string
	if strings.Contains(args, ",") {
		parts = strings.Split(args, ",")
	} else {
		parts = strings.Fields(args)
	}

	if len(parts) < 3 || len(parts) > 4 {
		return draw.Color{}, false
	}

	r, ok := parseColorComponent(strings.TrimSpace(parts[0]), 255)
	if !ok {
		return draw.Color{}, false
	}
	g, ok := parseColorComponent(strings.TrimSpace(parts[1]), 255)
	if !ok {
		return draw.Color{}, false
	}
	b, ok := parseColorComponent(strings.TrimSpace(parts[2]), 255)
	if !ok {
		return draw.Color{}, false
	}
	a := float32(1.0)
	if len(parts) == 4 {
		av, ok := parseAlphaComponent(strings.TrimSpace(parts[3]))
		if !ok {
			return draw.Color{}, false
		}
		a = av
	}
	return draw.Color{R: r, G: g, B: b, A: a}, true
}

func parseColorComponent(s string, max float32) (float32, bool) {
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 32)
		if err != nil {
			return 0, false
		}
		return float32(v) / 100, true
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, false
	}
	return float32(v) / max, true
}

func parseAlphaComponent(s string) (float32, bool) {
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 32)
		if err != nil {
			return 0, false
		}
		return float32(v) / 100, true
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, false
	}
	return float32(v), true
}

// FormatColor converts a draw.Color to a CSS color string.
// Uses rgba() when alpha < 1, otherwise #RRGGBB.
func FormatColor(c draw.Color) string {
	r := uint8(c.R * 255)
	g := uint8(c.G * 255)
	b := uint8(c.B * 255)
	a := uint8(c.A * 255)
	if a == 255 {
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r, g, b, c.A)
}

// namedColors contains all 148 CSS Named Colors (CSS Color Level 4).
var namedColors = map[string]draw.Color{
	"aliceblue":            draw.RGBA(240, 248, 255, 255),
	"antiquewhite":         draw.RGBA(250, 235, 215, 255),
	"aqua":                 draw.RGBA(0, 255, 255, 255),
	"aquamarine":           draw.RGBA(127, 255, 212, 255),
	"azure":                draw.RGBA(240, 255, 255, 255),
	"beige":                draw.RGBA(245, 245, 220, 255),
	"bisque":               draw.RGBA(255, 228, 196, 255),
	"black":                draw.RGBA(0, 0, 0, 255),
	"blanchedalmond":       draw.RGBA(255, 235, 205, 255),
	"blue":                 draw.RGBA(0, 0, 255, 255),
	"blueviolet":           draw.RGBA(138, 43, 226, 255),
	"brown":                draw.RGBA(165, 42, 42, 255),
	"burlywood":            draw.RGBA(222, 184, 135, 255),
	"cadetblue":            draw.RGBA(95, 158, 160, 255),
	"chartreuse":           draw.RGBA(127, 255, 0, 255),
	"chocolate":            draw.RGBA(210, 105, 30, 255),
	"coral":                draw.RGBA(255, 127, 80, 255),
	"cornflowerblue":       draw.RGBA(100, 149, 237, 255),
	"cornsilk":             draw.RGBA(255, 248, 220, 255),
	"crimson":              draw.RGBA(220, 20, 60, 255),
	"cyan":                 draw.RGBA(0, 255, 255, 255),
	"darkblue":             draw.RGBA(0, 0, 139, 255),
	"darkcyan":             draw.RGBA(0, 139, 139, 255),
	"darkgoldenrod":        draw.RGBA(184, 134, 11, 255),
	"darkgray":             draw.RGBA(169, 169, 169, 255),
	"darkgreen":            draw.RGBA(0, 100, 0, 255),
	"darkgrey":             draw.RGBA(169, 169, 169, 255),
	"darkkhaki":            draw.RGBA(189, 183, 107, 255),
	"darkmagenta":          draw.RGBA(139, 0, 139, 255),
	"darkolivegreen":       draw.RGBA(85, 107, 47, 255),
	"darkorange":           draw.RGBA(255, 140, 0, 255),
	"darkorchid":           draw.RGBA(153, 50, 204, 255),
	"darkred":              draw.RGBA(139, 0, 0, 255),
	"darksalmon":           draw.RGBA(233, 150, 122, 255),
	"darkseagreen":         draw.RGBA(143, 188, 143, 255),
	"darkslateblue":        draw.RGBA(72, 61, 139, 255),
	"darkslategray":        draw.RGBA(47, 79, 79, 255),
	"darkslategrey":        draw.RGBA(47, 79, 79, 255),
	"darkturquoise":        draw.RGBA(0, 206, 209, 255),
	"darkviolet":           draw.RGBA(148, 0, 211, 255),
	"deeppink":             draw.RGBA(255, 20, 147, 255),
	"deepskyblue":          draw.RGBA(0, 191, 255, 255),
	"dimgray":              draw.RGBA(105, 105, 105, 255),
	"dimgrey":              draw.RGBA(105, 105, 105, 255),
	"dodgerblue":           draw.RGBA(30, 144, 255, 255),
	"firebrick":            draw.RGBA(178, 34, 34, 255),
	"floralwhite":          draw.RGBA(255, 250, 240, 255),
	"forestgreen":          draw.RGBA(34, 139, 34, 255),
	"fuchsia":              draw.RGBA(255, 0, 255, 255),
	"gainsboro":            draw.RGBA(220, 220, 220, 255),
	"ghostwhite":           draw.RGBA(248, 248, 255, 255),
	"gold":                 draw.RGBA(255, 215, 0, 255),
	"goldenrod":            draw.RGBA(218, 165, 32, 255),
	"gray":                 draw.RGBA(128, 128, 128, 255),
	"green":                draw.RGBA(0, 128, 0, 255),
	"greenyellow":          draw.RGBA(173, 255, 47, 255),
	"grey":                 draw.RGBA(128, 128, 128, 255),
	"honeydew":             draw.RGBA(240, 255, 240, 255),
	"hotpink":              draw.RGBA(255, 105, 180, 255),
	"indianred":            draw.RGBA(205, 92, 92, 255),
	"indigo":               draw.RGBA(75, 0, 130, 255),
	"ivory":                draw.RGBA(255, 255, 240, 255),
	"khaki":                draw.RGBA(240, 230, 140, 255),
	"lavender":             draw.RGBA(230, 230, 250, 255),
	"lavenderblush":        draw.RGBA(255, 240, 245, 255),
	"lawngreen":            draw.RGBA(124, 252, 0, 255),
	"lemonchiffon":         draw.RGBA(255, 250, 205, 255),
	"lightblue":            draw.RGBA(173, 216, 230, 255),
	"lightcoral":           draw.RGBA(240, 128, 128, 255),
	"lightcyan":            draw.RGBA(224, 255, 255, 255),
	"lightgoldenrodyellow": draw.RGBA(250, 250, 210, 255),
	"lightgray":            draw.RGBA(211, 211, 211, 255),
	"lightgreen":           draw.RGBA(144, 238, 144, 255),
	"lightgrey":            draw.RGBA(211, 211, 211, 255),
	"lightpink":            draw.RGBA(255, 182, 193, 255),
	"lightsalmon":          draw.RGBA(255, 160, 122, 255),
	"lightseagreen":        draw.RGBA(32, 178, 170, 255),
	"lightskyblue":         draw.RGBA(135, 206, 250, 255),
	"lightslategray":       draw.RGBA(119, 136, 153, 255),
	"lightslategrey":       draw.RGBA(119, 136, 153, 255),
	"lightsteelblue":       draw.RGBA(176, 196, 222, 255),
	"lightyellow":          draw.RGBA(255, 255, 224, 255),
	"lime":                 draw.RGBA(0, 255, 0, 255),
	"limegreen":            draw.RGBA(50, 205, 50, 255),
	"linen":                draw.RGBA(250, 240, 230, 255),
	"magenta":              draw.RGBA(255, 0, 255, 255),
	"maroon":               draw.RGBA(128, 0, 0, 255),
	"mediumaquamarine":     draw.RGBA(102, 205, 170, 255),
	"mediumblue":           draw.RGBA(0, 0, 205, 255),
	"mediumorchid":         draw.RGBA(186, 85, 211, 255),
	"mediumpurple":         draw.RGBA(147, 112, 219, 255),
	"mediumseagreen":       draw.RGBA(60, 179, 113, 255),
	"mediumslateblue":      draw.RGBA(123, 104, 238, 255),
	"mediumspringgreen":    draw.RGBA(0, 250, 154, 255),
	"mediumturquoise":      draw.RGBA(72, 209, 204, 255),
	"mediumvioletred":      draw.RGBA(199, 21, 133, 255),
	"midnightblue":         draw.RGBA(25, 25, 112, 255),
	"mintcream":            draw.RGBA(245, 255, 250, 255),
	"mistyrose":            draw.RGBA(255, 228, 225, 255),
	"moccasin":             draw.RGBA(255, 228, 181, 255),
	"navajowhite":          draw.RGBA(255, 222, 173, 255),
	"navy":                 draw.RGBA(0, 0, 128, 255),
	"oldlace":              draw.RGBA(253, 245, 230, 255),
	"olive":                draw.RGBA(128, 128, 0, 255),
	"olivedrab":            draw.RGBA(107, 142, 35, 255),
	"orange":               draw.RGBA(255, 165, 0, 255),
	"orangered":            draw.RGBA(255, 69, 0, 255),
	"orchid":               draw.RGBA(218, 112, 214, 255),
	"palegoldenrod":        draw.RGBA(238, 232, 170, 255),
	"palegreen":            draw.RGBA(152, 251, 152, 255),
	"paleturquoise":        draw.RGBA(175, 238, 238, 255),
	"palevioletred":        draw.RGBA(219, 112, 147, 255),
	"papayawhip":           draw.RGBA(255, 239, 213, 255),
	"peachpuff":            draw.RGBA(255, 218, 185, 255),
	"peru":                 draw.RGBA(205, 133, 63, 255),
	"pink":                 draw.RGBA(255, 192, 203, 255),
	"plum":                 draw.RGBA(221, 160, 221, 255),
	"powderblue":           draw.RGBA(176, 224, 230, 255),
	"purple":               draw.RGBA(128, 0, 128, 255),
	"rebeccapurple":        draw.RGBA(102, 51, 153, 255),
	"red":                  draw.RGBA(255, 0, 0, 255),
	"rosybrown":            draw.RGBA(188, 143, 143, 255),
	"royalblue":            draw.RGBA(65, 105, 225, 255),
	"saddlebrown":          draw.RGBA(139, 69, 19, 255),
	"salmon":               draw.RGBA(250, 128, 114, 255),
	"sandybrown":           draw.RGBA(244, 164, 96, 255),
	"seagreen":             draw.RGBA(46, 139, 87, 255),
	"seashell":             draw.RGBA(255, 245, 238, 255),
	"sienna":               draw.RGBA(160, 82, 45, 255),
	"silver":               draw.RGBA(192, 192, 192, 255),
	"skyblue":              draw.RGBA(135, 206, 235, 255),
	"slateblue":            draw.RGBA(106, 90, 205, 255),
	"slategray":            draw.RGBA(112, 128, 144, 255),
	"slategrey":            draw.RGBA(112, 128, 144, 255),
	"snow":                 draw.RGBA(255, 250, 250, 255),
	"springgreen":          draw.RGBA(0, 255, 127, 255),
	"steelblue":            draw.RGBA(70, 130, 180, 255),
	"tan":                  draw.RGBA(210, 180, 140, 255),
	"teal":                 draw.RGBA(0, 128, 128, 255),
	"thistle":              draw.RGBA(216, 191, 216, 255),
	"tomato":               draw.RGBA(255, 99, 71, 255),
	"turquoise":            draw.RGBA(64, 224, 208, 255),
	"violet":               draw.RGBA(238, 130, 238, 255),
	"wheat":                draw.RGBA(245, 222, 179, 255),
	"white":                draw.RGBA(255, 255, 255, 255),
	"whitesmoke":           draw.RGBA(245, 245, 245, 255),
	"yellow":               draw.RGBA(255, 255, 0, 255),
	"yellowgreen":          draw.RGBA(154, 205, 50, 255),
	"transparent":          draw.RGBA(0, 0, 0, 0),
}
