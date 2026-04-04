package uitest

import (
	"testing"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/link"
	"github.com/timzifer/lux/ui/nav"
)

const (
	testW = 800
	testH = 600
)

// ── Text ──────────────────────────────────────────────────────────

func TestGoldenText(t *testing.T) {
	scene := BuildScene(display.Text("Hello World"), testW, testH)
	AssertScene(t, scene, "testdata/text.golden")
}

func TestGoldenTextEmpty(t *testing.T) {
	scene := BuildScene(display.Text(""), testW, testH)
	AssertScene(t, scene, "testdata/text_empty.golden")
}

// ── Row / Column ──────────────────────────────────────────────────

func TestGoldenColumn(t *testing.T) {
	scene := BuildScene(
		layout.Column(
			display.Text("First"),
			display.Text("Second"),
			display.Text("Third"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/column.golden")
}

func TestGoldenRow(t *testing.T) {
	scene := BuildScene(
		layout.Row(
			display.Text("Left"),
			display.Text("Right"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/row.golden")
}

// ── Stack ─────────────────────────────────────────────────────────

func TestGoldenStack(t *testing.T) {
	scene := BuildScene(
		layout.NewStack(
			display.Text("Bottom"),
			display.Text("Top"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/stack.golden")
}

// ── Padding ───────────────────────────────────────────────────────

func TestGoldenPadding(t *testing.T) {
	scene := BuildScene(
		layout.Pad(
			draw.Insets{Top: 20, Right: 30, Bottom: 20, Left: 30},
			display.Text("Padded"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/padding.golden")
}

// ── Card ──────────────────────────────────────────────────────────

func TestGoldenCard(t *testing.T) {
	scene := BuildScene(
		display.Card(
			display.Text("Card Title"),
			display.Text("Card body text"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/card.golden")
}

// ── Flex Layouts ──────────────────────────────────────────────────

func TestGoldenFlexRow(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("A"),
				display.Text("B"),
				display.Text("C"),
			},
			layout.WithDirection(layout.FlexRow),
			layout.WithGap(10),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/flex_row.golden")
}

func TestGoldenFlexColumnCenter(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("Top"),
				display.Text("Bottom"),
			},
			layout.WithDirection(layout.FlexColumn),
			layout.WithJustify(layout.JustifyCenter),
			layout.WithAlign(layout.AlignCenter),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/flex_column_center.golden")
}

// ── Nested Layout ─────────────────────────────────────────────────

func TestGoldenNestedLayout(t *testing.T) {
	scene := BuildScene(
		layout.Column(
			display.Text("Header"),
			layout.Row(
				display.Text("Left"),
				display.Text("Right"),
			),
			display.Text("Footer"),
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/nested_layout.golden")
}

// ── Form Components: Pickers & Numeric ───────────────────────────

func TestGoldenSpinner(t *testing.T) {
	scene := BuildScene(form.NewSpinner(0.0), testW, testH)
	AssertScene(t, scene, "testdata/spinner.golden")
}

func TestGoldenSpinnerMidPhase(t *testing.T) {
	scene := BuildScene(form.NewSpinner(0.5), testW, testH)
	AssertScene(t, scene, "testdata/spinner_mid.golden")
}

func TestGoldenNumericInput(t *testing.T) {
	scene := BuildScene(
		form.NewNumericInput(42, form.WithUnit("px")),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/numeric_input.golden")
}

func TestGoldenNumericInputDisabled(t *testing.T) {
	scene := BuildScene(
		form.NumericInputDisabled(10, form.WithUnit("em")),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/numeric_input_disabled.golden")
}

func TestGoldenColorPicker(t *testing.T) {
	scene := BuildScene(
		form.NewColorPicker(draw.Color{R: 0.25, G: 0.32, B: 0.71, A: 1}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/color_picker.golden")
}

func TestGoldenTimePicker(t *testing.T) {
	scene := BuildScene(
		form.NewTimePicker(14, 30),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/time_picker.golden")
}

func TestGoldenDatePicker(t *testing.T) {
	scene := BuildScene(
		form.NewDatePicker(time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/date_picker.golden")
}

// ── Inline Widgets (RFC-003 §5.5) ───────────────────────────────

func TestGoldenRichTextInlineWidget(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Status: "},
				display.InlineElement(display.BadgeText("OK")),
				display.Span{Text: " — all systems go."},
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_inline_widget.golden")
}

func TestGoldenRichTextInlineWidgetWrap(t *testing.T) {
	// Use a narrow width to force the inline widget to wrap.
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This is a long line that fills the width "},
				display.InlineElement(display.BadgeText("WRAPPED")),
				display.Span{Text: " after the badge."},
			},
		}),
		300, testH,
	)
	AssertScene(t, scene, "testdata/richtext_inline_widget_wrap.golden")
}

func TestGoldenRichTextInlineWidgetBaseline(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Shifted up: "},
				display.InlineElementWithBaseline(display.BadgeText("UP"), 4),
				display.Span{Text: " and default: "},
				display.InlineElement(display.BadgeText("DEF")),
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_inline_widget_baseline.golden")
}

func TestGoldenRichTextMixedContent(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Hello "},
					display.InlineElement(display.BadgeText("1")),
					display.Span{Text: " world "},
					display.InlineElement(display.BadgeText("2")),
					display.Span{Text: " end."},
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Second paragraph with "},
					display.InlineElement(
						display.BadgeColor(display.Text("color"), draw.Hex("#ef4444")),
					),
					display.Span{Text: " badge."},
				},
			},
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_mixed_content.golden")
}

// ── RichText Block Widgets ──────────────────────────────────────

func TestGoldenRichTextBlockWidget(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Before block."},
				display.BlockElement(display.BadgeText("BLOCK")),
				display.Span{Text: "After block."},
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_block_widget.golden")
}

func TestGoldenRichTextBlockWidgetMixed(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Inline "},
				display.InlineElement(display.BadgeText("TAG")),
				display.Span{Text: " text. "},
				display.BlockElement(
					display.BadgeColor(display.Text("Full-Width Block"), draw.Hex("#3b82f6")),
				),
				display.Span{Text: "More inline."},
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_block_widget_mixed.golden")
}

// ── RichText Images ─────────────────────────────────────────────

func TestGoldenRichTextInlineImage(t *testing.T) {
	// Use a fixed ImageID so the golden output is deterministic.
	const imgID = draw.ImageID(1)
	scene := BuildScene(
		display.RichTextContent(
			display.Span{Text: "Icon: "},
			display.InlineImage(imgID, display.WithImageSpanSize(20, 20), display.WithImageSpanAlt("icon")),
			display.Span{Text: " — end."},
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_image_inline.golden")
}

func TestGoldenRichTextFloatLeftImage(t *testing.T) {
	const imgID = draw.ImageID(2)
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.FloatLeftImage(imgID, display.WithImageSpanSize(48, 48), display.WithImageSpanAlt("float")),
				display.Span{Text: "Text wraps on the right side of the float-left image."},
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_image_float_left.golden")
}

func TestGoldenRichTextBlockImage(t *testing.T) {
	const imgID = draw.ImageID(3)
	// Block images are rendered after all inline content in their paragraph.
	// Use separate paragraphs for above/below captions around a block image.
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Caption above."},
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.BlockImage(imgID,
						display.WithImageSpanSize(0, 40),
						display.WithImageSpanScaleMode(draw.ImageScaleFit),
						display.WithImageSpanAlt("block"),
					),
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Caption below."},
				},
			},
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_image_block.golden")
}

func TestGoldenRichTextSpansBackcompat(t *testing.T) {
	// Ensure the old Spans-only API still works unchanged.
	scene := BuildScene(
		display.RichTextSpans(
			display.Span{Text: "Bold ", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
			display.Span{Text: "normal"},
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_spans_backcompat.golden")
}

// ── Paragraph Styling ───────────────────────────────────────────

func TestGoldenRichTextAlignCenter(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This text is centered within the paragraph."},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignCenter},
		}),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_align_center.golden")
}

func TestGoldenRichTextAlignRight(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This text is right-aligned."},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignRight},
		}),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_align_right.golden")
}

func TestGoldenRichTextAlignJustify(t *testing.T) {
	// Narrow width to force wrapping so justify has visible effect.
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This is a justified paragraph that should wrap across multiple lines to demonstrate the text-align justify effect."},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignJustify},
		}),
		250, testH,
	)
	AssertScene(t, scene, "testdata/richtext_align_justify.golden")
}

func TestGoldenRichTextIndent(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This paragraph has a first-line indent of 24dp, similar to traditional typographic indentation."},
			},
			Style: display.ParagraphStyle{Indent: 24},
		}),
		300, testH,
	)
	AssertScene(t, scene, "testdata/richtext_indent.golden")
}

func TestGoldenRichTextParagraphSpacing(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "First paragraph with large spacing after."},
				},
				Style: display.ParagraphStyle{SpaceAfter: 24},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Second paragraph with default spacing."},
				},
			},
		),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_paragraph_spacing.golden")
}

func TestGoldenRichTextLineHeight(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This paragraph uses 2x line height for increased readability and spacing between lines."},
			},
			Style: display.ParagraphStyle{LineHeight: 2.0},
		}),
		300, testH,
	)
	AssertScene(t, scene, "testdata/richtext_line_height.golden")
}

