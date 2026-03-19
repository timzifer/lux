//go:build nogui

package app

import (
	"runtime"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
)

// headlessConfig accumulates headless testing configuration.
// It's a package-level var so that multiple Option closures can share state.
var headlessConfig struct {
	frames     int
	clicks     []mouseClick
	mouseMoves []mouseMove
	scrolls    []scrollEvent
	keyEvents  []keyEvent
	charEvents []charEvent
}

func init() {
	resetHeadlessConfig()
}

func resetHeadlessConfig() {
	headlessConfig.frames = 0
	headlessConfig.clicks = nil
	headlessConfig.mouseMoves = nil
	headlessConfig.scrolls = nil
	headlessConfig.keyEvents = nil
	headlessConfig.charEvents = nil
}

func defaultPlatformFactory() platform.Platform {
	p := &headlessPlatform{
		frames:     headlessConfig.frames,
		clicks:     headlessConfig.clicks,
		mouseMoves: headlessConfig.mouseMoves,
		scrolls:    headlessConfig.scrolls,
		keyEvents:  headlessConfig.keyEvents,
		charEvents: headlessConfig.charEvents,
	}
	resetHeadlessConfig()
	return p
}

func defaultRendererFactory() gpu.Renderer {
	return &gpu.NoopRenderer{}
}

// headlessPlatform is a no-op platform for testing and CI environments.
type headlessPlatform struct {
	title      string
	w, h       int
	frames     int          // total frames to run (0 means 1)
	clicks     []mouseClick // injected mouse clicks
	mouseMoves []mouseMove  // injected mouse moves
	scrolls    []scrollEvent
	keyEvents  []keyEvent
	charEvents []charEvent
	clipboard  string       // simulated clipboard for testing
}

type mouseClick struct {
	frame int     // which frame to inject (0-based)
	x, y  float32 // click position
}

type mouseMove struct {
	frame int     // which frame to inject (0-based)
	x, y  float32 // cursor position
}

type scrollEvent struct {
	frame        int
	deltaX, deltaY float32
}

type keyEvent struct {
	frame  int
	key    string
	action int // 0=press, 1=release, 2=repeat
	mods   int
}

type charEvent struct {
	frame int
	ch    rune
}

func (p *headlessPlatform) Init(cfg platform.Config) error {
	p.title = cfg.Title
	p.w = cfg.Width
	p.h = cfg.Height
	if p.w <= 0 {
		p.w = 800
	}
	if p.h <= 0 {
		p.h = 600
	}
	return nil
}

func (p *headlessPlatform) Run(cb platform.Callbacks) error {
	nFrames := p.frames
	if nFrames <= 0 {
		nFrames = 1
	}
	for f := 0; f < nFrames; f++ {
		// Yield to allow goroutines (e.g. Cmd dispatch) to run.
		runtime.Gosched()

		// Inject mouse moves scheduled for this frame.
		for _, m := range p.mouseMoves {
			if m.frame == f && cb.OnMouseMove != nil {
				cb.OnMouseMove(m.x, m.y)
			}
		}
		// Inject clicks scheduled for this frame.
		for _, c := range p.clicks {
			if c.frame == f && cb.OnMouseButton != nil {
				cb.OnMouseButton(c.x, c.y, 0, true)
			}
		}
		// Inject scroll events.
		for _, s := range p.scrolls {
			if s.frame == f && cb.OnScroll != nil {
				cb.OnScroll(s.deltaX, s.deltaY)
			}
		}
		// Inject key events.
		for _, k := range p.keyEvents {
			if k.frame == f && cb.OnKey != nil {
				cb.OnKey(k.key, k.action, k.mods)
			}
		}
		// Inject char events.
		for _, c := range p.charEvents {
			if c.frame == f && cb.OnChar != nil {
				cb.OnChar(c.ch)
			}
		}
		if cb.OnFrame != nil {
			cb.OnFrame()
		}
	}
	return nil
}

func (p *headlessPlatform) Destroy()                    {}
func (p *headlessPlatform) SetTitle(title string)        { p.title = title }
func (p *headlessPlatform) WindowSize() (int, int)       { return p.w, p.h }
func (p *headlessPlatform) FramebufferSize() (int, int)  { return p.w, p.h }
func (p *headlessPlatform) ShouldClose() bool            { return true }
func (p *headlessPlatform) SetCursor(_ input.CursorKind)    {}
func (p *headlessPlatform) SetIMECursorRect(_, _, _, _ int) {}

// Phase 5 — Platform Extension (RFC §7.1).
func (p *headlessPlatform) SetSize(w, h int)                     { p.w = w; p.h = h }
func (p *headlessPlatform) SetFullscreen(_ bool)                 {}
func (p *headlessPlatform) RequestFrame()                        {}
func (p *headlessPlatform) SetClipboard(text string) error       { p.clipboard = text; return nil }
func (p *headlessPlatform) GetClipboard() (string, error)        { return p.clipboard, nil }
func (p *headlessPlatform) CreateWGPUSurface(_ uintptr) uintptr  { return 0 }

// WithHeadlessFrames sets how many frames the headless platform runs.
func WithHeadlessFrames(n int) Option {
	return func(o *options) {
		headlessConfig.frames = n
	}
}

// WithHeadlessClick injects a left-click at (x, y) on the given frame (0-based).
func WithHeadlessClick(frame int, x, y float32) Option {
	return func(o *options) {
		headlessConfig.clicks = append(headlessConfig.clicks, mouseClick{frame: frame, x: x, y: y})
	}
}

// WithHeadlessMouseMove injects a mouse move to (x, y) on the given frame (0-based).
func WithHeadlessMouseMove(frame int, x, y float32) Option {
	return func(o *options) {
		headlessConfig.mouseMoves = append(headlessConfig.mouseMoves, mouseMove{frame: frame, x: x, y: y})
	}
}

// WithHeadlessScroll injects a scroll event on the given frame (0-based).
func WithHeadlessScroll(frame int, deltaX, deltaY float32) Option {
	return func(o *options) {
		headlessConfig.scrolls = append(headlessConfig.scrolls, scrollEvent{frame: frame, deltaX: deltaX, deltaY: deltaY})
	}
}

// WithHeadlessKey injects a key event on the given frame (0-based).
// action: 0=press, 1=release, 2=repeat.  mods: bit 0=Shift, 1=Ctrl, 2=Alt, 3=Super.
func WithHeadlessKey(frame int, key string, action int, mods int) Option {
	return func(o *options) {
		headlessConfig.keyEvents = append(headlessConfig.keyEvents, keyEvent{frame: frame, key: key, action: action, mods: mods})
	}
}

// WithHeadlessChar injects a character input event on the given frame (0-based).
func WithHeadlessChar(frame int, ch rune) Option {
	return func(o *options) {
		headlessConfig.charEvents = append(headlessConfig.charEvents, charEvent{frame: frame, ch: ch})
	}
}
