// Acid Viewer — visual renderer for the three official ACID conformance tests.
//
// Renders the real W3C/WaSP Acid tests through the full lux pipeline:
//   HTML → DOM → CSS cascade → ui/html widget tree → GPU
//
// Most of Acid2 and Acid3 will look broken — that's the point.
// Each rendering gap is a concrete improvement to pick off.
//
//	go run ./examples/acid-viewer/
package main

import (
	_ "embed"
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/html"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
)

//go:embed testdata/acid1.html
var acid1HTML string

//go:embed testdata/acid2.html
var acid2HTML string

//go:embed testdata/acid3.html
var acid3HTML string

// ── Model ────────────────────────────────────────────────────────

type Model struct {
	Dark     bool
	Selected int // active tab index (0–2)
}

type SelectTabMsg struct{ Index int }
type ToggleThemeMsg struct{}

func update(m Model, msg app.Msg) Model {
	switch msg := msg.(type) {
	case SelectTabMsg:
		m.Selected = msg.Index
	case app.ModelRestoredMsg:
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	}
	return m
}

func view(m Model) ui.Element {
	themeLabel := "Light"
	if m.Dark {
		themeLabel = "Dark"
	}

	tabs := nav.New(
		[]nav.TabItem{
			{Header: display.Text("Acid1 — CSS1 (1998)"), Content: html.View(acid1HTML, html.WithScrollable(600))},
			{Header: display.Text("Acid2 — CSS2.1 (2005)"), Content: html.View(acid2HTML, html.WithScrollable(600))},
			{Header: display.Text("Acid3 — CSS3/JS (2008)"), Content: html.View(acid3HTML, html.WithScrollable(600))},
		},
		m.Selected,
		func(i int) { app.Send(SelectTabMsg{Index: i}) },
	)

	return layout.Column(
		layout.Row(
			display.Text("ACID Test Viewer"),
			layout.Expand(display.Empty()),
			button.Text(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
		),
		display.Divider(),
		tabs,
	)
}

func main() {
	if err := app.Run(Model{Dark: true, Selected: 0}, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("ACID Test Viewer"),
		app.WithSize(800, 700),
	); err != nil {
		log.Fatal(err)
	}
}
