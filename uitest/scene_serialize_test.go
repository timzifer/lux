package uitest

import (
	"strings"
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestSerializeEmptyScene(t *testing.T) {
	s := draw.Scene{}
	got := SerializeScene(s)
	if !strings.HasPrefix(got, "=== Scene ===") {
		t.Errorf("expected header, got: %s", got[:min(50, len(got))])
	}
	// Empty scene should only have the header + blank line
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 1 {
		t.Errorf("empty scene should produce 1 line (header), got %d:\n%s", len(lines), got)
	}
}

func TestSerializeRectsAndGlyphs(t *testing.T) {
	s := draw.Scene{
		Rects: []draw.DrawRect{
			{X: 10, Y: 20, W: 100, H: 50, Color: draw.RGBA(255, 0, 0, 255), Radius: 4},
			{X: 0, Y: 0, W: 800, H: 600, Color: draw.RGBA(30, 30, 30, 255)},
		},
		Glyphs: []draw.DrawGlyph{
			{X: 15, Y: 25, Scale: 13, Text: "Hello", Color: draw.RGBA(255, 255, 255, 255)},
		},
	}

	got := SerializeScene(s)

	// Check structure
	if !strings.Contains(got, "[Rects] (2)") {
		t.Error("missing Rects section header")
	}
	if !strings.Contains(got, "[Glyphs] (1)") {
		t.Error("missing Glyphs section header")
	}
	if !strings.Contains(got, `"Hello"`) {
		t.Error("missing glyph text")
	}
	if !strings.Contains(got, "color(255,0,0,255)") {
		t.Error("missing red color")
	}
	if !strings.Contains(got, "r=4.0") {
		t.Error("missing radius")
	}

	// Determinism: serialize again, must be identical
	got2 := SerializeScene(s)
	if got != got2 {
		t.Error("serialization is not deterministic")
	}
}

func TestSerializeShadowAndGradient(t *testing.T) {
	s := draw.Scene{
		ShadowRects: []draw.DrawShadowRect{
			{X: 5, Y: 5, W: 90, H: 40, Color: draw.RGBA(0, 0, 0, 128), Radius: 8, BlurRadius: 12, Inset: true},
		},
		GradientRects: []draw.DrawGradientRect{
			{
				X: 0, Y: 0, W: 200, H: 100,
				Kind:      draw.PaintLinearGradient,
				StopCount: 2,
				Stops: [8]draw.GradientStop{
					{Offset: 0, Color: draw.RGBA(255, 0, 0, 255)},
					{Offset: 1, Color: draw.RGBA(0, 0, 255, 255)},
				},
			},
		},
	}

	got := SerializeScene(s)

	if !strings.Contains(got, "inset") {
		t.Error("missing inset flag on shadow")
	}
	if !strings.Contains(got, "gradient-linear") {
		t.Error("missing gradient kind")
	}
	if !strings.Contains(got, "stop[0]") {
		t.Error("missing gradient stop")
	}
}

func TestDiffScenesIdentical(t *testing.T) {
	s := draw.Scene{
		Rects: []draw.DrawRect{
			{X: 0, Y: 0, W: 100, H: 50, Color: draw.RGBA(255, 255, 255, 255)},
		},
	}
	text := SerializeScene(s)
	diff := DiffScenes(text, text)
	if diff != "" {
		t.Errorf("identical scenes should produce no diff, got:\n%s", diff)
	}
}

func TestDiffScenesChanged(t *testing.T) {
	s1 := draw.Scene{
		Rects: []draw.DrawRect{
			{X: 0, Y: 0, W: 100, H: 50, Color: draw.RGBA(255, 0, 0, 255)},
		},
	}
	s2 := draw.Scene{
		Rects: []draw.DrawRect{
			{X: 0, Y: 0, W: 200, H: 50, Color: draw.RGBA(255, 0, 0, 255)},
		},
	}
	diff := DiffScenes(SerializeScene(s1), SerializeScene(s2))
	if diff == "" {
		t.Error("different scenes should produce a diff")
	}
	if !strings.Contains(diff, "100x50") && !strings.Contains(diff, "200x50") {
		t.Errorf("diff should mention changed dimensions, got:\n%s", diff)
	}
}
