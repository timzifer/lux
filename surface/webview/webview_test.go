package webview

import (
	"testing"

	"github.com/timzifer/lux/draw"
)

func TestHistoryNavigationState(t *testing.T) {
	w := New("https://example.com")
	w.Navigate("https://example.com/docs")
	w.Navigate("https://example.com/docs/rfc")

	if !w.CanGoBack() {
		t.Fatal("CanGoBack() = false, want true")
	}
	if w.CanGoForward() {
		t.Fatal("CanGoForward() = true, want false")
	}

	w.Back()
	if !w.CanGoBack() {
		t.Fatal("after Back(), CanGoBack() = false, want true")
	}
	if !w.CanGoForward() {
		t.Fatal("after Back(), CanGoForward() = false, want true")
	}

	w.Forward()
	if w.CanGoForward() {
		t.Fatal("after Forward(), CanGoForward() = true, want false")
	}
}

func TestAcquireFrameReturnsToken(t *testing.T) {
	w := New("https://example.com")
	tex, token := w.AcquireFrame(draw.Rect{X: 1, Y: 2, W: 300, H: 200})
	if tex != 0 {
		t.Fatalf("AcquireFrame() texture = %d, want 0 for stub backend", tex)
	}
	if token == 0 {
		t.Fatal("AcquireFrame() token = 0, want non-zero token")
	}

	w.ReleaseFrame(token)
}

func TestCloseIsIdempotent(t *testing.T) {
	w := New("https://example.com")
	if err := w.Close(); err != nil {
		t.Fatalf("Close() first call error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() second call error = %v", err)
	}
}
