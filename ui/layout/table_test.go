//go:build nogui

package layout_test

import (
	"testing"

	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

// ── Table: basic tests ──────────────────────────────────────────

func TestTableEmpty(t *testing.T) {
	tbl := layout.NewTable(nil)
	buildScene(tbl, 800, 600)
}

func TestTableSingleCell(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(100, 40))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableUniformGrid(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableVaryingContent(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(50, 20)), layout.TD(sizedBox(100, 40))),
		layout.TR(layout.TD(sizedBox(80, 60)), layout.TD(sizedBox(30, 10))),
	})
	buildScene(tbl, 800, 600)
}

// ── Table: column width tests ───────────────────────────────────

func TestTableAutoLayout(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(50, 30)), layout.TD(sizedBox(100, 30)), layout.TD(sizedBox(75, 30))),
		layout.TR(layout.TD(sizedBox(60, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(90, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableFixedLayout(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(100, 30)), layout.TD(sizedBox(200, 30))),
		layout.TR(layout.TD(sizedBox(50, 30)), layout.TD(sizedBox(50, 30))),
	}, layout.WithTableLayout(layout.TableLayoutFixed))
	buildScene(tbl, 800, 600)
}

func TestTableColGroup(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.NewTableColGroup(
			layout.Col(layout.Px(100)),
			layout.Col(layout.Px(200)),
			layout.Col(layout.Px(150)),
		),
		layout.TR(layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30))),
		layout.TR(layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30))),
	}, layout.WithTableLayout(layout.TableLayoutFixed))
	buildScene(tbl, 800, 600)
}

func TestTableMixedColumnWidths(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.NewTableColGroup(
			layout.Col(layout.Px(100)),
			layout.Col(layout.AutoTrack()),
		),
		layout.TR(layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(120, 30))),
	}, layout.WithTableLayout(layout.TableLayoutFixed))
	buildScene(tbl, 800, 600)
}

// ── Table: spanning tests ───────────────────────────────────────

func TestTableColSpan(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(200, 30), layout.WithColSpan(2)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableRowSpan(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 80), layout.WithRowSpan(2)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableComplexSpanning(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(
			layout.TD(sizedBox(160, 60), layout.WithColSpan(2), layout.WithRowSpan(2)),
			layout.TD(sizedBox(80, 30)),
		),
		layout.TR(layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

// ── Table: section tests ────────────────────────────────────────

func TestTableWithSections(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.THead(
			layout.TR(layout.TH(sizedBox(80, 30)), layout.TH(sizedBox(80, 30))),
		),
		layout.TBody(
			layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
			layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		),
		layout.TFoot(
			layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		),
	})
	buildScene(tbl, 800, 600)
}

func TestTableHeaderFooter(t *testing.T) {
	// Foot defined before body — should still render in correct order.
	tbl := layout.NewTable([]ui.Element{
		layout.TFoot(layout.TR(layout.TD(sizedBox(80, 25)))),
		layout.THead(layout.TR(layout.TH(sizedBox(80, 35)))),
		layout.TBody(layout.TR(layout.TD(sizedBox(80, 30)))),
	})
	buildScene(tbl, 800, 600)
}

// ── Table: border spacing tests ─────────────────────────────────

func TestTableBorderSeparate(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	}, layout.WithBorderSpacing(8, 8))
	buildScene(tbl, 800, 600)
}

func TestTableBorderCollapsed(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	}, layout.WithBorderCollapse(layout.BorderCollapsed))
	buildScene(tbl, 800, 600)
}

// ── Table: caption tests ────────────────────────────────────────

func TestTableCaptionTop(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.NewTableCaption(sizedBox(200, 30)),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableCaptionBottom(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.NewTableCaption(sizedBox(200, 30)),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	}, layout.WithCaptionSide(layout.CaptionBottom))
	buildScene(tbl, 800, 600)
}

// ── Table: edge cases ───────────────────────────────────────────

func TestTableSingleColumn(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableSingleRow(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

func TestTableMismatchedRowLengths(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30))),
		layout.TR(layout.TD(sizedBox(80, 30)), layout.TD(sizedBox(80, 30))),
	})
	buildScene(tbl, 800, 600)
}

// ── Table: SimpleTable convenience ──────────────────────────────

func TestSimpleTable(t *testing.T) {
	tbl := layout.SimpleTable(
		[]ui.Element{sizedBox(80, 30), sizedBox(80, 30), sizedBox(80, 30)},
		[][]ui.Element{
			{sizedBox(80, 30), sizedBox(80, 30), sizedBox(80, 30)},
			{sizedBox(80, 30), sizedBox(80, 30), sizedBox(80, 30)},
		},
	)
	buildScene(tbl, 800, 600)
}

func TestSimpleTableWithOptions(t *testing.T) {
	tbl := layout.SimpleTable(
		[]ui.Element{sizedBox(80, 30), sizedBox(80, 30)},
		[][]ui.Element{
			{sizedBox(80, 30), sizedBox(80, 30)},
		},
		layout.WithBorderSpacing(4, 4),
		layout.WithTableLayout(layout.TableLayoutFixed),
	)
	buildScene(tbl, 800, 600)
}

// ── Table: vertical alignment ───────────────────────────────────

func TestTableVAlignMiddle(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(
			layout.TD(sizedBox(80, 20), layout.WithVAlign(layout.VAlignMiddle)),
			layout.TD(sizedBox(80, 60)),
		),
	})
	buildScene(tbl, 800, 600)
}

func TestTableVAlignBottom(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.TR(
			layout.TD(sizedBox(80, 20), layout.WithVAlign(layout.VAlignBottom)),
			layout.TD(sizedBox(80, 60)),
		),
	})
	buildScene(tbl, 800, 600)
}

// ── Table: col span with colgroup ───────────────────────────────

func TestTableColGroupWithSpan(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.NewTableColGroup(
			layout.Col(layout.Px(100), 2),
			layout.Col(layout.Px(200)),
		),
		layout.TR(layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30)), layout.TD(sizedBox(0, 30))),
	}, layout.WithTableLayout(layout.TableLayoutFixed))
	buildScene(tbl, 800, 600)
}

// ── Table: with display.Text content ────────────────────────────

func TestTableWithTextContent(t *testing.T) {
	tbl := layout.NewTable([]ui.Element{
		layout.THead(
			layout.TR(layout.TH(display.Text("Name")), layout.TH(display.Text("Age"))),
		),
		layout.TBody(
			layout.TR(layout.TD(display.Text("Alice")), layout.TD(display.Text("30"))),
			layout.TR(layout.TD(display.Text("Bob")), layout.TD(display.Text("25"))),
		),
	})
	buildScene(tbl, 800, 600)
}
