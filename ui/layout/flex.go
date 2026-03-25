package layout

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// FlexDirection controls the main axis of the Flex layout.
type FlexDirection int

const (
	FlexRow    FlexDirection = iota // Left to right (default)
	FlexColumn                     // Top to bottom
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

// FlexOption configures a Flex element.
type FlexOption func(*Flex)

// Flex is a flexible layout container.
type Flex struct {
	ui.BaseElement
	Direction FlexDirection
	Justify   Justify
	Align     Align
	Gap       float32
	Children  []ui.Element
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

// WithGap sets the gap between children in dp.
func WithGap(gap float32) FlexOption {
	return func(e *Flex) { e.Gap = gap }
}

// expandedChild extracts the Expanded info from a child.
type expandedChild struct {
	child ui.Element
	grow  float32
}

func asExpanded(el ui.Element) (expandedChild, bool) {
	if e, ok := el.(Expanded); ok {
		return expandedChild{child: e.Child, grow: e.Grow}, true
	}
	return expandedChild{}, false
}

// LayoutSelf implements ui.Layouter.
func (n Flex) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	nn := len(n.Children)
	if nn == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	isRow := n.Direction == FlexRow
	gap := int(n.Gap)

	// Main axis available space.
	mainAvail := area.W
	if !isRow {
		mainAvail = area.H
	}

	// Pass 1: measure inflexible children to determine their natural size.
	type childInfo struct {
		mainSize  int
		crossSize int
		expanded  bool
		grow      float32
	}
	infos := make([]childInfo, nn)
	totalFixed := 0
	totalGrow := float32(0)

	for i, child := range n.Children {
		if exp, ok := asExpanded(child); ok {
			infos[i] = childInfo{expanded: true, grow: exp.grow}
			totalGrow += exp.grow
		} else {
			// Measure with NullCanvas (no paint).
			cb := ctx.MeasureChild(child, area)
			if isRow {
				infos[i] = childInfo{mainSize: cb.W, crossSize: cb.H}
			} else {
				infos[i] = childInfo{mainSize: cb.H, crossSize: cb.W}
			}
			totalFixed += infos[i].mainSize
		}
	}

	// Gaps between children.
	totalGaps := 0
	if nn > 1 {
		totalGaps = gap * (nn - 1)
	}

	// Remaining space for expanded children.
	remaining := mainAvail - totalFixed - totalGaps
	if remaining < 0 {
		remaining = 0
	}

	// Assign main sizes to expanded children.
	for i := range infos {
		if infos[i].expanded {
			if totalGrow > 0 {
				infos[i].mainSize = int(float32(remaining) * infos[i].grow / totalGrow)
			}
			// Measure expanded child to get cross size.
			exp, _ := asExpanded(n.Children[i])
			var measureArea ui.Bounds
			if isRow {
				measureArea = ui.Bounds{X: 0, Y: 0, W: infos[i].mainSize, H: area.H}
			} else {
				measureArea = ui.Bounds{X: 0, Y: 0, W: area.W, H: infos[i].mainSize}
			}
			cb := ctx.MeasureChild(exp.child, measureArea)
			if isRow {
				infos[i].crossSize = cb.H
			} else {
				infos[i].crossSize = cb.W
			}
		}
	}

	// Compute max cross size.
	maxCross := 0
	totalMain := totalGaps
	for _, info := range infos {
		totalMain += info.mainSize
		if info.crossSize > maxCross {
			maxCross = info.crossSize
		}
	}

	// Resolve logical justify/align for RTL.
	justify := n.Justify
	align := n.Align
	dir := ui.Direction()
	rtlRow := isRow && dir == draw.DirRTL
	if rtlRow {
		switch justify {
		case JustifyStart:
			justify = JustifyEnd
		case JustifyEnd:
			justify = JustifyStart
		}
	}
	rtlColumn := !isRow && dir == draw.DirRTL
	if rtlColumn {
		switch align {
		case AlignStart:
			align = AlignEnd
		case AlignEnd:
			align = AlignStart
		}
	}

	// Apply Justify: compute start offset and extra spacing.
	freeSpace := mainAvail - totalMain
	if freeSpace < 0 {
		freeSpace = 0
	}
	startOffset := 0
	extraGap := 0
	switch justify {
	case JustifyEnd:
		startOffset = freeSpace
	case JustifyCenter:
		startOffset = freeSpace / 2
	case JustifySpaceBetween:
		if nn > 1 {
			extraGap = freeSpace / (nn - 1)
		}
	case JustifySpaceAround:
		if nn > 0 {
			extraGap = freeSpace / nn
			startOffset = extraGap / 2
		}
	case JustifySpaceEvenly:
		if nn > 0 {
			extraGap = freeSpace / (nn + 1)
			startOffset = extraGap
		}
	}

	// For RTL rows, reverse the child order so they flow right-to-left.
	children := n.Children
	childInfos := infos
	if rtlRow {
		children = make([]ui.Element, nn)
		childInfos = make([]childInfo, nn)
		for i := range n.Children {
			children[i] = n.Children[nn-1-i]
			childInfos[i] = infos[nn-1-i]
		}
	}

	// Pass 2: paint children at computed positions.
	mainCursor := startOffset
	maxCrossActual := 0

	for i, child := range children {
		info := childInfos[i]

		// Compute cross-axis offset based on Align.
		crossOffset := 0
		crossAvail := maxCross
		if isRow {
			crossAvail = area.H
		} else {
			crossAvail = area.W
		}
		switch align {
		case AlignEnd:
			crossOffset = crossAvail - info.crossSize
		case AlignCenter:
			crossOffset = (crossAvail - info.crossSize) / 2
		case AlignStretch:
			info.crossSize = crossAvail
		}
		if crossOffset < 0 {
			crossOffset = 0
		}

		var childArea ui.Bounds
		if isRow {
			childArea = ui.Bounds{
				X: area.X + mainCursor,
				Y: area.Y + crossOffset,
				W: info.mainSize,
				H: info.crossSize,
			}
		} else {
			childArea = ui.Bounds{
				X: area.X + crossOffset,
				Y: area.Y + mainCursor,
				W: info.crossSize,
				H: info.mainSize,
			}
		}

		// For expanded children, layout the inner child.
		actualChild := child
		if info.expanded {
			if exp, ok := asExpanded(child); ok {
				actualChild = exp.child
			}
		}
		cb := ctx.LayoutChild(actualChild, childArea)

		if isRow {
			if cb.H > maxCrossActual {
				maxCrossActual = cb.H
			}
		} else {
			if cb.W > maxCrossActual {
				maxCrossActual = cb.W
			}
		}

		mainCursor += info.mainSize + gap + extraGap
	}

	maxMainActual := mainCursor - gap - extraGap
	if maxMainActual < 0 {
		maxMainActual = 0
	}

	if isRow {
		return ui.Bounds{X: area.X, Y: area.Y, W: maxMainActual, H: max(maxCrossActual, maxCross)}
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: max(maxCrossActual, maxCross), H: maxMainActual}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Flex) TreeEqual(other ui.Element) bool {
	o, ok := other.(Flex)
	return ok && n.Direction == o.Direction && n.Justify == o.Justify && n.Align == o.Align && n.Gap == o.Gap && len(n.Children) == len(o.Children)
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
