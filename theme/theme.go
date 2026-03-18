// Package theme provides the theming system (RFC §5, RFC-003 §1-2).
//
// A Theme supplies design tokens and optional custom DrawFuncs for
// widget rendering.  Themes are composable via a parent chain.
package theme

import (
	"time"

	"github.com/timzifer/lux/draw"
)

// WidgetKind identifies a built-in widget type for DrawFunc dispatch.
type WidgetKind uint16

const (
	WidgetKindButton WidgetKind = iota + 1
	WidgetKindText
	WidgetKindBox
	WidgetKindIcon
	WidgetKindStack
	WidgetKindScrollView
	WidgetKindDivider
	WidgetKindSpacer
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

// ── Token Types (RFC-003 §1.2) ──────────────────────────────────

// MotionSpec defines animation duration presets (RFC §12.6).
type MotionSpec struct {
	Standard   time.Duration // 200ms — standard transitions
	Emphasized time.Duration // 400ms — emphasized transitions
	Quick      time.Duration // 100ms — fast reactions (hover, ripple)
}

// ElevationScale defines shadow presets for different elevation levels.
type ElevationScale struct {
	None draw.Shadow
	Low  draw.Shadow
	Med  draw.Shadow
	High draw.Shadow
}

// ScrollSpec defines scroll physics parameters (RFC §14.4).
type ScrollSpec struct {
	Friction    float32
	Overscroll  float32
	TrackWidth  float32
	ThumbRadius float32
}

// TokenSet holds all design tokens for a theme (RFC-003 §1.2).
type TokenSet struct {
	Colors     ColorScheme
	Typography TypographyScale
	Spacing    SpacingScale
	Radii      RadiusScale
	Motion     MotionSpec
	Elevation  ElevationScale
	Scroll     ScrollSpec
}

// ── ColorScheme (RFC-003 §1.2) ──────────────────────────────────

// SurfaceColors defines background surface tokens.
type SurfaceColors struct {
	Base     draw.Color // Window background — deepest layer
	Elevated draw.Color // Cards, overlays — one level above
	Hovered  draw.Color // Widget hover state
	Pressed  draw.Color // Widget active/pressed state
}

// AccentColors defines primary interaction color tokens.
type AccentColors struct {
	Primary         draw.Color // Main color (buttons, links, focus indicator)
	PrimaryContrast draw.Color // Text on Primary (usually white or black)
	Secondary       draw.Color // Optional second accent color
}

// StrokeColors defines line and border tokens.
type StrokeColors struct {
	Border  draw.Color // Subtle separation (1px solid, low opacity)
	Focus   draw.Color // Strong contrast for keyboard navigation
	Divider draw.Color // Even subtler than Border (section separations)
}

// TextColors defines text color tokens.
type TextColors struct {
	Primary   draw.Color // Main text
	Secondary draw.Color // Labels, metadata (dimmed)
	Disabled  draw.Color // Disabled elements
	OnAccent  draw.Color // Text on Accent.Primary
}

// StatusColors defines semantic state color tokens.
type StatusColors struct {
	Success   draw.Color
	Warning   draw.Color
	Error     draw.Color
	Info      draw.Color
	OnSuccess draw.Color
	OnError   draw.Color
}

// ColorScheme uses semantic slots instead of hard-coded colors (RFC-003 §1.2).
type ColorScheme struct {
	Surface SurfaceColors
	Accent  AccentColors
	Stroke  StrokeColors
	Text    TextColors
	Status  StatusColors
	Custom  map[string]draw.Color
}

// ── TypographyScale (RFC-003 §1.2) ─────────────────────────────

// TypographyScale defines the desktop-first text style slots.
type TypographyScale struct {
	H1        draw.TextStyle // 20dp, SemiBold — page title
	H2        draw.TextStyle // 16dp, SemiBold — section title
	H3        draw.TextStyle // 14dp, Medium — subtitle
	Body      draw.TextStyle // 13dp, Regular — standard body text
	BodySmall draw.TextStyle // 12dp, Regular — metadata
	Label     draw.TextStyle // 12dp, Medium — button text, tab labels
	LabelSmall draw.TextStyle // 11dp, Medium — badges, chips
	Code      draw.TextStyle // 13dp, Regular, Monospace
	CodeSmall draw.TextStyle // 12dp, Regular, Monospace
}

// ── SpacingScale (RFC-003 §2) ───────────────────────────────────

// SpacingScale defines spacing constants.
type SpacingScale struct {
	XS  float32 // 4dp
	S   float32 // 8dp
	M   float32 // 16dp
	L   float32 // 24dp
	XL  float32 // 32dp
	XXL float32 // 48dp
}

// ── RadiusScale (RFC-003 §2) ────────────────────────────────────

// RadiusScale defines corner-radius constants.
type RadiusScale struct {
	Input  float32 // 4dp — input fields
	Button float32 // 6dp — buttons
	Card   float32 // 8dp — cards
	Pill   float32 // 999dp — pill shapes
}

// ── theme.Slate — The Default Theme (RFC-003 §2) ────────────────

// Slate is the built-in dark theme. Philosophy: the sobriety of Linear,
// the precision of Fluent Design — without the platform connotations
// of Material or Cupertino.
var Slate Theme = &slateTheme{}

// Default is an alias for Slate (backward compatibility).
var Default Theme = Slate

type slateTheme struct{}

var slateTokens = TokenSet{
	Colors: ColorScheme{
		Surface: SurfaceColors{
			Base:     draw.Hex("#09090b"), // Zinc-950
			Elevated: draw.Hex("#18181b"), // Zinc-900
			Hovered:  draw.Hex("#27272a"), // Zinc-800
			Pressed:  draw.Hex("#3f3f46"), // Zinc-700
		},
		Accent: AccentColors{
			Primary:         draw.Hex("#3b82f6"), // Blue-500
			PrimaryContrast: draw.Hex("#ffffff"),
			Secondary:       draw.Hex("#6366f1"), // Indigo-500
		},
		Stroke: StrokeColors{
			Border:  draw.Color{R: 1, G: 1, B: 1, A: 0.10}, // 10% white
			Focus:   draw.Hex("#3b82f6"),                      // = Accent.Primary
			Divider: draw.Color{R: 1, G: 1, B: 1, A: 0.06},  // 6% white
		},
		Text: TextColors{
			Primary:   draw.Hex("#fafafa"), // Zinc-50
			Secondary: draw.Hex("#a1a1aa"), // Zinc-400
			Disabled:  draw.Hex("#52525b"), // Zinc-600
			OnAccent:  draw.Hex("#ffffff"),
		},
		Status: StatusColors{
			Success:   draw.Hex("#22c55e"), // Green-500
			Warning:   draw.Hex("#f59e0b"), // Amber-500
			Error:     draw.Hex("#ef4444"), // Red-500
			Info:      draw.Hex("#3b82f6"), // Blue-500
			OnSuccess: draw.Hex("#ffffff"),
			OnError:   draw.Hex("#ffffff"),
		},
	},
	Typography: TypographyScale{
		H1:        draw.TextStyle{Size: 20, Weight: draw.FontWeightSemiBold, LineHeight: 1.3},
		H2:        draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold, LineHeight: 1.3},
		H3:        draw.TextStyle{Size: 14, Weight: draw.FontWeightMedium, LineHeight: 1.4},
		Body:      draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.5},
		BodySmall: draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.5},
		Label:     draw.TextStyle{Size: 12, Weight: draw.FontWeightMedium, LineHeight: 1.0},
		LabelSmall: draw.TextStyle{Size: 11, Weight: draw.FontWeightMedium, LineHeight: 1.0},
		Code:      draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.6, FontFamily: "JetBrains Mono"},
		CodeSmall: draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.6, FontFamily: "JetBrains Mono"},
	},
	Spacing: SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48},
	Radii:   RadiusScale{Input: 4, Button: 6, Card: 8, Pill: 999},
	Motion: MotionSpec{
		Standard:   200 * time.Millisecond,
		Emphasized: 400 * time.Millisecond,
		Quick:      100 * time.Millisecond,
	},
	Scroll: ScrollSpec{
		Friction:    0.95,
		Overscroll:  40,
		TrackWidth:  8,
		ThumbRadius: 4,
	},
}

