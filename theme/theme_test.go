package theme

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestSlateTokensNonZero(t *testing.T) {
	tokens := Slate.Tokens()
	zero := draw.Color{}

	if tokens.Colors.Surface.Base == zero {
		t.Error("Slate Surface.Base should not be zero")
	}
	if tokens.Colors.Accent.Primary == zero {
		t.Error("Slate Accent.Primary should not be zero")
	}
	if tokens.Colors.Text.Primary == zero {
		t.Error("Slate Text.Primary should not be zero")
	}
	if tokens.Colors.Status.Error == zero {
		t.Error("Slate Status.Error should not be zero")
	}
	if tokens.Typography.Body.Size == 0 {
		t.Error("Slate Body typography size should not be zero")
	}
}

func TestSlateLightDiffersFromSlate(t *testing.T) {
	dark := Slate.Tokens()
	light := SlateLight.Tokens()

	if dark.Colors.Surface.Base == light.Colors.Surface.Base {
		t.Error("Slate and SlateLight should have different Surface.Base")
	}
	if dark.Colors.Text.Primary == light.Colors.Text.Primary {
		t.Error("Slate and SlateLight should have different Text.Primary")
	}
	// Accent colors should be the same (inherited)
	if dark.Colors.Accent.Primary != light.Colors.Accent.Primary {
		t.Error("Slate and SlateLight should share Accent.Primary")
	}
}

func TestSlateLightParentIsSlate(t *testing.T) {
	if SlateLight.Parent() != Slate {
		t.Error("SlateLight.Parent() should be Slate")
	}
}

func TestDefaultAliasIsLuxDark(t *testing.T) {
	if Default != LuxDark {
		t.Error("Default should be an alias for LuxDark")
	}
}

func TestLightAliasIsSlateLight(t *testing.T) {
	if Light != SlateLight {
		t.Error("Light should be an alias for SlateLight")
	}
}

// ── Lux Theme Tests (RFC-008) ─────────────────────────────────

func TestLuxDarkTokensNonZero(t *testing.T) {
	tokens := LuxDark.Tokens()
	zero := draw.Color{}

	if tokens.Colors.Surface.Base == zero {
		t.Error("LuxDark Surface.Base should not be zero")
	}
	if tokens.Colors.Accent.Primary == zero {
		t.Error("LuxDark Accent.Primary should not be zero")
	}
	if tokens.Colors.Text.Primary == zero {
		t.Error("LuxDark Text.Primary should not be zero")
	}
	if tokens.Colors.Status.Error == zero {
		t.Error("LuxDark Status.Error should not be zero")
	}
	if tokens.Typography.Body.Size == 0 {
		t.Error("LuxDark Body typography size should not be zero")
	}
}

func TestLuxLightDiffersFromLuxDark(t *testing.T) {
	dark := LuxDark.Tokens()
	light := LuxLight.Tokens()

	if dark.Colors.Surface.Base == light.Colors.Surface.Base {
		t.Error("LuxDark and LuxLight should have different Surface.Base")
	}
	if dark.Colors.Text.Primary == light.Colors.Text.Primary {
		t.Error("LuxDark and LuxLight should have different Text.Primary")
	}
}

func TestLuxLightParentIsLuxDark(t *testing.T) {
	if LuxLight.Parent() != LuxDark {
		t.Error("LuxLight.Parent() should be LuxDark")
	}
}

func TestLuxMotion(t *testing.T) {
	tokens := LuxDark.Tokens()
	if tokens.Motion.Quick.Duration.Milliseconds() != 110 {
		t.Errorf("Lux Motion.Quick.Duration = %v, want 110ms", tokens.Motion.Quick.Duration)
	}
	if tokens.Motion.Standard.Duration.Milliseconds() != 220 {
		t.Errorf("Lux Motion.Standard.Duration = %v, want 220ms", tokens.Motion.Standard.Duration)
	}
	if tokens.Motion.Emphasized.Duration.Milliseconds() != 320 {
		t.Errorf("Lux Motion.Emphasized.Duration = %v, want 320ms", tokens.Motion.Emphasized.Duration)
	}
}

func TestLuxRadii(t *testing.T) {
	tokens := LuxDark.Tokens()
	if tokens.Radii.Card != 10 {
		t.Errorf("Lux Radii.Card = %f, want 10", tokens.Radii.Card)
	}
}

func TestLuxTypographyTracking(t *testing.T) {
	tokens := LuxDark.Tokens()
	if tokens.Typography.H1.Tracking != -0.01 {
		t.Errorf("Lux H1 Tracking = %f, want -0.01", tokens.Typography.H1.Tracking)
	}
}

func TestLuxAutoReturnsLuxDarkTokens(t *testing.T) {
	if LuxAuto.Tokens().Colors.Surface.Base != LuxDark.Tokens().Colors.Surface.Base {
		t.Error("LuxAuto should return LuxDark tokens by default")
	}
}

