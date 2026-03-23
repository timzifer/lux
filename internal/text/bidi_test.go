package text

import (
	"testing"
)

func TestBidiParagraphEmpty(t *testing.T) {
	runs := BidiParagraph("", TextDirectionAuto)
	if len(runs) != 0 {
		t.Errorf("empty input: got %d runs, want 0", len(runs))
	}
}

func TestBidiParagraphPureLTR(t *testing.T) {
	runs := BidiParagraph("Hello World", TextDirectionLTR)
	if len(runs) != 1 {
		t.Fatalf("pure LTR: got %d runs, want 1", len(runs))
	}
	if runs[0].Direction != TextDirectionLTR {
		t.Errorf("direction = %v, want LTR", runs[0].Direction)
	}
	if runs[0].Text != "Hello World" {
		t.Errorf("text = %q, want %q", runs[0].Text, "Hello World")
	}
	if runs[0].Script != "Latn" {
		t.Errorf("script = %q, want %q", runs[0].Script, "Latn")
	}
}

func TestBidiParagraphPureRTL(t *testing.T) {
	runs := BidiParagraph("مرحبا", TextDirectionRTL)
	if len(runs) != 1 {
		t.Fatalf("pure RTL: got %d runs, want 1", len(runs))
	}
	if runs[0].Direction != TextDirectionRTL {
		t.Errorf("direction = %v, want RTL", runs[0].Direction)
	}
	if runs[0].Script != "Arab" {
		t.Errorf("script = %q, want %q", runs[0].Script, "Arab")
	}
}

func TestBidiParagraphMixed(t *testing.T) {
	// Arabic with embedded Latin text should produce multiple runs.
	runs := BidiParagraph("Hello مرحبا World", TextDirectionLTR)
	if len(runs) < 2 {
		t.Fatalf("mixed text: got %d runs, want >= 2", len(runs))
	}

	// At least one run should be RTL (the Arabic part).
	hasRTL := false
	hasLTR := false
	for _, r := range runs {
		if r.Direction == TextDirectionRTL {
			hasRTL = true
		}
		if r.Direction == TextDirectionLTR {
			hasLTR = true
		}
	}
	if !hasRTL {
		t.Error("mixed text: expected at least one RTL run")
	}
	if !hasLTR {
		t.Error("mixed text: expected at least one LTR run")
	}
}

func TestBidiParagraphAutoLTR(t *testing.T) {
	runs := BidiParagraph("Hello World", TextDirectionAuto)
	if len(runs) < 1 {
		t.Fatal("auto LTR: got 0 runs")
	}
	if runs[0].Direction != TextDirectionLTR {
		t.Errorf("auto with Latin-first: first run direction = %v, want LTR", runs[0].Direction)
	}
}

func TestBidiParagraphAutoRTL(t *testing.T) {
	runs := BidiParagraph("مرحبا Hello", TextDirectionAuto)
	if len(runs) < 1 {
		t.Fatal("auto RTL: got 0 runs")
	}
	// The first run should be RTL since the text starts with Arabic.
	if runs[0].Direction != TextDirectionRTL {
		t.Errorf("auto with Arabic-first: first run direction = %v, want RTL", runs[0].Direction)
	}
}

func TestBidiParagraphNumbersInRTL(t *testing.T) {
	// Numbers embedded in RTL text retain LTR ordering.
	runs := BidiParagraph("العدد 123 هنا", TextDirectionRTL)
	if len(runs) < 2 {
		t.Fatalf("numbers in RTL: got %d runs, want >= 2", len(runs))
	}
	// Verify numbers are in an LTR run.
	foundNumLTR := false
	for _, r := range runs {
		if r.Direction == TextDirectionLTR {
			for _, ch := range r.Text {
				if ch >= '0' && ch <= '9' {
					foundNumLTR = true
					break
				}
			}
		}
	}
	if !foundNumLTR {
		t.Error("numbers in RTL text: expected digits in an LTR run")
	}
}

func TestDetectScriptCommon(t *testing.T) {
	tests := []struct {
		text   string
		script string
	}{
		{"Hello", "Latn"},
		{"مرحبا", "Arab"},
		{"שלום", "Hebr"},
		{"こんにちは", "Hira"},
		{"你好", "Hani"},
		{"안녕", "Hang"},
		{"Привет", "Cyrl"},
		{"สวัสดี", "Thai"},
		{"123", "Latn"}, // fallback for digits-only
	}
	for _, tt := range tests {
		got := detectScript(tt.text)
		if got != tt.script {
			t.Errorf("detectScript(%q) = %q, want %q", tt.text, got, tt.script)
		}
	}
}
