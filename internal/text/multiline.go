package text

import (
	"strings"

	"github.com/rivo/uniseg"
)

// LineSpan represents a single line within a multiline string.
type LineSpan struct {
	Start int // byte offset (inclusive)
	End   int // byte offset (exclusive, before the \n)
}

// Lines splits text by \n and returns the byte spans for each line.
// A trailing \n produces an empty final line.
func Lines(text string) []LineSpan {
	if len(text) == 0 {
		return []LineSpan{{0, 0}}
	}
	var spans []LineSpan
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			spans = append(spans, LineSpan{start, i})
			start = i + 1
		}
	}
	spans = append(spans, LineSpan{start, len(text)})
	return spans
}

// LineStart returns the byte offset of the start of the line containing offset.
func LineStart(text string, offset int) int {
	if offset <= 0 {
		return 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	// Scan backward for \n.
	for i := offset - 1; i >= 0; i-- {
		if text[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// LineEnd returns the byte offset of the end of the line containing offset
// (exclusive, before the \n or at len(text)).
func LineEnd(text string, offset int) int {
	if offset >= len(text) {
		return len(text)
	}
	if offset < 0 {
		offset = 0
	}
	idx := strings.IndexByte(text[offset:], '\n')
	if idx < 0 {
		return len(text)
	}
	return offset + idx
}

// LineCount returns the number of lines in text.
func LineCount(text string) int {
	return strings.Count(text, "\n") + 1
}

// CursorUp moves the cursor to the previous line at the same grapheme column.
// If the cursor is on the first line, returns 0.
func CursorUp(text string, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	ls := LineStart(text, offset)
	if ls == 0 {
		// Already on the first line.
		return 0
	}
	col := graphemeColumn(text, ls, offset)
	// Previous line ends at ls-1 (the \n). Its start is found by scanning back.
	prevLineEnd := ls - 1 // byte before the \n
	prevLineStart := LineStart(text, prevLineEnd)
	return advanceByGraphemes(text, prevLineStart, prevLineEnd, col)
}

// CursorDown moves the cursor to the next line at the same grapheme column.
// If the cursor is on the last line, returns len(text).
func CursorDown(text string, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	ls := LineStart(text, offset)
	le := LineEnd(text, offset)
	if le >= len(text) {
		// Already on the last line.
		return len(text)
	}
	col := graphemeColumn(text, ls, offset)
	// Next line starts at le+1 (after the \n).
	nextLineStart := le + 1
	nextLineEnd := LineEnd(text, nextLineStart)
	return advanceByGraphemes(text, nextLineStart, nextLineEnd, col)
}

// graphemeColumn counts the number of grapheme clusters from lineStart to offset.
func graphemeColumn(text string, lineStart, offset int) int {
	if offset <= lineStart {
		return 0
	}
	segment := text[lineStart:offset]
	count := 0
	state := -1
	for len(segment) > 0 {
		_, rest, _, newState := uniseg.FirstGraphemeClusterInString(segment, state)
		count++
		segment = rest
		state = newState
	}
	return count
}

// advanceByGraphemes advances count grapheme clusters from start, clamping at limit.
func advanceByGraphemes(text string, start, limit, count int) int {
	if start >= limit || count <= 0 {
		return start
	}
	segment := text[start:limit]
	pos := start
	state := -1
	for i := 0; i < count && len(segment) > 0; i++ {
		cluster, rest, _, newState := uniseg.FirstGraphemeClusterInString(segment, state)
		pos += len(cluster)
		segment = rest
		state = newState
	}
	return pos
}
