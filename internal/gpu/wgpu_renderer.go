//go:build !nogui && !windows

package gpu

import (
	"fmt"
	"math"
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
//   - Atlas-based textured glyph rendering
//   - MSDF text rendering for large text sizes
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
	rectPipeline   wgpu.RenderPipeline
	textPipeline   wgpu.RenderPipeline
	msdfPipeline   wgpu.RenderPipeline

	// Shared resources
	projBuffer     wgpu.Buffer
	rectVertBuffer wgpu.Buffer
	rectInstBuffer wgpu.Buffer
	textVertBuffer wgpu.Buffer
	atlasTexture   wgpu.Texture
	atlasView      wgpu.TextureView
	atlasSampler   wgpu.Sampler
	msdfTexture    wgpu.Texture
	msdfView       wgpu.TextureView

	// Bind groups
	projBindGroup  wgpu.BindGroup
	textBindGroup  wgpu.BindGroup
	msdfBindGroup  wgpu.BindGroup

	// State tracking
	inited    bool
	surfaceOK bool
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

	// Create rect vertex buffer (unit quad: 6 vertices * 2 floats * 4 bytes = 48 bytes).
	r.rectVertBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "rect-verts",
		Size:  48,
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Upload unit quad vertices.
	quadVerts := []float32{0, 0, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1}
	r.rectVertBuffer.Write(r.queue, float32SliceToBytes(quadVerts))

	// Create rect instance buffer (dynamic, resized as needed).
	r.rectInstBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "rect-instances",
		Size:  1024 * 9 * 4, // 1024 rects * 9 floats * 4 bytes
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Create text vertex buffer (dynamic).
	r.textVertBuffer = device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "text-verts",
		Size:  4096 * 6 * 4 * 4, // 4096 glyphs * 6 verts * 4 floats * 4 bytes
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})

	// Create atlas texture (initially 512x512, single-channel).
	r.atlasTexture = device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "glyph-atlas",
		Size:   wgpu.Extent3D{Width: 512, Height: 512, DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatR8Unorm,
		Usage:  wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	r.atlasView = r.atlasTexture.CreateView()

	// Create MSDF atlas texture (initially 512x512, RGBA).
	r.msdfTexture = device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "msdf-atlas",
		Size:   wgpu.Extent3D{Width: 512, Height: 512, DepthOrArrayLayers: 1},
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
		Label:  "text-shader",
		Source: wgslTextShader,
	})
	defer textShader.Destroy()

	msdfShader := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "msdf-shader",
		Source: wgslMSDFShader,
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

	textLayout := device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "text-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})
	defer textLayout.Destroy()

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

	// Create render pipelines.
	r.rectPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "rect-pipeline",
		Vertex: wgpu.VertexState{
			Module:     rectShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 8, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
				}},
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

	r.textPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "text-pipeline",
		Vertex: wgpu.VertexState{
			Module:     textShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 16, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0}, // pos
					{Format: wgpu.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1}, // uv
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     textShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{textLayout},
	})

	r.msdfPipeline = device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "msdf-pipeline",
		Vertex: wgpu.VertexState{
			Module:     msdfShader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{ArrayStride: 16, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
					{Format: wgpu.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1},
				}},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     msdfShader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: []wgpu.BindGroupLayout{textLayout},
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
		Layout: textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.projBuffer, Size: 64},
			{Binding: 1, Texture: r.atlasView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})

	r.msdfBindGroup = device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "msdf-bind-group",
		Layout: textLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.projBuffer, Size: 64},
			{Binding: 1, Texture: r.msdfView},
			{Binding: 2, Sampler: r.atlasSampler},
		},
	})

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
}

// BeginFrame starts a new frame.
func (r *WGPURenderer) BeginFrame() {
	// Frame setup is handled in Draw when we acquire the surface texture.
}

// Draw renders the current scene using wgpu.
func (r *WGPURenderer) Draw(scene draw.Scene) {
	if !r.inited || !r.surfaceOK {
		return
	}

	// Acquire current surface texture.
	textureView, err := r.surface.GetCurrentTexture()
	if err != nil {
		return
	}

	// Create command encoder.
	encoder := r.device.CreateCommandEncoder()

	// Begin render pass with clear.
	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    textureView,
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

	// Draw rounded rects via instanced rendering.
	if len(scene.Rects) > 0 {
		r.drawRects(renderPass, scene.Rects)
	}

	// Draw textured glyphs.
	if len(scene.TexturedGlyphs) > 0 {
		r.drawTexturedGlyphs(renderPass, scene.TexturedGlyphs)
	}

	// Draw MSDF glyphs.
	if len(scene.MSDFGlyphs) > 0 {
		r.drawMSDFGlyphs(renderPass, scene.MSDFGlyphs)
	}

	// Overlay pass.
	if len(scene.OverlayRects) > 0 {
		r.drawRects(renderPass, scene.OverlayRects)
	}
	if len(scene.OverlayTexturedGlyphs) > 0 {
		r.drawTexturedGlyphs(renderPass, scene.OverlayTexturedGlyphs)
	}
	if len(scene.OverlayMSDFGlyphs) > 0 {
		r.drawMSDFGlyphs(renderPass, scene.OverlayMSDFGlyphs)
	}

	renderPass.End()

	// Submit.
	cmdBuffer := encoder.Finish()
	r.queue.Submit(cmdBuffer)
}

