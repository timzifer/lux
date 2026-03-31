package richtext

import (
	"github.com/timzifer/lux/draw"
)

// ── AttributedString (user-facing, serializable) ────────────────

// AttributedString is the serializable document content (RFC-003 §5.6).
// It stores plain text and a parallel run-length-encoded list of style
// attributes. Inspired by Apple's NSAttributedString.
//
// Runs are contiguous and non-overlapping:
//
//	Run[0] covers bytes [0, Run[0].End)
//	Run[i] covers bytes [Run[i-1].End, Run[i].End)
//
// The last run's End equals len(Text).
type AttributedString struct {
	Text  string    // complete plain text including \n for paragraphs
	Attrs []AttrRun // run-length-encoded, ascending by End
}

// AttrRun describes formatting up to a byte offset (exclusive).
type AttrRun struct {
	End   int       // exclusive byte offset
	Style SpanStyle // formatting for this run
}

// ImageAttachment describes an image embedded inline in the document.
// An ImageID of 0 means "no image" (zero value is safe to embed in SpanStyle).
// The in-text placeholder is U+FFFC (OBJECT REPLACEMENT CHARACTER, 3 UTF-8 bytes).
//
// Use InsertImage to add an image to an AttributedString.
type ImageAttachment struct {
	ImageID   draw.ImageID        // 0 = no image (zero value)
	Alt       string              // accessibility / alt text
	Width     float32             // dp; 0 = use Height; both 0 = use line height
	Height    float32             // dp; 0 = use Width; both 0 = use line height
	ScaleMode draw.ImageScaleMode // default ImageScaleStretch
	Opacity   float32             // 0 = 1.0 (fully opaque)
}

// WhiteSpace controls how whitespace is handled within a run (CSS white-space).
type WhiteSpace uint8

const (
	// WhiteSpaceNormal collapses whitespace sequences into a single space
	// and allows wrapping at soft wrap opportunities (CSS: normal).
	WhiteSpaceNormal WhiteSpace = iota

	// WhiteSpacePre preserves all whitespace and only breaks at preserved
	// newline characters (CSS: pre).
	WhiteSpacePre

	// WhiteSpaceNoWrap collapses whitespace like Normal but suppresses
	// line breaks within the text (CSS: nowrap).
	WhiteSpaceNoWrap

	// WhiteSpacePreWrap preserves whitespace sequences and wraps at soft
	// wrap opportunities and preserved newlines (CSS: pre-wrap).
	WhiteSpacePreWrap

	// WhiteSpacePreLine collapses whitespace sequences into a single space
	// but preserves newline characters for line breaking (CSS: pre-line).
	WhiteSpacePreLine
)

// SpanStyle overrides text style for a run.
// Zero values inherit from the theme's Body style.
// When Image.ImageID != 0 the run represents an embedded image; formatting
// fields (Bold/Italic/Underline/…) are ignored for that run.
type SpanStyle struct {
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	FontFamily    string          // empty = inherit from theme
	Weight        draw.FontWeight // 0 = inherit (Bold flag overrides to 700)
	Color         draw.Color      // zero = theme Text.Primary
	BgColor       draw.Color      // zero = transparent (no highlight)
	Size          float32         // zero = inherit from theme Body
	Tracking      float32         // letter-spacing in em; 0 = inherit
	LineHeight    float32         // multiplier; 0 = inherit
	WhiteSpace    WhiteSpace      // 0 = WhiteSpaceNormal
	Image         ImageAttachment // zero value = no image (ImageID == 0)
}

// ── Constructors ────────────────────────────────────────────────

// NewAttributedString creates an AttributedString from plain text
// with default (unstyled) formatting.
func NewAttributedString(text string) AttributedString {
	if text == "" {
		return AttributedString{}
	}
	return AttributedString{
		Text:  text,
		Attrs: []AttrRun{{End: len(text)}},
	}
}

// Styled creates a single-run AttributedString with the given style.
func Styled(text string, style SpanStyle) AttributedString {
	if text == "" {
		return AttributedString{}
	}
	return AttributedString{
		Text:  text,
		Attrs: []AttrRun{{End: len(text), Style: style}},
	}
}

// Build constructs an AttributedString from styled segments,
// similar to how one would build an NSAttributedString.
//
//	richtext.Build(
//	    richtext.S("Hello ", richtext.SpanStyle{Bold: true}),
//	    richtext.S("World"),
//	)
func Build(segments ...Segment) AttributedString {
	if len(segments) == 0 {
		return AttributedString{}
	}
	var buf []byte
	var runs []AttrRun
	for _, seg := range segments {
		buf = append(buf, seg.Text...)
		runs = append(runs, AttrRun{End: len(buf), Style: seg.Style})
	}
	return AttributedString{Text: string(buf), Attrs: runs}
}

