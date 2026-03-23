package text

import (
	"testing"
)

func TestPrevGraphemeClusterASCII(t *testing.T) {
	text := "Hello"
	// From end of "Hello" (offset 5), should go to offset 4.
	got := PrevGraphemeCluster(text, 5)
	if got != 4 {
		t.Errorf("PrevGraphemeCluster(%q, 5) = %d, want 4", text, got)
	}
	// From offset 1, should go to 0.
	got = PrevGraphemeCluster(text, 1)
	if got != 0 {
		t.Errorf("PrevGraphemeCluster(%q, 1) = %d, want 0", text, got)
	}
}

func TestPrevGraphemeClusterAtStart(t *testing.T) {
	got := PrevGraphemeCluster("Hello", 0)
	if got != 0 {
		t.Errorf("PrevGraphemeCluster at start = %d, want 0", got)
	}
}

func TestPrevGraphemeClusterEmpty(t *testing.T) {
	got := PrevGraphemeCluster("", 0)
	if got != 0 {
		t.Errorf("PrevGraphemeCluster empty = %d, want 0", got)
	}
}

func TestPrevGraphemeClusterEmoji(t *testing.T) {
	// Flag emoji 🇩🇪 is 2 runes (regional indicators), 8 bytes, 1 grapheme cluster.
	text := "Hi 🇩🇪"
	end := len(text)
	got := PrevGraphemeCluster(text, end)
	// Should skip the entire flag emoji back to offset 3.
	if got != 3 {
		t.Errorf("PrevGraphemeCluster(%q, %d) = %d, want 3", text, end, got)
	}
}

func TestPrevGraphemeClusterCombining(t *testing.T) {
	// "e" + combining acute accent (U+0301) = 1 grapheme cluster.
	text := "e\u0301" // é as combining sequence
	got := PrevGraphemeCluster(text, len(text))
	if got != 0 {
		t.Errorf("PrevGraphemeCluster(combining é) = %d, want 0", got)
	}
}

func TestNextGraphemeClusterASCII(t *testing.T) {
	text := "Hello"
	got := NextGraphemeCluster(text, 0)
	if got != 1 {
		t.Errorf("NextGraphemeCluster(%q, 0) = %d, want 1", text, got)
	}
}

func TestNextGraphemeClusterEmoji(t *testing.T) {
	// ZWJ emoji: 👨‍👩‍👧 = 1 grapheme cluster, multiple runes.
	text := "👨\u200D👩\u200D👧"
	got := NextGraphemeCluster(text, 0)
	if got != len(text) {
		t.Errorf("NextGraphemeCluster(ZWJ emoji) = %d, want %d", got, len(text))
	}
}

func TestNextGraphemeClusterAtEnd(t *testing.T) {
	text := "Hi"
	got := NextGraphemeCluster(text, len(text))
	if got != len(text) {
		t.Errorf("NextGraphemeCluster at end = %d, want %d", got, len(text))
	}
}

func TestGraphemeClustersASCII(t *testing.T) {
	boundaries := GraphemeClusters("ABC")
	// Expected: [0, 1, 2, 3]
	expected := []int{0, 1, 2, 3}
	if len(boundaries) != len(expected) {
		t.Fatalf("GraphemeClusters(ABC) = %v, want %v", boundaries, expected)
	}
	for i, b := range boundaries {
		if b != expected[i] {
			t.Errorf("boundary[%d] = %d, want %d", i, b, expected[i])
		}
	}
}

func TestGraphemeClustersFlag(t *testing.T) {
	text := "A🇩🇪B"
	boundaries := GraphemeClusters(text)
	// A(1 byte) + 🇩🇪(8 bytes) + B(1 byte) = 3 clusters.
	if len(boundaries) != 4 { // 0, 1, 9, 10
		t.Fatalf("GraphemeClusters(%q) = %v, want 4 boundaries", text, boundaries)
	}
}

func TestGraphemeClustersEmpty(t *testing.T) {
	boundaries := GraphemeClusters("")
	if len(boundaries) != 1 || boundaries[0] != 0 {
		t.Errorf("GraphemeClusters empty = %v, want [0]", boundaries)
	}
}

func TestDeleteBackwardASCII(t *testing.T) {
	result, off := DeleteBackward("Hello", 5)
	if result != "Hell" || off != 4 {
		t.Errorf("DeleteBackward(Hello, 5) = (%q, %d), want (Hell, 4)", result, off)
	}
}

func TestDeleteBackwardEmoji(t *testing.T) {
	text := "Hello 🇩🇪"
	result, off := DeleteBackward(text, len(text))
	if result != "Hello " || off != 6 {
		t.Errorf("DeleteBackward(%q, %d) = (%q, %d), want (%q, %d)", text, len(text), result, off, "Hello ", 6)
	}
}

func TestDeleteBackwardCombining(t *testing.T) {
	// "He\u0301llo" — delete backward from offset 4 (after é) should remove "e\u0301".
	text := "He\u0301llo"
	offsetAfterE := len("He\u0301") // 1+1+2 = 4 bytes
	result, off := DeleteBackward(text, offsetAfterE)
	if result != "Hllo" || off != 1 {
		t.Errorf("DeleteBackward combining = (%q, %d), want (Hllo, 1)", result, off)
	}
}

func TestDeleteBackwardAtStart(t *testing.T) {
	result, off := DeleteBackward("Hello", 0)
	if result != "Hello" || off != 0 {
		t.Errorf("DeleteBackward at start = (%q, %d), want (Hello, 0)", result, off)
	}
}

func TestDeleteBackwardEmpty(t *testing.T) {
	result, off := DeleteBackward("", 0)
	if result != "" || off != 0 {
		t.Errorf("DeleteBackward empty = (%q, %d), want ('', 0)", result, off)
	}
}

func TestWordAtSimple(t *testing.T) {
	text := "Hello World"
	start, end := WordAt(text, 2) // within "Hello"
	if start != 0 || end != 5 {
		t.Errorf("WordAt(%q, 2) = (%d, %d), want (0, 5)", text, start, end)
	}
	start, end = WordAt(text, 7) // within "World"
	if start != 6 || end != 11 {
		t.Errorf("WordAt(%q, 7) = (%d, %d), want (6, 11)", text, start, end)
	}
}

func TestWordAtEmpty(t *testing.T) {
	start, end := WordAt("", 0)
	if start != 0 || end != 0 {
		t.Errorf("WordAt empty = (%d, %d), want (0, 0)", start, end)
	}
}

func TestPrevWordBoundary(t *testing.T) {
	text := "Hello World Test"
	got := PrevWordBoundary(text, 16) // from end
	if got != 12 {
		t.Errorf("PrevWordBoundary(%q, 16) = %d, want 12", text, got)
	}
	got = PrevWordBoundary(text, 12) // from start of "Test"
	if got != 6 {
		t.Errorf("PrevWordBoundary(%q, 12) = %d, want 6", text, got)
	}
}

func TestNextWordBoundary(t *testing.T) {
	text := "Hello World"
	got := NextWordBoundary(text, 0)
	if got != 5 {
		t.Errorf("NextWordBoundary(%q, 0) = %d, want 5", text, got)
	}
	got = NextWordBoundary(text, 6)
	if got != 11 {
		t.Errorf("NextWordBoundary(%q, 6) = %d, want 11", text, got)
	}
}
