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

// TestConfigDRMFields verifies the DRM fields on Config.
func TestConfigDRMFields(t *testing.T) {
	cfg := Config{
		Width:          1920,
		Height:         1080,
		DRMfd:          3,
		DRMConnectorID: 42,
	}
	if cfg.DRMfd != 3 {
		t.Errorf("DRMfd = %d, want 3", cfg.DRMfd)
	}
	if cfg.DRMConnectorID != 42 {
		t.Errorf("DRMConnectorID = %d, want 42", cfg.DRMConnectorID)
	}
	if cfg.NativeDisplay != 0 {
		t.Errorf("NativeDisplay = %d, want 0", cfg.NativeDisplay)
	}
}

// TestConfigDRMSentinel verifies that -1 is the sentinel for unused DRM fd.
func TestConfigDRMSentinel(t *testing.T) {
	cfg := Config{DRMfd: -1}
	if cfg.DRMfd >= 0 {
		t.Error("DRMfd -1 should be treated as unused")
	}
}
