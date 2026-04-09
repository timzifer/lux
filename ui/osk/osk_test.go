package osk

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/uitest"
)

// ── ComputeKeySize Tests ────────────────────────────────────────

func TestComputeKeySize_Alpha(t *testing.T) {
	keyW, keyH, gap := ComputeKeySize(1024, 600, 1.0, ModeAlpha)
	if gap != 4.0 {
		t.Errorf("expected gap=4.0, got %f", gap)
	}
	if keyW < 28 {
		t.Errorf("keyW too small: %f", keyW)
	}
	if keyW > 68 {
		t.Errorf("keyW exceeds DPI cap (68dp): %f", keyW)
	}
	if keyH < 28 {
		t.Errorf("keyH too small: %f", keyH)
	}
}

func TestComputeKeySize_DPICap(t *testing.T) {
	// Large screen should have keys capped at 68dp.
	keyW, keyH, _ := ComputeKeySize(3840, 2160, 2.0, ModeAlpha)
	if keyW > 68 {
		t.Errorf("keyW should be capped at 68dp, got %f", keyW)
	}
	if keyH > 68 {
		t.Errorf("keyH should be capped at 68dp, got %f", keyH)
	}
}

func TestComputeKeySize_SmallScreen(t *testing.T) {
	// Small screen: keys should hit minimum.
	keyW, keyH, _ := ComputeKeySize(200, 200, 1.0, ModeAlpha)
	if keyW < 28 {
		t.Errorf("keyW below minimum: %f", keyW)
	}
	if keyH < 28 {
		t.Errorf("keyH below minimum: %f", keyH)
	}
}

func TestComputeKeySize_NumPad(t *testing.T) {
	keyW, _, _ := ComputeKeySize(1024, 600, 1.0, ModeNumPad)
	// NumPad has 3 keys per row → should be wider than alpha keys.
	alphaKeyW, _, _ := ComputeKeySize(1024, 600, 1.0, ModeAlpha)
	// But capped, so both might be capped at the same value on large screens.
	if keyW < alphaKeyW {
		t.Errorf("numpad key (%f) should be >= alpha key (%f)", keyW, alphaKeyW)
	}
}

// ── Layout Tests ────────────────────────────────────────────────

func TestAlphaLayout_RowCount(t *testing.T) {
	rows := alphaRowsLower()
	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
}

func TestAlphaLayout_NumberRow(t *testing.T) {
	rows := alphaRowsLower()
	if len(rows[0]) != 10 {
		t.Errorf("number row should have 10 keys, got %d", len(rows[0]))
	}
	for i, k := range rows[0] {
		if k.Action != OSKActionChar {
			t.Errorf("number row key %d should be OSKActionChar", i)
		}
	}
}

func TestAlphaLayout_QWERTZRow(t *testing.T) {
	rows := alphaRowsLower()
	labels := ""
	for _, k := range rows[1] {
		labels += k.Label
	}
	if labels != "qwertzuiop" {
		t.Errorf("expected qwertzuiop, got %s", labels)
	}
}

func TestNumpadLayout_RowCount(t *testing.T) {
	rows := numericRows()
	if len(rows) != 4 {
		t.Errorf("expected 4 rows, got %d", len(rows))
	}
}

func TestNumpadLayout_Keys(t *testing.T) {
	rows := numericRows()
	// First row should be 7, 8, 9.
	if len(rows[0]) != 3 {
		t.Errorf("expected 3 keys in first row, got %d", len(rows[0]))
	}
	if rows[0][0].Char != '7' || rows[0][1].Char != '8' || rows[0][2].Char != '9' {
		t.Errorf("expected 7,8,9 in first row")
	}
}

func TestCondensedLayout_RowCount(t *testing.T) {
	rows := condensedRows(false)
	if len(rows) != 4 {
		t.Errorf("expected 4 rows, got %d", len(rows))
	}
}

func TestShiftToggle(t *testing.T) {
	lower := alphaRowsLower()
	upper := alphaRowsShifted()

	// Second row: lowercase should have q, uppercase should have Q.
	if lower[1][0].Label != "q" {
		t.Errorf("expected lowercase q, got %s", lower[1][0].Label)
	}
	if upper[1][0].Label != "Q" {
		t.Errorf("expected uppercase Q, got %s", upper[1][0].Label)
	}
}

func TestFullLayout_HasBothSections(t *testing.T) {
	rows := fullRows(false)
	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
	// Each row should have more keys than alpha alone (alpha + separator + numpad).
	alphaRows := alphaRowsLower()
	for i, row := range rows {
		if len(row) <= len(alphaRows[i]) {
			t.Errorf("row %d: full layout should have more keys than alpha alone", i)
		}
	}
}

// ── ModeForLayout Tests ─────────────────────────────────────────

