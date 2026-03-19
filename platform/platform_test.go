//go:build nogui

package platform_test

import (
	"testing"

	"github.com/timzifer/lux/input"
	"github.com/timzifer/lux/platform"
)

// testPlatform is a minimal Platform implementation for interface conformance.
type testPlatform struct {
	w, h       int
	title      string
	fullscreen bool
	clipboard  string
}

func (p *testPlatform) Init(cfg platform.Config) error {
	p.w = cfg.Width
	p.h = cfg.Height
	p.title = cfg.Title
	return nil
}

func (p *testPlatform) Run(cb platform.Callbacks) error               { return nil }
func (p *testPlatform) Destroy()                                       {}
func (p *testPlatform) SetTitle(title string)                          { p.title = title }
func (p *testPlatform) WindowSize() (int, int)                         { return p.w, p.h }
func (p *testPlatform) FramebufferSize() (int, int)                    { return p.w, p.h }
func (p *testPlatform) ShouldClose() bool                              { return false }
func (p *testPlatform) SetCursor(_ input.CursorKind)                   {}
func (p *testPlatform) SetIMECursorRect(_, _, _, _ int)                {}
func (p *testPlatform) SetSize(w, h int)                               { p.w = w; p.h = h }
func (p *testPlatform) SetFullscreen(fs bool)                          { p.fullscreen = fs }
func (p *testPlatform) RequestFrame()                                  {}
func (p *testPlatform) SetClipboard(text string) error                 { p.clipboard = text; return nil }
func (p *testPlatform) GetClipboard() (string, error)                  { return p.clipboard, nil }
func (p *testPlatform) CreateWGPUSurface(_ uintptr) uintptr            { return 0 }

// TestPlatformInterfaceConformance verifies the Platform interface has all required methods.
func TestPlatformInterfaceConformance(t *testing.T) {
	var _ platform.Platform = (*testPlatform)(nil)
}

func TestPlatformSetSize(t *testing.T) {
	p := &testPlatform{}
	_ = p.Init(platform.Config{Width: 800, Height: 600})

	w, h := p.WindowSize()
	if w != 800 || h != 600 {
		t.Errorf("initial size = (%d, %d), want (800, 600)", w, h)
	}

	p.SetSize(1024, 768)
	w, h = p.WindowSize()
	if w != 1024 || h != 768 {
		t.Errorf("after SetSize = (%d, %d), want (1024, 768)", w, h)
	}
}

func TestPlatformSetFullscreen(t *testing.T) {
	p := &testPlatform{}
	if p.fullscreen {
		t.Error("initial fullscreen should be false")
	}

	p.SetFullscreen(true)
	if !p.fullscreen {
		t.Error("SetFullscreen(true) should set fullscreen")
	}

	p.SetFullscreen(false)
	if p.fullscreen {
		t.Error("SetFullscreen(false) should unset fullscreen")
	}
}

func TestPlatformClipboard(t *testing.T) {
	p := &testPlatform{}

	text, err := p.GetClipboard()
	if err != nil || text != "" {
		t.Errorf("initial clipboard = (%q, %v), want (\"\", nil)", text, err)
	}

	if err := p.SetClipboard("Hello, World!"); err != nil {
		t.Fatalf("SetClipboard failed: %v", err)
	}

	text, err = p.GetClipboard()
	if err != nil || text != "Hello, World!" {
		t.Errorf("after SetClipboard = (%q, %v), want (\"Hello, World!\", nil)", text, err)
	}
}

func TestPlatformCreateWGPUSurface(t *testing.T) {
	p := &testPlatform{}
	surface := p.CreateWGPUSurface(0)
	if surface != 0 {
		t.Errorf("CreateWGPUSurface(0) = %d, want 0", surface)
	}
}
