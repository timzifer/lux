//go:build linux && !nogui

package atspi

import "github.com/timzifer/lux/a11y"

// AT-SPI2 role constants (from at-spi2-core/atspi/atspi-constants.h).
const (
	roleInvalid       uint32 = 0
	rolePushButton    uint32 = 42
	roleCheckBox      uint32 = 7
	roleComboBox      uint32 = 11
	roleDialog        uint32 = 16
	roleTable         uint32 = 73
	roleFiller        uint32 = 21
	roleHeading       uint32 = 81
	roleImage         uint32 = 26
	roleLink          uint32 = 80
	roleList          uint32 = 34
	roleMenu          uint32 = 38
	roleProgressBar   uint32 = 48
	roleScrollBar     uint32 = 55
	roleSlider        uint32 = 60
	roleSpinButton    uint32 = 63
	rolePageTab       uint32 = 46
	roleTreeTable     uint32 = 78
	roleText          uint32 = 74
	roleToggleButton  uint32 = 77
	roleTreeView      uint32 = 78
	rolePanel         uint32 = 18
	roleApplication   uint32 = 75
	roleFrame         uint32 = 22
	roleWindow        uint32 = 68
	roleStatusBar     uint32 = 66
	roleLabel         uint32 = 29
	roleLayeredPane   uint32 = 31
	roleMenuBar       uint32 = 37
	roleMenuBarItem   uint32 = 39
	roleToolBar       uint32 = 76
	roleListItem      uint32 = 35
	roleRadioButton   uint32 = 49
	roleDocumentFrame uint32 = 17
	roleUnknown       uint32 = 83
)

// mapRole converts a lux AccessRole to an AT-SPI2 role constant.
func mapRole(role a11y.AccessRole) uint32 {
	switch role {
	case a11y.RoleButton:
		return rolePushButton
	case a11y.RoleCheckbox:
		return roleCheckBox
	case a11y.RoleCombobox:
		return roleComboBox
	case a11y.RoleDialog:
		return roleDialog
	case a11y.RoleGrid:
		return roleTable
	case a11y.RoleGroup:
		return rolePanel
	case a11y.RoleHeading:
		return roleHeading
	case a11y.RoleImage:
		return roleImage
	case a11y.RoleLink:
		return roleLink
	case a11y.RoleListbox:
		return roleList
	case a11y.RoleMenu:
		return roleMenu
	case a11y.RoleProgressBar:
		return roleProgressBar
	case a11y.RoleScrollBar:
		return roleScrollBar
	case a11y.RoleSlider:
		return roleSlider
	case a11y.RoleSpinButton:
		return roleSpinButton
	case a11y.RoleTab:
		return rolePageTab
	case a11y.RoleTable:
		return roleTable
	case a11y.RoleTextInput:
		return roleText
	case a11y.RoleToggle:
		return roleToggleButton
	case a11y.RoleTree:
		return roleTreeView
	default:
		if role >= a11y.RoleCustomBase {
			return roleUnknown
		}
		return rolePanel
	}
}

// AT-SPI2 state bit positions (from at-spi2-core/atspi/atspi-constants.h).
const (
	stateActive     = 1
	stateChecked    = 3
	stateEnabled    = 8
	stateExpandable = 9
	stateExpanded   = 10
	stateFocusable  = 12
	stateFocused    = 13
	stateMultiLine  = 17
	stateReadOnly   = 41
	stateRequired   = 42
	stateSelectable = 23
	stateSelected   = 24
	stateSensitive  = 25
	stateShowing    = 26
	stateVisible    = 28
	stateEditable   = 7
	stateInvalid    = 43
	stateBusy       = 4
)

// mapStates converts lux AccessStates to AT-SPI2 state bitfield pair [2]uint32.
// AT-SPI2 uses two 32-bit words for state (64 possible states).
func mapStates(s a11y.AccessStates) [2]uint32 {
	var bits [2]uint32

	// set sets the bit at position p.
	set := func(p int) {
		bits[p/32] |= 1 << (p % 32)
	}

	// All nodes are visible and showing by default.
	set(stateVisible)
	set(stateShowing)
	set(stateSensitive)
	set(stateEnabled)

	if s.Focused {
		set(stateFocused)
	}
	if s.Checked {
		set(stateChecked)
	}
	if s.Selected {
		set(stateSelected)
	}
	if s.Expanded {
		set(stateExpanded)
		set(stateExpandable)
	}
	if s.Disabled {
		// Clear enabled/sensitive.
		bits[stateEnabled/32] &^= 1 << (stateEnabled % 32)
		bits[stateSensitive/32] &^= 1 << (stateSensitive % 32)
	}
	if s.ReadOnly {
		set(stateReadOnly)
	}
	if s.Required {
		set(stateRequired)
	}
	if s.Invalid {
		set(stateInvalid)
	}
	if s.Busy {
		set(stateBusy)
	}

	return bits
}
