package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// SplitView constants.
const (
	splitDividerDefault = 10 // dp — drag hit area width
	splitDividerLine    = 2  // dp — visible line thickness
	splitMinPane        = 32 // dp — minimum pane size
)

// SplitViewOption configures a SplitView element.
type SplitViewOption func(*SplitView)

// WithSplitAxis sets the split orientation. AxisRow (default) places panels
// side-by-side with a vertical divider; AxisColumn stacks them with a
// horizontal divider.
func WithSplitAxis(axis ui.LayoutAxis) SplitViewOption {
	return func(e *SplitView) { e.Axis = axis }
}

// WithDividerSize sets the drag-area width in dp (default 10).
func WithDividerSize(size float32) SplitViewOption {
	return func(e *SplitView) { e.DividerSize = size }
}

// SplitView displays two resizable panes separated by a draggable divider.
type SplitView struct {
	ui.BaseElement
	First       ui.Element
	Second      ui.Element
	Axis        ui.LayoutAxis // AxisRow = side-by-side, AxisColumn = stacked
	Ratio       float32       // 0.0–1.0, proportion of space for First panel
	OnResize    func(float32)
	DividerSize float32 // drag-area width in dp; 0 = use splitDividerDefault
}

// NewSplitView creates a SplitView element. ratio (0.0–1.0) controls
// how much space the first panel receives. onResize is called with the new
// ratio during drag; pass nil for a fixed (non-draggable) split.
func NewSplitView(first, second ui.Element, ratio float32, onResize func(float32), opts ...SplitViewOption) ui.Element {
	el := SplitView{
		First:    first,
		Second:   second,
		Axis:     ui.AxisRow,
		Ratio:    ratio,
		OnResize: onResize,
	}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n SplitView) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	divSize := n.DividerSize
	if divSize <= 0 {
		divSize = splitDividerDefault
	}

	// Clamp ratio.
	ratio := n.Ratio
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	horizontal := n.Axis == ui.AxisRow // side-by-side panels

	var totalSize float32
	if horizontal {
		totalSize = float32(area.W)
	} else {
		totalSize = float32(area.H)
	}

	divPx := divSize
	available := totalSize - divPx
	if available < 0 {
		available = 0
	}

	// Compute first-panel size, clamped to min pane.
	firstSize := available * ratio
	minPane := float32(splitMinPane)
	maxFirst := available - minPane
	if maxFirst < minPane {
		// Not enough room for two min panes — give everything to first.
		maxFirst = available
		minPane = 0
	}
	if firstSize < minPane {
		firstSize = minPane
	}
	if firstSize > maxFirst {
		firstSize = maxFirst
	}
	secondSize := available - firstSize

	var firstArea, secondArea ui.Bounds
	var divRect draw.Rect

	if horizontal {
		firstW := int(firstSize)
		secondW := int(secondSize)
		divX := float32(area.X) + firstSize

		// Measure children to determine actual height.
		m1 := ctx.MeasureChild(n.First, ui.Bounds{X: area.X, Y: area.Y, W: firstW, H: area.H})
		m2 := ctx.MeasureChild(n.Second, ui.Bounds{X: area.X + firstW + int(divPx), Y: area.Y, W: secondW, H: area.H})
		paneH := max(m1.H, m2.H)
		if paneH <= 0 {
			paneH = area.H // children have no intrinsic height — use available space
		} else if paneH > area.H {
			paneH = area.H
		}

		firstArea = ui.Bounds{X: area.X, Y: area.Y, W: firstW, H: paneH}
		secondArea = ui.Bounds{X: area.X + firstW + int(divPx), Y: area.Y, W: secondW, H: paneH}
		divRect = draw.R(divX, float32(area.Y), divPx, float32(paneH))
	} else {
		firstH := int(firstSize)
		secondH := int(secondSize)
		divY := float32(area.Y) + firstSize

		firstArea = ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: firstH}
		secondArea = ui.Bounds{X: area.X, Y: area.Y + firstH + int(divPx), W: area.W, H: secondH}
		divRect = draw.R(float32(area.X), divY, float32(area.W), divPx)
	}

	// Layout first child (clipped to its pane).
	ctx.Canvas.PushClip(draw.R(float32(firstArea.X), float32(firstArea.Y), float32(firstArea.W), float32(firstArea.H)))
	firstBounds := ctx.LayoutChild(n.First, firstArea)
	ctx.Canvas.PopClip()

	// Draw divider line (centered within the drag area).
	lineColor := ctx.Tokens.Colors.Surface.Hovered
	if horizontal {
		lineX := divRect.X + (divPx-splitDividerLine)/2
		ctx.Canvas.FillRect(draw.R(lineX, divRect.Y, splitDividerLine, divRect.H), draw.SolidPaint(lineColor))
	} else {
		lineY := divRect.Y + (divPx-splitDividerLine)/2
		ctx.Canvas.FillRect(draw.R(divRect.X, lineY, divRect.W, splitDividerLine), draw.SolidPaint(lineColor))
	}

	// Layout second child (clipped to its pane).
	ctx.Canvas.PushClip(draw.R(float32(secondArea.X), float32(secondArea.Y), float32(secondArea.W), float32(secondArea.H)))
	secondBounds := ctx.LayoutChild(n.Second, secondArea)
	ctx.Canvas.PopClip()

	// Register drag target for divider via Interactor to keep hover indices aligned.
	if n.OnResize != nil {
		onResize := n.OnResize
		areaStart := float32(area.X)
		if !horizontal {
			areaStart = float32(area.Y)
		}

		cursor := input.CursorResizeEW
		if !horizontal {
			cursor = input.CursorResizeNS
		}

		ctx.IX.RegisterDragCursor(divRect, cursor, func(x, y float32) {
			var pos float32
			if horizontal {
				pos = x
			} else {
				pos = y
			}
			newFirst := pos - areaStart
			if newFirst < minPane {
				newFirst = minPane
			}
			if newFirst > maxFirst {
				newFirst = maxFirst
			}
			var newRatio float32
			if available > 0 {
				newRatio = newFirst / available
			}
			onResize(newRatio)
		})
	}

	// Compute actual size from children.
	var resultW, resultH int
	if horizontal {
		resultW = area.W
		resultH = max(firstBounds.H, secondBounds.H)
	} else {
		resultW = max(firstBounds.W, secondBounds.W)
		resultH = int(firstSize) + int(divPx) + int(secondSize)
	}
	return ui.Bounds{X: area.X, Y: area.Y, W: resultW, H: resultH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n SplitView) TreeEqual(other ui.Element) bool {
	o, ok := other.(SplitView)
	if !ok {
		return false
	}
	return n.Ratio == o.Ratio && n.Axis == o.Axis && n.DividerSize == o.DividerSize
}

// ResolveChildren implements ui.ChildResolver.
func (n SplitView) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.First = resolve(n.First, 0)
	n.Second = resolve(n.Second, 1)
	return n
}

// WalkAccess implements ui.AccessWalker. Walks both child panes.
func (n SplitView) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.First, parentIdx)
	b.Walk(n.Second, parentIdx)
}
