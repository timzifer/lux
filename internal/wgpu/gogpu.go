//go:build (gogpu || (darwin && arm64) || windows) && !nogui

// This file provides a pure-Go WebGPU implementation using the gogpu build tag.
// It implements the wgpu interfaces using github.com/gogpu/wgpu, which provides
// a pure-Go WebGPU API backed by Vulkan/Metal/D3D12 via hal/ backends.
//
// Build with: go build -tags gogpu ./...

package wgpu

import (
	"fmt"
	"log"
	"runtime"

	"github.com/gogpu/gputypes"
	gpuwgpu "github.com/gogpu/wgpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
)

// CreateInstance creates a new wgpu Instance using the pure-Go backend.
// On Windows, DX12 is preferred (naga WGSL→HLSL is more reliable than
// WGSL→SPIR-V on NVIDIA Vulkan drivers).
func CreateInstance() (Instance, error) {
	desc := &gpuwgpu.InstanceDescriptor{
		Backends: gputypes.BackendsAll,
	}
	switch runtime.GOOS {
	case "windows":
		desc.Backends = gputypes.BackendsDX12
	case "darwin":
		desc.Backends = gputypes.BackendsMetal
	}
	inst, err := gpuwgpu.CreateInstance(desc)
	if err != nil {
		return nil, fmt.Errorf("wgpu/gogpu: failed to create instance: %w", err)
	}
	return &goInstance{inst: inst}, nil
}

// --- Instance ---

type goInstance struct {
	inst *gpuwgpu.Instance
}

func (i *goInstance) CreateSurface(desc *SurfaceDescriptor) Surface {
	if desc.DRMfd >= 0 {
		// DRM/KMS: create surface via VK_KHR_display.
		s, err := i.inst.CreateDisplaySurface(desc.DRMfd, desc.DRMConnectorID)
		if err != nil {
			log.Printf("wgpu/gogpu: CreateDisplaySurface (DRM fd=%d connector=%d) failed: %v",
				desc.DRMfd, desc.DRMConnectorID, err)
			return &goSurface{}
		}
		return &goSurface{surface: s}
	}
	s, err := i.inst.CreateSurface(desc.NativeDisplay, desc.NativeHandle)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateSurface failed: %v", err)
		return &goSurface{}
	}
	return &goSurface{surface: s}
}

func (i *goInstance) RequestAdapter(opts *RequestAdapterOptions) (Adapter, error) {
	var gopts *gpuwgpu.RequestAdapterOptions
	if opts != nil {
		gopts = &gpuwgpu.RequestAdapterOptions{
			PowerPreference: mapPowerPreference(opts.PowerPreference),
		}
		if opts.CompatibleSurface != nil {
			if gs, ok := opts.CompatibleSurface.(*goSurface); ok && gs.surface != nil {
				gopts.CompatibleSurface = gs.surface
			}
		}
	}
	adapter, err := i.inst.RequestAdapter(gopts)
	if err != nil {
		return nil, fmt.Errorf("wgpu/gogpu: RequestAdapter failed: %w", err)
	}
	return &goAdapter{adapter: adapter}, nil
}

func (i *goInstance) Destroy() {
	if i.inst != nil {
		i.inst.Release()
		i.inst = nil
	}
}

// --- Adapter ---

type goAdapter struct {
	adapter *gpuwgpu.Adapter
}

func (a *goAdapter) RequestDevice(desc *DeviceDescriptor) (Device, error) {
	gdesc := &gpuwgpu.DeviceDescriptor{
		RequiredLimits: gputypes.DefaultLimits(),
	}
	if desc != nil {
		gdesc.Label = desc.Label
	}
	dev, err := a.adapter.RequestDevice(gdesc)
	if err != nil {
		return nil, fmt.Errorf("wgpu/gogpu: RequestDevice failed: %w", err)
	}
	q := dev.Queue()
	return &goDevice{device: dev, queue: q}, nil
}

func (a *goAdapter) GetInfo() AdapterInfo {
	info := a.adapter.Info()
	return AdapterInfo{
		Name:        info.Name,
		Vendor:      info.Vendor,
		DriverInfo:  info.DriverInfo,
		AdapterType: info.DeviceType.String(),
		BackendType: info.Backend.String(),
	}
}

// --- Device ---

type goDevice struct {
	device *gpuwgpu.Device
	queue  *gpuwgpu.Queue
}

func (d *goDevice) CreateShaderModule(desc *ShaderModuleDescriptor) ShaderModule {
	sm, err := d.device.CreateShaderModule(&gpuwgpu.ShaderModuleDescriptor{
		Label: desc.Label,
		WGSL:  desc.Source,
	})
	if err != nil {
		log.Printf("wgpu/gogpu: CreateShaderModule failed: %v", err)
		return &goShaderModule{}
	}
	return &goShaderModule{module: sm}
}

