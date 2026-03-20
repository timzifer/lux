// Package theme provides the theming system (RFC §5, RFC-003 §1-2).
//
// A Theme supplies design tokens and optional custom DrawFuncs for
// widget rendering.  Themes are composable via a parent chain.
package theme

import (
	"time"

	"github.com/timzifer/lux/anim"
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
	// Tier 2
	WidgetKindTextField
	WidgetKindCheckbox
	WidgetKindRadio
	WidgetKindToggle
	WidgetKindSlider
	WidgetKindProgressBar
	WidgetKindSelect
	// Tier 3
	WidgetKindCard
	WidgetKindTabs
	WidgetKindAccordion
	WidgetKindTooltip
	WidgetKindBadge
	WidgetKindChip
	WidgetKindMenuBar
	WidgetKindContextMenu
	WidgetKindSplitView
	WidgetKindDialog
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

// DurationEasing pairs a duration with an easing function (RFC-002 §1.6).
type DurationEasing struct {
	Duration time.Duration
	Easing   anim.EasingFunc
}

// MotionSpec defines animation presets with duration and easing (RFC-002 §1.6).
type MotionSpec struct {
	Standard   DurationEasing // 250ms OutCubic — standard transitions
	Emphasized DurationEasing // 400ms InOutCubic — emphasized transitions
	Quick      DurationEasing // 100ms OutExpo — fast reactions (hover, ripple)
}

// ElevationScale defines shadow presets for different elevation levels.
type ElevationScale struct {
	None draw.Shadow
	Low  draw.Shadow
	Med  draw.Shadow
	High draw.Shadow
}

// ScrollSpec defines scroll physics parameters (RFC §14.4, RFC-002 §3.4).
type ScrollSpec struct {
	Friction          float32 // Deceleration factor per frame at 60fps (0.95 = smooth, 0.80 = fast stop)
	Overscroll        float32 // Maximum rubber-band displacement in dp
	TrackWidth        float32 // Scrollbar track width
	ThumbRadius       float32 // Scrollbar thumb corner radius
	SettlingThreshold float32 // Velocity below which scroll stops (dp/frame)
	StepSize          float32 // Scroll amount per mouse wheel click (dp)
	MultiplierPrecise float32 // Multiplier for trackpad deltas
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
	Scrim    draw.Color // Semi-transparent backdrop for modal dialogs
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
	H1         draw.TextStyle // 20dp, SemiBold — page title
	H2         draw.TextStyle // 16dp, SemiBold — section title
	H3         draw.TextStyle // 14dp, Medium — subtitle
	Body       draw.TextStyle // 13dp, Regular — standard body text
	BodySmall  draw.TextStyle // 12dp, Regular — metadata
	Label      draw.TextStyle // 12dp, Medium — button text, tab labels
	LabelSmall draw.TextStyle // 11dp, Medium — badges, chips
	Code       draw.TextStyle // 13dp, Regular, Monospace
	CodeSmall  draw.TextStyle // 12dp, Regular, Monospace
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

// Default is the recommended theme for new applications (RFC-008 §12.2).
var Default Theme = LuxDark

type slateTheme struct{}

var slateTokens = TokenSet{
	Colors: ColorScheme{
		Surface: SurfaceColors{
			Base:     draw.Hex("#09090b"), // Zinc-950
			Elevated: draw.Hex("#18181b"), // Zinc-900
			Hovered:  draw.Hex("#27272a"), // Zinc-800
			Pressed:  draw.Hex("#3f3f46"), // Zinc-700
			Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.5},
		},
		Accent: AccentColors{
			Primary:         draw.Hex("#3b82f6"), // Blue-500
			PrimaryContrast: draw.Hex("#ffffff"),
			Secondary:       draw.Hex("#6366f1"), // Indigo-500
		},
		Stroke: StrokeColors{
			Border:  draw.Color{R: 1, G: 1, B: 1, A: 0.10}, // 10% white
			Focus:   draw.Hex("#3b82f6"),                   // = Accent.Primary
			Divider: draw.Color{R: 1, G: 1, B: 1, A: 0.06}, // 6% white
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
		H1:         draw.TextStyle{Size: 20, Weight: draw.FontWeightSemiBold, LineHeight: 1.3},
		H2:         draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold, LineHeight: 1.3},
		H3:         draw.TextStyle{Size: 14, Weight: draw.FontWeightMedium, LineHeight: 1.4},
		Body:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.5},
		BodySmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.5},
		Label:      draw.TextStyle{Size: 12, Weight: draw.FontWeightMedium, LineHeight: 1.0},
		LabelSmall: draw.TextStyle{Size: 11, Weight: draw.FontWeightMedium, LineHeight: 1.0},
		Code:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.6, FontFamily: "JetBrains Mono"},
		CodeSmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.6, FontFamily: "JetBrains Mono"},
	},
	Spacing: SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48},
	Radii:   RadiusScale{Input: 4, Button: 6, Card: 8, Pill: 999},
	Motion: MotionSpec{
		Standard:   DurationEasing{250 * time.Millisecond, anim.OutCubic},
		Emphasized: DurationEasing{400 * time.Millisecond, anim.InOutCubic},
		Quick:      DurationEasing{100 * time.Millisecond, anim.OutExpo},
	},
	Elevation: ElevationScale{
		None: draw.Shadow{},
		Low:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.6}, BlurRadius: 4, OffsetY: 2, Radius: 8},
		Med:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.7}, BlurRadius: 8, OffsetY: 4, Radius: 8},
		High: draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.8}, BlurRadius: 16, OffsetY: 8, Radius: 8},
	},
	Scroll: ScrollSpec{
		Friction:          0.95,
		Overscroll:        40,
		TrackWidth:        8,
		ThumbRadius:       4,
		SettlingThreshold: 0.5,
		StepSize:          48,
		MultiplierPrecise: 1.5,
	},
}

