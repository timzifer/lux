package vellum

import (
	"bytes"
	"fmt"
	"io"
)

// FrameBuffer is a TLV-based byte buffer for encoding/decoding Vellum frames.
// Each entry is: opcode (1 byte) + varint length + payload.
type FrameBuffer struct {
	buf bytes.Buffer
}

// NewFrameBuffer creates a FrameBuffer with a pre-allocated capacity.
func NewFrameBuffer() *FrameBuffer {
	fb := &FrameBuffer{}
	fb.buf.Grow(65536) // 64 KB initial capacity
	return fb
}

// WriteOp writes a complete TLV entry: opcode + varint(len(payload)) + payload.
// The buildFn is called with a WireWriter to produce the payload.
func (fb *FrameBuffer) WriteOp(opcode byte, buildFn func(w *WireWriter)) {
	// Build payload into a temporary buffer.
	var payload bytes.Buffer
	if buildFn != nil {
		ww := NewWireWriter(&payload)
		buildFn(ww)
		if ww.Err() != nil {
			return
		}
	}

	// Write opcode.
	fb.buf.WriteByte(opcode)

	// Write varint length.
	writeVarintTo(&fb.buf, uint32(payload.Len()))

	// Write payload.
	fb.buf.Write(payload.Bytes())
}

// WriteOpRaw writes a pre-built payload.
func (fb *FrameBuffer) WriteOpRaw(opcode byte, payload []byte) {
	fb.buf.WriteByte(opcode)
	writeVarintTo(&fb.buf, uint32(len(payload)))
	fb.buf.Write(payload)
}

// ReadOp reads the next TLV entry from data starting at offset.
// Returns the opcode, payload bytes, bytes consumed, and any error.
func ReadOp(data []byte, offset int) (opcode byte, payload []byte, consumed int, err error) {
	if offset >= len(data) {
		return 0, nil, 0, io.EOF
	}
	opcode = data[offset]
	offset++
	consumed = 1

	// Read varint length.
	length, n, err := readVarintFrom(data, offset)
	if err != nil {
		return 0, nil, 0, err
	}
	offset += n
	consumed += n

	// Read payload.
	end := offset + int(length)
	if end > len(data) {
		return 0, nil, 0, fmt.Errorf("vellum: payload truncated: need %d bytes at offset %d, have %d", length, offset, len(data)-offset)
	}
	payload = data[offset:end]
	consumed += int(length)

	return opcode, payload, consumed, nil
}

// Bytes returns the accumulated buffer contents.
func (fb *FrameBuffer) Bytes() []byte {
	return fb.buf.Bytes()
}

// Len returns the current buffer length.
func (fb *FrameBuffer) Len() int {
	return fb.buf.Len()
}

// Reset clears the buffer for reuse.
func (fb *FrameBuffer) Reset() {
	fb.buf.Reset()
}

// writeVarintTo writes a LEB128-encoded uint32 to a bytes.Buffer.
func writeVarintTo(buf *bytes.Buffer, v uint32) {
	for v >= 0x80 {
		buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}

// readVarintFrom reads a LEB128 varint from data at the given offset.
// Returns the value, number of bytes consumed, and any error.
func readVarintFrom(data []byte, offset int) (uint32, int, error) {
	var result uint32
	var shift uint
	for i := 0; ; i++ {
		if offset+i >= len(data) {
			return 0, 0, fmt.Errorf("vellum: varint truncated")
		}
		b := data[offset+i]
		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, i + 1, nil
		}
		shift += 7
		if shift >= 35 {
			return 0, 0, fmt.Errorf("vellum: varint overflow")
		}
	}
}
