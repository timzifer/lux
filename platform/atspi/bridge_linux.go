//go:build linux && !nogui

// Package atspi implements the AT-SPI2 accessibility bridge for Linux.
// It exposes the Lux AccessTree over D-Bus so that screen readers
// (Orca, NVDA for Linux, etc.) can interact with the application.
//
// No CGo is required — communication is pure D-Bus via github.com/godbus/dbus.
package atspi

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/timzifer/lux/a11y"
)

const (
	basePath = "/org/lux/accessible"
	appPath  = "/org/lux/accessible/app"
)

// ATSPIBridge implements a11y.A11yBridge for Linux AT-SPI2.
type ATSPIBridge struct {
	conn    *dbus.Conn
	objects map[a11y.AccessNodeID]*accessibleObject
	send    func(any)
	mu      sync.RWMutex
	tree    a11y.AccessTree
	appName string
}

// NewATSPIBridge creates and initializes an AT-SPI2 bridge.
// The send function routes accessibility actions back to the app loop.
func NewATSPIBridge(appName string, send func(any)) (*ATSPIBridge, error) {
	conn, err := connectA11yBus()
	if err != nil {
		return nil, fmt.Errorf("atspi: failed to connect: %w", err)
	}

	b := &ATSPIBridge{
		conn:    conn,
		objects: make(map[a11y.AccessNodeID]*accessibleObject),
		send:    send,
		appName: appName,
	}

	// Register application with the AT-SPI2 registry.
	if regErr := registerApplication(conn, dbus.ObjectPath(appPath)); regErr != nil {
		// Non-fatal: screen reader may not be running.
		_ = regErr
	}

	return b, nil
}

// UpdateTree replaces the current access tree and updates D-Bus objects.
func (b *ATSPIBridge) UpdateTree(tree a11y.AccessTree) {
	b.mu.Lock()

	tree.EnsureIndex()
	b.tree = tree

	// Prune objects for nodes no longer in the tree.
	for id, obj := range b.objects {
		if tree.FindByID(id) == nil {
			b.conn.Export(nil, obj.path, ifaceAccessible)
			b.conn.Export(nil, obj.path, ifaceComponent)
			b.conn.Export(nil, obj.path, ifaceAction)
			b.conn.Export(nil, obj.path, ifaceValue)
			b.conn.Export(nil, obj.path, ifaceText)
			delete(b.objects, id)
		}
	}

	// Create or update objects for all nodes.
	for i := range tree.Nodes {
		id := tree.Nodes[i].ID
		if _, exists := b.objects[id]; !exists {
			obj := &accessibleObject{
				bridge: b,
				nodeID: id,
				path:   objectPath(id),
			}
			b.objects[id] = obj

			// Export the object on all interfaces.
			b.conn.Export(obj, obj.path, ifaceAccessible)
			b.conn.Export(obj, obj.path, ifaceComponent)

			node := &tree.Nodes[i]
			if len(node.Node.Actions) > 0 {
				b.conn.Export(obj, obj.path, ifaceAction)
			}
			if node.Node.NumericValue != nil {
				b.conn.Export(obj, obj.path, ifaceValue)
			}
			if node.Node.TextState != nil {
				b.conn.Export(obj, obj.path, ifaceText)
			}
		}
	}

	b.mu.Unlock()

	// Emit structural change signal.
	b.emitSignal(dbus.ObjectPath(appPath),
		"org.a11y.atspi.Event.Object", "ChildrenChanged",
		"add", 0, 0, dbus.MakeVariant(""),
	)
}

// NotifyFocus informs AT-SPI2 that keyboard focus moved to the given node.
func (b *ATSPIBridge) NotifyFocus(nodeID a11y.AccessNodeID) {
	b.mu.RLock()
	obj, ok := b.objects[nodeID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	b.emitSignal(obj.path,
		"org.a11y.atspi.Event.Focus", "Focus",
		"", 0, 0, dbus.MakeVariant(""),
	)
}

// NotifyLiveRegion announces a live-region content change via AT-SPI2.
func (b *ATSPIBridge) NotifyLiveRegion(nodeID a11y.AccessNodeID, text string) {
	b.mu.RLock()
	obj, ok := b.objects[nodeID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	b.emitSignal(obj.path,
		"org.a11y.atspi.Event.Object", "TextChanged",
		"insert", 0, int32(len([]rune(text))), dbus.MakeVariant(text),
	)
}

// Destroy releases all D-Bus resources and deregisters the application.
func (b *ATSPIBridge) Destroy() {
	b.mu.Lock()
	for _, obj := range b.objects {
		b.conn.Export(nil, obj.path, ifaceAccessible)
		b.conn.Export(nil, obj.path, ifaceComponent)
		b.conn.Export(nil, obj.path, ifaceAction)
		b.conn.Export(nil, obj.path, ifaceValue)
		b.conn.Export(nil, obj.path, ifaceText)
	}
	b.objects = nil
	b.mu.Unlock()

	deregisterApplication(b.conn, dbus.ObjectPath(appPath))
	b.conn.Close()
}

// emitSignal sends an AT-SPI2 event signal on the D-Bus.
func (b *ATSPIBridge) emitSignal(path dbus.ObjectPath, iface, member, detail string, v1, v2 int32, v3 dbus.Variant) {
	sig := dbus.Signal{
		Path: path,
		Name: iface + "." + member,
		Body: []interface{}{detail, v1, v2, v3},
	}
	_ = b.conn.Emit(sig.Path, sig.Name, sig.Body...)
}

// Compile-time check that ATSPIBridge implements a11y.A11yBridge.
var _ a11y.A11yBridge = (*ATSPIBridge)(nil)
