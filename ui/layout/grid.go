package layout

import (
	"github.com/timzifer/lux/ui"
)

// ── Track sizing ─────────────────────────────────────────────────

// TrackSizeType identifies the kind of track sizing.
type TrackSizeType int

const (
	TrackFixed  TrackSizeType = iota // Fixed dp value
	TrackFr                          // Fractional unit
	TrackAuto                        // Size to content
	TrackMinmax                      // minmax(min, max)
)

// TrackSize defines the size of a single grid track (column or row).
type TrackSize struct {
	Type  TrackSizeType
	Value float32 // dp for Fixed, fraction for Fr
	Min   float32 // min dp for Minmax
	Max   float32 // max dp for Minmax (0 = unlimited)
}

// Px returns a fixed-size track in dp.
func Px(v float32) TrackSize { return TrackSize{Type: TrackFixed, Value: v} }

// Fr returns a fractional track unit.
func Fr(v float32) TrackSize { return TrackSize{Type: TrackFr, Value: v} }

// AutoTrack returns an auto-sized track.
func AutoTrack() TrackSize { return TrackSize{Type: TrackAuto} }

// Minmax returns a track with minimum and maximum constraints.
func Minmax(min, max float32) TrackSize {
	return TrackSize{Type: TrackMinmax, Min: min, Max: max}
}

// ── Grid auto-flow ───────────────────────────────────────────────

// GridAutoFlow controls the auto-placement algorithm.
type GridAutoFlow int

const (
	GridFlowRow    GridAutoFlow = iota // Fill rows first (default)
	GridFlowColumn                     // Fill columns first
	GridFlowDense                      // Fill holes greedily
)

// ── Grid item placement ──────────────────────────────────────────

// GridItem wraps a child element with explicit grid placement.
type GridItem struct {
	ui.BaseElement
	Child    ui.Element
	ColStart int // 1-based column start; 0 = auto
	ColEnd   int // 1-based exclusive column end; 0 = auto (colStart+1)
	RowStart int // 1-based row start; 0 = auto
	RowEnd   int // 1-based exclusive row end; 0 = auto (rowStart+1)
}

// GridItemOption configures a GridItem.
type GridItemOption func(*GridItem)

// PlaceGridItem creates a GridItem with the given options.
func PlaceGridItem(child ui.Element, opts ...GridItemOption) ui.Element {
	item := GridItem{Child: child}
	for _, opt := range opts {
		opt(&item)
	}
	return item
}

// AtCol places the item at a specific column (1-based).
func AtCol(start int, end ...int) GridItemOption {
	return func(g *GridItem) {
		g.ColStart = start
		if len(end) > 0 {
			g.ColEnd = end[0]
		}
	}
}

// AtRow places the item at a specific row (1-based).
func AtRow(start int, end ...int) GridItemOption {
	return func(g *GridItem) {
		g.RowStart = start
		if len(end) > 0 {
			g.RowEnd = end[0]
		}
	}
}

// ColSpan sets the column span from a start column.
func ColSpan(start, span int) GridItemOption {
	return func(g *GridItem) {
		g.ColStart = start
		g.ColEnd = start + span
	}
}

// RowSpan sets the row span from a start row.
func RowSpan(start, span int) GridItemOption {
	return func(g *GridItem) {
		g.RowStart = start
		g.RowEnd = start + span
	}
}

func (n GridItem) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ctx.LayoutChild(n.Child, ctx.Area)
}
func (n GridItem) TreeEqual(other ui.Element) bool {
	o, ok := other.(GridItem)
	return ok && n.ColStart == o.ColStart && n.ColEnd == o.ColEnd &&
		n.RowStart == o.RowStart && n.RowEnd == o.RowEnd
}
func (n GridItem) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	out.Child = resolve(n.Child, 0)
	return out
}
func (n GridItem) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}

// ── Grid container ───────────────────────────────────────────────

// GridOption configures a Grid element.
type GridOption func(*Grid)

// Grid is a grid layout container implementing CSS Grid semantics.
type Grid struct {
	ui.BaseElement
	Columns      int          // Legacy: uniform column count (used when TemplateCols is nil)
	TemplateCols []TrackSize  // grid-template-columns
	TemplateRows []TrackSize  // grid-template-rows
	AutoFlow     GridAutoFlow // grid-auto-flow
	RowGap       float32
	ColGap       float32
	JustifyItems Align // Horizontal alignment of cell content
	AlignItems   Align // Vertical alignment of cell content
	Children     []ui.Element
}

