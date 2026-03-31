// Text Playground — interactive RichText demo with collapsible editor.
//
// Split-view layout: Section list on the left, demo or editor on the right.
// Each section can be loaded into a RichTextEditorWidget via the "Restore" button.
//
//	go run ./examples/text-playground/
package main

import (
	"fmt"
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	luximage "github.com/timzifer/lux/image"
	"github.com/timzifer/lux/richtext"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/data"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"

	"github.com/timzifer/lux/internal/text"
)

// ── Section Registry ──────────────────────────────────────────────

var sectionIDs = []string{
	"rich-text",
	"rich-text-editor",
	"font-formatting",
	"paragraph-styling",
	"inline-widgets",
	"rich-text-images",
	"lists",
	"text-shaping",
	"grapheme-nav",
	"line-breaking",
}

func sectionLabel(id string) string {
	switch id {
	case "rich-text":
		return "RichText"
	case "rich-text-editor":
		return "RichTextEditor"
	case "font-formatting":
		return "Font Formatting"
	case "paragraph-styling":
		return "Paragraph Styling"
	case "inline-widgets":
		return "Inline Widgets"
	case "rich-text-images":
		return "RichText Images"
	case "lists":
		return "Lists"
	case "text-shaping":
		return "Text Shaping"
	case "grapheme-nav":
		return "Grapheme Navigation"
	case "line-breaking":
		return "Line Breaking"
	default:
		return id
	}
}

// ── Model ─────────────────────────────────────────────────────────

type Model struct {
	Dark          bool
	NavTree       *ui.TreeState
	ActiveSection string
	Scroll        *ui.ScrollState
	NavSplitRatio float32
	// Collapsible right panel
	ShowEditor   bool
	EditorDoc    richtext.AttributedString
	EditorScroll *ui.ScrollState
	// RichTextEditor section state
	RichEditorDoc      richtext.AttributedString
	RichEditorScroll   *ui.ScrollState
	RichEditorReadOnly bool
	RichEditorDoc2     richtext.AttributedString
	RichEditorScroll2  *ui.ScrollState
	// Lists section state
	ListEditorDoc    richtext.AttributedString
	ListEditorScroll *ui.ScrollState
	// Images
	ImageStore  *luximage.Store
	ImgChecker1 draw.ImageID
	ImgChecker2 draw.ImageID
	ImgChecker3 draw.ImageID
}

// ── Messages ──────────────────────────────────────────────────────

type SelectSectionMsg struct{ Section string }
type ToggleThemeMsg struct{}
type SetNavSplitMsg struct{ Ratio float32 }
type SetRichEditorDocMsg struct{ Doc richtext.AttributedString }
type ToggleRichEditorReadOnlyMsg struct{}
type SetRichEditorDoc2Msg struct{ Doc richtext.AttributedString }
type SetListEditorDocMsg struct{ Doc richtext.AttributedString }
type SetEditorDocMsg struct{ Doc richtext.AttributedString }
type RestoreToEditorMsg struct{ Doc richtext.AttributedString }
type ShowDemoMsg struct{}

// ── Update ────────────────────────────────────────────────────────

