package richtext

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/icons"
)

// ToolbarCommand defines a pluggable toolbar action for the
// RichTextEditor. Implementations control how the command appears in
// the toolbar, when it is considered active, and what happens when
// the user clicks it.
type ToolbarCommand interface {
	// Icon returns the element displayed in the toolbar button.
	Icon() ui.Element

	// IsActive reports whether this command is "on" for the current
	// selection. When selStart == selEnd (no selection) the style at
	// the cursor position is checked.
	IsActive(doc AttributedString, selStart, selEnd int) bool

	// Execute applies the command. When there is a selection
	// (selStart != selEnd) it modifies the document and returns it.
	// When there is no selection it returns a pendingMod function
	// that will be applied to future typed text instead.
	Execute(doc AttributedString, selStart, selEnd int) (newDoc AttributedString, pendingMod func(SpanStyle) SpanStyle)
}

// DefaultCommands returns the standard formatting commands:
// Bold, Italic, Underline and Strikethrough.
func DefaultCommands() []ToolbarCommand {
	return []ToolbarCommand{
		BoldCommand{},
		ItalicCommand{},
		UnderlineCommand{},
		StrikethroughCommand{},
	}
}

// AlignmentCommands returns toolbar commands for paragraph alignment:
// Left, Center, Right, and Justify.
func AlignmentCommands() []ToolbarCommand {
	return []ToolbarCommand{
		AlignCommand{draw.TextAlignLeft},
		AlignCommand{draw.TextAlignCenter},
		AlignCommand{draw.TextAlignRight},
		AlignCommand{draw.TextAlignJustify},
	}
}

// ── Bold ────────────────────────────────────────────────────────

// BoldCommand toggles bold formatting.
type BoldCommand struct{}

func (BoldCommand) Icon() ui.Element {
	return display.Icon(icons.TextBolder)
}

func (BoldCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	if selStart == selEnd {
		return doc.RunAt(selStart).Bold
	}
	return doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Bold })
}

func (BoldCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	if selStart == selEnd {
		wasBold := doc.RunAt(selStart).Bold
		return doc, func(s SpanStyle) SpanStyle { s.Bold = !wasBold; return s }
	}
	allBold := doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Bold })
	return doc.ToggleStyleFunc(selStart, selEnd, func(s SpanStyle) SpanStyle {
		s.Bold = !allBold
		return s
	}), nil
}

// ── Italic ──────────────────────────────────────────────────────

// ItalicCommand toggles italic formatting.
type ItalicCommand struct{}

func (ItalicCommand) Icon() ui.Element {
	return display.Icon(icons.TextItalic)
}

func (ItalicCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	if selStart == selEnd {
		return doc.RunAt(selStart).Italic
	}
	return doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Italic })
}

func (ItalicCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	if selStart == selEnd {
		wasItalic := doc.RunAt(selStart).Italic
		return doc, func(s SpanStyle) SpanStyle { s.Italic = !wasItalic; return s }
	}
	allItalic := doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Italic })
	return doc.ToggleStyleFunc(selStart, selEnd, func(s SpanStyle) SpanStyle {
		s.Italic = !allItalic
		return s
	}), nil
}

// ── Underline ───────────────────────────────────────────────────

// UnderlineCommand toggles underline formatting.
type UnderlineCommand struct{}

func (UnderlineCommand) Icon() ui.Element {
	return display.Icon(icons.TextUnderline)
}

func (UnderlineCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	if selStart == selEnd {
		return doc.RunAt(selStart).Underline
	}
	return doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Underline })
}

func (UnderlineCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	if selStart == selEnd {
		wasUnderline := doc.RunAt(selStart).Underline
		return doc, func(s SpanStyle) SpanStyle { s.Underline = !wasUnderline; return s }
	}
	allUnderline := doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Underline })
	return doc.ToggleStyleFunc(selStart, selEnd, func(s SpanStyle) SpanStyle {
		s.Underline = !allUnderline
		return s
	}), nil
}

// ── Strikethrough ──────────────────────────────────────────────

// StrikethroughCommand toggles strikethrough formatting.
type StrikethroughCommand struct{}

func (StrikethroughCommand) Icon() ui.Element {
	return display.Icon(icons.TextStrikethrough)
}

func (StrikethroughCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	if selStart == selEnd {
		return doc.RunAt(selStart).Strikethrough
	}
	return doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Strikethrough })
}

