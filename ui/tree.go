package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
)

// TreeState tracks expand/collapse and selection state for a Tree widget.
type TreeState struct {
	Expanded map[string]bool
	Selected string
	Scroll   ScrollState
}

// NewTreeState creates a ready-to-use TreeState.
func NewTreeState() *TreeState {
	return &TreeState{Expanded: make(map[string]bool)}
}

// IsExpanded reports whether the given node is expanded.
func (ts *TreeState) IsExpanded(id string) bool {
	return ts != nil && ts.Expanded[id]
}

// Toggle flips the expand/collapse state of a node.
func (ts *TreeState) Toggle(id string) {
	if ts == nil {
		return
	}
	ts.Expanded[id] = !ts.Expanded[id]
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
	ID       string
	Depth    int
	HasKids  bool
	Expanded bool
}

func layoutTree(node treeElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState, focus *FocusState) bounds {
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

	// Flatten the visible tree.
	var flat []flatNode
	var walk func(ids []string, depth int)
	walk = func(ids []string, depth int) {
		for _, id := range ids {
			var kids []string
			if node.Children != nil {
				kids = node.Children(id)
			}
			hasKids := len(kids) > 0
			expanded := node.State != nil && node.State.IsExpanded(id)
			flat = append(flat, flatNode{ID: id, Depth: depth, HasKids: hasKids, Expanded: expanded})
			if expanded && hasKids {
				walk(kids, depth+1)
			}
		}
	}
	walk(node.RootIDs, 0)

	contentH := float32(len(flat) * nodeH)

	var offset float32
	if node.State != nil {
		offset = node.State.Scroll.Offset
	}

	// Visible range.
	firstVisible := int(offset) / nodeH
	if firstVisible < 0 {
		firstVisible = 0
	}
	lastVisible := (int(offset) + viewportH) / nodeH
	if lastVisible >= len(flat) {
		lastVisible = len(flat) - 1
	}

	// Clip to viewport.
	canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH)))

	indicatorStyle := draw.TextStyle{
		Size:   float32(nodeH - 8),
		Weight: draw.FontWeightRegular,
	}

	for i := firstVisible; i <= lastVisible; i++ {
		fn := flat[i]
		rowY := area.Y + i*nodeH - int(offset)
		indent := fn.Depth * indentW

		// Selection highlight.
		selected := node.State != nil && node.State.Selected == fn.ID
		if selected {
			canvas.FillRect(
				draw.R(float32(area.X), float32(rowY), float32(area.W), float32(nodeH)),
				draw.SolidPaint(tokens.Colors.Surface.Hovered))
		}

		// Expand/collapse indicator.
		indicatorW := 16
		if fn.HasKids {
			indicator := "▶"
			if fn.Expanded {
				indicator = "▼"
			}
			canvas.DrawText(indicator,
				draw.Pt(float32(area.X+indent), float32(rowY+4)),
				indicatorStyle, tokens.Colors.Text.Secondary)

			// Hit target for expand/collapse toggle.
			if hitMap != nil && node.State != nil {
				id := fn.ID
				ts := node.State
				if hover != nil {
					hover.nextButtonHoverOpacity()
				}
				hitMap.Add(
					draw.R(float32(area.X+indent), float32(rowY), float32(indicatorW), float32(nodeH)),
					func() { ts.Toggle(id) },
				)
			}
		}

		// Node content.
		nodeX := area.X + indent + indicatorW + 4
		nodeArea := bounds{X: nodeX, Y: rowY, W: area.W - indent - indicatorW - 4, H: nodeH}
		layoutElement(node.BuildNode(fn.ID, fn.Depth, fn.Expanded, selected), nodeArea, canvas, tokens, hitMap, hover, focus)

		// Row hit target for selection.
		if hitMap != nil && node.OnSelect != nil {
			id := fn.ID
			onSelect := node.OnSelect
			ts := node.State
			if hover != nil {
				hover.nextButtonHoverOpacity()
			}
			hitMap.Add(
				draw.R(float32(area.X+indent+indicatorW), float32(rowY), float32(area.W-indent-indicatorW), float32(nodeH)),
				func() {
					if ts != nil {
						ts.SetSelected(id)
					}
					onSelect(id)
				},
			)
		}
	}

	canvas.PopClip()

	// Clamp scroll state.
	if node.State != nil {
		maxScroll := contentH - float32(viewportH)
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

	w := area.W

	// Register scroll target.
	if hitMap != nil && node.State != nil && contentH > float32(viewportH) {
		state := &node.State.Scroll
		cH := contentH
		vH := float32(viewportH)
		hitMap.AddScroll(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(viewportH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	// Draw scrollbar.
	if contentH > float32(viewportH) && node.State != nil {
		w += drawScrollbar(canvas, tokens, hitMap, &node.State.Scroll, area.X+w, area.Y, viewportH, contentH, offset)
	}

	return bounds{X: area.X, Y: area.Y, W: w, H: viewportH}
}
