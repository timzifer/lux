# Accessibility

lux builds an **accessibility tree** from the widget tree on every frame and synchronises it
with the platform's native accessibility API. No extra work is required for built-in widgets —
they all expose semantic roles, labels, and states automatically.

```go
import "github.com/timzifer/lux/a11y"
```

---

## Platform Bridges

| Platform | Bridge | Dependency |
|----------|--------|------------|
| Windows | UI Automation (UIA) | `platform/windows` |
| macOS | NSAccessibility | `platform/cocoa` |
| Linux | AT-SPI2 | Pure-Go D-Bus (`godbus/dbus`) |

The Linux AT-SPI2 bridge requires no external C libraries — it communicates with the AT-SPI2
registry daemon entirely over D-Bus using `godbus/dbus`.

The bridge is initialised automatically when the platform backend supports it:

```go
type A11yBridgeProvider interface {
    A11yBridge() a11y.A11yBridge
}
```

If the backend returns a non-nil `A11yBridge`, lux registers and updates the tree on every
frame.

---

## AccessTree

The `AccessTree` is a parallel tree of `AccessNode` values built from the widget tree during
the render phase. Each node carries:

- **Role** — the semantic role of the widget (button, text field, list, etc.)
- **Label** — accessible name (from widget props or aria-label equivalent)
- **Value** — current value (for sliders, progress bars, text fields)
- **State** — checked, selected, focused, disabled, expanded, etc.
- **Bounds** — screen-space bounding rect
- **Children** — child nodes

The `AccessTree` is rebuilt on every frame by `ui.AccessTreeBuilder` and synchronised to the
platform bridge via diffing.

---

## Semantic Roles

Built-in widgets expose the following roles automatically:

| Widget | Role |
|--------|------|
| `button.Text` | Button |
| `button.Icon` | Button |
| `form.TextField` | TextField |
| `form.TextArea` | TextArea |
| `form.Checkbox` | Checkbox |
| `form.Radio` | RadioButton |
| `form.Toggle` | Toggle / Switch |
| `form.Slider` | Slider |
| `form.ProgressBar` | ProgressBar |
| `form.Select` | ComboBox |
| `display.Text` | StaticText |
| `nav.Tabs` | TabGroup / Tab |
| `nav.Accordion` | Tree / TreeItem |
| `data.VirtualList` | List / ListItem |
| `data.Tree` | Tree / TreeItem |
| `menu.MenuBar` | MenuBar / MenuItem |
| `dialog.*` | Dialog |

---

## Focus Management and Screen Readers

Keyboard focus and screen reader focus are kept in sync automatically. When the focused widget
changes (by Tab, arrow keys, or programmatic focus), the accessibility bridge announces the
new widget to the screen reader.

### Focus trap

For modal dialogs, use `ui.WithFocusTrap` to confine Tab navigation to the dialog. The
`aria-hidden` equivalent is set on all content behind the trap, telling screen readers to
ignore it:

```go
content := dialog.Panel(
    display.Text("Are you sure?"),
    layout.Row(
        button.Text("Cancel", onCancel),
        button.Text("Confirm", onConfirm),
    ),
)
// Wrap with focus trap (used internally by dialog widgets)
ui.WithFocusTrap(content)
```

---

## Accessible Labels

Built-in widgets derive their accessible name from visible labels. For icon-only buttons or
widgets without visible text, provide an explicit label:

```go
button.Icon(icons.Trash, func() { app.Send(DeleteMsg{}) },
    button.WithAccessLabel("Delete item"),
)
```

---

## `A11yBridge` Interface

Implement this interface to create a custom platform bridge:

```go
type A11yBridge interface {
    // Update synchronises the bridge with the new AccessTree.
    Update(tree *AccessTree)

    // Announce reads out a message to the screen reader (live region).
    Announce(text string, politeness Politeness)

    // Destroy releases all bridge resources.
    Destroy()
}
```

`Politeness` is either `PolitenessPolite` (waits for the user to be idle) or
`PolitenessAssertive` (interrupts immediately).
