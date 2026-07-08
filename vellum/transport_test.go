package vellum

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/timzifer/lux/draw"
)

func TestMessageRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("hello vellum")
	if err := WriteMessage(&buf, ChannelCanvas, payload); err != nil {
		t.Fatal(err)
	}
	ch, got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if ch != ChannelCanvas {
		t.Errorf("channel: got %d, want %d", ch, ChannelCanvas)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload mismatch")
	}
}

func TestMessageEmptyPayload(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMessage(&buf, ChannelControl, nil); err != nil {
		t.Fatal(err)
	}
	ch, got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if ch != ChannelControl {
		t.Errorf("channel: got %d, want %d", ch, ChannelControl)
	}
	if len(got) != 0 {
		t.Errorf("payload: got %d bytes, want 0", len(got))
	}
}

func TestServerClientHandshake(t *testing.T) {
	// Create a temporary socket path.
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "test.sock")
	addr := "unix://" + sockPath

	// Start server.
	srv, err := NewServer(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Give server a moment to start listening.
	time.Sleep(50 * time.Millisecond)

	// Connect client.
	client, err := Connect(addr, WithDebugExtensions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Give handshake time to complete.
	time.Sleep(50 * time.Millisecond)

	if !srv.HasClient() {
		t.Error("server reports no client after connect")
	}
	if !srv.DebugEnabled() {
		t.Error("server reports debug not enabled")
	}
}

func TestServerClientFrameTransfer(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "test.sock")

	// Check if socket directory is accessible.
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}

	addr := "unix://" + sockPath

	srv, err := NewServer(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	time.Sleep(50 * time.Millisecond)

	client, err := Connect(addr, WithDebugExtensions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// Build a frame.
	buf := NewFrameBuffer()
	enc := NewCanvasEncoder(nil, buf)
	enc.BeginFrame(1, draw.R(0, 0, 800, 600), 1.0)
	enc.FillRect(draw.R(0, 0, 100, 100), draw.SolidPaint(draw.RGBA(255, 0, 0, 255)))
	enc.EndFrame(nil)

	// Send frame.
	srv.SendCanvas(buf.Bytes())

	// Receive frame.
	select {
	case data := <-client.Frames():
		frame, err := DecodeFrame(data)
		if err != nil {
			t.Fatal(err)
		}
		if frame.FrameID != 1 {
			t.Errorf("frameID: got %d, want 1", frame.FrameID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for frame")
	}
}
