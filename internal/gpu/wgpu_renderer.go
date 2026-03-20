//go:build !nogui && (!windows || gogpu)

package gpu

import (
	"fmt"
	"log"
	"math"
	"time"
	"unsafe"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/text"
	"github.com/timzifer/lux/internal/wgpu"
)

// WGPURenderer implements Renderer using the wgpu abstraction layer (RFC §6.1).
// It provides the same rendering capabilities as OpenGLRenderer but using WebGPU:
//   - Sharp rectangles via clear operations
//   - Rounded rectangles via SDF fragment shader
//   - Bitmap glyph rendering via per-pixel fills
//   - Atlas-based textured glyph rendering (instanced)
//   - MSDF text rendering for large text sizes (instanced)
type WGPURenderer struct {
	width     int
	height    int
	bgColor   draw.Color
	atlas     *text.GlyphAtlas

	// wgpu resources
	instance       wgpu.Instance
	surface        wgpu.Surface
	device         wgpu.Device
	queue          wgpu.Queue

	// Rendering pipelines
	rectPipeline     wgpu.RenderPipeline
	textInstPipeline wgpu.RenderPipeline // instanced text pipeline
	msdfInstPipeline wgpu.RenderPipeline // instanced MSDF pipeline
	surfPipeline     wgpu.RenderPipeline // surface texture blit pipeline
	gradPipeline     wgpu.RenderPipeline // gradient rectangle pipeline

	// Shared resources
	projBuffer     wgpu.Buffer   // 64 bytes: mat4x4 projection
	msdfUniBuffer  wgpu.Buffer   // 80 bytes: mat4x4 projection + vec4 atlas_size
	rectVertBuffer wgpu.Buffer   // unit quad shared by rect + text + MSDF
	rectInstBuffer wgpu.Buffer
	glyphInstBuffer wgpu.Buffer  // unified GPU instance buffer for text + MSDF
	atlasTexture   wgpu.Texture
	atlasView      wgpu.TextureView
	atlasSampler   wgpu.Sampler
	msdfTexture    wgpu.Texture
	msdfView       wgpu.TextureView

	// Bind group layouts (kept for recreating bind groups on atlas resize)
	textLayout     wgpu.BindGroupLayout
	surfTexLayout  wgpu.BindGroupLayout // surface texture bind group layout (group 1)
	gradLayout     wgpu.BindGroupLayout // gradient params bind group layout (group 1)

	// Bind groups
	projBindGroup  wgpu.BindGroup
	textBindGroup  wgpu.BindGroup
	msdfBindGroup  wgpu.BindGroup

	// Surface texture registry
	surfaceTextures map[draw.TextureID]wgpu.TextureView
	surfSampler     wgpu.Sampler
	surfInstBuffer  wgpu.Buffer // per-surface instance (rect x,y,w,h = 16 bytes)

	// Gradient resources
	gradUniBuffer    wgpu.Buffer // gradient params uniform buffer (resizable)
	gradUniBufCap    uint64      // current capacity in bytes
	gradBindGroups   []wgpu.BindGroup // per-gradient bind groups (rebuilt each frame)

	// Shadow resources
	shadowPipeline    wgpu.RenderPipeline
	shadowInstBuffer  wgpu.Buffer
	shadowInstBufCap  uint64
	shadowBuf         []float32

	// Blur resources (fragment-shader-based, 2-pass ping-pong)
	blurPipeline        wgpu.RenderPipeline  // fullscreen-triangle blur pass
	blurBlitPipeline    wgpu.RenderPipeline  // blit blurred result back to surface
	blurUniformBuffer   wgpu.Buffer
	blurBindGroupLayout wgpu.BindGroupLayout // group 0: blur uniforms
	blurTexBindGroupLayout wgpu.BindGroupLayout // group 1: texture + sampler (blur pass)
	blurBlitBindGroupLayout wgpu.BindGroupLayout // group 1: texture + sampler (blit pass)
	blurSampler         wgpu.Sampler
	blurSrcTexture      wgpu.Texture  // original unblurred scene (read-only after scene render)
	blurSrcView         wgpu.TextureView
	blurTmpTexture      wgpu.Texture  // H-pass output
	blurTmpView         wgpu.TextureView
	blurDstTexture      wgpu.Texture  // V-pass output (final blurred result)
	blurDstView         wgpu.TextureView
	blurW, blurH        int // current blur texture dimensions

	// CPU-side retained buffers — grow-only, reset to [:0] each frame.
	rectBuf  []float32
	glyphBuf []float32 // unified: [text main|text overlay|msdf main|msdf overlay]

	// GPU buffer capacities (bytes) — for grow-on-demand.
	rectInstBufCap  uint64
	glyphInstBufCap uint64

	// State tracking
	inited         bool
	surfaceOK      bool
	atlasW, atlasH int // last known atlas texture size
	msdfW, msdfH   int // last known MSDF atlas texture size

	// Performance metrics
	perfFrames     int
	perfDrawStart  time.Time
	perfLastReport time.Time
	perfTotalDraw  time.Duration
	perfMinDraw    time.Duration
	perfMaxDraw    time.Duration
	perfRects      int
	perfTextGlyphs int
	perfMSDFGlyphs int
}

// NewWGPU creates a new wgpu-based renderer.
func NewWGPU() *WGPURenderer {
	return &WGPURenderer{}
}

