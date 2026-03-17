// Fenster (M1) — A black window that opens and closes.
//
// This is the first milestone of the lux UI toolkit.
// It demonstrates the Elm architecture skeleton: model → update → view → render.
//
// Usage:
//
//	go run ./examples/fenster/
package main

import (
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/ui"
)

// Model is the application state. Empty for M1.
type Model struct{}

// update processes messages. No-op for M1.
func update(m Model, msg app.Msg) Model {
	return m
}

// view renders the model. Returns an empty element for M1.
func view(m Model) ui.Element {
	return ui.Empty()
}

func main() {
	if err := app.Run(Model{}, update, view,
		app.WithTitle("LUX — M1 Fenster"),
		app.WithSize(800, 600),
	); err != nil {
		log.Fatal(err)
	}
}
