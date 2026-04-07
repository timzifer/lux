package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// ScrollView provides a scrollable viewport around a single child element.
type ScrollView struct {
	ui.BaseElement
	Child     ui.Element
	MaxHeight float32
	State     *ui.ScrollState
}

// NewScrollView creates a ScrollView element.
func NewScrollView(child ui.Element, maxHeight float32, state *ui.ScrollState) ui.Element {
	return ScrollView{Child: child, MaxHeight: maxHeight, State: state}
}

// LayoutSelf implements ui.Layouter.
func (n ScrollView) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	viewportH := int(n.MaxHeight)
	if viewportH <= 0 || viewportH > area.H {
		viewportH = area.H
	}

	// Determine scroll offset from state.
	var offset float32
	if n.State != nil {
		offset = n.State.Offset
	}

	// Reserve scrollbar width inside the allocated area so clipped
	// parents (e.g. SplitView) don't cut it off.
	scrollbarW := int(ctx.Tokens.Scroll.TrackWidth)
	if scrollbarW <= 0 {
		scrollbarW = 8
	}
	contentW := area.W // width available for child content

	// Pre-measure with unconstrained height to detect the child's natural
	// height. Using area.H would let the child clamp to the viewport,
	// hiding overflow and making needsScroll incorrectly false.
	measureArea := ui.Bounds{X: area.X, Y: area.Y, W: contentW, H: 1 << 20}
	mb := ctx.MeasureChild(n.Child, measureArea)
	needsScroll := mb.H > viewportH

	if needsScroll {
		contentW = max(area.W-scrollbarW, 0)
	}

	// Clip to viewport.
	ctx.Canvas.PushClip(draw.R(float32(area.X), float32(area.Y), float32(contentW), float32(viewportH)))

	// Render child offset by -offset in Y so content scrolls upward.
	childArea := ui.Bounds{X: area.X, Y: area.Y - int(offset), W: contentW, H: area.H + int(offset)}
	childBounds := ctx.LayoutChild(n.Child, childArea)

	ctx.Canvas.PopClip()

	contentH := childBounds.H

	// Correct needsScroll based on actual rendered content in case
	// the pre-measurement was inaccurate (e.g. touch-mode widgets
	// grow taller during the real layout pass than during measurement).
	if contentH > viewportH && !needsScroll {
		needsScroll = true
	}

	// Only clamp scroll state during actual render passes (ctx.IX != nil),
	// not during measurement passes. Measurement passes may run with a
	// different viewport height (e.g. Flex measures children at full area.H
	// before shrinking), which would incorrectly tighten the scroll bounds.
	if n.State != nil && ctx.IX != nil {
		maxScroll := float32(contentH) - float32(viewportH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if n.State.Offset > maxScroll {
			n.State.Offset = maxScroll
		}
		if n.State.Offset < 0 {
			n.State.Offset = 0
		}

		// Auto-scroll to keep the focused element visible (e.g. when the OSK
		// appears and shrinks the viewport). The focused bounds are in screen
		// space (post-scroll), so we convert to content space first.
		if ctx.Focus != nil && ctx.Focus.FocusedBounds != nil {
			fb := ctx.Focus.FocusedBounds
			areaY := float32(area.Y)
			contentTop := fb.Y + n.State.Offset - areaY
			contentBottom := contentTop + fb.H
			vH := float32(viewportH)
			padding := float32(16)

			if contentBottom > n.State.Offset+vH {
				n.State.Offset = contentBottom - vH + padding
			}
			if contentTop < n.State.Offset {
				n.State.Offset = contentTop - padding
				if n.State.Offset < 0 {
					n.State.Offset = 0
				}
			}
			// Re-clamp after adjustment.
			if n.State.Offset > maxScroll {
				n.State.Offset = maxScroll
			}
		}
	}

	// Register the viewport as a scroll target so the framework can
	// route mouse-wheel events directly to the ScrollState.
	if n.State != nil && needsScroll {
		state := n.State
		cH := float32(contentH)
		vH := float32(viewportH)
		ctx.IX.RegisterScroll(
			draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(viewportH)),
			cH, vH,
			func(deltaY float32) {
				state.ScrollBy(deltaY, cH, vH)
			},
		)
	}

	// Draw scrollbar inside allocated area.
	w := area.W
	if needsScroll {
		ui.DrawScrollbar(ctx.Canvas, ctx.Tokens, ctx.IX, n.State, area.X+contentW, area.Y, viewportH, float32(contentH), offset)

		// Fade hints: draw a short gradient at the top/bottom edges
		// to indicate more content is available in that direction.
		// Rendered in overlay mode so they appear above images/surfaces.
		fadeH := float32(24)
		bgColor := ctx.Tokens.Colors.Surface.Base
		transparent := draw.Color{R: bgColor.R, G: bgColor.G, B: bgColor.B, A: 0}

		maxScroll := float32(contentH) - float32(viewportH)
		vx := float32(area.X)
		vw := float32(contentW)

		type overlayModeSetter interface{ SetOverlayMode(bool) }
		oms, hasOverlay := ctx.Canvas.(overlayModeSetter)
		if hasOverlay {
			oms.SetOverlayMode(true)
		}

		// Top fade — visible when scrolled down.
		if offset > 1 {
			ctx.Canvas.FillRect(
				draw.R(vx, float32(area.Y), vw, fadeH),
				draw.LinearGradientPaint(
					draw.Pt(0, 0),
					draw.Pt(0, fadeH),
					draw.GradientStop{Offset: 0, Color: bgColor},
					draw.GradientStop{Offset: 1, Color: transparent},
				),
			)
		}

		// Bottom fade — visible when not scrolled to the end.
		if offset < maxScroll-1 {
			bottomY := float32(area.Y+viewportH) - fadeH
			ctx.Canvas.FillRect(
				draw.R(vx, bottomY, vw, fadeH),
				draw.LinearGradientPaint(
					draw.Pt(0, 0),
					draw.Pt(0, fadeH),
					draw.GradientStop{Offset: 0, Color: transparent},
					draw.GradientStop{Offset: 1, Color: bgColor},
				),
			)
		}

		if hasOverlay {
			oms.SetOverlayMode(false)
		}
	} else {
		w = max(childBounds.W, area.W)
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: viewportH}
}

// TreeEqual implements ui.TreeEqualizer.
func (n ScrollView) TreeEqual(other ui.Element) bool {
	o, ok := other.(ScrollView)
	if !ok {
		return false
	}
	return n.MaxHeight == o.MaxHeight
}

// ResolveChildren implements ui.ChildResolver.
func (n ScrollView) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.Child = resolve(n.Child, 0)
	return n
}

// WalkAccess implements ui.AccessWalker. Passes through to the child.
func (n ScrollView) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Child, parentIdx)
}
