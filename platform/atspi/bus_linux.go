//go:build linux && !nogui

package atspi

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

const (
	a11yBusName      = "org.a11y.Bus"
	a11yBusPath      = "/org/a11y/bus"
	a11yBusInterface = "org.a11y.Bus"
	a11yBusMethod    = "org.a11y.Bus.GetAddress"

	registryBusName = "org.a11y.atspi.Registry"
	registryPath    = "/org/a11y/atspi/registry"
	registryIface   = "org.a11y.atspi.Registry"
)

// connectA11yBus connects to the AT-SPI2 accessibility bus.
// First tries the dedicated a11y bus (query session bus for the address),
// then falls back to the session bus itself.
func connectA11yBus() (*dbus.Conn, error) {
	// Try the a11y bus address from the session bus.
	sessionConn, err := dbus.ConnectSessionBus()
	if err == nil {
		addr, busErr := getA11yBusAddress(sessionConn)
		sessionConn.Close()
		if busErr == nil && addr != "" {
			a11yConn, connErr := dbus.Connect(addr)
			if connErr == nil {
				return a11yConn, nil
			}
		}
	}

	// Try the AT_SPI_BUS_ADDRESS environment variable.
	if envAddr := os.Getenv("AT_SPI_BUS_ADDRESS"); envAddr != "" {
		conn, connErr := dbus.Connect(envAddr)
		if connErr == nil {
			return conn, nil
		}
	}

	// Fallback: use session bus directly.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("atspi: cannot connect to D-Bus: %w", err)
	}
	return conn, nil
}

// getA11yBusAddress queries the session bus for the AT-SPI2 accessibility
// bus address via the org.a11y.Bus interface.
func getA11yBusAddress(conn *dbus.Conn) (string, error) {
	obj := conn.Object(a11yBusName, a11yBusPath)
	call := obj.Call(a11yBusMethod, 0)
	if call.Err != nil {
		return "", call.Err
	}
	var addr string
	if err := call.Store(&addr); err != nil {
		return "", err
	}
	return addr, nil
}

// registerApplication registers this process as an AT-SPI2 application
// with the accessibility registry.
func registerApplication(conn *dbus.Conn, appPath dbus.ObjectPath) error {
	obj := conn.Object(registryBusName, registryPath)
	call := obj.Call(registryIface+".RegisterApplication", 0, appPath)
	return call.Err
}

// deregisterApplication removes this process from the AT-SPI2 registry.
func deregisterApplication(conn *dbus.Conn, appPath dbus.ObjectPath) {
	obj := conn.Object(registryBusName, registryPath)
	obj.Call(registryIface+".DeregisterApplication", 0, appPath)
}
