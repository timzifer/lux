package html

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

func TestResolveDisplay(t *testing.T) {
	tests := []struct {
		tag      string
		cssDisp  string
		expected string
	}{
		{"div", "", "block"},
		{"span", "", "inline"},
		{"table", "", "table"},
		{"tr", "", "table-row"},
		{"td", "", "table-cell"},
		{"li", "", "list-item"},
		{"script", "", "none"},
		{"div", "flex", "flex"},
		{"span", "block", "block"},
		{"div", "none", "none"},
		{"unknown", "", "inline"},
	}

	for _, tt := range tests {
		node := dom.NewElement(tt.tag)
		style := css.NewDecl()
		if tt.cssDisp != "" {
			style.Set("display", tt.cssDisp)
		}
		got := resolveDisplay(node, style)
		if got != tt.expected {
			t.Errorf("resolveDisplay(%q, %q) = %q, want %q", tt.tag, tt.cssDisp, got, tt.expected)
		}
	}
}

func TestIsBlockDisplay(t *testing.T) {
	if !isBlockDisplay("block") {
		t.Error("block should be block")
	}
	if !isBlockDisplay("flex") {
		t.Error("flex should be block")
	}
	if isBlockDisplay("inline") {
		t.Error("inline should not be block")
	}
}

func TestToSpanStyle(t *testing.T) {
	style := css.NewDecl()
	style.Set("font-family", "\"Helvetica\", Arial")
	style.Set("font-size", "16px")
	style.Set("font-weight", "bold")
	style.Set("font-style", "italic")
	style.Set("color", "#ff0000")

	ss := toSpanStyle(style)
	if ss.Style.FontFamily != "Helvetica" {
		t.Errorf("FontFamily = %q, want %q", ss.Style.FontFamily, "Helvetica")
	}
	if ss.Style.Size != 16 {
		t.Errorf("Size = %v, want 16", ss.Style.Size)
	}
	if ss.Style.Weight != draw.FontWeightBold {
		t.Errorf("Weight = %v, want %v", ss.Style.Weight, draw.FontWeightBold)
	}
	if ss.Style.Style != draw.FontStyleItalic {
		t.Errorf("Style = %v, want FontStyleItalic", ss.Style.Style)
	}
	if ss.Color == (draw.Color{}) {
		t.Error("Color should be non-zero")
	}
}

func TestToParagraphStyle(t *testing.T) {
	style := css.NewDecl()
	style.Set("text-align", "center")
	style.Set("line-height", "1.5")
	style.Set("text-indent", "20px")

	ps := toParagraphStyle(style)
	if ps.Align != draw.TextAlignCenter {
		t.Errorf("Align = %v, want Center", ps.Align)
	}
	if ps.LineHeight != 1.5 {
		t.Errorf("LineHeight = %v, want 1.5", ps.LineHeight)
	}
	if ps.Indent != 20 {
		t.Errorf("Indent = %v, want 20", ps.Indent)
	}
}

func TestToFlexContainer(t *testing.T) {
	style := css.NewDecl()
	style.Set("flex-direction", "column")
	style.Set("justify-content", "center")
	style.Set("align-items", "stretch")
	style.Set("gap", "8px")

	flex := toFlexContainer(style, nil)
	if flex.Direction != layout.FlexColumn {
		t.Errorf("Direction = %v, want FlexColumn", flex.Direction)
	}
	if flex.Justify != layout.JustifyCenter {
		t.Errorf("Justify = %v, want JustifyCenter", flex.Justify)
	}
	if flex.Align != layout.AlignStretch {
		t.Errorf("Align = %v, want AlignStretch", flex.Align)
	}
	if flex.RowGap != 8 || flex.ColGap != 8 {
		t.Errorf("Gap = (%v, %v), want (8, 8)", flex.RowGap, flex.ColGap)
	}
}

func TestApplyBoxStyle(t *testing.T) {
	style := css.NewDecl()
	style.Set("padding", "10px")
	style.Set("background-color", "#ffffff")
	style.Set("border", "2px solid #000000")

	inner := display.Text("test")
	result := applyBoxStyle(inner, style)

	// Should be wrapped in a StyledBox.
	box, ok := result.(StyledBox)
	if !ok {
		t.Fatalf("expected StyledBox, got %T", result)
	}
	if box.Padding != [4]float32{10, 10, 10, 10} {
		t.Errorf("Padding = %v, want [10,10,10,10]", box.Padding)
	}
	if box.Background == (draw.Color{}) {
		t.Error("Background should be set")
	}
	if box.BorderWidth != 2 {
		t.Errorf("BorderWidth = %v, want 2", box.BorderWidth)
	}
}

func TestApplyBoxStyleNoStyle(t *testing.T) {
	style := css.NewDecl()
	inner := display.Text("test")
	result := applyBoxStyle(inner, style)

	// No box styling — should return original element.
	if _, ok := result.(StyledBox); ok {
		t.Error("expected original element, not StyledBox")
	}
}

func TestApplyBoxStyleWithDimensions(t *testing.T) {
	style := css.NewDecl()
	style.Set("width", "200px")
	style.Set("height", "100px")
	style.Set("margin", "5px 10px")

	inner := display.Text("test")
	result := applyBoxStyle(inner, style)

	box, ok := result.(StyledBox)
	if !ok {
		t.Fatalf("expected StyledBox, got %T", result)
	}
	if box.Width != 200 {
		t.Errorf("Width = %v, want 200", box.Width)
	}
	if box.Height != 100 {
		t.Errorf("Height = %v, want 100", box.Height)
	}
	if box.Margin != [4]float32{5, 10, 5, 10} {
		t.Errorf("Margin = %v, want [5,10,5,10]", box.Margin)
	}
}

// Ensure display is imported.
var _ = display.Text
