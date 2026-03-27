package data

import (
	"fmt"
	"sort"
	"strings"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/ui/layout"
)

// ── Sort direction ──────────────────────────────────────────────

// SortDirection indicates the sort order for a column.
type SortDirection uint8

const (
	SortNone SortDirection = iota
	SortAsc
	SortDesc
)

// NextSortDirection cycles None → Asc → Desc → None.
func NextSortDirection(d SortDirection) SortDirection {
	switch d {
	case SortNone:
		return SortAsc
	case SortAsc:
		return SortDesc
	default:
		return SortNone
	}
}

// ── Column definition ───────────────────────────────────────────

// DataTableColumn defines a single DataTable column.
type DataTableColumn struct {
	Key      string                              // unique column identifier
	Header   string                              // display label
	Width    layout.TrackSize                    // column width (Px, Fr, AutoTrack, Minmax)
	Sortable bool                                // whether clicking header cycles sort
	VAlign   layout.VAlign                       // vertical alignment inside cells (default VAlignTop)
	Build    func(id int, loaded bool) ui.Element // cell content builder
	// FilterValue returns a string representation for client-side filtering.
	// If nil, the column is excluded from filtering.
	FilterValue func(id int) string
	// SortValue returns an integer for client-side sorting (SliceDataset).
	// Negative = less, positive = greater. If nil, sorts by index.
	SortLess func(i, j int) bool
}

// ── State ───────────────────────────────────────────────────────

// DataTableState holds mutable state (managed by the caller, like ScrollState).
type DataTableState struct {
	ScrollOffset float32       // vertical scroll position
	SortColumn   string        // key of the currently sorted column
	SortDir      SortDirection // current sort direction
	FilterText   string        // active filter string
	CurrentPage  int           // current page for display (paged mode)
}

// ── Messages ────────────────────────────────────────────────────

// DataTableSortMsg is sent when the user clicks a sortable column header.
type DataTableSortMsg struct {
	Column    string
	Direction SortDirection
}

// DataTableFilterMsg is sent when the user changes the filter text.
type DataTableFilterMsg struct {
	Text string
}

// DataTablePageMsg is sent when the user navigates to a different page.
type DataTablePageMsg struct {
	Page int
}

// ── DataTable component ─────────────────────────────────────────

const (
	dataTableOverscan   = 3
	dataTableHeaderH    = 32
	dataTableToolbarH   = 36
	dataTableFilterH    = 32
	dataTableDefaultRow = 32
	dataTableCellPad    = 8
)

// DataTable displays a data-driven table with sortable columns, optional
// filtering, and an auto-detected bottom toolbar (pagination for PagedDataset,
// row/item count for Slice/Stream).
//
// It works with all Dataset[int] implementations and uses virtualized
// rendering for the body rows.
type DataTable struct {
	ui.BaseElement

	// Dataset is the data source (SliceDataset, PagedDataset, or StreamDataset).
	Dataset Dataset[int]

	// Columns defines the column layout and cell builders.
	Columns []DataTableColumn

	// RowHeight is the height per body row in dp (default 32).
	RowHeight float32

	// MaxHeight is the max visible height before scrolling (0 = fill available space).
	MaxHeight float32

	// State is the caller-managed mutable state.
	State *DataTableState

	// Filterable enables a filter bar above the table.
	Filterable bool

	// OnSort is called when a sortable column header is clicked.
	OnSort func(col string, dir SortDirection)

	// OnFilter is called when the filter text changes.
	OnFilter func(text string)

	// OnPage is called when the user navigates to a different page.
	OnPage func(page int)

	// SelectedRow is the currently selected row index (-1 = none).
	SelectedRow int

	// OnSelectRow is called when a row is clicked.
	OnSelectRow func(index int)
}

// ── Options ─────────────────────────────────────────────────────

// DataTableOption configures a DataTable.
type DataTableOption func(*DataTable)

// WithDTRowHeight sets the row height in dp.
func WithDTRowHeight(h float32) DataTableOption {
	return func(dt *DataTable) { dt.RowHeight = h }
}

// WithDTMaxHeight sets the maximum visible height.
func WithDTMaxHeight(h float32) DataTableOption {
	return func(dt *DataTable) { dt.MaxHeight = h }
}

