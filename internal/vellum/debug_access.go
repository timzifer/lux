package vellum

import (
	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
)

// WidgetTreeFromAccess derives a DebugWidgetTree from an accessibility tree.
// Nodes whose ID appears in dirty are flagged as dirty so the Inspector can
// highlight widgets repainted in the current frame (RFC-012 §4.2).
//
// The accessibility tree is the closest stable, framework-level description of
// the widget hierarchy that is already built every frame, so the Inspector
// reuses it instead of requiring a parallel widget-introspection pass.
func WidgetTreeFromAccess(tree a11y.AccessTree, version uint64, dirty []uint64) *DebugWidgetTree {
	var dirtySet map[uint64]struct{}
	if len(dirty) > 0 {
		dirtySet = make(map[uint64]struct{}, len(dirty))
		for _, uid := range dirty {
			dirtySet[uid] = struct{}{}
		}
	}

	nodes := make([]DebugWidgetNode, 0, len(tree.Nodes))
	for i := range tree.Nodes {
		n := &tree.Nodes[i]

		var props map[string]string
		if n.Node.Label != "" || n.Node.Value != "" {
			props = make(map[string]string, 2)
			if n.Node.Label != "" {
				props["label"] = n.Node.Label
			}
			if n.Node.Value != "" {
				props["value"] = n.Node.Value
			}
		}

		_, dirtyNode := dirtySet[uint64(n.ID)]
		nodes = append(nodes, DebugWidgetNode{
			UID:      uint64(n.ID),
			TypeName: accessRoleName(n.Node.Role),
			Props:    props,
			Bounds: draw.R(
				float32(n.Bounds.X), float32(n.Bounds.Y),
				float32(n.Bounds.Width), float32(n.Bounds.Height),
			),
			Dirty: dirtyNode,
		})
	}

	return &DebugWidgetTree{Version: version, Nodes: nodes}
}

// accessRoleName maps an accessibility role to a human-readable widget type.
func accessRoleName(role a11y.AccessRole) string {
	switch role {
	case a11y.RoleButton:
		return "Button"
	case a11y.RoleCheckbox:
		return "Checkbox"
	case a11y.RoleCombobox:
		return "Combobox"
	case a11y.RoleDialog:
		return "Dialog"
	case a11y.RoleGrid:
		return "Grid"
	case a11y.RoleGroup:
		return "Group"
	case a11y.RoleHeading:
		return "Heading"
	case a11y.RoleImage:
		return "Image"
	case a11y.RoleLink:
		return "Link"
	case a11y.RoleListbox:
		return "Listbox"
	case a11y.RoleMenu:
		return "Menu"
	case a11y.RoleProgressBar:
		return "ProgressBar"
	case a11y.RoleScrollBar:
		return "ScrollBar"
	case a11y.RoleSlider:
		return "Slider"
	case a11y.RoleSpinButton:
		return "SpinButton"
	case a11y.RoleTab:
		return "Tab"
	case a11y.RoleTable:
		return "Table"
	case a11y.RoleTextInput:
		return "TextInput"
	case a11y.RoleToggle:
		return "Toggle"
	case a11y.RoleTree:
		return "Tree"
	case a11y.RoleToolbar:
		return "Toolbar"
	default:
		return "Group"
	}
}
