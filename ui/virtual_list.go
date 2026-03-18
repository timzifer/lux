package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
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

func layoutVirtualList(node virtualListElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
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

	lastVisible := (int(offset) + viewportH) / itemH
	lastVisible += virtualListOverscan
	if lastVisible >= node.ItemCount {
		lastVisible = node.ItemCount - 1
	}

	// Clip to viewport.
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH)))

	// Render visible items.
	for i := firstVisible; i <= lastVisible; i++ {
		itemY := area.Y + i*itemH - int(offset)
		child := node.BuildItem(i)
		childArea := bounds{X: area.X, Y: itemY, W: area.W, H: itemH}
		layoutElement(child, childArea, canvas, tokens, hitMap, hover, overlays, focus)
	}

	canvas.PopClip()

	// Clamp scroll state.
	if node.State != nil {
		maxScroll := contentH - float32(viewportH)
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

	w := area.W

	// Register scroll target.
	if hitMap != nil && node.State != nil && contentH > float32(viewportH) {
		state := node.State
		cH := contentH
		vH := float32(viewportH)
		hitMap.AddScroll(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(viewportH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	// Draw scrollbar if content exceeds viewport.
	if contentH > float32(viewportH) {
		w += drawScrollbar(canvas, tokens, hitMap, node.State, area.X+w, area.Y, viewportH, contentH, offset)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: viewportH}
}

// drawScrollbar renders a scrollbar track and thumb, returning the track width consumed.
// Shared by ScrollView, VirtualList, and Tree.
func drawScrollbar(canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, state *ScrollState, trackX, trackY, viewportH int, contentH, offset float32) int {
	trackW := int(tokens.Scroll.TrackWidth)
	if trackW <= 0 {
		trackW = 8
	}
	thumbR := tokens.Scroll.ThumbRadius
	if thumbR <= 0 {
		thumbR = 4
	}

	trackColor := tokens.Colors.Stroke.Divider
	thumbColor := tokens.Colors.Text.Secondary

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
	if hitMap != nil && state != nil {
		st := state
		ms := maxScroll
		tY := float32(trackY)
		vH := float32(viewportH)
		hitMap.AddAt(
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
