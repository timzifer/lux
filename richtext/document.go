package richtext

import (
	"sort"
	"strings"

	"github.com/timzifer/lux/draw"
)

// ── Attribute Interface (sealed) ───────────────────────────────

// attrTag is an unexported marker so only types in this package
// can implement Attribute.
type attrTag uint8

// Attribute is the sealed interface for typed attribute values.
// Each concrete type represents a single CSS-like property.
type Attribute interface {
	attrTag() attrTag
}

// ── Concrete Attribute Types ───────────────────────────────────

// Span-level attributes (inline text formatting).

type BoldAttr bool                  // CSS font-weight: bold
type ItalicAttr bool                // CSS font-style: italic
type UnderlineAttr bool             // CSS text-decoration: underline
type StrikethroughAttr bool         // CSS text-decoration: line-through
type FontFamilyAttr string          // CSS font-family
type WeightAttr draw.FontWeight     // CSS font-weight (100–900)
type ColorAttr draw.Color           // CSS color
type BgColorAttr draw.Color         // CSS background-color
type SizeAttr float32               // CSS font-size (dp)
type TrackingAttr float32           // CSS letter-spacing (em)
type LineHeightAttr float32         // CSS line-height (multiplier)
type WhiteSpaceAttr WhiteSpace      // CSS white-space
type ImageAttr ImageAttachment      // Inline image (U+FFFC placeholder)

// Paragraph-level attributes (block formatting).

type AlignAttr draw.TextAlign // CSS text-align
type IndentAttr float32       // CSS text-indent (dp)
type ParaSpacingAttr float32  // Paragraph spacing (dp)

// List-level attributes (paragraph formatting).

type ListTypeAttr draw.ListType     // ul/ol/none (CSS list-style-type category)
type ListLevelAttr int              // nesting depth (0-based)
type ListStartAttr int              // start number for ol (0 = default 1)
type ListMarkerAttr draw.ListMarker // bullet/number style (CSS list-style-type)

// attrTag implementations (sealed marker).

func (BoldAttr) attrTag() attrTag          { return 0 }
func (ItalicAttr) attrTag() attrTag        { return 1 }
func (UnderlineAttr) attrTag() attrTag     { return 2 }
func (StrikethroughAttr) attrTag() attrTag { return 3 }
func (FontFamilyAttr) attrTag() attrTag    { return 4 }
func (WeightAttr) attrTag() attrTag        { return 5 }
func (ColorAttr) attrTag() attrTag         { return 6 }
func (BgColorAttr) attrTag() attrTag       { return 7 }
func (SizeAttr) attrTag() attrTag          { return 8 }
func (TrackingAttr) attrTag() attrTag      { return 9 }
func (LineHeightAttr) attrTag() attrTag    { return 10 }
func (WhiteSpaceAttr) attrTag() attrTag    { return 11 }
func (ImageAttr) attrTag() attrTag         { return 12 }
func (AlignAttr) attrTag() attrTag         { return 13 }
func (IndentAttr) attrTag() attrTag        { return 14 }
func (ParaSpacingAttr) attrTag() attrTag   { return 15 }
func (ListTypeAttr) attrTag() attrTag      { return 16 }
func (ListLevelAttr) attrTag() attrTag     { return 17 }
func (ListStartAttr) attrTag() attrTag     { return 18 }
func (ListMarkerAttr) attrTag() attrTag    { return 19 }

// isParagraphAttr reports whether an attribute is paragraph-level (not inline).
// Paragraph-level attributes must be split at newline boundaries during
// InsertText and merged during DeleteRange so that each paragraph carries
// its own independent attribute ranges.
func isParagraphAttr(v Attribute) bool {
	switch v.(type) {
	case AlignAttr, IndentAttr, ParaSpacingAttr,
		ListTypeAttr, ListLevelAttr, ListStartAttr, ListMarkerAttr:
		return true
	}
	return false
}

// ── Attr (Tagged Range) ────────────────────────────────────────

// Attr is a typed attribute applied to a byte range [Start, End).
// Multiple Attrs of different types can overlap on the same text.
// Later Attrs in the slice take precedence over earlier ones for
// the same attribute type (last-writer-wins).
type Attr struct {
	Start int       // inclusive byte offset
	End   int       // exclusive byte offset
	Value Attribute // typed attribute value
}

