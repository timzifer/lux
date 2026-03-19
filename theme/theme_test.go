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

func TestDefaultAliasIsSlate(t *testing.T) {
	if Default != Slate {
		t.Error("Default should be an alias for Slate")
	}
}

func TestLightAliasIsSlateLight(t *testing.T) {
	if Light != SlateLight {
		t.Error("Light should be an alias for SlateLight")
	}
}

func TestOverrideColors(t *testing.T) {
	custom := Override(Slate, OverrideSpec{
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
	if tokens.Typography.Body.Size != Slate.Tokens().Typography.Body.Size {
		t.Error("Override should not change non-overridden fields")
	}
}

func TestOverridePreservesDrawFunc(t *testing.T) {
	custom := Override(Slate, OverrideSpec{})
	if custom.DrawFunc(WidgetKindButton) != nil {
		t.Error("Override should delegate DrawFunc to base")
	}
}

func TestOverrideParentIsBase(t *testing.T) {
	custom := Override(Slate, OverrideSpec{})
	if custom.Parent() != Slate {
		t.Error("Override.Parent() should be base theme")
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
