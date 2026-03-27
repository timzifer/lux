//go:build nogui

package app

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestDirectionFromLocaleRTL(t *testing.T) {
	rtlLocales := []string{"ar", "he", "fa", "ur", "ps", "sd", "yi", "ar-SA", "he-IL", "fa-IR"}
	for _, loc := range rtlLocales {
		if dir := DirectionFromLocale(loc); dir != draw.DirRTL {
			t.Errorf("DirectionFromLocale(%q) = %d, want DirRTL", loc, dir)
		}
	}
}

func TestDirectionFromLocaleLTR(t *testing.T) {
	ltrLocales := []string{"en", "de", "fr", "ja", "zh", "ko", "en-US", "de-DE", ""}
	for _, loc := range ltrLocales {
		if dir := DirectionFromLocale(loc); dir != draw.DirLTR {
			t.Errorf("DirectionFromLocale(%q) = %d, want DirLTR", loc, dir)
		}
	}
}

func TestDirectionFromLocaleCaseInsensitive(t *testing.T) {
	if dir := DirectionFromLocale("AR"); dir != draw.DirRTL {
		t.Errorf("DirectionFromLocale(AR) should be RTL")
	}
	if dir := DirectionFromLocale("He-IL"); dir != draw.DirRTL {
		t.Errorf("DirectionFromLocale(He-IL) should be RTL")
	}
}

func TestApplyLocale(t *testing.T) {
	// Should not panic.
	applyLocale("en")
	applyLocale("ar")
	applyLocale("")
}