// ── AttributedString ───────────────────────────────────────────

// AttributedString is the serializable document content (RFC-003 §5.6).
// It stores plain text and a list of typed, potentially overlapping
// attribute ranges. Inspired by Apple's NSAttributedString but using
// tagged ranges instead of run-length encoding.
type AttributedString struct {
	Text  string // complete plain text including \n for paragraphs
	Attrs []Attr // typed attribute ranges (order matters: last wins)
}

// ── SpanStyle (Resolved Output) ────────────────────────────────

// SpanStyle is the fully resolved style at a given byte offset.
// It is the output of ResolveAt() — NOT used for storage.
// Zero values mean "inherit from theme defaults".
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
	// Paragraph-level properties (CSS text-align, text-indent).
	Align       draw.TextAlign // 0 = TextAlignLeft (default)
	Indent      float32        // dp first-line indent; 0 = none
	ParaSpacing float32        // dp between paragraphs; 0 = theme default

	// List properties (CSS list-style).
	ListType   draw.ListType   // 0 = no list
	ListLevel  int             // nesting depth (0 = top level)
	ListStart  int             // ol start number; 0 = default (1)
	ListMarker draw.ListMarker // 0 = auto based on type and level
}

// ImageAttachment describes an image embedded inline in the document.
// An ImageID of 0 means "no image" (zero value is safe to embed).
// The in-text placeholder is U+FFFC (OBJECT REPLACEMENT CHARACTER, 3 UTF-8 bytes).
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
	WhiteSpaceNormal  WhiteSpace = iota // collapses whitespace, allows wrapping
	WhiteSpacePre                       // preserves all whitespace, breaks only at \n
	WhiteSpaceNoWrap                    // collapses whitespace, suppresses line breaks
	WhiteSpacePreWrap                   // preserves whitespace, allows soft wrapping
	WhiteSpacePreLine                   // collapses to single space, preserves \n
)

// ── Resolution ─────────────────────────────────────────────────

// ResolveAt returns the composed style at a byte offset.
// Later Attrs override earlier ones for the same attribute type.
func (as AttributedString) ResolveAt(offset int) SpanStyle {
	var s SpanStyle
	for _, a := range as.Attrs {
		if offset >= a.Start && offset < a.End {
			applyAttr(&s, a.Value)
		}
	}
	return s
}

// RunAt returns the style at the given byte offset.
// Alias for ResolveAt (backward compatibility).
func (as AttributedString) RunAt(offset int) SpanStyle {
	if offset < 0 || len(as.Attrs) == 0 {
		return SpanStyle{}
	}
	if offset >= len(as.Text) && len(as.Text) > 0 {
		// Past end — resolve at last valid offset.
		return as.ResolveAt(len(as.Text) - 1)
	}
	return as.ResolveAt(offset)
}

// ResolvedRun is a resolved style segment for efficient rendering.
type ResolvedRun struct {
	Start int
	End   int
	Style SpanStyle
}

// StyleRuns returns sorted, non-overlapping resolved style segments
// for the byte range [start, end). Each segment has a uniform style.
func (as AttributedString) StyleRuns(start, end int) []ResolvedRun {
	if start >= end {
		return nil
	}
	points := as.transitionsIn(start, end)
	if len(points) == 0 {
		return []ResolvedRun{{Start: start, End: end, Style: as.ResolveAt(start)}}
	}

	var runs []ResolvedRun
	for i := 0; i < len(points)-1; i++ {
		s := as.ResolveAt(points[i])
		run := ResolvedRun{Start: points[i], End: points[i+1], Style: s}
		// Merge with previous run if style is identical.
		if len(runs) > 0 && runs[len(runs)-1].Style == s {
			runs[len(runs)-1].End = run.End
		} else {
			runs = append(runs, run)
		}
	}
	return runs
}

// transitionsIn returns sorted unique transition points within [start, end).
// Transition points are all Attr Start/End offsets clipped to the range,
// plus start and end themselves.
func (as AttributedString) transitionsIn(start, end int) []int {
	seen := make(map[int]struct{})
	seen[start] = struct{}{}
	seen[end] = struct{}{}
	for _, a := range as.Attrs {
		if a.Start > start && a.Start < end {
			seen[a.Start] = struct{}{}
		}
		if a.End > start && a.End < end {
			seen[a.End] = struct{}{}
		}
	}
	points := make([]int, 0, len(seen))
	for p := range seen {
		points = append(points, p)
	}
	sort.Ints(points)
	return points
}

