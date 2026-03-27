// Hello World (M2) — renders text and a button.
//
// Usage:
//
//	go run ./examples/fenster/
package main

import (
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

type Model struct{}

func update(m Model, msg app.Msg) Model {
	return m
}

func view(m Model) ui.Element {
	return layout.Column(
		display.Text("HELLO WORLD"),
		button.Text("CLICK ME", nil),
	)
}

func main() {
	if err := app.Run(Model{}, update, view,
		app.WithTheme(theme.Default),
		app.WithTitle("LUX — M2 Hello World"),
		app.WithSize(800, 600),
	); err != nil {
		log.Fatal(err)
	}
}
