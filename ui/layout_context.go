package ui

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/interaction"
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
	Profile  *interaction.InteractionProfile // active profile, nil = desktop default (RFC-004 §2.4)
	SafeArea SafeAreaInsets                   // viewport insets from system UI (OSK, notch, etc.)
}

// IsTouch reports whether the active interaction profile uses touch input
// (finger or glove). Widgets use this to switch to touch-optimized layouts
// with larger hit targets.
func (ctx *LayoutContext) IsTouch() bool {
	return ctx.Profile != nil && ctx.Profile.PointerKind != interaction.PointerMouse
}

// LayoutChild dispatches layout for a child element within the given area.
// It delegates to layoutElement which handles both Layouter-implementing types
// and legacy types via the type switch. SafeArea insets are propagated from
// the parent context so child layouts can respect viewport-level insets.
func (ctx *LayoutContext) LayoutChild(el Element, area Bounds) Bounds {
	if el == nil {
		return Bounds{X: area.X, Y: area.Y}
	}
	return layoutElementCtx(el, area, ctx.Canvas, ctx.Theme, ctx.Tokens, ctx.IX, ctx.Overlays, ctx.Focus, ctx.Profile, ctx.SafeArea)
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
	return layoutElement(el, area, nc, ctx.Theme, ctx.Tokens, nil, nil, nil, ctx.Profile)
}