func (d *goDevice) CreateRenderPipeline(desc *RenderPipelineDescriptor) RenderPipeline {
	// Build PipelineLayout from BindGroupLayouts.
	var layout *gpuwgpu.PipelineLayout
	if len(desc.BindGroupLayouts) > 0 {
		bgls := make([]*gpuwgpu.BindGroupLayout, len(desc.BindGroupLayouts))
		for i, bgl := range desc.BindGroupLayouts {
			if g, ok := bgl.(*goBindGroupLayout); ok && g.layout != nil {
				bgls[i] = g.layout
			}
		}
		var err error
		layout, err = d.device.CreatePipelineLayout(&gpuwgpu.PipelineLayoutDescriptor{
			Label:            desc.Label + "-layout",
			BindGroupLayouts: bgls,
		})
		if err != nil {
			log.Printf("wgpu/gogpu: CreatePipelineLayout failed: %v", err)
			return &goRenderPipeline{}
		}
	}

	gdesc := &gpuwgpu.RenderPipelineDescriptor{
		Label:  desc.Label,
		Layout: layout,
		Vertex: gpuwgpu.VertexState{
			EntryPoint: desc.Vertex.EntryPoint,
			Buffers:    mapVertexBufferLayouts(desc.Vertex.Buffers),
		},
		Primitive: gpuwgpu.PrimitiveState{
			Topology:  mapPrimitiveTopology(desc.Primitive.Topology),
			CullMode:  mapCullMode(desc.Primitive.CullMode),
			FrontFace: mapFrontFace(desc.Primitive.FrontFace),
		},
		Multisample: gputypes.DefaultMultisampleState(),
	}
	if desc.DepthStencil != nil {
		stencilFace := gpuwgpu.StencilFaceState{
			Compare:     gputypes.CompareFunctionAlways,
			FailOp:      gpuwgpu.StencilOperationKeep,
			DepthFailOp: gpuwgpu.StencilOperationKeep,
			PassOp:      gpuwgpu.StencilOperationKeep,
		}
		gdesc.DepthStencil = &gpuwgpu.DepthStencilState{
			Format:            mapTextureFormat(desc.DepthStencil.Format),
			DepthWriteEnabled: desc.DepthStencil.DepthWriteEnabled,
			DepthCompare:      mapCompareFunction(desc.DepthStencil.DepthCompare),
			StencilFront:      stencilFace,
			StencilBack:       stencilFace,
		}
	}
	if sm, ok := desc.Vertex.Module.(*goShaderModule); ok && sm.module != nil {
		gdesc.Vertex.Module = sm.module
	}
	if desc.Fragment != nil {
		fs := &gpuwgpu.FragmentState{
			EntryPoint: desc.Fragment.EntryPoint,
			Targets:    mapColorTargets(desc.Fragment.Targets),
		}
		if sm, ok := desc.Fragment.Module.(*goShaderModule); ok && sm.module != nil {
			fs.Module = sm.module
		}
		gdesc.Fragment = fs
	}

	pipeline, err := d.device.CreateRenderPipeline(gdesc)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateRenderPipeline failed: %v", err)
		return &goRenderPipeline{}
	}
	return &goRenderPipeline{pipeline: pipeline}
}

func (d *goDevice) CreateBuffer(desc *BufferDescriptor) Buffer {
	buf, err := d.device.CreateBuffer(&gpuwgpu.BufferDescriptor{
		Label: desc.Label,
		Size:  desc.Size,
		Usage: mapBufferUsage(desc.Usage),
	})
	if err != nil {
		log.Printf("wgpu/gogpu: CreateBuffer failed: %v", err)
		return &goBuffer{queue: d.queue}
	}
	return &goBuffer{buffer: buf, queue: d.queue}
}

func (d *goDevice) CreateTexture(desc *TextureDescriptor) Texture {
	tex, err := d.device.CreateTexture(&gpuwgpu.TextureDescriptor{
		Label:         desc.Label,
		Size:          gpuwgpu.Extent3D(desc.Size),
		Format:        mapTextureFormat(desc.Format),
		Usage:         mapTextureUsage(desc.Usage),
		Dimension:     gpuwgpu.TextureDimension2D,
		MipLevelCount: 1,
		SampleCount:   1,
	})
	if err != nil {
		log.Printf("wgpu/gogpu: CreateTexture failed: %v", err)
		return &goTexture{device: d.device, queue: d.queue, size: desc.Size, format: desc.Format}
	}
	return &goTexture{texture: tex, device: d.device, queue: d.queue, size: desc.Size, format: desc.Format}
}

func (d *goDevice) CreateBindGroupLayout(desc *BindGroupLayoutDescriptor) BindGroupLayout {
	entries := make([]gpuwgpu.BindGroupLayoutEntry, len(desc.Entries))
	for i, e := range desc.Entries {
		entries[i] = gpuwgpu.BindGroupLayoutEntry{
			Binding:    e.Binding,
			Visibility: mapShaderStage(e.Visibility),
		}
		if e.Buffer != nil {
			entries[i].Buffer = &gputypes.BufferBindingLayout{
				Type: mapBufferBindingType(e.Buffer.Type),
			}
		}
		if e.Sampler != nil {
			entries[i].Sampler = &gputypes.SamplerBindingLayout{
				Type: gputypes.SamplerBindingTypeFiltering,
			}
		}
		if e.Texture != nil {
			entries[i].Texture = &gputypes.TextureBindingLayout{
				SampleType:    mapTextureSampleType(e.Texture.SampleType),
				ViewDimension: mapTextureViewDimension(e.Texture.ViewDimension),
			}
		}
		if e.StorageTexture != nil {
			entries[i].StorageTexture = &gputypes.StorageTextureBindingLayout{
				Access:        mapStorageTextureAccess(e.StorageTexture.Access),
				Format:        mapTextureFormat(e.StorageTexture.Format),
				ViewDimension: mapTextureViewDimension(e.StorageTexture.ViewDimension),
			}
		}
	}
	bgl, err := d.device.CreateBindGroupLayout(&gpuwgpu.BindGroupLayoutDescriptor{
		Label:   desc.Label,
		Entries: entries,
	})
	if err != nil {
		log.Printf("wgpu/gogpu: CreateBindGroupLayout failed: %v", err)
		return &goBindGroupLayout{}
	}
	return &goBindGroupLayout{layout: bgl}
}

