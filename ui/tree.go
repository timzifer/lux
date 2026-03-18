package ui

import (
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui/icons"
)

// TreeState tracks expand/collapse and selection state for a Tree widget.
type TreeState struct {
	Expanded   map[string]bool
	Selected   string
	Scroll     ScrollState
	expandAnim map[string]*anim.Anim[float32] // per-node expand animation (0=collapsed, 1=expanded)
}

// NewTreeState creates a ready-to-use TreeState.
func NewTreeState() *TreeState {
	return &TreeState{
		Expanded:   make(map[string]bool),
		expandAnim: make(map[string]*anim.Anim[float32]),
	}
}

// IsExpanded reports whether the given node is expanded.
func (ts *TreeState) IsExpanded(id string) bool {
	return ts != nil && ts.Expanded[id]
}

// Toggle flips the expand/collapse state of a node with animation.
func (ts *TreeState) Toggle(id string) {
	if ts == nil {
		return
	}
	ts.Expanded[id] = !ts.Expanded[id]

	a := ts.getOrCreateAnim(id)
	if ts.Expanded[id] {
		a.SetTarget(1.0, 150*time.Millisecond, anim.OutCubic)
	} else {
		a.SetTarget(0.0, 150*time.Millisecond, anim.OutCubic)
	}
}

// expandProgress returns the current expand animation progress for a node.
// Returns 1.0 if expanded (no animation), 0.0 if collapsed (no animation).
func (ts *TreeState) expandProgress(id string) float32 {
	if ts == nil {
		return 0
	}
	if a, ok := ts.expandAnim[id]; ok {
		return a.Value()
	}
	if ts.Expanded[id] {
		return 1.0
	}
	return 0.0
}

// isAnimating reports whether a node's expand/collapse is currently animating.
func (ts *TreeState) isAnimating(id string) bool {
	if ts == nil {
		return false
	}
	a, ok := ts.expandAnim[id]
	return ok && !a.IsDone()
}

func (ts *TreeState) getOrCreateAnim(id string) *anim.Anim[float32] {
	if ts.expandAnim == nil {
		ts.expandAnim = make(map[string]*anim.Anim[float32])
	}
	a, ok := ts.expandAnim[id]
	if !ok {
		a = &anim.Anim[float32]{}
		// Initialize to current state.
		if ts.Expanded[id] {
			// We just toggled TO expanded, so start from 0.
			a.SetImmediate(0.0)
		} else {
			// We just toggled TO collapsed, so start from 1.
			a.SetImmediate(1.0)
		}
		ts.expandAnim[id] = a
	}
	return a
}

// Tick advances all expand/collapse animations by dt.
func (ts *TreeState) Tick(dt time.Duration) {
	if ts == nil {
		return
	}
	for id, a := range ts.expandAnim {
		if !a.Tick(dt) {
			// Animation done — clean up.
			delete(ts.expandAnim, id)
		}
	}
}

// SetSelected sets the currently selected node.
func (ts *TreeState) SetSelected(id string) {
	if ts != nil {
		ts.Selected = id
	}
}

// TreeConfig configures a Tree element.
type TreeConfig struct {
	RootIDs     []string
	Children    func(id string) []string                              // returns child IDs; nil/empty = leaf
	BuildNode   func(id string, depth int, expanded, selected bool) Element // builds the display for a node
	NodeHeight  float32                                                // uniform height per node (dp); 0 = 28dp
	IndentWidth float32                                                // per-level indent (dp); 0 = 20dp
	MaxHeight   float32                                                // viewport height (dp)
	State       *TreeState
	OnSelect    func(id string) // called when a node is clicked
}

// Tree creates a hierarchical tree widget with expand/collapse
// and selection support (RFC-002 §5.2, RFC-001 §13.4 M5).
func Tree(config TreeConfig) Element {
	return treeElement{
		RootIDs:     config.RootIDs,
		Children:    config.Children,
		BuildNode:   config.BuildNode,
		NodeHeight:  config.NodeHeight,
		IndentWidth: config.IndentWidth,
		MaxHeight:   config.MaxHeight,
		State:       config.State,
		OnSelect:    config.OnSelect,
	}
}

type treeElement struct {
	RootIDs     []string
	Children    func(string) []string
	BuildNode   func(string, int, bool, bool) Element
	NodeHeight  float32
	IndentWidth float32
	MaxHeight   float32
	State       *TreeState
	OnSelect    func(string)
}

func (treeElement) isElement() {}

// flatNode is a node in the flattened visible tree.
type flatNode struct {
	ID             string
	Depth          int
	HasKids        bool
	Expanded       bool
	HeightFraction float32 // 0..1, animated expand progress for children-of-animating-parent
}

