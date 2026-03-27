//go:build nogui

package layout_test

import (
	"testing"

	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/ui/nav"
)

// TestTableHeightInColumn verifies that a table with THead/TBody inside a
// Column correctly propagates its full height (header + body rows).
func TestTableHeightInColumn(t *testing.T) {
	table10 := layout.NewTable([]ui.Element{
		layout.NewTableCaption(display.Text("Employee Schedule")),
		layout.THead(
			layout.TR(
				layout.TH(display.Text("Name")),
				layout.TH(display.Text("Mon")),
				layout.TH(display.Text("Tue")),
				layout.TH(display.Text("Wed")),
				layout.TH(display.Text("Thu")),
				layout.TH(display.Text("Fri")),
			),
		),
		layout.TBody(
			layout.TR(
				layout.TD(display.Text("Alice")),
				layout.TD(display.Text("9-5"), layout.WithColSpan(3)),
				layout.TD(display.Text("Off"), layout.WithColSpan(2)),
			),
			layout.TR(
				layout.TD(display.Text("Bob")),
				layout.TD(display.Text("Off")),
				layout.TD(display.Text("10-6"), layout.WithColSpan(4)),
			),
			layout.TR(
				layout.TD(display.Text("Carol")),
				layout.TD(display.Text("8-4"), layout.WithColSpan(5)),
			),
		),
	}, layout.WithBorderSpacing(8, 4))

	scene := buildScene(table10, 800, 5000)

	// All body-row names must appear as glyphs.
	bodyTexts := []string{"Alice", "Bob", "Carol"}
	for _, want := range bodyTexts {
		found := false
		for _, g := range scene.Glyphs {
			if g.Text == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("body-row glyph %q not found in scene", want)
		}
	}

	// Verify body rows are BELOW the header row.
	headerTexts := []string{"Name", "Mon", "Tue", "Wed", "Thu", "Fri"}
	headerMaxY := 0
	for _, g := range scene.Glyphs {
		for _, h := range headerTexts {
			if g.Text == h && g.Y > headerMaxY {
				headerMaxY = g.Y
			}
		}
	}
	for _, g := range scene.Glyphs {
		for _, b := range bodyTexts {
			if g.Text == b && g.Y <= headerMaxY {
				t.Errorf("body glyph %q (Y=%d) should be below header (maxY=%d)", b, g.Y, headerMaxY)
			}
		}
	}
}

// TestScrollViewTableSectionEndReachable verifies that Example 10's body rows
// are reachable when the full table section is wrapped in a ScrollView.
func TestScrollViewTableSectionEndReachable(t *testing.T) {
	// Build a simplified table section: several tables followed by Example 10.
	var children []ui.Element
	for i := 0; i < 9; i++ {
		children = append(children, display.Text("Section header"))
		children = append(children, display.Spacer(4))
		children = append(children, layout.NewTable([]ui.Element{
			layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
			layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		}, layout.WithBorderSpacing(8, 4)))
		children = append(children, display.Spacer(16))
	}
	children = append(children, display.Text("10. Complex Table"))
	children = append(children, display.Spacer(4))
	children = append(children,
		layout.NewTable([]ui.Element{
			layout.NewTableCaption(display.Text("Schedule")),
			layout.THead(layout.TR(
				layout.TH(display.Text("Name")),
				layout.TH(display.Text("Day")),
			)),
			layout.TBody(
				layout.TR(layout.TD(display.Text("Alice")), layout.TD(display.Text("Mon"))),
				layout.TR(layout.TD(display.Text("Bob")), layout.TD(display.Text("Tue"))),
				layout.TR(layout.TD(display.Text("Carol")), layout.TD(display.Text("Wed"))),
			),
		}, layout.WithBorderSpacing(8, 4)),
	)

	col := layout.Column(children...)
	scrollState := &ui.ScrollState{}
	sv := nav.NewScrollView(col, 400, scrollState)

	// Scroll to the end. We need a real Interactor so the ScrollView
	// clamps the offset during the render pass (IX != nil guard).
	scrollState.Offset = 99999
	ix := ui.NewInteractor(&hit.Map{}, nil)
	c1 := render.NewSceneCanvas(800, 800)
	ui.BuildScene(sv, c1, theme.Default, 800, 800, ix)

	maxScroll := scrollState.Offset
	t.Logf("maxScroll = %.1f", maxScroll)

	// Now render at clamped offset.
	c2 := render.NewSceneCanvas(800, 800)
	scene := ui.BuildScene(sv, c2, theme.Default, 800, 800, ix)

	// After scrolling to the end, "Carol" MUST be visible.
	carolVisible := false
	for _, g := range scene.Glyphs {
		if g.Text == "Carol" {
			carolVisible = true
		}
	}
	if !carolVisible {
		t.Error("Carol NOT visible at max scroll — last table body is clipped!")
	}
}
