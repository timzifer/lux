//go:build js && wasm

// Package web implements platform.Platform for browser environments via WebAssembly.
// It uses a <canvas id="lux-canvas"> element for rendering and DOM event listeners
// for input, with requestAnimationFrame driving the frame loop.
package web

import (
	"fmt"
	"log"
	"strconv"
	"syscall/js"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/wgpu"
	"github.com/timzifer/lux/platform"
)

// Platform implements platform.Platform for browser/WASM environments.
type Platform struct {
	canvas    js.Value
	document  js.Value
	window    js.Value
	width     int
	height    int
	fbWidth   int
	fbHeight  int
	dpr       float64
	title     string
	clipboard string
	closed    bool

	// JS callback functions — must be stored to prevent GC.
	funcs []js.Func
}

// New creates a new browser Platform.
func New() *Platform {
	return &Platform{}
}

func (p *Platform) Init(cfg platform.Config) error {
	log.Println("web/wasm: Init start")
	p.document = js.Global().Get("document")
	p.window = js.Global().Get("window")
	p.title = cfg.Title
	if cfg.Title != "" {
		p.document.Set("title", cfg.Title)
	}

	p.canvas = p.document.Call("getElementById", "lux-canvas")
	if p.canvas.IsUndefined() || p.canvas.IsNull() {
		log.Println("web/wasm: canvas not found, creating one")
		p.canvas = p.document.Call("createElement", "canvas")
		p.canvas.Set("id", "lux-canvas")
		p.document.Get("body").Call("appendChild", p.canvas)
	}

	p.dpr = p.window.Get("devicePixelRatio").Float()
	if p.dpr < 1 {
		p.dpr = 1
	}

	// Prefer getBoundingClientRect — it reflects the post-layout rendered size,
	// including transforms and CSS like `width: 100% !important` from an
	// embedding container, whereas clientWidth rounds to integer CSS pixels.
	rect := p.canvas.Call("getBoundingClientRect")
	rectW := rect.Get("width").Float()
	rectH := rect.Get("height").Float()

	p.width = cfg.Width
	p.height = cfg.Height
	if p.width <= 0 {
		p.width = int(rectW)
		if p.width <= 0 {
			p.width = int(p.canvas.Get("clientWidth").Float())
		}
		if p.width <= 0 {
			p.width = 800
		}
	}
	if p.height <= 0 {
		p.height = int(rectH)
		if p.height <= 0 {
			p.height = int(p.canvas.Get("clientHeight").Float())
		}
		if p.height <= 0 {
			p.height = 600
		}
	}

	p.fbWidth = int(float64(p.width) * p.dpr)
	p.fbHeight = int(float64(p.height) * p.dpr)

	p.canvas.Set("width", p.fbWidth)
	p.canvas.Set("height", p.fbHeight)
	// Only set inline style when an explicit size was requested. If cfg.Width/
	// Height are 0, the canvas is expected to follow its container's layout
	// (e.g. `width: 100% !important`); setting inline pixel values would fight
	// that layout, and the CSS `!important` override would make them stale.
	if cfg.Width > 0 {
		p.canvas.Get("style").Set("width", strconv.Itoa(p.width)+"px")
	}
	if cfg.Height > 0 {
		p.canvas.Get("style").Set("height", strconv.Itoa(p.height)+"px")
	}

	log.Printf("web/wasm: Init OK: size=%dx%d fb=%dx%d dpr=%.1f", p.width, p.height, p.fbWidth, p.fbHeight, p.dpr)
	wgpu.SetWASMCanvas(p.canvas)
	return nil
}

func (p *Platform) Run(cb platform.Callbacks) error {
	log.Println("web/wasm: Run start")
	done := make(chan error, 1)

	p.registerEventListeners(cb)
	p.observeCanvasResize(cb)

	// Inform the app of the initial framebuffer size. Without this the first
	// frame renders with whatever the app assumed at startup, which may not
	// match the actual canvas when it is embedded in a user-styled container.
	if cb.OnResize != nil {
		cb.OnResize(p.fbWidth, p.fbHeight)
	}

	frameCount := 0
	var raf js.Func
	raf = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("web/wasm: PANIC in rAF frame %d: %v", frameCount, r)
				log.Println(errMsg)
				js.Global().Get("console").Call("error", errMsg)
				errorDiv := p.document.Call("getElementById", "error")
				if !errorDiv.IsUndefined() && !errorDiv.IsNull() {
					errorDiv.Get("style").Set("display", "flex")
					errorDiv.Set("textContent", errMsg)
				}
				done <- fmt.Errorf("%s", errMsg)
			}
		}()
		if p.closed {
			done <- nil
			return nil
		}
		frameCount++
		if frameCount <= 3 {
			log.Printf("web/wasm: rAF frame %d", frameCount)
		}
		if cb.OnFrame != nil {
			cb.OnFrame()
		}
		if frameCount <= 3 {
			log.Printf("web/wasm: rAF frame %d done", frameCount)
		}
		if frameCount == 1 {
			fn := p.window.Get("luxStatus")
			if !fn.IsUndefined() {
				fn.Invoke(fmt.Sprintf("Frame %d rendered", frameCount))
			}
		}
		p.window.Call("requestAnimationFrame", raf)
		return nil
	})
	p.funcs = append(p.funcs, raf)
	p.window.Call("requestAnimationFrame", raf)
	log.Println("web/wasm: rAF loop started, waiting on done channel")

	return <-done
}

