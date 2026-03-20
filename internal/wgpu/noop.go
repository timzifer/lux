//go:build nogui

// This file provides no-op wgpu implementations for headless/CI builds.

package wgpu

import "fmt"

// CreateInstance returns a no-op instance for headless builds.
func CreateInstance() (Instance, error) {
	return &noopInstance{}, nil
}

type noopInstance struct{}

func (i *noopInstance) CreateSurface(desc *SurfaceDescriptor) Surface {
	return &noopSurface{}
}
func (i *noopInstance) RequestAdapter(opts *RequestAdapterOptions) (Adapter, error) {
	return &noopAdapter{}, nil
}
func (i *noopInstance) Destroy() {}

type noopAdapter struct{}

func (a *noopAdapter) RequestDevice(desc *DeviceDescriptor) (Device, error) {
	return &noopDevice{}, nil
}
func (a *noopAdapter) GetInfo() AdapterInfo {
	return AdapterInfo{Name: "noop", BackendType: "none"}
}

type noopDevice struct{}

func (d *noopDevice) CreateShaderModule(_ *ShaderModuleDescriptor) ShaderModule          { return &noopShaderModule{} }
func (d *noopDevice) CreateRenderPipeline(_ *RenderPipelineDescriptor) RenderPipeline    { return &noopRenderPipeline{} }
func (d *noopDevice) CreateBuffer(_ *BufferDescriptor) Buffer                            { return &noopBuffer{} }
func (d *noopDevice) CreateTexture(_ *TextureDescriptor) Texture                         { return &noopTexture{} }
func (d *noopDevice) CreateBindGroupLayout(_ *BindGroupLayoutDescriptor) BindGroupLayout  { return &noopBindGroupLayout{} }
func (d *noopDevice) CreateBindGroup(_ *BindGroupDescriptor) BindGroup                   { return &noopBindGroup{} }
func (d *noopDevice) CreateCommandEncoder() CommandEncoder                               { return &noopCommandEncoder{} }
func (d *noopDevice) CreateSampler(_ *SamplerDescriptor) Sampler                         { return &noopSampler{} }
func (d *noopDevice) GetQueue() Queue                                                    { return &noopQueue{} }
func (d *noopDevice) Destroy()                                                           {}

type noopSurface struct{}
func (s *noopSurface) Configure(Device, *SurfaceConfiguration) {}
func (s *noopSurface) GetCurrentTexture() (TextureView, error) { return &noopTextureView{}, nil }
func (s *noopSurface) Present()                                {}
func (s *noopSurface) Destroy()                                {}

type noopRenderPipeline struct{}
func (p *noopRenderPipeline) Destroy() {}

type noopBuffer struct{}
func (b *noopBuffer) Write(Queue, []byte) {}
func (b *noopBuffer) Destroy()            {}

type noopTexture struct{}
func (t *noopTexture) CreateView() TextureView                      { return &noopTextureView{} }
func (t *noopTexture) Write(Queue, []byte, uint32)                  {}
func (t *noopTexture) Destroy()                                     {}

type noopTextureView struct{}
func (v *noopTextureView) Destroy() {}

type noopShaderModule struct{}
func (m *noopShaderModule) Destroy() {}

type noopCommandEncoder struct{}
func (e *noopCommandEncoder) BeginRenderPass(*RenderPassDescriptor) RenderPass { return &noopRenderPass{} }
func (e *noopCommandEncoder) Finish() CommandBuffer                            { return nil }

type noopRenderPass struct{}
func (p *noopRenderPass) SetPipeline(RenderPipeline)                              {}
func (p *noopRenderPass) SetBindGroup(uint32, BindGroup)                          {}
func (p *noopRenderPass) SetVertexBuffer(uint32, Buffer, uint64, uint64)          {}
func (p *noopRenderPass) Draw(uint32, uint32, uint32, uint32)                     {}
func (p *noopRenderPass) DrawInstanced(uint32, uint32, uint32, uint32)            {}
func (p *noopRenderPass) SetScissorRect(uint32, uint32, uint32, uint32)           {}
func (p *noopRenderPass) End()                                                    {}

type noopQueue struct{}
func (q *noopQueue) Submit(...CommandBuffer)                                     {}
func (q *noopQueue) WriteBuffer(Buffer, uint64, []byte)                          {}
func (q *noopQueue) WriteTexture(*ImageCopyTexture, []byte, *TextureDataLayout, Extent3D) {}

type noopBindGroup struct{}
func (g *noopBindGroup) Destroy() {}

type noopBindGroupLayout struct{}
func (l *noopBindGroupLayout) Destroy() {}

type noopSampler struct{}
func (s *noopSampler) Destroy() {}

var _ = fmt.Errorf