// WithDTFilterable enables the filter bar.
func WithDTFilterable(f bool) DataTableOption {
	return func(dt *DataTable) { dt.Filterable = f }
}

// WithDTOnSort sets the sort callback.
func WithDTOnSort(fn func(string, SortDirection)) DataTableOption {
	return func(dt *DataTable) { dt.OnSort = fn }
}

// WithDTOnFilter sets the filter callback.
func WithDTOnFilter(fn func(string)) DataTableOption {
	return func(dt *DataTable) { dt.OnFilter = fn }
}

// WithDTOnPage sets the page navigation callback.
func WithDTOnPage(fn func(int)) DataTableOption {
	return func(dt *DataTable) { dt.OnPage = fn }
}

// WithDTSelectedRow sets the selected row and callback.
func WithDTSelectedRow(idx int, fn func(int)) DataTableOption {
	return func(dt *DataTable) {
		dt.SelectedRow = idx
		dt.OnSelectRow = fn
	}
}

// NewDataTable creates a DataTable element.
func NewDataTable(ds Dataset[int], columns []DataTableColumn, state *DataTableState, opts ...DataTableOption) ui.Element {
	dt := DataTable{
		Dataset:     ds,
		Columns:     columns,
		State:       state,
		SelectedRow: -1,
	}
	for _, opt := range opts {
		opt(&dt)
	}
	return dt
}

// ── Column width resolution ─────────────────────────────────────

// resolveColumnWidths computes column widths from TrackSize definitions.
func resolveColumnWidths(cols []DataTableColumn, available int) []int {
	n := len(cols)
	if n == 0 {
		return nil
	}
	widths := make([]int, n)
	remaining := available
	totalFr := float32(0)
	autoCount := 0

	// Pass 1: Fixed and Minmax columns.
	for i, col := range cols {
		// Zero-value TrackSize (all fields zero) is treated as Fr(1).
		if col.Width == (layout.TrackSize{}) {
			totalFr += 1
			continue
		}
		switch col.Width.Type {
		case layout.TrackFixed:
			w := int(col.Width.Value)
			widths[i] = w
			remaining -= w
		case layout.TrackMinmax:
			w := int(col.Width.Min)
			widths[i] = w
			remaining -= w
		case layout.TrackFr:
			totalFr += col.Width.Value
		case layout.TrackAuto:
			autoCount++
		}
	}

	if remaining < 0 {
		remaining = 0
	}

	// Pass 2: Distribute remaining space to Fr columns.
	if totalFr > 0 {
		frUnit := float32(remaining) / totalFr
		for i, col := range cols {
			fr := float32(0)
			if col.Width == (layout.TrackSize{}) {
				fr = 1
			} else if col.Width.Type == layout.TrackFr {
				fr = col.Width.Value
			}
			if fr > 0 {
				w := int(frUnit * fr)
				widths[i] = w
				remaining -= w
			}
		}
	}

	// Pass 3: Auto columns share leftover equally.
	if autoCount > 0 && remaining > 0 {
		each := remaining / autoCount
		for i, col := range cols {
			if col.Width.Type == layout.TrackAuto {
				widths[i] = each
			}
		}
	}

	// Clamp Minmax upper bounds.
	for i, col := range cols {
		if col.Width.Type == layout.TrackMinmax && col.Width.Max > 0 {
			if float32(widths[i]) > col.Width.Max {
				widths[i] = int(col.Width.Max)
			}
		}
		if widths[i] < 0 {
			widths[i] = 0
		}
	}

	return widths
}

// ── Dataset helpers ─────────────────────────────────────────────

// isPaged returns true if the dataset is a PagedDataset.
func isPaged(ds Dataset[int]) (*PagedDataset[int], bool) {
	pd, ok := ds.(*PagedDataset[int])
	return pd, ok
}

// isStream returns true if the dataset is a StreamDataset.
func isStream(ds Dataset[int]) (*StreamDataset[int], bool) {
	sd, ok := ds.(*StreamDataset[int])
	return sd, ok
}