// applyAttr merges a single Attribute value into a SpanStyle.
func applyAttr(s *SpanStyle, v Attribute) {
	switch a := v.(type) {
	case BoldAttr:
		s.Bold = bool(a)
	case ItalicAttr:
		s.Italic = bool(a)
	case UnderlineAttr:
		s.Underline = bool(a)
	case StrikethroughAttr:
		s.Strikethrough = bool(a)
	case FontFamilyAttr:
		s.FontFamily = string(a)
	case WeightAttr:
		s.Weight = draw.FontWeight(a)
	case ColorAttr:
		s.Color = draw.Color(a)
	case BgColorAttr:
		s.BgColor = draw.Color(a)
	case SizeAttr:
		s.Size = float32(a)
	case TrackingAttr:
		s.Tracking = float32(a)
	case LineHeightAttr:
		s.LineHeight = float32(a)
	case WhiteSpaceAttr:
		s.WhiteSpace = WhiteSpace(a)
	case ImageAttr:
		s.Image = ImageAttachment(a)
	case AlignAttr:
		s.Align = draw.TextAlign(a)
	case IndentAttr:
		s.Indent = float32(a)
	case ParaSpacingAttr:
		s.ParaSpacing = float32(a)
	case ListTypeAttr:
		s.ListType = draw.ListType(a)
	case ListLevelAttr:
		s.ListLevel = int(a)
	case ListStartAttr:
		s.ListStart = int(a)
	case ListMarkerAttr:
		s.ListMarker = draw.ListMarker(a)
	}
}

// ── Constructors ───────────────────────────────────────────────

// NewAttributedString creates an AttributedString from plain text
// with default (unstyled) formatting.
func NewAttributedString(text string) AttributedString {
	return AttributedString{Text: text}
}

// Styled creates a single-range AttributedString with the given style.
func Styled(text string, style SpanStyle) AttributedString {
	if text == "" {
		return AttributedString{}
	}
	attrs := spanStyleToAttrs(0, len(text), style)
	return AttributedString{Text: text, Attrs: attrs}
}

