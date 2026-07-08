package vellum

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

// Client connects to a Vellum server and receives canvas frames and
// debug data (RFC-012 §6).
type Client struct {
	conn net.Conn

	// Channels for received data.
	frames  chan []byte // canvas frames (channel 1)
	control chan []byte // control messages (channel 0)

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// ClientOption configures a Client.
type ClientOption func(*clientOptions)

type clientOptions struct {
	debugExtensions bool
}

// WithDebugExtensions requests debug data in the handshake.
func WithDebugExtensions() ClientOption {
	return func(o *clientOptions) { o.debugExtensions = true }
}

// Connect dials a Vellum server, performs the handshake, and starts
// receiving frames in the background.
func Connect(addr string, opts ...ClientOption) (*Client, error) {
	var cfg clientOptions
	for _, opt := range opts {
		opt(&cfg)
	}

	conn, err := Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("vellum: connect: %w", err)
	}

	// Send handshake.
	var hs bytes.Buffer
	ww := NewWireWriter(&hs)
	ww.writeByte(OpHandshake)
	ww.WriteUint32(1) // version
	ww.WriteBool(cfg.debugExtensions)
	if ww.Err() != nil {
		conn.Close()
		return nil, fmt.Errorf("vellum: handshake encode: %w", ww.Err())
	}
	if err := WriteMessage(conn, ChannelControl, hs.Bytes()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("vellum: handshake send: %w", err)
	}

	// Read handshake response.
	ch, payload, err := ReadMessage(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("vellum: handshake recv: %w", err)
	}
	if ch != ChannelControl {
		conn.Close()
		return nil, fmt.Errorf("vellum: expected control channel in handshake response, got %d", ch)
	}
	r := NewWireReader(bytes.NewReader(payload))
	opcode := r.readByte()
	if opcode != OpHandshake {
		conn.Close()
		return nil, fmt.Errorf("vellum: expected handshake opcode, got 0x%02X", opcode)
	}
	_ = r.ReadUint32() // version
	_ = r.ReadBool()   // debugExtensions echoed back

	c := &Client{
		conn:    conn,
		frames:  make(chan []byte, 4),
		control: make(chan []byte, 16),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// readLoop reads messages from the server and dispatches to channels.
func (c *Client) readLoop() {
	defer close(c.done)
	for {
		ch, payload, err := ReadMessage(c.conn)
		if err != nil {
			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if !closed {
				// Connection broken unexpectedly.
			}
			return
		}

		switch ch {
		case ChannelCanvas:
			select {
			case c.frames <- payload:
			default:
				// Drop oldest frame if consumer is slow.
				select {
				case <-c.frames:
				default:
				}
				c.frames <- payload
			}
		case ChannelControl:
			select {
			case c.control <- payload:
			default:
				// Drop oldest control message if consumer is slow.
				select {
				case <-c.control:
				default:
				}
				c.control <- payload
			}
		}
	}
}

// NextFrame blocks until the next canvas frame is received.
// Returns the raw frame data (BeginFrame … EndFrame TLV entries).
func (c *Client) NextFrame() ([]byte, error) {
	select {
	case data, ok := <-c.frames:
		if !ok {
			return nil, fmt.Errorf("vellum: client closed")
		}
		return data, nil
	case <-c.done:
		return nil, fmt.Errorf("vellum: connection closed")
	}
}

// NextControl blocks until the next control message is received.
func (c *Client) NextControl() ([]byte, error) {
	select {
	case data, ok := <-c.control:
		if !ok {
			return nil, fmt.Errorf("vellum: client closed")
		}
		return data, nil
	case <-c.done:
		return nil, fmt.Errorf("vellum: connection closed")
	}
}

// Frames returns the channel for receiving canvas frame data.
func (c *Client) Frames() <-chan []byte { return c.frames }

// Control returns the channel for receiving control messages.
func (c *Client) Control() <-chan []byte { return c.control }

// Done returns a channel that is closed when the connection ends.
func (c *Client) Done() <-chan struct{} { return c.done }

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return c.conn.Close()
}