// ── Toolbar ──────────────────────────────────────────────────────

func TestGoldenToolbar(t *testing.T) {
	scene := BuildScene(
		nav.NewToolbar([]nav.ToolbarItem{
			{Element: display.Text("Cut"), OnClick: func() {}},
			{Element: display.Text("Copy"), OnClick: func() {}},
			{Element: display.Text("Paste"), OnClick: func() {}},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/toolbar.golden")
}

func TestGoldenToolbarSeparator(t *testing.T) {
	scene := BuildScene(
		nav.NewToolbar([]nav.ToolbarItem{
			{Element: display.Text("New"), OnClick: func() {}},
			{Element: display.Text("Open"), OnClick: func() {}},
			nav.ToolbarSeparator(),
			{Element: display.Text("Undo"), OnClick: func() {}},
			{Element: display.Text("Redo"), OnClick: func() {}},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/toolbar_separator.golden")
}

func TestGoldenToolbarToggle(t *testing.T) {
	scene := BuildScene(
		nav.NewToolbar([]nav.ToolbarItem{
			{Element: display.Text("B"), OnClick: func() {}, Toggle: true, Active: true},
			{Element: display.Text("I"), OnClick: func() {}, Toggle: true, Active: false},
			{Element: display.Text("U"), OnClick: func() {}, Toggle: true, Active: true},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/toolbar_toggle.golden")
}

// ── RichText Lists ──────────────────────────────────────────────

func TestGoldenRichTextListUnordered(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Apples"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Bananas"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Cherries"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_unordered.golden")
}

func TestGoldenRichTextListOrdered(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "First"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Second"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Third"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_ordered.golden")
}

func TestGoldenRichTextListNested(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Top level"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Nested level 1"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Nested level 2"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListLevel: 2},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Back to top"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_nested.golden")
}

func TestGoldenRichTextListStartNumber(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Fifth"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Sixth"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Seventh"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_start_number.golden")
}

func TestGoldenRichTextListStyledContent(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Normal and "},
					display.Span{Text: "bold", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold}}},
				},
				Style: display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "With "},
					display.Span{Text: "color", Style: display.SpanStyle{Color: draw.Hex("#ef4444")}},
				},
				Style: display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_styled.golden")
}

func TestGoldenRichTextListMixed(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Bullet one"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Numbered sub"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Numbered sub"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Bullet two"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_mixed.golden")
}

func TestGoldenRichTextListOrderedNested(t *testing.T) {
	scene := BuildScene(
		display.RichText(
			// Level 0: decimal (1. 2.)
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "First chapter"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			// Level 1: lower-alpha (a. b.)
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Section alpha"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Section beta"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			// Level 2: lower-roman (i. ii.)
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Sub-item one"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 2},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Sub-item two"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 2},
			},
			// Back to level 0
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Second chapter"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
		),
		400, testH,
	)
	AssertScene(t, scene, "testdata/richtext_list_ordered_nested.golden")
}