func (d *goDevice) CreateBindGroup(desc *BindGroupDescriptor) BindGroup {
	entries := make([]gpuwgpu.BindGroupEntry, len(desc.Entries))
	for i, e := range desc.Entries {
		entries[i] = gpuwgpu.BindGroupEntry{
			Binding: e.Binding,
			Offset:  e.Offset,
			Size:    e.Size,
		}
		if e.Buffer != nil {
			if gb, ok := e.Buffer.(*goBuffer); ok && gb.buffer != nil {
				entries[i].Buffer = gb.buffer
			}
		}
		if e.Sampler != nil {
			if gs, ok := e.Sampler.(*goSampler); ok && gs.sampler != nil {
				entries[i].Sampler = gs.sampler
			}
		}
		if e.Texture != nil {
			if gv, ok := e.Texture.(*goTextureView); ok && gv.view != nil {
				entries[i].TextureView = gv.view
			}
		}
	}
	var bglayout *gpuwgpu.BindGroupLayout
	if gl, ok := desc.Layout.(*goBindGroupLayout); ok && gl.layout != nil {
		bglayout = gl.layout
	}
	bg, err := d.device.CreateBindGroup(&gpuwgpu.BindGroupDescriptor{
		Label:   desc.Label,
		Layout:  bglayout,
		Entries: entries,
	})
	if err != nil {
		log.Printf("wgpu/gogpu: CreateBindGroup failed: %v", err)
		return &goBindGroup{}
	}
	return &goBindGroup{group: bg}
}

func (d *goDevice) CreateCommandEncoder() CommandEncoder {
	enc, err := d.device.CreateCommandEncoder(nil)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateCommandEncoder failed: %v", err)
		return &goCommandEncoder{}
	}
	return &goCommandEncoder{encoder: enc, device: d.device}
}

func (d *goDevice) CreateSampler(desc *SamplerDescriptor) Sampler {
	gdesc := &gpuwgpu.SamplerDescriptor{
		Label:        desc.Label,
		AddressModeU: gputypes.AddressModeClampToEdge,
		AddressModeV: gputypes.AddressModeClampToEdge,
		AddressModeW: gputypes.AddressModeClampToEdge,
		MagFilter:    gputypes.FilterModeLinear,
		MinFilter:    gputypes.FilterModeLinear,
		MipmapFilter: gputypes.FilterModeLinear,
	}
	s, err := d.device.CreateSampler(gdesc)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateSampler failed: %v", err)
		return &goSampler{}
	}
	return &goSampler{sampler: s}
}

func (d *goDevice) CreateComputePipeline(desc *ComputePipelineDescriptor) ComputePipeline {
	var layout *gpuwgpu.PipelineLayout
	if len(desc.BindGroupLayouts) > 0 {
		bgls := make([]*gpuwgpu.BindGroupLayout, len(desc.BindGroupLayouts))
		for i, bgl := range desc.BindGroupLayouts {
			if g, ok := bgl.(*goBindGroupLayout); ok && g.layout != nil {
				bgls[i] = g.layout
			}
		}
		var err error
		layout, err = d.device.CreatePipelineLayout(&gpuwgpu.PipelineLayoutDescriptor{
			Label:            desc.Label + "-layout",
			BindGroupLayouts: bgls,
		})
		if err != nil {
			log.Printf("wgpu/gogpu: CreatePipelineLayout (compute) failed: %v", err)
			return &goComputePipeline{}
		}
	}
	gdesc := &gpuwgpu.ComputePipelineDescriptor{
		Label:      desc.Label,
		Layout:     layout,
		EntryPoint: desc.EntryPoint,
	}
	if sm, ok := desc.Module.(*goShaderModule); ok && sm.module != nil {
		gdesc.Module = sm.module
	}
	pipeline, err := d.device.CreateComputePipeline(gdesc)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateComputePipeline failed: %v", err)
		return &goComputePipeline{}
	}
	return &goComputePipeline{pipeline: pipeline}
}

func (d *goDevice) GetQueue() Queue {
	return &goQueue{queue: d.queue}
}

func (d *goDevice) Destroy() {
	if d.device != nil {
		d.device.Release()
		d.device = nil
	}
}

// --- Surface ---

type goSurface struct {
	surface        *gpuwgpu.Surface
	currentTexture *gpuwgpu.SurfaceTexture
	currentView    *gpuwgpu.TextureView
	device         *gpuwgpu.Device
	loggedSubopt   bool
}

