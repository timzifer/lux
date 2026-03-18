//go:build nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
)

// headlessConfig accumulates headless testing configuration.
// It's a package-level var so that multiple Option closures can share state.
var headlessConfig struct {
	frames int
	clicks []mouseClick
}

func init() {
	resetHeadlessConfig()
}

func resetHeadlessConfig() {
	headlessConfig.frames = 0
	headlessConfig.clicks = nil
}

func defaultPlatformFactory() platform.Platform {
	p := &headlessPlatform{
		frames: headlessConfig.frames,
		clicks: headlessConfig.clicks,
	}
	resetHeadlessConfig()
	return p
}

func defaultRendererFactory() gpu.Renderer {
	return &gpu.NoopRenderer{}
}

// headlessPlatform is a no-op platform for testing and CI environments.
type headlessPlatform struct {
	title  string
	w, h   int
	frames int          // total frames to run (0 means 1)
	clicks []mouseClick // injected mouse clicks
}

type mouseClick struct {
	frame int     // which frame to inject (0-based)
	x, y  float32 // click position
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
		// Inject clicks scheduled for this frame.
		for _, c := range p.clicks {
			if c.frame == f && cb.OnMouseButton != nil {
				cb.OnMouseButton(c.x, c.y, 0, true)
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
