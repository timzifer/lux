//go:build !nogui

package gpu

// WGSL shaders for the wgpu renderer, equivalent to the OpenGL 3.3 GLSL shaders.

// wgslRectShader renders rounded rectangles using SDF (Signed Distance Field).
// Instanced rendering: unit quad vertices + per-instance rect/color/radius data.
const wgslRectShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,  // x = grain intensity (RFC-008 §10.5), yzw reserved
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
    return length(max(q, vec2<f32>(0.0))) + min(max(q.x, q.y), 0.0) - r;
}

fn noise_hash(p: vec2<f32>) -> f32 {
    var n = (u32(p.x) * 1597334673u) ^ (u32(p.y) * 3812015801u);
    n = (n ^ (n >> 16u)) * 2246822519u;
    n = (n ^ (n >> 13u)) * 3266489917u;
    n = n ^ (n >> 16u);
    return f32(n) / 4294967295.0;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let dist = rounded_box_sdf(in.local_pos, in.half_size, in.radius);
    let alpha = 1.0 - smoothstep(-0.5, 0.5, dist);
    if (alpha < 0.001) {
        discard;
    }
    let grain = uniforms.params.x;
    let n = (noise_hash(in.position.xy) - 0.5) * grain;
    return vec4<f32>(in.color.rgb + n, in.color.a * alpha);
}
`

// wgslTextInstancedShader renders atlas-based textured glyphs (bitmap text < 24px)
// using instanced rendering: unit quad + per-instance glyph rect/uv/color.
const wgslTextInstancedShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var atlas_texture: texture_2d<f32>;
@group(0) @binding(2) var atlas_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,         // unit quad corner (0..1)
    @location(1) glyph_rect: vec4<f32>,  // dstX, dstY, dstW, dstH
    @location(2) glyph_uv: vec4<f32>,    // u0, v0, u1, v1
    @location(3) color: vec4<f32>,       // r, g, b, a
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) color: vec4<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let world_pos = in.glyph_rect.xy + in.pos * in.glyph_rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = mix(in.glyph_uv.xy, in.glyph_uv.zw, in.pos);
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let a = textureSample(atlas_texture, atlas_sampler, in.uv).r;
    return vec4<f32>(in.color.rgb, in.color.a * a);
}
`