func update(m Model, msg app.Msg) (Model, app.Cmd) {
	switch msg := msg.(type) {
	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	case SelectSectionMsg:
		m.ActiveSection = msg.Section
		m.ShowEditor = false
		m.Scroll = &ui.ScrollState{}
	case SetNavSplitMsg:
		m.NavSplitRatio = msg.Ratio
	case SetRichEditorDocMsg:
		m.RichEditorDoc = msg.Doc
	case ToggleRichEditorReadOnlyMsg:
		m.RichEditorReadOnly = !m.RichEditorReadOnly
	case SetRichEditorDoc2Msg:
		m.RichEditorDoc2 = msg.Doc
	case SetListEditorDocMsg:
		m.ListEditorDoc = msg.Doc
	case SetEditorDocMsg:
		m.EditorDoc = msg.Doc
	case RestoreToEditorMsg:
		m.EditorDoc = msg.Doc
		m.EditorScroll = &ui.ScrollState{}
		m.ShowEditor = true
	case ShowDemoMsg:
		m.ShowEditor = false
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────

func view(m Model) ui.Element {
	themeLabel := "Light"
	if !m.Dark {
		themeLabel = "Dark"
	}

	navTree := data.NewTree(ui.TreeConfig{
		RootIDs:  sectionIDs,
		Children: func(string) []string { return nil },
		BuildNode: func(id string, _ int, _, selected bool) ui.Element {
			label := sectionLabel(id)
			if selected {
				return display.TextStyled(label, draw.TextStyle{
					Size:   13,
					Weight: draw.FontWeightSemiBold,
				})
			}
			return display.Text(label)
		},
		NodeHeight: 28,
		MaxHeight:  0,
		State:      m.NavTree,
		OnSelect:   func(id string) { app.Send(SelectSectionMsg{id}) },
	})

	var rightPanel ui.Element
	if m.ShowEditor {
		rightPanel = editorPanel(m)
	} else {
		rightPanel = demoPanel(m)
	}

	content := nav.NewScrollView(rightPanel, 0, m.Scroll)

	return layout.Pad(ui.UniformInsets(16), layout.NewFlex(
		[]ui.Element{
			layout.Expand(nav.NewSplitView(
				navTree,
				content,
				m.NavSplitRatio,
				func(r float32) { app.Send(SetNavSplitMsg{r}) },
			)),
			display.Spacer(12),
			layout.Row(
				button.Text(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
			),
		},
		layout.WithDirection(layout.FlexColumn),
	))
}

// demoPanel shows the original section content with inline edit buttons.
func demoPanel(m Model) ui.Element {
	return sectionContent(m)
}

// editorPanel shows the RichTextEditorWidget with the loaded document.
func editorPanel(m Model) ui.Element {
	cmds := append(richtext.DefaultCommands(), richtext.AlignmentCommands()...)
	cmds = append(cmds, richtext.ListCommands()...)
	return layout.Column(
		sectionHeader(sectionLabel(m.ActiveSection)+" — Editor"),
		richtext.NewEditorWithToolbar(m.EditorDoc,
			richtext.WithWidgetOnChange(func(as richtext.AttributedString) { app.Send(SetEditorDocMsg{as}) }),
			richtext.WithWidgetFocus(app.Focus()),
			richtext.WithWidgetScroll(m.EditorScroll),
			richtext.WithWidgetRows(12),
			richtext.WithWidgetCommands(cmds),
		),
		display.Spacer(16),
		layout.Row(
			button.Text("◀ Demo-Ansicht", func() { app.Send(ShowDemoMsg{}) }),
		),
	)
}


// ── Section Content Switch ────────────────────────────────────────

func sectionContent(m Model) ui.Element {
	switch m.ActiveSection {
	case "rich-text":
		return richTextSection()
	case "rich-text-editor":
		return richTextEditorSection(m)
	case "font-formatting":
		return fontFormattingSection()
	case "paragraph-styling":
		return paragraphStylingSection()
	case "inline-widgets":
		return inlineWidgetsSection()
	case "rich-text-images":
		return richTextImagesSection(m)
	case "lists":
		return listsSection(m)
	case "text-shaping":
		return textShapingSection()
	case "grapheme-nav":
		return graphemeNavSection()
	case "line-breaking":
		return lineBreakingSection()
	default:
		return layout.Column(
			display.Spacer(24),
			display.Text("Select a section from the list on the left."),
		)
	}
}

// ── Helpers ───────────────────────────────────────────────────────

func sectionHeader(title string) ui.Element {
	return layout.Column(
		display.Spacer(8),
		display.TextStyled(title, draw.TextStyle{
			Size:   16,
			Weight: draw.FontWeightSemiBold,
		}),
		display.Spacer(4),
	)
}

// editBtn returns a small button that loads the given document into the editor.
func editBtn(doc richtext.AttributedString) ui.Element {
	return button.Text("Edit ▶", func() { app.Send(RestoreToEditorMsg{doc}) })
}

func generateColorChecker(store *luximage.Store, w, h, cellSize int, r1, g1, b1, r2, g2, b2 byte) draw.ImageID {
	rgba := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if ((x/cellSize)+(y/cellSize))%2 == 0 {
				rgba[off], rgba[off+1], rgba[off+2], rgba[off+3] = r1, g1, b1, 255
			} else {
				rgba[off], rgba[off+1], rgba[off+2], rgba[off+3] = r2, g2, b2, 255
			}
		}
	}
	id, err := store.LoadFromRGBA(w, h, rgba)
	if err != nil {
		log.Printf("generateColorChecker: %v", err)
		return 0
	}
	return id
}

// ── Section Views ─────────────────────────────────────────────────

func richTextSection() ui.Element {
	mixedDoc := richtext.Build(
		richtext.S("Bold text ", richtext.SpanStyle{Bold: true}),
		richtext.S("and normal text "),
		richtext.S("with color", richtext.SpanStyle{Color: draw.Hex("#3b82f6")}),
	)
	multiDoc := richtext.Build(
		richtext.S("First paragraph with "),
		richtext.S("bold", richtext.SpanStyle{Bold: true}),
		richtext.S(" and "),
		richtext.S("colored", richtext.SpanStyle{Color: draw.Hex("#ef4444")}),
		richtext.S(" spans.\nSecond paragraph. Rich text supports per-span styling."),
	)

	return layout.Column(
		sectionHeader("RichText"),
		display.Text("Mixed styles in a single line:"),
		display.Spacer(4),
		display.RichTextSpans(
			display.Span{Text: "Bold text ", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
			display.Span{Text: "and normal text "},
			display.Span{Text: "with color", Style: display.SpanStyle{Color: draw.Hex("#3b82f6")}},
		),
		display.Spacer(4),
		editBtn(mixedDoc),
		display.Spacer(12),
		display.Text("Multiple paragraphs:"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{Spans: []display.Span{
				{Text: "First paragraph with "},
				{Text: "bold", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
				{Text: " and "},
				{Text: "colored", Style: display.SpanStyle{Color: draw.Hex("#ef4444")}},
				{Text: " spans."},
			}},
			display.RichParagraph{Spans: []display.Span{
				{Text: "Second paragraph. Rich text supports per-span styling."},
			}},
		),
		display.Spacer(4),
		editBtn(multiDoc),
	)
}

func richTextEditorSection(m Model) ui.Element {
	styledDoc := richtext.Build(
		richtext.S("Bold text ", richtext.SpanStyle{Bold: true}),
		richtext.S("then italic ", richtext.SpanStyle{Italic: true}),
		richtext.S("then colored", richtext.SpanStyle{Color: draw.Hex("#ef4444")}),
		richtext.S(" and "),
		richtext.S("combined", richtext.SpanStyle{Bold: true, Italic: true, Color: draw.Hex("#8b5cf6")}),
		richtext.S(".\nSecond paragraph with "),
		richtext.S("blue text", richtext.SpanStyle{Color: draw.Hex("#3b82f6")}),
		richtext.S(" for variety."),
	)
	plainDoc := richtext.NewAttributedString("Line 1: The quick brown fox\nLine 2: jumps over\nLine 3: the lazy dog")

	return layout.Column(
		sectionHeader("RichTextEditor (RFC-003 §5.6)"),

		// ── Basic editable editor ──────────────────────────────
		display.Text("Editable rich text editor:"),
		display.Spacer(4),
		richtext.New(m.RichEditorDoc,
			richtext.WithOnChange(func(as richtext.AttributedString) { app.Send(SetRichEditorDocMsg{as}) }),
			richtext.WithFocus(app.Focus()),
			richtext.WithRows(5),
			richtext.WithScroll(m.RichEditorScroll),
			richtext.WithPlaceholder("Start typing..."),
		),
		display.Spacer(4),
		layout.Row(
			display.Text(fmt.Sprintf("Runs: %d | Plain text length: %d",
				len(m.RichEditorDoc.Attrs), m.RichEditorDoc.Len())),
		),

		display.Spacer(16),

		// ── Read-only mode toggle ──────────────────────────────
		display.Text("Read-only mode:"),
		display.Spacer(4),
		layout.Row(
			button.Text(func() string {
				if m.RichEditorReadOnly {
					return "Make Editable"
				}
				return "Make Read-Only"
			}(), func() { app.Send(ToggleRichEditorReadOnlyMsg{}) }),
		),
		display.Spacer(4),
		func() ui.Element {
			opts := []richtext.Option{
				richtext.WithRows(3),
				richtext.WithScroll(m.RichEditorScroll2),
			}
			if m.RichEditorReadOnly {
				opts = append(opts, richtext.WithReadOnly())
			} else {
				opts = append(opts, richtext.WithOnChange(func(as richtext.AttributedString) { app.Send(SetRichEditorDoc2Msg{as}) }))
				opts = append(opts, richtext.WithFocus(app.Focus()))
			}
			return richtext.New(m.RichEditorDoc2, opts...)
		}(),

		display.Spacer(16),

		// ── Styled content demo ────────────────────────────────
		display.Text("Pre-styled document (bold, italic, color):"),
		display.Spacer(4),
		richtext.New(styledDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(3),
		),
		display.Spacer(4),
		editBtn(styledDoc),

		display.Spacer(16),

		// ── Empty editor with placeholder ──────────────────────
		display.Text("Empty editor with placeholder:"),
		display.Spacer(4),
		richtext.New(richtext.NewAttributedString(""),
			richtext.WithPlaceholder("Enter your thoughts here..."),
			richtext.WithRows(2),
			richtext.WithOnChange(func(richtext.AttributedString) {}),
			richtext.WithFocus(app.Focus()),
		),

		display.Spacer(16),

		// ── Document from plain text ───────────────────────────
		display.Text("Document created from plain text:"),
		display.Spacer(4),
		richtext.New(plainDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(3),
		),
		display.Spacer(4),
		editBtn(plainDoc),
	)
}

func fontFormattingSection() ui.Element {
	strikeDoc := richtext.Build(
		richtext.S("Normal text, "),
		richtext.S("strikethrough text", richtext.SpanStyle{Strikethrough: true}),
		richtext.S(", and "),
		richtext.S("bold + strikethrough", richtext.SpanStyle{Bold: true, Strikethrough: true}),
		richtext.S("."),
	)
	weightsDoc := richtext.Build(
		richtext.S("Thin ", richtext.SpanStyle{Weight: draw.FontWeightThin}),
		richtext.S("Light ", richtext.SpanStyle{Weight: draw.FontWeightLight}),
		richtext.S("Regular ", richtext.SpanStyle{Weight: draw.FontWeightRegular}),
		richtext.S("Medium ", richtext.SpanStyle{Weight: draw.FontWeightMedium}),
		richtext.S("SemiBold ", richtext.SpanStyle{Weight: draw.FontWeightSemiBold}),
		richtext.S("Bold ", richtext.SpanStyle{Weight: draw.FontWeightBold}),
		richtext.S("Black", richtext.SpanStyle{Weight: draw.FontWeightBlack}),
	)
	decoDoc := richtext.Build(
		richtext.S("Bold+Italic", richtext.SpanStyle{Bold: true, Italic: true}),
		richtext.S(" | "),
		richtext.S("Underline+Strike", richtext.SpanStyle{Underline: true, Strikethrough: true}),
		richtext.S(" | "),
		richtext.S("All four", richtext.SpanStyle{Bold: true, Italic: true, Underline: true, Strikethrough: true}),
	)
	bgDoc := richtext.Build(
		richtext.S("Normal text with "),
		richtext.S("yellow highlight", richtext.SpanStyle{BgColor: draw.Hex("#fef08a")}),
		richtext.S(" and "),
		richtext.S("blue highlight", richtext.SpanStyle{BgColor: draw.Hex("#bfdbfe"), Color: draw.Hex("#1e40af")}),
		richtext.S(" inline."),
	)
	trackDoc := richtext.Build(
		richtext.S("Condensed ", richtext.SpanStyle{Tracking: -0.05}),
		richtext.S("Normal "),
		richtext.S("Expanded ", richtext.SpanStyle{Tracking: 0.1}),
		richtext.S("Very Expanded", richtext.SpanStyle{Tracking: 0.25}),
	)
	sizeDoc := richtext.Build(
		richtext.S("Small ", richtext.SpanStyle{Size: 10}),
		richtext.S("Normal ", richtext.SpanStyle{Size: 13}),
		richtext.S("Large ", richtext.SpanStyle{Size: 18}),
		richtext.S("XL", richtext.SpanStyle{Size: 24}),
	)
	wsDoc := richtext.Build(
		richtext.S("column1    column2    column3", richtext.SpanStyle{WhiteSpace: richtext.WhiteSpacePre}),
	)
	colorDecoDoc := richtext.Build(
		richtext.S("Red underline", richtext.SpanStyle{Underline: true, Color: draw.Hex("#ef4444")}),
		richtext.S(" | "),
		richtext.S("Blue strikethrough", richtext.SpanStyle{Strikethrough: true, Color: draw.Hex("#3b82f6")}),
		richtext.S(" | "),
		richtext.S("Purple all", richtext.SpanStyle{
			Bold: true, Italic: true, Underline: true, Strikethrough: true,
			Color: draw.Hex("#8b5cf6"),
		}),
	)
	displayItalicDoc := richtext.Build(
		richtext.S("Normal "),
		richtext.S("Italic ", richtext.SpanStyle{Italic: true}),
		richtext.S("Bold+Italic ", richtext.SpanStyle{Bold: true, Italic: true}),
		richtext.S("Tracked", richtext.SpanStyle{Tracking: 0.15}),
	)

	return layout.Column(
		sectionHeader("Font Formatting (CSS Inline)"),

		// ── Strikethrough ──────────────────────────────────────
		display.Text("Strikethrough:"),
		display.Spacer(4),
		richtext.New(strikeDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(strikeDoc),

		display.Spacer(16),

		// ── Font Weights (CSS font-weight 100–900) ─────────────
		display.Text("Font weights (100–900):"),
		display.Spacer(4),
		richtext.New(weightsDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(weightsDoc),

		display.Spacer(16),

		// ── Combined Decorations ───────────────────────────────
		display.Text("Combined text decorations:"),
		display.Spacer(4),
		richtext.New(decoDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(decoDoc),

		display.Spacer(16),

		// ── Background Highlight ───────────────────────────────
		display.Text("Background highlight (inline box):"),
		display.Spacer(4),
		richtext.New(bgDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(bgDoc),

		display.Spacer(16),

		// ── Letter-spacing / Tracking ──────────────────────────
		display.Text("Letter-spacing (CSS letter-spacing / tracking):"),
		display.Spacer(4),
		richtext.New(trackDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(trackDoc),

		display.Spacer(16),

		// ── Font Size Variations ───────────────────────────────
		display.Text("Mixed font sizes in one line:"),
		display.Spacer(4),
		richtext.New(sizeDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(sizeDoc),

		display.Spacer(16),

		// ── White-Space Pre ────────────────────────────────────
		display.Text("White-space: pre (preserves spaces):"),
		display.Spacer(4),
		richtext.New(wsDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(wsDoc),

		display.Spacer(16),

		// ── Colored text with decorations ──────────────────────
		display.Text("Colored decorations:"),
		display.Spacer(4),
		richtext.New(colorDecoDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(colorDecoDoc),

		display.Spacer(16),

		// ── Display-layer RichText with italic + tracking ──────
		display.Text("Display-layer RichText with italic + tracking:"),
		display.Spacer(4),
		display.RichTextSpans(
			display.Span{Text: "Normal "},
			display.Span{Text: "Italic ", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightRegular, Style: draw.FontStyleItalic}}},
			display.Span{Text: "Bold+Italic ", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Style: draw.FontStyleItalic}}},
			display.Span{Text: "Tracked", Style: display.SpanStyle{Style: draw.TextStyle{Tracking: 0.15}}},
		),
		display.Spacer(4),
		editBtn(displayItalicDoc),
	)
}

func paragraphStylingSection() ui.Element {
	lorem := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

	alignLeftDoc := richtext.Build(
		richtext.S(lorem),
	)
	alignCenterDoc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{Align: draw.TextAlignCenter}),
	)
	alignRightDoc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{Align: draw.TextAlignRight}),
	)
	alignJustifyDoc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{Align: draw.TextAlignJustify}),
	)
	indentDoc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{Indent: 24}),
	)
	paraSpacingDoc := richtext.Build(
		richtext.S("First paragraph.\nSecond paragraph with large gap before.\nThird paragraph with default spacing."),
	)
	lineHeight15Doc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{LineHeight: 1.5}),
	)
	lineHeight20Doc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{LineHeight: 2.0}),
	)
	combinedDoc := richtext.Build(
		richtext.S(lorem, richtext.SpanStyle{Align: draw.TextAlignCenter, Indent: 32, LineHeight: 1.5}),
	)
	mixedAlignDoc := richtext.Build(
		richtext.S("Left-aligned paragraph (default).\n"),
		richtext.S("Center-aligned paragraph.\n", richtext.SpanStyle{Align: draw.TextAlignCenter}),
		richtext.S("Right-aligned paragraph.", richtext.SpanStyle{Align: draw.TextAlignRight}),
	)

	return layout.Column(
		sectionHeader("Paragraph Styling (CSS Block-Level)"),

		// ── text-align ──────────────────────────────────────────
		display.Text("text-align: left (default)"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
		}),
		display.Spacer(4),
		editBtn(alignLeftDoc),

		display.Spacer(12),
		display.Text("text-align: center"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignCenter},
		}),
		display.Spacer(4),
		editBtn(alignCenterDoc),

		display.Spacer(12),
		display.Text("text-align: right"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignRight},
		}),
		display.Spacer(4),
		editBtn(alignRightDoc),

		display.Spacer(12),
		display.Text("text-align: justify"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{Align: draw.TextAlignJustify},
		}),
		display.Spacer(4),
		editBtn(alignJustifyDoc),

		// ── text-indent ─────────────────────────────────────────
		display.Spacer(16),
		display.Text("text-indent: 24dp"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{Indent: 24},
		}),
		display.Spacer(4),
		editBtn(indentDoc),

		// ── paragraph spacing ───────────────────────────────────
		display.Spacer(16),
		display.Text("Paragraph spacing: SpaceBefore=16, SpaceAfter=24"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "First paragraph."},
				},
				Style: display.ParagraphStyle{SpaceAfter: 24},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Second paragraph with large gap before."},
				},
				Style: display.ParagraphStyle{SpaceBefore: 16},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Third paragraph with default spacing."},
				},
			},
		),
		display.Spacer(4),
		editBtn(paraSpacingDoc),

		// ── line-height ─────────────────────────────────────────
		display.Spacer(16),
		display.Text("line-height: 1.5x"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{LineHeight: 1.5},
		}),
		display.Spacer(4),
		editBtn(lineHeight15Doc),

		display.Spacer(12),
		display.Text("line-height: 2.0x"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{LineHeight: 2.0},
		}),
		display.Spacer(4),
		editBtn(lineHeight20Doc),

		// ── Combined ────────────────────────────────────────────
		display.Spacer(16),
		display.Text("Combined: center + indent + line-height 1.5x"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: lorem},
			},
			Style: display.ParagraphStyle{
				Align:      draw.TextAlignCenter,
				Indent:     32,
				LineHeight: 1.5,
			},
		}),
		display.Spacer(4),
		editBtn(combinedDoc),

		// ── Mixed alignment ─────────────────────────────────────
		display.Spacer(16),
		display.Text("Mixed alignment in one RichText"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Left-aligned paragraph (default)."},
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Center-aligned paragraph."},
				},
				Style: display.ParagraphStyle{Align: draw.TextAlignCenter},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Right-aligned paragraph."},
				},
				Style: display.ParagraphStyle{Align: draw.TextAlignRight},
			},
		),
		display.Spacer(4),
		editBtn(mixedAlignDoc),
	)
}

