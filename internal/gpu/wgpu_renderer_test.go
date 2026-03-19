//go:build nogui

package gpu

import (
	"testing"
)

// TestNoopRendererInterface verifies NoopRenderer implements Renderer.
func TestNoopRendererInterface(t *testing.T) {
	var _ Renderer = (*NoopRenderer)(nil)
}

// TestNoopRendererLifecycle verifies the noop renderer doesn't panic.
func TestNoopRendererLifecycle(t *testing.T) {
	r := &NoopRenderer{}
	if err := r.Init(Config{Width: 800, Height: 600}); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	r.Resize(1024, 768)
	r.BeginFrame()
	r.EndFrame()
	r.Destroy()
}
