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
