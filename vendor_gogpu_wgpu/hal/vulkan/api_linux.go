//go:build linux

// Copyright 2025 The GoGPU Authors
// SPDX-License-Identifier: MIT

package vulkan

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/gogpu/wgpu/hal"
	"github.com/gogpu/wgpu/hal/vulkan/vk"
)

// platformSurfaceExtensions returns all Linux surface extensions to request.
// Both X11 and Wayland extensions are requested; the driver enables what it supports.
func platformSurfaceExtension() string {
	// Request both — Vulkan instance creation accepts unsupported extensions gracefully.
	// The actual surface creation checks HasCreate*SurfaceKHR at runtime.
	if isWayland() {
		return "VK_KHR_wayland_surface\x00"
	}
	return "VK_KHR_xlib_surface\x00"
}

// platformExtraExtensions returns additional extensions for Linux.
// VK_KHR_display is needed for DRM/KMS direct rendering without a window system.
func platformExtraExtensions() []string {
	return []string{"VK_KHR_display\x00"}
}

// isWayland returns true if the session is running under Wayland.
func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

// CreateSurface creates a Vulkan surface from platform-specific handles.
// On Linux, it auto-detects X11 vs Wayland based on available extensions:
//   - Wayland: display = wl_display*, window = wl_surface* (from libwayland-client)
//   - X11: display = Display* (from libX11), window = X11 Window ID
func (i *Instance) CreateSurface(display, window uintptr) (hal.Surface, error) {
	// Try Wayland first if the extension is available
	if i.cmds.HasCreateWaylandSurfaceKHR() && isWayland() {
		return i.createWaylandSurface(display, window)
	}

	// Fall back to X11
	if i.cmds.HasCreateXlibSurfaceKHR() {
		return i.createXlibSurface(display, window)
	}

	return nil, fmt.Errorf("vulkan: no surface creation extension available (need VK_KHR_xlib_surface or VK_KHR_wayland_surface)")
}

// CreateDisplaySurface creates a Vulkan surface via VK_KHR_display for DRM/KMS rendering.
// This enables direct rendering to a display without a window system (X11/Wayland).
// The drmFD is used to identify the physical device, and connectorID selects the display.
func (i *Instance) CreateDisplaySurface(drmFD int, connectorID uint32) (hal.Surface, error) {
	if !i.cmds.HasCreateDisplayPlaneSurfaceKHR() || !i.cmds.HasGetPhysicalDeviceDisplayPropertiesKHR() {
		return nil, fmt.Errorf("vulkan: VK_KHR_display not available")
	}

	// Find a physical device (we need one for display enumeration).
	var deviceCount uint32
	result := i.cmds.EnumeratePhysicalDevices(i.handle, &deviceCount, nil)
	if result != vk.Success || deviceCount == 0 {
		return nil, fmt.Errorf("vulkan: no physical devices found for display surface")
	}
	devices := make([]vk.PhysicalDevice, deviceCount)
	result = i.cmds.EnumeratePhysicalDevices(i.handle, &deviceCount, &devices[0])
	if result != vk.Success {
		return nil, fmt.Errorf("vulkan: EnumeratePhysicalDevices failed: %d", result)
	}

	// Try each physical device until we find one with display properties.
	for _, physDev := range devices {
		surface, err := i.tryCreateDisplaySurface(physDev, connectorID)
		if err == nil {
			return surface, nil
		}
	}

	return nil, fmt.Errorf("vulkan: no display found matching DRM connector %d on fd %d", connectorID, drmFD)
}

