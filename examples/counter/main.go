// Counter (M3) — interactive counter with increment/decrement buttons.
//
// This is the Appendix B example from RFC-001:
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
}

type IncrMsg struct{}
type DecrMsg struct{}

func update(m Model, msg app.Msg) Model {
	switch msg.(type) {
	case IncrMsg:
		m.Count++
	case DecrMsg:
		m.Count--
	}
	return m
}

func view(m Model) ui.Element {
	return ui.Column(
		ui.Text(fmt.Sprintf("Count: %d", m.Count)),
		ui.Row(
			ui.Button("-", func() { app.Send(DecrMsg{}) }),
			ui.Button("+", func() { app.Send(IncrMsg{}) }),
		),
	)
}

func main() {
	if err := app.Run(Model{Count: 0}, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("Counter"),
	); err != nil {
		log.Fatal(err)
	}
}
