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
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// ── Section Registry ──────────────────────────────────────────────

var sectionIDs = []string{
	"typography", "buttons", "form-controls", "range-progress",
	"selection", "layout", "rich-text", "virtual-list", "tree",
	"cards", "tabs", "accordion", "badges-chips", "menus",
	"shortcuts", "overlays", "canvas-paints", "scoped-themes",
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
	case "cards":
		return "Cards"
	case "tabs":
		return "Tabs"
	case "accordion":
		return "Accordion"
	case "badges-chips":
		return "Badges & Chips"
	case "menus":
		return "Menus"
	case "shortcuts":
		return "Shortcuts"
	case "overlays":
		return "Overlays"
	case "canvas-paints":
		return "Canvas & Paints"
	case "scoped-themes":
		return "Scoped Themes"
	default:
		return id
	}
}

func sectionChildren(_ string) []string { return nil }

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark           bool
	Count          int
	CheckA         bool
	CheckB         bool
	RadioChoice    string
	ToggleOn       bool
	SliderVal      float32
	Progress       float32
	SelectVal      string
	TextValue      string
	Scroll         *ui.ScrollState
	AnimTime       float64
	NavTree        *ui.TreeState
	ActiveSection  string
	ToggleAnim     *ui.ToggleState
	VListScroll    *ui.ScrollState
	DemoTree       *ui.TreeState
	TabIndex       int
	AccordionState *ui.AccordionState
	ChipASelected  bool
	ChipBSelected  bool
	ChipCSelected  bool
	ChipDismissed  bool
	LastMenuAction string
	MenuBarState   *ui.MenuBarState
	// Phase 1 features
	ShortcutLog   string
	OverlayOpen   bool
	HandlerLog    string
	KineticScroll *ui.KineticScroll
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
type SetTabMsg struct{ Index int }
type ToggleChipAMsg struct{}
type ToggleChipBMsg struct{}
type ToggleChipCMsg struct{}
type DismissChipMsg struct{}
type MenuActionMsg struct{ Action string }
type ToggleOverlayMsg struct{}
type DismissOverlayMsg struct{}
type SetHandlerLogMsg struct{ Text string }

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
	case SetTabMsg:
		m.TabIndex = msg.Index
	case ToggleChipAMsg:
		m.ChipASelected = !m.ChipASelected
	case ToggleChipBMsg:
		m.ChipBSelected = !m.ChipBSelected
	case ToggleChipCMsg:
		m.ChipCSelected = !m.ChipCSelected
	case DismissChipMsg:
		m.ChipDismissed = true
	case MenuActionMsg:
		m.LastMenuAction = msg.Action
	case input.ShortcutMsg:
		m.ShortcutLog = fmt.Sprintf("Shortcut: %s", msg.ID)
		switch msg.ID {
		case "incr":
			m.Count++
		case "decr":
			m.Count--
		}
	case ToggleOverlayMsg:
		m.OverlayOpen = !m.OverlayOpen
	case DismissOverlayMsg:
		m.OverlayOpen = false
	case ui.DismissOverlayMsg:
		m.OverlayOpen = false
	case SetHandlerLogMsg:
		m.HandlerLog = msg.Text

	case app.TickMsg:
		dt := msg.DeltaTime.Seconds()
		m.AnimTime += dt
		m.Progress = float32(math.Mod(m.AnimTime*0.15, 1.0))
		m.ToggleAnim.Tick(msg.DeltaTime)
		m.NavTree.Tick(msg.DeltaTime)
		m.DemoTree.Tick(msg.DeltaTime)
		if m.KineticScroll != nil {
			m.KineticScroll.Tick(msg.DeltaTime)
		}
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
			ui.ButtonText(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
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
	case "cards":
		return cardsSection()
	case "tabs":
		return tabsSection(m)
	case "accordion":
		return accordionSection(m)
	case "badges-chips":
		return badgesChipsSection(m)
	case "menus":
		return menusSection(m)
	case "shortcuts":
		return shortcutsSection(m)
	case "overlays":
		return overlaysSection(m)
	case "canvas-paints":
		return canvasPaintsSection()
	case "scoped-themes":
		return scopedThemesSection()
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
			ui.ButtonText("-", func() { app.Send(DecrMsg{}) }),
			ui.ButtonText("+", func() { app.Send(IncrMsg{}) }),
		),
		ui.Spacer(8),
		ui.Text("Icons (Phosphor):"),
		ui.Row(
			ui.Icon(icons.Star),
			ui.Icon(icons.ArrowRight),
			ui.Icon(icons.Heart),
			ui.Icon(icons.Gear),
			ui.Icon(icons.Eye),
			ui.Icon(icons.Sun),
			ui.Icon(icons.Moon),
		),
	)
}

func formControlsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Form Controls"),
		ui.TextField(m.TextValue, "Enter text...",
			ui.WithOnChange(func(v string) { app.Send(SetTextMsg{v}) }),
			ui.WithFocus(app.Focus()),
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
			ui.Toggle(m.ToggleOn, func(v bool) { app.Send(SetToggleMsg{v}) }, m.ToggleAnim),
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
			ui.ButtonText("Fixed", nil),
			ui.Expanded(ui.Text("← takes remaining space →")),
			ui.ButtonText("Fixed", nil),
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
			BuildNode: func(id string, _ int, expanded, _ bool) ui.Element {
				kids := demoTreeChildren(id)
				if len(kids) > 0 {
					icon := icons.Folder
					if expanded {
						icon = icons.FolderOpen
					}
					return ui.Row(ui.Icon(icon), ui.Text(id))
				}
				return ui.Row(ui.Icon(icons.FileText), ui.Text(id))
			},
			NodeHeight: 24,
			MaxHeight:  200,
			State:      m.DemoTree,
		}),
	)
}

// ── Tier 3 Section Views ─────────────────────────────────────────

func cardsSection() ui.Element {
	return ui.Column(
		sectionHeader("Cards"),
		ui.Text("Card with text content:"),
		ui.Spacer(4),
		ui.Card(
			ui.Text("This content lives inside a Card."),
			ui.Text("Cards have elevation and borders."),
		),
		ui.Spacer(12),
		ui.Text("Nested cards:"),
		ui.Spacer(4),
		ui.Card(
			ui.Text("Outer card"),
			ui.Spacer(8),
			ui.Card(ui.Text("Inner nested card")),
		),
	)
}

func tabsSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Tabs"),
		ui.Text("Tabs with rich headers (Icon + Text + Badge):"),
		ui.Spacer(4),
		ui.Tabs([]ui.TabItem{
			{
				Header:  ui.Row(ui.Icon(icons.Star), ui.Text("General")),
				Content: ui.Text("General settings content goes here."),
			},
			{
				Header:  ui.Row(ui.Icon(icons.Gear), ui.Text("Advanced"), ui.BadgeText("3")),
				Content: ui.Column(ui.Text("Advanced settings."), ui.Text("With multiple items.")),
			},
			{
				Header:  ui.Row(ui.Icon(icons.Eye), ui.Text("Preview")),
				Content: ui.Card(ui.Text("Preview content inside a Card.")),
			},
		}, m.TabIndex, func(idx int) { app.Send(SetTabMsg{idx}) }),
	)
}

func accordionSection(m Model) ui.Element {
	return ui.Column(
		sectionHeader("Accordion"),
		ui.Text("Collapsible sections (click to expand/collapse):"),
		ui.Spacer(4),
		ui.Accordion([]ui.AccordionSection{
			{
				Header:  ui.Text("Section 1 — Getting Started"),
				Content: ui.Text("Welcome! This section covers the basics."),
			},
			{
				Header:  ui.Text("Section 2 — Configuration"),
				Content: ui.Column(ui.Text("Configure your settings here."), ui.Text("Multiple widgets supported.")),
			},
			{
				Header:  ui.Text("Section 3 — Advanced Topics"),
				Content: ui.Card(ui.Text("Advanced content inside a Card.")),
			},
		}, m.AccordionState),
	)
}