// Segment is a text+style pair used by Build.
type Segment struct {
	Text  string
	Style SpanStyle
}

// S creates a Segment for use with Build.
func S(text string, style ...SpanStyle) Segment {
	seg := Segment{Text: text}
	if len(style) > 0 {
		seg.Style = style[0]
	}
	return seg
}

// ── Accessors ───────────────────────────────────────────────────

// PlainText returns the plain text content.
func (as AttributedString) PlainText() string {
	return as.Text
}

// Len returns the byte length of the text.
func (as AttributedString) Len() int {
	return len(as.Text)
}

// IsEmpty returns true if the attributed string has no text.
func (as AttributedString) IsEmpty() bool {
	return len(as.Text) == 0
}

// RunAt returns the style at the given byte offset.
// Returns the zero SpanStyle if offset is out of range.
func (as AttributedString) RunAt(offset int) SpanStyle {
	if offset < 0 || len(as.Attrs) == 0 {
		return SpanStyle{}
	}
	for _, r := range as.Attrs {
		if offset < r.End {
			return r.Style
		}
	}
	// Past end — return last run's style.
	return as.Attrs[len(as.Attrs)-1].Style
}

// ── Mutation ────────────────────────────────────────────────────

// InsertImage inserts an image at the given byte offset.
//
// The image is represented in the text by a U+FFFC (OBJECT REPLACEMENT
// CHARACTER) placeholder (3 UTF-8 bytes). The returned AttributedString has
// an attribute run at [offset, offset+3) carrying SpanStyle{Image: img}.
// Surrounding styles are preserved.
func (as AttributedString) InsertImage(offset int, img ImageAttachment) AttributedString {
	const placeholder = "\uFFFC" // 3 bytes in UTF-8
	as = as.InsertText(offset, placeholder)
	return as.ApplyStyle(offset, offset+len(placeholder), SpanStyle{Image: img})
}