func (s *goSurface) Configure(device Device, config *SurfaceConfiguration) {
	if s.surface == nil {
		return
	}
	gd, ok := device.(*goDevice)
	if !ok || gd.device == nil {
		return
	}
	s.device = gd.device
	alphaMode := gputypes.CompositeAlphaModeAuto
	if config.AlphaMode == CompositeAlphaModeOpaque {
		alphaMode = gputypes.CompositeAlphaModeOpaque
	}
	err := s.surface.Configure(gd.device, &gpuwgpu.SurfaceConfiguration{
		Format:      mapTextureFormat(config.Format),
		Usage:       mapTextureUsage(config.Usage),
		Width:       config.Width,
		Height:      config.Height,
		PresentMode: mapPresentMode(config.PresentMode),
		AlphaMode:   alphaMode,
	})
	if err != nil {
		log.Printf("wgpu/gogpu: Surface.Configure failed: %v", err)
	}
}

func (s *goSurface) GetCurrentTexture() (TextureView, error) {
	if s.surface == nil {
		return &goTextureView{}, fmt.Errorf("wgpu/gogpu: surface is nil")
	}
	st, ok, err := s.surface.GetCurrentTexture()
	if err != nil {
		return &goTextureView{}, fmt.Errorf("wgpu/gogpu: GetCurrentTexture failed: %w", err)
	}
	if !ok && !s.loggedSubopt {
		// ok=false means "suboptimal" — texture is still valid, surface should be reconfigured.
		log.Printf("wgpu/gogpu: surface suboptimal, consider reconfigure")
		s.loggedSubopt = true
	}
	if st == nil {
		return &goTextureView{}, fmt.Errorf("wgpu/gogpu: GetCurrentTexture returned nil texture")
	}
	s.currentTexture = st
	view, err := st.CreateView(nil)
	if err != nil {
		// Discard the acquired texture so the surface doesn't stay locked.
		s.surface.DiscardTexture()
		s.currentTexture = nil
		return &goTextureView{}, fmt.Errorf("wgpu/gogpu: SurfaceTexture.CreateView failed: %w", err)
	}
	s.currentView = view
	return &goTextureView{view: view}, nil
}

func (s *goSurface) Present() {
	if s.surface == nil || s.currentTexture == nil {
		return
	}
	if err := s.surface.Present(s.currentTexture); err != nil {
		log.Printf("wgpu/gogpu: Surface.Present failed: %v", err)
	}
	s.currentTexture = nil
	s.currentView = nil
}

func (s *goSurface) Destroy() {
	if s.surface != nil {
		s.surface.Release()
		s.surface = nil
	}
}

// --- RenderPipeline ---

type goRenderPipeline struct {
	pipeline *gpuwgpu.RenderPipeline
}

func (p *goRenderPipeline) Destroy() {
	if p.pipeline != nil {
		p.pipeline.Release()
		p.pipeline = nil
	}
}

// --- ComputePipeline ---

type goComputePipeline struct {
	pipeline *gpuwgpu.ComputePipeline
}

func (p *goComputePipeline) Destroy() {
	if p.pipeline != nil {
		p.pipeline.Release()
		p.pipeline = nil
	}
}

// --- ComputePass ---

type goComputePass struct {
	pass *gpuwgpu.ComputePassEncoder
}

func (p *goComputePass) SetPipeline(pipeline ComputePipeline) {
	if p.pass == nil {
		return
	}
	if gp, ok := pipeline.(*goComputePipeline); ok && gp.pipeline != nil {
		p.pass.SetPipeline(gp.pipeline)
	}
}

func (p *goComputePass) SetBindGroup(index uint32, group BindGroup) {
	if p.pass == nil {
		return
	}
	if gb, ok := group.(*goBindGroup); ok && gb.group != nil {
		p.pass.SetBindGroup(index, gb.group, nil)
	}
}

func (p *goComputePass) Dispatch(x, y, z uint32) {
	if p.pass == nil {
		return
	}
	p.pass.Dispatch(x, y, z)
}

func (p *goComputePass) End() {
	if p.pass == nil {
		return
	}
	if err := p.pass.End(); err != nil {
		log.Printf("wgpu/gogpu: ComputePass.End failed: %v", err)
	}
}

// --- Buffer ---

type goBuffer struct {
	buffer *gpuwgpu.Buffer
	queue  *gpuwgpu.Queue
}

func (b *goBuffer) Write(queue Queue, data []byte) {
	if b.buffer == nil || b.queue == nil {
		return
	}
	if err := b.queue.WriteBuffer(b.buffer, 0, data); err != nil {
		log.Printf("wgpu/gogpu: Queue.WriteBuffer failed: %v", err)
	}
}

func (b *goBuffer) Destroy() {
	if b.buffer != nil {
		b.buffer.Release()
		b.buffer = nil
	}
}

// --- Texture ---

type goTexture struct {
	texture *gpuwgpu.Texture
	device  *gpuwgpu.Device
	queue   *gpuwgpu.Queue
	size    Extent3D
	format  TextureFormat
}

func (t *goTexture) CreateView() TextureView {
	if t.texture == nil || t.device == nil {
		return &goTextureView{}
	}
	view, err := t.device.CreateTextureView(t.texture, nil)
	if err != nil {
		log.Printf("wgpu/gogpu: CreateTextureView failed: %v", err)
		return &goTextureView{}
	}
	return &goTextureView{view: view}
}