func badgesChipsSection(m Model) ui.Element {
	tokens := theme.Default.Tokens()
	children := []ui.Element{
		sectionHeader("Badges & Chips"),

		ui.Text("Badges (colorful pill indicators):"),
		ui.Spacer(4),
		ui.Row(
			ui.BadgeText("3"),
			ui.BadgeColor(ui.Text("99+"), tokens.Colors.Status.Error),
			ui.BadgeColor(ui.Icon(icons.Star), tokens.Colors.Status.Warning),
			ui.BadgeColor(ui.Text("New"), tokens.Colors.Status.Success),
			ui.BadgeColor(ui.Row(ui.Icon(icons.Heart), ui.Text("Hot")), tokens.Colors.Accent.Secondary),
		),

		ui.Spacer(12),
		ui.Text("Chips (selectable):"),
		ui.Spacer(4),
		ui.Row(
			ui.Chip(ui.Text("Go"), m.ChipASelected, func() { app.Send(ToggleChipAMsg{}) }),
			ui.Chip(ui.Text("Rust"), m.ChipBSelected, func() { app.Send(ToggleChipBMsg{}) }),
			ui.Chip(ui.Text("Python"), m.ChipCSelected, func() { app.Send(ToggleChipCMsg{}) }),
		),
	}

	// Dismissible chip (shown until dismissed)
	if !m.ChipDismissed {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Dismissible chip (click × to remove):"),
			ui.ChipDismissible(
				ui.Row(ui.Icon(icons.Star), ui.Text("Featured")),
				true,
				func() {},
				func() { app.Send(DismissChipMsg{}) },
			),
		)
	} else {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Chip dismissed!"),
		)
	}

	children = append(children,
		ui.Spacer(12),
		ui.Text("Tooltip (hover to show):"),
		ui.Spacer(4),
		ui.Row(
			ui.Tooltip(
				ui.Text("← Hover me for tooltip"),
				ui.Text("This is a tooltip with arbitrary content!"),
			),
		),
	)

	return ui.Column(children...)
}

func menusSection(m Model) ui.Element {
	menuAction := func(action string) func() {
		return func() { app.Send(MenuActionMsg{action}) }
	}

	children := []ui.Element{
		sectionHeader("Menus"),

		ui.Text("MenuBar (click to open dropdown):"),
		ui.Spacer(4),
		ui.MenuBar([]ui.MenuItem{
			{Label: ui.Text("File"), Items: []ui.MenuItem{
				{Label: ui.Text("New"), OnClick: menuAction("File > New")},
				{Label: ui.Text("Open"), OnClick: menuAction("File > Open")},
				{Label: ui.Text("Save"), OnClick: menuAction("File > Save")},
			}},
			{Label: ui.Text("Edit"), Items: []ui.MenuItem{
				{Label: ui.Text("Undo"), OnClick: menuAction("Edit > Undo")},
				{Label: ui.Text("Redo"), OnClick: menuAction("Edit > Redo")},
				{Label: ui.Text("Cut"), OnClick: menuAction("Edit > Cut")},
				{Label: ui.Text("Copy"), OnClick: menuAction("Edit > Copy")},
				{Label: ui.Text("Paste"), OnClick: menuAction("Edit > Paste")},
			}},
			{Label: ui.Text("View"), Items: []ui.MenuItem{
				{Label: ui.Text("Zoom In"), OnClick: menuAction("View > Zoom In")},
				{Label: ui.Text("Zoom Out"), OnClick: menuAction("View > Zoom Out")},
			}},
			{Label: ui.Text("Help"), OnClick: menuAction("Help")},
		}, m.MenuBarState),
	}

	if m.LastMenuAction != "" {
		children = append(children,
			ui.Spacer(4),
			ui.Text(fmt.Sprintf("Last action: %s", m.LastMenuAction)),
		)
	}

	children = append(children,
		ui.Spacer(12),
		ui.Text("ContextMenu:"),
		ui.Spacer(4),
		ui.ContextMenu([]ui.MenuItem{
			{Label: ui.Text("Cut"), OnClick: menuAction("Cut")},
			{Label: ui.Text("Copy"), OnClick: menuAction("Copy")},
			{Label: ui.Text("Paste"), OnClick: menuAction("Paste")},
			{Label: ui.Text("Delete"), OnClick: menuAction("Delete")},
		}, true, 300, 400),
	)

	return ui.Column(children...)
}

// ── Phase 1 Sections ──────────────────────────────────────────────

func shortcutsSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Keyboard Shortcuts"),
		ui.Text("Registered shortcuts:"),
		ui.Spacer(4),
		ui.Text("  Ctrl+I → Increment counter"),
		ui.Text("  Ctrl+D → Decrement counter"),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Counter value: %d", m.Count)),
	}

	if m.ShortcutLog != "" {
		children = append(children,
			ui.Spacer(8),
			ui.Text(fmt.Sprintf("Last shortcut: %s", m.ShortcutLog)),
		)
	}

	children = append(children,
		ui.Spacer(16),
		sectionHeader("Global Handler Layer"),
		ui.Text("A global handler logs all key events before widget dispatch."),
	)
	if m.HandlerLog != "" {
		children = append(children,
			ui.Spacer(4),
			ui.Text(fmt.Sprintf("Handler log: %s", m.HandlerLog)),
		)
	}

	return ui.Column(children...)
}