func (p *Platform) registerEventListeners(cb platform.Callbacks) {
	// Mouse down
	p.addEventListener(p.canvas, "mousedown", func(e js.Value) {
		if cb.OnMouseButton != nil {
			x, y := p.canvasPos(e)
			btn := e.Get("button").Int()
			cb.OnMouseButton(x, y, btn, true)
		}
	})

	// Mouse up
	p.addEventListener(p.canvas, "mouseup", func(e js.Value) {
		if cb.OnMouseButton != nil {
			x, y := p.canvasPos(e)
			btn := e.Get("button").Int()
			cb.OnMouseButton(x, y, btn, false)
		}
	})

	// Mouse move
	p.addEventListener(p.canvas, "mousemove", func(e js.Value) {
		if cb.OnMouseMove != nil {
			x, y := p.canvasPos(e)
			cb.OnMouseMove(x, y)
		}
	})

	// Wheel/scroll
	p.addEventListener(p.canvas, "wheel", func(e js.Value) {
		e.Call("preventDefault")
		if cb.OnScroll != nil {
			dx := float32(e.Get("deltaX").Float())
			dy := float32(e.Get("deltaY").Float())
			mode := e.Get("deltaMode").Int()
			if mode == 1 { // DOM_DELTA_LINE
				dx *= 20
				dy *= 20
			} else if mode == 2 { // DOM_DELTA_PAGE
				dx *= float32(p.height)
				dy *= float32(p.height)
			}
			cb.OnScroll(dx, dy)
		}
	})

	// Key down
	p.addEventListener(p.window, "keydown", func(e js.Value) {
		code := e.Get("code").String()
		key := e.Get("key").String()
		mods := jsModifiers(e)
		mapped := mapKeyCode(code)
		if cb.OnKey != nil && mapped != "" {
			e.Call("preventDefault")
			repeat := 0
			if e.Get("repeat").Bool() {
				repeat = 2
			}
			cb.OnKey(mapped, repeat, mods)
		}
		if cb.OnChar != nil && len(key) == 1 && mods&0x2 == 0 { // not Ctrl
			cb.OnChar([]rune(key)[0])
		}
	})

	// Key up
	p.addEventListener(p.window, "keyup", func(e js.Value) {
		if cb.OnKey != nil {
			code := e.Get("code").String()
			mods := jsModifiers(e)
			mapped := mapKeyCode(code)
			if mapped != "" {
				cb.OnKey(mapped, 1, mods)
			}
		}
	})

	// IME composition
	p.addEventListener(p.canvas, "compositionupdate", func(e js.Value) {
		if cb.OnIMECompose != nil {
			text := e.Get("data").String()
			cb.OnIMECompose(text, len([]rune(text)), len([]rune(text)))
		}
	})
	p.addEventListener(p.canvas, "compositionend", func(e js.Value) {
		if cb.OnIMECommit != nil {
			text := e.Get("data").String()
			cb.OnIMECommit(text)
		}
	})

	// Window resize covers DPR changes (browser zoom, moving between monitors);
	// container size changes are handled by ResizeObserver in observeCanvasResize.
	p.addEventListener(p.window, "resize", func(_ js.Value) {
		p.handleResize(cb)
	})

	// Context menu (suppress to allow right-click handling)
	p.addEventListener(p.canvas, "contextmenu", func(e js.Value) {
		e.Call("preventDefault")
	})
}

func (p *Platform) addEventListener(target js.Value, event string, handler func(js.Value)) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		handler(args[0])
		return nil
	})
	p.funcs = append(p.funcs, fn)
	target.Call("addEventListener", event, fn)
}

