package layout

import (
	"github.com/timzifer/lux/ui"
)

// ── Table layout mode ───────────────────────────────────────────

// TableLayoutMode selects the column-width algorithm (CSS table-layout).
type TableLayoutMode int

const (
	TableLayoutAuto  TableLayoutMode = iota // auto: content-driven widths
	TableLayoutFixed                        // fixed: first-row / col-definition widths
)

// ── Border collapse ─────────────────────────────────────────────

// BorderCollapse selects the border model (CSS border-collapse).
type BorderCollapse int

const (
	BorderSeparate  BorderCollapse = iota // separate: border-spacing applies
	BorderCollapsed                       // collapse: zero spacing, borders merge
)

// ── Caption side ────────────────────────────────────────────────

// CaptionSide controls caption placement (CSS caption-side).
type CaptionSide int

const (
	CaptionTop    CaptionSide = iota // caption above the table grid
	CaptionBottom                     // caption below the table grid
)

// ── Section type ────────────────────────────────────────────────

// SectionType identifies a table row-group kind.
type SectionType int

const (
	SectionHead SectionType = iota // thead
	SectionBody                    // tbody
	SectionFoot                    // tfoot
)

// ── Vertical alignment ─────────────────────────────────────────

// VAlign controls vertical alignment inside a table cell.
type VAlign int

const (
	VAlignTop      VAlign = iota // align to top (default)
	VAlignMiddle                 // center vertically
	VAlignBottom                 // align to bottom
	VAlignBaseline               // align to baseline
)

// ── TableCol ────────────────────────────────────────────────────

// TableCol defines a column template with an optional width and span.
type TableCol struct {
	ui.BaseElement
	Width TrackSize // reuse grid TrackSize (Px, Fr, Auto, Minmax)
	Span  int       // number of columns this definition covers (default 1)
}

// Col creates a column definition.
func Col(width TrackSize, span ...int) TableCol {
	s := 1
	if len(span) > 0 && span[0] > 1 {
		s = span[0]
	}
	return TableCol{Width: width, Span: s}
}

func (n TableCol) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}
func (n TableCol) TreeEqual(other ui.Element) bool {
	o, ok := other.(TableCol)
	return ok && n.Width == o.Width && n.Span == o.Span
}
func (n TableCol) ResolveChildren(_ func(ui.Element, int) ui.Element) ui.Element { return n }
func (n TableCol) WalkAccess(_ *ui.AccessTreeBuilder, _ int32)                   {}

// ── TableColGroup ───────────────────────────────────────────────

// TableColGroup groups column definitions (display: table-column-group).
type TableColGroup struct {
	ui.BaseElement
	Columns []TableCol
}

// NewTableColGroup creates a column group.
func NewTableColGroup(cols ...TableCol) ui.Element {
	return TableColGroup{Columns: cols}
}

func (n TableColGroup) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
}
func (n TableColGroup) TreeEqual(other ui.Element) bool {
	o, ok := other.(TableColGroup)
	if !ok || len(n.Columns) != len(o.Columns) {
		return false
	}
	for i := range n.Columns {
		if !n.Columns[i].TreeEqual(o.Columns[i]) {
			return false
		}
	}
	return true
}
func (n TableColGroup) ResolveChildren(_ func(ui.Element, int) ui.Element) ui.Element { return n }
func (n TableColGroup) WalkAccess(_ *ui.AccessTreeBuilder, _ int32)                   {}

// ── TableCaption ────────────────────────────────────────────────

// TableCaption holds the table caption element (display: table-caption).
type TableCaption struct {
	ui.BaseElement
	Child ui.Element
}

// NewTableCaption creates a table caption.
func NewTableCaption(child ui.Element) ui.Element {
	return TableCaption{Child: child}
}

func (n TableCaption) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if n.Child == nil {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}
	return ctx.LayoutChild(n.Child, ctx.Area)
}
func (n TableCaption) TreeEqual(other ui.Element) bool {
	_, ok := other.(TableCaption)
	return ok
}
func (n TableCaption) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	if n.Child != nil {
		out.Child = resolve(n.Child, 0)
	}
	return out
}
func (n TableCaption) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}

// ── TableCell ───────────────────────────────────────────────────

