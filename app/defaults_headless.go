//go:build nogui

package app

import (
	"github.com/timzifer/lux/internal/gpu"
	"github.com/timzifer/lux/platform"
)

func defaultPlatformFactory() platform.Platform {
	return &headlessPlatform{}
}

func defaultRendererFactory() gpu.Renderer {
	return &gpu.NoopRenderer{}
}

// headlessPlatform is a no-op platform for testing and CI environments.
type headlessPlatform struct {
	title string
	w, h  int
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
	// In headless mode, run exactly one frame then exit.
	if cb.OnFrame != nil {
		cb.OnFrame()
	}
	return nil
}

func (p *headlessPlatform) Destroy()                  {}
func (p *headlessPlatform) SetTitle(title string)      { p.title = title }
func (p *headlessPlatform) WindowSize() (int, int)     { return p.w, p.h }
func (p *headlessPlatform) FramebufferSize() (int, int) { return p.w, p.h }
func (p *headlessPlatform) ShouldClose() bool          { return true }
