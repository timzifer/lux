//go:build !nogui && !windows

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

// PyramidSurface renders a rotating color pyramid via OpenGL FBO.
// Implements ui.SurfaceProvider.
type PyramidSurface struct {
	// OpenGL resources (lazy-init).
	fbo, colorTex, depthRBO uint32
	program                 uint32
	vao, vbo                uint32
	mvpUniform              int32
	texW, texH              int
	inited                  bool

	// Rotation state.
	angleX, angleY, angleZ float32 // auto-rotation (radians)
	dragX, dragY           float32 // accumulated drag rotation
	dragging               bool
	lastMouseX, lastMouseY float32
	lastTime               time.Time

	nextToken ui.FrameToken
}

// NewPyramidSurface creates a new pyramid surface provider.
func NewPyramidSurface() *PyramidSurface {
	return &PyramidSurface{lastTime: time.Now()}
}

// Tick advances auto-rotation by dt.
func (p *PyramidSurface) Tick(dt time.Duration) {
	sec := float32(dt.Seconds())
	p.angleX += 0.3 * sec
	p.angleY += 0.5 * sec
	p.angleZ += 0.2 * sec
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

	gl.ClearColor(0.1, 0.1, 0.15, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.UseProgram(p.program)

	// Build MVP matrix.
	aspect := float32(w) / float32(h)
	proj := perspectiveMatrix(45.0*math.Pi/180.0, aspect, 0.1, 100.0)
	view := translationMatrix(0, 0, -4)
	model := matMul4(rotationX(p.angleX+p.dragX), rotationY(p.angleY+p.dragY))
	model = matMul4(model, rotationZ(p.angleZ))
	mvp := matMul4(proj, matMul4(view, model))

	gl.UniformMatrix4fv(p.mvpUniform, 1, false, &mvp[0])

	gl.BindVertexArray(p.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, 12) // 4 faces * 3 vertices
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

const pyramidVertShader = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aColor;

uniform mat4 uMVP;

out vec3 vColor;

void main() {
    gl_Position = uMVP * vec4(aPos, 1.0);
    vColor = aColor;
}
` + "\x00"

const pyramidFragShader = `#version 330 core
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
	prog, err := compilePyramidProgram(pyramidVertShader, pyramidFragShader)
	if err != nil {
		return
	}
	p.program = prog
	p.mvpUniform = gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))

	// Tetrahedron vertices: 4 faces, each with a distinct color.
	// Vertices of a regular tetrahedron centered at origin.
	top := [3]float32{0, 1.2, 0}
	frontLeft := [3]float32{-1, -0.6, 1}
	frontRight := [3]float32{1, -0.6, 1}
	back := [3]float32{0, -0.6, -1.2}

	red := [3]float32{1, 0.2, 0.2}
	green := [3]float32{0.2, 1, 0.2}
	blue := [3]float32{0.2, 0.4, 1}
	yellow := [3]float32{1, 0.95, 0.3}

	vertices := []float32{
		// Front face (red)
		top[0], top[1], top[2], red[0], red[1], red[2],
		frontLeft[0], frontLeft[1], frontLeft[2], red[0], red[1], red[2],
		frontRight[0], frontRight[1], frontRight[2], red[0], red[1], red[2],
		// Right face (green)
		top[0], top[1], top[2], green[0], green[1], green[2],
		frontRight[0], frontRight[1], frontRight[2], green[0], green[1], green[2],
		back[0], back[1], back[2], green[0], green[1], green[2],
		// Left face (blue)
		top[0], top[1], top[2], blue[0], blue[1], blue[2],
		back[0], back[1], back[2], blue[0], blue[1], blue[2],
		frontLeft[0], frontLeft[1], frontLeft[2], blue[0], blue[1], blue[2],
		// Bottom face (yellow)
		frontLeft[0], frontLeft[1], frontLeft[2], yellow[0], yellow[1], yellow[2],
		back[0], back[1], back[2], yellow[0], yellow[1], yellow[2],
		frontRight[0], frontRight[1], frontRight[2], yellow[0], yellow[1], yellow[2],
	}

	gl.GenVertexArrays(1, &p.vao)
	gl.GenBuffers(1, &p.vbo)

	gl.BindVertexArray(p.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, p.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

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

// ── Shader compilation (local to pyramid) ───────────────────────

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

// ── Matrix math ─────────────────────────────────────────────────

func perspectiveMatrix(fovY, aspect, near, far float32) [16]float32 {
	f := float32(1.0 / math.Tan(float64(fovY/2)))
	nf := near - far
	return [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / nf, -1,
		0, 0, (2 * far * near) / nf, 0,
	}
}

func translationMatrix(x, y, z float32) [16]float32 {
	return [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		x, y, z, 1,
	}
}

func rotationX(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{
		1, 0, 0, 0,
		0, c, s, 0,
		0, -s, c, 0,
		0, 0, 0, 1,
	}
}

func rotationY(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		0, 0, 0, 1,
	}
}

func rotationZ(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{
		c, s, 0, 0,
		-s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func matMul4(a, b [16]float32) [16]float32 {
	var r [16]float32
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				r[i*4+j] += a[i*4+k] * b[k*4+j]
			}
		}
	}
	return r
}

// Ensure unsafe is used (for gl.Ptr).
var _ = unsafe.Pointer(nil)