func inlineWidgetsSection() ui.Element {
	return layout.Column(
		sectionHeader("Inline Widgets (RFC-003 §5.5)"),

		// ── Basic: text + inline badge ──────────────────────────
		display.Text("Text with inline badge:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Build status: "},
				display.InlineElement(display.BadgeColor(display.Text("passing"), draw.Hex("#22c55e"))),
				display.Span{Text: " — all checks completed."},
			},
		}),

		display.Spacer(12),

		// ── Multiple inline widgets ─────────────────────────────
		display.Text("Multiple inline widgets in one paragraph:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Tags: "},
				display.InlineElement(display.BadgeColor(display.Text("Go"), draw.Hex("#00ADD8"))),
				display.Span{Text: " "},
				display.InlineElement(display.BadgeColor(display.Text("UI"), draw.Hex("#8B5CF6"))),
				display.Span{Text: " "},
				display.InlineElement(display.BadgeColor(display.Text("GPU"), draw.Hex("#F97316"))),
				display.Span{Text: " — framework tags."},
			},
		}),

		display.Spacer(12),

		// ── Line wrapping ───────────────────────────────────────
		display.Text("Line wrapping with inline widgets:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "This paragraph contains enough text to demonstrate "},
				display.InlineElement(display.BadgeText("wrap")),
				display.Span{Text: " behavior when inline widgets are mixed with text spans "},
				display.InlineElement(display.BadgeColor(display.Text("across"), draw.Hex("#3b82f6"))),
				display.Span{Text: " multiple lines, just like inline-block elements "},
				display.InlineElement(display.BadgeColor(display.Text("in CSS"), draw.Hex("#a855f7"))),
				display.Span{Text: "."},
			},
		}),

		display.Spacer(12),

		// ── Baseline alignment ──────────────────────────────────
		display.Text("Baseline alignment (default vs shifted up):"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Default "},
				display.InlineElement(display.BadgeText("0")),
				display.Span{Text: " vs shifted "},
				display.InlineElementWithBaseline(display.BadgeText("+4"), 4),
				display.Span{Text: " vs more shifted "},
				display.InlineElementWithBaseline(display.BadgeText("+8"), 8),
				display.Span{Text: " — notice the vertical offset."},
			},
		}),

		display.Spacer(12),

		// ── Mixed styled spans + widgets ────────────────────────
		display.Text("Mixed styled spans and inline widgets:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Normal text, "},
				display.Span{Text: "bold text", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
				display.Span{Text: ", "},
				display.InlineElement(display.BadgeColor(display.Text("info"), draw.Hex("#3b82f6"))),
				display.Span{Text: ", "},
				display.Span{Text: "colored text", Style: display.SpanStyle{Color: draw.Hex("#ef4444")}},
				display.Span{Text: ", "},
				display.InlineElement(display.BadgeColor(display.Text("warn"), draw.Hex("#f59e0b"))),
				display.Span{Text: " — all inline."},
			},
		}),

		display.Spacer(12),

		// ── Multi-paragraph with inline widgets ─────────────────
		display.Text("Multi-paragraph with inline widgets:"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "First paragraph: version "},
					display.InlineElement(display.BadgeText("v2.1")),
					display.Span{Text: " released."},
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Second paragraph: status "},
					display.InlineElement(display.BadgeColor(display.Text("stable"), draw.Hex("#22c55e"))),
					display.Span{Text: " confirmed."},
				},
			},
		),

		display.Spacer(12),

		// ── Block widget (display: block) ───────────────────────
		display.Text("Block widget (breaks flow, full-width):"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Text before the block widget. "},
				display.BlockElement(
					display.BadgeColor(display.Text("Block-Level Widget"), draw.Hex("#3B82F6")),
				),
				display.Span{Text: "Text after the block widget."},
			},
		}),

		display.Spacer(12),

		// ── Block + inline mixed ────────────────────────────────
		display.Text("Mixed inline + block widgets:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.Span{Text: "Inline: "},
				display.InlineElement(display.BadgeColor(display.Text("tag"), draw.Hex("#22c55e"))),
				display.Span{Text: " then a block: "},
				display.BlockElement(
					display.BadgeColor(display.Text("Full-Width Block"), draw.Hex("#8B5CF6")),
				),
				display.Span{Text: "Continues inline after block."},
			},
		}),
	)
}