// tryCreateDisplaySurface attempts to create a display surface on a specific physical device.
func (i *Instance) tryCreateDisplaySurface(physDev vk.PhysicalDevice, connectorID uint32) (hal.Surface, error) {
	// Get display properties.
	var displayCount uint32
	result := i.cmds.GetPhysicalDeviceDisplayPropertiesKHR(physDev, &displayCount, nil)
	if result != vk.Success || displayCount == 0 {
		return nil, fmt.Errorf("no displays on device")
	}
	displays := make([]vk.DisplayPropertiesKHR, displayCount)
	result = i.cmds.GetPhysicalDeviceDisplayPropertiesKHR(physDev, &displayCount, &displays[0])
	if result != vk.Success {
		return nil, fmt.Errorf("GetPhysicalDeviceDisplayPropertiesKHR failed: %d", result)
	}

	// Use the first display (or match by connectorID if we have multiple).
	// VK_KHR_display doesn't expose DRM connector IDs directly, so we use
	// the first available display and its preferred mode.
	for _, dispProp := range displays[:displayCount] {
		display := dispProp.Display
		if display == 0 {
			continue
		}

		// Get display modes.
		var modeCount uint32
		result = i.cmds.GetDisplayModePropertiesKHR(physDev, display, &modeCount, nil)
		if result != vk.Success || modeCount == 0 {
			continue
		}
		modes := make([]vk.DisplayModePropertiesKHR, modeCount)
		result = i.cmds.GetDisplayModePropertiesKHR(physDev, display, &modeCount, &modes[0])
		if result != vk.Success {
			continue
		}

		// Use the first (preferred) mode.
		mode := modes[0]

		// Create display surface.
		createInfo := vk.DisplaySurfaceCreateInfoKHR{
			SType:       vk.StructureTypeDisplaySurfaceCreateInfoKhr,
			DisplayMode: mode.DisplayMode,
			PlaneIndex:  0,
			Transform:   vk.SurfaceTransformFlagBitsKHR(vk.SurfaceTransformIdentityBitKhr),
			GlobalAlpha: 1.0,
			AlphaMode:   vk.DisplayPlaneAlphaOpaqueBitKhr,
			ImageExtent: vk.Extent2D{
				Width:  mode.Parameters.VisibleRegion.Width,
				Height: mode.Parameters.VisibleRegion.Height,
			},
		}

		var surface vk.SurfaceKHR
		result = i.cmds.CreateDisplayPlaneSurfaceKHR(i.handle, &createInfo, nil, &surface)
		if result != vk.Success {
			continue
		}
		if surface == 0 {
			continue
		}

		hal.Logger().Info("vulkan: created VK_KHR_display surface",
			"display", fmt.Sprintf("%x", display),
			"mode", fmt.Sprintf("%dx%d@%dHz",
				mode.Parameters.VisibleRegion.Width,
				mode.Parameters.VisibleRegion.Height,
				mode.Parameters.RefreshRate/1000),
		)

		return &Surface{
			handle:   surface,
			instance: i,
		}, nil
	}

	return nil, fmt.Errorf("no suitable display mode found")
}

// createXlibSurface creates an X11 surface.
func (i *Instance) createXlibSurface(display, window uintptr) (hal.Surface, error) {
	createInfo := vk.XlibSurfaceCreateInfoKHR{
		SType:  vk.StructureTypeXlibSurfaceCreateInfoKhr,
		Window: vk.XlibWindow(window),
	}
	// Write Display* value directly into the Dpy field memory.
	// Dpy is *XlibDisplay (a Go pointer type) but must hold the raw C Display*
	// address. We cannot use unsafe.Pointer(uintptr) — go vet rejects it.
	*(*uintptr)(unsafe.Pointer(&createInfo.Dpy)) = display

	var surface vk.SurfaceKHR
	result := i.cmds.CreateXlibSurfaceKHR(i.handle, &createInfo, nil, &surface)
	if result != vk.Success {
		return nil, fmt.Errorf("vulkan: vkCreateXlibSurfaceKHR failed: %d", result)
	}
	if surface == 0 {
		return nil, fmt.Errorf("vulkan: vkCreateXlibSurfaceKHR returned success but surface is null")
	}

	return &Surface{
		handle:   surface,
		instance: i,
	}, nil
}

// createWaylandSurface creates a Wayland surface.
func (i *Instance) createWaylandSurface(display, window uintptr) (hal.Surface, error) {
	createInfo := vk.WaylandSurfaceCreateInfoKHR{
		SType: vk.StructureTypeWaylandSurfaceCreateInfoKhr,
	}
	// Write wl_display* and wl_surface* values directly into fields.
	// Display is *WlDisplay and Surface is *WlSurface — both Go pointer types
	// that must hold raw C pointer values.
	*(*uintptr)(unsafe.Pointer(&createInfo.Display)) = display
	*(*uintptr)(unsafe.Pointer(&createInfo.Surface)) = window

	var surface vk.SurfaceKHR
	result := i.cmds.CreateWaylandSurfaceKHR(i.handle, &createInfo, nil, &surface)
	if result != vk.Success {
		return nil, fmt.Errorf("vulkan: vkCreateWaylandSurfaceKHR failed: %d", result)
	}
	if surface == 0 {
		return nil, fmt.Errorf("vulkan: vkCreateWaylandSurfaceKHR returned success but surface is null")
	}

	return &Surface{
		handle:   surface,
		instance: i,
	}, nil
}
