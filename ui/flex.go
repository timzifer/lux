package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// ── Flex Types (RFC-002 §4.4) ─────────────────────────────────────

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
type FlexOption func(*flexElement)

// Flex creates a flexible layout container (RFC-002 §4.4).
func Flex(children []Element, opts ...FlexOption) Element {
	el := flexElement{
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
	return func(e *flexElement) { e.Direction = d }
}

// WithJustify sets main-axis alignment.
func WithJustify(j Justify) FlexOption {
	return func(e *flexElement) { e.Justify = j }
}

// WithAlign sets cross-axis alignment.
func WithAlign(a Align) FlexOption {
	return func(e *flexElement) { e.Align = a }
}

// WithGap sets the gap between children in dp.
func WithGap(gap float32) FlexOption {
	return func(e *flexElement) { e.Gap = gap }
}

type flexElement struct {
	Direction FlexDirection
	Justify   Justify
	Align     Align
	Gap       float32
	Children  []Element
}

func (flexElement) isElement() {}

// layoutFlex performs a two-pass layout: measure with nullCanvas, then paint.
func layoutFlex(node flexElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	n := len(node.Children)
	if n == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	isRow := node.Direction == FlexRow
	gap := int(node.Gap)

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
	infos := make([]childInfo, n)
	nc := nullCanvas{delegate: canvas}
	totalFixed := 0
	totalGrow := float32(0)
	fixedCount := 0

	for i, child := range node.Children {
		if exp, ok := child.(expandedElement); ok {
			infos[i] = childInfo{expanded: true, grow: exp.Grow}
			totalGrow += exp.Grow
		} else {
			// Measure with nullCanvas (no paint).
			cb := layoutElement(child, area, nc, th, tokens, nil, nil)
			if isRow {
				infos[i] = childInfo{mainSize: cb.W, crossSize: cb.H}
			} else {
				infos[i] = childInfo{mainSize: cb.H, crossSize: cb.W}
			}
			totalFixed += infos[i].mainSize
			fixedCount++
		}
	}

	// Gaps between children.
	totalGaps := 0
	if n > 1 {
		totalGaps = gap * (n - 1)
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
			exp := node.Children[i].(expandedElement)
			var measureArea bounds
			if isRow {
				measureArea = bounds{X: 0, Y: 0, W: infos[i].mainSize, H: area.H}
			} else {
				measureArea = bounds{X: 0, Y: 0, W: area.W, H: infos[i].mainSize}
			}
			cb := layoutElement(exp.Child, measureArea, nc, th, tokens, nil, nil)
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

	// Apply Justify: compute start offset and extra spacing.
	freeSpace := mainAvail - totalMain
	if freeSpace < 0 {
		freeSpace = 0
	}
	startOffset := 0
	extraGap := 0
	switch node.Justify {
	case JustifyEnd:
		startOffset = freeSpace
	case JustifyCenter:
		startOffset = freeSpace / 2
	case JustifySpaceBetween:
		if n > 1 {
			extraGap = freeSpace / (n - 1)
		}
	case JustifySpaceAround:
		if n > 0 {
			extraGap = freeSpace / n
			startOffset = extraGap / 2
		}
	case JustifySpaceEvenly:
		if n > 0 {
			extraGap = freeSpace / (n + 1)
			startOffset = extraGap
		}
	}

	// Pass 2: paint children at computed positions.
	mainCursor := startOffset
	maxCrossActual := 0
	maxMainActual := 0

	for i, child := range node.Children {
		info := infos[i]

		// Compute cross-axis offset based on Align.
		crossOffset := 0
		crossAvail := maxCross
		if isRow {
			crossAvail = area.H
		} else {
			crossAvail = area.W
		}
		switch node.Align {
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

		var childArea bounds
		if isRow {
			childArea = bounds{
				X: area.X + mainCursor,
				Y: area.Y + crossOffset,
				W: info.mainSize,
				H: info.crossSize,
			}
		} else {
			childArea = bounds{
				X: area.X + crossOffset,
				Y: area.Y + mainCursor,
				W: info.crossSize,
				H: info.mainSize,
			}
		}

		// For expanded children, layout the inner child.
		actualChild := child
		if info.expanded {
			actualChild = child.(expandedElement).Child
		}
		cb := layoutElement(actualChild, childArea, canvas, th, tokens, ix, overlays, focus)

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

	maxMainActual = mainCursor - gap - extraGap
	if maxMainActual < 0 {
		maxMainActual = 0
	}

	if isRow {
		return bounds{X: area.X, Y: area.Y, W: maxMainActual, H: max(maxCrossActual, maxCross)}
	}
	return bounds{X: area.X, Y: area.Y, W: max(maxCrossActual, maxCross), H: maxMainActual}
}