func (s *slateTheme) Tokens() TokenSet             { return slateTokens }
func (s *slateTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (s *slateTheme) Parent() Theme                { return nil }

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
		Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.4},
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
func (s *slateLightTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (s *slateLightTheme) Parent() Theme                { return Slate }

type slateLightTheme struct{}

// ── ThemePair (RFC-008 §12.1) ──────────────────────────────────

// ThemePair is implemented by themes that provide matched dark/light variants.
// The app loop uses this interface to resolve SetDarkModeMsg without
// hard-coding theme names.
type ThemePair interface {
	DarkVariant() Theme
	LightVariant() Theme
}

// ── theme.Lux — The Default Theme (RFC-008) ────────────────────

// LuxDark is the built-in dark theme of the lux family (RFC-008 §5.2).
var LuxDark Theme = &luxTheme{}

// LuxLight is the built-in light theme of the lux family (RFC-008 §5.3).
var LuxLight Theme = &luxLightTheme{}

// LuxAuto follows the OS dark-mode signal, starting in dark (RFC-008 §12.1).
var LuxAuto Theme = &luxAutoTheme{}

type luxTheme struct{}

var luxDarkTokens = TokenSet{
	Colors: ColorScheme{
		Surface: SurfaceColors{
			Base:     draw.Hex("#0f1115"),
			Elevated: draw.Hex("#171a20"),
			Hovered:  draw.Hex("#1d222a"),
			Pressed:  draw.Hex("#252b35"),
			Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.46},
		},
		Accent: AccentColors{
			Primary:         draw.Hex("#4c8dff"),
			PrimaryContrast: draw.Hex("#ffffff"),
			Secondary:       draw.Hex("#7aa8ff"),
		},
		Stroke: StrokeColors{
			Border:  draw.Color{R: 1, G: 1, B: 1, A: 0.10},
			Focus:   draw.Hex("#7aa8ff"),
			Divider: draw.Color{R: 1, G: 1, B: 1, A: 0.06},
		},
		Text: TextColors{
			Primary:   draw.Hex("#eef2f7"),
			Secondary: draw.Hex("#a8b0bc"),
			Disabled:  draw.Hex("#606975"),
			OnAccent:  draw.Hex("#ffffff"),
		},
		Status: StatusColors{
			Success:   draw.Hex("#3bb273"),
			Warning:   draw.Hex("#d9a441"),
			Error:     draw.Hex("#de5b6d"),
			Info:      draw.Hex("#4c8dff"),
			OnSuccess: draw.Hex("#ffffff"),
			OnError:   draw.Hex("#ffffff"),
		},
	},
	Typography: TypographyScale{
		H1:         draw.TextStyle{Size: 20, Weight: draw.FontWeightSemiBold, LineHeight: 1.25, Tracking: -0.01},
		H2:         draw.TextStyle{Size: 16, Weight: draw.FontWeightSemiBold, LineHeight: 1.30},
		H3:         draw.TextStyle{Size: 14, Weight: draw.FontWeightMedium, LineHeight: 1.35},
		Body:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.50},
		BodySmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.45},
		Label:      draw.TextStyle{Size: 12, Weight: draw.FontWeightMedium, LineHeight: 1.00},
		LabelSmall: draw.TextStyle{Size: 11, Weight: draw.FontWeightMedium, LineHeight: 1.00},
		Code:       draw.TextStyle{Size: 13, Weight: draw.FontWeightRegular, LineHeight: 1.45, FontFamily: "JetBrains Mono"},
		CodeSmall:  draw.TextStyle{Size: 12, Weight: draw.FontWeightRegular, LineHeight: 1.40, FontFamily: "JetBrains Mono"},
	},
	Spacing: SpacingScale{XS: 4, S: 8, M: 16, L: 24, XL: 32, XXL: 48},
	Radii:   RadiusScale{Input: 4, Button: 6, Card: 10, Pill: 999},
	Motion: MotionSpec{
		Standard:   DurationEasing{220 * time.Millisecond, anim.OutCubic},
		Emphasized: DurationEasing{320 * time.Millisecond, anim.InOutCubic},
		Quick:      DurationEasing{110 * time.Millisecond, anim.OutExpo},
	},
	Elevation: ElevationScale{
		None: draw.Shadow{},
		Low:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.14}, BlurRadius: 10, OffsetY: 2, Radius: 8},
		Med:  draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.18}, BlurRadius: 18, OffsetY: 6, Radius: 12},
		High: draw.Shadow{Color: draw.Color{R: 0, G: 0, B: 0, A: 0.22}, BlurRadius: 28, OffsetY: 10, Radius: 14},
	},
	Scroll: ScrollSpec{
		Friction:          0.95,
		Overscroll:        40,
		TrackWidth:        8,
		ThumbRadius:       4,
		SettlingThreshold: 0.5,
		StepSize:          48,
		MultiplierPrecise: 1.5,
	},
}