// Init initializes the wgpu rendering context.
func (r *WGPURenderer) Init(cfg Config) error {
	r.width = cfg.Width
	r.height = cfg.Height

	instance, err := wgpu.CreateInstance()
	if err != nil {
		return fmt.Errorf("wgpu instance: %w", err)
	}
	r.instance = instance

	// Create surface from native handle.
	if cfg.NativeHandle != 0 {
		r.surface = instance.CreateSurface(&wgpu.SurfaceDescriptor{
			NativeHandle: cfg.NativeHandle,
		})
		r.surfaceOK = true
	}

	// Request adapter and device.
	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		CompatibleSurface: r.surface,
		PowerPreference:   wgpu.PowerPreferenceHighPerformance,
	})
	if err != nil {
		return fmt.Errorf("wgpu adapter: %w", err)
	}
	info := adapter.GetInfo()
	log.Printf("wgpu: adapter=%q backend=%s type=%s", info.Name, info.BackendType, info.AdapterType)

	device, err := adapter.RequestDevice(&wgpu.DeviceDescriptor{
		Label: "lux-device",
	})
	if err != nil {
		return fmt.Errorf("wgpu device: %w", err)
	}
	r.device = device
	r.queue = device.GetQueue()

	// Configure surface.
	if r.surfaceOK {
		r.surface.Configure(device, &wgpu.SurfaceConfiguration{
			Format:      wgpu.TextureFormatBGRA8Unorm,
			Usage:       wgpu.TextureUsageRenderAttachment,
			Width:       uint32(r.width),
			Height:      uint32(r.height),
			PresentMode: wgpu.PresentModeFifo,
		})
	}

	// Create projection uniform buffer (4x4 float32 matrix = 64 bytes).
	r.projBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "projection",
		Size:  64,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})

	// Create MSDF uniform buffer (projection + atlas_size vec4 = 80 bytes).
	r.msdfUniBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "msdf-uniforms",
		Size:  80,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})

	// Create rect vertex buffer (unit quad: 6 vertices * 2 floats * 4 bytes = 48 bytes).
	// Shared by rect, text, and MSDF pipelines.
	r.rectVertBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "rect-verts",
		Size:  48,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Upload unit quad vertices.
	quadVerts := []float32{0, 0, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1}
	r.rectVertBuffer.Write(r.queue, float32SliceToBytes(quadVerts))

	// Create rect instance buffer (dynamic, resized as needed).
	r.rectInstBufCap = 1024 * 9 * 4
	r.rectInstBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "rect-instances",
		Size:  r.rectInstBufCap,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Create unified glyph instance buffer (dynamic).
	r.glyphInstBufCap = 4096 * 12 * 4
	r.glyphInstBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "glyph-instances",
		Size:  r.glyphInstBufCap,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Create atlas texture (initially 512x512, single-channel).
	r.atlasW, r.atlasH = 512, 512
	r.atlasTexture = device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "glyph-atlas",
		Size:   wgpu.Extent3D{Width: uint32(r.atlasW), Height: uint32(r.atlasH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatR8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	r.atlasView = r.atlasTexture.CreateView()

	// Create MSDF atlas texture (initially 512x512, RGBA).
	r.msdfW, r.msdfH = 512, 512
	r.msdfTexture = device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "msdf-atlas",
		Size:   wgpu.Extent3D{Width: uint32(r.msdfW), Height: uint32(r.msdfH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatRGBA8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	r.msdfView = r.msdfTexture.CreateView()

	// Create sampler.
	r.atlasSampler = device.CreateSampler(&wgpu.SamplerDescriptor{
		Label: "atlas-sampler",
	})

	// Create shader modules.
	rectShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "rect-shader",
		Source: wgslRectShader,
	})
	defer rectShader.Destroy()

	textShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "text-instanced-shader",
		Source: wgslTextInstancedShader,
	})
	defer textShader.Destroy()

	msdfShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "msdf-instanced-shader",
		Source: wgslMSDFInstancedShader,
	})
	defer msdfShader.Destroy()

	// Create bind group layouts.
	projLayout := device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "proj-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})
	defer projLayout.Destroy()

	r.textLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "text-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})

	// Alpha blending state.
	blend := &wgpu.BlendState{
		Color: wgpu.BlendComponent{
			SrcFactor: wgpu.BlendFactorSrcAlpha,
			DstFactor: wgpu.BlendFactorOneMinusSrcAlpha,
			Operation: wgpu.BlendOperationAdd,
		},
		Alpha: wgpu.BlendComponent{
			SrcFactor: wgpu.BlendFactorOne,
			DstFactor: wgpu.BlendFactorOneMinusSrcAlpha,
			Operation: wgpu.BlendOperationAdd,
		},
	}

	// Instanced vertex buffer layout for glyph instances (shared by text + MSDF).
	glyphInstanceLayout := wgpu.VertexBufferLayout{
		ArrayStride: 48, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1},  // glyph_rect
			{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2}, // glyph_uv
			{Format: wgpu.VertexFormatFloat32x4, Offset: 32, ShaderLocation: 3}, // color
		},
	}

	// Unit quad vertex layout (shared by rect, text, MSDF).
	unitQuadLayout := wgpu.VertexBufferLayout{
		ArrayStride: 8, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
		},
	}

	// Create render pipelines.
	r.rectPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "rect-pipeline",
		Vertex: wgpu.VertexState{
			Module:     rectShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				unitQuadLayout,
				{ArrayStride: 36, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1},  // rect (x,y,w,h)
					{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2}, // color
					{Format: wgpu.VertexFormatFloat32, Offset: 32, ShaderLocation: 3},   // radius
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     rectShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{projLayout},
	})

	r.textInstPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "text-instanced-pipeline",
		Vertex: wgpu.VertexState{
			Module:     textShader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{unitQuadLayout, glyphInstanceLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     textShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{r.textLayout},
	})

	r.msdfInstPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "msdf-instanced-pipeline",
		Vertex: wgpu.VertexState{
			Module:     msdfShader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{unitQuadLayout, glyphInstanceLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     msdfShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{r.textLayout},
	})

	// Create bind groups.
	r.projBindGroup = device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "proj-bind-group",
		Layout: projLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.projBuffer, Size: 64},
		},
	})

	r.textBindGroup = device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "text-bind-group",
		Layout: r.textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.projBuffer, Size: 64},
			{Binding: 1, Texture: r.atlasView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})

	r.msdfBindGroup = device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "msdf-bind-group",
		Layout: r.textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.msdfUniBuffer, Size: 80},
			{Binding: 1, Texture: r.msdfView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})

	// --- Surface blit pipeline ---

	surfShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "surface-shader",
		Source: wgslSurfaceShader,
	})
	defer surfShader.Destroy()

	r.surfTexLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "surf-tex-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})

	r.surfSampler = device.CreateSampler(&wgpu.SamplerDescriptor{Label: "surface-sampler"})

	// Surface instance buffer (1 rect = 16 bytes).
	r.surfInstBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "surf-instance",
		Size:  16,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	r.surfPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "surface-pipeline",
		Vertex: wgpu.VertexState{
			Module:     surfShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				unitQuadLayout,
				{ArrayStride: 16, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1}, // rect
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     surfShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{projLayout, r.surfTexLayout},
	})

	// --- Gradient pipeline ---

	gradShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "gradient-shader",
		Source: wgslGradientShader,
	})
	defer gradShader.Destroy()

	r.gradLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "grad-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})

	// Gradient uniform buffer — resized per-frame to hold all gradients.
	// Each gradient = 304 bytes, padded to 512 bytes (256-byte offset alignment).
	r.gradUniBufCap = 512
	r.gradUniBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "grad-uniforms",
		Size:  r.gradUniBufCap,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})

	r.gradPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "gradient-pipeline",
		Vertex: wgpu.VertexState{
			Module:     gradShader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{unitQuadLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     gradShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{projLayout, r.gradLayout},
	})

	r.surfaceTextures = make(map[draw.TextureID]wgpu.TextureView)

	// --- Shadow pipeline ---

	shadowShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "shadow-shader",
		Source: wgslShadowShader,
	})
	defer shadowShader.Destroy()

	// Shadow instance buffer (12 floats = 48 bytes per instance).
	r.shadowInstBufCap = 256 * 12 * 4
	r.shadowInstBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "shadow-instances",
		Size:  r.shadowInstBufCap,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	r.shadowPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "shadow-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shadowShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				unitQuadLayout,
				{ArrayStride: 48, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1},  // rect (x,y,w,h)
					{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2}, // color
					{Format: wgpu.VertexFormatFloat32, Offset: 32, ShaderLocation: 3},   // radius
					{Format: wgpu.VertexFormatFloat32, Offset: 36, ShaderLocation: 4},   // blur_radius
					{Format: wgpu.VertexFormatFloat32, Offset: 40, ShaderLocation: 5},   // inset
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     shadowShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{projLayout},
	})

	// --- Blur pipeline (fragment-shader, 2-pass ping-pong) ---

	blurShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "blur-shader",
		Source: wgslBlurShader,
	})
	defer blurShader.Destroy()

	blurBlitShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "blur-blit-shader",
		Source: wgslBlurBlitShader,
	})
	defer blurBlitShader.Destroy()

	// Blur uniform buffer: direction vec2<f32>, radius u32, pad u32, texture_size vec2<f32>, pad2 vec2<f32> = 32 bytes.
	r.blurUniformBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "blur-uniforms",
		Size:  32,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})

	// Blur bind group layout group 0: uniforms (vertex+fragment visibility for the blur pass).
	r.blurBindGroupLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "blur-uni-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})

	// Blur bind group layout group 1: input texture + sampler (fragment visibility).
	r.blurTexBindGroupLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "blur-tex-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})

	// Blur render pipeline — fullscreen triangle, no vertex buffers.
	r.blurPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "blur-pipeline",
		Vertex: wgpu.VertexState{
			Module:     blurShader,
			EntryPoint: "vs_main",
		},
		Fragment: &wgpu.FragmentState{
			Module:     blurShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{r.blurBindGroupLayout, r.blurTexBindGroupLayout},
	})

	// Blur blit bind group layout group 1: texture + sampler (fragment visibility).
	r.blurBlitBindGroupLayout = device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "blur-blit-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})

	// Blur blit render pipeline (reuses projection layout for group 0, surface-like vertex layout).
	r.blurBlitPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "blur-blit-pipeline",
		Vertex: wgpu.VertexState{
			Module:     blurBlitShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				unitQuadLayout,
				{ArrayStride: 16, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1}, // rect
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     blurBlitShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{projLayout, r.blurBlitBindGroupLayout},
	})

	// Blur sampler.
	r.blurSampler = device.CreateSampler(&wgpu.SamplerDescriptor{Label: "blur-sampler"})

	// Create initial blur textures at framebuffer size.
	r.resizeBlurTextures()

	// Upload initial projection matrix.
	r.updateProjection()

	r.inited = true
	return nil
}

