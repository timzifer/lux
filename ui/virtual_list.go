package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// VirtualListConfig configures a VirtualList element.
type VirtualListConfig struct {
	ItemCount  int                   // Total number of items.
	ItemHeight float32               // Uniform height per item in dp.
	BuildItem  func(index int) Element // Builds the element for a given index.
	MaxHeight  float32               // Viewport height in dp.
	State      *ScrollState          // Scroll state (required for scrolling).
}

// VirtualList creates a virtualized list that only renders visible items
// (RFC-002 §5, RFC-001 §13.4 M5).
func VirtualList(config VirtualListConfig) Element {
	return virtualListElement{
		ItemCount:  config.ItemCount,
		ItemHeight: config.ItemHeight,
		BuildItem:  config.BuildItem,
		MaxHeight:  config.MaxHeight,
		State:      config.State,
	}
}

type virtualListElement struct {
	ItemCount  int
	ItemHeight float32
	BuildItem  func(int) Element
	MaxHeight  float32
	State      *ScrollState
}

func (virtualListElement) isElement() {}

const virtualListOverscan = 3

func layoutVirtualList(node virtualListElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, ix *Interactor, overlays *overlayStack, focus *FocusManager) bounds {
	if node.ItemCount <= 0 || node.BuildItem == nil {
		return bounds{X: area.X, Y: area.Y}
	}

	itemH := int(node.ItemHeight)
	if itemH <= 0 {
		itemH = 24
	}

	viewportH := int(node.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	contentH := float32(node.ItemCount * itemH)

	// The list grows to fit its content, capped at viewportH.
	// Only scroll when content exceeds the viewport.
	needsScroll := contentH > float32(viewportH)
	actualH := viewportH
	if !needsScroll {
		actualH = int(contentH)
		if actualH <= 0 {
			actualH = itemH
		}
	}

	// Determine scrollbar width so we can reserve space inside the clip.
	scrollbarW := 0
	if needsScroll {
		scrollbarW = int(tokens.Scroll.TrackWidth)
		if scrollbarW <= 0 {
			scrollbarW = 8
		}
	}

	// Content width excluding the scrollbar.
	contentW := area.W - scrollbarW

	var offset float32
	if node.State != nil {
		offset = node.State.Offset
	}

	// Determine visible range.
	firstVisible := int(offset) / itemH
	if firstVisible < 0 {
		firstVisible = 0
	}
	firstVisible -= virtualListOverscan
	if firstVisible < 0 {
		firstVisible = 0
	}

	lastVisible := (int(offset) + actualH) / itemH
	lastVisible += virtualListOverscan
	if lastVisible >= node.ItemCount {
		lastVisible = node.ItemCount - 1
	}

	// Clip to viewport (including scrollbar space).
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)))

	// Render visible items.
	for i := firstVisible; i <= lastVisible; i++ {
		itemY := area.Y + i*itemH - int(offset)
		child := node.BuildItem(i)
		childArea := bounds{X: area.X, Y: itemY, W: contentW, H: itemH}
		layoutElement(child, childArea, canvas, th, tokens, ix, overlays, focus)
	}

	// Draw scrollbar INSIDE the clip so it's visible even within a parent ScrollView.
	if needsScroll && node.State != nil {
		drawScrollbar(canvas, tokens, ix, node.State, area.X+contentW, area.Y, actualH, contentH, offset)
	}

	canvas.PopClip()

	// Clamp scroll state.
	if node.State != nil {
		maxScroll := contentH - float32(actualH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if node.State.Offset > maxScroll {
			node.State.Offset = maxScroll
		}
		if node.State.Offset < 0 {
			node.State.Offset = 0
		}
	}

	// Register scroll target.
	if node.State != nil && needsScroll {
		state := node.State
		cH := contentH
		vH := float32(actualH)
		ix.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	return bounds{X: area.X, Y: area.Y, W: area.W, H: actualH}
}

// drawScrollbar renders a scrollbar track and thumb, returning the track width consumed.
// Shared by ScrollView, VirtualList, and Tree.
func drawScrollbar(canvas draw.Canvas, tokens theme.TokenSet, ix *Interactor, state *ScrollState, trackX, trackY, viewportH int, contentH, offset float32) int {
	trackW := int(tokens.Scroll.TrackWidth)
	if trackW <= 0 {
		trackW = 8
	}
	thumbR := tokens.Scroll.ThumbRadius
	if thumbR <= 0 {
		thumbR = 4
	}

	trackColor := tokens.Colors.Surface.Hovered
	thumbColor := tokens.Colors.Surface.Pressed

	// Track
	canvas.FillRoundRect(
		draw.R(float32(trackX), float32(trackY), float32(trackW), float32(viewportH)),
		thumbR, draw.SolidPaint(trackColor))

	// Thumb
	ratio := float32(viewportH) / contentH
	thumbH := int(float32(viewportH) * ratio)
	if thumbH < 20 {
		thumbH = 20
	}

	maxScroll := contentH - float32(viewportH)
	thumbTravel := float32(viewportH - thumbH)
	var thumbY float32
	if maxScroll > 0 {
		thumbY = float32(trackY) + (offset/maxScroll)*thumbTravel
	} else {
		thumbY = float32(trackY)
	}

	canvas.FillRoundRect(
		draw.R(float32(trackX), thumbY, float32(trackW), float32(thumbH)),
		thumbR, draw.SolidPaint(thumbColor))

	// Track-click hit target.
	if state != nil {
		st := state
		ms := maxScroll
		tY := float32(trackY)
		vH := float32(viewportH)
		ix.RegisterClickAt(
			draw.R(float32(trackX), float32(trackY), float32(trackW), float32(viewportH)),
			func(_, y float32) {
				frac := (y - tY) / vH
				if frac < 0 {
					frac = 0
				}
				if frac > 1 {
					frac = 1
				}
				st.Offset = frac * ms
			},
		)
	}

	return trackW
}
