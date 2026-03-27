//go:build !nogui && gogpu

package main

import (
	"math"
	"time"
	"unsafe"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/internal/wgpu"
	"github.com/timzifer/lux/ui"
)

// cubeWGSL is the WGSL shader for the 3D RGB cube.
const cubeWGSL = `
struct Uniforms {
    mvp: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> u: Uniforms;

struct VsIn {
    @location(0) pos: vec3<f32>,
    @location(1) color: vec3<f32>,
};

struct VsOut {
    @builtin(position) pos: vec4<f32>,
    @location(0) color: vec3<f32>,
};

@vertex
fn vs_main(in: VsIn) -> VsOut {
    var out: VsOut;
    out.pos = u.mvp * vec4<f32>(in.pos, 1.0);
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VsOut) -> @location(0) vec4<f32> {
    return vec4<f32>(in.color, 1.0);
}
`

// PyramidSurface renders a rotating RGB cube via WGPU offscreen rendering.
type PyramidSurface struct {
	// WGPU resources (lazy-init).
	device   wgpu.Device
	queue    wgpu.Queue
	renderer *gpu.WGPURenderer

	// Offscreen render targets.
	colorTex  wgpu.Texture
	colorView wgpu.TextureView
	depthTex  wgpu.Texture
	depthView wgpu.TextureView

	// Cube pipeline.
	pipeline  wgpu.RenderPipeline
	vertBuf   wgpu.Buffer
	idxBuf    wgpu.Buffer
	mvpBuf    wgpu.Buffer
	bindGroup wgpu.BindGroup

	texW, texH int
	inited     bool

	// Rotation state.
	angleX, angleY float32
	dragX, dragY   float32
	dragging       bool
	lastMouseX     float32
	lastMouseY     float32
	lastTime       time.Time

	nextToken ui.FrameToken
	lastTexID draw.TextureID
}

// NewPyramidSurface creates a new WGPU RGB cube surface provider.
func NewPyramidSurface() *PyramidSurface {
	return &PyramidSurface{lastTime: time.Now()}
}

// SetRenderer connects the surface to the WGPU renderer for device access.
func (p *PyramidSurface) SetRenderer(r *gpu.WGPURenderer) {
	p.renderer = r
}

// Tick advances auto-rotation by dt.
func (p *PyramidSurface) Tick(dt time.Duration) {
	sec := float32(dt.Seconds())
	p.angleX += 0.3 * sec
	p.angleY += 0.5 * sec
}

// ── SurfaceProvider implementation ──────────────────────────────

func (p *PyramidSurface) AcquireFrame(bounds draw.Rect) (draw.TextureID, ui.FrameToken) {
	w := int(bounds.W)
	h := int(bounds.H)
	if w <= 0 || h <= 0 || p.renderer == nil {
		return 0, 0
	}

	if !p.inited {
		p.initWGPU()
		if !p.inited {
			return 0, 0
		}
	}

	// Resize offscreen textures if needed.
	if w != p.texW || h != p.texH {
		p.resizeOffscreen(w, h)
	}

	// Build MVP matrix.
	aspect := float32(w) / float32(h)
	proj := perspectiveMatrix(45.0*math.Pi/180.0, aspect, 0.1, 100.0)
	view := translationMatrix(0, 0, -4)
	model := matMul4(rotationX(p.angleX+p.dragX), rotationY(p.angleY+p.dragY))
	mvp := matMul4(proj, matMul4(view, model))

	// Upload MVP.
	p.mvpBuf.Write(p.queue, float32ToBytes(mvp[:]))

	// Render to offscreen texture.
	encoder := p.device.CreateCommandEncoder()
	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:       p.colorView,
				LoadOp:     wgpu.LoadOpClear,
				StoreOp:    wgpu.StoreOpStore,
				ClearValue: wgpu.Color{R: 0.08, G: 0.08, B: 0.12, A: 1.0},
			},
		},
		DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
			View:            p.depthView,
			DepthLoadOp:     wgpu.LoadOpClear,
			DepthStoreOp:    wgpu.StoreOpStore,
			DepthClearValue: 1.0,
		},
	})

	renderPass.SetPipeline(p.pipeline)
	renderPass.SetBindGroup(0, p.bindGroup)
	renderPass.SetVertexBuffer(0, p.vertBuf, 0, 8*6*4) // 8 verts * 6 floats * 4 bytes
	renderPass.SetIndexBuffer(p.idxBuf, wgpu.IndexFormatUint32, 0, 36*4)
	renderPass.DrawIndexed(36, 1, 0, 0, 0)

	renderPass.End()
	cmdBuf := encoder.Finish()
	p.queue.Submit(cmdBuf)

	// Register the texture view with the renderer.
	p.nextToken++
	if p.nextToken == 0 {
		p.nextToken++
	}
	texID := draw.TextureID(p.nextToken)
	p.renderer.RegisterSurfaceTexture(texID, p.colorView)

	// Unregister previous texture ID.
	if p.lastTexID != 0 && p.lastTexID != texID {
		p.renderer.UnregisterSurfaceTexture(p.lastTexID)
	}
	p.lastTexID = texID

	return texID, p.nextToken
}

func (p *PyramidSurface) ReleaseFrame(_ ui.FrameToken) {}

