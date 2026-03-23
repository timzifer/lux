package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
)

// LayoutContext bundles all parameters needed during element layout.
// Sub-packages use this to implement LayoutSelf on their element types.
type LayoutContext struct {
	Area     Bounds
	Canvas   draw.Canvas
	Theme    theme.Theme
	Tokens   theme.TokenSet
	IX       *Interactor
	Overlays *OverlayStack
	Focus    *FocusManager
}

// LayoutChild dispatches layout for a child element within the given area.
// It delegates to layoutElement which handles both Layouter-implementing types
// and legacy types via the type switch.
func (ctx *LayoutContext) LayoutChild(el Element, area Bounds) Bounds {
	if el == nil {
		return Bounds{X: area.X, Y: area.Y}
	}
	return layoutElement(el, area, ctx.Canvas, ctx.Theme, ctx.Tokens, ctx.IX, ctx.Overlays, ctx.Focus)
}

// WithArea returns a copy of the context with a different area.
func (ctx *LayoutContext) WithArea(area Bounds) *LayoutContext {
	c := *ctx
	c.Area = area
	return &c
}

// WithTheme returns a copy of the context with a different theme.
func (ctx *LayoutContext) WithTheme(th theme.Theme) *LayoutContext {
	c := *ctx
	c.Theme = th
	c.Tokens = th.Tokens()
	return &c
}

// MeasureChild measures a child element without painting,
// using a NullCanvas for the measurement pass.
func (ctx *LayoutContext) MeasureChild(el Element, area Bounds) Bounds {
	nc := NullCanvas{Delegate: ctx.Canvas}
	return layoutElement(el, area, nc, ctx.Theme, ctx.Tokens, nil, nil)
}