func (s *slateTheme) Tokens() TokenSet             { return slateTokens }
func (s *slateTheme) DrawFunc(WidgetKind) DrawFunc  { return nil }
func (s *slateTheme) Parent() Theme                 { return nil }

// ── theme.SlateLight (RFC-003 §2) ───────────────────────────────

// SlateLight is the built-in light theme, derived from Slate.
var SlateLight Theme = &slateLightTheme{}

// Light is an alias for SlateLight (backward compatibility).
var Light Theme = SlateLight

var slateLightTokens = func() TokenSet {
	t := slateTokens
	t.Colors.Surface = SurfaceColors{
		Base:     draw.Hex("#ffffff"),
		Elevated: draw.Hex("#f4f4f5"), // Zinc-100
		Hovered:  draw.Hex("#e4e4e7"), // Zinc-200
		Pressed:  draw.Hex("#d4d4d8"), // Zinc-300
	}
	t.Colors.Stroke = StrokeColors{
		Border:  draw.Color{R: 0, G: 0, B: 0, A: 0.10}, // 10% black
		Focus:   draw.Hex("#3b82f6"),
		Divider: draw.Color{R: 0, G: 0, B: 0, A: 0.06}, // 6% black
	}
	t.Colors.Text = TextColors{
		Primary:   draw.Hex("#09090b"), // Zinc-950
		Secondary: draw.Hex("#71717a"), // Zinc-500
		Disabled:  draw.Hex("#a1a1aa"), // Zinc-400
		OnAccent:  draw.Hex("#ffffff"),
	}
	return t
}()