// wgslSurfaceShader renders an external surface texture (blit) onto a screen-space quad.
// Bind group 0: projection uniform.
// Bind group 1: texture + sampler (per-surface).
const wgslSurfaceShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(1) @binding(0) var surf_texture: texture_2d<f32>;
@group(1) @binding(1) var surf_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,      // (x, y, w, h) in screen coords
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let world_pos = in.rect.xy + in.pos * in.rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = vec2<f32>(in.pos.x, in.pos.y);
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    return textureSample(surf_texture, surf_sampler, in.uv);
}
`

// wgslGradientShader renders gradient-filled rounded rectangles.
// Uses a uniform buffer per gradient rect (projection + gradient params).
const wgslGradientShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,  // x = grain intensity (RFC-008 §10.5), yzw reserved
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct GradientParams {
    rect: vec4<f32>,          // x, y, w, h
    radius: f32,
    gradient_type: f32,       // 0 = linear, 1 = radial
    stop_count: f32,
    _pad: f32,
    grad_start: vec4<f32>,    // linear: startX,startY,endX,endY / radial: centerX,centerY,radius,0
    stops: array<vec4<f32>, 16>,  // pairs: [offset,r,g,b], [a, 0, 0, 0] — 8 stops × 2 vec4
};
@group(1) @binding(0) var<uniform> grad: GradientParams;

struct VertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) local_pos: vec2<f32>,
    @location(1) half_size: vec2<f32>,
    @location(2) frag_pos: vec2<f32>,  // screen-space position
    @location(3) @interpolate(flat) radius: f32,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let expand = vec2<f32>(0.5);
    let expanded_size = grad.rect.zw + expand * 2.0;
    let world_pos = grad.rect.xy - expand + in.pos * expanded_size;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.half_size = grad.rect.zw * 0.5;
    out.local_pos = (in.pos - 0.5) * expanded_size;
    out.frag_pos = world_pos;
    out.radius = grad.radius;
    return out;
}

fn rounded_box_sdf_g(p: vec2<f32>, b: vec2<f32>, r: f32) -> f32 {
    let q = abs(p) - b + r;
    return length(max(q, vec2<f32>(0.0))) - r;
}

fn noise_hash_g(p: vec2<f32>) -> f32 {
    var n = (u32(p.x) * 1597334673u) ^ (u32(p.y) * 3812015801u);
    n = (n ^ (n >> 16u)) * 2246822519u;
    n = (n ^ (n >> 13u)) * 3266489917u;
    n = n ^ (n >> 16u);
    return f32(n) / 4294967295.0;
}

fn sample_gradient(t_raw: f32) -> vec4<f32> {
    let t = clamp(t_raw, 0.0, 1.0);
    let count = i32(grad.stop_count);
    if (count <= 0) {
        return vec4<f32>(0.0);
    }
    // First stop
    let c0 = vec4<f32>(grad.stops[0].y, grad.stops[0].z, grad.stops[0].w, grad.stops[1].x);
    if (count == 1 || t <= grad.stops[0].x) {
        return c0;
    }
    // Interpolate between stops
    var prev_offset = grad.stops[0].x;
    var prev_color = c0;
    for (var i = 1; i < 8; i++) {
        if (i >= count) { break; }
        let idx = i * 2;
        let offset = grad.stops[idx].x;
        let color = vec4<f32>(grad.stops[idx].y, grad.stops[idx].z, grad.stops[idx].w, grad.stops[idx + 1].x);
        if (t <= offset) {
            let f = (t - prev_offset) / max(offset - prev_offset, 0.0001);
            return mix(prev_color, color, f);
        }
        prev_offset = offset;
        prev_color = color;
    }
    return prev_color;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    // SDF for rounded corners
    let dist = rounded_box_sdf_g(in.local_pos, in.half_size, in.radius);
    let alpha = 1.0 - smoothstep(-0.5, 0.5, dist);
    if (alpha < 0.001) {
        discard;
    }

    // Compute gradient t
    var t: f32 = 0.0;
    if (grad.gradient_type < 0.5) {
        // Linear gradient
        let start = grad.grad_start.xy;
        let end = grad.grad_start.zw;
        let dir = end - start;
        let len2 = dot(dir, dir);
        if (len2 > 0.0001) {
            t = dot(in.frag_pos - start, dir) / len2;
        }
    } else {
        // Radial gradient
        let center = grad.grad_start.xy;
        let radius = grad.grad_start.z;
        if (radius > 0.0001) {
            t = distance(in.frag_pos, center) / radius;
        }
    }

    let color = sample_gradient(t);
    let grain = uniforms.params.x;
    let n = (noise_hash_g(in.position.xy) - 0.5) * grain;
    return vec4<f32>(color.rgb + n, color.a * alpha);
}
`

// wgslMSDFInstancedShader renders MSDF (Multi-channel Signed Distance Field) text
// using instanced rendering: unit quad + per-instance glyph rect/uv/color.
const wgslMSDFInstancedShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    atlas_size: vec4<f32>,  // xy = texture size, zw = px_range (replicated)
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var msdf_texture: texture_2d<f32>;
@group(0) @binding(2) var msdf_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,         // unit quad corner (0..1)
    @location(1) glyph_rect: vec4<f32>,  // dstX, dstY, dstW, dstH
    @location(2) glyph_uv: vec4<f32>,    // u0, v0, u1, v1
    @location(3) color: vec4<f32>,       // r, g, b, a
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) color: vec4<f32>,
    @location(2) screen_tex_size: vec2<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let world_pos = in.glyph_rect.xy + in.pos * in.glyph_rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = mix(in.glyph_uv.xy, in.glyph_uv.zw, in.pos);
    out.color = in.color;

    // Derivatives from fwidth(in.uv) are discontinuous across the quad's
    // internal triangle seam, which can show up as isolated black/white pixels
    // inside counters such as “a” and “e”. Because Lux renders glyph quads with
    // only translation+uniform scaling, we can derive the exact screen-pixel
    // coverage per atlas texel from the instance rect and UV span instead.
    let uv_span = max(in.glyph_uv.zw - in.glyph_uv.xy, vec2<f32>(1e-6, 1e-6));
    out.screen_tex_size = in.glyph_rect.zw / uv_span;
    return out;
}

