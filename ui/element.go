// Package ui defines the Element types for the virtual tree.
package ui

import "strings"

// Element is the base interface for all virtual tree nodes (RFC §4.3).
type Element interface {
	isElement()
}

// LayoutAxis controls how a box arranges its children.
type LayoutAxis int

const (
	AxisColumn LayoutAxis = iota
	AxisRow
)

// Color is a simple RGBA color used by the M2 software-style scene graph.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// DrawRect is a filled rectangle in framebuffer coordinates.
type DrawRect struct {
	X     int
	Y     int
	W     int
	H     int
	Color Color
}

// DrawText describes a text draw call in framebuffer coordinates.
type DrawText struct {
	X     int
	Y     int
	Scale int
	Text  string
	Color Color
}

// Scene is the fully laid-out draw list for a frame.
type Scene struct {
	Rects []DrawRect
	Texts []DrawText
}

// Palette for the M2 hello-world renderer.
var (
	BackgroundColor = Color{R: 18, G: 18, B: 20, A: 255}
	SurfaceColor    = Color{R: 28, G: 28, B: 32, A: 255}
	ButtonColor     = Color{R: 52, G: 120, B: 246, A: 255}
	ButtonEdgeColor = Color{R: 126, G: 177, B: 255, A: 255}
	TextColor       = Color{R: 245, G: 247, B: 250, A: 255}
)

// Empty returns an Element that renders nothing.
func Empty() Element {
	return emptyElement{}
}

// Text creates a text element.
func Text(content string) Element {
	return textElement{Content: content}
}

// Button creates a simple button element.
// The callback is accepted for API-shape compatibility with later milestones.
func Button(label string, onClick func()) Element {
	return buttonElement{Label: label, OnClick: onClick}
}

// Column stacks children vertically.
func Column(children ...Element) Element {
	return boxElement{Axis: AxisColumn, Children: children}
}

// Row stacks children horizontally.
func Row(children ...Element) Element {
	return boxElement{Axis: AxisRow, Children: children}
}

type emptyElement struct{}

func (emptyElement) isElement() {}

type textElement struct {
	Content string
}

func (textElement) isElement() {}

type buttonElement struct {
	Label   string
	OnClick func()
}

func (buttonElement) isElement() {}

type boxElement struct {
	Axis     LayoutAxis
	Children []Element
}

func (boxElement) isElement() {}

type bounds struct {
	X int
	Y int
	W int
	H int
}

const (
	framePadding   = 24
	columnGap      = 16
	rowGap         = 12
	textScale      = 3
	charWidth      = 6
	charHeight     = 7
	buttonPadX     = 18
	buttonPadY     = 12
	buttonMinWidth = 180
)

// BuildScene converts an Element tree into a simple scene for the M2 renderer.
func BuildScene(root Element, width, height int) Scene {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	scene := Scene{}
	layoutElement(root, bounds{X: framePadding, Y: framePadding, W: max(width-(framePadding*2), 0), H: max(height-(framePadding*2), 0)}, &scene)
	return scene
}

func layoutElement(el Element, area bounds, scene *Scene) bounds {
	switch node := el.(type) {
	case nil:
		return bounds{X: area.X, Y: area.Y}
	case emptyElement:
		return bounds{X: area.X, Y: area.Y}
	case textElement:
		w, h := measureText(node.Content, textScale)
		scene.Texts = append(scene.Texts, DrawText{
			X:     area.X,
			Y:     area.Y,
			Scale: textScale,
			Text:  node.Content,
			Color: TextColor,
		})
		return bounds{X: area.X, Y: area.Y, W: w, H: h}
	case buttonElement:
		labelW, labelH := measureText(node.Label, textScale)
		w := max(buttonMinWidth, labelW+(buttonPadX*2))
		h := labelH + (buttonPadY * 2)
		scene.Rects = append(scene.Rects,
			DrawRect{X: area.X, Y: area.Y, W: w, H: h, Color: ButtonEdgeColor},
			DrawRect{X: area.X + 2, Y: area.Y + 2, W: max(w-4, 0), H: max(h-4, 0), Color: ButtonColor},
		)
		scene.Texts = append(scene.Texts, DrawText{
			X:     area.X + (w-labelW)/2,
			Y:     area.Y + (h-labelH)/2,
			Scale: textScale,
			Text:  node.Label,
			Color: TextColor,
		})
		return bounds{X: area.X, Y: area.Y, W: w, H: h}
	case boxElement:
		return layoutBox(node, area, scene)
	default:
		return bounds{X: area.X, Y: area.Y}
	}
}

func layoutBox(node boxElement, area bounds, scene *Scene) bounds {
	cursorX := area.X
	cursorY := area.Y
	maxW := 0
	maxH := 0
	count := 0

	for _, child := range node.Children {
		childBounds := layoutElement(child, bounds{X: cursorX, Y: cursorY, W: area.W, H: area.H}, scene)
		if childBounds.W == 0 && childBounds.H == 0 {
			continue
		}
		count++
		if node.Axis == AxisRow {
			cursorX += childBounds.W + rowGap
			maxW = max(maxW, cursorX-area.X-rowGap)
			maxH = max(maxH, childBounds.H)
		} else {
			cursorY += childBounds.H + columnGap
			maxW = max(maxW, childBounds.W)
			maxH = max(maxH, cursorY-area.Y-columnGap)
		}
	}

	if count == 0 {
		return bounds{X: area.X, Y: area.Y}
	}

	return bounds{X: area.X, Y: area.Y, W: maxW, H: maxH}
}

func measureText(text string, scale int) (int, int) {
	if scale <= 0 {
		scale = 1
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" && text != "" {
		trimmed = text
	}
	length := len([]rune(trimmed))
	if length == 0 {
		length = len([]rune(text))
	}
	if length == 0 {
		return 0, 0
	}
	return length * charWidth * scale, charHeight * scale
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