func TestLuxAutoImplementsThemePair(t *testing.T) {
	tp, ok := LuxAuto.(ThemePair)
	if !ok {
		t.Fatal("LuxAuto should implement ThemePair")
	}
	if tp.DarkVariant() != LuxDark {
		t.Error("LuxAuto.DarkVariant() should be LuxDark")
	}
	if tp.LightVariant() != LuxLight {
		t.Error("LuxAuto.LightVariant() should be LuxLight")
	}
}

func TestOverrideColors(t *testing.T) {
	custom := Override(LuxDark, OverrideSpec{
		Colors: &ColorScheme{
			Surface: SurfaceColors{
				Base: draw.Hex("#ff0000"),
			},
		},
	})

	tokens := custom.Tokens()
	if tokens.Colors.Surface.Base != draw.Hex("#ff0000") {
		t.Error("Override should replace Surface.Base")
	}
	// Non-overridden fields should keep base values
	if tokens.Typography.Body.Size != LuxDark.Tokens().Typography.Body.Size {
		t.Error("Override should not change non-overridden fields")
	}
}

func TestOverridePreservesDrawFunc(t *testing.T) {
	custom := Override(LuxDark, OverrideSpec{})
	if custom.DrawFunc(WidgetKindButton) != nil {
		t.Error("Override should delegate DrawFunc to base")
	}
}

func TestOverrideParentIsBase(t *testing.T) {
	custom := Override(LuxDark, OverrideSpec{})
	if custom.Parent() != LuxDark {
		t.Error("Override.Parent() should be base theme")
	}
}

func TestTypographyScaled(t *testing.T) {
	base := LuxDark.Tokens().Typography
	scaled := base.Scaled(1.5)

	check := func(name string, got, wantBase float32) {
		t.Helper()
		want := wantBase * 1.5
		if got != want {
			t.Errorf("%s.Size = %v, want %v", name, got, want)
		}
	}
	check("H1", scaled.H1.Size, base.H1.Size)
	check("H2", scaled.H2.Size, base.H2.Size)
	check("H3", scaled.H3.Size, base.H3.Size)
	check("Body", scaled.Body.Size, base.Body.Size)
	check("BodySmall", scaled.BodySmall.Size, base.BodySmall.Size)
	check("Label", scaled.Label.Size, base.Label.Size)
	check("LabelSmall", scaled.LabelSmall.Size, base.LabelSmall.Size)
	check("Code", scaled.Code.Size, base.Code.Size)
	check("CodeSmall", scaled.CodeSmall.Size, base.CodeSmall.Size)
}

func TestTypographyScaledIdentity(t *testing.T) {
	base := LuxDark.Tokens().Typography
	scaled := base.Scaled(1.0)

	if scaled.Body.Size != base.Body.Size {
		t.Errorf("Scaled(1.0) should not change Body.Size: got %v, want %v", scaled.Body.Size, base.Body.Size)
	}
}

func TestTypographyScaledPreservesWeight(t *testing.T) {
	base := LuxDark.Tokens().Typography
	scaled := base.Scaled(1.5)

	if scaled.H1.Weight != base.H1.Weight {
		t.Errorf("Scaled should not change H1.Weight: got %v, want %v", scaled.H1.Weight, base.H1.Weight)
	}
}

func TestSlateRadii(t *testing.T) {
	tokens := Slate.Tokens()
	if tokens.Radii.Button != 6 {
		t.Errorf("Slate Radii.Button = %f, want 6", tokens.Radii.Button)
	}
	if tokens.Radii.Card != 8 {
		t.Errorf("Slate Radii.Card = %f, want 8", tokens.Radii.Card)
	}
}

func TestSlateSpacing(t *testing.T) {
	tokens := Slate.Tokens()
	if tokens.Spacing.XXL != 48 {
		t.Errorf("Slate Spacing.XXL = %f, want 48", tokens.Spacing.XXL)
	}
}

func TestSlateMotion(t *testing.T) {
	tokens := Slate.Tokens()
	if tokens.Motion.Quick.Duration.Milliseconds() != 100 {
		t.Errorf("Slate Motion.Quick.Duration = %v, want 100ms", tokens.Motion.Quick.Duration)
	}
	if tokens.Motion.Standard.Duration.Milliseconds() != 250 {
		t.Errorf("Slate Motion.Standard.Duration = %v, want 250ms", tokens.Motion.Standard.Duration)
	}
	if tokens.Motion.Emphasized.Duration.Milliseconds() != 400 {
		t.Errorf("Slate Motion.Emphasized.Duration = %v, want 400ms", tokens.Motion.Emphasized.Duration)
	}
	if tokens.Motion.Quick.Easing == nil {
		t.Error("Slate Motion.Quick.Easing should not be nil")
	}
	if tokens.Motion.Standard.Easing == nil {
		t.Error("Slate Motion.Standard.Easing should not be nil")
	}
}