fn median3(r: f32, g: f32, b: f32) -> f32 {
    return max(min(r, g), min(max(r, g), b));
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let s = textureSample(msdf_texture, msdf_sampler, in.uv).rgb;
    let d = median3(s.r, s.g, s.b);

    // Compute screen-pixel distance without derivatives. Using fwidth(in.uv)
    // here causes seam artifacts because each glyph quad is split into two
    // triangles and derivatives are evaluated per triangle.
    // Atlas size is passed via uniforms.atlas_size since textureDimensions()
    // is broken in naga's HLSL backend.
    let unit_range = uniforms.atlas_size.zw / uniforms.atlas_size.xy;
    let screen_px_range = max(0.5 * dot(unit_range, in.screen_tex_size), 1.0);

    let screen_px_dist = screen_px_range * (d - 0.5);
    let alpha = clamp(screen_px_dist + 0.5, 0.0, 1.0);
    if (alpha < 0.01) {
        discard;
    }
    return vec4<f32>(in.color.rgb, in.color.a * alpha);
}
`

// wgslEmojiInstancedShader renders color emoji (NRGBA bitmaps from CBDT font tables)
// using instanced rendering: unit quad + per-instance glyph rect/uv/color.
// The vertex shader is identical to the text shader. The fragment shader samples
// RGBA color directly from the emoji atlas (not as an alpha mask).
// in.color.a is used as an opacity modulator.
const wgslEmojiInstancedShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(0) @binding(1) var emoji_texture: texture_2d<f32>;
@group(0) @binding(2) var emoji_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,         // unit quad corner (0..1)
    @location(1) glyph_rect: vec4<f32>,  // dstX, dstY, dstW, dstH
    @location(2) glyph_uv: vec4<f32>,    // u0, v0, u1, v1
    @location(3) color: vec4<f32>,       // r, g, b, a
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) color: vec4<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let world_pos = in.glyph_rect.xy + in.pos * in.glyph_rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = mix(in.glyph_uv.xy, in.glyph_uv.zw, in.pos);
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let color = textureSample(emoji_texture, emoji_sampler, in.uv);
    return vec4<f32>(color.rgb * in.color.a, color.a * in.color.a);
}
`

// wgslShadowShader renders soft box shadows using SDF with blur falloff.
// Instanced rendering: unit quad vertices + per-instance rect/color/radius/blurRadius/inset.
// 12 floats per instance (48 bytes).
const wgslShadowShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct VertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,      // (x, y, w, h) in screen coords
    @location(2) color: vec4<f32>,     // RGBA color
    @location(3) radius: f32,          // corner radius
    @location(4) blur_radius: f32,     // shadow blur spread
    @location(5) inset: f32,           // 0.0 = outer, 1.0 = inner shadow
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) local_pos: vec2<f32>,
    @location(1) half_size: vec2<f32>,
    @location(2) color: vec4<f32>,
    @location(3) @interpolate(flat) radius: f32,
    @location(4) @interpolate(flat) blur_radius: f32,
    @location(5) @interpolate(flat) inset: f32,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;

    // Inset shadows render inside the rect — minimal expand for AA only.
    // Outer shadows expand by blur_radius + 0.5px so the soft falloff is fully visible.
    var expand: vec2<f32>;
    if (in.inset > 0.5) {
        expand = vec2<f32>(0.5);
    } else {
        expand = vec2<f32>(in.blur_radius + 0.5);
    }
    let expanded_size = in.rect.zw + expand * 2.0;
    let world_pos = in.rect.xy - expand + in.pos * expanded_size;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);

    // For outer shadows the CPU-side rect was already expanded by blur_radius
    // (for clipping). Shrink the SDF box back so the fade starts at the
    // original element edge, not blur_radius away from it.
    out.half_size = in.rect.zw * 0.5;
    if (in.inset < 0.5) {
        out.half_size -= vec2<f32>(in.blur_radius);
    }
    out.local_pos = (in.pos - 0.5) * expanded_size;
    out.color = in.color;
    out.radius = in.radius;
    out.blur_radius = in.blur_radius;
    out.inset = in.inset;
    return out;
}