// SetBackgroundColor sets the clear color for BeginFrame.
func (r *WGPURenderer) SetBackgroundColor(c draw.Color) {
	r.bgColor = c
}

// SetAtlas sets the glyph atlas for textured glyph rendering.
func (r *WGPURenderer) SetAtlas(a *text.GlyphAtlas) {
	r.atlas = a
}

// Device returns the wgpu device for external surface providers.
func (r *WGPURenderer) Device() wgpu.Device { return r.device }

// Queue returns the wgpu command queue for external surface providers.
func (r *WGPURenderer) Queue() wgpu.Queue { return r.queue }

// RegisterSurfaceTexture registers an external texture view under the given ID.
// Surface providers call this to make their rendered texture available for blitting.
func (r *WGPURenderer) RegisterSurfaceTexture(id draw.TextureID, view wgpu.TextureView) {
	if r.surfaceTextures == nil {
		r.surfaceTextures = make(map[draw.TextureID]wgpu.TextureView)
	}
	r.surfaceTextures[id] = view
}

// UnregisterSurfaceTexture removes a previously registered texture.
func (r *WGPURenderer) UnregisterSurfaceTexture(id draw.TextureID) {
	delete(r.surfaceTextures, id)
}

// Resize updates the viewport.
func (r *WGPURenderer) Resize(width, height int) {
	r.width = width
	r.height = height
	if r.surfaceOK {
		r.surface.Configure(r.device, &wgpu.SurfaceConfiguration{
			Format:      wgpu.TextureFormatBGRA8Unorm,
			Usage:       wgpu.TextureUsageRenderAttachment,
			Width:       uint32(width),
			Height:      uint32(height),
			PresentMode: wgpu.PresentModeFifo,
		})
	}
	r.updateProjection()
	r.resizeBlurTextures()
}

// BeginFrame starts a new frame.
func (r *WGPURenderer) BeginFrame() {
	// Frame setup is handled in Draw when we acquire the surface texture.
}

