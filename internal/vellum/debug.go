package vellum

import (
	"time"

	"github.com/timzifer/lux/draw"
)

// DebugFrameInfo carries per-frame debug data (RFC-012 §4.1).
// Opcode: 0xD0, attached to EndFrame.
type DebugFrameInfo struct {
	FrameID       uint64
	FrameTime     time.Duration
	UpdateTime    time.Duration
	ReconcileTime time.Duration
	LayoutTime    time.Duration
	PaintTime     time.Duration
	WidgetCount   uint32
	DirtyWidgets  []uint64
}

// DebugWidgetTree extends the AccessTree with inspector data (RFC-012 §4.2).
// Opcode: 0xD1, sent on Channel 0 after AccessTreeUpdate.
type DebugWidgetTree struct {
	Version uint64
	Nodes   []DebugWidgetNode
}

// DebugWidgetNode describes a single widget in the debug tree.
type DebugWidgetNode struct {
	UID       uint64
	TypeName  string
	Props     map[string]string
	StateDump string
	Bounds    draw.Rect
	Padding   draw.Insets
	Margin    draw.Insets
	Dirty     bool
}

// DebugEventLog mirrors dispatched events for the inspector (RFC-012 §4.3).
// Opcode: 0xD2, sent on Channel 0 after DebugWidgetTree.
type DebugEventLog struct {
	FrameID uint64
	Events  []DebugEvent
}

// DebugEvent describes a single dispatched input event.
type DebugEvent struct {
	Timestamp  time.Duration
	Kind       string
	TargetUID  uint64
	TargetType string
	Detail     string
	Consumed   bool
}
