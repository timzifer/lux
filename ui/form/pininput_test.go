package form

import (
	"regexp"
	"testing"
)

func TestPinInput_IsValidChar(t *testing.T) {
	p := PinInput{AllowedChars: DefaultPinChars}

	// Digits allowed.
	for _, ch := range "0123456789" {
		if !p.IsValidChar(ch) {
			t.Errorf("digit %c should be valid", ch)
		}
	}

	// Letters not allowed.
	for _, ch := range "abcABC" {
		if p.IsValidChar(ch) {
			t.Errorf("letter %c should not be valid", ch)
		}
	}
}

func TestPinInput_CustomAllowedChars(t *testing.T) {
	// Allow alphanumeric.
	p := PinInput{AllowedChars: regexp.MustCompile(`^[0-9a-zA-Z]$`)}

	if !p.IsValidChar('a') {
		t.Error("'a' should be valid with alphanumeric regex")
	}
	if !p.IsValidChar('5') {
		t.Error("'5' should be valid with alphanumeric regex")
	}
	if p.IsValidChar('!') {
		t.Error("'!' should not be valid with alphanumeric regex")
	}
}

func TestFilterPinChars(t *testing.T) {
	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"1234", 4, "1234"},
		{"12345", 4, "1234"},     // truncate
		{"12ab34", 4, "1234"},    // filter non-digits
		{"", 4, ""},
		{"abc", 4, ""},           // all filtered
	}
	for _, tt := range tests {
		got := filterPinChars(tt.in, DefaultPinChars, tt.maxLen)
		if got != tt.want {
			t.Errorf("filterPinChars(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
		}
	}
}