func richTextImagesSection(m Model) ui.Element {
	imgEditorDoc := richtext.Build(
		richtext.S("Image: "),
		richtext.S("\uFFFC", richtext.SpanStyle{
			Image: richtext.ImageAttachment{
				ImageID:   m.ImgChecker1,
				Alt:       "sample",
				Width:     24,
				Height:    24,
				ScaleMode: draw.ImageScaleStretch,
			},
		}),
		richtext.S(" — followed by more text."),
	)

	return layout.Column(
		sectionHeader("RichText Images (HTML §4.8.3)"),

		// ── Inline image in text flow ──────────────────────────────
		display.Text("Inline image in text flow:"),
		display.Spacer(4),
		display.RichTextContent(
			display.Span{Text: "Server status: "},
			display.InlineImage(m.ImgChecker1,
				display.WithImageSpanSize(20, 20),
				display.WithImageSpanAlt("status icon"),
			),
			display.Span{Text: " — system operational."},
		),

		display.Spacer(12),

		// ── Inline images of different sizes ──────────────────────
		display.Text("Inline images at different sizes:"),
		display.Spacer(4),
		display.RichTextContent(
			display.Span{Text: "Small "},
			display.InlineImage(m.ImgChecker1, display.WithImageSpanSize(16, 16), display.WithImageSpanAlt("16px")),
			display.Span{Text: " medium "},
			display.InlineImage(m.ImgChecker2, display.WithImageSpanSize(28, 28), display.WithImageSpanAlt("28px")),
			display.Span{Text: " large "},
			display.InlineImage(m.ImgChecker3, display.WithImageSpanSize(40, 40), display.WithImageSpanAlt("40px")),
			display.Span{Text: " — all inline."},
		),

		display.Spacer(12),

		// ── Scale modes ───────────────────────────────────────────
		display.Text("Scale modes — Fit / Fill / Stretch:"),
		display.Spacer(4),
		display.RichTextContent(
			display.Span{Text: "Fit "},
			display.InlineImage(m.ImgChecker2,
				display.WithImageSpanSize(48, 32),
				display.WithImageSpanScaleMode(draw.ImageScaleFit),
				display.WithImageSpanAlt("Fit"),
			),
			display.Span{Text: "  Fill "},
			display.InlineImage(m.ImgChecker2,
				display.WithImageSpanSize(48, 32),
				display.WithImageSpanScaleMode(draw.ImageScaleFill),
				display.WithImageSpanAlt("Fill"),
			),
			display.Span{Text: "  Stretch "},
			display.InlineImage(m.ImgChecker2,
				display.WithImageSpanSize(48, 32),
				display.WithImageSpanScaleMode(draw.ImageScaleStretch),
				display.WithImageSpanAlt("Stretch"),
			),
		),

		display.Spacer(12),

		// ── Opacity ───────────────────────────────────────────────
		display.Text("Opacity — 100% / 60% / 25%:"),
		display.Spacer(4),
		display.RichTextContent(
			display.InlineImage(m.ImgChecker3,
				display.WithImageSpanSize(40, 40),
				display.WithImageSpanOpacity(1.0),
				display.WithImageSpanAlt("100%"),
			),
			display.Span{Text: "  "},
			display.InlineImage(m.ImgChecker3,
				display.WithImageSpanSize(40, 40),
				display.WithImageSpanOpacity(0.6),
				display.WithImageSpanAlt("60%"),
			),
			display.Span{Text: "  "},
			display.InlineImage(m.ImgChecker3,
				display.WithImageSpanSize(40, 40),
				display.WithImageSpanOpacity(0.25),
				display.WithImageSpanAlt("25%"),
			),
		),

		display.Spacer(12),

		// ── Float-left ────────────────────────────────────────────
		display.Text("Float left — text wraps on the right:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.FloatLeftImage(m.ImgChecker1,
					display.WithImageSpanSize(72, 72),
					display.WithImageSpanAlt("float left"),
				),
				display.Span{Text: "This paragraph has a float-left image. The text flows on the right side of the image. A longer sentence demonstrates the wrapping behaviour across multiple lines."},
			},
		}),

		display.Spacer(12),

		// ── Float-right ───────────────────────────────────────────
		display.Text("Float right — text wraps on the left:"),
		display.Spacer(4),
		display.RichText(display.RichParagraph{
			Content: []display.ParagraphContent{
				display.FloatRightImage(m.ImgChecker2,
					display.WithImageSpanSize(72, 72),
					display.WithImageSpanAlt("float right"),
				),
				display.Span{Text: "This paragraph has a float-right image. Text flows on the left of the image, wrapping naturally when the line is too long for the reduced available width."},
			},
		}),

		display.Spacer(12),

		// ── Block image ───────────────────────────────────────────
		display.Text("Block image — full paragraph width (use separate paragraphs for captions):"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Caption above the image."},
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.BlockImage(m.ImgChecker3,
						display.WithImageSpanSize(0, 80),
						display.WithImageSpanScaleMode(draw.ImageScaleFit),
						display.WithImageSpanAlt("full-width block image"),
					),
				},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "Caption below the image."},
				},
			},
		),

		display.Spacer(12),

		// ── Editor with inline image ───────────────────────────────
		display.Text("RichText editor with inline image:"),
		display.Spacer(4),
		richtext.New(imgEditorDoc,
			richtext.WithReadOnly(),
			richtext.WithRows(2),
		),
		display.Spacer(4),
		editBtn(imgEditorDoc),
	)
}

