package css

import (
	"math"
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestParseStyleAttribute(t *testing.T) {
	decl := ParseStyleAttribute("color: red; font-weight: bold; font-size: 16px")
	if decl.Get("color") != "red" {
		t.Errorf("color = %q, want %q", decl.Get("color"), "red")
	}
	if decl.Get("font-weight") != "bold" {
		t.Errorf("font-weight = %q, want %q", decl.Get("font-weight"), "bold")
	}
	if decl.Get("font-size") != "16px" {
		t.Errorf("font-size = %q, want %q", decl.Get("font-size"), "16px")
	}
}

func TestParseStyleSheet(t *testing.T) {
	css := `.highlight { color: red; font-weight: bold }
p { font-size: 14px }`
	sheet, err := ParseStyleSheet(css)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet.Rules) < 2 {
		t.Fatalf("expected at least 2 rules, got %d", len(sheet.Rules))
	}

	// Check that we got rules for .highlight and p.
	foundHighlight := false
	foundP := false
	for _, r := range sheet.Rules {
		if r.Selector == ".highlight" {
			foundHighlight = true
			if r.Decl.Get("color") != "red" {
				t.Errorf(".highlight color = %q, want %q", r.Decl.Get("color"), "red")
			}
		}
		if r.Selector == "p" {
			foundP = true
		}
	}
	if !foundHighlight {
		t.Error("missing .highlight rule")
	}
	if !foundP {
		t.Error("missing p rule")
	}
}

func TestParseColor_Named(t *testing.T) {
	tests := []struct {
		input string
		r, g, b uint8
	}{
		{"red", 255, 0, 0},
		{"blue", 0, 0, 255},
		{"green", 0, 128, 0},
		{"white", 255, 255, 255},
		{"black", 0, 0, 0},
		{"cornflowerblue", 100, 149, 237},
		{"transparent", 0, 0, 0}, // alpha = 0
	}
	for _, tt := range tests {
		c, ok := ParseColor(tt.input)
		if !ok {
			t.Errorf("ParseColor(%q) failed", tt.input)
			continue
		}
		got := [3]uint8{uint8(c.R * 255), uint8(c.G * 255), uint8(c.B * 255)}
		want := [3]uint8{tt.r, tt.g, tt.b}
		if got != want {
			t.Errorf("ParseColor(%q) = rgb(%d,%d,%d), want rgb(%d,%d,%d)",
				tt.input, got[0], got[1], got[2], want[0], want[1], want[2])
		}
	}
}

func TestParseColor_Hex(t *testing.T) {
	tests := []struct {
		input string
		want  draw.Color
	}{
		{"#f00", draw.RGBA(255, 0, 0, 255)},
		{"#ff0000", draw.RGBA(255, 0, 0, 255)},
		{"#ff000080", draw.RGBA(255, 0, 0, 128)},
	}
	for _, tt := range tests {
		c, ok := ParseColor(tt.input)
		if !ok {
			t.Errorf("ParseColor(%q) failed", tt.input)
			continue
		}
		if !colorClose(c, tt.want) {
			t.Errorf("ParseColor(%q) = %v, want %v", tt.input, c, tt.want)
		}
	}
}

func TestParseColor_RGB(t *testing.T) {
	c, ok := ParseColor("rgb(255, 128, 0)")
	if !ok {
		t.Fatal("ParseColor(rgb) failed")
	}
	if !colorClose(c, draw.RGBA(255, 128, 0, 255)) {
		t.Errorf("got %v, want rgb(255,128,0)", c)
	}

	c, ok = ParseColor("rgba(255, 0, 0, 0.5)")
	if !ok {
		t.Fatal("ParseColor(rgba) failed")
	}
	if math.Abs(float64(c.A-0.5)) > 0.01 {
		t.Errorf("alpha = %f, want 0.5", c.A)
	}
}

func TestFormatColor(t *testing.T) {
	got := FormatColor(draw.RGBA(255, 0, 0, 255))
	if got != "#ff0000" {
		t.Errorf("FormatColor(red) = %q, want %q", got, "#ff0000")
	}
}

func TestStyleDeclaration_Merge(t *testing.T) {
	a := NewDecl()
	a.Set("color", "red")
	a.Set("font-size", "14px")

	b := NewDecl()
	b.Set("color", "blue") // Override
	b.Set("font-weight", "bold")

	a.Merge(b)
	if a.Get("color") != "blue" {
		t.Errorf("color = %q, want %q", a.Get("color"), "blue")
	}
	if a.Get("font-size") != "14px" {
		t.Errorf("font-size should be preserved")
	}
	if a.Get("font-weight") != "bold" {
		t.Errorf("font-weight should be merged")
	}
}

func colorClose(a, b draw.Color) bool {
	return math.Abs(float64(a.R-b.R)) < 0.02 &&
		math.Abs(float64(a.G-b.G)) < 0.02 &&
		math.Abs(float64(a.B-b.B)) < 0.02 &&
		math.Abs(float64(a.A-b.A)) < 0.02
}
