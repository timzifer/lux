// Package wgpu provides a Go abstraction over WebGPU for cross-platform GPU rendering (RFC §6.1).
//
// Two implementations are available:
//   - wgpu-native (CGo, default): uses the C wgpu-native library
//   - gogpu (pure Go, build tag "gogpu"): pure Go implementation
package wgpu

// TextureFormat describes the format of a texture.
type TextureFormat uint32

const (
	TextureFormatBGRA8Unorm TextureFormat = iota
	TextureFormatRGBA8Unorm
	TextureFormatR8Unorm
	TextureFormatDepth24Plus
)

// PrimitiveTopology describes how vertices form primitives.
type PrimitiveTopology uint32

const (
	PrimitiveTopologyTriangleList PrimitiveTopology = iota
	PrimitiveTopologyTriangleStrip
	PrimitiveTopologyLineList
	PrimitiveTopologyPointList
)

// BlendFactor describes a blend factor for color blending.
type BlendFactor uint32

const (
	BlendFactorZero BlendFactor = iota
	BlendFactorOne
	BlendFactorSrcAlpha
	BlendFactorOneMinusSrcAlpha
)

// BlendOperation describes a blend operation.
type BlendOperation uint32

const (
	BlendOperationAdd BlendOperation = iota
)

// LoadOp describes what to do with the previous contents of an attachment.
type LoadOp uint32

const (
	LoadOpClear LoadOp = iota
	LoadOpLoad
)

// StoreOp describes what to do with the rendered contents.
type StoreOp uint32

const (
	StoreOpStore StoreOp = iota
	StoreOpDiscard
)

// BufferUsage describes how a buffer will be used.
type BufferUsage uint32

const (
	BufferUsageVertex  BufferUsage = 1 << iota
	BufferUsageIndex
	BufferUsageUniform
	BufferUsageCopySrc
	BufferUsageCopyDst
)

// Color represents an RGBA color with float32 components.
type Color struct {
	R, G, B, A float64
}

// Extent3D describes a 3D extent.
type Extent3D struct {
	Width, Height, DepthOrArrayLayers uint32
}

// Instance is the entry point to the wgpu API.
type Instance interface {
	// CreateSurface creates a surface from a platform-specific handle.
	CreateSurface(desc *SurfaceDescriptor) Surface

	// RequestAdapter requests a GPU adapter.
	RequestAdapter(opts *RequestAdapterOptions) (Adapter, error)

	// Destroy releases the instance.
	Destroy()
}

// SurfaceDescriptor describes a surface to create.
type SurfaceDescriptor struct {
	// NativeHandle is a platform-specific window handle:
	//   - Windows: HWND
	//   - macOS: CAMetalLayer*
	//   - Linux/X11: (Display*, Window) encoded
	//   - Linux/Wayland: (wl_display*, wl_surface*) encoded
	NativeHandle uintptr
	// NativeDisplay is the display handle (X11 Display* or Wayland wl_display*).
	NativeDisplay uintptr
}

// RequestAdapterOptions configures adapter selection.
type RequestAdapterOptions struct {
	CompatibleSurface Surface
	PowerPreference   PowerPreference
}

// PowerPreference selects GPU power/performance trade-off.
type PowerPreference uint32

const (
	PowerPreferenceLowPower PowerPreference = iota
	PowerPreferenceHighPerformance
)

// Adapter represents a GPU adapter (physical device).
type Adapter interface {
	// RequestDevice requests a logical device from this adapter.
	RequestDevice(desc *DeviceDescriptor) (Device, error)

	// GetInfo returns adapter information.
	GetInfo() AdapterInfo
}

// DeviceDescriptor describes a device to create.
type DeviceDescriptor struct {
	Label string
}

// AdapterInfo contains information about an adapter.
type AdapterInfo struct {
	Name          string
	Vendor        string
	DriverInfo    string
	AdapterType   string
	BackendType   string
}