func (StrikethroughCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	if selStart == selEnd {
		wasStrike := doc.RunAt(selStart).Strikethrough
		return doc, func(s SpanStyle) SpanStyle { s.Strikethrough = !wasStrike; return s }
	}
	allStrike := doc.AllMatch(selStart, selEnd, func(s SpanStyle) bool { return s.Strikethrough })
	return doc.ToggleStyleFunc(selStart, selEnd, func(s SpanStyle) SpanStyle {
		s.Strikethrough = !allStrike
		return s
	}), nil
}

// ── Align ──────────────────────────────────────────────────────

// AlignCommand sets paragraph alignment. It operates on the full
// paragraph containing the cursor, not just the selection.
type AlignCommand struct {
	Alignment draw.TextAlign
}

func (c AlignCommand) Icon() ui.Element {
	switch c.Alignment {
	case draw.TextAlignCenter:
		return display.Icon(icons.TextAlignCenter)
	case draw.TextAlignRight:
		return display.Icon(icons.TextAlignRight)
	case draw.TextAlignJustify:
		return display.Icon(icons.TextAlignJustify)
	default:
		return display.Icon(icons.TextAlignLeft)
	}
}

func (c AlignCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	return doc.ResolveAt(selStart).Align == c.Alignment
}

func (c AlignCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	start, end := ParagraphRange(doc.Text, selStart)
	end = paragraphEndInclusive(doc.Text, end)
	if end <= start {
		end = start + 1
		if end > len(doc.Text) {
			end = len(doc.Text)
		}
	}
	return doc.Apply(start, end, AlignAttr(c.Alignment)), nil
}

// ── List ──────────────────────────────────────────────────────

// ListCommand toggles list type (ordered/unordered) for the paragraph
// containing the cursor. Follows the same paragraph-scope pattern as
// AlignCommand.
type ListCommand struct {
	Type draw.ListType // ListTypeUnordered or ListTypeOrdered
}

func (c ListCommand) Icon() ui.Element {
	if c.Type == draw.ListTypeOrdered {
		return display.Icon(icons.ListNumbers)
	}
	return display.Icon(icons.ListBullets)
}

func (c ListCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	return doc.ResolveAt(selStart).ListType == c.Type
}

func (c ListCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	start, end := ParagraphRange(doc.Text, selStart)
	end = paragraphEndInclusive(doc.Text, end)
	if end <= start {
		end = start + 1
		if end > len(doc.Text) {
			end = len(doc.Text)
		}
	}
	current := doc.ResolveAt(selStart).ListType
	if current == c.Type {
		// Toggle off — remove list formatting.
		return doc.Apply(start, end, ListTypeAttr(draw.ListTypeNone)), nil
	}
	return doc.Apply(start, end, ListTypeAttr(c.Type)), nil
}

// ── List Indent ──────────────────────────────────────────────

// IndentListCommand changes the nesting level of the list item at the
// cursor position. Delta is typically +1 (indent) or -1 (outdent).
type IndentListCommand struct {
	Delta int // +1 = indent, -1 = outdent
}

func (c IndentListCommand) Icon() ui.Element {
	if c.Delta > 0 {
		return display.Icon(icons.TextIndent)
	}
	return display.Icon(icons.TextOutdent)
}

func (c IndentListCommand) IsActive(doc AttributedString, selStart, selEnd int) bool {
	return doc.ResolveAt(selStart).ListType != draw.ListTypeNone
}

func (c IndentListCommand) Execute(doc AttributedString, selStart, selEnd int) (AttributedString, func(SpanStyle) SpanStyle) {
	start, end := ParagraphRange(doc.Text, selStart)
	end = paragraphEndInclusive(doc.Text, end)
	if end <= start {
		end = start + 1
		if end > len(doc.Text) {
			end = len(doc.Text)
		}
	}
	current := doc.ResolveAt(selStart).ListLevel
	newLevel := current + c.Delta
	if newLevel < 0 {
		newLevel = 0
	}
	if newLevel > 8 {
		newLevel = 8 // reasonable max nesting depth
	}
	return doc.Apply(start, end, ListLevelAttr(newLevel)), nil
}

// ListCommands returns toolbar commands for list formatting:
// Unordered List, Ordered List, Indent, and Outdent.
func ListCommands() []ToolbarCommand {
	return []ToolbarCommand{
		ListCommand{Type: draw.ListTypeUnordered},
		ListCommand{Type: draw.ListTypeOrdered},
		IndentListCommand{Delta: 1},
		IndentListCommand{Delta: -1},
	}
}
