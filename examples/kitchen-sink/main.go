// Kitchen Sink — demonstrates all Lux widgets (Tier 1 + Tier 2).
//
// This example serves as a living catalogue of every available widget,
// organized in sections. It supports dark/light theme switching and is
// designed to be easily extended as new widget tiers are added.
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
)

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark        bool
	Count       int
	CheckA      bool
	CheckB      bool
	RadioChoice string
	ToggleOn    bool
	SliderVal   float32
	Progress    float32
	SelectVal   string
	TextValue   string
	Scroll      *ui.ScrollState
	AnimTime    float64 // seconds, drives indeterminate progress + auto-progress
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

	// ── Scroll handling ──────────────────────────────────
	case input.ScrollMsg:
		// ScrollBy expects negative delta for "scroll down" (content moves up).
		// Platform sends positive deltaY for scroll-up, negative for scroll-down.
		m.Scroll.ScrollBy(msg.DeltaY*30, 1200, 500)

	// ── Keyboard handling for focused TextField ──────────
	case input.KeyMsg:
		if msg.Action == input.KeyPress || msg.Action == input.KeyRepeat {
			focus := app.Focus()
			if focus.FocusedID > 0 {
				ui.HandleKeyMsg(focus, msg.Key, m.TextValue, func(v string) {
					m.TextValue = v
				})
			}
		}
	case input.CharMsg:
		focus := app.Focus()
		if focus.FocusedID > 0 {
			ui.HandleCharInput(msg.Char, m.TextValue, func(v string) {
				m.TextValue = v
			})
		}

	// ── Animation tick ───────────────────────────────────
	case app.TickMsg:
		dt := msg.DeltaTime.Seconds()
		m.AnimTime += dt
		// Auto-cycle the determinate progress bar.
		m.Progress = float32(math.Mod(m.AnimTime*0.15, 1.0))
	}
	return m
}

// ── View ─────────────────────────────────────────────────────────

func view(m Model) ui.Element {
	themeLabel := "Switch to Light"
	if !m.Dark {
		themeLabel = "Switch to Dark"
	}

	return ui.ScrollView(ui.Column(
		// ── Typography ──────────────────────────────────────
		sectionHeader("Typography"),
		ui.TextStyled("Heading 1 (H1)", theme.Default.Tokens().Typography.H1),
		ui.TextStyled("Heading 2 (H2)", theme.Default.Tokens().Typography.H2),
		ui.TextStyled("Heading 3 (H3)", theme.Default.Tokens().Typography.H3),
		ui.Text("Body text — the quick brown fox jumps over the lazy dog."),
		ui.TextStyled("Body Small — metadata and captions", theme.Default.Tokens().Typography.BodySmall),
		ui.Divider(),

		// ── Buttons & Icons ─────────────────────────────────
		sectionHeader("Buttons & Icons"),
		ui.Text(fmt.Sprintf("Counter: %d", m.Count)),
		ui.Row(
			ui.Button("-", func() { app.Send(DecrMsg{}) }),
			ui.Button("+", func() { app.Send(IncrMsg{}) }),
		),
		ui.Row(
			ui.Icon("★"),
			ui.IconSize("→", 24),
			ui.Icon("♦"),
		),
		ui.Divider(),

		// ── Form Controls ───────────────────────────────────
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
		ui.Divider(),

		// ── Range & Progress ────────────────────────────────
		sectionHeader("Range & Progress"),
		ui.Text(fmt.Sprintf("Slider value: %.0f%%", m.SliderVal*100)),
		ui.Slider(m.SliderVal, func(v float32) { app.Send(SetSliderMsg{v}) }),
		ui.Spacer(8),
		ui.Text(fmt.Sprintf("Progress: %.0f%%", m.Progress*100)),
		ui.ProgressBar(m.Progress),
		ui.Spacer(4),
		ui.Text("Indeterminate:"),
		ui.ProgressBarIndeterminate(),
		ui.Divider(),

		// ── Selection ───────────────────────────────────────
		sectionHeader("Selection"),
		ui.Select(m.SelectVal, []string{"Option 1", "Option 2", "Option 3"}),
		ui.Divider(),

		// ── Layout ──────────────────────────────────────────
		sectionHeader("Layout"),
		ui.Text("Row:"),
		ui.Row(ui.Text("A"), ui.Text("B"), ui.Text("C")),
		ui.Spacer(4),
		ui.Text("Stack (overlapping):"),
		ui.Stack(ui.Text("Bottom"), ui.Text("Top")),
		ui.Spacer(4),
		ui.Text("Spacer (24dp gap below):"),
		ui.Spacer(24),
		ui.Divider(),

		// ── Theme Toggle ────────────────────────────────────
		ui.Spacer(8),
		ui.Button(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
		ui.Spacer(16),
	), 500, m.Scroll)
}

// sectionHeader renders a section title using H2 typography.
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

// ── Main ─────────────────────────────────────────────────────────

func main() {
	initial := Model{
		Dark:        true,
		RadioChoice: "alpha",
		SliderVal:   0.5,
		Progress:    0.0,
		SelectVal:   "Option 1",
		Scroll:      &ui.ScrollState{},
	}

	if err := app.Run(initial, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Lux Kitchen Sink"),
		app.WithSize(900, 700),
	); err != nil {
		log.Fatal(err)
	}
}
