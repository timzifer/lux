package text

import (
	"testing"
)

func TestLinesSingleLine(t *testing.T) {
	spans := Lines("abc")
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Start != 0 || spans[0].End != 3 {
		t.Errorf("expected {0,3}, got {%d,%d}", spans[0].Start, spans[0].End)
	}
}

func TestLinesMultiLine(t *testing.T) {
	spans := Lines("ab\ncd\nef")
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(spans))
	}
	expected := []LineSpan{{0, 2}, {3, 5}, {6, 8}}
	for i, want := range expected {
		if spans[i] != want {
			t.Errorf("span[%d]: expected %v, got %v", i, want, spans[i])
		}
	}
}

func TestLinesTrailingNewline(t *testing.T) {
	spans := Lines("abc\n")
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}
	if spans[0] != (LineSpan{0, 3}) {
		t.Errorf("span[0]: expected {0,3}, got %v", spans[0])
	}
	if spans[1] != (LineSpan{4, 4}) {
		t.Errorf("span[1]: expected {4,4}, got %v", spans[1])
	}
}

func TestLinesEmpty(t *testing.T) {
	spans := Lines("")
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0] != (LineSpan{0, 0}) {
		t.Errorf("expected {0,0}, got %v", spans[0])
	}
}

func TestLineStartMiddle(t *testing.T) {
	// "ab\ncd\nef" — offset 4 is within "cd"
	got := LineStart("ab\ncd\nef", 4)
	if got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestLineStartAtNewline(t *testing.T) {
	// offset at the \n between "ab" and "cd"
	got := LineStart("ab\ncd\nef", 2)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestLineEndMiddle(t *testing.T) {
	got := LineEnd("ab\ncd\nef", 4)
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestLineEndLastLine(t *testing.T) {
	got := LineEnd("ab\ncd", 4)
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestLineCount(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 1},
		{"abc", 1},
		{"a\nb", 2},
		{"a\nb\nc", 3},
		{"a\n", 2},
		{"\n\n", 3},
	}
	for _, tt := range tests {
		if got := LineCount(tt.text); got != tt.want {
			t.Errorf("LineCount(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestCursorUpBasic(t *testing.T) {
	// "abc\ndef" — cursor at offset 5 ("ef", col 1) → should go to offset 1 ("bc", col 1)
	got := CursorUp("abc\ndef", 5)
	if got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestCursorUpFirstLine(t *testing.T) {
	got := CursorUp("abc\ndef", 2)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestCursorUpShorterPrevLine(t *testing.T) {
	// "ab\ncdef" — cursor at offset 6 ("ef", col 3) → prev line has 2 chars, clamp to end
	got := CursorUp("ab\ncdef", 6)
	if got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestCursorDownBasic(t *testing.T) {
	// "abc\ndef" — cursor at offset 1 ("bc", col 1) → should go to offset 5 ("ef", col 1)
	got := CursorDown("abc\ndef", 1)
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestCursorDownLastLine(t *testing.T) {
	got := CursorDown("abc\ndef", 5)
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestCursorDownShorterNextLine(t *testing.T) {
	// "abcd\nef" — cursor at offset 3 ("d", col 3) → next line has 2 chars, clamp to end
	got := CursorDown("abcd\nef", 3)
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestCursorUpDownEmoji(t *testing.T) {
	// Each emoji flag is a single grapheme cluster but multiple bytes.
	// "🇺🇸x\n🇬🇧y" — col 0 = flag, col 1 = letter
	text := "🇺🇸x\n🇬🇧y"
	flagLen := len("🇺🇸") // 8 bytes
	// Cursor at "x" (col 1) on first line
	down := CursorDown(text, flagLen)
	// Should land on "y" (col 1) on second line
	expectedDown := flagLen + 1 + 1 + len("🇬🇧") // past first line \n + flag on second line
	if down != expectedDown {
		t.Errorf("CursorDown: expected %d, got %d", expectedDown, down)
	}
	// And back up
	up := CursorUp(text, down)
	if up != flagLen {
		t.Errorf("CursorUp: expected %d, got %d", flagLen, up)
	}
}

func TestCursorUpDownEmptyLines(t *testing.T) {
	// "abc\n\ndef"
	text := "abc\n\ndef"
	// Cursor at start of "def" (offset 5), col 0
	up := CursorUp(text, 5)
	// Should go to empty line (offset 4), col 0 → stays at 4
	if up != 4 {
		t.Errorf("CursorUp to empty line: expected 4, got %d", up)
	}
	// From empty line, go up to "abc"
	up2 := CursorUp(text, 4)
	if up2 != 0 {
		t.Errorf("CursorUp from empty line: expected 0, got %d", up2)
	}
	// From "abc" col 0, go down to empty line
	down := CursorDown(text, 0)
	if down != 4 {
		t.Errorf("CursorDown to empty line: expected 4, got %d", down)
	}
}

func TestCursorUpDownTableDriven(t *testing.T) {
	text := "Hello\nWorld\n!"
	tests := []struct {
		name   string
		fn     func(string, int) int
		offset int
		want   int
	}{
		{"up from World[0]", CursorUp, 6, 0},
		{"up from World[3]", CursorUp, 9, 3},
		{"up from ![0]", CursorUp, 12, 6},
		{"down from Hello[0]", CursorDown, 0, 6},
		{"down from Hello[4]", CursorDown, 4, 10},
		{"down from World[0]", CursorDown, 6, 12},
		{"down from ![0]", CursorDown, 12, 13},  // last line → len
		{"up from Hello[0]", CursorUp, 0, 0},    // first line → 0
		{"down clamp", CursorDown, 4, 10},        // Hello col 4 → World col 4 = 'l' at offset 10
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(text, tt.offset)
			if got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}
