// Package data provides data-driven UI components.
package data

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// Tree displays a hierarchical tree widget with expand/collapse
// and selection support.
//
// When DatasetRoots is set, it takes priority over RootIDs.
// When DatasetChildren is set, it takes priority over Children. (RFC-002 §6.2)
type Tree struct {
	ui.BaseElement

	// DatasetRoots provides root IDs with dynamic length support.
	// Takes priority over RootIDs when set.
	DatasetRoots Dataset[string]

	// DatasetChildren returns a Dataset of child IDs for a given parent.
	// Takes priority over Children when set.
	DatasetChildren func(id string) Dataset[string]

	// Legacy API — used when DatasetRoots/DatasetChildren are nil.
	RootIDs     []string
	Children    func(string) []string
	BuildNode   func(string, int, bool, bool) ui.Element
	NodeHeight  float32
	IndentWidth float32
	MaxHeight   float32
	State       *ui.TreeState
	OnSelect    func(string)
}

// NewTree creates a Tree element from a TreeConfig.
func NewTree(config ui.TreeConfig) ui.Element {
	return Tree{
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

// resolvedRootIDs returns root IDs from DatasetRoots or RootIDs.
func (n Tree) resolvedRootIDs() []string {
	if n.DatasetRoots != nil {
		count := n.DatasetRoots.Len()
		if count < 0 {
			// Unknown length — try counter interface (e.g. StreamDataset).
			type counter interface{ Count() int }
			if c, ok := n.DatasetRoots.(counter); ok {
				count = c.Count()
			} else {
				count = 0
			}
		}
		ids := make([]string, 0, count)
		for i := 0; i < count; i++ {
			id, loaded := n.DatasetRoots.Get(i)
			if loaded {
				ids = append(ids, id)
			}
		}
		return ids
	}
	return n.RootIDs
}

// resolvedChildren returns child IDs for a parent from DatasetChildren or Children.
func (n Tree) resolvedChildren(parentID string) []string {
	if n.DatasetChildren != nil {
		ds := n.DatasetChildren(parentID)
		if ds == nil {
			return nil
		}
		count := ds.Len()
		if count < 0 {
			type counter interface{ Count() int }
			if c, ok := ds.(counter); ok {
				count = c.Count()
			} else {
				count = 0
			}
		}
		ids := make([]string, 0, count)
		for i := 0; i < count; i++ {
			id, loaded := ds.Get(i)
			if loaded {
				ids = append(ids, id)
			}
		}
		return ids
	}
	if n.Children != nil {
		return n.Children(parentID)
	}
	return nil
}

// flatNode is a node in the flattened visible tree.
type flatNode struct {
	ID             string
	Depth          int
	HasKids        bool
	Expanded       bool
	HeightFraction float32 // 0..1, animated expand progress for children-of-animating-parent
}

// LayoutSelf implements ui.Layouter.
func (n Tree) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	rootIDs := n.resolvedRootIDs()
	if len(rootIDs) == 0 || n.BuildNode == nil {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Cache motion tokens so Toggle() picks them up.
	if n.State != nil {
		n.State.CacheMotion(ctx.Tokens.Motion.Standard.Duration, ctx.Tokens.Motion.Standard.Easing)
	}

	nodeH := int(n.NodeHeight)
	if nodeH <= 0 {
		nodeH = 28
	}
	indentW := int(n.IndentWidth)
	if indentW <= 0 {
		indentW = 20
	}

	viewportH := int(n.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	// Flatten the visible tree, including children of animating nodes.
	var flat []flatNode
	var walk func(ids []string, depth int, parentFraction float32)
	walk = func(ids []string, depth int, parentFraction float32) {
		for _, id := range ids {
			kids := n.resolvedChildren(id)
			hasKids := len(kids) > 0
			expanded := n.State != nil && n.State.IsExpanded(id)
			flat = append(flat, flatNode{ID: id, Depth: depth, HasKids: hasKids, Expanded: expanded, HeightFraction: parentFraction})

			if !hasKids {
				continue
			}

			// Include children if expanded OR if collapse animation is still running.
			progress := float32(0)
			if n.State != nil {
				progress = n.State.ExpandProgress(id)
			}
			childFraction := parentFraction * progress

			if expanded || (n.State != nil && n.State.IsAnimating(id)) {
				walk(kids, depth+1, childFraction)
			}
		}
	}
	walk(rootIDs, 0, 1.0)

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
		scrollbarW = int(ctx.Tokens.Scroll.TrackWidth)
		if scrollbarW <= 0 {
			scrollbarW = 8
		}
	}

	var offset float32
	if n.State != nil {
		offset = n.State.Scroll.Offset
	}

	// Content width excluding the scrollbar.
	contentW := area.W - scrollbarW

	// Clip to the viewport (including scrollbar space).
	ctx.Canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)))

	indicatorSize := float32(nodeH) * 0.5
	indicatorStyle := draw.TextStyle{
		FontFamily: "Phosphor",
		Size:       indicatorSize,
		Weight:     draw.FontWeightRegular,
	}
	indicatorCellSize := int(math.Ceil(float64(indicatorSize)))
	indicatorOffsetY := (nodeH - indicatorCellSize) / 2

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
			// Still need to consume hover/hit slots for off-screen targets.
			if fn.HasKids {
				ctx.IX.RegisterHit(draw.Rect{}, nil)
			}
			if n.OnSelect != nil {
				ctx.IX.RegisterHit(draw.Rect{}, nil)
			}
			continue
		}

		indent := fn.Depth * indentW

		// Clip partially visible (animating) rows.
		partialClip := fn.HeightFraction < 1.0
		if partialClip {
			ctx.Canvas.PushClip(draw.R(float32(area.X), rowY, float32(area.W), effectiveH))
		}

		// Selection highlight.
		selected := n.State != nil && n.State.Selected == fn.ID
		if selected {
			ctx.Canvas.FillRect(
				draw.R(float32(area.X), rowY, float32(contentW), float32(nodeH)),
				draw.SolidPaint(ctx.Tokens.Colors.Surface.Hovered))
		}

		// Expand/collapse indicator.
		indicatorW := 16
		if fn.HasKids {
			indicator := icons.CaretRight
			if fn.Expanded {
				indicator = icons.CaretDown
			}
			ctx.Canvas.DrawText(indicator,
				draw.Pt(float32(area.X+indent), rowY+float32(indicatorOffsetY)),
				indicatorStyle, ctx.Tokens.Colors.Text.Secondary)

			// Hit target for expand/collapse toggle.
			if n.State != nil {
				id := fn.ID
				ts := n.State
				ctx.IX.RegisterHit(
					draw.R(float32(area.X+indent), rowY, float32(indicatorW), effectiveH),
					func() { ts.Toggle(id) },
				)
			} else {
				ctx.IX.RegisterHit(
					draw.R(float32(area.X+indent), rowY, float32(indicatorW), effectiveH),
					nil,
				)
			}
		}

		// Node content — vertically centered within the row.
		nodeX := area.X + indent + indicatorW + 4
		nodeW := contentW - indent - indicatorW - 4
		nodeContent := n.BuildNode(fn.ID, fn.Depth, fn.Expanded, selected)
		cb := ctx.MeasureChild(nodeContent, ui.Bounds{X: nodeX, Y: int(rowY), W: nodeW, H: nodeH})
		nodeOffsetY := (nodeH - cb.H) / 2
		ctx.LayoutChild(nodeContent, ui.Bounds{X: nodeX, Y: int(rowY) + nodeOffsetY, W: nodeW, H: nodeH})

		// Row hit target for selection.
		if n.OnSelect != nil {
			id := fn.ID
			onSelect := n.OnSelect
			ts := n.State
			ctx.IX.RegisterHit(
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
			ctx.Canvas.PopClip()
		}
	}

	// Draw scrollbar INSIDE the clip so it's visible even within a parent ScrollView.
	if needsScroll && n.State != nil {
		ui.DrawScrollbar(ctx.Canvas, ctx.Tokens, ctx.IX, &n.State.Scroll, area.X+contentW, area.Y, actualH, contentH, offset)
	}

	ctx.Canvas.PopClip()

	// Clamp scroll state.
	if n.State != nil {
		maxScroll := contentH - float32(actualH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if n.State.Scroll.Offset > maxScroll {
			n.State.Scroll.Offset = maxScroll
		}
		if n.State.Scroll.Offset < 0 {
			n.State.Scroll.Offset = 0
		}
	}

	// Register scroll target.
	if n.State != nil && needsScroll {
		state := &n.State.Scroll
		cH := contentH
		vH := float32(actualH)
		ctx.IX.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(actualH)),
			cH, vH,
			func(deltaY float32) { state.ScrollBy(deltaY, cH, vH) },
		)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: actualH}
}