// CellOption configures a TableCell.
type CellOption func(*TableCell)

// TableCell represents a single cell (display: table-cell).
type TableCell struct {
	ui.BaseElement
	ColSpan int        // columns spanned (default 1)
	RowSpan int        // rows spanned (default 1)
	VAlign  VAlign     // vertical alignment within the cell
	IsHead  bool       // true for th (header cell)
	Child   ui.Element // cell content
}

// NewTableCell creates a table cell.
func NewTableCell(child ui.Element, opts ...CellOption) ui.Element {
	c := TableCell{Child: child, ColSpan: 1, RowSpan: 1}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// TD creates a data cell (display: table-cell).
func TD(child ui.Element, opts ...CellOption) ui.Element {
	return NewTableCell(child, opts...)
}

// TH creates a header cell (display: table-cell, th semantics).
func TH(child ui.Element, opts ...CellOption) ui.Element {
	opts = append([]CellOption{func(c *TableCell) { c.IsHead = true }}, opts...)
	return NewTableCell(child, opts...)
}

// WithColSpan sets the column span.
func WithColSpan(n int) CellOption {
	return func(c *TableCell) {
		if n < 1 {
			n = 1
		}
		c.ColSpan = n
	}
}

// WithRowSpan sets the row span.
func WithRowSpan(n int) CellOption {
	return func(c *TableCell) {
		if n < 1 {
			n = 1
		}
		c.RowSpan = n
	}
}

// WithVAlign sets the vertical alignment.
func WithVAlign(v VAlign) CellOption {
	return func(c *TableCell) { c.VAlign = v }
}

// When laid out outside a Table, a cell simply lays out its child.
func (n TableCell) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	if n.Child == nil {
		return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y}
	}
	return ctx.LayoutChild(n.Child, ctx.Area)
}
func (n TableCell) TreeEqual(other ui.Element) bool {
	o, ok := other.(TableCell)
	return ok && n.ColSpan == o.ColSpan && n.RowSpan == o.RowSpan &&
		n.VAlign == o.VAlign && n.IsHead == o.IsHead
}
func (n TableCell) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	out := n
	if n.Child != nil {
		out.Child = resolve(n.Child, 0)
	}
	return out
}
func (n TableCell) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	if n.Child != nil {
		b.Walk(n.Child, parentIdx)
	}
}

// ── TableRow ────────────────────────────────────────────────────

// TableRow represents a single row (display: table-row).
type TableRow struct {
	ui.BaseElement
	Children []ui.Element // TableCell elements
}

// NewTableRow creates a table row.
func NewTableRow(cells ...ui.Element) ui.Element {
	return TableRow{Children: cells}
}

// TR is an alias for NewTableRow.
func TR(cells ...ui.Element) ui.Element {
	return NewTableRow(cells...)
}

// When laid out outside a Table, a row lays out children sequentially.
func (n TableRow) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	x := area.X
	maxH := 0
	for _, child := range n.Children {
		cb := ctx.LayoutChild(child, ui.Bounds{X: x, Y: area.Y, W: area.W - (x - area.X), H: area.H})
		x += cb.W
		if cb.H > maxH {
			maxH = cb.H
		}
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: x - area.X, H: maxH}
}
func (n TableRow) TreeEqual(other ui.Element) bool {
	o, ok := other.(TableRow)
	return ok && len(n.Children) == len(o.Children)
}
func (n TableRow) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}
func (n TableRow) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}

// ── TableSection ────────────────────────────────────────────────

// TableSection represents a row group (thead/tbody/tfoot).
type TableSection struct {
	ui.BaseElement
	Type     SectionType  // SectionHead, SectionBody, SectionFoot
	Children []ui.Element // TableRow elements
}

// NewTableSection creates a table section.
func NewTableSection(typ SectionType, rows ...ui.Element) ui.Element {
	return TableSection{Type: typ, Children: rows}
}

// THead creates a header section.
func THead(rows ...ui.Element) ui.Element { return NewTableSection(SectionHead, rows...) }

// TBody creates a body section.
func TBody(rows ...ui.Element) ui.Element { return NewTableSection(SectionBody, rows...) }

