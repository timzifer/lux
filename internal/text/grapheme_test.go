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

// ── Skin Tone Emoji ──────────────────────────────────────────────

func TestPrevGraphemeClusterSkinTone(t *testing.T) {
	// Woman + medium skin tone = 1 grapheme cluster (8 bytes, 2 runes).
	text := "\U0001F469\U0001F3FD"
	got := PrevGraphemeCluster(text, len(text))
	if got != 0 {
		t.Errorf("PrevGraphemeCluster(skin-tone emoji, %d) = %d, want 0", len(text), got)
	}
}

func TestNextGraphemeClusterSkinTone(t *testing.T) {
	text := "\U0001F469\U0001F3FD"
	got := NextGraphemeCluster(text, 0)
	if got != len(text) {
		t.Errorf("NextGraphemeCluster(skin-tone emoji, 0) = %d, want %d", got, len(text))
	}
}

// ── Hangul Jamo ──────────────────────────────────────────────────

func TestGraphemeClustersHangulJamo(t *testing.T) {
	// Decomposed Hangul syllable: L + V + T = 1 grapheme cluster.
	text := "\u1100\u1161\u11A8" // ᄀ + ᅡ + ᆨ = 각
	boundaries := GraphemeClusters(text)
	// Should be [0, len(text)] — a single cluster.
	if len(boundaries) != 2 {
		t.Errorf("GraphemeClusters(Hangul jamo) = %v, want 2 boundaries (1 cluster)", boundaries)
	}
}

func TestPrevGraphemeClusterHangulJamo(t *testing.T) {
	text := "\u1100\u1161\u11A8"
	got := PrevGraphemeCluster(text, len(text))
	if got != 0 {
		t.Errorf("PrevGraphemeCluster(Hangul jamo, %d) = %d, want 0", len(text), got)
	}
}

// ── Variation Selector ───────────────────────────────────────────

func TestGraphemeClustersVariationSelector(t *testing.T) {
	// Heart + VS16 = 1 grapheme cluster.
	text := "\u2764\uFE0F"
	boundaries := GraphemeClusters(text)
	if len(boundaries) != 2 {
		t.Errorf("GraphemeClusters(heart+VS16) = %v, want 2 boundaries (1 cluster)", boundaries)
	}
}

func TestNextGraphemeClusterVariationSelector(t *testing.T) {
	text := "\u2764\uFE0F"
	got := NextGraphemeCluster(text, 0)
	if got != len(text) {
		t.Errorf("NextGraphemeCluster(heart+VS16, 0) = %d, want %d", got, len(text))
	}
}

// ── Keycap Sequence ──────────────────────────────────────────────

func TestGraphemeClustersKeycap(t *testing.T) {
	// Keycap 3: "3" + VS16 + combining enclosing keycap = 1 cluster.
	text := "3\uFE0F\u20E3"
	boundaries := GraphemeClusters(text)
	if len(boundaries) != 2 {
		t.Errorf("GraphemeClusters(keycap 3) = %v, want 2 boundaries (1 cluster)", boundaries)
	}
}

// ── Multi-person ZWJ ─────────────────────────────────────────────

func TestGraphemeClustersMultipleZWJ(t *testing.T) {
	// Family: man + ZWJ + woman + ZWJ + girl + ZWJ + boy = 1 cluster.
	text := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	boundaries := GraphemeClusters(text)
	if len(boundaries) != 2 {
		t.Errorf("GraphemeClusters(family emoji) = %v, want 2 boundaries (1 cluster)", boundaries)
	}
}

// ── Mid-cluster Offset ───────────────────────────────────────────

func TestPrevGraphemeClusterMidCluster(t *testing.T) {
	// Flag emoji is 8 bytes (2 regional indicators). Offset 4 is mid-cluster.
	text := "\U0001F1E9\U0001F1EA" // 🇩🇪
	got := PrevGraphemeCluster(text, 4)
	if got != 0 {
		t.Errorf("PrevGraphemeCluster(flag, mid-cluster offset 4) = %d, want 0", got)
	}
}

// ── Delete Backward for Complex Emoji ────────────────────────────

func TestDeleteBackwardSkinTone(t *testing.T) {
	text := "Hi\U0001F469\U0001F3FD"
	result, off := DeleteBackward(text, len(text))
	if result != "Hi" || off != 2 {
		t.Errorf("DeleteBackward(skin-tone) = (%q, %d), want (\"Hi\", 2)", result, off)
	}
}

func TestDeleteBackwardZWJFamily(t *testing.T) {
	text := "\U0001F468\u200D\U0001F469\u200D\U0001F467"
	result, off := DeleteBackward(text, len(text))
	if result != "" || off != 0 {
		t.Errorf("DeleteBackward(ZWJ family) = (%q, %d), want (\"\", 0)", result, off)
	}
}

// ── CJK Word Boundaries ─────────────────────────────────────────

func TestWordAtCJK(t *testing.T) {
	text := "你好世界"
	start, end := WordAt(text, 0)
	// CJK characters are each treated as separate word segments by uniseg.
	if start != 0 {
		t.Errorf("WordAt CJK start = %d, want 0", start)
	}
	if end > len(text) {
		t.Errorf("WordAt CJK end = %d, exceeds text length %d", end, len(text))
	}
}

func TestPrevWordBoundaryCJK(t *testing.T) {
	text := "你好世界"
	got := PrevWordBoundary(text, len(text))
	// Should move backward to some word boundary within the CJK text.
	if got >= len(text) {
		t.Errorf("PrevWordBoundary(CJK, end) = %d, want < %d", got, len(text))
	}
}

// ── Table-Driven Consolidation ───────────────────────────────────

func TestGraphemeClustersTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		clusters int // expected number of grapheme clusters
	}{
		{"ASCII", "Hello", 5},
		{"empty", "", 0},
		{"flag", "\U0001F1E9\U0001F1EA", 1},
		{"skin-tone", "\U0001F469\U0001F3FD", 1},
		{"combining", "e\u0301", 1},
		{"keycap", "3\uFE0F\u20E3", 1},
		{"ZWJ-family", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", 1},
		{"variation-selector", "\u2764\uFE0F", 1},
		{"Hangul-jamo", "\u1100\u1161\u11A8", 1},
		{"multi-flag", "\U0001F1E9\U0001F1EA\U0001F1EB\U0001F1F7", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boundaries := GraphemeClusters(tt.text)
			got := len(boundaries) - 1 // clusters = boundaries - 1
			if tt.text == "" {
				got = 0 // special case: [0] → 0 clusters
			}
			if got != tt.clusters {
				t.Errorf("GraphemeClusters(%q) = %d clusters, want %d (boundaries: %v)",
					tt.text, got, tt.clusters, boundaries)
			}
		})
	}
}