// resolvedItemCount returns the effective item count for the dataset.
func (n DataTable) resolvedItemCount() int {
	if n.Dataset == nil {
		return 0
	}
	l := n.Dataset.Len()
	if l >= 0 {
		return l
	}
	// Unknown length: paged or stream.
	if pd, ok := isPaged(n.Dataset); ok {
		loaded := pd.LoadedCount()
		return loaded + pd.PageSize
	}
	type counter interface{ Count() int }
	if c, ok := n.Dataset.(counter); ok {
		return c.Count()
	}
	return 0
}

// ── Client-side sort/filter (SliceDataset) ──────────────────────

// buildViewIndices returns the ordered indices to display, applying
// client-side filtering and sorting for SliceDataset.
func (n DataTable) buildViewIndices() []int {
	count := n.resolvedItemCount()
	if count <= 0 {
		return nil
	}

	indices := make([]int, count)
	for i := range indices {
		indices[i] = i
	}

	// Client-side filter: only for SliceDataset.
	if _, ok := n.Dataset.(*SliceDataset[int]); ok && n.State != nil && n.State.FilterText != "" {
		filterLower := strings.ToLower(n.State.FilterText)
		filtered := indices[:0]
		for _, idx := range indices {
			for _, col := range n.Columns {
				if col.FilterValue != nil {
					val := strings.ToLower(col.FilterValue(idx))
					if strings.Contains(val, filterLower) {
						filtered = append(filtered, idx)
						break
					}
				}
			}
		}
		indices = filtered
	}

	// Client-side sort: only for SliceDataset.
	if _, ok := n.Dataset.(*SliceDataset[int]); ok && n.State != nil && n.State.SortDir != SortNone && n.State.SortColumn != "" {
		var sortCol *DataTableColumn
		for ci := range n.Columns {
			if n.Columns[ci].Key == n.State.SortColumn {
				sortCol = &n.Columns[ci]
				break
			}
		}
		if sortCol != nil && sortCol.SortLess != nil {
			lessFunc := sortCol.SortLess
			dir := n.State.SortDir
			sort.SliceStable(indices, func(a, b int) bool {
				ia, ib := indices[a], indices[b]
				if dir == SortDesc {
					ia, ib = ib, ia
				}
				return lessFunc(ia, ib)
			})
		}
	}

	return indices
}

// ── LayoutSelf ──────────────────────────────────────────────────

