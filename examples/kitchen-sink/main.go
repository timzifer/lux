// Kitchen Sink — demonstrates all Lux widgets (Tier 1 + Tier 2 + M5).
//
// Split-view layout: Tree navigation on the left, active test case on the right.
// Showcases Flex, Grid, Padding, SizedBox, VirtualList, Tree, and RichText.
//
//	go run ./examples/kitchen-sink/
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// ── Section Registry ──────────────────────────────────────────────

var sectionIDs = []string{
	"typography", "buttons", "form-controls", "range-progress",
	"selection", "layout", "rich-text", "virtual-list", "tree",
}

func sectionLabel(id string) string {
	switch id {
	case "typography":
		return "Typography"
	case "buttons":
		return "Buttons & Icons"
	case "form-controls":
		return "Form Controls"
	case "range-progress":
		return "Range & Progress"
	case "selection":
		return "Selection"
	case "layout":
		return "Layout"
	case "rich-text":
		return "RichText"
	case "virtual-list":
		return "VirtualList"
	case "tree":
		return "Tree"
	default:
		return id
	}
}

func sectionChildren(_ string) []string { return nil }

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark          bool
	Count         int
	CheckA        bool
	CheckB        bool
	RadioChoice   string
	ToggleOn      bool
	SliderVal     float32
	Progress      float32
	SelectVal     string
	TextValue     string
	Scroll        *ui.ScrollState
	AnimTime      float64
	NavTree       *ui.TreeState
	ActiveSection string
	VListScroll   *ui.ScrollState
	DemoTree      *ui.TreeState
}

// ── Messages ─────────────────────────────────────────────────────

type IncrMsg struct{}
type DecrMsg struct{}
type ToggleThemeMsg struct{}
type SetCheckAMsg struct{ Value bool }
type SetCheckBMsg struct{ Value bool }
type SetRadioMsg struct{ Choice string }
type SetToggleMsg struct{ Value bool }
type SetSliderMsg struct{ Value float32 }
type SetTextMsg struct{ Value string }
type SelectSectionMsg struct{ Section string }

// ── Update ───────────────────────────────────────────────────────

func update(m Model, msg app.Msg) Model {
	switch msg := msg.(type) {
	case IncrMsg:
		m.Count++
	case DecrMsg:
		m.Count--
	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	case SetCheckAMsg:
		m.CheckA = msg.Value
	case SetCheckBMsg:
		m.CheckB = msg.Value
	case SetRadioMsg:
		m.RadioChoice = msg.Choice
	case SetToggleMsg:
		m.ToggleOn = msg.Value
	case SetSliderMsg:
		m.SliderVal = msg.Value
	case SetTextMsg:
		m.TextValue = msg.Value
	case SelectSectionMsg:
		m.ActiveSection = msg.Section

	case app.TickMsg:
		dt := msg.DeltaTime.Seconds()
		m.AnimTime += dt
		m.Progress = float32(math.Mod(m.AnimTime*0.15, 1.0))
	}
	return m
}

// ── View ─────────────────────────────────────────────────────────

