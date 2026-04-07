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
// It wraps all window content in a single tab strip rendered at the top
// of the main framebuffer.
type WindowTabPanel struct {
	tabs       []WindowTab
	selected   uint32 // ID of the currently visible tab
	onSelect   func(uint32)
	onClose    func(uint32)
	blocked    map[uint32]bool // tab IDs blocked by a modal child
}

// NewWindowTabPanel creates a new panel with a main tab.
func NewWindowTabPanel(onSelect func(uint32), onClose func(uint32)) *WindowTabPanel {
	return &WindowTabPanel{
		onSelect: onSelect,
		onClose:  onClose,
		blocked:  make(map[uint32]bool),
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
	wtabHeaderPadX   = 12
	wtabHeaderPadY   = 8
	wtabCloseW       = 20
	wtabCloseGap     = 6
	wtabIndicatorH   = 2
	wtabMinTabW      = 80
)

// LayoutSelf implements ui.Layouter.
func (n WindowTabPanelElement) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	panel := n.Panel
	area := ctx.Area
	if panel == nil || len(panel.tabs) == 0 {
		return ui.Bounds{X: area.X, Y: area.Y}
	}

	canvas := ctx.Canvas
	tokens := ctx.Tokens

	// Touch-adaptive sizing: scale tab headers to meet MinTouchTarget (RFC-004 §2).
	padY := wtabHeaderPadY
	closeW := wtabCloseW
	minTabW := wtabMinTabW
	if ctx.IsTouch() && ctx.Profile != nil {
		minT := int(ctx.Profile.MinTouchTarget)
		padY = (minT - int(ctx.Tokens.Typography.Label.Size)) / 2
		if padY < wtabHeaderPadY {
			padY = wtabHeaderPadY
		}
		closeW = minT / 2
		minTabW = minT * 2
	}

	headerStyle := tokens.Typography.Label
	headerH := int(headerStyle.Size) + padY*2

	// Draw tab header row.
	cursorX := area.X
	selectedIdx := 0
	for i, tab := range panel.tabs {
		if tab.ID == panel.selected {
			selectedIdx = i
		}

		// Measure tab title width.
		m := canvas.MeasureText(tab.Title, headerStyle)
		tw := int(m.Width) + wtabHeaderPadX*2
		if len(panel.tabs) > 1 {
			tw += wtabCloseGap + closeW // space for close button
		}
		if tw < minTabW {
			tw = minTabW
		}

		isBlocked := panel.blocked[tab.ID]
		isSelected := tab.ID == panel.selected

		// Tab background.
		tabRect := draw.R(float32(cursorX), float32(area.Y), float32(tw), float32(headerH))
		if isSelected {
			tonalBg := ui.LerpColor(tokens.Colors.Surface.Base, tokens.Colors.Accent.Primary, 0.08)
			canvas.FillRect(tabRect, draw.SolidPaint(tonalBg))
		}

		// Hit target for tab selection (unless blocked by modal).
		if !isBlocked && !isSelected && ctx.IX != nil {
			tabID := tab.ID
			onSel := panel.onSelect
			ctx.IX.RegisterHit(tabRect, func() {
				if onSel != nil {
					onSel(tabID)
				}
			})
		} else if ctx.IX != nil {
			// Eat the click so it doesn't fall through.
			ctx.IX.RegisterHit(tabRect, func() {})
		}

		// Tab title text.
		textColor := tokens.Colors.Text.Primary
		if isBlocked {
			textColor = tokens.Colors.Text.Disabled
		}
		textX := float32(cursorX + wtabHeaderPadX)
		textY := float32(area.Y + padY)
		canvas.DrawText(tab.Title, draw.Pt(textX, textY), headerStyle, textColor)

		// Close button (only if more than one tab and tab is selected).
		if len(panel.tabs) > 1 && isSelected && tab.ID != 0 {
			closeX := float32(cursorX+tw-wtabHeaderPadX-closeW)
			closeY := float32(area.Y) + float32(headerH-int(headerStyle.Size))/2
			closeRect := draw.R(closeX, closeY, float32(closeW), headerStyle.Size)
			if ctx.IX != nil {
				tabID := tab.ID
				onCl := panel.onClose
				ctx.IX.RegisterHit(closeRect, func() {
					if onCl != nil {
						onCl(tabID)
					}
				})
			}
			// Simple X icon.
			cx := closeX + float32(closeW)/2
			cy := closeY + headerStyle.Size/2
			sz := headerStyle.Size * 0.3
			canvas.FillRect(draw.R(cx-sz, cy-0.5, sz*2, 1), draw.SolidPaint(textColor))
			canvas.FillRect(draw.R(cx-0.5, cy-sz, 1, sz*2), draw.SolidPaint(textColor))
		}

		// Selection indicator.
		if isSelected {
			canvas.FillRect(
				draw.R(float32(cursorX), float32(area.Y+headerH-wtabIndicatorH), float32(tw), float32(wtabIndicatorH)),
				draw.SolidPaint(tokens.Colors.Accent.Primary))
		}

		// Separator between tabs.
		if i < len(panel.tabs)-1 {
			canvas.FillRect(
				draw.R(float32(cursorX+tw), float32(area.Y+2), 1, float32(headerH-4)),
				draw.SolidPaint(tokens.Colors.Stroke.Divider))
		}

		cursorX += tw
	}

	// Divider below headers.
	totalHeaderW := cursorX - area.X
	canvas.FillRect(
		draw.R(float32(area.X), float32(area.Y+headerH), float32(max(totalHeaderW, area.W)), 1),
		draw.SolidPaint(tokens.Colors.Stroke.Divider))

	// Selected tab content.
	contentY := area.Y + headerH + 1
	contentArea := ui.Bounds{X: area.X, Y: contentY, W: area.W, H: max(area.H-headerH-1, 0)}
	selectedTab := panel.tabs[selectedIdx]
	var cb ui.Bounds
	if selectedTab.Content != nil {
		cb = ctx.LayoutChild(selectedTab.Content, contentArea)
	}

	// If the selected tab is blocked by a modal, draw a scrim overlay.
	if panel.blocked[selectedTab.ID] {
		scrimRect := draw.R(float32(area.X), float32(contentY), float32(area.W), float32(contentArea.H))
		canvas.FillRect(scrimRect, draw.SolidPaint(tokens.Colors.Surface.Scrim))
	}

	totalH := headerH + 1 + cb.H
	return ui.Bounds{X: area.X, Y: area.Y, W: area.W, H: totalH}
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
