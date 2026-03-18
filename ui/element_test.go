package ui

import "testing"

func TestBuildSceneEmpty(t *testing.T) {
	scene := BuildScene(Empty(), 800, 600)
	if len(scene.Rects) != 0 {
		t.Errorf("Empty element should produce 0 rects, got %d", len(scene.Rects))
	}
	if len(scene.Texts) != 0 {
		t.Errorf("Empty element should produce 0 texts, got %d", len(scene.Texts))
	}
}

func TestBuildSceneText(t *testing.T) {
	scene := BuildScene(Text("HELLO WORLD"), 800, 600)
	if len(scene.Texts) != 1 {
		t.Fatalf("Text element should produce 1 text, got %d", len(scene.Texts))
	}
	txt := scene.Texts[0]
	if txt.Text != "HELLO WORLD" {
		t.Errorf("text content = %q, want %q", txt.Text, "HELLO WORLD")
	}
	if txt.X != framePadding {
		t.Errorf("text X = %d, want %d", txt.X, framePadding)
	}
	if txt.Y != framePadding {
		t.Errorf("text Y = %d, want %d", txt.Y, framePadding)
	}
	if txt.Scale != textScale {
		t.Errorf("text Scale = %d, want %d", txt.Scale, textScale)
	}
	if txt.Color != TextColor {
		t.Errorf("text Color = %v, want %v", txt.Color, TextColor)
	}
}

func TestBuildSceneButton(t *testing.T) {
	scene := BuildScene(Button("OK", nil), 800, 600)

	// Button produces 2 rects (edge + fill) and 1 text (label).
	if len(scene.Rects) != 2 {
		t.Fatalf("Button should produce 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Texts) != 1 {
		t.Fatalf("Button should produce 1 text, got %d", len(scene.Texts))
	}

	edge := scene.Rects[0]
	fill := scene.Rects[1]
	label := scene.Texts[0]

	// Edge rect is at framePadding origin.
	if edge.X != framePadding || edge.Y != framePadding {
		t.Errorf("edge origin = (%d,%d), want (%d,%d)", edge.X, edge.Y, framePadding, framePadding)
	}
	if edge.Color != ButtonEdgeColor {
		t.Errorf("edge color = %v, want %v", edge.Color, ButtonEdgeColor)
	}

	// Fill rect is inset by 2px.
	if fill.X != framePadding+2 || fill.Y != framePadding+2 {
		t.Errorf("fill origin = (%d,%d), want (%d,%d)", fill.X, fill.Y, framePadding+2, framePadding+2)
	}
	if fill.W != edge.W-4 || fill.H != edge.H-4 {
		t.Errorf("fill size = %dx%d, want %dx%d", fill.W, fill.H, edge.W-4, edge.H-4)
	}
	if fill.Color != ButtonColor {
		t.Errorf("fill color = %v, want %v", fill.Color, ButtonColor)
	}

	// Label text must match.
	if label.Text != "OK" {
		t.Errorf("label text = %q, want %q", label.Text, "OK")
	}
	if label.Color != TextColor {
		t.Errorf("label color = %v, want %v", label.Color, TextColor)
	}

	// Label must be inside the button bounds.
	if label.X < edge.X || label.X >= edge.X+edge.W {
		t.Errorf("label X=%d outside button [%d, %d)", label.X, edge.X, edge.X+edge.W)
	}
	if label.Y < edge.Y || label.Y >= edge.Y+edge.H {
		t.Errorf("label Y=%d outside button [%d, %d)", label.Y, edge.Y, edge.Y+edge.H)
	}
}

func TestBuildSceneColumnTextAndButton(t *testing.T) {
	// This is the M2 hello-world layout.
	scene := BuildScene(Column(
		Text("HELLO WORLD"),
		Button("CLICK ME", nil),
	), 800, 600)

	// Expect: 2 rects (button edge+fill), 2 texts (label + button label).
	if len(scene.Rects) != 2 {
		t.Errorf("M2 scene should have 2 rects, got %d", len(scene.Rects))
	}
	if len(scene.Texts) != 2 {
		t.Errorf("M2 scene should have 2 texts, got %d", len(scene.Texts))
	}

	if len(scene.Texts) < 2 {
		t.FailNow()
	}

	hello := scene.Texts[0]
	click := scene.Texts[1]

	if hello.Text != "HELLO WORLD" {
		t.Errorf("first text = %q, want %q", hello.Text, "HELLO WORLD")
	}
	if click.Text != "CLICK ME" {
		t.Errorf("second text = %q, want %q", click.Text, "CLICK ME")
	}

	// Button must be below the text (Y increases downward).
	if click.Y <= hello.Y {
		t.Errorf("button label Y=%d should be below text Y=%d", click.Y, hello.Y)
	}

	// Both must be within the framebuffer.
	for _, txt := range scene.Texts {
		if txt.X < 0 || txt.Y < 0 || txt.X >= 800 || txt.Y >= 600 {
			t.Errorf("text %q at (%d,%d) is outside 800x600 framebuffer", txt.Text, txt.X, txt.Y)
		}
	}
	for _, rect := range scene.Rects {
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 800 || rect.Y+rect.H > 600 {
			t.Errorf("rect at (%d,%d) size %dx%d extends outside 800x600 framebuffer", rect.X, rect.Y, rect.W, rect.H)
		}
	}
}

func TestBuildSceneRow(t *testing.T) {
	scene := BuildScene(Row(
		Text("A"),
		Text("B"),
	), 800, 600)

	if len(scene.Texts) != 2 {
		t.Fatalf("Row with 2 texts should produce 2 texts, got %d", len(scene.Texts))
	}

	a := scene.Texts[0]
	b := scene.Texts[1]

	// Same Y (horizontal layout).
	if a.Y != b.Y {
		t.Errorf("Row children should share Y: a.Y=%d, b.Y=%d", a.Y, b.Y)
	}
	// B must be to the right of A.
	if b.X <= a.X {
		t.Errorf("b.X=%d should be > a.X=%d", b.X, a.X)
	}
}

func TestMeasureText(t *testing.T) {
	w, h := measureText("ABC", 1)
	if w != 3*charWidth*1 {
		t.Errorf("width = %d, want %d", w, 3*charWidth)
	}
	if h != charHeight*1 {
		t.Errorf("height = %d, want %d", h, charHeight)
	}

	w2, h2 := measureText("ABC", 3)
	if w2 != 3*charWidth*3 {
		t.Errorf("scaled width = %d, want %d", w2, 3*charWidth*3)
	}
	if h2 != charHeight*3 {
		t.Errorf("scaled height = %d, want %d", h2, charHeight*3)
	}

	w0, h0 := measureText("", 1)
	if w0 != 0 || h0 != 0 {
		t.Errorf("empty text should be 0x0, got %dx%d", w0, h0)
	}
}

func TestBuildSceneDefaultSize(t *testing.T) {
	// Zero dimensions should fall back to 800x600.
	scene := BuildScene(Text("X"), 0, 0)
	if len(scene.Texts) != 1 {
		t.Fatalf("expected 1 text, got %d", len(scene.Texts))
	}
	if scene.Texts[0].X != framePadding {
		t.Errorf("X = %d, want %d (framePadding)", scene.Texts[0].X, framePadding)
	}
}
