// Package ui defines the Widget system and Element types for the
// virtual tree (RFC §4).
package ui

import (
	"time"

	"github.com/timzifer/lux/anim"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/hit"
	"github.com/timzifer/lux/theme"
)

// ── Widget System (RFC §4) ───────────────────────────────────────

// WidgetState is an open interface — any type qualifies (RFC §4.1).
type WidgetState interface{}

// UID identifies a widget instance across frames.
type UID uint64

// Widget is the core interface for stateful, renderable components
// (RFC §4.2).
type Widget interface {
	// Render returns an Element tree and (optionally updated) state.
	// state is nil on the first call.
	Render(ctx RenderCtx, state WidgetState) (Element, WidgetState)
}

// RenderCtx is the context passed to Widget.Render (RFC §4.2).
type RenderCtx struct {
	UID    UID
	Theme  theme.Theme
	Send   func(any) // local Send bound to this UID
}

// AdoptState is a generic helper that type-asserts the raw state or
// returns a zero-value pointer for the first render (RFC §4.2).
func AdoptState[S WidgetState](raw WidgetState) *S {
	if s, ok := raw.(*S); ok {
		return s
	}
	return new(S)
}

// ── Element Types (RFC §4.3) ─────────────────────────────────────

// Element is the base interface for all virtual-tree nodes.
type Element interface {
	isElement()
}

// LayoutAxis controls how a Box arranges its children.
type LayoutAxis int

const (
	AxisColumn LayoutAxis = iota
	AxisRow
)

// Empty returns an Element that renders nothing.
func Empty() Element { return emptyElement{} }

// Text creates a text element.
func Text(content string) Element { return textElement{Content: content} }

// Button creates a button element with an optional click callback.
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

// WithKey wraps an element with an explicit key for stable UIDs
// across re-parenting (RFC §4.4).
func WithKey(key string, el Element) Element {
	return keyedElement{Key: key, Child: el}
}

// ── Concrete element structs ─────────────────────────────────────

type emptyElement struct{}

func (emptyElement) isElement() {}

type textElement struct{ Content string }

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

type keyedElement struct {
	Key   string
	Child Element
}

func (keyedElement) isElement() {}

// ── Hover State (M4) ────────────────────────────────────────────

// HoverState tracks hover animations for interactive elements.
// It uses the previous frame's hit targets to determine hover,
// introducing at most one frame of latency (imperceptible at 60fps).
type HoverState struct {
	hoveredIdx int                  // currently hovered button index, -1 = none
	anims      []anim.Anim[float32] // per-button hover opacity [0,1]
	buttonIdx  int                  // counter during BuildScene
	inited     bool                 // tracks whether hoveredIdx has been set
}

// SetHovered updates which button (by index) is hovered and sets animation targets.
// idx == -1 means no button is hovered. dur is the animation duration.
func (h *HoverState) SetHovered(idx int, dur time.Duration) {
	if !h.inited {
		h.hoveredIdx = -1
		h.inited = true
	}
	prev := h.hoveredIdx
	h.hoveredIdx = idx

	// Animate previous button out.
	if prev >= 0 && prev < len(h.anims) && prev != idx {
		h.anims[prev].SetTarget(0.0, dur, anim.OutCubic)
	}

	// Animate new button in.
	if idx >= 0 {
		h.ensureSize(idx + 1)
		if h.anims[idx].Value() < 1.0 {
			h.anims[idx].SetTarget(1.0, dur, anim.OutCubic)
		}
	}
}

// Tick advances all hover animations by dt.
func (h *HoverState) Tick(dt time.Duration) {
	for i := range h.anims {
		h.anims[i].Tick(dt)
	}
}

// resetCounter prepares for a new BuildScene pass.
func (h *HoverState) resetCounter() { h.buttonIdx = 0 }

// nextButtonHoverOpacity returns the hover opacity for the current button
// and advances the internal counter.
func (h *HoverState) nextButtonHoverOpacity() float32 {
	idx := h.buttonIdx
	h.buttonIdx++
	h.ensureSize(h.buttonIdx)
	return h.anims[idx].Value()
}

func (h *HoverState) ensureSize(n int) {
	for len(h.anims) < n {
		h.anims = append(h.anims, anim.Anim[float32]{})
	}
}

// ── Layout & Scene Building ──────────────────────────────────────
// BuildScene converts an Element tree into draw commands via the
// Canvas interface (RFC §6).