// TFoot creates a footer section.
func TFoot(rows ...ui.Element) ui.Element { return NewTableSection(SectionFoot, rows...) }

func (n TableSection) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	y := area.Y
	maxW := 0
	for _, child := range n.Children {
		cb := ctx.LayoutChild(child, ui.Bounds{X: area.X, Y: y, W: area.W, H: area.H - (y - area.Y)})
		y += cb.H
		if cb.W > maxW {
			maxW = cb.W
		}
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: y - area.Y}
}
func (n TableSection) TreeEqual(other ui.Element) bool {
	o, ok := other.(TableSection)
	return ok && n.Type == o.Type && len(n.Children) == len(o.Children)
}
func (n TableSection) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}
func (n TableSection) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}

// ── Table ───────────────────────────────────────────────────────

// TableOption configures a Table element.
type TableOption func(*Table)

// Table is the top-level table container (display: table).
// It implements the CSS 2.1 §17 table layout algorithm.
type Table struct {
	ui.BaseElement
	Layout         TableLayoutMode // table-layout: auto | fixed
	BorderCollapse BorderCollapse  // border-collapse: separate | collapse
	BorderSpacingH float32         // horizontal border-spacing (dp)
	BorderSpacingV float32         // vertical border-spacing (dp)
	CaptionSide    CaptionSide    // caption-side: top | bottom
	Children       []ui.Element   // TableCaption, TableColGroup, TableSection, TableRow, TableCell
}

// NewTable creates a table from child elements (sections, rows, captions, col-groups).
func NewTable(children []ui.Element, opts ...TableOption) ui.Element {
	el := Table{Children: children}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// SimpleTable is a convenience constructor that builds a table from header labels
// and a 2D slice of row cells.
func SimpleTable(headers []ui.Element, rows [][]ui.Element, opts ...TableOption) ui.Element {
	children := make([]ui.Element, 0, 2+len(rows))

	if len(headers) > 0 {
		hcells := make([]ui.Element, len(headers))
		for i, h := range headers {
			hcells[i] = TH(h)
		}
		children = append(children, THead(TR(hcells...)))
	}

	bodyRows := make([]ui.Element, len(rows))
	for i, row := range rows {
		cells := make([]ui.Element, len(row))
		for j, cell := range row {
			cells[j] = TD(cell)
		}
		bodyRows[i] = TR(cells...)
	}
	children = append(children, TBody(bodyRows...))

	return NewTable(children, opts...)
}

// WithTableLayout sets the table layout algorithm.
func WithTableLayout(mode TableLayoutMode) TableOption {
	return func(t *Table) { t.Layout = mode }
}

// WithBorderCollapse sets the border collapse mode.
func WithBorderCollapse(bc BorderCollapse) TableOption {
	return func(t *Table) { t.BorderCollapse = bc }
}

// WithBorderSpacing sets horizontal and vertical border-spacing.
func WithBorderSpacing(h, v float32) TableOption {
	return func(t *Table) { t.BorderSpacingH = h; t.BorderSpacingV = v }
}

// WithCaptionSide sets the caption placement.
func WithCaptionSide(side CaptionSide) TableOption {
	return func(t *Table) { t.CaptionSide = side }
}

// ── Table: interface implementations ────────────────────────────

func (n Table) TreeEqual(other ui.Element) bool {
	o, ok := other.(Table)
	if !ok {
		return false
	}
	return n.Layout == o.Layout && n.BorderCollapse == o.BorderCollapse &&
		n.BorderSpacingH == o.BorderSpacingH && n.BorderSpacingV == o.BorderSpacingV &&
		n.CaptionSide == o.CaptionSide && len(n.Children) == len(o.Children)
}

func (n Table) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

func (n Table) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}

// ── Table: layout algorithm (CSS 2.1 §17) ──────────────────────

// tableModel is the normalized internal representation of a table.
type tableModel struct {
	caption  *TableCaption
	colDefs  []TableCol     // flattened column definitions
	sections []TableSection // ordered: head, body, foot
	rows     []TableRow     // all rows in order
}

// cellPlacement records the position of a cell in the grid.
type cellPlacement struct {
	cell     TableCell
	row, col int // 0-based grid position
}

