// Package nav — windowtabs.go provides a tab-based window manager for
// no-compositor environments (DRM/KMS). When the platform has no compositor,
// OpenWindow/CloseWindow calls are redirected to this tab panel instead of
// creating real OS windows.
package nav

import (
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// WindowTab represents a single tab in the window tab panel,
// corresponding to a logical window.
type WindowTab struct {
	ID      uint32
	Title   string
	Content ui.Element
	// Modal indicates this tab was opened as a dialog and blocks its parent.
	Modal bool
	// ParentID is the tab that opened this modal tab. Only relevant when Modal is true.
	ParentID uint32
}

// WindowTabPanel manages logical windows as tabs in no-compositor mode.
// It wraps all window content in a single tab strip rendered at the
// configured position of the main framebuffer.
type WindowTabPanel struct {
	tabs     []WindowTab
	selected uint32 // ID of the currently visible tab
	onSelect func(uint32)
	onClose  func(uint32)
	blocked  map[uint32]bool // tab IDs blocked by a modal child
	position TabPosition
}

// NewWindowTabPanel creates a new panel with the given tab position.
func NewWindowTabPanel(onSelect func(uint32), onClose func(uint32), position TabPosition) *WindowTabPanel {
	return &WindowTabPanel{
		onSelect: onSelect,
		onClose:  onClose,
		blocked:  make(map[uint32]bool),
		position: position,
	}
}

// AddTab adds a new tab. If modal is true, the currently selected tab is blocked.
func (p *WindowTabPanel) AddTab(id uint32, title string, modal bool) {
	parent := p.selected
	p.tabs = append(p.tabs, WindowTab{
		ID:       id,
		Title:    title,
		Modal:    modal,
		ParentID: parent,
	})
	if modal {
		p.blocked[parent] = true
	}
	p.selected = id
}

// RemoveTab removes a tab by ID. If it was modal, its parent is unblocked.
func (p *WindowTabPanel) RemoveTab(id uint32) {
	for i, tab := range p.tabs {
		if tab.ID == id {
			if tab.Modal {
				delete(p.blocked, tab.ParentID)
				p.selected = tab.ParentID
			} else if p.selected == id {
				// Select adjacent tab.
				if i > 0 {
					p.selected = p.tabs[i-1].ID
				} else if i+1 < len(p.tabs) {
					p.selected = p.tabs[i+1].ID
				}
			}
			p.tabs = append(p.tabs[:i], p.tabs[i+1:]...)
			return
		}
	}
}

// SetContent updates the content element for a given tab.
func (p *WindowTabPanel) SetContent(id uint32, content ui.Element) {
	for i := range p.tabs {
		if p.tabs[i].ID == id {
			p.tabs[i].Content = content
			return
		}
	}
}

// Selected returns the currently selected tab ID.
func (p *WindowTabPanel) Selected() uint32 { return p.selected }

// SelectTab switches to the tab with the given ID, if it exists and is not blocked.
func (p *WindowTabPanel) SelectTab(id uint32) {
	if p.blocked[id] {
		return
	}
	for _, tab := range p.tabs {
		if tab.ID == id {
			p.selected = id
			return
		}
	}
}

// SetPosition changes the tab bar position at runtime.
func (p *WindowTabPanel) SetPosition(pos TabPosition) {
	p.position = pos
}

// Position returns the current tab bar position.
func (p *WindowTabPanel) Position() TabPosition {
	return p.position
}

// TabCount returns the number of tabs.
func (p *WindowTabPanel) TabCount() int { return len(p.tabs) }

// HasTab reports whether a tab with the given ID exists.
func (p *WindowTabPanel) HasTab(id uint32) bool {
	for _, tab := range p.tabs {
		if tab.ID == id {
			return true
		}
	}
	return false
}

// WindowTabPanelElement is the renderable element for the window tab panel.
// It delegates tab header/content layout to the standard Tabs component
// and adds window-specific overlays (modal scrim, blocked tab dimming).
type WindowTabPanelElement struct {
	ui.BaseElement
	Panel *WindowTabPanel
}

// NewWindowTabPanelElement creates a WindowTabPanelElement.
func NewWindowTabPanelElement(panel *WindowTabPanel) ui.Element {
	if panel == nil || len(panel.tabs) == 0 {
		return ui.Empty()
	}
	return WindowTabPanelElement{Panel: panel}
}

// Window tab layout constants.
const (
	wtabCloseW   = 20
	wtabCloseGap = 6
)

// windowTabHeaderElement renders a single window-tab header: title text,
// optional close button, and respects blocked state for dimming.
type windowTabHeaderElement struct {
	ui.BaseElement
	title    string
	showClose bool
	blocked  bool
	onClose  func()
}

// LayoutSelf implements ui.Layouter for the window tab header.
func (h windowTabHeaderElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	tokens := ctx.Tokens
	headerStyle := tokens.Typography.Label

	textColor := tokens.Colors.Text.Primary
	if h.blocked {
		textColor = tokens.Colors.Text.Disabled
	}

	// Measure and draw title text.
	m := ctx.Canvas.MeasureText(h.title, headerStyle)
	ctx.Canvas.DrawText(h.title, draw.Pt(float32(area.X), float32(area.Y)), headerStyle, textColor)

	totalW := int(m.Width)

	// Close button.
	if h.showClose {
		closeX := float32(area.X + totalW + wtabCloseGap)
		closeY := float32(area.Y) + (headerStyle.Size-float32(wtabCloseW))/2
		closeRect := draw.R(closeX, closeY, float32(wtabCloseW), float32(wtabCloseW))
		if ctx.IX != nil && h.onClose != nil {
			ctx.IX.RegisterHit(closeRect, h.onClose)
		}
		// Simple X icon: two crossing lines.
		cx := closeX + float32(wtabCloseW)/2
		cy := closeY + float32(wtabCloseW)/2
		sz := float32(wtabCloseW) * 0.3
		ctx.Canvas.FillRect(draw.R(cx-sz, cy-0.5, sz*2, 1), draw.SolidPaint(textColor))
		ctx.Canvas.FillRect(draw.R(cx-0.5, cy-sz, 1, sz*2), draw.SolidPaint(textColor))

		totalW += wtabCloseGap + wtabCloseW
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: totalW, H: int(headerStyle.Size)}
}

// LayoutSelf implements ui.Layouter for the window tab panel.
// It builds TabItems from the window tabs and delegates to a Tabs element.
func (n WindowTabPanelElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	panel := n.Panel
	area := ctx.Area
	if panel == nil || len(panel.tabs) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	// Build TabItems from window tabs.
	items := make([]TabItem, len(panel.tabs))
	selectedIdx := 0
	for i, tab := range panel.tabs {
		if tab.ID == panel.selected {
			selectedIdx = i
		}

		showClose := len(panel.tabs) > 1 && tab.ID == panel.selected && tab.ID != 0
		isBlocked := panel.blocked[tab.ID]
		tabID := tab.ID

		var closeFn func()
		if showClose && panel.onClose != nil {
			onCl := panel.onClose
			closeFn = func() { onCl(tabID) }
		}

		items[i] = TabItem{
			Header: windowTabHeaderElement{
				title:     tab.Title,
				showClose: showClose,
				blocked:   isBlocked,
				onClose:   closeFn,
			},
			Content: tab.Content,
		}
	}

	// Build a Tabs element with the configured position.
	// We use a custom onSelect that respects blocked tabs.
	onSelect := func(idx int) {
		if idx >= 0 && idx < len(panel.tabs) {
			tab := panel.tabs[idx]
			if !panel.blocked[tab.ID] && panel.onSelect != nil {
				panel.onSelect(tab.ID)
			}
		}
	}

	tabsEl := Tabs{
		Items:    items,
		Selected: selectedIdx,
		OnSelect: onSelect,
		Position: panel.position,
	}

	// Delegate layout to the Tabs component.
	bounds := ctx.LayoutChild(tabsEl, area)

	// If the selected tab is blocked by a modal, draw a scrim overlay on the content area.
	selectedTab := panel.tabs[selectedIdx]
	if panel.blocked[selectedTab.ID] {
		// Calculate content area based on position to draw the scrim correctly.
		scrimRect := draw.R(float32(area.X), float32(area.Y), float32(bounds.W), float32(bounds.H))
		ctx.Canvas.FillRect(scrimRect, draw.SolidPaint(ctx.Tokens.Colors.Surface.Scrim))
	}

	return bounds
}

// TreeEqual implements ui.TreeEqualizer.
func (n WindowTabPanelElement) TreeEqual(other ui.Element) bool {
	return false // always re-render (dynamic content)
}

// ResolveChildren implements ui.ChildResolver.
func (n WindowTabPanelElement) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	if n.Panel == nil {
		return n
	}
	for i := range n.Panel.tabs {
		if n.Panel.tabs[i].Content != nil && n.Panel.tabs[i].ID == n.Panel.selected {
			n.Panel.tabs[i].Content = resolve(n.Panel.tabs[i].Content, int(n.Panel.tabs[i].ID))
		}
	}
	return n
}
