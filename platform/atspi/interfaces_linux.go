//go:build linux && !nogui

package atspi

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/timzifer/lux/a11y"
)

const (
	ifaceAccessible = "org.a11y.atspi.Accessible"
	ifaceComponent  = "org.a11y.atspi.Component"
	ifaceAction     = "org.a11y.atspi.Action"
	ifaceValue      = "org.a11y.atspi.Value"
	ifaceText       = "org.a11y.atspi.Text"
	ifaceApplication = "org.a11y.atspi.Application"
)

// accessibleObject exposes an AccessTreeNode over D-Bus, implementing
// the AT-SPI2 Accessible, Component, and optional Action/Value/Text interfaces.
type accessibleObject struct {
	bridge *ATSPIBridge
	nodeID a11y.AccessNodeID
	path   dbus.ObjectPath
}

// objectPath generates a D-Bus object path for the given node ID.
func objectPath(nodeID a11y.AccessNodeID) dbus.ObjectPath {
	return dbus.ObjectPath(fmt.Sprintf("/org/lux/accessible/%d", nodeID))
}

// ── org.a11y.atspi.Accessible ──────────────────────────────────

func (o *accessibleObject) GetName() (string, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return "", nil
	}
	return node.Node.Label, nil
}

func (o *accessibleObject) GetDescription() (string, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return "", nil
	}
	return node.Node.Description, nil
}

func (o *accessibleObject) GetRole() (uint32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return roleUnknown, nil
	}
	return mapRole(node.Node.Role), nil
}

func (o *accessibleObject) GetState() ([2]uint32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return [2]uint32{}, nil
	}
	return mapStates(node.Node.States), nil
}

func (o *accessibleObject) GetChildCount() (int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return 0, nil
	}
	return int32(node.ChildCount), nil
}

func (o *accessibleObject) GetChildAtIndex(idx int32) (dbus.ObjectPath, *dbus.Error) {
	o.bridge.mu.RLock()
	defer o.bridge.mu.RUnlock()

	node := o.bridge.tree.FindByID(o.nodeID)
	if node == nil || node.FirstChild < 0 {
		return "/", nil
	}
	children := o.bridge.tree.Children(node)
	if int(idx) >= len(children) {
		return "/", nil
	}
	return objectPath(children[idx].ID), nil
}

func (o *accessibleObject) GetIndexInParent() (int32, *dbus.Error) {
	o.bridge.mu.RLock()
	defer o.bridge.mu.RUnlock()

	node := o.bridge.tree.FindByID(o.nodeID)
	if node == nil || node.ParentIndex < 0 {
		return -1, nil
	}
	parent := o.bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		return -1, nil
	}
	children := o.bridge.tree.Children(parent)
	for i, c := range children {
		if c.ID == o.nodeID {
			return int32(i), nil
		}
	}
	return -1, nil
}

func (o *accessibleObject) GetParent() (dbus.ObjectPath, *dbus.Error) {
	o.bridge.mu.RLock()
	defer o.bridge.mu.RUnlock()

	node := o.bridge.tree.FindByID(o.nodeID)
	if node == nil || node.ParentIndex < 0 {
		return "/", nil
	}
	parent := o.bridge.tree.NodeByIndex(int(node.ParentIndex))
	if parent == nil {
		return "/", nil
	}
	return objectPath(parent.ID), nil
}

// ── org.a11y.atspi.Component ───────────────────────────────────

func (o *accessibleObject) GetExtents(coordType uint32) (int32, int32, int32, int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return 0, 0, 0, 0, nil
	}
	return int32(node.Bounds.X), int32(node.Bounds.Y),
		int32(node.Bounds.Width), int32(node.Bounds.Height), nil
}

func (o *accessibleObject) GetPosition(coordType uint32) (int32, int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return 0, 0, nil
	}
	return int32(node.Bounds.X), int32(node.Bounds.Y), nil
}

func (o *accessibleObject) GetSize() (int32, int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return 0, 0, nil
	}
	return int32(node.Bounds.Width), int32(node.Bounds.Height), nil
}

func (o *accessibleObject) Contains(x, y int32, coordType uint32) (bool, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return false, nil
	}
	b := node.Bounds
	fx, fy := float64(x), float64(y)
	return fx >= b.X && fx < b.X+b.Width && fy >= b.Y && fy < b.Y+b.Height, nil
}

// ── org.a11y.atspi.Action ──────────────────────────────────────

func (o *accessibleObject) GetNActions() (int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return 0, nil
	}
	return int32(len(node.Node.Actions)), nil
}

func (o *accessibleObject) GetActionName(idx int32) (string, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || int(idx) >= len(node.Node.Actions) {
		return "", nil
	}
	return node.Node.Actions[idx].Name, nil
}

func (o *accessibleObject) DoAction(idx int32) (bool, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || int(idx) >= len(node.Node.Actions) {
		return false, nil
	}
	action := node.Node.Actions[idx]
	if action.Trigger != nil {
		action.Trigger()
		return true, nil
	}
	return false, nil
}

// ── org.a11y.atspi.Value ───────────────────────────────────────

func (o *accessibleObject) GetCurrentValue() (float64, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.NumericValue == nil {
		return 0, nil
	}
	return node.Node.NumericValue.Current, nil
}

func (o *accessibleObject) GetMinimumValue() (float64, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.NumericValue == nil {
		return 0, nil
	}
	return node.Node.NumericValue.Min, nil
}

func (o *accessibleObject) GetMaximumValue() (float64, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.NumericValue == nil {
		return 0, nil
	}
	return node.Node.NumericValue.Max, nil
}

// ── org.a11y.atspi.Text ────────────────────────────────────────

func (o *accessibleObject) GetText(startOffset, endOffset int32) (string, *dbus.Error) {
	node := o.lookupNode()
	if node == nil {
		return "", nil
	}
	text := node.Node.Value
	runes := []rune(text)
	start := int(startOffset)
	end := int(endOffset)
	if end < 0 || end > len(runes) {
		end = len(runes)
	}
	if start < 0 {
		start = 0
	}
	if start > end {
		return "", nil
	}
	return string(runes[start:end]), nil
}

func (o *accessibleObject) GetCaretOffset() (int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.TextState == nil {
		return -1, nil
	}
	return int32(node.Node.TextState.CaretOffset), nil
}

func (o *accessibleObject) GetCharacterCount() (int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.TextState == nil {
		return 0, nil
	}
	return int32(node.Node.TextState.Length), nil
}

func (o *accessibleObject) GetNSelections() (int32, *dbus.Error) {
	node := o.lookupNode()
	if node == nil || node.Node.TextState == nil {
		return 0, nil
	}
	return int32(len(node.Node.TextState.Selections)), nil
}

// ── Helpers ────────────────────────────────────────────────────

func (o *accessibleObject) lookupNode() *a11y.AccessTreeNode {
	o.bridge.mu.RLock()
	defer o.bridge.mu.RUnlock()
	return o.bridge.tree.FindByID(o.nodeID)
}
