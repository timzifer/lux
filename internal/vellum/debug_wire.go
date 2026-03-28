package vellum

import (
	"bytes"
	"time"
)

func newBytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// debug_wire.go provides serialization for debug extension types (RFC-012 §4).

// writeDebugFrameInfo serializes a DebugFrameInfo into the wire format.
func writeDebugFrameInfo(w *WireWriter, info *DebugFrameInfo) {
	w.WriteUint64(info.FrameID)
	w.WriteInt64(int64(info.FrameTime))
	w.WriteInt64(int64(info.UpdateTime))
	w.WriteInt64(int64(info.ReconcileTime))
	w.WriteInt64(int64(info.LayoutTime))
	w.WriteInt64(int64(info.PaintTime))
	w.WriteUint32(info.WidgetCount)
	w.WriteVarint(uint32(len(info.DirtyWidgets)))
	for _, uid := range info.DirtyWidgets {
		w.WriteUint64(uid)
	}
}

// readDebugFrameInfo deserializes a DebugFrameInfo from the wire format.
func readDebugFrameInfo(r *WireReader) DebugFrameInfo {
	info := DebugFrameInfo{
		FrameID:       r.ReadUint64(),
		FrameTime:     time.Duration(r.ReadInt64()),
		UpdateTime:    time.Duration(r.ReadInt64()),
		ReconcileTime: time.Duration(r.ReadInt64()),
		LayoutTime:    time.Duration(r.ReadInt64()),
		PaintTime:     time.Duration(r.ReadInt64()),
		WidgetCount:   r.ReadUint32(),
	}
	n := r.ReadVarint()
	if n > 0 {
		info.DirtyWidgets = make([]uint64, n)
		for i := range info.DirtyWidgets {
			info.DirtyWidgets[i] = r.ReadUint64()
		}
	}
	return info
}

// writeDebugWidgetTree serializes a DebugWidgetTree.
func writeDebugWidgetTree(w *WireWriter, tree *DebugWidgetTree) {
	w.WriteUint64(tree.Version)
	w.WriteVarint(uint32(len(tree.Nodes)))
	for _, node := range tree.Nodes {
		writeDebugWidgetNode(w, &node)
	}
}

func writeDebugWidgetNode(w *WireWriter, node *DebugWidgetNode) {
	w.WriteUint64(node.UID)
	w.WriteString(node.TypeName)

	// Props as key-value pairs.
	w.WriteVarint(uint32(len(node.Props)))
	for k, v := range node.Props {
		w.WriteString(k)
		w.WriteString(v)
	}

	w.WriteString(node.StateDump)
	w.WriteRect(node.Bounds)
	w.WriteInsets(node.Padding)
	w.WriteInsets(node.Margin)
	w.WriteBool(node.Dirty)
}

// readDebugWidgetTree deserializes a DebugWidgetTree.
func readDebugWidgetTree(r *WireReader) DebugWidgetTree {
	tree := DebugWidgetTree{
		Version: r.ReadUint64(),
	}
	n := r.ReadVarint()
	if n > 0 {
		tree.Nodes = make([]DebugWidgetNode, n)
		for i := range tree.Nodes {
			tree.Nodes[i] = readDebugWidgetNode(r)
		}
	}
	return tree
}

func readDebugWidgetNode(r *WireReader) DebugWidgetNode {
	node := DebugWidgetNode{
		UID:      r.ReadUint64(),
		TypeName: r.ReadString(),
	}

	nProps := r.ReadVarint()
	if nProps > 0 {
		node.Props = make(map[string]string, nProps)
		for i := uint32(0); i < nProps; i++ {
			k := r.ReadString()
			v := r.ReadString()
			node.Props[k] = v
		}
	}

	node.StateDump = r.ReadString()
	node.Bounds = r.ReadRect()
	node.Padding = r.ReadInsets()
	node.Margin = r.ReadInsets()
	node.Dirty = r.ReadBool()
	return node
}

// writeDebugEventLog serializes a DebugEventLog.
func writeDebugEventLog(w *WireWriter, log *DebugEventLog) {
	w.WriteUint64(log.FrameID)
	w.WriteVarint(uint32(len(log.Events)))
	for _, ev := range log.Events {
		writeDebugEvent(w, &ev)
	}
}

func writeDebugEvent(w *WireWriter, ev *DebugEvent) {
	w.WriteInt64(int64(ev.Timestamp))
	w.WriteString(ev.Kind)
	w.WriteUint64(ev.TargetUID)
	w.WriteString(ev.TargetType)
	w.WriteString(ev.Detail)
	w.WriteBool(ev.Consumed)
}

// readDebugEventLog deserializes a DebugEventLog.
func readDebugEventLog(r *WireReader) DebugEventLog {
	log := DebugEventLog{
		FrameID: r.ReadUint64(),
	}
	n := r.ReadVarint()
	if n > 0 {
		log.Events = make([]DebugEvent, n)
		for i := range log.Events {
			log.Events[i] = readDebugEvent(r)
		}
	}
	return log
}

func readDebugEvent(r *WireReader) DebugEvent {
	return DebugEvent{
		Timestamp:  time.Duration(r.ReadInt64()),
		Kind:       r.ReadString(),
		TargetUID:  r.ReadUint64(),
		TargetType: r.ReadString(),
		Detail:     r.ReadString(),
		Consumed:   r.ReadBool(),
	}
}

// EncodeDebugWidgetTree serializes a DebugWidgetTree into a FrameBuffer entry.
func EncodeDebugWidgetTree(buf *FrameBuffer, tree *DebugWidgetTree) {
	buf.WriteOp(OpDebugWidgetTree, func(w *WireWriter) {
		writeDebugWidgetTree(w, tree)
	})
}

// EncodeDebugEventLog serializes a DebugEventLog into a FrameBuffer entry.
func EncodeDebugEventLog(buf *FrameBuffer, log *DebugEventLog) {
	buf.WriteOp(OpDebugEventLog, func(w *WireWriter) {
		writeDebugEventLog(w, log)
	})
}

// DecodeDebugWidgetTree parses a DebugWidgetTree from raw payload bytes.
func DecodeDebugWidgetTree(payload []byte) (*DebugWidgetTree, error) {
	r := NewWireReader(newBytesReader(payload))
	tree := readDebugWidgetTree(r)
	if r.Err() != nil {
		return nil, r.Err()
	}
	return &tree, nil
}

// DecodeDebugEventLog parses a DebugEventLog from raw payload bytes.
func DecodeDebugEventLog(payload []byte) (*DebugEventLog, error) {
	r := NewWireReader(newBytesReader(payload))
	log := readDebugEventLog(r)
	if r.Err() != nil {
		return nil, r.Err()
	}
	return &log, nil
}

// DecodeDebugFrameInfo parses a DebugFrameInfo from raw payload bytes.
func DecodeDebugFrameInfo(payload []byte) (*DebugFrameInfo, error) {
	r := NewWireReader(newBytesReader(payload))
	info := readDebugFrameInfo(r)
	if r.Err() != nil {
		return nil, r.Err()
	}
	return &info, nil
}