// TreeEqual implements ui.TreeEqualizer. Tree is always unequal (dynamic content).
func (n Tree) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver. Tree is a leaf in resolution
// (children are built dynamically via BuildNode).
func (n Tree) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. Builds a11y tree nodes for Tree.
func (n Tree) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	treeIdx := b.AddNode(a11y.AccessNode{Role: a11y.RoleTree, Label: "Tree"}, parentIdx, a11y.Rect{})
	walkTreeNodes(b, n, n.resolvedRootIDs(), int32(treeIdx), 0)
}

// walkTreeNodes recursively walks tree nodes and adds them to the access tree.
func walkTreeNodes(b *ui.AccessTreeBuilder, tree Tree, ids []string, parentIdx int32, depth int) {
	for _, id := range ids {
		expanded := tree.State != nil && tree.State.IsExpanded(id)
		selected := tree.State != nil && tree.State.Selected == id

		kids := tree.resolvedChildren(id)
		hasKids := len(kids) > 0

		// Build the display element to extract a label.
		var label string
		if tree.BuildNode != nil {
			nodeEl := tree.BuildNode(id, depth, expanded, selected)
			label = ui.ExtractElementLabel(nodeEl)
		}
		if label == "" {
			label = id
		}

		an := a11y.AccessNode{
			Role:  a11y.RoleGroup,
			Label: label,
			States: a11y.AccessStates{
				Selected: selected,
				Expanded: hasKids && expanded,
			},
		}
		if tree.OnSelect != nil {
			selectID := id
			onSelect := tree.OnSelect
			an.Actions = []a11y.AccessAction{{Name: "activate", Trigger: func() { onSelect(selectID) }}}
		}

		idx := b.AddNode(an, parentIdx, a11y.Rect{})

		// Recurse into expanded children.
		if hasKids && expanded {
			walkTreeNodes(b, tree, kids, int32(idx), depth+1)
		}
	}
}