// Draw renders the current scene using wgpu.
//
// WebGPU semantics: queue.WriteBuffer() executes immediately, but draw commands
// in a render pass only execute at queue.Submit(). We must upload ALL buffer data
// before beginning the render pass, then use firstInstance offsets to
// separate main and overlay draws within the same buffer.
func (r *WGPURenderer) Draw(scene draw.Scene) {
	if !r.inited || !r.surfaceOK {
		return
	}
	drawStart := time.Now()

	// Acquire current surface texture.
	textureView, err := r.surface.GetCurrentTexture()
	if err != nil {
		log.Printf("wgpu: GetCurrentTexture failed: %v", err)
		return
	}

	// --- Phase 1: Upload all buffer data before recording draw commands ---

	// Resize GPU textures if the atlas has grown, then upload.
	if r.atlas != nil {
		if r.atlas.Width != r.atlasW || r.atlas.Height != r.atlasH {
			r.resizeAtlasTexture()
		}
		if r.atlas.MSDFWidth != r.msdfW || r.atlas.MSDFHeight != r.msdfH {
			r.resizeMSDFTexture()
		}
		if r.atlas.Dirty {
			r.atlasTexture.Write(r.queue, r.atlas.Image.Pix, uint32(r.atlas.Image.Stride))
			r.atlas.Dirty = false
		}
		if r.atlas.MSDFDirty {
			r.msdfTexture.Write(r.queue, r.atlas.MSDFImage.Pix, uint32(r.atlas.MSDFImage.Stride))
			r.atlas.MSDFDirty = false
		}
	}

	// Rects: concatenate main + overlay instance data using retained buffer.
	mainRectCount := uint32(len(scene.Rects))
	overlayRectCount := uint32(len(scene.OverlayRects))
	if mainRectCount+overlayRectCount > 0 {
		r.rectBuf = r.rectBuf[:0]
		for _, rect := range scene.Rects {
			r.rectBuf = append(r.rectBuf,
				float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
				rect.Color.R, rect.Color.G, rect.Color.B, rect.Color.A,
				rect.Radius,
			)
		}
		for _, rect := range scene.OverlayRects {
			r.rectBuf = append(r.rectBuf,
				float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
				rect.Color.R, rect.Color.G, rect.Color.B, rect.Color.A,
				rect.Radius,
			)
		}
		needed := uint64(len(r.rectBuf)) * 4
		r.ensureGPUBuffer(&r.rectInstBuffer, &r.rectInstBufCap, needed, "rect-instances", wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst)
		r.rectInstBuffer.Write(r.queue, float32SliceToBytes(r.rectBuf))
	}

	// Glyph instances: unified buffer [text main | text overlay | msdf main | msdf overlay].
	// 12 floats per glyph instance (glyph_rect + glyph_uv + color).
	var mainTextGlyphs, overlayTextGlyphs int
	var mainMSDFGlyphs, overlayMSDFGlyphs int
	if r.atlas != nil {
		atlasW := float32(r.atlas.Width)
		atlasH := float32(r.atlas.Height)
		msdfW := float32(r.atlas.MSDFWidth)
		msdfH := float32(r.atlas.MSDFHeight)

		r.glyphBuf = r.glyphBuf[:0]

		// Text glyphs: main + overlay
		for _, g := range scene.TexturedGlyphs {
			r.glyphBuf = appendGlyphInstance(r.glyphBuf, g, atlasW, atlasH)
		}
		for _, g := range scene.OverlayTexturedGlyphs {
			r.glyphBuf = appendGlyphInstance(r.glyphBuf, g, atlasW, atlasH)
		}
		mainTextGlyphs = len(scene.TexturedGlyphs)
		overlayTextGlyphs = len(scene.OverlayTexturedGlyphs)

		// MSDF glyphs: main + overlay
		for _, g := range scene.MSDFGlyphs {
			r.glyphBuf = appendGlyphInstance(r.glyphBuf, g, msdfW, msdfH)
		}
		for _, g := range scene.OverlayMSDFGlyphs {
			r.glyphBuf = appendGlyphInstance(r.glyphBuf, g, msdfW, msdfH)
		}
		mainMSDFGlyphs = len(scene.MSDFGlyphs)
		overlayMSDFGlyphs = len(scene.OverlayMSDFGlyphs)

		// Upload unified glyph instance buffer.
		if len(r.glyphBuf) > 0 {
			needed := uint64(len(r.glyphBuf)) * 4
			r.ensureGPUBuffer(&r.glyphInstBuffer, &r.glyphInstBufCap, needed, "glyph-instances", wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst)
			r.glyphInstBuffer.Write(r.queue, float32SliceToBytes(r.glyphBuf))
		}
	}

	// Shadows: concatenate main + overlay shadow instance data.
	mainShadowCount := uint32(len(scene.ShadowRects))
	overlayShadowCount := uint32(len(scene.OverlayShadowRects))
	if mainShadowCount+overlayShadowCount > 0 {
		r.shadowBuf = r.shadowBuf[:0]
		for _, s := range scene.ShadowRects {
			var insetFloat float32
			if s.Inset {
				insetFloat = 1.0
			}
			r.shadowBuf = append(r.shadowBuf,
				float32(s.X), float32(s.Y), float32(s.W), float32(s.H),
				s.Color.R, s.Color.G, s.Color.B, s.Color.A,
				s.Radius, s.BlurRadius, insetFloat, 0.0,
			)
		}
		for _, s := range scene.OverlayShadowRects {
			var insetFloat float32
			if s.Inset {
				insetFloat = 1.0
			}
			r.shadowBuf = append(r.shadowBuf,
				float32(s.X), float32(s.Y), float32(s.W), float32(s.H),
				s.Color.R, s.Color.G, s.Color.B, s.Color.A,
				s.Radius, s.BlurRadius, insetFloat, 0.0,
			)
		}
		needed := uint64(len(r.shadowBuf)) * 4
		r.ensureGPUBuffer(&r.shadowInstBuffer, &r.shadowInstBufCap, needed, "shadow-instances", wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst)
		r.shadowInstBuffer.Write(r.queue, float32SliceToBytes(r.shadowBuf))
	}
	totalShadowBufSize := uint64((mainShadowCount + overlayShadowCount) * 12 * 4)

	// Gradients: pre-upload all gradient uniform data and create per-gradient bind groups.
	// Each gradient occupies 512 bytes (304 data + 208 padding for 256-byte alignment).
	const gradStride = 512 // bytes
	const gradStrideF = gradStride / 4 // 128 floats
	allGrads := append(scene.GradientRects, scene.OverlayGradientRects...)
	// Destroy previous frame's bind groups.
	for _, bg := range r.gradBindGroups {
		bg.Destroy()
	}
	r.gradBindGroups = r.gradBindGroups[:0]

	if len(allGrads) > 0 {
		needed := uint64(len(allGrads)) * gradStride
		r.ensureGPUBuffer(&r.gradUniBuffer, &r.gradUniBufCap, needed, "grad-uniforms", wgpu.BufferUsageUniform|wgpu.BufferUsageCopyDst)

		gradBuf := make([]float32, len(allGrads)*gradStrideF)
		for i, gr := range allGrads {
			off := i * gradStrideF
			packGradientUniform(gradBuf[off:off+76], gr)
		}
		r.gradUniBuffer.Write(r.queue, float32SliceToBytes(gradBuf))

		// Create per-gradient bind groups with buffer offsets.
		for i := range allGrads {
			bg := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:  "grad-bg",
				Layout: r.gradLayout,
				Entries: []wgpu.BindGroupEntry{
					{Binding: 0, Buffer: r.gradUniBuffer, Offset: uint64(i) * gradStride, Size: 304},
				},
			})
			r.gradBindGroups = append(r.gradBindGroups, bg)
		}
	}
	mainGradCount := len(scene.GradientRects)

	// --- Phase 2: Record render pass commands ---

	hasBlur := len(scene.BlurRegions) > 0 && r.blurPipeline != nil

	encoder := r.device.CreateCommandEncoder()

	// When blur is active, render the scene to an offscreen texture (blurSrc)
	// so we can read it back for the blur passes, then blit to the surface.
	mainTarget := textureView
	if hasBlur {
		mainTarget = r.blurSrcView
	}

	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    mainTarget,
				LoadOp:  wgpu.LoadOpClear,
				StoreOp: wgpu.StoreOpStore,
				ClearValue: wgpu.Color{
					R: float64(r.bgColor.R),
					G: float64(r.bgColor.G),
					B: float64(r.bgColor.B),
					A: float64(r.bgColor.A),
				},
			},
		},
	})

	// Set initial full-viewport scissor.
	vpW, vpH := uint32(r.width), uint32(r.height)
	renderPass.SetScissorRect(0, 0, vpW, vpH)

	totalRectBufSize := uint64((mainRectCount + overlayRectCount) * 9 * 4)
	glyphBufSize := uint64(len(r.glyphBuf)) * 4

	// MSDF instances start after all text instances in the unified buffer.
	msdfGPUOffset := mainTextGlyphs + overlayTextGlyphs

	// Draw main content via scissor clip batches.
	r.drawClipBatches(renderPass, scene.ClipBatches,
		int(mainRectCount), mainTextGlyphs, mainMSDFGlyphs, int(mainShadowCount),
		0, 0, msdfGPUOffset, 0, // MSDF starts after all text glyphs in unified buffer
		totalRectBufSize, glyphBufSize, totalShadowBufSize,
		scene.GradientRects, 0,
		vpW, vpH)

	// Draw surfaces (between main and overlay).
	r.drawSurfaces(renderPass, scene.Surfaces, vpW, vpH)

	if !hasBlur {
		// No blur: overlay in same pass (fast path).
		r.drawClipBatches(renderPass, scene.OverlayClipBatches,
			int(overlayRectCount), overlayTextGlyphs, overlayMSDFGlyphs, int(overlayShadowCount),
			int(mainRectCount), mainTextGlyphs, msdfGPUOffset+mainMSDFGlyphs, int(mainShadowCount),
			totalRectBufSize, glyphBufSize, totalShadowBufSize,
			scene.OverlayGradientRects, mainGradCount,
			vpW, vpH)
	}

	renderPass.End()

	// --- Blur post-processing + final blit to surface ---
	//
	// Flow when hasBlur:
	//   1. Scene was rendered to blurSrc (offscreen, unblurred — preserved read-only).
	//   2. Blit unblurred blurSrc → surface (sharp full-screen copy).
	//   3. Per blur region (each with its own radius):
	//      a. H-pass: blurSrc → blurTmp  (blurSrc stays pristine)
	//      b. V-pass: blurTmp → blurDst
	//      c. Scissor-blit: blurDst → surface (LoadOp=Load, exact region bounds)
	if hasBlur {
		// Upload blit rect instance: full viewport.
		blitRect := []float32{0, 0, float32(r.width), float32(r.height)}
		r.surfInstBuffer.Write(r.queue, float32SliceToBytes(blitRect))

		// --- Step 2: Blit unblurred scene to surface ---
		unblurredBG := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "blur-unblurred",
			Layout: r.blurBlitBindGroupLayout,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Texture: r.blurSrcView},
				{Binding: 1, Sampler: r.blurSampler},
			},
		})
		copyPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{{
				View:    textureView,
				LoadOp:  wgpu.LoadOpClear,
				StoreOp: wgpu.StoreOpStore,
				ClearValue: wgpu.Color{R: 0, G: 0, B: 0, A: 1},
			}},
		})
		copyPass.SetPipeline(r.blurBlitPipeline)
		copyPass.SetBindGroup(0, r.projBindGroup)
		copyPass.SetBindGroup(1, unblurredBG)
		copyPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
		copyPass.SetVertexBuffer(1, r.surfInstBuffer, 0, 16)
		copyPass.Draw(6, 1, 0, 0)
		copyPass.End()

		// Shared bind groups for reading blurSrc (original scene).
		srcTexBG := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "blur-src-tex",
			Layout: r.blurTexBindGroupLayout,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Texture: r.blurSrcView},
				{Binding: 1, Sampler: r.blurSampler},
			},
		})
		tmpTexBG := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "blur-tmp-tex",
			Layout: r.blurTexBindGroupLayout,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Texture: r.blurTmpView},
				{Binding: 1, Sampler: r.blurSampler},
			},
		})
		dstBlitBG := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "blur-dst-blit",
			Layout: r.blurBlitBindGroupLayout,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Texture: r.blurDstView},
				{Binding: 1, Sampler: r.blurSampler},
			},
		})

		// --- Step 3: Pre-upload all blur uniforms into a single buffer ---
		// Each region needs 2 slots (H-pass, V-pass). 256-byte alignment per slot.
		const blurUniStride = 256 // bytes (WebGPU minUniformBufferOffsetAlignment)
		const blurUniStrideF = blurUniStride / 4 // 64 floats
		numSlots := len(scene.BlurRegions) * 2
		needed := uint64(numSlots) * blurUniStride

		// Recreate blur uniform buffer if too small.
		if r.blurUniformBuffer != nil {
			r.blurUniformBuffer.Destroy()
		}
		r.blurUniformBuffer = r.device.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "blur-uniforms",
			Size:  needed,
			Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
		})

		blurUniBuf := make([]float32, numSlots*blurUniStrideF)
		slot := 0
		for _, br := range scene.BlurRegions {
			radius := br.Radius
			if radius > 64 {
				radius = 64
			}
			// H-pass slot
			off := slot * blurUniStrideF
			blurUniBuf[off+0] = 1.0 // direction.x
			blurUniBuf[off+1] = 0.0 // direction.y
			*(*uint32)(unsafe.Pointer(&blurUniBuf[off+2])) = uint32(radius)
			blurUniBuf[off+4] = float32(r.blurW)
			blurUniBuf[off+5] = float32(r.blurH)
			slot++
			// V-pass slot
			off = slot * blurUniStrideF
			blurUniBuf[off+0] = 0.0 // direction.x
			blurUniBuf[off+1] = 1.0 // direction.y
			*(*uint32)(unsafe.Pointer(&blurUniBuf[off+2])) = uint32(radius)
			blurUniBuf[off+4] = float32(r.blurW)
			blurUniBuf[off+5] = float32(r.blurH)
			slot++
		}
		r.blurUniformBuffer.Write(r.queue, float32SliceToBytes(blurUniBuf))

		// Create per-slot bind groups with buffer offsets.
		blurUniBGs := make([]wgpu.BindGroup, numSlots)
		for i := 0; i < numSlots; i++ {
			blurUniBGs[i] = r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:  "blur-uni",
				Layout: r.blurBindGroupLayout,
				Entries: []wgpu.BindGroupEntry{
					{Binding: 0, Buffer: r.blurUniformBuffer, Offset: uint64(i) * blurUniStride, Size: 32},
				},
			})
		}

		// --- Step 4: Per-region blur + scissor blit ---
		slot = 0
		for _, br := range scene.BlurRegions {
			if br.W <= 0 || br.H <= 0 || br.Radius <= 0 {
				slot += 2
				continue
			}
			hUniBG := blurUniBGs[slot]
			vUniBG := blurUniBGs[slot+1]
			slot += 2

			// H-pass: blurSrc → blurTmp
			hPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
				ColorAttachments: []wgpu.RenderPassColorAttachment{{
					View: r.blurTmpView, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore,
					ClearValue: wgpu.Color{R: 0, G: 0, B: 0, A: 0},
				}},
			})
			hPass.SetPipeline(r.blurPipeline)
			hPass.SetBindGroup(0, hUniBG)
			hPass.SetBindGroup(1, srcTexBG)
			hPass.Draw(3, 1, 0, 0)
			hPass.End()

			// V-pass: blurTmp → blurDst
			vPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
				ColorAttachments: []wgpu.RenderPassColorAttachment{{
					View: r.blurDstView, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore,
					ClearValue: wgpu.Color{R: 0, G: 0, B: 0, A: 0},
				}},
			})
			vPass.SetPipeline(r.blurPipeline)
			vPass.SetBindGroup(0, vUniBG)
			vPass.SetBindGroup(1, tmpTexBG)
			vPass.Draw(3, 1, 0, 0)
			vPass.End()

			// Scissor-blit: blurDst → surface (exact region bounds)
			sx, sy := uint32(br.X), uint32(br.Y)
			sw, sh := uint32(br.W), uint32(br.H)
			if sx+sw > vpW { sw = vpW - sx }
			if sy+sh > vpH { sh = vpH - sy }

			blitPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
				ColorAttachments: []wgpu.RenderPassColorAttachment{{
					View: textureView, LoadOp: wgpu.LoadOpLoad, StoreOp: wgpu.StoreOpStore,
				}},
			})
			blitPass.SetPipeline(r.blurBlitPipeline)
			blitPass.SetBindGroup(0, r.projBindGroup)
			blitPass.SetBindGroup(1, dstBlitBG)
			blitPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
			blitPass.SetVertexBuffer(1, r.surfInstBuffer, 0, 16)
			blitPass.SetScissorRect(sx, sy, sw, sh)
			blitPass.Draw(6, 1, 0, 0)
			blitPass.End()

		}

		// Cleanup bind groups.
		for _, bg := range blurUniBGs {
			bg.Destroy()
		}
		unblurredBG.Destroy()
		srcTexBG.Destroy()
		tmpTexBG.Destroy()
		dstBlitBG.Destroy()

		// --- Overlay pass (post-blur): render overlay content on top of blurred surface ---
		// This enables frosted glass: blurred backdrop + sharp overlay content.
		hasOverlay := overlayRectCount > 0 || overlayTextGlyphs > 0 || overlayMSDFGlyphs > 0 ||
			len(scene.OverlayGradientRects) > 0 || overlayShadowCount > 0
		if hasOverlay {
			overlayPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
				ColorAttachments: []wgpu.RenderPassColorAttachment{{
					View:    textureView,
					LoadOp:  wgpu.LoadOpLoad,
					StoreOp: wgpu.StoreOpStore,
				}},
			})
			overlayPass.SetScissorRect(0, 0, vpW, vpH)
			r.drawClipBatches(overlayPass, scene.OverlayClipBatches,
				int(overlayRectCount), overlayTextGlyphs, overlayMSDFGlyphs, int(overlayShadowCount),
				int(mainRectCount), mainTextGlyphs, msdfGPUOffset+mainMSDFGlyphs, int(mainShadowCount),
				totalRectBufSize, glyphBufSize, totalShadowBufSize,
				scene.OverlayGradientRects, mainGradCount,
				vpW, vpH)
			overlayPass.End()
		}
	}

	// Submit.
	cmdBuffer := encoder.Finish()
	r.queue.Submit(cmdBuffer)

	// Collect perf metrics.
	drawDur := time.Since(drawStart)
	r.perfFrames++
	r.perfTotalDraw += drawDur
	if drawDur < r.perfMinDraw || r.perfMinDraw == 0 {
		r.perfMinDraw = drawDur
	}
	if drawDur > r.perfMaxDraw {
		r.perfMaxDraw = drawDur
	}
	r.perfRects += int(mainRectCount + overlayRectCount)
	r.perfTextGlyphs += mainTextGlyphs + overlayTextGlyphs
	r.perfMSDFGlyphs += mainMSDFGlyphs + overlayMSDFGlyphs
}