func (t *goTexture) Write(queue Queue, data []byte, bytesPerRow uint32) {
	if t.texture == nil || t.queue == nil {
		return
	}
	dst := &gpuwgpu.ImageCopyTexture{
		Texture:  t.texture,
		MipLevel: 0,
	}
	layout := &gpuwgpu.ImageDataLayout{
		Offset:       0,
		BytesPerRow:  bytesPerRow,
		RowsPerImage: t.size.Height,
	}
	size := &gpuwgpu.Extent3D{
		Width:              t.size.Width,
		Height:             t.size.Height,
		DepthOrArrayLayers: 1,
	}
	if err := t.queue.WriteTexture(dst, data, layout, size); err != nil {
		log.Printf("wgpu/gogpu: Queue.WriteTexture failed: %v", err)
	}
}

func (t *goTexture) Destroy() {
	if t.texture != nil {
		t.texture.Release()
		t.texture = nil
	}
}

// --- TextureView ---

type goTextureView struct {
	view *gpuwgpu.TextureView
}

func (v *goTextureView) Destroy() {
	if v.view != nil {
		v.view.Release()
		v.view = nil
	}
}

// --- ShaderModule ---

type goShaderModule struct {
	module *gpuwgpu.ShaderModule
}

func (m *goShaderModule) Destroy() {
	if m.module != nil {
		m.module.Release()
		m.module = nil
	}
}

// --- CommandEncoder ---

type goCommandEncoder struct {
	encoder *gpuwgpu.CommandEncoder
	device  *gpuwgpu.Device
}

func (e *goCommandEncoder) BeginRenderPass(desc *RenderPassDescriptor) RenderPass {
	if e.encoder == nil {
		return &goRenderPass{}
	}
	colorAttachments := make([]gpuwgpu.RenderPassColorAttachment, len(desc.ColorAttachments))
	for i, ca := range desc.ColorAttachments {
		colorAttachments[i] = gpuwgpu.RenderPassColorAttachment{
			LoadOp:     mapLoadOp(ca.LoadOp),
			StoreOp:    mapStoreOp(ca.StoreOp),
			ClearValue: gpuwgpu.Color(ca.ClearValue),
		}
		if gv, ok := ca.View.(*goTextureView); ok && gv.view != nil {
			colorAttachments[i].View = gv.view
		}
	}
	rpDesc := &gpuwgpu.RenderPassDescriptor{
		ColorAttachments: colorAttachments,
	}
	if desc.DepthStencilAttachment != nil {
		dsa := &gpuwgpu.RenderPassDepthStencilAttachment{
			DepthLoadOp:     mapLoadOp(desc.DepthStencilAttachment.DepthLoadOp),
			DepthStoreOp:    mapStoreOp(desc.DepthStencilAttachment.DepthStoreOp),
			DepthClearValue: desc.DepthStencilAttachment.DepthClearValue,
		}
		if gv, ok := desc.DepthStencilAttachment.View.(*goTextureView); ok && gv.view != nil {
			dsa.View = gv.view
		}
		rpDesc.DepthStencilAttachment = dsa
	}
	rp, err := e.encoder.BeginRenderPass(rpDesc)
	if err != nil {
		log.Printf("wgpu/gogpu: BeginRenderPass failed: %v", err)
		return &goRenderPass{}
	}
	return &goRenderPass{pass: rp}
}

func (e *goCommandEncoder) Finish() CommandBuffer {
	if e.encoder == nil {
		return &goCommandBuffer{}
	}
	cb, err := e.encoder.Finish()
	if err != nil {
		log.Printf("wgpu/gogpu: CommandEncoder.Finish failed: %v", err)
		return &goCommandBuffer{}
	}
	return &goCommandBuffer{buffer: cb}
}

func (e *goCommandEncoder) BeginComputePass() ComputePass {
	if e.encoder == nil {
		return &goComputePass{}
	}
	cp, err := e.encoder.BeginComputePass(nil)
	if err != nil {
		log.Printf("wgpu/gogpu: BeginComputePass failed: %v", err)
		return &goComputePass{}
	}
	return &goComputePass{pass: cp}
}

func (e *goCommandEncoder) CopyTextureToTexture(src, dst *ImageCopyTexture, size Extent3D) {
	if e.encoder == nil {
		return
	}
	// gogpu's high-level CommandEncoder does not expose CopyTextureToTexture yet.
	// Log and skip; callers should use an alternative path if needed.
	log.Printf("wgpu/gogpu: CopyTextureToTexture not yet supported by gogpu high-level API")
}

// --- RenderPass ---

type goRenderPass struct {
	pass *gpuwgpu.RenderPassEncoder
}

func (p *goRenderPass) SetPipeline(pipeline RenderPipeline) {
	if p.pass == nil {
		return
	}
	if gp, ok := pipeline.(*goRenderPipeline); ok && gp.pipeline != nil {
		p.pass.SetPipeline(gp.pipeline)
	}
}

func (p *goRenderPass) SetBindGroup(index uint32, group BindGroup) {
	if p.pass == nil {
		return
	}
	if gb, ok := group.(*goBindGroup); ok && gb.group != nil {
		p.pass.SetBindGroup(index, gb.group, nil)
	}
}

