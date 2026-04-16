// Package data — drop_target.go implements the DropTarget wrapper element
// that makes an area accept drag-and-drop operations (RFC-005 §7).
package data

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// DropHighlightStyle determines the visual feedback when a drag hovers
// over an accepting drop target.
type DropHighlightStyle uint8

const (
	// DropHighlightBorder draws an accent-colored border around the zone.
	DropHighlightBorder DropHighlightStyle = iota
	// DropHighlightFill draws a semi-transparent accent fill.
	DropHighlightFill
	// DropHighlightInsert draws an insertion line at the drop position.
	DropHighlightInsert
	// DropHighlightNone disables automatic highlighting (for custom handling).
	DropHighlightNone
)

// DropTarget wraps a child element and makes it a drop zone for
// drag-and-drop operations. When a drag hovers over this element,
// visual feedback is shown and DnD events are dispatched.
type DropTarget struct {
	ui.BaseElement

	// Child is the element to render inside the drop zone.
	Child ui.Element

	// Accept tests whether this target accepts the dragged data with
	// the given operation. Return true to accept, false to reject.
	Accept func(data *input.DragData, op input.DragOperation) bool

	// OnDrop is called when data is dropped on this target.
	OnDrop func(data *input.DragData, pos input.GesturePoint, op input.DragOperation)

	// OnEnter is called when a drag enters this target (optional).
	OnEnter func(data *input.DragData)

	// OnLeave is called when a drag leaves this target (optional).
	OnLeave func()

	// Highlight determines the visual feedback style.
	// Default (zero): DropHighlightBorder.
	Highlight DropHighlightStyle

	// ID is an optional stable identifier for this drop zone.
	ID string

	// Priority for nested drop targets. Higher values win.
	Priority int
}

// dropTargetState is the per-instance state for DropTarget.
type dropTargetState struct {
	isHovered bool // a drag is hovering over this target
	accepts   bool // the hovered data is acceptable
}

// Render implements ui.Widget.
func (dt DropTarget) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
	s := ui.AdoptState[dropTargetState](raw)

	for _, ev := range ctx.Events {
		switch ev.Kind {
		case ui.EventDragEnter:
			s.isHovered = true
			if ev.DnDEnter != nil && dt.Accept != nil {
				s.accepts = dt.Accept(ev.DnDEnter.Data, ev.DnDEnter.Operation)
			}
			if dt.OnEnter != nil && ev.DnDEnter != nil {
				dt.OnEnter(ev.DnDEnter.Data)
			}
		case ui.EventDragLeave:
			s.isHovered = false
			s.accepts = false
			if dt.OnLeave != nil {
				dt.OnLeave()
			}
		case ui.EventDragOver:
			// Continuously re-evaluate acceptance (modifier keys may change).
			if ev.DnDOver != nil && dt.Accept != nil {
				s.accepts = dt.Accept(ev.DnDOver.Data, ev.DnDOver.Operation)
			}
		case ui.EventDrop:
			s.isHovered = false
			s.accepts = false
			if ev.DnDDrop != nil && dt.OnDrop != nil {
				op := input.DragOperationMove
				switch ev.DnDDrop.Effect {
				case input.DropEffectCopy:
					op = input.DragOperationCopy
				case input.DropEffectLink:
					op = input.DragOperationLink
				}
				dt.OnDrop(ev.DnDDrop.Data, ev.DnDDrop.Pos, op)
			}
		}
	}

	return dropTargetLayout{
		Child:     dt.Child,
		Accept:    dt.Accept,
		Highlight: dt.Highlight,
		Priority:  dt.Priority,
		State:     s,
		WidgetUID: ctx.UID,
	}, s
}

// AccessNode implements accessibility for drop targets.
func (dt DropTarget) AccessNode(state ui.WidgetState) a11y.AccessNode {
	s, _ := state.(*dropTargetState)
	return a11y.AccessNode{
		Role:   a11y.RoleGroup,
		Label:  "drop target",
		States: a11y.AccessStates{DropTarget: true, Grabbed: s != nil && s.isHovered},
	}
}

// dropTargetLayout is the layout wrapper for DropTarget.
type dropTargetLayout struct {
	ui.BaseElement
	Child     ui.Element
	Accept    func(*input.DragData, input.DragOperation) bool
	Highlight DropHighlightStyle
	Priority  int
	State     *dropTargetState
	WidgetUID ui.UID
}

// LayoutSelf implements ui.Layouter.
func (d dropTargetLayout) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	// Layout child first to get actual bounds.
	childBounds := ctx.LayoutChild(d.Child, area)
	zoneBounds := draw.R(float32(childBounds.X), float32(childBounds.Y),
		float32(childBounds.W), float32(childBounds.H))

	// Register as a drop zone using the child's actual bounds.
	if ctx.IX != nil {
		ctx.IX.RegisterDropZone(zoneBounds, d.WidgetUID, d.Accept, d.Priority)
	}

	// Draw highlight when hovered and accepting.
	if d.State != nil && d.State.isHovered && d.State.accepts {
		tokens := ctx.Tokens

		switch d.Highlight {
		case DropHighlightBorder:
			accentColor := tokens.Colors.Accent.Primary
			ctx.Canvas.StrokeRoundRect(zoneBounds, tokens.Radii.Card, draw.Stroke{
				Paint: draw.SolidPaint(accentColor),
				Width: 2.0,
			})
			bgColor := accentColor
			bgColor.A = 0.08
			ctx.Canvas.FillRoundRect(zoneBounds, tokens.Radii.Card, draw.SolidPaint(bgColor))

		case DropHighlightFill:
			accentColor := tokens.Colors.Accent.Primary
			accentColor.A = 0.15
			ctx.Canvas.FillRoundRect(zoneBounds, tokens.Radii.Card, draw.SolidPaint(accentColor))

		case DropHighlightInsert:
			// Insertion line at top of the zone.
			accentColor := tokens.Colors.Accent.Primary
			lineY := float32(childBounds.Y)
			ctx.Canvas.FillRect(draw.R(float32(childBounds.X), lineY, float32(childBounds.W), 3), draw.SolidPaint(accentColor))

		case DropHighlightNone:
			// No automatic highlighting.
		}
	}

	return childBounds
}