// EndFrame presents the rendered frame.
func (r *WGPURenderer) EndFrame() {
	if r.surfaceOK {
		r.surface.Present()
	}
	r.reportPerf()
}

func (r *WGPURenderer) reportPerf() {
	if r.perfLastReport.IsZero() {
		r.perfLastReport = time.Now()
		return
	}
	elapsed := time.Since(r.perfLastReport)
	if elapsed < 5*time.Second {
		return
	}
	fps := float64(r.perfFrames) / elapsed.Seconds()
	avgDraw := time.Duration(0)
	if r.perfFrames > 0 {
		avgDraw = r.perfTotalDraw / time.Duration(r.perfFrames)
	}
	avgRects := 0
	avgText := 0
	avgMSDF := 0
	if r.perfFrames > 0 {
		avgRects = r.perfRects / r.perfFrames
		avgText = r.perfTextGlyphs / r.perfFrames
		avgMSDF = r.perfMSDFGlyphs / r.perfFrames
	}
	log.Printf("wgpu perf: %.1f fps | draw avg=%v min=%v max=%v | rects=%d textGlyphs=%d msdfGlyphs=%d (per frame avg)",
		fps, avgDraw.Round(time.Microsecond), r.perfMinDraw.Round(time.Microsecond), r.perfMaxDraw.Round(time.Microsecond),
		avgRects, avgText, avgMSDF)

	// Reset.
	r.perfFrames = 0
	r.perfTotalDraw = 0
	r.perfMinDraw = 0
	r.perfMaxDraw = 0
	r.perfRects = 0
	r.perfTextGlyphs = 0
	r.perfMSDFGlyphs = 0
	r.perfLastReport = time.Now()
}

