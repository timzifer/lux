// Unicode line-breaking for the lux text stack (RFC-003 §3.6).
//
// Provides UAX #14-conformant line break detection using rivo/uniseg.
// Thai, CJK, and other complex scripts are handled correctly.
package text

import (
	"github.com/rivo/uniseg"
)

// LineBreakKind classifies a line break opportunity.
type LineBreakKind uint8

const (
	// LineBreakMandatory indicates a mandatory break (e.g. newline).
	LineBreakMandatory LineBreakKind = iota
	// LineBreakOpportunity indicates the text may be broken here.
	LineBreakOpportunity
)

// LineBreak describes a single line break position in text.
type LineBreak struct {
	Offset int           // byte offset in the original text
	Kind   LineBreakKind // mandatory or opportunity
}

// LineBreaker segments text into breakable units per UAX #14.
type LineBreaker interface {
	// Breaks returns the allowed break positions in text.
	// Each break is reported at the byte offset where a break may occur
	// (i.e. between the preceding segment and the next).
	Breaks(text string) []LineBreak
}

// UnicodeLineBreaker implements LineBreaker using rivo/uniseg (UAX #14).
type UnicodeLineBreaker struct{}

// Breaks returns all line break positions in text.
func (UnicodeLineBreaker) Breaks(text string) []LineBreak {
	if len(text) == 0 {
		return nil
	}

	var breaks []LineBreak
	remaining := text
	state := -1
	offset := 0

	for len(remaining) > 0 {
		var segment string
		var mustBreak bool
		segment, remaining, mustBreak, state = uniseg.FirstLineSegmentInString(remaining, state)
		offset += len(segment)

		// Don't report a break at the very end of the text.
		if len(remaining) == 0 {
			break
		}

		if mustBreak {
			breaks = append(breaks, LineBreak{Offset: offset, Kind: LineBreakMandatory})
		} else {
			breaks = append(breaks, LineBreak{Offset: offset, Kind: LineBreakOpportunity})
		}
	}
	return breaks
}

// DefaultLineBreaker is the package-level UAX #14-conformant line breaker.
var DefaultLineBreaker LineBreaker = UnicodeLineBreaker{}
