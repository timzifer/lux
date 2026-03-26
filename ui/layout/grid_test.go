//go:build nogui

package layout_test

import (
	"testing"

	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
)

// ── Grid: legacy uniform columns ────────────────────────────────

func TestGridUniformColumns(t *testing.T) {
	grid := layout.NewGrid(3, []ui.Element{
		sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30),
		sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30),
	}, layout.WithColGap(8), layout.WithRowGap(8))
	buildScene(grid, 400, 300)
}

func TestGridBackwardCompat(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40),
	})
	buildScene(grid, 300, 200)
}

func TestGridEmpty(t *testing.T) {
	grid := layout.NewGrid(3, nil)
	buildScene(grid, 400, 300)
}

func TestGridSingleChild(t *testing.T) {
	grid := layout.NewGrid(3, []ui.Element{sizedBox(80, 40)})
	buildScene(grid, 400, 300)
}

// ── Grid: template columns ──────────────────────────────────────

func TestGridTemplateFixedColumns(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Px(100), layout.Px(200), layout.Px(100)},
		[]ui.Element{sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40)})
	buildScene(grid, 500, 200)
}

func TestGridTemplateFrColumns(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Fr(1), layout.Fr(2), layout.Fr(1)},
		[]ui.Element{sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40)},
		layout.WithColGap(8))
	buildScene(grid, 400, 200)
}

func TestGridTemplateMixed(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Px(100), layout.Fr(1), layout.AutoTrack()},
		[]ui.Element{sizedBox(0, 40), sizedBox(0, 40), sizedBox(80, 40)},
		layout.WithColGap(12))
	buildScene(grid, 500, 200)
}

func TestGridTemplateMinmax(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Minmax(100, 200), layout.Fr(1)},
		[]ui.Element{sizedBox(0, 40), sizedBox(0, 40)})
	buildScene(grid, 500, 200)
}

// ── Grid: template rows ─────────────────────────────────────────

func TestGridTemplateRows(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Fr(1), layout.Fr(1)},
		[]ui.Element{sizedBox(0, 0), sizedBox(0, 0), sizedBox(0, 0), sizedBox(0, 0)},
		layout.WithTemplateRows(layout.Px(50), layout.Px(100)))
	buildScene(grid, 400, 200)
}

// ── Grid: item placement ────────────────────────────────────────

func TestGridItemPlacement(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Fr(1), layout.Fr(1), layout.Fr(1)},
		[]ui.Element{
			layout.PlaceGridItem(sizedBox(0, 40), layout.AtCol(2), layout.AtRow(1)),
			layout.PlaceGridItem(sizedBox(0, 40), layout.AtCol(1), layout.AtRow(2)),
			sizedBox(0, 40),
		})
	buildScene(grid, 400, 200)
}

func TestGridItemSpan(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Fr(1), layout.Fr(1), layout.Fr(1)},
		[]ui.Element{
			layout.PlaceGridItem(sizedBox(0, 40), layout.ColSpan(1, 2)),
			sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40),
		})
	buildScene(grid, 400, 200)
}

// ── Grid: auto-flow ─────────────────────────────────────────────

func TestGridAutoFlowRow(t *testing.T) {
	grid := layout.NewGrid(3, []ui.Element{
		sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30),
	}, layout.WithAutoFlow(layout.GridFlowRow))
	buildScene(grid, 400, 200)
}

func TestGridAutoFlowColumn(t *testing.T) {
	grid := layout.NewGrid(3, []ui.Element{
		sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30), sizedBox(0, 30),
	}, layout.WithAutoFlow(layout.GridFlowColumn))
	buildScene(grid, 400, 200)
}

// ── Grid: cell alignment ────────────────────────────────────────

func TestGridJustifyItems(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(50, 30), sizedBox(50, 30),
	}, layout.WithJustifyItems(layout.AlignCenter))
	buildScene(grid, 400, 200)
}

func TestGridAlignItems(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(50, 20), sizedBox(50, 40),
	}, layout.WithAlignItems(layout.AlignCenter))
	buildScene(grid, 400, 200)
}

func TestGridAlignItemsEnd(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(50, 20), sizedBox(50, 40),
	}, layout.WithAlignItems(layout.AlignEnd))
	buildScene(grid, 400, 200)
}

func TestGridAlignItemsStretch(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(50, 20), sizedBox(50, 40),
	}, layout.WithAlignItems(layout.AlignStretch))
	buildScene(grid, 400, 200)
}

// ── Grid: gaps ──────────────────────────────────────────────────

func TestGridGaps(t *testing.T) {
	grid := layout.NewGrid(2, []ui.Element{
		sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40), sizedBox(0, 40),
	}, layout.WithRowGap(16), layout.WithColGap(24))
	buildScene(grid, 400, 300)
}

// ── Grid: fr rows ───────────────────────────────────────────────

func TestGridFrRows(t *testing.T) {
	grid := layout.NewTemplateGrid(
		[]layout.TrackSize{layout.Fr(1), layout.Fr(1)},
		[]ui.Element{sizedBox(0, 0), sizedBox(0, 0), sizedBox(0, 0), sizedBox(0, 0)},
		layout.WithTemplateRows(layout.Fr(1), layout.Fr(2)),
		layout.WithRowGap(8))
	buildScene(grid, 400, 300)
}
