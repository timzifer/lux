//go:build nogui

package layout_test

import (
	"testing"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/render"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/layout"
)

// buildScene lays out and renders a root element into a scene.
func buildScene(root ui.Element, w, h int) draw.Scene {
	canvas := render.NewSceneCanvas(w, h)
	return ui.BuildScene(root, canvas, theme.Default, w, h, nil)
}

// sizedBox creates a SizedBox element with fixed dimensions.
func sizedBox(w, h float32) ui.Element {
	return layout.Sized(w, h, nil)
}

// ── Flex: basic row layout ──────────────────────────────────────

func TestFlexRowBasic(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30),
		sizedBox(60, 40),
		sizedBox(70, 20),
	})
	buildScene(flex, 800, 600)
}

func TestFlexColumnBasic(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30),
		sizedBox(60, 40),
		sizedBox(70, 20),
	}, layout.WithDirection(layout.FlexColumn))
	buildScene(flex, 800, 600)
}

func TestFlexEmpty(t *testing.T) {
	flex := layout.NewFlex(nil)
	buildScene(flex, 800, 600)
}

func TestFlexSingleChild(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{sizedBox(100, 50)})
	buildScene(flex, 800, 600)
}

// ── Flex: reverse directions ────────────────────────────────────

func TestFlexRowReverse(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30),
		sizedBox(60, 30),
		sizedBox(70, 30),
	}, layout.WithDirection(layout.FlexRowReverse))
	buildScene(flex, 800, 600)
}

func TestFlexColumnReverse(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30),
		sizedBox(60, 40),
	}, layout.WithDirection(layout.FlexColumnReverse))
	buildScene(flex, 800, 600)
}

// ── Flex: justify ───────────────────────────────────────────────

func TestFlexJustifyCenter(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{sizedBox(100, 30)},
		layout.WithJustify(layout.JustifyCenter))
	buildScene(flex, 400, 100)
}

func TestFlexJustifyEnd(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{sizedBox(100, 30)},
		layout.WithJustify(layout.JustifyEnd))
	buildScene(flex, 400, 100)
}

func TestFlexJustifySpaceBetween(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30), sizedBox(50, 30), sizedBox(50, 30),
	}, layout.WithJustify(layout.JustifySpaceBetween))
	buildScene(flex, 400, 100)
}

func TestFlexJustifySpaceAround(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30), sizedBox(50, 30),
	}, layout.WithJustify(layout.JustifySpaceAround))
	buildScene(flex, 400, 100)
}

func TestFlexJustifySpaceEvenly(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30), sizedBox(50, 30),
	}, layout.WithJustify(layout.JustifySpaceEvenly))
	buildScene(flex, 400, 100)
}

// ── Flex: align ─────────────────────────────────────────────────

func TestFlexAlignCenter(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 20), sizedBox(50, 60),
	}, layout.WithAlign(layout.AlignCenter))
	buildScene(flex, 400, 100)
}

func TestFlexAlignEnd(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 20), sizedBox(50, 60),
	}, layout.WithAlign(layout.AlignEnd))
	buildScene(flex, 400, 100)
}

func TestFlexAlignStretch(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 20), sizedBox(50, 60),
	}, layout.WithAlign(layout.AlignStretch))
	buildScene(flex, 400, 100)
}

// ── Flex: grow ──────────────────────────────────────────────────

func TestFlexGrow(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(100, 30),
		layout.Expand(sizedBox(0, 30)),
		sizedBox(100, 30),
	})
	buildScene(flex, 400, 100)
}

func TestFlexGrowMultiple(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.Expand(sizedBox(0, 30), 1),
		layout.Expand(sizedBox(0, 30), 2),
		layout.Expand(sizedBox(0, 30), 3),
	})
	buildScene(flex, 600, 100)
}

// ── Flex: shrink ────────────────────────────────────────────────

func TestFlexShrink(t *testing.T) {
	// Children total 600dp in 400dp container — must shrink.
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(200, 30), layout.WithShrink(1)),
		layout.FlexChild(sizedBox(200, 30), layout.WithShrink(1)),
		layout.FlexChild(sizedBox(200, 30), layout.WithShrink(1)),
	})
	buildScene(flex, 400, 100)
}

func TestFlexShrinkProportional(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(200, 30), layout.WithShrink(1)),
		layout.FlexChild(sizedBox(200, 30), layout.WithShrink(2)),
	})
	buildScene(flex, 300, 100)
}

// ── Flex: basis ─────────────────────────────────────────────────