func (s *slateLightTheme) Tokens() TokenSet             { return slateLightTokens }
func (s *slateLightTheme) DrawFunc(WidgetKind) DrawFunc  { return nil }
func (s *slateLightTheme) Parent() Theme                 { return Slate }

type slateLightTheme struct{}

// ── Override (RFC-003 §1.6) ─────────────────────────────────────

// OverrideSpec specifies partial token overrides. Non-nil pointer fields
// replace the corresponding tokens from the base theme.
type OverrideSpec struct {
	Colors     *ColorScheme
	Typography *TypographyScale
	Spacing    *SpacingScale
	Radii      *RadiusScale
	Motion     *MotionSpec
	Elevation  *ElevationScale
	Scroll     *ScrollSpec
}

// Override creates a new Theme that applies partial overrides to a base theme.
func Override(base Theme, spec OverrideSpec) Theme {
	return &overrideTheme{base: base, spec: spec}
}

type overrideTheme struct {
	base Theme
	spec OverrideSpec
}

func (o *overrideTheme) Tokens() TokenSet {
	t := o.base.Tokens()
	if o.spec.Colors != nil {
		t.Colors = *o.spec.Colors
	}
	if o.spec.Typography != nil {
		t.Typography = *o.spec.Typography
	}
	if o.spec.Spacing != nil {
		t.Spacing = *o.spec.Spacing
	}
	if o.spec.Radii != nil {
		t.Radii = *o.spec.Radii
	}
	if o.spec.Motion != nil {
		t.Motion = *o.spec.Motion
	}
	if o.spec.Elevation != nil {
		t.Elevation = *o.spec.Elevation
	}
	if o.spec.Scroll != nil {
		t.Scroll = *o.spec.Scroll
	}
	return t
}

func (o *overrideTheme) DrawFunc(kind WidgetKind) DrawFunc {
	return o.base.DrawFunc(kind)
}

func (o *overrideTheme) Parent() Theme {
	return o.base
}