// EndFrame presents the rendered frame.
func (r *WGPURenderer) EndFrame() {
	if r.surfaceOK {
		r.surface.Present()
	}
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
	r.rectVertBuffer.Destroy()
	r.rectInstBuffer.Destroy()
	r.textVertBuffer.Destroy()
	r.atlasView.Destroy()
	r.atlasTexture.Destroy()
	r.msdfView.Destroy()
	r.msdfTexture.Destroy()
	r.atlasSampler.Destroy()
	r.rectPipeline.Destroy()
	r.textPipeline.Destroy()
	r.msdfPipeline.Destroy()
	r.device.Destroy()
	if r.surface != nil {
		r.surface.Destroy()
	}
	r.instance.Destroy()
}

func (r *WGPURenderer) drawRects(pass wgpu.RenderPass, rects []draw.DrawRect) {
	if len(rects) == 0 {
		return
	}

	// Build instance data: 9 floats per rect (x, y, w, h, r, g, b, a, radius).
	instances := make([]float32, 0, len(rects)*9)
	for _, rect := range rects {
		instances = append(instances,
			float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
			rect.Color.R, rect.Color.G, rect.Color.B, rect.Color.A,
			rect.Radius,
		)
	}

	r.rectInstBuffer.Write(r.queue, float32SliceToBytes(instances))

	pass.SetPipeline(r.rectPipeline)
	pass.SetBindGroup(0, r.projBindGroup)
	pass.SetVertexBuffer(0, r.rectVertBuffer, 0, 48)
	pass.SetVertexBuffer(1, r.rectInstBuffer, 0, uint64(len(instances)*4))
	pass.DrawInstanced(6, uint32(len(rects)), 0, 0)
}

func (r *WGPURenderer) drawTexturedGlyphs(pass wgpu.RenderPass, glyphs []draw.TexturedGlyph) {
	if r.atlas == nil || len(glyphs) == 0 {
		return
	}

	// Upload atlas if dirty.
	if r.atlas.Dirty {
		r.atlasTexture.Write(r.queue, r.atlas.Image.Pix, uint32(r.atlas.Image.Stride))
		r.atlas.Dirty = false
	}

	atlasW := float32(r.atlas.Width)
	atlasH := float32(r.atlas.Height)

	// Build vertex data: 6 vertices per glyph, 4 floats per vertex.
	vertices := make([]float32, 0, len(glyphs)*6*4)
	for _, g := range glyphs {
		x0, y0 := g.DstX, g.DstY
		x1, y1 := g.DstX+g.DstW, g.DstY+g.DstH
		u0 := float32(g.SrcX) / atlasW
		v0 := float32(g.SrcY) / atlasH
		u1 := float32(g.SrcX+g.SrcW) / atlasW
		v1 := float32(g.SrcY+g.SrcH) / atlasH

		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,
			x1, y0, u1, v0,
			x1, y1, u1, v1,
			x0, y1, u0, v1,
		)
	}

	r.textVertBuffer.Write(r.queue, float32SliceToBytes(vertices))

	pass.SetPipeline(r.textPipeline)
	pass.SetBindGroup(0, r.textBindGroup)
	pass.SetVertexBuffer(0, r.textVertBuffer, 0, uint64(len(vertices)*4))
	pass.Draw(uint32(len(vertices)/4), 1, 0, 0)
}

func (r *WGPURenderer) drawMSDFGlyphs(pass wgpu.RenderPass, glyphs []draw.TexturedGlyph) {
	if r.atlas == nil || len(glyphs) == 0 {
		return
	}

	if r.atlas.MSDFDirty {
		r.msdfTexture.Write(r.queue, r.atlas.MSDFImage.Pix, uint32(r.atlas.MSDFImage.Stride))
		r.atlas.MSDFDirty = false
	}

	atlasW := float32(r.atlas.MSDFWidth)
	atlasH := float32(r.atlas.MSDFHeight)

	vertices := make([]float32, 0, len(glyphs)*6*4)
	for _, g := range glyphs {
		x0, y0 := g.DstX, g.DstY
		x1, y1 := g.DstX+g.DstW, g.DstY+g.DstH
		u0 := float32(g.SrcX) / atlasW
		v0 := float32(g.SrcY) / atlasH
		u1 := float32(g.SrcX+g.SrcW) / atlasW
		v1 := float32(g.SrcY+g.SrcH) / atlasH

		vertices = append(vertices,
			x0, y0, u0, v0,
			x1, y0, u1, v0,
			x0, y1, u0, v1,
			x1, y0, u1, v0,
			x1, y1, u1, v1,
			x0, y1, u0, v1,
		)
	}

	r.textVertBuffer.Write(r.queue, float32SliceToBytes(vertices))

	pass.SetPipeline(r.msdfPipeline)
	pass.SetBindGroup(0, r.msdfBindGroup)
	pass.SetVertexBuffer(0, r.textVertBuffer, 0, uint64(len(vertices)*4))
	pass.Draw(uint32(len(vertices)/4), 1, 0, 0)
}

func (r *WGPURenderer) updateProjection() {
	proj := wgpuOrthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	r.projBuffer.Write(r.queue, float32SliceToBytes(proj[:]))
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