func (n DataTable) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	if n.Dataset == nil || len(n.Columns) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	rowH := int(n.RowHeight)
	if rowH <= 0 {
		rowH = dataTableDefaultRow
	}

	// Resolve column widths.
	colWidths := resolveColumnWidths(n.Columns, area.W)

	// Compute vertical layout.
	y := area.Y
	filterBarY := y
	if n.Filterable {
		y += dataTableFilterH
	}
	headerY := y
	y += dataTableHeaderH
	bodyY := y

	// Determine toolbar presence and content.
	showToolbar := true
	toolbarH := dataTableToolbarH

	// Max visible height.
	maxH := int(n.MaxHeight)
	if maxH <= 0 || maxH > area.H {
		maxH = area.H
	}

	// Body viewport = maxH - header - filter - toolbar.
	usedH := (headerY - area.Y) + dataTableHeaderH
	if showToolbar {
		usedH += toolbarH
	}
	bodyViewportH := maxH - usedH
	if bodyViewportH < rowH {
		bodyViewportH = rowH
	}

	// Build view indices (applies client-side sort/filter).
	viewIndices := n.buildViewIndices()
	itemCount := len(viewIndices)

	contentH := float32(itemCount * rowH)
	needsScroll := contentH > float32(bodyViewportH)

	// Scrollbar width.
	scrollbarW := 0
	if needsScroll {
		scrollbarW = int(tokens.Scroll.TrackWidth)
		if scrollbarW <= 0 {
			scrollbarW = 8
		}
	}
	bodyContentW := area.W - scrollbarW

	// Scroll offset.
	var offset float32
	if n.State != nil {
		offset = n.State.ScrollOffset
	}

	totalH := (headerY - area.Y) + dataTableHeaderH + bodyViewportH
	if showToolbar {
		totalH += toolbarH
	}

	// ── Draw table background ───────────────────────────────
	tableRect := draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(totalH))
	canvas.FillRoundRect(tableRect, tokens.Radii.Card, draw.SolidPaint(tokens.Colors.Surface.Base))

	// ── Filter bar ──────────────────────────────────────────
	if n.Filterable {
		n.layoutFilterBar(ctx, area.X, filterBarY, area.W)
	}

	// ── Header row ──────────────────────────────────────────
	n.layoutHeader(ctx, area.X, headerY, colWidths, bodyContentW)

	// ── Body rows (virtualized) ─────────────────────────────
	canvas.PushClip(draw.R(float32(area.X), float32(bodyY), float32(area.W), float32(bodyViewportH)))

	firstVisible := int(offset) / rowH
	if firstVisible < 0 {
		firstVisible = 0
	}
	firstVisible -= dataTableOverscan
	if firstVisible < 0 {
		firstVisible = 0
	}

	lastVisible := (int(offset) + bodyViewportH) / rowH
	lastVisible += dataTableOverscan
	if lastVisible >= itemCount {
		lastVisible = itemCount - 1
	}

	// Track pages needing loading (for PagedDataset).
	var loadPages map[int]bool

	for vi := firstVisible; vi <= lastVisible; vi++ {
		if vi < 0 || vi >= len(viewIndices) {
			continue
		}
		dataIdx := viewIndices[vi]
		loaded := true
		if n.Dataset != nil {
			_, loaded = n.Dataset.Get(dataIdx)
			if !loaded {
				if pd, ok := isPaged(n.Dataset); ok {
					pg := pd.PageForIndex(dataIdx)
					if !pd.IsPageLoading(pg) && !pd.IsPageLoaded(pg) {
						if loadPages == nil {
							loadPages = make(map[int]bool)
						}
						loadPages[pg] = true
					}
				}
			}
		}

		rowY := bodyY + vi*rowH - int(offset)

		// Row background: alternating, hover, selection.
		rowRect := draw.R(float32(area.X), float32(rowY), float32(bodyContentW), float32(rowH))

		// Alternate row shading.
		bgColor := tokens.Colors.Surface.Base
		if vi%2 == 1 {
			bgColor = draw.Color{R: bgColor.R, G: bgColor.G, B: bgColor.B, A: bgColor.A}
			// Slight tint for alternating rows.
			bgColor = ui.LerpColor(bgColor, tokens.Colors.Surface.Elevated, 0.5)
		}

		// Selection highlight.
		if n.SelectedRow >= 0 && dataIdx == n.SelectedRow {
			accentBg := tokens.Colors.Accent.Primary
			accentBg.A = 0.15
			bgColor = ui.LerpColor(bgColor, accentBg, 1.0)
		}

		// Register row hit target.
		if n.OnSelectRow != nil {
			idx := dataIdx
			hoverOp := ctx.IX.RegisterHit(rowRect, func() {
				n.OnSelectRow(idx)
				app.Send(DataTableSortMsg{}) // trigger re-render
			})
			if hoverOp > 0 {
				bgColor = ui.LerpColor(bgColor, tokens.Colors.Surface.Hovered, hoverOp)
			}
		}

		canvas.FillRect(rowRect, draw.SolidPaint(bgColor))

		// Draw row divider.
		dividerRect := draw.R(float32(area.X), float32(rowY+rowH-1), float32(bodyContentW), 1)
		canvas.FillRect(dividerRect, draw.SolidPaint(tokens.Colors.Stroke.Divider))

		// Render cells.
		cellX := area.X
		for ci, col := range n.Columns {
			colW := colWidths[ci]
			if col.Build != nil {
				child := col.Build(dataIdx, loaded)
				cellArea := ui.Bounds{
					X: cellX + dataTableCellPad,
					Y: rowY,
					W: colW - 2*dataTableCellPad,
					H: rowH,
				}
				ctx.LayoutChild(child, cellArea)
			}
			cellX += colW
		}
	}

	// Send load requests for unloaded pages.
	if n.Dataset != nil && len(loadPages) > 0 {
		if pd, ok := isPaged(n.Dataset); ok {
			for pg := range loadPages {
				start := pg * pd.PageSize
				end := start + pd.PageSize - 1
				if n.Dataset.Len() >= 0 && end >= n.Dataset.Len() {
					end = n.Dataset.Len() - 1
				}
				app.Send(DatasetLoadRequestMsg{
					PageIndex:  pg,
					StartIndex: start,
					EndIndex:   end,
				})
			}
		}
	}

	// Draw scrollbar inside clip.
	if needsScroll && n.State != nil {
		ui.DrawScrollbar(canvas, tokens, ctx.IX, &ui.ScrollState{Offset: n.State.ScrollOffset},
			area.X+bodyContentW, bodyY, bodyViewportH, contentH, offset)
	}

	canvas.PopClip()

	// Clamp scroll state.
	if n.State != nil {
		maxScroll := contentH - float32(bodyViewportH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if n.State.ScrollOffset > maxScroll {
			n.State.ScrollOffset = maxScroll
		}
		if n.State.ScrollOffset < 0 {
			n.State.ScrollOffset = 0
		}
	}

	// Register scroll target.
	if n.State != nil && needsScroll {
		state := n.State
		cH := contentH
		vH := float32(bodyViewportH)
		ctx.IX.RegisterScroll(
			draw.R(float32(area.X), float32(bodyY), float32(area.W), float32(bodyViewportH)),
			cH, vH,
			func(deltaY float32) {
				state.ScrollOffset -= deltaY
				maxScroll := cH - vH
				if maxScroll < 0 {
					maxScroll = 0
				}
				if state.ScrollOffset < 0 {
					state.ScrollOffset = 0
				}
				if state.ScrollOffset > maxScroll {
					state.ScrollOffset = maxScroll
				}
			},
		)
	}

	// ── Bottom toolbar ──────────────────────────────────────
	if showToolbar {
		toolbarY := bodyY + bodyViewportH
		n.layoutToolbar(ctx, area.X, toolbarY, area.W)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: totalH}
}