func (p *PyramidSurface) HandleMsg(msg any) bool {
	switch m := msg.(type) {
	case ui.SurfaceMouseMsg:
		switch m.Action {
		case input.MousePress:
			if m.Button == input.MouseButtonLeft {
				p.dragging = true
				p.lastMouseX = m.Pos.X
				p.lastMouseY = m.Pos.Y
				return true
			}
		case input.MouseRelease:
			if m.Button == input.MouseButtonLeft {
				p.dragging = false
				return true
			}
		case input.MouseMove:
			if p.dragging {
				dx := m.Pos.X - p.lastMouseX
				dy := m.Pos.Y - p.lastMouseY
				p.dragY += dx * 0.01
				p.dragX += dy * 0.01
				p.lastMouseX = m.Pos.X
				p.lastMouseY = m.Pos.Y
				return true
			}
		}
	}
	return false
}

// ── WGPU initialization ─────────────────────────────────────────

func (p *PyramidSurface) initWGPU() {
	p.device = p.renderer.Device()
	p.queue = p.renderer.Queue()
	if p.device == nil || p.queue == nil {
		return
	}

	// Create cube shader.
	shader := p.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  "cube-shader",
		Source: cubeWGSL,
	})
	defer shader.Destroy()

	// MVP uniform buffer (64 bytes).
	p.mvpBuf = p.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "cube-mvp",
		Size:  64,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})

	// Bind group layout + bind group.
	bgl := p.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "cube-bgl",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})
	defer bgl.Destroy()

	p.bindGroup = p.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "cube-bg",
		Layout: bgl,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: p.mvpBuf, Size: 64},
		},
	})

	// Vertex buffer: 8 vertices × 6 floats (pos + color).
	type vert struct {
		x, y, z float32
		r, g, b float32
	}
	verts := []vert{
		{-0.5, -0.5, -0.5, 0, 0, 0}, // 0: black
		{+0.5, -0.5, -0.5, 1, 0, 0}, // 1: red
		{+0.5, +0.5, -0.5, 1, 1, 0}, // 2: yellow
		{-0.5, +0.5, -0.5, 0, 1, 0}, // 3: green
		{-0.5, -0.5, +0.5, 0, 0, 1}, // 4: blue
		{+0.5, -0.5, +0.5, 1, 0, 1}, // 5: magenta
		{+0.5, +0.5, +0.5, 1, 1, 1}, // 6: white
		{-0.5, +0.5, +0.5, 0, 1, 1}, // 7: cyan
	}
	vertData := make([]float32, 0, len(verts)*6)
	for _, v := range verts {
		vertData = append(vertData, v.x, v.y, v.z, v.r, v.g, v.b)
	}
	p.vertBuf = p.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "cube-verts",
		Size:  uint64(len(vertData) * 4),
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})
	p.vertBuf.Write(p.queue, float32ToBytes(vertData))

	// Index buffer: 36 indices.
	indices := []uint32{
		4, 5, 6, 6, 7, 4, // Front
		1, 0, 3, 3, 2, 1, // Back
		5, 1, 2, 2, 6, 5, // Right
		0, 4, 7, 7, 3, 0, // Left
		7, 6, 2, 2, 3, 7, // Top
		0, 1, 5, 5, 4, 0, // Bottom
	}
	p.idxBuf = p.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "cube-indices",
		Size:  uint64(len(indices) * 4),
		Usage: wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
	})
	p.idxBuf.Write(p.queue, uint32ToBytes(indices))

	// Render pipeline.
	p.pipeline = p.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "cube-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 24, // 6 floats × 4 bytes
					StepMode:    wgpu.VertexStepModeVertex,
					Attributes: []wgpu.VertexAttribute{
						{Format: wgpu.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},  // pos
						{Format: wgpu.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 1}, // color
					},
				},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm}},
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			CullMode: wgpu.CullModeBack,
			FrontFace: wgpu.FrontFaceCCW,
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth24Plus,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLess,
		},
		BindGroupLayouts: []wgpu.BindGroupLayout{bgl},
	})

	// Create initial offscreen textures.
	p.createOffscreen(400, 300)
	p.inited = true
}

func (p *PyramidSurface) createOffscreen(w, h int) {
	p.colorTex = p.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "cube-color",
		Size:   wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatBGRA8Unorm,
		Usage:  wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
	})
	p.colorView = p.colorTex.CreateView()

	p.depthTex = p.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:  "cube-depth",
		Size:   wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		Format: wgpu.TextureFormatDepth24Plus,
		Usage:  wgpu.TextureUsageRenderAttachment,
	})
	p.depthView = p.depthTex.CreateView()

	p.texW = w
	p.texH = h
}

func (p *PyramidSurface) resizeOffscreen(w, h int) {
	// Unregister before destroying — renderer still holds the view reference.
	if p.lastTexID != 0 && p.renderer != nil {
		p.renderer.UnregisterSurfaceTexture(p.lastTexID)
		p.lastTexID = 0
	}
	if p.colorView != nil {
		p.colorView.Destroy()
	}
	if p.colorTex != nil {
		p.colorTex.Destroy()
	}
	if p.depthView != nil {
		p.depthView.Destroy()
	}
	if p.depthTex != nil {
		p.depthTex.Destroy()
	}
	p.createOffscreen(w, h)
}

// ── Byte conversion helpers ──────────────────────────────────────

func float32ToBytes(s []float32) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
}

func uint32ToBytes(s []uint32) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
}