// Device represents a logical GPU device.
type Device interface {
	// CreateShaderModule creates a shader module from WGSL source.
	CreateShaderModule(desc *ShaderModuleDescriptor) ShaderModule

	// CreateRenderPipeline creates a render pipeline.
	CreateRenderPipeline(desc *RenderPipelineDescriptor) RenderPipeline

	// CreateBuffer creates a GPU buffer.
	CreateBuffer(desc *BufferDescriptor) Buffer

	// CreateTexture creates a GPU texture.
	CreateTexture(desc *TextureDescriptor) Texture

	// CreateBindGroupLayout creates a bind group layout.
	CreateBindGroupLayout(desc *BindGroupLayoutDescriptor) BindGroupLayout

	// CreateBindGroup creates a bind group.
	CreateBindGroup(desc *BindGroupDescriptor) BindGroup

	// CreateCommandEncoder creates a command encoder.
	CreateCommandEncoder() CommandEncoder

	// CreateSampler creates a texture sampler.
	CreateSampler(desc *SamplerDescriptor) Sampler

	// GetQueue returns the device's command queue.
	GetQueue() Queue

	// Destroy releases the device.
	Destroy()
}

// ShaderModuleDescriptor describes a shader module.
type ShaderModuleDescriptor struct {
	Label  string
	Source string // WGSL source code
}

// RenderPipelineDescriptor describes a render pipeline.
type RenderPipelineDescriptor struct {
	Label            string
	Vertex           VertexState
	Fragment         *FragmentState
	Primitive        PrimitiveState
	DepthStencil     *DepthStencilState
	BindGroupLayouts []BindGroupLayout
}

// VertexState describes vertex processing.
type VertexState struct {
	Module     ShaderModule
	EntryPoint string
	Buffers    []VertexBufferLayout
}

// VertexBufferLayout describes a vertex buffer layout.
type VertexBufferLayout struct {
	ArrayStride uint64
	StepMode    VertexStepMode
	Attributes  []VertexAttribute
}

// VertexStepMode describes how vertex data steps.
type VertexStepMode uint32

const (
	VertexStepModeVertex   VertexStepMode = iota
	VertexStepModeInstance
)

// VertexAttribute describes a vertex attribute.
type VertexAttribute struct {
	Format         VertexFormat
	Offset         uint64
	ShaderLocation uint32
}

// VertexFormat describes the format of a vertex attribute.
type VertexFormat uint32

const (
	VertexFormatFloat32x2 VertexFormat = iota
	VertexFormatFloat32x4
	VertexFormatFloat32
	VertexFormatFloat32x3
)

// FragmentState describes fragment processing.
type FragmentState struct {
	Module     ShaderModule
	EntryPoint string
	Targets    []ColorTargetState
}

// ColorTargetState describes a color target.
type ColorTargetState struct {
	Format TextureFormat
	Blend  *BlendState
}

// BlendState describes color blending.
type BlendState struct {
	Color BlendComponent
	Alpha BlendComponent
}

// BlendComponent describes a blend component.
type BlendComponent struct {
	SrcFactor BlendFactor
	DstFactor BlendFactor
	Operation BlendOperation
}

// CompareFunction describes a comparison function for depth/stencil tests.
type CompareFunction uint32

const (
	CompareFunctionNever CompareFunction = iota
	CompareFunctionLess
	CompareFunctionEqual
	CompareFunctionLessEqual
	CompareFunctionGreater
	CompareFunctionNotEqual
	CompareFunctionGreaterEqual
	CompareFunctionAlways
)

// DepthStencilState describes depth/stencil testing.
type DepthStencilState struct {
	Format            TextureFormat
	DepthWriteEnabled bool
	DepthCompare      CompareFunction
}

// PrimitiveState describes primitive assembly.
type PrimitiveState struct {
	Topology  PrimitiveTopology
	CullMode  CullMode
	FrontFace FrontFace
}

// CullMode describes which faces to cull.
type CullMode uint32

