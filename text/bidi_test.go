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

// ── Hebrew ───────────────────────────────────────────────────────

func TestBidiParagraphHebrew(t *testing.T) {
	runs := BidiParagraph("שלום עולם", TextDirectionRTL)
	if len(runs) != 1 {
		t.Fatalf("pure Hebrew: got %d runs, want 1", len(runs))
	}
	if runs[0].Direction != TextDirectionRTL {
		t.Errorf("direction = %v, want RTL", runs[0].Direction)
	}
	if runs[0].Script != "Hebr" {
		t.Errorf("script = %q, want %q", runs[0].Script, "Hebr")
	}
}

func TestBidiParagraphHebrewMixed(t *testing.T) {
	runs := BidiParagraph("שלום Hello עולם", TextDirectionRTL)
	if len(runs) < 2 {
		t.Fatalf("Hebrew-mixed: got %d runs, want >= 2", len(runs))
	}
	hasLTR := false
	hasRTL := false
	for _, r := range runs {
		if r.Direction == TextDirectionLTR {
			hasLTR = true
		}
		if r.Direction == TextDirectionRTL {
			hasRTL = true
		}
	}
	if !hasLTR {
		t.Error("expected at least one LTR run for embedded English")
	}
	if !hasRTL {
		t.Error("expected at least one RTL run for Hebrew")
	}
}

// ── Nested Embedding ─────────────────────────────────────────────

func TestBidiParagraphNestedEmbedding(t *testing.T) {
	// LRE (U+202A) + text + PDF (U+202C) create explicit embedding.
	text := "مرحبا \u202AHello World\u202C عالم"
	runs := BidiParagraph(text, TextDirectionRTL)
	if len(runs) < 2 {
		t.Fatalf("nested embedding: got %d runs, want >= 2", len(runs))
	}
}

// ── Brackets & Punctuation ───────────────────────────────────────

func TestBidiParagraphBrackets(t *testing.T) {
	// Parentheses in RTL context should be mirrored.
	text := "שלום (עולם)"
	runs := BidiParagraph(text, TextDirectionRTL)
	if len(runs) == 0 {
		t.Fatal("brackets in RTL: got 0 runs")
	}
	// Verify the text is fully covered.
	total := 0
	for _, r := range runs {
		total += len(r.Text)
	}
	if total != len(text) {
		t.Errorf("total run text = %d bytes, want %d", total, len(text))
	}
}

func TestBidiParagraphPunctuation(t *testing.T) {
	text := "Hello, مرحبا!"
	runs := BidiParagraph(text, TextDirectionLTR)
	if len(runs) < 2 {
		t.Fatalf("punctuation mixed: got %d runs, want >= 2", len(runs))
	}
}

// ── URL in RTL ───────────────────────────────────────────────────

func TestBidiParagraphURLInRTL(t *testing.T) {
	text := "ראה https://example.com כאן"
	runs := BidiParagraph(text, TextDirectionRTL)
	// URL should be in an LTR run.
	foundLTR := false
	for _, r := range runs {
		if r.Direction == TextDirectionLTR {
			foundLTR = true
		}
	}
	if !foundLTR {
		t.Error("URL in RTL: expected URL in an LTR run")
	}
}

// ── Edge Cases ───────────────────────────────────────────────────

func TestBidiParagraphOnlySpaces(t *testing.T) {
	runs := BidiParagraph("   ", TextDirectionLTR)
	if len(runs) == 0 {
		t.Fatal("whitespace-only: got 0 runs")
	}
	// Whitespace should follow the base direction.
	if runs[0].Direction != TextDirectionLTR {
		t.Errorf("whitespace direction = %v, want LTR (base direction)", runs[0].Direction)
	}
}

func TestBidiParagraphOnlyNeutrals(t *testing.T) {
	runs := BidiParagraph("12345", TextDirectionRTL)
	if len(runs) == 0 {
		t.Fatal("digits-only: got 0 runs")
	}
}

// ── Script Detection Extended ────────────────────────────────────

func TestDetectScriptHebrew(t *testing.T) {
	got := detectScript("שלום")
	if got != "Hebr" {
		t.Errorf("detectScript(Hebrew) = %q, want %q", got, "Hebr")
	}
}

func TestDetectScriptDevanagari(t *testing.T) {
	got := detectScript("नमस्ते")
	if got != "Deva" {
		t.Errorf("detectScript(Devanagari) = %q, want %q", got, "Deva")
	}
}

func TestDetectScriptMixed(t *testing.T) {
	// First strong character wins.
	got := detectScript("123Hello")
	if got != "Latn" {
		t.Errorf("detectScript(digits+Latin) = %q, want %q", got, "Latn")
	}
	got = detectScript("  مرحبا Hello")
	if got != "Arab" {
		t.Errorf("detectScript(spaces+Arabic) = %q, want %q", got, "Arab")
	}
}

func TestRuneScriptComprehensive(t *testing.T) {
	tests := []struct {
		r      rune
		script string
	}{
		{'A', "Latn"},
		{'z', "Latn"},
		{'\u0627', "Arab"}, // Arabic Alef
		{'\u05D0', "Hebr"}, // Hebrew Alef
		{'\u0915', "Deva"}, // Devanagari Ka
		{'\u4E00', "Hani"}, // CJK Unified
		{'\uAC00', "Hang"}, // Hangul syllable
		{'\u3042', "Hira"}, // Hiragana A
		{'\u30A2', "Kana"}, // Katakana A
		{'\u0E01', "Thai"}, // Thai Ko Kai
		{'\u0995', "Beng"}, // Bengali Ka
		{'\u0B95', "Taml"}, // Tamil Ka
		{'\u10D0', "Geor"}, // Georgian An
		{'\u0531', "Armn"}, // Armenian Ayb
		{'\u0391', "Grek"}, // Greek Alpha
		{'\u0410', "Cyrl"}, // Cyrillic A
		{' ', ""},           // space — no script
		{'1', ""},           // digit — no script
	}
	for _, tt := range tests {
		got := runeScript(tt.r)
		if got != tt.script {
			t.Errorf("runeScript(%q U+%04X) = %q, want %q", tt.r, tt.r, got, tt.script)
		}
	}
}

// ── Table-Driven Consolidation ───────────────────────────────────

func TestBidiParagraphTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		baseDir TextDirection
		minRuns int
		wantDir TextDirection // expected direction of first run
	}{
		{"pure-LTR", "Hello", TextDirectionLTR, 1, TextDirectionLTR},
		{"pure-RTL-Arabic", "مرحبا", TextDirectionRTL, 1, TextDirectionRTL},
		{"pure-RTL-Hebrew", "שלום", TextDirectionRTL, 1, TextDirectionRTL},
		{"mixed-LTR-base", "Hello مرحبا", TextDirectionLTR, 2, TextDirectionLTR},
		{"mixed-RTL-base", "مرحبا Hello", TextDirectionRTL, 2, TextDirectionRTL},
		{"auto-LTR", "Hello World", TextDirectionAuto, 1, TextDirectionLTR},
		{"auto-RTL", "مرحبا عالم", TextDirectionAuto, 1, TextDirectionRTL},
		{"digits-in-RTL", "العدد 42", TextDirectionRTL, 1, TextDirectionRTL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs := BidiParagraph(tt.text, tt.baseDir)
			if len(runs) < tt.minRuns {
				t.Errorf("got %d runs, want >= %d", len(runs), tt.minRuns)
			}
			if len(runs) > 0 && runs[0].Direction != tt.wantDir {
				t.Errorf("first run direction = %v, want %v", runs[0].Direction, tt.wantDir)
			}
		})
	}
}
