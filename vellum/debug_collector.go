package vellum

import (
	"fmt"
	"reflect"
	"time"

	"github.com/timzifer/lux/draw"
)

// DebugExtensionCollector gathers debug data from the Lux runtime for
// transmission to the Inspector (RFC-012 §4).
type DebugExtensionCollector struct {
	frameCounter uint64
}

// NewDebugExtensionCollector creates a collector.
func NewDebugExtensionCollector() *DebugExtensionCollector {
	return &DebugExtensionCollector{}
}

// FrameTimings holds timing data for a single frame.
type FrameTimings struct {
	UpdateTime    time.Duration
	ReconcileTime time.Duration
	LayoutTime    time.Duration
	PaintTime     time.Duration
}

// CollectFrameInfo builds a DebugFrameInfo from frame timings and dirty UIDs.
func (c *DebugExtensionCollector) CollectFrameInfo(
	timings FrameTimings,
	widgetCount uint32,
	dirtyUIDs []uint64,
) *DebugFrameInfo {
	c.frameCounter++
	return &DebugFrameInfo{
		FrameID:       c.frameCounter,
		FrameTime:     timings.UpdateTime + timings.ReconcileTime + timings.LayoutTime + timings.PaintTime,
		UpdateTime:    timings.UpdateTime,
		ReconcileTime: timings.ReconcileTime,
		LayoutTime:    timings.LayoutTime,
		PaintTime:     timings.PaintTime,
		WidgetCount:   widgetCount,
		DirtyWidgets:  dirtyUIDs,
	}
}

// WidgetInfo holds per-widget data for building DebugWidgetTree.
type WidgetInfo struct {
	UID       uint64
	TypeName  string
	Props     map[string]string
	StateDump string
	Bounds    draw.Rect
	Padding   draw.Insets
	Margin    draw.Insets
	Dirty     bool
}

// CollectWidgetTree builds a DebugWidgetTree from a list of widget infos.
func (c *DebugExtensionCollector) CollectWidgetTree(widgets []WidgetInfo) *DebugWidgetTree {
	nodes := make([]DebugWidgetNode, len(widgets))
	for i, w := range widgets {
		nodes[i] = DebugWidgetNode{
			UID:       w.UID,
			TypeName:  w.TypeName,
			Props:     w.Props,
			StateDump: w.StateDump,
			Bounds:    w.Bounds,
			Padding:   w.Padding,
			Margin:    w.Margin,
			Dirty:     w.Dirty,
		}
	}
	return &DebugWidgetTree{
		Version: c.frameCounter,
		Nodes:   nodes,
	}
}

// CollectEventLog builds a DebugEventLog from a list of debug events.
func (c *DebugExtensionCollector) CollectEventLog(events []DebugEvent) *DebugEventLog {
	return &DebugEventLog{
		FrameID: c.frameCounter,
		Events:  events,
	}
}

// WidgetTypeName returns a human-readable type name for a widget value.
func WidgetTypeName(w any) string {
	if w == nil {
		return "<nil>"
	}
	t := reflect.TypeOf(w)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	pkg := t.PkgPath()
	name := t.Name()
	if pkg == "" {
		return name
	}
	// Shorten package path to last segment.
	for i := len(pkg) - 1; i >= 0; i-- {
		if pkg[i] == '/' {
			pkg = pkg[i+1:]
			break
		}
	}
	return fmt.Sprintf("%s.%s", pkg, name)
}
