// BiDi support for the lux text stack (RFC-003 §3.5).
//
// BidiParagraph analyses a paragraph of text using the Unicode Bidirectional
// Algorithm (UAX #9) and returns visually ordered ShapingRuns.
package text

import (
	"unicode"

	"golang.org/x/text/unicode/bidi"
)

// BidiParagraph analyses text and returns visually ordered ShapingRuns.
// baseDir controls the paragraph base direction:
//   - TextDirectionLTR: explicit left-to-right
//   - TextDirectionRTL: explicit right-to-left
//   - TextDirectionAuto: derived from first strong directional character (UAX#9 P2/P3)
func BidiParagraph(text string, baseDir TextDirection) []ShapingRun {
	if len(text) == 0 {
		return nil
	}

	var p bidi.Paragraph
	opts := bidiOption(baseDir)
	if opts != nil {
		p.SetString(text, opts)
	} else {
		p.SetString(text)
	}

	ordering, err := p.Order()
	if err != nil {
		// Fallback: return the entire text as a single LTR run.
		return []ShapingRun{{
			Text:      text,
			Direction: TextDirectionLTR,
			Script:    detectScript(text),
		}}
	}

	n := ordering.NumRuns()
	if n == 0 {
		return nil
	}

	runs := make([]ShapingRun, 0, n)
	for i := 0; i < n; i++ {
		r := ordering.Run(i)
		dir := TextDirectionLTR
		if r.Direction() == bidi.RightToLeft {
			dir = TextDirectionRTL
		}
		runText := r.String()
		runs = append(runs, ShapingRun{
			Text:      runText,
			Direction: dir,
			Script:    detectScript(runText),
		})
	}
	return runs
}

// bidiOption maps TextDirection to a bidi.Option.
// Returns nil for TextDirectionAuto (let the algorithm detect).
func bidiOption(dir TextDirection) bidi.Option {
	switch dir {
	case TextDirectionLTR:
		return bidi.DefaultDirection(bidi.LeftToRight)
	case TextDirectionRTL:
		return bidi.DefaultDirection(bidi.RightToLeft)
	default:
		return nil
	}
}

// detectScript returns an ISO 15924 script tag for the dominant script
// in the given text. It checks the first strong script character found.
func detectScript(text string) string {
	for _, r := range text {
		if s := runeScript(r); s != "" {
			return s
		}
	}
	return "Latn" // default
}

// runeScript returns an ISO 15924 tag for common Unicode script ranges.
// Returns "" for characters that don't belong to a recognised script
// (digits, punctuation, whitespace, etc.).
func runeScript(r rune) string {
	switch {
	case unicode.Is(unicode.Latin, r):
		return "Latn"
	case unicode.Is(unicode.Arabic, r):
		return "Arab"
	case unicode.Is(unicode.Hebrew, r):
		return "Hebr"
	case unicode.Is(unicode.Devanagari, r):
		return "Deva"
	case unicode.Is(unicode.Han, r):
		return "Hani"
	case unicode.Is(unicode.Hangul, r):
		return "Hang"
	case unicode.Is(unicode.Hiragana, r):
		return "Hira"
	case unicode.Is(unicode.Katakana, r):
		return "Kana"
	case unicode.Is(unicode.Thai, r):
		return "Thai"
	case unicode.Is(unicode.Bengali, r):
		return "Beng"
	case unicode.Is(unicode.Tamil, r):
		return "Taml"
	case unicode.Is(unicode.Georgian, r):
		return "Geor"
	case unicode.Is(unicode.Armenian, r):
		return "Armn"
	case unicode.Is(unicode.Greek, r):
		return "Grek"
	case unicode.Is(unicode.Cyrillic, r):
		return "Cyrl"
	default:
		return ""
	}
}
