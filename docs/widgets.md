# Widget Catalogue

All widgets live under `github.com/timzifer/lux/ui/`. Import the sub-package for the widget
family you need.

---

## Buttons — `ui/button`

```go
import "github.com/timzifer/lux/ui/button"
```

| Constructor | Description |
|-------------|-------------|
| `button.Text(label string, onClick func()) ui.Element` | Standard text button |
| `button.Icon(icon ui.Element, onClick func()) ui.Element` | Icon-only button |
| `button.Segmented(items []button.SegmentedItem) ui.Element` | Mutually exclusive segment group |
| `button.Split(label string, onClick func(), menuItems []menu.Item) ui.Element` | Primary action + dropdown arrow |

Buttons participate in the focus system and support keyboard activation (Space/Enter).

---

## Form Controls — `ui/form`

```go
import "github.com/timzifer/lux/ui/form"
```

| Constructor | Description |
|-------------|-------------|
| `form.TextField(opts ...form.TextFieldOpt) ui.Element` | Single-line text input |
| `form.PasswordField(opts ...form.PasswordFieldOpt) ui.Element` | Text field with masked input |
| `form.TextArea(opts ...form.TextAreaOpt) ui.Element` | Multi-line text input |
| `form.Checkbox(label string, checked bool, onChange func(bool)) ui.Element` | Checkbox with label |
| `form.Radio(label string, selected bool, onChange func()) ui.Element` | Radio button |
| `form.Toggle(checked bool, onChange func(bool)) ui.Element` | On/off toggle switch |
| `form.Select(opts ...form.SelectOpt) ui.Element` | Dropdown selection |
| `form.Slider(value, min, max float32, onChange func(float32)) ui.Element` | Range slider |
| `form.ProgressBar(value float32) ui.Element` | Progress indicator (0–1) |

### TextField options

```go
form.TextField(
    form.WithValue(m.Text),
    form.WithPlaceholder("Enter text…"),
    form.WithOnChange(func(s string) { app.Send(TextChangedMsg{Text: s}) }),
    form.WithFocus(app.Focus()),
    form.WithLabel("Name"),
    form.WithHint("Your full name"),
    form.WithDisabled(false),
)
```

---

## Display — `ui/display`

```go
import "github.com/timzifer/lux/ui/display"
```

| Constructor | Description |
|-------------|-------------|
| `display.Text(content string, opts ...display.TextOpt) ui.Element` | Styled text label |
| `display.Card(children ...ui.Element) ui.Element` | Elevated card container |
| `display.Badge(label string) ui.Element` | Small status badge |
| `display.Chip(label string, opts ...display.ChipOpt) ui.Element` | Dismissible or selectable chip |
| `display.Tooltip(content ui.Element, trigger ui.Element) ui.Element` | Tooltip on hover |
| `display.Divider() ui.Element` | Horizontal divider line |
| `display.Spacer() ui.Element` | Flexible space filler |

### Text options

```go
display.Text("Hello",
    display.WithStyle(tokens.Typography.H1),
    display.WithColor(tokens.Colors.Text.Secondary),
    display.WithAlign(draw.AlignCenter),
)
```

---

## Layout — `ui/layout`

See the dedicated [Layout](layout.md) document.

```go
import "github.com/timzifer/lux/ui/layout"

layout.Row(children...)            // horizontal flex
layout.Column(children...)         // vertical flex
layout.Flex(opts, children...)     // full flexbox control
layout.Grid(opts, children...)     // CSS Grid
layout.Table(rows)                 // table layout
layout.Stack(children...)          // z-axis stack
layout.Box(opts, child)            // padding / margin / size constraints
```

---

## Navigation — `ui/nav`

```go
import "github.com/timzifer/lux/ui/nav"
```

| Constructor | Description |
|-------------|-------------|
| `nav.Tabs(items []nav.TabItem, selected int, onSelect func(int)) ui.Element` | Horizontal tab bar |
| `nav.Accordion(items []nav.AccordionItem) ui.Element` | Collapsible sections |
| `nav.SplitView(opts nav.SplitViewOpts) ui.Element` | Resizable pane divider |

---

## Data — `ui/data`

```go
import "github.com/timzifer/lux/ui/data"
```

| Constructor | Description |
|-------------|-------------|
| `data.VirtualList(opts data.VirtualListOpts) ui.Element` | Virtualised list (only renders visible rows) |
| `data.Tree(opts data.TreeOpts) ui.Element` | Hierarchical tree view |

`VirtualList` is the correct choice for lists with hundreds or thousands of items. It renders
only the visible window of rows and recycles widgets as the user scrolls.

---

## Menus — `ui/menu`

```go
import "github.com/timzifer/lux/ui/menu"
```

| Constructor | Description |
|-------------|-------------|
| `menu.MenuBar(items []menu.MenuItem) ui.Element` | Application menu bar |
| `menu.ContextMenu(items []menu.MenuItem, trigger ui.Element) ui.Element` | Right-click context menu |

---

## Dialogs — `dialog`

```go
import "github.com/timzifer/lux/dialog"
```

The `dialog` package provides modal dialogs that integrate with the native platform dialog
APIs where available, falling back to in-process modal dialogs.

---

## Effects — `ui/effects`

```go
import "github.com/timzifer/lux/ui/effects"
```

| Constructor | Description |
|-------------|-------------|
| `effects.Overlay(opts effects.OverlayOpts) ui.Element` | Anchored overlay (tooltips, dropdowns, popovers) with placement and animation |

`Overlay` supports configurable anchor edges, auto-flip when near screen edges, enter/exit
animations, and optional backdrop scrim.

---

## Icons — `ui/icons`

```go
import "github.com/timzifer/lux/ui/icons"
```

Icons use the bundled **Phosphor** icon font (`fonts/phosphor/`). Pass an icon constant to
`button.Icon`, `form.TextField` adornments, or `display.Text` with the icon style.

---

## Planned / Not Yet Implemented

The following widgets are documented in the RFCs but not yet available:

| Widget | RFC | Status |
|--------|-----|--------|
| Code editor | [RFC-010](rfc/RFC-010-lux-code-editor.md) | Planned |
| Date picker | RFC-003 | Planned |
| Color picker | RFC-003 | Planned |
| Rich text editor (level 3) | RFC-003 | Planned |
| Time picker | RFC-003 | Planned |

---

## See Also

- [Layout](layout.md) — layout widgets in depth
- [Theming](theming.md) — accessing design tokens inside widgets
- [Custom Widgets](advanced/custom-widgets.md) — building your own widgets
