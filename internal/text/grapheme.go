// Grapheme-cluster navigation for the lux text stack (RFC-003 §3.7).
//
// All cursor operations work on grapheme cluster boundaries (UAX #29),
// not byte or rune indices. This correctly handles emoji sequences,
// combining characters, and regional indicators.
package text

import (
	"github.com/rivo/uniseg"
)

// PrevGraphemeCluster returns the byte offset of the start of the
// grapheme cluster immediately before offset. Returns 0 if offset is
// at or before the start of text.
func PrevGraphemeCluster(text string, offset int) int {
	if offset <= 0 || len(text) == 0 {
		return 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	// Walk grapheme clusters from the start, tracking the boundary
	// just before the one containing offset.
	prev := 0
	pos := 0
	remaining := text
	state := -1

	for len(remaining) > 0 {
		cluster, rest, _, newState := uniseg.FirstGraphemeClusterInString(remaining, state)
		clusterEnd := pos + len(cluster)
		if clusterEnd >= offset {
			return prev
		}
		prev = pos + len(cluster)
		pos = clusterEnd
		remaining = rest
		state = newState
	}
	return prev
}

// NextGraphemeCluster returns the byte offset immediately after the
// grapheme cluster starting at offset. Returns len(text) if offset is
// at or past the end.
func NextGraphemeCluster(text string, offset int) int {
	if offset >= len(text) || len(text) == 0 {
		return len(text)
	}
	if offset < 0 {
		offset = 0
	}

	// Walk clusters until we pass offset.
	pos := 0
	remaining := text
	state := -1

	for len(remaining) > 0 {
		cluster, rest, _, newState := uniseg.FirstGraphemeClusterInString(remaining, state)
		clusterEnd := pos + len(cluster)
		if pos >= offset {
			return clusterEnd
		}
		pos = clusterEnd
		remaining = rest
		state = newState
	}
	return len(text)
}

// GraphemeClusters returns the byte offsets of all grapheme cluster
// boundaries in text, including 0 and len(text).
func GraphemeClusters(text string) []int {
	if len(text) == 0 {
		return []int{0}
	}

	boundaries := []int{0}
	remaining := text
	pos := 0
	state := -1

	for len(remaining) > 0 {
		cluster, rest, _, newState := uniseg.FirstGraphemeClusterInString(remaining, state)
		pos += len(cluster)
		boundaries = append(boundaries, pos)
		remaining = rest
		state = newState
	}
	return boundaries
}

// PrevWordBoundary returns the byte offset of the start of the word
// before offset (UAX #29 word boundaries). Skips non-word characters
// (whitespace, punctuation) to find the preceding word start.
func PrevWordBoundary(text string, offset int) int {
	if offset <= 0 || len(text) == 0 {
		return 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	// Collect word boundaries.
	var wordStarts []int
	pos := 0
	remaining := text
	state := -1
	inWord := false

	for len(remaining) > 0 {
		word, rest, newState := uniseg.FirstWordInString(remaining, state)
		// A "word" segment that starts with a letter/digit is a real word.
		isWord := false
		for _, r := range word {
			if isWordRune(r) {
				isWord = true
				break
			}
		}
		if isWord && !inWord {
			wordStarts = append(wordStarts, pos)
		}
		inWord = isWord
		pos += len(word)
		remaining = rest
		state = newState
	}

	// Find the last word start before offset.
	result := 0
	for _, ws := range wordStarts {
		if ws < offset {
			result = ws
		} else {
			break
		}
	}
	return result
}

// NextWordBoundary returns the byte offset of the end of the word
// after offset (UAX #29 word boundaries).
func NextWordBoundary(text string, offset int) int {
	if offset >= len(text) || len(text) == 0 {
		return len(text)
	}
	if offset < 0 {
		offset = 0
	}

	pos := 0
	remaining := text
	state := -1

	for len(remaining) > 0 {
		word, rest, newState := uniseg.FirstWordInString(remaining, state)
		wordEnd := pos + len(word)

		if wordEnd > offset {
			// This segment crosses offset.
			isWord := false
			for _, r := range word {
				if isWordRune(r) {
					isWord = true
					break
				}
			}
			if isWord {
				return wordEnd
			}
			// Non-word segment (whitespace/punctuation) — keep going.
		}

		pos = wordEnd
		remaining = rest
		state = newState
	}
	return len(text)
}

// WordAt returns the (start, end) byte offsets of the word containing
// the given offset. Useful for double-click word selection.
// If offset is in whitespace, returns the whitespace span.
func WordAt(text string, offset int) (int, int) {
	if len(text) == 0 {
		return 0, 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	pos := 0
	remaining := text
	state := -1

	for len(remaining) > 0 {
		word, rest, newState := uniseg.FirstWordInString(remaining, state)
		wordEnd := pos + len(word)
		if wordEnd > offset {
			return pos, wordEnd
		}
		pos = wordEnd
		remaining = rest
		state = newState
	}
	return pos, len(text)
}

// DeleteBackward removes one grapheme cluster before offset and returns
// the resulting text and the new cursor offset. If offset is 0 or text
// is empty, returns the original text and offset unchanged.
func DeleteBackward(text string, offset int) (string, int) {
	if offset <= 0 || len(text) == 0 {
		return text, offset
	}
	if offset > len(text) {
		offset = len(text)
	}
	prev := PrevGraphemeCluster(text, offset)
	return text[:prev] + text[offset:], prev
}

// DeleteForward removes one grapheme cluster after offset and returns
// the resulting text and the (unchanged) cursor offset.
func DeleteForward(text string, offset int) (string, int) {
	if offset >= len(text) || len(text) == 0 {
		return text, offset
	}
	if offset < 0 {
		offset = 0
	}
	next := NextGraphemeCluster(text, offset)
	return text[:offset] + text[next:], offset
}

// isWordRune returns true for runes that are part of a "word"
// (letters, digits, combining marks).
func isWordRune(r rune) bool {
	// Simple heuristic: letters and digits are word characters.
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' ||
		r > 127 // non-ASCII likely a letter in some script
}
