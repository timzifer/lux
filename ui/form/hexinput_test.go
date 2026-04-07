package form

import "testing"

func TestIsValidHexChar(t *testing.T) {
	valid := "0123456789abcdefABCDEF"
	for _, ch := range valid {
		if !IsValidHexChar(ch) {
			t.Errorf("IsValidHexChar(%c) = false, want true", ch)
		}
	}

	invalid := "ghGH!@# xyz"
	for _, ch := range invalid {
		if IsValidHexChar(ch) {
			t.Errorf("IsValidHexChar(%c) = true, want false", ch)
		}
	}
}

func TestHexInput_FormatHex(t *testing.T) {
	tests := []struct {
		name   string
		h      HexInput
		want   string
	}{
		{"4 digit upper", HexInput{Value: 0x00FF, Digits: 4, Upper: true}, "00FF"},
		{"4 digit lower", HexInput{Value: 0x00FF, Digits: 4, Upper: false}, "00ff"},
		{"8 digit", HexInput{Value: 0xDEADBEEF, Digits: 8, Upper: true}, "DEADBEEF"},
		{"2 digit", HexInput{Value: 0x0A, Digits: 2, Upper: true}, "0A"},
		{"zero", HexInput{Value: 0, Digits: 4, Upper: true}, "0000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.h.FormatHex()
			if got != tt.want {
				t.Errorf("FormatHex() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterHexChars(t *testing.T) {
	tests := []struct {
		in    string
		upper bool
		max   int
		want  string
	}{
		{"00FF", true, 4, "00FF"},
		{"00ffgg", true, 4, "00FF"},   // filtered + uppercased
		{"abcd", false, 4, "abcd"},
		{"ABCD", false, 4, "abcd"},    // lowercased
		{"12345678", true, 4, "5678"}, // shift-register: keep last maxDigits
	}
	for _, tt := range tests {
		got := filterHexChars(tt.in, tt.upper, tt.max)
		if got != tt.want {
			t.Errorf("filterHexChars(%q, upper=%v, %d) = %q, want %q", tt.in, tt.upper, tt.max, got, tt.want)
		}
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
		ok    bool
	}{
		{"FF", 0xFF, true},
		{"0xFF", 0xFF, true},
		{"0XFF", 0xFF, true},
		{"DEADBEEF", 0xDEADBEEF, true},
		{"0a", 0x0A, true},
		{"", 0, false},
		{"GG", 0, false},
		{"12xy", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseHex(tt.input)
			if ok != tt.ok {
				t.Errorf("ParseHex(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ParseHex(%q) = 0x%X, want 0x%X", tt.input, got, tt.want)
			}
		})
	}
}