func view(m Model) ui.Element {
	themeLabel := "Light"
	if !m.Dark {
		themeLabel = "Dark"
	}

	// Left panel: Tree navigation
	nav := ui.Tree(ui.TreeConfig{
		RootIDs:  sectionIDs,
		Children: sectionChildren,
		BuildNode: func(id string, _ int, _, selected bool) ui.Element {
			return ui.Text(sectionLabel(id))
		},
		NodeHeight: 28,
		MaxHeight:  500,
		State:      m.NavTree,
		OnSelect:   func(id string) { app.Send(SelectSectionMsg{id}) },
	})

	// Right panel: active section content
	content := ui.ScrollView(sectionContent(m), 500, m.Scroll)

	return ui.Padding(ui.UniformInsets(16), ui.Column(
		// Split view: nav + content
		ui.Row(
			ui.SizedBox(200, 500, nav),
			ui.Spacer(16),
			content,
		),
		// Footer
		ui.Spacer(12),
		ui.Row(
			ui.Button(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
		),
	))
}

func sectionContent(m Model) ui.Element {
	switch m.ActiveSection {
	case "typography":
		return typographySection()
	case "buttons":
		return buttonsSection(m)
	case "form-controls":
		return formControlsSection(m)
	case "range-progress":
		return rangeProgressSection(m)
	case "selection":
		return selectionSection(m)
	case "layout":
		return layoutSection()
	case "rich-text":
		return richTextSection()
	case "virtual-list":
		return virtualListSection(m)
	case "tree":
		return treeSection(m)
	default:
		return ui.Column(
			ui.Spacer(24),
			ui.Text("Select a section from the tree on the left."),
		)
	}
}

// ── Section Views ────────────────────────────────────────────────

func sectionHeader(title string) ui.Element {
	return ui.Column(
		ui.Spacer(8),
		ui.TextStyled(title, draw.TextStyle{
			Size:   16,
			Weight: draw.FontWeightSemiBold,
		}),
		ui.Spacer(4),
	)
}

func typographySection() ui.Element {
	return ui.Column(
		sectionHeader("Typography"),
		ui.TextStyled("Heading 1 (H1)", theme.Default.Tokens().Typography.H1),
		ui.TextStyled("Heading 2 (H2)", theme.Default.Tokens().Typography.H2),
		ui.TextStyled("Heading 3 (H3)", theme.Default.Tokens().Typography.H3),
		ui.Text("Body text — the quick brown fox jumps over the lazy dog."),
		ui.TextStyled("Body Small — metadata and captions", theme.Default.Tokens().Typography.BodySmall),
	)
}

func buttonsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Buttons & Icons"),
		ui.Text(fmt.Sprintf("Counter: %d", m.Count)),
		ui.Row(
			ui.Button("-", func() { app.Send(DecrMsg{}) }),
			ui.Button("+", func() { app.Send(IncrMsg{}) }),
		),
		ui.Spacer(8),
		ui.Text("Icons:"),
		ui.Row(
			ui.Icon("★"),
			ui.IconSize("→", 24),
			ui.Icon("♦"),
		),
	)
}

func formControlsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Form Controls"),
		ui.TextField(m.TextValue, "Enter text...",
			ui.WithOnChange(func(v string) { app.Send(SetTextMsg{v}) }),
			ui.WithFocusState(app.Focus()),
		),
		ui.Spacer(8),
		ui.Checkbox("Enable notifications", m.CheckA, func(v bool) { app.Send(SetCheckAMsg{v}) }),
		ui.Checkbox("Auto-save", m.CheckB, func(v bool) { app.Send(SetCheckBMsg{v}) }),
		ui.Spacer(8),
		ui.Radio("Alpha", m.RadioChoice == "alpha", func() { app.Send(SetRadioMsg{"alpha"}) }),
		ui.Radio("Beta", m.RadioChoice == "beta", func() { app.Send(SetRadioMsg{"beta"}) }),
		ui.Radio("Gamma", m.RadioChoice == "gamma", func() { app.Send(SetRadioMsg{"gamma"}) }),
		ui.Spacer(8),
		ui.Row(
			ui.Text("Dark mode:"),
			ui.Toggle(m.ToggleOn, func(v bool) { app.Send(SetToggleMsg{v}) }),
		),
	)
}

func rangeProgressSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Range & Progress"),
		ui.Text(fmt.Sprintf("Slider value: %.0f%%", m.SliderVal*100)),
		ui.Slider(m.SliderVal, func(v float32) { app.Send(SetSliderMsg{v}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Progress: %.0f%%", m.Progress*100)),
		ui.ProgressBar(m.Progress),
		ui.Spacer(4),
		ui.Text("Indeterminate:"),
		ui.ProgressBarIndeterminate(float32(math.Mod(m.AnimTime*0.8, 1.0))),
	)
}

func selectionSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Selection"),
		ui.Select(m.SelectVal, []string{"Option 1", "Option 2", "Option 3"}),
	)
}