const (
	CullModeNone CullMode = iota
	CullModeFront
	CullModeBack
)

// FrontFace describes which winding order is considered front-facing.
type FrontFace uint32

const (
	FrontFaceCCW FrontFace = iota
	FrontFaceCW
)

// BufferDescriptor describes a buffer to create.
type BufferDescriptor struct {
	Label string
	Size  uint64
	Usage BufferUsage
}

// TextureDescriptor describes a texture to create.
type TextureDescriptor struct {
	Label  string
	Size   Extent3D
	Format TextureFormat
	Usage  TextureUsage
}

// TextureUsage describes how a texture will be used.
type TextureUsage uint32

const (
	TextureUsageCopySrc         TextureUsage = 1 << iota
	TextureUsageCopyDst
	TextureUsageTextureBinding
	TextureUsageRenderAttachment
)

// BindGroupLayoutDescriptor describes a bind group layout.
type BindGroupLayoutDescriptor struct {
	Label   string
	Entries []BindGroupLayoutEntry
}

// BindGroupLayoutEntry describes a bind group layout entry.
type BindGroupLayoutEntry struct {
	Binding    uint32
	Visibility ShaderStage
	Buffer     *BufferBindingLayout
	Sampler    *SamplerBindingLayout
	Texture    *TextureBindingLayout
}

// ShaderStage flags.
type ShaderStage uint32

const (
	ShaderStageVertex   ShaderStage = 1 << iota
	ShaderStageFragment
)

// BufferBindingLayout describes a buffer binding.
type BufferBindingLayout struct {
	Type BufferBindingType
}

// BufferBindingType describes the type of buffer binding.
type BufferBindingType uint32

const (
	BufferBindingTypeUniform BufferBindingType = iota
)

// SamplerBindingLayout describes a sampler binding.
type SamplerBindingLayout struct{}

// TextureBindingLayout describes a texture binding.
type TextureBindingLayout struct {
	SampleType    TextureSampleType
	ViewDimension TextureViewDimension
}

// TextureSampleType describes the sample type of a texture.
type TextureSampleType uint32

const (
	TextureSampleTypeFloat TextureSampleType = iota
)

// TextureViewDimension describes the dimension of a texture view.
type TextureViewDimension uint32

const (
	TextureViewDimension2D TextureViewDimension = iota
)

// BindGroupDescriptor describes a bind group.
type BindGroupDescriptor struct {
	Label   string
	Layout  BindGroupLayout
	Entries []BindGroupEntry
}

// BindGroupEntry describes a bind group entry.
type BindGroupEntry struct {
	Binding uint32
	Buffer  Buffer
	Offset  uint64
	Size    uint64
	Sampler Sampler
	Texture TextureView
}

// SamplerDescriptor describes a sampler.
type SamplerDescriptor struct {
	Label string
}

// Surface represents a renderable surface.
type Surface interface {
	// Configure configures the surface for rendering.
	Configure(device Device, config *SurfaceConfiguration)

	// GetCurrentTexture returns the current texture for rendering.
	GetCurrentTexture() (TextureView, error)

	// Present presents the current frame.
	Present()

	// Destroy releases the surface.
	Destroy()
}

// SurfaceConfiguration configures a surface.
type SurfaceConfiguration struct {
	Format      TextureFormat
	Usage       TextureUsage
	Width       uint32
	Height      uint32
	PresentMode PresentMode
}

// PresentMode describes the presentation mode.
type PresentMode uint32

const (
	PresentModeFifo PresentMode = iota // VSync
	PresentModeImmediate
	PresentModeMailbox
)

// SwapChain manages the double/triple buffering.
type SwapChain interface {
	// GetCurrentTextureView returns the current back buffer.
	GetCurrentTextureView() TextureView

	// Present presents the current frame.
	Present()
}

// RenderPipeline represents a compiled render pipeline.
type RenderPipeline interface {
	Destroy()
}

