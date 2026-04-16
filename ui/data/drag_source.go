// Package data — drag_source.go implements the DragSource wrapper element
// that makes any child element draggable (RFC-005 §6).
//
// DragSource is a stateful Widget: it processes DragMsg events from
// RenderCtx.Events (touch) and registers draggable hit targets via the
// Interactor (mouse) to initiate drag-and-drop sessions.
package data

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// DragSource wraps a child element and makes it draggable.
// When the user starts a drag gesture on the child, a drag-and-drop
// session is initiated with the provided DragData.
type DragSource struct {
	ui.BaseElement

	// Child is the element to render (and to drag).
	Child ui.Element

	// Data provides the drag payload. Called lazily when the drag begins.
	Data func() *input.DragData

	// Operations declares which operations this source supports.
	// Default (zero): DragOperationMove.
	Operations input.DragOperation

	// Preview builds a custom drag preview element.
	// If nil, the framework renders a default ghost rectangle.
	Preview func() ui.Element

	// Placeholder controls whether a visual placeholder is shown at
	// the original position while the item is being dragged.
	Placeholder bool

	// HandleOnly restricts drag initiation to DragHandle children.
	// When true, the full element is not draggable — only the handle region.
	HandleOnly bool

	// OnDragStart is called when the drag begins (optional).
	OnDragStart func()

	// OnDragEnd is called when the drag completes with the result effect (optional).
	OnDragEnd func(input.DropEffect)
}

// mouseDragThreshold is the minimum mouse movement in dp before a
// mouse press becomes a drag gesture (matches GestureConfig default).
const mouseDragThreshold float32 = 10

// dragSourceState is the per-instance state for DragSource.
type dragSourceState struct {
	dragging bool
	bounds   draw.Rect
}

// Render implements ui.Widget.
func (ds DragSource) Render(ctx ui.RenderCtx, raw ui.WidgetState) (ui.Element, ui.WidgetState) {
	s := ui.AdoptState[dragSourceState](raw)

	// Handle touch-based drag events from the gesture recognizer.
	for _, ev := range ctx.Events {
		if ev.Kind == ui.EventDrag && ev.Drag != nil {
			switch ev.Drag.Phase {
			case input.DragBegan:
				if ds.Data == nil {
					continue
				}
				data := ds.Data()
				if data == nil {
					continue
				}
				ops := ds.Operations
				if ops == 0 {
					ops = input.DragOperationMove
				}
				data.AllowedOps = ops

				var preview ui.Element
				if ds.Preview != nil {
					preview = ds.Preview()
				}

				// Calculate offset so the preview stays aligned with where
				// the user grabbed the element (no jump).
				offset := draw.Point{
					X: s.bounds.X - ev.Drag.Start.X,
					Y: s.bounds.Y - ev.Drag.Start.Y,
				}

				ctx.Send(ui.StartDragSessionMsg{
					SourceUID:       ctx.UID,
					Data:            data,
					StartPos:        ev.Drag.Start,
					SourceBounds:    s.bounds,
					Preview:         preview,
					PreviewOffset:   offset,
					ShowPlaceholder: ds.Placeholder,
				})
				s.dragging = true

				if ds.OnDragStart != nil {
					ds.OnDragStart()
				}

			case input.DragEnded:
				if s.dragging {
					s.dragging = false
					if ds.OnDragEnd != nil {
						ds.OnDragEnd(input.DropEffectMove) // TODO: get actual effect
					}
				}

			case input.DragCancelled:
				if s.dragging {
					s.dragging = false
					if ds.OnDragEnd != nil {
						ds.OnDragEnd(input.DropEffectNone)
					}
				}
			}
		}
	}

	// When dragging with placeholder mode, show a dimmed version.
	if s.dragging && ds.Placeholder {
		return dragPlaceholder{Child: ds.Child, Bounds: &s.bounds}, s
	}

	// Pass the drag configuration and Send function to the layout element
	// so it can register a hit target for mouse-based drag (desktop).
	return dragSourceLayout{
		Child:  ds.Child,
		Bounds: &s.bounds,
		dragConfig: dragConfig{
			Send:        ctx.Send,
			WidgetUID:   ctx.UID,
			DataFn:      ds.Data,
			Operations:  ds.Operations,
			PreviewFn:   ds.Preview,
			Placeholder: ds.Placeholder,
			HandleOnly:  ds.HandleOnly,
			OnDragStart: ds.OnDragStart,
		},
	}, s
}

// AccessibleWidget implements ui.AccessibleWidget for screen reader support.
func (ds DragSource) AccessNode(state ui.WidgetState) a11y.AccessNode {
	s, _ := state.(*dragSourceState)
	grabbed := s != nil && s.dragging
	return a11y.AccessNode{
		Role:   a11y.RoleGroup,
		Label:  "draggable",
		States: a11y.AccessStates{Grabbed: grabbed},
		Actions: []a11y.AccessAction{
			{Name: "drag"},
		},
	}
}

