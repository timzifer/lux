package text

import (
	"testing"
)

func TestLineBreakEmpty(t *testing.T) {
	breaks := DefaultLineBreaker.Breaks("")
	if len(breaks) != 0 {
		t.Errorf("empty input: got %d breaks, want 0", len(breaks))
	}
}

func TestLineBreakSpaceSeparated(t *testing.T) {
	text := "Hello World Test"
	breaks := DefaultLineBreaker.Breaks(text)
	if len(breaks) == 0 {
		t.Fatal("space-separated words: got 0 breaks, want > 0")
	}
	// Should have break opportunities (at least at the spaces).
	opportunities := 0
	for _, b := range breaks {
		if b.Kind == LineBreakOpportunity {
			opportunities++
		}
	}
	if opportunities < 2 {
		t.Errorf("expected at least 2 break opportunities in %q, got %d", text, opportunities)
	}
}

func TestLineBreakMandatory(t *testing.T) {
	text := "Line1\nLine2"
	breaks := DefaultLineBreaker.Breaks(text)
	hasMandatory := false
	for _, b := range breaks {
		if b.Kind == LineBreakMandatory {
			hasMandatory = true
			if b.Offset != len("Line1\n") {
				t.Errorf("mandatory break at offset %d, want %d", b.Offset, len("Line1\n"))
			}
			break
		}
	}
	if !hasMandatory {
		t.Error("expected a mandatory break at newline")
	}
}

func TestLineBreakCRLF(t *testing.T) {
	text := "A\r\nB"
	breaks := DefaultLineBreaker.Breaks(text)
	hasMandatory := false
	for _, b := range breaks {
		if b.Kind == LineBreakMandatory {
			hasMandatory = true
			if b.Offset != len("A\r\n") {
				t.Errorf("CRLF break at offset %d, want %d", b.Offset, len("A\r\n"))
			}
			break
		}
	}
	if !hasMandatory {
		t.Error("expected a mandatory break at CRLF")
	}
}

func TestLineBreakCJK(t *testing.T) {
	// CJK text: break opportunities between almost every character.
	text := "你好世界测试"
	breaks := DefaultLineBreaker.Breaks(text)
	// With 6 CJK characters, we expect at least 4-5 break opportunities.
	if len(breaks) < 4 {
		t.Errorf("CJK text: got %d breaks, want >= 4", len(breaks))
	}
}

func TestLineBreakSingleWord(t *testing.T) {
	text := "Superlongword"
	breaks := DefaultLineBreaker.Breaks(text)
	// A single Latin word without spaces has no break opportunities.
	if len(breaks) != 0 {
		t.Errorf("single word: got %d breaks, want 0", len(breaks))
	}
}

func TestLineBreakOffsetsAreValid(t *testing.T) {
	text := "The quick brown fox"
	breaks := DefaultLineBreaker.Breaks(text)
	prev := 0
	for _, b := range breaks {
		if b.Offset <= prev {
			t.Errorf("non-increasing offset: %d <= %d", b.Offset, prev)
		}
		if b.Offset > len(text) {
			t.Errorf("offset %d exceeds text length %d", b.Offset, len(text))
		}
		prev = b.Offset
	}
}

// ── Soft Hyphen & Non-Breaking Space ─────────────────────────────

func TestLineBreakSoftHyphen(t *testing.T) {
	// Soft hyphen (U+00AD) should create a break opportunity.
	text := "self\u00ADcontained"
	breaks := DefaultLineBreaker.Breaks(text)
	found := false
	for _, b := range breaks {
		// Break should be near the soft hyphen position.
		if b.Offset > 0 && b.Offset < len(text) {
			found = true
		}
	}
	if !found {
		t.Error("expected a break opportunity at soft hyphen")
	}
}

func TestLineBreakNonBreakingSpace(t *testing.T) {
	// Non-breaking space (U+00A0) should NOT allow a break.
	text := "100\u00A0km"
	breaks := DefaultLineBreaker.Breaks(text)
	for _, b := range breaks {
		// NBSP is at byte offset 3 (before "km").
		if b.Offset == 3+len("\u00A0") {
			t.Errorf("break at NBSP offset %d — should be non-breaking", b.Offset)
		}
	}
}

// ── Mixed Scripts ────────────────────────────────────────────────

