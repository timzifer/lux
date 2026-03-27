package layout

import (
	"sort"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// FlexDirection controls the main axis of the Flex layout.
type FlexDirection int

const (
	FlexRow           FlexDirection = iota // Left to right (default)
	FlexColumn                            // Top to bottom
	FlexRowReverse                        // Right to left
	FlexColumnReverse                     // Bottom to top
)

// FlexWrap controls whether flex items wrap onto multiple lines.
type FlexWrap int

const (
	FlexNoWrap      FlexWrap = iota // Single line (default)
	FlexWrapOn                      // Wrap onto multiple lines
	FlexWrapReverse                 // Wrap in reverse order
)

// Justify controls alignment along the main axis.
type Justify int

const (
	JustifyStart        Justify = iota
	JustifyEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// Align controls alignment along the cross axis.
type Align int

const (
	AlignStart   Align = iota
	AlignEnd
	AlignCenter
	AlignStretch
)

// AlignContent controls distribution of lines along the cross axis when wrapping.
type AlignContent int

const (
	AlignContentStart        AlignContent = iota
	AlignContentEnd
	AlignContentCenter
	AlignContentSpaceBetween
	AlignContentSpaceAround
	AlignContentStretch
)

// AlignSelf overrides the container's Align for an individual child.
type AlignSelf int

const (
	AlignSelfAuto    AlignSelf = iota // Inherit from container
	AlignSelfStart
	AlignSelfEnd
	AlignSelfCenter
	AlignSelfStretch
)

// FlexBasis defines the initial main size of a flex item before grow/shrink.
type FlexBasis struct {
	Auto  bool    // Use natural size (default)
	Value float32 // Fixed dp value when Auto is false
}

// AutoBasis returns a FlexBasis that uses the item's natural size.
func AutoBasis() FlexBasis { return FlexBasis{Auto: true} }

// FixedBasis returns a FlexBasis with a fixed dp value.
func FixedBasis(v float32) FlexBasis { return FlexBasis{Value: v} }

// FlexOption configures a Flex element.
type FlexOption func(*Flex)

// Flex is a flexible layout container implementing CSS Flexbox semantics.
type Flex struct {
	ui.BaseElement
	Direction    FlexDirection
	Wrap         FlexWrap
	Justify      Justify
	Align        Align
	AlignContent AlignContent
	Gap          float32 // Legacy single gap; used when RowGap and ColGap are both 0.
	RowGap       float32
	ColGap       float32
	Children     []ui.Element
}

// NewFlex creates a Flex layout container.
func NewFlex(children []ui.Element, opts ...FlexOption) ui.Element {
	el := Flex{
		Direction: FlexRow,
		Justify:   JustifyStart,
		Align:     AlignStart,
		Children:  children,
	}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// WithDirection sets the main axis direction.
func WithDirection(d FlexDirection) FlexOption {
	return func(e *Flex) { e.Direction = d }
}

// WithJustify sets main-axis alignment.
func WithJustify(j Justify) FlexOption {
	return func(e *Flex) { e.Justify = j }
}

// WithAlign sets cross-axis alignment.
func WithAlign(a Align) FlexOption {
	return func(e *Flex) { e.Align = a }
}

// WithGap sets the gap between children in dp (shorthand for both row and column gap).
func WithGap(gap float32) FlexOption {
	return func(e *Flex) { e.Gap = gap }
}

// WithWrap sets the flex wrap mode.
func WithWrap(w FlexWrap) FlexOption {
	return func(e *Flex) { e.Wrap = w }
}

// WithAlignContent sets cross-axis alignment for wrapped lines.
func WithAlignContent(a AlignContent) FlexOption {
	return func(e *Flex) { e.AlignContent = a }
}

// WithFlexRowGap sets the gap between rows (cross-axis gap when wrapping in row direction).
func WithFlexRowGap(gap float32) FlexOption {
	return func(e *Flex) { e.RowGap = gap }
}

// WithFlexColGap sets the gap between columns (main-axis gap in row direction).
func WithFlexColGap(gap float32) FlexOption {
	return func(e *Flex) { e.ColGap = gap }
}

// flexChildInfo holds resolved per-child flex properties.
type flexChildInfo struct {
	child     ui.Element // the actual renderable child
	grow      float32
	shrink    float32
	basis     FlexBasis
	alignSelf AlignSelf
	order     int
	origIdx   int // original index before order-sorting
}

// resolveFlexChild extracts flex properties from a child.
// Non-Expanded children get default values (grow=0, shrink=1, basis=auto).
func resolveFlexChild(el ui.Element, idx int) flexChildInfo {
	if e, ok := el.(Expanded); ok {
		return flexChildInfo{
			child:     e.Child,
			grow:      e.Grow,
			shrink:    e.Shrink,
			basis:     e.Basis,
			alignSelf: e.AlignSelf,
			order:     e.Order,
			origIdx:   idx,
		}
	}
	return flexChildInfo{
		child:   el,
		grow:    0,
		shrink:  1,
		basis:   FlexBasis{Auto: true},
		origIdx: idx,
	}
}

// resolveGaps returns (mainGap, crossGap) in dp.
func (n Flex) resolveGaps() (int, int) {
	rg, cg := int(n.RowGap), int(n.ColGap)
	if rg == 0 && cg == 0 && n.Gap > 0 {
		rg = int(n.Gap)
		cg = int(n.Gap)
	}
	isRow := n.Direction == FlexRow || n.Direction == FlexRowReverse
	if isRow {
		return cg, rg // main=column-gap, cross=row-gap
	}
	return rg, cg // main=row-gap, cross=column-gap
}

// isRowDirection returns true for FlexRow and FlexRowReverse.
func (n Flex) isRowDirection() bool {
	return n.Direction == FlexRow || n.Direction == FlexRowReverse
}

// isReversed returns true for reverse directions.
func (n Flex) isReversed() bool {
	return n.Direction == FlexRowReverse || n.Direction == FlexColumnReverse
}

// LayoutSelf implements ui.Layouter with full CSS Flexbox semantics.
func (n Flex) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	nn := len(n.Children)
	if nn == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	isRow := n.isRowDirection()
	reversed := n.isReversed()
	mainGap, crossGap := n.resolveGaps()

	mainAvail := area.W
	crossAvail := area.H
	if !isRow {
		mainAvail = area.H
		crossAvail = area.W
	}

	// Resolve children and sort by order.
	items := make([]flexChildInfo, nn)
	for i, child := range n.Children {
		items[i] = resolveFlexChild(child, i)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].order < items[j].order
	})

	// Reverse visual order if direction is reversed.
	if reversed {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	// Measure natural (hypothetical main) sizes.
	type measuredItem struct {
		naturalMain  int
		naturalCross int
		basisMain    int // resolved basis size
	}
	measured := make([]measuredItem, nn)
	for i, item := range items {
		cb := ctx.MeasureChild(item.child, area)
		if isRow {
			measured[i] = measuredItem{naturalMain: cb.W, naturalCross: cb.H}
		} else {
			measured[i] = measuredItem{naturalMain: cb.H, naturalCross: cb.W}
		}
		// Resolve basis.
		if item.basis.Auto {
			measured[i].basisMain = measured[i].naturalMain
		} else {
			measured[i].basisMain = int(item.basis.Value)
		}
	}

	// Distribute items into flex lines.
	type flexLine struct {
		start, end int // indices into items[]
	}
	var lines []flexLine

	if n.Wrap == FlexNoWrap {
		lines = []flexLine{{start: 0, end: nn}}
	} else {
		lineStart := 0
		lineMain := 0
		for i := range items {
			itemMain := measured[i].basisMain
			gapBefore := 0
			if i > lineStart {
				gapBefore = mainGap
			}
			if i > lineStart && lineMain+gapBefore+itemMain > mainAvail {
				lines = append(lines, flexLine{start: lineStart, end: i})
				lineStart = i
				lineMain = itemMain
			} else {
				lineMain += gapBefore + itemMain
			}
		}
		lines = append(lines, flexLine{start: lineStart, end: nn})
	}

	// Reverse lines for wrap-reverse.
	if n.Wrap == FlexWrapReverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	// Resolve RTL logical direction.
	dir := ui.Direction()
	rtlRow := isRow && dir == draw.DirRTL
	rtlColumn := !isRow && dir == draw.DirRTL

	justify := n.Justify
	align := n.Align
	if rtlRow {
		switch justify {
		case JustifyStart:
			justify = JustifyEnd
		case JustifyEnd:
			justify = JustifyStart
		}
	}
	if rtlColumn {
		switch align {
		case AlignStart:
			align = AlignEnd
		case AlignEnd:
			align = AlignStart
		}
	}

	// Layout each line: apply grow/shrink, compute positions.
	type laidOutItem struct {
		mainPos   int
		crossPos  int // relative to line start
		mainSize  int
		crossSize int
		itemIdx   int // index into items[]
	}
	type laidOutLine struct {
		items     []laidOutItem
		crossSize int
	}
	laidLines := make([]laidOutLine, len(lines))

	for li, line := range lines {
		count := line.end - line.start
		if count == 0 {
			continue
		}

		// Sum basis sizes and grow/shrink factors for this line.
		totalBasis := 0
		totalGrow := float32(0)
		for i := line.start; i < line.end; i++ {
			totalBasis += measured[i].basisMain
			totalGrow += items[i].grow
		}
		lineGaps := 0
		if count > 1 {
			lineGaps = mainGap * (count - 1)
		}

		// Compute resolved main sizes with grow/shrink.
		mainSizes := make([]int, count)
		freeSpace := mainAvail - totalBasis - lineGaps

		if freeSpace > 0 && totalGrow > 0 {
			// Grow: distribute positive free space proportionally.
			for i := line.start; i < line.end; i++ {
				li := i - line.start
				mainSizes[li] = measured[i].basisMain + int(float32(freeSpace)*items[i].grow/totalGrow)
			}
		} else if freeSpace < 0 {
			// Shrink: distribute negative overflow proportionally.
			overflow := -freeSpace
			totalScaledShrink := float32(0)
			for i := line.start; i < line.end; i++ {
				totalScaledShrink += items[i].shrink * float32(measured[i].basisMain)
			}
			for i := line.start; i < line.end; i++ {
				li := i - line.start
				shrinkAmount := 0
				if totalScaledShrink > 0 {
					shrinkAmount = int(float32(overflow) * items[i].shrink * float32(measured[i].basisMain) / totalScaledShrink)
				}
				mainSizes[li] = measured[i].basisMain - shrinkAmount
				if mainSizes[li] < 0 {
					mainSizes[li] = 0
				}
			}
		} else {
			// No grow/shrink needed.
			for i := line.start; i < line.end; i++ {
				mainSizes[i-line.start] = measured[i].basisMain
			}
		}

		// Re-measure cross sizes at final main sizes.
		crossSizes := make([]int, count)
		for i := line.start; i < line.end; i++ {
			li := i - line.start
			var measureArea ui.Bounds
			if isRow {
				measureArea = ui.Bounds{W: mainSizes[li], H: area.H}
			} else {
				measureArea = ui.Bounds{W: area.W, H: mainSizes[li]}
			}
			cb := ctx.MeasureChild(items[i].child, measureArea)
			if isRow {
				crossSizes[li] = cb.H
			} else {
				crossSizes[li] = cb.W
			}
		}

		// Line cross size = max cross of items in this line.
		lineCross := 0
		for _, cs := range crossSizes {
			if cs > lineCross {
				lineCross = cs
			}
		}

		// Compute main-axis positions with justify.
		totalMain := lineGaps
		for _, ms := range mainSizes {
			totalMain += ms
		}
		lineFree := mainAvail - totalMain
		if lineFree < 0 {
			lineFree = 0
		}
		startOffset := 0
		extraGap := 0
		switch justify {
		case JustifyEnd:
			startOffset = lineFree
		case JustifyCenter:
			startOffset = lineFree / 2
		case JustifySpaceBetween:
			if count > 1 {
				extraGap = lineFree / (count - 1)
			}
		case JustifySpaceAround:
			if count > 0 {
				extraGap = lineFree / count
				startOffset = extraGap / 2
			}
		case JustifySpaceEvenly:
			if count > 0 {
				extraGap = lineFree / (count + 1)
				startOffset = extraGap
			}
		}

		// Build laid-out items for this line.
		lineItems := make([]laidOutItem, count)
		mainCursor := startOffset
		for i := line.start; i < line.end; i++ {
			li := i - line.start

			// Resolve per-child alignment.
			childAlign := align
			if items[i].alignSelf != AlignSelfAuto {
				switch items[i].alignSelf {
				case AlignSelfStart:
					childAlign = AlignStart
				case AlignSelfEnd:
					childAlign = AlignEnd
				case AlignSelfCenter:
					childAlign = AlignCenter
				case AlignSelfStretch:
					childAlign = AlignStretch
				}
			}

			cs := crossSizes[li]
			crossOffset := 0
			switch childAlign {
			case AlignEnd:
				crossOffset = lineCross - cs
			case AlignCenter:
				crossOffset = (lineCross - cs) / 2
			case AlignStretch:
				cs = lineCross
			}
			if crossOffset < 0 {
				crossOffset = 0
			}

			lineItems[li] = laidOutItem{
				mainPos:   mainCursor,
				crossPos:  crossOffset,
				mainSize:  mainSizes[li],
				crossSize: cs,
				itemIdx:   i,
			}
			mainCursor += mainSizes[li] + mainGap + extraGap
		}

		laidLines[li] = laidOutLine{items: lineItems, crossSize: lineCross}
	}

	// Compute cross-axis positions for lines (align-content).
	totalLineCross := 0
	for i, ll := range laidLines {
		totalLineCross += ll.crossSize
		if i > 0 {
			totalLineCross += crossGap
		}
	}
	crossFree := crossAvail - totalLineCross
	if crossFree < 0 {
		crossFree = 0
	}

	lineOffsets := make([]int, len(laidLines))
	if len(laidLines) == 1 {
		// Single line: align-content has no effect.
		lineOffsets[0] = 0
	} else {
		ac := n.AlignContent
		switch ac {
		case AlignContentEnd:
			cursor := crossFree
			for i, ll := range laidLines {
				lineOffsets[i] = cursor
				cursor += ll.crossSize + crossGap
			}
		case AlignContentCenter:
			cursor := crossFree / 2
			for i, ll := range laidLines {
				lineOffsets[i] = cursor
				cursor += ll.crossSize + crossGap
			}
		case AlignContentSpaceBetween:
			extraCross := 0
			if len(laidLines) > 1 {
				extraCross = crossFree / (len(laidLines) - 1)
			}
			cursor := 0
			for i, ll := range laidLines {
				lineOffsets[i] = cursor
				cursor += ll.crossSize + crossGap + extraCross
			}
		case AlignContentSpaceAround:
			extraCross := 0
			if len(laidLines) > 0 {
				extraCross = crossFree / len(laidLines)
			}
			cursor := extraCross / 2
			for i, ll := range laidLines {
				lineOffsets[i] = cursor
				cursor += ll.crossSize + crossGap + extraCross
			}
		case AlignContentStretch:
			extra := 0
			if len(laidLines) > 0 {
				extra = crossFree / len(laidLines)
			}
			cursor := 0
			for i := range laidLines {
				lineOffsets[i] = cursor
				laidLines[i].crossSize += extra
				cursor += laidLines[i].crossSize + crossGap
			}
		default: // AlignContentStart
			cursor := 0
			for i, ll := range laidLines {
				lineOffsets[i] = cursor
				cursor += ll.crossSize + crossGap
			}
		}
	}

	// For RTL rows, mirror main-axis positions.
	if rtlRow {
		for li := range laidLines {
			for i := range laidLines[li].items {
				item := &laidLines[li].items[i]
				item.mainPos = mainAvail - item.mainPos - item.mainSize
			}
		}
	}

	// Paint pass: layout all children.
	maxMainActual := 0
	maxCrossActual := 0

	for li, ll := range laidLines {
		for _, item := range ll.items {
			fi := items[item.itemIdx]

			var childArea ui.Bounds
			if isRow {
				childArea = ui.Bounds{
					X: area.X + item.mainPos,
					Y: area.Y + lineOffsets[li] + item.crossPos,
					W: item.mainSize,
					H: item.crossSize,
				}
			} else {
				childArea = ui.Bounds{
					X: area.X + lineOffsets[li] + item.crossPos,
					Y: area.Y + item.mainPos,
					W: item.crossSize,
					H: item.mainSize,
				}
			}

			ctx.LayoutChild(fi.child, childArea)

			endMain := item.mainPos + item.mainSize
			if endMain > maxMainActual {
				maxMainActual = endMain
			}
		}
		endCross := lineOffsets[li] + ll.crossSize
		if endCross > maxCrossActual {
			maxCrossActual = endCross
		}
	}

	if isRow {
		return ui.Bounds{X: area.X, Y: area.Y, W: maxMainActual, H: maxCrossActual}
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: maxCrossActual, H: maxMainActual}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Flex) TreeEqual(other ui.Element) bool {
	o, ok := other.(Flex)
	return ok && n.Direction == o.Direction && n.Wrap == o.Wrap &&
		n.Justify == o.Justify && n.Align == o.Align && n.AlignContent == o.AlignContent &&
		n.Gap == o.Gap && n.RowGap == o.RowGap && n.ColGap == o.ColGap &&
		len(n.Children) == len(o.Children)
}

// ResolveChildren implements ui.ChildResolver.
func (n Flex) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	resolved := make([]ui.Element, len(n.Children))
	for i, child := range n.Children {
		resolved[i] = resolve(child, i)
	}
	out := n
	out.Children = resolved
	return out
}

// WalkAccess implements ui.AccessWalker.
func (n Flex) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	for _, child := range n.Children {
		b.Walk(child, parentIdx)
	}
}