// Destroy releases wgpu resources.
func (r *WGPURenderer) Destroy() {
	if !r.inited {
		return
	}
	r.projBindGroup.Destroy()
	r.textBindGroup.Destroy()
	r.msdfBindGroup.Destroy()
	r.projBuffer.Destroy()
	r.msdfUniBuffer.Destroy()
	r.gradUniBuffer.Destroy()
	r.rectVertBuffer.Destroy()
	r.rectInstBuffer.Destroy()
	r.glyphInstBuffer.Destroy()
	r.surfInstBuffer.Destroy()
	r.atlasView.Destroy()
	r.atlasTexture.Destroy()
	r.msdfView.Destroy()
	r.msdfTexture.Destroy()
	r.atlasSampler.Destroy()
	r.surfSampler.Destroy()
	r.rectPipeline.Destroy()
	r.textInstPipeline.Destroy()
	r.msdfInstPipeline.Destroy()
	r.surfPipeline.Destroy()
	r.gradPipeline.Destroy()
	r.shadowPipeline.Destroy()
	r.shadowInstBuffer.Destroy()
	r.textLayout.Destroy()
	r.surfTexLayout.Destroy()
	r.gradLayout.Destroy()
	if r.blurPipeline != nil {
		r.blurPipeline.Destroy()
	}
	if r.blurBlitPipeline != nil {
		r.blurBlitPipeline.Destroy()
	}
	if r.blurUniformBuffer != nil {
		r.blurUniformBuffer.Destroy()
	}
	if r.blurBindGroupLayout != nil {
		r.blurBindGroupLayout.Destroy()
	}
	if r.blurTexBindGroupLayout != nil {
		r.blurTexBindGroupLayout.Destroy()
	}
	if r.blurBlitBindGroupLayout != nil {
		r.blurBlitBindGroupLayout.Destroy()
	}
	if r.blurSampler != nil {
		r.blurSampler.Destroy()
	}
	if r.blurSrcView != nil {
		r.blurSrcView.Destroy()
	}
	if r.blurSrcTexture != nil {
		r.blurSrcTexture.Destroy()
	}
	if r.blurTmpView != nil {
		r.blurTmpView.Destroy()
	}
	if r.blurTmpTexture != nil {
		r.blurTmpTexture.Destroy()
	}
	if r.blurDstView != nil {
		r.blurDstView.Destroy()
	}
	if r.blurDstTexture != nil {
		r.blurDstTexture.Destroy()
	}
	// Surface must be released before Device — DX12 needs the command queue for waitForGPU.
	if r.surface != nil {
		r.surface.Destroy()
	}
	r.device.Destroy()
	r.instance.Destroy()
}

func (r *WGPURenderer) resizeAtlasTexture() {
	r.atlasView.Destroy()
	r.atlasTexture.Destroy()
	r.atlasW, r.atlasH = r.atlas.Width, r.atlas.Height
	r.atlasTexture = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "glyph-atlas",
		Size:   wgpu.Extent3D{Width: uint32(r.atlasW), Height: uint32(r.atlasH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatR8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	r.atlasView = r.atlasTexture.CreateView()
	// Recreate text bind group with new texture view.
	r.textBindGroup.Destroy()
	r.textBindGroup = r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "text-bind-group",
		Layout: r.textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.projBuffer, Size: 64},
			{Binding: 1, Texture: r.atlasView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})
	r.atlas.Dirty = true
}

func (r *WGPURenderer) resizeMSDFTexture() {
	r.msdfView.Destroy()
	r.msdfTexture.Destroy()
	r.msdfW, r.msdfH = r.atlas.MSDFWidth, r.atlas.MSDFHeight
	r.msdfTexture = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "msdf-atlas",
		Size:   wgpu.Extent3D{Width: uint32(r.msdfW), Height: uint32(r.msdfH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatRGBA8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	r.msdfView = r.msdfTexture.CreateView()
	// Recreate MSDF bind group with new texture view.
	r.msdfBindGroup.Destroy()
	r.msdfBindGroup = r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "msdf-bind-group",
		Layout: r.textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.msdfUniBuffer, Size: 80},
			{Binding: 1, Texture: r.msdfView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})
	r.atlas.MSDFDirty = true
	r.updateMSDFUniforms()
}