func overlaysSection(m Model) ui.Element {
	children := []ui.Element{
		sectionHeader("Overlay System"),
		ui.Text("Click the button to toggle a dismissable overlay:"),
		ui.Spacer(4),
		ui.ButtonText("Toggle Overlay", func() { app.Send(ToggleOverlayMsg{}) }),
	}

	if m.OverlayOpen {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Overlay is OPEN (click outside or press button to close)"),
			ui.Spacer(4),
			// The actual Overlay element rendered above normal flow.
			ui.Overlay{
				ID:          "demo-overlay",
				Anchor:      draw.R(300, 300, 100, 30),
				Placement:   ui.PlacementBelow,
				Dismissable: true,
				OnDismiss:   func() { app.Send(DismissOverlayMsg{}) },
				Content: ui.Card(ui.Column(
					ui.Text("This is an overlay!"),
					ui.Spacer(4),
					ui.Text("It renders above normal content."),
					ui.Spacer(8),
					ui.ButtonText("Close", func() { app.Send(DismissOverlayMsg{}) }),
				)),
			},
		)
	} else {
		children = append(children,
			ui.Spacer(8),
			ui.Text("Overlay is closed."),
		)
	}

	children = append(children,
		ui.Spacer(16),
		sectionHeader("Kinetic Scrolling"),
		ui.Text("KineticScroll with friction-decay physics is available."),
		ui.Text("Use trackpad for smooth kinetic scrolling or mouse wheel for discrete steps."),
	)

	return ui.Column(children...)
}

// ── Phase 3 Sections ──────────────────────────────────────────

func canvasPaintsSection() ui.Element {
	// Demonstrate the new Phase 3 Canvas API and Paint variants.
	// Since the GPU backend doesn't render these yet, this section
	// serves as an API showcase and compile-time validation.

	// 1. PathBuilder with ArcTo
	arcPath := draw.NewPath().
		MoveTo(draw.Pt(0, 0)).
		ArcTo(30, 30, 0, false, true, draw.Pt(60, 0)).
		LineTo(draw.Pt(60, 40)).
		LineTo(draw.Pt(0, 40)).
		Close().
		Build()
	_ = arcPath // used for FillPath once GPU supports it

	// 2. Gradient paints
	linearPaint := draw.LinearGradientPaint(
		draw.Pt(0, 0), draw.Pt(200, 0),
		draw.GradientStop{Offset: 0, Color: draw.Hex("#3b82f6")},
		draw.GradientStop{Offset: 1, Color: draw.Hex("#6366f1")},
	)
	radialPaint := draw.RadialGradientPaint(
		draw.Pt(50, 50), 50,
		draw.GradientStop{Offset: 0, Color: draw.Hex("#ffffff")},
		draw.GradientStop{Offset: 1, Color: draw.Hex("#09090b")},
	)
	_ = linearPaint
	_ = radialPaint

	// 3. TextLayout
	_ = draw.TextLayout{
		Text:      "Centered text layout",
		Style:     draw.TextStyle{Size: 14, Weight: draw.FontWeightRegular},
		MaxWidth:  300,
		Alignment: draw.TextAlignCenter,
	}

	// 4. LayerOptions
	_ = draw.LayerOptions{
		BlendMode: draw.BlendNormal,
		Opacity:   0.8,
		CacheHint: true,
	}

	return ui.Column(
		sectionHeader("Canvas & Paints (Phase 3)"),

		ui.Text("New Canvas API (GPU stubs — API validation):"),
		ui.Spacer(4),
		ui.Text("  PathBuilder.ArcTo — elliptical arc segments"),
		ui.Text("  PushClipRoundRect / PushClipPath — advanced clipping"),
		ui.Text("  PushBlur / PopBlur — backdrop blur effects"),
		ui.Text("  PushLayer / PopLayer — compositing layers"),
		ui.Text("  PushScale — uniform/non-uniform scaling"),
		ui.Text("  DrawTextLayout — rich text layout with alignment"),
		ui.Text("  DrawImageSlice — 9-slice image rendering"),
		ui.Text("  DrawTexture — external texture surfaces"),

		ui.Spacer(12),
		ui.Text("Paint Variants:"),
		ui.Spacer(4),
		ui.Text(fmt.Sprintf("  LinearGradientPaint: %d stops", 2)),
		ui.Text(fmt.Sprintf("  RadialGradientPaint: radius=%.0f", float64(50))),
		ui.Text("  PatternPaint: tiled image fills"),

		ui.Spacer(12),
		ui.Text("Theme-Lookup-Cache:"),
		ui.Spacer(4),
		ui.Text("  CachedTheme wraps Theme with lazy resolution"),
		ui.Text("  Auto-invalidation on SetThemeMsg / SetDarkModeMsg"),
		ui.Text("  Warm-up before first frame in app.Run"),
	)
}