// Buffer represents a GPU buffer.
type Buffer interface {
	// Write writes data to the buffer via the queue.
	Write(queue Queue, data []byte)

	// Destroy releases the buffer.
	Destroy()
}

// Texture represents a GPU texture.
type Texture interface {
	// CreateView creates a texture view.
	CreateView() TextureView

	// Write writes data to the texture.
	Write(queue Queue, data []byte, bytesPerRow uint32)

	// Destroy releases the texture.
	Destroy()
}

// TextureView represents a view into a texture.
type TextureView interface {
	Destroy()
}

// CommandEncoder encodes GPU commands.
type CommandEncoder interface {
	// BeginRenderPass begins a render pass.
	BeginRenderPass(desc *RenderPassDescriptor) RenderPass

	// Finish finishes encoding and returns the command buffer.
	Finish() CommandBuffer
}

// RenderPassDescriptor describes a render pass.
type RenderPassDescriptor struct {
	ColorAttachments      []RenderPassColorAttachment
	DepthStencilAttachment *RenderPassDepthStencilAttachment
}

// RenderPassDepthStencilAttachment describes a depth/stencil attachment for a render pass.
type RenderPassDepthStencilAttachment struct {
	View              TextureView
	DepthLoadOp       LoadOp
	DepthStoreOp      StoreOp
	DepthClearValue   float32
}

// RenderPassColorAttachment describes a color attachment for a render pass.
type RenderPassColorAttachment struct {
	View       TextureView
	LoadOp     LoadOp
	StoreOp    StoreOp
	ClearValue Color
}

// IndexFormat describes the format of index buffer data.
type IndexFormat uint32

const (
	IndexFormatUint16 IndexFormat = iota
	IndexFormatUint32
)

// RenderPass encodes render commands.
type RenderPass interface {
	// SetPipeline sets the render pipeline.
	SetPipeline(pipeline RenderPipeline)

	// SetBindGroup sets a bind group.
	SetBindGroup(index uint32, group BindGroup)

	// SetVertexBuffer sets a vertex buffer.
	SetVertexBuffer(slot uint32, buffer Buffer, offset, size uint64)

	// SetIndexBuffer sets the index buffer.
	SetIndexBuffer(buffer Buffer, format IndexFormat, offset, size uint64)

	// Draw draws primitives.
	Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32)

	// DrawInstanced draws instanced primitives.
	DrawInstanced(vertexCount, instanceCount, firstVertex, firstInstance uint32)

	// DrawIndexed draws indexed primitives.
	DrawIndexed(indexCount, instanceCount, firstIndex, baseVertex int32, firstInstance uint32)

	// SetScissorRect sets the scissor rectangle for the render pass.
	SetScissorRect(x, y, width, height uint32)

	// End ends the render pass.
	End()
}

// CommandBuffer is a finished, ready-to-submit command buffer.
type CommandBuffer interface{}

// Queue submits command buffers to the GPU.
type Queue interface {
	// Submit submits command buffers for execution.
	Submit(buffers ...CommandBuffer)

	// WriteBuffer writes data to a buffer.
	WriteBuffer(buffer Buffer, offset uint64, data []byte)

	// WriteTexture writes data to a texture.
	WriteTexture(dst *ImageCopyTexture, data []byte, layout *TextureDataLayout, size Extent3D)
}

// ImageCopyTexture identifies a specific texture and mip level for copy operations.
type ImageCopyTexture struct {
	Texture Texture
	MipLevel uint32
}

// TextureDataLayout describes the layout of texture data in memory.
type TextureDataLayout struct {
	Offset       uint64
	BytesPerRow  uint32
	RowsPerImage uint32
}

// ShaderModule represents a compiled shader.
type ShaderModule interface {
	Destroy()
}

// BindGroup represents a set of resources bound to a pipeline.
type BindGroup interface {
	Destroy()
}

// BindGroupLayout describes the layout of a bind group.
type BindGroupLayout interface {
	Destroy()
}

// Sampler represents a texture sampler.
type Sampler interface {
	Destroy()
}