func TestGoldenRichTextInlineLinkBaseline(t *testing.T) {
	scene := BuildScene(
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Click "},
				display.InlineElement(link.Text("here", func() {})),
				display.Span{Text: " to continue."},
			},
		}),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/richtext_inline_link_baseline.golden")
}

// ── HMI Widgets (RFC-004 §6) ───────────────────────────────────

func TestGoldenStepper(t *testing.T) {
	scene := BuildScene(
		form.NewStepper(42, form.WithStepperRange(0, 100)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/stepper.golden")
}

func TestGoldenStepperVertical(t *testing.T) {
	scene := BuildScene(
		form.NewStepper(10, form.WithStepperRange(0, 50), form.WithStepperOrientation(form.Vertical)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/stepper_vertical.golden")
}

func TestGoldenDrumPicker(t *testing.T) {
	items := form.IntItems(0, 23)
	scene := BuildScene(
		form.NewDrumPicker(items, 10),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/drum_picker.golden")
}

func TestGoldenPinInput(t *testing.T) {
	scene := BuildScene(
		form.NewPinInput(4, "12"),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/pin_input.golden")
}

func TestGoldenPinInputMasked(t *testing.T) {
	scene := BuildScene(
		form.NewPinInput(4, "123", form.WithPinMasked()),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/pin_input_masked.golden")
}

func TestGoldenHexInput(t *testing.T) {
	scene := BuildScene(
		form.NewHexInput(0x00FF),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/hex_input.golden")
}

func TestGoldenIPInput(t *testing.T) {
	scene := BuildScene(
		form.NewIPInput("192.168.1.1"),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/ip_input.golden")
}

func TestGoldenUnitInput(t *testing.T) {
	units := []form.UnitDef{
		{Symbol: "mm", Label: "Millimeter", Factor: 1},
		{Symbol: "cm", Label: "Centimeter", Factor: 10},
		{Symbol: "m", Label: "Meter", Factor: 1000},
	}
	scene := BuildScene(
		form.NewUnitInput(25.0, "mm", units),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/unit_input.golden")
}

func TestGoldenRangeInput(t *testing.T) {
	scene := BuildScene(
		form.NewRangeInput(20, 80, 0, 100, form.WithRangeLabels()),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/range_input.golden")
}

func TestGoldenTimeInput(t *testing.T) {
	scene := BuildScene(
		form.NewTimeInput(time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/time_input.golden")
}

func TestGoldenDateInputDrum(t *testing.T) {
	scene := BuildScene(
		form.NewDateInput(time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)),
		testW, testH,
	)
	AssertScene(t, scene, "testdata/date_input_drum.golden")
}
