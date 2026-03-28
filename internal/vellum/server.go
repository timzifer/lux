package vellum

import (
	"bytes"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

// Server is a Vellum protocol server that streams canvas frames and debug
// data to a connected Inspector client (RFC-012 §5).
//
// The PoC supports a single connected client at a time.
type Server struct {
	listener net.Listener
	addr     string

	mu     sync.Mutex
	client net.Conn // current connected client (nil if none)
	debug  bool     // client requested debug extensions

	done chan struct{}
}

// NewServer creates and starts a Vellum server listening on addr.
// The server accepts connections in the background.
func NewServer(addr string) (*Server, error) {
	// For Unix sockets, remove stale socket file.
	if strings.HasPrefix(addr, "unix://") {
		path := strings.TrimPrefix(addr, "unix://")
		os.Remove(path)
	}

	ln, err := Listen(addr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: ln,
		addr:     addr,
		done:     make(chan struct{}),
	}
	go s.acceptLoop()
	return s, nil
}

// acceptLoop accepts client connections in the background.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Printf("vellum: accept error: %v", err)
				continue
			}
		}
		s.handleClient(conn)
	}
}

// handleClient performs the handshake and registers the client.
func (s *Server) handleClient(conn net.Conn) {
	// Read handshake from client.
	channel, payload, err := ReadMessage(conn)
	if err != nil {
		log.Printf("vellum: handshake read error: %v", err)
		conn.Close()
		return
	}
	if channel != ChannelControl {
		log.Printf("vellum: expected control channel for handshake, got %d", channel)
		conn.Close()
		return
	}

	// Parse handshake.
	r := NewWireReader(bytes.NewReader(payload))
	opcode := r.readByte()
	if opcode != OpHandshake {
		log.Printf("vellum: expected handshake opcode 0x%02X, got 0x%02X", OpHandshake, opcode)
		conn.Close()
		return
	}
	version := r.ReadUint32()
	debugExt := r.ReadBool()
	if r.Err() != nil {
		log.Printf("vellum: handshake decode error: %v", r.Err())
		conn.Close()
		return
	}

	_ = version // PoC: accept any version

	// Send handshake response.
	var resp bytes.Buffer
	ww := NewWireWriter(&resp)
	ww.writeByte(OpHandshake)
	ww.WriteUint32(1) // version
	ww.WriteBool(debugExt)
	if ww.Err() != nil {
		log.Printf("vellum: handshake encode error: %v", ww.Err())
		conn.Close()
		return
	}
	if err := WriteMessage(conn, ChannelControl, resp.Bytes()); err != nil {
		log.Printf("vellum: handshake write error: %v", err)
		conn.Close()
		return
	}

	// Disconnect previous client.
	s.mu.Lock()
	if s.client != nil {
		s.client.Close()
	}
	s.client = conn
	s.debug = debugExt
	s.mu.Unlock()

	log.Printf("vellum: inspector connected (debug=%v)", debugExt)
}

// HasClient reports whether an inspector client is currently connected.
func (s *Server) HasClient() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil
}

// DebugEnabled reports whether the connected client requested debug extensions.
func (s *Server) DebugEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.debug
}

// SendCanvas sends a canvas frame (channel 1) to the connected client.
func (s *Server) SendCanvas(data []byte) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return
	}
	if err := WriteMessage(client, ChannelCanvas, data); err != nil {
		log.Printf("vellum: send canvas error: %v", err)
		s.disconnectClient()
	}
}

// SendControl sends a control message (channel 0) to the connected client.
func (s *Server) SendControl(data []byte) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return
	}
	if err := WriteMessage(client, ChannelControl, data); err != nil {
		log.Printf("vellum: send control error: %v", err)
		s.disconnectClient()
	}
}

// disconnectClient closes and removes the current client connection.
func (s *Server) disconnectClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		s.client.Close()
		s.client = nil
		s.debug = false
		log.Printf("vellum: inspector disconnected")
	}
}

// Close shuts down the server.
func (s *Server) Close() error {
	close(s.done)
	s.disconnectClient()
	return s.listener.Close()
}

// Addr returns the server's listen address.
func (s *Server) Addr() string { return s.addr }
