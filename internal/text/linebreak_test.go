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
