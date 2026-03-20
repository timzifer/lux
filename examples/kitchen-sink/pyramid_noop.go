//go:build nogui || (windows && !gogpu)

package main

import (
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
)

// PyramidSurface is a no-op stub for headless/Windows builds.
type PyramidSurface struct{}

func NewPyramidSurface() *PyramidSurface { return &PyramidSurface{} }

func (p *PyramidSurface) Tick(_ time.Duration) {}

func (p *PyramidSurface) AcquireFrame(_ draw.Rect) (draw.TextureID, ui.FrameToken) {
	return 0, 0
}

func (p *PyramidSurface) ReleaseFrame(_ ui.FrameToken) {}

func (p *PyramidSurface) HandleMsg(_ any) bool { return false }
