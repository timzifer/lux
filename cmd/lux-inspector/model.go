package main

import (
	"time"

	"github.com/timzifer/lux/internal/vellum"
)

// Panel identifies an active debug panel.
type Panel int

const (
	PanelWidgetTree Panel = iota
	PanelEventLog
	PanelMetrics
	PanelState
)

// InspectorModel holds the complete state of the Inspector UI (RFC-012 §6.2).
type InspectorModel struct {
	Client *vellum.Client

	// Current frame state from the inspected app.
	CurrentFrame *vellum.DecodedFrame
	FrameInfo    *vellum.DebugFrameInfo
	WidgetTree   *vellum.DebugWidgetTree
	EventLog     []vellum.DebugEvent

	// UI state.
	SelectedUID uint64
	ActivePanel Panel
	ShowOverlay bool
	ShowDirty   bool
	Paused      bool

	// Metrics history (ring buffer of last 120 frames).
	FrameHistory []FrameMetric
}

// FrameMetric stores timing data for one frame (for the metrics chart).
type FrameMetric struct {
	FrameID       uint64
	FrameTime     time.Duration
	UpdateTime    time.Duration
	ReconcileTime time.Duration
	PaintTime     time.Duration
	WidgetCount   uint32
}

const maxFrameHistory = 120
const maxEventLog = 500

// NewInspectorModel creates an InspectorModel connected to a Vellum client.
func NewInspectorModel(client *vellum.Client) InspectorModel {
	return InspectorModel{
		Client:       client,
		ActivePanel:  PanelWidgetTree,
		ShowOverlay:  true,
		ShowDirty:    true,
		FrameHistory: make([]FrameMetric, 0, maxFrameHistory),
	}
}