fn rounded_box_sdf_s(p: vec2<f32>, b: vec2<f32>, r: f32) -> f32 {
    let q = abs(p) - b + r;
    return length(max(q, vec2<f32>(0.0))) - r;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let dist = rounded_box_sdf_s(in.local_pos, in.half_size, in.radius);
    var alpha: f32;
    if (in.inset > 0.5) {
        // Inner shadow: compute a second SDF with blur_radius as effective
        // corner radius so it reaches blur_radius deep into the shape.
        // This gives smooth concentric rounded-rect iso-lines (no corner
        // artifacts from min-of-edges) and the full blur_radius fade depth.
        let fade_r = min(in.blur_radius, min(in.half_size.x, in.half_size.y));
        let fade_dist = rounded_box_sdf_s(in.local_pos, in.half_size, fade_r);
        let fade = smoothstep(-fade_r, 0.0, fade_dist);
        // Mask to actual rounded shape (original corner radius) with AA.
        let mask = 1.0 - smoothstep(0.0, 0.5, dist);
        alpha = fade * mask;
    } else {
        // Outer shadow: Gaussian falloff for realistic soft shadows.
        // sigma = blur_radius/2 maps blur_radius to ~2 standard deviations,
        // giving a gentle tail that fades to ~13% at blur_radius distance.
        let sigma = max(in.blur_radius * 0.5, 0.001);
        let d = max(dist, 0.0);
        alpha = exp(-0.5 * d * d / (sigma * sigma));
    }
    if (alpha < 0.001) {
        discard;
    }
    return vec4<f32>(in.color.rgb, in.color.a * alpha);
}
`

// wgslBlurShader implements a separable Gaussian blur as a fullscreen-quad fragment shader.
// Two render passes: horizontal (direction=(1,0)) then vertical (direction=(0,1)).
// Bind group 0: blur uniforms (direction, radius, texture_size).
// Bind group 1: input texture + sampler.
// Output: render attachment (the other ping-pong texture).
const wgslBlurShader = `
struct BlurUniforms {
    direction: vec2<f32>,
    radius: u32,
    _pad: u32,
    texture_size: vec2<f32>,
    _pad2: vec2<f32>,
};
@group(0) @binding(0) var<uniform> params: BlurUniforms;
@group(1) @binding(0) var input_tex: texture_2d<f32>;
@group(1) @binding(1) var input_sampler: sampler;

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};

// Fullscreen triangle — 3 vertices cover the entire screen, no vertex buffer needed.
@vertex
fn vs_main(@builtin(vertex_index) vi: u32) -> VertexOutput {
    var out: VertexOutput;
    // Generates a triangle that covers clip space [-1,1]:
    //   vi=0 → (-1,-1), vi=1 → (3,-1), vi=2 → (-1,3)
    let x = f32(i32(vi & 1u)) * 4.0 - 1.0;
    let y = f32(i32(vi >> 1u)) * 4.0 - 1.0;
    out.position = vec4<f32>(x, y, 0.0, 1.0);
    // Map to UV [0,1] with Y flipped for texture coords.
    out.uv = vec2<f32>((x + 1.0) * 0.5, (1.0 - y) * 0.5);
    return out;
}

fn gaussian(x: f32, sigma: f32) -> f32 {
    return exp(-(x * x) / (2.0 * sigma * sigma));
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let r = i32(min(params.radius, 64u));
    let sigma = max(f32(r) / 3.0, 1.0);
    let texel = params.direction / params.texture_size;

    var color = vec4<f32>(0.0);
    var weight_sum = 0.0;

    for (var i = -r; i <= r; i++) {
        let offset = texel * f32(i);
        let uv = clamp(in.uv + offset, vec2<f32>(0.0), vec2<f32>(1.0));
        let w = gaussian(f32(i), sigma);
        color += textureSample(input_tex, input_sampler, uv) * w;
        weight_sum += w;
    }

    return color / weight_sum;
}
` + ""

// wgslBlurBlitShader blits a texture to the screen as a full-screen quad (used after blur).
const wgslBlurBlitShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(1) @binding(0) var blit_texture: texture_2d<f32>;
@group(1) @binding(1) var blit_sampler: sampler;

struct VertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,      // (x, y, w, h) in screen coords
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    let world_pos = in.rect.xy + in.pos * in.rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = vec2<f32>(in.pos.x, in.pos.y);
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    return textureSample(blit_texture, blit_sampler, in.uv);
}
`