// InsertText inserts text at the given byte offset, inheriting the
// style of the character before the insertion point. Returns a new
// AttributedString (immutable semantics).
func (as AttributedString) InsertText(offset int, text string) AttributedString {
	if len(text) == 0 {
		return as
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(as.Text) {
		offset = len(as.Text)
	}

	newText := as.Text[:offset] + text + as.Text[offset:]
	ins := len(text)

	if len(as.Attrs) == 0 {
		return AttributedString{
			Text:  newText,
			Attrs: []AttrRun{{End: len(newText)}},
		}
	}

	newAttrs := make([]AttrRun, len(as.Attrs))
	for i, r := range as.Attrs {
		newAttrs[i] = r
		if r.End > offset {
			newAttrs[i].End = r.End + ins
		} else if r.End == offset {
			// Insertion at run boundary: extend this run.
			newAttrs[i].End = r.End + ins
		}
	}

	return AttributedString{Text: newText, Attrs: newAttrs}.Normalized()
}

// DeleteRange removes bytes [start, end) and adjusts attribute runs.
// Returns a new AttributedString.
func (as AttributedString) DeleteRange(start, end int) AttributedString {
	if start < 0 {
		start = 0
	}
	if end > len(as.Text) {
		end = len(as.Text)
	}
	if start >= end {
		return as
	}

	deleted := end - start
	newText := as.Text[:start] + as.Text[end:]

	if len(as.Attrs) == 0 {
		if newText == "" {
			return AttributedString{}
		}
		return AttributedString{
			Text:  newText,
			Attrs: []AttrRun{{End: len(newText)}},
		}
	}

	var newAttrs []AttrRun
	for _, r := range as.Attrs {
		nr := r
		if r.End <= start {
			// Run entirely before deletion — unchanged.
			newAttrs = append(newAttrs, nr)
		} else if r.End <= end {
			// Run ends within or at deletion boundary.
			nr.End = start
			if len(newAttrs) == 0 || nr.End > 0 {
				if nr.End > start {
					newAttrs = append(newAttrs, nr)
				} else if nr.End == start && (len(newAttrs) == 0 || newAttrs[len(newAttrs)-1].End < start) {
					// Preserve run that ends exactly at start.
					newAttrs = append(newAttrs, nr)
				}
			}
		} else {
			// Run extends past deletion.
			nr.End = r.End - deleted
			newAttrs = append(newAttrs, nr)
		}
	}

	if newText == "" {
		return AttributedString{}
	}

	// Ensure runs cover the full text.
	if len(newAttrs) == 0 {
		newAttrs = []AttrRun{{End: len(newText)}}
	} else if newAttrs[len(newAttrs)-1].End < len(newText) {
		newAttrs[len(newAttrs)-1].End = len(newText)
	}

	return AttributedString{Text: newText, Attrs: newAttrs}.Normalized()
}

// ApplyStyle sets the style for the byte range [start, end).
// Returns a new AttributedString.
func (as AttributedString) ApplyStyle(start, end int, style SpanStyle) AttributedString {
	if start < 0 {
		start = 0
	}
	if end > len(as.Text) {
		end = len(as.Text)
	}
	if start >= end || len(as.Attrs) == 0 {
		return as
	}

	var newAttrs []AttrRun
	prevEnd := 0

	for _, r := range as.Attrs {
		runStart := prevEnd
		runEnd := r.End
		prevEnd = runEnd

		if runEnd <= start || runStart >= end {
			// Run entirely outside the styled range — keep as-is.
			newAttrs = append(newAttrs, r)
			continue
		}

		// Split: part before styled range.
		if runStart < start {
			newAttrs = append(newAttrs, AttrRun{End: start, Style: r.Style})
		}

		// The styled part of this run.
		styledStart := runStart
		if styledStart < start {
			styledStart = start
		}
		styledEnd := runEnd
		if styledEnd > end {
			styledEnd = end
		}
		newAttrs = append(newAttrs, AttrRun{End: styledEnd, Style: style})

		// Split: part after styled range.
		if runEnd > end {
			newAttrs = append(newAttrs, AttrRun{End: runEnd, Style: r.Style})
		}
	}

	return AttributedString{Text: as.Text, Attrs: newAttrs}.Normalized()
}

// ToggleStyleFunc transforms the style of each run overlapping [start, end)
// using fn. Unlike ApplyStyle, it preserves attributes that fn does not
// modify — e.g. toggling Bold without losing Italic.
func (as AttributedString) ToggleStyleFunc(start, end int, fn func(SpanStyle) SpanStyle) AttributedString {
	if start < 0 {
		start = 0
	}
	if end > len(as.Text) {
		end = len(as.Text)
	}
	if start >= end || len(as.Attrs) == 0 {
		return as
	}

	var newAttrs []AttrRun
	prevEnd := 0

	for _, r := range as.Attrs {
		runStart := prevEnd
		runEnd := r.End
		prevEnd = runEnd

		if runEnd <= start || runStart >= end {
			newAttrs = append(newAttrs, r)
			continue
		}

		// Split: part before the transformed range.
		if runStart < start {
			newAttrs = append(newAttrs, AttrRun{End: start, Style: r.Style})
		}

		// The transformed part of this run.
		styledEnd := runEnd
		if styledEnd > end {
			styledEnd = end
		}
		newAttrs = append(newAttrs, AttrRun{End: styledEnd, Style: fn(r.Style)})

		// Split: part after the transformed range.
		if runEnd > end {
			newAttrs = append(newAttrs, AttrRun{End: runEnd, Style: r.Style})
		}
	}

	return AttributedString{Text: as.Text, Attrs: newAttrs}.Normalized()
}

// AllMatch reports whether fn returns true for every attribute run
// overlapping [start, end). Returns true for empty ranges.
func (as AttributedString) AllMatch(start, end int, fn func(SpanStyle) bool) bool {
	if start < 0 {
		start = 0
	}
	if end > len(as.Text) {
		end = len(as.Text)
	}
	if start >= end {
		return true
	}
	if len(as.Attrs) == 0 {
		return fn(SpanStyle{})
	}

	prevEnd := 0
	for _, r := range as.Attrs {
		runStart := prevEnd
		runEnd := r.End
		prevEnd = runEnd

		if runEnd <= start {
			continue
		}
		if runStart >= end {
			break
		}
		if !fn(r.Style) {
			return false
		}
	}
	return true
}

// Normalized merges adjacent runs with identical styles.
func (as AttributedString) Normalized() AttributedString {
	if len(as.Attrs) <= 1 {
		return as
	}

	merged := make([]AttrRun, 0, len(as.Attrs))
	for _, r := range as.Attrs {
		if len(merged) > 0 && merged[len(merged)-1].Style == r.Style {
			merged[len(merged)-1].End = r.End
		} else if r.End > 0 && (len(merged) == 0 || r.End > merged[len(merged)-1].End) {
			merged = append(merged, r)
		}
	}

	if len(merged) == len(as.Attrs) {
		return as
	}
	return AttributedString{Text: as.Text, Attrs: merged}
}

// ── Equality ────────────────────────────────────────────────────

// Equal reports whether two AttributedStrings are structurally equal.
func (as AttributedString) Equal(other AttributedString) bool {
	if as.Text != other.Text {
		return false
	}
	if len(as.Attrs) != len(other.Attrs) {
		return false
	}
	for i, a := range as.Attrs {
		b := other.Attrs[i]
		if a.End != b.End || a.Style != b.Style {
			return false
		}
	}
	return true
}

// ── Legacy interop (DocumentChangedMsg) ─────────────────────────

// DocumentChangedMsg is sent when the user edits the document.
type DocumentChangedMsg struct {
	Value AttributedString
}
