//go:build !nogui && !windows && !(darwin && arm64)

package main

import (
	"math"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/ui"
)

// PyramidSurface renders a rotating RGB cube via OpenGL FBO.
// The cube's vertex colors correspond to their (R,G,B) corner positions,
// inspired by https://github.com/c2d7fa/opengl-cube.
// Implements ui.SurfaceProvider.
type PyramidSurface struct {
	// OpenGL resources (lazy-init).
	fbo, colorTex, depthRBO uint32
	program                 uint32
	vao, vbo, ebo           uint32
	mvpUniform              int32
	texW, texH              int
	inited                  bool

	// Rotation state.
	angleX, angleY float32 // auto-rotation (radians)
	dragX, dragY   float32 // accumulated drag rotation
	dragging       bool
	lastMouseX     float32
	lastMouseY     float32
	lastTime       time.Time

	nextToken ui.FrameToken
}

// NewPyramidSurface creates a new RGB cube surface provider.
func NewPyramidSurface() *PyramidSurface {
	return &PyramidSurface{lastTime: time.Now()}
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
	if w <= 0 || h <= 0 {
		return 0, 0
	}

	p.initGL(w, h)
	if !p.inited {
		return 0, 0
	}

	// Resize FBO if needed.
	if w != p.texW || h != p.texH {
		p.resizeFBO(w, h)
	}

	// Save GL state.
	var prevFBO, prevViewport [4]int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &prevFBO[0])
	gl.GetIntegerv(gl.VIEWPORT, &prevViewport[0])

	// Render to FBO.
	gl.BindFramebuffer(gl.FRAMEBUFFER, p.fbo)
	gl.Viewport(0, 0, int32(w), int32(h))
	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.ClearColor(0.08, 0.08, 0.12, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.UseProgram(p.program)

	// Build MVP matrix.
	aspect := float32(w) / float32(h)
	proj := perspectiveMatrix(45.0*math.Pi/180.0, aspect, 0.1, 100.0)
	view := translationMatrix(0, 0, -4)
	model := matMul4(rotationX(p.angleX+p.dragX), rotationY(p.angleY+p.dragY))
	mvp := matMul4(proj, matMul4(view, model))

	gl.UniformMatrix4fv(p.mvpUniform, 1, false, &mvp[0])

	gl.BindVertexArray(p.vao)
	gl.DrawElements(gl.TRIANGLES, 36, gl.UNSIGNED_INT, nil) // 6 faces * 2 tris * 3 verts
	gl.BindVertexArray(0)

	gl.UseProgram(0)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)

	// Restore GL state.
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(prevFBO[0]))
	gl.Viewport(prevViewport[0], prevViewport[1], prevViewport[2], prevViewport[3])

	p.nextToken++
	if p.nextToken == 0 {
		p.nextToken++
	}
	return draw.TextureID(p.colorTex), p.nextToken
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

// ── OpenGL initialization ───────────────────────────────────────

const cubeVertShader = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aColor;

uniform mat4 uMVP;

out vec3 vColor;

void main() {
    gl_Position = uMVP * vec4(aPos, 1.0);
    vColor = aColor;
}
` + "\x00"

const cubeFragShader = `#version 330 core
in vec3 vColor;
out vec4 fragColor;