func TestFlexBasisFixed(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(0, 30), layout.WithBasis(layout.FixedBasis(150)), layout.WithGrow(1)),
		layout.FlexChild(sizedBox(0, 30), layout.WithBasis(layout.FixedBasis(50)), layout.WithGrow(1)),
	})
	buildScene(flex, 400, 100)
}

// ── Flex: wrap ──────────────────────────────────────────────────

func TestFlexWrapRow(t *testing.T) {
	children := make([]ui.Element, 8)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithGap(8))
	buildScene(flex, 300, 400)
}

func TestFlexWrapColumn(t *testing.T) {
	children := make([]ui.Element, 6)
	for i := range children {
		children[i] = sizedBox(80, 50)
	}
	flex := layout.NewFlex(children,
		layout.WithDirection(layout.FlexColumn),
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithGap(8))
	buildScene(flex, 400, 150)
}

func TestFlexWrapReverse(t *testing.T) {
	children := make([]ui.Element, 6)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapReverse),
		layout.WithGap(8))
	buildScene(flex, 200, 400)
}

// ── Flex: align-self ────────────────────────────────────────────

func TestFlexAlignSelf(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(50, 20), layout.WithAlignSelf(layout.AlignSelfStart)),
		layout.FlexChild(sizedBox(50, 20), layout.WithAlignSelf(layout.AlignSelfCenter)),
		layout.FlexChild(sizedBox(50, 20), layout.WithAlignSelf(layout.AlignSelfEnd)),
		layout.FlexChild(sizedBox(50, 20), layout.WithAlignSelf(layout.AlignSelfStretch)),
	}, layout.WithAlign(layout.AlignStart))
	buildScene(flex, 400, 100)
}

// ── Flex: order ─────────────────────────────────────────────────

func TestFlexOrder(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(50, 30), layout.WithOrder(3)),
		layout.FlexChild(sizedBox(60, 30), layout.WithOrder(1)),
		layout.FlexChild(sizedBox(70, 30), layout.WithOrder(2)),
	})
	buildScene(flex, 400, 100)
}

// ── Flex: align-content ─────────────────────────────────────────

func TestFlexAlignContentCenter(t *testing.T) {
	children := make([]ui.Element, 6)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentCenter))
	buildScene(flex, 200, 400)
}

func TestFlexAlignContentSpaceBetween(t *testing.T) {
	children := make([]ui.Element, 6)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentSpaceBetween))
	buildScene(flex, 200, 400)
}

func TestFlexAlignContentEnd(t *testing.T) {
	children := make([]ui.Element, 4)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentEnd))
	buildScene(flex, 200, 300)
}

func TestFlexAlignContentSpaceAround(t *testing.T) {
	children := make([]ui.Element, 4)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentSpaceAround))
	buildScene(flex, 200, 300)
}

func TestFlexAlignContentStretch(t *testing.T) {
	children := make([]ui.Element, 4)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentStretch))
	buildScene(flex, 200, 300)
}

// ── Flex: separate gaps ─────────────────────────────────────────

func TestFlexSeparateGaps(t *testing.T) {
	children := make([]ui.Element, 6)
	for i := range children {
		children[i] = sizedBox(80, 30)
	}
	flex := layout.NewFlex(children,
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithFlexColGap(16),
		layout.WithFlexRowGap(8))
	buildScene(flex, 300, 400)
}

// ── Flex: combined features ─────────────────────────────────────

func TestFlexWrapWithGrowAndAlignContent(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.Expand(sizedBox(100, 30), 1),
		layout.Expand(sizedBox(100, 30), 1),
		layout.Expand(sizedBox(100, 30), 1),
		layout.Expand(sizedBox(100, 30), 1),
	},
		layout.WithWrap(layout.FlexWrapOn),
		layout.WithAlignContent(layout.AlignContentCenter),
		layout.WithGap(8))
	buildScene(flex, 250, 200)
}

func TestFlexReverseWithJustifyEnd(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		sizedBox(50, 30), sizedBox(50, 30),
	},
		layout.WithDirection(layout.FlexRowReverse),
		layout.WithJustify(layout.JustifyEnd))
	buildScene(flex, 400, 100)
}

func TestFlexOrderWithWrap(t *testing.T) {
	flex := layout.NewFlex([]ui.Element{
		layout.FlexChild(sizedBox(80, 30), layout.WithOrder(2)),
		layout.FlexChild(sizedBox(80, 30), layout.WithOrder(0)),
		layout.FlexChild(sizedBox(80, 30), layout.WithOrder(1)),
		layout.FlexChild(sizedBox(80, 30), layout.WithOrder(3)),
	}, layout.WithWrap(layout.FlexWrapOn))
	buildScene(flex, 200, 200)
}
