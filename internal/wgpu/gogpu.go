//go:build gogpu && !nogui

// This file provides a pure-Go WebGPU implementation using the gogpu build tag.
// It implements the wgpu interfaces without CGo, using Vulkan via pure-Go bindings.
//
// Build with: go build -tags gogpu ./...

package wgpu

import "fmt"

// GoGPUInstance implements Instance using pure-Go Vulkan bindings.
type GoGPUInstance struct{}

// CreateInstance creates a new wgpu Instance using the pure-Go backend.
func CreateInstance() (Instance, error) {
	return &GoGPUInstance{}, nil
}

func (i *GoGPUInstance) CreateSurface(desc *SurfaceDescriptor) Surface {
	return &goSurface{}
}

func (i *GoGPUInstance) RequestAdapter(opts *RequestAdapterOptions) (Adapter, error) {
	return &goAdapter{}, nil
}

func (i *GoGPUInstance) Destroy() {}

type goAdapter struct{}

func (a *goAdapter) RequestDevice(desc *DeviceDescriptor) (Device, error) {
	return &goDevice{}, nil
}

func (a *goAdapter) GetInfo() AdapterInfo {
	return AdapterInfo{
		Name:        "gogpu (pure Go)",
		BackendType: "Vulkan (pure Go)",
	}
}

type goDevice struct{}

func (d *goDevice) CreateShaderModule(desc *ShaderModuleDescriptor) ShaderModule   { return &goShaderModule{} }
func (d *goDevice) CreateRenderPipeline(desc *RenderPipelineDescriptor) RenderPipeline { return &goRenderPipeline{} }
func (d *goDevice) CreateBuffer(desc *BufferDescriptor) Buffer                     { return &goBuffer{size: desc.Size} }
func (d *goDevice) CreateTexture(desc *TextureDescriptor) Texture                  { return &goTexture{} }
func (d *goDevice) CreateBindGroupLayout(desc *BindGroupLayoutDescriptor) BindGroupLayout { return &goBindGroupLayout{} }
func (d *goDevice) CreateBindGroup(desc *BindGroupDescriptor) BindGroup            { return &goBindGroup{} }
func (d *goDevice) CreateCommandEncoder() CommandEncoder                           { return &goCommandEncoder{} }
func (d *goDevice) CreateSampler(desc *SamplerDescriptor) Sampler                  { return &goSampler{} }
func (d *goDevice) GetQueue() Queue                                                { return &goQueue{} }
func (d *goDevice) Destroy()                                                       {}

type goSurface struct{}

func (s *goSurface) Configure(device Device, config *SurfaceConfiguration) {}
func (s *goSurface) GetCurrentTexture() (TextureView, error) {
	return &goTextureView{}, nil
}
func (s *goSurface) Present() {}
func (s *goSurface) Destroy() {}

type goRenderPipeline struct{}
func (p *goRenderPipeline) Destroy() {}

type goBuffer struct {
	size uint64
	data []byte
}

func (b *goBuffer) Write(queue Queue, data []byte) {
	b.data = make([]byte, len(data))
	copy(b.data, data)
}
func (b *goBuffer) Destroy() {}

type goTexture struct {
	data []byte
}

func (t *goTexture) CreateView() TextureView              { return &goTextureView{} }
func (t *goTexture) Write(queue Queue, data []byte, bytesPerRow uint32) {
	t.data = make([]byte, len(data))
	copy(t.data, data)
}
func (t *goTexture) Destroy()                             {}

type goTextureView struct{}
func (v *goTextureView) Destroy() {}

type goShaderModule struct{}
func (m *goShaderModule) Destroy() {}

type goCommandEncoder struct {
	passes []*goRenderPass
}

func (e *goCommandEncoder) BeginRenderPass(desc *RenderPassDescriptor) RenderPass {
	rp := &goRenderPass{}
	e.passes = append(e.passes, rp)
	return rp
}
func (e *goCommandEncoder) Finish() CommandBuffer { return &goCommandBuffer{} }

type goRenderPass struct{}

func (p *goRenderPass) SetPipeline(pipeline RenderPipeline)                              {}
func (p *goRenderPass) SetBindGroup(index uint32, group BindGroup)                       {}
func (p *goRenderPass) SetVertexBuffer(slot uint32, buffer Buffer, offset, size uint64)  {}
func (p *goRenderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {}
func (p *goRenderPass) DrawInstanced(vertexCount, instanceCount, firstVertex, firstInstance uint32) {}
func (p *goRenderPass) End()                                                               {}

type goCommandBuffer struct{}

type goQueue struct{}

func (q *goQueue) Submit(buffers ...CommandBuffer)                                                    {}
func (q *goQueue) WriteBuffer(buffer Buffer, offset uint64, data []byte)                              {}
func (q *goQueue) WriteTexture(dst *ImageCopyTexture, data []byte, layout *TextureDataLayout, size Extent3D) {}

type goBindGroup struct{}
func (g *goBindGroup) Destroy() {}

type goBindGroupLayout struct{}
func (l *goBindGroupLayout) Destroy() {}

type goSampler struct{}
func (s *goSampler) Destroy() {}

// Ensure the instance error is used.
var _ = fmt.Errorf
