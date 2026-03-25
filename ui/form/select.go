package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// SelectState holds the open/closed state for a Select dropdown.
type SelectState struct {
	Open bool
}

// Select is a dropdown selector.
type Select struct {
	ui.BaseElement
	Value    string
	Options  []string
	State    *SelectState
	OnSelect func(string)
	Disabled bool
}

// SelectOption configures a Select element.
type SelectOption func(*Select)

// WithSelectState links the Select to a SelectState for dropdown behaviour.
func WithSelectState(s *SelectState) SelectOption {
	return func(e *Select) { e.State = s }
}

// WithOnSelect sets the callback invoked when an option is chosen.
func WithOnSelect(fn func(string)) SelectOption {
	return func(e *Select) { e.OnSelect = fn }
}

// WithSelectDisabled marks the Select as disabled.
func WithSelectDisabled() SelectOption {
	return func(e *Select) { e.Disabled = true }
}

// NewSelect creates a dropdown selector.
func NewSelect(value string, options []string, opts ...SelectOption) ui.Element {
	el := Select{Value: value, Options: options}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// LayoutSelf implements ui.Layouter.
func (n Select) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	th := ctx.Theme
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2

	w := textFieldW
	if area.W < w {
		w = area.W
	}

	// Register hit target and get hover opacity atomically.
	selectRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(selectRect, nil)
	} else {
		var selectClickFn func()
		if n.State != nil {
			state := n.State
			selectClickFn = func() { state.Open = !state.Open }
		}
		hoverOpacity = ix.RegisterHit(selectRect, selectClickFn)
	}

	isOpen := n.State != nil && n.State.Open && !n.Disabled

	// Focus management (RFC-008 §9.4).
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Custom theme DrawFunc dispatch (RFC §5.3).
	if df := th.DrawFunc(theme.WidgetKindSelect); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   selectRect,
			Hovered:  hoverOpacity > 0,
			Focused:  focused,
			Disabled: n.Disabled,
		}, tokens, n)
	} else {
		// Border
		borderColor := tokens.Colors.Stroke.Border
		if n.Disabled {
			borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(
			draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			tokens.Radii.Input, draw.SolidPaint(borderColor))

		// Fill
		fillColor := tokens.Colors.Surface.Elevated
		if n.Disabled {
			fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
		}
		canvas.FillRoundRect(
			draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
			maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

		// Value text
		textX := area.X + textFieldPadX
		textY := area.Y + textFieldPadY
		textColor := tokens.Colors.Text.Primary
		if n.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if n.Value != "" {
			canvas.DrawText(n.Value, draw.Pt(float32(textX), float32(textY)), style, textColor)
		}

		// Down arrow indicator (Phosphor icon for reliable rendering).
		arrowStyle := tokens.Typography.LabelSmall
		arrowStyle.FontFamily = "Phosphor"
		arrowX := area.X + w - textFieldPadX - int(arrowStyle.Size)
		arrowColor := tokens.Colors.Text.Secondary
		if n.Disabled {
			arrowColor = tokens.Colors.Text.Disabled
		}
		canvas.DrawText(icons.CaretDown, draw.Pt(float32(arrowX), float32(textY)), arrowStyle, arrowColor)

		// Focus glow (RFC-008 §9.4).
		if focused || isOpen {
			ui.DrawFocusRing(canvas, selectRect, tokens.Radii.Input, tokens)
		}
	}

	// Dropdown overlay when open.
	if isOpen && len(n.Options) > 0 {
		dropX := area.X
		dropY := area.Y + h
		dropW := w
		opts := n.Options
		onSelect := n.OnSelect
		state := n.State
		winW := overlays.WindowW
		winH := overlays.WindowH
		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Full-screen backdrop: clicking outside the dropdown closes it.
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					if state != nil {
						state.Open = false
					}
				})

				itemH := int(tokens.Typography.Body.Size) + textFieldPadY*2
				totalH := itemH * len(opts)

				// Dropdown background.
				canvas.FillRoundRect(
					draw.R(float32(dropX), float32(dropY), float32(dropW), float32(totalH)),
					tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				// Dropdown border.
				canvas.StrokeRoundRect(
					draw.R(float32(dropX), float32(dropY), float32(dropW), float32(totalH)),
					tokens.Radii.Input, draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				for i, opt := range opts {
					itemY := dropY + i*itemH
					o := opt
					var itemClickFn func()
					if onSelect != nil || state != nil {
						itemClickFn = func() {
							if onSelect != nil {
								onSelect(o)
							}
							if state != nil {
								state.Open = false
							}
						}
					}
					ho := ix.RegisterHit(draw.R(float32(dropX), float32(itemY), float32(dropW), float32(itemH)), itemClickFn)
					if ho > 0 {
						canvas.FillRect(
							draw.R(float32(dropX+1), float32(itemY), float32(max(dropW-2, 0)), float32(itemH)),
							draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}
					canvas.DrawText(opt,
						draw.Pt(float32(dropX+textFieldPadX), float32(itemY+textFieldPadY)),
						tokens.Typography.Body, tokens.Colors.Text.Primary)
				}
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n Select) TreeEqual(other ui.Element) bool {
	nb, ok := other.(Select)
	if !ok || n.Value != nb.Value || len(n.Options) != len(nb.Options) {
		return false
	}
	for i := range n.Options {
		if n.Options[i] != nb.Options[i] {
			return false
		}
	}
	return true
}

// ResolveChildren implements ui.ChildResolver. Select is a leaf.
func (n Select) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Select) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	b.AddNode(a11y.AccessNode{
		Role:  a11y.RoleCombobox,
		Value: n.Value,
	}, parentIdx, a11y.Rect{})
}
