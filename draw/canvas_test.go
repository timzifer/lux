package draw

import "testing"

func TestTextAlignConstants(t *testing.T) {
	if TextAlignLeft != 0 {
		t.Errorf("TextAlignLeft = %d, want 0", TextAlignLeft)
	}
	if TextAlignCenter != 1 {
		t.Errorf("TextAlignCenter = %d, want 1", TextAlignCenter)
	}
	if TextAlignRight != 2 {
		t.Errorf("TextAlignRight = %d, want 2", TextAlignRight)
	}
}

func TestTextLayoutDefaults(t *testing.T) {
	layout := TextLayout{
		Text:  "hello",
		Style: TextStyle{Size: 14},
	}
	if layout.MaxWidth != 0 {
		t.Error("default MaxWidth should be 0 (unbounded)")
	}
	if layout.Alignment != TextAlignLeft {
		t.Error("default Alignment should be TextAlignLeft")
	}
}

func TestImageSliceType(t *testing.T) {
	slice := ImageSlice{
		Image:  ImageID(1),
		Insets: Insets{Top: 10, Right: 10, Bottom: 10, Left: 10},
	}
	if slice.Image != 1 {
		t.Errorf("Image = %d, want 1", slice.Image)
	}
	if slice.Insets.Top != 10 {
		t.Errorf("Insets.Top = %f, want 10", slice.Insets.Top)
	}
}

func TestTextureIDType(t *testing.T) {
	var tex TextureID = 99
	if tex != 99 {
		t.Errorf("TextureID = %d, want 99", tex)
	}
}

func TestBlendModeConstants(t *testing.T) {
	if BlendNormal != 0 {
		t.Errorf("BlendNormal = %d, want 0", BlendNormal)
	}
	if BlendMultiply != 1 {
		t.Errorf("BlendMultiply = %d, want 1", BlendMultiply)
	}
}

func TestLayerOptionsDefaults(t *testing.T) {
	opts := LayerOptions{}
	if opts.BlendMode != BlendNormal {
		t.Error("default BlendMode should be BlendNormal")
	}
	if opts.Opacity != 0 {
		t.Error("default Opacity should be 0")
	}
	if opts.CacheHint {
		t.Error("default CacheHint should be false")
	}
}