// ── Scoped Themes Section ─────────────────────────────────────────

// Pre-built theme overrides for the scoped-themes demo.
var (
	dangerTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#dc2626"), // Red-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#f87171"), // Red-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})

	successTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#16a34a"), // Green-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#4ade80"), // Green-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})

	warningTheme = theme.Override(theme.Default, theme.OverrideSpec{
		Colors: &theme.ColorScheme{
			Surface: theme.Default.Tokens().Colors.Surface,
			Accent: theme.AccentColors{
				Primary:         draw.Hex("#d97706"), // Amber-600
				PrimaryContrast: draw.Hex("#ffffff"),
				Secondary:       draw.Hex("#fbbf24"), // Amber-400
			},
			Stroke: theme.Default.Tokens().Colors.Stroke,
			Text:   theme.Default.Tokens().Colors.Text,
			Status: theme.Default.Tokens().Colors.Status,
		},
	})
)

func scopedThemesSection() ui.Element {
	return ui.Column(
		sectionHeader("Scoped Themes"),
		ui.Text("ui.Themed() overrides the theme for a subtree."),
		ui.Text("Buttons below inherit their accent color from scoped themes:"),
		ui.Spacer(12),

		// Default (no override)
		ui.Text("Default theme:"),
		ui.Spacer(4),
		ui.Row(
			ui.ButtonText("Save", nil),
			ui.ButtonText("Submit", nil),
		),
		ui.Spacer(12),

		// Danger theme
		ui.Text("Danger theme (red accent):"),
		ui.Spacer(4),
		ui.Themed(dangerTheme,
			ui.Row(
				ui.ButtonText("Delete", nil),
				ui.ButtonText("Reset All", nil),
			),
		),
		ui.Spacer(12),

		// Success theme
		ui.Text("Success theme (green accent):"),
		ui.Spacer(4),
		ui.Themed(successTheme,
			ui.Row(
				ui.ButtonText("Confirm", nil),
				ui.ButtonText("Approve", nil),
			),
		),
		ui.Spacer(12),

		// Warning theme
		ui.Text("Warning theme (amber accent):"),
		ui.Spacer(4),
		ui.Themed(warningTheme,
			ui.Row(
				ui.ButtonText("Proceed", nil),
				ui.ButtonText("Override", nil),
			),
		),
		ui.Spacer(12),

		// Mixed: default and themed side by side
		ui.Text("Mixed — default and danger in one row:"),
		ui.Spacer(4),
		ui.Row(
			ui.ButtonText("Normal", nil),
			ui.Themed(dangerTheme,
				ui.ButtonText("Danger", nil),
			),
			ui.ButtonText("Normal", nil),
		),
	)
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	initial := Model{
		Dark:           true,
		RadioChoice:    "alpha",
		SliderVal:      0.5,
		Progress:       0.0,
		SelectVal:      "Option 1",
		Scroll:         &ui.ScrollState{},
		ToggleAnim:     ui.NewToggleState(),
		NavTree:        ui.NewTreeState(),
		ActiveSection:  "typography",
		VListScroll:    &ui.ScrollState{},
		DemoTree:       ui.NewTreeState(),
		AccordionState: ui.NewAccordionState(),
		MenuBarState:   ui.NewMenuBarState(),
		KineticScroll:  ui.NewKineticScroll(theme.Default.Tokens().Scroll),
	}

	// Global handler that logs key events (Phase 1: §2.8).
	keyLogger := func(ev ui.InputEvent) bool {
		if ev.Kind == ui.EventKey && ev.Key != nil {
			app.Send(SetHandlerLogMsg{Text: fmt.Sprintf("Key=%d Action=%d", ev.Key.Key, ev.Key.Action)})
		}
		return false // don't consume — let events pass through
	}

	if err := app.Run(initial, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Lux Kitchen Sink"),
		app.WithSize(900, 700),
		// Phase 1: Keyboard Shortcuts (RFC-002 §2.5)
		app.WithShortcut(input.Shortcut{Key: input.KeyI, Modifiers: input.ModCtrl}, "incr"),
		app.WithShortcut(input.Shortcut{Key: input.KeyD, Modifiers: input.ModCtrl}, "decr"),
		// Phase 1: Global Handler Layer (RFC-002 §2.8)
		app.WithGlobalHandler(keyLogger),
	); err != nil {
		log.Fatal(err)
	}
}
