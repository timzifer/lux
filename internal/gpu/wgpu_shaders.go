//go:build !nogui && !windows

package gpu

// WGSL shaders for the wgpu renderer, equivalent to the OpenGL 3.3 GLSL shaders.

// wgslRectShader renders rounded rectangles using SDF (Signed Distance Field).
// Instanced rendering: unit quad vertices + per-instance rect/color/radius data.
const wgslRectShader = `
struct Uniforms {
    proj: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct VertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,      // (x, y, w, h) in screen coords
    @location(2) color: vec4<f32>,     // RGBA color
    @location(3) radius: f32,          // corner radius
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) local_pos: vec2<f32>,
    @location(1) half_size: vec2<f32>,
    @location(2) color: vec4<f32>,
    @location(3) @interpolate(flat) radius: f32,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;

    // Expand quad by 0.5px for anti-aliasing.
    let expand = vec2<f32>(0.5);
    let expanded_size = in.rect.zw + expand * 2.0;
    let world_pos = in.rect.xy - expand + in.pos * expanded_size;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);

    out.half_size = in.rect.zw * 0.5;
    out.local_pos = (in.pos - 0.5) * expanded_size;
    out.color = in.color;
    out.radius = in.radius;
    return out;
}

fn rounded_box_sdf(p: vec2<f32>, b: vec2<f32>, r: f32) -> f32 {
    let q = abs(p) - b + r;
    return length(max(q, vec2<f32>(0.0))) - r;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let dist = rounded_box_sdf(in.local_pos, in.half_size, in.radius);
    let alpha = 1.0 - smoothstep(-0.5, 0.5, dist);
    if (alpha < 0.001) {
        discard;
    }
    return vec4<f32>(in.color.rgb, in.color.a * alpha);
}
`

// wgslTextShader renders atlas-based textured glyphs (bitmap text < 24px).
// Single-channel alpha atlas with per-batch color uniform.
const wgslTextShader = `
struct Uniforms {
    proj: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var atlas_texture: texture_2d<f32>;
@group(0) @binding(2) var atlas_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,
    @location(1) uv: vec2<f32>,
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.position = uniforms.proj * vec4<f32>(in.pos, 0.0, 1.0);
    out.uv = in.uv;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let a = textureSample(atlas_texture, atlas_sampler, in.uv).r;
    // Color is baked into vertex data for wgpu (unlike OpenGL uniform approach).
    // For now, use white — the color will be multiplied in the vertex stage.
    return vec4<f32>(1.0, 1.0, 1.0, a);
}
`

// wgslMSDFShader renders MSDF (Multi-channel Signed Distance Field) text.
// Uses the Chlumsky method for sharp text at any size (>= 24px).
const wgslMSDFShader = `
struct Uniforms {
    proj: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var msdf_texture: texture_2d<f32>;
@group(0) @binding(2) var msdf_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,
    @location(1) uv: vec2<f32>,
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.position = uniforms.proj * vec4<f32>(in.pos, 0.0, 1.0);
    out.uv = in.uv;
    return out;
}

fn median3(r: f32, g: f32, b: f32) -> f32 {
    return max(min(r, g), min(max(r, g), b));
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let s = textureSample(msdf_texture, msdf_sampler, in.uv).rgb;
    let d = median3(s.r, s.g, s.b);

    // Compute screen-pixel distance using UV derivatives (Chlumsky method).
    let tex_size = vec2<f32>(textureDimensions(msdf_texture, 0));
    let px_range = 4.0; // MSDF pixel range
    let unit_range = vec2<f32>(px_range) / tex_size;
    let screen_tex_size = vec2<f32>(1.0) / fwidth(in.uv);
    let screen_px_range = max(0.5 * dot(unit_range, screen_tex_size), 1.0);

    let screen_px_dist = screen_px_range * (d - 0.5);
    let alpha = clamp(screen_px_dist + 0.5, 0.0, 1.0);
    return vec4<f32>(1.0, 1.0, 1.0, alpha);
}
`