func (p *goRenderPass) SetVertexBuffer(slot uint32, buffer Buffer, offset, size uint64) {
	if p.pass == nil {
		return
	}
	if gb, ok := buffer.(*goBuffer); ok && gb.buffer != nil {
		// gogpu SetVertexBuffer does not take a size parameter.
		p.pass.SetVertexBuffer(slot, gb.buffer, offset)
	}
}

func (p *goRenderPass) SetIndexBuffer(buffer Buffer, format IndexFormat, offset, size uint64) {
	if p.pass == nil {
		return
	}
	if gb, ok := buffer.(*goBuffer); ok && gb.buffer != nil {
		f := gputypes.IndexFormatUint16
		if format == IndexFormatUint32 {
			f = gputypes.IndexFormatUint32
		}
		p.pass.SetIndexBuffer(gb.buffer, f, offset)
	}
}

func (p *goRenderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	if p.pass == nil {
		return
	}
	p.pass.Draw(vertexCount, instanceCount, firstVertex, firstInstance)
}

func (p *goRenderPass) DrawInstanced(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	if p.pass == nil {
		return
	}
	p.pass.Draw(vertexCount, instanceCount, firstVertex, firstInstance)
}

func (p *goRenderPass) DrawIndexed(indexCount, instanceCount, firstIndex, baseVertex int32, firstInstance uint32) {
	if p.pass == nil {
		return
	}
	p.pass.DrawIndexed(uint32(indexCount), uint32(instanceCount), uint32(firstIndex), baseVertex, firstInstance)
}

func (p *goRenderPass) SetScissorRect(x, y, width, height uint32) {
	if p.pass == nil {
		return
	}
	p.pass.SetScissorRect(x, y, width, height)
}

func (p *goRenderPass) End() {
	if p.pass == nil {
		return
	}
	if err := p.pass.End(); err != nil {
		log.Printf("wgpu/gogpu: RenderPass.End failed: %v", err)
	}
}

// --- CommandBuffer ---

type goCommandBuffer struct {
	buffer *gpuwgpu.CommandBuffer
}

// --- Queue ---

type goQueue struct {
	queue *gpuwgpu.Queue
}

func (q *goQueue) Submit(buffers ...CommandBuffer) {
	if q.queue == nil {
		return
	}
	cbs := make([]*gpuwgpu.CommandBuffer, 0, len(buffers))
	for _, b := range buffers {
		if gb, ok := b.(*goCommandBuffer); ok && gb.buffer != nil {
			cbs = append(cbs, gb.buffer)
		}
	}
	if err := q.queue.Submit(cbs...); err != nil {
		log.Printf("wgpu/gogpu: Queue.Submit failed: %v", err)
	}
}

func (q *goQueue) WriteBuffer(buffer Buffer, offset uint64, data []byte) {
	if q.queue == nil {
		return
	}
	if gb, ok := buffer.(*goBuffer); ok && gb.buffer != nil {
		if err := q.queue.WriteBuffer(gb.buffer, offset, data); err != nil {
			log.Printf("wgpu/gogpu: Queue.WriteBuffer failed: %v", err)
		}
	}
}

func (q *goQueue) WriteTexture(dst *ImageCopyTexture, data []byte, layout *TextureDataLayout, size Extent3D) {
	if q.queue == nil {
		return
	}
	gt, ok := dst.Texture.(*goTexture)
	if !ok || gt.texture == nil {
		return
	}
	gdst := &gpuwgpu.ImageCopyTexture{
		Texture:  gt.texture,
		MipLevel: dst.MipLevel,
	}
	glayout := &gpuwgpu.ImageDataLayout{
		Offset:       layout.Offset,
		BytesPerRow:  layout.BytesPerRow,
		RowsPerImage: layout.RowsPerImage,
	}
	gsize := &gpuwgpu.Extent3D{
		Width:              size.Width,
		Height:             size.Height,
		DepthOrArrayLayers: size.DepthOrArrayLayers,
	}
	if err := q.queue.WriteTexture(gdst, data, glayout, gsize); err != nil {
		log.Printf("wgpu/gogpu: Queue.WriteTexture failed: %v", err)
	}
}

// --- BindGroup ---

type goBindGroup struct {
	group *gpuwgpu.BindGroup
}

func (g *goBindGroup) Destroy() {
	if g.group != nil {
		g.group.Release()
		g.group = nil
	}
}

// --- BindGroupLayout ---

type goBindGroupLayout struct {
	layout *gpuwgpu.BindGroupLayout
}

func (l *goBindGroupLayout) Destroy() {
	if l.layout != nil {
		l.layout.Release()
		l.layout = nil
	}
}

// --- Sampler ---

type goSampler struct {
	sampler *gpuwgpu.Sampler
}

func (s *goSampler) Destroy() {
	if s.sampler != nil {
		s.sampler.Release()
		s.sampler = nil
	}
}

// --- Enum mapping functions ---

