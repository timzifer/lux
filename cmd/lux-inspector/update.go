package main

import (
	"log"

	"github.com/timzifer/lux/app"
	"github.com/timzifer/lux/internal/vellum"
)

// Messages for the Inspector update loop.

// FrameReceivedMsg is sent when a new frame arrives from the Vellum client.
type FrameReceivedMsg struct {
	Frame *vellum.DecodedFrame
}

// ControlReceivedMsg is sent when a control message arrives.
type ControlReceivedMsg struct {
	Data []byte
}

// SelectWidgetMsg selects a widget in the tree panel.
type SelectWidgetMsg struct {
	UID uint64
}

// SetPanelMsg switches the active debug panel.
type SetPanelMsg struct {
	Panel Panel
}

// ToggleOverlayMsg toggles the layout overlay.
type ToggleOverlayMsg struct{}

// ToggleDirtyMsg toggles paint highlighting.
type ToggleDirtyMsg struct{}

// TogglePauseMsg pauses/resumes the frame stream.
type TogglePauseMsg struct{}

// startFrameListener returns a Cmd that starts reading frames from the client.
func startFrameListener(client *vellum.Client) app.Cmd {
	return func() app.Msg {
		data, err := client.NextFrame()
		if err != nil {
			log.Printf("lux-inspector: frame error: %v", err)
			return nil
		}
		frame, err := vellum.DecodeFrame(data)
		if err != nil {
			log.Printf("lux-inspector: decode error: %v", err)
			return nil
		}
		return FrameReceivedMsg{Frame: frame}
	}
}

// startControlListener returns a Cmd that reads control messages.
func startControlListener(client *vellum.Client) app.Cmd {
	return func() app.Msg {
		data, err := client.NextControl()
		if err != nil {
			return nil
		}
		return ControlReceivedMsg{Data: data}
	}
}

// inspectorUpdate handles messages for the Inspector (RFC-012 §6.2).
func inspectorUpdate(model InspectorModel, msg app.Msg) InspectorModel {
	switch m := msg.(type) {
	case app.TickMsg:
		// On first tick, start listening for frames and control messages.
		// We use TickMsg as a bootstrap trigger since Cmd is not available
		// in the simple update signature.
		return model

	case FrameReceivedMsg:
		if model.Paused {
			return model
		}
		model.CurrentFrame = m.Frame
		if m.Frame.FrameInfo != nil {
			model.FrameInfo = m.Frame.FrameInfo
			// Append to frame history.
			metric := FrameMetric{
				FrameID:       m.Frame.FrameInfo.FrameID,
				FrameTime:     m.Frame.FrameInfo.FrameTime,
				UpdateTime:    m.Frame.FrameInfo.UpdateTime,
				ReconcileTime: m.Frame.FrameInfo.ReconcileTime,
				PaintTime:     m.Frame.FrameInfo.PaintTime,
				WidgetCount:   m.Frame.FrameInfo.WidgetCount,
			}
			if len(model.FrameHistory) >= maxFrameHistory {
				model.FrameHistory = append(model.FrameHistory[1:], metric)
			} else {
				model.FrameHistory = append(model.FrameHistory, metric)
			}
		}
		return model

	case ControlReceivedMsg:
		// Parse control messages for debug extensions.
		if len(m.Data) == 0 {
			return model
		}
		opcode, payload, _, err := vellum.ReadOp(m.Data, 0)
		if err != nil {
			return model
		}
		switch opcode {
		case vellum.OpDebugWidgetTree:
			tree, err := vellum.DecodeDebugWidgetTree(payload)
			if err == nil {
				model.WidgetTree = tree
			}
		case vellum.OpDebugEventLog:
			evLog, err := vellum.DecodeDebugEventLog(payload)
			if err == nil {
				model.EventLog = append(model.EventLog, evLog.Events...)
				if len(model.EventLog) > maxEventLog {
					model.EventLog = model.EventLog[len(model.EventLog)-maxEventLog:]
				}
			}
		}
		return model

	case SelectWidgetMsg:
		model.SelectedUID = m.UID
		return model

	case SetPanelMsg:
		model.ActivePanel = m.Panel
		return model

	case ToggleOverlayMsg:
		model.ShowOverlay = !model.ShowOverlay
		return model

	case ToggleDirtyMsg:
		model.ShowDirty = !model.ShowDirty
		return model

	case TogglePauseMsg:
		model.Paused = !model.Paused
		return model
	}

	return model
}