// Build constructs an AttributedString from styled segments.
func Build(segments ...Segment) AttributedString {
	if len(segments) == 0 {
		return AttributedString{}
	}
	var buf []byte
	var attrs []Attr
	for _, seg := range segments {
		start := len(buf)
		buf = append(buf, seg.Text...)
		end := len(buf)
		attrs = append(attrs, spanStyleToAttrs(start, end, seg.Style)...)
	}
	return AttributedString{Text: string(buf), Attrs: attrs}
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

// ── Accessors ──────────────────────────────────────────────────

// PlainText returns the plain text content.
func (as AttributedString) PlainText() string { return as.Text }

// Len returns the byte length of the text.
func (as AttributedString) Len() int { return len(as.Text) }

// IsEmpty returns true if the attributed string has no text.
func (as AttributedString) IsEmpty() bool { return len(as.Text) == 0 }

// ── Mutation ───────────────────────────────────────────────────

// Apply adds a single typed attribute to the given byte range.
// This never splits or modifies existing attributes.
func (as AttributedString) Apply(start, end int, attr Attribute) AttributedString {
	if start >= end {
		return as
	}
	newAttrs := make([]Attr, len(as.Attrs), len(as.Attrs)+1)
	copy(newAttrs, as.Attrs)
	newAttrs = append(newAttrs, Attr{Start: start, End: end, Value: attr})
	return AttributedString{Text: as.Text, Attrs: newAttrs}
}

// ApplyStyle sets the style for the byte range [start, end).
// Non-zero fields in style are converted to individual Attrs.
// Returns a new AttributedString.
func (as AttributedString) ApplyStyle(start, end int, style SpanStyle) AttributedString {
	if start >= end {
		return as
	}
	newAttrs := spanStyleToAttrs(start, end, style)
	if len(newAttrs) == 0 {
		return as
	}
	result := make([]Attr, len(as.Attrs), len(as.Attrs)+len(newAttrs))
	copy(result, as.Attrs)
	result = append(result, newAttrs...)
	return AttributedString{Text: as.Text, Attrs: result}
}

// InsertText inserts text at the given byte offset.
// Span-level attrs that contain the offset are extended to cover the insertion.
// Paragraph-level attrs use paragraph-aware boundaries:
//   - Insertion within a paragraph attr's range extends it (or splits it when
//     the inserted text contains newlines).
//   - Insertion exactly at the Start of a paragraph attr extends it rather than
//     shifting, because the insertion is still part of the same paragraph.
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

	ins := len(text)
	newText := as.Text[:offset] + text + as.Text[offset:]

	// Position of the last newline in the inserted text (-1 if none).
	lastNL := strings.LastIndex(text, "\n")

	newAttrs := make([]Attr, 0, len(as.Attrs)+4)
	for _, a := range as.Attrs {
		na := a

		if isParagraphAttr(na.Value) {
			// Paragraph-level attribute: [Start, End) should track paragraph
			// boundaries. The insertion is "within" when Start <= offset <= End.
			if na.End < offset {
				// Entirely before, not touching — keep as-is.
				newAttrs = append(newAttrs, na)
			} else if na.Start > offset {
				// Entirely after — shift.
				na.Start += ins
				na.End += ins
				newAttrs = append(newAttrs, na)
			} else {
				// Start <= offset <= End: insertion is within this paragraph.
				if lastNL >= 0 {
					// Inserted text contains newlines — split the attr so
					// each resulting paragraph has its own range.
					head := na
					head.End = offset + lastNL + 1 // through the last \n
					if head.Start < head.End {
						newAttrs = append(newAttrs, head)
					}
					tail := na
					tail.Start = offset + lastNL + 1
					tail.End = na.End + ins
					if tail.Start < tail.End {
						newAttrs = append(newAttrs, tail)
					}
				} else {
					// No newlines in insertion — simply extend.
					na.End += ins
					newAttrs = append(newAttrs, na)
				}
			}
		} else {
			// Span-level attribute — original logic.
			if na.End <= offset {
				if na.End == offset {
					na.End += ins
				}
			} else if na.Start >= offset {
				na.Start += ins
				na.End += ins
			} else {
				na.End += ins
			}
			newAttrs = append(newAttrs, na)
		}
	}

	return AttributedString{Text: newText, Attrs: newAttrs}
}

// DeleteRange removes bytes [start, end) and adjusts attributes.
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

	if newText == "" {
		return AttributedString{}
	}

	var newAttrs []Attr
	for _, a := range as.Attrs {
		na := a
		if na.End <= start {
			// Entirely before deleted range — keep as-is.
			newAttrs = append(newAttrs, na)
		} else if na.Start >= end {
			// Entirely after deleted range — shift.
			na.Start -= deleted
			na.End -= deleted
			newAttrs = append(newAttrs, na)
		} else {
			// Overlaps with deleted range — clip.
			if na.Start < start {
				// Starts before deletion.
				if na.End <= end {
					na.End = start
				} else {
					na.End -= deleted
				}
			} else {
				// Starts within or at deletion.
				if na.End <= end {
					continue // entirely within deletion — remove
				}
				na.Start = start
				na.End -= deleted
			}
			if na.Start < na.End {
				newAttrs = append(newAttrs, na)
			}
		}
	}

	return AttributedString{Text: newText, Attrs: newAttrs}
}

// InsertImage inserts an image at the given byte offset.
// The image is represented by a U+FFFC placeholder (3 UTF-8 bytes).
func (as AttributedString) InsertImage(offset int, img ImageAttachment) AttributedString {
	const placeholder = "\uFFFC"
	as = as.InsertText(offset, placeholder)
	return as.Apply(offset, offset+len(placeholder), ImageAttr(img))
}

// ToggleStyleFunc transforms the style of each segment overlapping
// [start, end) using fn. Preserves attributes that fn does not modify.
func (as AttributedString) ToggleStyleFunc(start, end int, fn func(SpanStyle) SpanStyle) AttributedString {
	if start < 0 {
		start = 0
	}
	if end > len(as.Text) {
		end = len(as.Text)
	}
	if start >= end {
		return as
	}

	points := as.transitionsIn(start, end)
	result := as
	for i := 0; i < len(points)-1; i++ {
		segStart := points[i]
		segEnd := points[i+1]
		current := as.ResolveAt(segStart)
		desired := fn(current)
		result = result.applyDiff(segStart, segEnd, current, desired)
	}
	return result
}