func TestModeForLayout(t *testing.T) {
	tests := []struct {
		layout OSKLayout
		want   OSKMode
	}{
		{OSKLayoutAlpha, ModeAlpha},
		{OSKLayoutNumeric, ModeNumPad},
		{OSKLayoutNumericInteger, ModeNumPad},
		{OSKLayoutPhone, ModeNumPad},
	}
	for _, tt := range tests {
		got := ModeForLayout(tt.layout)
		if got != tt.want {
			t.Errorf("ModeForLayout(%d) = %d, want %d", tt.layout, got, tt.want)
		}
	}
}

// ── OSKState Height Tests ───────────────────────────────────────

func TestOSKState_Height_Hidden(t *testing.T) {
	s := &OSKState{Visible: false}
	if h := s.Height(1024, 600, 1.0); h != 0 {
		t.Errorf("expected 0 height when hidden, got %f", h)
	}
}

func TestOSKState_Height_Visible(t *testing.T) {
	s := &OSKState{Visible: true, Mode: ModeAlpha}
	h := s.Height(1024, 600, 1.0)
	if h <= 0 {
		t.Errorf("expected positive height when visible, got %f", h)
	}
	if h > 600 {
		t.Errorf("height should not exceed screen height, got %f", h)
	}
}

func TestOSKState_Height_Nil(t *testing.T) {
	var s *OSKState
	if h := s.Height(1024, 600, 1.0); h != 0 {
		t.Errorf("expected 0 height for nil state, got %f", h)
	}
}

// ── RowsForState Tests ──────────────────────────────────────────

func TestRowsForState_AllModes(t *testing.T) {
	modes := []OSKMode{ModeAlpha, ModeNumPad, ModeFull, ModeCondensed}
	for _, mode := range modes {
		state := &OSKState{Visible: true, Mode: mode}
		rows := RowsForState(state)
		if len(rows) == 0 {
			t.Errorf("mode %d should produce non-empty rows", mode)
		}
		for ri, row := range rows {
			if len(row) == 0 && mode != ModeFull {
				t.Errorf("mode %d row %d is empty", mode, ri)
			}
		}
	}
}

func TestRowsForState_Nil(t *testing.T) {
	rows := RowsForState(nil)
	if len(rows) != 5 {
		t.Errorf("nil state should produce alpha layout (5 rows), got %d", len(rows))
	}
}

// ── Golden-File Tests ───────────────────────────────────────────

func buildOSKScene(state *OSKState, w, h int) draw.Scene {
	el := NewOSKElement(state, w, h)
	canvas := render.NewSceneCanvas(w, h)
	area := ui.Bounds{X: 0, Y: 0, W: w, H: h}
	if l, ok := el.(ui.Layouter); ok {
		ctx := &ui.LayoutContext{
			Area:   area,
			Canvas: canvas,
			Theme:  theme.Default,
			Tokens: theme.Default.Tokens(),
		}
		l.LayoutSelf(ctx)
	}
	return canvas.Scene()
}

func TestOSKGolden_Alpha(t *testing.T) {
	state := &OSKState{Visible: true, Mode: ModeAlpha, Layout: OSKLayoutAlpha}
	scene := buildOSKScene(state, 800, 600)
	uitest.AssertScene(t, scene, "testdata/osk_alpha.golden")
}

func TestOSKGolden_AlphaShifted(t *testing.T) {
	state := &OSKState{Visible: true, Mode: ModeAlpha, Layout: OSKLayoutAlpha, Shifted: true}
	scene := buildOSKScene(state, 800, 600)
	uitest.AssertScene(t, scene, "testdata/osk_alpha_shifted.golden")
}

func TestOSKGolden_NumPad(t *testing.T) {
	state := &OSKState{Visible: true, Mode: ModeNumPad, Layout: OSKLayoutNumeric}
	scene := buildOSKScene(state, 800, 600)
	uitest.AssertScene(t, scene, "testdata/osk_numpad.golden")
}

func TestOSKGolden_Full(t *testing.T) {
	state := &OSKState{Visible: true, Mode: ModeFull, Layout: OSKLayoutAlpha}
	scene := buildOSKScene(state, 1024, 600)
	uitest.AssertScene(t, scene, "testdata/osk_full.golden")
}

func TestOSKGolden_Condensed(t *testing.T) {
	state := &OSKState{Visible: true, Mode: ModeCondensed, Layout: OSKLayoutAlpha}
	scene := buildOSKScene(state, 400, 700)
	uitest.AssertScene(t, scene, "testdata/osk_condensed.golden")
}

func TestOSKGolden_Hidden(t *testing.T) {
	state := &OSKState{Visible: false}
	scene := buildOSKScene(state, 800, 600)
	uitest.AssertScene(t, scene, "testdata/osk_hidden.golden")
}