// dragConfig holds drag-and-drop parameters passed from DragSource.Render()
// to dragSourceLayout.LayoutSelf() for hit target registration.
type dragConfig struct {
	Send        func(any)
	WidgetUID   ui.UID
	DataFn      func() *input.DragData
	Operations  input.DragOperation
	PreviewFn   func() ui.Element
	Placeholder bool
	HandleOnly  bool
	OnDragStart func()
}

// activeDragHandleConfig is a package-level pointer used to pass drag
// configuration from a HandleOnly DragSource to its DragHandle child
// during the synchronous LayoutSelf pass. It is set before the child
// layout and cleared immediately after.
var activeDragHandleConfig *dragConfig

// dragSourceLayout is a layout wrapper that records its bounds for the
// DragSource state and registers a draggable hit target for mouse interaction.
type dragSourceLayout struct {
	ui.BaseElement
	Child  ui.Element
	Bounds *draw.Rect
	dragConfig
}

// LayoutSelf implements ui.Layouter.
func (d dragSourceLayout) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area

	// When HandleOnly, expose the drag config to DragHandle children
	// during the synchronous LayoutChild call.
	if d.HandleOnly {
		cfg := d.dragConfig
		activeDragHandleConfig = &cfg
	}
	childBounds := ctx.LayoutChild(d.Child, area)
	if d.HandleOnly {
		activeDragHandleConfig = nil
	}

	rect := draw.R(float32(area.X), float32(area.Y),
		float32(childBounds.W), float32(childBounds.H))
	if d.Bounds != nil {
		*d.Bounds = rect
	}

	// Register a draggable hit target for mouse-based drag (desktop).
	// Touch-based drag is handled separately via the gesture recognizer
	// and EventDrag in DragSource.Render().
	// When HandleOnly, the DragHandle child registers itself instead.
	if ctx.IX != nil && ctx.IX.DnD != nil && d.DataFn != nil && !d.HandleOnly {
		cfg := d.dragConfig
		sourceBounds := rect
		dnd := ctx.IX.DnD

		var startX, startY float32
		var pressing bool
		var dragSent bool

		ctx.IX.RegisterSurfaceDrag(rect,
			// OnClickAt: fires on press and on every move while held.
			func(x, y float32) {
				if dragSent {
					return
				}
				if !pressing {
					startX, startY = x, y
					pressing = true
					return
				}
				dx := x - startX
				dy := y - startY
				if dx*dx+dy*dy > mouseDragThreshold*mouseDragThreshold {
					data := cfg.DataFn()
					if data == nil {
						return
					}
					ops := cfg.Operations
					if ops == 0 {
						ops = input.DragOperationMove
					}
					data.AllowedOps = ops

					var preview ui.Element
					if cfg.PreviewFn != nil {
						preview = cfg.PreviewFn()
					}

					// Calculate offset so the preview stays where the
					// user grabbed the element (no jump).
					offset := draw.Point{
						X: sourceBounds.X - startX,
						Y: sourceBounds.Y - startY,
					}

					// Start DnD session directly — bypasses the message
					// queue to avoid timing issues with platform callbacks.
					dnd.StartDrag(
						cfg.WidgetUID, data,
						input.GesturePoint{X: startX, Y: startY},
						sourceBounds, preview,
						offset,
						cfg.Placeholder,
					)

					if cfg.OnDragStart != nil {
						cfg.OnDragStart()
					}

					dragSent = true
				}
			},
			// OnRelease: fires once on mouse release.
			func(x, y float32) {
				pressing = false
				dragSent = false
			},
		)
	}

	return childBounds
}

// dragPlaceholder renders a dimmed/dashed version of the child to show
// where the dragged item came from.
type dragPlaceholder struct {
	ui.BaseElement
	Child  ui.Element
	Bounds *draw.Rect
}

// LayoutSelf implements ui.Layouter.
func (d dragPlaceholder) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	if d.Bounds != nil {
		*d.Bounds = draw.R(float32(area.X), float32(area.Y),
			float32(area.W), float32(area.H))
	}

	// Draw a dashed border placeholder.
	tokens := ctx.Tokens
	placeholderColor := tokens.Colors.Stroke.Border
	placeholderColor.A = 0.3
	ctx.Canvas.StrokeRoundRect(
		draw.R(float32(area.X), float32(area.Y), float32(area.W), float32(area.H)),
		tokens.Radii.Card,
		draw.Stroke{
			Paint:      draw.SolidPaint(placeholderColor),
			Width:      2.0,
			Dash:       []float32{6, 4}, // dashed
			DashOffset: 0,
		},
	)

	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: area.H}
}