// LayoutSelf implements ui.Layouter — the main table layout entry point.
func (n Table) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if len(n.Children) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Resolve effective border spacing.
	spacingH, spacingV := int(n.BorderSpacingH), int(n.BorderSpacingV)
	if n.BorderCollapse == BorderCollapsed {
		spacingH, spacingV = 0, 0
	}

	// Phase 1: normalize structure.
	model := n.buildModel()
	if len(model.rows) == 0 && model.caption == nil {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Build flat cell list with grid positions.
	numCols := n.resolveColumnCount(model)
	if numCols < 1 {
		numCols = 1
	}
	placements := n.assignCellPositions(model, numCols)
	numRows := 0
	for _, p := range placements {
		end := p.row + p.cell.RowSpan
		if end > numRows {
			numRows = end
		}
	}
	if numRows < len(model.rows) {
		numRows = len(model.rows)
	}

	// Phase 2: column widths.
	colWidths := n.resolveColumnWidths(ctx, model, placements, numCols, numRows, area.W, spacingH)

	// Table grid width.
	gridW := n.gridWidth(colWidths, numCols, spacingH)

	// Phase 3: row heights.
	rowHeights := n.resolveRowHeights(ctx, placements, colWidths, numCols, numRows, spacingH, area.H)

	// Phase 4 & 5: place caption and cells.
	curY := area.Y

	// Caption top.
	var captionBounds ui.Bounds
	if model.caption != nil && n.CaptionSide == CaptionTop {
		captionBounds = ctx.LayoutChild(model.caption.Child, ui.Bounds{X: area.X, Y: curY, W: gridW, H: area.H})
		curY += captionBounds.H
	}

	// Outer border-spacing for separate model (CSS 2.1 §17.6.1).
	if n.BorderCollapse == BorderSeparate {
		curY += spacingV
	}

	gridY := curY

	// Compute column and row positions.
	colPositions := make([]int, numCols)
	cx := 0
	if n.BorderCollapse == BorderSeparate {
		cx = spacingH
	}
	for c := 0; c < numCols; c++ {
		colPositions[c] = cx
		cx += colWidths[c]
		if c < numCols-1 {
			cx += spacingH
		}
	}

	rowPositions := make([]int, numRows)
	ry := 0
	for r := 0; r < numRows; r++ {
		rowPositions[r] = ry
		ry += rowHeights[r]
		if r < numRows-1 {
			ry += spacingV
		}
	}

	gridH := ry
	if n.BorderCollapse == BorderSeparate && numRows > 0 {
		gridH += spacingV // trailing spacing
	}

	// Paint cells.
	for _, p := range placements {
		cellX := colPositions[p.col]
		cellY := rowPositions[p.row]
		cellW := 0
		for c := p.col; c < p.col+p.cell.ColSpan && c < numCols; c++ {
			cellW += colWidths[c]
			if c > p.col {
				cellW += spacingH
			}
		}
		cellH := 0
		for r := p.row; r < p.row+p.cell.RowSpan && r < numRows; r++ {
			cellH += rowHeights[r]
			if r > p.row {
				cellH += spacingV
			}
		}

		childArea := n.alignCellContent(ctx, p.cell, area.X+cellX, gridY+cellY, cellW, cellH)
		ctx.LayoutChild(p.cell.Child, childArea)
	}

	curY = gridY + gridH

	// Caption bottom.
	if model.caption != nil && n.CaptionSide == CaptionBottom {
		captionBounds = ctx.LayoutChild(model.caption.Child, ui.Bounds{X: area.X, Y: curY, W: gridW, H: area.H - (curY - area.Y)})
		curY += captionBounds.H
	}

	totalW := gridW
	if n.BorderCollapse == BorderSeparate {
		totalW += spacingH // trailing spacing
	}
	totalH := curY - area.Y

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: totalH}
}

// ── Table: internal helpers ─────────────────────────────────────