// wgslImageShader renders textured image quads with opacity and UV subregion support.
// Instance data: rect(4F) + uv_rect(4F) + opacity(1F) = 36 bytes per instance.
const wgslImageShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(1) @binding(0) var img_texture: texture_2d<f32>;
@group(1) @binding(1) var img_sampler: sampler;

struct ImageVertexInput {
    @location(0) pos: vec2<f32>,           // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,          // (x, y, w, h) in screen coords
    @location(2) uv_rect: vec4<f32>,       // (u0, v0, u1, v1) subregion
    @location(3) opacity: f32,
};

struct ImageVertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) opacity: f32,
};

@vertex
fn vs_main(in: ImageVertexInput) -> ImageVertexOutput {
    var out: ImageVertexOutput;
    let world_pos = in.rect.xy + in.pos * in.rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    // Map unit quad pos to UV subregion.
    out.uv = mix(in.uv_rect.xy, in.uv_rect.zw, in.pos);
    out.opacity = in.opacity;
    return out;
}

@fragment
fn fs_main(in: ImageVertexOutput) -> @location(0) vec4<f32> {
    let color = textureSample(img_texture, img_sampler, in.uv);
    return vec4<f32>(color.rgb * in.opacity, color.a * in.opacity);
}
`

// wgslCustomShaderPrefix is the fixed vertex part for custom shader backgrounds.
// User-supplied fragment code is appended after this prefix.
const wgslCustomShaderPrefix = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct CustomParams {
    rect: vec4<f32>,
    time: f32,
    user0: f32, user1: f32, user2: f32,
    user3: f32, user4: f32, user5: f32, user6: f32,
};
@group(1) @binding(0) var<uniform> custom: CustomParams;

struct CustomVertexInput {
    @location(0) pos: vec2<f32>,       // unit quad corner (0..1)
    @location(1) rect: vec4<f32>,      // (x, y, w, h) in screen coords
};

struct CustomVertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) world_pos: vec2<f32>,
};

@vertex
fn vs_main(in: CustomVertexInput) -> CustomVertexOutput {
    var out: CustomVertexOutput;
    let world_pos = in.rect.xy + in.pos * in.rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = in.pos;
    out.world_pos = world_pos;
    return out;
}
`

// wgslCustomShaderImagePrefix extends the custom shader prefix with a texture binding.
// Used for PaintShaderImage where the shader processes an image.
const wgslCustomShaderImagePrefix = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct CustomParams {
    rect: vec4<f32>,
    time: f32,
    user0: f32, user1: f32, user2: f32,
    user3: f32, user4: f32, user5: f32, user6: f32,
};
@group(1) @binding(0) var<uniform> custom: CustomParams;
@group(2) @binding(0) var src_texture: texture_2d<f32>;
@group(2) @binding(1) var src_sampler: sampler;

struct CustomImgVertexInput {
    @location(0) pos: vec2<f32>,
    @location(1) rect: vec4<f32>,
};

struct CustomImgVertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
    @location(1) world_pos: vec2<f32>,
};

@vertex
fn vs_main(in: CustomImgVertexInput) -> CustomImgVertexOutput {
    var out: CustomImgVertexOutput;
    let world_pos = in.rect.xy + in.pos * in.rect.zw;
    out.position = uniforms.proj * vec4<f32>(world_pos, 0.0, 1.0);
    out.uv = in.pos;
    out.world_pos = world_pos;
    return out;
}
`

// wgslNoiseShader is a built-in simplex noise background effect.
const wgslNoiseShader = wgslCustomShaderPrefix + `
// Simplex-like noise hash.
fn hash22(p: vec2<f32>) -> vec2<f32> {
    var p3 = fract(vec3<f32>(p.xyx) * vec3<f32>(0.1031, 0.1030, 0.0973));
    p3 = p3 + dot(p3, p3.yzx + 33.33);
    return fract((p3.xx + p3.yz) * p3.zy);
}