// ── Filter bar rendering ────────────────────────────────────────

func (n DataTable) layoutFilterBar(ctx *ui.LayoutContext, x, y, w int) {
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	// Background.
	barRect := draw.R(float32(x), float32(y), float32(w), float32(dataTableFilterH))
	canvas.FillRect(barRect, draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Magnifying glass icon.
	iconStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       tokens.Typography.Label.Size * 1.5,
		Weight:     draw.FontWeightRegular,
		LineHeight: 1.0,
		Raster:     true,
	}
	iconX := float32(x) + tokens.Spacing.S
	iconY := float32(y) + (float32(dataTableFilterH)-iconStyle.Size)/2
	canvas.DrawText(icons.MagnifyingGlass, draw.Pt(iconX, iconY), iconStyle, tokens.Colors.Text.Secondary)

	// Filter text.
	filterText := ""
	if n.State != nil {
		filterText = n.State.FilterText
	}
	textStyle := tokens.Typography.Body
	textX := iconX + iconStyle.Size*1.5 + tokens.Spacing.XS
	textY := float32(y) + (float32(dataTableFilterH)-textStyle.Size)/2

	if filterText == "" {
		canvas.DrawText("Filter…", draw.Pt(textX, textY), textStyle, tokens.Colors.Text.Disabled)
	} else {
		canvas.DrawText(filterText, draw.Pt(textX, textY), textStyle, tokens.Colors.Text.Primary)
	}

	// Divider below filter.
	divRect := draw.R(float32(x), float32(y+dataTableFilterH-1), float32(w), 1)
	canvas.FillRect(divRect, draw.SolidPaint(tokens.Colors.Stroke.Divider))
}

// ── Header rendering ────────────────────────────────────────────