void main() {
    fragColor = vec4(vColor, 1.0);
}
` + "\x00"

func (p *PyramidSurface) initGL(w, h int) {
	if p.inited {
		return
	}

	// Compile shaders.
	prog, err := compilePyramidProgram(cubeVertShader, cubeFragShader)
	if err != nil {
		return
	}
	p.program = prog
	p.mvpUniform = gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))

	// RGB Cube: 8 vertices at unit cube corners.
	// Each vertex color = its (x,y,z) position mapped to (R,G,B).
	//
	//     (0,1,0)────(1,1,0)
	//       /│          /│
	//      / │         / │
	// (0,1,1)────(1,1,1) │
	//     │(0,0,0)───│(1,0,0)
	//     │ /         │ /
	//     │/          │/
	// (0,0,1)────(1,0,1)
	//
	// Centered at origin: positions shifted by -0.5, colors stay 0..1.
	type vert struct {
		x, y, z float32 // position (centered)
		r, g, b float32 // color = original corner
	}

	verts := []vert{
		{-0.5, -0.5, -0.5, 0, 0, 0}, // 0: (0,0,0) — black
		{+0.5, -0.5, -0.5, 1, 0, 0}, // 1: (1,0,0) — red
		{+0.5, +0.5, -0.5, 1, 1, 0}, // 2: (1,1,0) — yellow
		{-0.5, +0.5, -0.5, 0, 1, 0}, // 3: (0,1,0) — green
		{-0.5, -0.5, +0.5, 0, 0, 1}, // 4: (0,0,1) — blue
		{+0.5, -0.5, +0.5, 1, 0, 1}, // 5: (1,0,1) — magenta
		{+0.5, +0.5, +0.5, 1, 1, 1}, // 6: (1,1,1) — white
		{-0.5, +0.5, +0.5, 0, 1, 1}, // 7: (0,1,1) — cyan
	}

	// Flatten to float32 slice.
	vertices := make([]float32, 0, len(verts)*6)
	for _, v := range verts {
		vertices = append(vertices, v.x, v.y, v.z, v.r, v.g, v.b)
	}

	// 12 triangles (6 faces × 2 tris), wound CCW when viewed from outside.
	indices := []uint32{
		// Front face (z = +0.5)
		4, 5, 6, 6, 7, 4,
		// Back face (z = -0.5)
		1, 0, 3, 3, 2, 1,
		// Right face (x = +0.5)
		5, 1, 2, 2, 6, 5,
		// Left face (x = -0.5)
		0, 4, 7, 7, 3, 0,
		// Top face (y = +0.5)
		7, 6, 2, 2, 3, 7,
		// Bottom face (y = -0.5)
		0, 1, 5, 5, 4, 0,
	}

	gl.GenVertexArrays(1, &p.vao)
	gl.GenBuffers(1, &p.vbo)
	gl.GenBuffers(1, &p.ebo)

	gl.BindVertexArray(p.vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, p.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, p.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	stride := int32(6 * 4) // 3 pos + 3 color floats
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, uintptr(3*4))

	gl.BindVertexArray(0)

	// Create FBO.
	p.createFBO(w, h)
	p.inited = true
}

func (p *PyramidSurface) createFBO(w, h int) {
	gl.GenFramebuffers(1, &p.fbo)
	gl.GenTextures(1, &p.colorTex)
	gl.GenRenderbuffers(1, &p.depthRBO)

	gl.BindTexture(gl.TEXTURE_2D, p.colorTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(w), int32(h), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	gl.BindRenderbuffer(gl.RENDERBUFFER, p.depthRBO)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, int32(w), int32(h))
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)

	gl.BindFramebuffer(gl.FRAMEBUFFER, p.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, p.colorTex, 0)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, p.depthRBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	p.texW = w
	p.texH = h
}

func (p *PyramidSurface) resizeFBO(w, h int) {
	gl.BindTexture(gl.TEXTURE_2D, p.colorTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(w), int32(h), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	gl.BindRenderbuffer(gl.RENDERBUFFER, p.depthRBO)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, int32(w), int32(h))
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)

	p.texW = w
	p.texH = h
}

// ── Shader compilation (local to cube) ──────────────────────────

func compilePyramidProgram(vertSrc, fragSrc string) (uint32, error) {
	vert := gl.CreateShader(gl.VERTEX_SHADER)
	csrc, free := gl.Strs(vertSrc)
	gl.ShaderSource(vert, 1, csrc, nil)
	free()
	gl.CompileShader(vert)
	var status int32
	gl.GetShaderiv(vert, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		gl.DeleteShader(vert)
		return 0, nil
	}

	frag := gl.CreateShader(gl.FRAGMENT_SHADER)
	csrc2, free2 := gl.Strs(fragSrc)
	gl.ShaderSource(frag, 1, csrc2, nil)
	free2()
	gl.CompileShader(frag)
	gl.GetShaderiv(frag, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		gl.DeleteShader(vert)
		gl.DeleteShader(frag)
		return 0, nil
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)
	gl.DeleteShader(vert)
	gl.DeleteShader(frag)

	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		gl.DeleteProgram(prog)
		return 0, nil
	}
	return prog, nil
}

// Ensure unsafe is used (for gl.Ptr).
var _ = unsafe.Pointer(nil)
