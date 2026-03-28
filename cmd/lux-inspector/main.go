// Command lux-inspector is the Lux Widget Inspector — a standalone Lux
// application that connects to a running Lux app via Vellum and visualizes
// the widget tree, events, layout, and performance metrics (RFC-012 §6).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/internal/vellum"
	"github.com/timzifer/lux/theme"
)

func main() {
	addr := flag.String("addr", "unix:///tmp/lux-inspector.sock", "Vellum inspector socket address")
	flag.Parse()

	client, err := vellum.Connect(*addr, vellum.WithDebugExtensions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "lux-inspector: connect: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	log.Printf("lux-inspector: connected to %s", *addr)

	model := NewInspectorModel(client)

	if err := app.Run(model, inspectorUpdate, inspectorView,
		app.WithTitle("Lux Inspector"),
		app.WithSize(1400, 900),
		app.WithTheme(theme.LuxDark),
	); err != nil {
		fmt.Fprintf(os.Stderr, "lux-inspector: %v\n", err)
		os.Exit(1)
	}
}