// observeCanvasResize attaches a ResizeObserver to the canvas so that size
// changes driven by CSS layout (e.g. a container with `width: 100% !important`
// resizing when its parent changes) are reflected in both the backing-store
// size and the app's framebuffer. Falling back to the window resize listener
// alone misses these cases and produces squashed rendering with off-by hitboxes.
func (p *Platform) observeCanvasResize(cb platform.Callbacks) {
	ro := js.Global().Get("ResizeObserver")
	if ro.IsUndefined() || ro.IsNull() {
		return
	}
	fn := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		p.handleResize(cb)
		return nil
	})
	p.funcs = append(p.funcs, fn)
	observer := ro.New(fn)
	observer.Call("observe", p.canvas)
}

func (p *Platform) handleResize(cb platform.Callbacks) {
	// Read the post-layout size directly from the DOM — do NOT trust any cached
	// value, since a container with `width: 100% !important` may be bigger or
	// smaller than what we last stored.
	rect := p.canvas.Call("getBoundingClientRect")
	w := int(rect.Get("width").Float())
	h := int(rect.Get("height").Float())
	if w <= 0 || h <= 0 {
		return
	}
	dpr := p.window.Get("devicePixelRatio").Float()
	if dpr < 1 {
		dpr = 1
	}
	fbW := int(float64(w) * dpr)
	fbH := int(float64(h) * dpr)
	if fbW == p.fbWidth && fbH == p.fbHeight && dpr == p.dpr {
		return
	}
	p.dpr = dpr
	p.width = w
	p.height = h
	p.fbWidth = fbW
	p.fbHeight = fbH
	p.canvas.Set("width", fbW)
	p.canvas.Set("height", fbH)
	if cb.OnResize != nil {
		cb.OnResize(fbW, fbH)
	}
}

// canvasPos maps a pointer event to backing-store pixel coordinates using
// getBoundingClientRect. This derives the scale factor from the DOM on every
// call, so it stays correct even when the rendered CSS size diverges from what
// we cached (e.g. mid-resize, or after a user stylesheet forces a different
// width than we set via inline style).
func (p *Platform) canvasPos(e js.Value) (float32, float32) {
	rect := p.canvas.Call("getBoundingClientRect")
	rw := rect.Get("width").Float()
	rh := rect.Get("height").Float()
	if rw <= 0 || rh <= 0 {
		return 0, 0
	}
	cx := e.Get("clientX").Float() - rect.Get("left").Float()
	cy := e.Get("clientY").Float() - rect.Get("top").Float()
	bw := p.canvas.Get("width").Float()
	bh := p.canvas.Get("height").Float()
	return float32(cx / rw * bw), float32(cy / rh * bh)
}

func (p *Platform) Destroy() {
	for _, fn := range p.funcs {
		fn.Release()
	}
	p.funcs = nil
}

func (p *Platform) SetTitle(title string) {
	p.title = title
	p.document.Set("title", title)
}

func (p *Platform) WindowSize() (int, int) { return p.width, p.height }

func (p *Platform) FramebufferSize() (int, int) { return p.fbWidth, p.fbHeight }

func (p *Platform) ShouldClose() bool { return p.closed }

func (p *Platform) SetCursor(kind input.CursorKind) {
	p.canvas.Get("style").Set("cursor", mapCursor(kind))
}

func (p *Platform) SetIMECursorRect(_, _, _, _ int) {}

func (p *Platform) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.fbWidth = int(float64(w) * p.dpr)
	p.fbHeight = int(float64(h) * p.dpr)
	p.canvas.Set("width", p.fbWidth)
	p.canvas.Set("height", p.fbHeight)
	p.canvas.Get("style").Set("width", js.ValueOf(w).Call("toString").String()+"px")
	p.canvas.Get("style").Set("height", js.ValueOf(h).Call("toString").String()+"px")
}

func (p *Platform) SetFullscreen(fullscreen bool) {
	if fullscreen {
		p.canvas.Call("requestFullscreen")
	} else {
		p.document.Call("exitFullscreen")
	}
}

func (p *Platform) RequestFrame() {}

func (p *Platform) SetClipboard(text string) error {
	p.clipboard = text
	nav := js.Global().Get("navigator")
	if cb := nav.Get("clipboard"); !cb.IsUndefined() {
		cb.Call("writeText", text)
	}
	return nil
}

func (p *Platform) GetClipboard() (string, error) {
	return p.clipboard, nil
}

func (p *Platform) CreateWGPUSurface(_ uintptr) uintptr { return 0 }

// NativeHandle returns a sentinel value (1) so that WGPURenderer.Init
// enters the surface-creation path. The WASM wgpu layer ignores this
// value and looks up the canvas directly.
func (p *Platform) NativeHandle() uintptr { return 1 }

// NativeDisplay is unused on WASM.
func (p *Platform) NativeDisplay() uintptr { return 0 }