func layoutTree(node treeElement, area bounds, canvas draw.Canvas, th theme.Theme, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, overlays *overlayStack, focus *FocusState) bounds {
	if len(node.RootIDs) == 0 || node.BuildNode == nil {
		return bounds{X: area.X, Y: area.Y}
	}

	nodeH := int(node.NodeHeight)
	if nodeH <= 0 {
		nodeH = 28
	}
	indentW := int(node.IndentWidth)
	if indentW <= 0 {
		indentW = 20
	}

	viewportH := int(node.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	// Flatten the visible tree, including children of animating nodes.
	var flat []flatNode
	var walk func(ids []string, depth int, parentFraction float32)
	walk = func(ids []string, depth int, parentFraction float32) {
		for _, id := range ids {
			var kids []string
			if node.Children != nil {
				kids = node.Children(id)
			}
			hasKids := len(kids) > 0
			expanded := node.State != nil && node.State.IsExpanded(id)
			flat = append(flat, flatNode{ID: id, Depth: depth, HasKids: hasKids, Expanded: expanded, HeightFraction: parentFraction})

			if !hasKids {
				continue
			}

			// Include children if expanded OR if collapse animation is still running.
			progress := float32(0)
			if node.State != nil {
				progress = node.State.expandProgress(id)
			}
			childFraction := parentFraction * progress

			if expanded || (node.State != nil && node.State.isAnimating(id)) {
				walk(kids, depth+1, childFraction)
			}
		}
	}
	walk(node.RootIDs, 0, 1.0)

	// Compute total content height considering animation fractions.
	var contentH float32
	for _, fn := range flat {
		contentH += float32(nodeH) * fn.HeightFraction
	}

	// The tree grows to fit its content, capped at viewportH.
	// Only scroll when content exceeds the viewport.
	needsScroll := contentH > float32(viewportH)
	actualH := viewportH
	if !needsScroll {
		actualH = int(contentH)
		if actualH <= 0 {
			actualH = nodeH
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

	var offset float32
	if node.State != nil {
		offset = node.State.Scroll.Offset
	}

	// Content width excluding the scrollbar.
	contentW := area.W - scrollbarW

	// Clip to the viewport (including scrollbar space).
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)))

	indicatorStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       float32(nodeH - 8),
		Weight:     draw.FontWeightRegular,
	}

	// Compute vertical centering offset for node content.
	bodyMetrics := canvas.MeasureText("Ag", tokens.Typography.Body)
	textH := int(bodyMetrics.Ascent + 0.5)
	centerY := (nodeH - textH) / 2

	// Compute cumulative y-positions for each row, considering animation.
	rowYPositions := make([]float32, len(flat))
	cumY := float32(0)
	for i, fn := range flat {
		rowYPositions[i] = cumY
		cumY += float32(nodeH) * fn.HeightFraction
	}

	for i, fn := range flat {
		effectiveH := float32(nodeH) * fn.HeightFraction
		if effectiveH < 0.5 {
			continue // Too small to render.
		}

		rowY := float32(area.Y) + rowYPositions[i] - offset
		rowYBottom := rowY + effectiveH

		// Skip rows outside the viewport.
		if rowYBottom < float32(area.Y) || rowY >= float32(area.Y+actualH) {
			// Still need to consume hover indices for hit targets.
			if fn.HasKids && hover != nil {
				hover.nextButtonHoverOpacity()
			}
			if node.OnSelect != nil && hover != nil {
				hover.nextButtonHoverOpacity()
			}
			continue
		}

		indent := fn.Depth * indentW

		// Clip partially visible (animating) rows.
		partialClip := fn.HeightFraction < 1.0
		if partialClip {
			canvas.PushClip(draw.R(float32(area.X), rowY, float32(area.W), effectiveH))
		}

		// Selection highlight.
		selected := node.State != nil && node.State.Selected == fn.ID
		if selected {
			canvas.FillRect(
				draw.R(float32(area.X), rowY, float32(contentW), float32(nodeH)),
				draw.SolidPaint(tokens.Colors.Surface.Hovered))
		}

		// Expand/collapse indicator.
		indicatorW := 16
		if fn.HasKids {
			indicator := icons.CaretRight
			if fn.Expanded {
				indicator = icons.CaretDown
			}
			canvas.DrawText(indicator,
				draw.Pt(float32(area.X+indent), rowY+4),
				indicatorStyle, tokens.Colors.Text.Secondary)

			// Hit target for expand/collapse toggle.
			if hitMap != nil && node.State != nil {
				id := fn.ID
				ts := node.State
				if hover != nil {
					hover.nextButtonHoverOpacity()
				}
				hitMap.Add(
					draw.R(float32(area.X+indent), rowY, float32(indicatorW), effectiveH),
					func() { ts.Toggle(id) },
				)
			}
		}

		// Node content — vertically centered within the row.
		nodeX := area.X + indent + indicatorW + 4
		nodeArea := bounds{X: nodeX, Y: int(rowY) + centerY, W: contentW - indent - indicatorW - 4, H: nodeH}
		layoutElement(node.BuildNode(fn.ID, fn.Depth, fn.Expanded, selected), nodeArea, canvas, th, tokens, hitMap, hover, overlays, focus)

		// Row hit target for selection.
		if hitMap != nil && node.OnSelect != nil {
			id := fn.ID
			onSelect := node.OnSelect
			ts := node.State
			if hover != nil {
				hover.nextButtonHoverOpacity()
			}
			hitMap.Add(
				draw.R(float32(area.X+indent+indicatorW), rowY, float32(contentW-indent-indicatorW), effectiveH),
				func() {
					if ts != nil {
						ts.SetSelected(id)
					}
					onSelect(id)
				},
			)
		}

		if partialClip {
			canvas.PopClip()
		}
	}

	// Draw scrollbar INSIDE the clip so it's visible even within a parent ScrollView.
	if needsScroll && node.State != nil {
		drawScrollbar(canvas, tokens, hitMap, &node.State.Scroll, area.X+contentW, area.Y, actualH, contentH, offset)
	}

	canvas.PopClip()

	// Clamp scroll state.
	if node.State != nil {
		maxScroll := contentH - float32(actualH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if node.State.Scroll.Offset > maxScroll {
			node.State.Scroll.Offset = maxScroll
		}
		if node.State.Scroll.Offset < 0 {
			node.State.Scroll.Offset = 0
		}
	}

	// Register scroll target.
	if hitMap != nil && node.State != nil && needsScroll {
		state := &node.State.Scroll
		cH := contentH
		vH := float32(actualH)
		hitMap.AddScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	return bounds{X: area.X, Y: area.Y, W: area.W, H: actualH}
}
