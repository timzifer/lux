package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
)

// ── SplitView constants ─────────────────────────────────────────

const (
	splitDividerDefault = 6  // dp — drag hit area width
	splitDividerLine    = 1  // dp — visible line thickness
	splitMinPane        = 32 // dp — minimum pane size
)

// ── Element struct ──────────────────────────────────────────────

type splitViewElement struct {
	First       Element
	Second      Element
	Axis        LayoutAxis // AxisRow = side-by-side (vertical divider), AxisColumn = stacked (horizontal divider)
	Ratio       float32    // 0.0–1.0, proportion of space for First panel
	OnResize    func(float32)
	DividerSize float32 // drag-area width in dp; 0 = use splitDividerDefault
}

func (splitViewElement) isElement() {}

// ── Constructor & Options ───────────────────────────────────────

// SplitViewOption configures a SplitView element.
type SplitViewOption func(*splitViewElement)

// WithSplitAxis sets the split orientation. AxisRow (default) places panels
// side-by-side with a vertical divider; AxisColumn stacks them with a
// horizontal divider.
func WithSplitAxis(axis LayoutAxis) SplitViewOption {
	return func(e *splitViewElement) { e.Axis = axis }
}

// WithDividerSize sets the drag-area width in dp (default 6).
func WithDividerSize(size float32) SplitViewOption {
	return func(e *splitViewElement) { e.DividerSize = size }
}

// SplitView creates a resizable split panel. The ratio (0.0–1.0) controls
// how much space the first panel receives. onResize is called with the new
// ratio during drag; pass nil for a fixed (non-draggable) split.
func SplitView(first, second Element, ratio float32, onResize func(float32), opts ...SplitViewOption) Element {
	el := splitViewElement{
		First:    first,
		Second:   second,
		Axis:     AxisRow,
		Ratio:    ratio,
		OnResize: onResize,
	}
	for _, opt := range opts {
		opt(&el)
	}
	return el
}

// ── Layout ──────────────────────────────────────────────────────

func layoutSplitView(node splitViewElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusManager) bounds {
	divSize := node.DividerSize
	if divSize <= 0 {
		divSize = splitDividerDefault
	}

	// Clamp ratio.
	ratio := node.Ratio
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	horizontal := node.Axis == AxisRow // side-by-side panels

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

	var firstArea, secondArea bounds
	var divRect draw.Rect

	if horizontal {
		firstW := int(firstSize)
		secondW := int(secondSize)
		divX := float32(area.X) + firstSize

		firstArea = bounds{X: area.X, Y: area.Y, W: firstW, H: area.H}
		secondArea = bounds{X: area.X + firstW + int(divPx), Y: area.Y, W: secondW, H: area.H}
		divRect = draw.R(divX, float32(area.Y), divPx, float32(area.H))
	} else {
		firstH := int(firstSize)
		secondH := int(secondSize)
		divY := float32(area.Y) + firstSize

		firstArea = bounds{X: area.X, Y: area.Y, W: area.W, H: firstH}
		secondArea = bounds{X: area.X, Y: area.Y + firstH + int(divPx), W: area.W, H: secondH}
		divRect = draw.R(float32(area.X), divY, float32(area.W), divPx)
	}

	// Layout first child.
	layoutElement(node.First, firstArea, canvas, th, tokens, hitMap, hover, overlays, focus)

	// Draw divider line (centered within the drag area).
	lineColor := tokens.Colors.Stroke.Divider
	if horizontal {
		lineX := divRect.X + (divPx-splitDividerLine)/2
		canvas.FillRect(draw.R(lineX, divRect.Y, splitDividerLine, divRect.H), draw.SolidPaint(lineColor))
	} else {
		lineY := divRect.Y + (divPx-splitDividerLine)/2
		canvas.FillRect(draw.R(divRect.X, lineY, divRect.W, splitDividerLine), draw.SolidPaint(lineColor))
	}

	// Layout second child.
	layoutElement(node.Second, secondArea, canvas, th, tokens, hitMap, hover, overlays, focus)

	// Register drag target for divider.
	if hitMap != nil && node.OnResize != nil {
		onResize := node.OnResize
		areaStart := float32(area.X)
		if !horizontal {
			areaStart = float32(area.Y)
		}

		cursor := input.CursorResizeEW
		if !horizontal {
			cursor = input.CursorResizeNS
		}

		hitMap.AddDragCursor(divRect, cursor, func(x, y float32) {
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

	return bounds{X: area.X, Y: area.Y, W: area.W, H: area.H}
}
