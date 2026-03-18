//go:build !nogui && !windows

package gpu

// GLSL 330 core shaders for textured glyph rendering.

const textVertexShader = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;

uniform mat4 uProj;

out vec2 vUV;

void main() {
    gl_Position = uProj * vec4(aPos, 0.0, 1.0);
    vUV = aUV;
}
` + "\x00"

const textFragmentShader = `#version 330 core
in vec2 vUV;

uniform sampler2D uAtlas;
uniform vec4 uColor;

out vec4 fragColor;

void main() {
    float a = texture(uAtlas, vUV).r;
    fragColor = vec4(uColor.rgb, uColor.a * a);
}
` + "\x00"

// GLSL 330 core shaders for rounded rectangle rendering.

const rectVertexShader = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec4 aRect;
layout(location = 2) in vec4 aColor;
layout(location = 3) in float aRadius;

uniform mat4 uProj;

out vec2 vLocalPos;
out vec2 vHalfSize;
out vec4 vColor;
flat out float vRadius;

void main() {
    // aRect = (x, y, w, h) in screen coords.
    // aPos = (0,0), (1,0), (0,1), (1,1) — unit quad corner.
    // Expand the quad by 0.5px on each side so the SDF anti-aliasing
    // transition has room on both sides of the mathematical boundary.
    vec2 expand = vec2(0.5);
    vec2 expandedSize = aRect.zw + expand * 2.0;
    vec2 pos = aRect.xy - expand + aPos * expandedSize;
    gl_Position = uProj * vec4(pos, 0.0, 1.0);

    // vHalfSize = original rect half-extents (for SDF computation).
    vHalfSize = aRect.zw * 0.5;
    // vLocalPos maps the expanded quad so center = (0,0).
    vLocalPos = (aPos - 0.5) * expandedSize;
    vColor = aColor;
    vRadius = aRadius;
}
` + "\x00"

const rectFragmentShader = `#version 330 core
in vec2 vLocalPos;
in vec2 vHalfSize;
in vec4 vColor;
flat in float vRadius;

out vec4 fragColor;

float roundedBoxSDF(vec2 p, vec2 b, float r) {
    vec2 q = abs(p) - b + r;
    return length(max(q, 0.0)) - r;
}

void main() {
    float dist = roundedBoxSDF(vLocalPos, vHalfSize, vRadius);
    // Anti-alias: smoothstep over ~1px.
    float alpha = 1.0 - smoothstep(-0.5, 0.5, dist);
    if (alpha < 0.001) discard;
    fragColor = vec4(vColor.rgb, vColor.a * alpha);
}
` + "\x00"