// buildModel normalizes the children into a tableModel.
func (n Table) buildModel() tableModel {
	var m tableModel
	var headSections, bodySections, footSections []TableSection

	for _, child := range n.Children {
		switch c := child.(type) {
		case TableCaption:
			m.caption = &c
		case TableColGroup:
			for _, col := range c.Columns {
				m.colDefs = append(m.colDefs, col)
			}
		case TableSection:
			switch c.Type {
			case SectionHead:
				headSections = append(headSections, c)
			case SectionBody:
				bodySections = append(bodySections, c)
			case SectionFoot:
				footSections = append(footSections, c)
			}
		case TableRow:
			// Bare rows wrapped in implicit body.
			bodySections = append(bodySections, TableSection{Type: SectionBody, Children: []ui.Element{c}})
		}
	}

	// Order: head, body, foot (CSS 2.1 §17.2).
	m.sections = append(m.sections, headSections...)
	m.sections = append(m.sections, bodySections...)
	m.sections = append(m.sections, footSections...)

	// Flatten rows.
	for _, sec := range m.sections {
		for _, child := range sec.Children {
			if row, ok := child.(TableRow); ok {
				m.rows = append(m.rows, row)
			}
		}
	}

	return m
}

// resolveColumnCount determines the number of columns in the table.
func (n Table) resolveColumnCount(m tableModel) int {
	// From col definitions.
	colDefCount := 0
	for _, col := range m.colDefs {
		s := col.Span
		if s < 1 {
			s = 1
		}
		colDefCount += s
	}

	// From row cell counts (considering colspan).
	rowMax := 0
	for _, row := range m.rows {
		count := 0
		for _, child := range row.Children {
			if cell, ok := child.(TableCell); ok {
				cs := cell.ColSpan
				if cs < 1 {
					cs = 1
				}
				count += cs
			} else {
				count++
			}
		}
		if count > rowMax {
			rowMax = count
		}
	}

	if colDefCount > rowMax {
		return colDefCount
	}
	return rowMax
}

// assignCellPositions maps cells onto the grid, handling colspan/rowspan.
func (n Table) assignCellPositions(m tableModel, numCols int) []cellPlacement {
	// occupied[row][col] = true if a cell already covers that slot.
	occupied := map[[2]int]bool{}
	var placements []cellPlacement

	for rowIdx, row := range m.rows {
		colIdx := 0
		for _, child := range row.Children {
			cell, ok := child.(TableCell)
			if !ok {
				// Wrap non-cell children as a simple cell.
				cell = TableCell{Child: child, ColSpan: 1, RowSpan: 1}
			}
			if cell.ColSpan < 1 {
				cell.ColSpan = 1
			}
			if cell.RowSpan < 1 {
				cell.RowSpan = 1
			}

			// Skip occupied slots.
			for colIdx < numCols && occupied[[2]int{rowIdx, colIdx}] {
				colIdx++
			}
			if colIdx >= numCols {
				break
			}

			placements = append(placements, cellPlacement{cell: cell, row: rowIdx, col: colIdx})

			// Mark occupied cells.
			for r := rowIdx; r < rowIdx+cell.RowSpan; r++ {
				for c := colIdx; c < colIdx+cell.ColSpan && c < numCols; c++ {
					occupied[[2]int{r, c}] = true
				}
			}

			colIdx += cell.ColSpan
		}
	}

	return placements
}

// resolveColumnWidths computes column widths using the selected algorithm.
func (n Table) resolveColumnWidths(ctx *ui.LayoutContext, m tableModel, placements []cellPlacement, numCols, numRows, availW, spacingH int) []int {
	if n.Layout == TableLayoutFixed {
		return n.fixedColumnWidths(ctx, m, placements, numCols, availW, spacingH)
	}
	return n.autoColumnWidths(ctx, placements, numCols, availW, spacingH)
}

