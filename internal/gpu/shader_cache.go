//go:build !nogui && (!windows || gogpu)

package gpu

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/internal/wgpu"
)

// ShaderCache compiles and caches custom shader pipelines on demand.
// Built-in effects (Noise, Plasma, Voronoi) are pre-compiled at init.
// Custom WGSL fragment code is compiled on first use and cached by
// content hash.
type ShaderCache struct {
	device  wgpu.Device
	blend   *wgpu.BlendState
	entries map[string]wgpu.RenderPipeline // cache key → pipeline

	// Bind group layout for custom shader params (group 1).
	paramsLayout wgpu.BindGroupLayout

	// Bind group layout for shader+image (group 2, optional).
	imageLayout wgpu.BindGroupLayout
}

// NewShaderCache creates a shader cache and pre-compiles built-in effects.
func NewShaderCache(device wgpu.Device, projLayout wgpu.BindGroupLayout, blend *wgpu.BlendState) *ShaderCache {
	paramsLayout := device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "custom-shader-params-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})

	imageLayout := device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "custom-shader-image-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Texture: &wgpu.TextureBindingLayout{SampleType: wgpu.TextureSampleTypeFloat, ViewDimension: wgpu.TextureViewDimension2D}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &wgpu.SamplerBindingLayout{}},
		},
	})

	sc := &ShaderCache{
		device:       device,
		blend:        blend,
		entries:      make(map[string]wgpu.RenderPipeline),
		paramsLayout: paramsLayout,
		imageLayout:  imageLayout,
	}

	// Pre-compile built-in effects.
	builtins := map[string]string{
		"_builtin:noise":   wgslNoiseShader,
		"_builtin:plasma":  wgslPlasmaShader,
		"_builtin:voronoi": wgslVoronoiShader,
	}
	for key, source := range builtins {
		pipeline := sc.compilePipeline(key, source, projLayout, false)
		if pipeline != nil {
			sc.entries[key] = pipeline
		}
	}

	return sc
}

// Get returns the cached pipeline for the given shader description.
// If the shader is not yet compiled, it compiles and caches it.
func (sc *ShaderCache) Get(desc *draw.ShaderDesc) wgpu.RenderPipeline {
	key := sc.cacheKey(desc)
	if pipeline, ok := sc.entries[key]; ok {
		return pipeline
	}

	// Custom shader — wrap user fragment code in the shader template.
	hasImage := desc.Image != 0
	var fullSource string
	if hasImage {
		fullSource = wgslCustomShaderImagePrefix + desc.Source
	} else {
		fullSource = wgslCustomShaderPrefix + desc.Source
	}

	// We need projLayout but don't store it — compile with nil and
	// let the pipeline infer from the shader. For custom shaders,
	// we use the paramsLayout as group 1.
	pipeline := sc.compileCustomPipeline(key, fullSource, hasImage)
	if pipeline != nil {
		sc.entries[key] = pipeline
	}
	return pipeline
}

// ParamsLayout returns the bind group layout for custom shader uniforms (group 1).
func (sc *ShaderCache) ParamsLayout() wgpu.BindGroupLayout {
	return sc.paramsLayout
}

// ImageLayout returns the bind group layout for shader image input (group 2).
func (sc *ShaderCache) ImageLayout() wgpu.BindGroupLayout {
	return sc.imageLayout
}

// Destroy releases all cached pipelines and layouts.
func (sc *ShaderCache) Destroy() {
	for _, p := range sc.entries {
		p.Destroy()
	}
	sc.paramsLayout.Destroy()
	sc.imageLayout.Destroy()
}

func (sc *ShaderCache) cacheKey(desc *draw.ShaderDesc) string {
	if desc.Source == "" {
		switch desc.Effect {
		case draw.ShaderEffectNoise:
			return "_builtin:noise"
		case draw.ShaderEffectPlasma:
			return "_builtin:plasma"
		case draw.ShaderEffectVoronoi:
			return "_builtin:voronoi"
		}
		return ""
	}
	// Hash custom source for cache key.
	h := sha256.Sum256([]byte(desc.Source))
	prefix := "custom"
	if desc.Image != 0 {
		prefix = "custom-img"
	}
	return fmt.Sprintf("%s:%x", prefix, h[:8])
}

func (sc *ShaderCache) compilePipeline(label, source string, projLayout wgpu.BindGroupLayout, hasImage bool) wgpu.RenderPipeline {
	shader := sc.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  label,
		Source: source,
	})
	if shader == nil {
		log.Printf("shader_cache: failed to compile %q", label)
		return nil
	}
	defer shader.Destroy()

	unitQuadLayout := wgpu.VertexBufferLayout{
		ArrayStride: 8, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
		},
	}
	instLayout := wgpu.VertexBufferLayout{
		ArrayStride: 16, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1}, // rect
		},
	}

	layouts := []wgpu.BindGroupLayout{projLayout, sc.paramsLayout}
	if hasImage {
		layouts = append(layouts, sc.imageLayout)
	}

	return sc.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: label + "-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{{}, {}, unitQuadLayout, instLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: sc.blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: layouts,
	})
}

func (sc *ShaderCache) compileCustomPipeline(key, source string, hasImage bool) wgpu.RenderPipeline {
	shader := sc.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:  key,
		Source: source,
	})
	if shader == nil {
		log.Printf("shader_cache: failed to compile custom shader %q", key)
		return nil
	}
	defer shader.Destroy()

	unitQuadLayout := wgpu.VertexBufferLayout{
		ArrayStride: 8, StepMode: wgpu.VertexStepModeVertex, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
		},
	}
	instLayout := wgpu.VertexBufferLayout{
		ArrayStride: 16, StepMode: wgpu.VertexStepModeInstance, Attributes: []wgpu.VertexAttribute{
			{Format: wgpu.VertexFormatFloat32x4, Offset: 0, ShaderLocation: 1},
		},
	}

	// For custom shaders we need projLayout from group 0.
	// Since we don't store it, we create a temporary one.
	projLayout := sc.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "custom-proj-layout",
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &wgpu.BufferBindingLayout{Type: wgpu.BufferBindingTypeUniform}},
		},
	})
	defer projLayout.Destroy()

	layouts := []wgpu.BindGroupLayout{projLayout, sc.paramsLayout}
	if hasImage {
		layouts = append(layouts, sc.imageLayout)
	}

	return sc.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: key + "-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{{}, {}, unitQuadLayout, instLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets:    []wgpu.ColorTargetState{{Format: wgpu.TextureFormatBGRA8Unorm, Blend: sc.blend}},
		},
		Primitive:        wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		BindGroupLayouts: layouts,
	})
}
