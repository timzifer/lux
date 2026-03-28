package main

import (
	"fmt"
	"strings"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/display"
	"github.com/timzifer/lux/ui/layout"
)

// inspectorView renders the Inspector UI (RFC-012 §6.3).
func inspectorView(m InspectorModel) ui.Element {
	return layout.Column(
		toolbar(m),
		display.Divider(),
		layout.NewFlex([]ui.Element{
			leftPanel(m),
			display.Divider(),
			rightPanel(m),
		},
			layout.WithDirection(layout.FlexRow),
			layout.WithGap(0),
		),
	)
}

// toolbar renders the top toolbar with Pause and panel tabs.
func toolbar(m InspectorModel) ui.Element {
	pauseLabel := "Pause"
	if m.Paused {
		pauseLabel = "Resume"
	}

	return layout.Pad(ui.UniformInsets(4), layout.NewFlex([]ui.Element{
		display.Text("Lux Inspector"),
		display.Spacer(1),
		panelTab("Tree", PanelWidgetTree, m.ActivePanel),
		panelTab("Events", PanelEventLog, m.ActivePanel),
		panelTab("Metrics", PanelMetrics, m.ActivePanel),
		panelTab("State", PanelState, m.ActivePanel),
		button.Text(pauseLabel, func() { app.Send(TogglePauseMsg{}) }),
	},
		layout.WithDirection(layout.FlexRow),
		layout.WithAlign(layout.AlignCenter),
		layout.WithGap(8),
	))
}

// panelTab renders a tab button for a debug panel.
func panelTab(label string, panel Panel, active Panel) ui.Element {
	if panel == active {
		label = "[" + label + "]"
	}
	return button.Text(label, func() { app.Send(SetPanelMsg{Panel: panel}) })
}

// leftPanel shows frame info and canvas replay status.
func leftPanel(m InspectorModel) ui.Element {
	items := []ui.Element{
		display.Text("Canvas Replay"),
		display.Divider(),
	}

	if m.FrameInfo != nil {
		items = append(items,
			display.Text(fmt.Sprintf("Frame: %d", m.FrameInfo.FrameID)),
			display.Text(fmt.Sprintf("Frame-Time: %s", m.FrameInfo.FrameTime)),
			display.Text(fmt.Sprintf("Update: %s", m.FrameInfo.UpdateTime)),
			display.Text(fmt.Sprintf("Reconcile: %s", m.FrameInfo.ReconcileTime)),
			display.Text(fmt.Sprintf("Paint: %s", m.FrameInfo.PaintTime)),
			display.Text(fmt.Sprintf("Widgets: %d", m.FrameInfo.WidgetCount)),
			display.Text(fmt.Sprintf("Dirty: %d", len(m.FrameInfo.DirtyWidgets))),
		)
	} else {
		items = append(items, display.Text("Waiting for frames..."))
	}

	if m.CurrentFrame != nil {
		items = append(items,
			display.Divider(),
			display.Text(fmt.Sprintf("Ops: %d", len(m.CurrentFrame.Ops))),
		)
	}

	return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
}

// rightPanel shows the active debug panel.
func rightPanel(m InspectorModel) ui.Element {
	switch m.ActivePanel {
	case PanelWidgetTree:
		return widgetTreePanel(m)
	case PanelEventLog:
		return eventLogPanel(m)
	case PanelMetrics:
		return metricsPanel(m)
	case PanelState:
		return statePanel(m)
	default:
		return display.Text("Unknown panel")
	}
}

// widgetTreePanel renders the widget tree (RFC-012 §6.3, left side of right panel).
func widgetTreePanel(m InspectorModel) ui.Element {
	items := []ui.Element{
		display.Text("Widget Tree"),
		display.Divider(),
	}

	if m.WidgetTree == nil {
		items = append(items, display.Text("No widget tree data"))
		return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
	}

	for _, node := range m.WidgetTree.Nodes {
		prefix := ""
		if node.Dirty {
			prefix = "* "
		}
		selected := ""
		if node.UID == m.SelectedUID {
			selected = " [selected]"
		}
		label := fmt.Sprintf("%s%s (uid:%d)%s", prefix, node.TypeName, node.UID, selected)
		uid := node.UID // capture for closure
		items = append(items, button.Text(label, func() {
			app.Send(SelectWidgetMsg{UID: uid})
		}))
	}

	return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
}

// eventLogPanel renders the event log (RFC-012 §6.3).
func eventLogPanel(m InspectorModel) ui.Element {
	items := []ui.Element{
		display.Text("Event Log"),
		display.Divider(),
	}

	if len(m.EventLog) == 0 {
		items = append(items, display.Text("No events"))
		return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
	}

	// Show last 20 events.
	start := 0
	if len(m.EventLog) > 20 {
		start = len(m.EventLog) - 20
	}
	for _, ev := range m.EventLog[start:] {
		consumed := ""
		if ev.Consumed {
			consumed = " [consumed]"
		}
		text := fmt.Sprintf("%s → %s (uid:%d) %s%s",
			ev.Kind, ev.TargetType, ev.TargetUID, ev.Detail, consumed)
		items = append(items, display.Text(text))
	}

	return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
}

// metricsPanel renders the frame metrics dashboard (RFC-012 §6.3).
func metricsPanel(m InspectorModel) ui.Element {
	items := []ui.Element{
		display.Text("Frame Metrics"),
		display.Divider(),
	}

	if len(m.FrameHistory) == 0 {
		items = append(items, display.Text("No frame data"))
		return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
	}

	// Simple text-based bar chart of last frames.
	for _, fm := range m.FrameHistory {
		barLen := int(fm.FrameTime.Milliseconds())
		if barLen > 60 {
			barLen = 60
		}
		bar := strings.Repeat("█", barLen)
		items = append(items, display.Text(fmt.Sprintf("%4d: %s %s", fm.FrameID, bar, fm.FrameTime)))
	}

	return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
}

// statePanel renders the state inspector for the selected widget (RFC-012 §6.3).
func statePanel(m InspectorModel) ui.Element {
	items := []ui.Element{
		display.Text("State Inspector"),
		display.Divider(),
	}

	if m.SelectedUID == 0 {
		items = append(items, display.Text("Select a widget in the Tree panel"))
		return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
	}

	if m.WidgetTree == nil {
		items = append(items, display.Text("No widget tree data"))
		return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
	}

	// Find the selected widget.
	for _, node := range m.WidgetTree.Nodes {
		if node.UID != m.SelectedUID {
			continue
		}

		items = append(items,
			display.Text(fmt.Sprintf("UID: %d", node.UID)),
			display.Text(fmt.Sprintf("Type: %s", node.TypeName)),
			display.Text(fmt.Sprintf("Bounds: (%.0f, %.0f, %.0f, %.0f)",
				node.Bounds.X, node.Bounds.Y, node.Bounds.W, node.Bounds.H)),
			display.Text(fmt.Sprintf("Dirty: %v", node.Dirty)),
		)

		if len(node.Props) > 0 {
			items = append(items, display.Divider(), display.Text("Props:"))
			for k, v := range node.Props {
				items = append(items, display.Text(fmt.Sprintf("  %s: %s", k, v)))
			}
		}

		if node.StateDump != "" {
			items = append(items, display.Divider(), display.Text("State:"), display.Text(node.StateDump))
		}

		break
	}

	return layout.Pad(ui.UniformInsets(8), layout.Column(items...))
}