func (r *WGPURenderer) resizeBlurTextures() {
	if r.blurSrcTexture != nil {
		r.blurSrcView.Destroy()
		r.blurSrcTexture.Destroy()
		r.blurTmpView.Destroy()
		r.blurTmpTexture.Destroy()
		r.blurDstView.Destroy()
		r.blurDstTexture.Destroy()
	}
	r.blurW, r.blurH = r.width, r.height
	blurUsage := wgpu.TextureUsageCopySrc | wgpu.TextureUsageCopyDst | wgpu.TextureUsageTextureBinding | wgpu.TextureUsageRenderAttachment
	r.blurSrcTexture = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "blur-src",
		Size:   wgpu.Extent3D{Width: uint32(r.blurW), Height: uint32(r.blurH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  blurUsage,
	})
	r.blurSrcView = r.blurSrcTexture.CreateView()
	r.blurTmpTexture = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "blur-tmp",
		Size:   wgpu.Extent3D{Width: uint32(r.blurW), Height: uint32(r.blurH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  blurUsage,
	})
	r.blurTmpView = r.blurTmpTexture.CreateView()
	r.blurDstTexture = r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "blur-dst",
		Size:   wgpu.Extent3D{Width: uint32(r.blurW), Height: uint32(r.blurH), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  blurUsage,
	})
	r.blurDstView = r.blurDstTexture.CreateView()
}

// ensureGPUBuffer grows a GPU buffer if the needed capacity exceeds the current one.
func (r *WGPURenderer) ensureGPUBuffer(buf *wgpu.Buffer, cap *uint64, needed uint64, label string, usage wgpu.BufferUsage) {
	if needed <= *cap {
		return
	}
	newCap := needed * 2
	(*buf).Destroy()
	*buf = r.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: label, Size: newCap, Usage: usage,
	})
	*cap = newCap
}

// drawClipBatches iterates over ClipBatches, setting scissor rects and drawing
// the appropriate ranges of rects/text/MSDF from the pre-uploaded GPU buffers.
//
// totalRects/totalTextGlyphs/totalMSDFGlyphs are counts for this layer (main or overlay).
// gpuRectOffset/gpuTextGlyphOffset/gpuMSDFGlyphOffset are the offsets into the
// concatenated GPU buffers (0 for main, mainCount for overlay).
func (r *WGPURenderer) drawClipBatches(
	renderPass wgpu.RenderPass,
	batches []draw.ClipBatch,
	totalRects, totalTextGlyphs, totalMSDFGlyphs, totalShadows int,
	gpuRectOffset, gpuTextGlyphOffset, gpuMSDFGlyphOffset, gpuShadowOffset int,
	rectBufSize, glyphBufSize, shadowBufSize uint64,
	gradientRects []draw.DrawGradientRect, gradBindGroupOffset int,
	vpW, vpH uint32,
) {
	if totalRects == 0 && totalTextGlyphs == 0 && totalMSDFGlyphs == 0 && len(gradientRects) == 0 && totalShadows == 0 {
		return
	}

	// Pipeline state tracking for draw-call merging.
	var lastPipeline int // 0=none, 1=rect, 2=text, 3=msdf, 4=shadow

	setShadowPipeline := func() {
		if lastPipeline != 4 {
			renderPass.SetPipeline(r.shadowPipeline)
			renderPass.SetBindGroup(0, r.projBindGroup)
			renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
			renderPass.SetVertexBuffer(1, r.shadowInstBuffer, 0, shadowBufSize)
			lastPipeline = 4
		}
	}

	setRectPipeline := func() {
		if lastPipeline != 1 {
			renderPass.SetPipeline(r.rectPipeline)
			renderPass.SetBindGroup(0, r.projBindGroup)
			renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
			renderPass.SetVertexBuffer(1, r.rectInstBuffer, 0, rectBufSize)
			lastPipeline = 1
		}
	}

	setTextPipeline := func() {
		if lastPipeline != 2 {
			renderPass.SetPipeline(r.textInstPipeline)
			renderPass.SetBindGroup(0, r.textBindGroup)
			renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
			renderPass.SetVertexBuffer(1, r.glyphInstBuffer, 0, glyphBufSize)
			lastPipeline = 2
		}
	}

	setMSDFPipeline := func() {
		if lastPipeline != 3 {
			renderPass.SetPipeline(r.msdfInstPipeline)
			renderPass.SetBindGroup(0, r.msdfBindGroup)
			renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
			renderPass.SetVertexBuffer(1, r.glyphInstBuffer, 0, glyphBufSize)
			lastPipeline = 3
		}
	}

	// No clip batches → draw everything as a single full-viewport batch.
	if len(batches) == 0 {
		renderPass.SetScissorRect(0, 0, vpW, vpH)
		if totalShadows > 0 {
			setShadowPipeline()
			renderPass.DrawInstanced(6, uint32(totalShadows), 0, uint32(gpuShadowOffset))
		}
		if totalRects > 0 {
			setRectPipeline()
			renderPass.DrawInstanced(6, uint32(totalRects), 0, uint32(gpuRectOffset))
		}
		if totalTextGlyphs > 0 {
			setTextPipeline()
			renderPass.DrawInstanced(6, uint32(totalTextGlyphs), 0, uint32(gpuTextGlyphOffset))
		}
		if totalMSDFGlyphs > 0 {
			setMSDFPipeline()
			renderPass.DrawInstanced(6, uint32(totalMSDFGlyphs), 0, uint32(gpuMSDFGlyphOffset))
		}
		for gi := range gradientRects {
			r.drawGradientRect(renderPass, gradBindGroupOffset+gi)
			lastPipeline = 0
		}
		return
	}

	// Batch indices are scene-list indices (e.g., batch.RectIdx is an index
	// into scene.Rects or scene.OverlayRects). The first batch's index is
	// the base for this layer.
	baseRectIdx := batches[0].RectIdx
	baseTextIdx := batches[0].TextIdx
	baseMSDFIdx := batches[0].MSDFIdx
	baseGradIdx := batches[0].GradientIdx
	baseShadowIdx := batches[0].ShadowIdx

	for i, batch := range batches {
		// Set scissor rect.
		if batch.FullViewport {
			renderPass.SetScissorRect(0, 0, vpW, vpH)
		} else {
			sx := uint32(batch.Clip.X)
			sy := uint32(batch.Clip.Y)
			sw := uint32(batch.Clip.W)
			sh := uint32(batch.Clip.H)
			if sx+sw > vpW {
				sw = vpW - sx
			}
			if sy+sh > vpH {
				sh = vpH - sy
			}
			renderPass.SetScissorRect(sx, sy, sw, sh)
		}

		// Compute draw counts from batch boundaries.
		var endRectIdx, endTextIdx, endMSDFIdx, endGradIdx, endShadowIdx int
		if i+1 < len(batches) {
			endRectIdx = batches[i+1].RectIdx
			endTextIdx = batches[i+1].TextIdx
			endMSDFIdx = batches[i+1].MSDFIdx
			endGradIdx = batches[i+1].GradientIdx
			endShadowIdx = batches[i+1].ShadowIdx
		} else {
			endRectIdx = baseRectIdx + totalRects
			endTextIdx = baseTextIdx + totalTextGlyphs
			endMSDFIdx = baseMSDFIdx + totalMSDFGlyphs
			endGradIdx = baseGradIdx + len(gradientRects)
			endShadowIdx = baseShadowIdx + totalShadows
		}

		nRects := uint32(endRectIdx - batch.RectIdx)
		nTextGlyphs := uint32(endTextIdx - batch.TextIdx)
		nMSDFGlyphs := uint32(endMSDFIdx - batch.MSDFIdx)
		nShadows := uint32(endShadowIdx - batch.ShadowIdx)

		// GPU offsets: scene index relative to base + layer offset in GPU buffer.
		rectFirst := uint32(batch.RectIdx-baseRectIdx) + uint32(gpuRectOffset)
		textFirst := uint32(batch.TextIdx-baseTextIdx) + uint32(gpuTextGlyphOffset)
		msdfFirst := uint32(batch.MSDFIdx-baseMSDFIdx) + uint32(gpuMSDFGlyphOffset)
		shadowFirst := uint32(batch.ShadowIdx-baseShadowIdx) + uint32(gpuShadowOffset)

		// Draw shadows BEFORE rects (shadows go behind content).
		if nShadows > 0 {
			setShadowPipeline()
			renderPass.DrawInstanced(6, nShadows, 0, shadowFirst)
		}
		if nRects > 0 {
			setRectPipeline()
			renderPass.DrawInstanced(6, nRects, 0, rectFirst)
		}
		if nTextGlyphs > 0 {
			setTextPipeline()
			renderPass.DrawInstanced(6, nTextGlyphs, 0, textFirst)
		}
		if nMSDFGlyphs > 0 {
			setMSDFPipeline()
			renderPass.DrawInstanced(6, nMSDFGlyphs, 0, msdfFirst)
		}

		// Draw gradient rects for this batch (1 draw call per gradient).
		gradStart := batch.GradientIdx - baseGradIdx
		gradEnd := endGradIdx - baseGradIdx
		for gi := gradStart; gi < gradEnd && gi < len(gradientRects); gi++ {
			r.drawGradientRect(renderPass, gradBindGroupOffset+gi)
			lastPipeline = 0 // gradient changes pipeline state
		}
	}
}