// NewGrid creates a Grid layout with the given number of columns.
// When columns > 0 and no TemplateCols is set, uniform column sizing is used.
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

// NewTemplateGrid creates a Grid with explicit column track definitions.
func NewTemplateGrid(cols []TrackSize, children []ui.Element, opts ...GridOption) ui.Element {
	el := Grid{TemplateCols: cols, Children: children}
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

// WithAutoFlow sets the auto-placement algorithm.
func WithAutoFlow(flow GridAutoFlow) GridOption {
	return func(e *Grid) { e.AutoFlow = flow }
}

// WithTemplateCols sets explicit column track sizes.
func WithTemplateCols(cols ...TrackSize) GridOption {
	return func(e *Grid) { e.TemplateCols = cols }
}

// WithTemplateRows sets explicit row track sizes.
func WithTemplateRows(rows ...TrackSize) GridOption {
	return func(e *Grid) { e.TemplateRows = rows }
}

// WithJustifyItems sets the default horizontal alignment of cell content.
func WithJustifyItems(a Align) GridOption {
	return func(e *Grid) { e.JustifyItems = a }
}

// WithAlignItems sets the default vertical alignment of cell content.
func WithAlignItems(a Align) GridOption {
	return func(e *Grid) { e.AlignItems = a }
}

// LayoutSelf implements ui.Layouter.
func (n Grid) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	nn := len(n.Children)
	if nn == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	colGap := int(n.ColGap)
	rowGap := int(n.RowGap)

	// Determine number of columns and resolve track sizes.
	cols := n.resolveColCount()
	if cols < 1 {
		cols = 1
	}

	// Place items into the grid.
	placements := make([]gridPlacement, 0, nn)
	occupied := map[[2]int]bool{} // [col,row] -> occupied

	// First pass: place explicitly positioned items.
	autoItems := make([]int, 0, nn)
	for i, child := range n.Children {
		if gi, ok := child.(GridItem); ok {
			p := gridPlacement{child: gi.Child}
			if gi.ColStart > 0 {
				p.col = gi.ColStart - 1
			}
			if gi.RowStart > 0 {
				p.row = gi.RowStart - 1
			}
			p.colSpan = 1
			if gi.ColEnd > gi.ColStart && gi.ColStart > 0 {
				p.colSpan = gi.ColEnd - gi.ColStart
			}
			p.rowSpan = 1
			if gi.RowEnd > gi.RowStart && gi.RowStart > 0 {
				p.rowSpan = gi.RowEnd - gi.RowStart
			}
			placements = append(placements, p)
			// Mark cells as occupied.
			for r := p.row; r < p.row+p.rowSpan; r++ {
				for c := p.col; c < p.col+p.colSpan; c++ {
					occupied[[2]int{c, r}] = true
				}
			}
		} else {
			autoItems = append(autoItems, i)
			placements = append(placements, gridPlacement{child: child, col: -1, row: -1, colSpan: 1, rowSpan: 1})
		}
	}

	// Second pass: auto-place remaining items.
	autoCursor := [2]int{0, 0} // [col, row]
	for _, idx := range autoItems {
		pi := -1
		for j := range placements {
			if placements[j].child == n.Children[idx] && placements[j].col == -1 {
				pi = j
				break
			}
		}
		if pi < 0 {
			continue
		}

		if n.AutoFlow == GridFlowColumn {
			// Column-first auto-placement.
			for {
				if !occupied[autoCursor] {
					break
				}
				autoCursor[1]++
				if autoCursor[1] >= 10000 { // safety limit
					autoCursor[0]++
					autoCursor[1] = 0
				}
			}
			placements[pi].col = autoCursor[0]
			placements[pi].row = autoCursor[1]
			occupied[autoCursor] = true
			autoCursor[1]++
		} else {
			// Row-first auto-placement (default).
			for {
				if !occupied[autoCursor] {
					break
				}
				autoCursor[0]++
				if autoCursor[0] >= cols {
					autoCursor[0] = 0
					autoCursor[1]++
				}
			}
			placements[pi].col = autoCursor[0]
			placements[pi].row = autoCursor[1]
			occupied[autoCursor] = true
			autoCursor[0]++
			if autoCursor[0] >= cols {
				autoCursor[0] = 0
				autoCursor[1]++
			}
		}
	}

	// Determine actual number of rows.
	rows := 0
	for _, p := range placements {
		endRow := p.row + p.rowSpan
		if endRow > rows {
			rows = endRow
		}
	}
	if rows < 1 {
		rows = 1
	}

	// Resolve column widths.
	colWidths := n.resolveColWidths(cols, area.W, colGap, ctx)
	// Resolve row heights.
	rowHeights := n.resolveRowHeights(rows, rowGap, area.H, colWidths, colGap, ctx, placements)

	// Compute column/row positions.
	colPositions := make([]int, cols)
	cursor := 0
	for c := 0; c < cols; c++ {
		colPositions[c] = cursor
		cursor += colWidths[c]
		if c < cols-1 {
			cursor += colGap
		}
	}
	rowPositions := make([]int, rows)
	cursor = 0
	for r := 0; r < rows; r++ {
		rowPositions[r] = cursor
		cursor += rowHeights[r]
		if r < rows-1 {
			cursor += rowGap
		}
	}

	// Paint pass.
	totalW := 0
	if cols > 0 {
		totalW = colPositions[cols-1] + colWidths[cols-1]
	}
	totalH := 0
	if rows > 0 {
		totalH = rowPositions[rows-1] + rowHeights[rows-1]
	}

	for _, p := range placements {
		if p.col < 0 || p.row < 0 || p.col >= cols || p.row >= rows {
			continue
		}
		cellX := colPositions[p.col]
		cellY := rowPositions[p.row]
		cellW := 0
		for c := p.col; c < p.col+p.colSpan && c < cols; c++ {
			cellW += colWidths[c]
			if c > p.col {
				cellW += colGap
			}
		}
		cellH := 0
		for r := p.row; r < p.row+p.rowSpan && r < rows; r++ {
			cellH += rowHeights[r]
			if r > p.row {
				cellH += rowGap
			}
		}

		// Apply cell alignment.
		childArea := n.alignInCell(ctx, p.child, area.X+cellX, area.Y+cellY, cellW, cellH)
		ctx.LayoutChild(p.child, childArea)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// resolveColCount returns the effective number of columns.
func (n Grid) resolveColCount() int {
	if len(n.TemplateCols) > 0 {
		return len(n.TemplateCols)
	}
	if n.Columns > 0 {
		return n.Columns
	}
	return 1
}

// resolveColWidths resolves column widths from template or uniform sizing.
func (n Grid) resolveColWidths(cols, availW, colGap int, ctx *ui.LayoutContext) []int {
	totalGaps := 0
	if cols > 1 {
		totalGaps = colGap * (cols - 1)
	}
	usable := availW - totalGaps
	if usable < 0 {
		usable = 0
	}

	if len(n.TemplateCols) == 0 {
		// Legacy uniform columns.
		cellW := usable / cols
		widths := make([]int, cols)
		for i := range widths {
			widths[i] = cellW
		}
		return widths
	}

	widths := make([]int, cols)
	totalFr := float32(0)
	fixedUsed := 0

	// First pass: resolve fixed and auto tracks, sum fr units.
	for i, ts := range n.TemplateCols {
		if i >= cols {
			break
		}
		switch ts.Type {
		case TrackFixed:
			widths[i] = int(ts.Value)
			fixedUsed += widths[i]
		case TrackMinmax:
			widths[i] = int(ts.Min)
			fixedUsed += widths[i]
		case TrackAuto:
			// Will be sized to content later; start at 0.
			widths[i] = 0
		case TrackFr:
			totalFr += ts.Value
		}
	}

	// Auto tracks: measure content to find natural width.
	// (simplified: measure all children in that column, take max)
	for i, ts := range n.TemplateCols {
		if i >= cols || ts.Type != TrackAuto {
			continue
		}
		// Auto-sized: we'll give it a fair share for now, then adjust.
		// A full implementation would measure each child in the column.
		widths[i] = 0
	}

	// Fr tracks: distribute remaining space.
	remaining := usable - fixedUsed
	if remaining < 0 {
		remaining = 0
	}
	if totalFr > 0 {
		for i, ts := range n.TemplateCols {
			if i >= cols || ts.Type != TrackFr {
				continue
			}
			widths[i] = int(float32(remaining) * ts.Value / totalFr)
		}
	}

	// Minmax clamping: ensure max is respected.
	for i, ts := range n.TemplateCols {
		if i >= cols || ts.Type != TrackMinmax {
			continue
		}
		if ts.Max > 0 && float32(widths[i]) > ts.Max {
			widths[i] = int(ts.Max)
		}
		// Give minmax tracks a share of remaining space up to max.
		frRemaining := remaining
		for j, ts2 := range n.TemplateCols {
			if j >= cols {
				break
			}
			if ts2.Type == TrackFr {
				frRemaining -= widths[j]
			}
		}
		if frRemaining > 0 && widths[i] < int(ts.Max) {
			extra := frRemaining
			if ts.Max > 0 && int(ts.Max)-widths[i] < extra {
				extra = int(ts.Max) - widths[i]
			}
			widths[i] += extra
		}
	}

	return widths
}

type gridPlacement struct {
	col, row         int
	colSpan, rowSpan int
	child            ui.Element
}

// resolveRowHeights resolves row heights, measuring children as needed.
func (n Grid) resolveRowHeights(rows, rowGap, availH int, colWidths []int, colGap int, ctx *ui.LayoutContext, placements []gridPlacement) []int {
	heights := make([]int, rows)

	// If TemplateRows is set, apply it.
	totalFr := float32(0)
	fixedUsed := 0
	for i := 0; i < rows && i < len(n.TemplateRows); i++ {
		ts := n.TemplateRows[i]
		switch ts.Type {
		case TrackFixed:
			heights[i] = int(ts.Value)
			fixedUsed += heights[i]
		case TrackFr:
			totalFr += ts.Value
		case TrackMinmax:
			heights[i] = int(ts.Min)
			fixedUsed += heights[i]
		}
	}

	// Measure children to determine auto/content row heights.
	cols := len(colWidths)
	for _, p := range placements {
		if p.row < 0 || p.row >= rows || p.col < 0 || p.col >= cols {
			continue
		}
		// Skip rows with explicit fixed/fr sizes (fr resolved below).
		if p.row < len(n.TemplateRows) && n.TemplateRows[p.row].Type == TrackFixed {
			continue
		}

		cellW := 0
		for c := p.col; c < p.col+p.colSpan && c < cols; c++ {
			cellW += colWidths[c]
			if c > p.col {
				cellW += colGap
			}
		}
		cb := ctx.MeasureChild(p.child, ui.Bounds{W: cellW, H: availH})
		for r := p.row; r < p.row+p.rowSpan && r < rows; r++ {
			rh := cb.H / p.rowSpan
			if rh > heights[r] {
				heights[r] = rh
			}
		}
	}

	// Distribute fr rows.
	if totalFr > 0 {
		totalRowGaps := 0
		if rows > 1 {
			totalRowGaps = rowGap * (rows - 1)
		}
		measured := 0
		for _, h := range heights {
			measured += h
		}
		remaining := availH - totalRowGaps - measured
		if remaining < 0 {
			remaining = 0
		}
		for i := 0; i < rows && i < len(n.TemplateRows); i++ {
			if n.TemplateRows[i].Type == TrackFr {
				heights[i] = int(float32(remaining) * n.TemplateRows[i].Value / totalFr)
			}
		}
	}

	return heights
}

// alignInCell applies JustifyItems/AlignItems alignment within a grid cell.
func (n Grid) alignInCell(ctx *ui.LayoutContext, child ui.Element, x, y, w, h int) ui.Bounds {
	if n.JustifyItems == AlignStart && n.AlignItems == AlignStart {
		return ui.Bounds{X: x, Y: y, W: w, H: h}
	}

	cb := ctx.MeasureChild(child, ui.Bounds{X: x, Y: y, W: w, H: h})

	cx, cy, cw, ch := x, y, cb.W, cb.H

	switch n.JustifyItems {
	case AlignEnd:
		cx = x + w - cw
	case AlignCenter:
		cx = x + (w-cw)/2
	case AlignStretch:
		cw = w
	}

	switch n.AlignItems {
	case AlignEnd:
		cy = y + h - ch
	case AlignCenter:
		cy = y + (h-ch)/2
	case AlignStretch:
		ch = h
	}

	return ui.Bounds{X: cx, Y: cy, W: cw, H: ch}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Grid) TreeEqual(other ui.Element) bool {
	o, ok := other.(Grid)
	if !ok {
		return false
	}
	if n.Columns != o.Columns || n.RowGap != o.RowGap || n.ColGap != o.ColGap ||
		n.AutoFlow != o.AutoFlow || n.JustifyItems != o.JustifyItems || n.AlignItems != o.AlignItems ||
		len(n.Children) != len(o.Children) || len(n.TemplateCols) != len(o.TemplateCols) ||
		len(n.TemplateRows) != len(o.TemplateRows) {
		return false
	}
	return true
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
