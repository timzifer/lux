# Drag and Drop

```go
import (
    "github.com/timzifer/lux/input"
    "github.com/timzifer/lux/ui"
    "github.com/timzifer/lux/ui/data"
)
```

---

## Overview

The lux drag-and-drop system provides a unified API for mouse, touch, and keyboard-driven
drag operations. Every drag carries a typed `data.DragData` payload that is negotiated between
`DragSource` and `DropTarget` widgets via MIME types. The framework manages the DnD session
lifecycle, visual feedback, and accessibility automatically.

---

## Drag Data

A drag payload is represented by `data.DragData`, which holds one or more `DragItem` entries.
Each item declares a MIME type and a byte payload.

```go
type DragItem struct {
    MIMEType string
    Data     []byte
}

type DragData struct {
    Items []DragItem
}
```

Convenience constructors cover the most common cases:

```go
dd := data.TextDragData("hello world")               // text/plain
dd := data.JSONDragData(myStruct)                     // application/json
dd := data.URIDragData("file:///tmp/photo.png")       // text/uri-list
dd := data.CustomDragData("application/x-my-app", b)  // custom type
```

---

## DragSource

Wrap any element with `ui.DragSource` to make it draggable.

```go
func view(m Model) ui.Element {
    card := ui.Box(ui.Text("Drag me"))

    return ui.DragSource(card,
        ui.DragDataFunc(func() data.DragData {
            return data.TextDragData(m.Label)
        }),
        ui.OnDragStart(func() { fmt.Println("drag started") }),
        ui.OnDragEnd(func(accepted bool) {
            if accepted {
                fmt.Println("drop accepted")
            }
        }),
    )
}
```

The `DragDataFunc` is called lazily when the drag actually begins, so expensive
serialisation only happens when needed.

---

## DropTarget

`ui.DropTarget` declares which MIME types a region accepts and handles the drop.

```go
func view(m Model) ui.Element {
    zone := ui.Box(ui.Text("Drop here"))

    return ui.DropTarget(zone,
        ui.AcceptTypes("text/plain", "application/json"),
        ui.OnDragOver(func(pos ui.Point, dd data.DragData) ui.DropEffect {
            return ui.DropMove
        }),
        ui.OnDrop(func(pos ui.Point, dd data.DragData) bool {
            text := string(dd.Items[0].Data)
            app.Send(ItemDroppedMsg{Text: text})
            return true // accepted
        }),
    )
}
```

Return `false` from `OnDrop` to reject and animate the drag back to its origin.

---

## SortableList

`ui.SortableList` is a higher-level widget for reorderable lists. It manages drag
handles, placeholders, and index calculation internally.

```go
func view(m Model) ui.Element {
    return ui.SortableList(m.Items,
        ui.SortableItemFunc(func(item Item, idx int) ui.Element {
            return ui.Box(ui.Text(item.Name))
        }),
        ui.OnReorder(func(fromIdx, toIdx int) {
            app.Send(ReorderMsg{From: fromIdx, To: toIdx})
        }),
        ui.SortableAxis(ui.Vertical),
    )
}
```

Horizontal sorting is supported via `ui.SortableAxis(ui.Horizontal)`.

---

## DragHandle

By default the entire `DragSource` surface is draggable. Use `ui.DragHandle` to
restrict the grab area to a specific grip element.

```go
row := ui.HStack(
    ui.DragHandle(ui.Icon("grip-vertical")), // only this initiates the drag
    ui.Text(item.Title),
)
return ui.DragSource(row, ...)
```

On touch devices the handle also serves as the long-press target.

---

## Keyboard DnD

All drag operations are accessible via keyboard. When a `DragSource` is focused:

1. Press **Space** or **Enter** to pick up the item.
2. Use **Arrow keys** to move the item between valid drop targets.
3. Press **Enter** to drop, or **Escape** to cancel.

The framework announces state changes via the accessibility tree so screen readers
can narrate the operation.

```go
// Keyboard DnD is enabled by default. To disable on a specific source:
ui.DragSource(el, ui.KeyboardDnD(false))
```

---

## Modifier Keys

Modifier keys change the drop operation while dragging:

| Modifier | Operation | Cursor |
|----------|-----------|--------|
| None | Move | `CursorMove` |
| Ctrl | Copy | `CursorCopy` |
| Shift | Link | `CursorAlias` |
| Ctrl+Shift | Ask (shows menu) | `CursorProgress` |

The `DropTarget` receives the resolved `ui.DropEffect` in `OnDragOver` and can
accept or reject individual operations.

---

## Visual Effects

The framework provides three automatic visual cues during a drag:

- **Ghost preview** -- a semi-transparent snapshot of the dragged element follows the
  cursor. Opacity defaults to 0.7 and is configurable via `ui.GhostOpacity(0.5)`.
- **Placeholder** -- the original position shows a dotted-outline placeholder so the
  user can see where the item came from. Disable with `ui.ShowPlaceholder(false)`.
- **Drop zone highlighting** -- valid `DropTarget` regions receive a highlight border
  when the cursor enters. The highlight colour follows the theme's `AccentColor`.

All effects are rendered in the overlay layer and do not affect layout reflow.

---

## Touch Devices

On touch screens, drag is initiated by a **long-press** (default 300 ms). After
activation the item follows the finger and the standard drop-target logic applies.

```go
// Customise the long-press duration:
ui.DragSource(el, ui.LongPressDuration(400*time.Millisecond))
```

The platform haptic API is called at three points:

1. **Pickup** -- light impact when the long-press threshold is reached.
2. **Enter valid target** -- selection tick when the finger enters a valid drop zone.
3. **Drop** -- medium impact on successful drop.

Haptics are a no-op on platforms that do not support them.
