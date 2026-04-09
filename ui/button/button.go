// Package button provides button element types for the Lux UI framework.
package button

import (
	"math"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/display"
)

// Button is a clickable button element with arbitrary content.
type Button struct {
	ui.BaseElement
	Content  ui.Element
	OnClick  func()
	Variant  ui.ButtonVariant
	Disabled bool
	Ripple   bool // enable touch ripple feedback (HMI/touch mode)
}

// Text creates a filled button with a text label.
func Text(label string, onClick func()) ui.Element {
	return Button{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		Variant: ui.ButtonFilled,
	}
}

// TextRipple creates a filled button with a text label and touch ripple feedback.
func TextRipple(label string, onClick func()) ui.Element {
	return Button{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		Variant: ui.ButtonFilled,
		Ripple:  true,
	}
}

// TextDisabled creates a disabled filled button with a text label.
func TextDisabled(label string) ui.Element {
	return Button{
		Content:  display.TextElement{Content: label},
		Variant:  ui.ButtonFilled,
		Disabled: true,
	}
}

// New creates a filled button with arbitrary content.
func New(content ui.Element, onClick func()) ui.Element {
	return Button{Content: content, OnClick: onClick, Variant: ui.ButtonFilled}
}

// VariantOf creates a button with the given variant and arbitrary content.
func VariantOf(variant ui.ButtonVariant, content ui.Element, onClick func()) ui.Element {
	return Button{Content: content, OnClick: onClick, Variant: variant}
}

// OutlinedText creates an outlined button with a text label.
func OutlinedText(label string, onClick func()) ui.Element {
	return Button{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		Variant: ui.ButtonOutlined,
	}
}

// GhostText creates a text-only (chromeless) button.
func GhostText(label string, onClick func()) ui.Element {
	return Button{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		Variant: ui.ButtonGhost,
	}
}

// TonalText creates a tonal button with a text label.
func TonalText(label string, onClick func()) ui.Element {
	return Button{
		Content: display.TextElement{Content: label},
		OnClick: onClick,
		Variant: ui.ButtonTonal,
	}
}