func (l *luxTheme) Tokens() TokenSet             { return luxDarkTokens }
func (l *luxTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (l *luxTheme) Parent() Theme                { return nil }

// ── theme.LuxLight (RFC-008 §5.3) ──────────────────────────────

type luxLightTheme struct{}

var luxLightTokens = func() TokenSet {
	t := luxDarkTokens
	t.Colors.Surface = SurfaceColors{
		Base:     draw.Hex("#f5f7fb"),
		Elevated: draw.Hex("#ffffff"),
		Hovered:  draw.Hex("#edf1f7"),
		Pressed:  draw.Hex("#e4e9f1"),
		Scrim:    draw.Color{R: 0, G: 0, B: 0, A: 0.18},
	}
	t.Colors.Accent = AccentColors{
		Primary:         draw.Hex("#2f6fe4"),
		PrimaryContrast: draw.Hex("#ffffff"),
		Secondary:       draw.Hex("#5e92ef"),
	}
	t.Colors.Stroke = StrokeColors{
		Border:  draw.Color{R: 0.09, G: 0.12, B: 0.18, A: 0.12},
		Focus:   draw.Hex("#2f6fe4"),
		Divider: draw.Color{R: 0.09, G: 0.12, B: 0.18, A: 0.08},
	}
	t.Colors.Text = TextColors{
		Primary:   draw.Hex("#17202b"),
		Secondary: draw.Hex("#5e6a78"),
		Disabled:  draw.Hex("#9aa4b2"),
		OnAccent:  draw.Hex("#ffffff"),
	}
	t.Colors.Status = StatusColors{
		Success:   draw.Hex("#278f5a"),
		Warning:   draw.Hex("#b27d1f"),
		Error:     draw.Hex("#c94b5d"),
		Info:      draw.Hex("#2f6fe4"),
		OnSuccess: draw.Hex("#ffffff"),
		OnError:   draw.Hex("#ffffff"),
	}
	return t
}()

func (l *luxLightTheme) Tokens() TokenSet             { return luxLightTokens }
func (l *luxLightTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (l *luxLightTheme) Parent() Theme                { return LuxDark }

// ── theme.LuxAuto (RFC-008 §12.1) ──────────────────────────────

type luxAutoTheme struct{}

func (l *luxAutoTheme) Tokens() TokenSet             { return luxDarkTokens }
func (l *luxAutoTheme) DrawFunc(WidgetKind) DrawFunc { return nil }
func (l *luxAutoTheme) Parent() Theme                { return nil }
func (l *luxAutoTheme) DarkVariant() Theme           { return LuxDark }
func (l *luxAutoTheme) LightVariant() Theme          { return LuxLight }

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