func TestLineBreakMixedScripts(t *testing.T) {
	text := "Hello你好World"
	breaks := DefaultLineBreaker.Breaks(text)
	if len(breaks) == 0 {
		t.Error("expected break opportunities at Latin/CJK transitions")
	}
}

// ── Thai Text ────────────────────────────────────────────────────

func TestLineBreakThaiText(t *testing.T) {
	// Thai doesn't use spaces between words; UAX #14 SA class provides
	// some break opportunities (dictionary-based or heuristic).
	text := "สวัสดีครับ"
	breaks := DefaultLineBreaker.Breaks(text)
	// Just verify no crash and offsets are valid.
	for _, b := range breaks {
		if b.Offset > len(text) {
			t.Errorf("Thai break offset %d exceeds length %d", b.Offset, len(text))
		}
	}
}

// ── URL-like Strings ─────────────────────────────────────────────

func TestLineBreakURLLike(t *testing.T) {
	text := "https://example.com/path/to/page"
	breaks := DefaultLineBreaker.Breaks(text)
	// Should have some break opportunities (at "/" or ".").
	if len(breaks) == 0 {
		t.Error("expected break opportunities in URL-like string")
	}
}

// ── Hyphenated Words ─────────────────────────────────────────────

func TestLineBreakHyphenated(t *testing.T) {
	text := "self-contained"
	breaks := DefaultLineBreaker.Breaks(text)
	found := false
	for _, b := range breaks {
		if b.Kind == LineBreakOpportunity {
			found = true
		}
	}
	if !found {
		t.Error("expected break opportunity after hyphen in \"self-contained\"")
	}
}

// ── Em Dash ──────────────────────────────────────────────────────

func TestLineBreakEmDash(t *testing.T) {
	text := "word\u2014word"
	breaks := DefaultLineBreaker.Breaks(text)
	if len(breaks) == 0 {
		t.Error("expected break opportunity around em dash")
	}
}

// ── Consecutive Newlines ─────────────────────────────────────────

func TestLineBreakConsecutiveNewlines(t *testing.T) {
	text := "A\n\nB"
	breaks := DefaultLineBreaker.Breaks(text)
	mandatory := 0
	for _, b := range breaks {
		if b.Kind == LineBreakMandatory {
			mandatory++
		}
	}
	if mandatory < 2 {
		t.Errorf("consecutive newlines: got %d mandatory breaks, want >= 2", mandatory)
	}
}

// ── Tab Character ────────────────────────────────────────────────

func TestLineBreakTab(t *testing.T) {
	text := "word\tword"
	breaks := DefaultLineBreaker.Breaks(text)
	if len(breaks) == 0 {
		t.Error("expected break opportunity at tab character")
	}
}

// ── Digit+Unit ───────────────────────────────────────────────────

func TestLineBreakDigitUnit(t *testing.T) {
	// UAX #14: percent sign after digits — verify behavior (may or may not break).
	text := "100%"
	breaks := DefaultLineBreaker.Breaks(text)
	// Just ensure no crash and offsets are valid.
	for _, b := range breaks {
		if b.Offset > len(text) {
			t.Errorf("digit-unit break offset %d exceeds length %d", b.Offset, len(text))
		}
	}
}

// ── Table-Driven Consolidation ───────────────────────────────────

func TestLineBreakTableDriven(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		minBreaks int  // minimum expected break count
		mandatory bool // expect at least one mandatory break
	}{
		{"spaces", "Hello World Test", 2, false},
		{"newline", "A\nB", 1, true},
		{"CRLF", "A\r\nB", 1, true},
		{"CJK", "你好世界", 2, false},
		{"empty", "", 0, false},
		{"single-word", "word", 0, false},
		{"hyphenated", "a-b", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breaks := DefaultLineBreaker.Breaks(tt.text)
			if len(breaks) < tt.minBreaks {
				t.Errorf("got %d breaks, want >= %d", len(breaks), tt.minBreaks)
			}
			if tt.mandatory {
				hasMandatory := false
				for _, b := range breaks {
					if b.Kind == LineBreakMandatory {
						hasMandatory = true
						break
					}
				}
				if !hasMandatory {
					t.Error("expected at least one mandatory break")
				}
			}
		})
	}
}