// applyDiff adds Attrs for fields that differ between current and desired.
func (as AttributedString) applyDiff(start, end int, current, desired SpanStyle) AttributedString {
	result := as
	if desired.Bold != current.Bold {
		result = result.Apply(start, end, BoldAttr(desired.Bold))
	}
	if desired.Italic != current.Italic {
		result = result.Apply(start, end, ItalicAttr(desired.Italic))
	}
	if desired.Underline != current.Underline {
		result = result.Apply(start, end, UnderlineAttr(desired.Underline))
	}
	if desired.Strikethrough != current.Strikethrough {
		result = result.Apply(start, end, StrikethroughAttr(desired.Strikethrough))
	}
	if desired.FontFamily != current.FontFamily {
		result = result.Apply(start, end, FontFamilyAttr(desired.FontFamily))
	}
	if desired.Weight != current.Weight {
		result = result.Apply(start, end, WeightAttr(desired.Weight))
	}
	if desired.Color != current.Color {
		result = result.Apply(start, end, ColorAttr(desired.Color))
	}
	if desired.BgColor != current.BgColor {
		result = result.Apply(start, end, BgColorAttr(desired.BgColor))
	}
	if desired.Size != current.Size {
		result = result.Apply(start, end, SizeAttr(desired.Size))
	}
	if desired.Tracking != current.Tracking {
		result = result.Apply(start, end, TrackingAttr(desired.Tracking))
	}
	if desired.LineHeight != current.LineHeight {
		result = result.Apply(start, end, LineHeightAttr(desired.LineHeight))
	}
	if desired.WhiteSpace != current.WhiteSpace {
		result = result.Apply(start, end, WhiteSpaceAttr(desired.WhiteSpace))
	}
	if desired.Image != current.Image {
		result = result.Apply(start, end, ImageAttr(desired.Image))
	}
	if desired.Align != current.Align {
		result = result.Apply(start, end, AlignAttr(desired.Align))
	}
	if desired.Indent != current.Indent {
		result = result.Apply(start, end, IndentAttr(desired.Indent))
	}
	if desired.ParaSpacing != current.ParaSpacing {
		result = result.Apply(start, end, ParaSpacingAttr(desired.ParaSpacing))
	}
	if desired.ListType != current.ListType {
		result = result.Apply(start, end, ListTypeAttr(desired.ListType))
	}
	if desired.ListLevel != current.ListLevel {
		result = result.Apply(start, end, ListLevelAttr(desired.ListLevel))
	}
	if desired.ListStart != current.ListStart {
		result = result.Apply(start, end, ListStartAttr(desired.ListStart))
	}
	if desired.ListMarker != current.ListMarker {
		result = result.Apply(start, end, ListMarkerAttr(desired.ListMarker))
	}
	return result
}

// ── Queries ────────────────────────────────────────────────────

// AllMatch reports whether fn returns true for every resolved style
// segment overlapping [start, end). Returns true for empty ranges.
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
	for _, run := range as.StyleRuns(start, end) {
		if !fn(run.Style) {
			return false
		}
	}
	return true
}

// ── Utility ────────────────────────────────────────────────────

// Normalized removes empty or redundant attributes.
func (as AttributedString) Normalized() AttributedString {
	var clean []Attr
	for _, a := range as.Attrs {
		if a.Start < a.End {
			clean = append(clean, a)
		}
	}
	if len(clean) == len(as.Attrs) {
		return as
	}
	return AttributedString{Text: as.Text, Attrs: clean}
}

// Equal reports whether two AttributedStrings resolve to the same
// styles at every position and have the same text.
func (as AttributedString) Equal(other AttributedString) bool {
	if as.Text != other.Text {
		return false
	}
	// Compare resolved styles at every transition point.
	allPoints := make(map[int]struct{})
	allPoints[0] = struct{}{}
	allPoints[len(as.Text)] = struct{}{}
	for _, a := range as.Attrs {
		allPoints[a.Start] = struct{}{}
		allPoints[a.End] = struct{}{}
	}
	for _, a := range other.Attrs {
		allPoints[a.Start] = struct{}{}
		allPoints[a.End] = struct{}{}
	}
	for p := range allPoints {
		if p < 0 || p >= len(as.Text) {
			continue
		}
		if as.ResolveAt(p) != other.ResolveAt(p) {
			return false
		}
	}
	return true
}

