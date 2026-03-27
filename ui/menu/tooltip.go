package menu

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
)

// Tooltip layout constants.
const (
	tooltipPadding = 8
)

// Tooltip renders an arbitrary hover popup anchored to a trigger element.
type Tooltip struct {
	ui.BaseElement
	Trigger ui.Element
	Content ui.Element // arbitrary widget content
	Visible bool       // controlled by hover state or explicit flag
	Blur    bool       // optional frosted-glass backdrop (RFC-008 §11.5)
}

// New creates a tooltip element with hover-based visibility.
func New(trigger, content ui.Element) ui.Element {
	return Tooltip{Trigger: trigger, Content: content}
}

// Visible creates a tooltip with explicit visibility control.
func Visible(trigger, content ui.Element, visible bool) ui.Element {
	return Tooltip{Trigger: trigger, Content: content, Visible: visible}
}

// Blur creates a tooltip with frosted-glass backdrop (RFC-008 §11.5).
func Blur(trigger, content ui.Element) ui.Element {
	return Tooltip{Trigger: trigger, Content: content, Blur: true}
}

// LayoutSelf implements ui.Layouter.
func (n Tooltip) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	// Layout trigger normally.
	triggerBounds := ctx.LayoutChild(n.Trigger, ctx.Area)

	// Determine visibility: explicit or hover-based.
	visible := n.Visible
	if !visible {
		// Register trigger as hover target so the hover system tracks it.
		hoverOpacity := ctx.IX.RegisterHit(draw.R(float32(triggerBounds.X), float32(triggerBounds.Y),
			float32(triggerBounds.W), float32(triggerBounds.H)), nil)
		visible = hoverOpacity > 0.1
	}

	if visible && ctx.Overlays != nil {
		tB := triggerBounds
		content := n.Content
		blur := n.Blur
		th := ctx.Theme
		ctx.Overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Measure content.
				nc := ui.NullCanvas{Delegate: canvas}
				measureCtx := &ui.LayoutContext{
					Canvas: nc,
					Theme:  th,
					Tokens: tokens,
				}
				cb := measureCtx.LayoutChild(content, ui.Bounds{X: 0, Y: 0, W: 300, H: 200})

				w := cb.W + tooltipPadding*2
				h := cb.H + tooltipPadding*2
				x := tB.X
				y := tB.Y + tB.H + 4

				tooltipRect := draw.R(float32(x), float32(y), float32(w), float32(h))
				innerRect := draw.R(float32(x+1), float32(y+1), float32(max(w-2, 0)), float32(max(h-2, 0)))
				innerRadius := maxf(tokens.Radii.Button-1, 0)

				if blur {
					// Frosted-glass backdrop (RFC-008 §11.5).
					canvas.PushClipRoundRect(tooltipRect, tokens.Radii.Button)
					canvas.PushBlur(8)
					canvas.FillRoundRect(tooltipRect, tokens.Radii.Button, draw.SolidPaint(draw.Color{A: 0.01}))
					canvas.PopBlur()
					canvas.PopClip()
					// Semi-transparent tinted fill.
					tint := tokens.Colors.Surface.Elevated
					tint.A = 0.75
					canvas.FillRoundRect(innerRect, innerRadius, draw.SolidPaint(tint))
				} else {
					// Border.
					canvas.FillRoundRect(tooltipRect,
						tokens.Radii.Button, draw.SolidPaint(tokens.Colors.Stroke.Border))
					// Opaque fill.
					canvas.FillRoundRect(innerRect, innerRadius, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				}

				// Border stroke (shared).
				canvas.StrokeRoundRect(tooltipRect, tokens.Radii.Button, draw.Stroke{
					Paint: draw.SolidPaint(tokens.Colors.Stroke.Border),
					Width: 1,
				})

				// Content.
				overlayCtx := &ui.LayoutContext{
					Area:   ui.Bounds{X: x + tooltipPadding, Y: y + tooltipPadding, W: max(w-tooltipPadding*2, 0), H: max(h-tooltipPadding*2, 0)},
					Canvas: canvas,
					Theme:  th,
					Tokens: tokens,
					IX:     ix,
				}
				overlayCtx.LayoutChild(content, overlayCtx.Area)
			},
		})
	}

	return triggerBounds
}

// TreeEqual implements ui.TreeEqualizer.
func (n Tooltip) TreeEqual(other ui.Element) bool {
	o, ok := other.(Tooltip)
	return ok && n.Visible == o.Visible && n.Blur == o.Blur
}

// ResolveChildren implements ui.ChildResolver.
func (n Tooltip) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	n.Trigger = resolve(n.Trigger, 0)
	n.Content = resolve(n.Content, 1)
	return n
}

// WalkAccess implements ui.AccessWalker.
// Only the trigger is relevant for a11y; tooltip content is overlay.
func (n Tooltip) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.Walk(n.Trigger, parentIdx)
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
