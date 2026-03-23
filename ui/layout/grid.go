package layout

import (
	"github.com/timzifer/lux/ui"
)

// GridOption configures a Grid element.
type GridOption func(*Grid)

// Grid is a uniform raster layout with a fixed number of columns.
type Grid struct {
	ui.BaseElement
	Columns  int
	RowGap   float32
	ColGap   float32
	Children []ui.Element
}

// NewGrid creates a Grid layout with the given number of columns.
func NewGrid(columns int, children []ui.Element, opts ...GridOption) ui.Element {
	if columns < 1 {
		columns = 1
	}
	el := Grid{Columns: columns, Children: children}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// WithRowGap sets the vertical gap between grid rows.
func WithRowGap(gap float32) GridOption {
	return func(e *Grid) { e.RowGap = gap }
}

// WithColGap sets the horizontal gap between grid columns.
func WithColGap(gap float32) GridOption {
	return func(e *Grid) { e.ColGap = gap }
}

// LayoutSelf implements ui.Layouter.
func (n Grid) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	nn := len(n.Children)
	if nn == 0 || n.Columns < 1 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	cols := n.Columns
	colGap := int(n.ColGap)
	rowGap := int(n.RowGap)

	// Cell width: divide available width evenly among columns.
	totalColGaps := colGap * (cols - 1)
	cellW := (area.W - totalColGaps) / cols
	if cellW < 0 {
		cellW = 0
	}

	// Determine number of rows.
	rows := (nn + cols - 1) / cols

	// Pass 1: measure to find max height per row.
	rowHeights := make([]int, rows)
	for i, child := range n.Children {
		row := i / cols
		col := i % cols
		cellX := area.X + col*(cellW+colGap)
		cb := ctx.MeasureChild(child, ui.Bounds{X: cellX, Y: 0, W: cellW, H: area.H})
		if cb.H > rowHeights[row] {
			rowHeights[row] = cb.H
		}
	}

	// Pass 2: paint at computed positions.
	cursorY := area.Y
	maxW := 0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= nn {
				break
			}
			cellX := area.X + col*(cellW+colGap)
			childArea := ui.Bounds{X: cellX, Y: cursorY, W: cellW, H: rowHeights[row]}
			ctx.LayoutChild(n.Children[idx], childArea)
		}
		rowW := cols*cellW + totalColGaps
		if rowW > maxW {
			maxW = rowW
		}
		cursorY += rowHeights[row]
		if row < rows-1 {
			cursorY += rowGap
		}
	}

	totalH := cursorY - area.Y
	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: totalH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Grid) TreeEqual(other ui.Element) bool {
	o, ok := other.(Grid)
	return ok && n.Columns == o.Columns && n.RowGap == o.RowGap && n.ColGap == o.ColGap && len(n.Children) == len(o.Children)
}

// ResolveChildren implements ui.ChildResolver.
func (n Grid) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Grid) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}