type bounds struct{ X, Y, W, H int }

const (
	framePadding   = 24
	columnGap      = 16
	rowGap         = 12
	buttonPadX     = 18
	buttonPadY     = 12
	buttonMinWidth = 180
	buttonBorder   = 2
)

// BuildScene lays out the element tree and paints it to the canvas.
// It returns the accumulated Scene. If hitMap is non-nil, clickable
// element bounds are registered for hit-testing (M3+).
// If hover is non-nil, hover animations are applied to buttons (M4).
func BuildScene(root Element, canvas draw.Canvas, th theme.Theme, width, height int, hitMap *hit.Map, hover *HoverState) draw.Scene {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}

	if hover != nil {
		hover.resetCounter()
	}

	tokens := th.Tokens()
	area := bounds{X: framePadding, Y: framePadding, W: max(width-(framePadding*2), 0), H: max(height-(framePadding*2), 0)}
	layoutElement(root, area, canvas, tokens, hitMap, hover)

	// The canvas is a SceneCanvas — retrieve its scene.
	type scener interface{ Scene() draw.Scene }
	if sc, ok := canvas.(scener); ok {
		return sc.Scene()
	}
	return draw.Scene{}
}

func layoutElement(el Element, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	switch node := el.(type) {
	case nil, emptyElement:
		return bounds{X: area.X, Y: area.Y}

	case keyedElement:
		return layoutElement(node.Child, area, canvas, tokens, hitMap, hover)

	case textElement:
		style := tokens.Typography.BodyMedium
		metrics := canvas.MeasureText(node.Content, style)
		w := int(metrics.Width)
		h := int(metrics.Ascent)
		canvas.DrawText(node.Content, draw.Pt(float32(area.X), float32(area.Y)), style, tokens.Colors.OnSurface)
		return bounds{X: area.X, Y: area.Y, W: w, H: h}

	case buttonElement:
		style := tokens.Typography.LabelSmall
		metrics := canvas.MeasureText(node.Label, style)
		labelW := int(metrics.Width)
		labelH := int(metrics.Ascent)
		w := max(buttonMinWidth, labelW+(buttonPadX*2))
		h := labelH + (buttonPadY * 2)

		// Edge (border)
		canvas.FillRect(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)),
			draw.SolidPaint(tokens.Colors.Outline))

		// Fill — blend with hover overlay (M4).
		fillColor := tokens.Colors.Primary
		var hoverOpacity float32
		if hover != nil {
			hoverOpacity = hover.nextButtonHoverOpacity()
		}
		if hoverOpacity > 0 {
			fillColor = lerpColor(fillColor, hoverHighlight(fillColor), hoverOpacity)
		}
		canvas.FillRect(draw.R(float32(area.X+buttonBorder), float32(area.Y+buttonBorder),
			float32(max(w-buttonBorder*2, 0)), float32(max(h-buttonBorder*2, 0))),
			draw.SolidPaint(fillColor))

		// Label, centered
		canvas.DrawText(node.Label,
			draw.Pt(float32(area.X+(w-labelW)/2), float32(area.Y+(h-labelH)/2)),
			style, tokens.Colors.OnPrimary)

		// Register hit target for click handling (M3).
		if hitMap != nil {
			hitMap.Add(draw.R(float32(area.X), float32(area.Y), float32(w), float32(h)), node.OnClick)
		}

		return bounds{X: area.X, Y: area.Y, W: w, H: h}

	case boxElement:
		return layoutBox(node, area, canvas, tokens, hitMap, hover)

	default:
		return bounds{X: area.X, Y: area.Y}
	}
}

// hoverHighlight returns a lightened version of c for hover feedback.
func hoverHighlight(c draw.Color) draw.Color {
	return draw.Color{
		R: c.R + (1-c.R)*0.2,
		G: c.G + (1-c.G)*0.2,
		B: c.B + (1-c.B)*0.2,
		A: c.A,
	}
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b draw.Color, t float32) draw.Color {
	return draw.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}

func layoutBox(node boxElement, area bounds, canvas draw.Canvas, tokens theme.TokenSet, hitMap *hit.Map, hover *HoverState) bounds {
	cursorX := area.X
	cursorY := area.Y
	maxW := 0
	maxH := 0
	count := 0

	for _, child := range node.Children {
		childBounds := layoutElement(child, bounds{X: cursorX, Y: cursorY, W: area.W, H: area.H}, canvas, tokens, hitMap, hover)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
