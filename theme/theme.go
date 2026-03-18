// Package theme provides the theming system (RFC §5).
//
// A Theme supplies design tokens and optional custom DrawFuncs for
// widget rendering.  Themes are composable via a parent chain.
package theme

import "github.com/timzifer/lux/draw"

// WidgetKind identifies a built-in widget type for DrawFunc dispatch.
type WidgetKind uint16

const (
	WidgetKindButton WidgetKind = iota + 1
	WidgetKindText
	WidgetKindBox
)

// DrawFunc is a custom rendering function for a widget kind (RFC §5.3).
type DrawFunc func(ctx DrawCtx, tokens TokenSet, state any)

// DrawCtx provides the rendering context passed to a DrawFunc.
type DrawCtx struct {
	Canvas  draw.Canvas
	Bounds  draw.Rect
	DPR     float32
	Focused bool
	Hovered bool
	Pressed bool
}

// Theme is the interface every theme must implement (RFC §5.1).
type Theme interface {
	// Tokens returns the design token set.
	Tokens() TokenSet

	// DrawFunc returns a custom DrawFunc for a widget kind,
	// or nil to use the framework default.
	DrawFunc(kind WidgetKind) DrawFunc

	// Parent returns the fallback theme, or nil for root themes.
	Parent() Theme
}

// TokenSet holds all design tokens for a theme.
type TokenSet struct {
	Colors     ColorScheme
	Typography TypographyScale
	Spacing    SpacingScale
	Radii      RadiusScale
}

// ColorScheme defines the colour slots of a theme.
type ColorScheme struct {
	Background draw.Color
	Surface    draw.Color
	Primary    draw.Color
	Secondary  draw.Color
	OnPrimary  draw.Color
	OnSurface  draw.Color
	Error      draw.Color
	Outline    draw.Color
}

// TypographyScale defines the text style slots.
type TypographyScale struct {
	DisplayLarge  draw.TextStyle
	HeadlineLarge draw.TextStyle
	BodyMedium    draw.TextStyle
	LabelSmall    draw.TextStyle
}

// SpacingScale defines spacing constants.
type SpacingScale struct {
	XS float32
	SM float32
	MD float32
	LG float32
	XL float32
}

// RadiusScale defines corner-radius constants.
type RadiusScale struct {
	None   float32
	Small  float32
	Medium float32
	Large  float32
	Full   float32
}

// ── Default Theme ────────────────────────────────────────────────

// Default is the built-in dark theme used by M2.
var Default Theme = &defaultTheme{}

type defaultTheme struct{}

func (d *defaultTheme) Tokens() TokenSet {
	return TokenSet{
		Colors: ColorScheme{
			Background: draw.RGBA(18, 18, 20, 255),
			Surface:    draw.RGBA(28, 28, 32, 255),
			Primary:    draw.RGBA(52, 120, 246, 255),
			Secondary:  draw.RGBA(126, 177, 255, 255),
			OnPrimary:  draw.RGBA(245, 247, 250, 255),
			OnSurface:  draw.RGBA(245, 247, 250, 255),
			Error:      draw.RGBA(207, 54, 54, 255),
			Outline:    draw.RGBA(126, 177, 255, 255),
		},
		Typography: TypographyScale{
			DisplayLarge: draw.TextStyle{
				FontFamily: "Bitmap5x7",
				Size:       21, // 7 * 3
				Weight:     draw.FontWeightRegular,
				LineHeight: 1.2,
			},
			HeadlineLarge: draw.TextStyle{
				FontFamily: "Bitmap5x7",
				Size:       21,
				Weight:     draw.FontWeightRegular,
				LineHeight: 1.2,
			},
			BodyMedium: draw.TextStyle{
				FontFamily: "Bitmap5x7",
				Size:       21,
				Weight:     draw.FontWeightRegular,
				LineHeight: 1.2,
			},
			LabelSmall: draw.TextStyle{
				FontFamily: "Bitmap5x7",
				Size:       21,
				Weight:     draw.FontWeightRegular,
				LineHeight: 1.2,
			},
		},
		Spacing: SpacingScale{XS: 4, SM: 8, MD: 16, LG: 24, XL: 32},
		Radii:   RadiusScale{None: 0, Small: 4, Medium: 8, Large: 16, Full: 9999},
	}
}

func (d *defaultTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (d *defaultTheme) Parent() Theme                { return nil }
