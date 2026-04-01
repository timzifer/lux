package css

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestParseFontSize(t *testing.T) {
	tests := []struct {
		input    string
		expected float32
		ok       bool
	}{
		{"xx-small", 8, true},
		{"small", 12, true},
		{"medium", 14, true},
		{"large", 16, true},
		{"xx-large", 24, true},
		{"2em", 28, true},
		{"16px", 16, true},
		{"12pt", 12, true},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		v, ok := ParseFontSize(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseFontSize(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && v != tt.expected {
			t.Errorf("ParseFontSize(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}

func TestParseDimension(t *testing.T) {
	tests := []struct {
		input    string
		expected float32
		ok       bool
	}{
		{"10px", 10, true},
		{"20dp", 20, true},
		{"12pt", 12, true},
		{"1.5em", 1.5, true},
		{"2rem", 2, true},
		{"100", 100, true},
		{"abc", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		v, ok := ParseDimension(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseDimension(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && v != tt.expected {
			t.Errorf("ParseDimension(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}

func TestParseLineHeight(t *testing.T) {
	tests := []struct {
		input    string
		expected float32
		ok       bool
	}{
		{"normal", 0, false},
		{"1.5", 1.5, true},
		{"20px", 20, true},
		{"2", 2, true},
	}

	for _, tt := range tests {
		v, ok := ParseLineHeight(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseLineHeight(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && v != tt.expected {
			t.Errorf("ParseLineHeight(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}

func TestParseTextAlign(t *testing.T) {
	tests := []struct {
		input    string
		expected draw.TextAlign
		ok       bool
	}{
		{"left", draw.TextAlignLeft, true},
		{"center", draw.TextAlignCenter, true},
		{"right", draw.TextAlignRight, true},
		{"justify", draw.TextAlignJustify, true},
		{"start", draw.TextAlignLeft, true},
		{"end", draw.TextAlignRight, true},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		v, ok := ParseTextAlign(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseTextAlign(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && v != tt.expected {
			t.Errorf("ParseTextAlign(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}

func TestParseFontWeight(t *testing.T) {
	tests := []struct {
		input  string
		weight draw.FontWeight
		bold   bool
		ok     bool
	}{
		{"bold", draw.FontWeightBold, true, true},
		{"normal", draw.FontWeightRegular, false, true},
		{"700", draw.FontWeight(700), true, true},
		{"400", draw.FontWeight(400), false, true},
		{"900", draw.FontWeight(900), true, true},
		{"invalid", 0, false, false},
	}

	for _, tt := range tests {
		w, bold, ok := ParseFontWeight(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseFontWeight(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok {
			if w != tt.weight {
				t.Errorf("ParseFontWeight(%q): weight = %v, want %v", tt.input, w, tt.weight)
			}
			if bold != tt.bold {
				t.Errorf("ParseFontWeight(%q): bold = %v, want %v", tt.input, bold, tt.bold)
			}
		}
	}
}

func TestParseBoxDimensions(t *testing.T) {
	tests := []struct {
		input    string
		expected [4]float32
		ok       bool
	}{
		{"10px", [4]float32{10, 10, 10, 10}, true},
		{"5px 10px", [4]float32{5, 10, 5, 10}, true},
		{"1px 2px 3px", [4]float32{1, 2, 3, 2}, true},
		{"1px 2px 3px 4px", [4]float32{1, 2, 3, 4}, true},
		{"invalid", [4]float32{}, false},
		{"", [4]float32{}, false},
	}

	for _, tt := range tests {
		v, ok := ParseBoxDimensions(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseBoxDimensions(%q): ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && v != tt.expected {
			t.Errorf("ParseBoxDimensions(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}
