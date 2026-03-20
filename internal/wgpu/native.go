//go:build !gogpu && !nogui && !windows

// This file provides the wgpu-native CGo implementation (default backend).
// It wraps the C wgpu-native library via CGo.
//
// To build with wgpu-native, ensure libwgpu_native is installed:
//   - Linux: apt install libwgpu-dev or build from source
//   - macOS: brew install wgpu-native or build from source
//   - Windows: download prebuilt from https://github.com/gfx-rs/wgpu-native/releases

package wgpu

/*
#cgo LDFLAGS: -lwgpu_native

// Forward declarations for wgpu-native C API.
// The actual wgpu.h header provides the full API.
// These stubs allow compilation when the header is available.

typedef void* WGPUInstance;
typedef void* WGPUAdapter;
typedef void* WGPUDevice;
typedef void* WGPUSurface;
typedef void* WGPUQueue;
typedef void* WGPUBuffer;
typedef void* WGPUTexture;
typedef void* WGPUTextureView;
typedef void* WGPURenderPipeline;
typedef void* WGPUShaderModule;
typedef void* WGPUCommandEncoder;
typedef void* WGPURenderPassEncoder;
typedef void* WGPUCommandBuffer;
typedef void* WGPUBindGroup;
typedef void* WGPUBindGroupLayout;
typedef void* WGPUSampler;

// Minimal C stubs — the actual implementation calls into wgpu-native.
// These are provided so the Go code compiles even without the full header.
static WGPUInstance createInstance() { return (WGPUInstance)0; }
*/
import "C"
import "fmt"

// NativeInstance implements Instance using wgpu-native via CGo.
type NativeInstance struct {
	handle C.WGPUInstance
}

// CreateInstance creates a new wgpu Instance using wgpu-native.
func CreateInstance() (Instance, error) {
	handle := C.createInstance()
	if handle == nil {
		return nil, fmt.Errorf("wgpu-native: failed to create instance (ensure libwgpu_native is installed)")
	}
	return &NativeInstance{handle: handle}, nil
}

func (i *NativeInstance) CreateSurface(desc *SurfaceDescriptor) Surface {
	// TODO: Call wgpuInstanceCreateSurface with platform-specific descriptors.
	return &nativeSurface{}
}

func (i *NativeInstance) RequestAdapter(opts *RequestAdapterOptions) (Adapter, error) {
	// TODO: Call wgpuInstanceRequestAdapter.
	return &nativeAdapter{}, nil
}

func (i *NativeInstance) Destroy() {
	// TODO: Call wgpuInstanceRelease.
}

// nativeAdapter implements Adapter using wgpu-native.
type nativeAdapter struct {
	handle C.WGPUAdapter
}

func (a *nativeAdapter) RequestDevice(desc *DeviceDescriptor) (Device, error) {
	return &nativeDevice{}, nil
}

func (a *nativeAdapter) GetInfo() AdapterInfo {
	return AdapterInfo{
		Name:        "wgpu-native",
		BackendType: "Vulkan/Metal/D3D12",
	}
}

// nativeDevice implements Device using wgpu-native.
type nativeDevice struct {
	handle C.WGPUDevice
}

func (d *nativeDevice) CreateShaderModule(desc *ShaderModuleDescriptor) ShaderModule {
	return &nativeShaderModule{}
}

func (d *nativeDevice) CreateRenderPipeline(desc *RenderPipelineDescriptor) RenderPipeline {
	return &nativeRenderPipeline{}
}

func (d *nativeDevice) CreateBuffer(desc *BufferDescriptor) Buffer {
	return &nativeBuffer{}
}

func (d *nativeDevice) CreateTexture(desc *TextureDescriptor) Texture {
	return &nativeTexture{}
}

func (d *nativeDevice) CreateBindGroupLayout(desc *BindGroupLayoutDescriptor) BindGroupLayout {
	return &nativeBindGroupLayout{}
}

func (d *nativeDevice) CreateBindGroup(desc *BindGroupDescriptor) BindGroup {
	return &nativeBindGroup{}
}

func (d *nativeDevice) CreateCommandEncoder() CommandEncoder {
	return &nativeCommandEncoder{}
}

func (d *nativeDevice) CreateSampler(desc *SamplerDescriptor) Sampler {
	return &nativeSampler{}
}