func listsSection(m Model) ui.Element {
	ulDoc := richtext.Build(
		richtext.S("Apples\n", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("Bananas\n", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("Cherries", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
	)
	olDoc := richtext.Build(
		richtext.S("First step\n", richtext.SpanStyle{ListType: draw.ListTypeOrdered}),
		richtext.S("Second step\n", richtext.SpanStyle{ListType: draw.ListTypeOrdered}),
		richtext.S("Third step", richtext.SpanStyle{ListType: draw.ListTypeOrdered}),
	)
	nestedDoc := func() richtext.AttributedString {
		doc := richtext.Build(
			richtext.S("Fruits\nApple\nBanana\nTropical\nVegetables\nCarrot\nBroccoli"),
		)
		// Fruits (line 0-7)
		doc = doc.Apply(0, 7, richtext.ListTypeAttr(draw.ListTypeUnordered))
		// Apple (7-13)
		doc = doc.Apply(7, 13, richtext.ListTypeAttr(draw.ListTypeUnordered))
		doc = doc.Apply(7, 13, richtext.ListLevelAttr(1))
		// Banana (13-20)
		doc = doc.Apply(13, 20, richtext.ListTypeAttr(draw.ListTypeUnordered))
		doc = doc.Apply(13, 20, richtext.ListLevelAttr(1))
		// Tropical (20-29)
		doc = doc.Apply(20, 29, richtext.ListTypeAttr(draw.ListTypeUnordered))
		doc = doc.Apply(20, 29, richtext.ListLevelAttr(2))
		// Vegetables (29-40)
		doc = doc.Apply(29, 40, richtext.ListTypeAttr(draw.ListTypeUnordered))
		// Carrot (40-47)
		doc = doc.Apply(40, 47, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(40, 47, richtext.ListLevelAttr(1))
		// Broccoli (47-55)
		doc = doc.Apply(47, 55, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(47, 55, richtext.ListLevelAttr(1))
		return doc
	}()
	nestedOlDoc := func() richtext.AttributedString {
		doc := richtext.Build(
			richtext.S("First chapter\nSection alpha\nSection beta\nDetail one\nDetail two\nSecond chapter\nAnother section"),
		)
		// First chapter (0-14)
		doc = doc.Apply(0, 14, richtext.ListTypeAttr(draw.ListTypeOrdered))
		// Section alpha (14-28)
		doc = doc.Apply(14, 28, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(14, 28, richtext.ListLevelAttr(1))
		// Section beta (28-41)
		doc = doc.Apply(28, 41, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(28, 41, richtext.ListLevelAttr(1))
		// Detail one (41-52)
		doc = doc.Apply(41, 52, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(41, 52, richtext.ListLevelAttr(2))
		// Detail two (52-62)
		doc = doc.Apply(52, 62, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(52, 62, richtext.ListLevelAttr(2))
		// Second chapter (62-76)
		doc = doc.Apply(62, 76, richtext.ListTypeAttr(draw.ListTypeOrdered))
		// Another section (76-91)
		doc = doc.Apply(76, 91, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(76, 91, richtext.ListLevelAttr(1))
		return doc
	}()
	olStartDoc := func() richtext.AttributedString {
		doc := richtext.Build(
			richtext.S("Fifth item\nSixth item\nSeventh item"),
		)
		doc = doc.Apply(0, len(doc.Text), richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(0, len(doc.Text), richtext.ListStartAttr(5))
		return doc
	}()
	styledListDoc := richtext.Build(
		richtext.S("This is ", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("bold", richtext.SpanStyle{Bold: true, ListType: draw.ListTypeUnordered}),
		richtext.S(" text\n", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("This is ", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("italic", richtext.SpanStyle{Italic: true, ListType: draw.ListTypeUnordered}),
		richtext.S(" text\n", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("This is ", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
		richtext.S("colored", richtext.SpanStyle{Color: draw.Hex("#ef4444"), ListType: draw.ListTypeUnordered}),
		richtext.S(" text", richtext.SpanStyle{ListType: draw.ListTypeUnordered}),
	)
	customMarkerDoc := func() richtext.AttributedString {
		doc := richtext.Build(
			richtext.S("Lower alpha\nLower alpha\nLower roman\nLower roman\nSquare bullet"),
		)
		// Lower alpha items (0-12, 12-24)
		doc = doc.Apply(0, 12, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(0, 12, richtext.ListMarkerAttr(draw.ListMarkerLowerAlpha))
		doc = doc.Apply(12, 24, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(12, 24, richtext.ListMarkerAttr(draw.ListMarkerLowerAlpha))
		// Lower roman items (24-36, 36-48)
		doc = doc.Apply(24, 36, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(24, 36, richtext.ListMarkerAttr(draw.ListMarkerLowerRoman))
		doc = doc.Apply(36, 48, richtext.ListTypeAttr(draw.ListTypeOrdered))
		doc = doc.Apply(36, 48, richtext.ListMarkerAttr(draw.ListMarkerLowerRoman))
		// Square bullet (48-61)
		doc = doc.Apply(48, 61, richtext.ListTypeAttr(draw.ListTypeUnordered))
		doc = doc.Apply(48, 61, richtext.ListMarkerAttr(draw.ListMarkerSquare))
		return doc
	}()

	return layout.Column(
		sectionHeader("Lists"),

		// ── 1. Simple unordered list ──
		display.Text("Unordered list (ul):"),
		display.Spacer(4),
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
		display.Spacer(4),
		editBtn(ulDoc),

		display.Spacer(16),

		// ── 2. Simple ordered list ──
		display.Text("Ordered list (ol):"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "First step"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Second step"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Third step"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
		),
		display.Spacer(4),
		editBtn(olDoc),

		display.Spacer(16),

		// ── 3. Nested list ──
		display.Text("Nested lists (mixed ul/ol):"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Fruits"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Apple"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Banana"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Tropical"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListLevel: 2},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Vegetables"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Carrot"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Broccoli"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
		),
		display.Spacer(4),
		editBtn(nestedDoc),

		display.Spacer(16),

		// ── 3b. Nested ordered list (decimal -> lower-alpha -> lower-roman) ──
		display.Text("Nested ordered list (auto marker cycling):"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "First chapter"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Section alpha"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Section beta"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Detail one"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 2},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Detail two"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 2},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Second chapter"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Another section"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListLevel: 1},
			},
		),
		display.Spacer(4),
		editBtn(nestedOlDoc),

		display.Spacer(16),

		// ── 4. Ordered list with start number ──
		display.Text("Ordered list with start=5:"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Fifth item"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Sixth item"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Seventh item"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListStart: 5},
			},
		),
		display.Spacer(4),
		editBtn(olStartDoc),

		display.Spacer(16),

		// ── 5. List with formatted text ──
		display.Text("List with styled content:"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "This is "},
					display.Span{Text: "bold", Style: display.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold}}},
					display.Span{Text: " text"},
				},
				Style: display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "This is "},
					display.Span{Text: "italic", Style: display.SpanStyle{Style: draw.TextStyle{Style: draw.FontStyleItalic}}},
					display.Span{Text: " text"},
				},
				Style: display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{
					display.Span{Text: "This is "},
					display.Span{Text: "colored", Style: display.SpanStyle{Color: draw.Hex("#ef4444")}},
					display.Span{Text: " text"},
				},
				Style: display.ParagraphStyle{ListType: draw.ListTypeUnordered},
			},
		),
		display.Spacer(4),
		editBtn(styledListDoc),

		display.Spacer(16),

		// ── 6. Custom marker styles ──
		display.Text("Custom marker styles:"),
		display.Spacer(4),
		display.RichText(
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Lower alpha"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListMarker: draw.ListMarkerLowerAlpha},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Lower alpha"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListMarker: draw.ListMarkerLowerAlpha},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Lower roman"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListMarker: draw.ListMarkerLowerRoman},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Lower roman"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeOrdered, ListMarker: draw.ListMarkerLowerRoman},
			},
			display.RichParagraph{
				Content: []display.ParagraphContent{display.Span{Text: "Square bullet"}},
				Style:   display.ParagraphStyle{ListType: draw.ListTypeUnordered, ListMarker: draw.ListMarkerSquare},
			},
		),
		display.Spacer(4),
		editBtn(customMarkerDoc),

		display.Spacer(16),

		// ── 7. Editable list in RichTextEditor ──
		display.Text("Editable list (with toolbar):"),
		display.Spacer(4),
		func() ui.Element {
			cmds := append(richtext.DefaultCommands(), richtext.ListCommands()...)
			return richtext.NewEditorWithToolbar(m.ListEditorDoc,
				richtext.WithWidgetOnChange(func(as richtext.AttributedString) { app.Send(SetListEditorDocMsg{as}) }),
				richtext.WithWidgetFocus(app.Focus()),
				richtext.WithWidgetScroll(m.ListEditorScroll),
				richtext.WithWidgetRows(6),
				richtext.WithWidgetCommands(cmds),
			)
		}(),
	)
}

func textShapingSection() ui.Element {
	return layout.Column(
		sectionHeader("Text Shaping (Phase 4)"),

		display.Text("GoTextShaper — go-text/typesetting with full OpenType GSUB/GPOS:"),
		display.Spacer(8),

		// Size comparison: MSDF vs bitmap threshold at 24px
		display.Text("Size Rendering (MSDF >= 24px, Bitmap < 24px):"),
		display.Spacer(4),
		display.TextStyled("12px — bitmap rasterized, hinted", draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular}),
		display.TextStyled("18px — bitmap rasterized, hinted", draw.TextStyle{Size: 18, Weight: draw.FontWeightRegular}),
		display.TextStyled("24px — MSDF rendered, scalable", draw.TextStyle{Size: 24, Weight: draw.FontWeightRegular}),
		display.TextStyled("32px — MSDF rendered, scalable", draw.TextStyle{Size: 32, Weight: draw.FontWeightRegular}),

		display.Spacer(12),
		display.Text("Font Fallback Chain (RFC-003 §3.4):"),
		display.Spacer(4),
		display.Text("Latin: The quick brown fox jumps over the lazy dog"),
		display.Text("Digits & Symbols: 0123456789 @#$%&*()[]{}"),
		display.Text("Punctuation: .,;:!? - — ' \" ... /"),

		display.Spacer(12),
		display.Text("Per-Glyph Fallback:"),
		display.Spacer(4),
		display.Text("Primary font -> Fallback chain -> Embedded Noto Sans -> U+FFFD"),
		display.Text("Missing glyphs are individually resolved, not entire runs."),

		display.Spacer(12),
		display.Text("Shaper Details:"),
		display.Spacer(4),
		display.Text("  Implementation: GoTextShaper (go-text/typesetting v0.3.4)"),
		display.Text("  Shaping: HarfBuzz-compatible, pure Go"),
		display.Text("  Scripts: Latin, Arabic, Devanagari, CJK (GSUB/GPOS)"),
		display.Text("  Fallback: Noto Sans (embedded)"),
		display.Text("  Rasterization: MSDF (>=24px) / Hinted Bitmap (<24px)"),
	)
}

// ── Grapheme Navigation Section ───────────────────────────────────

func graphemeNavSection() ui.Element {
	type sample struct {
		label string
		text  string
	}
	samples := []sample{
		{"ASCII", "Hello"},
		{"Flag \U0001F1E9\U0001F1EA", "\U0001F1E9\U0001F1EA"},
		{"ZWJ family \U0001F468\u200D\U0001F469\u200D\U0001F467", "\U0001F468\u200D\U0001F469\u200D\U0001F467"},
		{"Combining e\u0301", "e\u0301"},
		{"Skin tone \U0001F469\U0001F3FD", "\U0001F469\U0001F3FD"},
	}

	items := []ui.Element{
		sectionHeader("Grapheme Navigation"),
		display.Text("Grapheme cluster analysis (UAX #29) — each row shows cluster count,"),
		display.Text("byte length, and rune count for sample strings."),
		display.Spacer(12),
	}

	for _, s := range samples {
		clusters := text.GraphemeClusters(s.text)
		nClusters := len(clusters) - 1
		if len(s.text) == 0 {
			nClusters = 0
		}
		nRunes := 0
		for range s.text {
			nRunes++
		}
		info := fmt.Sprintf("  %s — %d cluster(s), %d bytes, %d rune(s)",
			s.label, nClusters, len(s.text), nRunes)
		items = append(items, display.Text(info))
		items = append(items, display.Spacer(4))
	}
	return layout.Column(items...)
}

// ── Line Breaking Section ────────────────────────────────────────

func lineBreakingSection() ui.Element {
	type sample struct {
		label string
		text  string
	}
	samples := []sample{
		{"English", "The quick brown fox jumps over the lazy dog."},
		{"CJK", "\u4f60\u597d\u4e16\u754c\u6d4b\u8bd5\u6587\u672c\u6362\u884c"},
		{"Mandatory (\\n)", "Line one.\nLine two.\nLine three."},
		{"Non-breaking space", "100\u00A0km should not break"},
	}

	items := []ui.Element{
		sectionHeader("Line Breaking"),
		display.Text("UAX #14 line break analysis — mandatory and opportunity breaks."),
		display.Spacer(12),
	}

	for _, s := range samples {
		breaks := text.DefaultLineBreaker.Breaks(s.text)
		mandatory := 0
		opportunity := 0
		for _, b := range breaks {
			if b.Kind == text.LineBreakMandatory {
				mandatory++
			} else {
				opportunity++
			}
		}
		info := fmt.Sprintf("  %s — %d mandatory, %d opportunity break(s)",
			s.label, mandatory, opportunity)
		items = append(items, display.Text(info))
		items = append(items, display.Spacer(4))
	}
	return layout.Column(items...)
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	initial := Model{
		Dark:          true,
		NavTree:       ui.NewTreeState(),
		ActiveSection: "rich-text",
		Scroll:        &ui.ScrollState{},
		NavSplitRatio: 0.22,
		EditorScroll:  &ui.ScrollState{},
		// RichTextEditor defaults
		RichEditorDoc: richtext.Build(
			richtext.S("Hello ", richtext.SpanStyle{Bold: true}),
			richtext.S("Rich World!\nThis is a "),
			richtext.S("fully editable", richtext.SpanStyle{Italic: true}),
			richtext.S(" rich text editor.\nTry typing, selecting, and using undo/redo."),
		),
		RichEditorScroll: &ui.ScrollState{},
		RichEditorDoc2: richtext.Build(
			richtext.S("Read-only content with "),
			richtext.S("styled spans", richtext.SpanStyle{Bold: true, Color: draw.Hex("#3b82f6")}),
			richtext.S("."),
		),
		RichEditorScroll2: &ui.ScrollState{},
		// Lists demo defaults
		ListEditorDoc: func() richtext.AttributedString {
			doc := richtext.Build(
				richtext.S("First item\nSecond item\nThird item\n"),
				richtext.S("Sub-item A\nSub-item B"),
			)
			doc = doc.Apply(0, 11, richtext.ListTypeAttr(draw.ListTypeUnordered))
			doc = doc.Apply(11, 23, richtext.ListTypeAttr(draw.ListTypeUnordered))
			doc = doc.Apply(23, 34, richtext.ListTypeAttr(draw.ListTypeUnordered))
			doc = doc.Apply(34, 45, richtext.ListTypeAttr(draw.ListTypeUnordered))
			doc = doc.Apply(34, 45, richtext.ListLevelAttr(1))
			doc = doc.Apply(45, 55, richtext.ListTypeAttr(draw.ListTypeUnordered))
			doc = doc.Apply(45, 55, richtext.ListLevelAttr(1))
			return doc
		}(),
		ListEditorScroll: &ui.ScrollState{},
		// Images
		ImageStore: luximage.NewStore(),
	}

	initial.ImgChecker1 = generateColorChecker(initial.ImageStore, 64, 64, 8,
		220, 220, 240,
		59, 130, 246,
	)
	initial.ImgChecker2 = generateColorChecker(initial.ImageStore, 128, 64, 12,
		255, 160, 50,
		30, 180, 160,
	)
	initial.ImgChecker3 = generateColorChecker(initial.ImageStore, 120, 60, 10,
		230, 80, 160,
		80, 200, 80,
	)

	if err := app.RunWithCmd(initial, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Lux Text Playground"),
		app.WithSize(900, 700),
		app.WithImageStore(initial.ImageStore),
	); err != nil {
		log.Fatal(err)
	}
}

// Ensure text package is used (grapheme-nav and line-breaking sections).
var _ = text.GraphemeClusters
