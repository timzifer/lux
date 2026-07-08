package vellum

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

// transport.go provides length-prefixed framing over net.Conn (RFC-012 §8B).
//
// Wire format per message:
//   [channel: 1 byte] [length: 4 bytes big-endian uint32] [payload: length bytes]

// Channel IDs.
const (
	ChannelControl byte = 0 // Handshake, AccessTree, Debug extensions
	ChannelCanvas  byte = 1 // Canvas stream (BeginFrame … EndFrame)
)

// WriteMessage writes a length-prefixed message to w.
func WriteMessage(w io.Writer, channel byte, payload []byte) error {
	var header [5]byte
	header[0] = channel
	binary.BigEndian.PutUint32(header[1:5], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("vellum: write header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return fmt.Errorf("vellum: write payload: %w", err)
		}
	}
	return nil
}

// ReadMessage reads a length-prefixed message from r.
func ReadMessage(r io.Reader) (channel byte, payload []byte, err error) {
	var header [5]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}
	channel = header[0]
	length := binary.BigEndian.Uint32(header[1:5])
	if length > 16*1024*1024 { // 16 MB max message size
		return 0, nil, fmt.Errorf("vellum: message too large: %d bytes", length)
	}
	payload = make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, fmt.Errorf("vellum: read payload: %w", err)
		}
	}
	return channel, payload, nil
}

// ParseAddr parses a Vellum address string into network and address parts.
// Supported formats:
//   - "unix:///path/to/socket" → ("unix", "/path/to/socket")
//   - "tcp://host:port"       → ("tcp", "host:port")
func ParseAddr(addr string) (network, address string, err error) {
	if strings.HasPrefix(addr, "unix://") {
		return "unix", strings.TrimPrefix(addr, "unix://"), nil
	}
	if strings.HasPrefix(addr, "tcp://") {
		return "tcp", strings.TrimPrefix(addr, "tcp://"), nil
	}
	return "", "", fmt.Errorf("vellum: unsupported address format: %q (expected unix:// or tcp://)", addr)
}

// Listen creates a net.Listener for the given Vellum address.
func Listen(addr string) (net.Listener, error) {
	network, address, err := ParseAddr(addr)
	if err != nil {
		return nil, err
	}
	return net.Listen(network, address)
}

// Dial connects to the given Vellum address.
func Dial(addr string) (net.Conn, error) {
	network, address, err := ParseAddr(addr)
	if err != nil {
		return nil, err
	}
	return net.Dial(network, address)
}
