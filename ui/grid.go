package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
)

// GridOption configures a Grid element.
type GridOption func(*gridElement)

// Grid creates a uniform raster layout with the given number of columns
// (RFC-002 §4.5).
func Grid(columns int, children []Element, opts ...GridOption) Element {
	if columns < 1 {
		columns = 1
	}
	el := gridElement{Columns: columns, Children: children}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// WithRowGap sets the vertical gap between grid rows.
func WithRowGap(gap float32) GridOption {
	return func(e *gridElement) { e.RowGap = gap }
}

// WithColGap sets the horizontal gap between grid columns.
func WithColGap(gap float32) GridOption {
	return func(e *gridElement) { e.ColGap = gap }
}

type gridElement struct {
	Columns  int
	RowGap   float32
	ColGap   float32
	Children []Element
}

func (gridElement) isElement() {}

func layoutGrid(node gridElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusManager) bounds {
	n := len(node.Children)
	if n == 0 || node.Columns < 1 {
		return bounds{X: area.X, Y: area.Y}
	}

	cols := node.Columns
	colGap := int(node.ColGap)
	rowGap := int(node.RowGap)

	// Cell width: divide available width evenly among columns.
	totalColGaps := colGap * (cols - 1)
	cellW := (area.W - totalColGaps) / cols
	if cellW < 0 {
		cellW = 0
	}

	// Determine number of rows.
	rows := (n + cols - 1) / cols

	// Pass 1: measure to find max height per row.
	nc := nullCanvas{delegate: canvas}
	rowHeights := make([]int, rows)
	for i, child := range node.Children {
		row := i / cols
		col := i % cols
		cellX := area.X + col*(cellW+colGap)
		cellY := 0 // doesn't matter for measurement
		cb := layoutElement(child, bounds{X: cellX, Y: cellY, W: cellW, H: area.H}, nc, th, tokens, nil, nil, nil)
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
			if idx >= n {
				break
			}
			cellX := area.X + col*(cellW+colGap)
			childArea := bounds{X: cellX, Y: cursorY, W: cellW, H: rowHeights[row]}
			layoutElement(node.Children[idx], childArea, canvas, th, tokens, hitMap, hover, overlays, focus)
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
	return bounds{X: area.X, Y: area.Y, W: maxW, H: totalH}
}