func mapTextureFormat(f TextureFormat) gpuwgpu.TextureFormat {
	switch f {
	case TextureFormatBGRA8Unorm:
		return gpuwgpu.TextureFormatBGRA8Unorm
	case TextureFormatRGBA8Unorm:
		return gpuwgpu.TextureFormatRGBA8Unorm
	case TextureFormatR8Unorm:
		return gputypes.TextureFormatR8Unorm
	case TextureFormatDepth24Plus:
		return gputypes.TextureFormatDepth24Plus
	default:
		return gpuwgpu.TextureFormatRGBA8Unorm
	}
}

func mapBlendFactor(f BlendFactor) gputypes.BlendFactor {
	switch f {
	case BlendFactorZero:
		return gputypes.BlendFactorZero
	case BlendFactorOne:
		return gputypes.BlendFactorOne
	case BlendFactorSrcAlpha:
		return gputypes.BlendFactorSrcAlpha
	case BlendFactorOneMinusSrcAlpha:
		return gputypes.BlendFactorOneMinusSrcAlpha
	default:
		return gputypes.BlendFactorOne
	}
}

func mapBlendOp(op BlendOperation) gputypes.BlendOperation {
	switch op {
	case BlendOperationAdd:
		return gputypes.BlendOperationAdd
	default:
		return gputypes.BlendOperationAdd
	}
}

func mapLoadOp(op LoadOp) gputypes.LoadOp {
	switch op {
	case LoadOpClear:
		return gputypes.LoadOpClear
	case LoadOpLoad:
		return gputypes.LoadOpLoad
	default:
		return gputypes.LoadOpClear
	}
}

func mapStoreOp(op StoreOp) gputypes.StoreOp {
	switch op {
	case StoreOpStore:
		return gputypes.StoreOpStore
	case StoreOpDiscard:
		return gputypes.StoreOpDiscard
	default:
		return gputypes.StoreOpStore
	}
}

func mapVertexFormat(f VertexFormat) gputypes.VertexFormat {
	switch f {
	case VertexFormatFloat32x2:
		return gputypes.VertexFormatFloat32x2
	case VertexFormatFloat32x4:
		return gputypes.VertexFormatFloat32x4
	case VertexFormatFloat32:
		return gputypes.VertexFormatFloat32
	case VertexFormatFloat32x3:
		return gputypes.VertexFormatFloat32x3
	default:
		return gputypes.VertexFormatFloat32x4
	}
}

func mapPrimitiveTopology(t PrimitiveTopology) gputypes.PrimitiveTopology {
	switch t {
	case PrimitiveTopologyTriangleList:
		return gputypes.PrimitiveTopologyTriangleList
	case PrimitiveTopologyTriangleStrip:
		return gputypes.PrimitiveTopologyTriangleStrip
	case PrimitiveTopologyLineList:
		return gputypes.PrimitiveTopologyLineList
	case PrimitiveTopologyPointList:
		return gputypes.PrimitiveTopologyPointList
	default:
		return gputypes.PrimitiveTopologyTriangleList
	}
}

func mapPresentMode(m PresentMode) gpuwgpu.PresentMode {
	switch m {
	case PresentModeFifo:
		return gpuwgpu.PresentModeFifo
	case PresentModeImmediate:
		return gpuwgpu.PresentModeImmediate
	case PresentModeMailbox:
		return gpuwgpu.PresentModeMailbox
	default:
		return gpuwgpu.PresentModeFifo
	}
}

func mapTextureUsage(u TextureUsage) gpuwgpu.TextureUsage {
	var result gpuwgpu.TextureUsage
	if u&TextureUsageCopySrc != 0 {
		result |= gpuwgpu.TextureUsageCopySrc
	}
	if u&TextureUsageCopyDst != 0 {
		result |= gpuwgpu.TextureUsageCopyDst
	}
	if u&TextureUsageTextureBinding != 0 {
		result |= gpuwgpu.TextureUsageTextureBinding
	}
	if u&TextureUsageRenderAttachment != 0 {
		result |= gpuwgpu.TextureUsageRenderAttachment
	}
	if u&TextureUsageStorageBinding != 0 {
		result |= gpuwgpu.TextureUsageStorageBinding
	}
	return result
}

func mapBufferUsage(u BufferUsage) gpuwgpu.BufferUsage {
	var result gpuwgpu.BufferUsage
	if u&BufferUsageVertex != 0 {
		result |= gpuwgpu.BufferUsageVertex
	}
	if u&BufferUsageIndex != 0 {
		result |= gpuwgpu.BufferUsageIndex
	}
	if u&BufferUsageUniform != 0 {
		result |= gpuwgpu.BufferUsageUniform
	}
	if u&BufferUsageCopySrc != 0 {
		result |= gpuwgpu.BufferUsageCopySrc
	}
	if u&BufferUsageCopyDst != 0 {
		result |= gpuwgpu.BufferUsageCopyDst
	}
	return result
}

func mapPowerPreference(p PowerPreference) gpuwgpu.PowerPreference {
	switch p {
	case PowerPreferenceLowPower:
		return gpuwgpu.PowerPreferenceLowPower
	case PowerPreferenceHighPerformance:
		return gpuwgpu.PowerPreferenceHighPerformance
	default:
		return gpuwgpu.PowerPreferenceNone
	}
}

