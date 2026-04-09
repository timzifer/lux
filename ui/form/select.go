package form

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
	"github.com/timzifer/lux/ui/menu"
)

// SelectState holds the open/closed state for a Select dropdown.
type SelectState struct {
	Open        bool
	TouchScroll ui.ScrollState // scroll offset for touch action sheet
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

	// Enforce minimum touch target height (RFC-004).
	if ctx.IsTouch() && ctx.Profile != nil {
		if minH := int(ctx.Profile.MinTouchTarget); h < minH {
			h = minH
		}
	}

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
			selectClickFn = func() {
				state.Open = !state.Open
				if state.Open {
					state.TouchScroll = ui.ScrollState{} // reset scroll on open
				}
			}
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

		// Down arrow indicator (Phosphor icon for reliable rendering).
		arrowStyle := tokens.Typography.LabelSmall
		arrowStyle.FontFamily = "Phosphor"
		arrowX := area.X + w - textFieldPadX - int(arrowStyle.Size)
		textX := area.X + textFieldPadX
		textY := area.Y + textFieldPadY
		arrowColor := tokens.Colors.Text.Secondary
		if n.Disabled {
			arrowColor = tokens.Colors.Text.Disabled
		}
		canvas.DrawText(icons.CaretDown, draw.Pt(float32(arrowX), float32(textY)), arrowStyle, arrowColor)

		// Value text — clipped to avoid overlapping the arrow indicator.
		textColor := tokens.Colors.Text.Primary
		if n.Disabled {
			textColor = tokens.Colors.Text.Disabled
		}
		if n.Value != "" {
			valueClip := draw.R(float32(textX), float32(area.Y), float32(arrowX-textX-textFieldPadX), float32(h))
			canvas.PushClip(valueClip)
			canvas.DrawText(n.Value, draw.Pt(float32(textX), float32(textY)), style, textColor)
			canvas.PopClip()
		}

		// Focus glow (RFC-008 §9.4).
		if focused || isOpen {
			ui.DrawFocusRing(canvas, selectRect, tokens.Radii.Input, tokens)
		}
	}

	// Dropdown overlay when open (skip during measurement passes where overlays is nil).
	if isOpen && len(n.Options) > 0 && overlays != nil {
		if ctx.IsTouch() {
			// Touch/HMI: centralized action sheet overlay.
			n.layoutTouchOverlay(ctx)
		} else {
			// Desktop: anchor-relative dropdown (existing behavior).
			n.layoutDesktopDropdown(ctx, area, w, h)
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// layoutDesktopDropdown renders the classic anchor-relative dropdown overlay
// for desktop profiles (unchanged legacy behavior).
func (n Select) layoutDesktopDropdown(ctx *ui.LayoutContext, area ui.Bounds, w, h int) {
	dropX := area.X
	dropW := w
	opts := n.Options
	onSelect := n.OnSelect
	state := n.State
	tokens := ctx.Tokens
	winW := ctx.Overlays.WindowW
	winH := ctx.Overlays.WindowH

	// Flip dropdown above the select if it would overflow the viewport bottom.
	itemH0 := int(tokens.Typography.Body.Size) + textFieldPadY*2
	totalH0 := itemH0 * len(opts)
	dropY := area.Y + h // default: open below
	if dropY+totalH0 > winH && area.Y-totalH0 >= 0 {
		dropY = area.Y - totalH0 // flip: open above
	}
	ctx.Overlays.Push(ui.OverlayEntry{
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
				itemClip := draw.R(float32(dropX+textFieldPadX), float32(itemY), float32(dropW-textFieldPadX*2), float32(itemH))
				canvas.PushClip(itemClip)
				canvas.DrawText(opt,
					draw.Pt(float32(dropX+textFieldPadX), float32(itemY+textFieldPadY)),
					tokens.Typography.Body, tokens.Colors.Text.Primary)
				canvas.PopClip()
			}
		},
	})
}

// layoutTouchOverlay renders a centralized action sheet overlay for touch/HMI profiles.
func (n Select) layoutTouchOverlay(ctx *ui.LayoutContext) {
	profile := ctx.Profile
	state := n.State
	opts := n.Options
	value := n.Value
	onSelect := n.OnSelect
	winW := ctx.Overlays.WindowW
	winH := ctx.Overlays.WindowH
	th := ctx.Theme

	ctx.Overlays.Push(ui.OverlayEntry{
		Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
			items := make([]menu.ActionSheetItem, len(opts))
			for i, opt := range opts {
				o := opt
				items[i] = menu.ActionSheetItem{
					Label:    o,
					Selected: o == value,
					OnClick: func() {
						if onSelect != nil {
							onSelect(o)
						}
						if state != nil {
							state.Open = false
						}
					},
				}
			}
			menu.RenderActionSheet(menu.ActionSheetConfig{
				Items:       items,
				Profile:     profile,
				WinW:        winW,
				WinH:        winH,
				OnDismiss:   func() { state.Open = false },
				ScrollState: &state.TouchScroll,
				Theme:       th,
			}, canvas, tokens, ix)
		},
	})
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
