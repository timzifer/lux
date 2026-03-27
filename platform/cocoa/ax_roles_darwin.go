//go:build darwin && cocoa && !nogui && arm64

package cocoa

import "github.com/timzifer/lux/a11y"

// roleToAXRole maps AccessRole to NSAccessibility role strings.
func roleToAXRole(role a11y.AccessRole) string {
	switch role {
	case a11y.RoleButton:
		return "AXButton"
	case a11y.RoleCheckbox:
		return "AXCheckBox"
	case a11y.RoleCombobox:
		return "AXComboBox"
	case a11y.RoleDialog:
		return "AXGroup"
	case a11y.RoleGrid:
		return "AXTable"
	case a11y.RoleGroup:
		return "AXGroup"
	case a11y.RoleHeading:
		return "AXHeading"
	case a11y.RoleImage:
		return "AXImage"
	case a11y.RoleLink:
		return "AXLink"
	case a11y.RoleListbox:
		return "AXList"
	case a11y.RoleMenu:
		return "AXMenu"
	case a11y.RoleProgressBar:
		return "AXProgressIndicator"
	case a11y.RoleScrollBar:
		return "AXScrollBar"
	case a11y.RoleSlider:
		return "AXSlider"
	case a11y.RoleSpinButton:
		return "AXIncrementor"
	case a11y.RoleTab:
		return "AXRadioButton"
	case a11y.RoleTable:
		return "AXTable"
	case a11y.RoleTextInput:
		return "AXTextField"
	case a11y.RoleToggle:
		return "AXCheckBox"
	case a11y.RoleTree:
		return "AXOutline"
	default:
		return "AXGroup"
	}
}

// subroleForRole returns the NSAccessibility subrole for roles that need one.
func subroleForRole(role a11y.AccessRole) string {
	if role == a11y.RoleDialog {
		return "AXDialog"
	}
	return ""
}

// actionsForRole returns the applicable NSAccessibility action names for a role.
func actionsForRole(role a11y.AccessRole) []string {
	switch role {
	case a11y.RoleButton, a11y.RoleLink, a11y.RoleCheckbox, a11y.RoleToggle, a11y.RoleTab:
		return []string{"AXPress"}
	case a11y.RoleSlider, a11y.RoleSpinButton:
		return []string{"AXIncrement", "AXDecrement"}
	default:
		return nil
	}
}