// spanStyleToAttrs converts a SpanStyle to individual typed Attrs.
// Only non-zero fields produce an Attr.
func spanStyleToAttrs(start, end int, s SpanStyle) []Attr {
	var out []Attr
	if s.Bold {
		out = append(out, Attr{start, end, BoldAttr(true)})
	}
	if s.Italic {
		out = append(out, Attr{start, end, ItalicAttr(true)})
	}
	if s.Underline {
		out = append(out, Attr{start, end, UnderlineAttr(true)})
	}
	if s.Strikethrough {
		out = append(out, Attr{start, end, StrikethroughAttr(true)})
	}
	if s.FontFamily != "" {
		out = append(out, Attr{start, end, FontFamilyAttr(s.FontFamily)})
	}
	if s.Weight > 0 {
		out = append(out, Attr{start, end, WeightAttr(s.Weight)})
	}
	if s.Color.A > 0 {
		out = append(out, Attr{start, end, ColorAttr(s.Color)})
	}
	if s.BgColor.A > 0 {
		out = append(out, Attr{start, end, BgColorAttr(s.BgColor)})
	}
	if s.Size > 0 {
		out = append(out, Attr{start, end, SizeAttr(s.Size)})
	}
	if s.Tracking != 0 {
		out = append(out, Attr{start, end, TrackingAttr(s.Tracking)})
	}
	if s.LineHeight > 0 {
		out = append(out, Attr{start, end, LineHeightAttr(s.LineHeight)})
	}
	if s.WhiteSpace != WhiteSpaceNormal {
		out = append(out, Attr{start, end, WhiteSpaceAttr(s.WhiteSpace)})
	}
	if s.Image.ImageID != 0 {
		out = append(out, Attr{start, end, ImageAttr(s.Image)})
	}
	if s.Align != draw.TextAlignLeft {
		out = append(out, Attr{start, end, AlignAttr(s.Align)})
	}
	if s.Indent != 0 {
		out = append(out, Attr{start, end, IndentAttr(s.Indent)})
	}
	if s.ParaSpacing != 0 {
		out = append(out, Attr{start, end, ParaSpacingAttr(s.ParaSpacing)})
	}
	if s.ListType != draw.ListTypeNone {
		out = append(out, Attr{start, end, ListTypeAttr(s.ListType)})
	}
	if s.ListLevel != 0 {
		out = append(out, Attr{start, end, ListLevelAttr(s.ListLevel)})
	}
	if s.ListStart != 0 {
		out = append(out, Attr{start, end, ListStartAttr(s.ListStart)})
	}
	if s.ListMarker != draw.ListMarkerDefault {
		out = append(out, Attr{start, end, ListMarkerAttr(s.ListMarker)})
	}
	return out
}

// paragraphEndInclusive returns the exclusive end of the paragraph range
// including its trailing \n (if any). Use this for paragraph-level attribute
// ranges so that the \n is covered, which is required for correct attribute
// resolution when scanning across paragraph boundaries.
func paragraphEndInclusive(text string, end int) int {
	if end < len(text) && text[end] == '\n' {
		return end + 1
	}
	return end
}

// ParagraphRange returns [start, end) of the paragraph containing offset.
// Paragraphs are delimited by \n characters.
func ParagraphRange(text string, offset int) (start, end int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	start = strings.LastIndex(text[:offset], "\n")
	if start < 0 {
		start = 0
	} else {
		start++ // skip the \n itself
	}
	end = strings.Index(text[offset:], "\n")
	if end < 0 {
		end = len(text)
	} else {
		end += offset
	}
	return
}

// ── Legacy Compatibility ───────────────────────────────────────

// AttrRun is the legacy type alias for backward compatibility.
// Deprecated: use Attr instead.
type AttrRun = Attr

// DocumentChangedMsg is sent when the user edits the document.
type DocumentChangedMsg struct {
	Value AttributedString
}