// fixedColumnWidths implements the CSS fixed table layout algorithm.
func (n Table) fixedColumnWidths(ctx *ui.LayoutContext, m tableModel, placements []cellPlacement, numCols, availW, spacingH int) []int {
	widths := make([]int, numCols)
	assigned := make([]bool, numCols)

	// 1. Column definitions take priority.
	colIdx := 0
	for _, col := range m.colDefs {
		s := col.Span
		if s < 1 {
			s = 1
		}
		for i := 0; i < s && colIdx < numCols; i++ {
			if col.Width.Type == TrackFixed {
				widths[colIdx] = int(col.Width.Value)
				assigned[colIdx] = true
			}
			colIdx++
		}
	}

	// 2. First row cells for unassigned columns.
	for _, p := range placements {
		if p.row != 0 || p.cell.ColSpan != 1 {
			continue
		}
		if assigned[p.col] {
			continue
		}
		if p.cell.Child != nil {
			cb := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: availW, H: 1<<30 - 1})
			widths[p.col] = cb.W
			assigned[p.col] = true
		}
	}

	// 3. Distribute remaining space equally among unassigned columns.
	totalSpacing := n.totalSpacing(numCols, spacingH)
	usedWidth := totalSpacing
	unassignedCount := 0
	for c := 0; c < numCols; c++ {
		if assigned[c] {
			usedWidth += widths[c]
		} else {
			unassignedCount++
		}
	}

	if unassignedCount > 0 {
		remaining := availW - usedWidth
		if remaining < 0 {
			remaining = 0
		}
		each := remaining / unassignedCount
		for c := 0; c < numCols; c++ {
			if !assigned[c] {
				widths[c] = each
			}
		}
	}

	return widths
}

// autoColumnWidths implements the CSS auto table layout algorithm.
func (n Table) autoColumnWidths(ctx *ui.LayoutContext, placements []cellPlacement, numCols, availW, spacingH int) []int {
	minWidths := make([]int, numCols)
	maxWidths := make([]int, numCols)

	// Measure single-column-span cells first.
	for _, p := range placements {
		if p.cell.ColSpan != 1 || p.cell.Child == nil {
			continue
		}
		// Min-content: measure with width=0.
		minB := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: 0, H: 1<<30 - 1})
		// Max-content: measure with unlimited width.
		maxB := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: availW, H: 1<<30 - 1})

		if minB.W > minWidths[p.col] {
			minWidths[p.col] = minB.W
		}
		if maxB.W > maxWidths[p.col] {
			maxWidths[p.col] = maxB.W
		}
	}

	// Multi-column-span cells: distribute excess min/max across spanned columns.
	for _, p := range placements {
		if p.cell.ColSpan <= 1 || p.cell.Child == nil {
			continue
		}
		endCol := p.col + p.cell.ColSpan
		if endCol > numCols {
			endCol = numCols
		}
		spanCols := endCol - p.col

		minB := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: 0, H: 1<<30 - 1})
		maxB := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: availW, H: 1<<30 - 1})

		// Sum current min/max of spanned columns.
		spanGaps := (spanCols - 1) * spacingH
		currentMin := spanGaps
		currentMax := spanGaps
		for c := p.col; c < endCol; c++ {
			currentMin += minWidths[c]
			currentMax += maxWidths[c]
		}

		// Distribute excess evenly.
		if minB.W > currentMin {
			excess := minB.W - currentMin
			each := excess / spanCols
			remainder := excess % spanCols
			for c := p.col; c < endCol; c++ {
				minWidths[c] += each
				if c-p.col < remainder {
					minWidths[c]++
				}
			}
		}
		if maxB.W > currentMax {
			excess := maxB.W - currentMax
			each := excess / spanCols
			remainder := excess % spanCols
			for c := p.col; c < endCol; c++ {
				maxWidths[c] += each
				if c-p.col < remainder {
					maxWidths[c]++
				}
			}
		}
	}

	// Ensure max >= min.
	for c := 0; c < numCols; c++ {
		if maxWidths[c] < minWidths[c] {
			maxWidths[c] = minWidths[c]
		}
	}

	// Distribute available width.
	totalSpacing := n.totalSpacing(numCols, spacingH)
	usable := availW - totalSpacing
	if usable < 0 {
		usable = 0
	}

	// Sum of min and max.
	totalMin := 0
	totalMax := 0
	for c := 0; c < numCols; c++ {
		totalMin += minWidths[c]
		totalMax += maxWidths[c]
	}

	widths := make([]int, numCols)

	if usable <= totalMin {
		// Not enough space: use min-content widths.
		copy(widths, minWidths)
	} else if usable >= totalMax {
		// Excess space: start with max-content, distribute surplus.
		copy(widths, maxWidths)
		surplus := usable - totalMax
		if surplus > 0 && numCols > 0 {
			each := surplus / numCols
			remainder := surplus % numCols
			for c := 0; c < numCols; c++ {
				widths[c] += each
				if c < remainder {
					widths[c]++
				}
			}
		}
	} else {
		// Between min and max: distribute proportionally.
		extra := usable - totalMin
		totalFlex := totalMax - totalMin
		for c := 0; c < numCols; c++ {
			widths[c] = minWidths[c]
			flex := maxWidths[c] - minWidths[c]
			if totalFlex > 0 && flex > 0 {
				widths[c] += extra * flex / totalFlex
			}
		}
	}

	return widths
}

