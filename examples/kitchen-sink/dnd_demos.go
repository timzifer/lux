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
		cards = append(cards, data.DragSource{
			Child: colorCard(color.name, color.color),
			Data: func() *input.DragData {
				return input.NewTextDragData(color.name)
			},
			Operations: input.DragOperationMove,
		})
	}

	source := layout.Column(layout.FlexConfig{Gap: 8}, cards...)

	target := data.DropTarget{
		Child: dropZoneBox("Drop cards here", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEText)
		},
		OnDrop: func(d *input.DragData, pos input.GesturePoint, op input.DragOperation) {
			fmt.Printf("Dropped: %s\n", d.Text())
		},
		Highlight: data.DropHighlightBorder,
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Basic Drag & Drop"),
		display.Text("Drag colored cards to the drop zone."),
		layout.Row(layout.FlexConfig{Gap: 24},
			source,
			target,
		),
	)
}

// ── Copy on Drag ────────────────────────────────────────────────

func dndCopySection(m Model) ui.Element {
	source := data.DragSource{
		Child: colorCard("Copy Me", draw.Hex("#8b5cf6")),
		Data: func() *input.DragData {
			return input.NewTextDragData("Copied Item")
		},
		Operations: input.DragOperationMove | input.DragOperationCopy,
	}

	target := data.DropTarget{
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
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Copy on Drag"),
		display.Text("Hold Ctrl while dragging to copy instead of move."),
		layout.Row(layout.FlexConfig{Gap: 24},
			source,
			target,
		),
	)
}

// ── Sortable List ───────────────────────────────────────────────

func dndSortableSection(m Model) ui.Element {
	items := []string{"Task A", "Task B", "Task C", "Task D", "Task E"}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Sortable List"),
		display.Text("Drag items to reorder."),
		data.SortableList{
			Items:      items,
			ItemHeight: 48,
			MaxHeight:  300,
			State:      data.NewSortableListState(),
			BuildItem: func(key string, index int, dragging bool) ui.Element {
				bg := draw.Hex("#1e293b")
				if dragging {
					bg = draw.Hex("#334155")
				}
				return sortableItemCard(key, bg)
			},
			OnReorder: func(from, to int) {
				fmt.Printf("Reorder: %d -> %d\n", from, to)
			},
			ShowPlaceholder: true,
		},
	)
}

// ── Multiple Drop Zones ─────────────────────────────────────────

func dndMultiZoneSection(m Model) ui.Element {
	source := data.DragSource{
		Child: colorCard("Drag Me", draw.Hex("#f59e0b")),
		Data: func() *input.DragData {
			d := input.NewTextDragData("Item")
			d.Items = append(d.Items, input.DragItem{
				MIMEType: input.MIMEJSON,
				Data:     `{"type":"widget"}`,
			})
			return d
		},
		Operations: input.DragOperationMove | input.DragOperationCopy,
	}

	textZone := data.DropTarget{
		Child: dropZoneBox("Text Only", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEText) && !d.HasType(input.MIMEJSON)
		},
		Highlight: data.DropHighlightBorder,
	}

	jsonZone := data.DropTarget{
		Child: dropZoneBox("JSON Only", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return d.HasType(input.MIMEJSON)
		},
		Highlight: data.DropHighlightFill,
	}

	anyZone := data.DropTarget{
		Child: dropZoneBox("Accepts All", 150),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Multiple Drop Zones"),
		display.Text("Three zones accepting different MIME types. Only matching zones highlight."),
		layout.Row(layout.FlexConfig{Gap: 12},
			source,
		),
		layout.Row(layout.FlexConfig{Gap: 12},
			textZone,
			jsonZone,
			anyZone,
		),
	)
}

// ── Placeholder Drag ────────────────────────────────────────────

