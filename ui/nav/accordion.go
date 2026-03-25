package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// AccordionSection defines a collapsible section with header and content.
type AccordionSection struct {
	Header  ui.Element
	Content ui.Element
}

// AccordionState tracks which accordion sections are expanded.
type AccordionState struct {
	Expanded map[int]bool
}

// NewAccordionState creates a ready-to-use AccordionState.
func NewAccordionState() *AccordionState {
	return &AccordionState{Expanded: make(map[int]bool)}
}

// Layout constants matching the core ui package values.
const (
	accordionHeaderH = 36
	cardPadding      = 16
)

// Accordion displays collapsible sections with headers and content.
type Accordion struct {
	ui.BaseElement
	Sections []AccordionSection
	State    *AccordionState
}

// NewAccordion creates an Accordion element.
func NewAccordion(sections []AccordionSection, state *AccordionState) ui.Element {
	return Accordion{Sections: sections, State: state}
}

// LayoutSelf implements ui.Layouter.
func (n Accordion) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if len(n.Sections) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	cursorY := area.Y
	maxW := 0

	chevronStyle := draw.TextStyle{
		Size:   12,
		Weight: draw.FontWeightBold,
	}

	for i, section := range n.Sections {
		expanded := n.State != nil && n.State.Expanded[i]

		// Divider between sections (not before first)
		if i > 0 {
			ctx.Canvas.FillRect(
				draw.R(float32(area.X), float32(cursorY), float32(area.W), 1),
				draw.SolidPaint(ctx.Tokens.Colors.Stroke.Divider))
			cursorY++
		}

		// Register hit target and get hover opacity atomically.
		var hoverOpacity float32
		if n.State != nil {
			idx := i
			state := n.State
			hoverOpacity = ctx.IX.RegisterHit(draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
				func() { state.Expanded[idx] = !state.Expanded[idx] })
		}

		// Header background (with hover blend)
		hdrColor := ctx.Tokens.Colors.Surface.Elevated
		if hoverOpacity > 0 {
			hdrColor = ui.LerpColor(hdrColor, ctx.Tokens.Colors.Surface.Hovered, hoverOpacity)
		}
		ctx.Canvas.FillRect(
			draw.R(float32(area.X), float32(cursorY), float32(area.W), float32(accordionHeaderH)),
			draw.SolidPaint(hdrColor))

		// Chevron indicator
		chevron := "\u25B6" // ▶
		if expanded {
			chevron = "\u25BC" // ▼
		}
		chevronX := area.X + 8
		chevronY := cursorY + (accordionHeaderH-int(chevronStyle.Size))/2
		ctx.Canvas.DrawText(chevron, draw.Pt(float32(chevronX), float32(chevronY)), chevronStyle, ctx.Tokens.Colors.Text.Secondary)

		// Header content
		headerX := area.X + 8 + int(chevronStyle.Size) + 8
		headerArea := ui.Bounds{X: headerX, Y: cursorY + (accordionHeaderH-16)/2, W: max(area.W-headerX+area.X, 0), H: 16}
		ctx.LayoutChild(section.Header, headerArea)

		if area.W > maxW {
			maxW = area.W
		}
		cursorY += accordionHeaderH

		// Content (if expanded)
		if expanded {
			contentArea := ui.Bounds{X: area.X + cardPadding, Y: cursorY + 8, W: max(area.W-cardPadding*2, 0), H: max(area.H-(cursorY-area.Y)-8, 0)}
			cb := ctx.LayoutChild(section.Content, contentArea)
			cursorY += cb.H + 16 // 8 top + 8 bottom padding
		}
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: maxW, H: cursorY - area.Y}
}

// TreeEqual implements ui.TreeEqualizer. Accordion is always unequal (dynamic content).
func (n Accordion) TreeEqual(other ui.Element) bool {
	return false
}

// ResolveChildren implements ui.ChildResolver. Accordion is a leaf in resolution.
func (n Accordion) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker. No-op for Accordion.
func (n Accordion) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {}
