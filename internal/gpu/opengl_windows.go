//go:build windows && !nogui

package gpu

import (
	"strings"
	"syscall"
	"unicode"
	"unsafe"

	"github.com/timzifer/lux/ui"
)

var (
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	user32gpu          = syscall.NewLazyDLL("user32.dll")
	procGetDC          = user32gpu.NewProc("GetDC")
	procReleaseDC      = user32gpu.NewProc("ReleaseDC")
	procSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
)

// bitmapInfoHeader mirrors BITMAPINFOHEADER for SetDIBitsToDevice.
type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// OpenGLRenderer implements the Windows M2 GDI software renderer.
type OpenGLRenderer struct {
	hwnd   uintptr
	width  int
	height int
	pixels []byte // BGRA pixel buffer, bottom-up row order for GDI.
}

// NewOpenGL creates the Windows software renderer.
func NewOpenGL() *OpenGLRenderer {
	return &OpenGLRenderer{}
}

// Init stores the framebuffer size and allocates the pixel buffer.
func (r *OpenGLRenderer) Init(cfg Config) error {
	r.hwnd = cfg.NativeHandle
	r.width = cfg.Width
	r.height = cfg.Height
	if r.width <= 0 {
		r.width = 800
	}
	if r.height <= 0 {
		r.height = 600
	}
	r.pixels = make([]byte, r.width*r.height*4)
	return nil
}

// Resize updates the framebuffer size and reallocates the pixel buffer.
func (r *OpenGLRenderer) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	r.width = width
	r.height = height
	r.pixels = make([]byte, r.width*r.height*4)
}

// BeginFrame clears the pixel buffer to the background color.
func (r *OpenGLRenderer) BeginFrame() {
	bg := ui.BackgroundColor
	for i := 0; i < len(r.pixels); i += 4 {
		r.pixels[i] = bg.B
		r.pixels[i+1] = bg.G
		r.pixels[i+2] = bg.R
		r.pixels[i+3] = bg.A
	}
}

// Draw renders rectangles and bitmap text to the pixel buffer.
func (r *OpenGLRenderer) Draw(scene ui.Scene) {
	for _, rect := range scene.Rects {
		r.fillRect(rect.X, rect.Y, rect.W, rect.H, rect.Color)
	}
	for _, text := range scene.Texts {
		r.drawText(text)
	}
}

// EndFrame blits the pixel buffer to the window via GDI.
func (r *OpenGLRenderer) EndFrame() {
	if r.hwnd == 0 {
		return
	}
	hdc, _, _ := procGetDC.Call(r.hwnd)
	if hdc == 0 {
		return
	}
	defer procReleaseDC.Call(r.hwnd, hdc)

	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(r.width),
		Height:   int32(r.height), // positive = bottom-up
		Planes:   1,
		BitCount: 32,
	}

	procSetDIBitsToDevice.Call(
		hdc,
		0, 0, // dest x, y
		uintptr(r.width), uintptr(r.height), // dest w, h
		0, 0, // src x, y
		0, uintptr(r.height), // start scan, num scans
		uintptr(unsafe.Pointer(&r.pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)
}

// Destroy releases renderer resources.
func (r *OpenGLRenderer) Destroy() {
	r.pixels = nil
}

func (r *OpenGLRenderer) fillRect(x, y, w, h int, color ui.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	for row := 0; row < h; row++ {
		py := y + row
		if py < 0 || py >= r.height {
			continue
		}
		// GDI DIB is bottom-up: row 0 is the bottom of the image.
		flippedY := r.height - 1 - py
		for col := 0; col < w; col++ {
			px := x + col
			if px < 0 || px >= r.width {
				continue
			}
			off := (flippedY*r.width + px) * 4
			r.pixels[off] = color.B
			r.pixels[off+1] = color.G
			r.pixels[off+2] = color.R
			r.pixels[off+3] = color.A
		}
	}
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
		glyph, ok := winGlyph5x7[unicode.ToUpper(raw)]
		if !ok {
			glyph = winGlyph5x7['?']
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

// winGlyph5x7 is the same 5×7 bitmap font used by the OpenGL renderer.
var winGlyph5x7 = map[rune][7]string{
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
		winGlyph5x7[ch] = winGlyph5x7[unicode.ToUpper(ch)]
	}
	for _, ch := range strings.Split("äöüßÄÖÜ", "") {
		if ch == "" {
			continue
		}
		r := []rune(ch)[0]
		winGlyph5x7[r] = winGlyph5x7['?']
	}
}
