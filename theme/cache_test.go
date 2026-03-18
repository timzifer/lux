package theme

import "testing"

// Compile-time check: CachedTheme satisfies Theme interface.
var _ Theme = (*CachedTheme)(nil)

func TestCachedThemeTokens(t *testing.T) {
	ct := NewCachedTheme(Slate)
	tokens1 := ct.Tokens()
	tokens2 := ct.Tokens()

	// Should return identical values (cached).
	if tokens1.Colors.Surface.Base != tokens2.Colors.Surface.Base {
		t.Error("consecutive Tokens() calls should return the same value")
	}

	// Tokens should match the base theme.
	if tokens1.Colors.Surface.Base != Slate.Tokens().Colors.Surface.Base {
		t.Error("cached tokens should match base theme tokens")
	}
}

func TestCachedThemeDrawFunc(t *testing.T) {
	ct := NewCachedTheme(Slate)

	// Slate returns nil for all DrawFuncs.
	if ct.DrawFunc(WidgetKindButton) != nil {
		t.Error("expected nil DrawFunc for Slate")
	}
}

func TestCachedThemeParent(t *testing.T) {
	ct := NewCachedTheme(SlateLight)
	if ct.Parent() != Slate {
		t.Error("CachedTheme(SlateLight).Parent() should be Slate")
	}
}

func TestCachedThemeBase(t *testing.T) {
	ct := NewCachedTheme(Slate)
	if ct.Base() != Slate {
		t.Error("Base() should return the wrapped theme")
	}
}

func TestCachedThemeInvalidate(t *testing.T) {
	ct := NewCachedTheme(Slate)
	ct.WarmUp()

	// Verify cache is populated.
	if ct.resolved == nil {
		t.Fatal("WarmUp should populate resolved cache")
	}

	ct.Invalidate()
	if ct.resolved != nil {
		t.Error("Invalidate should clear resolved cache")
	}

	// Next access should re-resolve.
	tokens := ct.Tokens()
	if tokens.Colors.Surface.Base != Slate.Tokens().Colors.Surface.Base {
		t.Error("tokens after invalidate should match base theme")
	}
	if ct.resolved == nil {
		t.Error("accessing Tokens after invalidate should re-resolve")
	}
}

func TestCachedThemeWarmUp(t *testing.T) {
	ct := NewCachedTheme(Slate)
	if ct.resolved != nil {
		t.Error("resolved should be nil before WarmUp")
	}
	ct.WarmUp()
	if ct.resolved == nil {
		t.Error("resolved should not be nil after WarmUp")
	}
}

// customTheme is a test theme that tracks DrawFunc calls.
type customTheme struct {
	drawFuncCalls int
}

func (ct *customTheme) Tokens() TokenSet {
	return Slate.Tokens()
}

func (ct *customTheme) DrawFunc(kind WidgetKind) DrawFunc {
	ct.drawFuncCalls++
	if kind == WidgetKindButton {
		return func(DrawCtx, TokenSet, any) {}
	}
	return nil
}

func (ct *customTheme) Parent() Theme { return nil }

func TestCachedThemeDrawFuncCaching(t *testing.T) {
	base := &customTheme{}
	ct := NewCachedTheme(base)

	// First call resolves and caches all DrawFuncs.
	fn1 := ct.DrawFunc(WidgetKindButton)
	initialCalls := base.drawFuncCalls

	// Second call should use cache — no additional base calls.
	fn2 := ct.DrawFunc(WidgetKindButton)
	if base.drawFuncCalls != initialCalls {
		t.Error("second DrawFunc call should use cache, not call base again")
	}

	// Both should return non-nil for WidgetKindButton.
	if fn1 == nil || fn2 == nil {
		t.Error("DrawFunc should return non-nil for WidgetKindButton")
	}
}

func TestCachedThemeWithOverride(t *testing.T) {
	overridden := Override(Slate, OverrideSpec{
		Spacing: &SpacingScale{XS: 99},
	})
	ct := NewCachedTheme(overridden)
	tokens := ct.Tokens()
	if tokens.Spacing.XS != 99 {
		t.Errorf("Spacing.XS = %f, want 99", tokens.Spacing.XS)
	}
}