func layoutSection() ui.Element {
	return ui.Column(
		sectionHeader("Layout"),

		// Row
		ui.Text("Row:"),
		ui.Row(ui.Text("A"), ui.Text("B"), ui.Text("C")),
		ui.Spacer(8),

		// Stack
		ui.Text("Stack (overlapping):"),
		ui.Stack(ui.Text("Bottom"), ui.Text("Top")),
		ui.Spacer(8),

		// Flex with Justify
		ui.Text("Flex (JustifySpaceBetween):"),
		ui.Flex([]ui.Element{
			ui.Text("Left"),
			ui.Text("Center"),
			ui.Text("Right"),
		}, ui.WithJustify(ui.JustifySpaceBetween)),
		ui.Spacer(8),

		// Flex with Expanded
		ui.Text("Flex with Expanded:"),
		ui.Flex([]ui.Element{
			ui.Button("Fixed", nil),
			ui.Expanded(ui.Text("← takes remaining space →")),
			ui.Button("Fixed", nil),
		}),
		ui.Spacer(8),

		// Grid
		ui.Text("Grid (3 columns):"),
		ui.Grid(3, []ui.Element{
			ui.Text("Cell 1"), ui.Text("Cell 2"), ui.Text("Cell 3"),
			ui.Text("Cell 4"), ui.Text("Cell 5"), ui.Text("Cell 6"),
		}, ui.WithColGap(12), ui.WithRowGap(8)),
		ui.Spacer(8),

		// Padding
		ui.Text("Padding (16dp):"),
		ui.Padding(ui.UniformInsets(16), ui.Text("Padded content")),
		ui.Spacer(8),

		// SizedBox
		ui.Text("SizedBox (100x50):"),
		ui.SizedBox(100, 50, ui.Text("Sized")),
	)
}

func richTextSection() ui.Element {
	return ui.Column(
		sectionHeader("RichText"),
		ui.Text("Mixed styles in a single line:"),
		ui.Spacer(4),
		ui.RichTextSpans(
			ui.Span{Text: "Bold text ", Style: ui.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
			ui.Span{Text: "and normal text "},
			ui.Span{Text: "with color", Style: ui.SpanStyle{Color: draw.Hex("#3b82f6")}},
		),
		ui.Spacer(12),
		ui.Text("Multiple paragraphs:"),
		ui.Spacer(4),
		ui.RichText(
			ui.RichParagraph{Spans: []ui.Span{
				{Text: "First paragraph with "},
				{Text: "bold", Style: ui.SpanStyle{Style: draw.TextStyle{Weight: draw.FontWeightBold, Size: 13}}},
				{Text: " and "},
				{Text: "colored", Style: ui.SpanStyle{Color: draw.Hex("#ef4444")}},
				{Text: " spans."},
			}},
			ui.RichParagraph{Spans: []ui.Span{
				{Text: "Second paragraph. Rich text supports per-span styling."},
			}},
		),
	)
}

func virtualListSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("VirtualList"),
		ui.Text("1000 items — only visible items are rendered:"),
		ui.Spacer(8),
		ui.VirtualList(ui.VirtualListConfig{
			ItemCount:  1000,
			ItemHeight: 24,
			BuildItem: func(i int) ui.Element {
				return ui.Text(fmt.Sprintf("  Item %d — virtualized row", i))
			},
			MaxHeight: 200,
			State:     m.VListScroll,
		}),
	)
}

// Demo tree data for the Tree section.
var demoTreeRoots = []string{"Documents", "Pictures", "Music"}

func demoTreeChildren(id string) []string {
	switch id {
	case "Documents":
		return []string{"Work", "Personal", "Archive"}
	case "Work":
		return []string{"Reports", "Presentations"}
	case "Pictures":
		return []string{"Vacation", "Family"}
	case "Music":
		return []string{"Rock", "Jazz", "Classical"}
	default:
		return nil
	}
}

func treeSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Tree"),
		ui.Text("Hierarchical tree with expand/collapse:"),
		ui.Spacer(8),
		ui.Tree(ui.TreeConfig{
			RootIDs:  demoTreeRoots,
			Children: demoTreeChildren,
			BuildNode: func(id string, _ int, _, _ bool) ui.Element {
				return ui.Text(id)
			},
			NodeHeight: 24,
			MaxHeight:  200,
			State:      m.DemoTree,
		}),
	)
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	initial := Model{
		Dark:          true,
		RadioChoice:   "alpha",
		SliderVal:     0.5,
		Progress:      0.0,
		SelectVal:     "Option 1",
		Scroll:        &ui.ScrollState{},
		NavTree:       ui.NewTreeState(),
		ActiveSection: "typography",
		VListScroll:   &ui.ScrollState{},
		DemoTree:      ui.NewTreeState(),
	}

	if err := app.Run(initial, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Lux Kitchen Sink"),
		app.WithSize(900, 700),
	); err != nil {
		log.Fatal(err)
	}
}