func (n DataTable) layoutHeader(ctx *ui.LayoutContext, x, y int, colWidths []int, contentW int) {
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	// Header background.
	headerRect := draw.R(float32(x), float32(y), float32(contentW), float32(dataTableHeaderH))
	canvas.FillRect(headerRect, draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Header divider.
	divRect := draw.R(float32(x), float32(y+dataTableHeaderH-1), float32(contentW), 1)
	canvas.FillRect(divRect, draw.SolidPaint(tokens.Colors.Stroke.Divider))

	labelStyle := tokens.Typography.Label
	cellX := x

	for ci, col := range n.Columns {
		colW := colWidths[ci]
		textX := float32(cellX + dataTableCellPad)
		textY := float32(y) + (float32(dataTableHeaderH)-labelStyle.Size)/2

		// Draw header label.
		canvas.DrawText(col.Header, draw.Pt(textX, textY), labelStyle, tokens.Colors.Text.Secondary)

		// Sort indicator for sortable columns.
		if col.Sortable && n.State != nil && n.State.SortColumn == col.Key && n.State.SortDir != SortNone {
			iconStyle := draw.TextStyle{
				FontFamily: "Phosphor",
				Size:       labelStyle.Size * 1.2,
				Weight:     draw.FontWeightRegular,
				LineHeight: 1.0,
				Raster:     true,
			}
			metrics := ctx.Canvas.MeasureText(col.Header, labelStyle)
			iconX := textX + metrics.Width + tokens.Spacing.XS
			iconGlyph := icons.CaretUp
			if n.State.SortDir == SortDesc {
				iconGlyph = icons.CaretDown
			}
			canvas.DrawText(iconGlyph, draw.Pt(iconX, textY), iconStyle, tokens.Colors.Accent.Primary)
		}

		// Register hit target for sortable columns.
		if col.Sortable {
			colRect := draw.R(float32(cellX), float32(y), float32(colW), float32(dataTableHeaderH))
			key := col.Key
			hoverOp := ctx.IX.RegisterHit(colRect, func() {
				newDir := SortNone
				if n.State != nil && n.State.SortColumn == key {
					newDir = NextSortDirection(n.State.SortDir)
				} else {
					newDir = SortAsc
				}
				app.Send(DataTableSortMsg{Column: key, Direction: newDir})
				if n.OnSort != nil {
					n.OnSort(key, newDir)
				}
			})
			if hoverOp > 0 {
				hoverRect := draw.R(float32(cellX), float32(y), float32(colW), float32(dataTableHeaderH))
				hoverColor := tokens.Colors.Surface.Hovered
				hoverColor.A = hoverOp * 0.3
				canvas.FillRect(hoverRect, draw.SolidPaint(hoverColor))
			}
		}

		cellX += colW
	}
}

// ── Toolbar rendering ───────────────────────────────────────────

func (n DataTable) layoutToolbar(ctx *ui.LayoutContext, x, y, w int) {
	tokens := ctx.Tokens
	canvas := ctx.Canvas

	// Background.
	toolbarRect := draw.R(float32(x), float32(y), float32(w), float32(dataTableToolbarH))
	canvas.FillRoundRect(toolbarRect, 0, draw.SolidPaint(tokens.Colors.Surface.Elevated))

	// Top divider.
	divRect := draw.R(float32(x), float32(y), float32(w), 1)
	canvas.FillRect(divRect, draw.SolidPaint(tokens.Colors.Stroke.Divider))

	labelStyle := tokens.Typography.BodySmall
	textY := float32(y) + (float32(dataTableToolbarH)-labelStyle.Size)/2

	if pd, ok := isPaged(n.Dataset); ok {
		// Paged toolbar: "Page X of Y" + prev/next.
		page := 0
		if n.State != nil {
			page = n.State.CurrentPage
		}
		totalPages := 1
		if pd.TotalCount > 0 && pd.PageSize > 0 {
			totalPages = (pd.TotalCount + pd.PageSize - 1) / pd.PageSize
		}

		// Page info.
		pageText := fmt.Sprintf("Page %d of %d", page+1, totalPages)
		if pd.TotalCount < 0 {
			pageText = fmt.Sprintf("Page %d", page+1)
		}

		// Row range.
		startRow := page*pd.PageSize + 1
		endRow := startRow + pd.PageSize - 1
		if pd.TotalCount >= 0 && endRow > pd.TotalCount {
			endRow = pd.TotalCount
		}
		rangeText := ""
		if pd.TotalCount >= 0 {
			rangeText = fmt.Sprintf("  ·  Rows %d–%d of %d", startRow, endRow, pd.TotalCount)
		}

		textX := float32(x) + tokens.Spacing.S
		canvas.DrawText(pageText+rangeText, draw.Pt(textX, textY), labelStyle, tokens.Colors.Text.Secondary)

		// Prev/Next buttons.
		btnStyle := draw.TextStyle{
			FontFamily: "Phosphor",
			Size:       labelStyle.Size * 1.8,
			Weight:     draw.FontWeightRegular,
			LineHeight: 1.0,
			Raster:     true,
		}
		btnSize := int(btnStyle.Size * 1.5)
		nextX := x + w - btnSize - int(tokens.Spacing.S)
		prevX := nextX - btnSize - int(tokens.Spacing.XS)

		// Prev button.
		if page > 0 {
			prevRect := draw.R(float32(prevX), float32(y+2), float32(btnSize), float32(dataTableToolbarH-4))
			pg := page
			hoverOp := ctx.IX.RegisterHit(prevRect, func() {
				app.Send(DataTablePageMsg{Page: pg - 1})
				if n.OnPage != nil {
					n.OnPage(pg - 1)
				}
			})
			btnColor := tokens.Colors.Text.Secondary
			if hoverOp > 0 {
				btnColor = ui.LerpColor(btnColor, tokens.Colors.Accent.Primary, hoverOp)
			}
			btnY := float32(y) + (float32(dataTableToolbarH)-btnStyle.Size)/2
			canvas.DrawText(icons.CaretLeft, draw.Pt(float32(prevX), btnY), btnStyle, btnColor)
		}

		// Next button.
		if page < totalPages-1 {
			nextRect := draw.R(float32(nextX), float32(y+2), float32(btnSize), float32(dataTableToolbarH-4))
			pg := page
			hoverOp := ctx.IX.RegisterHit(nextRect, func() {
				app.Send(DataTablePageMsg{Page: pg + 1})
				if n.OnPage != nil {
					n.OnPage(pg + 1)
				}
			})
			btnColor := tokens.Colors.Text.Secondary
			if hoverOp > 0 {
				btnColor = ui.LerpColor(btnColor, tokens.Colors.Accent.Primary, hoverOp)
			}
			btnY := float32(y) + (float32(dataTableToolbarH)-btnStyle.Size)/2
			canvas.DrawText(icons.CaretRight, draw.Pt(float32(nextX), btnY), btnStyle, btnColor)
		}

	} else if sd, ok := isStream(n.Dataset); ok {
		// Stream toolbar: item count.
		text := fmt.Sprintf("%d items", sd.Count())
		textX := float32(x) + tokens.Spacing.S
		canvas.DrawText(text, draw.Pt(textX, textY), labelStyle, tokens.Colors.Text.Secondary)

	} else {
		// SliceDataset or other: row count.
		count := n.resolvedItemCount()
		text := fmt.Sprintf("%d rows", count)
		if n.State != nil && n.State.FilterText != "" {
			viewCount := len(n.buildViewIndices())
			text = fmt.Sprintf("%d of %d rows", viewCount, count)
		}
		textX := float32(x) + tokens.Spacing.S
		canvas.DrawText(text, draw.Pt(textX, textY), labelStyle, tokens.Colors.Text.Secondary)
	}
}

// ── TreeEqual ───────────────────────────────────────────────────

func (n DataTable) TreeEqual(other ui.Element) bool {
	o, ok := other.(DataTable)
	if !ok {
		return false
	}
	return n.Dataset == o.Dataset &&
		len(n.Columns) == len(o.Columns) &&
		n.RowHeight == o.RowHeight &&
		n.MaxHeight == o.MaxHeight &&
		n.Filterable == o.Filterable &&
		n.SelectedRow == o.SelectedRow
}

// ── ResolveChildren ─────────────────────────────────────────────

func (n DataTable) ResolveChildren(_ func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// ── WalkAccess ──────────────────────────────────────────────────

func (n DataTable) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	tableIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleTable, Label: "DataTable"}, parentIdx, a11y.Rect{})

	// Header row.
	headerIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: "Header"}, int32(tableIdx), a11y.Rect{})
	for _, col := range n.Columns {
		b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: col.Header}, int32(headerIdx), a11y.Rect{})
	}

	// Body rows.
	itemCount := n.resolvedItemCount()
	for i := 0; i < itemCount; i++ {
		loaded := true
		if n.Dataset != nil {
			_, loaded = n.Dataset.Get(i)
		}
		label := fmt.Sprintf("Row %d", i+1)
		if !loaded {
			label += " (loading)"
		}
		b.AddNode(a11y.AccessNode{Role: a11y.RoleGroup, Label: label}, int32(tableIdx), a11y.Rect{})
	}
}
