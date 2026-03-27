//go:build nogui

package wgpu

import "testing"

// TestCreateInstance verifies the noop instance can be created.
func TestCreateInstance(t *testing.T) {
	inst, err := CreateInstance()
	if err != nil {
		t.Fatalf("CreateInstance() error: %v", err)
	}
	if inst == nil {
		t.Fatal("CreateInstance() returned nil")
	}
	defer inst.Destroy()
}

// TestInstanceCreateSurface verifies surface creation returns a valid Surface.
func TestInstanceCreateSurface(t *testing.T) {
	inst, _ := CreateInstance()
	defer inst.Destroy()

	surface := inst.CreateSurface(&SurfaceDescriptor{NativeHandle: 0})
	if surface == nil {
		t.Fatal("CreateSurface returned nil")
	}
	surface.Destroy()
}

// TestInstanceRequestAdapter verifies adapter request returns successfully.
func TestInstanceRequestAdapter(t *testing.T) {
	inst, _ := CreateInstance()
	defer inst.Destroy()

	adapter, err := inst.RequestAdapter(&RequestAdapterOptions{
		PowerPreference: PowerPreferenceHighPerformance,
	})
	if err != nil {
		t.Fatalf("RequestAdapter() error: %v", err)
	}
	if adapter == nil {
		t.Fatal("RequestAdapter returned nil")
	}

	info := adapter.GetInfo()
	if info.Name == "" {
		t.Error("adapter info name should not be empty")
	}
}

// TestAdapterRequestDevice verifies device creation.
func TestAdapterRequestDevice(t *testing.T) {
	inst, _ := CreateInstance()
	defer inst.Destroy()

	adapter, _ := inst.RequestAdapter(nil)
	device, err := adapter.RequestDevice(&DeviceDescriptor{Label: "test"})
	if err != nil {
		t.Fatalf("RequestDevice() error: %v", err)
	}
	if device == nil {
		t.Fatal("RequestDevice returned nil")
	}
	defer device.Destroy()
}

// TestDeviceCreateResources verifies all resource creation methods.
func TestDeviceCreateResources(t *testing.T) {
	inst, _ := CreateInstance()
	defer inst.Destroy()
	adapter, _ := inst.RequestAdapter(nil)
	device, _ := adapter.RequestDevice(&DeviceDescriptor{})
	defer device.Destroy()

	// Shader module
	shader := device.CreateShaderModule(&ShaderModuleDescriptor{
		Label:  "test-shader",
		Source: "// empty",
	})
	if shader == nil {
		t.Error("CreateShaderModule returned nil")
	}
	shader.Destroy()

	// Buffer
	buf := device.CreateBuffer(&BufferDescriptor{
		Label: "test-buffer",
		Size:  1024,
		Usage: BufferUsageVertex | BufferUsageCopyDst,
	})
	if buf == nil {
		t.Error("CreateBuffer returned nil")
	}
	buf.Destroy()

	// Texture
	tex := device.CreateTexture(&TextureDescriptor{
		Label:  "test-texture",
		Size:   Extent3D{Width: 256, Height: 256, DepthOrArrayLayers: 1},
		Format: TextureFormatRGBA8Unorm,
		Usage:  TextureUsageTextureBinding | TextureUsageCopyDst,
	})
	if tex == nil {
		t.Error("CreateTexture returned nil")
	}
	view := tex.CreateView()
	if view == nil {
		t.Error("CreateView returned nil")
	}
	view.Destroy()
	tex.Destroy()

	// Command encoder
	encoder := device.CreateCommandEncoder()
	if encoder == nil {
		t.Error("CreateCommandEncoder returned nil")
	}
	cmdBuf := encoder.Finish()
	_ = cmdBuf

	// Queue
	queue := device.GetQueue()
	if queue == nil {
		t.Error("GetQueue returned nil")
	}

	// Sampler
	sampler := device.CreateSampler(&SamplerDescriptor{Label: "test"})
	if sampler == nil {
		t.Error("CreateSampler returned nil")
	}
	sampler.Destroy()
}

// TestRenderPass verifies the render pass lifecycle.
func TestRenderPass(t *testing.T) {
	inst, _ := CreateInstance()
	defer inst.Destroy()
	adapter, _ := inst.RequestAdapter(nil)
	device, _ := adapter.RequestDevice(&DeviceDescriptor{})
	defer device.Destroy()

	surface := inst.CreateSurface(&SurfaceDescriptor{})
	view, err := surface.GetCurrentTexture()
	if err != nil {
		t.Fatalf("GetCurrentTexture() error: %v", err)
	}

	encoder := device.CreateCommandEncoder()
	pass := encoder.BeginRenderPass(&RenderPassDescriptor{
		ColorAttachments: []RenderPassColorAttachment{
			{
				View:       view,
				LoadOp:     LoadOpClear,
				StoreOp:    StoreOpStore,
				ClearValue: Color{R: 0.1, G: 0.1, B: 0.1, A: 1.0},
			},
		},
	})
	if pass == nil {
		t.Fatal("BeginRenderPass returned nil")
	}
	pass.End()

	cmdBuf := encoder.Finish()
	device.GetQueue().Submit(cmdBuf)
}

// TestTextureFormat constants exist.
func TestTextureFormats(t *testing.T) {
	formats := []TextureFormat{
		TextureFormatBGRA8Unorm,
		TextureFormatRGBA8Unorm,
		TextureFormatR8Unorm,
	}
	for i, f := range formats {
		if int(f) != i {
			t.Errorf("TextureFormat %d has value %d", i, f)
		}
	}
}
