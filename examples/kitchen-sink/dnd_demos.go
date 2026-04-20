// Kitchen Sink — Drag & Drop demo sections (RFC-005).
package main

import (
	"fmt"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/data"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

// Persistent sortable list state (survives across frames).
var (
	sortableItems = []string{"Task A", "Task B", "Task C", "Task D", "Task E"}
	sortableState = data.NewSortableListState()

	kanbanTodo       = []string{"Design", "Research", "Prototype"}
	kanbanInProgress = []string{"Implementation", "Testing"}
	kanbanDone       = []string{"Documentation"}
	kanbanStates     = [3]*data.SortableListState{
		data.NewSortableListState(),
		data.NewSortableListState(),
		data.NewSortableListState(),
	}
)

// ── Basic DnD ───────────────────────────────────────────────────

func dndBasicSection(m Model) ui.Element {
	colors := []struct {
		name  string
		color draw.Color
	}{
		{"Red", draw.Hex("#ef4444")},
		{"Green", draw.Hex("#22c55e")},
		{"Blue", draw.Hex("#3b82f6")},
	}

	var cards []ui.Element
	for _, c := range colors {
		color := c
		cards = append(cards, ui.Component(data.DragSource{
			Child: colorCardWithBG(color.name, color.color),
			Data: func() *input.DragData {
				return input.NewTextDragData(color.name)
			},
			Operations: input.DragOperationMove,
			Preview:    func() ui.Element { return colorCardWithBG(color.name, color.color) },
		}))
	}

	source := layout.NewFlex(cards, layout.WithDirection(layout.FlexColumn), layout.WithGap(8))

	target := ui.Component(data.DropTarget{
		Child: dropZoneBox("Drop cards here", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEText)
		},
		OnDrop: func(d *input.DragData, pos input.GesturePoint, op input.DragOperation) {
			fmt.Printf("Dropped: %s\n", d.Text())
		},
		Highlight: data.DropHighlightBorder,
	})

	return layout.NewFlex([]ui.Element{
		sectionHeader("Basic Drag & Drop"),
		display.Text("Drag colored cards to the drop zone."),
		layout.NewFlex([]ui.Element{source, target}, layout.WithGap(24)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Copy on Drag ────────────────────────────────────────────────

func dndCopySection(m Model) ui.Element {
	source := ui.Component(data.DragSource{
		Child: colorCardWithBG("Copy Me", draw.Hex("#8b5cf6")),
		Data: func() *input.DragData {
			return input.NewTextDragData("Copied Item")
		},
		Operations: input.DragOperationMove | input.DragOperationCopy,
		Preview:    func() ui.Element { return colorCardWithBG("Copy Me", draw.Hex("#8b5cf6")) },
	})

	target := ui.Component(data.DropTarget{
		Child: dropZoneBox("Drop here (Ctrl = Copy)", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		OnDrop: func(d *input.DragData, pos input.GesturePoint, op input.DragOperation) {
			opName := "moved"
			if op == input.DragOperationCopy {
				opName = "copied"
			}
			fmt.Printf("Item %s: %s\n", opName, d.Text())
		},
		Highlight: data.DropHighlightFill,
	})

	return layout.NewFlex([]ui.Element{
		sectionHeader("Copy on Drag"),
		display.Text("Hold Ctrl while dragging to copy instead of move."),
		layout.NewFlex([]ui.Element{source, target}, layout.WithGap(24)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Sortable List ───────────────────────────────────────────────

func dndSortableSection(m Model) ui.Element {
	return layout.NewFlex([]ui.Element{
		sectionHeader("Sortable List"),
		display.Text("Drag items to reorder."),
		data.SortableList{
			Items:      sortableItems,
			ItemHeight: 48,
			MaxHeight:  300,
			State:      sortableState,
			BuildItem: func(key string, index int, dragging bool) ui.Element {
				bg := draw.Hex("#1e293b")
				if dragging {
					bg = draw.Hex("#334155")
				}
				return sortableItemCard(key, bg)
			},
			OnReorder: func(from, to int) {
				fmt.Printf("Reorder: %d -> %d\n", from, to)
				reorderSlice(&sortableItems, from, to)
			},
			ShowPlaceholder: true,
		},
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Multiple Drop Zones ─────────────────────────────────────────

func dndMultiZoneSection(m Model) ui.Element {
	source := ui.Component(data.DragSource{
		Child:   colorCardWithBG("Drag Me", draw.Hex("#f59e0b")),
		Preview: func() ui.Element { return colorCardWithBG("Drag Me", draw.Hex("#f59e0b")) },
		Data: func() *input.DragData {
			d := input.NewTextDragData("Item")
			d.Items = append(d.Items, input.DragItem{
				MIMEType: input.MIMEJSON,
				Data:     `{"type":"widget"}`,
			})
			return d
		},
		Operations: input.DragOperationMove | input.DragOperationCopy,
	})

	textZone := ui.Component(data.DropTarget{
		Child: dropZoneBox("Text Only", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEText) && !d.HasType(input.MIMEJSON)
		},
		Highlight: data.DropHighlightBorder,
	})

	jsonZone := ui.Component(data.DropTarget{
		Child: dropZoneBox("JSON Only", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEJSON)
		},
		Highlight: data.DropHighlightFill,
	})

	anyZone := ui.Component(data.DropTarget{
		Child: dropZoneBox("Accepts All", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	})

	return layout.NewFlex([]ui.Element{
		sectionHeader("Multiple Drop Zones"),
		display.Text("Three zones accepting different MIME types. Only matching zones highlight."),
		layout.NewFlex([]ui.Element{source}, layout.WithGap(12)),
		layout.NewFlex([]ui.Element{textZone, jsonZone, anyZone}, layout.WithGap(12)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Placeholder Drag ────────────────────────────────────────────

func dndPlaceholderSection(m Model) ui.Element {
	source := ui.Component(data.DragSource{
		Child: colorCardWithBG("Placeholder Mode", draw.Hex("#ec4899")),
		Data: func() *input.DragData {
			return input.NewTextDragData("placeholder item")
		},
		Placeholder: true,
		Preview:     func() ui.Element { return colorCardWithBG("Placeholder Mode", draw.Hex("#ec4899")) },
	})

	target := ui.Component(data.DropTarget{
		Child: dropZoneBox("Drop here", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	})

	return layout.NewFlex([]ui.Element{
		sectionHeader("Placeholder Drag"),
		display.Text("A dashed placeholder appears at the original position during drag."),
		layout.NewFlex([]ui.Element{source, target}, layout.WithGap(24)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Kanban Board ────────────────────────────────────────────────

func dndKanbanSection(m Model) ui.Element {
	type column struct {
		title string
		items *[]string
		state *data.SortableListState
	}
	columns := []column{
		{"To Do", &kanbanTodo, kanbanStates[0]},
		{"In Progress", &kanbanInProgress, kanbanStates[1]},
		{"Done", &kanbanDone, kanbanStates[2]},
	}

	var cols []ui.Element
	for _, col := range columns {
		colTitle := col.title
		colItems := col.items
		list := data.SortableList{
			Items:      *col.items,
			ItemHeight: 40,
			MaxHeight:  250,
			State:      col.state,
			GroupID:    "kanban",
			BuildItem: func(key string, index int, dragging bool) ui.Element {
				bg := draw.Hex("#1e293b")
				if dragging {
					bg = draw.Hex("#334155")
				}
				return sortableItemCard(key, bg)
			},
			OnReorder: func(from, to int) {
				fmt.Printf("[%s] Reorder: %d -> %d\n", colTitle, from, to)
				reorderSlice(colItems, from, to)
			},
			OnInsert: func(index int, d *input.DragData) {
				key := ""
				if v, ok := d.Get(input.MIMESortableKey); ok {
					key, _ = v.(string)
				}
				fmt.Printf("[%s] Insert: %q at %d\n", colTitle, key, index)
				// Remove from source column first.
				for _, src := range []*[]string{&kanbanTodo, &kanbanInProgress, &kanbanDone} {
					if src == colItems {
						continue
					}
					for j, k := range *src {
						if k == key {
							*src = append((*src)[:j], (*src)[j+1:]...)
							goto removed
						}
					}
				}
			removed:
				// Insert into this column.
				s := *colItems
				s = append(s, "")
				copy(s[index+1:], s[index:])
				s[index] = key
				*colItems = s
			},
		}

		cols = append(cols, layout.NewFlex([]ui.Element{
			display.Text(colTitle),
			list,
		}, layout.WithDirection(layout.FlexColumn), layout.WithGap(8)))
	}

	return layout.NewFlex([]ui.Element{
		sectionHeader("Kanban Board"),
		display.Text("Drag items between columns and within columns to reorder."),
		layout.NewFlex(cols, layout.WithGap(16)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Drag Handle ─────────────────────────────────────────────────

func dndHandleSection(m Model) ui.Element {
	items := []string{"Item with handle 1", "Item with handle 2", "Item with handle 3"}

	var rows []ui.Element
	for _, item := range items {
		itemName := item
		row := ui.Component(data.DragSource{
			Child: layout.NewFlex([]ui.Element{
				data.DragHandle{Size: 24},
				display.Text(itemName),
			}, layout.WithGap(8), layout.WithAlign(layout.AlignCenter)),
			Data: func() *input.DragData {
				return input.NewTextDragData(itemName)
			},
			Preview:    func() ui.Element { return colorCard(itemName) },
			HandleOnly: true,
		})
		rows = append(rows, row)
	}

	return layout.NewFlex([]ui.Element{
		sectionHeader("Drag Handle"),
		display.Text("Items can only be dragged by the grip handle icon."),
		layout.NewFlex(rows, layout.WithDirection(layout.FlexColumn), layout.WithGap(4)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// ── Keyboard DnD ────────────────────────────────────────────────

func dndKeyboardSection(m Model) ui.Element {
	source := ui.Component(data.DragSource{
		Child:   colorCardWithBG("Focus & press Space", draw.Hex("#06b6d4")),
		Preview: func() ui.Element { return colorCardWithBG("Focus & press Space", draw.Hex("#06b6d4")) },
		Data: func() *input.DragData {
			return input.NewTextDragData("keyboard drag")
		},
	})

	target := ui.Component(data.DropTarget{
		Child: dropZoneBox("Tab to cycle, Enter to drop", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	})

	return layout.NewFlex([]ui.Element{
		sectionHeader("Keyboard Drag & Drop"),
		display.Text("Accessible DnD: Focus source → Space to grab → Tab to cycle targets → Enter to drop → Escape to cancel."),
		layout.NewFlex([]ui.Element{source, target}, layout.WithGap(24)),
	}, layout.WithDirection(layout.FlexColumn), layout.WithGap(16))
}

// reorderSlice moves the element at index from to index to, shifting
// intermediate elements. Operates in-place on the underlying array.
func reorderSlice(items *[]string, from, to int) {
	s := *items
	if from < 0 || from >= len(s) || to < 0 || to > len(s) || from == to {
		return
	}
	item := s[from]
	// Remove from original position.
	copy(s[from:], s[from+1:])
	s = s[:len(s)-1]
	// Adjust insertion index after removal.
	if to > from {
		to--
	}
	// Insert at new position.
	s = append(s, "")
	copy(s[to+1:], s[to:])
	s[to] = item
	*items = s
}

// ── Helper Elements ─────────────────────────────────────────────

// colorCardElement draws a rounded rectangle background with centered text.
type colorCardElement struct {
	ui.BaseElement
	Label string
	BG    draw.Color
}

func colorCard(label string) ui.Element {
	return colorCardElement{Label: label, BG: draw.Hex("#334155")}
}

func colorCardWithBG(label string, bg draw.Color) ui.Element {
	return colorCardElement{Label: label, BG: bg}
}

func (c colorCardElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	padX, padY := 16, 12
	style := ctx.Tokens.Typography.Label
	metrics := ctx.Canvas.MeasureText(c.Label, style)
	w := int(metrics.Width) + padX*2
	h := int(metrics.Ascent+metrics.Descent) + padY*2

	rect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))
	ctx.Canvas.FillRoundRect(rect, ctx.Tokens.Radii.Card, draw.SolidPaint(c.BG))

	ctx.Canvas.DrawText(c.Label,
		draw.Pt(float32(ctx.Area.X+padX), float32(ctx.Area.Y+padY)),
		style, ctx.Tokens.Colors.Text.Primary)

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}

// dropZoneElement draws a dashed border rectangle with centered text.
type dropZoneElement struct {
	ui.BaseElement
	Label  string
	Width  float32
	Height float32
}

func dropZoneBox(label string, height float32) ui.Element {
	return dropZoneElement{Label: label, Width: 200, Height: height}
}

func (d dropZoneElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	w, h := int(d.Width), int(d.Height)
	rect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))

	// Dashed border.
	borderColor := ctx.Tokens.Colors.Stroke.Border
	ctx.Canvas.StrokeRoundRect(rect, ctx.Tokens.Radii.Card, draw.Stroke{
		Paint: draw.SolidPaint(borderColor),
		Width: 2,
		Dash:  []float32{6, 4},
	})

	// Centered label.
	style := ctx.Tokens.Typography.Label
	metrics := ctx.Canvas.MeasureText(d.Label, style)
	tx := float32(ctx.Area.X) + (d.Width-metrics.Width)/2
	ty := float32(ctx.Area.Y) + (d.Height-metrics.Ascent)/2
	ctx.Canvas.DrawText(d.Label, draw.Pt(tx, ty), style, ctx.Tokens.Colors.Text.Secondary)

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}

// sortableItemElement draws a card for sortable list items with a background.
type sortableItemElement struct {
	ui.BaseElement
	Label string
	BG    draw.Color
}

func sortableItemCard(label string, bg draw.Color) ui.Element {
	return sortableItemElement{Label: label, BG: bg}
}

func (s sortableItemElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	padX, padY := 16, 10
	style := ctx.Tokens.Typography.Label
	metrics := ctx.Canvas.MeasureText(s.Label, style)
	w := ctx.Area.W
	h := int(metrics.Ascent+metrics.Descent) + padY*2

	rect := draw.R(float32(ctx.Area.X), float32(ctx.Area.Y), float32(w), float32(h))
	ctx.Canvas.FillRoundRect(rect, ctx.Tokens.Radii.Card, draw.SolidPaint(s.BG))

	ctx.Canvas.DrawText(s.Label,
		draw.Pt(float32(ctx.Area.X+padX), float32(ctx.Area.Y+padY)),
		style, ctx.Tokens.Colors.Text.Primary)

	return ui.Bounds{X: ctx.Area.X, Y: ctx.Area.Y, W: w, H: h}
}