func dndPlaceholderSection(m Model) ui.Element {
	source := data.DragSource{
		Child: colorCard("Placeholder Mode", draw.Hex("#06b6d4")),
		Data: func() *input.DragData {
			return input.NewTextDragData("placeholder item")
		},
		Placeholder: true,
	}

	target := data.DropTarget{
		Child: dropZoneBox("Drop here", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Placeholder Drag"),
		display.Text("A dashed placeholder appears at the original position during drag."),
		layout.Row(layout.FlexConfig{Gap: 24},
			source,
			target,
		),
	)
}

// ── Kanban Board ────────────────────────────────────────────────

func dndKanbanSection(m Model) ui.Element {
	columns := []struct {
		title string
		items []string
	}{
		{"To Do", []string{"Design", "Research", "Prototype"}},
		{"In Progress", []string{"Implementation", "Testing"}},
		{"Done", []string{"Documentation"}},
	}

	var cols []ui.Element
	for _, col := range columns {
		colTitle := col.title
		list := data.SortableList{
			Items:      col.items,
			ItemHeight: 40,
			MaxHeight:  250,
			State:      data.NewSortableListState(),
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
			},
		}

		cols = append(cols, layout.Column(layout.FlexConfig{Gap: 8},
			display.Text(colTitle),
			list,
		))
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Kanban Board"),
		display.Text("Drag items between columns and within columns to reorder."),
		layout.Row(layout.FlexConfig{Gap: 16}, cols...),
	)
}

// ── Drag Handle ─────────────────────────────────────────────────

func dndHandleSection(m Model) ui.Element {
	items := []string{"Item with handle 1", "Item with handle 2", "Item with handle 3"}

	var rows []ui.Element
	for _, item := range items {
		itemName := item
		row := data.DragSource{
			Child: layout.Row(layout.FlexConfig{Gap: 8, Align: layout.AlignCenter},
				data.DragHandle{Size: 24},
				display.Text(itemName),
			),
			Data: func() *input.DragData {
				return input.NewTextDragData(itemName)
			},
		}
		rows = append(rows, row)
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Drag Handle"),
		display.Text("Items can only be dragged by the grip handle icon."),
		layout.Column(layout.FlexConfig{Gap: 4}, rows...),
	)
}

// ── Keyboard DnD ────────────────────────────────────────────────

func dndKeyboardSection(m Model) ui.Element {
	source := data.DragSource{
		Child: colorCard("Focus & press Space", draw.Hex("#a855f7")),
		Data: func() *input.DragData {
			return input.NewTextDragData("keyboard drag")
		},
	}

	target := data.DropTarget{
		Child: dropZoneBox("Tab to cycle, Enter to drop", 200),
		Accept: func(d *input.DragData, op input.DragOperation) bool {
			return true
		},
		Highlight: data.DropHighlightBorder,
	}

	return layout.Column(layout.FlexConfig{Gap: 16},
		sectionHeader("Keyboard Drag & Drop"),
		display.Text("Accessible DnD: Focus source → Space to grab → Tab to cycle targets → Enter to drop → Escape to cancel."),
		layout.Row(layout.FlexConfig{Gap: 24},
			source,
			target,
		),
	)
}

// ── Helper Elements ─────────────────────────────────────────────

// colorCard creates a small colored card for drag demos.
func colorCard(label string, bg draw.Color) ui.Element {
	return layout.Padding(
		draw.Insets{Top: 12, Right: 16, Bottom: 12, Left: 16},
		display.Text(label),
	)
}

// dropZoneBox creates a drop zone visual with a dashed border appearance.
func dropZoneBox(label string, height float32) ui.Element {
	return layout.SizedBox(200, height,
		layout.Center(
			display.Text(label),
		),
	)
}

// sortableItemCard creates a card for sortable list items.
func sortableItemCard(label string, bg draw.Color) ui.Element {
	return layout.Padding(
		draw.Insets{Top: 10, Right: 16, Bottom: 10, Left: 16},
		display.Text(label),
	)
}