func mapShaderStage(s ShaderStage) gputypes.ShaderStage {
	var result gputypes.ShaderStage
	if s&ShaderStageVertex != 0 {
		result |= gputypes.ShaderStageVertex
	}
	if s&ShaderStageFragment != 0 {
		result |= gputypes.ShaderStageFragment
	}
	if s&ShaderStageCompute != 0 {
		result |= gputypes.ShaderStageCompute
	}
	return result
}

func mapBufferBindingType(t BufferBindingType) gputypes.BufferBindingType {
	switch t {
	case BufferBindingTypeUniform:
		return gputypes.BufferBindingTypeUniform
	case BufferBindingTypeStorage:
		return gputypes.BufferBindingTypeStorage
	default:
		return gputypes.BufferBindingTypeUniform
	}
}

func mapTextureSampleType(t TextureSampleType) gputypes.TextureSampleType {
	switch t {
	case TextureSampleTypeFloat:
		return gputypes.TextureSampleTypeFloat
	default:
		return gputypes.TextureSampleTypeFloat
	}
}

func mapTextureViewDimension(d TextureViewDimension) gputypes.TextureViewDimension {
	switch d {
	case TextureViewDimension2D:
		return gputypes.TextureViewDimension2D
	default:
		return gputypes.TextureViewDimension2D
	}
}

func mapStorageTextureAccess(a StorageTextureAccess) gputypes.StorageTextureAccess {
	switch a {
	case StorageTextureAccessWriteOnly:
		return gputypes.StorageTextureAccessWriteOnly
	case StorageTextureAccessReadOnly:
		return gputypes.StorageTextureAccessReadOnly
	case StorageTextureAccessReadWrite:
		return gputypes.StorageTextureAccessReadWrite
	default:
		return gputypes.StorageTextureAccessWriteOnly
	}
}

func mapVertexStepMode(m VertexStepMode) gputypes.VertexStepMode {
	switch m {
	case VertexStepModeVertex:
		return gputypes.VertexStepModeVertex
	case VertexStepModeInstance:
		return gputypes.VertexStepModeInstance
	default:
		return gputypes.VertexStepModeVertex
	}
}

// --- Helper functions for struct mapping ---

func mapVertexBufferLayouts(layouts []VertexBufferLayout) []gpuwgpu.VertexBufferLayout {
	if len(layouts) == 0 {
		return nil
	}
	result := make([]gpuwgpu.VertexBufferLayout, len(layouts))
	for i, l := range layouts {
		attrs := make([]gputypes.VertexAttribute, len(l.Attributes))
		for j, a := range l.Attributes {
			attrs[j] = gputypes.VertexAttribute{
				Format:         mapVertexFormat(a.Format),
				Offset:         a.Offset,
				ShaderLocation: a.ShaderLocation,
			}
		}
		result[i] = gpuwgpu.VertexBufferLayout{
			ArrayStride: l.ArrayStride,
			StepMode:    mapVertexStepMode(l.StepMode),
			Attributes:  attrs,
		}
	}
	return result
}

func mapColorTargets(targets []ColorTargetState) []gpuwgpu.ColorTargetState {
	if len(targets) == 0 {
		return nil
	}
	result := make([]gpuwgpu.ColorTargetState, len(targets))
	for i, t := range targets {
		result[i] = gpuwgpu.ColorTargetState{
			Format:    mapTextureFormat(t.Format),
			WriteMask: gputypes.ColorWriteMaskAll,
		}
		if t.Blend != nil {
			result[i].Blend = &gputypes.BlendState{
				Color: gputypes.BlendComponent{
					SrcFactor: mapBlendFactor(t.Blend.Color.SrcFactor),
					DstFactor: mapBlendFactor(t.Blend.Color.DstFactor),
					Operation: mapBlendOp(t.Blend.Color.Operation),
				},
				Alpha: gputypes.BlendComponent{
					SrcFactor: mapBlendFactor(t.Blend.Alpha.SrcFactor),
					DstFactor: mapBlendFactor(t.Blend.Alpha.DstFactor),
					Operation: mapBlendOp(t.Blend.Alpha.Operation),
				},
			}
		}
	}
	return result
}

func mapCompareFunction(f CompareFunction) gputypes.CompareFunction {
	switch f {
	case CompareFunctionNever:
		return gputypes.CompareFunctionNever
	case CompareFunctionLess:
		return gputypes.CompareFunctionLess
	case CompareFunctionEqual:
		return gputypes.CompareFunctionEqual
	case CompareFunctionLessEqual:
		return gputypes.CompareFunctionLessEqual
	case CompareFunctionGreater:
		return gputypes.CompareFunctionGreater
	case CompareFunctionNotEqual:
		return gputypes.CompareFunctionNotEqual
	case CompareFunctionGreaterEqual:
		return gputypes.CompareFunctionGreaterEqual
	case CompareFunctionAlways:
		return gputypes.CompareFunctionAlways
	default:
		return gputypes.CompareFunctionAlways
	}
}

func mapCullMode(m CullMode) gputypes.CullMode {
	switch m {
	case CullModeFront:
		return gputypes.CullModeFront
	case CullModeBack:
		return gputypes.CullModeBack
	default:
		return gputypes.CullModeNone
	}
}

func mapFrontFace(f FrontFace) gputypes.FrontFace {
	switch f {
	case FrontFaceCW:
		return gputypes.FrontFaceCW
	default:
		return gputypes.FrontFaceCCW
	}
}
