package interaction

import (
	"testing"
	"time"
)

func TestProfileDesktopDefaults(t *testing.T) {
	p := ProfileDesktop
	if p.PointerKind != PointerMouse {
		t.Errorf("PointerKind = %d, want PointerMouse", p.PointerKind)
	}
	if p.MinTouchTarget != 24 {
		t.Errorf("MinTouchTarget = %v, want 24", p.MinTouchTarget)
	}
	if !p.HasHover {
		t.Error("HasHover = false, want true for desktop")
	}
	if !p.HasPhysicalKeyboard {
		t.Error("HasPhysicalKeyboard = false, want true for desktop")
	}
	if p.ScaleTypography != 1.0 {
		t.Errorf("ScaleTypography = %v, want 1.0", p.ScaleTypography)
	}
	if p.DebounceInterval != 0 {
		t.Errorf("DebounceInterval = %v, want 0", p.DebounceInterval)
	}
}

func TestProfileTouchDefaults(t *testing.T) {
	p := ProfileTouch
	if p.PointerKind != PointerFinger {
		t.Errorf("PointerKind = %d, want PointerFinger", p.PointerKind)
	}
	if p.MinTouchTarget != 48 {
		t.Errorf("MinTouchTarget = %v, want 48", p.MinTouchTarget)
	}
	if p.HasHover {
		t.Error("HasHover = true, want false for touch")
	}
	if p.HasPhysicalKeyboard {
		t.Error("HasPhysicalKeyboard = true, want false for touch")
	}
	if p.DebounceInterval != 200*time.Millisecond {
		t.Errorf("DebounceInterval = %v, want 200ms", p.DebounceInterval)
	}
}

func TestProfileHMIDefaults(t *testing.T) {
	p := ProfileHMI
	if p.PointerKind != PointerGlove {
		t.Errorf("PointerKind = %d, want PointerGlove", p.PointerKind)
	}
	if p.MinTouchTarget != 64 {
		t.Errorf("MinTouchTarget = %v, want 64", p.MinTouchTarget)
	}
	if p.TouchTargetSpacing != 12 {
		t.Errorf("TouchTargetSpacing = %v, want 12", p.TouchTargetSpacing)
	}
	if p.HasHover {
		t.Error("HasHover = true, want false for HMI")
	}
	if p.DragThreshold != 14 {
		t.Errorf("DragThreshold = %v, want 14", p.DragThreshold)
	}
	if p.ScaleTypography != 1.5 {
		t.Errorf("ScaleTypography = %v, want 1.5", p.ScaleTypography)
	}
	if p.DebounceInterval != 250*time.Millisecond {
		t.Errorf("DebounceInterval = %v, want 250ms", p.DebounceInterval)
	}
}

func TestProfilesHavePositiveMinTouchTarget(t *testing.T) {
	for _, tc := range []struct {
		name    string
		profile InteractionProfile
	}{
		{"Desktop", ProfileDesktop},
		{"Touch", ProfileTouch},
		{"HMI", ProfileHMI},
	} {
		if tc.profile.MinTouchTarget <= 0 {
			t.Errorf("%s: MinTouchTarget = %v, want > 0", tc.name, tc.profile.MinTouchTarget)
		}
	}
}

func TestProfilesHavePositiveLongPressDuration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		profile InteractionProfile
	}{
		{"Desktop", ProfileDesktop},
		{"Touch", ProfileTouch},
		{"HMI", ProfileHMI},
	} {
		if tc.profile.LongPressDuration <= 0 {
			t.Errorf("%s: LongPressDuration = %v, want > 0", tc.name, tc.profile.LongPressDuration)
		}
	}
}
