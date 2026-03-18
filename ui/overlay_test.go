package ui

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestComputeOverlayPosition_Below(t *testing.T) {
	anchor := draw.R(100, 50, 200, 30) // x=100, y=50, w=200, h=30
	content := draw.Size{W: 150, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementBelow, content, 800, 600)
	if pos.X != 100 || pos.Y != 80 { // 50 + 30 = 80
		t.Errorf("PlacementBelow: got (%f, %f), want (100, 80)", pos.X, pos.Y)
	}
}

func TestComputeOverlayPosition_Above(t *testing.T) {
	anchor := draw.R(100, 200, 200, 30)
	content := draw.Size{W: 150, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementAbove, content, 800, 600)
	if pos.X != 100 || pos.Y != 100 { // 200 - 100 = 100
		t.Errorf("PlacementAbove: got (%f, %f), want (100, 100)", pos.X, pos.Y)
	}
}

func TestComputeOverlayPosition_Center(t *testing.T) {
	anchor := draw.R(0, 0, 0, 0) // ignored for center
	content := draw.Size{W: 200, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementCenter, content, 800, 600)
	if pos.X != 300 || pos.Y != 250 { // (800-200)/2=300, (600-100)/2=250
		t.Errorf("PlacementCenter: got (%f, %f), want (300, 250)", pos.X, pos.Y)
	}
}

func TestComputeOverlayPosition_Right(t *testing.T) {
	anchor := draw.R(100, 50, 200, 30)
	content := draw.Size{W: 150, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementRight, content, 800, 600)
	if pos.X != 300 || pos.Y != 50 { // 100 + 200 = 300
		t.Errorf("PlacementRight: got (%f, %f), want (300, 50)", pos.X, pos.Y)
	}
}

func TestComputeOverlayPosition_FlipBelowToAbove(t *testing.T) {
	// Anchor near bottom → PlacementBelow should flip to above.
	anchor := draw.R(100, 550, 200, 30)
	content := draw.Size{W: 150, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementBelow, content, 800, 600)
	// 550 + 30 = 580, 580 + 100 = 680 > 600 → flip above: 550 - 100 = 450
	if pos.Y != 450 {
		t.Errorf("PlacementBelow flip: Y = %f, want 450", pos.Y)
	}
}

func TestComputeOverlayPosition_ClampToWindow(t *testing.T) {
	// Overlay would exceed right edge.
	anchor := draw.R(700, 50, 50, 30)
	content := draw.Size{W: 200, H: 100}
	pos := ComputeOverlayPosition(anchor, PlacementBelow, content, 800, 600)
	if pos.X != 600 { // 800 - 200 = 600
		t.Errorf("Clamp right: X = %f, want 600", pos.X)
	}
}

func TestOverlayIsElement(t *testing.T) {
	o := Overlay{
		ID:          "test",
		Anchor:      draw.R(0, 0, 100, 30),
		Placement:   PlacementBelow,
		Content:     Empty(),
		Dismissable: true,
	}
	// Verify it implements Element.
	var _ Element = o
	o.isElement() // should compile
}