// LayoutSelf implements ui.Layouter.
func (n Button) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	canvas := ctx.Canvas
	th := ctx.Theme
	ix := ctx.IX
	overlays := ctx.Overlays
	fs := ctx.Focus

	// Pass 1: measure content via NullCanvas.
	cb := ctx.MeasureChild(n.Content, ui.Bounds{X: 0, Y: 0, W: area.W, H: area.H})

	contentW := cb.W
	contentH := cb.H
	w := contentW + (ui.ButtonPadX * 2)
	h := contentH + (ui.ButtonPadY * 2)

	// Enforce MinTouchTarget for touch/HMI profiles (RFC-004 §2.5).
	if ctx.Profile != nil && ctx.Profile.MinTouchTarget > 0 {
		minT := int(ctx.Profile.MinTouchTarget)
		if w < minT {
			w = minT
		}
		if h < minT {
			h = minT
		}
	}

	// Register hit target and get hover opacity atomically.
	buttonRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))
	var hoverOpacity float32
	var ripple *ui.RippleState
	if n.Disabled {
		// Disabled: register no-op to keep hover index aligned.
		ix.RegisterHit(buttonRect, nil)
	} else if n.Ripple {
		// Positional click with framework-managed ripple.
		hoverOpacity, ripple = ix.RegisterHitRipple(buttonRect, n.OnClick)
	} else {
		hoverOpacity = ix.RegisterHit(buttonRect, n.OnClick)
	}

	// Focus management.
	var focused bool
	if fs != nil && !n.Disabled {
		uid := fs.NextElementUID()
		fs.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = fs.IsElementFocused(uid)
	}

	// Custom theme DrawFunc dispatch.
	if df := th.DrawFunc(theme.WidgetKindButton); df != nil {
		df(theme.DrawCtx{
			Canvas:   canvas,
			Bounds:   buttonRect,
			Hovered:  hoverOpacity > 0,
			Focused:  focused,
			Disabled: n.Disabled,
		}, tokens, n)
	} else {
		fillColor, borderColor, textColor := ui.ButtonVariantColors(n.Variant, tokens, hoverOpacity)
		// Disabled muting.
		if n.Disabled {
			base := tokens.Colors.Surface.Base
			fillColor = ui.DisabledColor(fillColor, base)
			borderColor = ui.DisabledColor(borderColor, base)
			textColor = tokens.Colors.Text.Disabled
		}

		if n.Variant == ui.ButtonFilled {
			// Filled: border as background fill, opaque fill on top (2-rect approach).
			canvas.FillRoundRect(buttonRect,
				tokens.Radii.Button, draw.SolidPaint(borderColor))
			innerRadius := tokens.Radii.Button - float32(ui.ButtonBorder)
			if innerRadius < 0 {
				innerRadius = 0
			}
			canvas.FillRoundRect(draw.R(float32(area.X+ui.ButtonBorder), float32(area.Y+ui.ButtonBorder),
				float32(max(w-ui.ButtonBorder*2, 0)), float32(max(h-ui.ButtonBorder*2, 0))),
				innerRadius, draw.SolidPaint(fillColor))
		} else {
			// Non-filled: fill first, then stroke outline on top.
			if fillColor.A > 0 {
				canvas.FillRoundRect(buttonRect, tokens.Radii.Button, draw.SolidPaint(fillColor))
			}
			if borderColor.A > 0 {
				canvas.StrokeRoundRect(buttonRect, tokens.Radii.Button, draw.Stroke{
					Paint: draw.SolidPaint(borderColor),
					Width: float32(ui.ButtonBorder),
				})
			}
		}

		// Focus glow.
		if focused {
			ui.DrawFocusRing(canvas, buttonRect, tokens.Radii.Button, tokens)
		}

		// Ripple overlay — use text colour so the pulse contrasts with the button fill.
		if ripple != nil {
			ripple.Draw(canvas, buttonRect, tokens.Radii.Button, textColor)
		}

		// Pass 2: render content centered.
		if txt, ok := n.Content.(display.TextElement); ok {
			style := tokens.Typography.Label
			metrics := canvas.MeasureText(txt.Content, style)
			labelW := int(math.Ceil(float64(metrics.Width)))
			labelH := int(math.Ceil(float64(metrics.Ascent)))
			canvas.DrawText(txt.Content,
				draw.Pt(float32(area.X+(w-labelW)/2), float32(area.Y+(h-labelH)/2)),
				style, textColor)
		} else {
			contentX := area.X + (w-contentW)/2
			contentY := area.Y + (h-contentH)/2
			// For non-filled variants, override theme text/icon colors so
			// child elements (Text, Icon inside a Row) use the variant color.
			contentTokens := tokens
			if n.Variant != ui.ButtonFilled {
				contentTokens.Colors.Text.Primary = textColor
				contentTokens.Colors.Text.OnAccent = textColor
			}
			subCtx := &ui.LayoutContext{
				Area:     ui.Bounds{X: contentX, Y: contentY, W: contentW, H: contentH},
				Canvas:   canvas,
				Theme:    th,
				Tokens:   contentTokens,
				IX:       ix,
				Overlays: overlays,
				Focus:    fs,
			}
			subCtx.LayoutChild(n.Content, subCtx.Area)
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h, Baseline: ui.ButtonPadY + cb.Baseline}
}

// TreeEqual implements ui.TreeEqualizer.
// Buttons contain callbacks which are not comparable, so we return false.
func (n Button) TreeEqual(other ui.Element) bool {
	_, ok := other.(Button)
	return ok && false
}

// ResolveChildren implements ui.ChildResolver. Button is treated as a leaf
// for widget resolution purposes.
func (n Button) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n Button) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := extractLabel(n.Content)
	accessNode := a11y.AccessNode{
		Role:  a11y.RoleButton,
		Label: label,
	}
	if n.OnClick != nil {
		accessNode.Actions = []a11y.AccessAction{
			{Name: "activate", Trigger: n.OnClick},
		}
	}
	b.AddNode(accessNode, parentIdx, a11y.Rect{})
}

// extractLabel tries to get a text label from a button's content element.
func extractLabel(el ui.Element) string {
	if txt, ok := el.(display.TextElement); ok {
		return txt.Content
	}
	return ""
}
