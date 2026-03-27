package uitest

import (
	"testing"

	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

// ── Flex: Grow/Shrink ─────────────────────────────────────────────

func TestGoldenFlexGrow(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				layout.Expand(display.Text("Grow-1"), 1),
				layout.Expand(display.Text("Grow-2"), 2),
			},
			layout.WithDirection(layout.FlexRow),
		),
		400, 100,
	)
	AssertScene(t, scene, "testdata/flex_grow.golden")
}

func TestGoldenFlexWrap(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("Item-A"),
				display.Text("Item-B"),
				display.Text("Item-C"),
				display.Text("Item-D"),
				display.Text("Item-E"),
			},
			layout.WithDirection(layout.FlexRow),
			layout.WithWrap(layout.FlexWrapOn),
			layout.WithGap(5),
		),
		120, 200,
	)
	AssertScene(t, scene, "testdata/flex_wrap.golden")
}

func TestGoldenFlexOrder(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				layout.FlexChild(display.Text("Third"), layout.WithOrder(3)),
				layout.FlexChild(display.Text("First"), layout.WithOrder(1)),
				layout.FlexChild(display.Text("Second"), layout.WithOrder(2)),
			},
			layout.WithDirection(layout.FlexRow),
			layout.WithGap(10),
		),
		400, 100,
	)
	AssertScene(t, scene, "testdata/flex_order.golden")
}

func TestGoldenFlexSpaceBetween(t *testing.T) {
	scene := BuildScene(
		layout.NewFlex(
			[]ui.Element{
				display.Text("Start"),
				display.Text("End"),
			},
			layout.WithDirection(layout.FlexRow),
			layout.WithJustify(layout.JustifySpaceBetween),
		),
		400, 100,
	)
	AssertScene(t, scene, "testdata/flex_space_between.golden")
}

// ── Grid ──────────────────────────────────────────────────────────

func TestGoldenGrid2x2(t *testing.T) {
	scene := BuildScene(
		layout.NewGrid(2, []ui.Element{
			display.Text("A1"),
			display.Text("A2"),
			display.Text("B1"),
			display.Text("B2"),
		},
			layout.WithRowGap(5),
			layout.WithColGap(10),
		),
		400, 200,
	)
	AssertScene(t, scene, "testdata/grid_2x2.golden")
}

func TestGoldenGridFrUnits(t *testing.T) {
	scene := BuildScene(
		layout.NewTemplateGrid(
			[]layout.TrackSize{layout.Fr(1), layout.Fr(2), layout.Fr(1)},
			[]ui.Element{
				display.Text("1fr"),
				display.Text("2fr"),
				display.Text("1fr"),
			},
			layout.WithColGap(5),
		),
		400, 100,
	)
	AssertScene(t, scene, "testdata/grid_fr_units.golden")
}

// ── Table ─────────────────────────────────────────────────────────

func TestGoldenSimpleTable(t *testing.T) {
	scene := BuildScene(
		layout.SimpleTable(
			[]ui.Element{
				display.Text("Name"),
				display.Text("Value"),
			},
			[][]ui.Element{
				{display.Text("Alpha"), display.Text("100")},
				{display.Text("Beta"), display.Text("200")},
			},
		),
		400, 200,
	)
	AssertScene(t, scene, "testdata/table_simple.golden")
}

func TestGoldenTableFixed(t *testing.T) {
	scene := BuildScene(
		layout.SimpleTable(
			[]ui.Element{
				display.Text("Col-A"),
				display.Text("Col-B"),
				display.Text("Col-C"),
			},
			[][]ui.Element{
				{display.Text("X"), display.Text("Y"), display.Text("Z")},
			},
			layout.WithTableLayout(layout.TableLayoutFixed),
		),
		400, 150,
	)
	AssertScene(t, scene, "testdata/table_fixed.golden")
}
