//go:build !nogui && !windows

package gpu

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/timzifer/lux/ui"
)

// OpenGLRenderer implements Renderer using OpenGL 3.3 Core.
type OpenGLRenderer struct {
	width  int
	height int
}

// NewOpenGL creates an OpenGL-based renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
}

// Init initializes OpenGL.
func (r *OpenGLRenderer) Init(cfg Config) error {
	if err := gl.Init(); err != nil {
		return fmt.Errorf("opengl init: %w", err)
	}

	r.width = cfg.Width
	r.height = cfg.Height

	gl.Viewport(0, 0, int32(r.width), int32(r.height))
	gl.Enable(gl.SCISSOR_TEST)
	gl.ClearColor(colorFloats(ui.BackgroundColor))

	return nil
}

// Resize updates the viewport.
func (r *OpenGLRenderer) Resize(width, height int) {
	r.width = width
	r.height = height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// BeginFrame clears the screen to the M2 background color.
func (r *OpenGLRenderer) BeginFrame() {
	gl.Disable(gl.SCISSOR_TEST)
	gl.ClearColor(colorFloats(ui.BackgroundColor))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.Enable(gl.SCISSOR_TEST)
}

// Draw renders rectangles and bitmap text via gl.Scissor + gl.Clear.
func (r *OpenGLRenderer) Draw(scene ui.Scene) {
	for _, rect := range scene.Rects {
		r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
	}
	for _, text := range scene.Texts {
		r.drawText(text)
	}
}

// EndFrame is a no-op for OpenGL (swap is handled by GLFW).
func (r *OpenGLRenderer) EndFrame() {}

// Destroy releases OpenGL resources.
func (r *OpenGLRenderer) Destroy() {}

func (r *OpenGLRenderer) fillRect(x, y, w, h int, color ui.Color) {
	if w <= 0 || h <= 0 || r.width <= 0 || r.height <= 0 {
		return
	}
	gl.ClearColor(colorFloats(color))
	gl.Scissor(int32(x), int32(r.height-y-h), int32(w), int32(h))
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (r *OpenGLRenderer) drawText(cmd ui.DrawText) {
	if cmd.Scale <= 0 {
		cmd.Scale = 1
	}
	cursorX := cmd.X
	for _, raw := range cmd.Text {
		if raw == ' ' {
			cursorX += 6 * cmd.Scale
			continue
		}
		glyph, ok := glyph5x7[unicode.ToUpper(raw)]
		if !ok {
			glyph = glyph5x7['?']
		}
		for row, bits := range glyph {
			for col := 0; col < len(bits); col++ {
				if bits[col] != '1' {
					continue
				}
				r.fillRect(cursorX+(col*cmd.Scale), cmd.Y+(row*cmd.Scale), cmd.Scale, cmd.Scale, cmd.Color)
			}
		}
		cursorX += 6 * cmd.Scale
	}
}

func colorFloats(c ui.Color) (float32, float32, float32, float32) {
	return float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255
}

var glyph5x7 = map[rune][7]string{
	'?': {"11111", "00001", "00010", "00100", "00100", "00000", "00100"},
	'!': {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
	'-': {"00000", "00000", "11111", "00000", "00000", "00000", "00000"},
	'_': {"00000", "00000", "00000", "00000", "00000", "00000", "11111"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	':': {"00000", "00110", "00110", "00000", "00110", "00110", "00000"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01110", "10001", "10000", "10000", "10000", "10001", "01110"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01110", "10001", "10000", "10111", "10001", "10001", "01110"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"01110", "00100", "00100", "00100", "00100", "00100", "01110"},
	'J': {"00001", "00001", "00001", "00001", "10001", "10001", "01110"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "10001", "11001", "10101", "10011", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

func init() {
	for ch := 'a'; ch <= 'z'; ch++ {
		glyph5x7[ch] = glyph5x7[unicode.ToUpper(ch)]
	}
	for _, ch := range strings.Split("äöüßÄÖÜ", "") {
		if ch == "" {
			continue
		}
		r := []rune(ch)[0]
		glyph5x7[r] = glyph5x7['?']
	}
}