// ──────────────────────────────────────────────────────────────────────────────
// Key mapping: DOM event.code → lux key names (matching input.KeyNameToKey)
// ──────────────────────────────────────────────────────────────────────────────

func jsModifiers(e js.Value) int {
	var m int
	if e.Get("shiftKey").Bool() {
		m |= 0x1
	}
	if e.Get("ctrlKey").Bool() {
		m |= 0x2
	}
	if e.Get("altKey").Bool() {
		m |= 0x4
	}
	if e.Get("metaKey").Bool() {
		m |= 0x8
	}
	return m
}

var jsKeyMap = map[string]string{
	// Letters
	"KeyA": "A", "KeyB": "B", "KeyC": "C", "KeyD": "D",
	"KeyE": "E", "KeyF": "F", "KeyG": "G", "KeyH": "H",
	"KeyI": "I", "KeyJ": "J", "KeyK": "K", "KeyL": "L",
	"KeyM": "M", "KeyN": "N", "KeyO": "O", "KeyP": "P",
	"KeyQ": "Q", "KeyR": "R", "KeyS": "S", "KeyT": "T",
	"KeyU": "U", "KeyV": "V", "KeyW": "W", "KeyX": "X",
	"KeyY": "Y", "KeyZ": "Z",
	// Digits
	"Digit0": "0", "Digit1": "1", "Digit2": "2", "Digit3": "3",
	"Digit4": "4", "Digit5": "5", "Digit6": "6", "Digit7": "7",
	"Digit8": "8", "Digit9": "9",
	// Function keys
	"F1": "F1", "F2": "F2", "F3": "F3", "F4": "F4",
	"F5": "F5", "F6": "F6", "F7": "F7", "F8": "F8",
	"F9": "F9", "F10": "F10", "F11": "F11", "F12": "F12",
	// Navigation
	"ArrowUp": "Up", "ArrowDown": "Down", "ArrowLeft": "Left", "ArrowRight": "Right",
	"Home": "Home", "End": "End", "PageUp": "PageUp", "PageDown": "PageDown",
	// Editing
	"Backspace": "Backspace", "Delete": "Delete", "Tab": "Tab",
	"Enter": "Enter", "Escape": "Escape", "Space": "Space",
	"Insert": "Insert",
	// Modifiers
	"ShiftLeft": "LeftShift", "ShiftRight": "RightShift",
	"ControlLeft": "LeftControl", "ControlRight": "RightControl",
	"AltLeft": "LeftAlt", "AltRight": "RightAlt",
	"MetaLeft": "LeftSuper", "MetaRight": "RightSuper",
	// Punctuation
	"Minus":        "-", "Equal": "=",
	"BracketLeft":  "[", "BracketRight": "]",
	"Backslash":    "\\", "Semicolon": ";",
	"Quote":        "'", "Backquote": "`",
	"Comma":        ",", "Period": ".",
	"Slash":        "/",
	"CapsLock":     "CapsLock",
	"NumLock":      "NumLock",
	"ScrollLock":   "ScrollLock",
	"PrintScreen":  "PrintScreen",
	"Pause":        "Pause",
	// Numpad
	"Numpad0": "KP0", "Numpad1": "KP1", "Numpad2": "KP2",
	"Numpad3": "KP3", "Numpad4": "KP4", "Numpad5": "KP5",
	"Numpad6": "KP6", "Numpad7": "KP7", "Numpad8": "KP8",
	"Numpad9": "KP9",
	"NumpadDecimal": "KPDecimal", "NumpadEnter": "KPEnter",
	"NumpadAdd": "KPAdd", "NumpadSubtract": "KPSubtract",
	"NumpadMultiply": "KPMultiply", "NumpadDivide": "KPDivide",
}

func mapKeyCode(code string) string {
	if k, ok := jsKeyMap[code]; ok {
		return k
	}
	return ""
}

func mapCursor(kind input.CursorKind) string {
	switch kind {
	case input.CursorDefault:
		return "default"
	case input.CursorText:
		return "text"
	case input.CursorPointer:
		return "pointer"
	case input.CursorMove:
		return "move"
	case input.CursorResizeNS:
		return "ns-resize"
	case input.CursorResizeEW:
		return "ew-resize"
	case input.CursorResizeNESW:
		return "nesw-resize"
	case input.CursorResizeNWSE:
		return "nwse-resize"
	case input.CursorNotAllowed:
		return "not-allowed"
	case input.CursorCrosshair:
		return "crosshair"
	case input.CursorGrab:
		return "grab"
	case input.CursorGrabbing:
		return "grabbing"
	case input.CursorWait:
		return "wait"
	case input.CursorProgress:
		return "progress"
	case input.CursorNone:
		return "none"
	default:
		return "default"
	}
}