// packGradientUniform writes 76 floats (304 bytes) of gradient uniform data into dst.
func packGradientUniform(dst []float32, gr draw.DrawGradientRect) {
	dst[0] = float32(gr.X)
	dst[1] = float32(gr.Y)
	dst[2] = float32(gr.W)
	dst[3] = float32(gr.H)
	dst[4] = gr.Radius
	if gr.Kind == draw.PaintRadialGradient {
		dst[5] = 1.0
	}
	dst[6] = float32(gr.StopCount)
	if gr.Kind == draw.PaintLinearGradient {
		dst[8] = gr.StartX
		dst[9] = gr.StartY
		dst[10] = gr.EndX
		dst[11] = gr.EndY
	} else {
		dst[8] = gr.CenterX
		dst[9] = gr.CenterY
		dst[10] = gr.GradRadius
	}
	for i := 0; i < gr.StopCount && i < 8; i++ {
		base := 12 + i*8
		dst[base+0] = gr.Stops[i].Offset
		dst[base+1] = gr.Stops[i].Color.R
		dst[base+2] = gr.Stops[i].Color.G
		dst[base+3] = gr.Stops[i].Color.B
		dst[base+4] = gr.Stops[i].Color.A
	}
}

// drawGradientRect draws one gradient using the pre-built bind group at the given index.
func (r *WGPURenderer) drawGradientRect(renderPass wgpu.RenderPass, bindGroupIdx int) {
	if bindGroupIdx < 0 || bindGroupIdx >= len(r.gradBindGroups) {
		return
	}
	renderPass.SetPipeline(r.gradPipeline)
	renderPass.SetBindGroup(0, r.projBindGroup)
	renderPass.SetBindGroup(1, r.gradBindGroups[bindGroupIdx])
	renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
	renderPass.Draw(6, 1, 0, 0)
}

// drawSurfaces blits registered surface textures into the render pass.
func (r *WGPURenderer) drawSurfaces(renderPass wgpu.RenderPass, surfaces []draw.DrawSurface, vpW, vpH uint32) {
	for _, s := range surfaces {
		view, ok := r.surfaceTextures[s.TextureID]
		if !ok || s.TextureID == 0 || view == nil {
			continue
		}

		// Upload per-surface instance data (rect).
		instData := []float32{float32(s.X), float32(s.Y), float32(s.W), float32(s.H)}
		r.surfInstBuffer.Write(r.queue, float32SliceToBytes(instData))

		// Create per-surface bind group for texture.
		surfBindGroup := r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "surf-per-draw",
			Layout: r.surfTexLayout,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Texture: view},
				{Binding: 1, Sampler: r.surfSampler},
			},
		})

		renderPass.SetScissorRect(0, 0, vpW, vpH)
		renderPass.SetPipeline(r.surfPipeline)
		renderPass.SetBindGroup(0, r.projBindGroup)
		renderPass.SetBindGroup(1, surfBindGroup)
		renderPass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
		renderPass.SetVertexBuffer(1, r.surfInstBuffer, 0, 16)
		renderPass.Draw(6, 1, 0, 0)

		surfBindGroup.Destroy()
	}
}

// appendGlyphInstance appends a single glyph's instance data (12 floats) to buf.
func appendGlyphInstance(buf []float32, g draw.TexturedGlyph, atlasW, atlasH float32) []float32 {
	u0 := float32(g.SrcX) / atlasW
	v0 := float32(g.SrcY) / atlasH
	u1 := float32(g.SrcX+g.SrcW) / atlasW
	v1 := float32(g.SrcY+g.SrcH) / atlasH
	return append(buf,
		g.DstX, g.DstY, g.DstW, g.DstH, // glyph_rect
		u0, v0, u1, v1,                   // glyph_uv
		g.Color.R, g.Color.G, g.Color.B, g.Color.A, // color
	)
}

func (r *WGPURenderer) updateProjection() {
	proj := wgpuOrthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	r.projBuffer.Write(r.queue, float32SliceToBytes(proj[:]))
	r.updateMSDFUniforms()
}

func (r *WGPURenderer) updateMSDFUniforms() {
	proj := wgpuOrthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	// 20 floats: mat4x4 (16) + atlas_size vec4 (4)
	// atlas_size.xy = texture dimensions, atlas_size.zw = px_range (replicated for dot product)
	pxRange := float32(text.MSDFPxRange)
	var data [20]float32
	copy(data[:16], proj[:])
	data[16] = float32(r.msdfW)
	data[17] = float32(r.msdfH)
	data[18] = pxRange
	data[19] = pxRange
	r.msdfUniBuffer.Write(r.queue, float32SliceToBytes(data[:]))
}

func wgpuOrthoMatrix(left, right, bottom, top, near, far float32) [16]float32 {
	dx := right - left
	dy := top - bottom
	dz := far - near
	return [16]float32{
		2 / dx, 0, 0, 0,
		0, 2 / dy, 0, 0,
		0, 0, -2 / dz, 0,
		-(right + left) / dx, -(top + bottom) / dy, -(far + near) / dz, 1,
	}
}

func float32SliceToBytes(s []float32) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
}

// Ensure imports are used.
var (
	_ = math.MaxFloat32
	_ = fmt.Sprintf
)