fn noise(p: vec2<f32>) -> f32 {
    let i = floor(p);
    let f = fract(p);
    let u = f * f * (3.0 - 2.0 * f);
    let a = dot(hash22(i + vec2<f32>(0.0, 0.0)) - 0.5, f - vec2<f32>(0.0, 0.0));
    let b = dot(hash22(i + vec2<f32>(1.0, 0.0)) - 0.5, f - vec2<f32>(1.0, 0.0));
    let c = dot(hash22(i + vec2<f32>(0.0, 1.0)) - 0.5, f - vec2<f32>(0.0, 1.0));
    let d = dot(hash22(i + vec2<f32>(1.0, 1.0)) - 0.5, f - vec2<f32>(1.0, 1.0));
    return mix(mix(a, b, u.x), mix(c, d, u.x), u.y) + 0.5;
}

@fragment
fn fs_main(in: CustomVertexOutput) -> @location(0) vec4<f32> {
    let scale = custom.user0 + 4.0;  // user0 controls noise scale
    let n = noise(in.uv * scale + vec2<f32>(custom.time * 0.1, 0.0));
    let color = vec3<f32>(n * custom.user1, n * custom.user2, n * custom.user3);
    return vec4<f32>(color, 1.0);
}
`

// wgslPlasmaShader is a built-in animated plasma background effect.
const wgslPlasmaShader = wgslCustomShaderPrefix + `
@fragment
fn fs_main(in: CustomVertexOutput) -> @location(0) vec4<f32> {
    let t = custom.time;
    let uv = in.uv * (custom.user0 + 4.0);
    let v1 = sin(uv.x + t);
    let v2 = sin(uv.y + t);
    let v3 = sin(uv.x + uv.y + t);
    let v4 = sin(sqrt(uv.x * uv.x + uv.y * uv.y) + t);
    let v = (v1 + v2 + v3 + v4) * 0.25;
    let r = sin(v * 3.14159) * 0.5 + 0.5;
    let g = sin(v * 3.14159 + 2.094) * 0.5 + 0.5;
    let b = sin(v * 3.14159 + 4.189) * 0.5 + 0.5;
    return vec4<f32>(r, g, b, 1.0);
}
`

// wgslVoronoiShader is a built-in Voronoi cell pattern background effect.
const wgslVoronoiShader = wgslCustomShaderPrefix + `
fn hash2(p: vec2<f32>) -> vec2<f32> {
    return fract(sin(vec2<f32>(
        dot(p, vec2<f32>(127.1, 311.7)),
        dot(p, vec2<f32>(269.5, 183.3))
    )) * 43758.5453);
}

@fragment
fn fs_main(in: CustomVertexOutput) -> @location(0) vec4<f32> {
    let scale = custom.user0 + 4.0;
    let uv = in.uv * scale;
    let ip = floor(uv);
    let fp = fract(uv);
    var minDist: f32 = 1.0;
    var minCell: vec2<f32> = vec2<f32>(0.0);
    for (var j: i32 = -1; j <= 1; j = j + 1) {
        for (var i: i32 = -1; i <= 1; i = i + 1) {
            let neighbor = vec2<f32>(f32(i), f32(j));
            let point = hash2(ip + neighbor);
            let animated = neighbor + sin(point * 6.28 + custom.time) * 0.5 + 0.5;
            let diff = animated - fp;
            let dist = dot(diff, diff);
            if (dist < minDist) {
                minDist = dist;
                minCell = point;
            }
        }
    }
    let c = minCell.x * 0.5 + 0.25;
    return vec4<f32>(c, c * 0.8, c * 1.2, 1.0);
}
`

// wgslPathShader renders tessellated path triangles.
// Each vertex carries position (x, y) and per-vertex color (r, g, b, a).
const wgslPathShader = `
struct Uniforms {
    proj: mat4x4<f32>,
    params: vec4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct VertexInput {
    @location(0) pos: vec2<f32>,
    @location(1) color: vec4<f32>,
};

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) color: vec4<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.position = uniforms.proj * vec4<f32>(in.pos, 0.0, 1.0);
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    return in.color;
}
`