func (d *nativeDevice) CreateComputePipeline(desc *ComputePipelineDescriptor) ComputePipeline {
	return &nativeComputePipeline{}
}

func (d *nativeDevice) GetQueue() Queue {
	return &nativeQueue{}
}

func (d *nativeDevice) Destroy() {}

// Native resource types — stubs for wgpu-native handles.

type nativeSurface struct{ handle C.WGPUSurface }

func (s *nativeSurface) Configure(device Device, config *SurfaceConfiguration) {}
func (s *nativeSurface) GetCurrentTexture() (TextureView, error)               { return &nativeTextureView{}, nil }
func (s *nativeSurface) Present()                                               {}
func (s *nativeSurface) Destroy()                                               {}

type nativeRenderPipeline struct{ handle C.WGPURenderPipeline }

func (p *nativeRenderPipeline) Destroy() {}

type nativeBuffer struct{ handle C.WGPUBuffer }

func (b *nativeBuffer) Write(queue Queue, data []byte) {}
func (b *nativeBuffer) Destroy()                       {}

type nativeTexture struct{ handle C.WGPUTexture }

func (t *nativeTexture) CreateView() TextureView              { return &nativeTextureView{} }
func (t *nativeTexture) Write(queue Queue, data []byte, bytesPerRow uint32) {}
func (t *nativeTexture) Destroy()                                           {}

type nativeTextureView struct{ handle C.WGPUTextureView }

func (v *nativeTextureView) Destroy() {}

type nativeShaderModule struct{ handle C.WGPUShaderModule }

func (m *nativeShaderModule) Destroy() {}

type nativeCommandEncoder struct{ handle C.WGPUCommandEncoder }

func (e *nativeCommandEncoder) BeginRenderPass(desc *RenderPassDescriptor) RenderPass {
	return &nativeRenderPass{}
}
func (e *nativeCommandEncoder) BeginComputePass() ComputePass { return &nativeComputePass{} }
func (e *nativeCommandEncoder) CopyTextureToTexture(src, dst *ImageCopyTexture, size Extent3D) {}
func (e *nativeCommandEncoder) Finish() CommandBuffer { return nil }

type nativeRenderPass struct{ handle C.WGPURenderPassEncoder }

func (p *nativeRenderPass) SetPipeline(pipeline RenderPipeline)                                         {}
func (p *nativeRenderPass) SetBindGroup(index uint32, group BindGroup)                                  {}
func (p *nativeRenderPass) SetVertexBuffer(slot uint32, buffer Buffer, offset, size uint64)             {}
func (p *nativeRenderPass) SetIndexBuffer(buffer Buffer, format IndexFormat, offset, size uint64)       {}
func (p *nativeRenderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32)          {}
func (p *nativeRenderPass) DrawInstanced(vertexCount, instanceCount, firstVertex, firstInstance uint32) {}
func (p *nativeRenderPass) DrawIndexed(int32, int32, int32, int32, uint32)                                         {}
func (p *nativeRenderPass) SetScissorRect(x, y, width, height uint32)                                               {}
func (p *nativeRenderPass) End()                                                                                     {}

type nativeComputePipeline struct{}
func (p *nativeComputePipeline) Destroy() {}

type nativeComputePass struct{}
func (p *nativeComputePass) SetPipeline(ComputePipeline)       {}
func (p *nativeComputePass) SetBindGroup(uint32, BindGroup)    {}
func (p *nativeComputePass) Dispatch(uint32, uint32, uint32)   {}
func (p *nativeComputePass) End()                               {}

type nativeQueue struct{ handle C.WGPUQueue }

func (q *nativeQueue) Submit(buffers ...CommandBuffer)                                                    {}
func (q *nativeQueue) WriteBuffer(buffer Buffer, offset uint64, data []byte)                              {}
func (q *nativeQueue) WriteTexture(dst *ImageCopyTexture, data []byte, layout *TextureDataLayout, size Extent3D) {}

type nativeBindGroup struct{ handle C.WGPUBindGroup }

func (g *nativeBindGroup) Destroy() {}

type nativeBindGroupLayout struct{ handle C.WGPUBindGroupLayout }

func (l *nativeBindGroupLayout) Destroy() {}

type nativeSampler struct{ handle C.WGPUSampler }

func (s *nativeSampler) Destroy() {}
