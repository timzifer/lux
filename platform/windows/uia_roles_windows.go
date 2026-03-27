//go:build windows && !nogui

package windows

import (
	"github.com/timzifer/lux/a11y"
	"github.com/zzl/go-win32api/v2/win32"
)

// roleToControlType maps AccessRole to UIA_CONTROLTYPE_ID.
func roleToControlType(role a11y.AccessRole) win32.UIA_CONTROLTYPE_ID {
	switch role {
	case a11y.RoleButton:
		return win32.UIA_ButtonControlTypeId
	case a11y.RoleCheckbox:
		return win32.UIA_CheckBoxControlTypeId
	case a11y.RoleCombobox:
		return win32.UIA_ComboBoxControlTypeId
	case a11y.RoleDialog:
		return 50033 // UIA_PaneControlTypeId (Dialog maps to Pane with IsDialog property)
	case a11y.RoleGrid:
		return win32.UIA_TableControlTypeId
	case a11y.RoleGroup:
		return win32.UIA_GroupControlTypeId
	case a11y.RoleHeading:
		return win32.UIA_HeaderControlTypeId
	case a11y.RoleImage:
		return win32.UIA_ImageControlTypeId
	case a11y.RoleLink:
		return win32.UIA_HyperlinkControlTypeId
	case a11y.RoleListbox:
		return win32.UIA_ListControlTypeId
	case a11y.RoleMenu:
		return win32.UIA_MenuControlTypeId
	case a11y.RoleProgressBar:
		return win32.UIA_ProgressBarControlTypeId
	case a11y.RoleScrollBar:
		return win32.UIA_ScrollBarControlTypeId
	case a11y.RoleSlider:
		return win32.UIA_SliderControlTypeId
	case a11y.RoleSpinButton:
		return win32.UIA_SpinnerControlTypeId
	case a11y.RoleTab:
		return win32.UIA_TabControlTypeId
	case a11y.RoleTable:
		return win32.UIA_TableControlTypeId
	case a11y.RoleTextInput:
		return win32.UIA_EditControlTypeId
	case a11y.RoleToggle:
		return win32.UIA_CheckBoxControlTypeId
	case a11y.RoleTree:
		return win32.UIA_TreeControlTypeId
	default:
		return win32.UIA_GroupControlTypeId
	}
}

// patternsForRole returns the UIA pattern IDs that should be supported for the given role.
func patternsForRole(role a11y.AccessRole) []win32.UIA_PATTERN_ID {
	switch role {
	case a11y.RoleButton:
		return []win32.UIA_PATTERN_ID{win32.UIA_InvokePatternId}
	case a11y.RoleCheckbox, a11y.RoleToggle:
		return []win32.UIA_PATTERN_ID{win32.UIA_TogglePatternId}
	case a11y.RoleTextInput:
		return []win32.UIA_PATTERN_ID{win32.UIA_ValuePatternId}
	case a11y.RoleSlider, a11y.RoleProgressBar, a11y.RoleScrollBar, a11y.RoleSpinButton:
		return []win32.UIA_PATTERN_ID{win32.UIA_RangeValuePatternId}
	case a11y.RoleCombobox:
		return []win32.UIA_PATTERN_ID{win32.UIA_ExpandCollapsePatternId}
	default:
		return nil
	}
}
