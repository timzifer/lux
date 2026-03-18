// Counter (M4) — interactive counter with theme switching and hover animations.
//
// This builds on the M3 counter (Appendix B) and adds M4 features:
//   - Dark/Light theme toggle via SetDarkModeMsg
//   - Button hover animation (automatic via framework)
//
//	go run ./examples/counter/
package main

import (
	"fmt"
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

type Model struct {
	Count int
	Dark  bool
}

type IncrMsg struct{}
type DecrMsg struct{}
type ToggleThemeMsg struct{}

func update(m Model, msg app.Msg) Model {
	switch msg.(type) {
	case IncrMsg:
		m.Count++
	case DecrMsg:
		m.Count--
	case ToggleThemeMsg:
		m.Dark = !m.Dark
		app.Send(app.SetDarkModeMsg{Dark: m.Dark})
	}
	return m
}

func view(m Model) ui.Element {
	themeLabel := "LIGHT"
	if m.Dark {
		themeLabel = "DARK"
	}
	return ui.Column(
		ui.Text(fmt.Sprintf("Count: %d", m.Count)),
		ui.Divider(),
		ui.Row(
			ui.ButtonText("-", func() { app.Send(DecrMsg{}) }),
			ui.ButtonText("+", func() { app.Send(IncrMsg{}) }),
		),
		ui.Divider(),
		ui.ButtonText(themeLabel, func() { app.Send(ToggleThemeMsg{}) }),
	)
}

func main() {
	if err := app.Run(Model{Count: 0, Dark: true}, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Counter"),
	); err != nil {
		log.Fatal(err)
	}
}