// resolveRowHeights computes row heights by measuring cells.
func (n Table) resolveRowHeights(ctx *ui.LayoutContext, placements []cellPlacement, colWidths []int, numCols, numRows, spacingH, availH int) []int {
	heights := make([]int, numRows)

	// Single-row-span cells first.
	for _, p := range placements {
		if p.cell.RowSpan != 1 || p.cell.Child == nil {
			continue
		}
		cellW := 0
		for c := p.col; c < p.col+p.cell.ColSpan && c < numCols; c++ {
			cellW += colWidths[c]
			if c > p.col {
				cellW += spacingH
			}
		}
		cb := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: cellW, H: availH})
		if cb.H > heights[p.row] {
			heights[p.row] = cb.H
		}
	}

	// Multi-row-span cells: distribute excess height.
	for _, p := range placements {
		if p.cell.RowSpan <= 1 || p.cell.Child == nil {
			continue
		}
		endRow := p.row + p.cell.RowSpan
		if endRow > numRows {
			endRow = numRows
		}

		cellW := 0
		for c := p.col; c < p.col+p.cell.ColSpan && c < numCols; c++ {
			cellW += colWidths[c]
			if c > p.col {
				cellW += spacingH
			}
		}
		cb := ctx.MeasureChild(p.cell.Child, ui.Bounds{W: cellW, H: availH})

		spanRows := endRow - p.row
		currentH := 0
		for r := p.row; r < endRow; r++ {
			currentH += heights[r]
		}
		// Add inter-row spacing.
		if spanRows > 1 {
			currentH += (spanRows - 1) * int(n.BorderSpacingV)
		}

		if cb.H > currentH {
			excess := cb.H - currentH
			each := excess / spanRows
			remainder := excess % spanRows
			for r := p.row; r < endRow; r++ {
				heights[r] += each
				if r-p.row < remainder {
					heights[r]++
				}
			}
		}
	}

	return heights
}

// alignCellContent applies VAlign within the cell area.
func (n Table) alignCellContent(ctx *ui.LayoutContext, cell TableCell, x, y, w, h int) ui.Bounds {
	if cell.VAlign == VAlignTop || cell.Child == nil {
		return ui.Bounds{X: x, Y: y, W: w, H: h}
	}

	cb := ctx.MeasureChild(cell.Child, ui.Bounds{X: x, Y: y, W: w, H: h})

	switch cell.VAlign {
	case VAlignMiddle:
		y = y + (h-cb.H)/2
	case VAlignBottom:
		y = y + h - cb.H
	case VAlignBaseline:
		// Baseline alignment: use content's natural baseline.
		// For simplicity, treat as top alignment (full baseline support
		// requires per-row baseline coordination).
		return ui.Bounds{X: x, Y: y, W: w, H: h}
	}

	return ui.Bounds{X: x, Y: y, W: w, H: cb.H}
}

// gridWidth computes the total width of the table grid.
func (n Table) gridWidth(colWidths []int, numCols, spacingH int) int {
	w := 0
	for _, cw := range colWidths {
		w += cw
	}
	if numCols > 1 {
		w += (numCols - 1) * spacingH
	}
	if n.BorderCollapse == BorderSeparate {
		w += 2 * spacingH // left + right outer spacing
	}
	return w
}

// totalSpacing returns the total horizontal spacing consumed by gaps.
func (n Table) totalSpacing(numCols, spacingH int) int {
	s := 0
	if numCols > 1 {
		s = (numCols - 1) * spacingH
	}
	if n.BorderCollapse == BorderSeparate {
		s += 2 * spacingH // outer edges
	}
	return s
}
